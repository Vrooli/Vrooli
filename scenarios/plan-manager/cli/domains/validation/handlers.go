package validation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"connectrpc.com/connect"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation/validation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	validationTransportWaitBudget = 15 * time.Minute
	validationRecoveryBudget      = 30 * time.Second
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, validationTransportWaitBudget+time.Minute)
	return &handlers{
		core:   core,
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	resp, err := h.client.StartValidation(context.Background(), connect.NewRequest(&validationv1.StartValidationRequest{
		PlanId: ctx.Positional("plan"), PhaseId: ctx.Flag("phase"), IdempotencyKey: ctx.Flag("idempotency-key"),
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
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Validation operation: %s", op.GetId())},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("plan-manager validate wait %s", op.GetId())},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error   { return h.operation(ctx, "show", false) }
func (h *handlers) wait(ctx cliapp.RunContext) error   { return h.operation(ctx, "wait", true) }
func (h *handlers) resume(ctx cliapp.RunContext) error { return h.operation(ctx, "resume", true) }

func (h *handlers) operation(ctx cliapp.RunContext, command string, wait bool) error {
	operationID := ctx.Positional("operation")
	resp, recovered, err := h.getOperation(operationID, command, wait)
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
	if recovered {
		changes = append(changes, "The blocking attachment ended unexpectedly; this is the one recovery read by durable operation id.")
	}
	if result := op.GetResult(); result != nil && op.GetStatus() == validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL {
		changes = append(changes, fmt.Sprintf("Verdict: %s (staleness %s).", verdictLabel(result.GetVerdict()), stalenessLabel(result.GetStaleness())))
		if result.GetDetail() != "" {
			changes = append(changes, result.GetDetail())
		}
	}
	hints := []string{}
	if op.GetStatus() != validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL {
		hints = append(hints, fmt.Sprintf("`plan-manager validate resume %s` — reattach after interruption or server restart", op.GetId()))
	}
	if err := cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Validation operation: %s", op.GetId())}, Changes: changes, NextCommand: hints,
	}); err != nil {
		return err
	}
	if recovered && op.GetStatus() != validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL {
		return fmt.Errorf("validation wait detached before terminal truth; inspect or resume operation %s", op.GetId())
	}
	return nil
}

func (h *handlers) getOperation(operationID, command string, wait bool) (*connect.Response[validationv1.GetValidationOperationResponse], bool, error) {
	timeout := validationRecoveryBudget
	if wait {
		timeout = validationTransportWaitBudget
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req := connect.NewRequest(&validationv1.GetValidationOperationRequest{OperationId: operationID, Wait: wait})
	var resp *connect.Response[validationv1.GetValidationOperationResponse]
	var err error
	switch command {
	case "wait":
		resp, err = h.client.WaitValidationOperation(ctx, req)
	case "resume":
		resp, err = h.client.ResumeValidationOperation(ctx, req)
	default:
		resp, err = h.client.GetValidationOperation(ctx, req)
	}
	if err == nil || !wait || !isAttachmentEnd(err) {
		return resp, false, err
	}
	// Exactly one reconnect by durable id. This is an inspect read, never a
	// duplicate StartValidation call and never a polling loop.
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), validationRecoveryBudget)
	defer recoveryCancel()
	resp, err = h.client.GetValidationOperation(recoveryCtx, connect.NewRequest(&validationv1.GetValidationOperationRequest{
		OperationId: operationID, Wait: false,
	}))
	return resp, true, err
}

func isUnexpectedEOF(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || strings.Contains(strings.ToLower(err.Error()), "unexpected eof")
}

func isAttachmentEnd(err error) bool {
	code := connect.CodeOf(err)
	return isUnexpectedEOF(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || code == connect.CodeDeadlineExceeded || code == connect.CodeCanceled
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
