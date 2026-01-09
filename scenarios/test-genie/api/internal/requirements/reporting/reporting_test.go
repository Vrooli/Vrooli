package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// =============================================================================
// Reporter Tests
// =============================================================================

func TestReporter_New(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("expected non-nil reporter")
	}
}

func TestReporter_Generate_ContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.Generate(ctx, parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestReporter_Generate_JSON(t *testing.T) {
	r := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test Requirement", Status: types.StatusComplete},
		},
	}
	index.AddModule(module)

	summary := enrichment.Summary{
		Total:            1,
		ByDeclaredStatus: map[types.DeclaredStatus]int{types.StatusComplete: 1},
	}

	opts := Options{Format: FormatJSON}
	var buf bytes.Buffer

	err := r.Generate(context.Background(), index, summary, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var data ReportData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("invalid JSON output: %v", err)
	}
	if data.Summary.Total != 1 {
		t.Errorf("expected total 1, got: %d", data.Summary.Total)
	}
}

func TestReporter_Generate_Markdown(t *testing.T) {
	r := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test Requirement", Status: types.StatusComplete},
		},
	}
	index.AddModule(module)

	summary := enrichment.Summary{Total: 1}

	opts := Options{Format: FormatMarkdown, IncludePending: true}
	var buf bytes.Buffer

	err := r.Generate(context.Background(), index, summary, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# Requirements Report") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(output, "REQ-001") {
		t.Error("expected requirement ID in output")
	}
}

func TestReporter_Generate_Trace(t *testing.T) {
	r := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Title:  "Test Requirement",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts", Status: types.ValStatusImplemented},
				},
			},
		},
	}
	index.AddModule(module)

	summary := enrichment.Summary{Total: 1}

	opts := Options{Format: FormatTrace}
	var buf bytes.Buffer

	err := r.Generate(context.Background(), index, summary, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid trace JSON: %v", err)
	}
	if trace.Summary.Total != 1 {
		t.Errorf("expected total 1, got: %d", trace.Summary.Total)
	}
}

func TestReporter_Generate_Summary(t *testing.T) {
	r := New()

	summary := enrichment.Summary{
		Total:            5,
		ByDeclaredStatus: map[types.DeclaredStatus]int{types.StatusComplete: 3, types.StatusInProgress: 2},
		ByLiveStatus:     map[types.LiveStatus]int{types.LivePassed: 3, types.LiveFailed: 1},
	}

	opts := Options{Format: FormatSummary}
	var buf bytes.Buffer

	err := r.Generate(context.Background(), parsing.NewModuleIndex(), summary, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var data SummaryData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("invalid JSON output: %v", err)
	}
	if data.Total != 5 {
		t.Errorf("expected total 5, got: %d", data.Total)
	}
}

