package types

import (
	"errors"
	"testing"
	"time"
)

// =============================================================================
// Status Normalization Tests
// =============================================================================

func TestNormalizeLiveStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected LiveStatus
	}{
		// Passed variants
		{"passed", LivePassed},
		{"PASSED", LivePassed},
		{"pass", LivePassed},
		{"success", LivePassed},
		{"ok", LivePassed},
		{" passed ", LivePassed},

		// Failed variants
		{"failed", LiveFailed},
		{"FAILED", LiveFailed},
		{"fail", LiveFailed},
		{"failure", LiveFailed},
		{"error", LiveFailed},

		// Skipped variants
		{"skipped", LiveSkipped},
		{"skip", LiveSkipped},
		{"pending", LiveSkipped},

		// Not run variants
		{"not_run", LiveNotRun},
		{"notrun", LiveNotRun},
		{"not-run", LiveNotRun},

		// Unknown
		{"", LiveUnknown},
		{"garbage", LiveUnknown},
		{"invalid", LiveUnknown},
	}

	for _, tt := range tests {
		result := NormalizeLiveStatus(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeLiveStatus(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeDeclaredStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected DeclaredStatus
	}{
		// Pending
		{"pending", StatusPending},
		{"", StatusPending},
		{"garbage", StatusPending},

		// Planned
		{"planned", StatusPlanned},
		{"PLANNED", StatusPlanned},

		// In progress variants
		{"in_progress", StatusInProgress},
		{"inprogress", StatusInProgress},
		{"in-progress", StatusInProgress},
		{"wip", StatusInProgress},

		// Complete variants
		{"complete", StatusComplete},
		{"completed", StatusComplete},
		{"done", StatusComplete},

		// Not implemented
		{"not_implemented", StatusNotImplemented},
		{"notimplemented", StatusNotImplemented},
		{"not-implemented", StatusNotImplemented},
	}

	for _, tt := range tests {
		result := NormalizeDeclaredStatus(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeDeclaredStatus(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeValidationStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected ValidationStatus
	}{
		// Not implemented
		{"not_implemented", ValStatusNotImplemented},
		{"notimplemented", ValStatusNotImplemented},
		{"not-implemented", ValStatusNotImplemented},
		{"", ValStatusNotImplemented},
		{"garbage", ValStatusNotImplemented},

		// Planned
		{"planned", ValStatusPlanned},

		// Implemented
		{"implemented", ValStatusImplemented},
		{"passing", ValStatusImplemented},
		{"passed", ValStatusImplemented},

		// Failing
		{"failing", ValStatusFailing},
		{"failed", ValStatusFailing},
	}

	for _, tt := range tests {
		result := NormalizeValidationStatus(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeValidationStatus(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeCriticality(t *testing.T) {
	tests := []struct {
		input    string
		expected Criticality
	}{
		// P0
		{"P0", CriticalityP0},
		{"p0", CriticalityP0},
		{"CRITICAL", CriticalityP0},
		{"HIGH", CriticalityP0},

		// P1
		{"P1", CriticalityP1},
		{"MAJOR", CriticalityP1},
		{"MEDIUM", CriticalityP1},

		// P2 (default)
		{"P2", CriticalityP2},
		{"MINOR", CriticalityP2},
		{"LOW", CriticalityP2},
		{"", CriticalityP2},
		{"garbage", CriticalityP2},
	}

	for _, tt := range tests {
		result := NormalizeCriticality(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeCriticality(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeValidationType(t *testing.T) {
	tests := []struct {
		input    string
		expected ValidationType
	}{
		// Test (default)
		{"test", ValTypeTest},
		{"unit", ValTypeTest},
		{"integration", ValTypeTest},
		{"", ValTypeTest},
		{"garbage", ValTypeTest},

		// Automation
		{"automation", ValTypeAutomation},
		{"workflow", ValTypeAutomation},
		{"playbook", ValTypeAutomation},

		// Manual
		{"manual", ValTypeManual},

		// Lighthouse
		{"lighthouse", ValTypeLighthouse},
		{"perf", ValTypeLighthouse},
		{"performance", ValTypeLighthouse},
	}

	for _, tt := range tests {
		result := NormalizeValidationType(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeValidationType(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// =============================================================================
// Status Derivation Tests
// =============================================================================

func TestDeriveValidationStatus(t *testing.T) {
	tests := []struct {
		live     LiveStatus
		expected ValidationStatus
	}{
		{LivePassed, ValStatusImplemented},
		{LiveFailed, ValStatusFailing},
		{LiveSkipped, ValStatusPlanned},
		{LiveNotRun, ValStatusPlanned},
		{LiveUnknown, ValStatusNotImplemented},
	}

	for _, tt := range tests {
		result := DeriveValidationStatus(tt.live)
		if result != tt.expected {
			t.Errorf("DeriveValidationStatus(%v) = %v, want %v", tt.live, result, tt.expected)
		}
	}
}

func TestDeriveRequirementStatus(t *testing.T) {
	tests := []struct {
		name        string
		current     DeclaredStatus
		validations []ValidationStatus
		expected    DeclaredStatus
	}{
		{
			name:        "empty validations returns current",
			current:     StatusInProgress,
			validations: nil,
			expected:    StatusInProgress,
		},
		{
			name:        "all implemented promotes in_progress to complete",
			current:     StatusInProgress,
			validations: []ValidationStatus{ValStatusImplemented, ValStatusImplemented},
			expected:    StatusComplete,
		},
		{
			name:        "all implemented promotes planned to complete",
			current:     StatusPlanned,
			validations: []ValidationStatus{ValStatusImplemented},
			expected:    StatusComplete,
		},
		{
			name:        "all implemented keeps pending as pending",
			current:     StatusPending,
			validations: []ValidationStatus{ValStatusImplemented},
			expected:    StatusPending,
		},
		{
			name:        "failing demotes complete to in_progress",
			current:     StatusComplete,
			validations: []ValidationStatus{ValStatusFailing},
			expected:    StatusInProgress,
		},
		{
			name:        "failing keeps in_progress as in_progress",
			current:     StatusInProgress,
			validations: []ValidationStatus{ValStatusFailing},
			expected:    StatusInProgress,
		},
		{
			name:        "planned validation promotes pending to planned",
			current:     StatusPending,
			validations: []ValidationStatus{ValStatusPlanned},
			expected:    StatusPlanned,
		},
		{
			name:        "planned validation keeps in_progress unchanged",
			current:     StatusInProgress,
			validations: []ValidationStatus{ValStatusPlanned},
			expected:    StatusInProgress,
		},
		{
			name:        "mixed failing prevents complete",
			current:     StatusComplete,
			validations: []ValidationStatus{ValStatusImplemented, ValStatusFailing},
			expected:    StatusInProgress,
		},
		{
			name:        "not_implemented validation",
			current:     StatusInProgress,
			validations: []ValidationStatus{ValStatusNotImplemented},
			expected:    StatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveRequirementStatus(tt.current, tt.validations)
			if result != tt.expected {
				t.Errorf("DeriveRequirementStatus(%v, %v) = %v, want %v",
					tt.current, tt.validations, result, tt.expected)
			}
		})
	}
}

func TestDeriveLiveRollup(t *testing.T) {
	tests := []struct {
		name     string
		statuses []LiveStatus
		expected LiveStatus
	}{
		{"empty returns unknown", nil, LiveUnknown},
		{"single passed", []LiveStatus{LivePassed}, LivePassed},
		{"single failed", []LiveStatus{LiveFailed}, LiveFailed},
		{"failed > passed", []LiveStatus{LivePassed, LiveFailed}, LiveFailed},
		{"skipped > passed", []LiveStatus{LivePassed, LiveSkipped}, LiveSkipped},
		{"failed > skipped", []LiveStatus{LiveSkipped, LiveFailed}, LiveFailed},
		{"passed > not_run", []LiveStatus{LiveNotRun, LivePassed}, LivePassed},
		{"not_run > unknown", []LiveStatus{LiveUnknown, LiveNotRun}, LiveNotRun},
		{"all statuses - failed wins", []LiveStatus{LivePassed, LiveSkipped, LiveFailed, LiveNotRun, LiveUnknown}, LiveFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveLiveRollup(tt.statuses)
			if result != tt.expected {
				t.Errorf("DeriveLiveRollup(%v) = %v, want %v", tt.statuses, result, tt.expected)
			}
		})
	}
}

func TestDeriveDeclaredRollup(t *testing.T) {
	tests := []struct {
		name     string
		statuses []DeclaredStatus
		expected DeclaredStatus
	}{
		{"empty returns pending", nil, StatusPending},
		{"all complete", []DeclaredStatus{StatusComplete, StatusComplete}, StatusComplete},
		{"not_implemented overrides all", []DeclaredStatus{StatusComplete, StatusNotImplemented}, StatusNotImplemented},
		{"pending overrides planned", []DeclaredStatus{StatusPlanned, StatusPending}, StatusPending},
		{"planned overrides in_progress", []DeclaredStatus{StatusInProgress, StatusPlanned}, StatusPlanned},
		{"in_progress with complete", []DeclaredStatus{StatusComplete, StatusInProgress}, StatusInProgress},
		{"single not_implemented", []DeclaredStatus{StatusNotImplemented}, StatusNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveDeclaredRollup(tt.statuses)
			if result != tt.expected {
				t.Errorf("DeriveDeclaredRollup(%v) = %v, want %v", tt.statuses, result, tt.expected)
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   DeclaredStatus
		expected bool
	}{
		{StatusComplete, true},
		{StatusNotImplemented, true},
		{StatusPending, false},
		{StatusPlanned, false},
		{StatusInProgress, false},
	}

	for _, tt := range tests {
		result := IsTerminalStatus(tt.status)
		if result != tt.expected {
			t.Errorf("IsTerminalStatus(%v) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestIsCritical(t *testing.T) {
	tests := []struct {
		criticality Criticality
		expected    bool
	}{
		{CriticalityP0, true},
		{CriticalityP1, true},
		{CriticalityP2, false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsCritical(tt.criticality)
		if result != tt.expected {
			t.Errorf("IsCritical(%v) = %v, want %v", tt.criticality, result, tt.expected)
		}
	}
}

// =============================================================================
// Requirement Tests
// =============================================================================

func TestRequirement_Clone(t *testing.T) {
	t.Run("nil requirement", func(t *testing.T) {
		var r *Requirement
		if r.Clone() != nil {
			t.Error("Clone of nil should return nil")
		}
	})

	t.Run("full requirement", func(t *testing.T) {
		original := &Requirement{
			ID:           "REQ-001",
			Title:        "Test Requirement",
			Status:       StatusInProgress,
			PRDRef:       "PRD-001",
			Category:     "core",
			Criticality:  CriticalityP0,
			Description:  "A test requirement",
			Tags:         []string{"tag1", "tag2"},
			Children:     []string{"REQ-002"},
			DependsOn:    []string{"REQ-000"},
			Blocks:       []string{"REQ-003"},
			LiveStatus:   LivePassed,
			SourceFile:   "/path/to/file.json",
			SourceModule: "core",
			Validations: []Validation{
				{Type: ValTypeTest, Ref: "TestFunc", Status: ValStatusImplemented},
			},
		}

		clone := original.Clone()

		// Verify values are copied
		if clone.ID != original.ID {
			t.Errorf("ID mismatch: %v != %v", clone.ID, original.ID)
		}
		if clone.Title != original.Title {
			t.Errorf("Title mismatch: %v != %v", clone.Title, original.Title)
		}
		if len(clone.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: %v != %v", len(clone.Tags), len(original.Tags))
		}
		if len(clone.Validations) != len(original.Validations) {
			t.Errorf("Validations length mismatch: %v != %v", len(clone.Validations), len(original.Validations))
		}

		// Verify deep copy (modifying clone doesn't affect original)
		clone.Tags[0] = "modified"
		if original.Tags[0] == "modified" {
			t.Error("Clone shares Tags slice with original")
		}
	})

	t.Run("empty slices", func(t *testing.T) {
		original := &Requirement{ID: "REQ-001"}
		clone := original.Clone()

		if clone.Tags != nil {
			t.Error("Empty Tags should be nil in clone")
		}
		if clone.Children != nil {
			t.Error("Empty Children should be nil in clone")
		}
	})
}

func TestRequirement_HasValidations(t *testing.T) {
	tests := []struct {
		name     string
		req      *Requirement
		expected bool
	}{
		{"nil requirement", nil, false},
		{"no validations", &Requirement{ID: "REQ-001"}, false},
		{"has validations", &Requirement{
			ID:          "REQ-001",
			Validations: []Validation{{Type: ValTypeTest}},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.HasValidations() != tt.expected {
				t.Errorf("HasValidations() = %v, want %v", tt.req.HasValidations(), tt.expected)
			}
		})
	}
}

func TestRequirement_HasChildren(t *testing.T) {
	tests := []struct {
		name     string
		req      *Requirement
		expected bool
	}{
		{"nil requirement", nil, false},
		{"no children", &Requirement{ID: "REQ-001"}, false},
		{"has children", &Requirement{ID: "REQ-001", Children: []string{"REQ-002"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.HasChildren() != tt.expected {
				t.Errorf("HasChildren() = %v, want %v", tt.req.HasChildren(), tt.expected)
			}
		})
	}
}

func TestRequirement_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		req      *Requirement
		expected bool
	}{
		{"nil requirement", nil, false},
		{"not complete", &Requirement{ID: "REQ-001", Status: StatusInProgress}, false},
		{"complete", &Requirement{ID: "REQ-001", Status: StatusComplete}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.IsComplete() != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", tt.req.IsComplete(), tt.expected)
			}
		})
	}
}

func TestRequirement_IsCriticalReq(t *testing.T) {
	tests := []struct {
		name     string
		req      *Requirement
		expected bool
	}{
		{"nil requirement", nil, false},
		{"P0 criticality", &Requirement{ID: "REQ-001", Criticality: CriticalityP0}, true},
		{"P1 criticality", &Requirement{ID: "REQ-001", Criticality: CriticalityP1}, true},
		{"P2 criticality", &Requirement{ID: "REQ-001", Criticality: CriticalityP2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.IsCriticalReq() != tt.expected {
				t.Errorf("IsCriticalReq() = %v, want %v", tt.req.IsCriticalReq(), tt.expected)
			}
		})
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestValidation_Clone(t *testing.T) {
	t.Run("full validation", func(t *testing.T) {
		original := Validation{
			Type:       ValTypeTest,
			Ref:        "TestFunc",
			WorkflowID: "wf-123",
			Phase:      "unit",
			Status:     ValStatusImplemented,
			Notes:      "Test notes",
			Scenario:   "test-scenario",
			Folder:     "tests/",
			LiveStatus: LivePassed,
			Metadata:   map[string]any{"key": "value"},
			LiveDetails: &LiveDetails{
				Timestamp:       time.Now(),
				DurationSeconds: 1.5,
				Evidence:        "evidence.log",
				SourcePath:      "/path/to/test.go",
			},
		}

		clone := original.Clone()

		if clone.Type != original.Type {
			t.Errorf("Type mismatch: %v != %v", clone.Type, original.Type)
		}
		if clone.Ref != original.Ref {
			t.Errorf("Ref mismatch: %v != %v", clone.Ref, original.Ref)
		}

		// Verify deep copy of metadata
		clone.Metadata["key"] = "modified"
		if original.Metadata["key"] == "modified" {
			t.Error("Clone shares Metadata map with original")
		}

		// Verify deep copy of LiveDetails
		clone.LiveDetails.Evidence = "modified"
		if original.LiveDetails.Evidence == "modified" {
			t.Error("Clone shares LiveDetails with original")
		}
	})

	t.Run("nil metadata and details", func(t *testing.T) {
		original := Validation{Type: ValTypeTest}
		clone := original.Clone()

		if clone.Metadata != nil {
			t.Error("Nil Metadata should remain nil in clone")
		}
		if clone.LiveDetails != nil {
			t.Error("Nil LiveDetails should remain nil in clone")
		}
	})
}

func TestValidation_Key(t *testing.T) {
	tests := []struct {
		name     string
		v        Validation
		expected string
	}{
		{"ref takes precedence", Validation{Ref: "TestRef", WorkflowID: "wf-123"}, "TestRef"},
		{"workflow_id fallback", Validation{WorkflowID: "wf-123"}, "wf-123"},
		{"empty key", Validation{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.v.Key() != tt.expected {
				t.Errorf("Key() = %v, want %v", tt.v.Key(), tt.expected)
			}
		})
	}
}

func TestValidation_IsTest(t *testing.T) {
	if (&Validation{Type: ValTypeTest}).IsTest() != true {
		t.Error("IsTest() should return true for ValTypeTest")
	}
	if (&Validation{Type: ValTypeAutomation}).IsTest() != false {
		t.Error("IsTest() should return false for ValTypeAutomation")
	}
}

func TestValidation_IsAutomation(t *testing.T) {
	if (&Validation{Type: ValTypeAutomation}).IsAutomation() != true {
		t.Error("IsAutomation() should return true for ValTypeAutomation")
	}
	if (&Validation{Type: ValTypeTest}).IsAutomation() != false {
		t.Error("IsAutomation() should return false for ValTypeTest")
	}
}

func TestValidation_IsManual(t *testing.T) {
	if (&Validation{Type: ValTypeManual}).IsManual() != true {
		t.Error("IsManual() should return true for ValTypeManual")
	}
	if (&Validation{Type: ValTypeTest}).IsManual() != false {
		t.Error("IsManual() should return false for ValTypeTest")
	}
}

func TestComputeValidationSummary(t *testing.T) {
	validations := []Validation{
		{Status: ValStatusImplemented},
		{Status: ValStatusImplemented},
		{Status: ValStatusFailing},
		{Status: ValStatusPlanned},
		{Status: ValStatusNotImplemented},
	}

	summary := ComputeValidationSummary(validations)

	if summary.Total != 5 {
		t.Errorf("Total = %d, want 5", summary.Total)
	}
	if summary.Implemented != 2 {
		t.Errorf("Implemented = %d, want 2", summary.Implemented)
	}
	if summary.Failing != 1 {
		t.Errorf("Failing = %d, want 1", summary.Failing)
	}
	if summary.Planned != 1 {
		t.Errorf("Planned = %d, want 1", summary.Planned)
	}
	if summary.NotImplemented != 1 {
		t.Errorf("NotImplemented = %d, want 1", summary.NotImplemented)
	}
}

// =============================================================================
// Module Tests
// =============================================================================

func TestModuleMetadata_GetModuleName(t *testing.T) {
	tests := []struct {
		name     string
		metadata ModuleMetadata
		expected string
	}{
		{"module_name takes precedence", ModuleMetadata{ModuleName: "name", Module: "mod"}, "name"},
		{"falls back to module", ModuleMetadata{Module: "mod"}, "mod"},
		{"empty returns empty", ModuleMetadata{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metadata.GetModuleName() != tt.expected {
				t.Errorf("GetModuleName() = %v, want %v", tt.metadata.GetModuleName(), tt.expected)
			}
		})
	}
}

func TestModuleMetadata_IsAutoSyncEnabled(t *testing.T) {
	t.Run("nil defaults to true", func(t *testing.T) {
		m := ModuleMetadata{}
		if !m.IsAutoSyncEnabled() {
			t.Error("IsAutoSyncEnabled() should default to true")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		val := true
		m := ModuleMetadata{AutoSyncEnabled: &val}
		if !m.IsAutoSyncEnabled() {
			t.Error("IsAutoSyncEnabled() should return true")
		}
	})

	t.Run("explicit false", func(t *testing.T) {
		val := false
		m := ModuleMetadata{AutoSyncEnabled: &val}
		if m.IsAutoSyncEnabled() {
			t.Error("IsAutoSyncEnabled() should return false")
		}
	})
}

func TestRequirementModule_Clone(t *testing.T) {
	t.Run("nil module", func(t *testing.T) {
		var m *RequirementModule
		if m.Clone() != nil {
			t.Error("Clone of nil should return nil")
		}
	})

	t.Run("full module", func(t *testing.T) {
		autoSync := true
		original := &RequirementModule{
			Metadata: ModuleMetadata{
				Module:          "core",
				AutoSyncEnabled: &autoSync,
			},
			Imports:      []string{"import1", "import2"},
			FilePath:     "/path/to/module.json",
			RelativePath: "core/module.json",
			IsIndex:      false,
			ModuleName:   "core",
			Requirements: []Requirement{
				{ID: "REQ-001", Title: "Test"},
			},
		}

		clone := original.Clone()

		if clone.FilePath != original.FilePath {
			t.Errorf("FilePath mismatch: %v != %v", clone.FilePath, original.FilePath)
		}
		if len(clone.Imports) != len(original.Imports) {
			t.Errorf("Imports length mismatch: %v != %v", len(clone.Imports), len(original.Imports))
		}
		if len(clone.Requirements) != len(original.Requirements) {
			t.Errorf("Requirements length mismatch: %v != %v", len(clone.Requirements), len(original.Requirements))
		}

		// Verify deep copy
		clone.Imports[0] = "modified"
		if original.Imports[0] == "modified" {
			t.Error("Clone shares Imports slice with original")
		}
	})
}

func TestModuleMetadata_Clone(t *testing.T) {
	t.Run("with auto_sync_enabled", func(t *testing.T) {
		val := true
		original := ModuleMetadata{
			Module:          "core",
			AutoSyncEnabled: &val,
		}

		clone := original.Clone()

		if *clone.AutoSyncEnabled != *original.AutoSyncEnabled {
			t.Error("AutoSyncEnabled value mismatch")
		}

		// Verify deep copy
		*clone.AutoSyncEnabled = false
		if *original.AutoSyncEnabled == false {
			t.Error("Clone shares AutoSyncEnabled pointer with original")
		}
	})

	t.Run("nil auto_sync_enabled", func(t *testing.T) {
		original := ModuleMetadata{Module: "core"}
		clone := original.Clone()

		if clone.AutoSyncEnabled != nil {
			t.Error("Nil AutoSyncEnabled should remain nil in clone")
		}
	})
}

func TestRequirementModule_RequirementCount(t *testing.T) {
	tests := []struct {
		name     string
		module   *RequirementModule
		expected int
	}{
		{"nil module", nil, 0},
		{"empty module", &RequirementModule{}, 0},
		{"with requirements", &RequirementModule{
			Requirements: []Requirement{{ID: "REQ-001"}, {ID: "REQ-002"}},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.module.RequirementCount() != tt.expected {
				t.Errorf("RequirementCount() = %v, want %v", tt.module.RequirementCount(), tt.expected)
			}
		})
	}
}

func TestRequirementModule_GetRequirement(t *testing.T) {
	module := &RequirementModule{
		Requirements: []Requirement{
			{ID: "REQ-001", Title: "First"},
			{ID: "REQ-002", Title: "Second"},
		},
	}

	t.Run("existing requirement", func(t *testing.T) {
		req := module.GetRequirement("REQ-001")
		if req == nil || req.Title != "First" {
			t.Error("Should find REQ-001")
		}
	})

	t.Run("non-existing requirement", func(t *testing.T) {
		req := module.GetRequirement("REQ-999")
		if req != nil {
			t.Error("Should return nil for non-existing ID")
		}
	})

	t.Run("nil module", func(t *testing.T) {
		var m *RequirementModule
		if m.GetRequirement("REQ-001") != nil {
			t.Error("Should return nil for nil module")
		}
	})
}

func TestRequirementModule_AllRequirementIDs(t *testing.T) {
	t.Run("nil module", func(t *testing.T) {
		var m *RequirementModule
		if m.AllRequirementIDs() != nil {
			t.Error("Should return nil for nil module")
		}
	})

	t.Run("with requirements", func(t *testing.T) {
		module := &RequirementModule{
			Requirements: []Requirement{
				{ID: "REQ-001"},
				{ID: "REQ-002"},
			},
		}
		ids := module.AllRequirementIDs()
		if len(ids) != 2 || ids[0] != "REQ-001" || ids[1] != "REQ-002" {
			t.Errorf("AllRequirementIDs() = %v, want [REQ-001, REQ-002]", ids)
		}
	})
}

func TestRequirementModule_HasRequirements(t *testing.T) {
	tests := []struct {
		name     string
		module   *RequirementModule
		expected bool
	}{
		{"nil module", nil, false},
		{"empty module", &RequirementModule{}, false},
		{"has requirements", &RequirementModule{
			Requirements: []Requirement{{ID: "REQ-001"}},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.module.HasRequirements() != tt.expected {
				t.Errorf("HasRequirements() = %v, want %v", tt.module.HasRequirements(), tt.expected)
			}
		})
	}
}

func TestRequirementModule_HasImports(t *testing.T) {
	tests := []struct {
		name     string
		module   *RequirementModule
		expected bool
	}{
		{"nil module", nil, false},
		{"empty module", &RequirementModule{}, false},
		{"has imports", &RequirementModule{Imports: []string{"import1"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.module.HasImports() != tt.expected {
				t.Errorf("HasImports() = %v, want %v", tt.module.HasImports(), tt.expected)
			}
		})
	}
}

func TestRequirementModule_EffectiveName(t *testing.T) {
	tests := []struct {
		name     string
		module   *RequirementModule
		expected string
	}{
		{"nil module", nil, ""},
		{"module name takes precedence", &RequirementModule{
			ModuleName:   "direct-name",
			Metadata:     ModuleMetadata{ModuleName: "metadata-name"},
			RelativePath: "path/module.json",
		}, "direct-name"},
		{"metadata name fallback", &RequirementModule{
			Metadata:     ModuleMetadata{ModuleName: "metadata-name"},
			RelativePath: "path/module.json",
		}, "metadata-name"},
		{"metadata module fallback", &RequirementModule{
			Metadata:     ModuleMetadata{Module: "module-field"},
			RelativePath: "path/module.json",
		}, "module-field"},
		{"relative path fallback", &RequirementModule{
			RelativePath: "path/module.json",
		}, "path/module.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.module.EffectiveName() != tt.expected {
				t.Errorf("EffectiveName() = %v, want %v", tt.module.EffectiveName(), tt.expected)
			}
		})
	}
}

// =============================================================================
// Evidence Tests
// =============================================================================

func TestEvidenceMap_Get(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		var m EvidenceMap
		if m.Get("REQ-001") != nil {
			t.Error("Get on nil map should return nil")
		}
	})

	t.Run("existing key", func(t *testing.T) {
		m := EvidenceMap{
			"REQ-001": []EvidenceRecord{{Status: LivePassed}},
		}
		records := m.Get("REQ-001")
		if len(records) != 1 {
			t.Errorf("Get() returned %d records, want 1", len(records))
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		m := make(EvidenceMap)
		if m.Get("REQ-001") != nil {
			t.Error("Get on non-existing key should return nil")
		}
	})
}

func TestEvidenceMap_Add(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		var m EvidenceMap
		m.Add(EvidenceRecord{RequirementID: "REQ-001"}) // Should not panic
	})

	t.Run("adds record", func(t *testing.T) {
		m := make(EvidenceMap)
		m.Add(EvidenceRecord{RequirementID: "REQ-001", Status: LivePassed})
		m.Add(EvidenceRecord{RequirementID: "REQ-001", Status: LiveFailed})

		records := m.Get("REQ-001")
		if len(records) != 2 {
			t.Errorf("Expected 2 records, got %d", len(records))
		}
	})
}

func TestEvidenceMap_Merge(t *testing.T) {
	t.Run("nil maps", func(t *testing.T) {
		var m1 EvidenceMap
		m2 := make(EvidenceMap)
		m1.Merge(m2) // Should not panic

		var m3 EvidenceMap
		m2.Merge(m3) // Should not panic
	})

	t.Run("merges maps", func(t *testing.T) {
		m1 := EvidenceMap{
			"REQ-001": []EvidenceRecord{{Status: LivePassed}},
		}
		m2 := EvidenceMap{
			"REQ-001": []EvidenceRecord{{Status: LiveFailed}},
			"REQ-002": []EvidenceRecord{{Status: LiveSkipped}},
		}

		m1.Merge(m2)

		if len(m1.Get("REQ-001")) != 2 {
			t.Error("REQ-001 should have 2 records after merge")
		}
		if len(m1.Get("REQ-002")) != 1 {
			t.Error("REQ-002 should have 1 record after merge")
		}
	})
}

func TestEvidenceMap_RequirementIDs(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		var m EvidenceMap
		if m.RequirementIDs() != nil {
			t.Error("RequirementIDs on nil map should return nil")
		}
	})

	t.Run("returns all IDs", func(t *testing.T) {
		m := EvidenceMap{
			"REQ-001": []EvidenceRecord{{}},
			"REQ-002": []EvidenceRecord{{}},
		}
		ids := m.RequirementIDs()
		if len(ids) != 2 {
			t.Errorf("Expected 2 IDs, got %d", len(ids))
		}
	})
}

func TestEvidenceMap_Count(t *testing.T) {
	tests := []struct {
		name     string
		m        EvidenceMap
		expected int
	}{
		{"nil map", nil, 0},
		{"empty map", make(EvidenceMap), 0},
		{"with records", EvidenceMap{
			"REQ-001": []EvidenceRecord{{}, {}},
			"REQ-002": []EvidenceRecord{{}},
		}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.m.Count() != tt.expected {
				t.Errorf("Count() = %d, want %d", tt.m.Count(), tt.expected)
			}
		})
	}
}

func TestManualValidation_IsExpired(t *testing.T) {
	t.Run("nil validation", func(t *testing.T) {
		var m *ManualValidation
		if m.IsExpired() {
			t.Error("nil validation should not be expired")
		}
	})

	t.Run("zero expiry", func(t *testing.T) {
		m := &ManualValidation{RequirementID: "REQ-001"}
		if m.IsExpired() {
			t.Error("Zero expiry should not be expired")
		}
	})

	t.Run("future expiry", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			ExpiresAt:     time.Now().Add(time.Hour),
		}
		if m.IsExpired() {
			t.Error("Future expiry should not be expired")
		}
	})

	t.Run("past expiry", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			ExpiresAt:     time.Now().Add(-time.Hour),
		}
		if !m.IsExpired() {
			t.Error("Past expiry should be expired")
		}
	})
}

func TestManualValidation_IsValid(t *testing.T) {
	t.Run("nil validation", func(t *testing.T) {
		var m *ManualValidation
		if m.IsValid() {
			t.Error("nil validation should not be valid")
		}
	})

	t.Run("valid validation", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			ExpiresAt:     time.Now().Add(time.Hour),
		}
		if !m.IsValid() {
			t.Error("Should be valid")
		}
	})

	t.Run("expired validation", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			ExpiresAt:     time.Now().Add(-time.Hour),
		}
		if m.IsValid() {
			t.Error("Expired validation should not be valid")
		}
	})
}

func TestManualValidation_ToLiveStatus(t *testing.T) {
	t.Run("nil validation", func(t *testing.T) {
		var m *ManualValidation
		if m.ToLiveStatus() != LiveNotRun {
			t.Error("nil validation should return LiveNotRun")
		}
	})

	t.Run("expired validation", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			Status:        "passed",
			ExpiresAt:     time.Now().Add(-time.Hour),
		}
		if m.ToLiveStatus() != LiveNotRun {
			t.Error("Expired validation should return LiveNotRun")
		}
	})

	t.Run("valid validation", func(t *testing.T) {
		m := &ManualValidation{
			RequirementID: "REQ-001",
			Status:        "passed",
		}
		if m.ToLiveStatus() != LivePassed {
			t.Errorf("ToLiveStatus() = %v, want %v", m.ToLiveStatus(), LivePassed)
		}
	})
}

func TestPhaseResult_ToLiveStatus(t *testing.T) {
	t.Run("nil phase result", func(t *testing.T) {
		var p *PhaseResult
		if p.ToLiveStatus() != LiveNotRun {
			t.Error("nil should return LiveNotRun")
		}
	})

	t.Run("passed status", func(t *testing.T) {
		p := &PhaseResult{Status: "passed"}
		if p.ToLiveStatus() != LivePassed {
			t.Errorf("ToLiveStatus() = %v, want %v", p.ToLiveStatus(), LivePassed)
		}
	})
}

func TestPhaseResult_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		p        *PhaseResult
		expected bool
	}{
		{"nil", nil, false},
		{"no failures", &PhaseResult{FailCount: 0}, false},
		{"has failures", &PhaseResult{FailCount: 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.p.HasFailures() != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", tt.p.HasFailures(), tt.expected)
			}
		})
	}
}

