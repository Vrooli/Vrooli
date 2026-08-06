package programs

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 4 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}
