package runtime

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestDeclaredNVIDIAInvariantsResolveThroughProvider(t *testing.T) {
	invariants, err := embeddedSafeguardInvariants("nvidia-driver")
	if err != nil {
		t.Fatal(err)
	}
	if len(invariants) != 2 {
		t.Fatalf("declared invariants = %d, want 2", len(invariants))
	}
	registry := hostreqkit.NewRegistry(hostreqkit.AptProvider{}, hostreqkit.DarwinProvider{})
	results := hostreqkit.Evaluate(context.Background(), registry, invariants, hostreqkit.Facts{
		OS: "linux", VendorID: "10de", DriverPackage: "nvidia-driver-580-open", KernelRelease: "6.17.0-23-generic",
		PackageNames: []string{"linux-modules-nvidia-580-open-6.17.0-23-generic"},
	})
	for _, result := range results {
		if result.Verdict != hostreqkit.Satisfied {
			t.Fatalf("invariant %q verdict = %q (%s), want %q", result.InvariantID, result.Verdict, result.Reason, hostreqkit.Satisfied)
		}
	}
}
