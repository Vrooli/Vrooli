package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
)

// OwnerReclaimProvider delegates a root to the tool that owns it. Storage
// Manager plans and invokes; the owner decides what is safe to remove, binds
// its plan to the exact store with a token, enforces its own budget, and
// reports what it freed. Storage Manager never walks or deletes the root.
//
// The owner contract (owner-reclaim/v1) is two verbs appended to the root's
// declared command, each printing one JSON document:
//
//	<command> plan --cache <root> --json
//	  {schema, cache, store_id, token, store_bytes, budget_bytes,
//	   reclaimable_bytes, blocked_reason}
//	<command> reclaim --cache <root> --token <token> --max-bytes <n> [--time-budget <d>] --json
//	  {schema, cache, store_id, freed_bytes, removed_entries, stop_reason,
//	   plan_exhausted, blocked_reason}
//
// A blocked owner exits 0 and names the reason; a non-zero exit is a failure.
// The Go build cache is the first owner: `vrooli host go-cache`, whose
// protocol pins every output a live build holds and audits the host's go
// commands before each deletion batch.
type OwnerReclaimProvider struct {
	cfg    OwnerReclaimProviderConfig
	runner cleanup.ProcessRunner
	clock  cleanup.Clock

	mu   sync.Mutex
	last *cachedOwnerPlan
}

// OwnerReclaimProviderConfig is the resolved, host-specific view of a root
// declaring owner_reclaim_command.
type OwnerReclaimProviderConfig struct {
	ID        string
	Name      string
	Root      string
	Command   cleanup.ProcessCommand
	Platforms []string
}

type ownerPlan struct {
	Schema           string `json:"schema"`
	Cache            string `json:"cache"`
	StoreID          string `json:"store_id"`
	Token            string `json:"token"`
	StoreBytes       int64  `json:"store_bytes"`
	BudgetBytes      int64  `json:"budget_bytes"`
	ReclaimableBytes int64  `json:"reclaimable_bytes"`
	BlockedReason    string `json:"blocked_reason"`
}

type ownerReceipt struct {
	Schema         string `json:"schema"`
	Cache          string `json:"cache"`
	StoreID        string `json:"store_id"`
	FreedBytes     int64  `json:"freed_bytes"`
	RemovedEntries int    `json:"removed_entries"`
	StopReason     string `json:"stop_reason"`
	PlanExhausted  bool   `json:"plan_exhausted"`
	BlockedReason  string `json:"blocked_reason"`
}

type cachedOwnerPlan struct {
	plan ownerPlan
	at   time.Time
}

const (
	// ownerReclaimChunk sizes preview items so the recovery controller's
	// bounded batches map onto bounded owner applies.
	ownerReclaimChunk = int64(1 << 30)
	ownerReclaimItems = 256
	// ownerPlanReuse lets a preview reuse the plan its estimate just made.
	ownerPlanReuse = time.Minute
	// ownerDeadlineMargin leaves the owner time to stop between batches and
	// report before the caller's deadline.
	ownerDeadlineMargin = 5 * time.Second
	ownerReclaimAction  = "owner-reclaim"
)

func NewOwnerReclaimProvider(runner cleanup.ProcessRunner, clock cleanup.Clock, cfg OwnerReclaimProviderConfig) (*OwnerReclaimProvider, error) {
	p := &OwnerReclaimProvider{cfg: cfg, runner: runner, clock: clock}
	if err := cleanup.ValidateProvider(p); err != nil {
		return nil, err
	}
	return p, nil
}

// GovernedRoots reports the root the owner maintains.
func (p *OwnerReclaimProvider) GovernedRoots() []string {
	if strings.TrimSpace(p.cfg.Root) == "" {
		return nil
	}
	return []string{p.cfg.Root}
}

func (p *OwnerReclaimProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  p.cfg.ID,
		Name:                p.cfg.Name,
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          cleanup.SafetyTierSafeWithOwner,
		DefaultMode:         cleanup.ProviderModeDisabled,
		DefaultApproval:     cleanup.ApprovalModeOwner,
		OwnerBudget:         true,
		SupportedPlatforms:  append([]string(nil), p.cfg.Platforms...),
		IrreversibleEffects: []string{"the owning tool removes entries it proves unused, down to its declared budget"},
		TestSubstitute:      "fake-process-runner",
	}
}

func (p *OwnerReclaimProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	meta := p.Metadata()
	out := cleanup.Estimate{ProviderID: meta.ID, ProviderVersion: meta.Version, RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone, ObservedAt: p.now(req.Scope)}
	if reason := p.blockedReason(req.Policy); reason != "" {
		out.BlockedReason = reason
		return out, nil
	}
	plan, err := p.plan(ctx, false)
	if reason, blocked := ownerBlocked(plan, err); blocked {
		out.BlockedReason = reason
		return out, nil
	}
	if err != nil {
		return cleanup.Estimate{}, err
	}
	out.EstimatedBytes = plan.ReclaimableBytes
	out.ItemCount = len(ownerItems(meta, p.cfg.Root, plan))
	return out, nil
}

