package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	browserlessFunctionPath = "/chrome/function"
	defaultViewportWidth    = 1920
	defaultViewportHeight   = 1080
	defaultTimeoutMillis    = 30000
)

// Session manages low-level Browserless interactions.
type Session struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Logger
}

// NewSession creates a Browserless session wrapper.
func NewSession(baseURL string, httpClient *http.Client, log *logrus.Logger) *Session {
	trimmed := strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &Session{
		baseURL:    trimmed,
		httpClient: httpClient,
		log:        log,
	}
}

// Execute ships the provided instructions to Browserless and returns the execution response.
func (s *Session) Execute(ctx context.Context, instructions []Instruction) (*ExecutionResponse, error) {
	if len(instructions) == 0 {
		return &ExecutionResponse{Success: true, Steps: []StepResult{}}, nil
	}

	script, err := buildWorkflowScript(instructions)
	if err != nil {
		return nil, fmt.Errorf("failed to build browserless script: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+browserlessFunctionPath, strings.NewReader(script))
	if err != nil {
		return nil, fmt.Errorf("failed to build browserless request: %w", err)
	}
	req.Header.Set("Content-Type", "application/javascript")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("browserless request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("browserless returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response ExecutionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode browserless response: %w", err)
	}

	return &response, nil
}
