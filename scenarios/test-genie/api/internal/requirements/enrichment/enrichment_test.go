package enrichment

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// =============================================================================
// Matcher Tests
// =============================================================================

func TestMatcher_FindBestMatch_NilInputs(t *testing.T) {
	m := NewMatcher()

	tests := []struct {
		name     string
		val      *types.Validation
		evidence *types.EvidenceBundle
	}{
		{"nil validation", nil, types.NewEvidenceBundle()},
		{"nil evidence", &types.Validation{}, nil},
		{"both nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.FindBestMatch(tt.val, "REQ-001", tt.evidence)
			if result != nil {
				t.Error("expected nil result for nil input")
			}
		})
	}
}

func TestMatcher_FindBestMatch_ByRef(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
		SourcePath:    "src/components/Button.test.ts",
		UpdatedAt:     time.Now(),
	})

	val := &types.Validation{
		Type: types.ValTypeTest,
		Ref:  "src/components/Button.test.ts",
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != types.LivePassed {
		t.Errorf("expected status passed, got: %s", result.Status)
	}
}

func TestMatcher_FindBestMatch_ByRefNormalized(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LiveFailed,
		Phase:         "unit",
		SourcePath:    "./src/components/Button.test.ts",
	})

	// Validation ref without leading ./
	val := &types.Validation{
		Type: types.ValTypeTest,
		Ref:  "src/components/Button.test.ts",
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match with normalized path")
	}
	if result.Status != types.LiveFailed {
		t.Errorf("expected status failed, got: %s", result.Status)
	}
}

func TestMatcher_FindBestMatch_ByWorkflowID(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "playbooks",
		SourcePath:    "test/playbooks/user-login.yaml",
		Metadata:      map[string]any{"workflow_id": "user-login"},
	})

	val := &types.Validation{
		Type:       types.ValTypeAutomation,
		WorkflowID: "user-login",
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match by workflow ID")
	}
	if result.Status != types.LivePassed {
		t.Errorf("expected status passed, got: %s", result.Status)
	}
}

func TestMatcher_FindBestMatch_ByPhase(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LiveSkipped,
		Phase:         "integration",
	})

	val := &types.Validation{
		Type:  types.ValTypeTest,
		Phase: "integration",
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match by phase")
	}
	if result.Status != types.LiveSkipped {
		t.Errorf("expected status skipped, got: %s", result.Status)
	}
}

func TestMatcher_FindBestMatch_ByRequirementID(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
	})

	val := &types.Validation{
		Type: types.ValTypeTest,
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match by requirement ID")
	}
}

func TestMatcher_FindBestMatch_ManualValidation(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.ManualValidations = types.NewManualManifest("")
	evidence.ManualValidations.Add(types.ManualValidation{
		RequirementID: "REQ-001",
		Status:        "passed",
		ValidatedAt:   time.Now(),
		Notes:         "Manually verified",
	})

	val := &types.Validation{
		Type: types.ValTypeManual,
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match from manual validation")
	}
	if result.Status != types.LivePassed {
		t.Errorf("expected status passed, got: %s", result.Status)
	}
	if result.Evidence != "Manually verified" {
		t.Errorf("expected evidence notes, got: %s", result.Evidence)
	}
}

func TestMatcher_FindBestMatch_ExpiredManualValidation(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.ManualValidations = types.NewManualManifest("")
	evidence.ManualValidations.Add(types.ManualValidation{
		RequirementID: "REQ-001",
		Status:        "passed",
		ValidatedAt:   time.Now().Add(-48 * time.Hour),
		ExpiresAt:     time.Now().Add(-24 * time.Hour), // Expired
	})

	val := &types.Validation{
		Type: types.ValTypeManual,
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	// Should not match expired validation
	if result != nil {
		t.Error("expected nil for expired manual validation")
	}
}

func TestMatcher_FindBestMatch_SelectsBestByPriority(t *testing.T) {
	m := NewMatcher()

	now := time.Now()
	evidence := types.NewEvidenceBundle()

	// Add passed evidence
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
		UpdatedAt:     now,
	})

	// Add failed evidence (should win due to higher priority)
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LiveFailed,
		Phase:         "integration",
		UpdatedAt:     now,
	})

	val := &types.Validation{
		Type: types.ValTypeTest,
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match")
	}
	if result.Status != types.LiveFailed {
		t.Errorf("expected failed status (higher priority), got: %s", result.Status)
	}
}

