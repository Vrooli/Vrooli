package codex

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

type fixtureScreen struct {
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
	CursorX   int    `json:"cursor_x"`
	CursorY   int    `json:"cursor_y"`
	PlainText string `json:"plain_text"`
}

type decodedView struct {
	view terminal.ScreenView
	text string
}

func (v decodedView) Cols() int         { return v.view.Cols }
func (v decodedView) Rows() int         { return v.view.Rows }
func (v decodedView) CursorRow() int    { return v.view.Cursor.Y }
func (v decodedView) CursorCol() int    { return v.view.Cursor.X }
func (v decodedView) PlainText() string { return v.text }

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

func TestCodexIdlePromptIsAwaitingInput(t *testing.T) {
	// [REQ:P0-017e] codex-cli 0.153.4 at its input row: "› Ask Codex to do anything".
	got := DefaultPromptDetector().Analyze(loadFixture(t, "idle.json"))
	if !got.AwaitingInput || got.Prompt != nil {
		t.Fatalf("idle codex screen = %+v, want awaiting input", got)
	}
}

func TestCodexStreamingScreenIsNotIdle(t *testing.T) {
	// [REQ:P0-017e] A live codex screen mid-turn (cursor off the input row).
	got := DefaultPromptDetector().Analyze(loadFixture(t, "working.json"))
	if got.AwaitingInput {
		t.Fatalf("streaming codex screen read as idle: %+v", got)
	}
}

func TestCodexLiveWorkingScreenIsWorking(t *testing.T) {
	// [REQ:P0-017e] A live codex-cli 0.153.4 screen mid-turn: the cursor rests
	// on the input row ("› Ask Codex to do anything") while "• Working (… esc
	// to interrupt)" sits above it. The marker decides, not the glyph.
	got := DefaultPromptDetector().Analyze(loadFixture(t, "working_live.json"))
	if !got.Working || got.AwaitingInput || got.Prompt != nil {
		t.Fatalf("live working codex screen = %+v, want working", got)
	}
}

func TestCodexNumberedChoiceIsDetectedNotParsed(t *testing.T) {
	// [REQ:P0-017i] Codex approvals ship at level 1: detected, content not parsed.
	text := strings.Join([]string{"Would you like to run the following command?", "  $ go test ./...", "› 1. Yes, proceed", "  2. No, and tell Codex what to do differently"}, "\n")
	got := DefaultPromptDetector().Analyze(stubView{text: text, cursor: 2})
	if got.Prompt == nil || got.Prompt.Kind != "unknown" || got.Prompt.Text != "" {
		t.Fatalf("codex approval = %+v, want an unparsed prompt", got)
	}
}

type stubView struct {
	text   string
	cursor int
}

func (v stubView) Cols() int         { return 80 }
func (v stubView) Rows() int         { return 24 }
func (v stubView) CursorRow() int    { return v.cursor }
func (v stubView) CursorCol() int    { return 0 }
func (v stubView) PlainText() string { return v.text }