func (p *OwnerReclaimProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	meta := p.Metadata()
	out := cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version}
	if reason := p.blockedReason(req.Policy); reason != "" {
		out.BlockedReason = reason
		return out, nil
	}
	plan, err := p.plan(ctx, true)
	if reason, blocked := ownerBlocked(plan, err); blocked {
		out.BlockedReason = reason
		return out, nil
	}
	if err != nil {
		return cleanup.Preview{}, err
	}
	out.Items = ownerItems(meta, p.cfg.Root, plan)
	return out, nil
}

// ownerBlocked reports a plan the owner declined, or an owner that could not
// plan at all (not installed, refused, exited non-zero), as a blocked reason
// the catalog and recovery audit show verbatim. A contract violation stays an
// error.
func ownerBlocked(plan ownerPlan, err error) (string, bool) {
	var failed *ownerCommandError
	switch {
	case errors.As(err, &failed):
		return failed.Error(), true
	case err == nil && plan.BlockedReason != "":
		return "owner: " + plan.BlockedReason, true
	default:
		return "", false
	}
}

// ownerItems splits the plan into bounded items. Every item carries the plan
// token as its identity, so apply is bound to the plan and store previewed.
func ownerItems(meta cleanup.ProviderMetadata, root string, plan ownerPlan) []cleanup.PreviewItem {
	if plan.Token == "" || plan.ReclaimableBytes <= 0 {
		return nil
	}
	items := make([]cleanup.PreviewItem, 0, min(int((plan.ReclaimableBytes+ownerReclaimChunk-1)/ownerReclaimChunk), ownerReclaimItems))
	for offset := int64(0); offset < plan.ReclaimableBytes && len(items) < ownerReclaimItems; offset += ownerReclaimChunk {
		bytes := min(ownerReclaimChunk, plan.ReclaimableBytes-offset)
		items = append(items, cleanup.PreviewItem{
			// The remaining figure shrinks after every apply, so a later
			// batch of the same plan never reuses an earlier batch's ids.
			ID:          fmt.Sprintf("%s:%s:%d:%d", meta.ID, plan.Token, plan.ReclaimableBytes, offset),
			Path:        root,
			Description: fmt.Sprintf("%s: least-recently-used entries no live build holds (store %d bytes, owner budget %d bytes)", meta.Name, plan.StoreBytes, plan.BudgetBytes),
			Bytes:       bytes,
			Action:      ownerReclaimAction,
			SafetyTier:  meta.SafetyTier,
			Identity:    plan.Token,
		})
	}
	return items
}

func (p *OwnerReclaimProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	meta := p.Metadata()
	if req.ProviderVersion != meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", meta.ID, req.ProviderVersion, meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", meta.ID)
	}
	ids := previewItemIDs(req.Preview.Items)
	skip := func(reason string) (cleanup.ApplyResult, error) {
		return cleanup.ApplyResult{ProviderID: meta.ID, SkippedItems: ids, Warnings: []string{reason}}, nil
	}
	if req.ApprovalMode == cleanup.ApprovalModeDisabled {
		return skip("provider disabled by policy")
	}
	if reason := p.blockedReason(cleanup.ProviderPolicy{Enabled: true}); reason != "" {
		return skip(reason)
	}
	token, maxBytes, err := ownerApplyBounds(req.Preview.Items)
	if err != nil {
		return skip(err.Error())
	}
	args := append(append([]string(nil), p.cfg.Command.Args...), "reclaim", "--cache", p.cfg.Root, "--token", token, "--max-bytes", strconv.FormatInt(maxBytes, 10))
	if deadline, ok := ctx.Deadline(); ok {
		budget := time.Until(deadline) - ownerDeadlineMargin
		if budget < time.Second {
			return skip("too little time remains to reclaim within the caller's deadline")
		}
		args = append(args, "--time-budget", budget.Round(time.Second).String())
	}
	var receipt ownerReceipt
	err = p.run(ctx, append(args, "--json"), &receipt)
	p.forgetPlan()
	var failed *ownerCommandError
	if errors.As(err, &failed) {
		// The owner refused or could not start. Its bounded sweep removes
		// nothing it has not proven released, so a failed invocation is a
		// skipped batch, not a failed recovery run.
		return skip(failed.Error())
	}
	if err != nil {
		return cleanup.ApplyResult{}, err
	}
	if receipt.Cache != p.cfg.Root {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s owner reclaimed %q, not the declared root %q", meta.ID, receipt.Cache, p.cfg.Root)
	}
	if receipt.FreedBytes <= 0 && receipt.RemovedEntries == 0 {
		reason := receipt.BlockedReason
		if reason == "" {
			reason = "owner reclaimed nothing (" + receipt.StopReason + ")"
		}
		return skip("owner: " + reason)
	}
	result := cleanup.ApplyResult{ProviderID: meta.ID, Applied: true, AppliedItems: ids, ReclaimedBytes: receipt.FreedBytes}
	if receipt.BlockedReason != "" {
		result.Warnings = append(result.Warnings, "owner stopped early: "+receipt.BlockedReason)
	}
	return result, nil
}

