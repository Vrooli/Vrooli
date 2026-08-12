package render

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render/render_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

// brandTokens reads the repeatable --brand-token flag. It exists so a caller
// with no brand-manager can still bind ink slots directly, which is what makes
// the fail-closed slot contract usable from a bare shell.
func brandTokens(ctx cliapp.OperationContext) (map[string]string, error) {
	raw := ctx.FlagValues("brand-token")
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(raw))
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, value, ok := strings.Cut(entry, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !ok || name == "" || value == "" {
			return nil, fmt.Errorf("render: --brand-token expects <slot>=<value>, got %q", entry)
		}
		if !strings.HasPrefix(name, "$brand.") {
			name = "$brand." + strings.TrimPrefix(name, "brand.")
		}
		out[name] = value
	}
	return out, nil
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewRenderServiceClient(httpClient, baseURL)
	submit := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*v1.RenderJob, error) {
		seed, err := strconv.ParseInt(ctx.Flag("seed"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("render: invalid seed: %w", err)
		}
		count := int32(1)
		if raw := strings.TrimSpace(ctx.Flag("candidate-count")); raw != "" {
			n, e := strconv.ParseInt(raw, 10, 32)
			if e != nil {
				return nil, e
			}
			if n <= 0 || n > 2147483647 {
				return nil, fmt.Errorf("render: candidate-count must be between 1 and 2147483647")
			}
			count = int32(n)
		}
		tokens, err := brandTokens(ctx)
		if err != nil {
			return nil, err
		}
		resp, err := client.Submit(context.Background(), connect.NewRequest(&v1.SubmitRequest{Style: &sharedv1.Style{Id: ctx.Flag("style"), Strategy: ctx.Flag("strategy"), Subject: ctx.Flag("subject"), Placements: []string{ctx.Flag("placement")}, Treatments: strings.Split(ctx.Flag("treatments"), ",")}, Placement: ctx.Flag("placement"), Seed: seed, CandidateCount: count, BrandId: strings.TrimSpace(ctx.Flag("brand")), BrandTokens: tokens, SurfaceId: strings.TrimSpace(ctx.Flag("surface"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("submit render", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.RenderJob) cliapp.MutationReport {
		geometry := ""
		if len(msg.GetCandidates()) > 0 {
			geometry = fmt.Sprintf(" at %dx%d", msg.GetCandidates()[0].GetWidth(), msg.GetCandidates()[0].GetHeight())
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf(
			"Render %s completed with %d candidate(s) on surface %s%s; path=%s.",
			msg.GetId(), len(msg.GetCandidates()), msg.GetSurfaceId(), geometry, msg.GetExecutionPath())}}
	})
	get := cliapp.ProtoList(func(ctx cliapp.OperationContext) (*v1.RenderJob, error) {
		resp, err := client.GetJob(context.Background(), connect.NewRequest(&v1.GetJobRequest{JobId: ctx.Positional("job-id")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get render job", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.RenderJob) cliapp.ListReport {
		rows := []string{}
		for _, c := range msg.GetCandidates() {
			rows = append(rows, fmt.Sprintf("%s %dx%d seed=%d treated=%t", c.GetId(), c.GetWidth(), c.GetHeight(), c.GetSeed(), c.GetTreatmentApplied()))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Job %s status=%s selected=%s", msg.GetId(), msg.GetStatus(), msg.GetSelectedCandidateId())}, ResultsHeading: "Candidates", Results: rows}
	})
	candidates := cliapp.ProtoList(func(ctx cliapp.OperationContext) (*v1.ListCandidatesResponse, error) {
		resp, err := client.ListCandidates(context.Background(), connect.NewRequest(&v1.ListCandidatesRequest{JobId: ctx.Positional("job-id")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list render candidates", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.ListCandidatesResponse) cliapp.ListReport {
		rows := make([]string, 0, len(msg.GetCandidates()))
		for _, c := range msg.GetCandidates() {
			rows = append(rows, fmt.Sprintf("%s %dx%d seed=%d treated=%t", c.GetId(), c.GetWidth(), c.GetHeight(), c.GetSeed(), c.GetTreatmentApplied()))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d candidate(s).", len(rows))}, ResultsHeading: "Candidates", Results: rows}
	})
	selectCandidate := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*v1.RenderJob, error) {
		resp, err := client.SelectCandidate(context.Background(), connect.NewRequest(&v1.SelectCandidateRequest{JobId: ctx.Positional("job-id"), CandidateId: ctx.Positional("candidate-id"), Actor: ctx.Flag("actor")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("select candidate", err, nil)
		}
		return resp.Msg, nil
	}, func(_ cliapp.OperationContext, msg *v1.RenderJob) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Selected %s by %s.", msg.GetSelectedCandidateId(), msg.GetSelectedBy())}}
	})
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "render", map[string]cliapp.PrimitiveHandler{"RenderService.Submit": submit, "RenderService.GetJob": get, "RenderService.ListCandidates": candidates, "RenderService.SelectCandidate": selectCandidate})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("render: load manifest: %w", err)
	}
	return group, nil
}
