package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadTeamKeepsReadableInvalidTeamVisibleWithFindings(t *testing.T) {
	s := setupStateTestStore(t)
	teamPath := filepath.Join(s.StoreDir(), "teams", "team-1", "team.json")
	team, err := LoadJSON[Team](teamPath)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	team.Runtime.Mode = "invalid-runtime"
	if err := SaveJSON(teamPath, team); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}

	got, err := s.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("Get invalid but readable team: %v", err)
	}
	if got == nil || got.ID != "team-1" {
		t.Fatalf("Get team = %+v, want visible team", got)
	}
	if len(got.ValidationFindings) < 1 {
		t.Fatalf("validation findings = %+v, want runtime configuration defect", got.ValidationFindings)
	}
	if err := s.Update(context.Background(), "team-1", got); err == nil {
		t.Fatal("Update accepted invalid contract")
	}
}
