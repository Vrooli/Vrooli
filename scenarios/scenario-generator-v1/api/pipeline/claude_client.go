package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeClient handles all interactions with Claude via vrooli resource
type ClaudeClient struct {
	timeout        time.Duration
	maxRetries     int
	vrooliRoot     string
}

// NewClaudeClient creates a new Claude client
func NewClaudeClient() *ClaudeClient {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}
	
	return &ClaudeClient{
		timeout:    10 * time.Minute,
		maxRetries: 3,
		vrooliRoot: vrooliRoot,
	}
}

// ClaudeRequest represents a request to Claude
type ClaudeRequest struct {
	Prompt      string            `json:"prompt"`
	System      string            `json:"system,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// ClaudeResponse represents Claude's response
type ClaudeResponse struct {
	Content    string    `json:"content"`
	TokensUsed int       `json:"tokens_used"`
	Model      string    `json:"model"`
	Timestamp  time.Time `json:"timestamp"`
}

// Chat sends a chat request to Claude via vrooli resource
func (c *ClaudeClient) Chat(prompt string) (string, error) {
	return c.ChatWithOptions(ClaudeRequest{
		Prompt:      prompt,
		Temperature: 0.7,
		MaxTokens:   8000,
	})
}

// ChatWithOptions sends a chat request with custom options
func (c *ClaudeClient) ChatWithOptions(req ClaudeRequest) (string, error) {
	var lastErr error
	
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		response, err := c.executeClaudeCommand(req)
		if err == nil {
			return response, nil
		}
		
		lastErr = err
		
		// Log retry attempt
		fmt.Printf("Claude request attempt %d/%d failed: %v\n", attempt, c.maxRetries, err)
		
		// Exponential backoff
		if attempt < c.maxRetries {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}
	}
	
	return "", fmt.Errorf("claude request failed after %d attempts: %w", c.maxRetries, lastErr)
}

// executeClaudeCommand runs the resource-claude-code command
func (c *ClaudeClient) executeClaudeCommand(req ClaudeRequest) (string, error) {
	// Compose the full prompt with system message if provided
	var fullPrompt string
	if req.System != "" {
		fullPrompt = fmt.Sprintf("System: %s\n\n%s", req.System, req.Prompt)
	} else {
		fullPrompt = req.Prompt
	}
	
	// Add context information if provided
	if len(req.Context) > 0 {
		contextLines := []string{"Context:"}
		for key, value := range req.Context {
			contextLines = append(contextLines, fmt.Sprintf("- %s: %s", key, value))
		}
		fullPrompt = strings.Join(contextLines, "\n") + "\n\n" + fullPrompt
	}
	
	// Create the command - resource-claude-code execute runs the prompt
	// We use stdin to pass the prompt to avoid shell escaping issues
	cmd := exec.Command("resource-claude-code", "execute", "-")
	
	// Set working directory
	cmd.Dir = c.vrooliRoot
	
	// Set environment variables
	cmd.Env = os.Environ()
	
	// Set stdin with the prompt
	cmd.Stdin = strings.NewReader(fullPrompt)
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Set timeout using a channel
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			// Check stderr for more details
			stderrStr := stderr.String()
			
			// Check for common resource-claude-code errors
			if strings.Contains(stderrStr, "rate limit") {
				return "", fmt.Errorf("claude rate limit exceeded, please wait before retrying")
			}
			if strings.Contains(stderrStr, "not authenticated") || strings.Contains(stderrStr, "auth") {
				return "", fmt.Errorf("claude not authenticated - run 'claude auth login' to authenticate")
			}
			if strings.Contains(stderrStr, "command not found") {
				return "", fmt.Errorf("resource-claude-code not found - ensure it's installed")
			}
			
			if stderrStr != "" {
				return "", fmt.Errorf("claude command failed: %s", stderrStr)
			}
			return "", fmt.Errorf("claude command failed: %w", err)
		}
		
		output := stdout.String()
		if output == "" {
			return "", fmt.Errorf("claude returned empty response")
		}
		
		return output, nil
		
	case <-time.After(c.timeout):
		// Kill the process if it's still running
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("claude request timed out after %v", c.timeout)
	}
}

// ParseCodeBlocks extracts code blocks from Claude's markdown response
func (c *ClaudeClient) ParseCodeBlocks(response string) map[string]string {
	files := make(map[string]string)
	
	// Match code blocks with filename markers
	// Supports formats like ```filename.ext and ```language:filename.ext
	lines := strings.Split(response, "\n")
	inCodeBlock := false
	currentFile := ""
	currentContent := []string{}
	
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Starting a code block
				inCodeBlock = true
				
				// Extract filename if present
				marker := strings.TrimPrefix(line, "```")
				marker = strings.TrimSpace(marker)
				
				// Check for filename patterns
				if strings.Contains(marker, "/") || strings.Contains(marker, ".") {
					// Likely a filename
					// Remove language prefix if present (e.g., "go:main.go" -> "main.go")
					if colonIdx := strings.Index(marker, ":"); colonIdx != -1 {
						currentFile = strings.TrimSpace(marker[colonIdx+1:])
					} else {
						currentFile = marker
					}
				} else if marker == "json" || marker == "yaml" || marker == "sql" || 
				         marker == "go" || marker == "javascript" || marker == "typescript" ||
				         marker == "bash" || marker == "shell" {
					// Language marker without filename - we'll infer later
					currentFile = ""
				}
			} else {
				// Ending a code block
				inCodeBlock = false
				
				if currentFile != "" {
					files[currentFile] = strings.Join(currentContent, "\n")
				} else if len(currentContent) > 0 {
					// Try to infer filename from content
					content := strings.Join(currentContent, "\n")
					inferredFile := c.inferFilename(content)
					if inferredFile != "" {
						files[inferredFile] = content
					}
				}
				
				currentFile = ""
				currentContent = []string{}
			}
		} else if inCodeBlock {
			currentContent = append(currentContent, line)
		}
	}
	
	return files
}

