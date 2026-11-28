package gates

import (
	"strings"
	"testing"
)

func TestSelfHostingReportsMissingPolicyAsRunnerError(t *testing.T) {
	result, err := ValidateSelfHosting(Scope{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("ValidateSelfHosting returned error: %v", err)
	}
	if len(result.RunnerError) != 1 {
		t.Fatalf("runner errors = %+v, want one missing-policy finding", result.RunnerError)
	}
	finding := result.RunnerError[0]
	if finding.Code != "catalog.self_hosting_policy_unavailable" {
		t.Errorf("finding code = %q, want catalog.self_hosting_policy_unavailable", finding.Code)
	}
	if finding.AssetID != "__corpus__.self-hosting" {
		t.Errorf("finding asset = %q, want corpus self-hosting", finding.AssetID)
	}
	if !strings.Contains(finding.Message, "self-hosting policy is unavailable") {
		t.Errorf("finding message = %q, want policy-unavailable context", finding.Message)
	}
}
