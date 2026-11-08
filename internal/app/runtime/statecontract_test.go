package runtimeapp

import (
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/statecontract"
)

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
