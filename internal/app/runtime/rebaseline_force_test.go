package runtimeapp

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/rootauthority"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/stateformat"
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
	if _, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision()}); err == nil {
		t.Fatal("accepted a forced transition with no recorded reason")
	}
	controls, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "empty snapshot coverage"})
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
	if _, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "directory as a state path"}); err == nil {
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

// Operator authority carries no grant ID. Refusing it would make an operator
// at a terminal unable to take a transition without first asking themselves
// for a standing grant.
func TestPrepareForceControlsAcceptsOperatorAuthority(t *testing.T) {
	withRepositoryCodecStamp(t)
	home := t.TempDir()
	decision := rootauthority.OperatorDecision(rootauthority.Target("host", "/home/owner/.vrooli"))
	if decision.GrantID != "" || decision.ApprovedBy != "operator terminal" {
		t.Fatalf("operator decision = %+v", decision)
	}
	controls, err := PrepareForceControls(context.Background(), home, ForceRequest{Authority: decision, Reason: "operator transition"})
	if err != nil {
		t.Fatalf("operator authority was refused: %v", err)
	}
	if controls.Receipt.Authority.ApprovedBy != "operator terminal" || controls.Receipt.Authority.GrantID != "" {
		t.Fatalf("receipt authority = %+v", controls.Receipt.Authority)
	}
	if _, err := PrepareForceControls(context.Background(), home, ForceRequest{Authority: rootauthority.Decision{Target: decision.Target}, Reason: "no approver"}); err == nil {
		t.Fatal("an authority decision with no approver was accepted")
	}
}

func TestPrepareForceControlsRecordsStoreCoverage(t *testing.T) {
	withRepositoryCodecStamp(t)
	controls, err := PrepareForceControls(context.Background(), t.TempDir(), ForceRequest{Authority: forceDecision(), Reason: "coverage reporting"})
	if err != nil {
		t.Fatalf("prepare controls: %v", err)
	}
	if controls.Receipt.StoresTotal == 0 {
		t.Fatal("receipt reports no stores; the compiled manifest must have stores")
	}
	if controls.Receipt.StoresWithState > controls.Receipt.StoresTotal {
		t.Fatalf("declared stores %d exceed total %d", controls.Receipt.StoresWithState, controls.Receipt.StoresTotal)
	}
	// Every captured file must name a store the receipt counted, so the
	// coverage number and the snapshot cannot describe different things.
	covered := map[string]bool{}
	for _, file := range controls.Receipt.SnapshotFiles {
		covered[file.Store] = true
	}
	if len(covered) > controls.Receipt.StoresWithState {
		t.Fatalf("snapshot covers %d stores but the receipt declares %d", len(covered), controls.Receipt.StoresWithState)
	}
}

func TestForceSnapshotExpandsWildcardsWithoutColliding(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	steps := map[string]string{
		"alpha/start-api.json": `{"step":"start-api"}`,
		"beta/start-api.json":  `{"step":"start-api"}`,
	}
	for relative, content := range steps {
		target := filepath.Join(runtimeHome, "processes", "scenarios", filepath.FromSlash(relative))
		if _, err := config.EnsureOwnedDir(filepath.Dir(target)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := stateformat.Store{
		Name:       "process-records",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: strings.Repeat("a", 64)},
		StatePaths: []string{"processes/scenarios/*/*.json"},
	}
	snapshotDir := t.TempDir()
	captured, unreadable, err := captureStoreState(ctx, runtimeHome, snapshotDir, []stateformat.Store{store})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if len(unreadable) != 0 {
		t.Fatalf("unreadable = %v, want none", unreadable)
	}
	if len(captured) != len(steps) {
		t.Fatalf("captured %d files, want %d: %+v", len(captured), len(steps), captured)
	}
	for _, file := range captured {
		if file.Capture != file.Path {
			t.Fatalf("capture %q does not preserve the layout of %q", file.Capture, file.Path)
		}
	}

	// Overwrite both files with different content, then restore the snapshot.
	for relative := range steps {
		target := filepath.Join(runtimeHome, "processes", "scenarios", filepath.FromSlash(relative))
		if err := os.WriteFile(target, []byte(`{"step":"newer"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	controls := ForceControls{Dir: snapshotDir, Receipt: ForceReceipt{SnapshotFiles: captured}}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for relative, content := range steps {
		target := filepath.Join(runtimeHome, "processes", "scenarios", filepath.FromSlash(relative))
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		if string(data) != content {
			t.Fatalf("%s restored as %q, want %q", relative, data, content)
		}
	}
}

func TestForceRestoreAcceptsLegacyFlatReceipts(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	target := filepath.Join(runtimeHome, "state", "core-activation", "current.json")
	if _, err := config.EnsureOwnedDir(filepath.Dir(target)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshotDir := t.TempDir()
	legacyDir := filepath.Join(snapshotDir, "core-activation")
	if _, err := config.EnsureOwnedDir(legacyDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "current.json"), []byte("older"), 0o600); err != nil {
		t.Fatal(err)
	}
	controls := ForceControls{Dir: snapshotDir, Receipt: ForceReceipt{SnapshotFiles: []ForceSnapshotFile{{
		Store: "core-activation", Path: "state/core-activation/current.json",
	}}}}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "older" {
		t.Fatalf("legacy receipt restored %q", data)
	}
}

func TestForceRestoreDropsEverySqliteSidecar(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	target := filepath.Join(runtimeHome, "state", "runtime.db")
	if _, err := config.EnsureOwnedDir(filepath.Dir(target)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("newer database"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.WriteFile(target+suffix, []byte("stale"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	snapshotDir := t.TempDir()
	storeDir := filepath.Join(snapshotDir, "runtime-registry", "state")
	if _, err := config.EnsureOwnedDir(storeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "runtime.db"), []byte("older database"), 0o600); err != nil {
		t.Fatal(err)
	}
	controls := ForceControls{Dir: snapshotDir, Receipt: ForceReceipt{SnapshotFiles: []ForceSnapshotFile{{
		Store: "runtime-registry", Path: "state/runtime.db", Capture: "state/runtime.db",
	}}}}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "older database" {
		t.Fatalf("restored %q", data)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(target + suffix); !os.IsNotExist(err) {
			t.Fatalf("%s survived the restore; it would be replayed over the restored database", suffix)
		}
	}
}

func TestForceReceiptNamesTheServingImage(t *testing.T) {
	withRepositoryCodecStamp(t)
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	bin, err := config.EnsureOwnedDir(filepath.Join(runtimeHome, "bin"))
	if err != nil {
		t.Fatal(err)
	}
	serving := filepath.Join(bin, "vrooli")
	if err := os.WriteFile(serving, []byte("serving image"), 0o755); err != nil {
		t.Fatal(err)
	}
	controls, err := PrepareForceControls(context.Background(), home, ForceRequest{Authority: forceDecision(), Reason: "receipt completeness"})
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := digestFile(serving)
	if err != nil {
		t.Fatal(err)
	}
	if controls.Receipt.ServingSHA != want {
		t.Fatalf("receipt serving sha = %q, want %q; a receipt that names only the candidate records half the change", controls.Receipt.ServingSHA, want)
	}
}

func TestForceControlsGiveOverlappingAttemptsSeparateSnapshots(t *testing.T) {
	withRepositoryCodecStamp(t)
	ctx := context.Background()
	home := t.TempDir()
	first, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "first attempt"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PrepareForceControls(ctx, home, ForceRequest{Authority: forceDecision(), Reason: "second attempt"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Dir == second.Dir || first.ReceiptAt == second.ReceiptAt {
		t.Fatalf("overlapping attempts shared a snapshot or receipt: %s / %s", first.Dir, second.Dir)
	}
	// Neither attempt may see the other's files.
	for _, dir := range []string{first.Dir, second.Dir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Name() == filepath.Base(second.Dir) || entry.Name() == filepath.Base(first.Dir) {
				t.Fatalf("%s contains another attempt's directory", dir)
			}
		}
	}
}

func TestSnapshotDigestIsStableForDuplicatePaths(t *testing.T) {
	file := ForceSnapshotFile{Path: "state/shared.json", Capture: "state/shared.json", Digest: strings.Repeat("a", 64)}
	first := []ForceSnapshotFile{{Store: "beta", Path: file.Path, Capture: file.Capture, Digest: file.Digest}, {Store: "alpha", Path: file.Path, Capture: file.Capture, Digest: file.Digest}}
	reversed := []ForceSnapshotFile{first[1], first[0]}
	one, err := digestSnapshot(first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := digestSnapshot(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if one == two {
		t.Fatal("input order must not change the digest")
	}
	// The capture order is what makes it deterministic; sorting by store name
	// must produce one answer for both inputs.
	sortSnapshotFiles(first)
	sortSnapshotFiles(reversed)
	one, _ = digestSnapshot(first)
	two, _ = digestSnapshot(reversed)
	if one != two {
		t.Fatalf("sorted digest is not deterministic: %s != %s", one, two)
	}
}

func TestCaptureStoreStateCapturesASharedPathOnce(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(home, "state", "shared.json")
	if err := os.WriteFile(shared, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(t.TempDir(), "snap")
	stores := []stateformat.Store{
		{Name: "alpha", StatePaths: []string{"state/shared.json"}},
		{Name: "beta", StatePaths: []string{"state/shared.json"}},
	}
	files, unreadable, err := captureStoreState(context.Background(), home, snapshot, stores)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadable) != 0 {
		t.Fatalf("unreadable = %v", unreadable)
	}
	// Capturing the same file twice would take two different moments of it and
	// produce a receipt for a state that never existed.
	if len(files) != 1 {
		t.Fatalf("captured %d copies of one shared path: %+v", len(files), files)
	}
	if files[0].Store != "alpha" {
		t.Fatalf("shared path attributed to %q, want the first declarer", files[0].Store)
	}
}

func TestForceRestorePutsBackTheCapturedMode(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	dir := filepath.Join(runtimeHome, "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "marker.json")
	if err := os.WriteFile(target, []byte("before"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(t.TempDir(), "snap")
	stores := []stateformat.Store{{Name: "alpha", StatePaths: []string{"state/marker.json"}}}
	files, _, err := captureStoreState(ctx, runtimeHome, snapshot, stores)
	if err != nil {
		t.Fatal(err)
	}
	if files[0].Mode != 0o640 {
		t.Fatalf("captured mode = %o, want 640", files[0].Mode)
	}
	if err := os.WriteFile(target, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	controls := ForceControls{Dir: snapshot, Receipt: ForceReceipt{SnapshotFiles: files}}
	if err := RestoreForceControls(ctx, home, controls); err != nil {
		t.Fatalf("restore: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o640 {
		t.Fatalf("restored mode = %o, want the captured 640; a restore must not change who can read the file", mode)
	}
}

func TestForceRestoreRefusesACorruptSnapshotCopy(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	dir := filepath.Join(runtimeHome, "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "marker.json")
	if err := os.WriteFile(target, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(t.TempDir(), "snap")
	stores := []stateformat.Store{{Name: "alpha", StatePaths: []string{"state/marker.json"}}}
	files, _, err := captureStoreState(ctx, runtimeHome, snapshot, stores)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate bit rot or a truncated copy in the retained snapshot.
	captured := filepath.Join(snapshot, files[0].Store, snapshotRelative(files[0]))
	if err := os.WriteFile(captured, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	controls := ForceControls{Dir: snapshot, Receipt: ForceReceipt{SnapshotFiles: files}}
	err = RestoreForceControls(ctx, home, controls)
	if err == nil {
		t.Fatal("restored bytes that do not match the receipt digest")
	}
	if !strings.Contains(err.Error(), "state/marker.json") || !strings.Contains(err.Error(), "does not match its recorded digest") {
		t.Fatalf("refusal does not name the corrupt file: %v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "current" {
		t.Fatalf("a refused restore still replaced the store with %q", data)
	}
}

func TestPruneRootRebaselineSnapshotsKeepsTheRecentCopies(t *testing.T) {
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	root := filepath.Join(runtimeHome, "root-rebaseline")
	snapshots := filepath.Join(root, "snapshots")
	receipts := filepath.Join(root, "receipts")
	for _, dir := range []string{snapshots, receipts} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	names := []string{"20261119T040000Z", "20261119T050000Z", "20261119T060000Z", "20261119T070000Z"}
	for _, name := range names {
		dir := filepath.Join(snapshots, name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(receipts, name+".json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A recent snapshot is never pruned: another attempt may still be writing.
	recent := filepath.Join(snapshots, "20261119T235900Z")
	if err := os.MkdirAll(recent, 0o700); err != nil {
		t.Fatal(err)
	}

	pruneRootRebaselineSnapshots(runtimeHome, root)

	entries, err := os.ReadDir(snapshots)
	if err != nil {
		t.Fatal(err)
	}
	var remaining []string
	for _, entry := range entries {
		remaining = append(remaining, entry.Name())
	}
	sort.Strings(remaining)
	want := []string{"20261119T060000Z", "20261119T070000Z", "20261119T235900Z"}
	if strings.Join(remaining, ",") != strings.Join(want, ",") {
		t.Fatalf("remaining snapshots = %v, want %v", remaining, want)
	}
	// Receipts are the durable record of each forced transition, even after the
	// snapshot it names is retired.
	if receiptsAfter, err := os.ReadDir(receipts); err != nil || len(receiptsAfter) != len(names) {
		t.Fatalf("receipts = %d (%v), want %d", len(receiptsAfter), err, len(names))
	}
}

func TestPruneRootRebaselineSnapshotsLeavesAnUnfinishedTransitionAlone(t *testing.T) {
	home := t.TempDir()
	runtimeHome := runtimeHomeFor(t, home)
	root := filepath.Join(runtimeHome, "root-rebaseline")
	snapshots := filepath.Join(root, "snapshots")
	progressDir := filepath.Join(runtimeHome, "state", "root-rebaseline")
	for _, dir := range []string{snapshots, progressDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-48 * time.Hour)
	for _, name := range []string{"20261117T040000Z", "20261117T050000Z", "20261117T060000Z"} {
		dir := filepath.Join(snapshots, name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
	}
	progress := `{"version":1,"candidate_sha256":"` + strings.Repeat("a", 64) + `","watchdog_sha256":"` + strings.Repeat("b", 64) + `","complete":false}`
	if err := os.WriteFile(filepath.Join(progressDir, "progress.json"), []byte(progress), 0o600); err != nil {
		t.Fatal(err)
	}

	pruneRootRebaselineSnapshots(runtimeHome, root)

	entries, err := os.ReadDir(snapshots)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("pruned snapshots while a transition was unfinished; %d remain", len(entries))
	}
}
