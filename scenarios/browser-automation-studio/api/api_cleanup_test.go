package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/recovery"
	"github.com/vrooli/browser-automation-studio/services/scheduler"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
)

type cleanupScheduleRepo struct{}

func (cleanupScheduleRepo) ListSchedules(context.Context, *uuid.UUID, bool, int, int) ([]*database.ScheduleIndex, error) {
	return nil, nil
}
func (cleanupScheduleRepo) GetSchedule(context.Context, uuid.UUID) (*database.ScheduleIndex, error) {
	return nil, errors.New("unexpected schedule lookup")
}
func (cleanupScheduleRepo) UpdateScheduleNextRun(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (cleanupScheduleRepo) UpdateScheduleLastRun(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type cleanupDriver struct{}

func (cleanupDriver) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return nil, nil
}
func (cleanupDriver) ForceCloseSession(context.Context, string, string, string) error {
	return nil
}

type cleanupRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *cleanupRecorder) add(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}
func (r *cleanupRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func TestRunAPICleanupComposesProductionOwnersInDependencyOrder(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	recorder := &cleanupRecorder{}

	// These are the production scheduler and workflow owner implementations;
	// only their external repository/executor boundaries are inert in this case.
	sched := scheduler.New(scheduler.SchedulerOptions{Repo: cleanupScheduleRepo{}, Log: log})
	if err := sched.Start(); err != nil {
		t.Fatal(err)
	}
	workflowOwner := workflow.NewWorkflowServiceWithDeps(nil, nil, log, workflow.WorkflowServiceOptions{})

	// SessionReconciler is the production recovery loop with a bounded driver
	// owner at the process boundary. An empty inventory needs no DB lookup.
	reconciler := recovery.NewSessionReconciler(cleanupDriver{}, nil, log, recovery.WithSessionReconcileInterval(time.Hour))
	reconcileCtx, cancelReconcile := context.WithCancel(context.Background())
	reconcileDone := make(chan struct{})
	go func() {
		reconciler.Run(reconcileCtx)
		recorder.add("reconciler-joined")
		close(reconcileDone)
	}()

	plan := apiCleanupPlan{
		schedulerStop: func(ctx context.Context) error {
			recorder.add("scheduler-stop")
			return sched.StopContext(ctx)
		},
		workflowShutdown: func(ctx context.Context) error {
			recorder.add("workflow-shutdown")
			return workflowOwner.Shutdown(ctx)
		},
		cancelReconciler: func() {
			recorder.add("reconciler-cancel")
			cancelReconcile()
		},
		reconcilerDone: reconcileDone,
		stopSidecar: func(context.Context) error {
			recorder.add("sidecar-stop")
			return nil
		},
		closeDatabase: func() error {
			recorder.add("database-close")
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runAPICleanup(ctx, plan); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	want := []string{"scheduler-stop", "workflow-shutdown", "reconciler-cancel", "reconciler-joined", "sidecar-stop", "database-close"}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanup order = %v, want %v", got, want)
	}
}

func TestHTTPServerCleanupPersistsRecoveryAndReconcilesExactLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetOutput(io.Discard)
	dbPath := filepath.Join(t.TempDir(), "http-cleanup.db")
	artifactRoot := t.TempDir()
	t.Setenv("BAS_SQLITE_PATH", dbPath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	open := func() (*database.DB, database.Repository) {
		db, err := database.NewConnection(log)
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		return db, database.NewRepository(db, log)
	}
	db, repo := open()
	workflowID, interruptedID := uuid.New(), uuid.New()
	if err := repo.CreateWorkflow(ctx, &database.WorkflowIndex{ID: workflowID, Name: "HTTP cleanup", FolderPath: t.TempDir(), Version: 1}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-time.Minute)
	if err := repo.CreateExecution(ctx, &database.ExecutionIndex{ID: interruptedID, WorkflowID: workflowID, Status: database.ExecutionStatusRunning, TriggerType: "api", StartedAt: started, UpdatedAt: started}); err != nil {
		t.Fatal(err)
	}
	// The same production composition must preserve dependencies when an owner
	// misses its budget; the process-boundary restart below is the recovery path.
	deadlineCtx, cancelDeadline := context.WithTimeout(ctx, 10*time.Millisecond)
	var deadlineTeardown []string
	deadlineErr := runAPICleanup(deadlineCtx, apiCleanupPlan{
		workflowShutdown: func(waitCtx context.Context) error { <-waitCtx.Done(); return waitCtx.Err() },
		stopSidecar:      func(context.Context) error { deadlineTeardown = append(deadlineTeardown, "sidecar"); return nil },
		closeDatabase:    func() error { deadlineTeardown = append(deadlineTeardown, "database"); return nil },
	})
	cancelDeadline()
	if !errors.Is(deadlineErr, context.DeadlineExceeded) || len(deadlineTeardown) != 0 {
		t.Fatalf("deadline cleanup err=%v teardown=%v; dependent owners must remain live", deadlineErr, deadlineTeardown)
	}
	driverOwner := &cleanupRecoveryDriver{sessions: []driver.ObservedSession{{ID: "interrupted-session", OwnerExecutionID: interruptedID.String(), LeaseID: "lease-old"}}}
	recorder := &cleanupRecorder{}
	sched := scheduler.New(scheduler.SchedulerOptions{Repo: cleanupScheduleRepo{}, Log: log})
	if err := sched.Start(); err != nil {
		t.Fatal(err)
	}
	workflowOwner := workflow.NewWorkflowServiceWithDeps(nil, nil, log, workflow.WorkflowServiceOptions{})
	reconcileCtx, cancelReconcile := context.WithCancel(ctx)
	reconcileDone := make(chan struct{})
	reconciler := recovery.NewSessionReconciler(driverOwner, repo, log, recovery.WithSessionReconcileInterval(time.Hour))
	go func() {
		reconciler.Run(reconcileCtx)
		recorder.add("reconciler-joined")
		close(reconcileDone)
	}()

	// Reserve a loopback port, then exercise api-core's actual HTTP listener,
	// signal handling, request drain and cleanup callback without BAS services.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	runDone := make(chan error, 1)
	cleanup := func(cleanupCtx context.Context) error {
		plan := apiCleanupPlan{
			schedulerStop: func(callCtx context.Context) error {
				recorder.add("scheduler-stop")
				return sched.StopContext(callCtx)
			},
			workflowShutdown: func(callCtx context.Context) error {
				recorder.add("workflow-shutdown")
				return workflowOwner.Shutdown(callCtx)
			},
			cancelReconciler: func() {
				recorder.add("reconciler-cancel")
				cancelReconcile()
			},
			reconcilerDone: reconcileDone,
			stopSidecar: func(context.Context) error {
				recorder.add("sidecar-stop")
				return nil
			},
			closeDatabase: func() error {
				recorder.add("database-close")
				return db.Close()
			},
		}
		return runAPICleanup(cleanupCtx, plan)
	}
	go func() {
		runDone <- server.Run(server.Config{
			BindAddress: "127.0.0.1", Port: strconv.Itoa(port),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ready" {
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}),
			ShutdownTimeout: time.Second, CleanupTimeout: time.Second,
			Cleanup: cleanup, Signals: []os.Signal{syscall.SIGUSR1},
			Logger: func(string, ...interface{}) {},
		})
	}()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, requestErr := client.Get("http://" + address + "/ready")
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusNoContent {
				t.Fatalf("HTTP readiness status = %d", response.StatusCode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("HTTP server did not become ready: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("send test shutdown signal: %v", err)
	}
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("server.Run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not finish graceful cleanup")
	}
	wantOrder := []string{"scheduler-stop", "workflow-shutdown", "reconciler-cancel", "reconciler-joined", "sidecar-stop", "database-close"}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("HTTP cleanup order = %v, want %v", got, wantOrder)
	}

	// The next process marks the durable running row terminal. Session cleanup
	// must then close only the exact old owner/lease, preserving a new lease.
	db, repo = open()
	defer db.Close()
	recovered, err := recovery.NewService(repo, log, artifactRoot).RecoverInterruptedExecutions(ctx)
	if err != nil || recovered.Recovered != 1 {
		t.Fatalf("restart recovery=%+v err=%v", recovered, err)
	}
	row, err := repo.GetExecution(ctx, interruptedID)
	if err != nil || row == nil || row.Status != database.ExecutionStatusFailed {
		t.Fatalf("recovered execution=%+v err=%v", row, err)
	}
	newID := uuid.New()
	if err := repo.CreateExecution(ctx, &database.ExecutionIndex{ID: newID, WorkflowID: workflowID, Status: database.ExecutionStatusRunning, TriggerType: "api", StartedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	driverOwner.mu.Lock()
	driverOwner.sessions = append(driverOwner.sessions, driver.ObservedSession{ID: "new-session", OwnerExecutionID: newID.String(), LeaseID: "lease-new"})
	driverOwner.mu.Unlock()
	closed, err := recovery.NewSessionReconciler(driverOwner, repo, log, recovery.WithSessionTerminalGrace(0)).ReconcileOnce(ctx)
	if err != nil || closed.Closed != 1 || closed.Skipped != 1 {
		t.Fatalf("restart lease reconciliation=%+v err=%v", closed, err)
	}
	if len(driverOwner.closed) != 1 || driverOwner.closed[0].OwnerExecutionID != interruptedID.String() || driverOwner.closed[0].LeaseID != "lease-old" || len(driverOwner.sessions) != 1 || driverOwner.sessions[0].OwnerExecutionID != newID.String() || driverOwner.sessions[0].LeaseID != "lease-new" {
		t.Fatalf("lease reconciliation crossed owner boundary: closed=%+v live=%+v", driverOwner.closed, driverOwner.sessions)
	}
}

func TestRunAPICleanupDeadlineKeepsSidecarAndDatabaseAlive(t *testing.T) {
	var mu sync.Mutex
	var events []string
	add := func(event string) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	cleanupErr := runAPICleanup(ctx, apiCleanupPlan{
		schedulerStop: func(context.Context) error { add("scheduler-stop"); return nil },
		workflowShutdown: func(ctx context.Context) error {
			add("workflow-shutdown")
			<-ctx.Done()
			return ctx.Err()
		},
		cancelReconciler: func() { add("reconciler-cancel") },
		reconcilerDone:   make(chan struct{}),
		stopSidecar:      func(context.Context) error { add("sidecar-stop"); return nil },
		closeDatabase:    func() error { add("database-close"); return nil },
	})
	if !errors.Is(cleanupErr, context.DeadlineExceeded) {
		t.Fatalf("cleanup error = %v, want deadline", cleanupErr)
	}
	mu.Lock()
	got := append([]string(nil), events...)
	mu.Unlock()
	want := []string{"scheduler-stop", "workflow-shutdown"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deadline cleanup order = %v, want %v; unsafe owners must remain live", got, want)
	}
}

type cleanupRecoveryDriver struct {
	mu       sync.Mutex
	sessions []driver.ObservedSession
	closed   []driver.ObservedSession
}

func (d *cleanupRecoveryDriver) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]driver.ObservedSession(nil), d.sessions...), nil
}
func (d *cleanupRecoveryDriver) ForceCloseSession(_ context.Context, id, ownerID, leaseID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, session := range d.sessions {
		if session.ID != id {
			continue
		}
		if session.OwnerExecutionID != ownerID || session.LeaseID != leaseID {
			return errors.New("session owner or lease changed")
		}
		d.closed = append(d.closed, session)
		d.sessions = append(d.sessions[:i], d.sessions[i+1:]...)
		return nil
	}
	return nil
}

