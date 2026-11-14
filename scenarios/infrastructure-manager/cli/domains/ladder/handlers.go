package ladder

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
	ladderconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder/ladder_v1connect"
)

type handlers struct {
	client ladderconnect.LadderServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: ladderconnect.NewLadderServiceClient(httpClient, baseURL)}
}

func (h *handlers) statusCall(_ cliapp.OperationContext) (*ladderv1.GetLadderResponse, error) {
	resp, err := h.client.GetLadder(context.Background(), connect.NewRequest(&ladderv1.GetLadderRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ladder status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetLadder() == nil {
		return nil, fmt.Errorf("server returned no ladder readout")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, msg *ladderv1.GetLadderResponse) cliapp.ListReport {
	ladder := msg.GetLadder()
	results := make([]string, 0, len(ladder.GetCells())+len(ladder.GetSources())+len(ladder.GetFindings()))
	for _, source := range ladder.GetSources() {
		results = append(results, formatSource(source))
	}
	for _, cell := range ladder.GetCells() {
		results = append(results, formatCell(cell))
	}
	for _, finding := range ladder.GetFindings() {
		results = append(results, formatFinding(finding))
	}
	summary := []string{
		fmt.Sprintf("%d ladder cell(s), %d source(s), %d ranked finding(s) on %s.", len(ladder.GetCells()), len(ladder.GetSources()), len(ladder.GetFindings()), ladder.GetHostOs()),
	}
	if !ladder.GetCoverageAvailable() {
		summary = append(summary, "Authored cell set UNAVAILABLE: "+ladder.GetCoverageReason())
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Ladder",
		Results:        results,
		RetrievalHints: []string{"`ladder cells --cell-ref substrate/SB11`", "`ladder sources`", "`ladder findings --stage sensor-channel-integrity`"},
	}
}

func (h *handlers) cellsCall(ctx cliapp.OperationContext) (*ladderv1.ListCellsResponse, error) {
	rung, err := rungFlag(ctx.Flag("rung"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListCells(context.Background(), connect.NewRequest(&ladderv1.ListCellsRequest{
		DeviceClass: strings.TrimSpace(ctx.Flag("device-class")),
		Rung:        rung,
		HostOs:      strings.TrimSpace(ctx.Flag("host-os")),
		CellRef:     strings.TrimSpace(ctx.Flag("cell-ref")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ladder cells", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no cells response")
	}
	return resp.Msg, nil
}

func (h *handlers) cellsReport(_ cliapp.OperationContext, msg *ladderv1.ListCellsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetCells()))
	for _, cell := range msg.GetCells() {
		results = append(results, formatCell(cell))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d ladder cell(s).", len(results))},
		ResultsHeading: "Cells",
		Results:        results,
	}
}

func (h *handlers) devicesCall(ctx cliapp.OperationContext) (*ladderv1.ListDevicesResponse, error) {
	resp, err := h.client.ListDevices(context.Background(), connect.NewRequest(&ladderv1.ListDevicesRequest{
		DeviceClass: strings.TrimSpace(ctx.Flag("device-class")),
		DeviceId:    strings.TrimSpace(ctx.Flag("device-id")),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ladder devices", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no devices response")
	}
	return resp.Msg, nil
}

func (h *handlers) devicesReport(_ cliapp.OperationContext, msg *ladderv1.ListDevicesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetDevices())*2)
	for _, device := range msg.GetDevices() {
		results = append(results, formatDevice(device)...)
	}
	summary := []string{fmt.Sprintf("%d device(s).", len(msg.GetDevices()))}
	if !msg.GetAvailable() {
		// An empty list from an unread source is a failure to observe, never a
		// host with no hardware, and the summary says which.
		summary = []string{"Device graph UNAVAILABLE: " + msg.GetUnavailableReason()}
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Devices", Results: results}
}

// formatDevice renders one device and its whole ladder. Every rung is printed,
// including the ones that could not be graded: a rung that disappears when its
// probe fails is the failure this board exists to catch.
func formatDevice(device *ladderv1.Device) []string {
	out := make([]string, 0, len(device.GetRungs())+1)
	out = append(out, fmt.Sprintf("%-28s %-20s %s %s (%s)",
		device.GetId(), device.GetClass(),
		firstNonEmpty(device.GetVendor(), "unknown-vendor"),
		firstNonEmpty(device.GetModel(), "unknown-model"),
		firstNonEmpty(device.GetDriver(), "no driver bound")))
	for _, rung := range device.GetRungs() {
		grade := enumToken(rung.GetObservation().String(), "OBSERVATION_")
		ladder := enumToken(rung.GetLadderObservation().String(), "OBSERVATION_")
		if ladder != grade {
			grade = fmt.Sprintf("%s->%s", grade, ladder)
		}
		detail := rung.GetReason()
		if rung.GetRemediation() != "" {
			detail = detail + " | fix: " + rung.GetRemediation()
		}
		out = append(out, fmt.Sprintf("  %-13s %-26s %-14s %s",
			enumToken(rung.GetRung().String(), "RUNG_"), grade, rung.GetMechanism(), detail))
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *handlers) sourcesCall(_ cliapp.OperationContext) (*ladderv1.ListSourcesResponse, error) {
	resp, err := h.client.ListSources(context.Background(), connect.NewRequest(&ladderv1.ListSourcesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ladder sources", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no sources response")
	}
	return resp.Msg, nil
}

func (h *handlers) sourcesReport(_ cliapp.OperationContext, msg *ladderv1.ListSourcesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetSources()))
	unavailable := 0
	for _, source := range msg.GetSources() {
		if !source.GetAvailable() {
			unavailable++
		}
		results = append(results, formatSource(source))
	}
	for _, coverage := range msg.GetCheckPlatforms() {
		results = append(results, formatCheckPlatform(coverage))
	}
	// The denominator confidence travels with the cells it qualifies. A ratio
	// over this denominator printed without it is not a valid response.
	confidence := msg.GetConfidence()
	denominator := "Denominator confidence UNAVAILABLE: " + confidence.GetReason()
	if confidence.GetAvailable() {
		denominator = fmt.Sprintf("Denominator confidence %s — %s",
			enumToken(confidence.GetLevel().String(), "CONFIDENCE_LEVEL_"), confidence.GetRationale())
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d source(s), %d unavailable.", len(msg.GetSources()), unavailable), denominator},
		ResultsHeading: "Sources and declared host-OS sensing",
		Results:        results,
	}
}

// formatCheckPlatform renders the declared substrate sensing per host OS as a
// triple. A bare "4 checks apply on windows" is unreadable without knowing the
// registry holds five checks or fifty.
func formatCheckPlatform(coverage *ladderv1.CheckPlatformCoverage) string {
	if !coverage.GetAvailable() {
		return fmt.Sprintf("%-38s %-12s %s", "autoheal checks on "+coverage.GetHostOs(), "UNAVAILABLE", coverage.GetReason())
	}
	return fmt.Sprintf("%-38s %d of %d registered check(s) apply (%d declare no platform and apply everywhere)",
		"autoheal checks on "+coverage.GetHostOs(),
		coverage.GetApplicable(), coverage.GetTotal(), coverage.GetUniversal())
}

func (h *handlers) findingsCall(ctx cliapp.OperationContext) (*ladderv1.RankFindingsResponse, error) {
	stage, err := stageFlag(ctx.Flag("stage"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.RankFindings(context.Background(), connect.NewRequest(&ladderv1.RankFindingsRequest{Stage: stage}))
	if err != nil {
		return nil, cliapp.WrapAPIError("ladder findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no findings response")
	}
	return resp.Msg, nil
}

func (h *handlers) findingsReport(_ cliapp.OperationContext, msg *ladderv1.RankFindingsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetFindings()))
	for _, finding := range msg.GetFindings() {
		results = append(results, formatFinding(finding))
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%d ranked finding(s).", len(results)),
			"Cascade applied: " + msg.GetAppliedCascade(),
		},
		ResultsHeading: "Findings",
		Results:        results,
	}
}

func formatCell(cell *ladderv1.LadderCell) string {
	grade := "ungraded"
	if cell.GetGraded() {
		grade = fmt.Sprintf("%s sev=%d", enumToken(cell.GetBand().String(), "BAND_VERDICT_"), cell.GetSeverity())
	}
	age := "undated"
	if cell.GetGapDated() {
		age = fmt.Sprintf("%dd", cell.GetGapOpenDays())
	}
	detail := cell.GetReason()
	if cell.GetReasonCode() != "" {
		detail = cell.GetReasonCode() + ": " + detail
	}
	if !cell.GetGraded() && cell.GetUngradedReason() != "" {
		detail = cell.GetUngradedReason()
	}
	return fmt.Sprintf("%-14s %-46s %-12s %-13s %-14s %-22s %-8s %d/%d blind — %s",
		cell.GetCellRef(),
		cell.GetKey(),
		enumToken(cell.GetStatus().String(), "CELL_STATUS_"),
		enumToken(cell.GetObservation().String(), "OBSERVATION_"),
		enumToken(cell.GetTrust().String(), "TRUST_VERDICT_"),
		grade,
		age,
		cell.GetBlindDevices(), cell.GetDeviceCount(),
		detail)
}

func formatSource(source *ladderv1.SourceState) string {
	return fmt.Sprintf("%-38s %-12s %s", source.GetId(),
		enumToken(source.GetTrust().String(), "TRUST_VERDICT_"), source.GetReason())
}

func formatFinding(finding *ladderv1.RankedFinding) string {
	return fmt.Sprintf("%3d [%s] %s — %s (%s)",
		finding.GetRank(),
		enumToken(finding.GetStage().String(), "CASCADE_STAGE_"),
		finding.GetTitle(),
		finding.GetMessage(),
		finding.GetStageExplanation())
}

func rungFlag(raw string) (ladderv1.Rung, error) {
	name := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", "_"))
	if name == "" {
		return ladderv1.Rung_RUNG_UNSPECIFIED, nil
	}
	value, ok := ladderv1.Rung_value["RUNG_"+name]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown rung %q", raw)
	}
	return ladderv1.Rung(value), nil
}

func stageFlag(raw string) (ladderv1.CascadeStage, error) {
	name := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", "_"))
	if name == "" {
		return ladderv1.CascadeStage_CASCADE_STAGE_UNSPECIFIED, nil
	}
	value, ok := ladderv1.CascadeStage_value["CASCADE_STAGE_"+name]
	if !ok || value == 0 {
		return 0, fmt.Errorf("unknown cascade stage %q", raw)
	}
	return ladderv1.CascadeStage(value), nil
}

func enumToken(full, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(full, prefix))
}
