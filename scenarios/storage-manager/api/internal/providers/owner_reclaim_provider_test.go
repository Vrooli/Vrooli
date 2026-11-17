package providers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

const (
	ownerRoot  = "/home/builder/.cache/go-build"
	ownerToken = "v1.0123456789ab.100.200"
	gib        = int64(1 << 30)
)

// scriptedOwner answers owner-reclaim/v1 verbs the way a conforming owner
// does, and records every invocation.
type scriptedOwner struct {
	plan, reclaim cleanup.ProcessResult
	planErr       error
	reclaimErr    error
	calls         []cleanup.ProcessCommand
}

func (o *scriptedOwner) Run(_ context.Context, cmd cleanup.ProcessCommand) (cleanup.ProcessResult, error) {
	o.calls = append(o.calls, cmd)
	switch {
	case slices.Contains(cmd.Args, "plan"):
		return o.plan, o.planErr
	case slices.Contains(cmd.Args, "reclaim"):
		return o.reclaim, o.reclaimErr
	default:
		return cleanup.ProcessResult{ExitCode: 2}, errors.New("unknown owner verb")
	}
}

func (o *scriptedOwner) verbs() []string {
	out := make([]string, 0, len(o.calls))
	for _, call := range o.calls {
		out = append(out, call.Args[2])
	}
	return out
}

func ownerPlanJSON(reclaimable int64, token, blocked string) cleanup.ProcessResult {
	return cleanup.ProcessResult{Stdout: fmt.Sprintf(`{"schema":"vrooli.go-cache.plan/v1","cache":%q,"store_id":"0123456789ab","token":%q,"store_bytes":%d,"budget_bytes":%d,"reclaimable_bytes":%d,"blocked_reason":%q,"audit":{"go_commands":3}}`,
		ownerRoot, token, 136*gib, 40*gib, reclaimable, blocked)}
}

func ownerReceiptJSON(cache string, freed int64, removed int, stop, blocked string) cleanup.ProcessResult {
	return cleanup.ProcessResult{Stdout: fmt.Sprintf(`{"schema":"vrooli.go-cache.reclaim/v1","cache":%q,"store_id":"0123456789ab","freed_bytes":%d,"removed_entries":%d,"stop_reason":%q,"plan_exhausted":false,"blocked_reason":%q}`,
		cache, freed, removed, stop, blocked)}
}

