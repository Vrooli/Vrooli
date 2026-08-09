package localprincipal

import "testing"

func TestPrincipalIsOpaque(t *testing.T) {
	if Principal("unix:1000").String() != "unix:1000" {
		t.Fatal("principal representation changed")
	}
}
