package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	_ "modernc.org/sqlite"
)

func openIdentityTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testManifest(scenario, environment, host, domainName string) json.RawMessage {
	m := domain.CloudManifest{
		Version:     "1",
		Environment: environment,
		Target:      domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario:    domain.ManifestScenario{ID: scenario},
		Edge:        domain.ManifestEdge{Domain: domainName},
	}
	raw, _ := json.Marshal(m)
	return raw
}

func newDeployment(id, scenario, environment, host, domainName string) *domain.Deployment {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	return &domain.Deployment{
		ID:          id,
		Name:        scenario + " @ " + domainName,
		ScenarioID:  scenario,
		Environment: environment,
		Status:      domain.StatusPending,
		Manifest:    testManifest(scenario, environment, host, domainName),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// TestDeploymentIdentityStableAcrossHostChange [REQ:STC-P0-015] proves
// P03-A01: rebinding the target locator to a new address keeps the deployment
// id, and selectors follow the new address.
func TestDeploymentIdentityStableAcrossHostChange(t *testing.T) {
	db := openIdentityTestDB(t, "identity-host-change")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	dep := newDeployment("dep-1", "demo-app", "production", "203.0.113.10", "demo.example")
	if err := repo.CreateDeployment(ctx, dep); err != nil {
		t.Fatalf("create: %v", err)
	}
	if dep.Target.Transport != identity.TransportSSH || dep.Target.Locator.Host != "203.0.113.10" {
		t.Fatalf("binding not derived from manifest: %+v", dep.Target)
	}

	before, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "203.0.113.10"})
	if err != nil {
		t.Fatalf("resolve before: %v", err)
	}

	moved := dep.Target
	moved.Locator.Host = "198.51.100.7"
	if err := repo.UpdateTargetBinding(ctx, dep.ID, moved); err != nil {
		t.Fatalf("rebind: %v", err)
	}

	after, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "198.51.100.7"})
	if err != nil {
		t.Fatalf("resolve after: %v", err)
	}
	if before.ID != after.ID || after.ID != "dep-1" {
		t.Fatalf("identity changed with the address: before=%s after=%s", before.ID, after.ID)
	}
	if after.Target.Locator.Host != "198.51.100.7" || after.Target.Locator.Port != 22 {
		t.Fatalf("locator not updated: %+v", after.Target)
	}
	if _, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "203.0.113.10"}); !apierrors.Is(err, apierrors.CodeDeploymentNotFound) {
		t.Fatalf("old address must no longer resolve: %v", err)
	}
	byEnv, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Environment: "production"})
	if err != nil || byEnv.ID != "dep-1" {
		t.Fatalf("resolve by environment = %+v, %v", byEnv, err)
	}
	ref, err := repo.GetDeploymentRef(ctx, "dep-1")
	if err != nil || ref == nil || ref.Target.Key() != "host:198.51.100.7" {
		t.Fatalf("GetDeploymentRef = %+v, %v", ref, err)
	}
}

// TestTwoInstallationsResolveDistinctRecords [REQ:STC-P0-015] proves P03-A02
// against real storage: one scenario installed in two environments on two
// hosts yields two records, each reachable by environment, domain and host.
func TestTwoInstallationsResolveDistinctRecords(t *testing.T) {
	db := openIdentityTestDB(t, "identity-two-installs")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-prod", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create prod: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-staging", "demo-app", "staging", "203.0.113.20", "staging.demo.example")); err != nil {
		t.Fatalf("create staging: %v", err)
	}

	cases := map[string]identity.Selector{
		"dep-prod":    {ScenarioID: "demo-app", Environment: "production"},
		"dep-staging": {ScenarioID: "demo-app", Environment: "staging"},
	}
	for want, sel := range cases {
		got, err := identity.Resolve(ctx, repo, sel)
		if err != nil || got.ID != want {
			t.Fatalf("selector %+v resolved to %+v, %v (want %s)", sel, got, err, want)
		}
	}
	byDomain, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Domain: "staging.demo.example"})
	if err != nil || byDomain.ID != "dep-staging" {
		t.Fatalf("by domain = %+v, %v", byDomain, err)
	}
	byHost, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "203.0.113.10"})
	if err != nil || byHost.ID != "dep-prod" {
		t.Fatalf("by host = %+v, %v", byHost, err)
	}
	byID, err := identity.Resolve(ctx, repo, identity.Selector{ID: "dep-staging"})
	if err != nil || byID.ID != "dep-staging" || byID.Environment != "staging" {
		t.Fatalf("by id = %+v, %v", byID, err)
	}

	env := "staging"
	listed, err := repo.ListDeployments(ctx, domain.ListFilter{Environment: &env})
	if err != nil || len(listed) != 1 || listed[0].ID != "dep-staging" {
		t.Fatalf("list by environment = %v, %v", listed, err)
	}
}

