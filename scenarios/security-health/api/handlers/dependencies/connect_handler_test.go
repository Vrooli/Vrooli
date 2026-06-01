package dependencies

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	depdomain "security-health/internal/dependencies"

	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
)

type stubSearcher struct {
	gotReq depdomain.SearchRequest
	resp   depdomain.SearchResponse
	status depdomain.Status
}

func (s *stubSearcher) Search(_ context.Context, req depdomain.SearchRequest) (depdomain.SearchResponse, error) {
	s.gotReq = req
	return s.resp, nil
}
func (s *stubSearcher) Status(context.Context) (depdomain.Status, error) { return s.status, nil }

func TestSearch_MapsFiltersAndRecords(t *testing.T) {
	stub := &stubSearcher{resp: depdomain.SearchResponse{
		ModeUsed: depdomain.ModeText,
		Results: []depdomain.SearchResult{{
			Record: depdomain.DependencyRecord{Scenario: "a", Ecosystem: depdomain.EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-1"}, MaxSeverity: "high"},
			Score:  0.9,
		}},
	}}
	h := NewConnectHandler(Deps{Service: stub})
	resp, err := h.Search(context.Background(), connect.NewRequest(&dependenciesv1.SearchRequest{
		Query:          "net",
		Ecosystem:      dependenciesv1.Ecosystem_ECOSYSTEM_GO,
		VulnerableOnly: true,
		NameGlob:       "golang.org/x/*",
		Limit:          5,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if stub.gotReq.Ecosystem != depdomain.EcosystemGo || !stub.gotReq.VulnerableOnly || stub.gotReq.NameGlob != "golang.org/x/*" || stub.gotReq.Limit != 5 {
		t.Fatalf("request not mapped: %+v", stub.gotReq)
	}
	if len(resp.Msg.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(resp.Msg.Results))
	}
	rec := resp.Msg.Results[0].GetRecord()
	if rec.GetEcosystem() != dependenciesv1.Ecosystem_ECOSYSTEM_GO || rec.GetName() != "golang.org/x/net" || rec.GetMaxSeverity() != "high" {
		t.Errorf("record mapping wrong: %+v", rec)
	}
	if resp.Msg.GetModeUsed() != dependenciesv1.Mode_MODE_TEXT {
		t.Errorf("mode_used = %v, want TEXT", resp.Msg.GetModeUsed())
	}
}

func TestStatus_Maps(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &stubSearcher{status: depdomain.Status{Available: true, IndexedCount: 7, VulnerableCount: 2, LastReconcileAt: "t"}}})
	resp, err := h.Status(context.Background(), connect.NewRequest(&dependenciesv1.StatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetAvailable() || resp.Msg.GetIndexedCount() != 7 || resp.Msg.GetVulnerableCount() != 2 {
		t.Errorf("status mapping wrong: %+v", resp.Msg)
	}
}
