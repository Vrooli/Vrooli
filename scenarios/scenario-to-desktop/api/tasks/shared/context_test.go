package shared

import (
	"scenario-to-desktop-api/pipeline"
	"strings"
	"testing"
)

// --- Test helpers ---

// newTestPipeline creates a pipeline.Status suitable for testing context builders.
func newTestPipeline() *pipeline.Status {
	return &pipeline.Status{
		PipelineID:   "pipe-1",
		ScenarioName: "test-app",
		Status:       pipeline.StatusFailed,
		CurrentStage: "build",
		Error:        "npm ERR! peer dep missing: react@^18\nextra line",
		StageOrder:   []string{"bundle", "preflight", "generate", "build", "smoketest"},
		Stages: map[string]*pipeline.StageResult{
			"bundle": {
				Stage:       "bundle",
				Status:      pipeline.StatusCompleted,
				StartedAt:   1000,
				CompletedAt: 2000,
			},
			"preflight": {
				Stage:       "preflight",
				Status:      pipeline.StatusCompleted,
				StartedAt:   2000,
				CompletedAt: 3000,
			},
			"generate": {
				Stage:       "generate",
				Status:      pipeline.StatusCompleted,
				StartedAt:   3000,
				CompletedAt: 4000,
			},
			"build": {
				Stage:     "build",
				Status:    pipeline.StatusFailed,
				StartedAt: 4000,
				Error:     "npm ERR! peer dep missing: react@^18\nextra line",
				Logs:      []string{"line1", "line2", "npm ERR! peer dep missing"},
			},
		},
		Config: &pipeline.Config{
			ScenarioName: "test-app",
			Platforms:    []string{"linux"},
			Sign:         true,
		},
	}
}

// --- ContainsContext ---

func TestContainsContext(t *testing.T) {
	list := []string{"task-metadata", "error-info", "safety-rules"}

	if !ContainsContext(list, "error-info") {
		t.Error("expected to find error-info")
	}
	if ContainsContext(list, "build-logs") {
		t.Error("expected not to find build-logs")
	}
	if ContainsContext(nil, "anything") {
		t.Error("expected false for nil list")
	}
}

// --- SafeDeref ---

func TestSafeDeref(t *testing.T) {
	s := "hello"
	if got := SafeDeref(&s, "default"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if got := SafeDeref(nil, "default"); got != "default" {
		t.Errorf("got %q, want %q", got, "default")
	}
}

// --- ExtractErrorSummary ---

func TestExtractErrorSummary(t *testing.T) {
	tests := []struct {
		name     string
		err      string
		contains string
	}{
		{"empty error", "", ""},
		{"npm error with newline", "npm ERR! missing peer\nmore details", "npm ERR! missing peer"},
		{"npm error no newline short", "npm ERR! short", "npm ERR! short"},
		{"regular error with newline", "first line\nsecond line", "first line"},
		{"long error truncated", strings.Repeat("x", 200), strings.Repeat("x", 100) + "..."},
		{"short error", "oops", "oops"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ps := &pipeline.Status{Error: tc.err}
			got := ExtractErrorSummary(ps)
			if tc.contains == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.contains) {
				t.Errorf("got %q, want to contain %q", got, tc.contains)
			}
		})
	}
}

// --- BuildTaskMetadataAttachment ---

func TestBuildTaskMetadataAttachment(t *testing.T) {
	ps := newTestPipeline()
	att := BuildTaskMetadataAttachment(ps, "investigate:logs")

	if att.Key != "task-metadata" {
		t.Errorf("Key = %q, want %q", att.Key, "task-metadata")
	}
	if att.Priority != "high" {
		t.Errorf("Priority = %q, want %q", att.Priority, "high")
	}
	if !strings.Contains(att.Content, "pipe-1") {
		t.Error("expected pipeline ID in content")
	}
	if !strings.Contains(att.Content, "test-app") {
		t.Error("expected scenario name in content")
	}
	if !strings.Contains(att.Content, "failed_stage: build") {
		t.Error("expected failed stage in content")
	}
	if !strings.Contains(att.Summary, "failed at build") {
		t.Errorf("expected 'failed at build' in summary, got %q", att.Summary)
	}
}

func TestBuildTaskMetadataAttachment_NoFailedStage(t *testing.T) {
	ps := &pipeline.Status{
		PipelineID:   "pipe-2",
		ScenarioName: "healthy-app",
		Status:       pipeline.StatusRunning,
		Stages:       map[string]*pipeline.StageResult{},
	}
	att := BuildTaskMetadataAttachment(ps, "fix:immediate")

	if strings.Contains(att.Content, "failed_stage") {
		t.Error("expected no failed_stage when no stage failed")
	}
}

// --- BuildErrorInfoAttachment ---

