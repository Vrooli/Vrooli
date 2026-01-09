package metrics

import "testing"

func TestGetSingleton(t *testing.T) {
	first := Get()
	if first == nil {
		t.Fatal("expected metrics instance, got nil")
	}
	second := Get()
	if first != second {
		t.Error("expected Get() to return singleton instance")
	}
	if first.RunsTotal == nil || first.HTTPRequestsTotal == nil {
		t.Error("expected core metrics to be initialized")
	}
}

func TestHandlerNotNil(t *testing.T) {
	if Handler() == nil {
		t.Error("expected metrics handler, got nil")
	}
}
