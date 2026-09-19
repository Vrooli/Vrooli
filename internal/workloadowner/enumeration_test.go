package workloadowner

import "testing"

func TestParseServiceUnits(t *testing.T) {
	got := ParseServiceUnits([]byte("kubelet.service loaded active running Kubelet\nold.service loaded inactive dead Old\n"))
	if len(got) != 2 || !got[0].Running || got[1].Running {
		t.Fatalf("units=%+v", got)
	}
}
