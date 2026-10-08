package codex

import (
	"regexp"
	"strings"

	"web-console/internal/backend"
)

// promptDetector reads a Codex CLI screen at level 1: working, at its input
// prompt, or showing a numbered choice (an approval) it cannot yet parse.
// Codex 0.15x draws its input row as "› ..." and marks work in progress with
// "esc to interrupt". Fixtures captured from live screens live in
// testdata/prompts.
type promptDetector struct{}

// DefaultPromptDetector returns the Codex CLI screen detector.
func DefaultPromptDetector() backend.PromptDetector { return promptDetector{} }

const (
	codexPromptGlyph   = "›"
	codexWorkingMarker = "esc to interrupt"
)

var codexCapacityMarkers = []string{
	"selected model is at capacity",
	"model is at capacity",
	"temporarily at capacity",
	"try a different model",
}

// A selected numbered option, e.g. "› 1. Yes, proceed".
var codexSelectedOption = regexp.MustCompile(`^\s*[›❯>▌]\s*1\.\s+\S`)

func (promptDetector) Analyze(view backend.ScreenView) backend.PromptAnalysis {
	if view == nil {
		return backend.PromptAnalysis{}
	}
	text := view.PlainText()
	if text == "" {
		return backend.PromptAnalysis{}
	}
	lower := strings.ToLower(text)
	for _, marker := range codexCapacityMarkers {
		if strings.Contains(lower, marker) {
			return backend.PromptAnalysis{Interruption: &backend.ResumableInterruption{
				Kind: "model_capacity", Message: marker,
			}, Confidence: 0.95}
		}
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines[max(0, len(lines)-24):] {
		if codexSelectedOption.MatchString(line) {
			return backend.PromptAnalysis{Prompt: &backend.PendingPrompt{Kind: "unknown"}, Confidence: 0.75}
		}
	}
	if strings.Contains(strings.ToLower(text), codexWorkingMarker) {
		return backend.PromptAnalysis{Working: true, Confidence: 0.8}
	}
	cur := view.CursorRow()
	if cur < 0 || cur >= len(lines) {
		return backend.PromptAnalysis{}
	}
	row := strings.TrimSpace(lines[cur])
	if strings.HasPrefix(row, codexPromptGlyph) {
		return backend.PromptAnalysis{AwaitingInput: true, Confidence: 0.75}
	}
	return backend.PromptAnalysis{}
}
