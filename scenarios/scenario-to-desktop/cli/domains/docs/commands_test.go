package docs

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	request *domainv1.DocumentationManifestRequest
}

func (f *fakeRPC) GetDocumentationManifest(_ context.Context, req *connect.Request[domainv1.DocumentationManifestRequest]) (*connect.Response[domainv1.DocumentationManifestResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&domainv1.DocumentationManifestResponse{Sections: []*domainv1.DocumentationSection{{Title: "Guide"}}}), nil
}

func TestManifestPrimitiveUsesGeneratedRequest(t *testing.T) {
	fake := &fakeRPC{}
	modes := cliapptest.RunPrimitiveHandlerModes(t, (&Commands{rpc: fake}).manifestPrimitive(), cliapp.ArgSchema{}, nil, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("manifest primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.request == nil {
		t.Fatal("manifest request was not sent")
	}
}
