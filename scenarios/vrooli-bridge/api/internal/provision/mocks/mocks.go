// Package mocks holds the provision domain's co-located test fakes for its
// Repository and seams (NodeReader, Presence, AuditSink, CommandPusher).
// Deleting internal/provision/ takes these with it.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"vrooli-bridge/internal/provision"

	"github.com/google/uuid"
)

// FakeRepository is an in-memory provision.Repository with per-method error
// knobs. Service tests drive the service against a controllable persistence
// layer without sqlite.
type FakeRepository struct {
	mu       sync.Mutex
	ops      map[string]provision.ProvisioningOp
	events   map[string][]provision.ProvisionEvent
	versions map[string]provision.NodeVersion

	CreateErr         error
	GetErr            error
	ListErr           error
	UpdateErr         error
	AppendEventErr    error
	ListEventsErr     error
	UpsertVersionErr  error
	GetNodeVersionErr error

	// Now is the timestamp Create stamps; tests may set it for determinism.
	Now time.Time
}

// NewFakeRepository constructs an empty fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		ops:      make(map[string]provision.ProvisioningOp),
		events:   make(map[string][]provision.ProvisionEvent),
		versions: make(map[string]provision.NodeVersion),
	}
}

var _ provision.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) now() time.Time {
	if !f.Now.IsZero() {
		return f.Now
	}
	return time.Unix(0, 0).UTC()
}

func (f *FakeRepository) Create(_ context.Context, op provision.ProvisioningOp) (provision.ProvisioningOp, error) {
	if f.CreateErr != nil {
		return provision.ProvisioningOp{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = f.now()
	}
	if op.Status == provision.StatusUnspecified {
		op.Status = provision.StatusQueued
	}
	f.ops[op.ID] = op
	return op, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (provision.ProvisioningOp, error) {
	if f.GetErr != nil {
		return provision.ProvisioningOp{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	op, ok := f.ops[id]
	if !ok {
		return provision.ProvisioningOp{}, provision.ErrOpNotFound{ID: id}
	}
	return op, nil
}

func (f *FakeRepository) List(_ context.Context, filter provision.ListFilter) ([]provision.ProvisioningOp, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]provision.ProvisioningOp, 0, len(f.ops))
	for _, op := range f.ops {
		if filter.NodeID != "" && op.NodeID != filter.NodeID {
			continue
		}
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (f *FakeRepository) Update(_ context.Context, op provision.ProvisioningOp) (provision.ProvisioningOp, error) {
	if f.UpdateErr != nil {
		return provision.ProvisioningOp{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.ops[op.ID]
	if !ok {
		return provision.ProvisioningOp{}, provision.ErrOpNotFound{ID: op.ID}
	}
	existing.Status = op.Status
	existing.ResultingRevision = op.ResultingRevision
	existing.ExitCode = op.ExitCode
	existing.StartedAt = op.StartedAt
	existing.FinishedAt = op.FinishedAt
	f.ops[op.ID] = existing
	return existing, nil
}

func (f *FakeRepository) AppendEvent(_ context.Context, ev provision.ProvisionEvent) error {
	if f.AppendEventErr != nil {
		return f.AppendEventErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	// Mirror the sqlite INSERT OR IGNORE de-dup on (op_id, sequence).
	for _, existing := range f.events[ev.OpID] {
		if existing.Sequence == ev.Sequence {
			return nil
		}
	}
	f.events[ev.OpID] = append(f.events[ev.OpID], ev)
	return nil
}

func (f *FakeRepository) ListEvents(_ context.Context, opID string) ([]provision.ProvisionEvent, error) {
	if f.ListEventsErr != nil {
		return nil, f.ListEventsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	evs := append([]provision.ProvisionEvent(nil), f.events[opID]...)
	sort.Slice(evs, func(i, j int) bool { return evs[i].Sequence < evs[j].Sequence })
	return evs, nil
}

func (f *FakeRepository) UpsertNodeVersion(_ context.Context, v provision.NodeVersion) error {
	if f.UpsertVersionErr != nil {
		return f.UpsertVersionErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if v.ReportedAt.IsZero() {
		v.ReportedAt = f.now()
	}
	f.versions[v.NodeID] = v
	return nil
}

func (f *FakeRepository) GetNodeVersion(_ context.Context, nodeID string) (provision.NodeVersion, error) {
	if f.GetNodeVersionErr != nil {
		return provision.NodeVersion{}, f.GetNodeVersionErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.versions[nodeID]
	if !ok {
		return provision.NodeVersion{}, provision.ErrNoNodeVersion{NodeID: nodeID}
	}
	return v, nil
}

// Seed inserts an op directly for test setup, bypassing Create's stamping.
func (f *FakeRepository) Seed(op provision.ProvisioningOp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops[op.ID] = op
}

// SeedVersion inserts a node version directly for test setup.
func (f *FakeRepository) SeedVersion(v provision.NodeVersion) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.versions[v.NodeID] = v
}

// ---- seam fakes ----

// FakeNodeReader resolves canned TargetNodes by id.
type FakeNodeReader struct {
	Nodes  map[string]provision.TargetNode
	GetErr error
}

var _ provision.NodeReader = (*FakeNodeReader)(nil)

func (f *FakeNodeReader) GetTarget(_ context.Context, id string) (provision.TargetNode, error) {
	if f.GetErr != nil {
		return provision.TargetNode{}, f.GetErr
	}
	n, ok := f.Nodes[id]
	if !ok {
		return provision.TargetNode{}, provision.ErrNodeNotFound{ID: id}
	}
	return n, nil
}

// FakePresence reports online state from a set.
type FakePresence struct {
	Online map[string]bool
}

var _ provision.Presence = (*FakePresence)(nil)

func (f *FakePresence) IsOnline(nodeID string) bool { return f.Online[nodeID] }

// FakeAuditSink records audit entries; RecordErr forces the fail-closed path.
type FakeAuditSink struct {
	mu        sync.Mutex
	Entries   []provision.Entry
	RecordErr error
}

var _ provision.AuditSink = (*FakeAuditSink)(nil)

func (f *FakeAuditSink) Record(_ context.Context, e provision.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.RecordErr != nil {
		return f.RecordErr
	}
	f.Entries = append(f.Entries, e)
	return nil
}

// Recorded returns a copy of the recorded entries.
func (f *FakeAuditSink) Recorded() []provision.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]provision.Entry(nil), f.Entries...)
}

// FakeCommandPusher records pushed commands and controls delivery count/error.
type FakeCommandPusher struct {
	mu        sync.Mutex
	Pushed    []provision.PushedCommand
	Delivered int
	PushErr   error
}

var _ provision.CommandPusher = (*FakeCommandPusher)(nil)

func (f *FakeCommandPusher) PushProvision(_ context.Context, _ string, cmd provision.PushedCommand) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PushErr != nil {
		return 0, f.PushErr
	}
	f.Pushed = append(f.Pushed, cmd)
	return f.Delivered, nil
}

// PushedCommands returns a copy of the pushed commands.
func (f *FakeCommandPusher) PushedCommands() []provision.PushedCommand {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]provision.PushedCommand(nil), f.Pushed...)
}
