package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/invocationreadmodel"
)

func TestDurableCanaryMetricsSeparatesArmsAndComputesMedian(t *testing.T) {
	incumbent, challenger := durableCanaryMetrics([]invocationreadmodel.CanaryRun{
		{Role: "code.default", Arm: "incumbent", Status: "complete", DurationMS: 100, CostUSD: 1},
		{Role: "code.default", Arm: "incumbent", Status: "failed", DurationMS: 300, CostUSD: 3},
		{Role: "code.default", Arm: "challenger", Status: "complete", DurationMS: 80, CostUSD: 0.5},
		{Role: "other", Arm: "challenger", Status: "complete", DurationMS: 1, CostUSD: 1},
	}, "code.default")
	if incumbent.Count != 2 || incumbent.SuccessRate != 0.5 || incumbent.MedianMS != 200 || incumbent.CostPerRun != 2 {
		t.Fatalf("incumbent=%+v", incumbent)
	}
	if challenger.Count != 1 || challenger.MedianMS != 80 || challenger.CostPerRun != 0.5 {
		t.Fatalf("challenger=%+v", challenger)
	}
}

func TestCanaryCompareRejectsNonExecutedRunClasses(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/model-policy/canary/compare", strings.NewReader(`{"included_run_classes":["interactive"]}`))
	resp := httptest.NewRecorder()
	NewCanaryHandler().Compare(resp, req)
	if resp.Code != 400 {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
