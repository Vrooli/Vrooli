package main

import "testing"

func TestAcceptanceStartupSelectionRefusesBeforeProtectedAcquisition(t *testing.T) {
	for _, phase := range []string{"", "off"} {
		t.Setenv("VROOLI_FINITE_OWNER_PHASE", phase)
		owner, e := loadProtectedFiniteReadPublication()
		if e != nil || owner != nil {
			t.Fatal("inert acceptance startup changed")
		}
	}
	for _, test := range []struct{ phase, enabled string }{{"live", "true"}, {"anonymous", "false"}, {"commission", "true"}, {"commission", "invalid"}} {
		t.Setenv("VROOLI_FINITE_OWNER_PHASE", test.phase)
		t.Setenv("VROOLI_FINITE_ENABLED", test.enabled)
		if owner, e := loadProtectedFiniteReadPublication(); e == nil || owner != nil {
			t.Fatal("unsupported acceptance selection admitted")
		}
	}
}
