package resourcecli

import "testing"

func TestHealthOnlyStatusIsExplicitAndIndependentOfProbeDepth(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		health, fast bool
	}{
		{nil, false, true},
		{[]string{"ollama"}, false, false},
		{[]string{"--health-only"}, true, true},
		{[]string{"ollama", "--health-only", "--no-fast"}, true, false},
	} {
		req, err := ParseStatusRequest(tc.args)
		if err != nil || req.HealthOnly != tc.health || req.Fast != tc.fast {
			t.Fatalf("args=%v request=%+v err=%v", tc.args, req, err)
		}
	}
}
