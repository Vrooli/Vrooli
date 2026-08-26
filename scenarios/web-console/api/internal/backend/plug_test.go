package backend

import "testing"

// TestDescriptor_PlugPointsNilByDefault asserts the new optional plug
// points stay nil on a zero-valued descriptor — callers must check before
// invoking. If this regresses (e.g. a default-initialized struct field
// silently turns non-nil), every caller would need a re-audit.
func TestDescriptor_PlugPointsNilByDefault(t *testing.T) {
	var d Descriptor
	if d.KeyMap != nil {
		t.Errorf("KeyMap default should be nil, got %T", d.KeyMap)
	}
	if d.PromptDetector != nil {
		t.Errorf("PromptDetector default should be nil, got %T", d.PromptDetector)
	}
	if d.IdleHeuristic != nil {
		t.Errorf("IdleHeuristic default should be nil, got %T", d.IdleHeuristic)
	}
}