func newOwnerProvider(t *testing.T, owner *scriptedOwner) *OwnerReclaimProvider {
	t.Helper()
	var runner cleanup.ProcessRunner
	if owner != nil {
		runner = owner
	}
	provider, err := NewOwnerReclaimProvider(runner, cleanupfakes.Clock{Time: now}, OwnerReclaimProviderConfig{
		ID: "go-build-cache", Name: "Governed go-build-cache", Root: ownerRoot,
		Command:   cleanup.ProcessCommand{Name: "vrooli", Args: []string{"host", "go-cache"}},
		Platforms: []string{"linux"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

var (
	ownerEnabled  = cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeNone}
	recoveryScope = cleanup.ObservationScope{Now: now, Recovery: true}
)

func ownerApply(t *testing.T, provider *OwnerReclaimProvider, ctx context.Context, items []cleanup.PreviewItem) cleanup.ApplyResult {
	t.Helper()
	result, err := provider.Apply(ctx, cleanup.ApplyRequest{
		ProviderVersion: "v1", IdempotencyKey: "run|go-build-cache|0", ApprovalMode: cleanup.ApprovalModeNone,
		Preview: cleanup.Preview{ProviderID: "go-build-cache", ProviderVersion: "v1", Items: items},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestOwnerReclaimProviderIsAnOwnerBudgetedSafeWithOwnerProvider(t *testing.T) {
	meta := newOwnerProvider(t, &scriptedOwner{}).Metadata()
	if err := meta.Validate(); err != nil {
		t.Fatal(err)
	}
	if meta.SafetyTier != cleanup.SafetyTierSafeWithOwner || !meta.OwnerBudget || meta.DefaultApproval != cleanup.ApprovalModeOwner || meta.NoLease {
		t.Fatalf("metadata = %#v, want owner-budgeted safe_with_owner with owner approval and no lease waiver", meta)
	}
}

// The recovery loop estimates, previews a bounded batch, and applies it. The
// owner is asked for one plan per batch, and the apply is bound to that plan's
// token, the declared root, the batch's bytes, and the caller's deadline.
func TestOwnerReclaimRecoveryBatchIsBoundToThePlanAndBudget(t *testing.T) {
	owner := &scriptedOwner{plan: ownerPlanJSON(3*gib+gib/2, ownerToken, ""), reclaim: ownerReceiptJSON(ownerRoot, 2*gib-4096, 9000, "max_bytes", "")}
	provider := newOwnerProvider(t, owner)

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: recoveryScope, Policy: ownerEnabled})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.BlockedReason != "" || estimate.EstimatedBytes != 3*gib+gib/2 || estimate.ItemCount != 4 {
		t.Fatalf("estimate = %#v, want the owner's reclaimable bytes in four bounded items", estimate)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: recoveryScope, Policy: ownerEnabled, Estimate: estimate})
	if err != nil {
		t.Fatal(err)
	}
	if got := owner.verbs(); !slices.Equal(got, []string{"plan"}) {
		t.Fatalf("owner invocations = %v, want one plan shared by estimate and preview", got)
	}
	if len(preview.Items) != 4 || preview.AllowSingleOvershoot {
		t.Fatalf("preview = %#v, want four divisible items", preview)
	}
	var total int64
	ids := map[string]bool{}
	for _, item := range preview.Items {
		total += item.Bytes
		ids[item.ID] = true
		if item.Identity != ownerToken || item.Path != ownerRoot || item.Bytes > gib || item.Action != ownerReclaimAction {
			t.Errorf("item %#v is not a bounded owner item bound to the plan token", item)
		}
	}
	if total != 3*gib+gib/2 || len(ids) != 4 {
		t.Fatalf("items total %d bytes in %d distinct ids, want the plan exactly", total, len(ids))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	result := ownerApply(t, provider, ctx, preview.Items[:2])
	if !result.Applied || result.ReclaimedBytes != 2*gib-4096 || len(result.AppliedItems) != 2 || len(result.SkippedItems) != 0 {
		t.Fatalf("result = %#v, want the owner's freed bytes for the applied batch", result)
	}
	call := owner.calls[len(owner.calls)-1]
	want := []string{"host", "go-cache", "reclaim", "--cache", ownerRoot, "--token", ownerToken, "--max-bytes", fmt.Sprint(2 * gib), "--time-budget"}
	if call.Name != "vrooli" || !slices.Equal(call.Args[:len(want)], want) || call.Args[len(call.Args)-1] != "--json" {
		t.Fatalf("reclaim invocation = %s %v, want %v <budget> --json", call.Name, call.Args, want)
	}
	budget, err := time.ParseDuration(call.Args[len(want)])
	if err != nil || budget <= 0 || budget > 40*time.Second-ownerDeadlineMargin {
		t.Fatalf("time budget %q must leave the owner margin before the caller's deadline", call.Args[len(want)])
	}

	// The next batch asks the owner again; its ids never repeat the last batch.
	owner.plan = ownerPlanJSON(gib+gib/2, ownerToken, "")
	next, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: recoveryScope, Policy: ownerEnabled})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range next.Items {
		if ids[item.ID] {
			t.Fatalf("batch after an apply reused item id %q", item.ID)
		}
	}
	if got := owner.verbs(); !slices.Equal(got, []string{"plan", "reclaim", "plan"}) {
		t.Fatalf("owner invocations = %v, want a fresh plan after the apply", got)
	}
}

func TestOwnerReclaimReportsTheOwnersRefusalVerbatim(t *testing.T) {
	for name, owner := range map[string]*scriptedOwner{
		"owner blocked":   {plan: ownerPlanJSON(0, "", "pid 4242 (go) runs outside the build-cache protocol: GOCACHEPROG unset")},
		"owner missing":   {plan: cleanup.ProcessResult{ExitCode: 1, Stdout: `{"success":false,"error":"Unknown command: go-cache"}`}, planErr: errors.New("exit status 1")},
		"owner unstarted": {planErr: errors.New(`exec: "vrooli": executable file not found in $PATH`)},
	} {
		t.Run(name, func(t *testing.T) {
			provider := newOwnerProvider(t, owner)
			estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: recoveryScope, Policy: ownerEnabled})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: recoveryScope, Policy: ownerEnabled})
			if err != nil {
				t.Fatal(err)
			}
			if estimate.EstimatedBytes != 0 || estimate.BlockedReason == "" || len(preview.Items) != 0 || preview.BlockedReason == "" {
				t.Fatalf("estimate %#v preview %#v, want a blocked reason and nothing to reclaim", estimate, preview)
			}
			for _, fragment := range []string{"GOCACHEPROG unset", "Unknown command", "executable file not found"} {
				if strings.Contains(owner.plan.Stdout+fmt.Sprint(owner.planErr), fragment) && !strings.Contains(estimate.BlockedReason, fragment) {
					t.Errorf("blocked reason %q lost the owner's explanation %q", estimate.BlockedReason, fragment)
				}
			}
			if slices.Contains(owner.verbs(), "reclaim") {
				t.Fatal("a blocked owner was asked to reclaim")
			}
		})
	}
}

