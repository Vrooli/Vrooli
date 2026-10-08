package ai

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"landing-page-business-suite/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := []cliapp.Command{modelsCommand(deps), healthCommand(deps), chatCommand(deps), usageCommand(deps)}
	commands = append(commands, cliapp.Command{
		Name:        "ai-stream",
		NeedsAPI:    true,
		Description: "Stream AI chat completion",
		Run:         func(args []string) error { return runStream(deps, args) },
	})
	return cliapp.CommandGroup{Title: "Metered Inference", Commands: commands}
}

func intelligenceClient(deps support.Dependencies) (lpbsconnect.IntelligenceServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewIntelligenceServiceClient(httpClient, baseURL), nil
}

func intelligenceAction(deps support.Dependencies, description string, call func(context.Context, lpbsconnect.IntelligenceServiceClient) (proto.Message, error)) cliapp.Command {
	operation := cliapp.Action(func(cliapp.OperationContext) (json.RawMessage, error) {
		client, err := intelligenceClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := call(context.Background(), client)
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
	return (cliapp.Command{Name: "", NeedsAPI: true, Description: description, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func modelsCommand(deps support.Dependencies) cliapp.Command {
	command := intelligenceAction(deps, "List AI models through the generated Connect contract.", func(ctx context.Context, client lpbsconnect.IntelligenceServiceClient) (proto.Message, error) {
		response, err := client.ListModels(ctx, connect.NewRequest(&lpbsv1.ListModelsRequest{}))
		return response.Msg, err
	})
	command.Name = "ai-models"
	return command
}

func healthCommand(deps support.Dependencies) cliapp.Command {
	command := intelligenceAction(deps, "Get metered inference health through the generated Connect contract.", func(ctx context.Context, client lpbsconnect.IntelligenceServiceClient) (proto.Message, error) {
		response, err := client.Health(ctx, connect.NewRequest(&lpbsv1.HealthRequest{}))
		return response.Msg, err
	})
	command.Name = "ai-health"
	return command
}

func usageCommand(deps support.Dependencies) cliapp.Command {
	command := intelligenceAction(deps, "Get AI usage through the generated Connect contract.", func(ctx context.Context, client lpbsconnect.IntelligenceServiceClient) (proto.Message, error) {
		response, err := client.GetUsage(ctx, connect.NewRequest(&lpbsv1.GetUsageRequest{}))
		return response.Msg, err
	})
	command.Name = "ai-usage"
	return command
}

func chatCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(ctx cliapp.OperationContext) (json.RawMessage, error) {
		body, err := support.ParseBody(ctx.Flag("body"))
		if err != nil {
			return nil, err
		}
		if body == nil {
			return nil, fmt.Errorf("usage: ai-chat --body '{...}'")
		}
		request := &lpbsv1.ChatRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, request); err != nil {
			return nil, fmt.Errorf("decode chat request: %w", err)
		}
		client, err := intelligenceClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.Chat(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("AI chat", err, nil)
		}
		payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
		if err != nil {
			return nil, fmt.Errorf("encode AI chat: %w", err)
		}
		return json.RawMessage(payload), nil
	}, func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"AI chat completed."}}
	})
	return (cliapp.Command{Name: "ai-chat", NeedsAPI: true, Description: "Run AI chat through the generated Connect contract [--body JSON|@file|-]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "ChatRequest JSON payload or @file.json"}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func runStream(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("ai-stream", flag.ContinueOnError)
	body := fs.String("body", "", "JSON body payload or @file.json")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	payload, err := support.ParseBody(*body)
	if err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("usage: ai-stream --body '{...}'")
	}
	return deps.StreamEndpoint(support.EndpointDef{Method: "POST", Path: "/ai/stream"}, "/ai/stream", nil, payload)
}