func TestMatcher_FindBestMatch_SelectsMoreRecentForSamePriority(t *testing.T) {
	m := NewMatcher()

	earlier := time.Now().Add(-1 * time.Hour)
	later := time.Now()

	evidence := types.NewEvidenceBundle()

	// Add earlier passed evidence
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
		SourcePath:    "old.test.ts",
		UpdatedAt:     earlier,
	})

	// Add later passed evidence (should win due to recency)
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
		SourcePath:    "new.test.ts",
		UpdatedAt:     later,
	})

	val := &types.Validation{
		Type: types.ValTypeTest,
	}

	result := m.FindBestMatch(val, "REQ-001", evidence)

	if result == nil {
		t.Fatal("expected match")
	}
	if result.SourcePath != "new.test.ts" {
		t.Errorf("expected more recent evidence, got: %s", result.SourcePath)
	}
}

func TestMatcher_FindDirectEvidence_NilEvidence(t *testing.T) {
	m := NewMatcher()

	result := m.FindDirectEvidence("REQ-001", nil)

	if result != nil {
		t.Error("expected nil for nil evidence")
	}
}

func TestMatcher_FindDirectEvidence_FromPhaseResults(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
	})
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LiveFailed,
		Phase:         "integration",
	})
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-002",
		Status:        types.LivePassed,
		Phase:         "unit",
	})

	result := m.FindDirectEvidence("REQ-001", evidence)

	if len(result) != 2 {
		t.Errorf("expected 2 evidence records, got: %d", len(result))
	}
}

func TestMatcher_FindDirectEvidence_FromVitestEvidence(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.VitestEvidence["REQ-001"] = []types.VitestResult{
		{FilePath: "test1.test.ts", Status: "passed"},
		{FilePath: "test2.test.ts", Status: "passed"},
	}

	result := m.FindDirectEvidence("REQ-001", evidence)

	if len(result) != 2 {
		t.Errorf("expected 2 evidence records, got: %d", len(result))
	}
	for _, r := range result {
		if r.Phase != "unit" {
			t.Errorf("expected phase 'unit' for vitest evidence, got: %s", r.Phase)
		}
	}
}

func TestMatcher_FindDirectEvidence_FromManualValidations(t *testing.T) {
	m := NewMatcher()

	evidence := types.NewEvidenceBundle()
	evidence.ManualValidations = types.NewManualManifest("")
	evidence.ManualValidations.Add(types.ManualValidation{
		RequirementID: "REQ-001",
		Status:        "passed",
		ValidatedAt:   time.Now(),
		Notes:         "Manually verified",
	})

	result := m.FindDirectEvidence("REQ-001", evidence)

	if len(result) != 1 {
		t.Errorf("expected 1 evidence record, got: %d", len(result))
	}
	if result[0].Phase != "manual" {
		t.Errorf("expected phase 'manual', got: %s", result[0].Phase)
	}
}

