package surfaces

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces/surfaces_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewSurfacesServiceClient(httpClient, baseURL)
	list := cliapp.ProtoList(
		func(_ cliapp.OperationContext) (*v1.ListSurfacesResponse, error) {
			resp, err := client.ListSurfaces(context.Background(), connect.NewRequest(&v1.ListSurfacesRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("list surfaces", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, msg *v1.ListSurfacesResponse) cliapp.ListReport {
			rows := make([]string, 0, len(msg.GetSurfaces()))
			for _, s := range msg.GetSurfaces() {
				rows = append(rows, fmt.Sprintf("%s %dx%d kind=%s placements=%v authority=%s", s.GetId(), s.GetWidth(), s.GetHeight(), s.GetKind(), s.GetPlacements(), s.GetAuthority()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d surfaces.", len(rows))}, ResultsHeading: "Surfaces", Results: rows}
		},
	)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "surfaces", map[string]cliapp.PrimitiveHandler{
		"SurfacesService.ListSurfaces": list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("surfaces: load from manifest: %w", err)
	}
	return group, nil
}
