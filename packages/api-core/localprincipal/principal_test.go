package localprincipal

import "testing"

func TestUnixPrincipalIsOpaqueAndStable(t *testing.T) {
	got := UnixUID(1000)
	if got.String() != "unix:1000" {
		t.Fatalf("principal = %q", got)
	}
	uid, err := ParseUnixUID(got)
	if err != nil || uid != 1000 {
		t.Fatalf("parse = %d, %v", uid, err)
	}
	if _, err := ParseUnixUID(Principal("1000")); err == nil {
		t.Fatal("unqualified principal accepted")
	}
}
