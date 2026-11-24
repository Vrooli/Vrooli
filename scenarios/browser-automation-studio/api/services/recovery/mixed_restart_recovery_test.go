package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/executionartifactlock"
	"github.com/vrooli/browser-automation-studio/storage"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	"google.golang.org/protobuf/encoding/protojson"
)

// mixedRestartDriver models the independently owned driver process. It keeps
// sessions across API/database client reconstruction and enforces the same
// owner+lease match required by ForceCloseSession.
type mixedRestartDriver struct {
	mu          sync.Mutex
	sessions    []driver.ObservedSession
	closed      []driver.ObservedSession
	beforeClose func()
}

func (d *mixedRestartDriver) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]driver.ObservedSession(nil), d.sessions...), nil
}

func (d *mixedRestartDriver) ForceCloseSession(_ context.Context, id, ownerID, leaseID string) error {
	if d.beforeClose != nil {
		callback := d.beforeClose
		d.beforeClose = nil
		callback()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, session := range d.sessions {
		if session.ID != id {
			continue
		}
		if session.OwnerExecutionID != ownerID || session.LeaseID != leaseID {
			return errors.New("session owner or lease changed before close")
		}
		d.closed = append(d.closed, session)
		d.sessions = append(d.sessions[:i], d.sessions[i+1:]...)
		return nil
	}
	return nil
}

func TestMixedRestartRecoveryPreservesArtifactsAndClosesOnlyInterruptedOwner(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetOutput(io.Discard)
	dbPath := filepath.Join(t.TempDir(), "bas-restart.db")
	artifactRoot := t.TempDir()
	// BAS's local object store shares its recordings root and writes under
	// artifacts/<execution-id>, which the writer's purge inventory recognizes.
	objectRoot := artifactRoot
	t.Setenv("BAS_SQLITE_PATH", dbPath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	openRepo := func() (*database.DB, database.Repository) {
		db, err := database.NewConnection(log)
		if err != nil {
			t.Fatalf("open BAS database: %v", err)
		}
		return db, database.NewRepository(db, log)
	}
	db, repo := openRepo()
	interruptedID, partialID := uuid.New(), uuid.New()
	workflowID := uuid.New()
	if err := repo.CreateWorkflow(ctx, &database.WorkflowIndex{ID: workflowID, Name: "restart fixture", FolderPath: t.TempDir(), Version: 1}); err != nil {
		t.Fatalf("persist restart fixture workflow: %v", err)
	}
	started := time.Now().UTC().Add(-time.Minute)
	for _, id := range []uuid.UUID{interruptedID, partialID} {
		if err := repo.CreateExecution(ctx, &database.ExecutionIndex{
			ID: id, WorkflowID: workflowID, Status: database.ExecutionStatusRunning,
			TriggerType: "api", StartedAt: started, UpdatedAt: started,
		}); err != nil {
			t.Fatalf("persist running execution %s: %v", id, err)
		}
	}

	fileStorage, err := storage.NewFileStorage(objectRoot, log)
	if err != nil {
		t.Fatalf("create durable artifact storage: %v", err)
	}
	writer := executionwriter.NewFileWriter(repo, fileStorage, log, executionwriter.NewStaticRoot(artifactRoot))
	plan := contracts.ExecutionPlan{ExecutionID: interruptedID, WorkflowID: workflowID}
	traceBytes := []byte("trace bytes committed before simulated process loss")
	traceSource := filepath.Join(t.TempDir(), "trace.zip")
	if err := os.WriteFile(traceSource, traceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writer.RecordExecutionArtifacts(ctx, plan, []executionwriter.ExternalArtifact{{ArtifactType: "trace", Path: traceSource, ContentType: "application/zip"}}); err != nil {
		t.Fatalf("persist trace and evidence manifest before loss: %v", err)
	}
	completedAt := time.Now().UTC()
	if _, err := writer.RecordStepOutcome(ctx, plan, contracts.StepOutcome{
		StepIndex: 1, NodeID: "restart-fixture", StepType: "navigate", Success: true,
		StartedAt: started, CompletedAt: &completedAt, FinalURL: "https://fixture.invalid/",
	}); err != nil {
		t.Fatalf("commit result file and database result path: %v", err)
	}

	interruptedDir := filepath.Join(artifactRoot, interruptedID.String())
	committedEvidencePath := filepath.Join(interruptedDir, "evidence.proto.json")
	committedEvidence, err := os.ReadFile(committedEvidencePath)
	if err != nil {
		t.Fatalf("read committed evidence manifest before process loss: %v", err)
	}
	// A writer-authored purge receipt protects any file it already references,
	// even when its name matches the atomic temp prefix.
	referencedTemp := filepath.Join(interruptedDir, ".tmp-inventoried-evidence")
	referencedBytes := []byte("published receipt reference")
	if err := os.WriteFile(referencedTemp, referencedBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(interruptedDir, "purge-inventory.v1.json")
	inventoryRaw, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("read writer inventory: %v", err)
	}
	var inventoryDocument map[string]any
	if err := json.Unmarshal(inventoryRaw, &inventoryDocument); err != nil {
		t.Fatalf("decode writer inventory: %v", err)
	}
	var currentFiles []struct {
		Path   string `json:"path"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	}
	filesRaw, err := json.Marshal(inventoryDocument["files"])
	if err != nil || json.Unmarshal(filesRaw, &currentFiles) != nil {
		t.Fatalf("decode inventory file bindings: %v", err)
	}
	referencedDigest := sha256.Sum256(referencedBytes)
	currentFiles = append(currentFiles, struct {
		Path   string `json:"path"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	}{Path: filepath.Base(referencedTemp), Bytes: int64(len(referencedBytes)), SHA256: hex.EncodeToString(referencedDigest[:])})
	filesJSON, err := json.Marshal(currentFiles)
	if err != nil {
		t.Fatal(err)
	}
	inventoryDocument["files"] = json.RawMessage(filesJSON)
	updatedInventory, err := json.Marshal(inventoryDocument)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventoryPath, updatedInventory, 0o600); err != nil {
		t.Fatal(err)
	}
	// An atomic-write temporary file is the state left if the process dies
	// before rename. It is never the published manifest and must not replace it.
	partialTemp := filepath.Join(interruptedDir, ".tmp-interrupted-evidence-write")
	if err := os.WriteFile(partialTemp, []byte(`{"evidence":{"artifacts":[`), 0o600); err != nil {
		t.Fatal(err)
	}
	partialDir := filepath.Join(artifactRoot, partialID.String())
	if err := os.MkdirAll(partialDir, 0o700); err != nil {
		t.Fatal(err)
	}
	partialOnlyTemp := filepath.Join(partialDir, ".tmp-partial-first-manifest")
	if err := os.WriteFile(partialOnlyTemp, []byte(`{"schema_version":"bas-replay-package/v1"`), 0o600); err != nil {
		t.Fatal(err)
	}
	uncertainID := uuid.New()
	uncertainDir := filepath.Join(artifactRoot, uncertainID.String())
	if err := os.MkdirAll(uncertainDir, 0o700); err != nil {
		t.Fatal(err)
	}
	uncertainTemp := filepath.Join(uncertainDir, ".tmp-owner-unknown")
	if err := os.WriteFile(uncertainTemp, []byte("owner row missing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close previous API database handle: %v", err)
	}
	// Drop the previous writer; its in-memory timeline is intentionally not
	// available to the next API process. The driver owner remains alive.
	writer = nil

	db, repo = openRepo()
	t.Cleanup(func() { _ = db.Close() })
	startup := NewService(repo, log, artifactRoot)
	firstRecovery, err := startup.RecoverInterruptedExecutions(ctx)
	if err != nil || firstRecovery.TotalActive != 2 || firstRecovery.Recovered != 2 || firstRecovery.ArtifactTempsRemoved != 2 {
		t.Fatalf("startup recovery = %+v, err=%v", firstRecovery, err)
	}
	secondRecovery, err := startup.RecoverInterruptedExecutions(ctx)
	if err != nil || secondRecovery.TotalActive != 0 || secondRecovery.Recovered != 0 || secondRecovery.ArtifactTempsRemoved != 0 {
		t.Fatalf("second startup recovery was not idempotent: %+v, err=%v", secondRecovery, err)
	}

	interruptedRow, err := repo.GetExecution(ctx, interruptedID)
	if err != nil || interruptedRow == nil || interruptedRow.Status != database.ExecutionStatusFailed || interruptedRow.ResultPath == "" {
		t.Fatalf("recovered execution row = %+v, err=%v", interruptedRow, err)
	}
	resultBytes, err := os.ReadFile(interruptedRow.ResultPath)
	if err != nil {
		t.Fatalf("recovered result path is not durable: %v", err)
	}
	var resultManifest map[string]any
	if err := json.Unmarshal(resultBytes, &resultManifest); err != nil || len(resultManifest) == 0 {
		t.Fatalf("committed result manifest is partial or empty: err=%v bytes=%q", err, resultBytes)
	}
	partialRow, err := repo.GetExecution(ctx, partialID)
	if err != nil || partialRow == nil || partialRow.Status != database.ExecutionStatusFailed || partialRow.ResultPath != "" {
		t.Fatalf("partial-write execution row = %+v, err=%v", partialRow, err)
	}
	afterEvidence, err := os.ReadFile(committedEvidencePath)
	if err != nil || string(afterEvidence) != string(committedEvidence) {
		t.Fatalf("recovery changed committed manifest: read err=%v equal=%v", err, string(afterEvidence) == string(committedEvidence))
	}
	var pack basevidence.ReplayPackage
	if err := (protojson.UnmarshalOptions{}).Unmarshal(afterEvidence, &pack); err != nil {
		t.Fatalf("committed evidence manifest no longer decodes after recovery: %v", err)
	}
	var inventory struct {
		ArtifactBindings []struct {
			ObjectKey string `json:"object_key"`
			Bytes     int64  `json:"bytes"`
			SHA256    string `json:"sha256"`
		} `json:"artifact_bindings"`
	}
	inventoryBytes, err := os.ReadFile(filepath.Join(interruptedDir, "purge-inventory.v1.json"))
	if err != nil || json.Unmarshal(inventoryBytes, &inventory) != nil {
		t.Fatalf("read durable artifact receipts after recovery: %v", err)
	}
	var traceBinding *struct {
		ObjectKey string `json:"object_key"`
		Bytes     int64  `json:"bytes"`
		SHA256    string `json:"sha256"`
	}
	for i := range inventory.ArtifactBindings {
		if inventory.ArtifactBindings[i].ObjectKey != "" && inventory.ArtifactBindings[i].Bytes == int64(len(traceBytes)) {
			traceBinding = &inventory.ArtifactBindings[i]
			break
		}
	}
	if traceBinding == nil || traceBinding.ObjectKey == "" {
		t.Fatalf("trace object receipt missing from durable inventory: %+v", inventory.ArtifactBindings)
	}
	traceReader, _, err := fileStorage.GetArtifact(ctx, traceBinding.ObjectKey)
	if err != nil {
		t.Fatalf("read committed trace object after API restart: %v", err)
	}
	storedTrace, readErr := io.ReadAll(traceReader)
	_ = traceReader.Close()
	if readErr != nil {
		t.Fatalf("read trace bytes: %v", readErr)
	}
	digest := sha256.Sum256(storedTrace)
	if string(storedTrace) != string(traceBytes) || traceBinding.Bytes != int64(len(storedTrace)) || traceBinding.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("trace receipt integrity mismatch: bytes=%d receipt=%+v", len(storedTrace), traceBinding)
	}
	if len(pack.GetEvidence().GetArtifacts()) == 0 || pack.GetEvidence().GetArtifacts()[0].GetSha256() != traceBinding.SHA256 {
		t.Fatalf("evidence manifest and object receipt disagree: manifest=%+v receipt=%+v", pack.GetEvidence().GetArtifacts(), traceBinding)
	}
	if _, err := os.Stat(filepath.Join(partialDir, "evidence.proto.json")); !os.IsNotExist(err) {
		t.Fatalf("partial temporary manifest was published during recovery: err=%v", err)
	}
	for _, tempPath := range []string{partialTemp, partialOnlyTemp} {
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Fatalf("proven-abandoned atomic temp survived recovery: %s err=%v", tempPath, err)
		}
	}
	if _, err := os.Stat(uncertainTemp); err != nil {
		t.Fatalf("temp without a persisted execution owner was not preserved: %v", err)
	}
	if _, err := os.Stat(referencedTemp); err != nil {
		t.Fatalf("inventory-referenced temp was incorrectly removed: %v", err)
	}

	// A run admitted by the new API process must remain live when old-session
	// cleanup reconciles the interrupted execution's exact owner and lease.
	newRunID := uuid.New()
	if err := repo.CreateExecution(ctx, &database.ExecutionIndex{
		ID: newRunID, WorkflowID: workflowID, Status: database.ExecutionStatusRunning,
		TriggerType: "api", StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create new-process execution: %v", err)
	}
	newRunDir := filepath.Join(artifactRoot, newRunID.String())
	if err := os.MkdirAll(newRunDir, 0o700); err != nil {
		t.Fatal(err)
	}
	newRunTemp := filepath.Join(newRunDir, ".tmp-active-new-run")
	if err := os.WriteFile(newRunTemp, []byte("active writer staging"), 0o600); err != nil {
		t.Fatal(err)
	}
	driverOwner := &mixedRestartDriver{sessions: []driver.ObservedSession{
		{ID: "old-session", OwnerExecutionID: interruptedID.String(), LeaseID: "old-lease"},
		{ID: "new-session", OwnerExecutionID: newRunID.String(), LeaseID: "new-lease"},
	}}
	reconciler := NewSessionReconciler(driverOwner, repo, log, WithSessionTerminalGrace(0))
	firstClose, err := reconciler.ReconcileOnce(ctx)
	if err != nil || firstClose.Closed != 1 || firstClose.Skipped != 1 {
		t.Fatalf("mixed owner reconciliation = %+v, err=%v", firstClose, err)
	}
	if len(driverOwner.closed) != 1 || driverOwner.closed[0].ID != "old-session" || driverOwner.closed[0].OwnerExecutionID != interruptedID.String() || driverOwner.closed[0].LeaseID != "old-lease" {
		t.Fatalf("reconciler closed the wrong driver owner lease: %+v", driverOwner.closed)
	}
	if len(driverOwner.sessions) != 1 || driverOwner.sessions[0].ID != "new-session" || driverOwner.sessions[0].OwnerExecutionID != newRunID.String() || driverOwner.sessions[0].LeaseID != "new-lease" {
		t.Fatalf("reconciler crossed into the new run: %+v", driverOwner.sessions)
	}
	if _, err := os.Stat(newRunTemp); err != nil {
		t.Fatalf("cleanup removed a new run's active temp: %v", err)
	}
	secondClose, err := reconciler.ReconcileOnce(ctx)
	if err != nil || secondClose.Closed != 0 || secondClose.Skipped != 1 || len(driverOwner.closed) != 1 {
		t.Fatalf("second reconciliation was not idempotent: result=%+v closed=%+v err=%v", secondClose, driverOwner.closed, err)
	}
}

