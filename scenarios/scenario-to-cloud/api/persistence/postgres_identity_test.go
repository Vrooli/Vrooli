package persistence

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	_ "github.com/lib/pq"
)

// TestPostgresIdentityMigrationAndConversion [REQ:STC-P0-015] proves the
// production engine path: the PostgreSQL migration adds the identity columns
// to a predecessor-shaped table, converts the legacy row, installs the unique
// index, and the resolver, fence and request-key scope behave exactly as on
// SQLite. It runs only when STC_PG_TEST_DSN names a disposable database; it
// must never be pointed at the production database.
func TestPostgresIdentityMigrationAndConversion(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("STC_PG_TEST_DSN"))
	if dsn == "" {
		t.Skip("STC_PG_TEST_DSN not set; PostgreSQL identity migration test skipped")
	}
	if strings.Contains(dsn, "vrooli_scenario_to_cloud?") || strings.HasSuffix(dsn, "vrooli_scenario_to_cloud") {
		t.Fatalf("refusing to run against the production scenario database")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS cloud_operations, cloud_recovery_operations, deployment_investigations, cloud_instances, deployments CASCADE`); err != nil {
		t.Fatalf("reset scratch schema: %v", err)
	}

	// Predecessor shape: the production DDL before this phase.
	const legacyDDL = `
	CREATE TABLE deployments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		scenario_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending'
			CHECK (status IN ('pending', 'setup_running', 'setup_complete', 'deploying', 'deployed', 'failed', 'stopped')),
		manifest JSONB NOT NULL,
		bundle_path TEXT, bundle_sha256 TEXT, bundle_size_bytes BIGINT,
		setup_result JSONB, deploy_result JSONB, preflight_result JSONB, last_inspect_result JSONB, ssh_identity JSONB,
		error_message TEXT, error_step TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_deployed_at TIMESTAMPTZ, last_inspected_at TIMESTAMPTZ,
		progress_step TEXT, progress_percent REAL DEFAULT 0,
		deployment_history JSONB DEFAULT '[]'::jsonb, completed_steps JSONB DEFAULT '[]'::jsonb, run_id TEXT
	)`
	if _, err := db.ExecContext(ctx, legacyDDL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	const legacyID = "7e493e04-0000-4000-8000-000000000002"
	legacyManifest := `{"version":"1","target":{"type":"vps","vps":{"host":"198.51.100.42","port":22,"user":"root","workdir":"/root/Vrooli","key_path":"/home/operator/.ssh/canary-key-9f3a"}},"scenario":{"id":"demo-app"},"edge":{"domain":"demo.example","caddy":{"enabled":true}}}`
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments (id, name, scenario_id, status, manifest) VALUES ($1, $2, $3, $4, $5::jsonb)`,
		legacyID, "demo-app @ demo.example", "demo-app", "deployed", legacyManifest); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "postgres"); err != nil {
		t.Fatalf("init postgres schema over legacy table: %v", err)
	}
	if err := repo.InitSchemaOnDialect(ctx, db, "postgres"); err != nil {
		t.Fatalf("second init must be idempotent: %v", err)
	}

	converted, err := repo.GetDeployment(ctx, legacyID)
	if err != nil || converted == nil {
		t.Fatalf("read converted row: %v", err)
	}
	want := identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "198.51.100.42", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
	if converted.Environment != identity.DefaultEnvironment || converted.Target != want || converted.Fence != 0 {
		t.Fatalf("converted = env=%q target=%+v fence=%d", converted.Environment, converted.Target, converted.Fence)
	}
	var rawBinding string
	if err := db.QueryRowContext(ctx, `SELECT target_binding::text FROM deployments WHERE id = $1`, legacyID).Scan(&rawBinding); err != nil {
		t.Fatalf("read raw binding: %v", err)
	}
	if strings.Contains(rawBinding, "canary-key-9f3a") {
		t.Fatalf("key_path leaked into target_binding")
	}

	byDomain, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Domain: "demo.example"})
	if err != nil || byDomain.ID != legacyID {
		t.Fatalf("resolve by domain = %+v, %v", byDomain, err)
	}
	byHost, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "198.51.100.42"})
	if err != nil || byHost.ID != legacyID {
		t.Fatalf("resolve by host = %+v, %v", byHost, err)
	}

	// Uniqueness is enforced by the engine, not by application checks.
	dup := newDeployment("2e493e04-0000-4000-8000-000000000003", "demo-app", "production", "198.51.100.42", "other.example")
	if err := repo.CreateDeployment(ctx, dup); !apierrors.Is(err, apierrors.CodeDeploymentIdentityConflict) {
		t.Fatalf("duplicate identity on postgres: %v", err)
	}
	staging := newDeployment("2e493e04-0000-4000-8000-000000000004", "demo-app", "staging", "198.51.100.42", "staging.demo.example")
	if err := repo.CreateDeployment(ctx, staging); err != nil {
		t.Fatalf("create staging: %v", err)
	}
	if _, err := identity.Resolve(ctx, repo, identity.Selector{ScenarioID: "demo-app", Host: "198.51.100.42"}); !apierrors.Is(err, apierrors.CodeDeploymentSelectorAmbiguous) {
		t.Fatalf("ambiguous host on postgres: %v", err)
	}

	fence, err := repo.BumpFence(ctx, legacyID)
	if err != nil || fence != 1 {
		t.Fatalf("bump fence = %d, %v", fence, err)
	}
	first, err := repo.CreateOperation(ctx, &domain.CloudOperation{ID: "3e493e04-0000-4000-8000-000000000005", DeploymentID: legacyID, RequestKey: "k1", PlanDigest: "sha256:a", Fence: fence})
	if err != nil || first.State != domain.OperationAdmitted {
		t.Fatalf("admit = %+v, %v", first, err)
	}
	if _, err := repo.CreateOperation(ctx, &domain.CloudOperation{ID: "3e493e04-0000-4000-8000-000000000006", DeploymentID: legacyID, RequestKey: "k1", PlanDigest: "sha256:b"}); !apierrors.Is(err, apierrors.CodeRequestKeyConflict) {
		t.Fatalf("request key conflict on postgres: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS cloud_operations, cloud_recovery_operations, deployment_investigations, cloud_instances, deployments CASCADE`); err != nil {
		t.Fatalf("cleanup scratch schema: %v", err)
	}
}
