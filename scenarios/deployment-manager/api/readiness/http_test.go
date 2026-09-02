package readiness

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerReturnsAggregatedVerdict(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "check", Title: "Check", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "must pass"}}}
	body := []byte(`{"scenario":"demo","commit":"abc","signals":[{"item_id":"check","status":"passed","source":"test-genie","observed_at":"2026-09-01T00:00:00Z"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/verdict", bytes.NewReader(body))
	res := httptest.NewRecorder()
	Handler(checklist).ServeHTTP(res, req)
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"approved":true`)) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	_ = checked
}

func TestHandlerAdaptsCrossOSVerdictIntoReadiness(t *testing.T) {
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "ramp-evidence-complete", Title: "Ramp", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "must pass"}}}
	body := []byte(`{"scenario":"demo","commit":"abc","cross_os_verdict":{"production_ready":true,"verdict":"passed","gate_id":"gate-1"}}`)
	res := httptest.NewRecorder()
	Handler(checklist).ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/readiness/verdict", bytes.NewReader(body)))
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"approved":true`)) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
