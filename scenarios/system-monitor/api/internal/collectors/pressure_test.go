package collectors

import "testing"

func TestParsePSI(t *testing.T) {
	got := parsePSI("some avg10=1.25 avg60=0.50 avg300=0.10 total=12\nfull avg10=0.25 total=4\n")
	if got["some"]["avg10"] != 1.25 || got["full"]["total"] != 4 {
		t.Fatalf("parsePSI = %#v", got)
	}
}

func TestVMStatValue(t *testing.T) {
	if got := vmStatValue("oom_kill 9\noom 11\n", "oom_kill"); got != 9 {
		t.Fatalf("oom_kill = %d, want 9", got)
	}
}
