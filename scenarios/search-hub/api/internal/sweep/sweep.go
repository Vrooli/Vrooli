// Package sweep is search-hub's automated, overfit-safe search-tuning optimizer
// — the closed loop that turns the eval harness into a self-tuning system. It is
// the TRANSPORT-FREE core (boundary-of-responsibility): it depends only on small
// consumer-declared seams (a suite reader, a provider reader, an arm runner, a
// config controller, a clock) and never on HTTP, Connect, or a concrete store.
// The handler wiring (handlers/eval) satisfies the seams with the real eval
// Runner + store, the registry store, and the control client.
//
// The algorithm (plan §7 Phase 6):
//
//	incumbent  = the provider's current tuning (always evaluated, the baseline)
//	query-time = full-factorial query-time arms via per-request overrides (cheap)
//	index-time = coordinate-ascent index-time arms via config-push → reindex →
//	             poll → run (expensive; skipped when query_time_only)
//	decide     = rank feasible arms, then promote the best ONLY if it clears all
//	             four overfit guards (significance, held-out, constraints,
//	             complexity tie-break); otherwise keep the incumbent
//	write-back = on a cleared winner with apply=true, persist via config-write
//
// Every arm is one stored, tagged, self-describing EvalRun.
package sweep

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/vrooli/api-core/schedule"

	aisearch "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/proto"
)

// Arm tiers (the SweepArm.Tier value).
const (
	tierIncumbent = "incumbent"
	tierQueryTime = "query_time"
	tierIndexTime = "index_time"
)

// SuiteReader resolves the suite to optimize. eval.Store satisfies it.
type SuiteReader interface {
	GetSuite(ctx context.Context, suiteID string) (*evalv1.EvalSuite, error)
}

// ProviderReader resolves the provider descriptor (its incumbent tuning + the
// control endpoints) and the server-side control token. The registry SQLite
// store satisfies both methods.
type ProviderReader interface {
	Get(ctx context.Context, providerID string) (*registryv1.ProviderDescriptor, error)
	Token(ctx context.Context, providerID string) (string, error)
}

// ArmRunner runs the suite under one arm and returns the stored, tagged run. A
// nil overrides is the baseline path (the provider's live config — used for the
// incumbent and every index-time arm, whose config the orchestrator has already
// pushed live). A non-nil overrides carries the arm's query-time factors + the
// control token. The production adapter calls eval.Runner.RunWith then persists.
type ArmRunner interface {
	Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, overrides *aisearch.SearchOverrides, controlToken string, limit int32) (*evalv1.EvalRun, error)
}

// ConfigController drives a provider's secured control plane for the index-time
// tier and the write-back. search-hub's internal/control.Client satisfies it.
type ConfigController interface {
	WriteConfig(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken string, tuning *registryv1.Tuning, dryRun bool) (*controlv1.WriteConfigResponse, error)
	ReindexStatus(ctx context.Context, d *registryv1.ProviderDescriptor, controlToken, jobID string) (*controlv1.ReindexStatusResponse, error)
}

