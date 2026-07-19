package logs

import (
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

func TestCandidatePathsIncludesCanonicalLogsDirAndVolumeSources(t *testing.T) {
	manifest := manifestpkg.ResourceManifest{
		Runtime: manifestpkg.ResourceRuntime{
			Volumes: []manifestpkg.ResourceVolume{
				{Source: "/tmp/service/logs", Target: "/app/logs"},
				{Source: "/tmp/service/data", Target: "/app/data"},
			},
		},
	}
	paths := runtimestorage.Paths{LogsDir: "/var/log/vrooli/resources/demo"}
	got := CandidatePaths(manifest, paths)
	if len(got) != 2 {
		t.Fatalf("CandidatePaths len = %d, want 2 (%v)", len(got), got)
	}
	if got[0] != "/var/log/vrooli/resources/demo" {
		t.Fatalf("first candidate = %q", got[0])
	}
}
