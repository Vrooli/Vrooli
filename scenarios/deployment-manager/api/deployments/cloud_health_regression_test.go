package deployments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCloudHealthConsumerRejectsHTTP200FalseGreen [REQ:STC-P0-034] is the
// P16 step-1 regression, kept after the wire-contract migration. HTTP 200 is
// transport success, never deployment health: a report that is unhealthy,
// or a bare "ok" status with no deployment identity, must not be accepted.
//
// Against the pre-P16 consumer (CheckLPBSHealth on the hardcoded
// landing-page-business-suite path) the "ok without identity" case passed
// as healthy; see evidence/P16-health-truth.md for the captured run.
func TestCloudHealthConsumerRejectsHTTP200FalseGreen(t *testing.T) {
	const deploymentID = "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11"
	cases := map[string]string{
		"unhealthy observation": `{"schema_version":"1","observation":{"deployment_id":"` + deploymentID + `","status":"HEALTH_STATUS_UNHEALTHY","freshness":"FRESHNESS_CURRENT","observed_at":"2026-09-09T00:00:00Z"}}`,
		"ok without identity":   `{"status":"ok"}`,
		"legacy healthy report": `{"ok":true,"health":"healthy","deployment_id":"` + deploymentID + `"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL}
			result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: deploymentID})
			if err != nil {
				t.Fatalf("health check error = %v", err)
			}
			if result.Healthy {
				t.Fatalf("HTTP 200 body %s was accepted as healthy: %#v", body, result)
			}
			if result.ReasonCode == "" {
				t.Fatalf("refusal carries no reason: %#v", result)
			}
		})
	}
}