func TestOwnerReclaimRunsNothingWhenDisabledOrUnconfigured(t *testing.T) {
	owner := &scriptedOwner{plan: ownerPlanJSON(gib, ownerToken, "")}
	provider := newOwnerProvider(t, owner)
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{}})
	if err != nil || estimate.BlockedReason != "provider disabled by policy" {
		t.Fatalf("disabled estimate = %#v %v", estimate, err)
	}
	unwired := newOwnerProvider(t, nil)
	if estimate, err := unwired.Estimate(context.Background(), cleanup.EstimateRequest{Scope: recoveryScope, Policy: ownerEnabled}); err != nil || estimate.BlockedReason == "" {
		t.Fatalf("estimate without a process runner = %#v %v", estimate, err)
	}
	disabled, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "k", ApprovalMode: cleanup.ApprovalModeDisabled, Preview: cleanup.Preview{Items: []cleanup.PreviewItem{{ID: "b", Bytes: gib, Action: ownerReclaimAction, Identity: ownerToken}}}})
	if err != nil || disabled.Applied || len(disabled.SkippedItems) != 1 {
		t.Fatalf("disabled apply = %#v %v", disabled, err)
	}
	if len(owner.calls) != 0 {
		t.Fatalf("owner invocations = %v; a disabled provider must neither plan nor reclaim", owner.verbs())
	}
}

func TestOwnerReclaimApplyRefusesItemsNotBoundToOnePlan(t *testing.T) {
	for name, items := range map[string][]cleanup.PreviewItem{
		"two plans":     {{ID: "a", Bytes: gib, Action: ownerReclaimAction, Identity: ownerToken}, {ID: "b", Bytes: gib, Action: ownerReclaimAction, Identity: "v1.other.1.2"}},
		"no token":      {{ID: "a", Bytes: gib, Action: ownerReclaimAction}},
		"file item":     {{ID: "a", Bytes: gib, Action: "cache-remove", Identity: ownerToken, Path: ownerRoot + "/00"}},
		"no bytes":      {{ID: "a", Action: ownerReclaimAction, Identity: ownerToken}},
		"empty preview": nil,
	} {
		t.Run(name, func(t *testing.T) {
			owner := &scriptedOwner{reclaim: ownerReceiptJSON(ownerRoot, gib, 1, "complete", "")}
			result := ownerApply(t, newOwnerProvider(t, owner), context.Background(), items)
			if result.Applied || result.ReclaimedBytes != 0 || len(result.SkippedItems) != len(items) || len(result.Warnings) == 0 {
				t.Fatalf("result = %#v, want every item skipped with a reason", result)
			}
			if len(owner.calls) != 0 {
				t.Fatalf("owner invoked for an unbound preview: %v", owner.calls)
			}
		})
	}
}

