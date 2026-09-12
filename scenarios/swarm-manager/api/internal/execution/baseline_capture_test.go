package execution

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeBaselineClient is a test double for BaselineClient, shared by the
// pre-exec capture and finalization-diff tests.
type fakeBaselineClient struct {
	mu          sync.Mutex
	ensured     map[string]bool // "scenario|name" -> already captured (so next call reports cached)
	ensureErr   error
	diffs       map[string]BaselineDiffResult // scenario -> diff result
	diffErr     error
	deleted     []string    // "scenario|name" recorded on Delete
	ensureCalls chan string // optional: signals "scenario|name" on each EnsureSnapshot
}

func newFakeBaselineClient() *fakeBaselineClient {
	return &fakeBaselineClient{ensured: map[string]bool{}, diffs: map[string]BaselineDiffResult{}}
}

func (f *fakeBaselineClient) EnsureSnapshot(_ context.Context, scenario, name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ensureCalls != nil {
		f.ensureCalls <- scenario + "|" + name
	}
	if f.ensureErr != nil {
		return false, f.ensureErr
	}
	key := scenario + "|" + name
	cached := f.ensured[key]
	f.ensured[key] = true
	return cached, nil
}

func (f *fakeBaselineClient) Diff(_ context.Context, scenario, _ string) (BaselineDiffResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.diffErr != nil {
		return BaselineDiffResult{}, f.diffErr
	}
	if r, ok := f.diffs[scenario]; ok {
		return r, nil
	}
	return BaselineDiffResult{ScenarioName: scenario, Verdict: baselineVerdictClean, Comparable: true}, nil
}

func (f *fakeBaselineClient) Delete(_ context.Context, scenario, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, scenario+"|"+name)
	return nil
}

func (f *fakeBaselineClient) Ping(_ context.Context) error { return nil }

func (f *fakeBaselineClient) deletedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deleted...)
}

func TestComputeStateHash_Deterministic(t *testing.T) {
	contents := map[string][]byte{
		"scenarios/a/main.go": []byte("package main"),
		"scenarios/a/util.go": []byte("package util"),
	}
	h1 := computeStateHash("tree123", " M scenarios/a/main.go", contents)
	h2 := computeStateHash("tree123", " M scenarios/a/main.go", contents)
	if h1 != h2 {
		t.Fatalf("hash not deterministic: %s != %s", h1, h2)
	}
}

func TestComputeStateHash_SensitiveToContent(t *testing.T) {
	base := computeStateHash("t", "status", map[string][]byte{"f": []byte("v1")})
	changedContent := computeStateHash("t", "status", map[string][]byte{"f": []byte("v2")})
	changedTree := computeStateHash("t2", "status", map[string][]byte{"f": []byte("v1")})
	changedStatus := computeStateHash("t", "status2", map[string][]byte{"f": []byte("v1")})
	for name, h := range map[string]string{"content": changedContent, "tree": changedTree, "status": changedStatus} {
		if h == base {
			t.Errorf("hash should change when %s changes", name)
		}
	}
}

func TestChangedPathsFromPorcelain(t *testing.T) {
	porcelain := " M scenarios/a/main.go\n?? scenarios/a/new.txt\nR  scenarios/a/old.go -> scenarios/a/renamed.go\nA  scenarios/a/added.go"
	got := changedPathsFromPorcelain(porcelain)
	want := map[string]bool{
		"scenarios/a/main.go":    true,
		"scenarios/a/new.txt":    true,
		"scenarios/a/renamed.go": true, // rename resolves to new path
		"scenarios/a/added.go":   true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d paths %v, want %d", len(got), got, len(want))
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected path %q", p)
		}
	}
}

func TestBaselineNameFor(t *testing.T) {
	name := baselineNameFor("swarm-manager", "abcdef0123456789deadbeef")
	want := "preexec-swarm-manager-abcdef012345"
	if name != want {
		t.Errorf("baselineNameFor = %q, want %q", name, want)
	}
	short := baselineNameFor("x", "abc")
	if short != "preexec-x-abc" {
		t.Errorf("short hash name = %q, want preexec-x-abc", short)
	}
}

func TestCapturePreExecBaselines_DisabledOrNilClient(t *testing.T) {
	item := backlogItem{AcceptanceAllow: []string{"scenarios/git-control-tower/**"}}

	disabled := &Service{
		repoRoot:         ".",
		selfScenarioName: "swarm-manager",
		baselineClient:   newFakeBaselineClient(),
		finalizationCfg:  FinalizationConfig{BaselineDiffEnabled: false},
	}
	if got := disabled.capturePreExecBaselinesLocked(context.Background(), item); got != nil {
		t.Errorf("disabled feature should return nil, got %v", got)
	}

	nilClient := &Service{
		repoRoot:         ".",
		selfScenarioName: "swarm-manager",
		finalizationCfg:  FinalizationConfig{BaselineDiffEnabled: true},
	}
	if got := nilClient.capturePreExecBaselinesLocked(context.Background(), item); got != nil {
		t.Errorf("nil client should return nil, got %v", got)
	}
}

func TestCapturePreExecBaselines_DeclaredScenarioSkipsSelf(t *testing.T) {
	fake := newFakeBaselineClient()
	fake.ensureCalls = make(chan string, 4)
	svc := &Service{
		repoRoot:         ".",
		selfScenarioName: "swarm-manager",
		baselineClient:   fake,
		finalizationCfg:  FinalizationConfig{BaselineDiffEnabled: true},
	}
	// Declares both self (skipped) and git-control-tower (captured).
	item := backlogItem{AcceptanceAllow: []string{
		"scenarios/swarm-manager/**",
		"scenarios/git-control-tower/**",
	}}

	names := svc.capturePreExecBaselinesLocked(context.Background(), item)
	if _, ok := names["swarm-manager"]; ok {
		t.Errorf("self scenario must be skipped, got %v", names)
	}
	name, ok := names["git-control-tower"]
	if !ok {
		t.Fatalf("expected git-control-tower baseline name, got %v", names)
	}
	if name == "" || name[:8] != "preexec-" {
		t.Errorf("unexpected baseline name %q", name)
	}

	// The detached goroutine should call EnsureSnapshot exactly once (gct only).
	select {
	case call := <-fake.ensureCalls:
		if call != "git-control-tower|"+name {
			t.Errorf("EnsureSnapshot called with %q, want git-control-tower|%s", call, name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("EnsureSnapshot was not called within timeout")
	}
}
