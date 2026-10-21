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

func TestCodexVoiceObservationUsesLiveFooter(t *testing.T) {
	for _, test := range []struct {
		text   string
		cursor int
		active bool
	}{
		{"Earlier: voice listening /voice stop\n\n› Ask Codex", 2, false},
		{"voice ● listening    ^x mute    /voice stop\n mic ___ codex ___\n\n› Ask Codex", 3, true},
		{"voice muted    /voice stop\n\n› Ask Codex", 2, true},
		{"voice listening /voice stop\n\nquoted text", 2, false},
		{"voice listening /voice stop\na\nb\nc\nd\ne\nf\n› Ask Codex", 7, false},
	} {
		got := ObserveVoice(stubView{text: test.text, cursor: test.cursor})
		if got.Active != test.active {
			t.Fatalf("voice observation=%+v for %q", got, test.text)
		}
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

func TestCodexCapacityInterruptionIsResumable(t *testing.T) {
	got := DefaultPromptDetector().Analyze(stubView{text: "Selected model is at capacity. Please try a different model."})
	if got.Interruption == nil || got.Interruption.Kind != "model_capacity" || got.Confidence < 0.9 {
		t.Fatalf("capacity screen = %+v, want high-confidence model_capacity interruption", got)
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

func TestCodexVisibleStalledGoalFooterIsResumable(t *testing.T) {
	// This is the retained redacted A135 40x120 screen. A later bounded safe
	// read of this same worker confirmed the input cursor at x=2, y=19. The
	// historical fixture omits coordinates; use the confirmed current row and
	// column because its footer neighborhood is unchanged.
	got := DefaultPromptDetector().Analyze(loadCapturedStalledGoalScreen(t))
	if got.Interruption == nil || got.Interruption.Kind != "goal_stalled" || got.Confidence < 0.9 {
		t.Fatalf("stalled goal footer = %+v, want high-confidence goal_stalled interruption", got)
	}
}

func loadCapturedStalledGoalScreen(t *testing.T) backend.ScreenView {
	t.Helper()
	const text = "                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  The latest vmstat interval showed 3,552   \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  pages/s of swap-in (so=0), so the host    \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  prerequisite remains active. The display  \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  peer also still does not match the active \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  session. I did not launch builds,         \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  captures, the matrix, or Test Genie.      \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  The refreshed epoch check shows G1\u2013G7 met;\u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  G8 and G9 unmet/unverified. The entry     \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  records the blockers and next action:     \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  recheck after swap-in clears, confirm     \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  display-session pairing, then run the     \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  prepared governed matrix and owner        \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  validation within existing limits.        \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  Worked for 44s \u2022 13:26                    \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n\u203a Ask Codex to do anything                  \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n                                            \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  GPT-6-Luna \u2026 Goal stalled (/goal resume)  \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n  ? for shortcuts                  \u26a0 2 \u00b7 f2 \u2502\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\u00b7\n[wc-16d2eb0:node*                       [0,6] \"Update E3 slice log |\" 18:21 17-O\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"
	emu := terminal.New(terminal.Options{Cols: 120, Rows: 40})
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	payload := "\x1b[H" + strings.Join(lines, "\r\n") + "\x1b[20;3H"
	if _, err := emu.Feed([]byte(payload)); err != nil {
		t.Fatalf("feed retained stalled-goal screen: %v", err)
	}
	return decodedView{view: emu.View(), text: emu.PlainText(false)}
}

func TestCodexGoalStalledTextOutsideVisibleFooterIsNotAnInterruption(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{
			name: "quoted transcript text",
			text: strings.Join([]string{
				"The runbook says: Goal stalled (/goal resume)",
				"› Ask Codex to do anything",
				"? for shortcuts",
			}, "\n"),
		},
		{
			name: "quoted footer-shaped sequence is not current footer",
			text: strings.Join([]string{
				"Quoted assistant output:",
				"› Ask Codex to do anything",
				"",
				"GPT-6-Luna … Goal stalled (/goal resume)",
				"? for shortcuts",
				"• Working (3s · esc to interrupt)",
				"› Ask Codex to do anything",
				"",
				"GPT-6-Luna",
				"? for shortcuts",
			}, "\n"),
		},
		{
			name: "ordinary prompt",
			text: strings.Join([]string{
				"› Ask Codex to do anything",
				"GPT-6-Luna",
				"? for shortcuts",
			}, "\n"),
		},
		{
			name: "status phrase without resume action",
			text: strings.Join([]string{
				"› Ask Codex to do anything",
				"GPT-6-Luna … Goal stalled",
				"? for shortcuts",
			}, "\n"),
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultPromptDetector().Analyze(stubView{text: tt.text})
			if got.Interruption != nil || got.Prompt != nil {
				t.Fatalf("screen = %+v, want no interruption or prompt", got)
			}
		})
	}
}