func TestReporter_Generate_DefaultFormat(t *testing.T) {
	r := New()

	opts := Options{Format: "unknown-format"}
	var buf bytes.Buffer

	err := r.Generate(context.Background(), parsing.NewModuleIndex(), enrichment.Summary{}, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should default to JSON
	if !strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
		t.Error("expected JSON output for unknown format")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Format != FormatJSON {
		t.Errorf("expected default format JSON, got: %s", opts.Format)
	}
	if !opts.IncludePending {
		t.Error("expected IncludePending to be true")
	}
	if !opts.IncludeValidations {
		t.Error("expected IncludeValidations to be true")
	}
	if opts.Verbose {
		t.Error("expected Verbose to be false")
	}
}

// =============================================================================
// JSONRenderer Tests
// =============================================================================

func TestJSONRenderer_Render_ContextCancellation(t *testing.T) {
	r := NewJSONRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.Render(ctx, parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestJSONRenderer_Render_EmptyIndex(t *testing.T) {
	r := NewJSONRenderer()

	var buf bytes.Buffer
	err := r.Render(context.Background(), parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var data ReportData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
}

func TestJSONRenderer_Render_WithModules(t *testing.T) {
	r := NewJSONRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{
				ID:          "REQ-001",
				Title:       "First Requirement",
				Status:      types.StatusComplete,
				LiveStatus:  types.LivePassed,
				Criticality: types.CriticalityP0,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts", Status: types.ValStatusImplemented},
				},
			},
			{
				ID:     "REQ-002",
				Title:  "Second Requirement",
				Status: types.StatusInProgress,
			},
		},
	}
	index.AddModule(module)

	summary := enrichment.Summary{
		Total: 2,
		ByDeclaredStatus: map[types.DeclaredStatus]int{
			types.StatusComplete:   1,
			types.StatusInProgress: 1,
		},
		ByLiveStatus: map[types.LiveStatus]int{
			types.LivePassed: 1,
		},
		ByCriticality: map[types.Criticality]int{
			types.CriticalityP0: 1,
		},
		ValidationStats: enrichment.ValidationStats{
			Total:       1,
			Implemented: 1,
		},
	}

	opts := Options{
		Format:             FormatJSON,
		IncludePending:     true,
		IncludeValidations: true,
	}

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, summary, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var data ReportData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	if data.Summary.Total != 2 {
		t.Errorf("expected total 2, got: %d", data.Summary.Total)
	}
	if data.Summary.Complete != 1 {
		t.Errorf("expected complete 1, got: %d", data.Summary.Complete)
	}
	if len(data.Modules) != 1 {
		t.Errorf("expected 1 module, got: %d", len(data.Modules))
	}
	if len(data.Modules[0].Requirements) != 2 {
		t.Errorf("expected 2 requirements, got: %d", len(data.Modules[0].Requirements))
	}
}

func TestJSONRenderer_RenderSummary(t *testing.T) {
	r := NewJSONRenderer()

	summary := enrichment.Summary{
		Total:            10,
		CriticalityGap:   2,
		ByDeclaredStatus: map[types.DeclaredStatus]int{"complete": 5, "in_progress": 3, "pending": 2},
		ByLiveStatus:     map[types.LiveStatus]int{"passed": 6, "failed": 2},
	}

	var buf bytes.Buffer
	err := r.RenderSummary(context.Background(), summary, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var data SummaryData
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	if data.Total != 10 {
		t.Errorf("expected total 10, got: %d", data.Total)
	}
	if data.CriticalGap != 2 {
		t.Errorf("expected critical gap 2, got: %d", data.CriticalGap)
	}
}

func TestJSONRenderer_RenderSummary_ContextCancellation(t *testing.T) {
	r := NewJSONRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.RenderSummary(ctx, enrichment.Summary{}, &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestJSONRenderer_RenderRequirements(t *testing.T) {
	r := NewJSONRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test", Status: types.StatusComplete},
		},
	}
	index.AddModule(module)

	opts := Options{IncludeValidations: false}
	var buf bytes.Buffer

	err := r.RenderRequirements(context.Background(), index, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var reqs []RequirementData
	if err := json.Unmarshal(buf.Bytes(), &reqs); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if len(reqs) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(reqs))
	}
}

func TestJSONRenderer_RenderRequirements_ContextCancellation(t *testing.T) {
	r := NewJSONRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.RenderRequirements(ctx, parsing.NewModuleIndex(), DefaultOptions(), &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestJSONRenderer_RenderValidations(t *testing.T) {
	r := NewJSONRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test1.spec.ts", Phase: "unit", Status: types.ValStatusImplemented},
					{Type: types.ValTypeTest, Ref: "test2.spec.ts", Phase: "integration", Status: types.ValStatusPlanned},
				},
			},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.RenderValidations(context.Background(), index, "", &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var validations []struct {
		RequirementID string `json:"requirement_id"`
		Type          string `json:"type"`
	}
	if err := json.Unmarshal(buf.Bytes(), &validations); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if len(validations) != 2 {
		t.Errorf("expected 2 validations, got: %d", len(validations))
	}
}

func TestJSONRenderer_RenderValidations_FilterByPhase(t *testing.T) {
	r := NewJSONRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Phase: "unit", Status: types.ValStatusImplemented},
					{Type: types.ValTypeTest, Phase: "integration", Status: types.ValStatusPlanned},
					{Type: types.ValTypeTest, Phase: "unit", Status: types.ValStatusFailing},
				},
			},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.RenderValidations(context.Background(), index, "unit", &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var validations []struct {
		Phase string `json:"phase"`
	}
	if err := json.Unmarshal(buf.Bytes(), &validations); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if len(validations) != 2 {
		t.Errorf("expected 2 unit validations, got: %d", len(validations))
	}
}

