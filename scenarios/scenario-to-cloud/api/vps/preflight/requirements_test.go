package preflight

import "testing"

func TestBuildRequirementsResponse_UsesCanonicalDefaults(t *testing.T) {
	t.Parallel()

	resp := BuildRequirementsResponse()

	if resp.VPS.Resources.MinRAMKB != 512*1024 {
		t.Fatalf("expected min RAM floor of 512 MiB, got %d KB", resp.VPS.Resources.MinRAMKB)
	}
	if resp.VPS.Resources.MinRAMBytes != 512*1024*1024 {
		t.Fatalf("expected min RAM bytes for 512 MiB, got %d", resp.VPS.Resources.MinRAMBytes)
	}
	if resp.VPS.Resources.RecommendedRAMKB != 2*1024*1024 {
		t.Fatalf("expected recommended RAM of 2 GiB, got %d KB", resp.VPS.Resources.RecommendedRAMKB)
	}
}
