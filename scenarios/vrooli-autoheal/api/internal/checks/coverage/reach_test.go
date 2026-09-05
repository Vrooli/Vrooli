package coverage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

func TestEvaluateRemediationReachReportsCriticalShortfall(t *testing.T) {
	result := EvaluateRemediationReach([]CriticalFinding{{ID: "incident-1", Check: "host-kernel-module-drift"}, {ID: "incident-2", Check: "host-package-state"}}, map[string]int{"host-kernel-module-drift": 1})
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	if got := result.Details["missingFindingIds"]; fmt.Sprint(got) != "[incident-2]" {
		t.Fatalf("missing findings = %v", got)
	}
	if result.Details["coveredFindings"] != 1 {
		t.Fatalf("covered findings = %v, want 1", result.Details["coveredFindings"])
	}
}

func TestEvaluateRemediationReachIsOKWhenThereAreNoCriticalFindings(t *testing.T) {
	if got := EvaluateRemediationReach(nil, nil).Status; got != checks.StatusOK {
		t.Fatalf("status = %s, want ok", got)
	}
}

func TestEvaluateDeliveryReachReportsGapWithoutAttempt(t *testing.T) {
	result := EvaluateDeliveryReach(DeliverySnapshot{Incidents: []CriticalFinding{{ID: "incident-1"}, {ID: "incident-2"}}, Attempts: []DeliveryAttempt{{IncidentID: "incident-1", Outcome: "unroutable", Channel: "none"}}})
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	// Neither incident reached a person: one was never attempted, one was
	// unroutable. Both are missing, and each names its cause.
	if got := result.Details["missingIncidentIds"]; fmt.Sprint(got) != "[incident-1 incident-2]" {
		t.Fatalf("missing incidents = %v", got)
	}
	if got := result.Details["undeliveredOutcomes"].(map[string]string); got["incident-1"] != "unroutable" || got["incident-2"] != "no attempt" {
		t.Fatalf("undelivered outcomes = %v", got)
	}
}

// [REQ:STORM-003] An 'unroutable' or 'failed' attempt proves intake ran, not
// that a person saw anything; only a delivered attempt counts as reach.
func TestReachCountsDeliveredAttemptsOnly(t *testing.T) {
	result := EvaluateDeliveryReach(DeliverySnapshot{
		Incidents: []CriticalFinding{{ID: "incident-1"}, {ID: "incident-2"}, {ID: "incident-3"}},
		Attempts: []DeliveryAttempt{
			{IncidentID: "incident-1", Outcome: "unroutable", Channel: "none"},
			{IncidentID: "incident-2", Outcome: "failed", Channel: "linux_notification"},
			{IncidentID: "incident-2", Outcome: OutcomeDelivered, Channel: "linux_notification"},
			{IncidentID: "incident-3", Outcome: "failed", Channel: "email"},
		},
	})
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s, want critical", result.Status)
	}
	if got := fmt.Sprint(result.Details["missingIncidentIds"]); got != "[incident-1 incident-3]" {
		t.Fatalf("missing incidents = %v", got)
	}
	if got := result.Details["undeliveredOutcomes"].(map[string]string); got["incident-1"] != "unroutable" || got["incident-3"] != "failed" {
		t.Fatalf("undelivered outcomes = %v", got)
	}
	if got := result.Details["deliveredIncidents"]; got != 1 {
		t.Fatalf("deliveredIncidents = %v, want 1", got)
	}
}

func TestReachReadsRealDeliveryProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/integrations/deliveries" || r.URL.Query().Get("prefix") != "incident." {
			t.Errorf("unexpected request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"notifications":[
		 {"notification_id":"n1","dedupe_key":"incident.opened.v1:inc-1","attempts":[{"channel":"none","outcome":"unroutable"}]},
		 {"notification_id":"n2","dedupe_key":"incident.opened.v1:inc-2","attempts":[{"channel":"linux_notification","outcome":"delivered"}]},
		 {"notification_id":"n3","dedupe_key":"other","attempts":[{"channel":"email","outcome":"delivered"}]}]}`))
	}))
	defer server.Close()
	lister := fakeIncidentLister{incidents: []incidents.Incident{{ID: "inc-1", Title: "one"}, {ID: "inc-2", Title: "two"}}}
	reader := NotificationHubDeliveryReader(lister, func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	snapshot, err := reader(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Incidents) != 2 || len(snapshot.Attempts) != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	result := EvaluateDeliveryReach(snapshot)
	if result.Status != checks.StatusCritical || fmt.Sprint(result.Details["missingIncidentIds"]) != "[inc-1]" {
		t.Fatalf("result = %+v", result)
	}
	down := NotificationHubDeliveryReader(lister, func(context.Context) (string, error) { return "http://127.0.0.1:1", nil }, &http.Client{Timeout: time.Second})
	if _, err := down(context.Background()); err == nil {
		t.Fatal("an unreachable hub must be an error, not an empty projection")
	}
	if id := IncidentIDFromDedupeKey("incident.opened.v1:inc-9"); id != "inc-9" {
		t.Fatalf("IncidentIDFromDedupeKey = %q", id)
	}
}

type fakeIncidentLister struct{ incidents []incidents.Incident }

func (f fakeIncidentLister) ListIncidents(context.Context, incidents.ListFilters) (*incidents.ListResponse, error) {
	return &incidents.ListResponse{Incidents: f.incidents, Total: len(f.incidents)}, nil
}

func TestDeliveryReachIsUnreadableWhenProjectionCannotBeRead(t *testing.T) {
	check := NewDeliveryReachCheck(func(context.Context) (DeliverySnapshot, error) {
		return DeliverySnapshot{}, errors.New("notification hub unavailable")
	})
	result := check.Run(context.Background())
	if result.Status != checks.StatusUndetermined {
		t.Fatalf("status = %s, want undetermined", result.Status)
	}
	if result.Details["readable"] != false {
		t.Fatalf("readable = %v, want false", result.Details["readable"])
	}
}
