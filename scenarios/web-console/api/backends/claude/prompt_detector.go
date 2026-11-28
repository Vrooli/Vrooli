package claude

import (
	"regexp"
	"strings"

	"web-console/internal/backend"
)

// promptDetector reads a Claude Code screen. Claude Code keeps its input box
// (a row starting with the "❯" glyph) on screen even while it works, so the
// glyph alone does not mean idle: the in-progress marker "esc to interrupt"
// separates working from waiting at the prompt. A question or permission box
// (options above an "Enter to confirm/select" hint) means it is asking.
//
// Fixtures captured from live Claude Code 2.1.x screens live in
// testdata/prompts and are the contract for changes here.
type promptDetector struct{}

// DefaultPromptDetector returns the Claude Code screen detector.
func DefaultPromptDetector() backend.PromptDetector { return promptDetector{} }

const (
	claudePromptGlyph   = "❯"
	claudeWorkingMarker = "esc to interrupt"
	// claudeWorkingScanLines bounds how far above the bottom the in-progress
	// line is looked for; it sits just above the input box.
	claudeWorkingScanLines = 12
)

// claudeWorkingLine matches the in-progress line 2.1.268 draws above the input
// box — a spinner glyph, a gerund with an ellipsis, and the elapsed time:
// "✽ Effecting… (3h 14m 29s · ↓ 655.6k tokens)". The finished form
// ("✻ Cooked for 0s · done 11:04 PM") has no ellipsis and does not match.
var claudeWorkingLine = regexp.MustCompile(`^\s*[^\s\w]\s+[^\n(]{1,40}…\s*\(\s*\d+[hms]`)

// working reports whether the bottom of the screen shows Claude working.
func working(lines []string) bool {
	tail, _ := visibleTail(lines, claudeWorkingScanLines)
	for _, line := range tail {
		if claudeWorkingLine.MatchString(line) || strings.Contains(strings.ToLower(line), claudeWorkingMarker) {
			return true
		}
	}
	return false
}

func (promptDetector) Analyze(view backend.ScreenView) backend.PromptAnalysis {
	if view == nil {
		return backend.PromptAnalysis{}
	}
	text := view.PlainText()
	if text == "" {
		return backend.PromptAnalysis{}
	}
	lines := strings.Split(text, "\n")
	if prompt := parsePromptBox(lines, view.CursorRow()); prompt != nil {
		return backend.PromptAnalysis{Prompt: prompt, Confidence: 0.85}
	}
	if working(lines) {
		return backend.PromptAnalysis{Working: true, Confidence: 0.85}
	}
	cur := view.CursorRow()
	if cur >= 0 && cur < len(lines) && strings.Contains(lines[cur], claudePromptGlyph) {
		return backend.PromptAnalysis{AwaitingInput: true, Confidence: 0.75}
	}
	return backend.PromptAnalysis{}
}
