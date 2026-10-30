package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	capsH "web-console/handlers/capabilities"
)

func TestLocalAgentUsageKeepsEvidenceAndNeverClaimsVerified(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name, match, observed, percent, reset string
		wantMatch                             string
		wantWindows                           int
	}{
		{"zero is a real value", "likely", now.Format(time.RFC3339), "0", now.Add(time.Hour).Format(time.RFC3339), "likely", 1},
		{"stale", "likely", now.Add(-time.Hour).Format(time.RFC3339), "30", now.Add(time.Hour).Format(time.RFC3339), "unavailable", 0},
		{"past reset", "likely", now.Format(time.RFC3339), "30", now.Add(-time.Minute).Format(time.RFC3339), "unavailable", 0},
		{"absent percent", "likely", now.Format(time.RFC3339), "null", "", "unavailable", 0},
		{"out of range", "likely", now.Format(time.RFC3339), "101", "", "unavailable", 0},
		{"unproven verified claim", "verified", now.Format(time.RFC3339), "30", "", "unavailable", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/pricing/live-quota" || r.URL.Query().Get("agent") != "codex" {
					t.Errorf("unexpected request %s", r.URL.String())
				}
				fmt.Fprintf(w, `{"match":%q,"reason":"Default account","source":"provider","observedAt":%q,"windows":[{"pool":"codex","windowMinutes":10080,"usedPercent":%s,"resetsAt":%q}]}`, tc.match, tc.observed, tc.percent, tc.reset)
			}))
			defer server.Close()
			base := capsH.AgentUsageResult{TargetID: "local", AgentID: "codex", Match: "unavailable"}
			got := fetchLocalAgentUsage(context.Background(), server.Client(), server.URL, base)
			if got.Match != tc.wantMatch || len(got.Windows) != tc.wantWindows || got.TargetID != "local" || got.AgentID != "codex" {
				t.Fatalf("got %+v", got)
			}
			if tc.wantMatch == "likely" && (got.Windows[0].UsedPercent != 0 || got.Reason == "Default account") {
				t.Fatalf("missing evidence or uncertainty: %+v", got)
			}
		})
	}
}
