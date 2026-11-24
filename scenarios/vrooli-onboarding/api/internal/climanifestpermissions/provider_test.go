package climanifestpermissions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

func TestSnapshotUsesValidatedCanonicalManifestsAndSeparatesDeclarationsFromGrants(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := New(root).snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.SchemaID != SchemaID || snapshot.SourceDigest == "" || snapshot.SourceRevision == "" {
		t.Fatalf("snapshot source binding incomplete: %+v", snapshot)
	}
	if snapshot.AuthorizationNote == "" {
		t.Fatal("authorization distinction missing")
	}
	found := false
	for _, item := range snapshot.Declarations {
		if item.ProcessSpawnDeclared {
			found = true
			if item.AuthorizationSource == "" || item.DerivedScope == "" {
				t.Fatalf("process declaration lacks effective gate provenance: %+v", item)
			}
			if item.RunEligible != item.DispatchEligible {
				// Dispatch eligibility is sourced from the catalog and should agree
				// with the same manifest field for this exact command.
				t.Fatalf("manifest/catalog dispatch gate drift: %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("expected at least one canonical process:spawn declaration")
	}
}

func TestSnapshotFailsClosedWhenCanonicalManifestIsMalformed(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".vrooli", "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(root, SchemaPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, SchemaPath), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cli", "manifest.json"), []byte(`{"name":"vrooli","groups":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(tmp).snapshot(context.Background()); err == nil {
		t.Fatal("malformed manifest unexpectedly admitted")
	}
}

func TestReadAuthorizationRefusesStaleManifestSource(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli", "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	actualRoot, err := filepath.Abs("../../../../../")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(actualRoot, SchemaPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, SchemaPath), schema, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"vrooli","groups":[{"name":"ops","commands":[{"name":"spawn","binding":{"kind":"local","handler":"spawn"},"governance":{"effect":"write","run_eligible":true,"permissions":["process:spawn"]}}]}]}`
	manifestPath := filepath.Join(root, "cli", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true})
	p := New(root)
	descriptors, err := p.List(ctx)
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("List() = %+v, err=%v", descriptors, err)
	}
	if !strings.Contains(descriptors[0].SchemaVersion, "cli-manifest/v1#sha256:") {
		t.Fatalf("descriptor is not version/source-bound: %+v", descriptors[0])
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(manifest, `"write"`, `"read"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = p.AuthorizeRead(ctx, descriptors[0], permissions.ReadRequest{ActorID: "operator", Owner: Owner})
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale source error = %v, want FailedPrecondition", err)
	}
}