// TestAmbiguousSelectorIsTypedConflictFromStorage [REQ:STC-P0-015] proves
// P03-A03 end to end: two environments on one host make (scenario, host)
// ambiguous and the resolver refuses with the typed conflict.
func TestAmbiguousSelectorIsTypedConflictFromStorage(t *testing.T) {
	db := openIdentityTestDB(t, "identity-ambiguous")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-b", "demo-app", "staging", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create b: %v", err)
	}
	for _, sel := range []identity.Selector{
		{ScenarioID: "demo-app", Host: "203.0.113.10"},
		{ScenarioID: "demo-app", Domain: "demo.example"},
	} {
		_, err := identity.Resolve(ctx, repo, sel)
		typed := apierrors.As(err)
		if typed.Code != apierrors.CodeDeploymentSelectorAmbiguous || typed.Status() != 409 {
			t.Fatalf("selector %+v: code=%q status=%d err=%v", sel, typed.Code, typed.Status(), err)
		}
	}
}

// TestDuplicateIdentityIsRefused [REQ:STC-P0-015] proves the uniqueness
// constraint: a second record for the same (scenario, environment, target) is
// a typed deployment_identity_conflict, on create and on rebind.
func TestDuplicateIdentityIsRefused(t *testing.T) {
	db := openIdentityTestDB(t, "identity-duplicate")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create a: %v", err)
	}
	err := repo.CreateDeployment(ctx, newDeployment("dep-b", "demo-app", "production", "203.0.113.10", "other.example"))
	if !apierrors.Is(err, apierrors.CodeDeploymentIdentityConflict) {
		t.Fatalf("duplicate create: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-c", "demo-app", "production", "203.0.113.11", "other.example")); err != nil {
		t.Fatalf("create c: %v", err)
	}
	err = repo.UpdateTargetBinding(ctx, "dep-c", identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10"}})
	if !apierrors.Is(err, apierrors.CodeDeploymentIdentityConflict) {
		t.Fatalf("duplicate rebind: %v", err)
	}
}

// TestBumpFenceIsMonotonic proves the fence increments per acquisition and
// is visible on the record.
func TestBumpFenceIsMonotonic(t *testing.T) {
	db := openIdentityTestDB(t, "identity-fence")
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create: %v", err)
	}
	first, err := repo.BumpFence(ctx, "dep-a")
	if err != nil || first != 1 {
		t.Fatalf("first bump = %d, %v", first, err)
	}
	second, err := repo.BumpFence(ctx, "dep-a")
	if err != nil || second != 2 {
		t.Fatalf("second bump = %d, %v", second, err)
	}
	d, err := repo.GetDeployment(ctx, "dep-a")
	if err != nil || d.Fence != 2 {
		t.Fatalf("stored fence = %d, %v", d.Fence, err)
	}
	if _, err := repo.BumpFence(ctx, "missing"); err == nil {
		t.Fatalf("bumping a missing deployment must fail")
	}
}

// legacyDeploymentsDDL is the predecessor SQLite shape before the identity
// columns existed. It mirrors the production PostgreSQL row layout.
const legacyDeploymentsDDL = `
CREATE TABLE deployments (
	 id TEXT PRIMARY KEY,
	 name TEXT NOT NULL,
	 scenario_id TEXT NOT NULL,
	 status TEXT NOT NULL DEFAULT 'pending',
	 manifest TEXT NOT NULL,
	 bundle_path TEXT,
	 bundle_sha256 TEXT,
	 bundle_size_bytes INTEGER,
	 setup_result TEXT,
	 deploy_result TEXT,
	 preflight_result TEXT,
	 last_inspect_result TEXT,
	 ssh_identity TEXT,
	 error_message TEXT,
	 error_step TEXT,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 last_deployed_at TIMESTAMP,
	 last_inspected_at TIMESTAMP,
	 progress_step TEXT,
	 progress_percent REAL DEFAULT 0,
	 deployment_history TEXT DEFAULT '[]',
	 completed_steps TEXT DEFAULT '[]',
	 run_id TEXT
);`

