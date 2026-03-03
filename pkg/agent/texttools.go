// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// fileCodeBlockRegex matches code blocks with file path annotations
// Format: ```lang:path or ```path (where path must look like a file path)
// We use two separate regex patterns for clarity
var fileCodeBlockWithLangRegex = regexp.MustCompile("(?s)```([a-zA-Z0-9+-]+)[ \t]*:[ \t]*([^\\s`\\n]+)\\s*\\n(.*?)\\s*```")
var fileCodeBlockNoLangRegex = regexp.MustCompile("(?s)```([a-zA-Z0-9_./~-][^\\s`:\\n]*\\.[a-zA-Z0-9]+)\\s*\\n(.*?)\\s*```")

// FileBlock represents a parsed code block with file path
type FileBlock struct {
	Language string
	Path     string
	Content  string
}

// ParseFileBlocks extracts code blocks that appear to be file writes from text content
func ParseFileBlocks(content string) []FileBlock {
	var blocks []FileBlock
	seen := make(map[string]bool) // Avoid duplicates

	// First, match blocks with language:path format
	matches := fileCodeBlockWithLangRegex.FindAllStringSubmatchIndex(content, -1)
	for _, matchIdx := range matches {
		if len(matchIdx) < 8 {
			continue
		}

		// Extract groups using indices
		lang := strings.TrimSpace(content[matchIdx[2]:matchIdx[3]])
		path := strings.TrimSpace(content[matchIdx[4]:matchIdx[5]])
		codeContent := strings.TrimRight(content[matchIdx[6]:matchIdx[7]], " \t")

		if !isValidFilePath(path) || seen[path] {
			continue
		}
		seen[path] = true

		blocks = append(blocks, FileBlock{
			Language: lang,
			Path:     path,
			Content:  codeContent,
		})
	}

	// Then, match blocks with just a file path (no language)
	// Only if not already matched by the lang:path regex
	matches2 := fileCodeBlockNoLangRegex.FindAllStringSubmatchIndex(content, -1)
	for _, matchIdx := range matches2 {
		if len(matchIdx) < 6 {
			continue
		}

		path := strings.TrimSpace(content[matchIdx[2]:matchIdx[3]])
		codeContent := strings.TrimRight(content[matchIdx[4]:matchIdx[5]], " \t")

		if !isValidFilePath(path) || seen[path] {
			continue
		}
		seen[path] = true

		blocks = append(blocks, FileBlock{
			Language: "",
			Path:     path,
			Content:  codeContent,
		})
	}

	return blocks
}

// isValidFilePath checks if a string looks like a valid file path
func isValidFilePath(s string) bool {
	if s == "" {
		return false
	}

	// Must contain at least one non-extension character
	if len(s) < 2 {
		return false
	}

	// Skip common non-file patterns
	invalidPatterns := []string{
		"example", "filename", "path", "file", "your-file",
		"output", "result", "code", "snippet",
	}
	lowerS := strings.ToLower(s)
	for _, pattern := range invalidPatterns {
		if lowerS == pattern {
			return false
		}
	}

	// Check for file-like characteristics
	hasExtension := strings.Contains(s, ".")
	hasPathSeparator := strings.Contains(s, "/") || strings.Contains(s, "\\")
	startsWithLetter := (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_' || s[0] == '.' || s[0] == '/' || s[0] == '~'

	// If it has a dot but looks like just an extension (e.g., ".py"), skip it
	if hasExtension && strings.HasPrefix(s, ".") && len(s) <= 4 {
		return false
	}

	return startsWithLetter && (hasExtension || hasPathSeparator || len(s) <= 64)
}

// ConvertFileBlocksToToolCalls converts parsed file blocks to write_file tool calls
func ConvertFileBlocksToToolCalls(blocks []FileBlock) []providers.ToolCall {
	var toolCalls []providers.ToolCall

	for i, block := range blocks {
		toolCalls = append(toolCalls, providers.ToolCall{
			ID:   generateToolCallID(i),
			Type: "function",
			Name: "write_file",
			Function: &providers.FunctionCall{
				Name:      "write_file",
				Arguments: block.Path,
			},
			Arguments: map[string]any{
				"path":    block.Path,
				"content": block.Content,
			},
		})
	}

	return toolCalls
}

// generateToolCallID creates a unique ID for synthetic tool calls
func generateToolCallID(index int) string {
	return "text_parsed_" + strings.Repeat("0", max(0, 4-len(string(rune('0'+index))))) + string(rune('0'+index))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
