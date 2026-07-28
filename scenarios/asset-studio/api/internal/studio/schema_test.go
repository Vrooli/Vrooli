package studio

import (
	"strings"
	"testing"
)

func TestSchemaCapturesP0ImmutabilityAndGateColumns(t *testing.T) {
	for _, want := range []string{"studio_identity_versions", "conditioning_references_json", "provenance_json", "actual_cost", "studio_conformance_verdicts", "actor_kind = 'operator'", "credential_claims", "disclosure"} {
		if !strings.Contains(Schema(), want) {
			t.Errorf("schema missing %q", want)
		}
	}
}
