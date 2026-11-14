package agentsession

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// DefaultReapOlderThan is the youngest session `vrooli agent reap` stops.
const DefaultReapOlderThan = time.Hour

// Reap decisions.
const (
	// DecisionStop is an eligible session.
	DecisionStop = "stop"
	// DecisionKeep is an orphaned session the policy protects; Reason says
	// why.
	DecisionKeep = "keep"
)

// Reap outcomes.
const (
	OutcomeWouldStop    = "would-stop"
	OutcomeStopped      = scenarioruntime.AgentSessionReapStopped
	OutcomeFailed       = scenarioruntime.AgentSessionReapFailed
	OutcomeUnsupported  = scenarioruntime.AgentSessionReapUnsupported
	OutcomeOwnerReturns = "skipped-owner-not-gone"
	OutcomeKept         = "kept"
)

// ReapPolicy bounds what a reap may stop.
type ReapPolicy struct {
	// OlderThan is the youngest session age eligible; zero means
	// DefaultReapOlderThan.
	OlderThan time.Duration
}

// Candidate is one orphaned session and the policy's decision on it.
type Candidate struct {
	Session  Session
	Decision string
	Reason   string
}

// ReapDecision is the reap policy for one session, whatever its state: it
// reports whether `vrooli agent reap --orphaned` would stop it and why. Only
// an orphaned session can be reapable: a terminal, unknown, idle-only, live
// or agentless session never is. An orphaned session is still kept when:
//
//   - its owner is a web-console session not known to be programmatic: an
//     operator's own UI terminal (origin ui or unspecified, or an origin never
//     observed and no Agent Manager evidence) is never auto-reaped;
//   - it is younger than the policy's age;
//   - its scope is not an agent session scope unit the platform can stop;
//   - its scope is frozen (the storm authority owns it until it is thawed).
func ReapDecision(session Session, policy ReapPolicy, now time.Time) (bool, string) {
	olderThan := policy.OlderThan
	if olderThan <= 0 {
		olderThan = DefaultReapOlderThan
	}
	switch {
	case session.State != StateOrphaned:
		return false, "state " + session.State + ": only orphaned sessions are reaped"
	case !session.Owner.Known():
		return false, "owner kind " + session.Owner.Kind + " is never reaped"
	case session.Owner.Kind == KindWebConsoleSession && !programmaticWebConsole(session.Owner):
		return false, "web-console session origin " + nonEmpty(session.Owner.Evidence.WebConsoleOrigin, "not observed") + ": an operator terminal is never auto-reaped"
	case session.ScopeUnit() == "":
		return false, "scope " + session.Scope + " is not an agent session scope unit"
	case session.StartedAt.IsZero():
		return false, "session start unknown, so its age cannot be compared with --older-than"
	case now.Sub(session.StartedAt) < olderThan:
		return false, fmt.Sprintf("younger than %s", olderThan)
	case session.Frozen == "yes":
		return false, "scope is frozen by the storm authority; thaw it first"
	default:
		return true, "owner confirmed gone: " + session.Liveness.Detail
	}
}

// Plan lists every orphaned session with ReapDecision's verdict on it.
func Plan(sessions []Session, policy ReapPolicy, now time.Time) []Candidate {
	out := make([]Candidate, 0)
	for _, session := range sessions {
		if session.State != StateOrphaned {
			continue
		}
		stop, reason := ReapDecision(session, policy, now)
		candidate := Candidate{Session: session, Decision: DecisionKeep, Reason: reason}
		if stop {
			candidate.Decision = DecisionStop
		}
		out = append(out, candidate)
	}
	return out
}

// programmaticWebConsole reports whether a web-console owner is known not to
// be an operator terminal: web-console reported origin programmatic while the
// session lived, or (origin never observed) the pane carries Agent Manager
// evidence, which only Agent Manager's launch command puts there. An
// observed ui, remote or unspecified origin always protects the session.
func programmaticWebConsole(owner Owner) bool {
	switch owner.Evidence.WebConsoleOrigin {
	case OriginProgrammatic:
		return true
	case "":
		return owner.Evidence.RunID != "" || owner.Evidence.AgentTag != ""
	default:
		return false
	}
}

