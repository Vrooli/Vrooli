package server

import (
	"path/filepath"
	"testing"
)

func TestDefaultWriterRootsAvoidBroadRuntimeHomeScan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))

	roots := defaultWriterRoots()
	for _, root := range roots {
		if root.ID == "runtime-home" {
			t.Fatal("default writer roots must not scan the entire runtime home")
		}
	}

	for _, root := range roots {
		if root.ID == "go-work-dirs" {
			want := filepath.Join(home, ".vrooli", "tmp", "go-work")
			if root.Root != want {
				t.Fatalf("go-work root = %q, want %q", root.Root, want)
			}
			if root.HotWriterBytesHour != 1<<30 {
				t.Fatalf("go-work hot-writer threshold = %d, want 1 GiB/hour", root.HotWriterBytesHour)
			}
			if root.MeasureBudget.String() != "2s" {
				t.Fatalf("go-work measure budget = %s, want 2s", root.MeasureBudget)
			}
			return
		}
	}
	t.Fatal("go-work-dirs root was not configured")
}
