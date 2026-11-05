package server

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
)

func TestDefaultWriterRootsAvoidBroadRuntimeHomeScan(t *testing.T) {
	home := t.TempDir()
	cfg, err := config.LoadFrom(func(key string) (string, bool) {
		if key == "API_PORT" {
			return "8080", true
		}
		if key == "VROOLI_HOME" {
			return filepath.Join(home, ".vrooli"), true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	roots := defaultWriterRoots(cfg)
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
