// This file enforces per-run weighted-token caps from persisted usage events.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agentcatalog"
)

// TokenCapMode selects what a weighted-token cap does (DL-8).
type TokenCapMode string

const (
	// TokenCapOff disables cap tracking. It is the default.
	TokenCapOff TokenCapMode = "off"
	// TokenCapReport records the 85% nudge and a "would stop" event on the run
	// timeline but never stops a run (decision P-08: report-only first).
	TokenCapReport TokenCapMode = "report"
	// TokenCapEnforce records the nudge and stops the run at its cap with
	// stop_reason=token_cap.
	TokenCapEnforce TokenCapMode = "enforce"
)

// Environment keys for the cap configuration. A run's cap is its resolved
// MaxWeightedTokens (the profile's AgentProfile.MaxWeightedTokens, lowered by
// a CreateRun override); AGENT_MANAGER_TOKEN_CAPS, keyed by profile key, is
// the fallback for runs without one. The mode applies to both.
const (
	tokenCapModeEnv    = "AGENT_MANAGER_TOKEN_CAP_MODE"
	tokenCapsEnv       = "AGENT_MANAGER_TOKEN_CAPS"        // JSON object: profile key -> weighted cap
	tokenCapWeightsEnv = "AGENT_MANAGER_TOKEN_CAP_WEIGHTS" // "sol=10,luna=1"
)

// tokenCapCheckTimeout bounds one exact recount of a run's persisted usage.
const tokenCapCheckTimeout = 30 * time.Second

// TokenCapPolicy is the internal weighted-token cap configuration. Weighted
// tokens are non-cache tokens (input, output and cache writes; cache reads
// excluded) times the weight of the first tier name the model contains, the
// same weighting `agent-manager run report` reports. Unknown enforcement retains the legacy weight 1; cost comparisons remain unknown.
type TokenCapPolicy struct {
	Mode TokenCapMode
	// Caps maps an agent profile key to its weighted-token cap per run. A
	// missing key or a non-positive value means no cap.
	Caps map[string]int64
	// Weights maps a model-name substring (tier) to its weight.
	Weights map[string]float64
	// NudgeFraction is the share of the cap at which the run is nudged once.
	NudgeFraction float64
}

// DefaultTokenCapPolicy is off, with the shared heuristic weights (sol=10, terra=5, luna=1)
// and the 85% nudge.
func DefaultTokenCapPolicy() TokenCapPolicy {
	return TokenCapPolicy{Mode: TokenCapOff, Weights: agentcatalog.DefaultTokenWeights(), NudgeFraction: 0.85}
}

// TokenCapPolicyFromEnv reads the internal cap configuration. Unset or invalid
// values keep the defaults, so a bad value can never enable enforcement.
func TokenCapPolicyFromEnv() TokenCapPolicy {
	policy := DefaultTokenCapPolicy()
	switch mode := TokenCapMode(strings.ToLower(strings.TrimSpace(os.Getenv(tokenCapModeEnv)))); mode {
	case TokenCapReport, TokenCapEnforce:
		policy.Mode = mode
	}
	if raw := strings.TrimSpace(os.Getenv(tokenCapsEnv)); raw != "" {
		var caps map[string]int64
		if err := json.Unmarshal([]byte(raw), &caps); err != nil {
			obs.Component("token-cap").Warn("ignoring invalid token caps", "env", tokenCapsEnv, obs.KeyError, err.Error())
		} else {
			policy.Caps = caps
		}
	}
	if raw := strings.TrimSpace(os.Getenv(tokenCapWeightsEnv)); raw != "" {
		if weights, err := parseTokenCapWeights(raw); err != nil {
			obs.Component("token-cap").Warn("ignoring invalid token cap weights", "env", tokenCapWeightsEnv, obs.KeyError, err.Error())
		} else {
			policy.Weights = weights
		}
	}
	return policy
}

func parseTokenCapWeights(raw string) (map[string]float64, error) {
	return agentcatalog.ParseTokenWeights(raw, nil)
}

// WithTokenCapPolicy installs the weighted-token cap configuration.
func WithTokenCapPolicy(policy TokenCapPolicy) Option {
	return func(o *Orchestrator) {
		o.tokenCaps = &tokenCapEnforcer{policy: policy}
	}
}

