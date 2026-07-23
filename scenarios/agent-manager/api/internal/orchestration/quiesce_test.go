package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
)

// quiesceFixture builds an orchestrator and a task scoped to scenarios/<scope>,
// returning the service, the run repository (for seeding runs directly), and the
// task ID.
func quiesceFixture(t *testing.T, scope string) (*orchestration.Orchestrator, repository.RunRepository, uuid.UUID) {
	t.Helper()
	svc, runRepo := newTestOrchestratorWithLimit(t, 50)
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Quiesce fixture task",
		ScopePath: scope,
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)
	return svc, runRepo, task.ID
}

func seedRun(t *testing.T, runRepo repository.RunRepository, taskID uuid.UUID, status domain.RunStatus, tag string) uuid.UUID {
	t.Helper()
	run := &domain.Run{
		ID:        uuid.New(),
		TaskID:    taskID,
		Status:    status,
		Tag:       tag,
		RunMode:   domain.RunModeInPlace,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := runRepo.Create(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	return run.ID
}

// fastQuiesce returns options with short cadence so timeout-driven tests finish
// quickly.
func fastQuiesce(scenario string, force bool) orchestration.QuiesceOptions {
	return orchestration.QuiesceOptions{
		Scenario:     scenario,
		Timeout:      40 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
		Force:        force,
	}
}

func TestQuiesce_NoActiveRuns_DrainsImmediately(t *testing.T) {
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target")
	// A terminal run must not count as in-flight.
	seedRun(t, runRepo, taskID, domain.RunStatusComplete, "")

	res, err := svc.QuiesceScenario(context.Background(), fastQuiesce("quiesce-target", false))
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if !res.Drained || res.Aborted {
		t.Fatalf("expected drained, got %+v", res)
	}
	if res.Initial != 0 {
		t.Fatalf("expected 0 initial in-flight, got %d", res.Initial)
	}
}

func TestQuiesce_AbortsOnTimeout(t *testing.T) {
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target")
	seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	res, err := svc.QuiesceScenario(context.Background(), fastQuiesce("quiesce-target", false))
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if res.Drained {
		t.Fatalf("expected not drained, got %+v", res)
	}
	if !res.Aborted {
		t.Fatalf("expected aborted, got %+v", res)
	}
	if res.Initial != 1 || len(res.InFlight) != 1 {
		t.Fatalf("expected 1 in-flight, got initial=%d inflight=%d", res.Initial, len(res.InFlight))
	}
	if !strings.Contains(res.Reason, "--force") {
		t.Fatalf("abort reason should suggest --force, got %q", res.Reason)
	}
}

func TestQuiesce_ForceCancelsSurvivors(t *testing.T) {
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target")
	runID := seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	res, err := svc.QuiesceScenario(context.Background(), fastQuiesce("quiesce-target", true))
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if !res.Drained {
		t.Fatalf("expected drained after force, got %+v", res)
	}
	if len(res.Cancelled) != 1 || res.Cancelled[0].ID != runID.String() {
		t.Fatalf("expected the run cancelled, got %+v", res.Cancelled)
	}
	// The run must now be in a terminal (cancelled) state.
	got, err := svc.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != domain.RunStatusCancelled {
		t.Fatalf("expected run cancelled, got %s", got.Status)
	}
}

func TestQuiesce_SelfGuardRejectsPromoterInScenario(t *testing.T) {
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target")
	promoterID := seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	opts := fastQuiesce("quiesce-target", false)
	opts.ExcludeRunID = &promoterID
	_, err := svc.QuiesceScenario(context.Background(), opts)
	if err == nil {
		t.Fatal("expected self-deadlock rejection, got nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "deadlock") {
		t.Fatalf("expected deadlock message, got %v", err)
	}
}

func TestQuiesce_ExcludingNonMemberPromoterIsNoOp(t *testing.T) {
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target")
	seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	// The promoter is some external run that does NOT target this scenario.
	external := uuid.New()
	opts := fastQuiesce("quiesce-target", false)
	opts.ExcludeRunID = &external
	res, err := svc.QuiesceScenario(context.Background(), opts)
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	// No rejection; the one real in-flight run is still drained/aborted on.
	if !res.Aborted || res.Initial != 1 {
		t.Fatalf("expected abort with 1 in-flight, got %+v", res)
	}
}

func TestQuiesce_ScopeBoundaryPrecision(t *testing.T) {
	// A run scoped to scenarios/quiesce-target-sibling must NOT be drained when
	// quiescing scenarios/quiesce-target (the SQL LIKE prefix would otherwise
	// match it).
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target-sibling")
	seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	res, err := svc.QuiesceScenario(context.Background(), fastQuiesce("quiesce-target", false))
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if !res.Drained || res.Initial != 0 {
		t.Fatalf("sibling-scoped run must not count; got %+v", res)
	}
}

func TestQuiesce_DescendantScopeIsDrained(t *testing.T) {
	// A run scoped to a subdirectory of the scenario IS in-flight for it.
	svc, runRepo, taskID := quiesceFixture(t, "scenarios/quiesce-target/api")
	seedRun(t, runRepo, taskID, domain.RunStatusRunning, "")

	res, err := svc.QuiesceScenario(context.Background(), fastQuiesce("quiesce-target", false))
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if res.Initial != 1 {
		t.Fatalf("descendant-scoped run should count; got %+v", res)
	}
}

func TestQuiesce_TagPrefixCatchesWholeRepoRuns(t *testing.T) {
	// ecosystem-manager scopes its runs to the repo root, not scenarios/<X>, so
	// scope matching can't find them — the tag prefix does.
	svc, runRepo, taskID := quiesceFixture(t, "/repo-root")
	seedRun(t, runRepo, taskID, domain.RunStatusRunning, "ecosystem-quiesce-target-abc123")

	opts := fastQuiesce("quiesce-target", false)
	opts.TagPrefix = "ecosystem-quiesce-target"
	res, err := svc.QuiesceScenario(context.Background(), opts)
	if err != nil {
		t.Fatalf("QuiesceScenario: %v", err)
	}
	if !res.Aborted || res.Initial != 1 {
		t.Fatalf("tag-matched whole-repo run should count; got %+v", res)
	}
	if len(res.InFlight) != 1 || res.InFlight[0].Tag != "ecosystem-quiesce-target-abc123" {
		t.Fatalf("expected the tagged run in-flight, got %+v", res.InFlight)
	}
}

func TestQuiesce_RequiresScenario(t *testing.T) {
	svc := newTestOrchestrator(t)
	_, err := svc.QuiesceScenario(context.Background(), orchestration.QuiesceOptions{})
	if err == nil {
		t.Fatal("expected validation error for empty scenario")
	}
}
