package claude

import (
	"strings"

	"web-console/internal/backend"
)

// PromptDetector is the default heuristic detector for "claude-code is at
// an input prompt". Claude renders an Unicode-bordered input box; the
// cheap signal is the box-drawing prompt glyph "❯" (U+276F) appearing on
// the cursor row, paired with the alt-buffer footer hint. Future
// agent-driver work can register a more precise detector; this one is
// the contract shape, not the final accuracy bar.
type promptDetector struct{}

// DefaultPromptDetector returns the heuristic detector used by the claude
// backend descriptor when no override is registered.
func DefaultPromptDetector() backend.PromptDetector { return promptDetector{} }

const claudePromptGlyph = "❯"

func (promptDetector) IsAwaitingInput(view backend.ScreenView) bool {
	if view == nil {
		return false
	}
	text := view.PlainText()
	if text == "" {
		return false
	}
	// Quick rejection: no prompt glyph anywhere on screen.
	if !strings.Contains(text, claudePromptGlyph) {
		return false
	}
	// The prompt row is where the cursor lives when claude is waiting.
	// Split into rows and check the cursor row carries the glyph.
	rows := strings.Split(text, "\n")
	cur := view.CursorRow()
	if cur < 0 || cur >= len(rows) {
		return false
	}
	return strings.Contains(rows[cur], claudePromptGlyph)
}
