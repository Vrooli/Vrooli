package fix

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/shared"
)

// --- Test helpers ---

func newTestInput() shared.TaskInput {
	return shared.TaskInput{
		Pipeline: &pipeline.Status{
			PipelineID:   "pipe-1",
			ScenarioName: "test-app",
			Status:       pipeline.StatusFailed,
			CurrentStage: "build",
			StageOrder:   []string{"bundle", "preflight", "generate", "build", "smoketest"},
			Stages: map[string]*pipeline.StageResult{
				"build": {
					Stage:     "build",
					Status:    pipeline.StatusFailed,
					StartedAt: 4000,
					Error:     "npm ERR! missing peer",
				},
			},
			Config: &pipeline.Config{
				ScenarioName: "test-app",
				Platforms:    []string{"linux"},
			},
		},
		Request: &domain.CreateTaskRequest{
			PipelineID:    "pipe-1",
			TaskType:      domain.TaskTypeFix,
			Focus:         domain.TaskFocus{Harness: true, Subject: true},
			Permissions:   domain.FixPermissions{Immediate: true, Permanent: true},
			MaxIterations: 5,
		},
		Iteration: 1,
	}
}

// =============================================================================
// BuildPromptAndContext
// =============================================================================

func TestBuildPromptAndContext_Valid(t *testing.T) {
	input := newTestInput()
	result, err := BuildPromptAndContext(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if len(result.Attachments) == 0 {
		t.Error("expected attachments")
	}
}

func TestBuildPromptAndContext_NilRequest(t *testing.T) {
	input := shared.TaskInput{Pipeline: &pipeline.Status{}}
	_, err := BuildPromptAndContext(input)
	if err == nil || !strings.Contains(err.Error(), "request is required") {
		t.Errorf("expected 'request is required' error, got %v", err)
	}
}

func TestBuildPromptAndContext_NilPipeline(t *testing.T) {
	input := shared.TaskInput{Request: &domain.CreateTaskRequest{}}
	_, err := BuildPromptAndContext(input)
	if err == nil || !strings.Contains(err.Error(), "pipeline is required") {
		t.Errorf("expected 'pipeline is required' error, got %v", err)
	}
}

// =============================================================================
// buildIterationPrompt
// =============================================================================

func TestBuildIterationPrompt_IncludesHeader(t *testing.T) {
	input := newTestInput()
	input.Iteration = 3
	input.Request.MaxIterations = 7

	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Iteration 3/7") {
		t.Error("expected iteration header")
	}
}

func TestBuildIterationPrompt_IncludesMission(t *testing.T) {
	input := newTestInput()
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "test-app") {
		t.Error("expected scenario name in mission")
	}
}

func TestBuildIterationPrompt_IncludesPreviousIterations(t *testing.T) {
	input := newTestInput()
	input.Iteration = 2
	input.PreviousIterations = []domain.FixIterationRecord{
		{
			Number:           1,
			DiagnosisSummary: "missing peer dependency",
			ChangesSummary:   "added react@18",
			RebuildTriggered: true,
			VerifyResult:     "fail",
			Outcome:          "continue",
		},
	}

	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "missing peer dependency") {
		t.Error("expected previous diagnosis")
	}
	if !strings.Contains(prompt, "added react@18") {
		t.Error("expected previous changes")
	}
}

func TestBuildIterationPrompt_IncludesFailedStage(t *testing.T) {
	input := newTestInput()
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Failed Stage: build") {
		t.Error("expected failed stage")
	}
}

func TestBuildIterationPrompt_IncludesFocusScope(t *testing.T) {
	input := newTestInput()
	input.Request.Focus = domain.TaskFocus{Harness: true, Subject: true}
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Harness") {
		t.Error("expected harness focus")
	}
	if !strings.Contains(prompt, "Subject") {
		t.Error("expected subject focus")
	}
}

func TestBuildIterationPrompt_PermissionsAllowed(t *testing.T) {
	input := newTestInput()
	input.Request.Permissions = domain.FixPermissions{Immediate: true, Permanent: true, Prevention: true}
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Immediate Fixes (ALLOWED)") {
		t.Error("expected immediate allowed")
	}
	if !strings.Contains(prompt, "Permanent Fixes (ALLOWED)") {
		t.Error("expected permanent allowed")
	}
	if !strings.Contains(prompt, "Prevention (ALLOWED)") {
		t.Error("expected prevention allowed")
	}
}

