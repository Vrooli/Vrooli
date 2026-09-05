package codex

import (
	"strings"

	"web-console/internal/backend"
)

// PromptDetector is the default heuristic detector for "codex is at an
// input prompt". Codex's TUI renders a "user:" or "▶" indicator on the
// last interactive row; this is a low-precision starting point that
// future agent-driver work will refine.
type promptDetector struct{}

// DefaultPromptDetector returns the heuristic detector used by the codex
// CLI when wired onto a backend descriptor.
func DefaultPromptDetector() backend.PromptDetector { return promptDetector{} }

func (promptDetector) IsAwaitingInput(view backend.ScreenView) bool {
	if view == nil {
		return false
	}
	text := view.PlainText()
	if text == "" {
		return false
	}
	rows := strings.Split(text, "\n")
	cur := view.CursorRow()
	if cur < 0 || cur >= len(rows) {
		return false
	}
	row := rows[cur]
	return strings.Contains(row, "user:") || strings.Contains(row, "▶")
}