func TestVitestResult_ToLiveStatus(t *testing.T) {
	t.Run("nil vitest result", func(t *testing.T) {
		var v *VitestResult
		if v.ToLiveStatus() != LiveNotRun {
			t.Error("nil should return LiveNotRun")
		}
	})

	t.Run("failed status", func(t *testing.T) {
		v := &VitestResult{Status: "failed"}
		if v.ToLiveStatus() != LiveFailed {
			t.Errorf("ToLiveStatus() = %v, want %v", v.ToLiveStatus(), LiveFailed)
		}
	})
}

func TestVitestResult_CoveragePercent(t *testing.T) {
	tests := []struct {
		name     string
		v        *VitestResult
		expected float64
	}{
		{"nil", nil, 0},
		{"zero total lines", &VitestResult{TotalLines: 0}, 0},
		{"50% coverage", &VitestResult{CoveredLines: 50, TotalLines: 100}, 50},
		{"100% coverage", &VitestResult{CoveredLines: 100, TotalLines: 100}, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.v.CoveragePercent() != tt.expected {
				t.Errorf("CoveragePercent() = %v, want %v", tt.v.CoveragePercent(), tt.expected)
			}
		})
	}
}

func TestNewEvidenceBundle(t *testing.T) {
	bundle := NewEvidenceBundle()

	if bundle == nil {
		t.Fatal("NewEvidenceBundle() returned nil")
	}
	if bundle.PhaseResults == nil {
		t.Error("PhaseResults should be initialized")
	}
	if bundle.VitestEvidence == nil {
		t.Error("VitestEvidence should be initialized")
	}
	if bundle.LoadedAt.IsZero() {
		t.Error("LoadedAt should be set")
	}
}

