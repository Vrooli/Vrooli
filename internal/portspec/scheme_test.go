package portspec

import "testing"

func TestCanonicalBand(t *testing.T) {
	cases := []struct {
		port int
		want CanonicalRole
		ok   bool
	}{
		{0, RoleUnknown, false},
		{80, RoleUnknown, false},
		{14999, RoleUnknown, false},
		{15000, RoleAPI, true},
		{18000, RoleAPI, true},
		{19999, RoleAPI, true},
		{20000, RoleUI, true},
		{21234, RoleUI, true},
		{24999, RoleUI, true},
		{25000, RoleWS, true},
		{29999, RoleWS, true},
		{30000, RoleHeadroom, true},
		{32767, RoleHeadroom, true},
		{32768, RoleUnknown, false},
		{36234, RoleUnknown, false},
	}
	for _, c := range cases {
		got, ok := CanonicalBand(c.port)
		if got != c.want || ok != c.ok {
			t.Errorf("CanonicalBand(%d) = (%s, %v), want (%s, %v)", c.port, got, ok, c.want, c.ok)
		}
	}
}

func TestIsAboveCanonicalMax(t *testing.T) {
	if IsAboveCanonicalMax(CanonicalMax) {
		t.Errorf("CanonicalMax (%d) should be inclusive", CanonicalMax)
	}
	if !IsAboveCanonicalMax(CanonicalMax + 1) {
		t.Errorf("CanonicalMax+1 (%d) should be above canonical max", CanonicalMax+1)
	}
}

func TestCanonicalBandsAreContiguousAndDistinct(t *testing.T) {
	// Guard against future edits that accidentally overlap the bands.
	if APIRangeEnd+1 != UIRangeStart {
		t.Errorf("API/UI bands not contiguous: APIRangeEnd=%d UIRangeStart=%d", APIRangeEnd, UIRangeStart)
	}
	if UIRangeEnd+1 != WSRangeStart {
		t.Errorf("UI/WS bands not contiguous: UIRangeEnd=%d WSRangeStart=%d", UIRangeEnd, WSRangeStart)
	}
	if WSRangeEnd+1 != ReservedHeadroomStart {
		t.Errorf("WS/Headroom bands not contiguous: WSRangeEnd=%d ReservedHeadroomStart=%d", WSRangeEnd, ReservedHeadroomStart)
	}
	if ReservedHeadroomEnd != CanonicalMax {
		t.Errorf("Headroom must reach CanonicalMax: ReservedHeadroomEnd=%d CanonicalMax=%d", ReservedHeadroomEnd, CanonicalMax)
	}
}