// Stopper ends a session's process tree through the platform's containment
// primitive.
type Stopper interface {
	Stop(ctx context.Context, session Session) error
}

// AuditSink records a reap.
type AuditSink interface {
	RecordAgentSessionReap(ctx context.Context, reap scenarioruntime.AgentSessionReap) (scenarioruntime.AgentSessionReap, error)
}

// Executor carries out a plan.
type Executor struct {
	Stopper Stopper
	Audit   AuditSink
	// Recheck asks the owner's authority again immediately before a stop;
	// a session whose owner is no longer confirmed gone is skipped.
	Recheck func(ctx context.Context, session Session) Liveness
	// Initiator names who asked (cli, supervisor) for the audit row.
	Initiator string
	Now       func() time.Time
}

// Result is the outcome for one candidate.
type Result struct {
	Candidate Candidate
	Outcome   string
	Error     string
	Usage     ScopeUsage
	AuditID   string
}

// Execute stops each eligible candidate. With dryRun it reports what would be
// stopped and never calls the stopper, the recheck, or the audit.
func (e Executor) Execute(ctx context.Context, plan []Candidate, dryRun bool) []Result {
	results := make([]Result, 0, len(plan))
	for _, candidate := range plan {
		// The figures are the ones the reconciler measured for the list
		// (MeasureUsage): one implementation for both surfaces.
		result := Result{Candidate: candidate, Usage: candidate.Session.Usage}
		if candidate.Decision != DecisionStop {
			result.Outcome = OutcomeKept
			results = append(results, result)
			continue
		}
		if dryRun {
			result.Outcome = OutcomeWouldStop
			results = append(results, result)
			continue
		}
		results = append(results, e.stop(ctx, candidate, result))
	}
	return results
}

func (e Executor) stop(ctx context.Context, candidate Candidate, result Result) Result {
	session := candidate.Session
	if e.Recheck != nil {
		again := e.Recheck(ctx, session)
		if again.State != OwnerGone {
			result.Outcome = OutcomeOwnerReturns
			result.Error = "owner recheck: " + again.State + ": " + again.Detail
			return result
		}
		session.Liveness = again
	}
	if e.Stopper == nil {
		result.Outcome, result.Error = OutcomeUnsupported, "no stopper for this platform"
	} else if err := e.Stopper.Stop(ctx, session); err != nil {
		result.Outcome, result.Error = OutcomeFailed, err.Error()
		if errors.Is(err, platformgo.ErrUnsupported) {
			result.Outcome = OutcomeUnsupported
		}
	} else {
		result.Outcome = OutcomeStopped
	}
	if e.Audit != nil {
		evidence := session.Owner.LeaseOwner().EvidenceJSON
		now := time.Now().UTC()
		if e.Now != nil {
			now = e.Now().UTC()
		}
		recorded, err := e.Audit.RecordAgentSessionReap(ctx, scenarioruntime.AgentSessionReap{
			SessionID: session.SessionID, Scope: session.ScopeUnit(), OwnerKind: session.Owner.Kind, OwnerRef: session.Owner.Ref,
			OwnerEvidenceJSON: evidence, LivenessDetail: session.Liveness.Detail,
			MemoryBytes: orUnknown(result.Usage.MemoryBytes), SwapBytes: orUnknown(result.Usage.SwapBytes), Tasks: nonNegative(session.Tasks),
			Outcome: result.Outcome, Error: result.Error, Initiator: e.Initiator, CreatedAt: now,
		})
		if err != nil {
			result.Error = strings.TrimSpace(result.Error + "; audit not recorded: " + err.Error())
		} else {
			result.AuditID = recorded.ReapID
		}
	}
	return result
}

func nonNegative(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}
