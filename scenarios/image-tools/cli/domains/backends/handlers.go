package backends

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client modelsconnect.ModelsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: modelsconnect.NewModelsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) doctor(ctx cliapp.RunContext) error {
	resp, err := h.client.DoctorBackends(context.Background(), connect.NewRequest(&modelsv1.DoctorBackendsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("doctor backends", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no backend doctor response")
	}
	results := make([]string, 0, len(resp.Msg.Backends))
	for _, b := range resp.Msg.Backends {
		results = append(results, formatBackend(b))
	}
	if len(results) == 0 {
		results = append(results, "no registered backends")
	}
	status := "Backend doctor passed."
	if !resp.Msg.Ok {
		status = fmt.Sprintf("Backend doctor found %d backend provisioning issue(s).", countUnavailableLocal(resp.Msg.Backends))
	}
	if err := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{status},
		ResultsHeading: "Backends",
		Results:        results,
		RetrievalHints: []string{
			"`models select <operation>` — preview model and backend selection for an operation",
			"Provision missing local backends through Scenario Dependency Analyzer; do not run raw package managers.",
		},
	}); err != nil {
		return err
	}
	if !resp.Msg.Ok {
		return fmt.Errorf("backend doctor failed")
	}
	return nil
}

func formatBackend(b *modelsv1.BackendStatus) string {
	availability := "missing"
	if b.Available {
		availability = "ready"
	}
	tier := "local"
	if b.Cloud {
		tier = "byok-cloud"
	} else if !b.Standalone {
		tier = "local-comfyui"
	}
	gpu := "cpu-only"
	if b.GpuCapable {
		gpu = "gpu-capable"
	}
	line := fmt.Sprintf("%s [%s, %s, %s] ops=%s — %s; provision: %s",
		b.Name,
		availability,
		tier,
		gpu,
		strings.Join(b.Operations, ","),
		b.Detail,
		b.Provision,
	)
	// Surface the exact remediation command for a missing host tool so the gap
	// is fixable straight from doctor output (Phase 6 doctor unification).
	if b.Remediation != "" {
		line += fmt.Sprintf("; remediation: %s", b.Remediation)
	}
	return line
}

// ensure installs a missing host-tool backend on demand (EnsureBackend → durable
// job, mirroring `models install`). Manual / capability-gated tools return
// guidance with no job.
func (h *handlers) ensure(ctx cliapp.RunContext) error {
	tool := ctx.Positional("tool")
	dryRun := ctx.BoolFlag("dry-run")
	resp, err := h.client.EnsureBackend(context.Background(), connect.NewRequest(&modelsv1.EnsureBackendRequest{
		Tool:   tool,
		DryRun: dryRun,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("ensure backend %q", tool), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no ensure response")
	}
	msg := resp.Msg
	switch {
	case msg.AlreadyInstalled:
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Backend tool %s is already installed.", msg.Tool)},
			Changes: []string{"no install job submitted"},
		})
	case msg.Manual:
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Backend tool %s needs a manual install (%s).", msg.Tool, msg.State)},
			Changes: []string{"no install job submitted"},
			NextCommand: []string{
				strings.TrimSpace(msg.Detail),
			},
		})
	case msg.JobId == "":
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Backend tool %s: %s.", msg.Tool, msg.State)},
			Changes: []string{strings.TrimSpace(msg.Detail)},
		})
	default:
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Installing backend tool %s (job %s, ~%ds).", msg.Tool, msg.JobId, msg.EtaSeconds)},
			Changes: []string{"durable install job submitted"},
			NextCommand: []string{
				fmt.Sprintf("`jobs wait %s` — block once on completion", msg.JobId),
			},
		})
	}
}

func countUnavailableLocal(backends []*modelsv1.BackendStatus) int {
	count := 0
	for _, b := range backends {
		if !b.Available && !b.Cloud {
			count++
		}
	}
	return count
}
