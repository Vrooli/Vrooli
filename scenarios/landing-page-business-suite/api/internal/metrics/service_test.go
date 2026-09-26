package metrics

import "testing"

func TestValidateEventRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []Event{
		{VariantSlug: "control", SessionID: "session"},
		{EventType: "unknown", VariantSlug: "control", SessionID: "session"},
		{EventType: "click", SessionID: "session"},
		{EventType: "click", VariantSlug: "control"},
	}
	for _, event := range cases {
		if err := ValidateEvent(event); err == nil {
			t.Fatalf("ValidateEvent(%+v) unexpectedly succeeded", event)
		}
	}
}

// [REQ:METRIC-EVENTS] The ingestion boundary accepts every event emitted by the landing client.
func TestValidateEventAcceptsRequiredEventTypes(t *testing.T) {
	t.Parallel()
	for _, eventType := range []string{"page_view", "scroll_depth", "click", "form_submit", "conversion"} {
		if err := ValidateEvent(Event{EventType: eventType, VariantSlug: "control", SessionID: "session"}); err != nil {
			t.Fatalf("ValidateEvent(%q) error = %v", eventType, err)
		}
	}
}
