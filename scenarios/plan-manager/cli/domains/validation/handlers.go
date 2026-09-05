package validation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation/validation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	generation, err := parseScopeGeneration(ctx.Flag("scope-generation"))
	if err != nil {
		return cliapp.WrapAPIError("start validation", err, nil)
	}
	resp, err := h.client.StartValidation(context.Background(), connect.NewRequest(&validationv1.StartValidationRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"), IdempotencyKey: ctx.Flag("idempotency-key"), ExecutionId: ctx.Flag("execution"), ScopeGeneration: generation, Member: commaValues(ctx.Flag("members")), TestRuns: testRunEvidence(ctx.Flag("test-runs")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("start validation", err, nil)
	}
	op := resp.Msg.GetOperation()
	changes := []string{fmt.Sprintf("Status: %s.", operationStatusLabel(op.GetStatus()))}
	if op.GetScopeFingerprint() != "" {
		changes = append(changes, fmt.Sprintf("Scope fingerprint: %s.", op.GetScopeFingerprint()))
	}
	if op.GetQueueReason() != "" {
		changes = append(changes, fmt.Sprintf("Queue reason: %s.", op.GetQueueReason()))
	}
	if resp.Msg.GetDeduplicated() {
		changes = append(changes, "Idempotency key matched the existing operation; no child work was duplicated.")
	}
	for _, child := range op.GetChildren() {
		if child.GetCommand() != "" {
			changes = append(changes, "Producer action: "+child.GetCommand())
		}
	}
	if len(op.GetProducerWaitArgv()) > 0 {
		changes = append(changes, "Producer wait: "+strings.Join(op.GetProducerWaitArgv(), " "))
	}
	syncCommand := "plan-manager validate sync " + op.GetId()
	if len(op.GetSyncArgv()) > 0 {
		syncCommand = strings.Join(op.GetSyncArgv(), " ")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Validation operation: %s", op.GetId())},
		Changes:     changes,
		NextCommand: []string{"Run the exact producer action above, use Git Control Tower's printed native wait, then " + syncCommand + "."},
	})
}

func parseScopeGeneration(raw string) (int32, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("--scope-generation must be a non-negative integer")
	}
	return int32(value), nil
}

func commaValues(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func testRunEvidence(raw string) []*validationv1.TestRunEvidence {
	var runs []*validationv1.TestRunEvidence
	for _, value := range commaValues(raw) {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			runs = append(runs, &validationv1.TestRunEvidence{Scenario: strings.TrimSpace(parts[0]), RunId: strings.TrimSpace(parts[1])})
		}
	}
	return runs
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	resp, err := h.client.SyncValidation(context.Background(), connect.NewRequest(&validationv1.SyncValidationRequest{OperationId: ctx.Positional("operation")}))
	if err != nil {
		return cliapp.WrapAPIError("synchronize validation", err, nil)
	}
	op := resp.Msg.GetOperation()
	changes := []string{fmt.Sprintf("Status: %s.", operationStatusLabel(op.GetStatus()))}
	if op.GetQueueReason() != "" {
		changes = append(changes, op.GetQueueReason())
	}
	if op.GetResult() != nil {
		changes = append(changes, fmt.Sprintf("Verdict: %s.", verdictLabel(op.GetResult().GetVerdict())))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Validation operation: %s", op.GetId())}, Changes: changes, NextCommand: op.GetProducerWaitArgv()})
}

func (h *handlers) show(ctx cliapp.RunContext) error   { return h.operation(ctx, "show") }
func (h *handlers) wait(ctx cliapp.RunContext) error   { return h.operation(ctx, "wait") }
func (h *handlers) resume(ctx cliapp.RunContext) error { return h.operation(ctx, "resume") }

func (h *handlers) operation(ctx cliapp.RunContext, command string) error {
	operationID := ctx.Positional("operation")
	resp, err := h.client.GetValidationOperation(context.Background(), connect.NewRequest(&validationv1.GetValidationOperationRequest{OperationId: operationID}))
	if err != nil {
		return cliapp.WrapAPIError(command+" validation", err, nil)
	}
	op := resp.Msg.GetOperation()
	changes := []string{
		fmt.Sprintf("Status: %s; attempt %d; %d child operation(s).", operationStatusLabel(op.GetStatus()), op.GetAttempt(), len(op.GetChildren())),
	}
	if op.GetScopeFingerprint() != "" {
		changes = append(changes, fmt.Sprintf("Scope fingerprint: %s.", op.GetScopeFingerprint()))
	}
	if op.GetQueueReason() != "" {
		changes = append(changes, fmt.Sprintf("Queue reason: %s.", op.GetQueueReason()))
	}
	if result := op.GetResult(); result != nil && op.GetStatus() == validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL {
		changes = append(changes, fmt.Sprintf("Verdict: %s (staleness %s).", verdictLabel(result.GetVerdict()), stalenessLabel(result.GetStaleness())))
		if result.GetDetail() != "" {
			changes = append(changes, result.GetDetail())
		}
	}
	hints := []string{}
	if op.GetStatus() != validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL {
		hints = append(hints, "Run the ticket's Git Control Tower/Test Genie producer wait command, then `plan-manager validate sync "+op.GetId()+"`.")
	}
	if err := cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Validation operation: %s", op.GetId())}, Changes: changes, NextCommand: hints,
	}); err != nil {
		return err
	}
	return nil
}

