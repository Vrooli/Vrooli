package claude

import (
	"testing"

	"web-console/internal/backend"
)

type fakeView struct {
	text string
	row  int
}

func (f fakeView) Cols() int         { return 80 }
func (f fakeView) Rows() int         { return 24 }
func (f fakeView) CursorRow() int    { return f.row }
func (f fakeView) CursorCol() int    { return 0 }
func (f fakeView) PlainText() string { return f.text }

func TestDefaultPromptDetector_NilView(t *testing.T) {
	if DefaultPromptDetector().IsAwaitingInput(nil) {
		t.Error("nil view must not be classified as awaiting input")
	}
}

func TestDefaultPromptDetector_GlyphOnCursorRow(t *testing.T) {
	// Two rows; cursor on row 1 which carries the prompt glyph.
	view := fakeView{text: "banner\n❯ ", row: 1}
	if !DefaultPromptDetector().IsAwaitingInput(view) {
		t.Error("expected awaiting-input on cursor row with ❯ glyph")
	}
}

func TestDefaultPromptDetector_GlyphOffCursorRow(t *testing.T) {
	// Glyph exists but on a different row than the cursor.
	view := fakeView{text: "❯ old prompt\nstatus row", row: 1}
	if DefaultPromptDetector().IsAwaitingInput(view) {
		t.Error("glyph not on cursor row should not classify as awaiting input")
	}
}

func TestDefaultPromptDetector_NoGlyph(t *testing.T) {
	view := fakeView{text: "running thing\nstill working", row: 1}
	if DefaultPromptDetector().IsAwaitingInput(view) {
		t.Error("no glyph means not awaiting input")
	}
}

// Compile-time assertion that the package-local detector satisfies the
// backend.PromptDetector contract.
var _ backend.PromptDetector = DefaultPromptDetector()
