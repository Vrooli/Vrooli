package journal

import (
	"testing"

	"github.com/vrooli/api-core/provenance"
)

func verifiedAgent() provenance.Provenance {
	return provenance.Provenance{
		Actor:               provenance.ActorAgent,
		VerificationStatus:  provenance.VerificationVerified,
		RunID:               "run-1234",
		WorkflowExecutionID: "wfx-9",
		Invocation:          provenance.Invocation{Scenario: "vrooli-memory"},
	}
}

// A verified agent write must persist correlation regardless of which seam it
// entered through — this is the behaviour that was missing for direct callers.
func TestAttributionFromVerifiedAgentPopulatesCorrelation(t *testing.T) {
	attribution, correlation := AttributionFrom(verifiedAgent(), Attribution{})

	if attribution.ActorID != "run-1234" {
		t.Fatalf("actor id = %q, want run-1234", attribution.ActorID)
	}
	if attribution.VerificationStatus != provenance.VerificationVerified {
		t.Fatalf("verification status = %q, want verified", attribution.VerificationStatus)
	}
	if correlation.RunID != "run-1234" {
		t.Fatalf("correlation run id = %q, want run-1234", correlation.RunID)
	}
	if correlation.WorkflowExecutionID != "wfx-9" {
		t.Fatalf("correlation workflow execution id = %q, want wfx-9", correlation.WorkflowExecutionID)
	}
}

// Internal writers keep their own descriptive labels; only the verified fields
// are taken from provenance.
func TestAttributionFromPreservesCallerDescriptiveFields(t *testing.T) {
	existing := Attribution{ActorKind: "harness-import", SourceRuntime: "claude-code"}

	attribution, correlation := AttributionFrom(verifiedAgent(), existing)

	if attribution.ActorKind != "harness-import" {
		t.Fatalf("actor kind = %q, want harness-import", attribution.ActorKind)
	}
	if attribution.SourceRuntime != "claude-code" {
		t.Fatalf("source runtime = %q, want claude-code", attribution.SourceRuntime)
	}
	if attribution.ActorID != "run-1234" {
		t.Fatalf("actor id = %q, want the verified run id", attribution.ActorID)
	}
	if correlation.ActorKind != "harness-import" {
		t.Fatalf("correlation actor kind = %q, want harness-import", correlation.ActorKind)
	}
}

// An unverified caller must never acquire correlation, and the verification
// outcome must stay distinguishable from a verified write.
func TestAttributionFromUnverifiedCallerHasNoCorrelation(t *testing.T) {
	source := provenance.Provenance{
		Actor:              provenance.ActorOperator,
		VerificationStatus: provenance.VerificationInvalid,
		RunID:              "forged-run",
		Invocation: provenance.Invocation{
			HarnessSessionID: "0f916b7c",
			HarnessKind:      "claude-code",
		},
	}

	attribution, correlation := AttributionFrom(source, Attribution{})

	if attribution.ActorID != "" {
		t.Fatalf("actor id = %q, want empty for an unverified caller", attribution.ActorID)
	}
	if correlation.RunID != "" {
		t.Fatalf("correlation run id = %q, want empty for an unverified caller", correlation.RunID)
	}
	if attribution.VerificationStatus != provenance.VerificationInvalid {
		t.Fatalf("verification status = %q, want invalid", attribution.VerificationStatus)
	}
	if attribution.HarnessSessionID != "0f916b7c" || attribution.HarnessKind != "claude-code" {
		t.Fatalf("harness observation not carried: %+v", attribution)
	}
}
