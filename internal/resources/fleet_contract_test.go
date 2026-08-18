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

func TestFleetContractRejectsManagedServiceGPUBlock(t *testing.T) {
	fixture := fleetContractFixture()
	fixture.GPU = &manifestpkg.ResourceGPU{Probe: "nvidia"}
	if err := checkManagedArtifact("fixture", fixture); err == nil || !strings.Contains(err.Error(), "obsolete gpu block") {
		t.Fatalf("checkManagedArtifact error = %v, want managed-service GPU failure", err)
	}
}
