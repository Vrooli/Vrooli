package landing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := deps.EndpointCommands([]support.EndpointDef{
		{Name: "plans", Method: "GET", Path: "/plans", Description: "List pricing plans"},
		{Name: "customize", Method: "POST", Path: "/customize", Description: "Customize landing content"},
	})
	commands = append(commands, landingConfigCommand(deps), variantSpaceCommand(deps))
	return cliapp.CommandGroup{
		Title:    "Landing",
		Commands: commands,
	}
}

func landingConfigClient(deps support.Dependencies) (lpbsconnect.LandingConfigServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewLandingConfigServiceClient(httpClient, baseURL), nil
}

func landingConfigCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(
		func(op cliapp.OperationContext) (json.RawMessage, error) {
			variantSlug, err := landingConfigVariantSlug(op)
			if err != nil {
				return nil, err
			}
			client, err := landingConfigClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.GetLandingConfig(context.Background(), connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: variantSlug}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get landing configuration", err, nil)
			}
			payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
			if err != nil {
				return nil, fmt.Errorf("encode landing configuration: %w", err)
			}
			return json.RawMessage(payload), nil
		},
		func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Fetched landing configuration."}}
		},
	)
	return (cliapp.Command{
		Name:        "landing-config",
		NeedsAPI:    true,
		Description: "Fetch landing configuration through the generated Connect contract [--variant SLUG] [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "variant", Description: "Variant slug to render"},
			{Name: "query", Description: "Legacy query compatibility; only variant=SLUG is accepted"},
		}},
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction},
	}).WithPrimitive(operation)
}

func landingConfigVariantSlug(op cliapp.OperationContext) (string, error) {
	variant := strings.TrimSpace(op.Flag("variant"))
	legacyQuery := strings.TrimSpace(op.Flag("query"))
	if legacyQuery == "" {
		return variant, nil
	}
	query, err := url.ParseQuery(legacyQuery)
	if err != nil {
		return "", fmt.Errorf("parse legacy --query: %w", err)
	}
	for key := range query {
		if key != "variant" {
			return "", fmt.Errorf("legacy --query only supports variant=SLUG")
		}
	}
	legacyVariant := strings.TrimSpace(query.Get("variant"))
	if variant != "" && legacyVariant != "" && variant != legacyVariant {
		return "", fmt.Errorf("--variant and legacy --query variant disagree")
	}
	if variant != "" {
		return variant, nil
	}
	return legacyVariant, nil
}

func variantSpaceClient(deps support.Dependencies) (lpbsconnect.VariantSpaceServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewVariantSpaceServiceClient(httpClient, baseURL), nil
}

func variantSpaceCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(
		func(cliapp.OperationContext) (json.RawMessage, error) {
			client, err := variantSpaceClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.GetVariantSpace(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSpaceRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get variant space", err, nil)
			}
			payload := response.Msg.GetRawJson()
			if !json.Valid(payload) {
				return nil, fmt.Errorf("variant space response is not valid JSON")
			}
			return json.RawMessage(append([]byte(nil), payload...)), nil
		},
		func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Fetched variant space."}}
		},
	)
	return (cliapp.Command{
		Name:         "variant-space",
		NeedsAPI:     true,
		Description:  "Fetch variant space through the generated Connect contract",
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction},
	}).WithPrimitive(operation)
}
