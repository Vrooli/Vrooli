package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRelationFixture builds a store whose relation directory binds the given
// agents to one team, and returns the store dir.
func writeRelationFixture(t *testing.T, teamID string, boundAgents []string) string {
	t.Helper()
	storeDir := t.TempDir()
	dir := filepath.Join(storeDir, "relations", "team-member")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, a := range boundAgents {
		body := `{"kind":"team-member","schemaVersion":1,"teamId":"` + teamID + `","agentId":"` + a + `","status":"active"}`
		if err := os.WriteFile(filepath.Join(dir, teamID+"__"+a+".json"), []byte(body), 0o644); err != nil {
			t.Fatalf("write relation: %v", err)
		}
	}
	return storeDir
}

func TestRelationBindingDriftReportsRecordWithNoMember(t *testing.T) {
	// The 2026-07-28 marketing-crew shape: the contract kept three members
	// while four retired agents kept their relation records.
	storeDir := writeRelationFixture(t, "marketing-crew",
		[]string{"brand-manager", "producer", "oss-advertiser", "publisher"})
	members := map[string]bool{"brand-manager": true, "producer": true}

	got := checkRelationBindingDrift(storeDir, "marketing-crew", members)
	if len(got) != 2 {
		t.Fatalf("findings = %d, want 2; got %+v", len(got), got)
	}
	for _, f := range got {
		if f.Rule != "team_role_member_drift" || f.Severity != SeverityError {
			t.Fatalf("unexpected rule/severity: %+v", f)
		}
		if !strings.Contains(f.Detail, "relations/team-member/marketing-crew__") {
			t.Fatalf("detail must name the offending record path: %q", f.Detail)
		}
	}
}

func TestRelationBindingDriftReportsMemberWithNoRecord(t *testing.T) {
	// The other direction, also live on marketing-crew: producer had a member
	// directory and a role but no relation record.
	storeDir := writeRelationFixture(t, "marketing-crew", []string{"brand-manager"})
	members := map[string]bool{"brand-manager": true, "producer": true}

	got := checkRelationBindingDrift(storeDir, "marketing-crew", members)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1; got %+v", len(got), got)
	}
	if got[0].Member != "producer" {
		t.Fatalf("member = %q, want producer", got[0].Member)
	}
}

func TestRelationBindingDriftSilentWhenSurfacesAgree(t *testing.T) {
	storeDir := writeRelationFixture(t, "marketing-crew", []string{"brand-manager", "producer"})
	members := map[string]bool{"brand-manager": true, "producer": true}
	if got := checkRelationBindingDrift(storeDir, "marketing-crew", members); len(got) != 0 {
		t.Fatalf("findings = %+v, want none", got)
	}
}

func TestRelationBindingDriftSilentWithoutStoreDir(t *testing.T) {
	if got := checkRelationBindingDrift("", "marketing-crew", map[string]bool{"a": true}); got != nil {
		t.Fatalf("findings = %+v, want nil", got)
	}
	if got := checkRelationBindingDrift(t.TempDir(), "marketing-crew", map[string]bool{"a": true}); got != nil {
		t.Fatalf("no relations dir must be silent, got %+v", got)
	}
}

func TestRelationBindingDriftIgnoresOtherTeams(t *testing.T) {
	storeDir := writeRelationFixture(t, "other-team", []string{"stranger"})
	if got := checkRelationBindingDrift(storeDir, "marketing-crew", map[string]bool{}); len(got) != 0 {
		t.Fatalf("another team's records must not leak: %+v", got)
	}
}
