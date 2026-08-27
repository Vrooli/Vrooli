package deployability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateManifestDeclarationsAcceptsTheAuthoredVocabulary(t *testing.T) {
	evidence := &Evidence{
		RunID: "run-123", Host: "builder-1", OS: "linux", Arch: "amd64",
		Date: "2026-08-25", Surface: "manifest-validation", ArtifactURI: "artifact://run-123",
	}
	declarations := []ManifestDeclaration{{
		Path: "scenarios/system-monitor/.vrooli/service.json", Name: "system-monitor/system-monitor-cpu",
		Capability: "host-metrics", Role: "primary",
		Platforms: map[string]PlatformDeclaration{
			"linux":   {Status: string(StatusSupported), Evidence: evidence},
			"macos":   {Status: string(StatusBuildVerified), Mechanism: "host_statistics"},
			"windows": {Status: string(StatusBuildVerified), Mechanism: "performance counters"},
		},
	}, {
		Path: "scenarios/vrooli-autoheal/.vrooli/service.json", Name: "vrooli-autoheal/autoheal-watchdog",
		Capability: "self-healing", Role: "peer",
		Platforms: map[string]PlatformDeclaration{
			"linux": {Status: string(StatusSupported), Evidence: evidence},
			"macos": {Status: string(StatusExperimental)},
		},
	}}
	if err := ValidateManifestDeclarations(declarations, []string{"host-metrics", "self-healing"}); err != nil {
		t.Fatalf("expected the authored vocabulary to validate, got %v", err)
	}
}

func TestValidateManifestDeclarationsRejectsUnevidencedQualifiedClaim(t *testing.T) {
	path := "internal/tools/git/tool.json"
	err := ValidateManifestDeclarations([]ManifestDeclaration{{
		Path: path, Name: "git", Capability: "source-control", Role: "primary",
		Platforms: map[string]PlatformDeclaration{"linux": {Status: string(StatusSupported)}},
	}}, []string{"source-control"})
	if err == nil {
		t.Fatal("expected a qualified claim without structured evidence to be rejected")
	}
	for _, fragment := range []string{path, "linux", "structured evidence"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error must contain %q, got %q", fragment, err)
		}
	}
}

func TestValidateManifestDeclarationsAcceptsCompleteEvidenceForQualifiedClaim(t *testing.T) {
	err := ValidateManifestDeclarations([]ManifestDeclaration{{
		Path: "internal/tools/git/tool.json", Name: "git", Capability: "source-control", Role: "primary",
		Platforms: map[string]PlatformDeclaration{"linux": {
			Status: string(StatusSupported),
			Evidence: &Evidence{
				RunID: "run-123", Host: "builder-1", OS: "linux", Arch: "amd64",
				Date: "2026-08-25", Surface: "manifest-validation", ArtifactURI: "artifact://run-123",
			},
		}},
	}}, []string{"source-control"})
	if err != nil {
		t.Fatalf("expected complete structured evidence to validate, got %v", err)
	}
}

func TestValidateManifestDeclarationsAcceptsControlRole(t *testing.T) {
	err := ValidateManifestDeclarations([]ManifestDeclaration{{
		Path: "internal/safeguards/example/safeguard.json", Name: "example",
		Capability: "host-metrics", Role: "control",
	}}, []string{"host-metrics"})
	if err != nil {
		t.Fatalf("control role should be part of the manifest contract: %v", err)
	}
}

func TestValidateManifestDeclarationsRejectsAStatusOutsideTheVocabulary(t *testing.T) {
	path := "internal/tools/git/tool.json"
	err := ValidateManifestDeclarations([]ManifestDeclaration{{
		Path: path, Name: "git", Capability: "source-control", Role: "primary",
		Platforms: map[string]PlatformDeclaration{"macos": {Status: "bundled"}},
	}}, []string{"source-control"})
	if err == nil {
		t.Fatal("expected a malformed platform status to be rejected")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error must name the offending file, got %q", err)
	}
	if !strings.Contains(err.Error(), "bundled") {
		t.Fatalf("error must name the offending token, got %q", err)
	}
	if !strings.Contains(err.Error(), "macos") {
		t.Fatalf("error must name the offending host OS, got %q", err)
	}
}

func TestValidateManifestDeclarationsRejectsUnknownCapabilityAndRole(t *testing.T) {
	cases := []struct {
		name        string
		declaration ManifestDeclaration
		fragment    string
	}{
		{
			name:        "unknown capability",
			declaration: ManifestDeclaration{Path: "internal/tools/git/tool.json", Name: "git", Capability: "teleportation", Role: "primary"},
			fragment:    "teleportation",
		},
		{
			name:        "unknown role",
			declaration: ManifestDeclaration{Path: "internal/tools/git/tool.json", Name: "git", Capability: "source-control", Role: "backup"},
			fragment:    "backup",
		},
		{
			name:        "missing capability",
			declaration: ManifestDeclaration{Path: "internal/tools/git/tool.json", Name: "git", Role: "primary"},
			fragment:    "has no capability",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateManifestDeclarations([]ManifestDeclaration{testCase.declaration}, []string{"source-control"})
			if err == nil {
				t.Fatal("expected a rejection")
			}
			if !strings.Contains(err.Error(), testCase.declaration.Path) {
				t.Fatalf("error must name the offending file, got %q", err)
			}
			if !strings.Contains(err.Error(), testCase.fragment) {
				t.Fatalf("error must contain %q, got %q", testCase.fragment, err)
			}
		})
	}
}