func TestBuildIterationPrompt_PermissionsNotAllowed(t *testing.T) {
	input := newTestInput()
	input.Request.Permissions = domain.FixPermissions{Immediate: true}
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Immediate Fixes (ALLOWED)") {
		t.Error("expected immediate allowed")
	}
	if !strings.Contains(prompt, "Permanent Fixes (NOT ALLOWED)") {
		t.Error("expected permanent not allowed")
	}
	if !strings.Contains(prompt, "Prevention (NOT ALLOWED)") {
		t.Error("expected prevention not allowed")
	}
}

func TestBuildIterationPrompt_SourceFindings(t *testing.T) {
	input := newTestInput()
	findings := "root cause: missing dep"
	input.SourceFindings = &findings
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "root cause: missing dep") {
		t.Error("expected source findings")
	}
}

func TestBuildIterationPrompt_DefaultsMaxIterations(t *testing.T) {
	input := newTestInput()
	input.Request.MaxIterations = 0
	prompt := buildIterationPrompt(input)

	if !strings.Contains(prompt, "Iteration 1/5") {
		t.Error("expected default max iterations of 5")
	}
}

// =============================================================================
// buildAttachments
// =============================================================================

func TestBuildAttachments_AlwaysIncludesCore(t *testing.T) {
	input := newTestInput()
	atts := buildAttachments(input)

	keys := make(map[string]bool)
	for _, a := range atts {
		keys[a.Key] = true
	}

	required := []string{"task-metadata", "safety-rules", "generator-connection", "pipeline-results"}
	for _, key := range required {
		if !keys[key] {
			t.Errorf("expected attachment %q", key)
		}
	}
}

func TestBuildAttachments_IncludesErrorInfo(t *testing.T) {
	input := newTestInput()
	atts := buildAttachments(input)

	for _, a := range atts {
		if a.Key == "error-info" {
			return
		}
	}
	t.Error("expected error-info attachment for failed pipeline")
}

func TestBuildAttachments_ElevatesGeneratorPriority(t *testing.T) {
	input := newTestInput()
	atts := buildAttachments(input)

	for _, a := range atts {
		if a.Key == "generator-connection" {
			if a.Priority != "high" {
				t.Errorf("expected generator-connection priority 'high', got %q", a.Priority)
			}
			return
		}
	}
	t.Error("expected generator-connection attachment")
}

func TestBuildAttachments_IncludesFocusAttachments(t *testing.T) {
	input := newTestInput()
	input.Request.Focus = domain.TaskFocus{Harness: true, Subject: true}
	atts := buildAttachments(input)

	keys := make(map[string]bool)
	for _, a := range atts {
		keys[a.Key] = true
	}

	if !keys["focus-harness-fix"] {
		t.Error("expected harness fix focus attachment")
	}
	if !keys["focus-subject-fix"] {
		t.Error("expected subject fix focus attachment")
	}
}

func TestBuildAttachments_IncludesIterationState(t *testing.T) {
	input := newTestInput()
	input.PreviousIterations = []domain.FixIterationRecord{
		{Number: 1, Outcome: "continue"},
	}
	atts := buildAttachments(input)

	for _, a := range atts {
		if a.Key == "iteration-state" {
			return
		}
	}
	t.Error("expected iteration-state attachment when previous iterations exist")
}

func TestBuildAttachments_NoIterationStateOnFirstIteration(t *testing.T) {
	input := newTestInput()
	input.PreviousIterations = nil
	atts := buildAttachments(input)

	for _, a := range atts {
		if a.Key == "iteration-state" {
			t.Error("expected no iteration-state on first iteration")
		}
	}
}

func TestBuildAttachments_IncludesUserNote(t *testing.T) {
	input := newTestInput()
	input.Request.Note = "check the icons"
	atts := buildAttachments(input)

	for _, a := range atts {
		if a.Key == "user-note" {
			if !strings.Contains(a.Content, "check the icons") {
				t.Error("expected user note content")
			}
			return
		}
	}
	t.Error("expected user-note attachment")
}

