package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"brand-manager/domain"
)

// Unit tests for decision boundary functions in the handlers package.
// [REQ:BM-REQ-APPLY-PARTIAL] [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-CRUD-CREATE]

func TestIsDryRun_True(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Dry-Run", "true")
	if !isDryRun(req) {
		t.Error("expected isDryRun = true")
	}
}

func TestIsDryRun_False(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	if isDryRun(req) {
		t.Error("expected isDryRun = false without header")
	}
}

func TestIsDryRun_WrongValue(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Dry-Run", "yes")
	if isDryRun(req) {
		t.Error("expected isDryRun = false for non-'true' value")
	}
}

func TestDryRunResponse_ContainsMarkers(t *testing.T) {
	input := domain.Brand{ID: "b1", Name: "Test"}
	result := dryRunResponse(input)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}
	if result["success"] != true {
		t.Error("expected success = true")
	}
	if result["id"] != "b1" {
		t.Errorf("id = %v, want b1", result["id"])
	}
}

func TestDryRunResponse_NilSafe(t *testing.T) {
	result := dryRunResponse(nil)
	if result["dry_run"] != true {
		t.Error("expected dry_run = true even for nil input")
	}
}

func TestApplyElement_KnownElements(t *testing.T) {
	h := &Handlers{}
	brand := &domain.Brand{
		ID:      "b1",
		Colors:  &domain.Colors{Primary: "#ff0000"},
		Version: 1,
	}

	tests := []struct {
		element     string
		expectApply bool
	}{
		{"colors", true},
		{"typography", false}, // nil typography → skip
		{"identity", false},   // nil identity → skip
		{"favicon", false},    // nil identity → skip
		{"logo", false},       // nil identity → skip
	}

	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			actions, skip := h.applyElement(brand, t.TempDir(), tt.element, true)
			if tt.expectApply {
				if len(actions) == 0 {
					t.Error("expected actions for element")
				}
				if skip != nil {
					t.Errorf("unexpected skip: %v", skip.Reason)
				}
			} else {
				if skip == nil {
					t.Error("expected skip reason for missing data")
				}
			}
		})
	}
}

func TestApplyElement_UnknownElement(t *testing.T) {
	h := &Handlers{}
	brand := &domain.Brand{ID: "b1", Version: 1}

	_, skip := h.applyElement(brand, t.TempDir(), "unknown_thing", true)
	if skip == nil {
		t.Fatal("expected skip for unknown element")
	}
	if skip.Element != "unknown_thing" {
		t.Errorf("skip.Element = %q, want unknown_thing", skip.Element)
	}
	if !strings.Contains(skip.Reason, "unknown") {
		t.Errorf("skip.Reason = %q, want to contain 'unknown'", skip.Reason)
	}
}

func TestAllApplyElements_Complete(t *testing.T) {
	expected := []string{"colors", "typography", "identity", "favicon", "logo"}
	if len(allApplyElements) != len(expected) {
		t.Fatalf("allApplyElements has %d items, want %d", len(allApplyElements), len(expected))
	}
	for i, want := range expected {
		if allApplyElements[i] != want {
			t.Errorf("allApplyElements[%d] = %q, want %q", i, allApplyElements[i], want)
		}
	}
}

func TestGenerateCSSBlock_EmptyPairs(t *testing.T) {
	css := generateCSSBlock("test", nil)
	if !strings.Contains(css, "brand-manager:test") {
		t.Error("missing section header")
	}
	if !strings.Contains(css, ":root {") {
		t.Error("missing :root selector")
	}
	if strings.Contains(css, "--brand-") {
		t.Error("no variables should be emitted for nil pairs")
	}
}

func TestGenerateCSSBlock_SkipsEmptyValues(t *testing.T) {
	pairs := []nameValue{
		{"filled", "value"},
		{"empty", ""},
		{"also-filled", "v2"},
	}
	css := generateCSSBlock("section", pairs)

	if !strings.Contains(css, "--brand-filled: value") {
		t.Error("missing filled variable")
	}
	if strings.Contains(css, "--brand-empty") {
		t.Error("empty value should be skipped")
	}
	if !strings.Contains(css, "--brand-also-filled: v2") {
		t.Error("missing also-filled variable")
	}
}