func TestEvidenceBundle_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		bundle   *EvidenceBundle
		expected bool
	}{
		{"nil bundle", nil, true},
		{"empty bundle", NewEvidenceBundle(), true},
		{"has phase results", &EvidenceBundle{
			PhaseResults: EvidenceMap{"REQ-001": []EvidenceRecord{{}}},
		}, false},
		{"has vitest evidence", &EvidenceBundle{
			VitestEvidence: map[string][]VitestResult{"REQ-001": {{}}},
		}, false},
		{"has manual validations", &EvidenceBundle{
			ManualValidations: &ManualManifest{
				Entries: []ManualValidation{{}},
			},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bundle.IsEmpty() != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", tt.bundle.IsEmpty(), tt.expected)
			}
		})
	}
}

func TestManualManifest_Add(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		var m *ManualManifest
		m.Add(ManualValidation{RequirementID: "REQ-001"}) // Should not panic
	})

	t.Run("adds and indexes", func(t *testing.T) {
		m := NewManualManifest("/path/to/log.jsonl")
		m.Add(ManualValidation{RequirementID: "REQ-001", Status: "first"})
		m.Add(ManualValidation{RequirementID: "REQ-001", Status: "second"})

		if len(m.Entries) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(m.Entries))
		}

		// Later entry should override in ByRequirement
		v, ok := m.Get("REQ-001")
		if !ok {
			t.Error("Should find REQ-001")
		}
		if v.Status != "second" {
			t.Errorf("Expected 'second', got %s", v.Status)
		}
	})
}

