// This file raises an operator decision onto Vrooli Events when a session
// mutation-list proposal reaches `ready`, and applies the operator's answer when
// it returns from notification-hub. It is swarm-manager's seat on the shared
// operatordecision substrate (packages/api-core/operatordecision): the same rail
// agent-manager uses for run approvals. notification-hub turns the generic
// `swarm-manager.proposal.decision_requested.v1` event into an operator Ask
// (Apply/Reject + a review link) with no per-scenario code in the hub, and the
// answer returns as notification_hub.ask.resolved.v1, routed here through the
// single mutation authority DecideMutationListProposal.
package agentsessions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/attempt"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// ProposalDecisionSource is this scenario's Vrooli Events source; it matches the
// source token embedded in the event type, so BuildEvent composes one spelling.
const ProposalDecisionSource = "swarm-manager"

// ProposalDecisionKind is the decision kind. Together with the source it composes
// ProposalDecisionEventType via operatordecision.DecisionRequestedEventType.
const ProposalDecisionKind = "proposal"

// ProposalDecisionEventType is the decision event a ready mutation-list proposal
// raises for operator review: "swarm-manager.proposal.decision_requested.v1". The
// `.decision_requested.v1` suffix is the generic contract notification-hub already
// subscribes to (**.decision_requested.v1), so no hub change is needed.
var ProposalDecisionEventType = operatordecision.DecisionRequestedEventType(ProposalDecisionSource, ProposalDecisionKind)

// DecisionEventPublisher publishes the decision event onto Vrooli Events. The
// api-core eventbus.Client satisfies it in production. It is an alias for the
// shared operatordecision.Publisher so the raise path and the substrate speak one
// interface. Nil leaves a ready proposal in the review queue without an
// out-of-band notification (no regression: the operator can still decide from the
// inbox), exactly like agent-manager's absent run-approval publisher.
type DecisionEventPublisher = operatordecision.Publisher

// SetDecisionEventPublisher wires the Vrooli Events publisher used to raise a
// ready proposal's decision event. Without one, proposals still reach `ready`;
// they just do not notify the operator out-of-band.
func (s *Service) SetDecisionEventPublisher(p DecisionEventPublisher) {
	if s == nil {
		return
	}
	s.decisionPublisher = p
}

// emitProposalDecisionRequested raises the operator decision for a mutation-list
// proposal that has reached `ready`. Delivery is best-effort: a publish failure is
// logged and never blocks RecordProposal — the durable proposal already stands and
// a later raise (or the inbox) can still drive the decision. Only ready
// mutation-list proposals raise: a needs-revision proposal has no operator
// decision yet, and a no-change recommendation is reviewed on the keep-as-is path,
// not the Apply/Reject rail.
func (s *Service) emitProposalDecisionRequested(ctx context.Context, sessionID string, proposal Proposal) {
	if s == nil || s.decisionPublisher == nil {
		return
	}
	if proposal.Kind != ProposalMutationList || proposal.Status != ProposalStatusReady || proposal.NeedsRevision {
		return
	}
	if proposal.Target == nil {
		return
	}
	event := s.proposalDecisionEvent(sessionID, proposal)
	if err := s.decisionPublisher.PublishDomainEvent(ctx, event); err != nil {
		slog.Warn("agentsessions: proposal decision event publish failed",
			"session", sessionID, "proposal", proposal.ID, "err", err)
	}
}

// proposalDecisionEvent builds the decision_requested event for a ready proposal.
// The correlation carries session_id/proposal_id verbatim so the resolution
// webhook locates the exact proposal through the SAME SubjectRef scheme the
// AttemptDecider already parses (session_id/proposal_id, subject kind
// agent-session-proposal). The occurrence time is the proposal's CreatedAt (a
// stable, proposal-scoped instant), so the event id is deterministic and a
// redelivery — or a re-raise of the same ready proposal — maps to the one ask
// instead of a duplicate notification.
func (s *Service) proposalDecisionEvent(sessionID string, proposal Proposal) eventbus.DomainEvent {
	target := proposal.Target.Name
	if strings.TrimSpace(target) == "" {
		target = proposal.Target.Ref
	}
	facts := operatordecision.Facts{
		Subject:  "Mutation proposal for " + target + " needs review",
		Question: proposalDecisionQuestion(proposal, target),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Apply the proposed mutations"},
			{Key: "reject", Label: "Reject & discard"},
		},
		Recommended: "approve",
		// reversible=true: a rejected or revised proposal can be re-raised, and the
		// graph change is itself a reviewable mutation. No DefaultAnswer is set, so
		// nothing is applied without an explicit operator answer.
		Reversible:  true,
		Urgency:     "normal",
		Note:        proposalDecisionNote(proposal),
		Correlation: proposalDecisionCorrelation(sessionID, proposal),
	}

	occurred := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339Nano, proposal.CreatedAt); err == nil {
		occurred = parsed.UTC()
	}
	return operatordecision.BuildEvent(ProposalDecisionSource, ProposalDecisionKind, occurred, facts)
}