func TestStatusPriority(t *testing.T) {
	tests := []struct {
		status   types.LiveStatus
		expected int
	}{
		{types.LiveFailed, 4},
		{types.LiveSkipped, 3},
		{types.LivePassed, 2},
		{types.LiveNotRun, 1},
		{types.LiveUnknown, 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := statusPriority(tt.status)
			if result != tt.expected {
				t.Errorf("expected priority %d for %s, got: %d", tt.expected, tt.status, result)
			}
		})
	}
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"./src/test.ts", "src/test.ts"},
		{"src\\test.ts", "src/test.ts"},
		{"  SRC/TEST.ts  ", "src/test.ts"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeRef(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestPathContains(t *testing.T) {
	tests := []struct {
		fullPath string
		pattern  string
		expected bool
	}{
		{"/src/components/Button.test.ts", "Button.test.ts", true},
		{"/src/components/Button.test.ts", "components/Button.test.ts", true},
		{"/src/components/Button.test.ts", "other.test.ts", false},
		{"./src/Button.test.ts", "src/Button.test.ts", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := pathContains(tt.fullPath, tt.pattern)
			if result != tt.expected {
				t.Errorf("pathContains(%q, %q) = %v, want %v",
					tt.fullPath, tt.pattern, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Aggregator Tests
// =============================================================================

func TestAggregator_AggregateRequirement_Nil(t *testing.T) {
	a := NewAggregator()

	// Should not panic
	a.AggregateRequirement(nil)
}

func TestAggregator_AggregateRequirement_NoValidations(t *testing.T) {
	a := NewAggregator()

	req := &types.Requirement{
		ID:     "REQ-001",
		Status: types.StatusInProgress,
	}

	a.AggregateRequirement(req)

	if req.LiveStatus != "" {
		t.Errorf("expected empty live status, got: %s", req.LiveStatus)
	}
}

func TestAggregator_AggregateRequirement_WithValidations(t *testing.T) {
	a := NewAggregator()

	req := &types.Requirement{
		ID:     "REQ-001",
		Status: types.StatusInProgress,
		Validations: []types.Validation{
			{Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
			{Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
		},
	}

	a.AggregateRequirement(req)

	if req.LiveStatus != types.LivePassed {
		t.Errorf("expected live status passed, got: %s", req.LiveStatus)
	}
}

func TestAggregator_AggregateRequirement_FailedTakesPrecedence(t *testing.T) {
	a := NewAggregator()

	req := &types.Requirement{
		ID:     "REQ-001",
		Status: types.StatusInProgress,
		Validations: []types.Validation{
			{Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
			{Status: types.ValStatusFailing, LiveStatus: types.LiveFailed},
			{Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
		},
	}

	a.AggregateRequirement(req)

	if req.LiveStatus != types.LiveFailed {
		t.Errorf("expected live status failed (highest priority), got: %s", req.LiveStatus)
	}
}

func TestAggregator_AggregateRequirement_SkipsUnknownStatus(t *testing.T) {
	a := NewAggregator()

	req := &types.Requirement{
		ID:     "REQ-001",
		Status: types.StatusInProgress,
		Validations: []types.Validation{
			{Status: types.ValStatusImplemented, LiveStatus: types.LiveUnknown},
			{Status: types.ValStatusImplemented, LiveStatus: types.LivePassed},
		},
	}

	a.AggregateRequirement(req)

	// Should only consider the non-unknown status
	if req.LiveStatus != types.LivePassed {
		t.Errorf("expected live status passed (ignoring unknown), got: %s", req.LiveStatus)
	}
}

func TestAggregator_AggregateRequirement_ValidationSummary(t *testing.T) {
	a := NewAggregator()

	req := &types.Requirement{
		ID:     "REQ-001",
		Status: types.StatusInProgress,
		Validations: []types.Validation{
			{Status: types.ValStatusImplemented},
			{Status: types.ValStatusFailing},
			{Status: types.ValStatusPlanned},
			{Status: types.ValStatusNotImplemented},
		},
	}

	a.AggregateRequirement(req)

	summary := req.AggregatedStatus.ValidationSummary
	if summary.Total != 4 {
		t.Errorf("expected total 4, got: %d", summary.Total)
	}
	if summary.Implemented != 1 {
		t.Errorf("expected implemented 1, got: %d", summary.Implemented)
	}
	if summary.Failing != 1 {
		t.Errorf("expected failing 1, got: %d", summary.Failing)
	}
	if summary.Planned != 1 {
		t.Errorf("expected planned 1, got: %d", summary.Planned)
	}
	if summary.NotImplemented != 1 {
		t.Errorf("expected not_implemented 1, got: %d", summary.NotImplemented)
	}
}

func TestAggregator_AggregateModule(t *testing.T) {
	a := NewAggregator()

	module := &types.RequirementModule{
		FilePath:   "/test/requirements/module.json",
		ModuleName: "test-module",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, LiveStatus: types.LivePassed},
			{ID: "REQ-002", Status: types.StatusInProgress, LiveStatus: types.LiveFailed},
			{ID: "REQ-003", Status: types.StatusPending, LiveStatus: types.LiveNotRun, Criticality: types.CriticalityP0},
		},
	}

	summary := a.AggregateModule(module)

	if summary.ModuleName != "test-module" {
		t.Errorf("expected module name 'test-module', got: %s", summary.ModuleName)
	}
	if summary.Total != 3 {
		t.Errorf("expected total 3, got: %d", summary.Total)
	}
	if summary.Complete != 1 {
		t.Errorf("expected complete 1, got: %d", summary.Complete)
	}
	if summary.InProgress != 1 {
		t.Errorf("expected in_progress 1, got: %d", summary.InProgress)
	}
	if summary.Pending != 1 {
		t.Errorf("expected pending 1, got: %d", summary.Pending)
	}
	if summary.LivePassed != 1 {
		t.Errorf("expected live_passed 1, got: %d", summary.LivePassed)
	}
	if summary.LiveFailed != 1 {
		t.Errorf("expected live_failed 1, got: %d", summary.LiveFailed)
	}
	if summary.Critical != 1 {
		t.Errorf("expected critical 1, got: %d", summary.Critical)
	}
	if summary.CriticalGap != 1 {
		t.Errorf("expected critical gap 1, got: %d", summary.CriticalGap)
	}
}

func TestAggregator_AggregateModule_CompletionPercent(t *testing.T) {
	a := NewAggregator()

	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete},
			{ID: "REQ-002", Status: types.StatusComplete},
			{ID: "REQ-003", Status: types.StatusInProgress},
			{ID: "REQ-004", Status: types.StatusPending},
		},
	}

	summary := a.AggregateModule(module)

	expectedPercent := 50.0 // 2/4 * 100
	if summary.CompletionPercent != expectedPercent {
		t.Errorf("expected completion percent %.1f, got: %.1f",
			expectedPercent, summary.CompletionPercent)
	}
}

func TestAggregator_AggregateAll(t *testing.T) {
	a := NewAggregator()

	modules := []*types.RequirementModule{
		{
			FilePath:   "/test/module1.json",
			ModuleName: "module1",
			Requirements: []types.Requirement{
				{ID: "REQ-001", Status: types.StatusComplete, LiveStatus: types.LivePassed},
				{ID: "REQ-002", Status: types.StatusComplete, LiveStatus: types.LivePassed},
			},
		},
		{
			FilePath:   "/test/module2.json",
			ModuleName: "module2",
			Requirements: []types.Requirement{
				{ID: "REQ-003", Status: types.StatusInProgress, LiveStatus: types.LiveFailed},
			},
		},
	}

	summary := a.AggregateAll(modules)

	if summary.Total != 3 {
		t.Errorf("expected total 3, got: %d", summary.Total)
	}
	if summary.Complete != 2 {
		t.Errorf("expected complete 2, got: %d", summary.Complete)
	}
	if summary.InProgress != 1 {
		t.Errorf("expected in_progress 1, got: %d", summary.InProgress)
	}
	if summary.LivePassed != 2 {
		t.Errorf("expected live_passed 2, got: %d", summary.LivePassed)
	}
	if summary.LiveFailed != 1 {
		t.Errorf("expected live_failed 1, got: %d", summary.LiveFailed)
	}
	if len(summary.ModuleSummaries) != 2 {
		t.Errorf("expected 2 module summaries, got: %d", len(summary.ModuleSummaries))
	}
}

func TestAggregator_AggregateAll_PassRate(t *testing.T) {
	a := NewAggregator()

	modules := []*types.RequirementModule{
		{
			FilePath: "/test/module.json",
			Requirements: []types.Requirement{
				{ID: "REQ-001", LiveStatus: types.LivePassed},
				{ID: "REQ-002", LiveStatus: types.LivePassed},
				{ID: "REQ-003", LiveStatus: types.LiveFailed},
				{ID: "REQ-004", LiveStatus: types.LiveNotRun}, // Not counted in pass rate
			},
		},
	}

	summary := a.AggregateAll(modules)

	// Pass rate = 2 passed / 3 tested (excluding not_run) = 66.67%
	expectedPassRate := 66.66666666666667
	if summary.PassRate < 66.6 || summary.PassRate > 66.7 {
		t.Errorf("expected pass rate ~%.1f%%, got: %.2f%%",
			expectedPassRate, summary.PassRate)
	}
}

func TestOverallSummary_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		summary  OverallSummary
		expected bool
	}{
		{
			name:     "healthy - no failures, no critical gap",
			summary:  OverallSummary{LiveFailed: 0, CriticalGap: 0},
			expected: true,
		},
		{
			name:     "unhealthy - has failures",
			summary:  OverallSummary{LiveFailed: 1, CriticalGap: 0},
			expected: false,
		},
		{
			name:     "unhealthy - has critical gap",
			summary:  OverallSummary{LiveFailed: 0, CriticalGap: 1},
			expected: false,
		},
		{
			name:     "unhealthy - both issues",
			summary:  OverallSummary{LiveFailed: 2, CriticalGap: 1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.summary.IsHealthy()
			if result != tt.expected {
				t.Errorf("expected IsHealthy() = %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestOverallSummary_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		failed   int
		expected bool
	}{
		{"no failures", 0, false},
		{"one failure", 1, true},
		{"multiple failures", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := OverallSummary{LiveFailed: tt.failed}
			if summary.HasFailures() != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", summary.HasFailures(), tt.expected)
			}
		})
	}
}

func TestOverallSummary_HasCriticalGap(t *testing.T) {
	tests := []struct {
		name     string
		gap      int
		expected bool
	}{
		{"no gap", 0, false},
		{"has gap", 1, true},
		{"large gap", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := OverallSummary{CriticalGap: tt.gap}
			if summary.HasCriticalGap() != tt.expected {
				t.Errorf("HasCriticalGap() = %v, want %v", summary.HasCriticalGap(), tt.expected)
			}
		})
	}
}

// =============================================================================
// HierarchyResolver Tests
// =============================================================================

func TestHierarchyResolver_ResolveHierarchy_NilIndex(t *testing.T) {
	h := NewHierarchyResolver()

	err := h.ResolveHierarchy(context.Background(), nil)

	if err != nil {
		t.Errorf("expected nil error for nil index, got: %v", err)
	}
}

func TestHierarchyResolver_ResolveHierarchy_ContextCancellation(t *testing.T) {
	h := NewHierarchyResolver()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	index := parsing.NewModuleIndex()

	err := h.ResolveHierarchy(ctx, index)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestHierarchyResolver_ResolveHierarchy_LeafNodes(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusComplete, LiveStatus: types.LivePassed},
			{ID: "REQ-002", Status: types.StatusInProgress, LiveStatus: types.LiveFailed},
		},
	}
	index.AddModule(module)

	err := h.ResolveHierarchy(context.Background(), index)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Leaf nodes should keep their own status
	req1 := index.GetRequirement("REQ-001")
	if req1.LiveStatus != types.LivePassed {
		t.Errorf("expected REQ-001 to keep LivePassed, got: %s", req1.LiveStatus)
	}
}

func TestHierarchyResolver_ResolveHierarchy_ParentChildRollup(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:       "REQ-PARENT",
				Status:   types.StatusInProgress,
				Children: []string{"REQ-CHILD-1", "REQ-CHILD-2"},
			},
			{ID: "REQ-CHILD-1", Status: types.StatusComplete, LiveStatus: types.LivePassed},
			{ID: "REQ-CHILD-2", Status: types.StatusComplete, LiveStatus: types.LiveFailed},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	err := h.ResolveHierarchy(context.Background(), index)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Parent should have failed status (worst of children)
	parent := index.GetRequirement("REQ-PARENT")
	if parent.LiveStatus != types.LiveFailed {
		t.Errorf("expected parent LiveStatus to be failed (rollup from children), got: %s",
			parent.LiveStatus)
	}
}

func TestHierarchyResolver_ResolveHierarchy_DeepHierarchy(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "L0", Status: types.StatusInProgress, Children: []string{"L1"}},
			{ID: "L1", Status: types.StatusInProgress, Children: []string{"L2"}},
			{ID: "L2", Status: types.StatusInProgress, Children: []string{"L3"}},
			{ID: "L3", Status: types.StatusComplete, LiveStatus: types.LivePassed},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	err := h.ResolveHierarchy(context.Background(), index)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Root should have passed status (from deepest child)
	root := index.GetRequirement("L0")
	if root.LiveStatus != types.LivePassed {
		t.Errorf("expected root LiveStatus to be passed, got: %s", root.LiveStatus)
	}
}

func TestHierarchyResolver_DetectCycles_NoCycles(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Children: []string{"REQ-002"}},
			{ID: "REQ-002", Children: []string{"REQ-003"}},
			{ID: "REQ-003"},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	cycles := h.DetectCycles(index)

	if len(cycles) != 0 {
		t.Errorf("expected no cycles, found: %d", len(cycles))
	}
}

