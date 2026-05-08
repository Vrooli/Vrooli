package hostinventory

import "testing"

func TestParseLSPCIExtractsDriverAndModules(t *testing.T) {
	devices := ParseLSPCI(`01:00.0 VGA compatible controller: Example GPU [1234:5678]
	Kernel driver in use: example
	Kernel modules: example, fallback
02:00.0 Network controller: Example NIC [1111:2222]
	Kernel modules: nicmod`)

	if len(devices) != 2 {
		t.Fatalf("device count = %d, want 2", len(devices))
	}
	if devices[0].Address != "01:00.0" {
		t.Fatalf("address = %q, want 01:00.0", devices[0].Address)
	}
	if devices[0].BoundDriver != "example" {
		t.Fatalf("bound driver = %q, want example", devices[0].BoundDriver)
	}
	if len(devices[0].AvailableModules) != 2 {
		t.Fatalf("available modules = %#v, want two modules", devices[0].AvailableModules)
	}
	if devices[1].BoundDriver != "" {
		t.Fatalf("second device bound driver = %q, want empty", devices[1].BoundDriver)
	}
}
