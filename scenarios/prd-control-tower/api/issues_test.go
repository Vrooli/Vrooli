package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// fakeBacklogService is a stand-in BacklogService for the Connect contract.
type fakeBacklogService struct {
	created *apipb.CreateBacklogItemRequest
	create  func(*apipb.CreateBacklogItemRequest) *apipb.BacklogItemResponse
	get     func(*apipb.GetBacklogItemRequest) (*apipb.BacklogItemResponse, error)
}

func (f *fakeBacklogService) CreateItem(_ context.Context, req *connect.Request[apipb.CreateBacklogItemRequest]) (*connect.Response[apipb.BacklogItemResponse], error) {
	f.created = req.Msg
	return connect.NewResponse(f.create(req.Msg)), nil
}

func (f *fakeBacklogService) GetItem(_ context.Context, req *connect.Request[apipb.GetBacklogItemRequest]) (*connect.Response[apipb.BacklogItemResponse], error) {
	resp, err := f.get(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func withFakeBacklog(t *testing.T, svc *fakeBacklogService) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := apiconnect.NewBacklogServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	prevClient := backlogHTTPClient
	prevResolver := resolveSwarmManagerURL
	backlogHTTPClient = srv.Client()
	resolveSwarmManagerURL = func(context.Context) (string, error) { return srv.URL, nil }
	t.Cleanup(func() {
		backlogHTTPClient = prevClient
		resolveSwarmManagerURL = prevResolver
	})
}

func itemResp(kind, name, status string, priority int32, pos *int32, deduped bool) *apipb.BacklogItemResponse {
	return &apipb.BacklogItemResponse{
		Item: &domainpb.BacklogItem{
			Kind: kind, Name: name, Status: status, Priority: priority, QueuePosition: pos,
		},
		Deduped: deduped,
	}
}

func TestHandleSubmitIssueReport_FilesBacklogFix(t *testing.T) {
	pos := int32(2)
	svc := &fakeBacklogService{
		create: func(*apipb.CreateBacklogItemRequest) *apipb.BacklogItemResponse {
			return itemResp("fix", "report-quality-issues", "backlog", 3, &pos, false)
		},
	}
	withFakeBacklog(t, svc)

	payload := ScenarioIssueReportRequest{
		EntityType:  "scenario",
		EntityName:  "gamma",
		Source:      "user",
		Title:       "Report quality issues",
		Description: "details",
		Priority:    "high",
		Selections: []IssueReportSelection{
			{ID: "missing-section", Title: "Missing section", Detail: "Vision", Category: "structure", Severity: "high"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/report", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleSubmitIssueReport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var fb BacklogFeedback
	if err := json.NewDecoder(rec.Body).Decode(&fb); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if fb.ItemID != "fix/report-quality-issues" {
		t.Errorf("unexpected item_id %q", fb.ItemID)
	}
	if fb.QueuePosition == nil || *fb.QueuePosition != 2 {
		t.Errorf("expected queue_position 2, got %v", fb.QueuePosition)
	}
	if fb.DeepLink != "/apps/swarm-manager/proxy/backlog/fix/report-quality-issues" {
		t.Errorf("unexpected deep_link %q", fb.DeepLink)
	}

	// Verify the create request carried origin + target + kind=fix.
	got := svc.created
	if got == nil {
		t.Fatal("create was not called")
	}
	if got.Kind != "fix" {
		t.Errorf("expected kind fix, got %s", got.Kind)
	}
	if got.GetPriority() != 3 {
		t.Errorf("expected priority 3 (high), got %d", got.GetPriority())
	}
	assertContains(t, got.Tags, "origin:prd-control-tower")
	assertContains(t, got.Tags, "sla:user-initiated")
	assertContains(t, got.AcceptanceAllow, "scenarios/gamma/**")
}

func TestHandleGetScenarioIssuesStatus_ReadsItem(t *testing.T) {
	svc := &fakeBacklogService{
		get: func(req *apipb.GetBacklogItemRequest) (*apipb.BacklogItemResponse, error) {
			if req.Kind != "fix" || req.Name != "some-item" {
				t.Fatalf("unexpected get %s/%s", req.Kind, req.Name)
			}
			return itemResp("fix", "some-item", "in_progress", 5, nil, false), nil
		},
	}
	withFakeBacklog(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/status?kind=fix&name=some-item", nil)
	rec := httptest.NewRecorder()
	handleGetScenarioIssuesStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var fb BacklogFeedback
	if err := json.NewDecoder(rec.Body).Decode(&fb); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if fb.Status != "in_progress" {
		t.Errorf("expected status in_progress, got %s", fb.Status)
	}
	if fb.QueuePosition != nil {
		t.Errorf("expected nil queue_position for non-pending item, got %v", *fb.QueuePosition)
	}
}

func TestHandleGetScenarioIssuesStatus_NotFound(t *testing.T) {
	svc := &fakeBacklogService{
		get: func(*apipb.GetBacklogItemRequest) (*apipb.BacklogItemResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		},
	}
	withFakeBacklog(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/status?kind=fix&name=missing", nil)
	rec := httptest.NewRecorder()
	handleGetScenarioIssuesStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSubmitIssueReport_Validation(t *testing.T) {
	payload := ScenarioIssueReportRequest{
		EntityType:  "scenario",
		EntityName:  "delta",
		Title:       "Incomplete",
		Description: "",
		Selections:  nil,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/report", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleSubmitIssueReport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected validation error, got %d", rec.Code)
	}
}

func TestSlaClassTag(t *testing.T) {
	if got := slaClassTag("quality-scanner"); got != "sla:auto-detected" {
		t.Errorf("expected auto-detected, got %s", got)
	}
	if got := slaClassTag("user"); got != "sla:user-initiated" {
		t.Errorf("expected user-initiated, got %s", got)
	}
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Errorf("expected %q in %v", needle, haystack)
}
