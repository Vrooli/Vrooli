package contract

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract/contract_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx
// func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client contractconnect.ContractServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: contractconnect.NewContractServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&contractv1.ValidateScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg

	summary := []string{fmt.Sprintf("%s: %s", msg.Scenario, msg.Status)}
	if msg.DegradedReason != "" {
		summary = append(summary, fmt.Sprintf("Degraded: %s", msg.DegradedReason))
	}
	results := make([]string, 0)
	if msg.Report != nil {
		for _, f := range msg.Report.Findings {
			results = append(results, fmt.Sprintf("[%s] %s — %s (%s)", f.Severity, f.Code, f.Message, f.Location))
		}
	}
	if len(results) == 0 {
		results = append(results, "No business-contract findings.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			"`business-health matrix show <scenario>` — traceability matrix",
			"`business-health fix preview <scenario>` — deterministic fixes as diffs",
		},
	})
}

// getMatrix renders the matrix join. One command, three renderings
// (manifest keeps a strict 1:1 procedure↔command mapping): the default
// table, `--format summary|markdown` (the report surface), and `--phase`
// (per-phase validation inspection).
func (h *handlers) getMatrix(ctx cliapp.RunContext) error {
	resp, err := h.client.GetMatrix(context.Background(), connect.NewRequest(&contractv1.GetMatrixRequest{
		Scenario: ctx.Positional("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get matrix", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no matrix response")
	}
	if phase := ctx.Flag("phase"); phase != "" {
		return h.renderPhase(ctx, resp.Msg, phase)
	}
	if format := ctx.Flag("format"); format != "" && format != "table" {
		return h.renderReport(ctx, resp.Msg, format)
	}
	results := make([]string, 0, len(resp.Msg.Matrix))
	for _, row := range resp.Msg.Matrix {
		marker := ""
		if row.Unproven {
			marker = " ⚠ " + row.UnprovenReason
		}
		results = append(results, fmt.Sprintf("%s ↔ %s [%s]%s", dash(row.OtId), dash(row.RequirementId), dash(row.RequirementStatus), marker))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d matrix row(s).", len(resp.Msg.Matrix))},
		ResultsHeading: "Matrix",
		Results:        results,
		RetrievalHints: []string{
			"`matrix show <scenario> --format summary` — rollup report",
			"`matrix show <scenario> --phase unit` — validations by phase",
		},
	})
}

func (h *handlers) getDrift(ctx cliapp.RunContext) error {
	resp, err := h.client.GetDrift(context.Background(), connect.NewRequest(&contractv1.GetDriftRequest{
		Scenario: ctx.Positional("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get drift", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no drift response")
	}
	results := make([]string, 0, len(resp.Msg.Drift))
	for _, d := range resp.Msg.Drift {
		results = append(results, fmt.Sprintf("%s: %s — %s", d.Kind, d.SubjectId, d.Detail))
	}
	if len(results) == 0 {
		results = append(results, "No evidence drift.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d drift entr(y/ies).", len(resp.Msg.Drift))},
		ResultsHeading: "Drift",
		Results:        results,
	})
}

func (h *handlers) logManual(ctx cliapp.RunContext) error {
	resp, err := h.client.LogManualValidation(context.Background(), connect.NewRequest(&contractv1.LogManualValidationRequest{
		Scenario:      ctx.Positional("scenario"),
		RequirementId: ctx.Positional("requirement"),
		AttestedBy:    ctx.Flag("by"),
		Notes:         ctx.Flag("notes"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("log manual validation", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no attestation")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Attestation recorded for %s.", resp.Msg.GetAttestation().GetRequirementId())},
		Changes: []string{fmt.Sprintf("Appended to %s", resp.Msg.GetLedgerPath())},
	})
}
