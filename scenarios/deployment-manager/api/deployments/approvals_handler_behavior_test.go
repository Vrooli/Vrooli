package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type approvalHandlerFakeRepo struct {
	approval  *DeploymentApproval
	list      []*DeploymentApproval
	targets   []RequiredTarget
	platforms []string
	gate      *ReleaseGateStatus
	err       error
}

func (f *approvalHandlerFakeRepo) Create(_ context.Context, approval *DeploymentApproval) error {
	f.approval = approval
	return f.err
}

func (f *approvalHandlerFakeRepo) Get(_ context.Context, id string) (*DeploymentApproval, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.approval != nil && (id == f.approval.ID || id == "approval-1") {
		return f.approval, nil
	}
	return nil, errors.New("approval not found")
}

func (f *approvalHandlerFakeRepo) ListByCommit(context.Context, string, string) ([]*DeploymentApproval, error) {
	return f.list, f.err
}

func (f *approvalHandlerFakeRepo) ListByProfile(context.Context, string, int) ([]*DeploymentApproval, error) {
	return f.list, f.err
}

func (f *approvalHandlerFakeRepo) UpdateDecision(_ context.Context, _ string, decision, reviewer, notes string) error {
	if f.err != nil {
		return f.err
	}
	if f.approval != nil {
		f.approval.Status, f.approval.ApprovedBy, f.approval.Notes = decision, reviewer, notes
	}
	return nil
}

func (f *approvalHandlerFakeRepo) MarkStale(context.Context, string, string, string) error {
	return f.err
}

func (f *approvalHandlerFakeRepo) GetRequiredPlatforms(context.Context, string) ([]string, error) {
	return f.platforms, f.err
}

func (f *approvalHandlerFakeRepo) SetRequiredPlatforms(_ context.Context, _ string, platforms []string) error {
	f.platforms = platforms
	return f.err
}

func (f *approvalHandlerFakeRepo) CheckReleaseGate(context.Context, string, string) (*ReleaseGateStatus, error) {
	return f.gate, f.err
}

func (f *approvalHandlerFakeRepo) GetRequiredTargets(context.Context, string) ([]RequiredTarget, error) {
	return f.targets, f.err
}

func (f *approvalHandlerFakeRepo) SetRequiredTargets(_ context.Context, _ string, targets []RequiredTarget) error {
	f.targets = targets
	return f.err
}

func approvalRequest(t *testing.T, method, path string, body interface{}, vars map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}
	return req, httptest.NewRecorder()
}