func TestJSONRenderer_RenderValidations_ContextCancellation(t *testing.T) {
	r := NewJSONRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.RenderValidations(ctx, parsing.NewModuleIndex(), "", &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// =============================================================================
// MarkdownRenderer Tests
// =============================================================================

func TestMarkdownRenderer_Render_ContextCancellation(t *testing.T) {
	r := NewMarkdownRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.Render(ctx, parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestMarkdownRenderer_Render_Header(t *testing.T) {
	r := NewMarkdownRenderer()

	var buf bytes.Buffer
	err := r.Render(context.Background(), parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(buf.String(), "# Requirements Report") {
		t.Error("expected markdown header")
	}
}

func TestMarkdownRenderer_Render_Summary(t *testing.T) {
	r := NewMarkdownRenderer()

	summary := enrichment.Summary{
		Total:            10,
		CriticalityGap:   2,
		ByDeclaredStatus: map[types.DeclaredStatus]int{types.StatusComplete: 5, types.StatusInProgress: 3, types.StatusPending: 2},
		ByLiveStatus:     map[types.LiveStatus]int{types.LivePassed: 6, types.LiveFailed: 2},
		ValidationStats:  enrichment.ValidationStats{Implemented: 4, Failing: 1, Planned: 2, NotImplemented: 1},
	}

	var buf bytes.Buffer
	err := r.Render(context.Background(), parsing.NewModuleIndex(), summary, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "## Summary") {
		t.Error("expected summary section")
	}
	if !strings.Contains(output, "| Total Requirements | 10 |") {
		t.Error("expected total requirements row")
	}
	if !strings.Contains(output, "| Complete | 5 |") {
		t.Error("expected complete row")
	}
	if !strings.Contains(output, "### Validations") {
		t.Error("expected validations section")
	}
}

func TestMarkdownRenderer_Render_Modules(t *testing.T) {
	r := NewMarkdownRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test Requirement", Status: types.StatusComplete, LiveStatus: types.LivePassed, Criticality: types.CriticalityP0},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, Options{IncludePending: true}, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "## Modules") {
		t.Error("expected modules section")
	}
	if !strings.Contains(output, "### test-module") {
		t.Error("expected module header")
	}
	if !strings.Contains(output, "REQ-001") {
		t.Error("expected requirement ID")
	}
}

func TestMarkdownRenderer_Render_ExcludePending(t *testing.T) {
	r := NewMarkdownRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete},
			{ID: "REQ-002", Status: types.StatusPending},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, Options{IncludePending: false}, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "REQ-001") {
		t.Error("expected complete requirement")
	}
	if strings.Contains(output, "REQ-002") {
		t.Error("pending requirement should be excluded")
	}
}

func TestMarkdownRenderer_Render_EmptyModule(t *testing.T) {
	r := NewMarkdownRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:     "/test/module.json",
		ModuleName:   "empty-module",
		Requirements: []types.Requirement{},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "_No requirements_") {
		t.Error("expected 'No requirements' message")
	}
}

func TestMarkdownRenderer_Render_WithValidations(t *testing.T) {
	r := NewMarkdownRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts", Phase: "unit", Status: types.ValStatusImplemented},
				},
			},
		},
	}
	index.AddModule(module)

	opts := Options{
		IncludePending:     true,
		IncludeValidations: true,
		Verbose:            true,
	}

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "#### Validations") {
		t.Error("expected validations section")
	}
	if !strings.Contains(output, "test.spec.ts") {
		t.Error("expected validation ref")
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		status   types.DeclaredStatus
		expected string
	}{
		{types.StatusComplete, "Complete"},
		{types.StatusInProgress, "In Progress"},
		{types.StatusPlanned, "Planned"},
		{types.StatusPending, "Pending"},
		{types.StatusNotImplemented, "Not Implemented"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := statusBadge(tt.status)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected badge to contain %q, got: %s", tt.expected, result)
			}
		})
	}
}

