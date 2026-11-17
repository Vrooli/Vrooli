package runtimeapp

import (
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestForeignLauncherBindingHasAnOwnerFact(t *testing.T) {
	const binding = "github.com/vrooli/vrooli/internal/resources.ManagedUpdateRuntime.NativeLauncherProfile|github.com/vrooli/platform-go.NativeLauncherProfileOpenRGB"
	if statecontract.StoredForeignValueFacts[binding].Value != platform.NativeLauncherProfileOpenRGB || resources.PersistentState().Facts["native_launcher_publication_profile"] != platform.NativeLauncherProfileOpenRGB {
		t.Fatal("launcher selector must remain covered by the resource owner's compiled fact")
	}
	withRepositoryCodecStamp(t)
	owners := nativePersistenceOwners()
	for i := range owners {
		if owners[i].Name == "resources" {
			delete(owners[i].Facts, "native_launcher_publication_profile")
		}
	}
	if _, err := statecontract.Compute(owners); err == nil || !strings.Contains(err.Error(), "foreign stored value binding") {
		t.Fatalf("missing owner fact did not prevent qualification: %v", err)
	}
}

// withRepositoryCodecStamp stamps the test process exactly as a governed
// builder stamps an executable built from this checkout.
func withRepositoryCodecStamp(t *testing.T) string {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := buildinfo.PersistentStateCodecStamp(root)
	if err != nil {
		t.Fatal(err)
	}
	old := statecontract.CodecStamp
	t.Cleanup(func() { statecontract.CodecStamp = old })
	statecontract.CodecStamp = stamp
	return root
}

func TestNativePersistenceOwnersQualifyWithTheGovernedStamp(t *testing.T) {
	withRepositoryCodecStamp(t)
	manifest, err := NativePersistentStateManifest()
	if err != nil {
		t.Fatalf("native persistence owners are unqualified: %v", err)
	}
	if len(manifest.Contract) != 64 || manifest.Contract != NativePersistentStateContract() {
		t.Fatalf("contract digest is not the manifest's: %q", manifest.Contract)
	}
	if len(manifest.Owners) != len(nativePersistenceOwners()) {
		t.Fatalf("manifest owners = %d, declared = %d", len(manifest.Owners), len(nativePersistenceOwners()))
	}
	for _, owner := range manifest.Owners {
		if owner.Owner == "capacity" {
			if _, ok := owner.Types["github.com/vrooli/vrooli/internal/capacity.ResidencyTransitionRecord"]; !ok {
				t.Fatal("capacity contract omits persisted residency transition records")
			}
			if _, ok := owner.Types["github.com/vrooli/vrooli/internal/capacity.PlacementQualification"]; !ok {
				t.Fatal("capacity contract omits persisted placement qualifications")
			}
		}
		if owner.Owner == "secure-store" {
			if _, ok := owner.Types["github.com/vrooli/vrooli/internal/securestore.CopyStatus"]; !ok {
				t.Fatal("secure-store contract omits persisted copy receipt")
			}
		}
		if owner.Owner == "native-operation-identities" {
			if _, failed := owner.Facts["operation_intent_sample_error"]; failed {
				t.Fatal("native operation identity sample no longer validates; fix the sample spec")
			}
		}
	}
}

func TestUnstampedExecutableIsUnqualified(t *testing.T) {
	old := statecontract.CodecStamp
	t.Cleanup(func() { statecontract.CodecStamp = old })
	statecontract.CodecStamp = ""
	if NativePersistentStateContract() != "" {
		t.Fatal("an unstamped executable reported a persistent-state contract")
	}
	if _, err := NativePersistentStateManifest(); err == nil || !strings.Contains(err.Error(), "codec stamp") {
		t.Fatalf("unstamped refusal does not name the missing stamp: %v", err)
	}
}

// agent-catalogs is declared here because cli-core is a separate module that
// cannot import the repository's checks; this is its package source test.
func TestAgentCatalogOwnerPassesItsSourceChecks(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	dirs := statecontract.PreciseSources["agent-catalogs"]
	if len(dirs) == 0 {
		t.Fatal("agent-catalogs has no precise source inventory")
	}
	for _, dir := range dirs {
		findings, err := buildinfo.PersistentStateSourceFindings(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s: %s", dir, finding)
		}
	}
}
