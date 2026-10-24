//nolint:goconst // test data deliberately reuses stable deployment fixtures.
package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestFleetContractRepository(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Clean(filepath.Join(root, "..", ".."))
	if err := CheckFleetContract(root); err != nil {
		t.Fatal(err)
	}
}

func fleetContractFixture() manifestpkg.ResourceManifest {
	const digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return manifestpkg.ResourceManifest{
		Name:     "fixture",
		Bundling: "vendorable",
		Deployment: manifestpkg.ResourceDeployment{Profiles: map[string]manifestpkg.ResourceDeploymentProfile{
			"desktop": {
				Linux:   &manifestpkg.ResourceDeploymentTarget{Support: "supported", Mode: "manual", Architectures: []string{"amd64"}},
				MacOS:   &manifestpkg.ResourceDeploymentTarget{Support: "supported", Mode: "manual", Architectures: []string{"amd64"}},
				Windows: &manifestpkg.ResourceDeploymentTarget{Support: "supported", Mode: "manual", Architectures: []string{"amd64"}},
			},
		}},
		ManagedService: &manifestpkg.ResourceManagedService{
			Artifact: resourcedeployment.ServiceArtifact{Path: "server", Version: "1.0.0", SHA256ByPlatform: map[string]string{
				"linux-amd64": digest, "macos-amd64": digest, "windows-amd64": digest,
			}},
			Acquisition: &binaryfetch.Acquisition{Kind: "url", License: "MIT", Targets: []binaryfetch.AcquisitionTarget{
				{URL: "https://example.com/server", SHA256: digest, Archive: "none", Layout: "file", Mode: "0755"},
			}},
		},
	}
}

func TestFleetContractRejectsUnboundRawDigest(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.ManagedService.Artifact.SHA256ByPlatform["linux-amd64"] = strings.Repeat("b", 64)
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "download/artifact digest mismatch") {
		t.Fatalf("checkManagedArtifact error = %v, want raw digest binding failure", err)
	}
}

func TestFleetContractRejectsClaimedPlatformWithoutAcquisition(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.ManagedService.Acquisition = nil
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "has no acquisition contract") {
		t.Fatalf("checkManagedArtifact error = %v, want missing acquisition failure", err)
	}
}

func TestFleetContractRejectsUnsupportedPlatformWithoutReason(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.Deployment.Profiles["desktop"].Windows.Reason = ""
	fixture.Deployment.Profiles["desktop"].Windows.Support = "unsupported"
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "unsupported windows target has no reason") {
		t.Fatalf("checkManagedArtifact error = %v, want unsupported-reason failure", err)
	}
}

func TestFleetContractRejectsRuntimeFactForVendorableArtifact(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.ManagedService.Acquisition.Targets[0].When = map[string]string{"gpu.cuda_compute": ">=8.9"}
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "uses runtime facts") {
		t.Fatalf("checkManagedArtifact error = %v, want runtime-fact failure", err)
	}
}

func TestFleetContractRejectsBundlingModeContradiction(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.Deployment.Profiles["desktop"].Linux.Mode = "bundled-service"
	fixture.Deployment.Profiles["desktop"].MacOS.Mode = "bundled-service"
	fixture.Deployment.Profiles["desktop"].Windows.Mode = "bundled-service"
	fixture.Bundling = "host-required"
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "host-required bundling") {
		t.Fatalf("checkManagedArtifact error = %v, want bundling contradiction", err)
	}
}

// Scenario: every surface the acceleration block replaced is rejected, on every
// driver.
//
// The rejection used to fire only for `gpu` on a managed-service resource,
// while internal/capacity used that same block as its only test for "this
// resource uses the GPU". The two disagreed in opposite directions. It now
// reads the manifest's raw JSON, because the parsed struct no longer has fields
// for those surfaces: a manifest still declaring one would load cleanly and be
// silently ignored, which is worse than failing.
func TestFleetContractRejectsEveryLegacyAcceleratorSurface(t *testing.T) {
	cases := []struct {
		scenario string
		manifest string
		wantErr  string
	}{
		{
			scenario: "Given the deprecated gpu block, Then it is rejected and names its replacement",
			manifest: `{"name":"fixture","gpu":{"probe":"nvidia"}}`,
			wantErr:  "deprecated gpu block",
		},
		{
			scenario: "Given the deprecated top-level capacity block, Then it is rejected",
			manifest: `{"name":"fixture","capacity":{"resource_kind":"vram","preferred_bytes":1}}`,
			wantErr:  "deprecated top-level capacity block",
		},
		{
			scenario: "Given the deprecated requirements.gpu block, Then it is rejected",
			manifest: `{"name":"fixture","requirements":{"gpu":{}}}`,
			wantErr:  "deprecated requirements.gpu block",
		},
		{
			scenario: "Given a null legacy key, Then it is not a declaration",
			manifest: `{"name":"fixture","gpu":null,"capacity":null}`,
		},
		{
			scenario: "Given only an acceleration block, Then it passes",
			manifest: `{"name":"fixture","acceleration":{"backends":["cuda","cpu"],"cuda":{},"cpu":{}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a manifest carrying that surface
			path := filepath.Join(t.TempDir(), "resource.json")
			if err := os.WriteFile(path, []byte(tc.manifest), 0o600); err != nil {
				t.Fatalf("write fixture manifest: %v", err)
			}

			// When the fleet contract checks it
			err := checkLegacyAcceleratorSurfaces("fixture", path)

			// Then it is rejected only when a legacy surface is actually declared
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkLegacyAcceleratorSurfaces() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("checkLegacyAcceleratorSurfaces() = %v, want an error containing %q", err, tc.wantErr)
			}
			// And the message names where the value goes instead
			if !strings.Contains(err.Error(), "acceleration") {
				t.Fatalf("checkLegacyAcceleratorSurfaces() = %v, want the message to name the acceleration block", err)
			}
		})
	}
}
