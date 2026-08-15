package release

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewReleaseServiceClient(httpClient, baseURL)
	release := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*v1.ReleasedBackdrop, error) {
		w, e := positiveInt32(ctx.Flag("width"))
		if e != nil {
			return nil, e
		}
		h, e := positiveInt32(ctx.Flag("height"))
		if e != nil {
			return nil, e
		}
		decorative := false
		if raw := ctx.Flag("decorative"); raw != "" {
			decorative, e = strconv.ParseBool(raw)
			if e != nil {
				return nil, fmt.Errorf("release: invalid decorative value: %w", e)
			}
		}
		resp, e := client.Release(context.Background(), connect.NewRequest(&v1.ReleaseRequest{CandidateId: ctx.Flag("candidate"), StyleId: ctx.Flag("style"), Strategy: ctx.Flag("strategy"), SurfaceId: ctx.Flag("surface"), Width: w, Height: h, ExpectedWidth: w, ExpectedHeight: h, Placement: ctx.Flag("placement"), AltText: ctx.Flag("alt-text"), Decorative: decorative, LegibilityPasses: true}))
		if e != nil {
			return nil, cliapp.WrapAPIError("release backdrop", e, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.ReleasedBackdrop) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Released %s (%dx%d, ai_generated=%t).", msg.GetId(), msg.GetWidth(), msg.GetHeight(), msg.GetAiGenerated())}}
	})
	get := cliapp.ProtoList(func(ctx cliapp.OperationContext) (*v1.ReleasedBackdrop, error) {
		resp, e := client.GetReference(context.Background(), connect.NewRequest(&v1.GetReferenceRequest{Id: ctx.Positional("id")}))
		if e != nil {
			return nil, cliapp.WrapAPIError("get backdrop reference", e, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.ReleasedBackdrop) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s %dx%d style=%s alt=%q", msg.GetId(), msg.GetWidth(), msg.GetHeight(), msg.GetStyleId(), msg.GetAltText())}}
	})
	group, e := cliapp.LoadFromManifestPrimitives(manifest, "release", map[string]cliapp.PrimitiveHandler{"ReleaseService.Release": release, "ReleaseService.GetReference": get})
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("release: load manifest: %w", e)
	}
	return group, nil
}

func positiveInt32(raw string) (int32, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("release: invalid positive int32 %q", raw)
	}
	return int32(n), nil
}