func TestRunAPICleanupDeadlineRecoversSQLiteOwnerAndPreservesNewLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetOutput(io.Discard)
	dbPath := filepath.Join(t.TempDir(), "cleanup-restart.db")
	artifactRoot := t.TempDir()
	t.Setenv("BAS_SQLITE_PATH", dbPath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	open := func() (*database.DB, database.Repository) {
		db, err := database.NewConnection(log)
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		return db, database.NewRepository(db, log)
	}
	db, repo := open()
	workflowID, staleID := uuid.New(), uuid.New()
	if err := repo.CreateWorkflow(ctx, &database.WorkflowIndex{ID: workflowID, Name: "cleanup recovery", FolderPath: t.TempDir(), Version: 1}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-time.Minute)
	if err := repo.CreateExecution(ctx, &database.ExecutionIndex{ID: staleID, WorkflowID: workflowID, Status: database.ExecutionStatusRunning, TriggerType: "api", StartedAt: started, UpdatedAt: started}); err != nil {
		t.Fatal(err)
	}
	fileStore, err := storage.NewFileStorage(artifactRoot, log)
	if err != nil {
		t.Fatal(err)
	}
	writer := executionwriter.NewFileWriter(repo, fileStore, log, executionwriter.NewStaticRoot(artifactRoot))
	trace := []byte("committed before cleanup deadline")
	traceSource := filepath.Join(t.TempDir(), "trace.zip")
	if err := os.WriteFile(traceSource, trace, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writer.RecordExecutionArtifacts(ctx, contracts.ExecutionPlan{ExecutionID: staleID, WorkflowID: workflowID}, []executionwriter.ExternalArtifact{{ArtifactType: "trace", Path: traceSource, ContentType: "application/zip"}}); err != nil {
		t.Fatalf("commit pre-shutdown artifact: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Model the production callback's unresolved runner result: its deadline is
	// propagated, while sidecar/database teardown is withheld for process exit.
	deadline, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	var teardown []string
	cleanupErr := runAPICleanup(deadline, apiCleanupPlan{
		workflowShutdown: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
		stopSidecar:      func(context.Context) error { teardown = append(teardown, "sidecar"); return nil },
		closeDatabase:    func() error { teardown = append(teardown, "database"); return nil },
	})
	if !errors.Is(cleanupErr, context.DeadlineExceeded) || len(teardown) != 0 {
		t.Fatalf("cleanup deadline err=%v teardown=%v", cleanupErr, teardown)
	}

	// A fresh API persistence owner performs ordinary startup recovery against
	// the exact persisted row and committed artifact receipt.
	db, repo = open()
	defer db.Close()
	recoveryResult, err := recovery.NewService(repo, log, artifactRoot).RecoverInterruptedExecutions(ctx)
	if err != nil || recoveryResult.Recovered != 1 {
		t.Fatalf("startup recovery result=%+v err=%v", recoveryResult, err)
	}
	row, err := repo.GetExecution(ctx, staleID)
	if err != nil || row == nil || row.Status != database.ExecutionStatusFailed {
		t.Fatalf("stale row after recovery=%+v err=%v", row, err)
	}
	if _, err := os.Stat(filepath.Join(artifactRoot, staleID.String(), "evidence.proto.json")); err != nil {
		t.Fatalf("startup recovery lost the already committed evidence manifest: %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(artifactRoot, staleID.String(), "purge-inventory.v1.json"))
	if err != nil {
		t.Fatalf("read committed inventory: %v", err)
	}
	digest := sha256.Sum256(trace)
	if !strings.Contains(string(manifestBytes), hex.EncodeToString(digest[:])) {
		t.Fatalf("committed artifact checksum missing from receipt: %s", manifestBytes)
	}
	second, err := recovery.NewService(repo, log, artifactRoot).RecoverInterruptedExecutions(ctx)
	if err != nil || second.Recovered != 0 {
		t.Fatalf("second recovery not idempotent: %+v err=%v", second, err)
	}

	newID := uuid.New()
	if err := repo.CreateExecution(ctx, &database.ExecutionIndex{ID: newID, WorkflowID: workflowID, Status: database.ExecutionStatusRunning, TriggerType: "api", StartedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	driverOwner := &cleanupRecoveryDriver{sessions: []driver.ObservedSession{
		{ID: "stale-session", OwnerExecutionID: staleID.String(), LeaseID: "stale-lease"},
		{ID: "new-session", OwnerExecutionID: newID.String(), LeaseID: "new-lease"},
	}}
	reconciler := recovery.NewSessionReconciler(driverOwner, repo, log, recovery.WithSessionTerminalGrace(0))
	closed, err := reconciler.ReconcileOnce(ctx)
	if err != nil || closed.Closed != 1 || closed.Skipped != 1 {
		t.Fatalf("reconcile result=%+v err=%v", closed, err)
	}
	if len(driverOwner.closed) != 1 || driverOwner.closed[0].OwnerExecutionID != staleID.String() || driverOwner.closed[0].LeaseID != "stale-lease" || len(driverOwner.sessions) != 1 || driverOwner.sessions[0].OwnerExecutionID != newID.String() || driverOwner.sessions[0].LeaseID != "new-lease" {
		t.Fatalf("reconciliation crossed owner lease: closed=%+v live=%+v", driverOwner.closed, driverOwner.sessions)
	}
}
