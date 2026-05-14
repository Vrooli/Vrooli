package backend

import "testing"

// fakeView is a minimal ScreenView for nil-safety probing only — it has
// no production callers and intentionally returns trivial values.
type fakeView struct {
	text string
	row  int
}

func (f fakeView) Cols() int         { return 80 }
func (f fakeView) Rows() int         { return 24 }
func (f fakeView) CursorRow() int    { return f.row }
func (f fakeView) CursorCol() int    { return 0 }
func (f fakeView) PlainText() string { return f.text }

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
