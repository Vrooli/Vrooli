package scaffold

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/scaffold"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/scaffold/scaffold_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewScaffoldServiceClient(httpClient, baseURL)
	list := cliapp.ProtoList(
		func(_ cliapp.OperationContext) (*v1.ListPresetsResponse, error) {
			resp, err := client.ListPresets(context.Background(), connect.NewRequest(&v1.ListPresetsRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("list scaffold presets", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, msg *v1.ListPresetsResponse) cliapp.ListReport {
			rows := make([]string, 0, len(msg.GetPresets()))
			for _, p := range msg.GetPresets() {
				rows = append(rows, fmt.Sprintf("%s — %s (%s), params=%v", p.GetId(), p.GetName(), p.GetSubject(), p.GetParameters()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d scaffold presets.", len(rows))}, ResultsHeading: "Presets", Results: rows}
		},
	)
	render := cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*v1.RenderResponse, error) {
			width, err := positiveInt(ctx.Flag("width"), 1024)
			if err != nil {
				return nil, err
			}
			height, err := positiveInt(ctx.Flag("height"), 576)
			if err != nil {
				return nil, err
			}
			seed, err := strconv.ParseInt(ctx.Flag("seed"), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("scaffold: invalid seed: %w", err)
			}
			resp, err := client.Render(context.Background(), connect.NewRequest(&v1.RenderRequest{Preset: ctx.Flag("preset"), Width: width, Height: height, Seed: seed, Conditioner: ctx.Flag("conditioner"), ParamsJson: ctx.Flag("params-json")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("render scaffold", err, nil)
			}
			return resp.Msg, nil
		},
		func(_ cliapp.OperationContext, msg *v1.RenderResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Rendered %d×%d scaffold (%s), sha256=%s.", msg.GetWidth(), msg.GetHeight(), msg.GetConditioner(), msg.GetSha256())}}
		},
	)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "scaffold", map[string]cliapp.PrimitiveHandler{"ScaffoldService.ListPresets": list, "ScaffoldService.Render": render})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("scaffold: load from manifest: %w", err)
	}
	return group, nil
}

func positiveInt(raw string, fallback int32) (int32, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("scaffold: invalid dimension %q", raw)
	}
	return int32(n), nil
}