func TestManualManifest_Get(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		var m *ManualManifest
		_, ok := m.Get("REQ-001")
		if ok {
			t.Error("Get on nil manifest should return false")
		}
	})

	t.Run("nil ByRequirement", func(t *testing.T) {
		m := &ManualManifest{}
		_, ok := m.Get("REQ-001")
		if ok {
			t.Error("Get with nil ByRequirement should return false")
		}
	})

	t.Run("existing entry", func(t *testing.T) {
		m := NewManualManifest("/path")
		m.Add(ManualValidation{RequirementID: "REQ-001", Status: "passed"})

		v, ok := m.Get("REQ-001")
		if !ok || v.Status != "passed" {
			t.Error("Should find REQ-001 with correct status")
		}
	})
}

func TestManualManifest_Count(t *testing.T) {
	tests := []struct {
		name     string
		m        *ManualManifest
		expected int
	}{
		{"nil manifest", nil, 0},
		{"empty manifest", NewManualManifest("/path"), 0},
		{"with entries", &ManualManifest{
			Entries: []ManualValidation{{}, {}},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.m.Count() != tt.expected {
				t.Errorf("Count() = %d, want %d", tt.m.Count(), tt.expected)
			}
		})
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func TestParseError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParseError
		expected string
	}{
		{
			"file only",
			&ParseError{FilePath: "/path/to/file.json", Err: errors.New("bad json")},
			"/path/to/file.json: bad json",
		},
		{
			"with line",
			&ParseError{FilePath: "/path/to/file.json", Line: 10, Err: errors.New("bad json")},
			"/path/to/file.json:10: bad json",
		},
		{
			"with line and column",
			&ParseError{FilePath: "/path/to/file.json", Line: 10, Column: 5, Err: errors.New("bad json")},
			"/path/to/file.json:10:5: bad json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestParseError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &ParseError{FilePath: "/path", Err: inner}

	if !errors.Is(err, inner) {
		t.Error("Unwrap should allow errors.Is to find inner error")
	}
}

func TestNewParseError(t *testing.T) {
	inner := errors.New("test error")
	err := NewParseError("/path/to/file.json", inner)

	if err.FilePath != "/path/to/file.json" {
		t.Errorf("FilePath = %q, want /path/to/file.json", err.FilePath)
	}
	if err.Err != inner {
		t.Error("Err should be the inner error")
	}
}

func TestNewParseErrorAt(t *testing.T) {
	inner := errors.New("test error")
	err := NewParseErrorAt("/path/to/file.json", 42, inner)

	if err.Line != 42 {
		t.Errorf("Line = %d, want 42", err.Line)
	}
}

func TestValidationIssue_Error(t *testing.T) {
	tests := []struct {
		name     string
		issue    *ValidationIssue
		expected string
	}{
		{
			"with requirement ID",
			&ValidationIssue{
				FilePath:      "/path/to/file.json",
				RequirementID: "REQ-001",
				Field:         "title",
				Message:       "missing title",
			},
			"/path/to/file.json: REQ-001 [title]: missing title",
		},
		{
			"without requirement ID",
			&ValidationIssue{
				FilePath: "/path/to/file.json",
				Message:  "invalid format",
			},
			"/path/to/file.json: invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.issue.Error() != tt.expected {
				t.Errorf("Error() = %q, want %q", tt.issue.Error(), tt.expected)
			}
		})
	}
}

