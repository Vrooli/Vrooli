package agentmanager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// AgentService is the narrow Agent Manager adapter Test Genie needs for
// evidence-bound remediation. Agent Manager owns execution policy.
type AgentService struct {
	client     *Client
	profileKey string
	profileID  string
	mu         sync.RWMutex
	enabled    bool
}

type Config struct {
	ProfileKey string
	Timeout    time.Duration
	Enabled    bool
}

func NewAgentService(cfg Config) *AgentService {
	return &AgentService{client: NewClient(cfg.Timeout), profileKey: cfg.ProfileKey, enabled: cfg.Enabled}
}

func (s *AgentService) IsEnabled() bool { return s.enabled }

func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if !s.enabled {
		return false
	}
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// Initialize reconciles the manifest-owned profile reference once at startup.
func (s *AgentService) Initialize(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	response, err := s.client.ReconcileScenarioProfiles(ctx, "test-genie")
	if err != nil {
		return fmt.Errorf("reconcile profile: %w", err)
	}
	s.mu.Lock()
	for _, item := range response.Results {
		if item.ProfileKey == s.profileKey {
			s.profileID = item.ProfileId
		}
	}
	profileID := s.profileID
	s.mu.Unlock()
	if profileID == "" {
		return fmt.Errorf("reconciliation returned no profile %q", s.profileKey)
	}
	return nil
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	return &apipb.ProfileRef{ProfileKey: s.profileKey}
}
func (s *AgentService) GetProfileID() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.profileID }

func (s *AgentService) GetRoleCatalog(ctx context.Context) (*apipb.GetRolePolicyCatalogResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRolePolicyCatalog(ctx)
}

func (s *AgentService) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRun(ctx, runID)
}

func (s *AgentService) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRunByTag(ctx, tag)
}

func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return fmt.Errorf("agent-manager not enabled")
	}
	return s.client.StopRun(ctx, runID)
}

type SpawnSingleResult struct {
	TaskID string
	RunID  string
	Tag    string
	Status string
	Error  string
}
type RemediationSpawnRequest struct {
	Task           *domainpb.Task
	Tag            string
	RoleRef        string
	IdempotencyKey string
}

// SpawnRemediation creates exactly one server-built evidence task. The only
// portable caller choice is roleRef; no runner, sandbox, tool, network, or
// review settings cross this boundary.
func (s *AgentService) SpawnRemediation(ctx context.Context, req RemediationSpawnRequest) (*SpawnSingleResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	if req.Task == nil || strings.TrimSpace(req.Task.ScopePath) == "" {
		return nil, fmt.Errorf("remediation task with repo-relative scope_path is required")
	}
	if strings.TrimSpace(req.RoleRef) == "" {
		return nil, fmt.Errorf("remediation role reference is required")
	}
	result := &SpawnSingleResult{Tag: req.Tag, Status: "pending"}
	if existing, err := s.client.GetRunByTag(ctx, req.Tag); err != nil {
		return result, fmt.Errorf("find remediation run by tag: %w", err)
	} else if existing != nil {
		result.TaskID, result.RunID, result.Status = existing.GetTaskId(), existing.GetId(), MapRunStatus(existing.GetStatus())
		return result, nil
	}
	task, err := s.client.CreateTask(ctx, req.Task)
	if err != nil {
		result.Status, result.Error = "failed", fmt.Sprintf("create task: %v", err)
		return result, err
	}
	result.TaskID = task.Id
	runRequest := &apipb.CreateRunRequest{TaskId: task.Id, ProfileRef: s.defaultProfileRef(), Tag: &req.Tag, Force: true, InlineConfig: &domainpb.RunConfigOverrides{RoleRef: &req.RoleRef}}
	if strings.TrimSpace(req.IdempotencyKey) != "" {
		runRequest.IdempotencyKey = &req.IdempotencyKey
	}
	run, err := s.client.CreateRun(ctx, runRequest)
	if err != nil {
		result.Status, result.Error = "failed", fmt.Sprintf("create run: %v", err)
		return result, err
	}
	result.RunID, result.Status = run.Id, MapRunStatus(run.Status)
	return result, nil
}

func MapRunStatus(status domainpb.RunStatus) string {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING, domainpb.RunStatus_RUN_STATUS_STARTING:
		return "pending"
	case domainpb.RunStatus_RUN_STATUS_RUNNING, domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return "running"
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return "completed"
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return "stopped"
	default:
		return "unknown"
	}
}

func MapStatusToRun(status string) domainpb.RunStatus {
	switch status {
	case "pending":
		return domainpb.RunStatus_RUN_STATUS_PENDING
	case "running":
		return domainpb.RunStatus_RUN_STATUS_RUNNING
	case "completed":
		return domainpb.RunStatus_RUN_STATUS_COMPLETE
	case "failed":
		return domainpb.RunStatus_RUN_STATUS_FAILED
	case "stopped":
		return domainpb.RunStatus_RUN_STATUS_CANCELLED
	default:
		return domainpb.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}
