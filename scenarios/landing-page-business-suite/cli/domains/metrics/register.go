package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	"landing-page-business-suite/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Metrics", Commands: []cliapp.Command{trackCommand(deps), summaryCommand(deps), variantsCommand(deps)}}
}

func client(deps support.Dependencies) (lpbsconnect.MetricsServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewMetricsServiceClient(httpClient, baseURL), nil
}

func action(deps support.Dependencies, description string, call func(context.Context, lpbsconnect.MetricsServiceClient) (proto.Message, error)) cliapp.Command {
	op := cliapp.Action(func(cliapp.OperationContext) (json.RawMessage, error) {
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := call(context.Background(), service)
		if err != nil {
			return nil, cliapp.WrapAPIError(description, err, nil)
		}
		payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", description, err)
		}
		return json.RawMessage(payload), nil
	}, func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{description}}
	})
	return (cliapp.Command{NeedsAPI: true, Description: description, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(op)
}

func trackCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.Action(func(ctx cliapp.OperationContext) (json.RawMessage, error) {
		body, err := support.ParseBody(ctx.Flag("body"))
		if err != nil {
			return nil, err
		}
		if body == nil {
			return nil, fmt.Errorf("usage: metrics-track --body '{...}'")
		}
		request := &lpbsv1.TrackEventRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, request); err != nil {
			return nil, fmt.Errorf("decode metrics event: %w", err)
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.TrackEvent(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("track metrics event", err, nil)
		}
		payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
		if err != nil {
			return nil, fmt.Errorf("encode metrics event: %w", err)
		}
		return json.RawMessage(payload), nil
	}, func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Metrics event tracked."}}
	})
	return (cliapp.Command{Name: "metrics-track", NeedsAPI: true, Description: "Track a metrics event through the generated Connect contract [--body JSON|@file|-]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "TrackEventRequest JSON payload or @file.json"}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(op)
}

func summaryCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, "Get metrics summary through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.MetricsServiceClient) (proto.Message, error) {
		response, err := service.GetAnalyticsSummary(ctx, connect.NewRequest(&lpbsv1.GetAnalyticsSummaryRequest{}))
		return response.Msg, err
	})
	command.Name = "metrics-summary"
	return command
}

func variantsCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, "Get variant metrics through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.MetricsServiceClient) (proto.Message, error) {
		response, err := service.GetVariantStats(ctx, connect.NewRequest(&lpbsv1.GetVariantStatsRequest{}))
		return response.Msg, err
	})
	command.Name = "metrics-variants"
	return command
}
