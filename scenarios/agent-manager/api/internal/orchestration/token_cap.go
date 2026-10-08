// This file enforces per-run weighted-token caps from persisted usage events.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
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

// Environment keys for the internal cap configuration. Profile and run fields
// that set a cap need proto changes and arrive later; until then a cap is set
// per agent profile key.
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
// same weighting `agent-manager run tokens` reports. An unknown model weighs 1.
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

// DefaultTokenCapPolicy is off, with the `run tokens` weights (sol=10, luna=1)
// and the 85% nudge.
func DefaultTokenCapPolicy() TokenCapPolicy {
	return TokenCapPolicy{Mode: TokenCapOff, Weights: map[string]float64{"sol": 10, "luna": 1}, NudgeFraction: 0.85}
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
	weights := map[string]float64{}
	for _, pair := range strings.Split(raw, ",") {
		tier, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		weight, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if !ok || strings.TrimSpace(tier) == "" || err != nil || weight < 0 {
			return nil, fmt.Errorf("invalid weight %q (want tier=number)", pair)
		}
		weights[strings.ToLower(strings.TrimSpace(tier))] = weight
	}
	return weights, nil
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
	model = strings.ToLower(model)
	tiers := make([]string, 0, len(p.Weights))
	for tier := range p.Weights {
		tiers = append(tiers, tier)
	}
	sort.Strings(tiers)
	for _, tier := range tiers {
		if strings.Contains(model, tier) {
			return p.Weights[tier]
		}
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
		exact, err := o.weightedRunTokens(ctx, run, policy)
		if err != nil {
			obs.Component("token-cap").Warn("token cap recount failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
			return
		}
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

// tokenCapFor resolves a run's cap from its agent profile key.
func (o *Orchestrator) tokenCapFor(ctx context.Context, run *domain.Run, policy TokenCapPolicy) int64 {
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
// projection as RunAccounting, so the cap and `run tokens` never disagree.
func (o *Orchestrator) weightedRunTokens(ctx context.Context, run *domain.Run, policy TokenCapPolicy) (float64, error) {
	events, err := o.allRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeMetric}})
	if err != nil {
		return 0, err
	}
	accounting := withTokenComponents(RunAccounting{}, run, events, o.now())
	return float64(accounting.NonCacheTokens) * policy.weight(accounting.Model), nil
}

func (o *Orchestrator) recordTokenCapNudge(ctx context.Context, run *domain.Run, weighted float64, limit int64, policy TokenCapPolicy) {
	message := fmt.Sprintf("token cap: %.0f of %d weighted tokens used (%.0f%%); log a slice line, write the handoff, and end the run", weighted, limit, 100*weighted/float64(limit))
	if policy.Mode == TokenCapReport {
		message = fmt.Sprintf("token cap (report-only): %.0f of %d weighted tokens used (%.0f%%)", weighted, limit, 100*weighted/float64(limit))
	}
	o.appendTokenCapEvent(ctx, run.ID, message)
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

// stopRunForTokenCap stops the run through the ordinary graceful stop, then
// records the typed terminal pair. A run that another path already ended keeps
// its own stop reason.
func (o *Orchestrator) stopRunForTokenCap(ctx context.Context, runID uuid.UUID) error {
	current, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	if current.Status.IsTerminal() {
		return nil
	}
	if err := o.StopRun(ctx, runID); err != nil {
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