// proposalDecisionCorrelation is the verbatim map notification-hub returns on
// resolution. subject_kind and subject_ref reproduce the AttemptDecider contract
// (subject kind agent-session-proposal, ref session_id/proposal_id) so the
// resolution effect can rebuild the exact attempt.DecisionRequest without a
// separate lookup table; session_id/proposal_id are also carried split for a
// human-readable inbox.
func proposalDecisionCorrelation(sessionID string, proposal Proposal) map[string]string {
	return map[string]string{
		"subject_kind":  AttemptSubjectProposal,
		"subject_ref":   strings.TrimSpace(sessionID) + "/" + strings.TrimSpace(proposal.ID),
		"session_id":    strings.TrimSpace(sessionID),
		"proposal_id":   strings.TrimSpace(proposal.ID),
		"proposal_kind": string(proposal.Kind),
	}
}

// proposalDecisionQuestion renders the operator prompt: the target, a mutation
// count, and a short per-mutation line (op + target + rationale) for each
// candidate mutation, bounded so the prompt stays within the Facts question
// limit. A no-mutation (keep-as-is) envelope never reaches here — it raises no
// decision — so an empty mutation list is only a defensive case.
func proposalDecisionQuestion(proposal Proposal, target string) string {
	mutations := decodeMutationDescriptions(proposal.PayloadJSON)
	var b strings.Builder
	fmt.Fprintf(&b, "Apply %s to %s?", pluralizeMutations(len(mutations)), target)
	for i, line := range mutations {
		// Keep the prompt inside operatordecision.MaxQuestion; truncate the list
		// rather than overflow. The full payload is on the review surface.
		if b.Len()+len(line)+2 > operatordecision.MaxQuestion-20 {
			fmt.Fprintf(&b, "\n…and %d more.", len(mutations)-i)
			break
		}
		b.WriteString("\n- ")
		b.WriteString(line)
	}
	return b.String()
}

// mutationSummary is the minimal shape of a mutation payload the operator prompt
// needs: enough to say what each change does. It deliberately mirrors
// proposals.Mutation's wire tags without importing that package into the event
// path.
type mutationSummary struct {
	ID        string `json:"id"`
	Op        string `json:"op"`
	Target    string `json:"target"`
	Rationale string `json:"rationale"`
}

