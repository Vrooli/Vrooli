package goals

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
)

type fakeCatalogClient struct {
	offersconnect.CatalogServiceClient
	calls atomic.Int32
	err   error
	block chan struct{}
}

func (f *fakeCatalogClient) ListNodes(context.Context, *connect.Request[offersv1.ListNodesRequest]) (*connect.Response[offersv1.ListNodesResponse], error) {
	f.calls.Add(1)
	if f.block != nil {
		<-f.block
	}
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&offersv1.ListNodesResponse{Nodes: []*offersv1.Node{
		{Name: "web-console", ReleaseRank: 2},
		{Name: "money-ledger", ReleaseRank: 5},
	}}), nil
}

type fakeLadderClient struct {
	offersconnect.ReleaseLadderServiceClient
	calls atomic.Int32
}

func (f *fakeLadderClient) GetEnablingDeliverables(context.Context, *connect.Request[offersv1.ReleaseLadderRequest]) (*connect.Response[offersv1.ReleaseLadderResponse], error) {
	f.calls.Add(1)
	return connect.NewResponse(&offersv1.ReleaseLadderResponse{Enabling: []*offersv1.PrerequisiteNode{
		{Node: &offersv1.Node{Name: "scenario-to-desktop"}, DerivedUrgency: 1},
	}}), nil
}

func newCachedReaderForTest(catalog *fakeCatalogClient, ladder *fakeLadderClient, now *time.Time) *OfferDeskRankReader {
	r := NewOfferDeskRankReaderAt("http://offer-desk.test", nil,
		WithOfferDeskCacheTTL(15*time.Second),
		WithOfferDeskNegativeTTL(5*time.Second),
		WithOfferDeskClock(func() time.Time { return *now }))
	r.client = catalog
	r.ladder = ladder
	return r
}

func TestOfferDeskRankReader_SharesOneListNodesAcrossCallsWithinTTL(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	catalog := &fakeCatalogClient{}
	ladder := &fakeLadderClient{}
	reader := newCachedReaderForTest(catalog, ladder, &now)

	for i := 0; i < 20; i++ {
		if rank, err := reader.ReleaseRank("web-console"); err != nil || rank != 2 {
			t.Fatalf("ReleaseRank = %d, %v; want 2", rank, err)
		}
		if max, err := reader.MaxReleaseRank(); err != nil || max != 5 {
			t.Fatalf("MaxReleaseRank = %d, %v; want 5", max, err)
		}
		if ok, err := reader.ValidateDeliverable("money-ledger"); err != nil || !ok {
			t.Fatalf("ValidateDeliverable = %v, %v; want true", ok, err)
		}
		if urgency, err := reader.DerivedUrgency("scenario-to-desktop"); err != nil || urgency != 1 {
			t.Fatalf("DerivedUrgency = %d, %v; want 1", urgency, err)
		}
	}
	if got := catalog.calls.Load(); got != 1 {
		t.Fatalf("ListNodes calls = %d, want 1 inside the TTL", got)
	}
	if got := ladder.calls.Load(); got != 1 {
		t.Fatalf("GetEnablingDeliverables calls = %d, want 1 inside the TTL", got)
	}

	now = now.Add(15 * time.Second)
	if _, err := reader.ReleaseRank("web-console"); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.DerivedUrgency("scenario-to-desktop"); err != nil {
		t.Fatal(err)
	}
	if got := catalog.calls.Load(); got != 2 {
		t.Fatalf("ListNodes calls after TTL = %d, want 2", got)
	}
	if got := ladder.calls.Load(); got != 2 {
		t.Fatalf("GetEnablingDeliverables calls after TTL = %d, want 2", got)
	}
}

func TestOfferDeskRankReader_ConcurrentCallersShareOneInFlightRPC(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	catalog := &fakeCatalogClient{block: make(chan struct{})}
	reader := newCachedReaderForTest(catalog, &fakeLadderClient{}, &now)

	var wg sync.WaitGroup
	results := make(chan int, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rank, err := reader.ReleaseRank("web-console")
			if err != nil {
				t.Errorf("ReleaseRank: %v", err)
			}
			results <- rank
		}()
	}
	deadline := time.After(2 * time.Second)
	for catalog.calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("no ListNodes call started")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(catalog.block)
	wg.Wait()
	close(results)
	for rank := range results {
		if rank != 2 {
			t.Fatalf("rank = %d, want 2", rank)
		}
	}
	if got := catalog.calls.Load(); got != 1 {
		t.Fatalf("ListNodes calls = %d, want 1 shared in-flight RPC", got)
	}
}

func TestOfferDeskRankReader_CachesFailuresBriefly(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	catalog := &fakeCatalogClient{err: errors.New("offer desk down")}
	reader := newCachedReaderForTest(catalog, &fakeLadderClient{}, &now)

	for i := 0; i < 10; i++ {
		if _, err := reader.ReleaseRank("web-console"); err == nil {
			t.Fatal("expected error while offer desk is down")
		}
	}
	if got := catalog.calls.Load(); got != 1 {
		t.Fatalf("ListNodes calls during outage = %d, want 1 (negative cache)", got)
	}
	now = now.Add(5 * time.Second)
	catalog.err = nil
	if rank, err := reader.ReleaseRank("web-console"); err != nil || rank != 2 {
		t.Fatalf("after recovery ReleaseRank = %d, %v; want 2", rank, err)
	}
	if got := catalog.calls.Load(); got != 2 {
		t.Fatalf("ListNodes calls after negative TTL = %d, want 2", got)
	}
}
