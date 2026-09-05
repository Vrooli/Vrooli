package infra

import (
	"context"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func TestCredentialRecoveryCheckReportsStaleCoverageWithoutExporting(t *testing.T) {
	called := 0
	check := NewCredentialRecoveryCheck(WithCredentialDoctor(func(context.Context) (credentialclient.DoctorResponse, error) {
		called++
		return credentialclient.DoctorResponse{
			Provider: credentialclient.ProviderDiagnosis{Backend: "libsecret", Condition: "available"},
			Recovery: credentialclient.RecoveryStatus{ReceiptExists: true, EntryCount: 2, Uncovered: []string{"vrooli/example:api-key"}},
		}, nil
	}))
	result := check.Run(context.Background())
	if called != 1 {
		t.Fatalf("doctor calls = %d, want one report-only check", called)
	}
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning", result.Status)
	}
	if got := result.Details["uncovered"].([]string); len(got) != 1 || got[0] != "vrooli/example:api-key" {
		t.Fatalf("uncovered = %#v", got)
	}
}
