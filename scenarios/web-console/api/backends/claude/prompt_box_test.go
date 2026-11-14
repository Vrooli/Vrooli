package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"web-console/internal/backend"
	"web-console/terminal"
)

// fixtureScreen is a screen captured from a live session with
// `web-console terminal screen <id> --json`.
type fixtureScreen struct {
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
	CursorX   int    `json:"cursor_x"`
	CursorY   int    `json:"cursor_y"`
	PlainText string `json:"plain_text"`
}

// decodedView implements backend.ScreenView over an emulator's decoded screen.
type decodedView struct {
	view terminal.ScreenView
	text string
}

func (v decodedView) Cols() int         { return v.view.Cols }
func (v decodedView) Rows() int         { return v.view.Rows }
func (v decodedView) CursorRow() int    { return v.view.Cursor.Y }
func (v decodedView) CursorCol() int    { return v.view.Cursor.X }
func (v decodedView) PlainText() string { return v.text }

// loadFixture replays a captured screen through the terminal emulator so the
// detector reads a decoded screen, not a hand-typed string.
func loadFixture(t *testing.T, name string) backend.ScreenView {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "prompts", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var screen fixtureScreen
	if err := json.Unmarshal(raw, &screen); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	emu := terminal.New(terminal.Options{Cols: screen.Cols, Rows: screen.Rows})
	lines := strings.Split(strings.TrimRight(screen.PlainText, "\n"), "\n")
	if len(lines) > screen.Rows {
		lines = lines[len(lines)-screen.Rows:]
	}
	payload := "\x1b[H" + strings.Join(lines, "\r\n") + fmt.Sprintf("\x1b[%d;%dH", screen.CursorY+1, screen.CursorX+1)
	if _, err := emu.Feed([]byte(payload)); err != nil {
		t.Fatalf("feed fixture %s: %v", name, err)
	}
	return decodedView{view: emu.View(), text: emu.PlainText(false)}
}

func TestClaudeWorkspaceTrustDialogParsesAsPermission(t *testing.T) {
	// [REQ:P0-017i] A live Claude Code 2.1.268 trust dialog: unnumbered options
	// above "Enter to confirm · Esc to cancel".
	got := DefaultPromptDetector().Analyze(loadFixture(t, "trust.json"))
	if got.Prompt == nil {
		t.Fatalf("trust dialog was not recognized as a prompt: %+v", got)
	}
	if got.Prompt.Kind != "permission" {
		t.Fatalf("kind = %q, want permission", got.Prompt.Kind)
	}
	if !strings.Contains(got.Prompt.Text, "trust") {
		t.Fatalf("question text = %q, want the trust question", got.Prompt.Text)
	}
	want := []backend.PromptOption{{Key: "1", Label: "No, exit", Selected: true}, {Key: "2", Label: "Yes, I trust this folder"}}
	if len(got.Prompt.Options) != len(want) {
		t.Fatalf("options = %+v, want %+v", got.Prompt.Options, want)
	}
	for i := range want {
		if got.Prompt.Options[i] != want[i] {
			t.Fatalf("option %d = %+v, want %+v", i, got.Prompt.Options[i], want[i])
		}
	}
	if got.AwaitingInput || got.Working || got.Confidence < 0.8 {
		t.Fatalf("prompt analysis = %+v, want a confident prompt only", got)
	}
}

func TestClaudeIdlePromptIsAwaitingInput(t *testing.T) {
	// [REQ:P0-017e] A live idle Claude Code screen: the input row carries the
	// prompt glyph, no prompt box, no in-progress marker.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "idle.json"))
	if !got.AwaitingInput || got.Prompt != nil || got.Working {
		t.Fatalf("idle screen analysis = %+v, want awaiting input", got)
	}
}

func TestClaudeLiveWorkingScreenIsWorking(t *testing.T) {
	// [REQ:P0-017e] A live 2.1.268 screen mid-turn: the input row still shows
	// "❯" and there is no "esc to interrupt" anywhere; the in-progress line
	// "✽ Effecting… (3h 14m 29s · ↓ 655.6k tokens)" decides. Failing value
	// before the fix: {AwaitingInput:true Working:false}, read as idle.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "working.json"))
	if !got.Working || got.AwaitingInput || got.Prompt != nil {
		t.Fatalf("live working screen analysis = %+v, want working", got)
	}
}