func TestLiveStatusBadge(t *testing.T) {
	tests := []struct {
		status   types.LiveStatus
		expected string
	}{
		{types.LivePassed, "Passed"},
		{types.LiveFailed, "Failed"},
		{types.LiveSkipped, "Skipped"},
		{types.LiveNotRun, "Not Run"},
		{types.LiveUnknown, "-"},
		{"", "-"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := liveStatusBadge(tt.status)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected badge to contain %q, got: %s", tt.expected, result)
			}
		})
	}
}

func TestValidationStatusBadge(t *testing.T) {
	tests := []struct {
		status   types.ValidationStatus
		expected string
	}{
		{types.ValStatusImplemented, "Implemented"},
		{types.ValStatusFailing, "Failing"},
		{types.ValStatusPlanned, "Planned"},
		{types.ValStatusNotImplemented, "Not Implemented"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := validationStatusBadge(tt.status)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected badge to contain %q, got: %s", tt.expected, result)
			}
		})
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text|with|pipes", "text\\|with\\|pipes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// TraceRenderer Tests
// =============================================================================

func TestTraceRenderer_Render_ContextCancellation(t *testing.T) {
	r := NewTraceRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := r.Render(ctx, parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestTraceRenderer_Render_EmptyIndex(t *testing.T) {
	r := NewTraceRenderer()

	var buf bytes.Buffer
	err := r.Render(context.Background(), parsing.NewModuleIndex(), enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if trace.Summary.Total != 0 {
		t.Errorf("expected total 0, got: %d", trace.Summary.Total)
	}
}

func TestTraceRenderer_Render_WithRequirements(t *testing.T) {
	r := NewTraceRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{
				ID:          "REQ-001",
				Title:       "Test Requirement",
				Status:      types.StatusComplete,
				LiveStatus:  types.LivePassed,
				Criticality: types.CriticalityP0,
				PRDRef:      "PRD-123",
				Children:    []string{"REQ-002"},
				DependsOn:   []string{"REQ-003"},
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts", Phase: "unit", Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
				},
			},
			{
				ID:     "REQ-002",
				Title:  "Child Requirement",
				Status: types.StatusInProgress,
			},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	if trace.Summary.Total != 2 {
		t.Errorf("expected total 2, got: %d", trace.Summary.Total)
	}
	if trace.Summary.WithValidations != 1 {
		t.Errorf("expected 1 with validations, got: %d", trace.Summary.WithValidations)
	}
	if trace.Summary.WithoutValidations != 1 {
		t.Errorf("expected 1 without validations, got: %d", trace.Summary.WithoutValidations)
	}
	if len(trace.Requirements) != 2 {
		t.Errorf("expected 2 requirements, got: %d", len(trace.Requirements))
	}

	// Check first requirement details
	req := trace.Requirements[0]
	if req.ID != "REQ-001" {
		t.Errorf("expected ID REQ-001, got: %s", req.ID)
	}
	if req.PRDRef != "PRD-123" {
		t.Errorf("expected PRDRef PRD-123, got: %s", req.PRDRef)
	}
	if len(req.Children) != 1 {
		t.Errorf("expected 1 child, got: %d", len(req.Children))
	}
	if len(req.Validations) != 1 {
		t.Errorf("expected 1 validation, got: %d", len(req.Validations))
	}
}

func TestTraceRenderer_Render_TraceCoverage(t *testing.T) {
	r := NewTraceRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Validations: []types.Validation{{Type: types.ValTypeTest}}},
			{ID: "REQ-002", Validations: []types.Validation{{Type: types.ValTypeTest}}},
			{ID: "REQ-003"}, // No validations
			{ID: "REQ-004"}, // No validations
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	// 2 out of 4 have validations = 50% coverage
	if trace.Summary.TraceCoverage != 50 {
		t.Errorf("expected 50%% trace coverage, got: %.1f%%", trace.Summary.TraceCoverage)
	}
}

func TestTraceRenderer_Render_PhaseCoverage(t *testing.T) {
	r := NewTraceRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Phase: "unit", Status: types.ValStatusImplemented},
					{Phase: "unit", Status: types.ValStatusFailing},
					{Phase: "integration", Status: types.ValStatusPlanned},
				},
			},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	unitCoverage := trace.Coverage.ByPhase["unit"]
	if unitCoverage.Total != 2 {
		t.Errorf("expected unit total 2, got: %d", unitCoverage.Total)
	}
	if unitCoverage.Passing != 1 {
		t.Errorf("expected unit passing 1, got: %d", unitCoverage.Passing)
	}
	if unitCoverage.Failing != 1 {
		t.Errorf("expected unit failing 1, got: %d", unitCoverage.Failing)
	}
}

