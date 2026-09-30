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
