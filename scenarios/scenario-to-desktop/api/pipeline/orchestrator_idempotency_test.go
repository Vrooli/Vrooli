package pipeline

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Idempotency & Replay Safety Tests [REQ:IDEM-001]
// =============================================================================
// These tests verify that "running twice is no worse than running once" and
// that the system behaves predictably under retries, replays, and repeated execution.

func TestIdempotencyKeyBasic(t *testing.T) {
	// [REQ:IDEM-001] Verify that idempotency key returns existing pipeline instead of creating new one
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	idempotencyKey := "idem-fixture-001"

	// First request
	config1 := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: idempotencyKey,
	}
	status1, err := orchestrator.RunPipeline(ctx, config1)
	if err != nil {
		t.Fatalf("first RunPipeline error: %v", err)
	}

	// Second request with same idempotency key should return the SAME pipeline
	config2 := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: idempotencyKey,
	}
	status2, err := orchestrator.RunPipeline(ctx, config2)
	if err != nil {
		t.Fatalf("second RunPipeline error: %v", err)
	}

	// Should be the exact same pipeline ID
	if status1.PipelineID != status2.PipelineID {
		t.Errorf("expected same pipeline ID for idempotent requests: got %s and %s", status1.PipelineID, status2.PipelineID)
	}

	// Should have only ONE pipeline in the store
	pipelines := store.List()
	if len(pipelines) != 1 {
		t.Errorf("expected exactly 1 pipeline in store, got %d", len(pipelines))
	}
}

func TestIdempotencyKeyStored(t *testing.T) {
	// [REQ:IDEM-001] Verify that idempotency key is stored on the pipeline status
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	idempotencyKey := "stored-key-test"

	config := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: idempotencyKey,
	}
	status, _ := orchestrator.RunPipeline(ctx, config)

	// The idempotency key should be stored on the status
	if status.IdempotencyKey != idempotencyKey {
		t.Errorf("expected IdempotencyKey %q, got %q", idempotencyKey, status.IdempotencyKey)
	}

	// Should be retrievable by idempotency key
	retrieved, ok := store.GetByIdempotencyKey(idempotencyKey)
	if !ok {
		t.Fatalf("expected to retrieve pipeline by idempotency key")
	}
	if retrieved.PipelineID != status.PipelineID {
		t.Errorf("expected pipeline ID %q, got %q", status.PipelineID, retrieved.PipelineID)
	}
}

func TestIdempotencyKeyDifferent(t *testing.T) {
	// [REQ:IDEM-001] Verify that different idempotency keys create different pipelines
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()

	config1 := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: "key-1",
	}
	status1, _ := orchestrator.RunPipeline(ctx, config1)

	config2 := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: "key-2",
	}
	status2, _ := orchestrator.RunPipeline(ctx, config2)

	// Should be different pipelines
	if status1.PipelineID == status2.PipelineID {
		t.Errorf("expected different pipeline IDs for different idempotency keys")
	}

	// Should have 2 pipelines in the store
	pipelines := store.List()
	if len(pipelines) != 2 {
		t.Errorf("expected 2 pipelines in store, got %d", len(pipelines))
	}
}

func TestIdempotencyKeyEmpty(t *testing.T) {
	// [REQ:IDEM-001] Verify that empty idempotency key creates new pipelines each time
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()

	// Two requests without idempotency key
	config := &Config{
		ScenarioName: "test-scenario",
		// No IdempotencyKey
	}

	status1, _ := orchestrator.RunPipeline(ctx, config)
	status2, _ := orchestrator.RunPipeline(ctx, config)

	// Should be different pipelines (default behavior)
	if status1.PipelineID == status2.PipelineID {
		t.Errorf("expected different pipeline IDs without idempotency key")
	}

	// Should have 2 pipelines in the store
	pipelines := store.List()
	if len(pipelines) != 2 {
		t.Errorf("expected 2 pipelines in store, got %d", len(pipelines))
	}
}

