package config

import "testing"

func TestDefaultsAreTypedAndStable(t *testing.T) {
	got := Defaults()
	if got.HTTPPort != 6333 || got.GRPCPort != 6334 || got.Host != "127.0.0.1" {
		t.Fatalf("unexpected Qdrant defaults: %#v", got)
	}
}

func TestDefaultMessages(t *testing.T) {
	got := DefaultMessages()
	if got.Healthy == "" || got.HealthCheckFailed == "" {
		t.Fatalf("unexpected messages: %#v", got)
	}
}
