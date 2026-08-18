package measures

import "testing"

func TestDeclarationsCoverStatefulDomains(t *testing.T) {
	if len(declarations()) != 10 {
		t.Fatalf("declarations=%d", len(declarations()))
	}
}
