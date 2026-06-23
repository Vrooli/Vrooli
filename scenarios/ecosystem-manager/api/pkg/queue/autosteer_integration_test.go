package queue

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/ecosystem-manager/api/pkg/tasks"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestShouldContinueTaskRespectsManualDisable(t *testing.T) {
	integration := AutoSteerIntegration{}
	task := tasks.TaskItem{
		ID:                   "task-123",
		AutoSteerProfileID:   "profile-abc",
		ProcessorAutoRequeue: false,
	}

	shouldContinue, err := integration.ShouldContinueTask(&task, "scenario-name")
	if err != nil {
		t.Fatalf("expected no error when auto-enqueue disabled, got %v", err)
	}
	if shouldContinue {
		t.Fatalf("expected shouldContinue=false when ProcessorAutoRequeue is false")
	}
}

// objectiveProfile builds a valid objective-function profile for these tests.
func objectiveProfile(id, name string, allowed ...string) *autosteer.AutoSteerProfile {
	return &autosteer.AutoSteerProfile{
		ID:   id,
		Name: name,
		Objective: autosteer.Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          autosteer.ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: allowed,
		Budget:        autosteer.Budget{MaxIterations: 10, DiminishingReturnsFloor: 0.02},
		AuditPreset:   "comprehensive",
	}
}

// newTestOrchestrator wires an ExecutionOrchestrator with in-memory fakes: a
// canned test-genie audit (one open standards finding) and a catalog declaring
// each steer skill against the standards dimension.
func newTestOrchestrator(profileRepo *autosteer.MockProfileRepository) (*autosteer.ExecutionOrchestrator, *autosteer.MockExecutionStateRepository) {
	stateRepo := autosteer.NewMockExecutionStateRepository()
	completenessProvider := autosteer.NewMockCompletenessProvider()
	promptEnhancer := autosteer.NewMockPromptEnhancerAPI()

	audit := &findings.Audit{
		ScenarioName: "test-scenario",
		Phases: []findings.AuditPhase{
			{
				Name:   "standards",
				Status: "fail",
				Findings: []findings.AuditFinding{
					{
						Source:   int32(architecturev1.FindingSource_FINDING_SOURCE_STANDARDS),
						Severity: int32(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
						Code:     "STD001",
						StableID: "std-1",
					},
				},
			},
		},
	}
	runner := &findings.FakeRunner{Audits: map[string]*findings.Audit{"test-scenario": audit}}
	catalog := &skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "progress", Dimensions: []string{"standards"}},
		{ID: "test", Dimensions: []string{"standards"}},
		{ID: "screaming-architecture-audit", Dimensions: []string{"standards"}},
	}}

	orchestrator := autosteer.NewExecutionOrchestrator(
		stateRepo,
		profileRepo,
		runner,
		catalog,
		promptEnhancer,
		completenessProvider,
		autosteer.NewTraceStore(nil),
		effectiveness.NewMemoryStore(),
	)
	return orchestrator, stateRepo
}

// TestInitializeAutoSteerResetsOnProfileChange verifies that when a task's
// auto_steer_profile_id changes between executions, the stale execution state
// is deleted and re-initialized with the new profile.
func TestInitializeAutoSteerResetsOnProfileChange(t *testing.T) {
	profileRepo := autosteer.NewMockProfileRepository()
	if err := profileRepo.CreateProfile(objectiveProfile("profile-old", "Old Profile", "progress", "test")); err != nil {
		t.Fatalf("failed to create old profile: %v", err)
	}
	if err := profileRepo.CreateProfile(objectiveProfile("profile-new", "New Profile", "screaming-architecture-audit", "progress")); err != nil {
		t.Fatalf("failed to create new profile: %v", err)
	}

	orchestrator, stateRepo := newTestOrchestrator(profileRepo)
	integration := NewAutoSteerIntegration(orchestrator, "")

	taskID := "task-profile-change"
	scenarioName := "test-scenario"

	taskV1 := &tasks.TaskItem{ID: taskID, Type: "scenario", Operation: "improver", AutoSteerProfileID: "profile-old"}
	if err := integration.InitializeAutoSteer(taskV1, scenarioName); err != nil {
		t.Fatalf("first InitializeAutoSteer failed: %v", err)
	}

	state, err := stateRepo.Get(taskID)
	if err != nil || state == nil {
		t.Fatalf("expected execution state after first init, err=%v", err)
	}
	if state.ProfileID != "profile-old" {
		t.Fatalf("expected ProfileID=profile-old, got %q", state.ProfileID)
	}

	taskV2 := &tasks.TaskItem{ID: taskID, Type: "scenario", Operation: "improver", AutoSteerProfileID: "profile-new"}
	if err := integration.InitializeAutoSteer(taskV2, scenarioName); err != nil {
		t.Fatalf("second InitializeAutoSteer failed: %v", err)
	}

	state, err = stateRepo.Get(taskID)
	if err != nil || state == nil {
		t.Fatalf("expected execution state after profile change, err=%v", err)
	}
	if state.ProfileID != "profile-new" {
		t.Fatalf("expected ProfileID=profile-new after change, got %q", state.ProfileID)
	}
	// A fresh run starts at iteration 1 (the first selected skill).
	if state.Iteration != 1 {
		t.Fatalf("expected Iteration=1 after reset, got %d", state.Iteration)
	}
}

// TestInitializeAutoSteerPreservesStateWhenProfileUnchanged verifies that
// re-initializing with the same profile does NOT reset execution state.
func TestInitializeAutoSteerPreservesStateWhenProfileUnchanged(t *testing.T) {
	profileRepo := autosteer.NewMockProfileRepository()
	if err := profileRepo.CreateProfile(objectiveProfile("profile-same", "Same Profile", "progress")); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	orchestrator, stateRepo := newTestOrchestrator(profileRepo)
	integration := NewAutoSteerIntegration(orchestrator, "")

	taskID := "task-same-profile"
	scenarioName := "test-scenario"
	task := &tasks.TaskItem{ID: taskID, Type: "scenario", Operation: "improver", AutoSteerProfileID: "profile-same"}

	if err := integration.InitializeAutoSteer(task, scenarioName); err != nil {
		t.Fatalf("first InitializeAutoSteer failed: %v", err)
	}

	// Advance the iteration to simulate progress.
	state, _ := stateRepo.Get(taskID)
	state.Iteration = 5
	if err := stateRepo.Save(state); err != nil {
		t.Fatalf("failed to save advanced state: %v", err)
	}

	// Re-initialize with the same profile — should NOT reset.
	if err := integration.InitializeAutoSteer(task, scenarioName); err != nil {
		t.Fatalf("second InitializeAutoSteer failed: %v", err)
	}

	state, _ = stateRepo.Get(taskID)
	if state.Iteration != 5 {
		t.Fatalf("expected Iteration=5 (preserved), got %d", state.Iteration)
	}
}
