package credits

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	legacy := deps.EndpointCommands([]support.EndpointDef{
		{Name: "admin-tiers-limits", Method: "GET", Path: "/admin/tiers/limits", Description: "List tier limits (admin)"},
		{Name: "admin-tier-limits", Method: "GET", Path: "/admin/tiers/{tier}/limits", Description: "Get tier limits (admin)"},
		{Name: "admin-tier-limits-update", Method: "PUT", Path: "/admin/tiers/{tier}/limits", Description: "Update tier limits (admin)"},
		{Name: "admin-limits-create", Method: "POST", Path: "/admin/limits", Description: "Create tier limit (admin)"},
		{Name: "admin-limits-delete", Method: "DELETE", Path: "/admin/limits", Description: "Delete tier limit (admin)"},
		{Name: "admin-app-limits", Method: "GET", Path: "/admin/apps/{app}/limits", Description: "Get app limits (admin)"},
		{Name: "usage-report", Method: "POST", Path: "/usage/report", Description: "Report usage (service auth)"},
		{Name: "usage-summary", Method: "GET", Path: "/usage/summary", Description: "Get usage summary"},
		{Name: "usage-check", Method: "GET", Path: "/usage/check", Description: "Check usage limits"},
		{Name: "usage-health", Method: "GET", Path: "/usage/health", Description: "Usage health"},
		{Name: "admin-usage-summary", Method: "GET", Path: "/admin/usage", Description: "Admin usage summary"},
	})
	commands := []cliapp.Command{listAPIKeysCommand(deps), createAPIKeyCommand(deps), deleteAPIKeyCommand(deps), testAPIKeyCommand(deps), setAPIKeyActiveCommand(deps)}
	return cliapp.CommandGroup{Title: "Credits", Commands: append(commands, legacy...)}
}

func administrationClient(deps support.Dependencies) (lpbsconnect.AdministrationServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewAdministrationServiceClient(httpClient, baseURL), nil
}

func listAPIKeysCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.ListAPIKeysResponse, error) {
		client, err := administrationClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ListAPIKeys(context.Background(), connect.NewRequest(&lpbsv1.ListAPIKeysRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list API keys", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.ListAPIKeysResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Configured API keys (credentials are never returned)."}, ResultsHeading: "API keys"}
	})
	return (cliapp.Command{Name: "admin-api-keys-list", NeedsAPI: true, Description: "List API keys through the generated Connect contract (admin)", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func createAPIKeyCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.CreateAPIKeyResponse, error) {
		request := &lpbsv1.CreateAPIKeyRequest{}
		if err := decodeBody(ctx, request); err != nil {
			return nil, err
		}
		client, err := administrationClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.CreateAPIKey(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("create API key", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.CreateAPIKeyResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"API key created."}}
	})
	return (cliapp.Command{Name: "admin-api-keys-create", NeedsAPI: true, Description: "Create API key through the generated Connect contract (admin; --body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Generated request JSON or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func deleteAPIKeyCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.DeleteAPIKeyResponse, error) {
		request := &lpbsv1.DeleteAPIKeyRequest{}
		if err := decodeBody(ctx, request); err != nil {
			return nil, err
		}
		client, err := administrationClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.DeleteAPIKey(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete API key", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.DeleteAPIKeyResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"API key deleted."}}
	})
	return (cliapp.Command{Name: "admin-api-keys-delete", NeedsAPI: true, Description: "Delete API key through the generated Connect contract (admin; --body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Generated request JSON or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func testAPIKeyCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.TestAPIKeyResponse, error) {
		request := &lpbsv1.TestAPIKeyRequest{}
		if err := decodeBody(ctx, request); err != nil {
			return nil, err
		}
		client, err := administrationClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.TestAPIKey(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("test API key", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.TestAPIKeyResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"API key test completed."}}
	})
	return (cliapp.Command{Name: "admin-api-keys-test", NeedsAPI: true, Description: "Test API key through the generated Connect contract (admin; --body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Generated request JSON or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func setAPIKeyActiveCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.SetAPIKeyActiveResponse, error) {
		request := &lpbsv1.SetAPIKeyActiveRequest{}
		if err := decodeBody(ctx, request); err != nil {
			return nil, err
		}
		client, err := administrationClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.SetAPIKeyActive(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("set API key active state", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.SetAPIKeyActiveResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"API key active state updated."}}
	})
	return (cliapp.Command{Name: "admin-api-keys-toggle", NeedsAPI: true, Description: "Set API key active state through the generated Connect contract (admin; --body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Generated request JSON or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func decodeBody[M proto.Message](ctx cliapp.OperationContext, message M) error {
	payload, err := support.ParseBody(ctx.Flag("body"))
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(payload, message); err != nil {
		return fmt.Errorf("decode API key request: %w", err)
	}
	return nil
}