func TestIdempotencyKeyWithRunningPipeline(t *testing.T) {
	// [REQ:IDEM-001] Verify that idempotency key returns running pipeline status
	store := NewInMemoryStore()
	executeCh := make(chan struct{})
	slowStage := &mockStage{
		name:        "slow-stage",
		executeTime: 5 * time.Second,
		executeCh:   executeCh,
	}

	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(slowStage),
	)

	ctx := context.Background()
	idempotencyKey := "running-pipeline-key"

	config := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: idempotencyKey,
	}

	// Start first pipeline
	status1, _ := orchestrator.RunPipeline(ctx, config)

	// Wait for it to start executing
	<-executeCh

	// Try to start another with same key while first is running
	status2, _ := orchestrator.RunPipeline(ctx, config)

	// Should return the same pipeline
	if status1.PipelineID != status2.PipelineID {
		t.Errorf("expected same pipeline ID for idempotent request while running")
	}

	// Clean up
	orchestrator.CancelPipeline(status1.PipelineID)
}

func TestIdempotencyKeyWithCompletedPipeline(t *testing.T) {
	// [REQ:IDEM-001] Verify that idempotency key returns completed pipeline status
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	idempotencyKey := "completed-pipeline-key"

	config := &Config{
		ScenarioName:   "test-scenario",
		IdempotencyKey: idempotencyKey,
	}

	// Start first pipeline and wait for its persisted terminal state. A fixed
	// sleep is scheduler-dependent and made this contract test flaky under load.
	status1, _ := orchestrator.RunPipeline(ctx, config)
	deadline := time.Now().Add(2 * time.Second)
	var final1 *Status
	for time.Now().Before(deadline) {
		final1, _ = orchestrator.GetStatus(status1.PipelineID)
		if final1 != nil && final1.Status == StatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if final1.Status != StatusCompleted {
		t.Fatalf("expected first pipeline to complete, got %s", final1.Status)
	}

	// Try to start another with same key after completion
	status2, _ := orchestrator.RunPipeline(ctx, config)

	// Should return the completed pipeline (not start a new one)
	if status1.PipelineID != status2.PipelineID {
		t.Errorf("expected same pipeline ID for idempotent request after completion")
	}

	// Status should still show completed
	if status2.Status != StatusCompleted {
		t.Errorf("expected completed status to be returned, got %s", status2.Status)
	}
}

func TestStoreGetByIdempotencyKeyEmpty(t *testing.T) {
	// [REQ:IDEM-001] Verify that GetByIdempotencyKey returns nil for empty key
	store := NewInMemoryStore()

	// Empty key should return nil, false
	status, ok := store.GetByIdempotencyKey("")
	if ok || status != nil {
		t.Errorf("expected nil, false for empty idempotency key")
	}
}

func TestStoreGetByIdempotencyKeyNotFound(t *testing.T) {
	// [REQ:IDEM-001] Verify that GetByIdempotencyKey returns nil for non-existent key
	store := NewInMemoryStore()

	// Add a pipeline without idempotency key
	store.Save(&Status{
		PipelineID:   "pipeline-1",
		ScenarioName: "test",
		Status:       StatusRunning,
	})

	// Search for non-existent key
	status, ok := store.GetByIdempotencyKey("non-existent-key")
	if ok || status != nil {
		t.Errorf("expected nil, false for non-existent idempotency key")
	}
}

func TestReplayWithSameInputsProducesSameOutput(t *testing.T) {
	// [REQ:IDEM-001] Verify deterministic behavior: same inputs produce same pipeline ID
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()
	idempotencyKey := "deterministic-test"

	// Run multiple times with same config
	for i := 0; i < 5; i++ {
		config := &Config{
			ScenarioName:   "test-scenario",
			IdempotencyKey: idempotencyKey,
			Platforms:      []string{"linux"},
		}
		status, _ := orchestrator.RunPipeline(ctx, config)

		// All should have the same pipeline ID
		if i > 0 {
			first, _ := store.GetByIdempotencyKey(idempotencyKey)
			if status.PipelineID != first.PipelineID {
				t.Errorf("iteration %d: expected pipeline ID %s, got %s", i, first.PipelineID, status.PipelineID)
			}
		}
	}

	// Should still have only one pipeline
	pipelines := store.List()
	if len(pipelines) != 1 {
		t.Errorf("expected 1 pipeline after 5 identical requests, got %d", len(pipelines))
	}
}

func TestNoDuplicateWorkOnRetry(t *testing.T) {
	// [REQ:IDEM-001] Verify that retries don't create duplicate pipelines
	store := NewInMemoryStore()
	orchestrator := NewOrchestrator(
		WithStore(store),
		WithStages(&mockStage{name: "test"}),
	)

	ctx := context.Background()

	// Simulate a client retry scenario: same request sent multiple times
	// (e.g., due to network timeout where client didn't receive response)
	idempotencyKey := "retry-scenario-" + time.Now().Format("20060102150405")

	var pipelineIDs []string
	for i := 0; i < 3; i++ {
		config := &Config{
			ScenarioName:   "test-scenario",
			IdempotencyKey: idempotencyKey,
		}
		status, _ := orchestrator.RunPipeline(ctx, config)
		pipelineIDs = append(pipelineIDs, status.PipelineID)
	}

	// All pipeline IDs should be identical
	for i := 1; i < len(pipelineIDs); i++ {
		if pipelineIDs[i] != pipelineIDs[0] {
			t.Errorf("retry %d returned different pipeline ID: %s vs %s", i, pipelineIDs[i], pipelineIDs[0])
		}
	}

	// Only one pipeline should exist
	if count := len(store.List()); count != 1 {
		t.Errorf("expected 1 pipeline after retries, got %d", count)
	}
}

// =============================================================================
// Stage Filtering Tests
// =============================================================================
// These tests verify that the --stages flag correctly filters which stages run.

func TestRunPipeline_StageFiltering_SingleStage(t *testing.T) {
	// Track which stages were executed
	var executedStages []string
	var mu sync.Mutex

	stage1 := &trackingStage{name: "bundle", executed: &executedStages, mu: &mu}
	stage2 := &trackingStage{name: "preflight", executed: &executedStages, mu: &mu}
	stage3 := &trackingStage{name: "generate", executed: &executedStages, mu: &mu}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2, stage3),
		WithStore(NewInMemoryStore()),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"bundle"}, // Only run bundle
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	final := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	// Only bundle should be executed
	mu.Lock()
	defer mu.Unlock()
	if len(executedStages) != 1 || executedStages[0] != "bundle" {
		t.Errorf("expected only 'bundle' to execute, got %v", executedStages)
	}

	// Stage order should only contain bundle
	if len(final.StageOrder) != 1 || final.StageOrder[0] != "bundle" {
		t.Errorf("expected stage order [bundle], got %v", final.StageOrder)
	}
}

