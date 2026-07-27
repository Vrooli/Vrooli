package evidence

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	listRequest    *domainv1.ListEvidenceCapturesRequest
	summaryRequest *domainv1.ListEvidenceCapturesRequest
}

func (f *fakeRPC) ListEvidenceCaptures(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error) {
	f.listRequest = req.Msg
	return connect.NewResponse(&domainv1.ListEvidenceCapturesResponse{Captures: []*domainv1.EvidenceCapture{{CaptureId: "capture-1"}}}), nil
}

func (f *fakeRPC) GetEvidenceCapturesSummary(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error) {
	f.summaryRequest = req.Msg
	return connect.NewResponse(&domainv1.EvidenceCapturesSummary{Count: 1, TotalBytes: 42}), nil
}

func TestEvidencePrimitivesUseTypedScenarioRequests(t *testing.T) {
	fake := &fakeRPC{}
	commands := &Commands{rpc: fake}

	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
	listModes := cliapptest.RunPrimitiveHandlerModes(t, commands.listPrimitive(), schema, []string{"demo"}, nil)
	if listModes.HumanErr != nil || listModes.JSONErr != nil {
		t.Fatalf("list primitive errors: human=%v json=%v", listModes.HumanErr, listModes.JSONErr)
	}
	if fake.listRequest.GetScenarioName() != "demo" {
		t.Fatalf("list request scenario = %q, want demo", fake.listRequest.GetScenarioName())
	}

	summaryModes := cliapptest.RunPrimitiveHandlerModes(t, commands.summaryPrimitive(), schema, []string{"demo"}, nil)
	if summaryModes.HumanErr != nil || summaryModes.JSONErr != nil {
		t.Fatalf("summary primitive errors: human=%v json=%v", summaryModes.HumanErr, summaryModes.JSONErr)
	}
	if fake.summaryRequest.GetScenarioName() != "demo" {
		t.Fatalf("summary request scenario = %q, want demo", fake.summaryRequest.GetScenarioName())
	}
}

func TestEvidenceCommandsRequireScenario(t *testing.T) {
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
	if _, err := cliapptest.NewTestRunContextFromArgs(schema, nil, nil, nil, nil); err == nil {
		t.Fatal("missing scenario should fail production argument parsing")
	}
}
