package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
	sessionID  string
	lastHTML   string
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
		sessionID:  uuid.New().String(),
	}
}

func (s *Session) LastHTML() string {
	return s.lastHTML
}

func (s *Session) SetLastHTML(html string) {
	s.lastHTML = html
}

// ExecuteInstruction runs a single instruction within a persistent Browserless session.
func (s *Session) ExecuteInstruction(ctx context.Context, instruction Instruction) (*ExecutionResponse, error) {
	script, err := buildStepScript(instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to build browserless script: %w", err)
	}

	payload := map[string]any{
		"code": script,
	}

	if strings.TrimSpace(s.sessionID) != "" {
		payload["context"] = map[string]any{
			"sessionId": s.sessionID,
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode browserless payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+browserlessFunctionPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build browserless request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

// SessionID returns the underlying Browserless session identifier.
func (s *Session) SessionID() string {
	return s.sessionID
}
