package checks

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
)

func runCommand() cliapp.Command {
	return (cliapp.Command{
		Name: "run", Description: "Refresh one check without recovery actions",
		Args:         cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "check-id", Required: true}}},
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation},
	}).WithPrimitive(cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*checksv1.RunCheckResponse, error) {
			id := strings.TrimSpace(ctx.Positional("check-id"))
			if id == "" {
				return nil, fmt.Errorf("check-id is required")
			}
			httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core())
			resp, err := checksconnect.NewChecksServiceClient(httpClient, baseURL).RunCheck(context.Background(), connect.NewRequest(&checksv1.RunCheckRequest{CheckId: id}))
			if err != nil {
				return nil, cliapp.WrapAPIError("refresh check observation", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, resp *checksv1.RunCheckResponse) cliapp.MutationReport {
			r := resp.GetResult()
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s: %s — %s", r.GetCheckId(), r.GetStatus(), r.GetMessage())}, Changes: []string{"Refreshed and persisted one observation; recovery actions were not executed."}}
		},
	))
}
