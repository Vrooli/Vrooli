package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
)

const (
	apiURLEnv       = "AGENT_MANAGER_API_URL"
	baseURLEnv      = "AGENT_MANAGER_BASE_URL"
	defaultScenario = "agent-manager"
	defaultPortKey  = "API_PORT"
)

var (
	ErrUnavailable     = errors.New("agent-manager unavailable")
	ErrStopUnconfirmed = errors.New("agent run stop is unconfirmed")
)

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
	AdmissionID   string
	ChatID        string
	Prompt        string
	ProjectRoot   string
	ParentRunID   string
	AttachmentIDs []string
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

// RunState is owner-reported state, not a transport-success acknowledgement.
type RunState struct {
	RunID    string
	Status   string
	Terminal bool
}

type APIClient interface {
	Start(ctx context.Context, input StartInput) (Session, error)
	FindAdmission(context.Context, string) (Session, error)
	Stop(ctx context.Context, runID string) (RunState, error)
	Run(ctx context.Context, runID string) (RunState, error)
}

// AttachmentUploader is implemented by transports that can hand a rendered
// image to agent-manager without putting private pixels in prompt text.
type AttachmentUploader interface {
	UploadAttachment(context.Context, []byte, string, string) (string, error)
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

func (s *Service) Stop(ctx context.Context, runID string) (RunState, error) {
	if s == nil || s.client == nil {
		return RunState{}, ErrUnavailable
	}
	return s.client.Stop(ctx, runID)
}

func (s *Service) Run(ctx context.Context, runID string) (RunState, error) {
	if s == nil || s.client == nil {
		return RunState{}, ErrUnavailable
	}
	return s.client.Run(ctx, runID)
}

func (s *Service) FindAdmission(ctx context.Context, id string) (Session, error) {
	if s == nil || s.client == nil {
		return Session{}, ErrUnavailable
	}
	return s.client.FindAdmission(ctx, id)
}

func admissionTag(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil || parsed.String() != id {
		return "", fmt.Errorf("canonical admission UUID required")
	}
	return "portal-admission-" + id, nil
}

func (c *HTTPClient) FindAdmission(ctx context.Context, id string) (Session, error) {
	tag, err := admissionTag(id)
	if err != nil {
		return Session{}, err
	}
	var response struct {
		Runs []struct {
			ID     string `json:"id"`
			TaskID string `json:"task_id"`
			Tag    string `json:"tag"`
		} `json:"runs"`
		Total   int  `json:"total"`
		HasMore bool `json:"has_more"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/runs?tag_prefix="+url.QueryEscape(tag)+"&limit=2", nil, &response); err != nil {
		return Session{}, err
	}
	if len(response.Runs) != 1 || response.Total != 1 || response.HasMore || response.Runs[0].Tag != tag {
		return Session{}, fmt.Errorf("agent admission has no unique exact run match")
	}
	run := response.Runs[0]
	for _, id := range []string{run.ID, run.TaskID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || parsed.String() != id {
			return Session{}, fmt.Errorf("agent admission returned invalid owner identity")
		}
	}
	return Session{RunID: run.ID, TaskID: run.TaskID}, nil
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
	if _, err := admissionTag(input.AdmissionID); err != nil {
		return Session{}, err
	}
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

func (c *HTTPClient) UploadAttachment(ctx context.Context, content []byte, filename, mime string) (string, error) {
	if len(content) == 0 || len(content) > 32*1024*1024 || strings.TrimSpace(filename) == "" || strings.TrimSpace(mime) == "" {
		return "", fmt.Errorf("invalid agent attachment")
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", err
	}
	if _, err = part.Write(content); err != nil {
		return "", err
	}
	if err = form.Close(); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/attachments/upload", &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := c.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("upload agent attachment: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload agent attachment: status %d", response.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result); err != nil || strings.TrimSpace(result.ID) == "" {
		return "", fmt.Errorf("upload agent attachment: invalid response")
	}
	return result.ID, nil
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
	tag, err := admissionTag(input.AdmissionID)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"task_id":     taskID,
		"tag":         tag,
		"run_mode":    "RUN_MODE_SANDBOXED",
		"profile_ref": map[string]any{"profile_key": "portal/agent-chat"},
		"prompt":      strings.TrimSpace(input.Prompt),
	}
	if len(input.AttachmentIDs) > 0 {
		payload["attachmentIds"] = append([]string(nil), input.AttachmentIDs...)
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

func (c *HTTPClient) Stop(ctx context.Context, runID string) (RunState, error) {
	state, err := c.runOperation(ctx, runID, true)
	if err != nil {
		return state, errors.Join(ErrStopUnconfirmed, err)
	}
	if !state.Terminal {
		return state, ErrStopUnconfirmed
	}
	return state, nil
}

// Run reads authoritative state without repeating a possibly applied Stop.
func (c *HTTPClient) Run(ctx context.Context, runID string) (RunState, error) {
	return c.runOperation(ctx, runID, false)
}

func (c *HTTPClient) runOperation(ctx context.Context, runID string, stop bool) (RunState, error) {
	id, err := uuid.Parse(runID)
	if err != nil || id == uuid.Nil || id.String() != runID {
		return RunState{}, fmt.Errorf("canonical agent run UUID required")
	}
	method, path := http.MethodGet, "/api/v1/runs/"+runID
	if stop {
		method, path = http.MethodPost, path+"/stop"
	}
	var response struct {
		Run struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"run"`
	}
	if err := c.request(ctx, method, path, nil, &response); err != nil {
		return RunState{}, err
	}
	if response.Run.ID != runID {
		return RunState{}, fmt.Errorf("agent-manager returned a different or missing run identity")
	}
	status := normalizeRunStatus(response.Run.Status)
	if status == "" {
		return RunState{}, fmt.Errorf("agent-manager returned no run status")
	}
	return RunState{RunID: runID, Status: status, Terminal: terminalStatus(status)}, nil
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
	return c.request(ctx, http.MethodPost, path, payload, out)
}

func (c *HTTPClient) request(ctx context.Context, method, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
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
