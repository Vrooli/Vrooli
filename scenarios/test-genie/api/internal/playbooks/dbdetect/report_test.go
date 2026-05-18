package dbdetect_test

import (
	"strings"
	"testing"

	"test-genie/internal/playbooks/dbdetect"
)

func TestFormatHumanRequired(t *testing.T) {
	rep := dbdetect.DetectionReport{
		Order: []string{"postgres", "redis", "sqlite"},
		Results: map[string]dbdetect.DBResult{
			"postgres": {
				DB:       "postgres",
				Required: true,
				Decision: &dbdetect.Evidence{Source: "manifest:resource[type=postgres]", Priority: dbdetect.PriorityHigh},
				Corroborating: []dbdetect.Evidence{
					{Source: "godeps:postgres-driver", Priority: dbdetect.PriorityMedium},
				},
			},
			"redis": {DB: "redis"},
			"sqlite": {
				DB:       "sqlite",
				Required: true,
				Decision: &dbdetect.Evidence{Source: "godeps:sqlite-driver", Priority: dbdetect.PriorityHigh},
				Corroborating: []dbdetect.Evidence{
					{Source: "source:sqlite-tokens", Priority: dbdetect.PriorityMedium, Locations: []string{"a.go", "b.go", "c.go"}},
				},
			},
		},
	}
	out := rep.FormatHuman()
	for _, want := range []string{
		"db-detect:",
		"postgres:",
		"required",
		"manifest:resource[type=postgres]",
		"+ godeps:postgres-driver",
		"redis:",
		"not needed",
		"sqlite:",
		"+ source:sqlite-tokens ×3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatHumanSQLiteNoEvidence(t *testing.T) {
	rep := dbdetect.DetectionReport{
		Order: []string{"sqlite"},
		Results: map[string]dbdetect.DBResult{
			"sqlite": {DB: "sqlite"},
		},
	}
	out := rep.FormatHuman()
	if !strings.Contains(out, "not needed") {
		t.Fatalf("expected 'not needed', got: %s", out)
	}
	if !strings.Contains(out, "manifest declares no sqlite resource") {
		t.Fatalf("expected sqlite annotation, got: %s", out)
	}
}

func TestFormatHumanConflict(t *testing.T) {
	rep := dbdetect.DetectionReport{
		Order: []string{"postgres"},
		Results: map[string]dbdetect.DBResult{
			"postgres": {
				DB:       "postgres",
				Required: true,
				Decision: &dbdetect.Evidence{Source: "manifest:resource[type=postgres]", Priority: dbdetect.PriorityHigh},
				Conflicts: []dbdetect.Conflict{
					{Kind: "missing-corroboration", Detail: "godeps:postgres-driver"},
				},
			},
		},
	}
	out := rep.FormatHuman()
	if !strings.Contains(out, "! missing-corroboration: godeps:postgres-driver") {
		t.Fatalf("expected conflict line, got: %s", out)
	}
}
