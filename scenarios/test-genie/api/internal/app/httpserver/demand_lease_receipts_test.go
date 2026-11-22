package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/demand"
	appruns "test-genie/internal/app/runs"
	"test-genie/internal/orchestrator/targetruntime"
)

type demandReceiptURLResolver struct{}

func (demandReceiptURLResolver) ResolveScenarioURL(context.Context, string, string) (string, error) {
	return "http://127.0.0.1:12345", nil
}

type demandReceiptLeaseClient struct{}

func (demandReceiptLeaseClient) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	return demand.Lease{LeaseID: req.LeaseID, Scenario: req.Scenario, ConsumerID: req.ConsumerID, RequestID: req.RequestID, Kind: req.Kind, Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (demandReceiptLeaseClient) Renew(_ context.Context, id string, _ time.Duration) (demand.Lease, error) {
	return demand.Lease{LeaseID: id, Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (demandReceiptLeaseClient) Release(_ context.Context, id, _ string) (demand.Lease, error) {
	return demand.Lease{LeaseID: id, Status: "released"}, nil
}

func TestVerifyDemandLeaseReceiptEndpointUsesOnlyOpaqueReceipt(t *testing.T) {
	const runID = "audio-soak-run-1"
	manager := targetruntime.New("audio-tools", filepath.Join(t.TempDir(), "audio-tools")).
		WithURLResolver(demandReceiptURLResolver{}).
		WithDemandLease(demandReceiptLeaseClient{}, "test-genie:job:"+runID, runID)
	lease, err := manager.EnsureRunning(context.Background(), targetruntime.Needs{API: true}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Cleanup(context.Background(), lease, io.Discard) })
	receipt, err := lease.DemandLeaseReceipt()
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{runsService: appruns.NewService(t.TempDir(), nil, nil, nil)}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs/demand-lease-receipts/verify?target=audio-tools", nil)
	request.Header.Set("X-Test-Genie-Lease-Receipt", receipt.ReceiptID)
	response := httptest.NewRecorder()
	s.handleVerifyDemandLeaseReceipt(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", response.Code, response.Body.String())
	}
	var body demandLeaseReceiptVerification
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Verified || body.ReceiptID != receipt.ReceiptID || body.RunID != runID || body.Target != "audio-tools" || body.LeaseID != receipt.LeaseID {
		t.Fatalf("verification response = %+v", body)
	}

	missing := httptest.NewRequest(http.MethodPost, "/api/v1/runs/demand-lease-receipts/verify?target=audio-tools", nil)
	missingResponse := httptest.NewRecorder()
	s.handleVerifyDemandLeaseReceipt(missingResponse, missing)
	if missingResponse.Code != http.StatusUnauthorized {
		t.Fatalf("missing receipt status=%d, want 401", missingResponse.Code)
	}

	wrongScope := httptest.NewRequest(http.MethodPost, "/api/v1/runs/demand-lease-receipts/verify?target=other", nil)
	wrongScope.Header.Set("X-Test-Genie-Lease-Receipt", receipt.ReceiptID)
	wrongScopeResponse := httptest.NewRecorder()
	s.handleVerifyDemandLeaseReceipt(wrongScopeResponse, wrongScope)
	if wrongScopeResponse.Code != http.StatusBadRequest {
		t.Fatalf("wrong target status=%d, want 400", wrongScopeResponse.Code)
	}

	unknown := httptest.NewRequest(http.MethodPost, "/api/v1/runs/demand-lease-receipts/verify?target=audio-tools", nil)
	unknown.Header.Set("X-Test-Genie-Lease-Receipt", "tglease_client-chosen")
	unknownResponse := httptest.NewRecorder()
	s.handleVerifyDemandLeaseReceipt(unknownResponse, unknown)
	if unknownResponse.Code != http.StatusForbidden {
		t.Fatalf("unknown receipt status=%d, want 403", unknownResponse.Code)
	}

	if err := manager.Cleanup(context.Background(), lease, io.Discard); err != nil {
		t.Fatal(err)
	}
	afterRun := httptest.NewRequest(http.MethodPost, "/api/v1/runs/demand-lease-receipts/verify?target=audio-tools", nil)
	afterRun.Header.Set("X-Test-Genie-Lease-Receipt", receipt.ReceiptID)
	afterRunResponse := httptest.NewRecorder()
	s.handleVerifyDemandLeaseReceipt(afterRunResponse, afterRun)
	if afterRunResponse.Code != http.StatusForbidden {
		t.Fatalf("post-run receipt status=%d, want 403", afterRunResponse.Code)
	}
}
