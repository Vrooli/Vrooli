package bindings

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 3 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}