func TestValidationIssue_IsError(t *testing.T) {
	tests := []struct {
		name     string
		issue    *ValidationIssue
		expected bool
	}{
		{"nil", nil, false},
		{"error severity", &ValidationIssue{Severity: SeverityError}, true},
		{"warning severity", &ValidationIssue{Severity: SeverityWarning}, false},
		{"info severity", &ValidationIssue{Severity: SeverityInfo}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.issue.IsError() != tt.expected {
				t.Errorf("IsError() = %v, want %v", tt.issue.IsError(), tt.expected)
			}
		})
	}
}

func TestValidationIssue_IsWarning(t *testing.T) {
	tests := []struct {
		name     string
		issue    *ValidationIssue
		expected bool
	}{
		{"nil", nil, false},
		{"error severity", &ValidationIssue{Severity: SeverityError}, false},
		{"warning severity", &ValidationIssue{Severity: SeverityWarning}, true},
		{"info severity", &ValidationIssue{Severity: SeverityInfo}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.issue.IsWarning() != tt.expected {
				t.Errorf("IsWarning() = %v, want %v", tt.issue.IsWarning(), tt.expected)
			}
		})
	}
}

func TestValidationResult_AddAndCount(t *testing.T) {
	result := NewValidationResult()

	result.AddError("/path", "REQ-001", "id", "missing")
	result.AddError("/path", "REQ-002", "title", "empty")
	result.AddWarning("/path", "REQ-003", "status", "deprecated")
	result.AddInfo("/path", "REQ-004", "description", "optional")

	if len(result.Issues) != 4 {
		t.Errorf("Expected 4 issues, got %d", len(result.Issues))
	}
	if result.ErrorCount() != 2 {
		t.Errorf("ErrorCount() = %d, want 2", result.ErrorCount())
	}
	if result.WarningCount() != 1 {
		t.Errorf("WarningCount() = %d, want 1", result.WarningCount())
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := NewValidationResult()
		result.AddWarning("/path", "", "", "warning")
		if result.HasErrors() {
			t.Error("HasErrors() should be false")
		}
	})

	t.Run("has errors", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError("/path", "", "", "error")
		if !result.HasErrors() {
			t.Error("HasErrors() should be true")
		}
	})
}

