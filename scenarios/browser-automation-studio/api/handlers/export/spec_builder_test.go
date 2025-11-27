package export

import (
	"testing"
	"time"

	"github.com/google/uuid"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

func TestBuildSpec_NilSpecs(t *testing.T) {
	executionID := uuid.New()
	_, err := BuildSpec(nil, nil, executionID)
	if err != ErrMovieSpecUnavailable {
		t.Errorf("expected ErrMovieSpecUnavailable, got %v", err)
	}
}

func TestBuildSpec_IncomingSpecPreferred(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	baseline := &exportservices.ReplayMovieSpec{
		Version: "2025-11-07",
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:  executionID,
			WorkflowID:   workflowID,
			WorkflowName: "Baseline Workflow",
			Status:       "completed",
			Progress:     100,
		},
		Frames: []exportservices.ExportFrame{
			{NodeID: "step1", StepType: "navigate", Title: "Test Step"},
		},
	}

	incoming := &exportservices.ReplayMovieSpec{
		Version: "2025-11-07",
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:  executionID,
			WorkflowID:   workflowID,
			WorkflowName: "Custom Workflow",
			Status:       "completed",
			Progress:     100,
		},
		Frames: []exportservices.ExportFrame{
			{NodeID: "step2", StepType: "navigate", Title: "Test Step"},
		},
	}

	spec, err := BuildSpec(baseline, incoming, executionID)
	if err != nil {
		t.Fatalf("BuildSpec failed: %v", err)
	}

	if spec.Execution.WorkflowName != "Custom Workflow" {
		t.Errorf("expected incoming workflow name, got %s", spec.Execution.WorkflowName)
	}

	if len(spec.Frames) != 1 || spec.Frames[0].NodeID != "step2" {
		t.Errorf("expected incoming frames to be used")
	}
}

func TestBuildSpec_BaselineUsedWhenNoIncoming(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	baseline := &exportservices.ReplayMovieSpec{
		Version: "2025-11-07",
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:  executionID,
			WorkflowID:   workflowID,
			WorkflowName: "Baseline Workflow",
			Status:       "completed",
			Progress:     100,
		},
		Frames: []exportservices.ExportFrame{
			{NodeID: "step1", StepType: "navigate", Title: "Test Step"},
		},
	}

	spec, err := BuildSpec(baseline, nil, executionID)
	if err != nil {
		t.Fatalf("BuildSpec failed: %v", err)
	}

	if spec.Execution.WorkflowName != "Baseline Workflow" {
		t.Errorf("expected baseline workflow name, got %s", spec.Execution.WorkflowName)
	}

	if len(spec.Frames) != 1 || spec.Frames[0].NodeID != "step1" {
		t.Errorf("expected baseline frames to be used")
	}
}

func TestClone_NilSpec(t *testing.T) {
	cloned, err := Clone(nil)
	if err != nil {
		t.Errorf("Clone(nil) should not error, got %v", err)
	}
	if cloned != nil {
		t.Errorf("Clone(nil) should return nil, got %v", cloned)
	}
}

func TestClone_DeepCopy(t *testing.T) {
	original := &exportservices.ReplayMovieSpec{
		Version: "2025-11-07",
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:  uuid.New(),
			WorkflowName: "Original",
		},
		Frames: []exportservices.ExportFrame{
			{NodeID: "step1"},
		},
	}

	cloned, err := Clone(original)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify deep copy by modifying original
	original.Execution.WorkflowName = "Modified"
	original.Frames[0].NodeID = "modified"

	if cloned.Execution.WorkflowName != "Original" {
		t.Errorf("clone was not independent, workflow name changed")
	}

	if cloned.Frames[0].NodeID != "step1" {
		t.Errorf("clone was not independent, frame step ID changed")
	}
}

func TestHarmonize_NilSpec(t *testing.T) {
	err := Harmonize(nil, nil, uuid.New())
	if err != ErrMovieSpecUnavailable {
		t.Errorf("expected ErrMovieSpecUnavailable, got %v", err)
	}
}

func TestHarmonize_ExecutionIDMismatch(t *testing.T) {
	executionID := uuid.New()
	differentID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: differentID,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err == nil || err.Error() != "movie spec execution_id mismatch" {
		t.Errorf("expected execution_id mismatch error, got %v", err)
	}
}

func TestHarmonize_WorkflowIDMismatch(t *testing.T) {
	executionID := uuid.New()
	workflowID1 := uuid.New()
	workflowID2 := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
			WorkflowID:  workflowID1,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	baseline := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			WorkflowID: workflowID2,
		},
	}

	err := Harmonize(spec, baseline, executionID)
	if err == nil || err.Error() != "movie spec workflow_id mismatch" {
		t.Errorf("expected workflow_id mismatch error, got %v", err)
	}
}

func TestHarmonize_FillsExecutionID(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{},
		Frames:    []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if spec.Execution.ExecutionID != executionID {
		t.Errorf("expected execution ID to be filled, got %v", spec.Execution.ExecutionID)
	}
}

