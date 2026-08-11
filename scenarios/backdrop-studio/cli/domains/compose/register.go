package compose

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/compose/compose_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewComposeServiceClient(httpClient, baseURL)
	plan := cliapp.ProtoOperational(
		func(ctx cliapp.OperationContext) (*v1.ResolvedPlan, error) {
			styleID := strings.TrimSpace(ctx.Flag("style"))
			if styleID == "" {
				return nil, fmt.Errorf("compose: --style is required")
			}
			resp, err := client.ResolvePlan(context.Background(), connect.NewRequest(&v1.ResolvePlanRequest{Style: &v1.Style{Id: styleID, Strategy: ctx.Flag("strategy"), Treatments: strings.Split(ctx.Flag("treatments"), ","), Placements: []string{ctx.Flag("placement")}}, Brief: &v1.Brief{Placement: ctx.Flag("placement")}, AdapterCommercialUse: true}))
			if err != nil {
				return nil, cliapp.WrapAPIError("resolve compose plan", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, msg *v1.ResolvedPlan) cliapp.OperationalReport {
			return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Style %s strategy=%s path=%s executable=%t operations=%d.", msg.GetStyleId(), msg.GetStrategy(), msg.GetExpectedExecutionPath(), msg.GetExecutable(), len(msg.GetOperations()))}}
		},
	)
	deviceFrame := cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*v1.ComposeDeviceFrameResponse, error) {
			backdrop, err := os.ReadFile(ctx.Flag("backdrop"))
			if err != nil {
				return nil, fmt.Errorf("compose: read backdrop: %w", err)
			}
			screenshot, err := os.ReadFile(ctx.Flag("screenshot"))
			if err != nil {
				return nil, fmt.Errorf("compose: read screenshot: %w", err)
			}
			width, err := positiveInt32(ctx.Flag("width"))
			if err != nil {
				return nil, fmt.Errorf("compose: invalid width")
			}
			height, err := positiveInt32(ctx.Flag("height"))
			if err != nil {
				return nil, fmt.Errorf("compose: invalid height")
			}
			resp, err := client.ComposeDeviceFrame(context.Background(), connect.NewRequest(&v1.ComposeDeviceFrameRequest{BackdropPng: backdrop, ScreenshotPng: screenshot, SurfaceId: ctx.Flag("surface"), Arrangement: ctx.Flag("arrangement"), Caption: ctx.Flag("caption"), Width: width, Height: height}))
			if err != nil {
				return nil, cliapp.WrapAPIError("compose device frame", err, nil)
			}
			if err := os.WriteFile(ctx.Flag("output"), resp.Msg.GetImagePng(), 0o600); err != nil {
				return nil, fmt.Errorf("compose: write output: %w", err)
			}
			return resp.Msg, nil
		},
		func(ctx cliapp.OperationContext, msg *v1.ComposeDeviceFrameResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Composed %dx%d device frame to %s; occlusion=%s.", msg.GetWidth(), msg.GetHeight(), ctx.Flag("output"), msg.GetOcclusionRegion().GetKind())}}
		},
	)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "compose", map[string]cliapp.PrimitiveHandler{"ComposeService.ResolvePlan": plan, "ComposeService.ComposeDeviceFrame": deviceFrame})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("compose: load from manifest: %w", err)
	}
	return group, nil
}

func positiveInt32(raw string) (int32, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid positive int32 %q", raw)
	}
	return int32(n), nil
}
