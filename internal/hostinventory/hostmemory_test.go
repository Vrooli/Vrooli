package hostinventory

import "testing"

func TestHostMemoryUntrustworthyCannotBecomeBudget(t *testing.T) {
	facts := HostMemory{}
	if facts.Trustworthy {
		t.Fatal("zero host memory must not be trustworthy")
	}
	if facts.AvailableBytes != 0 {
		t.Fatalf("available memory = %d, want zero", facts.AvailableBytes)
	}
}

func TestHostMemoryFactsReportsPositiveTrustedValuesOnSupportedHost(t *testing.T) {
	facts, err := HostMemoryFacts()
	if err != nil {
		t.Skipf("host memory probe unavailable: %v", err)
	}
	if facts.Trustworthy && (facts.TotalBytes == 0 || facts.AvailableBytes == 0) {
		t.Fatalf("trusted host memory has zero value: %+v", facts)
	}
}

func TestHostMemoryBoundsRejectCounterOverflow(t *testing.T) {
	for _, facts := range []HostMemory{{TotalBytes: 1, AvailableBytes: 2, Trustworthy: true}, {TotalBytes: ^uint64(0), AvailableBytes: 1, Trustworthy: true}, {TotalBytes: 1, AvailableBytes: 0, Trustworthy: true}} {
		if _, err := validateHostMemoryFacts(facts); err == nil {
			t.Fatalf("inconsistent trusted memory: %+v", facts)
		}
	}
	for _, counts := range [][4]uint64{{1, 0, 0, 0}, {1, 4096, ^uint64(0), 1}, {1, ^uint64(0), 2, 0}, {4096, 4096, 2, 0}} {
		if _, err := hostMemoryFromDarwinPages(counts[0], counts[1], counts[2], counts[3]); err == nil {
			t.Fatalf("invalid native page counters accepted: %v", counts)
		}
	}
	// The Mach free counter already includes speculative pages. This API accepts
	// only the free+inactive counters, preventing a second speculative addition.
	facts, err := hostMemoryFromDarwinPages(16<<30, 16384, 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	if facts.AvailableBytes != 300*16384 || !facts.Trustworthy {
		t.Fatalf("native page accounting: %+v", facts)
	}
	zero, err := hostMemoryFromDarwinPages(16<<30, 4096, 0, 0)
	if err != nil || zero.Trustworthy {
		t.Fatal("zero free pages became a trusted capacity grant")
	}
}
