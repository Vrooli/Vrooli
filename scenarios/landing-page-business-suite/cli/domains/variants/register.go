package variants

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := []cliapp.Command{
		selectCommand(deps),
		publicVariantCommand(deps),
		listVariantsCommand(deps),
		getVariantCommand(deps),
		updateVariantCommand(deps),
		deleteVariantCommand(deps),
		exportVariantCommand(deps),
		importVariantCommand(deps),
		syncVariantsCommand(deps),
	}
	commands = append(commands, deps.EndpointCommands([]support.EndpointDef{
		{Name: "public-variant-sections", Method: "GET", Path: "/public/variants/{variant_slug}/sections", Description: "Get public sections for a variant"},
		{Name: "variants-sections", Method: "GET", Path: "/variants/{variant_slug}/sections", Description: "Get variant sections (admin)"},
	})...)
	return cliapp.CommandGroup{Title: "Variants", Commands: commands}
}

func variantClient(deps support.Dependencies) (lpbsconnect.VariantServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewVariantServiceClient(httpClient, baseURL), nil
}

func adminVariantClient(deps support.Dependencies) (lpbsconnect.VariantServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewVariantServiceClient(httpClient, baseURL), nil
}

func variantResponse(message proto.Message) (map[string]any, error) {
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("encode variant response: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode variant response: %w", err)
	}
	return result, nil
}

func selectCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(cliapp.OperationContext) (map[string]any, error) {
		client, err := variantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.SelectVariant(context.Background(), connect.NewRequest(&lpbsv1.SelectVariantRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("select variant", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Selected a landing variant through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "variants-select", NeedsAPI: true, Description: "Select a variant through the generated Connect contract [--json]", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func publicVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		slug := strings.TrimSpace(op.Positional("slug"))
		client, err := variantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.GetPublicVariant(context.Background(), connect.NewRequest(&lpbsv1.GetPublicVariantRequest{Slug: slug}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get public variant", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Fetched a public variant through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "public-variant", NeedsAPI: true, Description: "Get a public variant through the generated Connect contract (SLUG) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func listVariantsCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ListVariants(context.Background(), connect.NewRequest(&lpbsv1.ListVariantsRequest{StatusFilter: strings.TrimSpace(op.Flag("status"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list variants", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Listed variants through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "variants-list", NeedsAPI: true, Description: "List admin variants through the generated Connect contract [--status active|archived] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "status", Description: "Optional status filter: active or archived"}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func getVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.GetVariant(context.Background(), connect.NewRequest(&lpbsv1.GetVariantRequest{Slug: strings.TrimSpace(op.Positional("slug"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get variant", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Fetched an admin variant through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "variants-get", NeedsAPI: true, Description: "Get an admin variant through the generated Connect contract (SLUG) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func deleteVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.DeleteVariant(context.Background(), connect.NewRequest(&lpbsv1.DeleteVariantRequest{Slug: strings.TrimSpace(op.Positional("slug"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete variant", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Deleted a variant through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "variants-delete", NeedsAPI: true, Description: "Delete a variant through the generated Connect contract (SLUG) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func updateVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		body, err := support.ParseBody(op.Flag("body"))
		if err != nil {
			return nil, err
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("--body JSON payload is required")
		}
		request, err := legacyUpdateRequest(strings.TrimSpace(op.Positional("slug")), body)
		if err != nil {
			return nil, err
		}
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.UpdateVariant(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("update variant", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Updated a variant through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "variants-update", NeedsAPI: true, Description: "Update a variant through the generated Connect contract (SLUG --body JSON) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "Legacy variant update JSON payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

// legacyUpdateRequest preserves the CLI's established JSON payload while
// translating its flat axes map into the generated AxesSelection message.
func legacyUpdateRequest(slug string, body []byte) (*lpbsv1.UpdateVariantRequest, error) {
	if slug == "" {
		return nil, fmt.Errorf("variant slug is required")
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse variant update payload: %w", err)
	}
	if axes, ok := payload["axes"]; ok {
		var axesMap map[string]string
		if err := json.Unmarshal(axes, &axesMap); err == nil {
			wrapped, err := json.Marshal(map[string]map[string]string{"values": axesMap})
			if err != nil {
				return nil, fmt.Errorf("encode variant axes: %w", err)
			}
			payload["axes"] = wrapped
		}
	}
	payload["slug"] = json.RawMessage(fmt.Sprintf("%q", slug))
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode variant update payload: %w", err)
	}
	request := &lpbsv1.UpdateVariantRequest{}
	decoder := protojson.UnmarshalOptions{DiscardUnknown: false}
	if err := decoder.Unmarshal(normalized, request); err != nil {
		return nil, fmt.Errorf("decode variant update payload: %w", err)
	}
	return request, nil
}

func exportVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ExportVariantSnapshot(context.Background(), connect.NewRequest(&lpbsv1.ExportVariantSnapshotRequest{Slug: strings.TrimSpace(op.Positional("slug"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("export variant snapshot", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Exported a variant snapshot through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "admin-variants-export", NeedsAPI: true, Description: "Export a variant snapshot through the generated Connect contract (SLUG) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func importVariantCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(op cliapp.OperationContext) (map[string]any, error) {
		body, err := support.ParseBody(op.Flag("body"))
		if err != nil {
			return nil, err
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("--body JSON payload is required")
		}
		request, err := legacyImportRequest(strings.TrimSpace(op.Positional("slug")), body)
		if err != nil {
			return nil, err
		}
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.ImportVariantSnapshot(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("import variant snapshot", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Imported a variant snapshot through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "admin-variants-import", NeedsAPI: true, Description: "Import a variant snapshot through the generated Connect contract (SLUG --body JSON) [--json]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "Variant snapshot JSON payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

func syncVariantsCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(func(cliapp.OperationContext) (map[string]any, error) {
		client, err := adminVariantClient(deps)
		if err != nil {
			return nil, err
		}
		response, err := client.SyncVariantSnapshots(context.Background(), connect.NewRequest(&lpbsv1.SyncVariantSnapshotsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("sync variant snapshots", err, nil)
		}
		return variantResponse(response.Msg)
	}, func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Reloaded variant snapshots through the generated Connect contract."}}
	})
	return (cliapp.Command{Name: "admin-variants-sync", NeedsAPI: true, Description: "Reload persisted variant snapshots through the generated Connect contract [--json]", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(operation)
}

// legacyImportRequest preserves the existing snapshot-file shape: metadata is
// nested under "variant" while sections remain a sibling array. The generated
// contract represents the same information as one VariantSnapshot message.
func legacyImportRequest(slug string, body []byte) (*lpbsv1.ImportVariantSnapshotRequest, error) {
	if slug == "" {
		return nil, fmt.Errorf("variant slug is required")
	}
	var payload struct {
		Variant  json.RawMessage `json:"variant"`
		Sections json.RawMessage `json:"sections"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse variant snapshot payload: %w", err)
	}
	if len(payload.Variant) == 0 || string(payload.Variant) == "null" {
		return nil, fmt.Errorf("variant snapshot payload requires a variant object")
	}
	var snapshotPayload map[string]json.RawMessage
	if err := json.Unmarshal(payload.Variant, &snapshotPayload); err != nil {
		return nil, fmt.Errorf("parse variant snapshot metadata: %w", err)
	}
	var payloadSlug string
	if rawSlug, ok := snapshotPayload["slug"]; ok {
		if err := json.Unmarshal(rawSlug, &payloadSlug); err != nil {
			return nil, fmt.Errorf("parse variant snapshot slug: %w", err)
		}
	}
	if payloadSlug != slug {
		return nil, fmt.Errorf("payload slug does not match positional slug")
	}
	if len(payload.Sections) > 0 && string(payload.Sections) != "null" {
		snapshotPayload["sections"] = payload.Sections
	}
	normalized, err := json.Marshal(snapshotPayload)
	if err != nil {
		return nil, fmt.Errorf("encode variant snapshot payload: %w", err)
	}
	snapshot := &lpbsv1.VariantSnapshot{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(normalized, snapshot); err != nil {
		return nil, fmt.Errorf("decode variant snapshot payload: %w", err)
	}
	return &lpbsv1.ImportVariantSnapshotRequest{Slug: slug, Snapshot: snapshot}, nil
}