func operationStatusLabel(status validationv1.ValidationOperationStatus) string {
	switch status {
	case validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_QUEUED:
		return "queued"
	case validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_RUNNING:
		return "running"
	case validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL:
		return "terminal"
	default:
		return "unspecified"
	}
}

func (h *handlers) references(ctx cliapp.RunContext) error {
	resp, err := h.client.ResolveReferences(context.Background(), connect.NewRequest(&validationv1.ResolveReferencesRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("resolve references", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.References))
	for _, r := range resp.Msg.References {
		results = append(results, formatReference(r))
	}
	summary := fmt.Sprintf("Resolved %d reference(s).", len(resp.Msg.References))
	if resp.Msg.Degraded {
		summary += " (degraded: code-facts unavailable)"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{summary}, ResultsHeading: "References", Results: results,
	})
}

func (h *handlers) staleness(ctx cliapp.RunContext) error {
	resp, err := h.client.ComputeStaleness(context.Background(), connect.NewRequest(&validationv1.ComputeStalenessRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("compute staleness", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.References))
	for _, r := range resp.Msg.References {
		results = append(results, formatReference(r))
	}
	summary := fmt.Sprintf("Overall staleness: %s.", stalenessLabel(resp.Msg.Overall))
	if resp.Msg.Degraded {
		summary += " (degraded)"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{summary}, ResultsHeading: "References", Results: results,
	})
}

func (h *handlers) baselineScope(ctx cliapp.RunContext) error {
	resp, err := h.client.DeriveBaselineScope(context.Background(), connect.NewRequest(&validationv1.DeriveBaselineScopeRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("derive baseline scope", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Derived %d command(s) across %d location(s).", len(resp.Msg.Commands), len(resp.Msg.Locations))},
		ResultsHeading: "Baseline commands",
		Results:        resp.Msg.Commands,
		RetrievalHints: []string{"`validate run <plan>` — run this command set and report the verdict"},
	})
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunValidation(context.Background(), connect.NewRequest(&validationv1.RunValidationRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"),
	}))
	if err != nil {
		return wrapRunValidationError(err)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Verdict: %s (staleness %s).", verdictLabel(resp.Msg.Result.GetVerdict()), stalenessLabel(resp.Msg.Result.GetStaleness()))},
		Changes: []string{resp.Msg.Result.GetDetail()},
	})
}

func wrapRunValidationError(err error) error {
	wrapped := cliapp.WrapAPIError("run validation", err, nil)
	if wrapped == nil {
		return nil
	}
	return fmt.Errorf("%w; draft authoring sessions are validated with `plan-manager author validate <session>` before finalize", wrapped)
}

func (h *handlers) verifyDoD(ctx cliapp.RunContext) error {
	resp, err := h.client.VerifyDefinitionOfDone(context.Background(), connect.NewRequest(&validationv1.VerifyDefinitionOfDoneRequest{
		PlanId: ctx.Positional("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("verify definition of done", err, nil)
	}
	met := "NOT met"
	if resp.Msg.DodMet {
		met = "met"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Definition of Done %s (verdict %s).", met, verdictLabel(resp.Msg.Result.GetVerdict()))},
		Changes: []string{resp.Msg.Result.GetDetail()},
	})
}

func formatReference(r *sharedv1.Reference) string {
	marker := "CODE"
	switch r.Kind {
	case sharedv1.ReferenceKind_REFERENCE_KIND_REQ:
		marker = "REQ"
	case sharedv1.ReferenceKind_REFERENCE_KIND_DOC:
		marker = "DOC"
	}
	line := fmt.Sprintf("[%s: %s] resolution=%s staleness=%s", marker, r.Target, resolutionLabel(r.Resolution), stalenessLabel(r.Staleness))
	if r.Note != "" {
		line += " (" + r.Note + ")"
	}
	return line
}

func verdictLabel(v sharedv1.ValidationVerdict) string {
	switch v {
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS:
		return "pass"
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL:
		return "fail"
	case sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN:
		return "unknown"
	default:
		return "unspecified"
	}
}

func stalenessLabel(s sharedv1.StalenessTier) string {
	switch s {
	case sharedv1.StalenessTier_STALENESS_TIER_FRESH:
		return "fresh"
	case sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE:
		return "lightly_stale"
	case sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE:
		return "definitely_stale"
	default:
		return "unknown"
	}
}

func resolutionLabel(r sharedv1.ReferenceResolution) string {
	switch r {
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED:
		return "resolved"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED:
		return "unresolved"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE:
		return "future"
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING:
		return "missing"
	default:
		return "unspecified"
	}
}
