package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"test-genie/internal/remediation"
)

type remediationStub struct{}

func (remediationStub) Create(context.Context, remediation.Plan, []string, []string, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) Get(context.Context, string) (remediation.Job, error) {
	return remediation.Job{}, remediation.ErrNotFound
}
func (remediationStub) List(context.Context, string, int) ([]remediation.Job, error) { return nil, nil }
func (remediationStub) Cancel(context.Context, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) PrepareLaunch(context.Context, string, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) RetryLaunch(context.Context, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) RecordLaunchFailure(context.Context, string, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) MarkRunning(context.Context, string, remediation.Attribution) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) MarkAgentCompleted(context.Context, string, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) StartVerification(context.Context, string, remediation.Verification) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) ReserveVerification(context.Context, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) SetVerificationRun(context.Context, string, remediation.Verification) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) ReleaseVerificationReservation(context.Context, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) CompleteVerification(context.Context, string, remediation.Verification, remediation.FindingDelta, remediation.RequirementDelta, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}
func (remediationStub) Fail(context.Context, string, string) (remediation.Job, error) {
	return remediation.Job{}, nil
}

type launcherStub struct{}

func (launcherStub) Launch(context.Context, remediation.Job, string) (remediation.Attribution, error) {
	return remediation.Attribution{}, nil
}
func (launcherStub) Cancel(context.Context, remediation.Job) error { return nil }

func TestCreateRemediationJobRejectsGenericAgentPayload(t *testing.T) {
	server := &Server{remediationService: remediationStub{}, remediationLauncher: launcherStub{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/remediation/jobs", strings.NewReader(`{"sourceExecutionId":"00000000-0000-0000-0000-000000000001","findingIds":["afid:1"],"roleRef":"developer","prompts":["write tests"],"networkEnabled":true}`))
	res := httptest.NewRecorder()
	server.handleCreateRemediationJob(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "unsupported remediation payload") {
		t.Fatalf("response = %s", res.Body.String())
	}
}
