package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

const validReceiptBody = `{"eventId":"receipt-provenance-1","sourceScenario":"agent-manager","targetScenario":"plan-manager","eventType":"vrooli.receipt.observed.v1","correlationId":"run-1","metadata":{"operation":"plans.create","outcome":"success","retention_days":"30"}}`

// REQ: a receipt may be stored only after the production provenance boundary
// verifies an Agent Manager token with a run claim.
func TestReceiptIngestionRequiresVerifiedProvenance(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.policyStore.CreateReceiptProjection(t.Context(), policy.ReceiptProjectionRule{
		SourceScenario: "agent-manager", TargetScenario: "plan-manager", OperationPattern: "plans.create",
		ResponseFields: []string{"plan_id"}, MaxBytes: 1024, SamplePerTenK: 10000, RetentionDays: 30, Enabled: true,
	}); err != nil {
		t.Fatalf("create receipt projection: %v", err)
	}
	handler := provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token != "valid" {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	}))(s.routes())

	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "absent identity", want: http.StatusUnauthorized},
		{name: "invalid identity", token: "invalid", want: http.StatusUnauthorized},
		{name: "verified identity", token: "valid", want: http.StatusAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(validReceiptBody))
			if tc.token != "" {
				req.Header.Set(cliutil.HeaderAgentIdentityToken, tc.token)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestVerifiedReceiptRejectsProjectionOutsideCentralAllowList(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.policyStore.CreateReceiptProjection(t.Context(), policy.ReceiptProjectionRule{
		SourceScenario: "agent-manager", TargetScenario: "plan-manager", OperationPattern: "plans.create",
		ResponseFields: []string{"plan_id"}, MaxBytes: 1024, SamplePerTenK: 10000, RetentionDays: 30, Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	handler := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	}))(s.routes())
	body := strings.Replace(validReceiptBody, `"outcome":"success"`, `"outcome":"success","safe_projection":"{\"unapproved\":true}"`, 1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "valid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
