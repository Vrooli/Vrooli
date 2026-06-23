package inventory

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "inventory" {
		t.Fatalf("module name = %q, want inventory", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected inventory endpoints")
	}
}

func TestSampleDeviceDoesNotInventNetworkAddress(t *testing.T) {
	if got := sampleDevice().GetIpAddress(); got != "" {
		t.Fatalf("sample device IP = %q, want empty until discovery is implemented", got)
	}
}
