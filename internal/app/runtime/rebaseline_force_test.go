package runtimeapp

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/rootauthority"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func forceDecision() rootauthority.Decision {
	now := time.Now().UTC()
	return rootauthority.Decision{
		GrantID: "grant-1", ApprovedBy: "owner", RunID: "run-1",
		Operation: rootauthority.Operation, Target: rootauthority.Target("host", "/home/owner/.vrooli"),
		VerifiedAt: now, ExpiresAt: now.Add(time.Hour),
	}
}

// runtimeHomeFor resolves the runtime home a store path is relative to. The
// forced-transition API takes the OS home, because that is what every other
// root owner resolves; a store path is never relative to $HOME itself.
func runtimeHomeFor(t *testing.T, osHome string) string {
	t.Helper()
	home, err := repocontract.VrooliUserRoot(osHome)
	if err != nil {
		t.Fatalf("resolve runtime home: %v", err)
	}
	return home
}

func TestPrepareForceControlsCapturesAndRestoresDeclaredState(t *testing.T) {
	withRepositoryCodecStamp(t)
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	dbPath := filepath.Join(runtimeHome, "state", "runtime.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateInstance(ctx, scenarioruntime.Instance{InstanceID: "before", Scenario: "demo"}); err != nil {
		t.Fatal(err)
	}
	controls, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "cross the additive schema", CandidateSHA: "cand"})
	if err != nil {
		t.Fatalf("prepare controls: %v", err)
	}
	if len(controls.Receipt.SnapshotFiles) != 1 || controls.Receipt.SnapshotFiles[0].Path != "state/runtime.db" {
		t.Fatalf("snapshot did not capture the declared store: %+v", controls.Receipt.SnapshotFiles)
	}
	if !strings.HasPrefix(controls.Dir, runtimeHome+string(os.PathSeparator)) {
		t.Fatalf("snapshot %s is outside the runtime home %s", controls.Dir, runtimeHome)
	}
	if !strings.HasPrefix(controls.ReceiptAt, runtimeHome+string(os.PathSeparator)) {
		t.Fatalf("receipt %s is outside the runtime home %s", controls.ReceiptAt, runtimeHome)
	}
	if controls.Receipt.SnapshotSHA == "" || controls.Receipt.Authority.GrantID != "grant-1" {
		t.Fatalf("receipt is incomplete: %+v", controls.Receipt)
	}
	if _, err := os.Stat(controls.ReceiptAt); err != nil {
		t.Fatalf("receipt was not written: %v", err)
	}
	if controls.Receipt.Reason != "cross the additive schema" {
		t.Fatalf("reason = %q", controls.Receipt.Reason)
	}

	if _, err := store.CreateInstance(ctx, scenarioruntime.Instance{InstanceID: "after", Scenario: "demo"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runtime_instances WHERE instance_id = 'after'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("restore kept post-snapshot state: %d rows", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM runtime_instances WHERE instance_id = 'before'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("restore lost the snapshotted row")
	}
}

func TestPrepareForceControlsRefusesWithoutAuthorityOrSnapshot(t *testing.T) {
	withRepositoryCodecStamp(t)
	ctx := context.Background()
	home := t.TempDir()
	if _, err := PrepareForceControls(ctx, home, ForceRequest{Reason: "no authority"}); err == nil {
		t.Fatal("accepted a forced transition with no verified authority")
	}
	// A fresh home has no state file yet, so the snapshot is empty but legal:
	// there is nothing to lose. The receipt still has to be written.
	controls, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision()})
	if err != nil {
		t.Fatalf("empty snapshot must still be legal: %v", err)
	}
	if len(controls.Receipt.SnapshotFiles) != 0 {
		t.Fatalf("unexpected files: %+v", controls.Receipt.SnapshotFiles)
	}
	// An existing, non-empty directory is a store declaring itself wrongly.
	if err := os.MkdirAll(filepath.Join(runtimeHomeFor(t, home), "state", "runtime.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision()}); err == nil {
		t.Fatal("accepted a directory as a store state path")
	}
}

// A forced transition must snapshot every declared store, not only the registry.
// The activation record is what makes a failed activation resumable, so losing
// it during the transition that activation depends on is exactly the failure the
// controls exist to prevent.
func TestPrepareForceControlsCapturesEveryDeclaredStore(t *testing.T) {
	withRepositoryCodecStamp(t)
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	activationDir := filepath.Join(runtimeHome, "state", "core-activation")
	if err := os.MkdirAll(activationDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"current.json":           `{"schema_version":1}`,
		"initialized":            "vrooli-core-activation-store-v1\n",
		"enrollment.json":        `{"schema_version":1}`,
		"enrollment-initialized": "vrooli-root-update-enrollment-v1\n",
	}
	for name, content := range want {
		if err := os.WriteFile(filepath.Join(activationDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	controls, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "activation store coverage"})
	if err != nil {
		t.Fatalf("prepare controls: %v", err)
	}
	captured := map[string]bool{}
	for _, file := range controls.Receipt.SnapshotFiles {
		captured[filepath.Base(file.Path)] = true
		if file.Store != "core-activation" {
			t.Fatalf("unexpected store %q in %s", file.Store, file.Path)
		}
	}
	for name := range want {
		if !captured[name] {
			t.Fatalf("activation store file %s was not snapshotted: %+v", name, controls.Receipt.SnapshotFiles)
		}
	}

	// The restore puts the pre-transition bytes back.
	if err := os.WriteFile(filepath.Join(activationDir, "current.json"), []byte(`{"schema_version":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(activationDir, "current.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want["current.json"] {
		t.Fatalf("restore left %q", data)
	}
}