func TestValidationResult_HasWarnings(t *testing.T) {
	t.Run("no warnings", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError("/path", "", "", "error")
		if result.HasWarnings() {
			t.Error("HasWarnings() should be false")
		}
	})

	t.Run("has warnings", func(t *testing.T) {
		result := NewValidationResult()
		result.AddWarning("/path", "", "", "warning")
		if !result.HasWarnings() {
			t.Error("HasWarnings() should be true")
		}
	})
}

func TestValidationResult_Errors(t *testing.T) {
	result := NewValidationResult()
	result.AddError("/path", "REQ-001", "id", "error1")
	result.AddWarning("/path", "REQ-002", "status", "warning")
	result.AddError("/path", "REQ-003", "title", "error2")

	errs := result.Errors()
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}
}

func TestValidationResult_Warnings(t *testing.T) {
	result := NewValidationResult()
	result.AddWarning("/path", "REQ-001", "id", "warning1")
	result.AddError("/path", "REQ-002", "status", "error")
	result.AddWarning("/path", "REQ-003", "title", "warning2")

	warnings := result.Warnings()
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}
}

func TestValidationResult_Merge(t *testing.T) {
	t.Run("merge nil", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError("/path", "", "", "error")
		result.Merge(nil) // Should not panic
		if len(result.Issues) != 1 {
			t.Error("Merging nil should not change issues")
		}
	})

	t.Run("merge results", func(t *testing.T) {
		r1 := NewValidationResult()
		r1.AddError("/path1", "", "", "error1")

		r2 := NewValidationResult()
		r2.AddWarning("/path2", "", "", "warning")
		r2.AddError("/path2", "", "", "error2")

		r1.Merge(r2)

		if len(r1.Issues) != 3 {
			t.Errorf("Expected 3 issues after merge, got %d", len(r1.Issues))
		}
	})
}

