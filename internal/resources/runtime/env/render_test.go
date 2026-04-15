package env

import (
	"path/filepath"
	"testing"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

func TestRenderValue(t *testing.T) {
	r := NewRenderer("/repo", "/home/tester", "redis", runtimestorage.Paths{
		ConfigDir: filepath.Join("/cfg", "vrooli", "resources", "redis"),
		DataDir:   filepath.Join("/data", "vrooli", "resources", "redis"),
		CacheDir:  filepath.Join("/cache", "vrooli", "resources", "redis"),
		LogsDir:   filepath.Join("/logs", "vrooli", "resources", "redis"),
		StateDir:  filepath.Join("/state", "vrooli", "resources", "redis"),
	})

	got := r.RenderValue("${RESOURCE_DATA_DIR}/dump.rdb")
	want := filepath.Join("/data", "vrooli", "resources", "redis", "dump.rdb")
	if got != want {
		t.Fatalf("RenderValue = %q, want %q", got, want)
	}

	got = r.RenderValue("${ROOT}/resources/${RESOURCE_ROOT}")
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute rendered value, got %q", got)
	}
}

