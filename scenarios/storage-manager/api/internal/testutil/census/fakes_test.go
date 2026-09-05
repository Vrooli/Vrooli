package census

import "testing"

func TestDeviceProbeFakeCanBeSeeded(t *testing.T) {
	probe := DeviceProbe{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}
	got, err := probe.Probe("/synthetic")
	if err != nil || got.TotalBytes != probe.TotalBytes || got.AvailableBytes != probe.AvailableBytes || got.Privilege != probe.Privilege {
		t.Fatalf("probe = %+v, err=%v", got, err)
	}
}
