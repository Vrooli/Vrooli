package dependencyhealth

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/vrooli/vrooli/packages/deployability"
)

func TestDeclaredToolPlatformsUsesCanonicalPlatformStatus(t *testing.T) {
	got := declaredToolPlatforms(portabilityToolManifest{
		PlatformStatus: map[string]struct {
			Status string `json:"status"`
		}{
			"linux":   {Status: "unqualified"},
			"macos":   {Status: "unqualified"},
			"windows": {Status: "unsupported"},
		},
	})
	want := []deployability.HostOS{deployability.HostOSLinux, deployability.HostOSMacOS}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("declaredToolPlatforms() = %v, want %v", got, want)
	}
}

func TestDeclaredToolPlatformsPrefersLegacyApplicabilityList(t *testing.T) {
	got := declaredToolPlatforms(portabilityToolManifest{
		Platforms: []string{"macos"},
		PlatformStatus: map[string]struct {
			Status string `json:"status"`
		}{
			"linux": {Status: "unqualified"},
		},
	})
	want := []deployability.HostOS{deployability.HostOSMacOS}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("declaredToolPlatforms() = %v, want %v", got, want)
	}
}

func TestHasAcquisitionTargetsRecognizesGovernedSourceTargets(t *testing.T) {
	for name, raw := range map[string]string{
		"declared targets": `{"kind":"url","targets":[{"when":{"os":"darwin"},"url":"https://example.invalid/buf"}]}`,
		"empty targets":    `{"kind":"url","targets":[]}`,
		"null":             `null`,
	} {
		t.Run(name, func(t *testing.T) {
			var manifest json.RawMessage = []byte(raw)
			got := hasAcquisitionTargets(manifest)
			want := name == "declared targets"
			if got != want {
				t.Fatalf("hasAcquisitionTargets() = %t, want %t", got, want)
			}
		})
	}
}

func TestValidateToolAcquisitionCoverageEnumeratesRepositoryTools(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
	if err := validateToolAcquisitionCoverage(repoRoot); err != nil {
		t.Fatalf("repository tool acquisition validation failed: %v", err)
	}
}
