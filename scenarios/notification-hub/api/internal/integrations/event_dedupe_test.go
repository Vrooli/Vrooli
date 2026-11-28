package integrations

import (
	"encoding/json"
	"testing"
)

func TestEventDedupeKeyKeepsAlertOpenAndRecoveryDistinct(t *testing.T) {
	open := eventDedupeKey("scenario-to-cloud.deployment.alert.v1", json.RawMessage(`{"incident_id":"dep-1:certificate_expiring","status":"open"}`))
	repeatOpen := eventDedupeKey("scenario-to-cloud.deployment.alert.v1", json.RawMessage(`{"incident_id":"dep-1:certificate_expiring","status":"open"}`))
	resolved := eventDedupeKey("scenario-to-cloud.deployment.alert.v1", json.RawMessage(`{"incident_id":"dep-1:certificate_expiring","status":"resolved"}`))
	if open != "scenario-to-cloud.deployment.alert.v1:dep-1:certificate_expiring:open" {
		t.Fatalf("open key = %q", open)
	}
	if repeatOpen != open {
		t.Fatalf("repeat open key %q differs from %q", repeatOpen, open)
	}
	if resolved == open {
		t.Fatal("recovery event would be deduplicated with open event")
	}
}
