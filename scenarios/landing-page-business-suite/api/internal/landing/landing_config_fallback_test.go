package landing

import (
	"errors"
	"testing"
)

func TestFallbackWithReasonEmitsStructuredEvent(t *testing.T) {
	service := &LandingConfigService{fallbackProvider: defaultFallbackProvider}
	var message string
	var fields map[string]interface{}
	service.UseEventLogger(func(gotMessage string, gotFields map[string]interface{}) {
		message = gotMessage
		fields = gotFields
	})

	response, err := service.fallbackWithReason("pricing_fetch_failed", errors.New("database unavailable"), map[string]interface{}{"variant_slug": "control"})
	if err != nil {
		t.Fatalf("fallbackWithReason returned error: %v", err)
	}
	if response == nil || !response.Fallback {
		t.Fatalf("expected marked fallback response, got %+v", response)
	}
	if message != "landing_config_fallback" {
		t.Fatalf("expected structured event name, got %q", message)
	}
	if fields["reason"] != "pricing_fetch_failed" || fields["error"] != "database unavailable" || fields["variant_slug"] != "control" {
		t.Fatalf("unexpected fallback event fields: %+v", fields)
	}
}
