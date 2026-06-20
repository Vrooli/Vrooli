package scenariocli

import (
	"strings"
	"testing"
)

func TestStripExampleDomainBlocks(t *testing.T) {
	in := strings.Join([]string{
		"# Domains",
		"",
		"The real `health` domain lives here.",
		"",
		"<!-- EXAMPLE-DOMAIN:notes START (delete when removed) -->",
		"## notes (example)",
		"",
		"The notes domain is a worked example.",
		"<!-- EXAMPLE-DOMAIN:notes END -->",
		"",
		"Binding guidance continues.",
		"",
	}, "\n")

	out, removed := StripExampleDomainBlocks([]byte(in), "notes")
	if removed != 1 {
		t.Fatalf("expected 1 block removed, got %d", removed)
	}
	got := string(out)
	if strings.Contains(got, "notes") {
		t.Fatalf("notes residue survived block strip:\n%s", got)
	}
	if !strings.Contains(got, "real `health` domain") {
		t.Fatalf("binding zone clobbered:\n%s", got)
	}
	if !strings.Contains(got, "Binding guidance continues.") {
		t.Fatalf("post-fence content lost:\n%s", got)
	}
	// No double blank line where the fence used to be.
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("stray blank gap left behind:\n%q", got)
	}
}

func TestStripExampleDomainBlocksMultiple(t *testing.T) {
	in := "a\n<!-- EXAMPLE-DOMAIN:notes START -->\nx\n<!-- EXAMPLE-DOMAIN:notes END -->\nb\n" +
		"<!-- EXAMPLE-DOMAIN:notes START -->\ny\n<!-- EXAMPLE-DOMAIN:notes END -->\nc\n"
	out, removed := StripExampleDomainBlocks([]byte(in), "notes")
	if removed != 2 {
		t.Fatalf("expected 2 blocks, got %d", removed)
	}
	if got := string(out); got != "a\nb\nc\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStripExampleDomainBlocksCodeFence(t *testing.T) {
	in := strings.Join([]string{
		"func SubcommandGroups() ([]Group, error) {",
		"	groups := []Group{}",
		"	// EXAMPLE-DOMAIN:notes START",
		"	g, err := notes.Register()",
		"	if err != nil {",
		"		return nil, err",
		"	}",
		"	groups = append(groups, g)",
		"	// EXAMPLE-DOMAIN:notes END",
		"	return groups, nil",
		"}",
		"",
	}, "\n")
	out, removed := StripExampleDomainBlocks([]byte(in), "notes")
	if removed != 1 {
		t.Fatalf("expected 1 code block removed, got %d", removed)
	}
	got := string(out)
	if strings.Contains(got, "notes") {
		t.Fatalf("notes residue survived code-fence strip:\n%s", got)
	}
	if !strings.Contains(got, "groups := []Group{}") || !strings.Contains(got, "return groups, nil") {
		t.Fatalf("binding zone clobbered:\n%s", got)
	}
}

func TestStripExampleDomainBlocksNoop(t *testing.T) {
	in := "no markers here\n"
	out, removed := StripExampleDomainBlocks([]byte(in), "notes")
	if removed != 0 || string(out) != in {
		t.Fatalf("expected no-op, got removed=%d out=%q", removed, out)
	}
}

func TestStripExampleDomainLines(t *testing.T) {
	in := strings.Join([]string{
		`import "scenario/handlers/health"`,
		`notesH "scenario/handlers/notes" // EXAMPLE-DOMAIN:notes`,
		`{ path: "notes", element: <NotesPage /> }, // EXAMPLE-DOMAIN:notes`,
		`	"notes": "Notes", // EXAMPLE-DOMAIN:notes`,
		`realLine()`,
		"",
	}, "\n")
	out, removed := StripExampleDomainLines([]byte(in), "notes")
	if removed != 3 {
		t.Fatalf("expected 3 lines removed, got %d", removed)
	}
	got := string(out)
	if strings.Contains(got, "notes") {
		t.Fatalf("notes residue survived line strip:\n%s", got)
	}
	if !strings.Contains(got, `handlers/health`) || !strings.Contains(got, "realLine()") {
		t.Fatalf("non-marked lines lost:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("trailing newline not preserved: %q", got)
	}
}

func TestStripExampleDomainLinesWordBoundary(t *testing.T) {
	// A different marker (notes-archive) must NOT be stripped when removing
	// the `notes` example domain.
	in := "keep // EXAMPLE-DOMAIN:notes-archive\ndrop // EXAMPLE-DOMAIN:notes\n"
	out, removed := StripExampleDomainLines([]byte(in), "notes")
	if removed != 1 {
		t.Fatalf("expected 1 line removed, got %d", removed)
	}
	if got := string(out); got != "keep // EXAMPLE-DOMAIN:notes-archive\n" {
		t.Fatalf("word-boundary strip wrong: %q", got)
	}
}

func TestStripExampleDomainFileCombined(t *testing.T) {
	in := "top // EXAMPLE-DOMAIN:notes\n<!-- EXAMPLE-DOMAIN:notes START -->\nblock\n<!-- EXAMPLE-DOMAIN:notes END -->\nkeep\n"
	out, summary, changed := StripExampleDomainFile("x.md", []byte(in), "notes")
	if !changed {
		t.Fatalf("expected changed")
	}
	if summary.BlocksRemoved != 1 || summary.LinesStripped != 1 {
		t.Fatalf("summary wrong: %+v", summary)
	}
	if got := string(out); got != "keep\n" {
		t.Fatalf("combined strip wrong: %q", got)
	}
	if ContainsExampleDomainMarker(out, "notes") {
		t.Fatalf("marker survived combined strip")
	}
}

func TestContainsExampleDomainMarker(t *testing.T) {
	if !ContainsExampleDomainMarker([]byte("x // EXAMPLE-DOMAIN:notes\n"), "notes") {
		t.Fatalf("expected marker detected")
	}
	if ContainsExampleDomainMarker([]byte("x // EXAMPLE-DOMAIN:notesxyz\n"), "notes") {
		t.Fatalf("false positive on longer marker")
	}
	if ContainsExampleDomainMarker([]byte("clean\n"), "notes") {
		t.Fatalf("false positive on clean content")
	}
}
