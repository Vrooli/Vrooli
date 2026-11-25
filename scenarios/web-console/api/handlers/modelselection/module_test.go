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