// tokenCapEnforcer tracks each active run's weighted usage between exact
// recounts. The per-event estimate is an upper bound (it ignores the
// receipt/per-turn deduplication accounting applies), so an exact recount from
// persisted events runs only when the estimate crosses a threshold.
type tokenCapEnforcer struct {
	policy TokenCapPolicy
	runs   sync.Map // uuid.UUID -> *tokenCapRun
}

type tokenCapRun struct {
	mu       sync.Mutex
	resolved bool    // the cap has been resolved from the run's profile
	limit    int64   // weighted cap; 0 means none
	estimate float64 // upper bound of weighted tokens
	nudged   bool
	reached  bool
	checking bool
}

// weight returns the tier weight of a model; an unknown model weighs 1.
func (p TokenCapPolicy) weight(model string) float64 {
	if weight, known := agentcatalog.TokenWeight(model, p.Weights); known {
		return weight
	}
	return 1
}

// maxWeight keeps the per-event estimate an upper bound for usage that names
// no model.
func (p TokenCapPolicy) maxWeight() float64 {
	highest := 1.0
	for _, weight := range p.Weights {
		if weight > highest {
			highest = weight
		}
	}
	return highest
}

// observeTokenCapUsage runs after a usage event is persisted. It only updates
// the in-memory estimate; recounts and actions run off the event-emission path
// so a stop can never wait on the stream that is reporting the usage.
func (o *Orchestrator) observeTokenCapUsage(runID uuid.UUID, evt *domain.RunEvent) {
	enforcer := o.tokenCaps
	if enforcer == nil || enforcer.policy.Mode == TokenCapOff || evt == nil {
		return
	}
	usage, ok := evt.Data.(*domain.UsageEventData)
	if !ok {
		return
	}
	weight := enforcer.policy.maxWeight()
	if strings.TrimSpace(usage.Model) != "" && usage.Model != "unknown" {
		weight = enforcer.policy.weight(usage.Model)
	}
	value, _ := enforcer.runs.LoadOrStore(runID, &tokenCapRun{})
	state := value.(*tokenCapRun)
	state.mu.Lock()
	state.estimate += float64(usage.InputTokens+usage.OutputTokens+usage.CacheCreationTokens) * weight
	if usage.Cumulative != nil {
		// A report whose amount is unknown still bounds the run by its
		// session counter, so a resumed session cannot slip past the cap
		// unrecounted.
		state.estimate = max(state.estimate, float64(usage.Cumulative.Total()-int64(usage.Cumulative.CacheReadTokens))*weight)
	}
	check := !state.checking && !state.reached && (!state.resolved || state.crossesThreshold(enforcer.policy.NudgeFraction))
	if check {
		state.checking = true
	}
	state.mu.Unlock()
	if check {
		go o.checkTokenCap(runID, state)
	}
}

// crossesThreshold reports whether the estimate reached the next unreported
// threshold. Callers hold state.mu.
func (s *tokenCapRun) crossesThreshold(nudgeFraction float64) bool {
	if s.limit <= 0 {
		return false
	}
	if s.nudged {
		return s.estimate >= float64(s.limit)
	}
	return s.estimate >= nudgeFraction*float64(s.limit)
}

