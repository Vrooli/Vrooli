// Package mocks holds the audit domain's co-located test fakes. Deleting
// internal/audit/ takes these with it.
package mocks

import (
	"context"
	"sync"
	"time"

	"vrooli-bridge/internal/audit"

	"github.com/google/uuid"
)

// FakeSink is an in-memory audit.Sink. The dispatch handler tests wire it to
// assert exactly which records a dispatch routes to the accountability
// substrate (and that a denied verb is still audited). It is also the stand-in
// for the workspace-sandbox substrate in the sink-routing integration test:
// the dispatch domain holds the narrow Sink seam and never knows whether the
// concrete substrate is local SQLite or workspace-sandbox.
type FakeSink struct {
	mu        sync.Mutex
	Records   []audit.Record
	AppendErr error
	Now       time.Time
}

var _ audit.Sink = (*FakeSink)(nil)

// Append records the entry and returns it, stamping ID/RecordedAt the way the
// real store does.
func (f *FakeSink) Append(_ context.Context, r audit.Record) (audit.Record, error) {
	if f.AppendErr != nil {
		return audit.Record{}, f.AppendErr
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.RecordedAt.IsZero() {
		if !f.Now.IsZero() {
			r.RecordedAt = f.Now
		} else {
			r.RecordedAt = time.Unix(0, 0).UTC()
		}
	}
	f.mu.Lock()
	f.Records = append(f.Records, r)
	f.mu.Unlock()
	return r, nil
}

// Appended returns a copy of the recorded entries.
func (f *FakeSink) Appended() []audit.Record {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]audit.Record(nil), f.Records...)
}

// FakeReader is an audit.Reader double for handler tests.
type FakeReader struct {
	ListOut []audit.Record
	ListErr error

	LastFilter audit.ListFilter
}

var _ audit.Reader = (*FakeReader)(nil)

func (f *FakeReader) List(_ context.Context, filter audit.ListFilter) ([]audit.Record, error) {
	f.LastFilter = filter
	return f.ListOut, f.ListErr
}
