package planworkshop

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLegacyRound(t *testing.T, root, kind, name, contents string) string {
	t.Helper()
	path := filepath.Join(root, kind, name, "workshop", "round-001.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeLegacySpec(t *testing.T, root, kind, name, contents string) {
	t.Helper()
	path := filepath.Join(root, kind, name, "spec.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLegacyHistoryBacksUpUnacceptedStateAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	legacy := writeLegacyRound(t, root, "idea", "historic", `{"round":1,"readiness":{"problem_clarity":3}}`)
	writeLegacySpec(t, root, "idea", "historic", `{"name":"historic"}`)
	now := time.Date(2026, 7, 22, 4, 0, 0, 0, time.UTC)
	report, err := MigrateLegacyHistory(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != legacyMigrationVersion || len(report.Entries) != 1 || report.ArchivedUnaccepted != 1 {
		t.Fatalf("report = %+v", report)
	}
	entry := report.Entries[0]
	if !entry.ArchivedUnaccepted || entry.BackupPath == "" {
		t.Fatalf("entry did not record archived backup: %+v", entry)
	}
	after, err := os.ReadFile(legacy)
	if err != nil || string(after) != `{"round":1,"readiness":{"problem_clarity":3}}` {
		t.Fatalf("legacy source changed: %q, %v", after, err)
	}
	backup, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.BackupPath), "round-001.json"))
	if err != nil || string(backup) != string(after) {
		t.Fatalf("backup = %q, %v", backup, err)
	}
	repeated, err := MigrateLegacyHistory(root, now.Add(time.Hour))
	if err != nil || repeated.CompletedAt != report.CompletedAt || repeated.ArchivedUnaccepted != 1 {
		t.Fatalf("repeated=%+v err=%v", repeated, err)
	}
}

func TestMigrateLegacyHistoryPreservesAcceptedAndCorruptHistory(t *testing.T) {
	root := t.TempDir()
	writeLegacyRound(t, root, "execute", "accepted", `{"round":1}`)
	writeLegacySpec(t, root, "execute", "accepted", `{"plan_acceptance":{"actor":"operator","plan_content_hash":"sha256:accepted","subject_version":"v1"}}`)
	writeLegacyRound(t, root, "research", "corrupt", `{not-json`)
	writeLegacySpec(t, root, "research", "corrupt", `{"name":"corrupt"}`)
	report, err := MigrateLegacyHistory(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Entries) != 2 || report.ArchivedUnaccepted != 1 || len(report.Errors) != 1 {
		t.Fatalf("report=%+v", report)
	}
	for _, entry := range report.Entries {
		if entry.SourcePath == "execute/accepted/workshop" && entry.ArchivedUnaccepted {
			t.Fatalf("accepted history was archived: %+v", entry)
		}
	}
}

func TestMigrateLegacyHistoryUpgradesV1Marker(t *testing.T) {
	root := t.TempDir()
	writeLegacyRound(t, root, "fix", "upgrade", `{"round":1}`)
	writeLegacySpec(t, root, "fix", "upgrade", `{"name":"upgrade"}`)
	marker := filepath.Join(root, "plan-workshops", legacyMigrationMarker)
	if err := os.MkdirAll(filepath.Dir(marker), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte(`{"version":"v1","completed_at":"old","entries":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateLegacyHistory(root, time.Now())
	if err != nil || report.Version != legacyMigrationVersion || report.ArchivedUnaccepted != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestOpenLinksMigratedLegacyHistory(t *testing.T) {
	root := t.TempDir()
	writeLegacyRound(t, root, "idea", "historic", `{"round":1}`)
	if _, err := MigrateLegacyHistory(root, time.Now()); err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewStore(root), func(Subject) (string, string, string, error) { return "v1", "", "", nil })
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "idea/historic"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	if session.LegacyHistory == nil || session.LegacyHistory.SourcePath != "idea/historic/workshop" || !session.LegacyHistory.ArchivedUnaccepted {
		t.Fatalf("legacy history = %+v", session.LegacyHistory)
	}
}
