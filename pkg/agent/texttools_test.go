// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"testing"
)

func TestParseFileBlocks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []FileBlock
	}{
		{
			name: "simple shell script with colon separator",
			content: "Here's a test script:\n\n```sh:test.sh\n#!/bin/sh\necho hello\n```\n\nDone!",
			expected: []FileBlock{
				{Language: "sh", Path: "test.sh", Content: "#!/bin/sh\necho hello"},
			},
		},
		{
			name: "go file with colon separator",
			content: "```go:main.go\npackage main\n\nfunc main() {}\n```",
			expected: []FileBlock{
				{Language: "go", Path: "main.go", Content: "package main\n\nfunc main() {}"},
			},
		},
		{
			name: "path without language",
			content: "```test.py\nprint('hello')\n```",
			expected: []FileBlock{
				{Language: "", Path: "test.py", Content: "print('hello')"},
			},
		},
		{
			name: "multiple files",
			content: "```sh:script.sh\necho script\n```\n\n```py:app.py\nprint('app')\n```",
			expected: []FileBlock{
				{Language: "sh", Path: "script.sh", Content: "echo script"},
				{Language: "py", Path: "app.py", Content: "print('app')"},
			},
		},
		{
			name: "relative path",
			content: "```go:./cmd/server/main.go\npackage main\n```",
			expected: []FileBlock{
				{Language: "go", Path: "./cmd/server/main.go", Content: "package main"},
			},
		},
		{
			name:     "invalid path - placeholder",
			content:  "```sh:filename\necho test\n```",
			expected: nil,
		},
		{
			name:     "invalid path - example",
			content:  "```sh:example\necho test\n```",
			expected: nil,
		},
		{
			name:     "no file path",
			content:  "```sh\necho test\n```",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := ParseFileBlocks(tt.content)

			if len(blocks) != len(tt.expected) {
				t.Errorf("expected %d blocks, got %d", len(tt.expected), len(blocks))
				return
			}

			for i, block := range blocks {
				if block.Language != tt.expected[i].Language {
					t.Errorf("block %d: expected language %q, got %q", i, tt.expected[i].Language, block.Language)
				}
				if block.Path != tt.expected[i].Path {
					t.Errorf("block %d: expected path %q, got %q", i, tt.expected[i].Path, block.Path)
				}
				if block.Content != tt.expected[i].Content {
					t.Errorf("block %d: expected content %q, got %q", i, tt.expected[i].Content, block.Content)
				}
			}
		})
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.sh", true},
		{"main.go", true},
		{"app.py", true},
		{"./src/index.ts", true},
		{"~/scripts/deploy.sh", true},
		{"/etc/config.yaml", true},
		{"", false},
		{"a", false},
		{"example", false},
		{"filename", false},
		{"path", false},
		{"your-file", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isValidFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("isValidFilePath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestConvertFileBlocksToToolCalls(t *testing.T) {
	blocks := []FileBlock{
		{Language: "sh", Path: "test.sh", Content: "#!/bin/sh\necho hello"},
		{Language: "go", Path: "main.go", Content: "package main"},
	}

	toolCalls := ConvertFileBlocksToToolCalls(blocks)

	if len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(toolCalls))
	}

	if toolCalls[0].Name != "write_file" {
		t.Errorf("expected tool name 'write_file', got %q", toolCalls[0].Name)
	}

	if toolCalls[0].Arguments["path"] != "test.sh" {
		t.Errorf("expected path 'test.sh', got %v", toolCalls[0].Arguments["path"])
	}

	if toolCalls[0].Arguments["content"] != "#!/bin/sh\necho hello" {
		t.Errorf("unexpected content: %v", toolCalls[0].Arguments["content"])
	}
}
