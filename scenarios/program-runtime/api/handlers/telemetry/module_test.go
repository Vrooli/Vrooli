package telemetry

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 1 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}
