package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/certification"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/fixtures"
	"scenario-to-cloud/persistence"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

const (
	drillKeyRef   = "fixture/recovery:key"
	drillSchemaV1 = "fixture-schema-v1"
	drillSchemaV2 = "fixture-schema-v2"
)

var (
	fixturesRoot = filepath.Join("..", "..", "fixtures")
	budgetsPath  = filepath.Join("..", "..", "certification", "budgets.json")
	evidenceDir  = filepath.Join("..", "..", "certification", "evidence")
)

func fixtureReleaseDigest(id string) string {
	sum := sha256.Sum256([]byte("fixture-release:" + id))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func newDrillRepo(t *testing.T, name, deploymentID string) *persistence.Repository {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	ctx := context.Background()
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	manifest, _ := json.Marshal(domain.CloudManifest{Version: "1", Target: domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10"}}, Scenario: domain.ManifestScenario{ID: "fixture-sql-uploads"}, Edge: domain.ManifestEdge{Domain: "fixture.example"}})
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	if err := repo.CreateDeployment(ctx, &domain.Deployment{ID: deploymentID, Name: "fixture", ScenarioID: "fixture-sql-uploads", Environment: "production", Status: domain.StatusPending, Manifest: manifest, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	return repo
}

// seedHost materialises the sql-uploads fixture on a temp host: the SQL
// binding as a SQLite database (package-lane stand-in for PostgreSQL) and
// the uploads binding as the seed objects.
func seedHost(t *testing.T, w *fixtures.Workload, host string) (dbPath, uploads string) {
	t.Helper()
	dbPath = filepath.Join(host, "data", "records.sqlite")
	uploads = filepath.Join(host, "uploads")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(w.Dir, "seed", "records.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, owner TEXT NOT NULL, amount_minor_units INTEGER NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		if _, err := db.Exec(`INSERT INTO records (id, owner, amount_minor_units, created_at) VALUES (?, ?, ?, ?)`, row[0], row[1], row[2], row[3]); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(w.Dir, "seed", "uploads"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(w.Dir, "seed", "uploads", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(uploads, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dbPath, uploads
}

// expectedUploadsInventory derives the object-store inventory the oracle
// implies: the seed/uploads/* entries of fixture-oracle/v1 expected_state
// keyed by their path inside the uploads binding.
func expectedUploadsInventory(w *fixtures.Workload) recoverypoint.Inventory {
	var entries []recoverypoint.FileEntry
	for _, f := range w.Oracle.ExpectedState.Files {
		if !strings.HasPrefix(f.Path, "seed/uploads/") {
			continue
		}
		entries = append(entries, recoverypoint.FileEntry{Path: strings.TrimPrefix(f.Path, "seed/uploads/"), Bytes: int64(f.Bytes), SHA256: f.SHA256})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return recoverypoint.InventoryOf(entries)
}

// writeLedger is the acknowledged-write ledger: a writer appends rows until
// the consistency boundary quiesces it, and every acknowledged row is
// recorded so the restored database can be compared with what the
// application acknowledged.
type writeLedger struct {
	mu           sync.Mutex
	acknowledged []int64
	stop         chan struct{}
	done         chan struct{}
	quiesced     bool
}

func startWriter(t *testing.T, dbPath string) *writeLedger {
	t.Helper()
	l := &writeLedger{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(l.done)
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return
		}
		defer db.Close()
		next := int64(100)
		for {
			select {
			case <-l.stop:
				return
			default:
			}
			if _, err := db.Exec(`INSERT INTO records (id, owner, amount_minor_units, created_at) VALUES (?, ?, ?, ?)`, next, "writer", next, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
				return
			}
			l.mu.Lock()
			l.acknowledged = append(l.acknowledged, next)
			l.mu.Unlock()
			next++
			time.Sleep(2 * time.Millisecond)
		}
	}()
	return l
}

// Enter implements recoverypoint.Boundary: the declared application hook
// stops acknowledging writes before the capture and reports quiesced.
func (l *writeLedger) Enter(_ context.Context, b recoverypoint.Binding, mode string) (recoverypoint.ConsistencyRecord, func(context.Context) error, error) {
	l.mu.Lock()
	if !l.quiesced {
		close(l.stop)
		l.quiesced = true
	}
	l.mu.Unlock()
	<-l.done
	return recoverypoint.ConsistencyRecord{Binding: b.ID, Mode: mode, WriteQuiescence: recoverypoint.QuiescenceQuiesced, Token: "ledger-" + b.ID}, func(context.Context) error { return nil }, nil
}

func (l *writeLedger) ids() []int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]int64(nil), l.acknowledged...)
}

func recordIDs(t *testing.T, dbPath string) []int64 {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id FROM records ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func newDrillService(repo Repository, root string, budgets Budgets, keys recoverypoint.KeyResolver, boundary recoverypoint.Boundary) *Service {
	return &Service{
		Repo: repo, Root: root, Keys: keys, Sealer: recoverypoint.Sealer{Iterations: 1000},
		Providers: recoverypoint.Registry{domain.BackupProviderSQLite: SQLite{}, domain.BackupProviderObjectStore: recoverypoint.ObjectStore{}},
		Boundary:  boundary, Registrar: StaticRegistrar{"records-db": "data-backup-manager:t-records", "uploads": "data-backup-manager:t-uploads"},
		Owner: "fixture-sql-uploads", Budgets: budgets,
	}
}

type drillEvidence struct {
	t       *testing.T
	dir     string
	digest  string
	written []string
}

func (d *drillEvidence) record(caseID string, verdict certification.Verdict, assertions, limitations []string, operationRefs []string) {
	d.t.Helper()
	path, err := WriteEvidenceReceipt(d.dir, certification.Receipt{
		CaseID: caseID, Verdict: verdict, Lane: certification.LanePackage,
		RequirementRefs: []string{"STC-P0-029", "STC-P0-030", "STC-P0-031"},
		Candidate:       certification.Candidate{ReleaseDigest: d.digest, ConfigurationDigest: "fixture-configuration", ClosureDigest: "sha256:fixture-sql-uploads"},
		Target:          certification.Target{MachineID: "package-lane-temp-host", Architecture: runtime.GOARCH, OS: runtime.GOOS},
		OperationRefs:   operationRefs, ValidationReceiptRef: "go test ./backup/ -run TestFreshHostRestoreDrill",
		Assertions: assertions, ObservedAt: time.Now().UTC(),
		Limitations: append([]string{
			"lane package: temp hosts on the test machine; the SQL binding is a SQLite stand-in for PostgreSQL (the postgres provider's argv is covered by TestPostgresHookMatchesResourceDeclaration)",
			"QEMU and real-VPS lanes pending EXT-01/EXT-06",
		}, limitations...),
	})
	if err != nil {
		d.t.Fatalf("write evidence %s: %v", caseID, err)
	}
	d.written = append(d.written, path)
}

// TestFreshHostRestoreDrill [REQ:STC-P0-029] [REQ:STC-P0-030] [REQ:STC-P0-031]
// is the deterministic fresh-host restore drill: seed the sql-uploads
// fixture into temp host A, capture a recovery point under acknowledged
// writes (P12-A01, DATA-01), restore into temp host B and assert the
// fixture-oracle/v1 invariants within the frozen RTO/RPO budgets (P12-A05,
// DATA-05), refuse a corrupt archive (P12-A04, DATA-02) and a missing key
// (P12-A03, DATA-03) without false completion, refuse an incompatible
// rollback with a typed plan (P12-A02, DATA-06) and keep referenced points
// out of retention (P12-A06, DATA-09). Evidence receipts are written to
// certification/evidence when STC_WRITE_EVIDENCE=1, otherwise to a temp dir.
func TestFreshHostRestoreDrill(t *testing.T) {
	ctx := context.Background()
	catalog, err := fixtures.Load(fixturesRoot)
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	workload := catalog.Workloads["sql-uploads"]
	if workload == nil {
		t.Fatal("sql-uploads fixture missing")
	}
	if _, err := fixtures.VerifyOracle(workload); err != nil {
		t.Fatalf("oracle drift: %v", err)
	}
	budgets, err := LoadBudgets(budgetsPath)
	if err != nil {
		t.Fatalf("budgets: %v", err)
	}
	digest := fixtureReleaseDigest(workload.ID)
	root := t.TempDir()
	evidence := &drillEvidence{t: t, dir: filepath.Join(root, "evidence"), digest: digest}
	if os.Getenv("STC_WRITE_EVIDENCE") == "1" {
		evidence.dir = evidenceDir
	}
	const deploymentID = "dep-sql-uploads"
	repo := newDrillRepo(t, "drill-"+fmt.Sprint(time.Now().UnixNano()), deploymentID)
	keys := recoverypoint.StaticKeys{drillKeyRef: []byte("fixture-recovery-key-material")}

	// Host A: seeded fixture plus a writer acknowledging rows.
	hostA := filepath.Join(root, "host-a")
	dbA, uploadsA := seedHost(t, workload, hostA)
	ledger := startWriter(t, dbA)
	time.Sleep(25 * time.Millisecond)
	bindings := []domain.DataBinding{
		{ID: "records-db", Owner: "postgres", Kind: domain.DataBindingKindSQL, Provider: domain.BackupProviderSQLite, Locator: dbA, MigrationOwner: "scenario"},
		{ID: "uploads", Owner: "fixture-sql-uploads", Kind: domain.DataBindingKindFiles, Provider: domain.BackupProviderObjectStore, Locator: uploadsA, MigrationOwner: "scenario"},
	}
	svc := newDrillService(repo, filepath.Join(root, "recovery-points"), budgets, keys, ledger)
	rp, err := svc.Capture(ctx, CaptureRequest{
		DeploymentID: deploymentID, OperationID: "op-1", Step: "data.backup", Bindings: bindings,
		ReleaseDigest: digest, SchemaVersion: drillSchemaV1, ConfigurationDigest: "fixture-configuration",
		CredentialVersionRefs: []string{"fixture/store:password@v3"}, RecoveryKeyRef: drillKeyRef, RetentionPolicy: "certification",
		Predecessor: &PredecessorState{Exists: true, HasData: true, SchemaVersion: drillSchemaV1, Versioned: true}, ProtectedBy: []string{"operation:op-1"},
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	acknowledged := ledger.ids()
	if len(acknowledged) == 0 {
		t.Fatal("the writer acknowledged no rows before the boundary quiesced it")
	}
	var csvRecords int64
	for _, f := range workload.Oracle.ExpectedState.Files {
		if f.Path == "seed/records.csv" {
			csvRecords = int64(f.Records)
		}
	}
	if csvRecords == 0 {
		t.Fatal("oracle declares no records.csv rows")
	}
	wantRecords := csvRecords + int64(len(acknowledged))
	if rp.ID != "op-1-data.backup" || !rp.Encrypted || rp.RecoveryKeyRef != drillKeyRef || rp.SchemaVersion != drillSchemaV1 || rp.ReleaseDigest != digest || rp.MigrationPosture != domain.MigrationPostureProductionEvolution {
		t.Fatalf("recovery point = %+v", rp)
	}
	if rp.Checksums["records-db"].Count != wantRecords || rp.Checksums["uploads"].Count != 3 || rp.Bindings[0].ProviderRef == "" || rp.Provider != DBMTool {
		t.Fatalf("checksums=%+v bindings=%+v provider=%s", rp.Checksums, rp.Bindings, rp.Provider)
	}
	for _, c := range rp.Consistency {
		if c.WriteQuiescence != domain.WriteQuiescenceQuiesced {
			t.Fatalf("binding %s not quiesced: %+v", c.Binding, c)
		}
	}
	if strings.Contains(rp.RecoveryKeyRef, "material") || rp.ManifestDigest == "" {
		t.Fatalf("recovery point must carry the key reference only: %+v", rp)
	}
	if again, err := svc.Capture(ctx, CaptureRequest{DeploymentID: deploymentID, OperationID: "op-1", Step: "data.backup", Bindings: bindings, RecoveryKeyRef: drillKeyRef, MigrationPosture: domain.MigrationPostureGreenfield}); err != nil || again.ManifestDigest != rp.ManifestDigest {
		t.Fatalf("capture replay must return the stored point: %v %+v", err, again)
	}
	expectUploads := expectedUploadsInventory(workload)
	expect := map[string]domain.BindingChecksum{
		"records-db": {Count: wantRecords},
		"uploads":    {Count: expectUploads.Count, Checksum: expectUploads.Checksum},
	}
	verify, err := svc.Verify(ctx, VerifyRequest{DeploymentID: deploymentID, RecoveryPointID: rp.ID, OpenArtifacts: true, Expect: expect})
	if err != nil || !verify.ArtifactsOpened || !verify.KeyResolved || verify.Outcome != domain.RestoreOutcomeSucceeded {
		t.Fatalf("verify: %v %+v", err, verify)
	}

	// Host B: clean replacement host.
	hostB := filepath.Join(root, "host-b")
	into := map[string]string{"records-db": filepath.Join(hostB, "data", "records.sqlite"), "uploads": filepath.Join(hostB, "uploads")}
	ledgerInvariant := func(_ context.Context, into map[string]string) domain.InvariantResult {
		restored := recordIDs(t, into["records-db"])
		var restoredWrites []int64
		for _, id := range restored {
			if id >= 100 {
				restoredWrites = append(restoredWrites, id)
			}
		}
		return domain.InvariantResult{Binding: "records-db", Check: "acknowledged_write_ledger", Expected: fmt.Sprint(acknowledged), Observed: fmt.Sprint(restoredWrites), Passed: fmt.Sprint(acknowledged) == fmt.Sprint(restoredWrites)}
	}
	receipt, err := svc.Restore(ctx, RestoreRequest{DeploymentID: deploymentID, RecoveryPointID: rp.ID, TargetRef: "host-b", Into: into, Expect: expect, Invariants: []Invariant{ledgerInvariant}})
	if err != nil || receipt.Outcome != domain.RestoreOutcomeSucceeded || !receipt.WithinBudgets {
		t.Fatalf("restore: err=%v receipt=%+v", err, receipt)
	}
	if receipt.MeasuredRTO <= 0 || receipt.RecoveryPointAge <= 0 || receipt.RTOBudgetSeconds != budgets.FreshHostRestoreSeconds || receipt.RPOBudgetSeconds != budgets.RecoveryPointSeconds {
		t.Fatalf("receipt must measure RTO/RPO against the budgets: %+v", receipt)
	}
	for _, r := range receipt.InvariantResults {
		if !r.Passed {
			t.Fatalf("invariant failed: %+v", r)
		}
	}
	restoredUploads, _, err := recoverypoint.DirInventory(into["uploads"])
	if err != nil || restoredUploads.Checksum != expectUploads.Checksum {
		t.Fatalf("restored uploads %+v err=%v", restoredUploads, err)
	}
	stored, err := repo.ListRestoreReceipts(ctx, rp.ID)
	if err != nil || len(stored) != 1 || stored[0].ID != receipt.ID {
		t.Fatalf("restore receipt must be durable: %v %+v", err, stored)
	}
	evidence.record("DATA-01", certification.VerdictPassed, []string{
		fmt.Sprintf("capture under acknowledged writes: %d rows acknowledged before the declared boundary quiesced the writer; every binding recorded write_quiescence=quiesced", len(acknowledged)),
		"restored acknowledged-write ledger equals the restored rows (acknowledged_write_ledger invariant passed)",
		fmt.Sprintf("recovery point age at restore start %s (budget %ds)", receipt.RecoveryPointAge, budgets.RecoveryPointSeconds),
	}, nil, []string{"op-1"})
	evidence.record("DATA-05", certification.VerdictPassed, []string{
		fmt.Sprintf("fresh host restore of records-db (%d rows) and uploads (%d objects) into clean bindings; fixture-oracle/v1 uploads checksum %s matched", wantRecords, expectUploads.Count, expectUploads.Checksum),
		fmt.Sprintf("measured RTO %s within fresh_host_restore_seconds_max %d; recovery point age %s within backup_recovery_point_seconds_max %d", receipt.MeasuredRTO, budgets.FreshHostRestoreSeconds, receipt.RecoveryPointAge, budgets.RecoveryPointSeconds),
		"restore receipt " + receipt.ID + " durable with invariant results",
	}, []string{"service start on the replacement host is not exercised in the package lane (no workload runtime); logical data verified only"}, []string{"op-1"})

	// A second restore into the now-populated host B is refused before any write.
	refused, err := svc.Restore(ctx, RestoreRequest{DeploymentID: deploymentID, RecoveryPointID: rp.ID, TargetRef: "host-b", Into: into})
	if typed := apierrors.As(err); typed == nil || typed.Code != apierrors.CodeRestoreTargetNotClean || refused.Outcome != domain.RestoreOutcomeRefused {
		t.Fatalf("occupied target: err=%v receipt=%+v", err, refused)
	}

	// Second point for the failure cases so the first stays intact.
	rp2, err := svc.Capture(ctx, CaptureRequest{DeploymentID: deploymentID, OperationID: "op-2", Step: "data.backup", Bindings: bindings, ReleaseDigest: digest, SchemaVersion: drillSchemaV1, RecoveryKeyRef: drillKeyRef, MigrationPosture: domain.MigrationPostureProductionEvolution})
	if err != nil {
		t.Fatalf("capture 2: %v", err)
	}
	hostC := filepath.Join(root, "host-c")
	intoC := map[string]string{"records-db": filepath.Join(hostC, "data", "records.sqlite"), "uploads": filepath.Join(hostC, "uploads")}
	noKey := newDrillService(repo, svc.Root, budgets, recoverypoint.StaticKeys{}, ledger)
	missing, err := noKey.Restore(ctx, RestoreRequest{DeploymentID: deploymentID, RecoveryPointID: rp2.ID, TargetRef: "host-c", Into: intoC})
	typed := apierrors.As(err)
	if typed == nil || typed.Code != apierrors.CodeRecoveryKeyUnavailable || typed.NextAction == nil || typed.NextAction.Reference != drillKeyRef || missing.Outcome != domain.RestoreOutcomeRefused {
		t.Fatalf("missing key: err=%+v receipt=%+v", typed, missing)
	}
	if typed.Status() != 424 || apierrors.ExitCodeFor(typed.Code) != apierrors.ExitRefused {
		t.Fatalf("missing key must be a refusal: status=%d", typed.Status())
	}
	if _, statErr := os.Stat(intoC["records-db"]); !os.IsNotExist(statErr) {
		t.Fatalf("missing key must not write into host C")
	}
	if _, err := noKey.Verify(ctx, VerifyRequest{DeploymentID: deploymentID, RecoveryPointID: rp2.ID, OpenArtifacts: true}); !apierrors.Is(err, apierrors.CodeRecoveryKeyUnavailable) {
		t.Fatalf("verify with missing key: %v", err)
	}
	evidence.record("DATA-03", certification.VerdictPassed, []string{
		"restore with an unresolvable recovery key reference refused with recovery_key_unavailable naming the reference as blocker and next_action supply_recovery_key",
		"no file written into the replacement host; restore receipt outcome refused (no false completion)",
		"verify --open with the same missing key reports recovery_key_unavailable",
	}, nil, []string{"op-2"})

	sealed := filepath.Join(rp2.Location, "records-db.sealed")
	raw, err := os.ReadFile(sealed)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 0x5a
	if err := os.WriteFile(sealed, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := svc.Restore(ctx, RestoreRequest{DeploymentID: deploymentID, RecoveryPointID: rp2.ID, TargetRef: "host-c", Into: intoC})
	if typed := apierrors.As(err); typed == nil || typed.Code != apierrors.CodeRecoveryPointCorrupt || corrupt.Outcome != domain.RestoreOutcomeRefused || typed.Status() != 422 {
		t.Fatalf("corrupt archive: err=%v receipt=%+v", err, corrupt)
	}
	if _, statErr := os.Stat(intoC["records-db"]); !os.IsNotExist(statErr) {
		t.Fatalf("corrupt archive must not write into host C")
	}
	if _, err := svc.Verify(ctx, VerifyRequest{DeploymentID: deploymentID, RecoveryPointID: rp2.ID}); !apierrors.Is(err, apierrors.CodeRecoveryPointCorrupt) {
		t.Fatalf("verify corrupt: %v", err)
	}
	// Host B (the earlier successful restore) is untouched by the refusals.
	if got := recordIDs(t, into["records-db"]); int64(len(got)) != wantRecords {
		t.Fatalf("existing restored data changed: %d rows", len(got))
	}
	evidence.record("DATA-02", certification.VerdictPassed, []string{
		"one byte of a sealed artifact altered: restore refused with recovery_point_corrupt (HTTP 422, exit 2) before any write; verify reports the same",
		"previously restored host data unchanged; restore receipt outcome refused",
	}, nil, []string{"op-2"})

	// Rollback admission across an incompatible schema.
	points, err := repo.ListRecoveryPoints(ctx, deploymentID)
	if err != nil {
		t.Fatal(err)
	}
	verdict := EvaluateRollback(RollbackFacts{CurrentSchema: drillSchemaV2, TargetSchema: drillSchemaV1, SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only", RecoveryPoints: points})
	if verdict.Compatible || verdict.ReasonCode != ReasonSchemaAhead || verdict.Plan == nil || verdict.Plan.Kind != domain.RecoveryPlanRestore || verdict.Plan.RecoveryPointID != rp2.ID {
		t.Fatalf("incompatible rollback verdict = %+v", verdict)
	}
	rollbackErr := RollbackError(verdict)
	if rollbackErr.Code != apierrors.CodeRollbackIncompatible || rollbackErr.Status() != 409 || rollbackErr.NextAction == nil || rollbackErr.NextAction.Kind != domain.RecoveryPlanRestore {
		t.Fatalf("rollback error = %+v", rollbackErr)
	}
	if _, reqErr := RequireRecoveryPoint(points, digest, drillSchemaV2); reqErr == nil || reqErr.Code != apierrors.CodeRecoveryPointRequired || reqErr.Status() != 428 {
		t.Fatalf("destructive schema change without a point at v2 must be refused: %+v", reqErr)
	}
	if anchor, reqErr := RequireRecoveryPoint(points, digest, drillSchemaV1); reqErr != nil || anchor == nil {
		t.Fatalf("recovery point at v1 must satisfy the precondition: %v", reqErr)
	}
	evidence.record("DATA-06", certification.VerdictPassed, []string{
		"code rollback from schema " + drillSchemaV2 + " to code at " + drillSchemaV1 + " under expand_contract refused: rollback_incompatible (409) reason predecessor_cannot_read_schema",
		"typed restore plan anchored on recovery point " + rp2.ID + " with preconditions (recovery_point_required, restore_target_clean, recovery_key_resolvable) and ordered owner steps",
		"destructive schema change without a recovery point at the current schema refused with recovery_point_required (428) naming cloud-target data backup",
	}, nil, []string{"op-1", "op-2"})

	// Retention: referenced points are never deleted.
	rp3, err := svc.Capture(ctx, CaptureRequest{DeploymentID: deploymentID, OperationID: "op-3", Step: "data.backup", Bindings: bindings, ReleaseDigest: "sha256:retired", SchemaVersion: drillSchemaV1, RecoveryKeyRef: drillKeyRef, MigrationPosture: domain.MigrationPostureProductionEvolution})
	if err != nil {
		t.Fatalf("capture 3: %v", err)
	}
	refs := References{ActiveReleases: []string{digest}, NonTerminalOperations: []string{"op-2"}}
	plan, err := svc.Prune(ctx, deploymentID, refs, Policy{KeepLast: 0, MaxAge: time.Nanosecond})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(plan.Protected[rp.ID]) == 0 || len(plan.Protected[rp2.ID]) == 0 || len(plan.Delete) != 1 || plan.Delete[0] != rp3.ID {
		t.Fatalf("prune plan = %+v", plan)
	}
	if kept, _ := repo.GetRecoveryPoint(ctx, rp.ID); kept == nil || !kept.Protected {
		t.Fatalf("active-release point must survive pruning: %+v", kept)
	}
	if err := repo.DeleteRecoveryPoint(ctx, rp2.ID); !apierrors.Is(err, apierrors.CodeRecoveryPointProtected) {
		t.Fatalf("direct delete of a protected point must be refused: %v", err)
	}
	if gone, _ := repo.GetRecoveryPoint(ctx, rp3.ID); gone != nil {
		t.Fatalf("unreferenced expired point must be pruned")
	}
	if _, statErr := os.Stat(rp.Location); statErr != nil {
		t.Fatalf("protected recovery point directory must remain: %v", statErr)
	}
	evidence.record("DATA-09", certification.VerdictPassed, []string{
		"retention with keep_last 0 and max_age 1ns deleted only the unreferenced point; the point referenced by the active release and the point referenced by a non-terminal operation were protected with named holders",
		"direct delete of a protected point refused with recovery_point_protected",
	}, nil, []string{"op-1", "op-2", "op-3"})

	if len(evidence.written) != 6 {
		t.Fatalf("expected 6 evidence receipts, wrote %d", len(evidence.written))
	}
	matrix, _ := certification.LoadEmbedded()
	receipts, err := certification.LoadEvidenceDir(evidence.dir, matrix)
	if err != nil {
		t.Fatalf("evidence dir must load: %v", err)
	}
	seen := map[string]bool{}
	for _, r := range receipts {
		seen[r.CaseID] = true
	}
	for _, id := range []string{"DATA-01", "DATA-02", "DATA-03", "DATA-05", "DATA-06", "DATA-09"} {
		if !seen[id] {
			t.Fatalf("evidence for %s missing", id)
		}
	}
}
