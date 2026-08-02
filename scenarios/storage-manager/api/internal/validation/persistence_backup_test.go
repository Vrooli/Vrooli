package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeServiceJSON writes a .vrooli/service.json under dir with the given body.
func writeServiceJSON(t *testing.T, dir, body string) {
	t.Helper()
	vd := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vd, "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

func TestDependencyKeysBothShapes(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want []string
	}{
		"array of strings":   {`["postgres","redis"]`, []string{"postgres", "redis"}},
		"object map":         {`{"postgres":{"type":"postgres"},"qdrant":{}}`, []string{"postgres", "qdrant"}},
		"array of objects":   {`[{"type":"postgres"},{"id":"redis"}]`, []string{"postgres", "redis"}},
		"empty array":        {`[]`, nil},
		"empty object":       {`{}`, nil},
		"dedupe + lowercase": {`["Postgres","postgres"]`, []string{"postgres"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := dependencyKeys([]byte(tc.raw))
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestDetectEnginesObjectMapShape(t *testing.T) {
	dir := t.TempDir()
	// Older-style object-map resources — must still classify Postgres.
	writeServiceJSON(t, dir, `{"dependencies":{"resources":{"postgres":{"type":"postgres"},"redis":{"type":"redis"}}}}`)
	engines := detectEngines(dir)
	hasPG, hasRedis := false, false
	for _, e := range engines {
		if e == EnginePostgres {
			hasPG = true
		}
		if e == EngineRedis {
			hasRedis = true
		}
	}
	if !hasPG || !hasRedis {
		t.Fatalf("expected postgres+redis from object-map resources, got %v", engines)
	}
}

func TestHasBackupTarget(t *testing.T) {
	t.Run("dependency on data-backup-manager", func(t *testing.T) {
		dir := t.TempDir()
		writeServiceJSON(t, dir, `{"dependencies":{"scenarios":["data-backup-manager"]}}`)
		if !hasBackupTarget(dir) {
			t.Fatal("expected backup target via dependency")
		}
	})
	t.Run("backup block", func(t *testing.T) {
		dir := t.TempDir()
		writeServiceJSON(t, dir, `{"backup":{"destination":"x"}}`)
		if !hasBackupTarget(dir) {
			t.Fatal("expected backup target via backup block")
		}
	})
	t.Run("none", func(t *testing.T) {
		dir := t.TempDir()
		writeServiceJSON(t, dir, `{"dependencies":{"scenarios":["code-facts"]}}`)
		if hasBackupTarget(dir) {
			t.Fatal("expected no backup target")
		}
	})
}

func TestBackupTargetMissingAnalyzer(t *testing.T) {
	a := backupTargetMissing{}

	// Greenfield is exempt even without a backup target.
	if a.Applies(AnalyzerContext{Engines: []Engine{EnginePostgres}, StorageStage: "greenfield"}) {
		t.Fatal("greenfield should be exempt")
	}
	// Redis-only (cache) is not data-persisting.
	if a.Applies(AnalyzerContext{Engines: []Engine{EngineRedis}, StorageStage: "production"}) {
		t.Fatal("redis-only should not be data-persisting")
	}

	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"maturity":"production","dependencies":{"resources":["postgres"]}}`)
	ac := AnalyzerContext{Scenario: "svc", ScenarioDir: dir, Engines: []Engine{EnginePostgres}, StorageStage: "production"}
	if !a.Applies(ac) {
		t.Fatal("deployed postgres scenario should apply")
	}
	findings, err := a.Analyze(context.Background(), ac)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(findings) != 1 || findings[0].Code != "BACKUP_TARGET_MISSING" {
		t.Fatalf("expected BACKUP_TARGET_MISSING, got %+v", findings)
	}

	// With a backup dependency declared, no finding.
	writeServiceJSON(t, dir, `{"maturity":"production","dependencies":{"resources":["postgres"],"scenarios":["data-backup-manager"]}}`)
	findings, _ = a.Analyze(context.Background(), ac)
	if len(findings) != 0 {
		t.Fatalf("expected no finding with backup target, got %+v", findings)
	}
}

func TestMigrationDebtNote(t *testing.T) {
	cases := []struct {
		stage         string
		hasMigrations bool
		wantNote      bool
	}{
		{"production", false, true},  // deployed, no migration path
		{"pilot", false, true},       // deployed, no migration path
		{"production", true, false},  // deployed with migrations — fine
		{"greenfield", true, true},   // greenfield carrying migrations — debt
		{"greenfield", false, false}, // greenfield, schema-as-state — fine
		{"", false, false},           // default greenfield, fine
	}
	for _, tc := range cases {
		note, rem := migrationDebtNote(tc.stage, tc.hasMigrations)
		if (note != "") != tc.wantNote {
			t.Fatalf("stage=%q hasMig=%v: note=%q wantNote=%v", tc.stage, tc.hasMigrations, note, tc.wantNote)
		}
		if tc.wantNote && rem == "" {
			t.Fatalf("stage=%q: expected remediation", tc.stage)
		}
	}
}