func TestBuildErrorInfoAttachment(t *testing.T) {
	ps := newTestPipeline()
	att := BuildErrorInfoAttachment(ps)

	if att == nil {
		t.Fatal("expected non-nil attachment")
	}
	if att.Key != "error-info" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "Failed Stage: build") {
		t.Error("expected failed stage in content")
	}
	if !strings.Contains(att.Content, "Pipeline Error:") || !strings.Contains(att.Content, "npm ERR!") {
		t.Error("expected pipeline error in content")
	}
}

func TestBuildErrorInfoAttachment_NilWhenNoErrors(t *testing.T) {
	ps := &pipeline.Status{
		Stages: map[string]*pipeline.StageResult{
			"bundle": {Status: pipeline.StatusCompleted},
		},
	}
	att := BuildErrorInfoAttachment(ps)
	if att != nil {
		t.Error("expected nil when no errors")
	}
}

// --- BuildSafetyRulesAttachment ---

func TestBuildSafetyRulesAttachment(t *testing.T) {
	att := BuildSafetyRulesAttachment()
	if att.Key != "safety-rules" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "Max lines") {
		t.Error("expected safety rules content")
	}
}

// --- BuildDiagnosticChecklistAttachment ---

func TestBuildDiagnosticChecklistAttachment(t *testing.T) {
	att := BuildDiagnosticChecklistAttachment()
	if att.Key != "diagnostic-checklist" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "Which stage failed") {
		t.Error("expected checklist questions in content")
	}
}

// --- BuildOutputFormatAttachment ---

func TestBuildOutputFormatAttachment(t *testing.T) {
	t.Run("without autofix", func(t *testing.T) {
		att := BuildOutputFormatAttachment(false)
		if att.Key != "output-format" {
			t.Errorf("Key = %q", att.Key)
		}
		if strings.Contains(att.Content, "applied fixes") {
			t.Error("expected no autofix content when autoFix=false")
		}
	})

	t.Run("with autofix", func(t *testing.T) {
		att := BuildOutputFormatAttachment(true)
		if !strings.Contains(att.Content, "applied fixes") {
			t.Error("expected autofix content when autoFix=true")
		}
	})
}

// --- BuildPipelineConfigAttachment ---

func TestBuildPipelineConfigAttachment(t *testing.T) {
	cfg := &pipeline.Config{
		ScenarioName:  "my-app",
		Platforms:     []string{"linux", "windows"},
		SkipPreflight: true,
		Sign:          true,
	}
	att := BuildPipelineConfigAttachment(cfg)

	if att.Key != "pipeline-config" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "my-app") {
		t.Error("expected scenario name")
	}
	if !strings.Contains(att.Content, "linux, windows") {
		t.Error("expected platforms")
	}
	if !strings.Contains(att.Content, "skip_preflight: true") {
		t.Error("expected skip_preflight")
	}
	if !strings.Contains(att.Content, "sign: true") {
		t.Error("expected sign")
	}
}

// --- BuildPipelineResultsAttachment ---

func TestBuildPipelineResultsAttachment(t *testing.T) {
	ps := newTestPipeline()
	att := BuildPipelineResultsAttachment(ps)

	if att.Key != "pipeline-results" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "FAILED") {
		t.Error("expected FAILED status for build stage")
	}
	if !strings.Contains(att.Content, "completed") {
		t.Error("expected completed for passing stages")
	}
	if !strings.Contains(att.Summary, "Failed at build") {
		t.Errorf("summary = %q, expected failed at build", att.Summary)
	}
}

func TestBuildPipelineResultsAttachment_LongErrorTruncated(t *testing.T) {
	ps := &pipeline.Status{
		Status:     pipeline.StatusFailed,
		StageOrder: []string{"build"},
		Stages: map[string]*pipeline.StageResult{
			"build": {
				Stage:  "build",
				Status: pipeline.StatusFailed,
				Error:  strings.Repeat("e", 300),
			},
		},
	}
	att := BuildPipelineResultsAttachment(ps)
	if !strings.Contains(att.Content, "...") {
		t.Error("expected long error to be truncated with ...")
	}
}

// --- BuildBuildLogsAttachment ---

func TestBuildBuildLogsAttachment(t *testing.T) {
	ps := newTestPipeline()
	att := BuildBuildLogsAttachment(ps)

	if att == nil {
		t.Fatal("expected non-nil")
	}
	if att.Key != "build-logs" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "npm ERR! peer dep missing") {
		t.Error("expected build log content")
	}
}

func TestBuildBuildLogsAttachment_NilWhenNoBuildStage(t *testing.T) {
	ps := &pipeline.Status{
		Stages: map[string]*pipeline.StageResult{},
	}
	if att := BuildBuildLogsAttachment(ps); att != nil {
		t.Error("expected nil when no build stage")
	}
}

