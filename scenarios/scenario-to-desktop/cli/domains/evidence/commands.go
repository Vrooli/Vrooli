// Package evidence exposes durable desktop-validation evidence to operators.
package evidence

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"

	"scenario-to-desktop/cli/internal/support"
)

type evidenceRPC interface {
	ListEvidenceCaptures(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error)
	GetEvidenceCapturesSummary(context.Context, *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error)
}

type Commands struct{ rpc evidenceRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewEvidenceServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	scenarioArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	return cliapp.SubcommandGroup{Name: "evidence", Description: "Inspect durable desktop validation evidence", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "summary", Description: "Summarize persisted evidence captures", Args: scenarioArgs}).WithPrimitive(c.summaryPrimitive()),
	}}
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.ListEvidenceCapturesResponse, error) {
		response, err := c.rpc.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list evidence captures", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListEvidenceCapturesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Evidence captures retrieved"}, Results: []string{fmt.Sprintf("Captures: %d", len(response.GetCaptures()))}}
	})
}

func (c *Commands) summaryPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.EvidenceCapturesSummary, error) {
		response, err := c.rpc.GetEvidenceCapturesSummary(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get evidence captures summary", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.EvidenceCapturesSummary) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Evidence capture summary retrieved"}, Results: []string{fmt.Sprintf("Captures: %d", response.GetCount()), fmt.Sprintf("Bytes: %d", response.GetTotalBytes())}}
	})
}