// ownerApplyBounds requires every item to name the same plan token and sums
// their bytes into the owner's bound for this apply.
func ownerApplyBounds(items []cleanup.PreviewItem) (string, int64, error) {
	token := ""
	var total int64
	for _, item := range items {
		if item.Action != ownerReclaimAction || item.Identity == "" {
			return "", 0, errors.New("preview item is not an owner-reclaim item with a plan token")
		}
		if token != "" && item.Identity != token {
			return "", 0, errors.New("preview items name different owner plans")
		}
		token = item.Identity
		total += item.Bytes
	}
	if token == "" || total <= 0 {
		return "", 0, errors.New("preview holds no owner-reclaim bytes")
	}
	return token, total, nil
}

func (p *OwnerReclaimProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "owner-reclaim provider delegates removal to the root's declared owner and reports the owner's freed bytes"}, nil
}

func (p *OwnerReclaimProvider) blockedReason(policy cleanup.ProviderPolicy) string {
	switch {
	case !policy.Enabled:
		return "provider disabled by policy"
	case p.runner == nil:
		return "process runner unavailable"
	case strings.TrimSpace(p.cfg.Command.Name) == "":
		return "no owner reclaim command configured"
	case strings.TrimSpace(p.cfg.Root) == "":
		return "root unavailable on this host"
	default:
		return ""
	}
}

// plan asks the owner for its current plan. A preview reuses the plan its
// estimate made moments earlier instead of asking the owner twice.
func (p *OwnerReclaimProvider) plan(ctx context.Context, reuse bool) (ownerPlan, error) {
	p.mu.Lock()
	last := p.last
	p.mu.Unlock()
	if reuse && last != nil && p.wallNow().Sub(last.at) < ownerPlanReuse {
		return last.plan, nil
	}
	args := append(append([]string(nil), p.cfg.Command.Args...), "plan", "--cache", p.cfg.Root, "--json")
	var plan ownerPlan
	if err := p.run(ctx, args, &plan); err != nil {
		return ownerPlan{}, err
	}
	if plan.Cache != p.cfg.Root {
		return ownerPlan{}, fmt.Errorf("provider %s owner planned %q, not the declared root %q", p.cfg.ID, plan.Cache, p.cfg.Root)
	}
	if plan.BlockedReason == "" && plan.ReclaimableBytes > 0 && plan.Token == "" {
		return ownerPlan{}, fmt.Errorf("provider %s owner proposed %d bytes without a plan token", p.cfg.ID, plan.ReclaimableBytes)
	}
	p.mu.Lock()
	p.last = &cachedOwnerPlan{plan: plan, at: p.wallNow()}
	p.mu.Unlock()
	return plan, nil
}

func (p *OwnerReclaimProvider) forgetPlan() {
	p.mu.Lock()
	p.last = nil
	p.mu.Unlock()
}

// ownerCommandError is an owner invocation that did not complete: the
// command could not start, or it exited non-zero. A completed invocation that
// breaks the contract is a different, plain error.
type ownerCommandError struct {
	verb   string
	exit   int
	detail string
}

func (e *ownerCommandError) Error() string {
	return fmt.Sprintf("owner %s failed (exit %d): %s", e.verb, e.exit, e.detail)
}

// run invokes one owner verb and decodes its single JSON document.
func (p *OwnerReclaimProvider) run(ctx context.Context, args []string, into any) error {
	command := cleanup.ProcessCommand{Name: p.cfg.Command.Name, Args: args}
	verb := args[len(p.cfg.Command.Args)]
	result, err := p.runner.Run(ctx, command)
	if err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			// A CLI that reports failures as a JSON envelope does so on stdout.
			detail = strings.TrimSpace(result.Stdout)
		}
		if detail == "" && err != nil {
			detail = err.Error()
		}
		return &ownerCommandError{verb: verb, exit: result.ExitCode, detail: cleanup.Redact(detail)}
	}
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil {
		return fmt.Errorf("provider %s owner %s printed no valid JSON document: %w", p.cfg.ID, verb, err)
	}
	if !strings.HasSuffix(envelope.Schema, "/v1") {
		return fmt.Errorf("provider %s owner %s document schema %q is not an owner-reclaim/v1 document", p.cfg.ID, verb, envelope.Schema)
	}
	if err := json.Unmarshal([]byte(result.Stdout), into); err != nil {
		return fmt.Errorf("provider %s owner %s document does not match owner-reclaim/v1: %w", p.cfg.ID, verb, err)
	}
	return nil
}

func (p *OwnerReclaimProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	return p.wallNow()
}

func (p *OwnerReclaimProvider) wallNow() time.Time {
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now()
}