func TestRunPipeline_StageFiltering_MultipleStages(t *testing.T) {
	var executedStages []string
	var mu sync.Mutex

	stage1 := &trackingStage{name: "bundle", executed: &executedStages, mu: &mu}
	stage2 := &trackingStage{name: "preflight", executed: &executedStages, mu: &mu}
	stage3 := &trackingStage{name: "generate", executed: &executedStages, mu: &mu}
	stage4 := &trackingStage{name: "build", executed: &executedStages, mu: &mu}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2, stage3, stage4),
		WithStore(NewInMemoryStore()),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"bundle", "preflight"}, // Only run these two
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	final := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	mu.Lock()
	defer mu.Unlock()
	if len(executedStages) != 2 {
		t.Errorf("expected 2 stages to execute, got %d: %v", len(executedStages), executedStages)
	}
	// Stages should execute in pipeline order
	if len(executedStages) >= 2 && (executedStages[0] != "bundle" || executedStages[1] != "preflight") {
		t.Errorf("expected [bundle, preflight], got %v", executedStages)
	}

	// generate and build should NOT be in stages
	if _, ok := final.Stages["generate"]; ok {
		t.Errorf("expected generate stage to not be started")
	}
	if _, ok := final.Stages["build"]; ok {
		t.Errorf("expected build stage to not be started")
	}
}

func TestRunPipeline_StageFiltering_PreservesPipelineOrder(t *testing.T) {
	// User specifies stages in different order - should still run in pipeline order
	var executedStages []string
	var mu sync.Mutex

	stage1 := &trackingStage{name: "bundle", executed: &executedStages, mu: &mu}
	stage2 := &trackingStage{name: "preflight", executed: &executedStages, mu: &mu}
	stage3 := &trackingStage{name: "generate", executed: &executedStages, mu: &mu}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2, stage3),
		WithStore(NewInMemoryStore()),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"generate", "bundle"}, // User order differs from pipeline order
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	final := waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	mu.Lock()
	defer mu.Unlock()
	// Should execute in pipeline order (bundle before generate), not user order
	if len(executedStages) != 2 {
		t.Errorf("expected 2 stages, got %d: %v", len(executedStages), executedStages)
	}
	if len(executedStages) >= 2 && (executedStages[0] != "bundle" || executedStages[1] != "generate") {
		t.Errorf("expected [bundle, generate] (pipeline order), got %v", executedStages)
	}

	if len(final.StageOrder) != 2 || final.StageOrder[0] != "bundle" || final.StageOrder[1] != "generate" {
		t.Errorf("expected stage order [bundle, generate], got %v", final.StageOrder)
	}
}

