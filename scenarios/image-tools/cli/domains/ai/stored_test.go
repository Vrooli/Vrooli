package ai

import (
	"bytes"
	"connectrpc.com/connect"
	"context"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
)

func TestSubmitStoredRetainsReferencesTypedParamsAndIdentity(t *testing.T) {
	const id = "46d04887-6c68-465d-81d9-3e8247078ce9"
	calls := 0
	_, server := aiconnect.NewAIServiceHandler(&storedCLIServer{submit: func(request *aiv1.SubmitStoredAIRequest) {
		calls++
		if request.Operation != "segment" || request.InputRef != "input/owned-source" || request.MaskRef != "mask/owned-mask" || request.OperationId != id || len(request.GetParams().GetSegmentation().GetPoints()) != 1 || request.Params.Segmentation.Points[0].X != .7 {
			t.Errorf("request lost fields: %v", request)
		}
	}})
	core := testutil.NewTestApp(t, server)
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(core, manifest)
	if err != nil {
		t.Fatal(err)
	}
	cmd := findCommand(t, group.Subcommands, "submit-stored")
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cmd.Args, Positionals: map[string]string{"operation": "segment"}, Flags: map[string]string{"input-ref": "input/owned-source", "mask-ref": "mask/owned-mask", "operation-id": id, "params": `{"segmentation":{"points":[{"x":0.7,"y":0.4}]}}`}, Core: core, Stdout: &output, Stderr: &output})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !bytes.Contains(output.Bytes(), []byte("job-owned")) {
		t.Fatalf("missing durable result calls=%d %s", calls, output.String())
	}
}

type storedCLIServer struct {
	aiconnect.UnimplementedAIServiceHandler
	submit func(*aiv1.SubmitStoredAIRequest)
}

func (s *storedCLIServer) SubmitStoredAI(_ context.Context, request *connect.Request[aiv1.SubmitStoredAIRequest]) (*connect.Response[aiv1.SubmitAIResponse], error) {
	s.submit(request.Msg)
	return connect.NewResponse(&aiv1.SubmitAIResponse{JobId: "job-owned", ModelId: "sam2.1-hiera-tiny", Tier: "local-gpu"}), nil
}
