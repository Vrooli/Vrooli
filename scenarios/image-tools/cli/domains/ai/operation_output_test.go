package ai

import (
	"strings"
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

func TestOperationRowShowsTheDefaultWithItsVerdictOrWhyThereIsNone(t *testing.T) {
	with := formatOperation(&aiv1.AIOperationInfo{
		Name: "upscale", Category: "enhance", Summary: "Super-resolve", DefaultModelId: "real-esrgan",
		DefaultVerdict: &modelsv1.Verdict{Grade: modelsv1.VerdictGrade_VERDICT_GRADE_WORKS},
	})
	if !strings.HasSuffix(with, "(default: real-esrgan, works)") {
		t.Fatalf("row = %q", with)
	}
	without := formatOperation(&aiv1.AIOperationInfo{Name: "face_swap", Category: "generate", Summary: "Swap", DefaultUnavailable: "no enabled model for operation"})
	if !strings.HasSuffix(without, "(no default: no enabled model for operation)") {
		t.Fatalf("row = %q", without)
	}
	if !strings.Contains(formatOperation(&aiv1.AIOperationInfo{Name: "x"}), "no default: no model selected") {
		t.Fatal("a row with neither a default nor a reason still says so")
	}
}
