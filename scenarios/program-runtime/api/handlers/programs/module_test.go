package programs

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 9 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

// TestWaitEndpointIsDeclared pins the block-once primitive to the declared
// surface. It is named separately from the count so a future change that
// removes it fails with a reason rather than an arithmetic mismatch.
func TestWaitEndpointIsDeclared(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "programs_wait" {
			return
		}
	}
	t.Fatal("programs_wait is not declared; callers would fall back to polling GetProgram")
}
