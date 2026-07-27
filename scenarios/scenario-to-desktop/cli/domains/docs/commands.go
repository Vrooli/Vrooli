// Package docs exposes the scenario documentation contract to CLI operators.
package docs

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"

	"scenario-to-desktop/cli/internal/support"
)

type documentationRPC interface {
	GetDocumentationManifest(context.Context, *connect.Request[domainv1.DocumentationManifestRequest]) (*connect.Response[domainv1.DocumentationManifestResponse], error)
}

type Commands struct{ rpc documentationRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewDocumentationServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	return cliapp.SubcommandGroup{Name: "docs", Description: "Inspect scenario documentation", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "manifest", Description: "Show the scenario documentation manifest"}).WithPrimitive(c.manifestPrimitive()),
	}}
}

func (c *Commands) manifestPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(_ cliapp.OperationContext) (*domainv1.DocumentationManifestResponse, error) {
		response, err := c.rpc.GetDocumentationManifest(context.Background(), connect.NewRequest(&domainv1.DocumentationManifestRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get documentation manifest", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.DocumentationManifestResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Documentation manifest retrieved"}, Results: []string{fmt.Sprintf("Sections: %d", len(response.GetSections()))}}
	})
}
