package modelselection

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	modelselectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/modelselection"
	"google.golang.org/protobuf/proto"
)

type fixedService struct{ result Result }

func (s fixedService) GetRoleModelChoices(context.Context, string, string, string) Result {
	return s.result
}

func TestContextWindowSourceSurvivesTheModelChoiceWireContract(t *testing.T) {
	service := fixedService{result: Result{Choices: []Choice{{
		Model: "gpt-6.1-terra", ContextWindowTokens: 500_000,
		ContextWindowSource: "codex-models-cache/v1",
	}}}}
	response, err := (&handler{service: service}).GetRoleModelChoices(context.Background(), connect.NewRequest(&modelselectionv1.GetRoleModelChoicesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	wire, err := proto.Marshal(response.Msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded modelselectionv1.GetRoleModelChoicesResponse
	if err := proto.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.GetChoices()) != 1 || decoded.GetChoices()[0].GetContextWindowSource() != "codex-models-cache/v1" {
		t.Fatalf("context window provenance did not survive API mapping and protobuf encoding: %+v", decoded.GetChoices())
	}
}

func TestRouteKindAndApplicableLimitSurviveTheOwnerWireContract(t *testing.T) {
	kinds := []struct {
		route RouteKind
		limit LimitKind
	}{
		{RouteKindLocal, LimitKindNoMonetarySpend},
		{RouteKindOwnKey, LimitKindNotReported},
		{RouteKindVrooliCredit, LimitKindSelectableCredits},
		{RouteKindHosted, LimitKindNotReported},
	}
	choices := make([]Choice, 0, len(kinds))
	for i, kind := range kinds {
		choices = append(choices, Choice{Model: "fixture", RouteKind: kind.route, ApplicableLimit: ApplicableLimit{Kind: kind.limit, MaxValue: int64(i + 1), Unit: "credits", Provenance: "typed owner fixture", Options: []int64{1000, 5000}, DefaultValue: 1000}})
	}
	response, err := (&handler{service: fixedService{result: Result{Choices: choices}}}).GetRoleModelChoices(context.Background(), connect.NewRequest(&modelselectionv1.GetRoleModelChoicesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	wire, err := proto.Marshal(response.Msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded modelselectionv1.GetRoleModelChoicesResponse
	if err := proto.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.GetChoices()) != len(kinds) {
		t.Fatalf("decoded %d route choices, want %d", len(decoded.GetChoices()), len(kinds))
	}
	for i, choice := range decoded.GetChoices() {
		if choice.GetRouteKind() != routeKindToProto(kinds[i].route) || choice.GetApplicableLimit().GetKind() != limitKindToProto(kinds[i].limit) || choice.GetApplicableLimit().GetProvenance() != "typed owner fixture" || choice.GetApplicableLimit().GetDefaultValue() != 1000 || len(choice.GetApplicableLimit().GetOptions()) != 2 {
			t.Fatalf("route/limit did not survive wire encoding: %+v", choice)
		}
	}
}
