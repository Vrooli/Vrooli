package verify

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"
	verificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications/verifications_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client verificationsconnect.VerificationsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: verificationsconnect.NewVerificationsServiceClient(httpClient, baseURL)}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	return h.start(ctx, verificationsv1.VerificationMode_VERIFICATION_MODE_GENERATE)
}

func (h *handlers) check(ctx cliapp.RunContext) error {
	return h.start(ctx, verificationsv1.VerificationMode_VERIFICATION_MODE_CHECK)
}

func (h *handlers) start(ctx cliapp.RunContext, mode verificationsv1.VerificationMode) error {
	resp, err := h.client.StartVerification(context.Background(), connect.NewRequest(&verificationsv1.StartVerificationRequest{
		Root:   ctx.Flag("root"),
		FlowId: ctx.Flag("flow"),
		Mode:   mode,
	}))
	if err != nil {
		return cliapp.WrapAPIError("start verification", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Runs))
	for _, r := range resp.Msg.Runs {
		results = append(results, fmt.Sprintf("%s | %s | %s", r.FlowId, r.Status, r.Mode))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Verification %s — %d run(s).", resp.Msg.Status, len(resp.Msg.Runs))},
		ResultsHeading: "Runs",
		Results:        results,
	}
	if resp.Msg.ErrorMessage != "" {
		report.Summary = append(report.Summary, "error: "+resp.Msg.ErrorMessage)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("run-id")
	resp, err := h.client.GetVerification(context.Background(), connect.NewRequest(&verificationsv1.GetVerificationRequest{RunId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get verification %q", id), err, nil)
	}
	r := resp.Msg.Run
	if r == nil {
		return fmt.Errorf("server returned no run")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Verification %s — %s", r.Id, r.Status)},
		ResultsHeading: "Detail",
		Results: []string{
			fmt.Sprintf("flow       = %s", r.FlowId),
			fmt.Sprintf("mode       = %s", r.Mode),
			fmt.Sprintf("duration   = %dms", r.DurationMs),
		},
	})
}
