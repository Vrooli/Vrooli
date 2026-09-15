package digest

import "testing"

func TestProbabilityBeatsControl(t *testing.T) {
	for _, tc := range []struct{sB,nB,sA,nA int64; want float64}{
		{50,100,50,100,.5}, {90,100,10,100,.999}, {10,100,90,100,.001},
	} { got := ProbabilityBeatsControl(tc.sB,tc.nB,tc.sA,tc.nA); if got < 0 || got > 1 || (tc.want == .5 && (got < .49 || got > .51)) || (tc.want > .9 && got < .9) || (tc.want < .1 && got > .1) { t.Errorf("%+v: %v",tc,got) } }
}
