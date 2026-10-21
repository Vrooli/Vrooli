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

// VoiceObservation is mode discovery only: a footer cannot prove browser audio
// capture. Restrict it to the live input's neighboring rows, never scrollback.
type VoiceObservation struct {
	Active bool
	Muted  bool
}

func ObserveVoice(view backend.ScreenView) VoiceObservation {
	if view == nil {
		return VoiceObservation{}
	}
	lines := strings.Split(view.PlainText(), "\n")
	cursor := view.CursorRow()
	if cursor < 0 || cursor >= len(lines) {
		return VoiceObservation{}
	}
	input := strings.TrimSpace(lines[cursor])
	if !strings.HasPrefix(input, "›") && !strings.HasPrefix(input, ">") {
		return VoiceObservation{}
	}
	for i := max(0, cursor-6); i < cursor; i++ {
		line := strings.ToLower(strings.TrimSpace(codexMainPaneText(lines[i])))
		if !strings.HasPrefix(line, "voice ") || !strings.Contains(line, "/voice stop") {
			continue
		}
		if strings.Contains(line, "listening") || strings.Contains(line, "muted") || strings.Contains(line, "connecting") {
			return VoiceObservation{Active: true, Muted: strings.Contains(line, "muted")}
		}
	}
	return VoiceObservation{}
}

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
var codexGoalStalledFooter = regexp.MustCompile(`(?i)^.+\bgoal stalled \(\s*/goal resume\s*\)$`)
var codexShortcutFooter = regexp.MustCompile(`(?i)^\?\s*for shortcuts\b`)

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
	if codexGoalStalledVisibleFooter(view, lines) {
		return backend.PromptAnalysis{Interruption: &backend.ResumableInterruption{
			Kind: "goal_stalled", Message: "Goal stalled (/goal resume)",
		}, Confidence: 0.95}
	}
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

// codexGoalStalledVisibleFooter recognizes the current Codex footer, not text
// in the transcript. The stalled status must occupy the row between the
// input prompt and Codex's shortcut footer; a matching phrase elsewhere on
// screen is historical or quoted content and is ignored.
func codexGoalStalledVisibleFooter(view backend.ScreenView, lines []string) bool {
	// The visible Codex footer sits below the input row. A blank row separates
	// the input from the model/status row; the shortcuts row follows. Anchor to
	// the active cursor row so transcript text elsewhere cannot look like a
	// current footer. Split the Web Console side-pane divider before matching:
	// that pane can render beside every main-terminal row.
	cursor := view.CursorRow()
	if cursor < 0 || cursor+3 >= len(lines) {
		return false
	}
	promptRow := codexMainPaneText(lines[cursor])
	spacerRow := codexMainPaneText(lines[cursor+1])
	statusRow := codexMainPaneText(lines[cursor+2])
	shortcutRow := codexMainPaneText(lines[cursor+3])
	return strings.HasPrefix(promptRow, codexPromptGlyph) && spacerRow == "" &&
		codexGoalStalledFooter.MatchString(statusRow) &&
		codexShortcutFooter.MatchString(shortcutRow)
}

func codexMainPaneText(line string) string {
	if divider := strings.IndexRune(line, '│'); divider >= 0 {
		line = line[:divider]
	}
	return strings.TrimSpace(line)
}
