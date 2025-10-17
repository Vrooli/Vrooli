package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var ollamaURL = getEnv("OLLAMA_URL", "http://localhost:11434/api/generate")
var ollamaModel = getEnv("OLLAMA_MODEL", "llama3.2:3b")

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func generateCommandWithOllama(userPrompt string, context []string) (string, error) {
	// Build the prompt
	systemPrompt := `You are a helpful Unix/Linux command line assistant. Your job is to generate shell commands based on user requests.

Rules:
1. Only output the command itself, nothing else
2. Do not include explanations, markdown, or code blocks
3. Do not include comments
4. Generate practical, safe commands
5. If context is provided, consider it when generating the command

Examples:
User: list all docker containers
Output: docker ps -a

User: find large files
Output: find . -type f -size +100M

User: check disk usage
Output: df -h
`

	// Add context if provided
	if len(context) > 0 {
		systemPrompt += "\n\nTerminal context (recent output):\n"
		for _, line := range context {
			systemPrompt += line + "\n"
		}
	}

	systemPrompt += fmt.Sprintf("\n\nUser request: %s\nOutput:", userPrompt)

	// Make request to Ollama
	reqBody := ollamaRequest{
		Model:  ollamaModel,
		Prompt: systemPrompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w (is Ollama running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	if !ollamaResp.Done {
		return "", errors.New("Ollama response incomplete")
	}

	// Clean up the response
	command := strings.TrimSpace(ollamaResp.Response)

	// Remove markdown code blocks if present
	command = strings.TrimPrefix(command, "```bash")
	command = strings.TrimPrefix(command, "```sh")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimSpace(command)

	// Remove any $ prompt prefix
	command = strings.TrimPrefix(command, "$ ")

	if command == "" {
		return "", errors.New("Ollama generated empty command")
	}

	return command, nil
}
