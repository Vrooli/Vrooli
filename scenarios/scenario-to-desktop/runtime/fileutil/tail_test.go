package fileutil

import (
	"testing"

	"scenario-to-desktop-runtime/testutil"
)

func TestTailFile(t *testing.T) {
	t.Run("returns last N lines", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()
		fs.Files["/test.log"] = []byte("line1\nline2\nline3\nline4\nline5")

		got, err := TailFile(fs, "/test.log", 3)
		if err != nil {
			t.Fatalf("TailFile() error = %v", err)
		}

		want := "line3\nline4\nline5"
		if string(got) != want {
			t.Errorf("TailFile() = %q, want %q", string(got), want)
		}
	})

	t.Run("returns all lines when N exceeds file length", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()
		content := "line1\nline2"
		fs.Files["/test.log"] = []byte(content)

		got, err := TailFile(fs, "/test.log", 100)
		if err != nil {
			t.Fatalf("TailFile() error = %v", err)
		}

		if string(got) != content {
			t.Errorf("TailFile() = %q, want %q", string(got), content)
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()
		fs.Files["/empty.log"] = []byte("")

		got, err := TailFile(fs, "/empty.log", 10)
		if err != nil {
			t.Fatalf("TailFile() error = %v", err)
		}

		if string(got) != "" {
			t.Errorf("TailFile() = %q, want empty string", string(got))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()

		_, err := TailFile(fs, "/nonexistent/path/file.log", 10)
		if err == nil {
			t.Error("TailFile() should return error for missing file")
		}
	})

	t.Run("ignores trailing newline when slicing", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()
		fs.Files["/test.log"] = []byte("line1\nline2\nline3\n")

		got, err := TailFile(fs, "/test.log", 2)
		if err != nil {
			t.Fatalf("TailFile() error = %v", err)
		}

		want := "line2\nline3"
		if string(got) != want {
			t.Errorf("TailFile() = %q, want %q", string(got), want)
		}
	})

	t.Run("returns empty slice when asked for non-positive lines", func(t *testing.T) {
		fs := testutil.NewMockFileSystem()
		fs.Files["/test.log"] = []byte("line1\nline2\n")

		got, err := TailFile(fs, "/test.log", 0)
		if err != nil {
			t.Fatalf("TailFile() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("TailFile() = %q, want empty output for non-positive lines", string(got))
		}
	})
}
