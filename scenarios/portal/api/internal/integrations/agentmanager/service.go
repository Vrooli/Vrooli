package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	apiURLEnv       = "AGENT_MANAGER_API_URL"
	baseURLEnv      = "AGENT_MANAGER_BASE_URL"
	defaultScenario = "agent-manager"
	defaultPortKey  = "API_PORT"
)

var ErrUnavailable = errors.New("agent-manager unavailable")

type EventKind string

const (
	EventKindStatus   EventKind = "status"
	EventKindProgress EventKind = "progress"
	EventKindLog      EventKind = "log"
	EventKindMessage  EventKind = "message"
	EventKindTool     EventKind = "tool"
	EventKindError    EventKind = "error"
	EventKindDone     EventKind = "done"
)

type StartInput struct {
	ChatID      string
	Prompt      string
	ProjectRoot string
	ParentRunID string
}

type Session struct {
	TaskID string
	RunID  string
}

type ActivityEvent struct {
	Kind     EventKind
	RunID    string
	Sequence int64
	Text     string
	Done     bool
}

type APIClient interface {
	Start(ctx context.Context, input StartInput) (Session, error)
}

type EventSource interface {
	StreamRunEvents(ctx context.Context, runID string, emit func(ActivityEvent) error) error
}

type Service struct {
	client APIClient
	events EventSource
}

func NewService(client APIClient, events EventSource) *Service {
	return &Service{client: client, events: events}
}

func NewServiceFromEnv() (*Service, error) {
	baseURL, err := ResolveBaseURL(context.Background())
	if err != nil {
		return nil, err
	}
	return NewService(
		NewHTTPClient(baseURL, &http.Client{Timeout: 30 * time.Second}),
		NewWebSocketEventSource(baseURL, nil),
	), nil
}

func ResolveBaseURL(ctx context.Context) (string, error) {
	if value := strings.TrimSpace(os.Getenv(apiURLEnv)); value != "" {
		return strings.TrimRight(value, "/"), nil
	}
	if value := strings.TrimSpace(os.Getenv(baseURLEnv)); value != "" {
		return strings.TrimRight(value, "/"), nil
	}
	portCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	baseURL, err := discovery.ResolveScenarioURL(portCtx, defaultScenario, defaultPortKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func (s *Service) Start(ctx context.Context, input StartInput) (Session, error) {
	if s == nil || s.client == nil || s.events == nil {
		return Session{}, ErrUnavailable
	}
	return s.client.Start(ctx, input)
}

func (s *Service) StreamRunEvents(ctx context.Context, runID string, emit func(ActivityEvent) error) error {
	if s == nil || s.events == nil {
		return ErrUnavailable
	}
	return s.events.StreamRunEvents(ctx, runID, emit)
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string, client *http.Client) *HTTPClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *HTTPClient) Start(ctx context.Context, input StartInput) (Session, error) {
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return Session{}, fmt.Errorf("agent prompt is required")
	}
	projectRoot := strings.TrimSpace(input.ProjectRoot)
	if projectRoot == "" {
		projectRoot = defaultProjectRoot()
	}
	if err := c.reconcileProfile(ctx); err != nil {
		return Session{}, err
	}
	taskID, err := c.createTask(ctx, prompt, projectRoot)
	if err != nil {
		return Session{}, err
	}
	runID, err := c.createRun(ctx, taskID, input)
	if err != nil {
		return Session{}, err
	}
	return Session{TaskID: taskID, RunID: runID}, nil
}

func (c *HTTPClient) createTask(ctx context.Context, prompt, projectRoot string) (string, error) {
	var resp struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	err := c.post(ctx, "/api/v1/tasks", map[string]any{
		"task": map[string]any{
			"title":        "Portal agent chat",
			"description":  prompt,
			"scope_path":   projectRoot,
			"project_root": projectRoot,
			"created_by":   "portal",
		},
	}, &resp)
	if err != nil {
		return "", fmt.Errorf("create agent task: %w", err)
	}
	if strings.TrimSpace(resp.Task.ID) == "" {
		return "", fmt.Errorf("agent-manager returned task without id")
	}
	return resp.Task.ID, nil
}

func (c *HTTPClient) createRun(ctx context.Context, taskID string, input StartInput) (string, error) {
	payload := map[string]any{
		"task_id":     taskID,
		"tag":         "portal-" + strings.TrimSpace(input.ChatID),
		"run_mode":    "RUN_MODE_SANDBOXED",
		"profile_ref": map[string]any{"profile_key": "portal/agent-chat"},
		"prompt":      strings.TrimSpace(input.Prompt),
	}
	if parent := strings.TrimSpace(input.ParentRunID); parent != "" {
		payload["parent_run_id"] = parent
	}

	var resp struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	if err := c.post(ctx, "/api/v1/runs", payload, &resp); err != nil {
		return "", fmt.Errorf("create agent run: %w", err)
	}
	if strings.TrimSpace(resp.Run.ID) == "" {
		return "", fmt.Errorf("agent-manager returned run without id")
	}
	return resp.Run.ID, nil
}

func (c *HTTPClient) reconcileProfile(ctx context.Context) error {
	var result struct {
		Results []struct {
			ProfileKey string `json:"profile_key"`
			ProfileID  string `json:"profile_id"`
		} `json:"results"`
	}
	if err := c.post(ctx, "/api/v1/profiles/reconcile-scenario", map[string]any{"scenario": "portal"}, &result); err != nil {
		return fmt.Errorf("reconcile agent profile: %w", err)
	}
	for _, item := range result.Results {
		if item.ProfileKey == "portal/agent-chat" && item.ProfileID != "" {
			return nil
		}
	}
	return fmt.Errorf("reconcile agent profile returned no portal/agent-chat profile")
}

func (c *HTTPClient) post(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func WebSocketURL(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v1/ws")
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported agent-manager URL scheme %q", parsed.Scheme)
	}
	return parsed.String(), nil
}

func defaultProjectRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	wd, err := os.Getwd()
	if err == nil {
		return wd
	}
	return "."
}