func TestDiscoveryError(t *testing.T) {
	inner := errors.New("permission denied")
	err := NewDiscoveryError("/path/to/requirements", inner)

	expected := "discovery error at /path/to/requirements: permission denied"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	if !errors.Is(err, inner) {
		t.Error("Unwrap should allow errors.Is to find inner error")
	}
}

func TestSyncError(t *testing.T) {
	inner := errors.New("write failed")

	t.Run("without phase", func(t *testing.T) {
		err := NewSyncError("/path/to/file.json", inner)
		expected := "sync error for /path/to/file.json: write failed"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("with phase", func(t *testing.T) {
		err := NewSyncErrorWithPhase("/path/to/file.json", "enrichment", inner)
		expected := "sync error in enrichment for /path/to/file.json: write failed"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := NewSyncError("/path", inner)
		if !errors.Is(err, inner) {
			t.Error("Unwrap should allow errors.Is to find inner error")
		}
	})
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{ErrNoRequirementsDir, true},
		{ErrIndexNotFound, true},
		{ErrMissingReference, true},
		{ErrInvalidJSON, false},
		{ErrDuplicateID, false},
		{errors.New("other"), false},
	}

	for _, tt := range tests {
		if IsNotFound(tt.err) != tt.expected {
			t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, IsNotFound(tt.err), tt.expected)
		}
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{ErrDuplicateID, true},
		{ErrMissingID, true},
		{ErrCycleDetected, true},
		{ErrInvalidImport, true},
		{ErrInvalidJSON, false},
		{ErrNoRequirementsDir, false},
		{errors.New("other"), false},
	}

	for _, tt := range tests {
		if IsValidationError(tt.err) != tt.expected {
			t.Errorf("IsValidationError(%v) = %v, want %v", tt.err, IsValidationError(tt.err), tt.expected)
		}
	}
}

func TestIsParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ParseError type", NewParseError("/path", errors.New("bad")), true},
		{"ErrInvalidJSON", ErrInvalidJSON, true},
		{"wrapped ParseError", &DiscoveryError{
			Path: "/path",
			Err:  NewParseError("/path", errors.New("bad")),
		}, true},
		{"other error", ErrDuplicateID, false},
		{"plain error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsParseError(tt.err) != tt.expected {
				t.Errorf("IsParseError(%v) = %v, want %v", tt.err, IsParseError(tt.err), tt.expected)
			}
		})
	}
}
