package gate

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/targetmodel"
)

// Service is the application-layer surface the gate handler depends on. It owns
// the RunGate fan-out: select one eligible node per target OS, dispatch the
// validation run to each (delegating to the Runner seam — dispatch + runs), and
// record a durable Gate with the per-OS ledger; plus the read/await paths that
// recompute the live cross-OS verdict from the per-OS runs.
type Service interface {
	// Run fans a scenario's validation out across the target OSes, dispatching a
	// validation run per eligible node and recording a durable gate. On a dry-run
	// it selects + classifies each OS and short-circuits before creating any
	// gate/run.
	Run(ctx context.Context, in RunInput) (RunDecision, error)

	// GetGate returns a gate and its per-OS ledger with the verdict recomputed
	// live from the current state of each OS's validation run.
	GetGate(ctx context.Context, id string) (Gate, []OSResult, error)

	// WaitGate blocks once until every target OS's run is terminal (or the
	// timeout elapses), then returns the final verdict + ledger. timedOut is true
	// when the deadline elapsed with targets still running.
	WaitGate(ctx context.Context, id string, timeout time.Duration) (g Gate, results []OSResult, timedOut bool, err error)

	// ListGates returns gates newest-first, narrowed by filter.
	ListGates(ctx context.Context, filter ListFilter) ([]Gate, error)
}

type service struct {
	repo     Repository
	nodes    NodeLister
	presence Presence
	runner   Runner
	clock    schedule.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, nodes NodeLister, presence Presence, runner Runner, clk schedule.Clock) Service {
	return &service{repo: repo, nodes: nodes, presence: presence, runner: runner, clock: clk}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Run(ctx context.Context, in RunInput) (RunDecision, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return RunDecision{}, ErrInvalidGate{Field: "scenario", Reason: "required"}
	}
	revision := strings.TrimSpace(in.TargetRevision)
	if revision == "" {
		return RunDecision{}, ErrInvalidGate{Field: "target_revision", Reason: "required"}
	}
	oses := normaliseOSes(in.TargetOSes)
	if len(oses) == 0 {
		return RunDecision{}, ErrInvalidGate{Field: "target_oses", Reason: "at least one target OS required"}
	}

	verb := strings.TrimSpace(in.Verb)
	if verb == "" {
		verb = DefaultVerb
	}

	all, err := s.nodes.ListNodes(ctx)
	if err != nil {
		return RunDecision{}, err
	}
	byOS := s.eligibleByOS(all)

	results := make([]OSResult, 0, len(oses))
	for _, os := range oses {
		results = append(results, s.selectAndDispatch(ctx, os, byOS[os], in, scenario, verb, revision))
	}

	verdict := aggregateVerdict(results)

	// Dry-run: report what WOULD happen without persisting a gate or dispatching.
	if in.DryRun {
		return RunDecision{DryRun: true, Verdict: verdict, Results: results}, nil
	}

	passed, failed, pending := counts(results)
	g := Gate{
		Scenario:       scenario,
		TargetRevision: revision,
		Verb:           verb,
		Args:           in.Args,
		Verdict:        verdict,
		TotalTargets:   len(results),
		Passed:         passed,
		Failed:         failed,
		Pending:        pending,
	}
	persisted, err := s.repo.Create(ctx, g, results)
	if err != nil {
		return RunDecision{}, err
	}
	return RunDecision{GateID: persisted.ID, Verdict: verdict, Results: results}, nil
}

// selectAndDispatch picks the eligible node for one OS and (unless a dry-run)
// dispatches its validation run.
func (s *service) selectAndDispatch(ctx context.Context, os string, candidate *targetmodel.Selection, in RunInput, scenario, verb, revision string) OSResult {
	if candidate == nil || !candidate.Found || !candidate.Available {
		detail := "no eligible online node runs this OS"
		if candidate != nil && strings.TrimSpace(candidate.Reason) != "" {
			detail = candidate.Reason
		}
		return OSResult{OS: os, Disposition: OSDispositionNoNode, Detail: detail}
	}

	if in.DryRun {
		return OSResult{OS: os, NodeID: candidate.Target.ID, Disposition: OSDispositionPending, Detail: "would dispatch validation run"}
	}

	runID, err := s.runner.Dispatch(ctx, DispatchRequest{
		Actor:          in.Actor,
		NodeID:         candidate.Target.ID,
		Scenario:       scenario,
		Verb:           verb,
		Args:           in.Args,
		TimeoutSeconds: in.TimeoutSeconds,
	})
	if err != nil {
		return OSResult{OS: os, NodeID: candidate.Target.ID, Disposition: OSDispositionDispatchFailed, Detail: err.Error()}
	}
	return OSResult{OS: os, NodeID: candidate.Target.ID, RunID: runID, Disposition: OSDispositionPending, Detail: "validation run dispatched"}
}

