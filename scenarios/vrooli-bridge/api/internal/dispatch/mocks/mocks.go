// Package mocks holds the dispatch domain's co-located test fakes for its
// seams (NodeReader, Presence, RunController, AuditSink, JobPusher). Deleting
// internal/dispatch/ takes these with it.
package mocks

import (
	"context"
	"sync"

	"vrooli-bridge/internal/dispatch"
)

// FakeNodeReader resolves canned TargetNodes by id.
type FakeNodeReader struct {
	Nodes  map[string]dispatch.TargetNode
	GetErr error
}

var _ dispatch.NodeReader = (*FakeNodeReader)(nil)

func (f *FakeNodeReader) GetTarget(_ context.Context, id string) (dispatch.TargetNode, error) {
	if f.GetErr != nil {
		return dispatch.TargetNode{}, f.GetErr
	}
	n, ok := f.Nodes[id]
	if !ok {
		return dispatch.TargetNode{}, dispatch.ErrNodeNotFound{ID: id}
	}
	return n, nil
}

// FakePresence reports online state from a set.
type FakePresence struct {
	Online map[string]bool
}

var _ dispatch.Presence = (*FakePresence)(nil)

func (f *FakePresence) IsOnline(nodeID string) bool { return f.Online[nodeID] }

// FakeRunController records created/aborted runs and hands out deterministic ids.
type FakeRunController struct {
	mu        sync.Mutex
	Created   []dispatch.CreateRunInput
	Aborted   []string
	NextRunID string
	CreateErr error
	AbortErr  error
	seq       int
}

var _ dispatch.RunController = (*FakeRunController)(nil)

func (f *FakeRunController) CreateRun(_ context.Context, in dispatch.CreateRunInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.CreateErr != nil {
		return "", f.CreateErr
	}
	f.Created = append(f.Created, in)
	if f.NextRunID != "" {
		return f.NextRunID, nil
	}
	f.seq++
	return "run-" + string(rune('0'+f.seq)), nil
}

func (f *FakeRunController) AbortRun(_ context.Context, runID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Aborted = append(f.Aborted, runID)
	return f.AbortErr
}

// FakeAuditSink records audit entries; AppendErr forces the fail-closed path.
type FakeAuditSink struct {
	mu        sync.Mutex
	Entries   []dispatch.Entry
	RecordErr error
}

var _ dispatch.AuditSink = (*FakeAuditSink)(nil)

func (f *FakeAuditSink) Record(_ context.Context, e dispatch.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.RecordErr != nil {
		return f.RecordErr
	}
	f.Entries = append(f.Entries, e)
	return nil
}

// Recorded returns a copy of the audit entries.
func (f *FakeAuditSink) Recorded() []dispatch.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]dispatch.Entry(nil), f.Entries...)
}

// FakeJobPusher records pushed jobs and controls delivery count / error.
type FakeJobPusher struct {
	mu        sync.Mutex
	Pushed    []dispatch.PushedJob
	Delivered int
	PushErr   error
	// DeliveredSet marks Delivered as meaningful; when false the fake delivers 1.
	DeliveredSet bool
}

var _ dispatch.JobPusher = (*FakeJobPusher)(nil)

func (f *FakeJobPusher) PushJob(_ context.Context, _ string, job dispatch.PushedJob) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PushErr != nil {
		return 0, f.PushErr
	}
	f.Pushed = append(f.Pushed, job)
	if f.DeliveredSet {
		return f.Delivered, nil
	}
	return 1, nil
}

// PushedJobs returns a copy of the pushed jobs.
func (f *FakeJobPusher) PushedJobs() []dispatch.PushedJob {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]dispatch.PushedJob(nil), f.Pushed...)
}
