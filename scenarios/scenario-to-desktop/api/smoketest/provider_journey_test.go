package smoketest

import (
	"testing"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/validationdesktop"
)

func TestWorkflowReferenceFromProviderCarriesBASArtifactsAndIdentity(t *testing.T) {
	result := validationdesktop.Result{
		Disposition:   "pass",
		ProviderRunID: "provider-run-1",
		Evidence: []*domainv1.LayeredEvidence{{
			Kind:       domainv1.LayeredEvidence_KIND_BAS_WORKFLOW,
			EvidenceId: "timeline.json",
			Uri:        "workflow-health://run-1/timeline.json",
			Sha256:     "sha256:timeline",
			MediaType:  stringPtr("application/json"),
			Redacted:   true,
		}},
	}
	ref := workflowReferenceFromProvider(result, "smoke-1", "local-linux", "cell-1", "web-console:bas/cases/smoke.json", "sha256:artifact")
	if err := ref.ValidateLink("smoke-1", "sha256:artifact", "local-linux", "cell-1"); err != nil {
		t.Fatalf("workflow reference failed link validation: %v", err)
	}
	if ref.Provider != "workflow-health" || ref.ExecutionID != "provider-run-1" || len(ref.Artifacts) != 1 {
		t.Fatalf("workflow reference = %+v, want provider identity and one BAS artifact", ref)
	}
}

func stringPtr(value string) *string { return &value }