func TestClaudeFinishedTurnLineIsNotWorking(t *testing.T) {
	// [REQ:P0-017e] The finished form of the same line ("✻ Cooked for 0s ·
	// done 11:04 PM") above an empty input row is idle.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "idle_live.json"))
	if got.Working {
		t.Fatalf("finished-turn screen read as working: %+v", got)
	}
}

func TestClaudeWorkingMarkerIsNotIdle(t *testing.T) {
	// [REQ:P0-017e] The input box stays on screen while Claude works; the
	// in-progress marker decides.
	lines := make([]string, 0, 12)
	lines = append(lines, "● Reading files", "", "✻ Thinking… (12s · ↓ 1.2k tokens · esc to interrupt)", "", strings.Repeat("─", 40), "❯ ", strings.Repeat("─", 40))
	got := DefaultPromptDetector().Analyze(textView{text: strings.Join(lines, "\n"), cursor: 5})
	if !got.Working || got.AwaitingInput {
		t.Fatalf("working screen analysis = %+v, want working", got)
	}
}

func TestClaudeQuestionBoxParsesWithoutHint(t *testing.T) {
	// [REQ:P0-017i] A live Claude Code 2.1.268 AskUserQuestion box: no hint
	// line, the cursor on the selected numbered option, descriptions indented
	// under each option, and a rule before the last option.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "question.json"))
	if got.Prompt == nil {
		t.Fatalf("question box was not recognized: %+v", got)
	}
	if got.Prompt.Kind != "question" {
		t.Fatalf("kind = %q, want question", got.Prompt.Kind)
	}
	if !strings.HasPrefix(got.Prompt.Text, "Empty dir") || !strings.Contains(got.Prompt.Text, "Should I rmdir it") {
		t.Fatalf("question text = %q, want the header and the question", got.Prompt.Text)
	}
	want := []backend.PromptOption{
		{Key: "1", Label: "rmdir and write file (Recommended)", Selected: true},
		{Key: "2", Label: "Leave it, stop"},
		{Key: "3", Label: "Type something."},
		{Key: "4", Label: "Chat about this"},
	}
	if len(got.Prompt.Options) != len(want) {
		t.Fatalf("options = %+v, want %+v", got.Prompt.Options, want)
	}
	for i := range want {
		if got.Prompt.Options[i] != want[i] {
			t.Fatalf("option %d = %+v, want %+v", i, got.Prompt.Options[i], want[i])
		}
	}
	if got.Prompt.FreeTextHint != "Type something." {
		t.Fatalf("free-text hint = %q, want the Type something option", got.Prompt.FreeTextHint)
	}
}

func TestClaudeShellPermissionBoxParsesAsPermission(t *testing.T) {
	// [REQ:P0-017i] A live Claude Code 2.1.268 Bash permission box (default
	// permission mode): the tool call and command above numbered options, a
	// hint line below.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "permission_bash.json"))
	if got.Prompt == nil {
		t.Fatalf("permission box was not recognized: %+v", got)
	}
	if got.Prompt.Kind != "permission" {
		t.Fatalf("kind = %q, want permission", got.Prompt.Kind)
	}
	if !strings.Contains(got.Prompt.Text, "touch perm-probe.txt") || !strings.HasSuffix(got.Prompt.Text, "Do you want to proceed?") {
		t.Fatalf("permission text = %q, want the command and the question", got.Prompt.Text)
	}
	options := got.Prompt.Options
	if len(options) != 4 || options[0] != (backend.PromptOption{Key: "1", Label: "Yes", Selected: true}) || options[3] != (backend.PromptOption{Key: "4", Label: "No"}) {
		t.Fatalf("options = %+v, want Yes (selected) through No", options)
	}
	if got.Prompt.FreeTextHint != "" {
		t.Fatalf("free-text hint = %q, want none", got.Prompt.FreeTextHint)
	}
}

func TestClaudeTwoOptionQuestionParsesWithFreeText(t *testing.T) {
	// [REQ:P0-017i] A live AskUserQuestion box with two listed options, the
	// free-text row, and "Chat about this" below a rule.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "question_color.json"))
	if got.Prompt == nil || got.Prompt.Kind != "question" {
		t.Fatalf("question box analysis = %+v, want a question", got)
	}
	if got.Prompt.Text != "Color\n\nWhich color do you prefer?" {
		t.Fatalf("question text = %q", got.Prompt.Text)
	}
	want := []backend.PromptOption{
		{Key: "1", Label: "Red", Selected: true},
		{Key: "2", Label: "Blue"},
		{Key: "3", Label: "Type something."},
		{Key: "4", Label: "Chat about this"},
	}
	if len(got.Prompt.Options) != len(want) {
		t.Fatalf("options = %+v, want %+v", got.Prompt.Options, want)
	}
	for i := range want {
		if got.Prompt.Options[i] != want[i] {
			t.Fatalf("option %d = %+v, want %+v", i, got.Prompt.Options[i], want[i])
		}
	}
	if got.Prompt.FreeTextHint != "Type something." {
		t.Fatalf("free-text hint = %q", got.Prompt.FreeTextHint)
	}
}

