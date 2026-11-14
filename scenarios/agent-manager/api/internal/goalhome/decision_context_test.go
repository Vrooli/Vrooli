package goalhome

import (
	"strings"
	"testing"

	"github.com/vrooli/api-core/operatordecision"
)

func TestDecisionContextUsesSharedCodePointBounds(t *testing.T) {
	d := Decision{ID: "d-1", Question: "Choose?", Options: []DecisionOption{{Key: "yes"}, {Key: "no"}}, Recommended: "yes"}
	d.BigPicture = strings.Repeat("😀", operatordecision.MaxBigPictureCodePoints)
	d.TechnicalDetails = strings.Repeat("🧭", operatordecision.MaxTechnicalDetailsCodePoints)
	if err := d.Validate(); err != nil {
		t.Fatalf("shared context limits should accept their exact code-point bounds: %v", err)
	}
	d.BigPicture += "x"
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "big_picture") {
		t.Fatalf("purpose beyond the shared limit should be rejected, got %v", err)
	}
	d.BigPicture = ""
	d.TechnicalDetails += "x"
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "technical_details") {
		t.Fatalf("technical context beyond the shared limit should be rejected, got %v", err)
	}
}