func TestOwnerReclaimApplyReportsOwnerOutcomesHonestly(t *testing.T) {
	items := []cleanup.PreviewItem{{ID: "a", Bytes: gib, Action: ownerReclaimAction, Identity: ownerToken}, {ID: "b", Bytes: gib, Action: ownerReclaimAction, Identity: ownerToken}}
	t.Run("blocked before deleting", func(t *testing.T) {
		owner := &scriptedOwner{reclaim: ownerReceiptJSON(ownerRoot, 0, 0, "blocked", "pid 9 (go) runs outside the build-cache protocol")}
		result := ownerApply(t, newOwnerProvider(t, owner), context.Background(), items)
		if result.Applied || result.ReclaimedBytes != 0 || len(result.SkippedItems) != 2 || !strings.Contains(strings.Join(result.Warnings, " "), "outside the build-cache protocol") {
			t.Fatalf("result = %#v, want a skipped batch naming the owner's reason", result)
		}
	})
	t.Run("stopped part way", func(t *testing.T) {
		owner := &scriptedOwner{reclaim: ownerReceiptJSON(ownerRoot, gib/3, 120, "blocked", "pid 9 (go) runs outside the build-cache protocol")}
		result := ownerApply(t, newOwnerProvider(t, owner), context.Background(), items)
		if !result.Applied || result.ReclaimedBytes != gib/3 || !strings.Contains(strings.Join(result.Warnings, " "), "stopped early") {
			t.Fatalf("result = %#v, want the bytes actually freed and the early stop", result)
		}
	})
	t.Run("owner failed", func(t *testing.T) {
		owner := &scriptedOwner{reclaim: cleanup.ProcessResult{ExitCode: 1, Stderr: "reclaim: store token mismatch"}, reclaimErr: errors.New("exit status 1")}
		result := ownerApply(t, newOwnerProvider(t, owner), context.Background(), items)
		if result.Applied || len(result.SkippedItems) != 2 || !strings.Contains(strings.Join(result.Warnings, " "), "token mismatch") {
			t.Fatalf("result = %#v, want a skipped batch, not a failed recovery run", result)
		}
	})
	t.Run("too little time", func(t *testing.T) {
		owner := &scriptedOwner{}
		ctx, cancel := context.WithTimeout(context.Background(), ownerDeadlineMargin)
		defer cancel()
		result := ownerApply(t, newOwnerProvider(t, owner), ctx, items)
		if result.Applied || len(result.SkippedItems) != 2 || len(owner.calls) != 0 {
			t.Fatalf("result = %#v calls=%v, want no reclaim started without time to finish", result, owner.calls)
		}
	})
}

func TestOwnerReclaimRejectsDocumentsThatBreakTheContract(t *testing.T) {
	items := []cleanup.PreviewItem{{ID: "a", Bytes: gib, Action: ownerReclaimAction, Identity: ownerToken}}
	apply := func(receipt cleanup.ProcessResult) error {
		_, err := newOwnerProvider(t, &scriptedOwner{reclaim: receipt}).Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "k", Preview: cleanup.Preview{Items: items}})
		return err
	}
	if err := apply(ownerReceiptJSON("/home/builder/elsewhere", gib, 1, "complete", "")); err == nil || !strings.Contains(err.Error(), "not the declared root") {
		t.Errorf("receipt for another cache accepted: %v", err)
	}
	if err := apply(cleanup.ProcessResult{Stdout: "freed 1 GiB"}); err == nil {
		t.Error("non-JSON receipt accepted")
	}
	if err := apply(cleanup.ProcessResult{Stdout: `{"schema":"other/v2","cache":"` + ownerRoot + `"}`}); err == nil {
		t.Error("receipt of another schema accepted")
	}
	plan := func(result cleanup.ProcessResult) error {
		_, err := newOwnerProvider(t, &scriptedOwner{plan: result}).Estimate(context.Background(), cleanup.EstimateRequest{Scope: recoveryScope, Policy: ownerEnabled})
		return err
	}
	if err := plan(ownerPlanJSON(gib, "", "")); err == nil || !strings.Contains(err.Error(), "without a plan token") {
		t.Errorf("plan proposing bytes without a token accepted: %v", err)
	}
	wrongRoot := ownerPlanJSON(gib, ownerToken, "")
	wrongRoot.Stdout = strings.Replace(wrongRoot.Stdout, ownerRoot, "/tmp/other", 1)
	if err := plan(wrongRoot); err == nil || !strings.Contains(err.Error(), "not the declared root") {
		t.Errorf("plan for another cache accepted: %v", err)
	}
}

func TestOwnerReclaimWithinBudgetOffersNothing(t *testing.T) {
	provider := newOwnerProvider(t, &scriptedOwner{plan: ownerPlanJSON(0, "", "")})
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: recoveryScope, Policy: ownerEnabled})
	if err != nil || estimate.EstimatedBytes != 0 || estimate.ItemCount != 0 || estimate.BlockedReason != "" {
		t.Fatalf("within-budget estimate = %#v %v", estimate, err)
	}
}