// decodeMutationDescriptions returns one human line per candidate mutation,
// carrying the mutation id so the operator can tie a line to the applied/rejected
// id set the resolution records.
func decodeMutationDescriptions(payloadJSON string) []string {
	var payload struct {
		Mutations []mutationSummary `json:"mutations"`
	}
	if json.Unmarshal([]byte(payloadJSON), &payload) != nil {
		return nil
	}
	lines := make([]string, 0, len(payload.Mutations))
	for _, m := range payload.Mutations {
		parts := make([]string, 0, 4)
		if id := strings.TrimSpace(m.ID); id != "" {
			parts = append(parts, "["+id+"]")
		}
		if op := strings.TrimSpace(m.Op); op != "" {
			parts = append(parts, op)
		}
		if t := strings.TrimSpace(m.Target); t != "" {
			parts = append(parts, t)
		}
		if r := strings.TrimSpace(m.Rationale); r != "" {
			parts = append(parts, "— "+r)
		}
		if len(parts) == 0 {
			continue
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return lines
}

// proposalDecisionNote carries the proposal's rationale/summary as the free-text
// operator note, so the validation status and intent travel with the ask.
func proposalDecisionNote(proposal Proposal) string {
	reason := proposalResolutionReason(proposal, "")
	note := "validation: ready"
	if strings.TrimSpace(reason) != "" {
		note = note + "; " + reason
	}
	return note
}

func pluralizeMutations(n int) string {
	if n == 1 {
		return "1 mutation"
	}
	return fmt.Sprintf("%d mutations", n)
}

// ProposalDecisionEffect applies a notification-hub ask resolution that answers a
// proposal decision this scenario raised. It is the consumer side of the rail: it
// translates the operator answer into an attempt.DecisionRequest and routes it
// through the attempt decider, which calls the single mutation authority
// DecideMutationListProposal. It introduces NO new mutation path.
//
// It holds the attempt.Decider (so it uses the registered AttemptDecider, not a
// private mutation call) and the Service (for the resolution-time readiness
// re-check). Satisfies operatordecision.DomainEffect.
type ProposalDecisionEffect struct {
	decider attempt.Decider
	service *Service
}

// NewProposalDecisionEffect builds the DomainEffect for a scenario's decision
// webhook. decider is the SAME attempt.Router (or AttemptDecider) registered for
// AttemptSubjectProposal, so the resolution reuses the single authority.
func NewProposalDecisionEffect(decider attempt.Decider, service *Service) *ProposalDecisionEffect {
	return &ProposalDecisionEffect{decider: decider, service: service}
}

// Handles reports whether this effect owns the resolution's source event type.
func (e *ProposalDecisionEffect) Handles(sourceEventType string) bool {
	return strings.TrimSpace(sourceEventType) == ProposalDecisionEventType
}

// Apply translates the operator answer into an attempt.DecisionRequest and calls
// the decider. It reports handled=true for every resolution it owns, including the
// benign no-op cases (a stale proposal, an unknown proposal, a non-answered
// resolution) so vrooli-events does not retry a poison message.
//
// Resolution-time readiness re-check: the proposal is re-read NOW, not trusted
// from emission. A proposal that is no longer ready (already applied/rejected,
// revised, superseded, or gone) records a no-op — nothing is mutated. Only a
// still-ready proposal is dispatched to the authority, which itself re-validates
// fresh graph state immediately before changing anything (defense in depth).
func (e *ProposalDecisionEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if e == nil || res.SourceEventType != ProposalDecisionEventType {
		return false, nil
	}
	sessionID := strings.TrimSpace(res.Correlation["session_id"])
	proposalID := strings.TrimSpace(res.Correlation["proposal_id"])
	subjectRef := strings.TrimSpace(res.Correlation["subject_ref"])
	if subjectRef == "" && sessionID != "" && proposalID != "" {
		subjectRef = sessionID + "/" + proposalID
	}
	if subjectRef == "" {
		return true, fmt.Errorf("proposal decision resolution lacks a session_id/proposal_id correlation")
	}
	if sessionID == "" || proposalID == "" {
		if parts := strings.SplitN(subjectRef, "/", 2); len(parts) == 2 {
			sessionID, proposalID = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}

	// Only a concrete operator answer acts; an expired/defaulted resolution leaves
	// the proposal in the queue for manual handling.
	if res.State != operatordecision.StateAnswered {
		slog.Info("agentsessions: proposal decision resolution ignored (no operator answer)",
			"state", res.State, "session", sessionID, "proposal", proposalID)
		return true, nil
	}

	// Resolution-time readiness re-check. A proposal that is not ready NOW is a
	// no-op: do not route it to the authority. This covers a duplicate delivery
	// (already applied/rejected), a superseded/revised proposal, and a proposal
	// that disappeared.
	if e.service != nil && sessionID != "" && proposalID != "" {
		ready, err := e.service.proposalIsReady(ctx, sessionID, proposalID)
		if err != nil {
			return true, err
		}
		if !ready {
			slog.Info("agentsessions: proposal decision resolution is a no-op; proposal no longer ready",
				"session", sessionID, "proposal", proposalID, "answer", res.Answer)
			return true, nil
		}
	}

	decision, explicitAccepted, err := proposalDecisionFromAnswer(res)
	if err != nil {
		return true, err
	}
	// On approve, an empty accepted set would be read by the authority as a
	// rejection — so "apply the whole proposal" must resolve to the full candidate
	// mutation id set. An explicit subset carried in the correlation is honored
	// as-is.
	accepted := explicitAccepted
	if decision == "accept" && len(accepted) == 0 && e.service != nil && sessionID != "" && proposalID != "" {
		candidates, candErr := e.service.proposalCandidateMutationIDs(ctx, sessionID, proposalID)
		if candErr != nil {
			return true, fmt.Errorf("resolve proposal mutation ids: %w", candErr)
		}
		accepted = candidates
	}
	actor := strings.TrimSpace(res.AnsweredBy)
	if actor == "" {
		actor = "operator"
	}
	rationale := strings.TrimSpace(res.Note)
	if rationale == "" {
		rationale = strings.TrimSpace(res.Reason)
	}
	if _, err := e.decider.DecideAttempt(ctx, attempt.DecisionRequest{
		SubjectKind:         AttemptSubjectProposal,
		SubjectRef:          subjectRef,
		RoundNum:            1,
		Decision:            decision,
		Actor:               actor,
		Rationale:           rationale,
		AcceptedProposalIDs: accepted,
	}); err != nil {
		return true, fmt.Errorf("apply proposal decision (%s): %w", res.Answer, err)
	}
	slog.Info("agentsessions: proposal decision applied from notification-hub",
		"session", sessionID, "proposal", proposalID, "answer", res.Answer, "actor", actor)
	return true, nil
}

// proposalDecisionFromAnswer maps the operator's answer onto the attempt decision
// vocabulary: approve → accept (all candidate mutation ids accepted), reject →
// drop (no mutation ids accepted, which DecideMutationListProposal records as a
// rejection).
func proposalDecisionFromAnswer(res operatordecision.DecisionResolution) (decision string, accepted []string, err error) {
	switch res.Answer {
	case "approve":
		return "accept", answeredAcceptedMutationIDs(res), nil
	case "reject":
		return "drop", nil, nil
	default:
		return "", nil, fmt.Errorf("proposal decision resolution has an unknown answer %q", res.Answer)
	}
}

// answeredAcceptedMutationIDs returns the accepted-mutation subset the operator
// chose, if the resolution carried one in its correlation (comma-separated
// accepted_mutation_ids). An empty result means "accept the whole proposal",
// which the authority resolves to every candidate mutation.
func answeredAcceptedMutationIDs(res operatordecision.DecisionResolution) []string {
	raw := strings.TrimSpace(res.Correlation["accepted_mutation_ids"])
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		if id := strings.TrimSpace(p); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
