package validate

import (
	"testing"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestStatusLabelExposesNotEvaluated(t *testing.T) {
	if got := statusLabel(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED); got != "not-evaluated" {
		t.Fatalf("NOT_EVALUATED label = %q", got)
	}
}
