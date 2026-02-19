package queue

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
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

// newTestOrchestrator creates an ExecutionOrchestrator backed by in-memory mocks.
func newTestOrchestrator(profileRepo *autosteer.MockProfileRepository) (*autosteer.ExecutionOrchestrator, *autosteer.MockExecutionStateRepository) {
	stateRepo := autosteer.NewMockExecutionStateRepository()
	metricsProvider := autosteer.NewMockMetricsProvider()
	phaseCoord := autosteer.NewMockPhaseCoordinatorAPI()
	iterEval := autosteer.NewMockIterationEvaluatorAPI()
	promptEnhancer := autosteer.NewMockPromptEnhancerAPI()

	orchestrator := autosteer.NewExecutionOrchestrator(
		stateRepo,
		phaseCoord,
		iterEval,
		profileRepo,
		metricsProvider,
		promptEnhancer,
	)
	return orchestrator, stateRepo
}

// TestInitializeAutoSteerResetsOnProfileChange verifies that when a task's
// auto_steer_profile_id changes between executions, the stale execution state
// is deleted and re-initialized with the new profile.
//
// Regression test for: phase advancement not triggering because execution state
// retained the old profile's phase/iteration data after the user switched profiles.
func TestInitializeAutoSteerResetsOnProfileChange(t *testing.T) {
	profileRepo := autosteer.NewMockProfileRepository()

	profileOld := &autosteer.AutoSteerProfile{
		ID:   "profile-old",
		Name: "Old Profile",
		Phases: []autosteer.SteerPhase{
			{ID: "p1", SkillID: "progress", SkillName: "Progress", MaxIterations: 10},
			{ID: "p2", SkillID: "test", SkillName: "Test", MaxIterations: 5},
		},
	}
	profileNew := &autosteer.AutoSteerProfile{
		ID:   "profile-new",
		Name: "New Profile",
		Phases: []autosteer.SteerPhase{
			{ID: "n1", SkillID: "screaming-architecture-audit", SkillName: "Screaming Architecture", MaxIterations: 1},
			{ID: "n2", SkillID: "progress", SkillName: "Progress", MaxIterations: 3},
		},
	}

	if err := profileRepo.CreateProfile(profileOld); err != nil {
		t.Fatalf("failed to create old profile: %v", err)
	}
	if err := profileRepo.CreateProfile(profileNew); err != nil {
		t.Fatalf("failed to create new profile: %v", err)
	}

	orchestrator, stateRepo := newTestOrchestrator(profileRepo)
	integration := NewAutoSteerIntegration(orchestrator)

	taskID := "task-profile-change"
	scenarioName := "test-scenario"

	// Step 1: Initialize with the old profile.
	taskV1 := &tasks.TaskItem{
		ID:                 taskID,
		Type:               "scenario",
		Operation:          "improver",
		AutoSteerProfileID: "profile-old",
	}

	if err := integration.InitializeAutoSteer(taskV1, scenarioName); err != nil {
		t.Fatalf("first InitializeAutoSteer failed: %v", err)
	}

	// Verify state was created with the old profile.
	state, err := stateRepo.Get(taskID)
	if err != nil {
		t.Fatalf("failed to get state after init: %v", err)
	}
	if state == nil {
		t.Fatal("expected execution state to exist after first init")
	}
	if state.ProfileID != "profile-old" {
		t.Fatalf("expected ProfileID=%q, got %q", "profile-old", state.ProfileID)
	}

	// Step 2: Simulate user changing the profile on the task.
	taskV2 := &tasks.TaskItem{
		ID:                 taskID,
		Type:               "scenario",
		Operation:          "improver",
		AutoSteerProfileID: "profile-new",
	}

	if err := integration.InitializeAutoSteer(taskV2, scenarioName); err != nil {
		t.Fatalf("second InitializeAutoSteer failed: %v", err)
	}

	// Verify execution state was reset to the new profile.
	state, err = stateRepo.Get(taskID)
	if err != nil {
		t.Fatalf("failed to get state after profile change: %v", err)
	}
	if state == nil {
		t.Fatal("expected execution state to exist after profile change re-init")
	}
	if state.ProfileID != "profile-new" {
		t.Fatalf("expected ProfileID=%q after profile change, got %q", "profile-new", state.ProfileID)
	}
	if state.CurrentPhaseIndex != 0 {
		t.Fatalf("expected CurrentPhaseIndex=0 after reset, got %d", state.CurrentPhaseIndex)
	}
	if state.CurrentPhaseIteration != 0 {
		t.Fatalf("expected CurrentPhaseIteration=0 after reset, got %d", state.CurrentPhaseIteration)
	}
	if state.AutoSteerIteration != 0 {
		t.Fatalf("expected AutoSteerIteration=0 after reset, got %d", state.AutoSteerIteration)
	}
}

// TestInitializeAutoSteerPreservesStateWhenProfileUnchanged verifies that
// re-initializing with the same profile does NOT reset execution state.
func TestInitializeAutoSteerPreservesStateWhenProfileUnchanged(t *testing.T) {
	profileRepo := autosteer.NewMockProfileRepository()

	profile := &autosteer.AutoSteerProfile{
		ID:   "profile-same",
		Name: "Same Profile",
		Phases: []autosteer.SteerPhase{
			{ID: "p1", SkillID: "progress", SkillName: "Progress", MaxIterations: 10},
		},
	}

	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	orchestrator, stateRepo := newTestOrchestrator(profileRepo)
	integration := NewAutoSteerIntegration(orchestrator)

	taskID := "task-same-profile"
	scenarioName := "test-scenario"

	task := &tasks.TaskItem{
		ID:                 taskID,
		Type:               "scenario",
		Operation:          "improver",
		AutoSteerProfileID: "profile-same",
	}

	// First initialization.
	if err := integration.InitializeAutoSteer(task, scenarioName); err != nil {
		t.Fatalf("first InitializeAutoSteer failed: %v", err)
	}

	// Manually advance the state to simulate progress.
	state, _ := stateRepo.Get(taskID)
	state.CurrentPhaseIteration = 5
	state.AutoSteerIteration = 5
	if err := stateRepo.Save(state); err != nil {
		t.Fatalf("failed to save advanced state: %v", err)
	}

	// Re-initialize with the same profile — should NOT reset.
	if err := integration.InitializeAutoSteer(task, scenarioName); err != nil {
		t.Fatalf("second InitializeAutoSteer failed: %v", err)
	}

	state, _ = stateRepo.Get(taskID)
	if state.CurrentPhaseIteration != 5 {
		t.Fatalf("expected CurrentPhaseIteration=5 (preserved), got %d", state.CurrentPhaseIteration)
	}
	if state.AutoSteerIteration != 5 {
		t.Fatalf("expected AutoSteerIteration=5 (preserved), got %d", state.AutoSteerIteration)
	}
}
