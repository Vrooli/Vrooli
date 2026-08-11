package catalog

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/catalog"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/catalog/catalog_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewCatalogServiceClient(httpClient, baseURL)
	list := cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*v1.ListStylesResponse, error) {
			req := &v1.ListStylesRequest{}
			axis := strings.TrimSpace(ctx.Flag("axis"))
			if axis != "" {
				parts := strings.SplitN(axis, "=", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("catalog: invalid axis %q; want name=value", axis)
				}
				switch parts[0] {
				case "role":
					req.Role = parts[1]
				case "subject":
					req.Subject = parts[1]
				case "treatment":
					req.Treatment = parts[1]
				case "lineage":
					req.Lineage = parts[1]
				case "placement":
					req.Placement = parts[1]
				default:
					return nil, fmt.Errorf("catalog: invalid axis %q", parts[0])
				}
			}
			resp, err := client.ListStyles(context.Background(), connect.NewRequest(req))
			if err != nil {
				return nil, cliapp.WrapAPIError("list styles", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, msg *v1.ListStylesResponse) cliapp.ListReport {
			rows := make([]string, 0, len(msg.GetStyles()))
			for _, s := range msg.GetStyles() {
				rows = append(rows, fmt.Sprintf("%s v%d role=%s subject=%s strategy=%s treatments=%v placements=%v", s.GetId(), s.GetVersion(), s.GetRole(), s.GetSubject(), s.GetStrategy(), s.GetTreatments(), s.GetPlacements()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d styles.", len(rows))}, ResultsHeading: "Styles", Results: rows}
		},
	)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "catalog", map[string]cliapp.PrimitiveHandler{
		"CatalogService.ListStyles": list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("catalog: load from manifest: %w", err)
	}
	return group, nil
}
