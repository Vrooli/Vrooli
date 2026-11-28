package model

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/domains/labels"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models/models_v1connect"
)

type handlers struct {
	client modelsconnect.ModelRegistryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: modelsconnect.NewModelRegistryServiceClient(httpClient, baseURL),
	}
}

// list prints the registry catalogue. Filters map directly onto
// ListModelsRequest; --json is handled by cli-core and emits protojson.
func (h *handlers) listCall(ctx cliapp.OperationContext) (*modelsv1.ListModelsResponse, error) {
	resp, err := h.client.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{
		Capability:     strings.TrimSpace(ctx.Flag("capability")),
		Language:       strings.TrimSpace(ctx.Flag("language")),
		CommercialOnly: ctx.BoolFlag("commercial-only"),
		OnlyReady:      ctx.BoolFlag("only-ready"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("model list", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no model list response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response *modelsv1.ListModelsResponse) cliapp.ListReport {
	records := response.GetModels()
	summary := []string{fmt.Sprintf("Models: %d", len(records))}
	if footprint := footprintSummary(records); footprint != "" {
		summary = append(summary, footprint)
	}
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Models",
		Results:        formatRecordTable(records),
		RetrievalHints: []string{
			"audio-tools model list --commercial-only --only-ready   # selectable, commercially usable models",
			"audio-tools model resolve --capability stt             # deterministic default",
		},
		ListShaped:  true,
		ResultCount: len(records),
	}
}

// resolve prints the deterministic selection and the policy explanation.
// NOT_FOUND (no eligible record) is surfaced as a non-zero CLI error.
func (h *handlers) resolveCall(ctx cliapp.OperationContext) (*modelsv1.ResolveModelResponse, error) {
	resp, err := h.client.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: strings.TrimSpace(ctx.Flag("capability")),
		Language:   strings.TrimSpace(ctx.Flag("language")),
		EngineId:   strings.TrimSpace(ctx.Flag("engine")),
		Commercial: ctx.BoolFlag("commercial"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("model resolve", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetModel() == nil {
		return nil, fmt.Errorf("server returned no resolved model")
	}
	return resp.Msg, nil
}

func (h *handlers) resolveReport(ctx cliapp.OperationContext, response *modelsv1.ResolveModelResponse) cliapp.ListReport {
	record := response.GetModel()
	summary := []string{"Resolved model: " + orDash(record.GetId())}
	if explanation := strings.TrimSpace(response.GetExplanation()); explanation != "" {
		summary = append(summary, explanation)
	}
	if engine := strings.TrimSpace(ctx.Flag("engine")); engine != "" {
		summary = append(summary, "Engine-scoped resolution: "+engine+"; this is not a promotion to the generic capability default.")
	}

	results := []string{formatRecordRow(record)}
	results = append(results, formatFacts(record)...)

	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Selection",
		Results:        results,
		RetrievalHints: []string{
			"audio-tools model list --capability " + orDash(record.GetCapability()) + "   # see the ranked candidates",
		},
		ListShaped:  true,
		ResultCount: 1,
	}
}

const (
	idWidth         = 30
	capabilityWidth = 10
	readinessWidth  = 12
	commercialWidth = 11
	licenseWidth    = 18
	sizeWidth       = 12
)

// formatRecordRow is the single ModelRecord -> row formatter shared by the
// `list` table and the `resolve` selection block.
func formatRecordRow(record *modelsv1.ModelRecord) string {
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %s",
		idWidth, truncateField(record.GetId(), idWidth),
		capabilityWidth, truncateField(orDash(record.GetCapability()), capabilityWidth),
		readinessWidth, truncateField(readinessLabel(record.GetReadiness()), readinessWidth),
		commercialWidth, truncateField(commercialLabel(record.GetCommercialUse()), commercialWidth),
		licenseWidth, truncateField(orDash(record.GetLicense()), licenseWidth),
		sizeWidth, formatSize(record.GetSizeMbApprox()),
		defaultLabel(record),
	)
}

func formatRecordTable(records []*modelsv1.ModelRecord) []string {
	if len(records) == 0 {
		return []string{"(no models matched the filters)"}
	}
	rows := []string{fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %s",
		idWidth, "ID",
		capabilityWidth, "CAPABILITY",
		readinessWidth, "READINESS",
		commercialWidth, "COMMERCIAL",
		licenseWidth, "LICENSE",
		sizeWidth, "SIZE",
		"DEFAULT",
	)}
	for _, record := range records {
		rows = append(rows, formatRecordRow(record))
	}
	return rows
}

func formatFacts(record *modelsv1.ModelRecord) []string {
	facts := []string{
		"  runtime:   " + orDash(record.GetRuntime()),
		"  readiness: " + readinessLabel(record.GetReadiness()),
		"  license:   " + orDash(record.GetLicense()),
		"  roles:     " + joinOrDash(record.GetRoles()),
		"  size:      " + formatSize(record.GetSizeMbApprox()),
	}
	if len(record.GetEngineDefaults()) > 0 {
		facts = append(facts, "  engine defaults: "+strings.Join(record.GetEngineDefaults(), ","))
	}
	if record.GetCapability() == "tts" && strings.TrimSpace(record.GetNotes()) != "" {
		facts = append(facts, "  TTS route notes: "+strings.TrimSpace(record.GetNotes()))
	}
	return facts
}

// footprintSummary names the resolved default and its declared size per
// capability. It is the operator-facing answer to "what will this machine
// actually run?" without adding a second registry surface.
func footprintSummary(records []*modelsv1.ModelRecord) string {
	byCapability := map[string]*modelsv1.ModelRecord{}
	for _, record := range records {
		if record.GetIsDefault() {
			byCapability[record.GetCapability()] = record
		}
	}
	order := []string{"stt", "tts", "summarize", "speaker"}
	parts := make([]string, 0, len(order))
	for _, capability := range order {
		record, ok := byCapability[capability]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s (%s)", capability, record.GetId(), formatSize(record.GetSizeMbApprox())))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Defaults: " + strings.Join(parts, ", ")
}

// formatSize renders a declared model size. 0 means "not established", not
// "weightless", so it is shown as unknown rather than 0 MB.
func formatSize(mb float64) string {
	if mb <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%.0f MB", mb)
}

func readinessLabel(readiness modelsv1.Readiness) string {
	return labels.Enum(readiness, "READINESS_", "unknown", labels.LowerWords)
}

func commercialLabel(use modelsv1.CommercialUse) string {
	return labels.Enum(use, "COMMERCIAL_USE_", "unknown", labels.LowerWords)
}

func defaultLabel(record *modelsv1.ModelRecord) string {
	if record.GetIsDefault() {
		return "yes"
	}
	return "-"
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

// truncateField keeps fixed-width columns deterministic and marks elision
// with an ellipsis instead of silently dropping the tail. Width is measured in
// runes so a multi-byte licence or model id is never split mid-character.
func truncateField(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}