func TestHierarchyResolver_DetectCycles_WithCycle(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-A", Children: []string{"REQ-B"}},
			{ID: "REQ-B", Children: []string{"REQ-C"}},
			{ID: "REQ-C", Children: []string{"REQ-A"}}, // Creates cycle
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	cycles := h.DetectCycles(index)

	if len(cycles) == 0 {
		t.Error("expected to detect a cycle")
	}
}

func TestHierarchyResolver_DetectCycles_NilIndex(t *testing.T) {
	h := NewHierarchyResolver()

	cycles := h.DetectCycles(nil)

	if cycles != nil {
		t.Error("expected nil for nil index")
	}
}

func TestHierarchyResolver_GetAncestors(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "L0", Children: []string{"L1"}},
			{ID: "L1", Children: []string{"L2"}},
			{ID: "L2", Children: []string{"L3"}},
			{ID: "L3"},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	ancestors := h.GetAncestors("L3", index)

	if len(ancestors) != 3 {
		t.Errorf("expected 3 ancestors, got: %d", len(ancestors))
	}
}

func TestHierarchyResolver_GetAncestors_NilIndex(t *testing.T) {
	h := NewHierarchyResolver()

	ancestors := h.GetAncestors("REQ-001", nil)

	if ancestors != nil {
		t.Error("expected nil for nil index")
	}
}

