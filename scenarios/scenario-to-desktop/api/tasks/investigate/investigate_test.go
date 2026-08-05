package investigate

import (
	"strings"
	"testing"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/shared"
)

// --- Test helpers ---

func newTestInput(effort domain.InvestigationEffort) shared.TaskInput {
	return shared.TaskInput{
		Pipeline: &pipeline.Status{
			PipelineID:   "pipe-1",
			ScenarioName: "test-app",
			Status:       pipeline.StatusFailed,
			CurrentStage: "build",
			Error:        "npm ERR! missing peer dep\nmore details",
			StageOrder:   []string{"bundle", "preflight", "generate", "build", "smoketest"},
			Stages: map[string]*pipeline.StageResult{
				"build": {
					Stage:     "build",
					Status:    pipeline.StatusFailed,
					StartedAt: 4000,
					Error:     "npm ERR! missing peer dep",
					Logs:      []string{"building...", "npm ERR! missing peer dep"},
				},
			},
			Config: &pipeline.Config{
				ScenarioName: "test-app",
				Platforms:    []string{"linux"},
			},
		},
		Request: &domain.CreateTaskRequest{
			PipelineID: "pipe-1",
			TaskType:   domain.TaskTypeInvestigate,
			Focus:      domain.TaskFocus{Harness: true, Subject: true},
			Effort:     effort,
		},
	}
}

// =============================================================================
// BuildPromptAndContext
// =============================================================================

func TestBuildPromptAndContext_Valid(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
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
		t.Errorf("expected 'request is required', got %v", err)
	}
}

func TestBuildPromptAndContext_NilPipeline(t *testing.T) {
	input := shared.TaskInput{Request: &domain.CreateTaskRequest{}}
	_, err := BuildPromptAndContext(input)
	if err == nil || !strings.Contains(err.Error(), "pipeline is required") {
		t.Errorf("expected 'pipeline is required', got %v", err)
	}
}

func TestBuildPromptAndContext_UsesDefaultContexts(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	input.Request.IncludeContexts = nil // should default

	result, err := BuildPromptAndContext(input)
	if err != nil {
		t.Fatal(err)
	}

	keys := make(map[string]bool)
	for _, a := range result.Attachments {
		keys[a.Key] = true
	}

	// Default contexts should include task-metadata and safety-rules
	if !keys["task-metadata"] {
		t.Error("expected task-metadata attachment from defaults")
	}
	if !keys["safety-rules"] {
		t.Error("expected safety-rules attachment from defaults")
	}
}

func TestBuildPromptAndContext_CustomContexts(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	input.Request.IncludeContexts = []string{"task-metadata", "safety-rules"}

	result, err := BuildPromptAndContext(input)
	if err != nil {
		t.Fatal(err)
	}

	keys := make(map[string]bool)
	for _, a := range result.Attachments {
		keys[a.Key] = true
	}

	if !keys["task-metadata"] {
		t.Error("expected task-metadata")
	}
	if !keys["safety-rules"] {
		t.Error("expected safety-rules")
	}
	// pipeline-results not requested, should not appear (unless added via focus)
	if keys["pipeline-results"] {
		t.Error("expected pipeline-results to be excluded when not in custom contexts")
	}
}

// =============================================================================
// buildBasePrompt
// =============================================================================

func TestBuildBasePrompt_IncludesScenarioName(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "test-app") {
		t.Error("expected scenario name")
	}
}

func TestBuildBasePrompt_IncludesErrorSummary(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "npm ERR!") {
		t.Error("expected error summary")
	}
}

func TestBuildBasePrompt_IncludesFailedStage(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "Failed Stage: build") {
		t.Error("expected failed stage")
	}
}

func TestBuildBasePrompt_EffortChecks(t *testing.T) {
	input := newTestInput(domain.EffortChecks)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "Quick checks") {
		t.Error("expected checks mode instructions")
	}
}

func TestBuildBasePrompt_EffortLogs(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "Log analysis") {
		t.Error("expected logs mode instructions")
	}
}

func TestBuildBasePrompt_EffortTrace(t *testing.T) {
	input := newTestInput(domain.EffortTrace)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "Full trace") {
		t.Error("expected trace mode instructions")
	}
}

