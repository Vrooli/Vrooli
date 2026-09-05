package pipeline

import (
	"strings"
	"testing"
)

func TestIsWhisperHallucination(t *testing.T) {
	cases := map[string]bool{
		"thanks for watching":  true,
		"Thanks for watching.": true,
		"THANKS!":              true,
		"(beep)":               true,
		"[BLANK_AUDIO]":        true,
		"good morning":         false,
		"genuine text here":    false,
	}
	for in, want := range cases {
		if got := IsWhisperHallucination(in); got != want {
			t.Errorf("IsWhisperHallucination(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLastNWords(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"one two three four", 2, "three four"},
		{"single", 5, "single"},
		{"", 3, ""},
		{"a b c", 0, ""},
		{"a b c d e", 3, "c d e"},
	}
	for _, tc := range cases {
		if got := LastNWords(tc.in, tc.n); got != tc.want {
			t.Errorf("LastNWords(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
		}
	}
}

func TestTruncateForLog(t *testing.T) {
	if got := TruncateForLog("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
	if got := TruncateForLog("abcdefghij", 5); got != "abcde…" {
		t.Fatalf("got %q", got)
	}
}

func TestDeduplicateOverlap(t *testing.T) {
	cases := []struct {
		acc, new, want string
	}{
		{"", "hello world", "hello world"},
		{"hello world", "", "hello world"},
		{"hello world", "world today", "hello world today"},
		{"good morning everyone", "everyone please", "good morning everyone please"},
		{"a b c", "d e", "a b c d e"},
	}
	for _, tc := range cases {
		got := DeduplicateOverlap(tc.acc, tc.new)
		if got != tc.want {
			t.Errorf("DeduplicateOverlap(%q, %q) = %q, want %q", tc.acc, tc.new, got, tc.want)
		}
	}
}

func TestDeduplicateOverlapBoundedDoesNotDeleteARepeatedPhrase(t *testing.T) {
	accumulated := "the quick brown fox jumps over the lazy dog and then the team reviews the final checklist"
	newText := "the team reviews the final checklist before the release"

	got := DeduplicateOverlapBounded(accumulated, newText, 5)
	want := accumulated + " " + newText
	if got != want {
		t.Fatalf("bounded overlap deleted a repeated phrase: got %q, want %q", got, want)
	}
}

func TestDeduplicateOverlapBoundedStillRemovesPhysicalOverlap(t *testing.T) {
	got := DeduplicateOverlapBounded("one two three four five", "four five six seven", 2)
	if got != "one two three four five six seven" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveWhisperURLDefault(t *testing.T) {
	t.Setenv("AUDIO_WHISPER_URL", "")
	t.Setenv("WC_WHISPER_URL", "")
	got := ResolveWhisperURL()
	if got == "" {
		t.Fatalf("expected non-empty default whisper URL")
	}
	if !strings.HasPrefix(got, "http") {
		t.Fatalf("expected http(s) URL, got %q", got)
	}
}