// TuningCache refreshes search-hub's cached copy of a provider's tuning after a
// write-back. WriteConfig persists the winner into the provider's search.json (the
// SSOT), but the registry cache still holds the old tuning until the provider's
// next boot re-registers — so ListProviders/Get would report stale tuning in the
// meantime. Re-upserting the descriptor with the freshly written tuning closes
// that symmetric staleness immediately. The registry store's Upsert satisfies it.
// Optional: nil skips the refresh (the next boot re-syncs regardless).
type TuningCache interface {
	Upsert(ctx context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (created bool, controlToken string, err error)
}

// Deps are the seams the orchestrator is constructed over.
type Deps struct {
	Suites    SuiteReader
	Providers ProviderReader
	Runner    ArmRunner
	Control   ConfigController
	// Cache, when set, is refreshed with the winning tuning after a successful
	// write-back so the registry cache mirrors the file without a reboot.
	Cache TuningCache
	Clock schedule.Clock
	// Sleep waits between reindex-status polls; tests inject a no-op. Defaults to
	// time.Sleep when nil.
	Sleep func(time.Duration)
	// Rand seeds the bootstrap resampler; tests inject a fixed seed for
	// determinism. Defaults to a time-seeded source when nil.
	Rand *rand.Rand
}

// Options tune the guards and the index-time poll loop. The zero value fills
// every field with the documented default via withDefaults.
type Options struct {
	BootstrapIters    int           // paired-CI resamples (default 2000)
	HeldoutFraction   float64       // held-out fold size (default 0.30)
	MinHeldout        int           // min held-out cases to validate (default 3)
	GibberishFloor    float64       // absolute gibberish ceiling floor (default 0.50)
	LatencyMultiplier float64       // p95 budget = incumbent p95 × this (default 3.0)
	ReindexTimeout    time.Duration // per-arm reindex budget (default 5m)
	PollInterval      time.Duration // reindex-status poll cadence (default 2s)
}

func (o Options) withDefaults() Options {
	if o.BootstrapIters <= 0 {
		o.BootstrapIters = 2000
	}
	if o.HeldoutFraction <= 0 {
		o.HeldoutFraction = 0.30
	}
	if o.MinHeldout <= 0 {
		o.MinHeldout = 3
	}
	if o.GibberishFloor <= 0 {
		o.GibberishFloor = 0.50
	}
	if o.LatencyMultiplier <= 0 {
		o.LatencyMultiplier = 3.0
	}
	if o.ReindexTimeout <= 0 {
		o.ReindexTimeout = 5 * time.Minute
	}
	if o.PollInterval <= 0 {
		o.PollInterval = 2 * time.Second
	}
	return o
}

// Orchestrator runs sweeps. It holds no transport — only seams.
type Orchestrator struct {
	deps Deps
	opts Options
}

// New constructs an Orchestrator, filling option + seam defaults.
func New(deps Deps, opts Options) *Orchestrator {
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.Rand == nil {
		deps.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &Orchestrator{deps: deps, opts: opts.withDefaults()}
}

// armEval bundles a stored run with the decision-time data the guards need.
type armEval struct {
	proto       *evalv1.SweepArm
	tuning      aisearch.TuningConfig
	recall      map[string]float64 // per positive case (0/1)
	isIncumbent bool
}

// Run executes the two-tier sweep for req.SuiteId and returns the ranked result
// + promotion verdict. It returns an error only on a precondition that makes the
// whole sweep impossible (unknown suite, unregistered provider, no positive
// cases). A single arm's failure (a reindex that times out, a search error)
// degrades that arm — never the whole sweep.
func (o *Orchestrator) Run(ctx context.Context, req *evalv1.SweepRequest) (*evalv1.SweepResult, error) {
	suite, err := o.deps.Suites.GetSuite(ctx, req.GetSuiteId())
	if err != nil {
		return nil, fmt.Errorf("get suite %q: %w", req.GetSuiteId(), err)
	}
	providerID := suite.GetProviderId()
	desc, err := o.deps.Providers.Get(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q: %w", providerID, err)
	}
	token, err := o.deps.Providers.Token(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("control token for %q: %w", providerID, err)
	}

	positive := positiveCaseIDs(suite)
	if len(positive) == 0 {
		return nil, fmt.Errorf("suite %q has no positive (recall) cases to optimize", req.GetSuiteId())
	}

	incumbent := tuningFromProto(desc.GetTuning()).WithDefaults()
	limit := req.GetLimit()

	var arms []*armEval

	// --- Incumbent (baseline): the provider's live config, no overrides. ------
	incRun, err := o.deps.Runner.Run(ctx, suite, armTag(tierIncumbent, incumbent), nil, token, limit)
	if err != nil {
		return nil, fmt.Errorf("run incumbent arm: %w", err)
	}
	incArm := o.evalArm(suite, incRun, incumbent, tierIncumbent, "incumbent (current tuning)", true)
	arms = append(arms, incArm)

	// --- Query-time tier: full-factorial via per-request overrides. -----------
	qtArms := queryTimeArms(incumbent)
	for _, t := range qtArms {
		ov := queryTimeOverrides(t)
		run, runErr := o.deps.Runner.Run(ctx, suite, armTag(tierQueryTime, t), &ov, token, limit)
		if runErr != nil {
			arms = append(arms, deadArm(armTag(tierQueryTime, t), tierQueryTime, t, "search failed: "+runErr.Error()))
			continue
		}
		arms = append(arms, o.evalArm(suite, run, t, tierQueryTime, "", false))
	}

	// --- Index-time tier: coordinate-ascent via config-push → reindex → run. --
	var itArms []aisearch.TuningConfig
	dropped := 0
	if !req.GetQueryTimeOnly() {
		itArms, dropped = indexTimeArms(incumbent)
		for _, t := range itArms {
			arm := o.runIndexArm(ctx, suite, desc, token, t, limit)
			arms = append(arms, arm)
		}
		// Restore the incumbent live config so the provider is never left serving a
		// sweep arm. The write-back (below) re-applies the winner if one is promoted.
		if len(itArms) > 0 {
			if _, _, restoreErr := o.applyTuning(ctx, desc, token, incumbent); restoreErr != nil {
				// Best-effort: a failed restore is reported, not fatal — the operator
				// can re-run config-write. Recorded on the result recommendation.
				incArm.proto.Note += fmt.Sprintf(" [warning: failed to restore incumbent config after index-time tier: %v]", restoreErr)
			}
		}
	}

	result := o.decide(suite, arms, incArm, len(qtArms), len(itArms), dropped, positive)
	result.SuiteId = req.GetSuiteId()
	result.ProviderId = providerID

	// --- Write-back: persist a cleared winner when apply=true. ----------------
	if req.GetApply() && result.GetWinnerTag() != "" && result.GetWinnerTag() != result.GetIncumbentTag() {
		winner := winnerTuning(arms, result.GetWinnerTag())
		resp, written, werr := o.applyTuning(ctx, desc, token, winner)
		if werr != nil {
			result.Recommendation += fmt.Sprintf("\nwrite-back FAILED: %v (provider unchanged)", werr)
		} else {
			result.Promoted = written
			if written {
				result.Recommendation += "\nwrite-back: persisted the winning tuning via config-write."
				// Refresh the registry cache so search-hub reflects the new tuning
				// immediately, not only after the provider's next boot re-registers.
				if cerr := o.refreshTuningCache(ctx, desc, token, resp.GetEffective()); cerr != nil {
					result.Recommendation += fmt.Sprintf("\nwrite-back: registry cache refresh deferred to next boot (%v).", cerr)
				}
			} else {
				result.Recommendation += "\nwrite-back: no-op (provider already on the winning tuning)."
			}
		}
	}
	return result, nil
}

// refreshTuningCache re-upserts the provider descriptor with the freshly written
// tuning so search-hub's registry cache mirrors the file's SSOT immediately. The
// control token is presented as the existing-registration secret (an idempotent
// update, not a new mint). Best-effort: a failure is surfaced on the
// recommendation, never fatal — the provider's next boot re-registration heals it.
func (o *Orchestrator) refreshTuningCache(ctx context.Context, desc *registryv1.ProviderDescriptor, token string, effective *registryv1.Tuning) error {
	if o.deps.Cache == nil || effective == nil {
		return nil
	}
	updated := proto.Clone(desc).(*registryv1.ProviderDescriptor)
	updated.Tuning = effective
	_, _, err := o.deps.Cache.Upsert(ctx, updated, token)
	return err
}

// runIndexArm realizes one index-time arm: push its config live, wait for the
// triggered reindex to finish, then run the suite on the (now live) config. Any
// step's failure degrades the arm to infeasible-with-reason; it never aborts the
// sweep.
func (o *Orchestrator) runIndexArm(ctx context.Context, suite *evalv1.EvalSuite, desc *registryv1.ProviderDescriptor, token string, t aisearch.TuningConfig, limit int32) *armEval {
	tag := armTag(tierIndexTime, t)
	wc, _, err := o.applyTuning(ctx, desc, token, t)
	if err != nil {
		return deadArm(tag, tierIndexTime, t, "config-write failed: "+err.Error())
	}
	if wc.GetReindexTriggered() {
		if err := o.awaitReindex(ctx, desc, token, wc.GetReindexJobId()); err != nil {
			return deadArm(tag, tierIndexTime, t, err.Error())
		}
	}
	run, err := o.deps.Runner.Run(ctx, suite, tag, nil, token, limit)
	if err != nil {
		return deadArm(tag, tierIndexTime, t, "search failed: "+err.Error())
	}
	return o.evalArm(suite, run, t, tierIndexTime, "", false)
}

// applyTuning pushes a tuning to the provider via config-write (not a dry run)
// and returns the response. Index-time changes trigger a reindex server-side;
// the caller polls when wc.ReindexTriggered.
func (o *Orchestrator) applyTuning(ctx context.Context, desc *registryv1.ProviderDescriptor, token string, t aisearch.TuningConfig) (resp *controlv1.WriteConfigResponse, written bool, err error) {
	resp, err = o.deps.Control.WriteConfig(ctx, desc, token, tuningToProto(t), false)
	if err != nil {
		return nil, false, err
	}
	return resp, resp.GetWritten(), nil
}

// awaitReindex polls ReindexStatus until the job is terminal or the budget
// elapses. A failed/cancelled/timed-out job is an error (the arm is dropped).
func (o *Orchestrator) awaitReindex(ctx context.Context, desc *registryv1.ProviderDescriptor, token, jobID string) error {
	deadline := o.deps.Clock.Now().Add(o.opts.ReindexTimeout)
	for {
		st, err := o.deps.Control.ReindexStatus(ctx, desc, token, jobID)
		if err != nil {
			return fmt.Errorf("reindex status %s: %w", jobID, err)
		}
		switch st.GetState() {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("reindex %s failed: %s", jobID, st.GetError())
		case "cancelled":
			return fmt.Errorf("reindex %s cancelled", jobID)
		}
		if !o.deps.Clock.Now().Before(deadline) {
			return fmt.Errorf("reindex %s did not finish within %s", jobID, o.opts.ReindexTimeout)
		}
		o.deps.Sleep(o.opts.PollInterval)
	}
}

// evalArm builds the armEval for a successful run: it stamps the arm's tuning
// onto the run's ConfigSnapshot (so the stored, self-describing run records the
// full config that produced it) and computes the per-case recall vector.
func (o *Orchestrator) evalArm(suite *evalv1.EvalSuite, run *evalv1.EvalRun, t aisearch.TuningConfig, tier, note string, incumbent bool) *armEval {
	run.Config = snapshotFor(run.GetConfig(), t)
	recall := recallByCase(suite, run)
	score := meanOver(recall, positiveCaseIDs(suite))
	return &armEval{
		proto: &evalv1.SweepArm{
			Tag:       run.GetTag(),
			Tier:      tier,
			Config:    run.GetConfig(),
			RunId:     run.GetRunId(),
			Score:     score,
			Aggregate: run.GetAggregate(),
			Note:      note,
		},
		tuning:      t,
		recall:      recall,
		isIncumbent: incumbent,
	}
}

// deadArm is an arm that never produced a usable run (a failed reindex or
// search). It is infeasible by construction and carries the reason.
func deadArm(tag, tier string, t aisearch.TuningConfig, reason string) *armEval {
	return &armEval{
		proto: &evalv1.SweepArm{
			Tag:      tag,
			Tier:     tier,
			Config:   snapshotFor(nil, t),
			Feasible: false,
			Note:     reason,
		},
		tuning: t,
		recall: map[string]float64{},
	}
}

// decide ranks the arms and renders the promotion verdict under the four guards.
func (o *Orchestrator) decide(suite *evalv1.EvalSuite, arms []*armEval, incumbent *armEval, qtCount, itCount, dropped int, positive []string) *evalv1.SweepResult {
	tuningFold, heldout := splitCases(positive, generatedCaseIDs(suite), o.opts.HeldoutFraction)
	// Constraints are anchored to the incumbent (the baseline arm): a winner may
	// not leak materially more junk nor blow a latency budget set relative to it.
	constraints := deriveConstraints(incumbent.runAggregateHolder(), o.opts.GibberishFloor, o.opts.LatencyMultiplier)

	// Mark feasibility on every arm that produced a run.
	for _, a := range arms {
		if a.proto.GetRunId() == "" { // dead arm: already infeasible
			continue
		}
		ok, reason := constraints.feasible(a.runAggregateHolder())
		a.proto.Feasible = ok
		if !ok && a.proto.GetNote() == "" {
			a.proto.Note = reason
		}
	}

	incScore := meanOver(incumbent.recall, positive)
	incTuningFold := meanOver(incumbent.recall, tuningFold)
	incHeldout := meanOver(incumbent.recall, heldout)

	// Candidates: feasible, non-incumbent arms that beat the incumbent on the
	// tuning fold. Ranked by tuning-fold recall desc, then complexity asc (the
	// tie-break), then tag for stability.
	var candidates []*armEval
	for _, a := range arms {
		if a.isIncumbent || !a.proto.GetFeasible() {
			continue
		}
		if meanOver(a.recall, tuningFold) > incTuningFold+1e-9 {
			candidates = append(candidates, a)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		ri := meanOver(candidates[i].recall, tuningFold)
		rj := meanOver(candidates[j].recall, tuningFold)
		if ri != rj {
			return ri > rj
		}
		ci, cj := complexity(candidates[i].tuning), complexity(candidates[j].tuning)
		if ci != cj {
			return ci < cj
		}
		return candidates[i].proto.GetTag() < candidates[j].proto.GetTag()
	})

	stats := &evalv1.SweepStats{
		IncumbentScore:           incScore,
		WinnerScore:              incScore,
		HeldoutIncumbentScore:    incHeldout,
		QueryTimeArms:            int32(qtCount),
		IndexTimeArms:            int32(itCount),
		DroppedIndexInteractions: int32(dropped),
	}

	var winner *armEval
	var verdict string
	// candidates is pre-sorted best-first (significance-then-tie-break), so only the
	// top arm can win: a later arm is by construction no better. Evaluate that one
	// arm against the incumbent's three guards.
	if len(candidates) > 0 {
		cand := candidates[0]
		wVec := vectorOver(cand.recall, tuningFold)
		iVec := vectorOver(incumbent.recall, tuningFold)
		mean, lo, hi := pairedMarginCI(wVec, iVec, o.opts.BootstrapIters, o.deps.Rand)
		held, heldReason := heldoutHolds(cand.recall, incumbent.recall, heldout, o.opts.MinHeldout)
		switch {
		case lo <= 0:
			verdict = fmt.Sprintf("best candidate %s: margin %+.3f, 95%% CI [%+.3f,%+.3f] overlaps 0 — within noise, not promoted.", shortTag(cand.proto.GetTag()), mean, lo, hi)
			stats.Margin, stats.CiLow, stats.CiHigh = mean, lo, hi
		case !held:
			verdict = fmt.Sprintf("best candidate %s: significant on tuning fold (CI [%+.3f,%+.3f]) but %s — not promoted.", shortTag(cand.proto.GetTag()), lo, hi, heldReason)
			stats.Margin, stats.CiLow, stats.CiHigh = mean, lo, hi
		default:
			// Cleared significance + held-out + (already) feasibility + tie-break order.
			winner = cand
			stats.Margin, stats.CiLow, stats.CiHigh = mean, lo, hi
			stats.WinnerScore = meanOver(cand.recall, positive)
			stats.HeldoutWinnerScore = meanOver(cand.recall, heldout)
			verdict = fmt.Sprintf("winner %s: recall %.3f vs incumbent %.3f, paired margin %+.3f (95%% CI [%+.3f,%+.3f] > 0), held-out %.3f ≥ %.3f — promotable.",
				shortTag(cand.proto.GetTag()), stats.WinnerScore, incScore, mean, lo, hi, stats.HeldoutWinnerScore, incHeldout)
		}
	}
	if winner == nil && verdict == "" {
		verdict = fmt.Sprintf("no feasible arm beat the incumbent (recall %.3f) on the tuning fold — incumbent retained.", incTuningFold)
	}
	if dropped > 0 {
		verdict += fmt.Sprintf(" [%d index-time factor interaction(s) not explored — coordinate-ascent.]", dropped)
	}

	res := &evalv1.SweepResult{
		Arms:           rankedArms(arms, tuningFold),
		IncumbentTag:   incumbent.proto.GetTag(),
		Recommendation: verdict,
		Stats:          stats,
	}
	if winner != nil {
		res.WinnerTag = winner.proto.GetTag()
	}
	return res
}

// rankedArms returns the arms' proto views best-first: feasible above
// infeasible, then by descending tuning-fold recall, then tag.
func rankedArms(arms []*armEval, tuningFold []string) []*evalv1.SweepArm {
	sorted := append([]*armEval(nil), arms...)
	sort.SliceStable(sorted, func(i, j int) bool {
		fi, fj := sorted[i].proto.GetFeasible(), sorted[j].proto.GetFeasible()
		if fi != fj {
			return fi // feasible first
		}
		ri := meanOver(sorted[i].recall, tuningFold)
		rj := meanOver(sorted[j].recall, tuningFold)
		if ri != rj {
			return ri > rj
		}
		return sorted[i].proto.GetTag() < sorted[j].proto.GetTag()
	})
	out := make([]*evalv1.SweepArm, len(sorted))
	for i, a := range sorted {
		out[i] = a.proto
	}
	return out
}

// runAggregateHolder returns a minimal EvalRun carrying just the aggregate so
// the constraint check (which reads only the aggregate) works off the arm proto.
func (a *armEval) runAggregateHolder() *evalv1.EvalRun {
	return &evalv1.EvalRun{Aggregate: a.proto.GetAggregate()}
}

// winnerTuning finds the tuning of the arm with the given tag.
func winnerTuning(arms []*armEval, tag string) aisearch.TuningConfig {
	for _, a := range arms {
		if a.proto.GetTag() == tag {
			return a.tuning
		}
	}
	return aisearch.TuningConfig{}
}

// snapshotFor overlays an arm's tuning onto a (possibly probe-derived) snapshot
// so the stored run is fully self-describing — every factor that produced the
// result is on the run, not just what a status probe happened to expose.
func snapshotFor(base *evalv1.ConfigSnapshot, t aisearch.TuningConfig) *evalv1.ConfigSnapshot {
	t = t.WithDefaults()
	var s *evalv1.ConfigSnapshot
	if base != nil {
		s = proto.Clone(base).(*evalv1.ConfigSnapshot)
	} else {
		s = &evalv1.ConfigSnapshot{}
	}
	s.Engine = t.Engine
	s.EmbedModel = t.EmbedModel
	s.EmbedTaskPrefix = t.EmbedTaskPrefix
	s.RerankEnabled = t.RerankEnabled
	s.RerankBlend = t.RerankBlend
	s.FloorRegime = floorRegime(t)
	return s
}

// tuningToProto / tuningFromProto convert between the aisearch TuningConfig and
// the registry wire Tuning. Inlined (rather than importing searchregister-go) to
// keep this core package's dependency surface minimal — it already depends on
// aisearch + the registry proto.
func tuningToProto(t aisearch.TuningConfig) *registryv1.Tuning {
	t = t.WithDefaults()
	return &registryv1.Tuning{
		Engine:           t.Engine,
		EmbedModel:       t.EmbedModel,
		EmbedTaskPrefix:  t.EmbedTaskPrefix,
		RerankEnabled:    t.RerankEnabled,
		RerankBlend:      t.RerankBlend,
		RerankShortlist:  int32(t.RerankShortlist),
		RerankPreference: t.RerankPreference,
		Floor:            &registryv1.FloorConfig{MaxGap: t.Floor.MaxGap, HardFloor: t.Floor.HardFloor},
	}
}

func tuningFromProto(t *registryv1.Tuning) aisearch.TuningConfig {
	if t == nil {
		return aisearch.TuningConfig{}
	}
	cfg := aisearch.TuningConfig{
		Engine:           t.GetEngine(),
		EmbedModel:       t.GetEmbedModel(),
		EmbedTaskPrefix:  t.GetEmbedTaskPrefix(),
		RerankEnabled:    t.GetRerankEnabled(),
		RerankBlend:      t.GetRerankBlend(),
		RerankShortlist:  int(t.GetRerankShortlist()),
		RerankPreference: t.GetRerankPreference(),
	}
	if f := t.GetFloor(); f != nil {
		cfg.Floor = aisearch.FloorTuning{MaxGap: f.GetMaxGap(), HardFloor: f.GetHardFloor()}
	}
	return cfg
}

// shortTag trims the canonical "sweep:<tier>:" prefix for legible verdicts.
func shortTag(tag string) string {
	const p = "sweep:"
	if len(tag) > len(p) && tag[:len(p)] == p {
		return tag[len(p):]
	}
	return tag
}
