package bundleruntime

import (
	"os"
	"path/filepath"
	"testing"
)

// Note: Tests for copyStringMap, envMapToList, and intersection have been
// moved to internal/strutil/slice_test.go as they test the canonical
// implementations.
//
// Tests for TopoSort and FindService have been moved to deps/resolver_test.go.

func TestParsePositiveInt(t *testing.T) {
	t.Run("parses valid positive integers", func(t *testing.T) {
		tests := []struct {
			input string
			want  int
		}{
			{"1", 1},
			{"42", 42},
			{"  100  ", 100},
			{"999999", 999999},
		}

		for _, tc := range tests {
			got, err := parsePositiveInt(tc.input)
			if err != nil {
				t.Errorf("parsePositiveInt(%q) error = %v, want nil", tc.input, err)
				continue
			}
			if got != tc.want {
				t.Errorf("parsePositiveInt(%q) = %d, want %d", tc.input, got, tc.want)
			}
		}
	})

	t.Run("rejects zero", func(t *testing.T) {
		_, err := parsePositiveInt("0")
		if err == nil {
			t.Error("parsePositiveInt(\"0\") should return error")
		}
	})

	t.Run("rejects negative numbers", func(t *testing.T) {
		_, err := parsePositiveInt("-5")
		if err == nil {
			t.Error("parsePositiveInt(\"-5\") should return error")
		}
	})

	t.Run("rejects non-numeric input", func(t *testing.T) {
		_, err := parsePositiveInt("abc")
		if err == nil {
			t.Error("parsePositiveInt(\"abc\") should return error")
		}
	})

	t.Run("rejects empty string", func(t *testing.T) {
		_, err := parsePositiveInt("")
		if err == nil {
			t.Error("parsePositiveInt(\"\") should return error")
		}
	})
}

func TestTailFile(t *testing.T) {
	// Tests the Supervisor.tailFile method which delegates to fileutil.TailFile
	newTestSupervisor := func() *Supervisor {
		return &Supervisor{fs: RealFileSystem{}}
	}

	t.Run("returns last N lines", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.log")
		content := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 3)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		want := "line3\nline4\nline5"
		if string(got) != want {
			t.Errorf("tailFile() = %q, want %q", string(got), want)
		}
	})

	t.Run("returns all lines when N exceeds file length", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.log")
		content := "line1\nline2"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 100)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		if string(got) != content {
			t.Errorf("tailFile() = %q, want %q", string(got), content)
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.log")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 10)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		if string(got) != "" {
			t.Errorf("tailFile() = %q, want empty string", string(got))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		s := newTestSupervisor()
		_, err := s.tailFile("/nonexistent/path/file.log", 10)
		if err == nil {
			t.Error("tailFile() should return error for missing file")
		}
	})
}
