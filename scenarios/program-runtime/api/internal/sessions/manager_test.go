package sessions

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

var testContext = context.Background()

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type fakeWorkspaceResolver struct {
	paths map[string]string
}

func (r fakeWorkspaceResolver) Resolve(_ context.Context, id string) (string, error) {
	path, ok := r.paths[id]
	if !ok {
		return "", fmt.Errorf("workspace %q not found", id)
	}
	return path, nil
}

func TestVariablesSurviveSubmissionBoundary(t *testing.T) { // [REQ:PRT-P0-004]
	m := NewManager(Options{})
	s, err := m.Create(testContext, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Touch(testContext, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(testContext, s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSessionsAreIsolatedFromEachOther(t *testing.T) { // [REQ:PRT-P0-004]
	m := NewManager(Options{})
	a, _ := m.Create(testContext, "", "", nil)
	b, _ := m.Create(testContext, "", "", nil)
	if a.ID == b.ID || m.HasGrant(testContext, b.ID, "state:a") {
		t.Fatal("session state or identity leaked")
	}
}

func TestSessionBindsToSandboxWorkspace(t *testing.T) { // [REQ:PRT-P1-004]
	root := t.TempDir()
	m := NewManager(Options{WorkspaceResolver: fakeWorkspaceResolver{paths: map[string]string{"workspace-one": root}}})
	s, err := m.Create(testContext, "", "workspace-one", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.SandboxWorkspace != root {
		t.Fatalf("workspace=%q", s.SandboxWorkspace)
	}
}

func TestSessionRejectsUnknownSandboxWorkspace(t *testing.T) { // [REQ:PRT-P1-004]
	m := NewManager(Options{WorkspaceResolver: fakeWorkspaceResolver{paths: map[string]string{}}})
	_, err := m.Create(testContext, "", "does-not-exist", nil)
	if !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("err=%v, want ErrInvalidWorkspace", err)
	}
}

func TestLocalWorkspaceFallbackRequiresExistingAbsoluteDirectory(t *testing.T) {
	root := t.TempDir()
	resolver := &localWorkspaceResolver{}
	got, err := resolver.Resolve(testContext, filepath.Clean(root))
	if err != nil || got != root {
		t.Fatalf("resolved=%q err=%v", got, err)
	}
	if _, err := resolver.Resolve(testContext, "relative-workspace"); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("relative path err=%v", err)
	}
}

func TestReclaimsIdleSessionWithStatedReason(t *testing.T) { // [REQ:PRT-P1-005]
	clock := &fakeClock{now: time.Unix(100, 0)}
	m := NewManager(Options{Clock: clock, IdleTimeout: time.Minute})
	s, _ := m.Create(testContext, "", "", nil)
	clock.now = clock.now.Add(2 * time.Minute)
	if got := m.ReclaimIdle(testContext, clock.now); len(got) != 1 || got[0] != s.ID {
		t.Fatalf("reclaimed=%v", got)
	}
	if _, err := m.Get(testContext, s.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get err=%v", err)
	}
}

func TestEnforcesWallClockAndMemoryCeilings(t *testing.T) { // [REQ:PRT-P1-005]
	clock := &fakeClock{now: time.Unix(100, 0)}
	m := NewManager(Options{Clock: clock, WallTimeout: time.Minute, MemoryLimit: 10})
	s, _ := m.Create(testContext, "", "", nil)
	if err := m.SetMemoryBytes(testContext, s.ID, 11); !errors.Is(err, ErrReclaimed) {
		t.Fatalf("memory err=%v", err)
	}
	s, _ = m.Create(testContext, "", "", nil)
	clock.now = clock.now.Add(2 * time.Minute)
	if len(m.ReclaimIdle(testContext, clock.now)) != 1 {
		t.Fatal("wall clock session was not reclaimed")
	}
}

func TestNamedSessionSurvivesAcrossAgentRuns(t *testing.T) { // [REQ:PRT-P2-003]
	m := NewManager(Options{})
	s, _ := m.Create(testContext, "investigation", "", []string{"network:internal"})
	got, _ := m.Get(testContext, s.ID)
	if got.Name != "investigation" || !m.HasGrant(testContext, s.ID, "network:internal") {
		t.Fatal("named session metadata did not survive lookup")
	}
}

func TestManagerRehydratesNamedSessionAfterRestart(t *testing.T) { // [REQ:PRT-P2-003]
	ctx := context.Background()
	d := newSessionTestDB(t)
	root := t.TempDir()
	resolver := fakeWorkspaceResolver{paths: map[string]string{"workspace-durable": root}}
	first := NewManager(Options{Store: d, WorkspaceResolver: resolver})
	want, err := first.Create(ctx, "durable-run", "workspace-durable", []string{"network:internal"})
	if err != nil {
		t.Fatal(err)
	}

	second := NewManager(Options{Store: d, WorkspaceResolver: resolver})
	got, err := second.Get(ctx, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name || got.SandboxWorkspace != want.SandboxWorkspace || !second.HasGrant(ctx, want.ID, "network:internal") {
		t.Fatalf("rehydrated session lost durable metadata: %+v", got)
	}
}

func TestSQLiteManagerPersistsReclamationReason(t *testing.T) { // [REQ:PRT-P1-005]
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: now}
	d := newSessionTestDB(t)
	m := NewManager(Options{Store: d, Clock: clock, IdleTimeout: time.Minute})
	s, err := m.Create(ctx, "idle", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	clock.now = now.Add(2 * time.Minute)
	if got := m.ReclaimIdle(ctx, clock.now); len(got) != 1 || got[0] != s.ID {
		t.Fatalf("reclaimed=%v", got)
	}
	var reason string
	if err := d.QueryRowContext(ctx, `SELECT reason FROM reclamation_reasons WHERE session_id = ?`, s.ID).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "idle timeout exceeded" {
		t.Fatalf("reason=%q", reason)
	}
}

func TestKernelSpawnFailureLeavesExistingSessionsUntouched(t *testing.T) {
	m := NewManager(Options{})
	existing, _ := m.Create(testContext, "", "", nil)
	failing := NewManager(Options{KernelFactory: func(string) (Kernel, error) { return nil, errors.New("python unavailable") }})
	if _, err := failing.Create(testContext, "", "", nil); !errors.Is(err, ErrKernelStart) {
		t.Fatalf("err=%v", err)
	}
	if _, err := m.Get(testContext, existing.ID); err != nil {
		t.Fatal(err)
	}
}
