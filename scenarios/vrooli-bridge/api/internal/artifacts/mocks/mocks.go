// Package mocks holds in-memory fakes for the artifacts domain's seams, used by
// the service unit + integration tests. Each fake satisfies an artifacts seam
// with a compile-time assertion.
package mocks

import (
	"context"
	"sync"

	"vrooli-bridge/internal/artifacts"
)

// FakeNodeReader returns nodes from a fixed map.
type FakeNodeReader struct {
	Nodes map[string]artifacts.TargetNode
}

var _ artifacts.NodeReader = (*FakeNodeReader)(nil)

func (f *FakeNodeReader) GetTarget(_ context.Context, id string) (artifacts.TargetNode, error) {
	n, ok := f.Nodes[id]
	if !ok {
		return artifacts.TargetNode{}, artifacts.ErrNodeNotFound{ID: id}
	}
	return n, nil
}

// FakeDelivery is an in-process stand-in for device-sync-hub directed delivery.
// It records each handoff and returns a deterministic delivery ref. Delivered
// controls whether the artifact is reported as already on the node; Err forces a
// transport failure.
type FakeDelivery struct {
	mu        sync.Mutex
	Requests  []artifacts.DeliveryRequest
	Delivered bool
	Err       error
}

var _ artifacts.DirectedDelivery = (*FakeDelivery)(nil)

func (f *FakeDelivery) Deliver(_ context.Context, req artifacts.DeliveryRequest) (artifacts.DeliveryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return artifacts.DeliveryResult{}, f.Err
	}
	f.Requests = append(f.Requests, req)
	return artifacts.DeliveryResult{
		Ref:       "dsh://" + req.NodeID + "/" + req.Name,
		Delivered: f.Delivered,
		Detail:    "accepted by device-sync-hub",
	}, nil
}

// Delivered returns the recorded delivery requests.
func (f *FakeDelivery) DeliveredRequests() []artifacts.DeliveryRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]artifacts.DeliveryRequest(nil), f.Requests...)
}

// FakeRepository is an in-memory artifacts.Repository.
type FakeRepository struct {
	mu    sync.Mutex
	items map[string]artifacts.Distribution
	order []string
	seq   int
}

var _ artifacts.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty fake repository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{items: map[string]artifacts.Distribution{}}
}

func (f *FakeRepository) Create(_ context.Context, d artifacts.Distribution) (artifacts.Distribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d.ID == "" {
		f.seq++
		d.ID = "dist-" + itoa(f.seq)
	}
	if d.Status == artifacts.StatusUnspecified {
		d.Status = artifacts.StatusPending
	}
	f.items[d.ID] = d
	f.order = append([]string{d.ID}, f.order...)
	return d, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (artifacts.Distribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.items[id]
	if !ok {
		return artifacts.Distribution{}, artifacts.ErrDistributionNotFound{ID: id}
	}
	return d, nil
}

func (f *FakeRepository) List(_ context.Context, filter artifacts.ListFilter) ([]artifacts.Distribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]artifacts.Distribution, 0, len(f.order))
	for _, id := range f.order {
		d := f.items[id]
		if filter.NodeID != "" && d.NodeID != filter.NodeID {
			continue
		}
		out = append(out, d)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (f *FakeRepository) UpdateStatus(_ context.Context, id string, status artifacts.DeliveryStatus, deliveryRef, detail string) (artifacts.Distribution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.items[id]
	if !ok {
		return artifacts.Distribution{}, artifacts.ErrDistributionNotFound{ID: id}
	}
	d.Status = status
	if deliveryRef != "" {
		d.DeliveryRef = deliveryRef
	}
	d.Detail = detail
	f.items[id] = d
	return d, nil
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
