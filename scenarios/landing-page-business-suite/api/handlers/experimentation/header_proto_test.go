package variant

import (
	"math"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/experimentation"
)

func TestHeaderProtoPreservesNestedNavigation(t *testing.T) {
	sectionID := 7
	message, err := HeaderProto(experimentation.LandingHeaderConfig{Nav: experimentation.HeaderNavConfig{Links: []experimentation.HeaderNavLink{{ID: "docs", Label: "Docs", SectionID: &sectionID, Children: []experimentation.HeaderNavLink{{ID: "api", Label: "API"}}}}}})
	if err != nil {
		t.Fatalf("HeaderProto() error = %v", err)
	}
	link := message.GetNav().GetLinks()[0]
	if link.GetSectionId() != int32(sectionID) || link.GetChildren()[0].GetId() != "api" {
		t.Fatalf("header link = %#v", link)
	}
}

func TestHeaderProtoRejectsOutOfRangeSectionID(t *testing.T) {
	overflow := int(math.MaxInt32) + 1
	_, err := HeaderProto(experimentation.LandingHeaderConfig{Nav: experimentation.HeaderNavConfig{Links: []experimentation.HeaderNavLink{{ID: "overflow", SectionID: &overflow}}}})
	if err == nil || !strings.Contains(err.Error(), "outside the protobuf int32 range") {
		t.Fatalf("HeaderProto() error = %v, want range error", err)
	}
}
