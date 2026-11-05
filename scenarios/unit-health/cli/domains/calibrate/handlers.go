package calibrate

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	partition := strings.TrimSpace(ctx.Flag("partition"))
	if !ctx.FlagDeclared("partition") || partition == "" {
		partition = "inventory"
	}
	return h.call(ctx, partition)
}

func (h *handlers) corpus(ctx cliapp.RunContext) error {
	return h.call(ctx, "inventory")
}

func (h *handlers) call(ctx cliapp.RunContext, partition string) error {
	client := h.client
	if client == nil {
		httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
		client = validationconnect.NewValidationServiceClient(httpClient, baseURL)
	}
	holdout, rule := "", ""
	includeNative := false
	if ctx.FlagDeclared("holdout") {
		holdout = strings.TrimSpace(ctx.Flag("holdout"))
	}
	if ctx.FlagDeclared("rule") {
		rule = strings.TrimSpace(ctx.Flag("rule"))
	}
	if ctx.FlagDeclared("include-native") {
		includeNative = ctx.BoolFlag("include-native")
	}
	resp, err := client.RunCalibration(context.Background(), connect.NewRequest(&validationv1.RunCalibrationRequest{
		Partition: partition, HoldoutId: holdout, RuleId: rule, IncludeNative: includeNative,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("calibrate %s", partition), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no calibration response")
	}
	msg := resp.Msg
	results := []string{}
	if partition != "inventory" {
		results = calibrationCaseLines(msg.GetCases())
	}
	if len(results) == 0 && partition != "inventory" {
		results = []string{"No case outcomes were supplied."}
	}
	matched := 0
	for _, row := range msg.GetCases() {
		if row.GetMatched() {
			matched++
		}
	}
	summary := []string{fmt.Sprintf("Calibration %s (%s): %d/%d matched/implemented, %d retired, %d specified", msg.GetPartition(), msg.GetRunId(), matched, msg.GetCorpus().GetImplemented(), msg.GetCorpus().GetRetired(), msg.GetCorpus().GetSpecified())}
	summary = append(summary, corpusLines(msg.GetCorpus())...)
	summary = append(summary, holdoutLines(msg.GetHoldout())...)
	for _, limitation := range msg.GetLimitations() {
		summary = append(summary, "Limitation: "+limitation)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{Summary: summary, ResultsHeading: "Calibration cases", Results: results})
}

func corpusLines(corpus *validationv1.CorpusInventory) []string {
	if corpus == nil {
		return []string{"Corpus inventory: unknown"}
	}
	lines := []string{fmt.Sprintf("Corpus: floor=%s; spec_codes_without_emitter=%d", corpus.GetDevelopmentFloor(), corpus.GetSpecCodesWithoutEmitter())}
	for _, family := range corpus.GetFamilies() {
		lines = append(lines, fmt.Sprintf("  family %s: implemented=%d retired=%d specified=%d", family.GetFamily(), family.GetImplemented(), family.GetRetired(), family.GetSpecified()))
	}
	return lines
}

func calibrationCaseLines(cases []*validationv1.CaseOutcome) []string {
	lines := make([]string, 0, len(cases))
	for _, row := range cases {
		status := "matched"
		if !row.GetMatched() {
			status = row.GetStatus()
		}
		line := fmt.Sprintf("[%s] %s rule=%s", status, row.GetId(), row.GetRuleId())
		if len(row.GetDifferences()) > 0 {
			line += ": " + strings.Join(row.GetDifferences(), "; ")
		}
		lines = append(lines, line)
	}
	return lines
}

func holdoutLines(rows []*validationv1.HoldoutComparison) []string {
	lines := []string{"Holdout comparisons:"}
	if len(rows) == 0 {
		return append(lines, "  none supplied")
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s/%s labelled=%d observed=%d fp=%d fn=%d unknown=%d fp_rate=%.4f fn_rate=%.4f budget=%.4f within_budget=%t promotion_allowed=%t", row.GetHoldoutId(), row.GetRuleId(), row.GetLabelled(), row.GetObserved(), row.GetFalsePositives(), row.GetFalseNegatives(), row.GetUnknown(), row.GetFpRate(), row.GetFnRate(), row.GetBudget(), row.GetWithinBudget(), row.GetPromotionAllowed()))
	}
	return lines
}
