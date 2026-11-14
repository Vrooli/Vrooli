package attention

import (
	"bytes"
	"strings"
	"testing"

	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
)

func TestRenderClaimResponseDistinguishesNoWorkFromSweepCompletion(t *testing.T) {
	three, zero := uint32(3), uint32(0)
	var output bytes.Buffer
	renderClaimResponse(&output, &attentionv1.ClaimResponse{NoWork: true, UnreviewedCount: &three})
	if !strings.Contains(output.String(), "No unclaimed work is available.") || !strings.Contains(output.String(), "Unreviewed files: 3") {
		t.Fatalf("claim output conflates temporary availability and first-pass progress: %q", output.String())
	}

	output.Reset()
	renderClaimResponse(&output, &attentionv1.ClaimResponse{UnreviewedCount: &zero})
	if !strings.Contains(output.String(), "Unreviewed files: 0") {
		t.Fatalf("completed first sweep is not visible: %q", output.String())
	}
}
