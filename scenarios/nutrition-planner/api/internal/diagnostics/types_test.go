package diagnostics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	cost "nutrition-planner/internal/cost"
	jobs "nutrition-planner/internal/jobs"
	nutrition "nutrition-planner/internal/nutrition"
	planning "nutrition-planner/internal/planning"
	recipe "nutrition-planner/internal/recipe"
)

func TestBuildReportsScopedActionableFindings(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, schema := range []string{jobs.Schema(), nutrition.Schema(), cost.Schema(), recipe.Schema(), planning.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	old := now.Add(-31 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO jobs(id,workspace_id,type,dedup_key,state,created_at,updated_at) VALUES('j1','w1','research','d1','failed',?,?)`, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,active) VALUES('t1',1,'w1','b12','','','local_day','planned','required','manual',?,1)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO price_observations(id,workspace_id,item_id,package_amount,package_unit,price_minor,currency,currency_exponent,observed_at,source) VALUES('p1','w1','rice','1','kg',100,'USD',2,?,'manual')`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,groups_json,status,created_at) VALUES('r1',1,'w1','Rice','[{"item":"rice","status":"unknown"}]','draft',?)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO plan_state(workspace_id,revision,metadata_json,unresolved_json,updated_at) VALUES('w1',1,'{}','[{"code":"open_slot"}]',?)`, old); err != nil {
		t.Fatal(err)
	}
	report, err := Build(context.Background(), db, "w1", now, []ProviderStatus{{Name: "ai_gateway", State: "not_configured"}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.DatabaseOK || report.SchemaVersion != 0 || len(report.Providers) != 1 || len(report.Findings) != 5 {
		t.Fatalf("report=%#v", report)
	}
	for _, finding := range report.Findings {
		if finding.Count != 1 {
			t.Fatalf("finding=%#v", finding)
		}
	}
	other, err := Build(context.Background(), db, "w2", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Findings) != 0 {
		t.Fatalf("workspace leakage: %#v", other.Findings)
	}
}

func TestImageGenerationClassificationFailsClosed(t *testing.T) {
	tests := []struct {
		name           string
		serviceFound   bool
		readSucceeded  bool
		readyStates    []string
		wantCapability string
	}{
		{name: "service missing", wantCapability: "unknown"},
		{name: "read failure", serviceFound: true, wantCapability: "unknown"},
		{name: "empty candidates", serviceFound: true, readSucceeded: true, wantCapability: "unavailable"},
		{name: "known non-ready candidate", serviceFound: true, readSucceeded: true, readyStates: []string{"needs_model_install"}, wantCapability: "unavailable"},
		{name: "environment not provisioned", serviceFound: true, readSucceeded: true, readyStates: []string{"env_not_provisioned"}, wantCapability: "unavailable"},
		{name: "smoke failed", serviceFound: true, readSucceeded: true, readyStates: []string{"smoke_failed"}, wantCapability: "unavailable"},
		{name: "ready candidate", serviceFound: true, readSucceeded: true, readyStates: []string{"disabled", "ready"}, wantCapability: "available"},
		{name: "unknown candidate state", serviceFound: true, readSucceeded: true, readyStates: []string{"future_state"}, wantCapability: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ImageGenerationFromCandidates(tt.serviceFound, tt.readSucceeded, tt.readyStates)
			if status.Capability != tt.wantCapability {
				t.Errorf("capability = %q, want %q", status.Capability, tt.wantCapability)
			}
			if status.Permission != "off" || status.QuoteAvailable || status.DispatchAllowed {
				t.Errorf("classification did not keep generation disabled: %#v", status)
			}
			if status.Reason == "" {
				t.Error("classification reason is empty")
			}
		})
	}
}
