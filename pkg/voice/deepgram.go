package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// DeepgramTranscriber implements Transcriber using Deepgram's API
type DeepgramTranscriber struct {
	apiKey     string
	apiBase    string
	model      string
	language   string
	httpClient *http.Client
}

type DeepgramConfig struct {
	APIKey   string
	Model    string // default: "nova-3"
	Language string // default: "en"
}

func NewDeepgramTranscriber(cfg DeepgramConfig) *DeepgramTranscriber {
	if cfg.Model == "" {
		cfg.Model = "nova-2"
	}
	if cfg.Language == "" {
		cfg.Language = "zh"
	}

	logger.DebugCF("voice", "Creating Deepgram transcriber", map[string]any{
		"has_api_key": cfg.APIKey != "",
		"model":       cfg.Model,
		"language":    cfg.Language,
	})

	apiBase := "https://api.deepgram.com/v1"
	return &DeepgramTranscriber{
		apiKey:   cfg.APIKey,
		apiBase:  apiBase,
		model:    cfg.Model,
		language: cfg.Language,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (t *DeepgramTranscriber) Transcribe(ctx context.Context, audioFilePath string) (*TranscriptionResponse, error) {
	logger.InfoCF("voice", "Starting Deepgram transcription", map[string]any{"audio_file": audioFilePath})

	audioData, err := os.ReadFile(audioFilePath)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read audio file", map[string]any{"path": audioFilePath, "error": err})
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	logger.DebugCF("voice", "Audio file details", map[string]any{
		"size_bytes": len(audioData),
		"file_name":  filepath.Base(audioFilePath),
	})

	// Build URL with query parameters
	url := fmt.Sprintf("%s/listen?model=%s&language=%s&smart_format=true&punctuate=true",
		t.apiBase, t.model, t.language)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(audioData))
	if err != nil {
		logger.ErrorCF("voice", "Failed to create request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type based on file extension
	contentType := "audio/ogg"
	ext := filepath.Ext(audioFilePath)
	switch ext {
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	case ".m4a":
		contentType = "audio/mp4"
	case ".flac":
		contentType = "audio/flac"
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Token "+t.apiKey)

	logger.DebugCF("voice", "Sending transcription request to Deepgram API", map[string]any{
		"url":             url,
		"content_type":    contentType,
		"audio_size_bytes": len(audioData),
	})

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "API error", map[string]any{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.DebugCF("voice", "Received response from Deepgram API", map[string]any{
		"status_code":         resp.StatusCode,
		"response_size_bytes": len(body),
	})

	// Parse Deepgram response format
	var deepgramResp struct {
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string  `json:"transcript"`
					Confidence float64 `json:"confidence"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &deepgramResp); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract transcript
	var transcript string
	if len(deepgramResp.Results.Channels) > 0 &&
		len(deepgramResp.Results.Channels[0].Alternatives) > 0 {
		transcript = deepgramResp.Results.Channels[0].Alternatives[0].Transcript
	}

	result := &TranscriptionResponse{
		Text:     transcript,
		Language: t.language,
	}

	logger.InfoCF("voice", "Transcription completed successfully", map[string]any{
		"text_length":           len(result.Text),
		"language":              result.Language,
		"transcription_preview": utils.Truncate(result.Text, 50),
	})

	return result, nil
}

func (t *DeepgramTranscriber) IsAvailable() bool {
	available := t.apiKey != ""
	logger.DebugCF("voice", "Checking Deepgram transcriber availability", map[string]any{"available": available})
	return available
}
