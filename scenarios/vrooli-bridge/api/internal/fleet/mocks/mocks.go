// Package mocks holds in-memory fakes for the fleet domain's seams, used by the
// fleet service unit tests and the handler tests. Each fake satisfies a fleet
// seam with a compile-time assertion.
package mocks

import (
	"context"
	"sort"
	"sync"

	"vrooli-bridge/internal/fleet"
)

// FakeNodeLister returns a fixed set of nodes.
type FakeNodeLister struct {
	Nodes []fleet.NodeRef
	Err   error
}

var _ fleet.NodeLister = (*FakeNodeLister)(nil)

func (f *FakeNodeLister) ListNodes(context.Context) ([]fleet.NodeRef, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]fleet.NodeRef, len(f.Nodes))
	copy(out, f.Nodes)
	return out, nil
}

// FakePresence reports online + dispatchable state from sets. A node is
// dispatchable when online and not flagged.
type FakePresence struct {
	Online  map[string]bool
	Flagged map[string]bool
}

var _ fleet.Presence = (*FakePresence)(nil)

func (f *FakePresence) IsOnline(nodeID string) bool { return f.Online[nodeID] }

func (f *FakePresence) Dispatchable(nodeID string) bool {
	return f.Online[nodeID] && !f.Flagged[nodeID]
}

// FakeProvisioner records the nodes it was asked to provision and hands out
// deterministic op ids. FailNodes forces a dispatch error for named nodes.
type FakeProvisioner struct {
	mu        sync.Mutex
	Requested []fleet.ProvisionRequest
	FailNodes map[string]error
	seq       int
}

var _ fleet.Provisioner = (*FakeProvisioner)(nil)

func (f *FakeProvisioner) Provision(_ context.Context, in fleet.ProvisionRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Requested = append(f.Requested, in)
	if err, ok := f.FailNodes[in.NodeID]; ok {
		return "", err
	}
	f.seq++
	return "op-" + in.NodeID, nil
}

// RequestedNodes returns the ids the provisioner was asked to provision, sorted.
func (f *FakeProvisioner) RequestedNodes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.Requested))
	for _, r := range f.Requested {
		out = append(out, r.NodeID)
	}
	sort.Strings(out)
	return out
}

// FakeRepository is an in-memory fleet.Repository.
type FakeRepository struct {
	mu       sync.Mutex
	rollouts map[string]fleet.Rollout
	results  map[string][]fleet.NodeResult
	order    []string
	seq      int
}

var _ fleet.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty fake repository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{rollouts: map[string]fleet.Rollout{}, results: map[string][]fleet.NodeResult{}}
}

func (f *FakeRepository) Create(_ context.Context, rollout fleet.Rollout, results []fleet.NodeResult) (fleet.Rollout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if rollout.ID == "" {
		f.seq++
		rollout.ID = "rollout-" + itoa(f.seq)
	}
	f.rollouts[rollout.ID] = rollout
	cp := make([]fleet.NodeResult, len(results))
	copy(cp, results)
	f.results[rollout.ID] = cp
	f.order = append([]string{rollout.ID}, f.order...) // newest-first
	return rollout, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (fleet.Rollout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rollouts[id]
	if !ok {
		return fleet.Rollout{}, fleet.ErrRolloutNotFound{ID: id}
	}
	return r, nil
}

func (f *FakeRepository) Results(_ context.Context, rolloutID string) ([]fleet.NodeResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fleet.NodeResult, len(f.results[rolloutID]))
	copy(out, f.results[rolloutID])
	return out, nil
}

func (f *FakeRepository) List(_ context.Context, filter fleet.ListFilter) ([]fleet.Rollout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fleet.Rollout, 0, len(f.order))
	for _, id := range f.order {
		out = append(out, f.rollouts[id])
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
