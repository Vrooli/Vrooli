package maintenance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func openGate(t *testing.T, path string) (*Gate, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(t.Context(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	gate := NewGate(NewRepository(db))
	return gate, db
}

func TestConcurrentAdmissionAndMaintenanceFence(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	first, err := gate.Admit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var admitted atomic.Int64
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, err := gate.Admit(t.Context())
			if err == nil {
				admitted.Add(1)
				release()
			} else if !errors.Is(err, ErrClosed) {
				t.Error(err)
			}
		}()
	}
	close(start)
	state, err := gate.Enter(t.Context(), "operator", "planned rollout")
	if err != nil || !state.Closed {
		t.Fatalf("fence=%+v err=%v", state, err)
	}
	wg.Wait()
	for i := 0; i < 32; i++ {
		if release, err := gate.Admit(t.Context()); !errors.Is(err, ErrClosed) {
			if release != nil {
				release()
			}
			t.Fatalf("post-fence admission: %v", err)
		}
	}
	standing, err := gate.Status(t.Context())
	if err != nil || standing.Admitting != 1 {
		t.Fatalf("admitted work lost: %+v %v", standing, err)
	}
	first()
	first() // duplicate release cannot erase another admission's accounting.
	standing, err = gate.Status(t.Context())
	if err != nil || standing.Admitting != 0 {
		t.Fatalf("admission completion: %+v %v", standing, err)
	}
}

func TestDrainTimeoutPreservesAdmittedWorkAndFence(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	release, err := gate.Admit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := gate.Enter(t.Context(), "operator", "maintenance"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(t.Context(), time.Unix(0, 0))
	defer cancel()
	result, err := gate.Wait(ctx, func(context.Context) (int, error) { return 1, nil }, time.Hour)
	if !errors.Is(err, context.DeadlineExceeded) || result.Drained {
		t.Fatalf("timeout=%+v err=%v", result, err)
	}
	standing, err := gate.Status(t.Context())
	if err != nil || !standing.Closed || standing.Admitting != 1 {
		t.Fatalf("timeout altered admitted work: %+v %v", standing, err)
	}
	release()
	result, err = gate.Wait(t.Context(), func(context.Context) (int, error) { return 0, nil }, time.Hour)
	if err != nil || !result.Drained {
		t.Fatalf("finished work did not drain: %+v %v", result, err)
	}
}

func TestCrashRestartRetainsFenceAndRejectsStaleResume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gate.db")
	gate, db := openGate(t, path)
	state, err := gate.Enter(t.Context(), "operator", "rollout")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, _ := openGate(t, path)
	if _, err := restarted.Admit(t.Context()); !errors.Is(err, ErrClosed) {
		t.Fatalf("restart admitted work: %v", err)
	}
	standing, err := restarted.Status(t.Context())
	if err != nil || standing.Revision != state.Revision || standing.Reason != "rollout" {
		t.Fatalf("lost durable intent: %+v %v", standing, err)
	}
	if _, err := restarted.Resume(t.Context(), "operator", state.Revision-1); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale resume: %v", err)
	}
	if _, err := restarted.Resume(t.Context(), "operator", state.Revision); err != nil {
		t.Fatal(err)
	}
	release, err := restarted.Admit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestAbruptProcessExitRetainsMaintenanceFence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gate.db")
	command := exec.Command(os.Args[0], "-test.run=^TestMaintenanceCrashWriter$")
	command.Env = append(os.Environ(), "AM_MAINTENANCE_CRASH_FIXTURE="+path)
	err := command.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("crash fixture exit=%v", err)
	}
	gate, _ := openGate(t, path)
	if _, err := gate.Admit(t.Context()); !errors.Is(err, ErrClosed) {
		t.Fatalf("process crash reopened admission: %v", err)
	}
	result, err := gate.Wait(t.Context(), func(context.Context) (int, error) { return 0, errors.New("recovery not finished") }, time.Hour)
	if err == nil || result.Drained {
		t.Fatalf("restart guessed admitted work was gone: %+v %v", result, err)
	}
}

func TestMaintenanceCrashWriter(t *testing.T) {
	path := os.Getenv("AM_MAINTENANCE_CRASH_FIXTURE")
	if path == "" {
		return
	}
	gate, _ := openGate(t, path)
	if _, err := gate.Enter(t.Context(), "operator", "crash fixture"); err != nil {
		t.Fatal(err)
	}
	os.Exit(23) // no deferred database close or graceful lifecycle cleanup
}

func TestMaintenanceStatusKeepsUnobservedWorkUnknown(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	state, err := gate.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var projection map[string]any
	if err := json.Unmarshal(raw, &projection); err != nil {
		t.Fatal(err)
	}
	if projection["remaining"] != nil {
		t.Fatalf("unobserved work was presented as a measured count: %s", raw)
	}
}

func TestDrainCannotClaimSuccessFromUnknownWork(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	if _, err := gate.Enter(t.Context(), "operator", "maintenance"); err != nil {
		t.Fatal(err)
	}
	result, err := gate.Wait(t.Context(), func(context.Context) (int, error) { return 0, errors.New("owner unavailable") }, time.Hour)
	if err == nil || result.Drained {
		t.Fatalf("unknown work became drained: %+v %v", result, err)
	}
}