func TestBuildBuildLogsAttachment_TruncatesLongLogs(t *testing.T) {
	logs := make([]string, 150)
	for i := range logs {
		logs[i] = "log line"
	}
	ps := &pipeline.Status{
		Stages: map[string]*pipeline.StageResult{
			pipeline.StageBuild: {
				Stage:  pipeline.StageBuild,
				Status: pipeline.StatusCompleted,
				Logs:   logs,
			},
		},
	}
	att := BuildBuildLogsAttachment(ps)
	if att == nil {
		t.Fatal("expected non-nil")
	}
	if !strings.Contains(att.Content, "truncated") {
		t.Error("expected truncation notice for >100 lines")
	}
}

// --- BuildGeneratorConnectionAttachment ---

func TestBuildGeneratorConnectionAttachment(t *testing.T) {
	ps := newTestPipeline()
	ps.FinalArtifacts = map[string]string{"linux": "/path/to/app.AppImage"}

	att := BuildGeneratorConnectionAttachment(ps)
	if att.Key != "generator-connection" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "test-app") {
		t.Error("expected scenario name")
	}
	if !strings.Contains(att.Content, "app.AppImage") {
		t.Error("expected artifact path")
	}
}

// --- BuildUserNoteAttachment ---

func TestBuildUserNoteAttachment(t *testing.T) {
	if att := BuildUserNoteAttachment(""); att != nil {
		t.Error("expected nil for empty note")
	}

	att := BuildUserNoteAttachment("check the icons")
	if att == nil {
		t.Fatal("expected non-nil")
	}
	if att.Content != "check the icons" {
		t.Errorf("Content = %q", att.Content)
	}
}

// --- BuildHarnessFocusAttachment ---

func TestBuildHarnessFocusAttachment(t *testing.T) {
	att := BuildHarnessFocusAttachment()
	if att.Key != "harness-focus" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "scenario-to-desktop") {
		t.Error("expected harness reference")
	}
}

// --- BuildSubjectFocusAttachment ---

func TestBuildSubjectFocusAttachment(t *testing.T) {
	att := BuildSubjectFocusAttachment("my-app")
	if att.Key != "subject-focus" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "my-app") {
		t.Error("expected scenario name in content")
	}
	if !strings.Contains(att.Summary, "my-app") {
		t.Error("expected scenario name in summary")
	}
}

// --- BuildSourceInvestigationAttachment ---

func TestBuildSourceInvestigationAttachment(t *testing.T) {
	if att := BuildSourceInvestigationAttachment(""); att != nil {
		t.Error("expected nil for empty findings")
	}

	att := BuildSourceInvestigationAttachment("root cause: missing dep")
	if att == nil {
		t.Fatal("expected non-nil")
	}
	if att.Key != "source-investigation" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "missing dep") {
		t.Error("expected findings content")
	}
}

// --- BuildStageDetailsAttachment ---

func TestBuildStageDetailsAttachment(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		if att := BuildStageDetailsAttachment("build", nil); att != nil {
			t.Error("expected nil for nil result")
		}
	})

	t.Run("with error and logs", func(t *testing.T) {
		result := &pipeline.StageResult{
			Stage:  "build",
			Status: pipeline.StatusFailed,
			Error:  "compilation failed",
			Logs:   []string{"compiling...", "error: undefined"},
		}
		att := BuildStageDetailsAttachment("build", result)
		if att == nil {
			t.Fatal("expected non-nil")
		}
		if att.Key != "stage-build-details" {
			t.Errorf("Key = %q", att.Key)
		}
		if !strings.Contains(att.Content, "compilation failed") {
			t.Error("expected error in content")
		}
		if !strings.Contains(att.Content, "error: undefined") {
			t.Error("expected log line in content")
		}
	})

	t.Run("truncates long logs", func(t *testing.T) {
		logs := make([]string, 80)
		for i := range logs {
			logs[i] = "line"
		}
		result := &pipeline.StageResult{
			Stage:  "generate",
			Status: pipeline.StatusCompleted,
			Logs:   logs,
		}
		att := BuildStageDetailsAttachment("generate", result)
		if !strings.Contains(att.Content, "truncated") {
			t.Error("expected truncation notice for >50 lines")
		}
	})
}

// --- BuildArchitectureGuideAttachment ---

func TestBuildArchitectureGuideAttachment(t *testing.T) {
	att := BuildArchitectureGuideAttachment()
	if att.Key != "architecture-guide" {
		t.Errorf("Key = %q", att.Key)
	}
	if !strings.Contains(att.Content, "Pipeline Stages") {
		t.Error("expected architecture content")
	}
}

// --- DefaultIncludeContexts ---

func TestDefaultIncludeContexts(t *testing.T) {
	expected := []string{"task-metadata", "error-info", "safety-rules"}
	for _, key := range expected {
		if !ContainsContext(DefaultIncludeContexts, key) {
			t.Errorf("expected %q in DefaultIncludeContexts", key)
		}
	}
}
