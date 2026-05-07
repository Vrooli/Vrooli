package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// runOllamaJSONGenerate calls `resource-ollama gateway generate` with the given
// prompt and returns the raw text response. All daemon traffic goes through
// the CLI so the host-wide semaphore can bound fleet-wide parallelism — never
// call Ollama HTTP directly.
func runOllamaJSONGenerate(prompt string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "resource-ollama", "gateway", "generate",
		"--model", "llama3.2", "--json", "--prompt-stdin")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("resource-ollama gateway generate failed: %v: %s", err, stderr)
	}

	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway generate response: %w", err)
	}

	// The response field contains a JSON document the prompt asks for; surface
	// it to callers in the same shape the legacy /api/generate path returned.
	return map[string]interface{}{"response": decoded.Response}, nil
}
