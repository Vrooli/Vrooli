package monetization

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestLiveMonetizationManifestConforms(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
	for _, scenario := range []string{"landing-page-business-suite"} {
		findings := scan(filepath.Join(root, "scenarios", scenario))
		for _, finding := range findings {
			t.Errorf("%s live conformance finding %s at %s: %s", scenario, finding.Code, finding.Location, finding.Message)
		}
	}
	if findings := scan(filepath.Join(root, "scenarios", "browser-automation-studio")); len(findings) != 0 {
		t.Errorf("browser-automation-studio should have no paid-surface findings: %v", findings)
	}
	for _, scenario := range []string{"web-console"} {
		findings := scan(filepath.Join(root, "scenarios", scenario))
		if len(findings) != 0 {
			t.Errorf("%s has unexpected paid-surface findings: %v", scenario, findings)
		}
	}
}

func TestLiveFindingsReconcileFixtureDatabase(t *testing.T) {
	root := writeFixture(t, fixtureOptions{})
	db := openCatalogFixture(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO download_apps(app_key, bundle_key) VALUES ('fixture', 'business_suite')`); err != nil {
		t.Fatalf("insert app: %v", err)
	}
	for _, tier := range []string{"free", "pro"} {
		if _, err := db.Exec(`INSERT INTO subscription_tier_limits(tier_id, limit_key, app_bundle_key) VALUES (?, 'workflow_executions', 'business_suite')`, tier); err != nil {
			t.Fatalf("insert tier %s: %v", tier, err)
		}
	}

	h := Handler{catalog: db, bundleKey: func() string { return "business_suite" }}
	findings, err := h.liveFindings(context.Background(), root)
	if err != nil {
		t.Fatalf("live reconciliation: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("conforming live fixture emitted findings: %v", findings)
	}

	if _, err := db.Exec(`DELETE FROM subscription_tier_limits WHERE limit_key = 'workflow_executions'`); err != nil {
		t.Fatalf("delete tier limit: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM download_apps WHERE app_key = 'fixture'`); err != nil {
		t.Fatalf("delete app: %v", err)
	}
	findings, err = h.liveFindings(context.Background(), root)
	if err != nil {
		t.Fatalf("live reconciliation after mutation: %v", err)
	}
	if !hasFindingCode(findings, "money.unregistered_app_key") || !hasFindingCode(findings, "money.meter_missing_tier_limits") {
		t.Fatalf("live catalog mutation was not reconciled: %v", findings)
	}
}

func TestLiveFindingsReturnCatalogUnavailable(t *testing.T) {
	root := writeFixture(t, fixtureOptions{})
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open empty catalog: %v", err)
	}
	defer db.Close()
	h := Handler{catalog: db, bundleKey: func() string { return "business_suite" }}
	if _, err := h.liveFindings(context.Background(), root); err == nil {
		t.Fatal("expected an unavailable catalog error")
	}
}

func openCatalogFixture(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open catalog fixture: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE download_apps (app_key TEXT NOT NULL, bundle_key TEXT NOT NULL)`,
		`CREATE TABLE subscription_tier_limits (tier_id TEXT NOT NULL, limit_key TEXT NOT NULL, app_bundle_key TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			t.Fatalf("create catalog fixture table: %v", err)
		}
	}
	return db
}

func hasFindingCode(findings []*commonv1.AssessmentFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