func TestRunPipeline_StageFiltering_InvalidStageName(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(&mockStage{name: "bundle"}),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"bundle", "invalid-stage"},
	}

	_, err := orchestrator.RunPipeline(ctx, config)
	if err == nil {
		t.Fatalf("expected error for invalid stage name")
	}
	if !strings.Contains(err.Error(), "invalid stage name") {
		t.Errorf("expected 'invalid stage name' error, got: %v", err)
	}
}

func TestRunPipeline_StageFiltering_EmptyRunsAll(t *testing.T) {
	var executedStages []string
	var mu sync.Mutex

	stage1 := &trackingStage{name: "bundle", executed: &executedStages, mu: &mu}
	stage2 := &trackingStage{name: "preflight", executed: &executedStages, mu: &mu}

	orchestrator := NewOrchestrator(
		WithStages(stage1, stage2),
		WithStore(NewInMemoryStore()),
	)

	ctx := context.Background()
	config := &Config{
		ScenarioName: "test-scenario",
		// Stages not specified - should run all
	}

	status, err := orchestrator.RunPipeline(ctx, config)
	if err != nil {
		t.Fatalf("RunPipeline error: %v", err)
	}

	waitForPipelineTerminal(t, orchestrator, status.PipelineID)

	mu.Lock()
	defer mu.Unlock()
	if len(executedStages) != 2 {
		t.Errorf("expected all 2 stages to execute, got %d: %v", len(executedStages), executedStages)
	}
}

func TestConfig_GetStages(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected []string
	}{
		{"nil config", nil, nil},
		{"empty stages", &Config{Stages: []string{}}, nil},
		{"with single stage", &Config{Stages: []string{"bundle"}}, []string{"bundle"}},
		{"with multiple stages", &Config{Stages: []string{"bundle", "preflight"}}, []string{"bundle", "preflight"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.config.GetStages()
			if tc.expected == nil && got != nil {
				t.Errorf("expected nil, got %v", got)
			}
			if tc.expected != nil {
				if len(got) != len(tc.expected) {
					t.Errorf("expected %v, got %v", tc.expected, got)
				}
				for i := range tc.expected {
					if i < len(got) && got[i] != tc.expected[i] {
						t.Errorf("expected %v, got %v", tc.expected, got)
						break
					}
				}
			}
		})
	}
}

func TestCreateIdlePipeline_StageFiltering(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(
			&mockStage{name: "bundle"},
			&mockStage{name: "preflight"},
			&mockStage{name: "generate"},
		),
		WithStore(NewInMemoryStore()),
	)

	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"bundle", "generate"}, // Skip preflight
	}

	status, err := orchestrator.CreateIdlePipeline(config)
	if err != nil {
		t.Fatalf("CreateIdlePipeline error: %v", err)
	}

	// Stage order should reflect filtered stages
	if len(status.StageOrder) != 2 {
		t.Errorf("expected 2 stages, got %d: %v", len(status.StageOrder), status.StageOrder)
	}
	if len(status.StageOrder) >= 2 && (status.StageOrder[0] != "bundle" || status.StageOrder[1] != "generate") {
		t.Errorf("expected [bundle, generate], got %v", status.StageOrder)
	}
}

func TestCreateIdlePipeline_InvalidStageName(t *testing.T) {
	orchestrator := NewOrchestrator(
		WithStages(&mockStage{name: "bundle"}),
	)

	config := &Config{
		ScenarioName: "test-scenario",
		Stages:       []string{"invalid"},
	}

	_, err := orchestrator.CreateIdlePipeline(config)
	if err == nil {
		t.Fatalf("expected error for invalid stage name")
	}
}