func TestHierarchyResolver_GetDescendants(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "ROOT", Children: []string{"CHILD-1", "CHILD-2"}},
			{ID: "CHILD-1", Children: []string{"GRANDCHILD-1"}},
			{ID: "CHILD-2"},
			{ID: "GRANDCHILD-1"},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	descendants := h.GetDescendants("ROOT", index)

	if len(descendants) != 3 {
		t.Errorf("expected 3 descendants, got: %d", len(descendants))
	}
}

func TestHierarchyResolver_GetDescendants_NilIndex(t *testing.T) {
	h := NewHierarchyResolver()

	descendants := h.GetDescendants("REQ-001", nil)

	if descendants != nil {
		t.Error("expected nil for nil index")
	}
}

func TestHierarchyResolver_GetDepth(t *testing.T) {
	h := NewHierarchyResolver()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "L0", Children: []string{"L1"}},
			{ID: "L1", Children: []string{"L2"}},
			{ID: "L2"},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	tests := []struct {
		id       string
		expected int
	}{
		{"L0", 0},
		{"L1", 1},
		{"L2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			depth := h.GetDepth(tt.id, index)
			if depth != tt.expected {
				t.Errorf("expected depth %d for %s, got: %d", tt.expected, tt.id, depth)
			}
		})
	}
}

