package backlog

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func createViaConnect(t *testing.T, svc *ConnectService, req *apipb.CreateBacklogItemRequest) *apipb.BacklogItemResponse {
	t.Helper()
	resp, err := svc.CreateItem(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
	return resp.Msg
}

func TestConnectCreateItem_FilesFixWithOriginTag(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	resp := createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name:            "scenario-x-broken",
		Title:           "Scenario X is broken",
		Kind:            "fix",
		Description:     strPtr("It does not start"),
		Tags:            []string{"origin:business-health", "user-initiated"},
		AcceptanceAllow: []string{"scenarios/scenario-x/**"},
	})

	if resp.Deduped {
		t.Fatalf("first create should not be deduped")
	}
	item := resp.Item
	if item.Kind != "fix" {
		t.Errorf("expected kind fix, got %s", item.Kind)
	}
	if item.GetQueuePosition() != 0 {
		// Only item pending → position 0.
		t.Errorf("expected queue_position 0 for sole pending item, got %d", item.GetQueuePosition())
	}
	// origin tag round-trips and the dedup signature tag was added.
	var hasOrigin, hasSig bool
	for _, tag := range item.Tags {
		if tag == "origin:business-health" {
			hasOrigin = true
		}
		if len(tag) > 4 && tag[:4] == "sig:" {
			hasSig = true
		}
	}
	if !hasOrigin {
		t.Errorf("origin tag did not round-trip: %v", item.Tags)
	}
	if !hasSig {
		t.Errorf("dedup signature tag not attached: %v", item.Tags)
	}
}

func TestConnectCreateItem_DedupReturnsExisting(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	base := &apipb.CreateBacklogItemRequest{
		Name:            "dup",
		Title:           "Duplicate report",
		Kind:            "fix",
		Tags:            []string{"origin:app-monitor"},
		AcceptanceAllow: []string{"scenarios/foo/**"},
	}
	first := createViaConnect(t, svc, base)
	if first.Deduped {
		t.Fatalf("first create unexpectedly deduped")
	}

	// Same target + title + origin (different name) → must dedup onto first.
	second := createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name:            "dup-again",
		Title:           "Duplicate report",
		Kind:            "fix",
		Tags:            []string{"origin:app-monitor"},
		AcceptanceAllow: []string{"scenarios/foo/**"},
	})
	if !second.Deduped {
		t.Fatalf("second create should be deduped")
	}
	if second.Item.Name != first.Item.Name {
		t.Errorf("dedup returned wrong item: got %s want %s", second.Item.Name, first.Item.Name)
	}
}

func TestConnectCreateItem_DistinctTargetsNotDeduped(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "a", Title: "Same title", Kind: "fix",
		AcceptanceAllow: []string{"scenarios/foo/**"},
	})
	second := createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "b", Title: "Same title", Kind: "fix",
		AcceptanceAllow: []string{"scenarios/bar/**"},
	})
	if second.Deduped {
		t.Errorf("distinct targets should not dedup")
	}
}

func TestConnectGetItem_QueuePosition(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	// Two pending items, distinct priorities → lower priority number ranks first.
	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "high", Title: "High prio", Kind: "fix", Priority: int32Ptr(2),
		AcceptanceAllow: []string{"scenarios/h/**"},
	})
	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "low", Title: "Low prio", Kind: "fix", Priority: int32Ptr(8),
		AcceptanceAllow: []string{"scenarios/l/**"},
	})

	got, err := svc.GetItem(context.Background(), connect.NewRequest(&apipb.GetBacklogItemRequest{Kind: "fix", Name: "low"}))
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if got.Msg.Item.GetQueuePosition() != 1 {
		t.Errorf("expected low-priority item at position 1, got %d", got.Msg.Item.GetQueuePosition())
	}

	gotHigh, err := svc.GetItem(context.Background(), connect.NewRequest(&apipb.GetBacklogItemRequest{Kind: "fix", Name: "high"}))
	if err != nil {
		t.Fatalf("GetItem high failed: %v", err)
	}
	if gotHigh.Msg.Item.GetQueuePosition() != 0 {
		t.Errorf("expected high-priority item at position 0, got %d", gotHigh.Msg.Item.GetQueuePosition())
	}
}

func TestConnectGetItem_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	_, err := svc.GetItem(context.Background(), connect.NewRequest(&apipb.GetBacklogItemRequest{Kind: "fix", Name: "nope"}))
	if err == nil {
		t.Fatalf("expected error for missing item")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connect.CodeOf(err))
	}
}

func TestConnectListItems_AppliesTypedFilters(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "research-item", Title: "Research item", Kind: "research",
		AcceptanceAllow: []string{"scenarios/research/**"},
	})
	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "fix-item", Title: "Fix item", Kind: "fix",
		AcceptanceAllow: []string{"scenarios/fix/**"},
	})

	response, err := svc.ListItems(context.Background(), connect.NewRequest(&apipb.ListBacklogItemsRequest{
		Kinds:    []string{"research"},
		Archived: apipb.ArchivedFilter_ARCHIVED_FILTER_EXCLUDE,
	}))
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(response.Msg.GetItems()) != 1 {
		t.Fatalf("expected one filtered item, got %d", len(response.Msg.GetItems()))
	}
	item := response.Msg.GetItems()[0]
	if item.GetKind() != "research" || item.GetName() != "research-item" {
		t.Errorf("unexpected filtered item: %s/%s", item.GetKind(), item.GetName())
	}
}

func TestConnectListItems_InvalidKind(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	_, err := svc.ListItems(context.Background(), connect.NewRequest(&apipb.ListBacklogItemsRequest{Kinds: []string{"bogus"}}))
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestConnectDeleteItem_IsIdempotent(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)
	createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
		Name: "delete-me", Title: "Delete me", Kind: "fix",
		AcceptanceAllow: []string{"scenarios/delete/**"},
	})

	first, err := svc.DeleteItem(context.Background(), connect.NewRequest(&apipb.DeleteBacklogItemRequest{Kind: "fix", Name: "delete-me"}))
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	if !first.Msg.GetDeleted() {
		t.Fatalf("first delete should report deleted")
	}
	second, err := svc.DeleteItem(context.Background(), connect.NewRequest(&apipb.DeleteBacklogItemRequest{Kind: "fix", Name: "delete-me"}))
	if err != nil {
		t.Fatalf("idempotent DeleteItem failed: %v", err)
	}
	if second.Msg.GetDeleted() {
		t.Error("second delete should report an already-absent item")
	}
}

func TestConnectCreateItem_InvalidKind(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)

	_, err := svc.CreateItem(context.Background(), connect.NewRequest(&apipb.CreateBacklogItemRequest{
		Name: "x", Title: "X", Kind: "bogus",
	}))
	if err == nil {
		t.Fatalf("expected error for invalid kind")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}