// eligibleByOS projects the bridge observations into the shared target model
// and delegates deterministic selection to the same selector used by the ramp.
// Presence remains a live bridge concern, so it overlays the durable inventory
// before selection. The lowest target id wins when several targets share an OS.
func (s *service) eligibleByOS(all []NodeRef) map[string]*targetmodel.Selection {
	targets := make([]targetmodel.Target, 0, len(all))
	oses := make([]string, 0, len(all))
	for _, node := range all {
		target := node.Target
		if strings.TrimSpace(target.ID) == "" {
			target = targetmodel.Target{
				ID: node.ID, Platform: "desktop", OS: node.OS, Architecture: node.Arch,
				DeviceKind: "desktop", Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: node.ID, Available: true},
				Available: true,
			}
		}
		if strings.TrimSpace(target.OS) == "" {
			target.OS = node.OS
		}
		if strings.TrimSpace(target.Architecture) == "" {
			target.Architecture = node.Arch
		}
		if target.Transport.Kind == "" {
			target.Transport.Kind = targetmodel.TransportBridge
		}
		if target.Transport.ID == "" {
			target.Transport.ID = target.ID
		}
		if node.Revoked {
			target.Revoked = true
			target.Available = false
			target.Reason = targetmodel.ReasonBridgeRevoked
			target.MissingCapability = "bridge node authorization"
			target.NextAction = "revoke the stale target and register an authorized node"
		} else if target.BridgeTrust != nil && !target.BridgeTrust.DispatchAuthorized {
			target.Available = false
			target.Reason = targetmodel.ReasonBridgeNoDispatchScope
			target.MissingCapability = "scenario-test dispatch scope"
			target.NextAction = "grant the scenario-test dispatch scope to the registered node"
		} else if !s.presence.IsOnline(target.ID) || !s.presence.Dispatchable(target.ID) {
			target.Available = false
			target.Reason = targetmodel.ReasonBridgeOffline
			target.MissingCapability = "bridge dispatch reachability"
			target.NextAction = "restore the node channel and protocol compatibility"
		}
		targets = append(targets, target)
		oses = append(oses, target.OS)
	}
	selected := targetmodel.SelectByOS(targetmodel.Inventory{Targets: targets}, oses, targetmodel.SelectionRequest{TransportKinds: []targetmodel.TransportKind{targetmodel.TransportBridge}})
	out := make(map[string]*targetmodel.Selection, len(selected))
	for os, selection := range selected {
		picked := selection
		out[os] = &picked
	}
	return out
}

func (s *service) GetGate(ctx context.Context, id string) (Gate, []OSResult, error) {
	g, err := s.repo.Get(ctx, id)
	if err != nil {
		return Gate{}, nil, err
	}
	stored, err := s.repo.Results(ctx, id)
	if err != nil {
		return Gate{}, nil, err
	}
	refreshed := s.refresh(ctx, stored)
	return applyVerdict(g, refreshed), refreshed, nil
}

func (s *service) WaitGate(ctx context.Context, id string, timeout time.Duration) (Gate, []OSResult, bool, error) {
	if timeout <= 0 {
		timeout = DefaultWaitTimeout
	}
	g, err := s.repo.Get(ctx, id)
	if err != nil {
		return Gate{}, nil, false, err
	}
	stored, err := s.repo.Results(ctx, id)
	if err != nil {
		return Gate{}, nil, false, err
	}

	deadline := s.clock.Now().Add(timeout)
	out := make([]OSResult, len(stored))
	copy(out, stored)
	for i, r := range out {
		if r.Disposition != OSDispositionPending || r.RunID == "" {
			continue
		}
		remaining := deadline.Sub(s.clock.Now())
		if remaining <= 0 {
			break // out of budget; remaining pending targets stay pending
		}
		v, err := s.runner.Wait(ctx, r.RunID, remaining)
		if err != nil {
			out[i].Disposition = OSDispositionDispatchFailed
			out[i].Detail = err.Error()
			continue
		}
		out[i] = applyVerdictToResult(out[i], v)
	}

	timedOut := hasPending(out)
	return applyVerdict(g, out), out, timedOut, nil
}

func (s *service) ListGates(ctx context.Context, filter ListFilter) ([]Gate, error) {
	gates, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	for i, g := range gates {
		stored, err := s.repo.Results(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		gates[i] = applyVerdict(g, s.refresh(ctx, stored))
	}
	return gates, nil
}

// refresh recomputes the still-PENDING results against the live run state
// (non-blocking). Terminal results are returned unchanged.
func (s *service) refresh(ctx context.Context, stored []OSResult) []OSResult {
	out := make([]OSResult, len(stored))
	copy(out, stored)
	for i, r := range out {
		if r.Disposition != OSDispositionPending || r.RunID == "" {
			continue
		}
		v, err := s.runner.Verdict(ctx, r.RunID)
		if err != nil {
			// A transient read error leaves the target PENDING (re-read later).
			continue
		}
		out[i] = applyVerdictToResult(out[i], v)
	}
	return out
}

// applyVerdictToResult folds a refreshed run verdict into a per-OS result.
func applyVerdictToResult(r OSResult, v RunVerdict) OSResult {
	r.Disposition = v.disposition()
	r.ExitCode = v.ExitCode
	if d := strings.TrimSpace(v.Detail); d != "" {
		r.Detail = d
	}
	return r
}

// applyVerdict recomputes a gate's verdict + counts from a fresh result set.
func applyVerdict(g Gate, results []OSResult) Gate {
	g.Verdict = aggregateVerdict(results)
	g.TotalTargets = len(results)
	g.Passed, g.Failed, g.Pending = counts(results)
	return g
}

// hasPending reports whether any result is still awaiting a terminal run.
func hasPending(results []OSResult) bool {
	for _, r := range results {
		if r.Disposition == OSDispositionPending {
			return true
		}
	}
	return false
}
