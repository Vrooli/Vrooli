package closure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/domain"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("repository root not found")
		}
		dir = parent
	}
}

// TestResolveAgainstRepositoryCatalog exercises the production seams: the
// repo-contract catalog and packages/hostreq. It proves the closure of a real
// scenario is derived from declarations without any product-name rule.
// [REQ:STC-P0-021]
func TestResolveAgainstRepositoryCatalog(t *testing.T) {
	if testing.Short() {
		t.Skip("repository catalog resolution is skipped in short mode")
	}
	root := findRepoRoot(t)
	catalog, err := NewRepoCatalog(root)
	if err != nil {
		t.Fatalf("repo catalog: %v", err)
	}
	closure, err := Resolve(context.Background(), Inputs{
		ScenarioID:       "scenario-to-cloud",
		Environment:      "production",
		Platform:         Platform{OS: "linux", Arch: "amd64"},
		RepoRoot:         root,
		Catalog:          catalog,
		HostRequirements: ControlPlaneHostRequirements{},
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	postgres := component(t, closure, domain.ClosureKindResource, "postgres")
	if !postgres.Required || !hasReason(postgres, domain.ClosureReasonDeclaredBy, "scenario-to-cloud") {
		t.Fatalf("postgres: %+v", postgres)
	}
	if postgres.Artifact == nil || postgres.Artifact.Eligibility != domain.ClosureArtifactEligible || postgres.Artifact.Digest == "" {
		t.Fatalf("postgres must bind a linux-amd64 artifact: %+v", postgres.Artifact)
	}
	if !hasComponent(closure, domain.ClosureKindCredentialDescriptor, "vrooli/postgres:password") {
		t.Fatalf("postgres credential descriptor missing: %v", componentIDs(closure))
	}
	if closure.Sources.HostRequirements != "packages/hostreq" {
		t.Fatalf("sources: %+v", closure.Sources)
	}
	if ok, err := Verify(closure); err != nil || !ok {
		t.Fatalf("digest: ok=%v err=%v", ok, err)
	}
}