func TestSessionReconcilerCloseRaceCannotCloseReassignedRun(t *testing.T) {
	oldRunID, newRunID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	repo := &fakeExecutionLookup{executions: map[uuid.UUID]*database.ExecutionIndex{
		oldRunID: {ID: oldRunID, Status: database.ExecutionStatusFailed, UpdatedAt: now.Add(-time.Minute)},
		newRunID: {ID: newRunID, Status: database.ExecutionStatusRunning, UpdatedAt: now},
	}}
	driverOwner := &mixedRestartDriver{sessions: []driver.ObservedSession{{
		ID: "reused-session", OwnerExecutionID: oldRunID.String(), LeaseID: "old-lease",
	}}}
	driverOwner.beforeClose = func() {
		driverOwner.mu.Lock()
		defer driverOwner.mu.Unlock()
		driverOwner.sessions[0] = driver.ObservedSession{ID: "reused-session", OwnerExecutionID: newRunID.String(), LeaseID: "new-lease"}
	}
	reconciler := newSessionReconciler(driverOwner, repo, nil, WithSessionTerminalGrace(0))
	reconciler.now = func() time.Time { return now }
	result, err := reconciler.ReconcileOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Closed != 0 || len(driverOwner.closed) != 0 {
		t.Fatalf("stale close request removed a reassigned session: result=%+v closed=%+v", result, driverOwner.closed)
	}
	if len(driverOwner.sessions) != 1 || driverOwner.sessions[0].OwnerExecutionID != newRunID.String() || driverOwner.sessions[0].LeaseID != "new-lease" {
		t.Fatalf("reassigned run owner was lost: %+v", driverOwner.sessions)
	}
	result, err = reconciler.ReconcileOnce(context.Background())
	if err != nil || result.Closed != 0 || result.Skipped != 1 {
		t.Fatalf("reconciler did not leave the active reassigned run alone: result=%+v err=%v", result, err)
	}
}

