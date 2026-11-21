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
	if got := DefaultPromptDetector().Analyze(nil); got.AwaitingInput || got.Prompt != nil || got.Working {
		t.Errorf("nil view analysis = %+v, want nothing", got)
	}
}

func TestDefaultPromptDetector_GlyphOnCursorRow(t *testing.T) {
	// Two rows; cursor on row 1 which carries the prompt glyph.
	got := DefaultPromptDetector().Analyze(fakeView{text: "banner\n❯ ", row: 1})
	if !got.AwaitingInput {
		t.Errorf("expected awaiting input with the glyph on the cursor row, got %+v", got)
	}
}

func TestDefaultPromptDetector_GlyphOffCursorRow(t *testing.T) {
	// Glyph exists but on a different row than the cursor.
	if got := DefaultPromptDetector().Analyze(fakeView{text: "❯ old prompt\nstatus row", row: 1}); got.AwaitingInput {
		t.Errorf("glyph off the cursor row must not read as awaiting input, got %+v", got)
	}
}

func TestDefaultPromptDetector_NoGlyph(t *testing.T) {
	if got := DefaultPromptDetector().Analyze(fakeView{text: "running thing\nstill working", row: 1}); got.AwaitingInput {
		t.Errorf("no glyph means not awaiting input, got %+v", got)
	}
}

func TestDefaultPromptDetector_WorkingMarkerOverridesGlyph(t *testing.T) {
	// Claude Code keeps its input row visible while it works.
	got := DefaultPromptDetector().Analyze(fakeView{text: "✻ Thinking… (esc to interrupt)\n❯ ", row: 1})
	if !got.Working || got.AwaitingInput {
		t.Errorf("working marker must win over the glyph, got %+v", got)
	}
}

// Compile-time assertion that the package-local detector satisfies the
// backend.PromptDetector contract.
var _ backend.PromptDetector = DefaultPromptDetector()
