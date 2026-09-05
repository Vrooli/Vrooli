package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/orchestration"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

type maintenanceServiceFake struct {
	request orchestration.PurgeRequest
	result  *orchestration.PurgeResult
	calls   int
}

func (f *maintenanceServiceFake) PurgeData(_ context.Context, request orchestration.PurgeRequest) (*orchestration.PurgeResult, error) {
	f.calls++
	f.request = request
	return f.result, nil
}

func TestPurgeDataHandlerValidatesRequestBeforeCallingService(t *testing.T) {
	fake := &maintenanceServiceFake{}
	h := New(orchestration.HandlerServices{MaintenanceService: fake})
	for _, request := range []*apipb.PurgeDataRequest{
		{Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_RUNS}},
		{Pattern: "remove"},
		{Pattern: "remove", Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_UNSPECIFIED}},
	} {
		rr := httptest.NewRecorder()
		h.PurgeData(rr, httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/purge", bytes.NewReader(encodeProtoJSON(t, request))))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("request=%+v status=%d body=%s", request, rr.Code, rr.Body.String())
		}
	}
	if fake.calls != 0 {
		t.Fatalf("service calls=%d, want validation to reject before service", fake.calls)
	}
}

func TestPurgeDataHandlerMapsTargetsAndCounts(t *testing.T) {
	fake := &maintenanceServiceFake{result: &orchestration.PurgeResult{
		Matched: orchestration.PurgeCounts{Profiles: 2, Tasks: 3, Runs: 4},
		Deleted: orchestration.PurgeCounts{Profiles: 1, Tasks: 2, Runs: 3},
		DryRun:  true,
	}}
	h := New(orchestration.HandlerServices{MaintenanceService: fake})
	request := &apipb.PurgeDataRequest{Pattern: "remove", Targets: []apipb.PurgeTarget{apipb.PurgeTarget_PURGE_TARGET_PROFILES, apipb.PurgeTarget_PURGE_TARGET_TASKS, apipb.PurgeTarget_PURGE_TARGET_RUNS}, DryRun: true}
	rr := httptest.NewRecorder()
	h.PurgeData(rr, httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/purge", bytes.NewReader(encodeProtoJSON(t, request))))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if fake.calls != 1 || fake.request.Pattern != "remove" || !fake.request.DryRun || len(fake.request.Targets) != 3 {
		t.Fatalf("service request=%+v calls=%d", fake.request, fake.calls)
	}
	if fake.request.Targets[0] != orchestration.PurgeTargetProfiles || fake.request.Targets[1] != orchestration.PurgeTargetTasks || fake.request.Targets[2] != orchestration.PurgeTargetRuns {
		t.Fatalf("mapped targets=%v", fake.request.Targets)
	}
	var response apipb.PurgeDataResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	if !response.GetDryRun() || response.GetMatched().GetProfiles() != 2 || response.GetMatched().GetRuns() != 4 || response.GetDeleted().GetTasks() != 2 {
		t.Fatalf("response=%+v", &response)
	}
}
