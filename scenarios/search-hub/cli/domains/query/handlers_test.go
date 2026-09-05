package query

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestFormatHitConfidenceAndLocations(t *testing.T) {
	hit := &routingv1.SearchHit{
		Id:            "plan-manager/authoring",
		Title:         "plan-manager/authoring",
		Snippet:       "Turns intent into plans.",
		ProviderGroup: "architecture-cartographer",
		Confidence:    &commonv1.Confidence{Weak: true, Regime: "fused"},
		Locations: []string{
			"scenarios/plan-manager/api/internal/authoring/",
			"scenarios/plan-manager/api/handlers/authoring/",
			"packages/proto/schemas/plan-manager/v1/authoring/",
		},
	}

	got := formatHit(1, hit)
	for _, want := range []string{
		"confidence=weak/fused",
		"architecture-cartographer/scenarios/plan-manager/api/internal/authoring/",
		"locations: scenarios/plan-manager/api/internal/authoring/, scenarios/plan-manager/api/handlers/authoring/ (+1 more)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatHit() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "score=") {
		t.Fatalf("formatHit() should not render raw score: %q", got)
	}
}

func TestRenderGroupsAllWeakNotice(t *testing.T) {
	lines := renderGroups([]*routingv1.ProviderResultGroup{{
		ProviderId: "architecture-cartographer.domain-map",
		Count:      1,
		Hits: []*routingv1.SearchHit{{
			Id:            "x",
			ProviderGroup: "architecture-cartographer",
			Confidence:    &commonv1.Confidence{Weak: true, Regime: "fused"},
		}},
	}})

	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "no confident match") {
		t.Fatalf("renderGroups() = %q, want no confident match notice", got)
	}
}
