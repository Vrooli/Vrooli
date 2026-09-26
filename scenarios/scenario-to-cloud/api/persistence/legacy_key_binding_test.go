package persistence

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-032] A legacy row whose manifest still carries
// target.vps.key_path is converted at startup into the credential binding
// vrooli/scenario-to-cloud:ssh-key that names the operator-held key file.
// The manifest column is not rewritten, no key bytes are stored, and running
// the conversion again changes nothing.
func TestLegacyKeyPathBecomesACredentialBinding(t *testing.T) {
	db := openIdentityTestDB(t, "legacy-key-binding-conversion")
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, legacyDeploymentsDDL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	// Shaped like the production row: identity columns absent, key_path on
	// the manifest, an explicit_key ssh_identity derived from it.
	const legacyID = "7e493e04-0000-4000-8000-00000000c0de"
	const keyPath = "/home/operator/.ssh/canary-key-9f3a"
	legacyManifest := `{
		"version": "1",
		"target": {"type": "vps", "vps": {"host": "138.197.95.182", "port": 22, "user": "root", "workdir": "/root/Vrooli", "key_path": "` + keyPath + `"}},
		"scenario": {"id": "landing-page-business-suite", "ref": "1.2.3"},
		"dependencies": {},
		"bundle": {"include_packages": true, "include_autoheal": true},
		"ports": {"api": 15000, "ui": 3000},
		"edge": {"domain": "vrooli.com", "caddy": {"enabled": true}}
	}`
	if _, err := db.ExecContext(ctx,
		`INSERT INTO deployments (id, name, scenario_id, status, manifest, bundle_sha256, ssh_identity) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		legacyID, "landing-page-business-suite @ vrooli.com", "landing-page-business-suite", "deployed", legacyManifest, "abc123",
		`{"key_path":"`+keyPath+`","auth_mode":"explicit_key","verification_state":"authorized"}`,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	repo := NewRepository(db)
	for pass := 1; pass <= 2; pass++ {
		if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
			t.Fatalf("init schema pass %d: %v", pass, err)
		}
	}

	resolved, err := credentials.ResolveSSHKeyPath(ctx, repo, legacyID)
	if err != nil {
		t.Fatalf("resolve key path: %v", err)
	}
	if resolved != keyPath {
		t.Fatalf("resolved key path = %q, want %q", resolved, keyPath)
	}
	bindings, err := repo.ListBindings(ctx, legacyID)
	if err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected exactly one binding after two conversions, got %d", len(bindings))
	}
	b := bindings[0]
	if b.Descriptor != credentials.SSHKeyDescriptor || b.Class != domain.CredentialClassMachineEnrollment || b.SourceClass != domain.SecretClassInfrastructure || b.Target.Type != "file" || b.State != domain.CredentialBindingMaterialized {
		t.Fatalf("binding shape = %+v", b)
	}

	// The row's manifest column is untouched and the binding holds a
	// locator, not a value: nothing but the path string appears anywhere.
	var rawManifest string
	if err := db.QueryRowContext(ctx, `SELECT manifest FROM deployments WHERE id = $1`, legacyID).Scan(&rawManifest); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rawManifest, keyPath) {
		t.Fatal("conversion must not rewrite the stored manifest")
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cloud_credential_bindings WHERE deployment_id = $1 AND field = 'ssh-key'`, legacyID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("binding rows = %d err=%v", count, err)
	}

	// A row without a key path gets no binding.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO deployments (id, name, scenario_id, status, manifest, environment, target_binding, target_key) VALUES ($1, $2, $3, $4, $5, 'staging', '{"transport":"ssh","locator":{"host":"203.0.113.9"}}', 'ssh:203.0.113.9')`,
		"7e493e04-0000-4000-8000-00000000beef", "ambient", "landing-page-business-suite", "pending",
		`{"version":"1","target":{"type":"vps","vps":{"host":"203.0.113.9","port":22,"user":"root","workdir":"/root/Vrooli"}},"scenario":{"id":"landing-page-business-suite"}}`,
	); err != nil {
		t.Fatalf("insert ambient row: %v", err)
	}
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema after ambient row: %v", err)
	}
	if resolved, err := credentials.ResolveSSHKeyPath(ctx, repo, "7e493e04-0000-4000-8000-00000000beef"); err != nil || resolved != "" {
		t.Fatalf("ambient row must resolve no key path, got %q err=%v", resolved, err)
	}
}
