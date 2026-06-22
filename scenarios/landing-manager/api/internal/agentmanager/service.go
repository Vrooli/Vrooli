package agentmanager

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ProfileKey is the agent profile key landing-manager runs against.
// It must match .vrooli/agent-profiles/default.json.
const ProfileKey = "landing-manager/default"

// Service orchestrates landing-page customization runs via agent-manager.
type Service struct {
	client     *Client
	vrooliRoot string
}

// RunRequest contains parameters for a customization run.
type RunRequest struct {
	// ScenarioID is the generated landing-page scenario being customized.
	ScenarioID string
	// Title is a human-readable task title.
	Title string
	// Prompt is the full customization prompt handed to the agent.
	Prompt string
	// Timeout bounds the run; zero leaves it to the profile default.
	Timeout time.Duration
}

// NewService creates a customization run service.
func NewService(timeout time.Duration, vrooliRoot string) *Service {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Service{
		client:     NewClient(timeout),
		vrooliRoot: vrooliRoot,
	}
}

// IsAvailable reports whether agent-manager is reachable.
func (s *Service) IsAvailable(ctx context.Context) bool {
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// ReconcileProfiles registers landing-manager's declared agent profiles
// (.vrooli/agent-profiles/*) idempotently. Safe to call on startup.
func (s *Service) ReconcileProfiles(ctx context.Context) error {
	resp, err := s.client.ReconcileScenarioProfiles(ctx, "landing-manager")
	if err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}
	log.Printf("[agent-manager] Reconciled profiles for %s (created=%d updated=%d unchanged=%d failed=%d)",
		resp.Scenario, resp.Created, resp.Updated, resp.Unchanged, resp.Failed)
	return nil
}

// CreateRun creates a task + run for a customization request and returns the run ID.
func (s *Service) CreateRun(ctx context.Context, req RunRequest) (string, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fmt.Sprintf("Customize landing page scenario: %s", req.ScenarioID)
	}

	tag := fmt.Sprintf("landing-manager-%s", strings.TrimSpace(req.ScenarioID))

	amTask := &domainpb.Task{
		Title:       title,
		Description: fmt.Sprintf("Landing-page customization: %s", req.ScenarioID),
		ScopePath:   s.vrooliRoot,
		ProjectRoot: s.vrooliRoot,
		CreatedBy:   "landing-manager",
	}

	createdTask, err := s.client.CreateTask(ctx, amTask)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: &apipb.ProfileRef{ProfileKey: ProfileKey},
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true,
		Prompt:     proto.String(req.Prompt),
	}
	if req.Timeout > 0 {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{
			Timeout: durationpb.New(req.Timeout),
		}
	}

	// Customizations mutate template-safe areas of a generated scenario. Defer
	// apply at run end (ManualReview) so changes land as pending-review for the
	// operator rather than auto-applying to the working tree.
	if runReq.InlineConfig == nil {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
	}
	runReq.InlineConfig.SandboxConfig = &domainpb.SandboxConfig{ManualReview: true}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}
	return run.Id, nil
}

// GetRun retrieves a run by ID (nil if not found). Used for UI status polling.
func (s *Service) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	return s.client.GetRun(ctx, runID)
}
