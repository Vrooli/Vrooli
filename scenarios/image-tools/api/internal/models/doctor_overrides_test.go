package models

import (
	"strings"
	"testing"
	"time"

	"image-tools/internal/catalogstate"
)

func TestDoctorOverridesNamesEveryOverrideThatIsNotInvisible(t *testing.T) {
	yes := true
	overrides := map[string]catalogstate.Override{
		"legacy":     {ID: "legacy", Enabled: false, SetBy: catalogstate.UnknownActor, SeedEnabled: &yes},
		"agent":      {ID: "agent", Enabled: false, SetBy: "agent-3", Reason: "judge flagged it", SeedEnabled: &yes, SetAt: time.Now()},
		"pinned":     {ID: "pinned", Enabled: false, SetBy: "operator", Reason: "licence", Pinned: true, SeedEnabled: &yes},
		"agrees":     {ID: "agrees", Enabled: true, SetBy: "operator", SeedEnabled: &yes},
		"superseded": {ID: "superseded", Enabled: false, SetBy: "agent-9", SupersededAt: time.Now(), SupersededNote: "seed enabled changed false to true"},
		"gone":       {ID: "gone", Enabled: false, SetBy: "agent-9"},
	}
	seed := func(id string) (bool, bool) { return true, id != "gone" }
	got := map[string]CatalogFinding{}
	for _, f := range DoctorOverrides(overrides, seed) {
		got[f.ModelID] = f
	}
	if len(got) != 4 {
		t.Fatalf("findings = %+v, want legacy, agent, pinned and superseded only", got)
	}
	if f := got["legacy"]; f.Severity != FindingWarning || !strings.Contains(f.Message, "models reset legacy") || !strings.Contains(f.Message, "--pin") {
		t.Fatalf("legacy finding = %+v", f)
	}
	if f := got["agent"]; f.Severity != FindingInfo || !strings.Contains(f.Message, "agent-3") || !strings.Contains(f.Message, "judge flagged it") || !strings.Contains(f.Message, "supersede") {
		t.Fatalf("agent finding = %+v", f)
	}
	if f := got["pinned"]; f.Severity != FindingInfo || !strings.Contains(f.Message, "pinned, so it stays") {
		t.Fatalf("pinned finding = %+v", f)
	}
	if f := got["superseded"]; f.Code != "override_superseded" || !strings.Contains(f.Message, "no longer applies") {
		t.Fatalf("superseded finding = %+v", f)
	}
}