func TestTraceRenderer_Render_CriticalityCoverage(t *testing.T) {
	r := NewTraceRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, Criticality: types.CriticalityP0},
			{ID: "REQ-002", Status: types.StatusInProgress, Criticality: types.CriticalityP0},
			{ID: "REQ-003", Status: types.StatusComplete, Criticality: types.CriticalityP1},
		},
	}
	index.AddModule(module)

	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, DefaultOptions(), &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	p0Coverage := trace.Coverage.ByCriticality["P0"]
	if p0Coverage.Total != 2 {
		t.Errorf("expected P0 total 2, got: %d", p0Coverage.Total)
	}
	if p0Coverage.Complete != 1 {
		t.Errorf("expected P0 complete 1, got: %d", p0Coverage.Complete)
	}
	if p0Coverage.Gap != 1 {
		t.Errorf("expected P0 gap 1, got: %d", p0Coverage.Gap)
	}
}

func TestTraceRenderer_Render_FilterByPhase(t *testing.T) {
	r := NewTraceRenderer()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID: "REQ-001",
				Validations: []types.Validation{
					{Phase: "unit", Status: types.ValStatusImplemented},
					{Phase: "integration", Status: types.ValStatusPlanned},
				},
			},
		},
	}
	index.AddModule(module)

	opts := Options{Phase: "unit"}
	var buf bytes.Buffer
	err := r.Render(context.Background(), index, enrichment.Summary{}, opts, &buf)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var trace TraceReport
	if err := json.Unmarshal(buf.Bytes(), &trace); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	// Should only include unit phase validations
	if len(trace.Requirements[0].Validations) != 1 {
		t.Errorf("expected 1 validation (filtered), got: %d", len(trace.Requirements[0].Validations))
	}
	if trace.Requirements[0].Validations[0].Phase != "unit" {
		t.Errorf("expected unit phase, got: %s", trace.Requirements[0].Validations[0].Phase)
	}
}

// =============================================================================
// BuildReportData Tests
// =============================================================================

func TestBuildReportData_EmptySummary(t *testing.T) {
	data := BuildReportData(nil, enrichment.Summary{}, DefaultOptions())

	if data.Summary.Total != 0 {
		t.Errorf("expected total 0, got: %d", data.Summary.Total)
	}
	if len(data.Modules) != 0 {
		t.Errorf("expected 0 modules, got: %d", len(data.Modules))
	}
}

