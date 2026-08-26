package hostcapability

import (
	"context"
	"testing"
)

func TestRegistryKeepsStructuralSatisfactionDistinct(t *testing.T) {
	registry := NewRegistry(AptProvider{ResolveFn: func(context.Context, Invariant, Facts) Result {
		return Result{Verdict: SatisfiedStructurally, Reason: "the platform cannot enter this failure mode"}
	}})
	result := registry.Resolve(context.Background(), Invariant{ID: "i1"}, Facts{OS: "linux"})
	if result.Verdict != SatisfiedStructurally {
		t.Fatalf("verdict = %q, want %q", result.Verdict, SatisfiedStructurally)
	}
	if result.Verdict == Satisfied {
		t.Fatal("structural satisfaction was coerced to ordinary satisfaction")
	}
}

func TestDarwinProviderReturnsNotApplicable(t *testing.T) {
	result := DarwinProvider{}.Resolve(context.Background(), Invariant{ID: "i1"}, Facts{OS: "darwin"})
	if result.Verdict != NotApplicable {
		t.Fatalf("verdict = %q, want %q", result.Verdict, NotApplicable)
	}
}

func TestAptProviderDerivesCoupledPackageInsideProvider(t *testing.T) {
	invariant := Invariant{ID: "alignment", Kind: "kernel_module_alignment", Applicability: map[string]string{"platforms": "linux", "vendor_id": "10de"}}
	result := (AptProvider{}).Resolve(context.Background(), invariant, Facts{
		OS: "linux", VendorID: "10de", DriverPackage: "nvidia-driver-580-open", KernelRelease: "6.17.0-23-generic",
		CandidatePackageNames: []string{"linux-modules-nvidia-580-open-6.17.0-23-generic"},
	})
	if result.Verdict != Failed {
		t.Fatalf("verdict = %q, want %q (%s)", result.Verdict, Failed, result.Reason)
	}
	if result.Evidence["expectedPackage"] != "linux-modules-nvidia-580-open-6.17.0-23-generic" {
		t.Fatalf("evidence = %#v", result.Evidence)
	}
}

func TestDeclaredNVIDIAInvariantsResolveThroughProvider(t *testing.T) {
	invariants, err := EmbeddedSafeguardInvariants("nvidia-driver")
	if err != nil {
		t.Fatal(err)
	}
	if len(invariants) != 2 {
		t.Fatalf("declared invariants = %d, want 2", len(invariants))
	}
	registry := NewRegistry(AptProvider{}, DarwinProvider{})
	results := Evaluate(context.Background(), registry, invariants, Facts{
		OS: "linux", VendorID: "10de", DriverPackage: "nvidia-driver-580-open", KernelRelease: "6.17.0-23-generic",
		PackageNames: []string{"linux-modules-nvidia-580-open-6.17.0-23-generic"},
	})
	for _, result := range results {
		if result.Verdict != Satisfied {
			t.Fatalf("invariant %q verdict = %q (%s), want %q", result.InvariantID, result.Verdict, result.Reason, Satisfied)
		}
	}
}

func TestAggregateReportsUnwalkedDeclarationSite(t *testing.T) {
	coverage := Aggregate([]Site{{Name: "walked", Walked: true, Invariants: []Invariant{{ID: "i1"}}}, {Name: "missing", Reason: "registry walker omitted this site"}}, map[string]int{"walked": 1})
	if coverage.SitesWalked != 1 || coverage.InvariantsDeclared != 1 || coverage.InvariantsEvaluated != 1 {
		t.Fatalf("coverage = %+v", coverage)
	}
	if len(coverage.Gaps) != 1 || coverage.Gaps[0] != "missing: registry walker omitted this site" {
		t.Fatalf("gaps = %v", coverage.Gaps)
	}
}