// =============================================================================
// LoopState
// =============================================================================

func TestLoopState_StartIteration(t *testing.T) {
	state := NewLoopState(DefaultLoopConfig(3, "http://localhost:15020"))

	if n := state.StartIteration(); n != 1 {
		t.Errorf("first iteration = %d, want 1", n)
	}
	if n := state.StartIteration(); n != 2 {
		t.Errorf("second iteration = %d, want 2", n)
	}
}

func TestLoopState_ShouldContinue(t *testing.T) {
	t.Run("continues when under max and last outcome is continue", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(3, ""))
		state.StartIteration()
		state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "continue"})

		if !state.ShouldContinue() {
			t.Error("expected ShouldContinue = true")
		}
	})

	t.Run("stops at max iterations", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(2, ""))
		state.StartIteration()
		state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "continue"})
		state.StartIteration()
		state.RecordIteration(domain.FixIterationRecord{Number: 2, Outcome: "continue"})

		if state.ShouldContinue() {
			t.Error("expected ShouldContinue = false at max iterations")
		}
	})

	t.Run("stops when last outcome is success", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		state.StartIteration()
		state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "success"})

		if state.ShouldContinue() {
			t.Error("expected ShouldContinue = false after success")
		}
	})

	t.Run("stops when last outcome is gave_up", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		state.StartIteration()
		state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "gave_up"})

		if state.ShouldContinue() {
			t.Error("expected ShouldContinue = false after gave_up")
		}
	})

	t.Run("continues with no iterations recorded", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		if !state.ShouldContinue() {
			t.Error("expected ShouldContinue = true with no iterations")
		}
	})
}

func TestLoopState_ToFixIterationState(t *testing.T) {
	state := NewLoopState(DefaultLoopConfig(3, ""))
	state.StartIteration()
	state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "continue"})
	state.FinalStatus = "max_iterations"

	result := state.ToFixIterationState()

	if result.CurrentIteration != 1 {
		t.Errorf("CurrentIteration = %d, want 1", result.CurrentIteration)
	}
	if result.MaxIterations != 3 {
		t.Errorf("MaxIterations = %d, want 3", result.MaxIterations)
	}
	if len(result.Iterations) != 1 {
		t.Errorf("len(Iterations) = %d, want 1", len(result.Iterations))
	}
	if result.FinalStatus != "max_iterations" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "max_iterations")
	}
}

func TestLoopState_ToJSON(t *testing.T) {
	state := NewLoopState(DefaultLoopConfig(3, ""))
	state.StartIteration()
	state.RecordIteration(domain.FixIterationRecord{Number: 1, Outcome: "success"})

	data, err := state.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
	if !strings.Contains(string(data), `"current_iteration":1`) {
		t.Error("expected current_iteration in JSON")
	}
}

func TestDefaultLoopConfig(t *testing.T) {
	t.Run("uses provided values", func(t *testing.T) {
		cfg := DefaultLoopConfig(7, "http://api:15020")
		if cfg.MaxIterations != 7 {
			t.Errorf("MaxIterations = %d, want 7", cfg.MaxIterations)
		}
		if cfg.PipelineAPIURL != "http://api:15020" {
			t.Errorf("PipelineAPIURL = %q", cfg.PipelineAPIURL)
		}
	})

	t.Run("defaults max iterations to 5", func(t *testing.T) {
		cfg := DefaultLoopConfig(0, "")
		if cfg.MaxIterations != 5 {
			t.Errorf("MaxIterations = %d, want 5", cfg.MaxIterations)
		}
	})

	t.Run("sets timeouts", func(t *testing.T) {
		cfg := DefaultLoopConfig(1, "")
		if cfg.IterationTimeout != 15*time.Minute {
			t.Errorf("IterationTimeout = %v", cfg.IterationTimeout)
		}
		if cfg.RebuildTimeout != 10*time.Minute {
			t.Errorf("RebuildTimeout = %v", cfg.RebuildTimeout)
		}
	})
}

// =============================================================================
// ParseIterationReport
// =============================================================================

