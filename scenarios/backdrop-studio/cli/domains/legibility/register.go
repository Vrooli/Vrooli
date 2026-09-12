package legibility

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/legibility"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/legibility/legibility_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewLegibilityServiceClient(httpClient, baseURL)
	measure := cliapp.ProtoOperational(func(ctx cliapp.OperationContext) (*v1.Verdict, error) {
		img, err := os.ReadFile(ctx.Flag("image"))
		if err != nil {
			return nil, err
		}
		threshold := 4.5
		if raw := ctx.Flag("threshold"); raw != "" {
			threshold, err = strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, err
			}
		}
		resp, err := client.Measure(context.Background(), connect.NewRequest(&v1.MeasureRequest{ImagePng: img, Threshold: threshold, Placement: ctx.Flag("placement")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("measure legibility", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.Verdict) cliapp.OperationalReport {
		return cliapp.OperationalReport{Status: []string{fmt.Sprintf("passes=%t minimum=%.3f threshold=%.3f amendments=%d", msg.GetPasses(), msg.GetMinimumRatio(), msg.GetThreshold(), len(msg.GetAmendments()))}}
	})
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "legibility", map[string]cliapp.PrimitiveHandler{"LegibilityService.Measure": measure})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("legibility: load manifest: %w", err)
	}
	return group, nil
}