func TestClaudePromptCancellableFollowsItsHint(t *testing.T) {
	// [REQ:P0-017i] A box whose hint offers "Esc to cancel" can be dismissed
	// from Messages; the AskUserQuestion box has no hint and cannot.
	trust := DefaultPromptDetector().Analyze(loadFixture(t, "trust.json"))
	if trust.Prompt == nil || !trust.Prompt.Cancellable {
		t.Fatalf("trust dialog = %+v, want a cancellable prompt", trust.Prompt)
	}
	question := DefaultPromptDetector().Analyze(loadFixture(t, "question.json"))
	if question.Prompt == nil || question.Prompt.Cancellable {
		t.Fatalf("question box = %+v, want a prompt that is not cancellable", question.Prompt)
	}
}

func TestClaudeLiveIdleScreenIsAwaitingInput(t *testing.T) {
	// [REQ:P0-017e] A live idle screen whose input row is "❯" plus a no-break
	// space, between two rules, after a finished turn.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "idle_live.json"))
	if !got.AwaitingInput || got.Prompt != nil || got.Working {
		t.Fatalf("live idle screen analysis = %+v, want awaiting input", got)
	}
}

func TestClaudeBoxWithoutHintIsNotParsed(t *testing.T) {
	// [REQ:P0-017i] Without a hint line, a box is read only when the cursor
	// sits on its selected option; anything else is not guessed at.
	lines := []string{" Do you want to proceed?", " ❯ 1. Yes", "   2. No"}
	got := DefaultPromptDetector().Analyze(textView{text: strings.Join(lines, "\n"), cursor: 2})
	if got.Prompt != nil {
		t.Fatalf("truncated box parsed as %+v, want no prompt", got.Prompt)
	}
}

func TestClaudeQuestionBoxCutAtTheBottomDegradesToAwaitingInput(t *testing.T) {
	// [REQ:P0-017i] The live question box drawn only down to option 2: the
	// free-text row and "Chat about this" are not on screen yet. Reading the
	// two visible options as the whole question would be a wrong parse; the
	// detector says "asking you something" (level 1) instead.
	view := loadFixtureCut(t, "question.json", 42)
	got := DefaultPromptDetector().Analyze(view)
	if got.Prompt != nil {
		t.Fatalf("cut box parsed as %+v, want no prompt", got.Prompt)
	}
	if !got.AwaitingInput || got.Confidence != 0.75 {
		t.Fatalf("cut box analysis = %+v, want awaiting input at 0.75", got)
	}
}

// loadFixtureCut replays a fixture keeping only its first keep lines, as a
// screen caught mid-draw; the cursor stays where the fixture put it.
func loadFixtureCut(t *testing.T, name string, keep int) backend.ScreenView {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "prompts", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var screen fixtureScreen
	if err := json.Unmarshal(raw, &screen); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	lines := strings.Split(strings.TrimRight(screen.PlainText, "\n"), "\n")
	if keep < len(lines) {
		lines = lines[:keep]
	}
	emu := terminal.New(terminal.Options{Cols: screen.Cols, Rows: screen.Rows})
	payload := "\x1b[H" + strings.Join(lines, "\r\n") + fmt.Sprintf("\x1b[%d;%dH", screen.CursorY+1, screen.CursorX+1)
	if _, err := emu.Feed([]byte(payload)); err != nil {
		t.Fatalf("feed fixture %s: %v", name, err)
	}
	return decodedView{view: emu.View(), text: emu.PlainText(false)}
}

type textView struct {
	text   string
	cursor int
}

func (v textView) Cols() int         { return 80 }
func (v textView) Rows() int         { return 24 }
func (v textView) CursorRow() int    { return v.cursor }
func (v textView) CursorCol() int    { return 0 }
func (v textView) PlainText() string { return v.text }