func TestBuildBasePrompt_FocusAreas(t *testing.T) {
	t.Run("harness only", func(t *testing.T) {
		input := newTestInput(domain.EffortLogs)
		input.Request.Focus = domain.TaskFocus{Harness: true}
		prompt := buildBasePrompt(input)
		if !strings.Contains(prompt, "build pipeline harness") {
			t.Error("expected harness focus")
		}
	})

	t.Run("subject only", func(t *testing.T) {
		input := newTestInput(domain.EffortLogs)
		input.Request.Focus = domain.TaskFocus{Subject: true}
		prompt := buildBasePrompt(input)
		if !strings.Contains(prompt, "target scenario") {
			t.Error("expected subject focus")
		}
	})

	t.Run("both", func(t *testing.T) {
		input := newTestInput(domain.EffortLogs)
		input.Request.Focus = domain.TaskFocus{Harness: true, Subject: true}
		prompt := buildBasePrompt(input)
		if !strings.Contains(prompt, "build pipeline harness") || !strings.Contains(prompt, "target scenario") {
			t.Error("expected both focus areas")
		}
	})
}

func TestBuildBasePrompt_ReportOnly(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	prompt := buildBasePrompt(input)
	if !strings.Contains(prompt, "Report only") {
		t.Error("expected report-only mode")
	}
}

// =============================================================================
// buildAttachments
// =============================================================================

func TestBuildAttachments_DiagnosticChecklist(t *testing.T) {
	t.Run("included for logs effort", func(t *testing.T) {
		input := newTestInput(domain.EffortLogs)
		atts := buildAttachments(input, shared.DefaultIncludeContexts)
		for _, a := range atts {
			if a.Key == "diagnostic-checklist" {
				return
			}
		}
		t.Error("expected diagnostic-checklist for logs effort")
	})

	t.Run("included for trace effort", func(t *testing.T) {
		input := newTestInput(domain.EffortTrace)
		atts := buildAttachments(input, shared.DefaultIncludeContexts)
		for _, a := range atts {
			if a.Key == "diagnostic-checklist" {
				return
			}
		}
		t.Error("expected diagnostic-checklist for trace effort")
	})

	t.Run("excluded for checks effort", func(t *testing.T) {
		input := newTestInput(domain.EffortChecks)
		atts := buildAttachments(input, shared.DefaultIncludeContexts)
		for _, a := range atts {
			if a.Key == "diagnostic-checklist" {
				t.Error("expected no diagnostic-checklist for checks effort")
			}
		}
	})
}

func TestBuildAttachments_ArchitectureGuide(t *testing.T) {
	t.Run("included for trace effort", func(t *testing.T) {
		input := newTestInput(domain.EffortTrace)
		atts := buildAttachments(input, shared.DefaultIncludeContexts)
		for _, a := range atts {
			if a.Key == "architecture-guide" {
				return
			}
		}
		t.Error("expected architecture-guide for trace effort")
	})

	t.Run("excluded for logs effort", func(t *testing.T) {
		input := newTestInput(domain.EffortLogs)
		atts := buildAttachments(input, shared.DefaultIncludeContexts)
		for _, a := range atts {
			if a.Key == "architecture-guide" {
				t.Error("expected no architecture-guide for logs effort")
			}
		}
	})
}

func TestBuildAttachments_FocusAttachments(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	input.Request.Focus = domain.TaskFocus{Harness: true, Subject: true}
	atts := buildAttachments(input, shared.DefaultIncludeContexts)

	keys := make(map[string]bool)
	for _, a := range atts {
		keys[a.Key] = true
	}

	if !keys["harness-focus"] {
		t.Error("expected harness-focus attachment")
	}
	if !keys["subject-focus"] {
		t.Error("expected subject-focus attachment")
	}
}

func TestBuildAttachments_UserNote(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	input.Request.Note = "check preflight"
	atts := buildAttachments(input, shared.DefaultIncludeContexts)

	for _, a := range atts {
		if a.Key == "user-note" {
			if !strings.Contains(a.Content, "check preflight") {
				t.Error("expected note content")
			}
			return
		}
	}
	t.Error("expected user-note attachment")
}

func TestBuildAttachments_BuildLogs(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	atts := buildAttachments(input, shared.DefaultIncludeContexts)

	for _, a := range atts {
		if a.Key == "build-logs" {
			if !strings.Contains(a.Content, "npm ERR!") {
				t.Error("expected build log content")
			}
			return
		}
	}
	t.Error("expected build-logs attachment")
}

func TestBuildAttachments_SkipsExcludedContexts(t *testing.T) {
	input := newTestInput(domain.EffortLogs)
	// Only include task-metadata
	atts := buildAttachments(input, []string{"task-metadata"})

	for _, a := range atts {
		switch a.Key {
		case "task-metadata", "harness-focus", "subject-focus":
			// These are always included (focus attachments aren't context-gated)
		default:
			t.Errorf("unexpected attachment %q when only task-metadata requested", a.Key)
		}
	}
}