// =============================================================================
// Enricher Tests
// =============================================================================

func TestEnricher_New(t *testing.T) {
	e := New()

	if e == nil {
		t.Fatal("expected non-nil enricher")
	}
}

func TestEnricher_Enrich_NilInputs(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		index    *parsing.ModuleIndex
		evidence *types.EvidenceBundle
	}{
		{"nil index", nil, types.NewEvidenceBundle()},
		{"nil evidence", parsing.NewModuleIndex(), nil},
		{"both nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.Enrich(context.Background(), tt.index, tt.evidence)
			if err != nil {
				t.Errorf("expected nil error for nil inputs, got: %v", err)
			}
		})
	}
}

func TestEnricher_Enrich_ContextCancellation(t *testing.T) {
	e := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	index := parsing.NewModuleIndex()
	evidence := types.NewEvidenceBundle()

	err := e.Enrich(ctx, index, evidence)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestEnricher_Enrich_MatchesEvidence(t *testing.T) {
	e := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:     "REQ-001",
				Status: types.StatusInProgress,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test/example.test.ts"},
				},
			},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LivePassed,
		Phase:         "unit",
		SourcePath:    "test/example.test.ts",
		UpdatedAt:     time.Now(),
	})

	err := e.Enrich(context.Background(), index, evidence)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	req := index.GetRequirement("REQ-001")
	if req.Validations[0].LiveStatus != types.LivePassed {
		t.Errorf("expected validation LiveStatus passed, got: %s",
			req.Validations[0].LiveStatus)
	}
	if req.Validations[0].Status != types.ValStatusImplemented {
		t.Errorf("expected validation Status implemented, got: %s",
			req.Validations[0].Status)
	}
}

