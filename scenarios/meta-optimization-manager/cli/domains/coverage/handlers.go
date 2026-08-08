package coverage

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage/coverage_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client coverageconnect.CoverageServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: coverageconnect.NewCoverageServiceClient(httpClient, baseURL)}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	proj, err := projectionFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.GetStatus(context.Background(), connect.NewRequest(&coveragev1.GetStatusRequest{Projection: proj}))
	if err != nil {
		return cliapp.WrapAPIError("coverage status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	results := make([]string, 0, len(resp.Msg.Projections))
	for _, pc := range resp.Msg.Projections {
		results = append(results, formatProjection(pc))
	}
	summary := []string{fmt.Sprintf("Readiness across %d projection(s).", len(resp.Msg.Projections))}
	if t := resp.Msg.GetLatestTrialTrend(); t != nil {
		summary = append(summary, fmt.Sprintf("Latest trial trend: success=%.0f%% tokens=%d.", t.GetSuccessRate()*100, t.GetMedianTokens()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Projections",
		Results:        results,
		RetrievalHints: []string{
			"`coverage list-cells --projection answer` — drill into a projection's cells",
			"`coverage validate-docs` — run the self-honesty gate",
		},
	})
}

func (h *handlers) listCells(ctx cliapp.RunContext) error {
	proj, err := projectionFlag(ctx)
	if err != nil {
		return err
	}
	status, err := statusFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ListCells(context.Background(), connect.NewRequest(&coveragev1.ListCellsRequest{Projection: proj, Status: status}))
	if err != nil {
		return cliapp.WrapAPIError("coverage list-cells", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cells response")
	}
	results := make([]string, 0, len(resp.Msg.Cells))
	for _, c := range resp.Msg.Cells {
		results = append(results, formatCell(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d cell(s).", len(resp.Msg.Cells))},
		ResultsHeading: "Cells",
		Results:        results,
		RetrievalHints: []string{"`coverage explain-cell <id>` — provenance for one cell"},
	})
}

func (h *handlers) explainCell(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("cell-id"))
	resp, err := h.client.ExplainCell(context.Background(), connect.NewRequest(&coveragev1.ExplainCellRequest{CellId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("coverage explain-cell %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Cell == nil {
		return fmt.Errorf("server returned no cell")
	}
	c := resp.Msg.Cell
	results := []string{formatCell(c)}
	for _, cite := range c.GetCitations() {
		results = append(results, fmt.Sprintf("  ↳ %s [%s] %s", cite.GetLocator(), cite.GetKind(), cite.GetNote()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Cell %s.", c.GetId())},
		ResultsHeading: "Cell + provenance",
		Results:        results,
	})
}

func (h *handlers) validateDocs(ctx cliapp.RunContext) error {
	proj, err := projectionFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ValidateBaseDocs(context.Background(), connect.NewRequest(&coveragev1.ValidateBaseDocsRequest{Projection: proj}))
	if err != nil {
		return cliapp.WrapAPIError("coverage validate-docs", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	results := make([]string, 0, len(resp.Msg.Issues))
	for _, is := range resp.Msg.Issues {
		results = append(results, fmt.Sprintf("[%s] %s — %s (%s)", severityLabel(is.GetSeverity()), is.GetCode(), is.GetMessage(), is.GetLocation()))
	}
	verdict := "ok"
	if !resp.Msg.GetOk() {
		verdict = "FAILED (ERROR-severity issue present)"
	}
	if rerr := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Base-doc gate: %s. %d issue(s).", verdict, len(resp.Msg.Issues))},
		ResultsHeading: "Issues",
		Results:        results,
	}); rerr != nil {
		return rerr
	}
	if !resp.Msg.GetOk() {
		// The self-honesty gate: a non-ok report exits non-zero.
		return fmt.Errorf("base-doc validation failed: %d issue(s)", len(resp.Msg.Issues))
	}
	return nil
}

// projectionFlag maps the --projection flag to the shared proto enum. Empty =>
// UNSPECIFIED (all projections). An unknown value is a usage error, never a
// silent all-projections fallback.
func projectionFlag(ctx cliapp.RunContext) (sharedv1.Projection, error) {
	switch strings.ToLower(strings.TrimSpace(ctx.Flag("projection"))) {
	case "":
		return sharedv1.Projection_PROJECTION_UNSPECIFIED, nil
	case "answer":
		return sharedv1.Projection_PROJECTION_ANSWER, nil
	case "validate":
		return sharedv1.Projection_PROJECTION_VALIDATE, nil
	case "guide":
		return sharedv1.Projection_PROJECTION_GUIDE, nil
	case "act":
		return sharedv1.Projection_PROJECTION_ACT, nil
	default:
		return 0, fmt.Errorf("unknown projection %q (use answer|validate|guide|act)", ctx.Flag("projection"))
	}
}

func statusFlag(ctx cliapp.RunContext) (sharedv1.CellStatus, error) {
	switch strings.ToLower(strings.TrimSpace(ctx.Flag("status"))) {
	case "":
		return sharedv1.CellStatus_CELL_STATUS_UNSPECIFIED, nil
	case "now":
		return sharedv1.CellStatus_CELL_STATUS_NOW, nil
	case "in_reach", "in-reach":
		return sharedv1.CellStatus_CELL_STATUS_IN_REACH, nil
	case "missing":
		return sharedv1.CellStatus_CELL_STATUS_MISSING, nil
	default:
		return 0, fmt.Errorf("unknown status %q (use now|in_reach|missing)", ctx.Flag("status"))
	}
}

func formatProjection(pc *coveragev1.ProjectionCoverage) string {
	label := projectionLabel(pc.GetProjection())
	if !pc.GetAvailable() {
		// The live numerator join failed, so NowCount/CoverageRatio are an
		// authored fallback, not a measurement. Printing a coverage % here reads
		// as "measured" when it was not — show a dash and the honest reason. The
		// denominator (authored cell count) is still real, so surface it.
		return fmt.Sprintf("%s: — (denominator=%d, confidence=%s) [UNAVAILABLE: %s]",
			label, pc.GetTotalCells(), confidenceLabel(pc.GetDenominatorConfidence()), pc.GetUnavailableReason())
	}
	return fmt.Sprintf("%s: %.0f%% NOW (%d/%d) — in_reach=%d missing=%d — confidence=%s",
		label, pc.GetCoverageRatio()*100, pc.GetNowCount(), pc.GetTotalCells(),
		pc.GetInReachCount(), pc.GetMissingCount(), confidenceLabel(pc.GetDenominatorConfidence()))
}

func formatCell(c *coveragev1.Cell) string {
	return fmt.Sprintf("%s [%s] %s — %s", c.GetId(), statusLabel(c.GetStatus()), c.GetQuestion(), c.GetOwner())
}

func projectionLabel(p sharedv1.Projection) string {
	switch p {
	case sharedv1.Projection_PROJECTION_ANSWER:
		return "answer"
	case sharedv1.Projection_PROJECTION_VALIDATE:
		return "validate"
	case sharedv1.Projection_PROJECTION_GUIDE:
		return "guide"
	default:
		return "unspecified"
	}
}

func statusLabel(s sharedv1.CellStatus) string {
	switch s {
	case sharedv1.CellStatus_CELL_STATUS_NOW:
		return "now"
	case sharedv1.CellStatus_CELL_STATUS_IN_REACH:
		return "in_reach"
	case sharedv1.CellStatus_CELL_STATUS_MISSING:
		return "missing"
	default:
		return "?"
	}
}

func confidenceLabel(c sharedv1.DenominatorConfidence) string {
	switch c {
	case sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_AUTHORITATIVE:
		return "authoritative"
	case sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_PARTIAL:
		return "partial"
	case sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_SKETCH:
		return "sketch"
	default:
		return "unspecified"
	}
}

func severityLabel(s sharedv1.Severity) string {
	switch s {
	case sharedv1.Severity_SEVERITY_ERROR:
		return "ERROR"
	case sharedv1.Severity_SEVERITY_WARN:
		return "WARN"
	case sharedv1.Severity_SEVERITY_INFO:
		return "INFO"
	default:
		return "?"
	}
}
