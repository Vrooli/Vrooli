package runner_test

import (
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
)

func TestStripANSI_BasicCSI(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"foreground color", "\x1b[31mred\x1b[0m", "red"},
		{"bright color", "\x1b[91mlight red\x1b[0m", "light red"},
		{"bold + color", "\x1b[1;31mbold red\x1b[0m", "bold red"},
		{"reset alone", "\x1b[0m", ""},
		{"clear line", "\x1b[Kerased", "erased"},
		{"cursor moves", "\x1b[2;3Hpositioned", "positioned"},
		{"multiple colors", "\x1b[31mred\x1b[32mgreen\x1b[34mblue\x1b[0m", "redgreenblue"},
		{"mixed text and ANSI", "before\x1b[31mred\x1b[0mafter", "beforeredafter"},
		{"empty string", "", ""},
		{"only ANSI", "\x1b[31m\x1b[0m", ""},
		{"no escape sequences", "no special chars here", "no special chars here"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runner.StripANSI(tc.input)
			if got != tc.expect {
				t.Errorf("StripANSI(%q)=%q want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestStripANSI_HighVolumeRetainsContent(t *testing.T) {
	// 1000 colored entries; output should retain just the inner content.
	parts := []string{}
	for i := 0; i < 1000; i++ {
		parts = append(parts, "\x1b[32mok\x1b[0m")
	}
	got := runner.StripANSI(strings.Join(parts, ""))
	want := strings.Repeat("ok", 1000)
	if got != want {
		t.Errorf("high-volume ANSI strip mismatch (lengths got=%d want=%d)", len(got), len(want))
	}
}

func TestIsOnlyANSI(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"pure ANSI", "\x1b[31m\x1b[0m", true},
		{"ANSI + whitespace", "\x1b[31m  \x1b[0m\n", true},
		{"plain text", "hello world", false},
		{"empty string", "", false},
		{"mixed content + ANSI", "\x1b[32mhello\x1b[0m", false},
		{"only whitespace", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runner.IsOnlyANSI(tt.input); got != tt.expect {
				t.Errorf("IsOnlyANSI(%q)=%v want %v", tt.input, got, tt.expect)
			}
		})
	}
}
