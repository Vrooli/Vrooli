package providerconformance

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// TestProviderConformanceSpecCoversEveryCode pins Test Genie's own descriptor
// spec to the validator's finding vocabulary: every emitted code must map in
// the embedded maturity block so no finding falls through to the fallback.
func TestProviderConformanceSpecCoversEveryCode(t *testing.T) {
	spec := mustLoadSpec(t)
	if spec.Provider != "test-genie" {
		t.Fatalf("provider = %q, want test-genie", spec.Provider)
	}
	if spec.Phase != "provider-conformance" {
		t.Fatalf("phase = %q, want provider-conformance", spec.Phase)
	}
	codes := []string{
		CodeDescriptorMissing,
		CodeDescriptorInvalid,
		CodeIdentityMismatch,
		CodeMaturityInvalid,
		CodeStaleMaturityFile,
		CodePolicyUnsafe,
		CodeDocsMissing,
		CodeDocsSkeletonIncomplete,
		CodeNorthStarMissing,
		CodeLadderIncomplete,
		CodeRungUngated,
		CodeAutofixDeclarationIncomplete,
		CodeProviderUnreachable,
		CodeContractInvalid,
		CodeContractIdentityMismatch,
		CodeMetricsMissing,
	}
	for _, code := range codes {
		if _, ok := spec.Findings[code]; !ok {
			t.Errorf("descriptor maturity block is missing finding mapping for %s", code)
		}
	}
	for declared := range spec.Findings {
		known := false
		for _, code := range codes {
			if declared == code {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("descriptor maturity block declares %s, which the validator never emits", declared)
		}
	}
	coverage := assessment.ComputeAutofixCoverage(*spec)
	if !coverage.DeclarationComplete {
		t.Fatalf("test-genie's own autofix declaration must be complete; declared %d of %d", coverage.Declared, coverage.Total)
	}
}

func mustLoadSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	scenarioRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot)
	if err != nil {
		t.Fatalf("LoadSpecFromScenario: %v", err)
	}
	return spec
}
