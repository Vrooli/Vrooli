// Package testing provides LLM-based skill testing via the resource-ollama gateway.
package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

type ollamaRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

const defaultOllamaGatewayBin = "resource-ollama"

// OllamaClient handles skill-test generation through the resource-ollama gateway.
// This is a testing seam: inject a runner to test without a real Ollama instance.
type OllamaClient struct {
	enabled bool
	bin     string
	run     ollamaRunner
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(enabled bool, bin string) *OllamaClient {
	return newOllamaClientWithRunner(enabled, bin, defaultOllamaRunner)
}

func newOllamaClientWithRunner(enabled bool, bin string, run ollamaRunner) *OllamaClient {
	if bin == "" {
		bin = defaultOllamaGatewayBin
	}
	return &OllamaClient{enabled: enabled, bin: bin, run: run}
}

// IsEnabled returns true if Ollama is configured.
func (c *OllamaClient) IsEnabled() bool {
	return c.enabled
}

// Generate runs a skill through Ollama and returns the response.
func (c *OllamaClient) Generate(role, prompt string, maxTokens int, temperature float64) (*OllamaResponse, float64, error) {
	if !c.IsEnabled() {
		return nil, 0, fmt.Errorf("ollama not configured")
	}
	if c.run == nil {
		return nil, 0, fmt.Errorf("ollama gateway runner is not configured")
	}

	args := []string{
		c.bin,
		"gateway",
		"generate",
		"--role", role,
		"--json",
		"--prompt-stdin",
	}
	if maxTokens > 0 {
		args = append(args, "--max-tokens", strconv.Itoa(maxTokens))
	}
	if temperature >= 0 {
		args = append(args, "--temperature", strconv.FormatFloat(temperature, 'g', -1, 64))
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := c.run(ctx, args, []byte(prompt))
	responseTime := float64(time.Since(startTime).Milliseconds())
	if err != nil {
		return nil, responseTime, fmt.Errorf("failed to call Ollama gateway: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &ollamaResp); err != nil {
		return nil, responseTime, fmt.Errorf("failed to parse Ollama gateway response: %w", err)
	}

	return &ollamaResp, responseTime, nil
}

func defaultOllamaRunner(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := bytes.TrimSpace(stderr.Bytes())
		if len(msg) == 0 {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", string(msg), err)
	}
	return stdout.Bytes(), nil
}
