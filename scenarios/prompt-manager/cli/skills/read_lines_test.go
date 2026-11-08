package skills

import (
	"strings"
	"testing"
)

func TestReadLineRangePagesWithTheNextCommand(t *testing.T) {
	text := "one\ntwo\nthree\nfour\nfive\n"
	page, err := parseLineRange("2-3")
	if err != nil {
		t.Fatal(err)
	}
	next := readPageCommand([]string{"adaptive-research"}, []string{"--lines", "2-3", "adaptive-research"})
	got := page.render(text, next)
	want := "     2\ttwo\n     3\tthree\n[lines 2-3 of 5; next: prompt-manager skill read adaptive-research --lines 4-5]\n"
	if got != want {
		t.Fatalf("page 2-3:\n%q\nwant\n%q", got, want)
	}

	last, _ := parseLineRange("4-160")
	if got := last.render(text, next); !strings.HasSuffix(got, "[lines 4-5 of 5; end of output]\n") || !strings.HasPrefix(got, "     4\tfour\n") {
		t.Fatalf("last page = %q", got)
	}
	beyond, _ := parseLineRange("9-10")
	if got := beyond.render(text, next); got != "[lines 9-10 requested; the output has 5 lines]\n" {
		t.Fatalf("beyond end = %q", got)
	}
	for _, bad := range []string{"0-5", "5-2", "x", "3-y"} {
		if _, err := parseLineRange(bad); err == nil {
			t.Errorf("--lines %s accepted", bad)
		}
	}
	if got := readPageCommand([]string{"a", "b"}, []string{"--format", "markdown", "--lines=1-9", "a", "b"}); got != "prompt-manager skill read a b --format markdown" {
		t.Fatalf("readPageCommand = %q", got)
	}
}