func TestParseIterationReport(t *testing.T) {
	t.Run("valid wrapped report", func(t *testing.T) {
		output := `Some agent output...
{"iteration_report": {"diagnosis": "missing dep", "changes_made": ["added react"], "rebuild_triggered": true, "verification_result": "pass", "outcome": "success", "notes": ""}}
More text`

		report, err := ParseIterationReport(output)
		if err != nil {
			t.Fatal(err)
		}
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if report.Diagnosis != "missing dep" {
			t.Errorf("Diagnosis = %q", report.Diagnosis)
		}
		if report.Outcome != "success" {
			t.Errorf("Outcome = %q", report.Outcome)
		}
		if !report.RebuildTriggered {
			t.Error("expected RebuildTriggered = true")
		}
		if len(report.ChangesMade) != 1 || report.ChangesMade[0] != "added react" {
			t.Errorf("ChangesMade = %v", report.ChangesMade)
		}
	})

	t.Run("no report in output", func(t *testing.T) {
		report, err := ParseIterationReport("just some text without json")
		if err != nil {
			t.Fatal(err)
		}
		if report != nil {
			t.Error("expected nil report for output without JSON")
		}
	})

	t.Run("empty output", func(t *testing.T) {
		report, err := ParseIterationReport("")
		if err != nil {
			t.Fatal(err)
		}
		if report != nil {
			t.Error("expected nil report for empty output")
		}
	})

	t.Run("gave_up outcome", func(t *testing.T) {
		output := `{"iteration_report": {"diagnosis": "unfixable", "changes_made": [], "rebuild_triggered": false, "verification_result": "skip", "outcome": "gave_up", "notes": "beyond scope"}}`

		report, err := ParseIterationReport(output)
		if err != nil {
			t.Fatal(err)
		}
		if report.Outcome != "gave_up" {
			t.Errorf("Outcome = %q, want gave_up", report.Outcome)
		}
	})
}

// =============================================================================
// ExtractIterationRecord
// =============================================================================

func TestExtractIterationRecord(t *testing.T) {
	t.Run("with parseable report", func(t *testing.T) {
		result := &shared.AgentResult{
			RunID:  "run-1",
			Output: `{"iteration_report": {"diagnosis": "dep issue", "changes_made": ["fixed dep", "rebuilt"], "rebuild_triggered": true, "verification_result": "pass", "outcome": "success"}}`,
		}

		record := ExtractIterationRecord(2, result)

		if record.Number != 2 {
			t.Errorf("Number = %d, want 2", record.Number)
		}
		if record.AgentRunID != "run-1" {
			t.Errorf("AgentRunID = %q", record.AgentRunID)
		}
		if record.DiagnosisSummary != "dep issue" {
			t.Errorf("DiagnosisSummary = %q", record.DiagnosisSummary)
		}
		if record.ChangesSummary != "fixed dep; rebuilt" {
			t.Errorf("ChangesSummary = %q", record.ChangesSummary)
		}
		if !record.RebuildTriggered {
			t.Error("expected RebuildTriggered")
		}
		if record.VerifyResult != "pass" {
			t.Errorf("VerifyResult = %q", record.VerifyResult)
		}
		if record.Outcome != "success" {
			t.Errorf("Outcome = %q", record.Outcome)
		}
	})

	t.Run("without parseable report falls back to defaults", func(t *testing.T) {
		result := &shared.AgentResult{
			RunID:  "run-2",
			Output: "no json here",
		}

		record := ExtractIterationRecord(1, result)

		if record.Outcome != "continue" {
			t.Errorf("Outcome = %q, want 'continue'", record.Outcome)
		}
		if record.VerifyResult != "skip" {
			t.Errorf("VerifyResult = %q, want 'skip'", record.VerifyResult)
		}
	})
}

// =============================================================================
// Handler.ShouldContinue
// =============================================================================

