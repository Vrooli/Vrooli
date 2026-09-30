package policy

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
)

type handlers struct {
	client scopesconnect.ScopesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scopesconnect.NewScopesServiceClient(http, base)}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*scopesv1.GetPolicyResponse, error) {
	response, err := h.client.GetPolicy(context.Background(), connect.NewRequest(&scopesv1.GetPolicyRequest{Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read source-ledger policy", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) setCall(ctx cliapp.OperationContext) (*scopesv1.SetPolicyResponse, error) {
	request := &scopesv1.SetPolicyRequest{Scope: ctx.Flag("scope")}
	fields := []struct {
		name string
		set  func(int32)
	}{
		{"frontier-target", func(v int32) { request.FrontierTarget = &v }},
		{"wake-budget-lines", func(v int32) { request.WakeBudgetLines = &v }},
		{"wake-budget-chars", func(v int32) { request.WakeBudgetChars = &v }},
		{"max-entry-lines", func(v int32) { request.MaxEntryLines = &v }},
		{"max-entry-chars", func(v int32) { request.MaxEntryChars = &v }},
	}
	changed := 0
	for _, field := range fields {
		raw := ctx.Flag(field.name)
		if raw == "" {
			continue
		}
		value, err := positive(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", field.name, err)
		}
		field.set(int32(value))
		changed++
	}
	if changed == 0 {
		return nil, fmt.Errorf("set requires at least one policy value")
	}
	response, err := h.client.SetPolicy(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("set source-ledger policy", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) resetCall(ctx cliapp.OperationContext) (*scopesv1.ResetPolicyResponse, error) {
	response, err := h.client.ResetPolicy(context.Background(), connect.NewRequest(&scopesv1.ResetPolicyRequest{Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("reset source-ledger policy", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *scopesv1.GetPolicyResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Effective Source Ledger policy:"}, Results: append(snapshotLines(msg.GetEffective(), msg.GetDefaults()), livenessLines(msg.GetLiveness())...)}
}

func (h *handlers) setReport(_ cliapp.OperationContext, msg *scopesv1.SetPolicyResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: append(snapshotLines(msg.GetEffective(), msg.GetDefaults()), livenessLines(msg.GetLiveness())...)}
}

func (h *handlers) resetReport(_ cliapp.OperationContext, msg *scopesv1.ResetPolicyResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: append(append([]string{"Policy reset; scope now inherits file defaults."}, snapshotLines(msg.GetEffective(), msg.GetDefaults())...), livenessLines(msg.GetLiveness())...)}
}

func livenessLines(value *scopesv1.CompactionLiveness) []string {
	if value == nil {
		return []string{"compaction-liveness=unavailable"}
	}
	return []string{
		fmt.Sprintf("unsummarized-leaves=%d", value.GetUnsummarizedLeafCount()),
		fmt.Sprintf("oldest-unsummarized-leaf-at=%s", value.GetOldestUnsummarizedLeafAt()),
		fmt.Sprintf("last-summary-at=%s", value.GetLastSummaryAt()),
	}
}

func snapshotLines(effective, defaults *scopesv1.PolicySnapshot) []string {
	if effective == nil {
		return []string{"No policy returned."}
	}
	return []string{
		fmt.Sprintf("frontier-target=%d (origin=%s, default=%d)", effective.GetFrontierTarget(), effective.GetFrontierTargetOrigin(), defaults.GetFrontierTarget()),
		fmt.Sprintf("wake-budget-lines=%d (origin=%s, default=%d)", effective.GetWakeBudgetLines(), effective.GetWakeBudgetLinesOrigin(), defaults.GetWakeBudgetLines()),
		fmt.Sprintf("wake-budget-chars=%d (origin=%s, default=%d)", effective.GetWakeBudgetChars(), effective.GetWakeBudgetCharsOrigin(), defaults.GetWakeBudgetChars()),
		fmt.Sprintf("max-entry-lines=%d (origin=%s, default=%d)", effective.GetMaxEntryLines(), effective.GetMaxEntryLinesOrigin(), defaults.GetMaxEntryLines()),
		fmt.Sprintf("max-entry-chars=%d (origin=%s, default=%d)", effective.GetMaxEntryChars(), effective.GetMaxEntryCharsOrigin(), defaults.GetMaxEntryChars()),
	}
}

func positive(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return value, nil
}