func TestValidateManifestDeclarationsRejectsAMalformedVocabulary(t *testing.T) {
	if err := ValidateManifestDeclarations(nil, []string{"source-control", "source-control"}); err == nil {
		t.Fatal("expected a duplicated vocabulary entry to be rejected")
	}
	if err := ValidateManifestDeclarations(nil, []string{" "}); err == nil {
		t.Fatal("expected an empty vocabulary entry to be rejected")
	}
}

func TestValidateManifestDeclarationsRejectsClosedCellMechanismAndMissingReview(t *testing.T) {
	declaration := ManifestDeclaration{
		Path: "internal/safeguards/example/safeguard.json", Name: "example", Capability: "host-metrics", Role: "control",
		Platforms: map[string]PlatformDeclaration{"windows": {
			Status: string(StatusNotApplicable), Mechanism: "handler", EvidenceRaw: `"Windows has no equivalent"`,
		}},
	}
	err := ValidateManifestDeclarations([]ManifestDeclaration{declaration}, []string{"host-metrics"})
	if err == nil || !strings.Contains(err.Error(), "not_applicable") {
		t.Fatalf("expected closed-cell contract rejection, got %v", err)
	}
}

func TestValidateManifestDeclarationsRejectsUnwiredGenericAndCircularClaim(t *testing.T) {
	declaration := ManifestDeclaration{
		Path: "internal/safeguards/example/safeguard.json", Name: "example", Capability: "host-metrics", Role: "control",
		Platforms: map[string]PlatformDeclaration{"windows": {
			Status: string(StatusNotImplemented), Mechanism: "handler", Since: "2026-01-01", EvidenceRaw: `"no implementation for the Windows host mechanism"`,
		}},
	}
	err := ValidateManifestDeclarations([]ManifestDeclaration{declaration}, []string{"host-metrics"})
	if err == nil || !strings.Contains(err.Error(), "generic handler") {
		t.Fatalf("expected unwired mechanism rejection, got %v", err)
	}
}

// TestDeclarationFixturesStayExecutable keeps the negative examples named by
// the portability plan on disk. Inline examples are useful unit tests, but
// they do not protect the reviewed fixture corpus from being deleted or
// drifting away from the manifest shape that the repository gate reads.
func TestDeclarationFixturesStayExecutable(t *testing.T) {
	root := filepath.Join("testdata", "declarations")
	cases := []struct {
		name     string
		contains string
	}{
		{name: "not_applicable-with-mechanism.json", contains: "not_applicable with mechanism"},
		{name: "not_implemented-without-mechanism.json", contains: "not_implemented without mechanism"},
		{name: "circular-evidence.json", contains: "circular evidence phrase"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(root, testCase.name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Path       string `json:"path"`
				Name       string `json:"name"`
				Capability string `json:"capability"`
				Role       string `json:"capability_role"`
				Platforms  map[string]struct {
					Status    string          `json:"status"`
					Mechanism string          `json:"mechanism"`
					Since     string          `json:"since"`
					ReviewBy  string          `json:"review_by"`
					Evidence  json.RawMessage `json:"evidence"`
				} `json:"platform_status"`
			}
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			declarations := make(map[string]PlatformDeclaration, len(fixture.Platforms))
			for hostOS, platform := range fixture.Platforms {
				declarations[hostOS] = PlatformDeclaration{
					Status: platform.Status, Mechanism: platform.Mechanism,
					Since: platform.Since, ReviewBy: platform.ReviewBy,
					EvidenceRaw: string(platform.Evidence),
				}
			}
			err = ValidateManifestDeclarations([]ManifestDeclaration{{
				Path: fixture.Path, Name: fixture.Name, Capability: fixture.Capability,
				Role: fixture.Role, Platforms: declarations,
			}}, []string{fixture.Capability})
			if err == nil {
				t.Fatal("negative declaration fixture unexpectedly validated")
			}
			if !strings.Contains(err.Error(), fixture.Path) {
				t.Fatalf("failure does not name fixture manifest path %q: %v", fixture.Path, err)
			}
			if !strings.Contains(err.Error(), testCase.contains) {
				t.Fatalf("failure does not name expected defect %q: %v", testCase.contains, err)
			}
		})
	}
}
