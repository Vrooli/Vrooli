package cleanup

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cleanup-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cleanup-manager/v1/cleanup/cleanup_v1connect"
)

type handlers struct {
	client cleanupconnect.CleanupServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: cleanupconnect.NewCleanupServiceClient(httpClient, baseURL)}
}

func (h *handlers) listProvidersCall(cliapp.OperationContext) (*cleanupv1.ListProvidersResponse, error) {
	resp, err := h.client.ListProviders(context.Background(), connect.NewRequest(&cleanupv1.ListProvidersRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list cleanup providers", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listProvidersReport(_ cliapp.OperationContext, msg *cleanupv1.ListProvidersResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetProviders()))
	for _, provider := range msg.GetProviders() {
		results = append(results, fmt.Sprintf("%s — %s [%s, default=%s/%s]", provider.GetId(), provider.GetName(), provider.GetSafetyTier(), provider.GetDefaultMode(), provider.GetDefaultApproval()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d cleanup provider(s).", len(results))}, ResultsHeading: "Providers", Results: results}
}

func (h *handlers) getPolicyCall(cliapp.OperationContext) (*cleanupv1.GetPolicyResponse, error) {
	resp, err := h.client.GetPolicy(context.Background(), connect.NewRequest(&cleanupv1.GetPolicyRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get cleanup policy", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) getPolicyReport(_ cliapp.OperationContext, msg *cleanupv1.GetPolicyResponse) cliapp.ListReport {
	pol := msg.GetPolicy()
	results := make([]string, 0, len(pol.GetProviders()))
	for _, provider := range pol.GetProviders() {
		results = append(results, fmt.Sprintf("%s enabled=%t approval=%s min_age=%ds max_bytes=%d", provider.GetProviderId(), provider.GetEnabled(), provider.GetApprovalMode(), provider.GetMinAgeSeconds(), provider.GetMaxBytes()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Policy %s (%s).", pol.GetVersion(), pol.GetProfile())}, ResultsHeading: "Provider policy", Results: results}
}

func (h *handlers) setPolicyProfileCall(ctx cliapp.OperationContext) (*cleanupv1.SetPolicyProfileResponse, error) {
	resp, err := h.client.SetPolicyProfile(context.Background(), connect.NewRequest(&cleanupv1.SetPolicyProfileRequest{Profile: ctx.Flag("profile")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set cleanup policy profile", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) setPolicyProfileReport(_ cliapp.OperationContext, msg *cleanupv1.SetPolicyProfileResponse) cliapp.MutationReport {
	pol := msg.GetPolicy()
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Policy set to %s.", pol.GetProfile())},
		Changes: []string{
			fmt.Sprintf("version=%s providers=%d", pol.GetVersion(), len(pol.GetProviders())),
		},
		NextCommand: []string{"`cleanup policy` — inspect provider gates", "`cleanup plan` — preview reclaimable data"},
	}
}

func (h *handlers) createPlanCall(cliapp.OperationContext) (*cleanupv1.CreatePlanResponse, error) {
	resp, err := h.client.CreatePlan(context.Background(), connect.NewRequest(&cleanupv1.CreatePlanRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create cleanup plan", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) createPlanReport(_ cliapp.OperationContext, msg *cleanupv1.CreatePlanResponse) cliapp.OperationalReport {
	plan := msg.GetPlan()
	results := make([]string, 0, len(plan.GetProviders()))
	for _, provider := range plan.GetProviders() {
		results = append(results, fmt.Sprintf("%s %d bytes %d item(s) blocked=%q approval=%s", provider.GetProviderId(), provider.GetEstimatedBytes(), provider.GetItemCount(), provider.GetBlockedReason(), provider.GetApprovalMode()))
	}
	return cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Plan %s estimates %d bytes across %d item(s).", plan.GetId(), plan.GetTotalBytes(), plan.GetTotalItems())},
		Triage:    []cliapp.TriageGroup{{Heading: "Providers", Items: results}},
		NextSteps: []string{fmt.Sprintf("`cleanup apply --plan-id %s --policy-version %s --idempotency-key <key> --approval-mode operator --approval-token <token>`", plan.GetId(), plan.GetPolicyVersion())},
	}
}

func (h *handlers) applyPlanCall(ctx cliapp.OperationContext) (*cleanupv1.ApplyPlanResponse, error) {
	resp, err := h.client.ApplyPlan(context.Background(), connect.NewRequest(&cleanupv1.ApplyPlanRequest{
		PlanId:         ctx.Flag("plan-id"),
		PolicyVersion:  ctx.Flag("policy-version"),
		ApprovalMode:   ctx.Flag("approval-mode"),
		ApprovalToken:  ctx.Flag("approval-token"),
		IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("apply cleanup plan", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) applyPlanReport(_ cliapp.OperationContext, msg *cleanupv1.ApplyPlanResponse) cliapp.MutationReport {
	changes := make([]string, 0, len(msg.GetResults()))
	for _, result := range msg.GetResults() {
		changes = append(changes, fmt.Sprintf("%s applied=%t reclaimed=%d skipped=%d", result.GetProviderId(), result.GetApplied(), result.GetReclaimedBytes(), len(result.GetSkippedItems())))
	}
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Apply %s reclaimed %d bytes (replay=%t).", msg.GetIdempotencyKey(), msg.GetReclaimedBytes(), msg.GetAlreadyApplied())},
		Changes:     changes,
		NextCommand: []string{"`cleanup audit` — inspect immutable apply history"},
	}
}

func (h *handlers) listAuditCall(cliapp.OperationContext) (*cleanupv1.ListAuditResponse, error) {
	resp, err := h.client.ListAudit(context.Background(), connect.NewRequest(&cleanupv1.ListAuditRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list cleanup audit", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listAuditReport(_ cliapp.OperationContext, msg *cleanupv1.ListAuditResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetEvents()))
	for _, event := range msg.GetEvents() {
		results = append(results, fmt.Sprintf("%s %s plan=%s provider=%s redacted=%t %s", event.GetId(), event.GetType(), event.GetPlanId(), event.GetProviderId(), event.GetRedacted(), event.GetMessage()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d audit event(s).", len(results))}, ResultsHeading: "Audit events", Results: results}
}