// checkTokenCap resolves the run's cap on first sight, recounts its weighted
// usage exactly from persisted events, and records the nudge or the cap. It
// repeats while usage that arrived during a recount crossed a threshold, so a
// final receipt is never left unchecked.
func (o *Orchestrator) checkTokenCap(runID uuid.UUID, state *tokenCapRun) {
	defer obs.RecoverToFailure("token cap check", nil)
	defer func() {
		state.mu.Lock()
		state.checking = false
		state.mu.Unlock()
	}()
	policy := o.tokenCaps.policy
	ctx, cancel := context.WithTimeout(context.Background(), tokenCapCheckTimeout)
	defer cancel()

	run, err := o.runs.Get(ctx, runID)
	if err != nil || run == nil {
		return
	}
	state.mu.Lock()
	resolved := state.resolved
	state.mu.Unlock()
	if !resolved {
		limit := o.tokenCapFor(ctx, run, policy)
		state.mu.Lock()
		state.resolved, state.limit = true, limit
		state.mu.Unlock()
	}
	for {
		state.mu.Lock()
		limit, before := state.limit, state.estimate
		state.mu.Unlock()
		if limit <= 0 {
			return
		}
		observed, err := o.weightedRunTokens(ctx, run, policy)
		if err != nil {
			obs.Component("token-cap").Warn("token cap recount failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
			return
		}
		if observed == nil {
			return
		}
		exact := *observed
		state.mu.Lock()
		// Keep usage observed during the recount; replace only the counted part.
		arrived := state.estimate - before
		state.estimate = exact + arrived
		nudge := !state.nudged && exact >= policy.NudgeFraction*float64(limit)
		reached := !state.reached && exact >= float64(limit)
		if nudge || reached {
			state.nudged = true
		}
		if reached {
			state.reached = true
		}
		again := !state.reached && arrived > 0 && state.crossesThreshold(policy.NudgeFraction)
		state.mu.Unlock()

		switch {
		case reached:
			o.recordTokenCapReached(ctx, run, exact, limit, policy.Mode)
		case nudge:
			o.recordTokenCapNudge(ctx, run, exact, limit, policy)
		}
		if !again {
			return
		}
	}
}

// tokenCapFor resolves a run's cap: its resolved MaxWeightedTokens, else the
// environment cap of its agent profile key.
func (o *Orchestrator) tokenCapFor(ctx context.Context, run *domain.Run, policy TokenCapPolicy) int64 {
	if run.ResolvedConfig != nil && run.ResolvedConfig.MaxWeightedTokens > 0 {
		return run.ResolvedConfig.MaxWeightedTokens
	}
	if run.AgentProfileID == nil || o.profiles == nil || len(policy.Caps) == 0 {
		return 0
	}
	profile, err := o.profiles.Get(ctx, *run.AgentProfileID)
	if err != nil || profile == nil {
		return 0
	}
	return policy.Caps[profile.ProfileKey]
}

// weightedRunTokens recounts a run's weighted non-cache tokens with the same
// projection as RunAccounting, so the cap and `run report` never disagree.
func (o *Orchestrator) weightedRunTokens(ctx context.Context, run *domain.Run, policy TokenCapPolicy) (*float64, error) {
	events, err := o.meteredRunEvents(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if _, ok := event.Data.(*domain.UsageEventData); ok {
			accounting := withTokenComponents(RunAccounting{}, run, events, o.now())
			weighted := float64(accounting.NonCacheTokens) * policy.weight(accounting.Model)
			return &weighted, nil
		}
	}
	return nil, nil
}

// recordTokenCapNudge records the nudge on the run timeline. With
// enforcement on, an interactive run also receives it as input, so the agent
// can log its slice and write the handoff before the stop; a codec-pipe run
// cannot take input mid-turn, so it sees only the stop's stop_reason.
func (o *Orchestrator) recordTokenCapNudge(ctx context.Context, run *domain.Run, weighted float64, limit int64, policy TokenCapPolicy) {
	message := fmt.Sprintf("token cap: %.0f of %d weighted tokens used (%.0f%%); log a slice line, write the handoff, and end the run", weighted, limit, 100*weighted/float64(limit))
	if policy.Mode == TokenCapReport {
		message = fmt.Sprintf("token cap (report-only): %.0f of %d weighted tokens used (%.0f%%)", weighted, limit, 100*weighted/float64(limit))
	}
	o.appendTokenCapEvent(ctx, run.ID, message)
	if policy.Mode != TokenCapEnforce || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive {
		return
	}
	nudge := fmt.Sprintf("[agent-manager] Token cap: %s. At 100%% Agent Manager stops this run with stop_reason=%s.", message, domain.RunStopReasonTokenCap)
	if _, err := o.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: nudge, IdempotencyKey: "token-cap-nudge:" + run.ID.String()}); err != nil {
		obs.Component("token-cap").Warn("token cap nudge not delivered", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

func (o *Orchestrator) recordTokenCapReached(ctx context.Context, run *domain.Run, weighted float64, limit int64, mode TokenCapMode) {
	if mode != TokenCapEnforce {
		o.appendTokenCapEvent(ctx, run.ID, fmt.Sprintf("token cap (report-only): %.0f weighted tokens reached the cap of %d; enforcement would stop this run with stop_reason=%s", weighted, limit, domain.RunStopReasonTokenCap))
		return
	}
	o.appendTokenCapEvent(ctx, run.ID, fmt.Sprintf("token cap: %.0f weighted tokens reached the cap of %d; stopping the run with stop_reason=%s", weighted, limit, domain.RunStopReasonTokenCap))
	if err := o.stopRunForTokenCap(ctx, run.ID); err != nil {
		obs.Component("token-cap").Warn("token cap stop failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

func (o *Orchestrator) appendTokenCapEvent(ctx context.Context, runID uuid.UUID, message string) {
	if o.events == nil {
		return
	}
	if err := o.appendAndBroadcastEvents(ctx, runID, domain.NewLogEvent(runID, "warn", message)); err != nil {
		obs.Component("token-cap").Warn("token cap event append failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
	}
}

// stopRunForTokenCap stops the run through the ordinary graceful stop with
// stop_reason=token_cap written in the same update as the cancelled status.
// The interactive path, and a process exit that finalizes the cancellation
// first, end without a reason; for those the typed pair is written after. A
// run that another path already ended keeps its own stop reason.
func (o *Orchestrator) stopRunForTokenCap(ctx context.Context, runID uuid.UUID) error {
	current, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	if current.Status.IsTerminal() {
		return nil
	}
	if err := o.stopRun(ctx, runID, domain.RunStopReasonTokenCap); err != nil {
		return err
	}
	stopped, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	if stopped.Status != domain.RunStatusCancelled || stopped.StopReason != "" {
		return nil
	}
	stopped.TerminalClass = domain.TerminalClassForStopReason(domain.RunStopReasonTokenCap)
	stopped.StopReason = domain.RunStopReasonTokenCap
	stopped.UpdatedAt = o.now()
	return o.runs.Update(ctx, stopped)
}

// capPolicy is the installed cap policy, or the default (off, sol=10,
// luna=1) when none is installed.
func (o *Orchestrator) capPolicy() TokenCapPolicy {
	if o.tokenCaps != nil {
		return o.tokenCaps.policy
	}
	return DefaultTokenCapPolicy()
}

// WeightedRunTokens is a run's metered non-cache tokens times its model tier
// weight, the weighting the caps use. known is false when the run's usage is
// not authoritatively metered.
func (o *Orchestrator) WeightedRunTokens(ctx context.Context, runID uuid.UUID) (float64, bool, error) {
	accounting, err := o.RunAccounting(ctx, runID)
	if err != nil {
		return 0, false, err
	}
	weight, known := agentcatalog.TokenWeight(accounting.Model, o.capPolicy().Weights)
	return float64(accounting.NonCacheTokens) * weight, accounting.TokensKnown && known, nil
}

// RunTokens reads a run's token total from its usage account.
func (o *Orchestrator) RunTokens(ctx context.Context, runID uuid.UUID) (int64, bool, error) {
	accounting, err := o.RunAccounting(ctx, runID)
	if err != nil {
		return 0, false, err
	}
	return accounting.Tokens, accounting.TokensKnown, nil
}

// TokenCapUsage reads an active run's weighted usage recounted from persisted
// usage events (nil before any receipt or when unreadable), its resolved cap
// (0 for none), and the cap mode.
func (o *Orchestrator) TokenCapUsage(ctx context.Context, runID uuid.UUID) (*float64, int64, string) {
	policy := o.capPolicy()
	run, err := o.runs.Get(ctx, runID)
	if err != nil || run == nil {
		return nil, 0, string(policy.Mode)
	}
	limit := o.tokenCapFor(ctx, run, policy)
	weighted, err := o.weightedRunTokens(ctx, run, policy)
	if err != nil {
		return nil, limit, string(policy.Mode)
	}
	return weighted, limit, string(policy.Mode)
}
