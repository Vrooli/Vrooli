package waitlist

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"landing-page-business-suite/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Engagement - Waitlist", Commands: []cliapp.Command{
		createCommand(deps), listCommand(deps), deleteCommand(deps), exportCommand(deps),
	}}
}

func publicClient(deps support.Dependencies) lpbsconnect.WaitlistServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return lpbsconnect.NewWaitlistServiceClient(httpClient, baseURL)
}

func adminClient(deps support.Dependencies) (lpbsconnect.WaitlistServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewWaitlistServiceClient(httpClient, baseURL), nil
}

func createCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.CreateWaitlistEntryResponse, error) {
		request := &lpbsv1.CreateWaitlistEntryRequest{}
		payload, err := support.ParseBody(ctx.Flag("body"))
		if err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal(payload, request); err != nil {
			return nil, fmt.Errorf("decode waitlist creation: %w", err)
		}
		response, err := publicClient(deps).CreateWaitlistEntry(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("create waitlist entry", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.CreateWaitlistEntryResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Waitlist entry created."}}
	})
	return (cliapp.Command{Name: "waitlist-create", NeedsAPI: true, Description: "Create a waitlist entry through the generated Connect contract (--body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Waitlist JSON payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func listCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.ListWaitlistEntriesResponse, error) {
		client, err := adminClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ListWaitlistEntries(context.Background(), connect.NewRequest(&lpbsv1.ListWaitlistEntriesRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list waitlist entries", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.ListWaitlistEntriesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Waitlist signups."}, ResultsHeading: "Waitlist"}
	})
	return (cliapp.Command{Name: "admin-waitlist-list", NeedsAPI: true, Description: "List waitlist entries through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func deleteCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.DeleteWaitlistEntryResponse, error) {
		id, err := waitlistID(ctx)
		if err != nil {
			return nil, err
		}
		client, err := adminClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.DeleteWaitlistEntry(context.Background(), connect.NewRequest(&lpbsv1.DeleteWaitlistEntryRequest{Id: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete waitlist entry", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.DeleteWaitlistEntryResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Waitlist entry deleted."}}
	})
	return (cliapp.Command{Name: "admin-waitlist-delete", NeedsAPI: true, Description: "Delete a waitlist entry through the generated Connect contract (ID)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func exportCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.ExportWaitlistEntriesResponse, error) {
		client, err := adminClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ExportWaitlistEntries(context.Background(), connect.NewRequest(&lpbsv1.ExportWaitlistEntriesRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("export waitlist entries", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.ExportWaitlistEntriesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"CSV export returned in the generated response."}, ResultsHeading: "Waitlist export"}
	})
	return (cliapp.Command{Name: "admin-waitlist-export", NeedsAPI: true, Description: "Export waitlist entries through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func waitlistID(ctx cliapp.OperationContext) (int64, error) {
	value := strings.TrimSpace(ctx.Positional("id"))
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	return id, nil
}
