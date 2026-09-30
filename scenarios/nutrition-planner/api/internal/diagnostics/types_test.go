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
	if _, err := db.Exec(`INSERT INTO plans(workspace_id,revision,plan_json,updated_at) VALUES('w1',1,'{"unresolved":[{"code":"open_slot"}]}',?)`, old); err != nil {
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
