package codex

import "testing"

type screenView struct {
	text string
	row  int
}

func (s screenView) Cols() int         { return 80 }
func (s screenView) Rows() int         { return 3 }
func (s screenView) CursorRow() int    { return s.row }
func (s screenView) CursorCol() int    { return 0 }
func (s screenView) PlainText() string { return s.text }

func TestPromptDetector(t *testing.T) {
	d := DefaultPromptDetector()
	for _, tc := range []struct {
		name, text string
		row        int
		want       bool
	}{
		{"nil", "", 0, false},
		{"user prompt", "header\nuser: ", 1, true},
		{"glyph prompt", "header\n▶ ", 1, true},
		{"other row", "user: \nbody", 1, false},
		{"out of range", "user:", 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var view screenView
			if tc.name != "nil" {
				view = screenView{text: tc.text, row: tc.row}
			}
			var got bool
			if tc.name == "nil" {
				got = d.IsAwaitingInput(nil)
			} else {
				got = d.IsAwaitingInput(view)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