type artifactCleanupRepo struct {
	mu        sync.Mutex
	execution *database.ExecutionIndex
}

func (r *artifactCleanupRepo) ListExecutions(_ context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.execution == nil || r.execution.Status != query.Status {
		return nil, 0, nil
	}
	copy := *r.execution
	return []*database.ExecutionIndex{&copy}, 1, nil
}

func (r *artifactCleanupRepo) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.execution == nil || r.execution.ID != id {
		return nil, nil
	}
	copy := *r.execution
	return &copy, nil
}

func (r *artifactCleanupRepo) UpdateExecutionStatus(_ context.Context, id uuid.UUID, status string, message *string, completed *time.Time, updated time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.execution == nil || r.execution.ID != id {
		return nil
	}
	r.execution.Status = status
	if message != nil {
		r.execution.ErrorMessage = *message
	}
	r.execution.CompletedAt = completed
	r.execution.UpdatedAt = updated
	return nil
}

func TestInterruptedArtifactCleanupWaitsForWriterLock(t *testing.T) {
	ctx := context.Background()
	executionID := uuid.New()
	root := t.TempDir()
	runDir := filepath.Join(root, executionID.String())
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	tempPath := filepath.Join(runDir, ".tmp-active-result")
	if err := os.WriteFile(tempPath, []byte("writer still owns this"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := &artifactCleanupRepo{execution: &database.ExecutionIndex{
		ID: executionID, Status: database.ExecutionStatusRunning, UpdatedAt: time.Now().UTC(),
	}}
	releaseWriterLock, err := executionartifactlock.Acquire(ctx, root, executionID)
	if err != nil {
		t.Fatal(err)
	}
	lockHeld := true
	t.Cleanup(func() {
		if lockHeld {
			releaseWriterLock()
		}
	})
	service := NewService(repo, logrus.New(), root)
	type recoveryOutput struct {
		result *RecoveryResult
		err    error
	}
	done := make(chan recoveryOutput, 1)
	go func() {
		result, err := service.RecoverInterruptedExecutions(ctx)
		done <- recoveryOutput{result: result, err: err}
	}()
	select {
	case out := <-done:
		t.Fatalf("recovery passed the active writer lock: result=%+v err=%v", out.result, out.err)
	case <-time.After(60 * time.Millisecond):
	}
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("active writer temp was removed while its lock was held: %v", err)
	}
	releaseWriterLock()
	lockHeld = false
	select {
	case out := <-done:
		if out.err != nil || out.result == nil || out.result.ArtifactTempsRemoved != 1 || repo.execution.Status != database.ExecutionStatusFailed {
			t.Fatalf("recovery did not clean after owner lock release: result=%+v row=%+v err=%v", out.result, repo.execution, out.err)
		}
	case <-time.After(time.Second):
		t.Fatal("recovery did not complete after writer lock release")
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("abandoned temp remains after lock-protected recovery: %v", err)
	}
}

type cleanupRaceRepo struct {
	mu        sync.Mutex
	rows      map[uuid.UUID]*database.ExecutionIndex
	afterScan func()
}

func (r *cleanupRaceRepo) ListExecutions(_ context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	r.mu.Lock()
	callback := r.afterScan
	if query.Status == database.ExecutionStatusFailed {
		r.afterScan = nil
	} else {
		callback = nil
	}
	r.mu.Unlock()
	if callback != nil {
		callback()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*database.ExecutionIndex
	for _, row := range r.rows {
		if row.Status != query.Status {
			continue
		}
		copy := *row
		out = append(out, &copy)
	}
	return out, len(out), nil
}

func (r *cleanupRaceRepo) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[id]
	if row == nil {
		return nil, nil
	}
	copy := *row
	return &copy, nil
}

func (r *cleanupRaceRepo) UpdateExecutionStatus(_ context.Context, id uuid.UUID, status string, message *string, completed *time.Time, updated time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[id]
	if row == nil {
		return nil
	}
	row.Status = status
	if message != nil {
		row.ErrorMessage = *message
	}
	row.CompletedAt = completed
	row.UpdatedAt = updated
	return nil
}

func TestInterruptedArtifactCleanupCannotCaptureNewRunDirectory(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	oldID, newID := uuid.New(), uuid.New()
	oldDir := filepath.Join(root, oldID.String())
	newDir := filepath.Join(root, newID.String())
	for _, dir := range []string{oldDir, newDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	oldTemp := filepath.Join(oldDir, ".tmp-old-partial")
	newTemp := filepath.Join(newDir, ".tmp-new-active")
	if err := os.WriteFile(oldTemp, []byte("old partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := &cleanupRaceRepo{rows: map[uuid.UUID]*database.ExecutionIndex{
		oldID: {ID: oldID, Status: database.ExecutionStatusRunning},
	}}
	repo.afterScan = func() {
		// A new run is admitted after the startup active-row snapshot but before
		// cleanup enumerates terminal recovery markers. Its UUID directory must
		// remain outside the old run's cleanup scope.
		repo.mu.Lock()
		repo.rows[newID] = &database.ExecutionIndex{ID: newID, Status: database.ExecutionStatusRunning}
		repo.mu.Unlock()
		if err := os.WriteFile(newTemp, []byte("new active write"), 0o600); err != nil {
			t.Errorf("create new run temp: %v", err)
		}
	}
	result, err := NewService(repo, logrus.New(), root).RecoverInterruptedExecutions(ctx)
	if err != nil || result.Recovered != 1 || result.ArtifactTempsRemoved != 1 {
		t.Fatalf("recovery = %+v err=%v", result, err)
	}
	if _, err := os.Stat(oldTemp); !os.IsNotExist(err) {
		t.Fatalf("old abandoned temp not removed: %v", err)
	}
	if _, err := os.Stat(newTemp); err != nil {
		t.Fatalf("new run temp was captured by old-run cleanup: %v", err)
	}
	if row, _ := repo.GetExecution(ctx, newID); row == nil || row.Status != database.ExecutionStatusRunning {
		t.Fatalf("new run owner changed during old-run cleanup: %+v", row)
	}
}

func TestInterruptedArtifactCleanupFailsClosedOnSymlinkAndRetries(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	executionID := uuid.New()
	runDir := filepath.Join(root, executionID.String())
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "preserve-me")
	if err := os.WriteFile(target, []byte("external target"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(runDir, ".tmp-symlink")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	marker := interruptedExecutionErrorPrefix + time.Now().UTC().Format(time.RFC3339)
	repo := &artifactCleanupRepo{execution: &database.ExecutionIndex{
		ID: executionID, Status: database.ExecutionStatusFailed, ErrorMessage: marker,
	}}
	service := NewService(repo, logrus.New(), root)
	first, err := service.RecoverInterruptedExecutions(ctx)
	if err == nil || first.ArtifactCleanupFailed != 1 {
		t.Fatalf("symlink cleanup should fail closed and report: result=%+v err=%v", first, err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("cleanup removed the uncertain symlink: %v", err)
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "external target" {
		t.Fatalf("cleanup touched symlink target: contents=%q err=%v", contents, err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(runDir, ".tmp-retry")
	if err := os.WriteFile(partial, []byte("abandoned partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := service.RecoverInterruptedExecutions(ctx)
	if err != nil || second.ArtifactTempsRemoved != 1 || second.ArtifactCleanupFailed != 0 {
		t.Fatalf("retry after cleanup fault = %+v err=%v", second, err)
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatalf("retry did not remove proven-abandoned temp: %v", err)
	}
}