// TestLegacyRecordConversionPreservesIdentityAndBinding [REQ:STC-P0-015]
// proves P03-A05 with a row shaped like the real production record: schema
// init converts it in place, keeps its id, defaults the environment to
// production, binds the SSH locator from the manifest, and leaves key_path in
// the manifest only.
func TestLegacyRecordConversionPreservesIdentityAndBinding(t *testing.T) {
	db := openIdentityTestDB(t, "identity-legacy-conversion")
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, legacyDeploymentsDDL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	const legacyID = "7e493e04-0000-4000-8000-000000000001"
	legacyManifest := `{
		"version": "1",
		"target": {"type": "vps", "vps": {"host": "138.197.95.182", "port": 22, "user": "root", "workdir": "/root/Vrooli", "key_path": "/home/operator/.ssh/canary-key-9f3a"}},
		"scenario": {"id": "landing-page-business-suite", "ref": "1.2.3"},
		"dependencies": {},
		"bundle": {"include_packages": true, "include_autoheal": true},
		"ports": {"api": 15000, "ui": 3000},
		"edge": {"domain": "vrooli.com", "caddy": {"enabled": true}}
	}`
	if _, err := db.ExecContext(ctx,
		`INSERT INTO deployments (id, name, scenario_id, status, manifest, bundle_sha256, ssh_identity) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		legacyID, "landing-page-business-suite @ vrooli.com", "landing-page-business-suite", "deployed", legacyManifest, "abc123", `{"fingerprint":"SHA256:legacy"}`,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema over legacy table: %v", err)
	}

	d, err := repo.GetDeployment(ctx, legacyID)
	if err != nil || d == nil {
		t.Fatalf("read converted row: %v", err)
	}
	if d.ID != legacyID || d.ScenarioID != "landing-page-business-suite" || d.Status != domain.StatusDeployed {
		t.Fatalf("identity not preserved: %+v", d)
	}
	if d.Environment != identity.DefaultEnvironment {
		t.Fatalf("environment = %q, want production", d.Environment)
	}
	if d.Fence != 0 {
		t.Fatalf("fence = %d, want 0", d.Fence)
	}
	want := identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "138.197.95.182", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
	if d.Target != want {
		t.Fatalf("target binding = %+v, want %+v", d.Target, want)
	}
	if d.Target.MachineID != "" || d.Target.NodeID != "" {
		t.Fatalf("legacy SSH target must not invent a machine identity: %+v", d.Target)
	}

	var rawBinding string
	if err := db.QueryRowContext(ctx, `SELECT target_binding FROM deployments WHERE id = $1`, legacyID).Scan(&rawBinding); err != nil {
		t.Fatalf("read raw binding: %v", err)
	}
	if strings.Contains(rawBinding, "key_path") || strings.Contains(rawBinding, "canary-key-9f3a") {
		t.Fatalf("key_path leaked into target_binding: %s", rawBinding)
	}
	if !strings.Contains(string(d.Manifest), "canary-key-9f3a") {
		t.Fatalf("manifest key_path must remain untouched until the reach layer retires it")
	}
	if d.SSHIdentity.Data == nil || d.BundleSHA256 == nil || *d.BundleSHA256 != "abc123" {
		t.Fatalf("sibling columns lost in conversion: %+v", d)
	}

	ref, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "landing-page-business-suite", Domain: "vrooli.com"})
	if err != nil || ref.ID != legacyID {
		t.Fatalf("resolve converted row by domain = %+v, %v", ref, err)
	}
	ref, err = identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "landing-page-business-suite", Host: "138.197.95.182"})
	if err != nil || ref.ID != legacyID {
		t.Fatalf("resolve converted row by host = %+v, %v", ref, err)
	}

	// Init is idempotent: a second run must not rewrite a converted row.
	if err := repo.UpdateTargetBinding(ctx, legacyID, identity.TargetRef{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 3, Transport: identity.TransportBridge, Locator: want.Locator}); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("second init: %v", err)
	}
	again, err := repo.GetDeploymentRef(ctx, legacyID)
	if err != nil || again.Target.MachineID != "machine-1" || again.Target.EnrollmentGeneration != 3 {
		t.Fatalf("second init rewrote the binding: %+v, %v", again, err)
	}
}

// TestLegacyConversionHonoursManifestEnvironment proves a predecessor row
// whose manifest names an environment keeps it instead of the default.
func TestLegacyConversionHonoursManifestEnvironment(t *testing.T) {
	db := openIdentityTestDB(t, "identity-legacy-env")
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, legacyDeploymentsDDL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments (id, name, scenario_id, status, manifest) VALUES ($1, $2, $3, $4, $5)`,
		"dep-staging", "demo", "demo-app", "deployed", string(testManifest("demo-app", "staging", "203.0.113.5", "staging.demo.example"))); err != nil {
		t.Fatalf("insert: %v", err)
	}
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init: %v", err)
	}
	ref, err := repo.GetDeploymentRef(ctx, "dep-staging")
	if err != nil || ref.Environment != "staging" || ref.Target.Locator.Host != "203.0.113.5" {
		t.Fatalf("converted ref = %+v, %v", ref, err)
	}
}
