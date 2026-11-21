package journal

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGroupNameMatchesManifestDomain(t *testing.T) {
	if GroupName != "journal" {
		t.Fatalf("GroupName = %q, want journal", GroupName)
	}
}

func TestResolveNoteBodySupportsPositionalFileAndStdin(t *testing.T) {
	t.Run("positional compatibility", func(t *testing.T) {
		got, err := resolveNoteBody("legacy `body`", true, "", strings.NewReader("unused"))
		if err != nil || got != "legacy `body`" {
			t.Fatalf("resolveNoteBody() = %q, %v", got, err)
		}
	})

	t.Run("body file preserves multiline shell-sensitive text", func(t *testing.T) {
		want := "line one\ncommand `literal`\n"
		path := filepath.Join(t.TempDir(), "body.txt")
		if err := os.WriteFile(path, []byte(want), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveNoteBody("", false, path, strings.NewReader("unused"))
		if err != nil || got != want {
			t.Fatalf("resolveNoteBody() = %q, %v; want %q", got, err, want)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		want := "line one\nliteral `code`\n"
		got, err := resolveNoteBody("", false, "-", strings.NewReader(want))
		if err != nil || got != want {
			t.Fatalf("resolveNoteBody() = %q, %v; want %q", got, err, want)
		}
	})

	t.Run("requires one source", func(t *testing.T) {
		if _, err := resolveNoteBody("", false, "", strings.NewReader("")); err == nil || !strings.Contains(err.Error(), "work-record fields do not replace it") {
			t.Fatal("expected missing-body error")
		}
		if _, err := resolveNoteBody("body", true, "file.txt", strings.NewReader("")); err == nil {
			t.Fatal("expected conflicting-input error")
		}
		if _, err := resolveNoteBody("", true, "file.txt", strings.NewReader("")); err == nil {
			t.Fatal("expected conflict for an explicitly empty positional argument")
		}
	})

	t.Run("stdin read failure", func(t *testing.T) {
		if _, err := resolveNoteBody("", false, "-", errorReader{}); err == nil {
			t.Fatal("expected stdin read error")
		}
	})
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("stdin unavailable") }

var _ io.Reader = errorReader{}
