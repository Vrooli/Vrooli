package query

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestOperationHintReadsBundleWithoutExecutingRetrievedCode(t *testing.T) {
	msg := &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{
		{ProviderId: "program-runtime.library", Id: "device-control.do-task"},
	}}
	hint := strings.Join(operationHints(msg), "\n")
	if !strings.Contains(hint, "library run program-runtime.prepare-operation --input name=device-control.do-task") {
		t.Fatal(hint)
	}
	for _, unsafe := range []string{"thing;echo secret", "thing.$(whoami)", "thing.name\nnext"} {
		msg.Ranked[0].Id = unsafe
		if len(operationHints(msg)) != 0 {
			t.Fatal("unsafe command suggestion", unsafe)
		}
	}
}

func TestOperationHintPrintsLibraryUsage(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{"usage": float64(4)})
	if err != nil {
		t.Fatal(err)
	}
	msg := &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
		ProviderId: "program-runtime.library",
		Id:         "program-runtime.setpoint-read",
		Metadata:   metadata,
	}}}
	hint := strings.Join(operationHints(msg), "\n")
	if !strings.Contains(hint, "recorded usage: 4") {
		t.Fatalf("operation hint = %q", hint)
	}
}

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