func TestApprovalsHandlerHappyPaths(t *testing.T) {
	repo := &approvalHandlerFakeRepo{list: []*DeploymentApproval{{ID: "approval-1", ProfileID: "profile-1", Platform: "linux", Status: ApprovalStatusPending}}}
	h := NewApprovalsHandler(repo, func(string, map[string]interface{}) {})
	vars := map[string]string{"id": "profile-1"}

	req, rr := approvalRequest(t, http.MethodPost, "/api/v1/profiles/profile-1/approvals", CreateApprovalRequest{GitCommitHash: "abc", Platform: "linux", ValidationID: "run-1"}, vars)
	h.Create(rr, req)
	if rr.Code != http.StatusCreated || repo.approval == nil || repo.approval.GitCommitHash != "abc" {
		t.Fatalf("Create() status=%d body=%s approval=%+v", rr.Code, rr.Body.String(), repo.approval)
	}

	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/profiles/profile-1/approvals?commit=abc", nil, vars)
	h.ListByProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ListByProfile(commit) status=%d", rr.Code)
	}
	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/profiles/profile-1/approvals", nil, vars)
	h.ListByProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("ListByProfile(profile) status=%d", rr.Code)
	}

	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/approvals/approval-1", nil, map[string]string{"id": "approval-1"})
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Get() status=%d", rr.Code)
	}

	req, rr = approvalRequest(t, http.MethodPost, "/api/v1/approvals/approval-1/decide", ApprovalDecisionRequest{Decision: ApprovalStatusApproved, Reviewer: "reviewer", Notes: "looks good"}, map[string]string{"id": repo.approval.ID})
	h.Decide(rr, req)
	if rr.Code != http.StatusOK || repo.approval.Status != ApprovalStatusApproved || repo.approval.ApprovedBy != "reviewer" {
		t.Fatalf("Decide() status=%d body=%s approval=%+v", rr.Code, rr.Body.String(), repo.approval)
	}

	repo.gate = &ReleaseGateStatus{ProfileID: "profile-1", GitCommitHash: "abc", Ready: true, Reason: "all evidence passed"}
	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/profiles/profile-1/release-gate?commit=abc", nil, vars)
	h.CheckReleaseGate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("CheckReleaseGate() status=%d", rr.Code)
	}

	req, rr = approvalRequest(t, http.MethodPut, "/api/v1/profiles/profile-1/required-platforms", SetRequiredPlatformsRequest{Platforms: []string{"linux", "windows"}}, vars)
	h.SetRequiredPlatforms(rr, req)
	if rr.Code != http.StatusOK || len(repo.platforms) != 2 {
		t.Fatalf("SetRequiredPlatforms() status=%d platforms=%v", rr.Code, repo.platforms)
	}
	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/profiles/profile-1/required-platforms", nil, vars)
	h.GetRequiredPlatforms(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GetRequiredPlatforms() status=%d", rr.Code)
	}

	targets := []RequiredTarget{{Ramp: "desktop", Platform: "linux", OS: "linux"}}
	req, rr = approvalRequest(t, http.MethodPut, "/api/v1/profiles/profile-1/required-targets", SetRequiredTargetsRequest{Targets: targets}, vars)
	h.SetRequiredTargets(rr, req)
	if rr.Code != http.StatusOK || len(repo.targets) != 1 {
		t.Fatalf("SetRequiredTargets() status=%d targets=%v", rr.Code, repo.targets)
	}
	req, rr = approvalRequest(t, http.MethodGet, "/api/v1/profiles/profile-1/required-targets", nil, vars)
	h.GetRequiredTargets(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GetRequiredTargets() status=%d", rr.Code)
	}
}

func TestApprovalsHandlerRejectsInvalidRequests(t *testing.T) {
	h := NewApprovalsHandler(&approvalHandlerFakeRepo{}, func(string, map[string]interface{}) {})
	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
		handle func(*ApprovalsHandler, http.ResponseWriter, *http.Request)
		want   int
	}{
		{"create json", http.MethodPost, "/approvals", "{", (*ApprovalsHandler).Create, http.StatusBadRequest},
		{"create commit", http.MethodPost, "/approvals", CreateApprovalRequest{Platform: "linux"}, (*ApprovalsHandler).Create, http.StatusBadRequest},
		{"create platform", http.MethodPost, "/approvals", CreateApprovalRequest{GitCommitHash: "abc"}, (*ApprovalsHandler).Create, http.StatusBadRequest},
		{"decide json", http.MethodPost, "/decide", "{", (*ApprovalsHandler).Decide, http.StatusBadRequest},
		{"decide choice", http.MethodPost, "/decide", ApprovalDecisionRequest{Decision: "pending", Reviewer: "r"}, (*ApprovalsHandler).Decide, http.StatusBadRequest},
		{"decide reviewer", http.MethodPost, "/decide", ApprovalDecisionRequest{Decision: ApprovalStatusApproved}, (*ApprovalsHandler).Decide, http.StatusBadRequest},
		{"gate commit", http.MethodGet, "/gate", nil, (*ApprovalsHandler).CheckReleaseGate, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := approvalRequest(t, tt.method, tt.path, tt.body, map[string]string{"id": "approval-1"})
			tt.handle(h, rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d body=%s want=%d", rr.Code, rr.Body.String(), tt.want)
			}
		})
	}
	// Repository failures remain server errors and are not presented as success.
	failing := NewApprovalsHandler(&approvalHandlerFakeRepo{err: errors.New("storage unavailable")}, func(string, map[string]interface{}) {})
	req, rr := approvalRequest(t, http.MethodGet, "/approvals/approval-1", nil, map[string]string{"id": "approval-1"})
	failing.Get(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Get() failure status=%d", rr.Code)
	}
}
