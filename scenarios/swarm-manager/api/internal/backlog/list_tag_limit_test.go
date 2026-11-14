package backlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

func seedTaggedItems(t *testing.T, svc *ConnectService) {
	t.Helper()
	for _, item := range []struct {
		name string
		tags []string
	}{
		{"drift-one", []string{"drift", "cli"}},
		{"drift-two", []string{"drift"}},
		{"perf-one", []string{"perf"}},
		{"untagged", nil},
	} {
		createViaConnect(t, svc, &apipb.CreateBacklogItemRequest{
			Name: item.name, Title: item.name, Kind: "fix", Tags: item.tags,
			AcceptanceAllow: []string{"scenarios/" + item.name + "/**"},
		})
	}
}

func TestConnectListItems_TagFilterMatchesAny(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)
	seedTaggedItems(t, svc)

	names := func(tags []string) map[string]bool {
		t.Helper()
		resp, err := svc.ListItems(context.Background(), connect.NewRequest(&apipb.ListBacklogItemsRequest{Tags: tags}))
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if int(resp.Msg.GetTotal()) != len(resp.Msg.GetItems()) {
			t.Fatalf("total=%d items=%d without limit", resp.Msg.GetTotal(), len(resp.Msg.GetItems()))
		}
		out := map[string]bool{}
		for _, item := range resp.Msg.GetItems() {
			out[item.GetName()] = true
		}
		return out
	}

	if got := names([]string{"drift"}); len(got) != 2 || !got["drift-one"] || !got["drift-two"] {
		t.Fatalf("tag=drift: got %v", got)
	}
	if got := names([]string{"perf", "cli"}); len(got) != 2 || !got["perf-one"] || !got["drift-one"] {
		t.Fatalf("tag=perf,cli (match-any): got %v", got)
	}
	if got := names([]string{"nope"}); len(got) != 0 {
		t.Fatalf("tag=nope: got %v", got)
	}
	if got := names(nil); len(got) != 4 {
		t.Fatalf("no tag filter: got %v", got)
	}
}

func TestConnectListItems_LimitCapsItemsAndReportsTotal(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)
	seedTaggedItems(t, svc)

	limit := int32(1)
	resp, err := svc.ListItems(context.Background(), connect.NewRequest(&apipb.ListBacklogItemsRequest{Tags: []string{"drift"}, Limit: &limit}))
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(resp.Msg.GetItems()) != 1 || resp.Msg.GetTotal() != 2 {
		t.Fatalf("want 1 item of total 2, got items=%d total=%d", len(resp.Msg.GetItems()), resp.Msg.GetTotal())
	}

	negative := int32(-1)
	if _, err := svc.ListItems(context.Background(), connect.NewRequest(&apipb.ListBacklogItemsRequest{Limit: &negative})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("negative limit: want InvalidArgument, got %v", err)
	}
}

func TestRESTListItems_TagAndLimitQuery(t *testing.T) {
	h, _ := setupTestHandler(t)
	svc := NewConnectService(h)
	seedTaggedItems(t, svc)

	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/v1/backlog?tag=drift,perf&limit=2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp apipb.ListBacklogItemsResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.GetItems()) != 2 || resp.GetTotal() != 3 {
		t.Fatalf("want 2 items of total 3, got items=%d total=%d", len(resp.GetItems()), resp.GetTotal())
	}

	bad := httptest.NewRecorder()
	h.List(bad, httptest.NewRequest(http.MethodGet, "/api/v1/backlog?limit=-3", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("negative limit status=%d", bad.Code)
	}
}
