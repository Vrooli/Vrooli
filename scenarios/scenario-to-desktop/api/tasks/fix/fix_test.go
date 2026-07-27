package fix

import (
	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/shared"
	"strings"
	"testing"
)

// --- Test helpers ---

func newTestInput() shared.TaskInput {
	return shared.TaskInput{
		PipelineAPIURL: "http://127.0.0.1:19001",
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
