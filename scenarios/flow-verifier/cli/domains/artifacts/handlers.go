package artifacts

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts/artifacts_v1connect"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

type handlers struct {
	core      *cliapp.ScenarioApp
	artifacts artifactsconnect.ArtifactsServiceClient
	scenarios scenariosconnect.ScenariosServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:      core,
		artifacts: artifactsconnect.NewArtifactsServiceClient(httpClient, baseURL),
		scenarios: scenariosconnect.NewScenariosServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	flowID, scenarioID, _, err := readScope(ctx, false)
	if err != nil {
		return err
	}
	resp, err := h.artifacts.GetArtifactStatus(context.Background(), connect.NewRequest(&artifactsv1.GetArtifactStatusRequest{
		FlowId:     flowID,
		ScenarioId: scenarioID,
		Root:       ctx.Flag("root"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("artifacts status", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Artifact status for %s", resp.Msg.Report.FlowId)},
		ResultsHeading: "Files",
		Results:        formatReport(resp.Msg.Report),
	})
}

func (h *handlers) generate(ctx cliapp.RunContext) error {
	flowID, scenarioID, all, err := readScope(ctx, false)
	if err != nil {
		return err
	}
	if scenarioID != "" || all {
		return h.generateStream(ctx, scenarioID)
	}
	resp, err := h.artifacts.GenerateArtifacts(context.Background(), connect.NewRequest(&artifactsv1.GenerateArtifactsRequest{
		FlowId: flowID,
		Root:   ctx.Flag("root"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("artifacts generate", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Generated artifacts for %s.", resp.Msg.Report.FlowId)},
		Changes: formatReport(resp.Msg.Report),
	})
}

// generateStream consumes the server-streaming scenarios RPC and prints
// one line per flow as messages arrive — the CLI counterpart of the UI's
// live progress.
func (h *handlers) generateStream(ctx cliapp.RunContext, scenarioID string) error {
	if scenarioID == "" {
		return errors.New("--all requires --scenario <id> in the v1 cutover; pass a specific scenario")
	}
	stream, err := h.scenarios.GenerateScenarioArtifacts(context.Background(), connect.NewRequest(&scenariosv1.GenerateScenarioArtifactsRequest{ScenarioId: scenarioID}))
	if err != nil {
		return cliapp.WrapAPIError("artifacts generate --scenario", err, nil)
	}
	out := ctx.Stdout()
	count := 0
	for stream.Receive() {
		msg := stream.Msg()
		count++
		if msg.ErrorMessage != "" {
			fmt.Fprintf(out, "  %s — error: %s\n", msg.FlowId, msg.ErrorMessage)
		} else {
			fmt.Fprintf(out, "  %s — %s\n", msg.FlowId, msg.Report.Status)
		}
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError("artifacts generate stream", err, nil)
	}
	fmt.Fprintf(out, "Generated artifacts for %d flow(s).\n", count)
	return nil
}

func (h *handlers) clear(ctx cliapp.RunContext) error {
	flowID, scenarioID, all, err := readScope(ctx, true)
	if err != nil {
		return err
	}
	if scenarioID != "" || all {
		if !ctx.BoolFlag("yes") {
			return errors.New("--scenario / --all clear requires --yes")
		}
		resp, err := h.scenarios.ClearScenarioArtifacts(context.Background(), connect.NewRequest(&scenariosv1.ClearScenarioArtifactsRequest{ScenarioId: scenarioID}))
		if err != nil {
			return cliapp.WrapAPIError("artifacts clear --scenario", err, nil)
		}
		total := 0
		results := make([]string, 0, len(resp.Msg.Flows))
		for _, r := range resp.Msg.Flows {
			results = append(results, fmt.Sprintf("%s removed %d file(s)", r.FlowId, len(r.Removed)))
			total += len(r.Removed)
		}
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Cleared %d file(s) across %d flow(s).", total, len(resp.Msg.Flows))},
			Changes: results,
		})
	}
	resp, err := h.artifacts.ClearArtifacts(context.Background(), connect.NewRequest(&artifactsv1.ClearArtifactsRequest{
		FlowId: flowID,
		Root:   ctx.Flag("root"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("artifacts clear", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Cleared %d file(s) for %s.", len(resp.Msg.Removed), resp.Msg.FlowId)},
		Changes: resp.Msg.Removed,
	})
}

func readScope(ctx cliapp.RunContext, _ bool) (flowID, scenarioID string, all bool, err error) {
	flowID = ctx.Flag("flow")
	scenarioID = ctx.Flag("scenario")
	all = ctx.BoolFlag("all")
	count := 0
	for _, b := range []bool{flowID != "", scenarioID != "", all} {
		if b {
			count++
		}
	}
	if count == 0 {
		err = errors.New("specify exactly one of --flow, --scenario, or --all")
	} else if count > 1 {
		err = errors.New("--flow, --scenario, and --all are mutually exclusive")
	}
	return
}

func formatReport(r *artifactsv1.ArtifactReport) []string {
	if r == nil {
		return []string{"(no report)"}
	}
	results := []string{
		fmt.Sprintf("status        = %s", r.Status),
		fmt.Sprintf("generatedDir  = %s", r.GeneratedDir),
	}
	for _, f := range r.Files {
		results = append(results, fmt.Sprintf("  %s (exists=%t size=%d)", f.Path, f.Exists, f.Size))
	}
	return results
}
