package orchestration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// Independent qualification fixture: all persistence and sandbox mutations
// are counted; a closed dispatcher makes native execution impossible.
type auth01DelegationRuns struct {
	*auth01Runs
	created *domain.Run
}

func (r *auth01DelegationRuns) Create(_ context.Context, run *domain.Run) error {
	r.effects.runWrites++
	r.created = run
	return nil
}

type auth01DelegationTasks struct {
	*auth01Tasks
	task *domain.Task
}

func (r *auth01DelegationTasks) Get(_ context.Context, id uuid.UUID) (*domain.Task, error) {
	if id != r.task.ID {
		return nil, nil
	}
	copy := *r.task
	return &copy, nil
}

type auth01DelegationProfiles struct {
	repository.ProfileRepository
	writes int
}

func (r *auth01DelegationProfiles) Create(context.Context, *domain.AgentProfile) error {
	r.writes++
	return errors.New("unexpected profile write")
}
func (r *auth01DelegationProfiles) Update(context.Context, *domain.AgentProfile) error {
	r.writes++
	return errors.New("unexpected profile write")
}

type auth01DelegationSandbox struct {
	sandbox.Provider
	writes   int
	retained *sandbox.Sandbox
}

func (s *auth01DelegationSandbox) Get(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	if s.retained == nil || s.retained.ID != id {
		return nil, errors.New("unexpected sandbox read")
	}
	copy := *s.retained
	return &copy, nil
}

func (s *auth01DelegationSandbox) Create(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	s.writes++
	return nil, errors.New("unexpected sandbox create")
}
func (s *auth01DelegationSandbox) Start(context.Context, uuid.UUID) error {
	s.writes++
	return errors.New("unexpected sandbox start")
}

type auth01DelegationIdempotency struct{ *auth01Idempotency }

func (r *auth01DelegationIdempotency) Reserve(context.Context, string, time.Duration) (*domain.IdempotencyRecord, error) {
	r.effects.reservations++
	return &domain.IdempotencyRecord{}, nil
}

func TestAuth01IndependentDelegationEffectsAndExactParentAdmission(t *testing.T) {
	for _, name := range []string{"restricted-child-missing-scope", "restricted-child-stopped-sandbox", "exact-parent-reaches-closed-dispatcher"} {
		restricted := strings.HasPrefix(name, "restricted-child")
		t.Run(name, func(t *testing.T) {
			o, baseRuns, effects, req, token, _ := auth01CallerFixture(t)
			root := t.TempDir()
			missing := filepath.Join(root, "missing-child")
			tasks := &auth01DelegationTasks{auth01Tasks: &auth01Tasks{effects: effects}, task: &domain.Task{ID: req.TaskID, Title: "isolated child", ScopePath: "missing-child", ProjectRoot: root}}
			runs := &auth01DelegationRuns{auth01Runs: baseRuns}
			profiles := &auth01DelegationProfiles{}
			sandboxes := &auth01DelegationSandbox{}
			dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
			dispatcher.Close()
			o.runs, o.tasks, o.profiles, o.sandbox = runs, tasks, profiles, sandboxes
			o.idempotency = &auth01DelegationIdempotency{auth01Idempotency: &auth01Idempotency{effects: effects}}
			o.dispatcher = dispatcher
			o.config = DefaultConfig()
			o.config.MaxConcurrentRuns = 0
			o.config.DefaultProjectRoot = root
			newCurrentModelPolicyFixtureOption(t)(o)
			role := "code.default"
			req.RoleRef = &role
			effort := domain.EffortMedium
			req.Effort = &effort
			req.OwnerToken = ""
			req.RunIdentityToken = token
			req.ParentRunID = &baseRuns.parent.ID
			if name == "restricted-child-stopped-sandbox" {
				id := uuid.New()
				req.ExistingSandboxID = &id
				sandboxes.retained = &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusStopped, ProjectRoot: root, ScopePath: "missing-child", WorkDir: missing}
			}
			cfg, profile, err := o.resolveRunConfig(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			cfg.SandboxConfig, err = o.resolveSandboxConfig(req, profile)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Admission = buildRunAdmission(req, cfg)
			cfg.Admission.Receipt = &domain.QualificationReceipt{Route: role, EffectiveRunner: string(cfg.RunnerType), EffectiveModel: cfg.Model, EffectiveEffort: string(cfg.Effort), RuntimeVersion: "fixture-only/1", AcceptedOutput: true}
			baseRuns.parent.ResolvedConfig = cfg
			if restricted {
				cfg.NetworkAccess = domain.NetworkAccessNone
			}
			got, err := o.CreateRun(context.Background(), req)
			if restricted {
				if got != nil || err == nil || !strings.Contains(err.Error(), "networkAccess") {
					t.Fatalf("wanted dependent network refusal, got run=%v err=%v", got, err)
				}
				assertAuth01NoEffects(t, effects)
				if profiles.writes != 0 || sandboxes.writes != 0 {
					t.Fatalf("profile=%d sandbox=%d effects", profiles.writes, sandboxes.writes)
				}
				if _, err := os.Stat(missing); !os.IsNotExist(err) {
					t.Fatalf("refused child created directory: %v", err)
				}
				if dispatcher.Stats().ActiveCount != 0 {
					t.Fatal("refused child reached dispatcher")
				}
				return
			}
			if got != nil || !errors.Is(err, spawn.ErrDispatcherClosed) {
				t.Fatalf("wanted closed disposable dispatcher, got run=%v err=%v", got, err)
			}
			if effects.reservations != 1 || effects.runWrites != 1 || effects.taskWrites != 0 || profiles.writes != 0 || sandboxes.writes != 0 {
				t.Fatalf("unexpected admission effects: %+v profile=%d sandbox=%d", effects, profiles.writes, sandboxes.writes)
			}
			created := runs.created
			if created == nil || created.ParentRunID == nil || *created.ParentRunID != baseRuns.parent.ID || created.ResolvedConfig.Admission.CreateCaller == nil || created.ResolvedConfig.Admission.CreateCaller.RunID != baseRuns.parent.ID || created.OwnerSubject != "owner-a" {
				t.Fatal("exact-parent admission lost durable caller attribution")
			}
			if info, err := os.Stat(missing); err != nil || !info.IsDir() {
				t.Fatalf("accepted admission did not prepare scope: %v", err)
			}
		})
	}
}
