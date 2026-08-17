package sessions

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

func TestExecutionBudgetReportsCeilingAndConsumedValue(t *testing.T) {
	m := NewManager(Options{WallBudget: 100 * time.Millisecond, CPUBudget: time.Second})
	s, err := m.Create(testContext, "budget", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ChargeExecution(testContext, s.ID, 75*time.Millisecond, 0); err != nil {
		t.Fatal(err)
	}
	budget, err := m.ExecutionBudget(testContext, s.ID)
	if err != nil || budget.WallConsumed != 75*time.Millisecond {
		t.Fatalf("budget=%+v err=%v", budget, err)
	}
	err = m.ChargeExecution(testContext, s.ID, 30*time.Millisecond, 0)
	if err == nil || !strings.Contains(err.Error(), "wall-clock budget exhausted") || !strings.Contains(err.Error(), "ceiling=100ms") || !strings.Contains(err.Error(), "consumed=105ms") {
		t.Fatalf("budget exhaustion error=%v", err)
	}
	if _, err := m.Get(testContext, s.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("session after budget exhaustion err=%v", err)
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

func TestInferenceUsageAccumulatesAndCeilingRejects(t *testing.T) { // [REQ:PRT-P1-010]
	m := NewManager(Options{})
	s, err := m.CreateWithBudgets(testContext, "metered", "", nil, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.RecordInferenceUsage(testContext, s.ID, 40, 12); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordInferenceUsage(testContext, s.ID, 60, 8); err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureInferenceAvailable(testContext, s.ID); err == nil || !strings.Contains(err.Error(), "inference_spend_exceeded") || !strings.Contains(err.Error(), "ceiling=100") {
		t.Fatalf("ceiling error=%v", err)
	}
	got, err := m.Get(testContext, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.InferenceCostMicros != 100 || got.InferenceTokens != 20 {
		t.Fatalf("usage=%+v", got)
	}
}

func TestSessionWithoutInferenceCeilingRemainsUnlimited(t *testing.T) { // [REQ:PRT-P1-010]
	m := NewManager(Options{})
	s, err := m.Create(testContext, "unlimited", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := m.RecordInferenceUsage(testContext, s.ID, 100, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.EnsureInferenceAvailable(testContext, s.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDelegationSpendReceiptAccumulatesAndCeilingRejects(t *testing.T) { // [REQ:PRT-P1-011]
	m := NewManager(Options{})
	s, err := m.CreateWithBudgets(testContext, "delegated-meter", "", nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.RecordDelegationUsage(testContext, s.ID, 1, true, "metered child charge"); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordDelegationUsage(testContext, s.ID, 1, true, "second child charge"); err == nil || !strings.Contains(err.Error(), "delegated_run_spend_exceeded") {
		t.Fatalf("second delegated charge error=%v", err)
	}
	got, err := m.Get(testContext, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DelegationCostMicros != 1 || !got.DelegationSpendMeasured || got.DelegationSpendNote != "metered child charge" {
		t.Fatalf("delegation spend=%+v", got)
	}
}

func TestDelegationIdentityIsPersistedAndSessionScoped(t *testing.T) {
	db := newSessionTestDB(t)
	first := NewManager(Options{Store: db})
	s, err := first.Create(testContext, "delegation-owner", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	delegation := &Delegation{SessionID: s.ID, ExecutionID: "execution-1", Owner: "owner", WorkflowKey: "owner/workflow", CreatedAt: time.Now().UTC(), LastStatus: "running"}
	if err := first.SaveDelegation(testContext, delegation); err != nil {
		t.Fatal(err)
	}
	second := NewManager(Options{Store: db})
	got, err := second.GetDelegation(testContext, s.ID, delegation.ExecutionID)
	if err != nil || got.WorkflowKey != delegation.WorkflowKey {
		t.Fatalf("delegation=%+v err=%v", got, err)
	}
	if _, err := second.GetDelegation(testContext, "sess_other", delegation.ExecutionID); !errors.Is(err, ErrDelegationNotOwned) {
		t.Fatalf("cross-session lookup err=%v", err)
	}
}

func TestSQLiteSessionSpendSurvivesManagerRestart(t *testing.T) { // [REQ:PRT-P1-010]
	db := newSessionTestDB(t)
	first := NewManager(Options{Store: db})
	s, err := first.CreateWithBudgets(testContext, "durable-meter", "", nil, 500, 700)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.RecordInferenceUsage(testContext, s.ID, 125, 17); err != nil {
		t.Fatal(err)
	}
	if err := first.RecordDelegationUsage(testContext, s.ID, 0, false, "agent-manager charge unavailable"); err != nil {
		t.Fatal(err)
	}
	second := NewManager(Options{Store: db})
	got, err := second.Get(testContext, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.InferenceCostMicros != 125 || got.InferenceTokens != 17 || got.InferenceCeilingMicros != 500 || got.DelegationCeilingMicros != 700 || got.DelegationSpendMeasured || got.DelegationSpendNote == "" {
		t.Fatalf("durable spend=%+v", got)
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

// TestDeleteReclaimsTheKernelProcess guards a leak that idle reclamation never
// had: Delete closed the manager's kernel handle but skipped OnReclaimed, which
// is the hook that kills the kernel process group and removes the session's
// pinned work directory. One validation sweep left 26 live interpreters holding
// 625 MB, all parented to the running API.
func TestDeleteReclaimsTheKernelProcess(t *testing.T) {
	var reclaimed []string
	manager := NewManager(Options{
		Store:       newSessionTestDB(t),
		OnReclaimed: func(id string) { reclaimed = append(reclaimed, id) },
	})
	session, err := manager.Create(context.Background(), "", "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := manager.Delete(context.Background(), session.ID, "test"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if len(reclaimed) != 1 || reclaimed[0] != session.ID {
		t.Fatalf("Delete must reclaim the kernel exactly as idle reclamation does; got %v", reclaimed)
	}
}
