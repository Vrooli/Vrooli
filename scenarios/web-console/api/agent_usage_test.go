package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	capsH "web-console/handlers/capabilities"
)

func TestAgentUsageResolvesDesktopFeaturePerCall(t *testing.T) {
	token := filepath.Join(t.TempDir(), "runtime-token")
	if err := os.WriteFile(token, []byte("usage-test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN_FILE", token)
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pricing/live-quota" || r.URL.Query().Get("agent") != "codex" {
			t.Errorf("unexpected owner request %s", r.URL)
		}
		fmt.Fprintf(w, `{"match":"likely","source":"Codex app-server","observedAt":%q,"windows":[{"pool":"codex","windowMinutes":300,"usedPercent":27}]}`, time.Now().UTC().Format(time.RFC3339))
	}))
	defer owner.Close()
	selected := true
	lookups := 0
	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lookups++
		if r.URL.Path != "/features" || r.URL.Query().Get("scenario") != "agent-manager" || r.URL.Query().Get("feature") != "usage.live" || r.Header.Get("Authorization") != "Bearer usage-test-token" {
			t.Errorf("unexpected feature lookup %s", r.URL)
		}
		if !selected {
			http.Error(w, "component removed", 404)
			return
		}
		fmt.Fprintf(w, `{"url":%q,"source":"component","component_id":"agent-manager-usage"}`, owner.URL)
	}))
	defer runtime.Close()
	t.Setenv("VROOLI_RUNTIME_URL", runtime.URL)
	server := &Server{}
	got, err := server.readAgentUsage(context.Background(), "local", "codex")
	if err != nil || got.Match != "likely" || len(got.Windows) != 1 || got.Windows[0].UsedPercent != 27 {
		t.Fatalf("feature usage: %+v %v", got, err)
	}
	selected = false
	got, err = server.readAgentUsage(context.Background(), "local", "codex")
	if err != nil || got.Match != "unavailable" || len(got.Windows) != 0 || lookups != 2 {
		t.Fatalf("removed feature: %+v %v lookups=%d", got, err, lookups)
	}
}

func TestLocalAgentUsageKeepsEvidenceAndNeverClaimsVerified(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name, match, observed, percent, reset string
		wantMatch                             string
		wantWindows                           int
	}{
		{"zero is a real value", "likely", now.Format(time.RFC3339), "0", now.Add(time.Hour).Format(time.RFC3339), "likely", 1},
		{"quiet session within the owner's bound", "likely", now.Add(-time.Hour).Format(time.RFC3339), "30", now.Add(time.Hour).Format(time.RFC3339), "likely", 1},
		{"past the owner's bound", "likely", now.Add(-7 * time.Hour).Format(time.RFC3339), "30", now.Add(time.Hour).Format(time.RFC3339), "unavailable", 0},
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
			if tc.wantMatch == "likely" && (got.Reason == "Default account" || got.Windows[0].ObservedAt != tc.observed) {
				t.Fatalf("missing evidence or uncertainty: %+v", got)
			}
			if tc.wantMatch == "unavailable" && got.Reason == "" {
				t.Fatalf("an unavailable reading must say why: %+v", got)
			}
		})
	}
}

func usageWindow(percent float64, observedAt, resetsAt string) localUsageWindow {
	return localUsageWindow{Pool: "subscription", WindowMinutes: 300, UsedPercent: &percent, ObservedAt: observedAt, ResetsAt: resetsAt}
}

func TestLocalUsageJudgesEachWindowByItsOwnReading(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	at := func(age time.Duration) string { return now.Add(-age).Format(time.RFC3339) }
	base := capsH.AgentUsageResult{TargetID: "local", AgentID: "claude", Match: "unavailable"}

	fresh, old := usageWindow(0, at(2*time.Hour), ""), usageWindow(86, at(30*time.Minute), "")
	old.WindowMinutes = 10080
	aged := usageWindow(50, at(7*time.Hour), "")
	aged.WindowMinutes = 60
	got := localUsageResult(base, "Sample.", "statusline", at(30*time.Minute), []localUsageWindow{fresh, old, aged}, now)
	if got.Match != "likely" || len(got.Windows) != 2 {
		t.Fatalf("the current windows must survive one aged window: %+v", got)
	}
	if got.Windows[0].ObservedAt != at(2*time.Hour) || got.Windows[1].ObservedAt != at(30*time.Minute) {
		t.Fatalf("each window keeps its own age: %+v", got.Windows)
	}
	if got.ObservedAt != at(30*time.Minute) {
		t.Fatalf("the result carries the newest reading: %q", got.ObservedAt)
	}

	inherited := usageWindow(10, "", "")
	if got := localUsageResult(base, "", "app-server", at(time.Minute), []localUsageWindow{inherited}, now); got.Match != "likely" || got.Windows[0].ObservedAt != at(time.Minute) {
		t.Fatalf("a window without its own time is as old as the response: %+v", got)
	}

	for name, tc := range map[string]struct {
		windows []localUsageWindow
		reason  string
	}{
		"aged":        {[]localUsageWindow{usageWindow(1, at(7*time.Hour), "")}, "The latest usage reading is 7 hours old"},
		"reset":       {[]localUsageWindow{usageWindow(1, at(time.Hour), at(time.Minute))}, "The usage window has reset and no new reading has arrived yet"},
		"future":      {[]localUsageWindow{usageWindow(1, at(-time.Hour), "")}, "No current usage windows are available"},
		"no windows":  {nil, "No current usage windows are available"},
		"bad reading": {[]localUsageWindow{usageWindow(101, at(time.Minute), "")}, "No current usage windows are available"},
	} {
		t.Run(name, func(t *testing.T) {
			got := localUsageResult(base, "", "", at(time.Minute), tc.windows, now)
			if got.Match != "unavailable" || got.Reason != tc.reason || len(got.Windows) != 0 {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
