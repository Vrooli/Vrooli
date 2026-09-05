package shapes

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	shapesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shapes"
	shapesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shapes/shapes_v1connect"
)

const GroupName = "shapes"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := shapesconnect.NewShapeServiceClient(httpClient, baseURL)
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ShapeService.ListShapes": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*shapesv1.ListShapesResponse, error) {
			req := &shapesv1.ListShapesRequest{UncoveredOnly: ctx.BoolFlag("uncovered")}
			if value, err := strconv.ParseInt(ctx.Flag("min-occurrences"), 10, 64); err == nil {
				req.MinOccurrences = value
			}
			if value, err := strconv.ParseInt(ctx.Flag("min-sessions"), 10, 64); err == nil {
				req.MinSessions = value
			}
			if value, err := strconv.ParseInt(ctx.Flag("limit"), 10, 32); err == nil {
				req.Limit = int32(value)
			}
			r, err := client.ListShapes(context.Background(), connect.NewRequest(req))
			if err != nil {
				return nil, cliapp.WrapAPIError("list shapes", err, nil)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, r *shapesv1.ListShapesResponse) cliapp.ListReport {
			results := make([]string, 0, len(r.GetShapes()))
			for _, shape := range r.GetShapes() {
				results = append(results, fmt.Sprintf("%s [%s] occurrences=%d sessions=%d bindings=%s", shape.GetShapeKey(), shape.GetState(), shape.GetOccurrences(), shape.GetSessions(), strings.Join(shape.GetBindingIds(), ",")))
			}
			if len(results) == 0 {
				results = []string{"No program shapes matched the filters."}
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Program shapes: %d", len(r.GetShapes()))}, ResultsHeading: "Shapes", Results: results, ListShaped: true, ResultCount: len(r.GetShapes())}
		}),
		"ShapeService.GetShape": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*shapesv1.GetShapeResponse, error) {
			r, err := client.GetShape(context.Background(), connect.NewRequest(&shapesv1.GetShapeRequest{ShapeKey: ctx.Positional("shape-key")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get shape", err, nil)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, r *shapesv1.GetShapeResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{"Shape: " + r.GetShape().GetShapeKey()}, Results: []string{r.GetShape().String()}, ResultCount: 1}
		}),
		"ShapeService.ExpireShapes": cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*shapesv1.ExpireShapesResponse, error) {
			r, err := client.ExpireShapes(context.Background(), connect.NewRequest(&shapesv1.ExpireShapesRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("expire shapes", err, nil)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, r *shapesv1.ExpireShapesResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Expired shapes: %d", r.GetDeleted())}}
		}),
	})
}
