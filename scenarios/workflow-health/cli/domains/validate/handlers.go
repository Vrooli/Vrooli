package validate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
)

type handlers struct {
	client  scenariovalidationconnect.ScenarioValidationServiceClient
	durable scenariovalidationconnect.DurableValidationRunServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL), durable: scenariovalidationconnect.NewDurableValidationRunServiceClient(httpClient, baseURL)}
}

func (h *handlers) scenario(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: scenario,
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	for _, finding := range assessment.GetFindings() {
		results = append(results, formatFinding(finding))
	}
	summary := []string{
		fmt.Sprintf("Validated %s: status=%s errors=%d warnings=%d infos=%d",
			msg.GetScenario(),
			statusLabel(msg.GetStatus()),
			countSeverity(assessment, "ERROR"),
			countSeverity(assessment, "WARN"),
			countSeverity(assessment, "INFO"),
		),
	}
	if local := assessment.GetLocal(); local != nil {
		line := fmt.Sprintf("Maturity: %s", local.GetCurrentLevel())
		if local.GetNextLevel() != "" {
			line += " -> " + local.GetNextLevel()
		}
		summary = append(summary, line)
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Workflow findings",
		Results:        results,
	}); err != nil {
		return err
	}
	switch msg.GetStatus() {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return fmt.Errorf("scenario %s did not pass workflow validation", msg.GetScenario())
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return fmt.Errorf("scenario %s workflow validation errored", msg.GetScenario())
	default:
		return nil
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	return cliapp.RunDurable(cliapp.DurableRunModeFrom(ctx.JSON(), false), cliapp.DurableRunSpec[*scenariovalidationv1.ValidationRun]{
		Start: func() (*scenariovalidationv1.ValidationRun, error) {
			resp, err := h.durable.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: ctx.Positional("scenario"), Path: ctx.Flag("path"), IdempotencyKey: ctx.Flag("idempotency-key"), ParentRunId: ctx.Flag("parent-run-id")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("start workflow validation run", err, nil)
			}
			if resp == nil || resp.Msg == nil || resp.Msg.GetRun() == nil {
				return nil, fmt.Errorf("server returned no validation run")
			}
			return resp.Msg.GetRun(), nil
		},
		Human: func(run *scenariovalidationv1.ValidationRun) error { return h.renderRun(ctx, run) },
		JSON:  func(run *scenariovalidationv1.ValidationRun) error { return h.renderRun(ctx, run) },
		JSONL: func(run *scenariovalidationv1.ValidationRun) error { return h.renderRun(ctx, run) },
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.durable.GetValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.GetValidationRunRequest{RunId: ctx.Positional("run-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get workflow validation run", err, nil)
	}
	return h.renderRun(ctx, resp.Msg.GetRun())
}

func (h *handlers) wait(ctx cliapp.RunContext) error {
	var timeout time.Duration
	if raw := strings.TrimSpace(ctx.Flag("timeout")); raw != "" {
		var err error
		timeout, err = time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("timeout must be a Go duration: %w", err)
		}
	}
	resp, err := h.durable.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: ctx.Positional("run-id"), Timeout: durationpb.New(timeout)}))
	if err != nil {
		return cliapp.WrapAPIError("wait for workflow validation run", err, nil)
	}
	return h.renderRun(ctx, resp.Msg.GetRun())
}

func (h *handlers) abort(ctx cliapp.RunContext) error {
	resp, err := h.durable.AbortValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.AbortValidationRunRequest{RunId: ctx.Positional("run-id"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("abort workflow validation run", err, nil)
	}
	return h.renderRun(ctx, resp.Msg.GetRun())
}

func (h *handlers) renderRun(ctx cliapp.RunContext, run *scenariovalidationv1.ValidationRun) error {
	if run == nil {
		return fmt.Errorf("server returned no validation run")
	}
	return cliapp.RenderProtoList(ctx, run, cliapp.ListReport{Summary: []string{fmt.Sprintf("Validation run %s: state=%s scenario=%s", run.GetRunId(), strings.TrimPrefix(strings.ToLower(run.GetState().String()), "validation_run_state_"), run.GetScenario())}, ResultsHeading: "Validation run", Results: []string{fmt.Sprintf("idempotency_key=%s", run.GetIdempotencyKey()), fmt.Sprintf("cancellation_requested=%t", run.GetCancellationRequested())}})
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return "passed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return "failed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return "degraded"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return "error"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		return "skipped"
	default:
		return "unspecified"
	}
}

func countSeverity(a *commonv1.MaturityAssessment, severity string) int {
	if a == nil {
		return 0
	}
	total := 0
	for key, count := range a.GetFindingsBySeverity() {
		if severityLabel(key) == severity {
			total += int(count)
		}
	}
	return total
}

func formatFinding(f *commonv1.AssessmentFinding) string {
	if f == nil {
		return ""
	}
	line := fmt.Sprintf("[%s] %s - %s", severityLabel(f.GetSeverity()), f.GetCode(), f.GetMessage())
	if f.GetLocation() != "" {
		line += " (" + f.GetLocation() + ")"
	}
	if f.GetRemediation() != "" {
		line += "\n    remediation: " + f.GetRemediation()
	}
	return line
}

func severityLabel(severity string) string {
	switch severity {
	case "SEVERITY_ERROR", "FINDING_SEVERITY_ERROR", "ERROR":
		return "ERROR"
	case "SEVERITY_WARNING", "FINDING_SEVERITY_WARNING", "WARNING", "WARN":
		return "WARN"
	case "SEVERITY_INFO", "FINDING_SEVERITY_INFO", "INFO":
		return "INFO"
	case "SEVERITY_BLOCKER", "FINDING_SEVERITY_BLOCKER", "BLOCKER":
		return "BLOCKER"
	default:
		return "UNSPECIFIED"
	}
}