func TestEnricher_Enrich_DirectEvidence(t *testing.T) {
	e := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{ID: "REQ-001", Status: types.StatusInProgress},
		},
	}
	index.AddModule(module)

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-001",
		Status:        types.LiveFailed,
		Phase:         "integration",
	})

	err := e.Enrich(context.Background(), index, evidence)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	req := index.GetRequirement("REQ-001")
	if req.LiveStatus != types.LiveFailed {
		t.Errorf("expected requirement LiveStatus failed from direct evidence, got: %s",
			req.LiveStatus)
	}
}

func TestEnricher_Enrich_FullPipeline(t *testing.T) {
	e := New()

	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/module.json",
		Requirements: []types.Requirement{
			{
				ID:       "REQ-PARENT",
				Status:   types.StatusInProgress,
				Children: []string{"REQ-CHILD-1", "REQ-CHILD-2"},
			},
			{
				ID:     "REQ-CHILD-1",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test/child1.test.ts"},
				},
			},
			{
				ID:     "REQ-CHILD-2",
				Status: types.StatusComplete,
				Validations: []types.Validation{
					{Type: types.ValTypeTest, Ref: "test/child2.test.ts"},
				},
			},
		},
	}
	index.AddModule(module)
	index.BuildHierarchy()

	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-CHILD-1",
		Status:        types.LivePassed,
		SourcePath:    "test/child1.test.ts",
	})
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-CHILD-2",
		Status:        types.LivePassed,
		SourcePath:    "test/child2.test.ts",
	})

	err := e.Enrich(context.Background(), index, evidence)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check children enriched
	child1 := index.GetRequirement("REQ-CHILD-1")
	if child1.Validations[0].LiveStatus != types.LivePassed {
		t.Errorf("expected child1 validation LiveStatus passed, got: %s",
			child1.Validations[0].LiveStatus)
	}

	// Check parent rollup
	parent := index.GetRequirement("REQ-PARENT")
	if parent.LiveStatus != types.LivePassed {
		t.Errorf("expected parent LiveStatus passed from child rollup, got: %s",
			parent.LiveStatus)
	}
}

