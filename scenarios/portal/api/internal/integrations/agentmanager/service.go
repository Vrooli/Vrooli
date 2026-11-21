package agentmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
		NewConnectClient(baseURL, &http.Client{Timeout: 30 * time.Second}),
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
		return "", fmt.Errorf("%w: %s", ErrUnavailable, redactCreateCaller(ctx, err.Error()))
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
