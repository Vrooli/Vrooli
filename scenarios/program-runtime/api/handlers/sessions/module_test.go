package sessions

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 6 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}
