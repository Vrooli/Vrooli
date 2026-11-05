package mutation

import (
	"context"
	"fmt"
	"strconv"
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

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

func (h *handlers) pilot(ctx cliapp.RunContext) error {
	client := h.client
	if client == nil {
		httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
		client = validationconnect.NewValidationServiceClient(httpClient, baseURL)
	}
	maxMutants := uint32(0)
	if value := first(ctx.FlagValues("max-mutants")); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("max-mutants: %w", err)
		}
		maxMutants = uint32(parsed)
	}
	operators := ctx.FlagValues("operators")
	if len(operators) == 0 {
		operators = []string{"boundary", "negate-condition", "return-constant"}
	}
	seed := first(ctx.FlagValues("seed"))
	if seed == "" {
		return fmt.Errorf("seed is required")
	}
	resp, err := client.RunMutationPilot(context.Background(), connect.NewRequest(&validationv1.RunMutationPilotRequest{
		Scenario: ctx.Positional("scenario"), Workspace: firstOr(ctx.FlagValues("workspace"), "api"), Package: firstOr(ctx.FlagValues("package"), "./internal/testquality/..."), Operators: operators, MaxMutants: maxMutants, Seed: seed,
	}))
	if err != nil {
		return cliapp.WrapAPIError("run mutation pilot", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no mutation-pilot response")
	}
	msg := resp.Msg
	summary := msg.GetSummary()
	lines := []string{fmt.Sprintf("Mutation pilot %s: generated=%d killed=%d survived=%d invalid=%d equivalent=%d out_of_contract=%d infrastructure_failure=%d unknown=%d kill_rate=%.4f", msg.GetRunId(), summary.GetGenerated(), summary.GetKilled(), summary.GetSurvived(), summary.GetInvalid(), summary.GetEquivalent(), summary.GetOutOfContract(), summary.GetInfrastructureFailure(), summary.GetUnknown(), summary.GetKillRate())}
	results := make([]string, 0, len(msg.GetReceipts()))
	for _, receipt := range msg.GetReceipts() {
		results = append(results, fmt.Sprintf("[%s] %s %s:%d owning_test=%s: %s", receipt.GetDisposition(), receipt.GetOperator(), receipt.GetFile(), receipt.GetLine(), receipt.GetOwningTest(), receipt.GetDetail()))
	}
	for _, limitation := range msg.GetLimitations() {
		lines = append(lines, "Limitation: "+limitation)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{Summary: lines, ResultsHeading: "Mutation receipts", Results: results})
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func firstOr(values []string, fallback string) string {
	if value := first(values); value != "" {
		return value
	}
	return fallback
}
