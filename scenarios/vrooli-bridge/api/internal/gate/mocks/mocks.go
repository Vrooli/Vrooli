// Package mocks holds in-memory fakes for the gate domain's seams, used by the
// gate service unit tests and the handler tests. Each fake satisfies a gate seam
// with a compile-time assertion.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"vrooli-bridge/internal/gate"
)

// FakeNodeLister returns a fixed set of nodes.
type FakeNodeLister struct {
	Nodes []gate.NodeRef
	Err   error
}

var _ gate.NodeLister = (*FakeNodeLister)(nil)

func (f *FakeNodeLister) ListNodes(context.Context) ([]gate.NodeRef, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]gate.NodeRef, len(f.Nodes))
	copy(out, f.Nodes)
	return out, nil
}

// FakePresence reports online + dispatchable state from sets. A node is
// dispatchable when online and not flagged.
type FakePresence struct {
	Online  map[string]bool
	Flagged map[string]bool
}

var _ gate.Presence = (*FakePresence)(nil)

func (f *FakePresence) IsOnline(nodeID string) bool { return f.Online[nodeID] }

func (f *FakePresence) Dispatchable(nodeID string) bool {
	return f.Online[nodeID] && !f.Flagged[nodeID]
}

// FakeRunner records the validation runs it was asked to dispatch and serves
// per-run verdicts from a map. FailDispatch forces a dispatch rejection for
// named nodes. Verdicts maps a run id to the verdict Verdict/Wait returns.
type FakeRunner struct {
	mu           sync.Mutex
	Dispatched   []gate.DispatchRequest
	FailDispatch map[string]error
	Verdicts     map[string]gate.RunVerdict
	seq          int
}

var _ gate.Runner = (*FakeRunner)(nil)

// NewFakeRunner constructs an empty fake runner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{FailDispatch: map[string]error{}, Verdicts: map[string]gate.RunVerdict{}}
}

func (f *FakeRunner) Dispatch(_ context.Context, in gate.DispatchRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Dispatched = append(f.Dispatched, in)
	if err, ok := f.FailDispatch[in.NodeID]; ok {
		return "", err
	}
	f.seq++
	return "run-" + in.NodeID, nil
}

func (f *FakeRunner) Verdict(_ context.Context, runID string) (gate.RunVerdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Verdicts[runID], nil
}

func (f *FakeRunner) Wait(_ context.Context, runID string, _ time.Duration) (gate.RunVerdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Verdicts[runID], nil
}

// SetVerdict records the verdict a run id resolves to.
func (f *FakeRunner) SetVerdict(runID string, v gate.RunVerdict) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Verdicts[runID] = v
}

// DispatchedNodes returns the ids the runner was asked to validate, sorted.
func (f *FakeRunner) DispatchedNodes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.Dispatched))
	for _, r := range f.Dispatched {
		out = append(out, r.NodeID)
	}
	sort.Strings(out)
	return out
}

// FakeRepository is an in-memory gate.Repository.
type FakeRepository struct {
	mu      sync.Mutex
	gates   map[string]gate.Gate
	results map[string][]gate.OSResult
	order   []string
	seq     int
}

var _ gate.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty fake repository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{gates: map[string]gate.Gate{}, results: map[string][]gate.OSResult{}}
}

func (f *FakeRepository) Create(_ context.Context, g gate.Gate, results []gate.OSResult) (gate.Gate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if g.ID == "" {
		f.seq++
		g.ID = "gate-" + itoa(f.seq)
	}
	f.gates[g.ID] = g
	cp := make([]gate.OSResult, len(results))
	copy(cp, results)
	f.results[g.ID] = cp
	f.order = append([]string{g.ID}, f.order...) // newest-first
	return g, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (gate.Gate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.gates[id]
	if !ok {
		return gate.Gate{}, gate.ErrGateNotFound{ID: id}
	}
	return g, nil
}

func (f *FakeRepository) Results(_ context.Context, gateID string) ([]gate.OSResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]gate.OSResult, len(f.results[gateID]))
	copy(out, f.results[gateID])
	return out, nil
}

func (f *FakeRepository) List(_ context.Context, filter gate.ListFilter) ([]gate.Gate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]gate.Gate, 0, len(f.order))
	for _, id := range f.order {
		out = append(out, f.gates[id])
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