func TestHandler_ShouldContinue(t *testing.T) {
	h := NewHandler()

	t.Run("nil result", func(t *testing.T) {
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, nil)
		if cont {
			t.Error("expected false for nil result")
		}
	})

	t.Run("empty output", func(t *testing.T) {
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, &shared.AgentResult{})
		if cont {
			t.Error("expected false for empty output")
		}
	})

	t.Run("success outcome", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "success"}}`,
		}
		cont, reason := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if cont {
			t.Error("expected false for success")
		}
		if !strings.Contains(reason, "successful") {
			t.Errorf("reason = %q", reason)
		}
	})

	t.Run("gave_up outcome", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "gave_up"}}`,
		}
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if cont {
			t.Error("expected false for gave_up")
		}
	})

	t.Run("continue outcome", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "continue"}}`,
		}
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if !cont {
			t.Error("expected true for continue")
		}
	})

	t.Run("unparseable output with success keywords", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: "Build successful! All tests pass.",
		}
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if cont {
			t.Error("expected false when success keywords found")
		}
	})

	t.Run("unparseable output continues", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: "Working on the issue, need more time.",
		}
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if !cont {
			t.Error("expected true when output is unparseable without success keywords")
		}
	})

	t.Run("unknown outcome with verification pass", func(t *testing.T) {
		result := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "unknown", "verification_result": "pass"}}`,
		}
		cont, _ := h.ShouldContinue(context.Background(), &domain.Investigation{}, result)
		if cont {
			t.Error("expected false when verification passed")
		}
	})
}

func TestHandler_TaskType(t *testing.T) {
	h := NewHandler()
	if h.TaskType() != domain.TaskTypeFix {
		t.Errorf("TaskType = %q, want %q", h.TaskType(), domain.TaskTypeFix)
	}
}

func TestHandler_AgentTag(t *testing.T) {
	h := NewHandler()
	if h.AgentTag() != "scenario-to-desktop-fix" {
		t.Errorf("AgentTag = %q", h.AgentTag())
	}
}

// =============================================================================
// TerminationReason
// =============================================================================

func TestTerminationReason(t *testing.T) {
	tests := []struct {
		status   string
		iter     int
		max      int
		contains string
	}{
		{shared.FixStatusSuccess, 2, 5, "successful after 2"},
		{shared.FixStatusMaxIterations, 5, 5, "maximum iterations (5)"},
		{shared.FixStatusAgentGaveUp, 3, 5, "cannot be fixed after 3"},
		{shared.FixStatusUserStopped, 1, 5, "stopped by user"},
		{shared.FixStatusTimeout, 2, 5, "timed out after 2"},
		{"weird", 1, 5, "weird"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			reason := TerminationReason(tc.status, tc.iter, tc.max)
			if !strings.Contains(reason, tc.contains) {
				t.Errorf("reason = %q, want to contain %q", reason, tc.contains)
			}
		})
	}
}

// =============================================================================
// DetermineOutcome
// =============================================================================

func TestDetermineOutcome(t *testing.T) {
	t.Run("artifacts exist means success", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		result := DetermineOutcome(state, nil, true)
		if result != shared.FixStatusSuccess {
			t.Errorf("got %q, want %q", result, shared.FixStatusSuccess)
		}
	})

	t.Run("agent reports success", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		agentResult := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "success"}}`,
		}
		result := DetermineOutcome(state, agentResult, false)
		if result != shared.FixStatusSuccess {
			t.Errorf("got %q, want %q", result, shared.FixStatusSuccess)
		}
	})

	t.Run("agent gave up", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		agentResult := &shared.AgentResult{
			Output: `{"iteration_report": {"outcome": "gave_up"}}`,
		}
		result := DetermineOutcome(state, agentResult, false)
		if result != shared.FixStatusAgentGaveUp {
			t.Errorf("got %q, want %q", result, shared.FixStatusAgentGaveUp)
		}
	})

	t.Run("max iterations reached", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(2, ""))
		state.StartIteration()
		state.StartIteration()
		result := DetermineOutcome(state, nil, false)
		if result != shared.FixStatusMaxIterations {
			t.Errorf("got %q, want %q", result, shared.FixStatusMaxIterations)
		}
	})

	t.Run("still running returns empty", func(t *testing.T) {
		state := NewLoopState(DefaultLoopConfig(5, ""))
		result := DetermineOutcome(state, nil, false)
		if result != "" {
			t.Errorf("got %q, want empty", result)
		}
	})
}
