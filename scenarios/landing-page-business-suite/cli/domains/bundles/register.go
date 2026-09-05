package bundles

import (
	"context"
	"fmt"
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
		{Name: "admin-bundle-price-create", Method: "POST", Path: "/admin/bundles/{bundle_key}/prices", Description: "Create bundle price"},
		{Name: "admin-bundle-price-delete", Method: "DELETE", Path: "/admin/bundles/{bundle_key}/prices/{price_id}", Description: "Delete bundle price"},
	})
	commands = append(commands, listCatalogCommand(deps), updatePriceCommand(deps))
	return cliapp.CommandGroup{Title: "Admin Commerce - Bundles", Commands: commands}
}

func client(deps support.Dependencies) (lpbsconnect.BundleAdminServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewBundleAdminServiceClient(httpClient, baseURL), nil
}

func listCatalogCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.ProtoList(
		func(cliapp.OperationContext) (*lpbsv1.ListBundleCatalogResponse, error) {
			service, err := client(deps)
			if err != nil {
				return nil, err
			}
			response, err := service.ListBundleCatalog(context.Background(), connect.NewRequest(&lpbsv1.ListBundleCatalogRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("list bundle catalog", err, nil)
			}
			return response.Msg, nil
		},
		func(cliapp.OperationContext, *lpbsv1.ListBundleCatalogResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{"Editable bundle catalog."}, ResultsHeading: "Bundles"}
		},
	)
	return (cliapp.Command{
		Name: "admin-bundles", NeedsAPI: true, Description: "List editable bundles through the generated Connect contract",
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList},
	}).WithPrimitive(operation)
}

func updatePriceCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*lpbsv1.UpdateBundlePriceResponse, error) {
			payload, err := support.ParseBody(op.Flag("body"))
			if err != nil {
				return nil, err
			}
			request := &lpbsv1.UpdateBundlePriceRequest{}
			if err := protojson.Unmarshal(payload, request); err != nil {
				return nil, fmt.Errorf("decode bundle price update: %w", err)
			}
			bundleKey, priceID := strings.TrimSpace(op.Positional("bundle_key")), strings.TrimSpace(op.Positional("price_id"))
			if bundleKey == "" || priceID == "" {
				return nil, fmt.Errorf("bundle_key and price_id are required")
			}
			if request.GetBundleKey() != "" && request.GetBundleKey() != bundleKey {
				return nil, fmt.Errorf("bundle_key in --body disagrees with positional bundle_key")
			}
			if request.GetPriceId() != "" && request.GetPriceId() != priceID {
				return nil, fmt.Errorf("price_id in --body disagrees with positional price_id")
			}
			request.BundleKey, request.PriceId = bundleKey, priceID
			service, err := client(deps)
			if err != nil {
				return nil, err
			}
			response, err := service.UpdateBundlePrice(context.Background(), connect.NewRequest(request))
			if err != nil {
				return nil, cliapp.WrapAPIError("update bundle price", err, nil)
			}
			return response.Msg, nil
		},
		func(cliapp.OperationContext, *lpbsv1.UpdateBundlePriceResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Bundle price updated."}}
		},
	)
	return (cliapp.Command{
		Name: "admin-bundle-price-update", NeedsAPI: true, Description: "Update bundle price through the generated Connect contract (BUNDLE_KEY PRICE_ID --body JSON)",
		Args:         cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "bundle_key", Required: true}, {Name: "price_id", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "Partial price JSON payload or @file.json", Required: true}}},
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation},
	}).WithPrimitive(operation)
}