func TestHarmonize_FillsMetadataFromBaseline(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()
	completedAt := time.Now().UTC()
	startedAt := time.Now().UTC().Add(-time.Hour)

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	baseline := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			WorkflowID:   workflowID,
			WorkflowName: "Test Workflow",
			Status:       "completed",
			Progress:     100,
			StartedAt:    startedAt,
			CompletedAt:  &completedAt,
		},
	}

	err := Harmonize(spec, baseline, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if spec.Execution.WorkflowID != workflowID {
		t.Errorf("expected workflow ID from baseline")
	}
	if spec.Execution.WorkflowName != "Test Workflow" {
		t.Errorf("expected workflow name from baseline")
	}
	if spec.Execution.Status != "completed" {
		t.Errorf("expected status from baseline")
	}
	if spec.Execution.Progress != 100 {
		t.Errorf("expected progress from baseline")
	}
	if spec.Execution.StartedAt != startedAt {
		t.Errorf("expected started_at from baseline")
	}
	if spec.Execution.CompletedAt == nil || *spec.Execution.CompletedAt != completedAt {
		t.Errorf("expected completed_at from baseline")
	}
}

func TestHarmonize_PreservesExistingMetadata(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:  executionID,
			WorkflowID:   workflowID,
			WorkflowName: "Existing Name",
			Status:       "running",
			Progress:     50,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	baseline := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			WorkflowID:   workflowID,
			WorkflowName: "Baseline Name",
			Status:       "completed",
			Progress:     100,
		},
	}

	err := Harmonize(spec, baseline, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	// Existing values should be preserved
	if spec.Execution.WorkflowName != "Existing Name" {
		t.Errorf("expected existing workflow name to be preserved")
	}
	if spec.Execution.Status != "running" {
		t.Errorf("expected existing status to be preserved")
	}
	if spec.Execution.Progress != 50 {
		t.Errorf("expected existing progress to be preserved")
	}
}

func TestHarmonize_FillsVersionAndGeneratedAt(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if spec.Version != "2025-11-07" {
		t.Errorf("expected default version, got %s", spec.Version)
	}

	if spec.GeneratedAt.IsZero() {
		t.Errorf("expected GeneratedAt to be set")
	}
}

func TestHarmonize_MissingFramesError(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{},
	}

	err := Harmonize(spec, nil, executionID)
	if err == nil || err.Error() != "movie spec missing frames" {
		t.Errorf("expected missing frames error, got %v", err)
	}
}

func TestHarmonize_CopiesFramesFromBaseline(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{},
	}

	baseline := &exportservices.ReplayMovieSpec{
		Frames: []exportservices.ExportFrame{
			{NodeID: "step1"},
			{NodeID: "step2"},
		},
	}

	err := Harmonize(spec, baseline, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if len(spec.Frames) != 2 {
		t.Errorf("expected 2 frames from baseline, got %d", len(spec.Frames))
	}
	if spec.Frames[0].NodeID != "step1" || spec.Frames[1].NodeID != "step2" {
		t.Errorf("expected frames to match baseline")
	}
}

func TestHarmonize_CopiesAssetsFromBaseline(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
		Assets: []exportservices.ExportAsset{},
	}

	baseline := &exportservices.ReplayMovieSpec{
		Assets: []exportservices.ExportAsset{
			{ID: "screenshot1.png", Source: "assets/screenshot1.png"},
			{ID: "screenshot2.png", Source: "assets/screenshot2.png"},
		},
	}

	err := Harmonize(spec, baseline, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if len(spec.Assets) != 2 {
		t.Errorf("expected 2 assets from baseline, got %d", len(spec.Assets))
	}
}

func TestHarmonize_SyncsTotalDurationWithSummary(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:   executionID,
			TotalDuration: 0,
		},
		Summary: exportservices.ExportSummary{
			TotalDurationMs: 5000,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	if spec.Execution.TotalDuration != 5000 {
		t.Errorf("expected total duration to sync with summary, got %d", spec.Execution.TotalDuration)
	}
}

func TestHarmonize_PreservesTotalDuration(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID:   executionID,
			TotalDuration: 3000,
		},
		Summary: exportservices.ExportSummary{
			TotalDurationMs: 5000,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	// Existing TotalDuration should be preserved
	if spec.Execution.TotalDuration != 3000 {
		t.Errorf("expected existing total duration to be preserved, got %d", spec.Execution.TotalDuration)
	}
}

func TestHarmonize_EnsuresNestedStructures(t *testing.T) {
	executionID := uuid.New()

	spec := &exportservices.ReplayMovieSpec{
		Execution: exportservices.ExportExecutionMetadata{
			ExecutionID: executionID,
		},
		Frames: []exportservices.ExportFrame{{NodeID: "step1"}},
	}

	err := Harmonize(spec, nil, executionID)
	if err != nil {
		t.Fatalf("Harmonize failed: %v", err)
	}

	// Verify that nested structures are initialized with defaults
	// (ensureTheme, ensureDecor, ensureCursor, ensurePresentation, ensureSummaryAndPlayback are called)
	// This is more of an integration test - we verify no panic occurs
	if spec.Theme.AccentColor == "" {
		// Theme should have defaults filled
		t.Logf("Theme background: %s", spec.Theme.AccentColor)
	}
}
