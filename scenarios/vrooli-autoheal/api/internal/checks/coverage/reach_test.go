package coverage

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
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
	if got := result.Details["missingIncidentIds"]; fmt.Sprint(got) != "[incident-2]" {
		t.Fatalf("missing incidents = %v", got)
	}
}

func TestEvaluateDeliveryReachAcceptsAnUnroutableAttemptAsIntakeEvidence(t *testing.T) {
	result := EvaluateDeliveryReach(DeliverySnapshot{Incidents: []CriticalFinding{{ID: "incident-1"}}, Attempts: []DeliveryAttempt{{IncidentID: "incident-1", Outcome: "unroutable"}}})
	if result.Status != checks.StatusOK {
		t.Fatalf("status = %s, want ok", result.Status)
	}
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