func TestBuildReportData_WithSummary(t *testing.T) {
	summary := enrichment.Summary{
		Total:          10,
		CriticalityGap: 2,
		ByDeclaredStatus: map[types.DeclaredStatus]int{
			types.StatusComplete:   5,
			types.StatusInProgress: 3,
			types.StatusPending:    2,
		},
		ByLiveStatus: map[types.LiveStatus]int{
			types.LivePassed: 6,
			types.LiveFailed: 2,
		},
		ByCriticality: map[types.Criticality]int{
			types.CriticalityP0: 3,
			types.CriticalityP1: 4,
		},
		ValidationStats: enrichment.ValidationStats{
			Total:          8,
			Implemented:    5,
			Failing:        1,
			Planned:        1,
			NotImplemented: 1,
		},
	}

	data := BuildReportData(nil, summary, DefaultOptions())

	if data.Summary.Total != 10 {
		t.Errorf("expected total 10, got: %d", data.Summary.Total)
	}
	if data.Summary.Complete != 5 {
		t.Errorf("expected complete 5, got: %d", data.Summary.Complete)
	}
	if data.Summary.InProgress != 3 {
		t.Errorf("expected in_progress 3, got: %d", data.Summary.InProgress)
	}
	if data.Summary.LivePassed != 6 {
		t.Errorf("expected live_passed 6, got: %d", data.Summary.LivePassed)
	}
	if data.Summary.CriticalGap != 2 {
		t.Errorf("expected critical_gap 2, got: %d", data.Summary.CriticalGap)
	}
	if data.Summary.CompletionRate != 50 {
		t.Errorf("expected completion_rate 50, got: %.1f", data.Summary.CompletionRate)
	}
	// Pass rate = 6 / (6 + 2) = 75%
	if data.Summary.PassRate != 75 {
		t.Errorf("expected pass_rate 75, got: %.1f", data.Summary.PassRate)
	}
	if data.Statistics.Validations.Total != 8 {
		t.Errorf("expected validation total 8, got: %d", data.Statistics.Validations.Total)
	}
}

func TestBuildReportData_WithModules(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath:   "/test/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Title: "Test", Status: types.StatusComplete, LiveStatus: types.LivePassed},
		},
	}
	index.AddModule(module)

	data := BuildReportData(index, enrichment.Summary{}, Options{IncludePending: true, IncludeValidations: true})

	if len(data.Modules) != 1 {
		t.Errorf("expected 1 module, got: %d", len(data.Modules))
	}
	if data.Modules[0].Name != "test-module" {
		t.Errorf("expected module name 'test-module', got: %s", data.Modules[0].Name)
	}
	if len(data.Modules[0].Requirements) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(data.Modules[0].Requirements))
	}
}

func TestBuildModuleData_ExcludePending(t *testing.T) {
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete},
			{ID: "REQ-002", Status: types.StatusPending},
		},
	}

	data := buildModuleData(module, Options{IncludePending: false})

	if len(data.Requirements) != 1 {
		t.Errorf("expected 1 requirement (pending excluded), got: %d", len(data.Requirements))
	}
}

func TestBuildModuleData_WithValidations(t *testing.T) {
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test.spec.ts", Phase: "unit", Status: types.ValStatusImplemented},
				},
			},
		},
	}

	data := buildModuleData(module, Options{IncludePending: true, IncludeValidations: true})

	if len(data.Requirements[0].Validations) != 1 {
		t.Errorf("expected 1 validation, got: %d", len(data.Requirements[0].Validations))
	}
}

func TestBuildModuleData_FilterValidationsByPhase(t *testing.T) {
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Phase: "unit", Status: types.ValStatusImplemented},
					{Phase: "integration", Status: types.ValStatusPlanned},
				},
			},
		},
	}

	data := buildModuleData(module, Options{IncludePending: true, IncludeValidations: true, Phase: "unit"})

	if len(data.Requirements[0].Validations) != 1 {
		t.Errorf("expected 1 validation (filtered by phase), got: %d", len(data.Requirements[0].Validations))
	}
}