func TestEnricher_ComputeSummary(t *testing.T) {
	e := New()

	modules := []*types.RequirementModule{
		{
			FilePath: "/test/module.json",
			Requirements: []types.Requirement{
				{
					ID:          "REQ-001",
					Status:      types.StatusComplete,
					LiveStatus:  types.LivePassed,
					Criticality: types.CriticalityP0,
					Validations: []types.Validation{
						{Status: types.ValStatusImplemented},
					},
				},
				{
					ID:          "REQ-002",
					Status:      types.StatusInProgress,
					LiveStatus:  types.LiveFailed,
					Criticality: types.CriticalityP1,
					Validations: []types.Validation{
						{Status: types.ValStatusFailing},
						{Status: types.ValStatusPlanned},
					},
				},
				{
					ID:         "REQ-003",
					Status:     types.StatusPending,
					LiveStatus: types.LiveNotRun,
				},
			},
		},
	}

	summary := e.ComputeSummary(modules)

	if summary.Total != 3 {
		t.Errorf("expected total 3, got: %d", summary.Total)
	}
	if summary.ByDeclaredStatus[types.StatusComplete] != 1 {
		t.Errorf("expected 1 complete, got: %d", summary.ByDeclaredStatus[types.StatusComplete])
	}
	if summary.ByDeclaredStatus[types.StatusInProgress] != 1 {
		t.Errorf("expected 1 in_progress, got: %d", summary.ByDeclaredStatus[types.StatusInProgress])
	}
	if summary.ByLiveStatus[types.LivePassed] != 1 {
		t.Errorf("expected 1 passed, got: %d", summary.ByLiveStatus[types.LivePassed])
	}
	if summary.ByLiveStatus[types.LiveFailed] != 1 {
		t.Errorf("expected 1 failed, got: %d", summary.ByLiveStatus[types.LiveFailed])
	}
	if summary.ByCriticality[types.CriticalityP0] != 1 {
		t.Errorf("expected 1 P0, got: %d", summary.ByCriticality[types.CriticalityP0])
	}
	if summary.CriticalityGap != 1 {
		t.Errorf("expected criticality gap 1 (P1 not complete), got: %d", summary.CriticalityGap)
	}
	if summary.ValidationStats.Total != 3 {
		t.Errorf("expected 3 total validations, got: %d", summary.ValidationStats.Total)
	}
	if summary.ValidationStats.Implemented != 1 {
		t.Errorf("expected 1 implemented validation, got: %d", summary.ValidationStats.Implemented)
	}
	if summary.ValidationStats.Failing != 1 {
		t.Errorf("expected 1 failing validation, got: %d", summary.ValidationStats.Failing)
	}
}

func TestDefaultEnrichOptions(t *testing.T) {
	opts := DefaultEnrichOptions()

	if !opts.IncludeManual {
		t.Error("expected IncludeManual to be true")
	}
	if !opts.PreferRecent {
		t.Error("expected PreferRecent to be true")
	}
	if !opts.RollupHierarchy {
		t.Error("expected RollupHierarchy to be true")
	}
}
