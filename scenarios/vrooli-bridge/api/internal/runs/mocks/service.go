package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"vrooli-bridge/internal/runs"
)

// FakeService is a runs.Service double for handler tests (and the dispatch
// handler's RunCreator) — it records inputs and returns canned values without
// repository plumbing or coordination.
type FakeService struct {
	mu sync.Mutex

	CreateInputs []runs.CreateInput
	AppendEvents []runs.RunEvent
	DeliveryAcks []runs.DeliveryAck
	WaitIDs      []string
	AbortIDs     []string

	CreateOut runs.Run
	CreateErr error

	GetRun    runs.Run
	GetEvents []runs.RunEvent
	GetErr    error

	ListOut []runs.Run
	ListErr error

	AppendAccepted       bool
	AppendErr            error
	RecordDeliveryAckErr error
	MarkDeliveryStateErr error

	WaitOut     runs.Run
	WaitTimedM  bool
	WaitErr     error
	WaitBlockMS int

	AbortOut runs.Run
	AbortErr error

	SubCh chan runs.RunEvent

	CreateCalls atomic.Int64
	AppendCalls atomic.Int64
}

var _ runs.Service = (*FakeService)(nil)

func (f *FakeService) Create(_ context.Context, in runs.CreateInput) (runs.Run, error) {
	f.CreateCalls.Add(1)
	f.mu.Lock()
	f.CreateInputs = append(f.CreateInputs, in)
	f.mu.Unlock()
	if f.CreateErr != nil {
		return runs.Run{}, f.CreateErr
	}
	return f.CreateOut, nil
}

func (f *FakeService) Get(_ context.Context, id string) (runs.Run, []runs.RunEvent, error) {
	if f.GetErr != nil {
		return runs.Run{}, nil, f.GetErr
	}
	return f.GetRun, f.GetEvents, nil
}

func (f *FakeService) List(_ context.Context, _ runs.ListFilter) ([]runs.Run, error) {
	return f.ListOut, f.ListErr
}

func (f *FakeService) AppendEvent(_ context.Context, ev runs.RunEvent) (bool, error) {
	f.AppendCalls.Add(1)
	f.mu.Lock()
	f.AppendEvents = append(f.AppendEvents, ev)
	f.mu.Unlock()
	return f.AppendAccepted, f.AppendErr
}

func (f *FakeService) RecordDeliveryAck(_ context.Context, ack runs.DeliveryAck) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DeliveryAcks = append(f.DeliveryAcks, ack)
	return f.RecordDeliveryAckErr
}

func (f *FakeService) MarkDeliveryState(_ context.Context, _ string, _ runs.RunStatus, _ string, _ time.Time, _ ...time.Time) error {
	return f.MarkDeliveryStateErr
}

func (f *FakeService) Wait(ctx context.Context, id string, _ time.Duration) (runs.Run, bool, error) {
	f.mu.Lock()
	f.WaitIDs = append(f.WaitIDs, id)
	f.mu.Unlock()
	if f.WaitBlockMS > 0 {
		select {
		case <-ctx.Done():
			return runs.Run{}, false, ctx.Err()
		case <-time.After(time.Duration(f.WaitBlockMS) * time.Millisecond):
		}
	}
	return f.WaitOut, f.WaitTimedM, f.WaitErr
}

func (f *FakeService) Abort(_ context.Context, id, _ string) (runs.Run, error) {
	f.mu.Lock()
	f.AbortIDs = append(f.AbortIDs, id)
	f.mu.Unlock()
	return f.AbortOut, f.AbortErr
}

func (f *FakeService) Subscribe(string) (<-chan runs.RunEvent, func()) {
	if f.SubCh == nil {
		f.SubCh = make(chan runs.RunEvent)
	}
	return f.SubCh, func() {}
}
