package sessions

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func TestVariablesSurviveSubmissionBoundary(t *testing.T) { // [REQ:PRT-P0-004]
	m := NewManager(Options{})
	s, err := m.Create("", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Touch(s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSessionsAreIsolatedFromEachOther(t *testing.T) { // [REQ:PRT-P0-004]
	m := NewManager(Options{})
	a, _ := m.Create("", "", nil)
	b, _ := m.Create("", "", nil)
	if a.ID == b.ID || m.HasGrant(b.ID, "state:a") {
		t.Fatal("session state or identity leaked")
	}
}

func TestSessionBindsToSandboxWorkspace(t *testing.T) { // [REQ:PRT-P1-004]
	m := NewManager(Options{})
	s, err := m.Create("", "/workspace/one", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.SandboxWorkspace != "/workspace/one" {
		t.Fatalf("workspace=%q", s.SandboxWorkspace)
	}
}

func TestReclaimsIdleSessionWithStatedReason(t *testing.T) { // [REQ:PRT-P1-005]
	clock := &fakeClock{now: time.Unix(100, 0)}
	m := NewManager(Options{Clock: clock, IdleTimeout: time.Minute})
	s, _ := m.Create("", "", nil)
	clock.now = clock.now.Add(2 * time.Minute)
	if got := m.ReclaimIdle(clock.now); len(got) != 1 || got[0] != s.ID {
		t.Fatalf("reclaimed=%v", got)
	}
	if _, err := m.Get(s.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get err=%v", err)
	}
}

func TestEnforcesWallClockAndMemoryCeilings(t *testing.T) { // [REQ:PRT-P1-005]
	clock := &fakeClock{now: time.Unix(100, 0)}
	m := NewManager(Options{Clock: clock, WallTimeout: time.Minute, MemoryLimit: 10})
	s, _ := m.Create("", "", nil)
	if err := m.SetMemoryBytes(s.ID, 11); !errors.Is(err, ErrReclaimed) {
		t.Fatalf("memory err=%v", err)
	}
	s, _ = m.Create("", "", nil)
	clock.now = clock.now.Add(2 * time.Minute)
	if len(m.ReclaimIdle(clock.now)) != 1 {
		t.Fatal("wall clock session was not reclaimed")
	}
}

func TestNamedSessionSurvivesAcrossAgentRuns(t *testing.T) { // [REQ:PRT-P2-003]
	m := NewManager(Options{})
	s, _ := m.Create("investigation", "", []string{"network:internal"})
	got, _ := m.Get(s.ID)
	if got.Name != "investigation" || !m.HasGrant(s.ID, "network:internal") {
		t.Fatal("named session metadata did not survive lookup")
	}
}

func TestKernelSpawnFailureLeavesExistingSessionsUntouched(t *testing.T) {
	m := NewManager(Options{})
	existing, _ := m.Create("", "", nil)
	failing := NewManager(Options{KernelFactory: func(string) (Kernel, error) { return nil, errors.New("python unavailable") }})
	if _, err := failing.Create("", "", nil); !errors.Is(err, ErrKernelStart) {
		t.Fatalf("err=%v", err)
	}
	if _, err := m.Get(existing.ID); err != nil {
		t.Fatal(err)
	}
}