// inferFilename attempts to infer a filename from code content
func (c *ClaudeClient) inferFilename(content string) string {
	trimmed := strings.TrimSpace(content)
	
	// Check for common file patterns
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"name\"") {
		if strings.Contains(trimmed, "\"version\"") && strings.Contains(trimmed, "\"resources\"") {
			return "service.json"
		}
		if strings.Contains(trimmed, "\"dependencies\"") || strings.Contains(trimmed, "\"scripts\"") {
			return "package.json"
		}
	}
	
	if strings.HasPrefix(trimmed, "CREATE TABLE") || strings.HasPrefix(trimmed, "-- ") {
		return "schema.sql"
	}
	
	if strings.HasPrefix(trimmed, "package main") {
		return "main.go"
	}
	
	if strings.HasPrefix(trimmed, "<!DOCTYPE html") || strings.HasPrefix(trimmed, "<html") {
		return "index.html"
	}
	
	if strings.HasPrefix(trimmed, "#!/usr/bin/env bash") || strings.HasPrefix(trimmed, "#!/bin/bash") {
		return "script.sh"
	}
	
	// Default to empty - caller should handle
	return ""
}

// StreamChat sends a chat request and processes the response in chunks
// Note: resource-claude-code doesn't support true streaming yet, so we get the full response and simulate streaming
func (c *ClaudeClient) StreamChat(prompt string, onChunk func(string)) error {
	// Get the full response using the standard Chat method
	response, err := c.Chat(prompt)
	if err != nil {
		return err
	}
	
	// Process response in chunks for UI responsiveness
	// This helps with large responses and provides visual feedback
	chunkSize := 200  // Larger chunks for better performance
	for i := 0; i < len(response); i += chunkSize {
		end := i + chunkSize
		if end > len(response) {
			end = len(response)
		}
		onChunk(response[i:end])
		
		// Small delay to allow UI updates
		time.Sleep(5 * time.Millisecond)
	}
	
	return nil
}