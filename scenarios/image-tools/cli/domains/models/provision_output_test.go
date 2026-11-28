package models

import (
	"strings"
	"testing"
)

func TestHostFitProvisionOutputExplainsScopeAndOmissions(t *testing.T) {
	report := provisionEvidence{
		Mode:                "host_fit_operation_coverage",
		Ready:               false,
		CoverageComplete:    false,
		CoveredOperations:   []string{"remove"},
		UncoveredOperations: []provisionUncoveredOp{{Operation: "edit_instruct", Reason: "enabled_candidates_unavailable: qwen-image-edit:requires_32_gib_ram_host_has_18_gib_available"}},
		ExcludedModels:      []provisionExcludedModel{{ModelID: "qwen-image-edit", Operations: []string{"edit_instruct"}, Reason: "requires_32_gib_ram_host_has_18_gib_available"}},
		CandidateGaps:       []provisionCandidateGap{{ModelID: "mask-fill", Operation: "edit_instruct", Reason: "no_registered_local_backend_for_operation"}},
	}
	lines := report.resultLines("succeeded", "job-1")
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"All enabled capabilities ready: false",
		"Enabled capability coverage complete: false",
		"Operation edit_instruct: not ready",
		"Candidate qwen-image-edit excluded (requires_32_gib_ram_host_has_18_gib_available)",
		"Candidate mask-fill cannot serve edit_instruct (no_registered_local_backend_for_operation)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("CLI output omitted %q: %s", want, joined)
		}
	}
}

func TestExplicitProvisionOutputDoesNotClaimHostFitExpansion(t *testing.T) {
	report := provisionEvidence{Mode: "explicit_models", Ready: true, Models: []provisionModelResult{{}}}
	joined := strings.Join(report.resultLines("succeeded", "job-2"), "\n")
	if strings.Contains(joined, "host-fit") || strings.Contains(joined, "All enabled capabilities ready") {
		t.Fatalf("explicit model output was described as a host-fit expansion: %s", joined)
	}
}
