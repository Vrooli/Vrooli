// This file raises an operator decision onto Vrooli Events when a session
// mutation-list proposal reaches `ready`, and applies the operator's answer when
// it returns from notification-hub. It is swarm-manager's seat on the shared
// operatordecision substrate (packages/api-core/operatordecision): the same rail
// agent-manager uses for run approvals. notification-hub turns the generic
// `swarm-manager.proposal.decision_requested.v1` event into an operator Ask
// (Apply/Reject/Request changes + a review link) with no per-scenario code in the hub, and the
// answer returns as notification_hub.ask.resolved.v1, routed here through the
// single mutation authority DecideMutationListProposal.
package agentsessions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

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
// logged and never blocks RecordProposal — the exact event snapshot stays pending
// for the bounded owner reconciler to retry. Only ready
// mutation-list proposals raise: a needs-revision proposal has no operator
// decision yet, and a no-change recommendation is reviewed on the keep-as-is path,
// not the Apply/Reject rail.
func (s *Service) emitProposalDecisionRequested(ctx context.Context, sessionID string, proposal Proposal) *ProposalDecisionPublication {
	if s == nil || proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationPending {
		return proposal.DecisionPublication
	}
	if !eligibleProposalDecision(proposal) {
		return proposal.DecisionPublication
	}
	publication := *proposal.DecisionPublication
	store, err := s.storeFor(ctx)
	if err != nil {
		slog.Warn("agentsessions: proposal decision receipt store unavailable", "session", sessionID, "proposal", proposal.ID, "err", err)
		return proposal.DecisionPublication
	}
	receipts, ok := store.(ProposalDecisionPageStore)
	if !ok {
		slog.Warn("agentsessions: proposal decision receipt store unsupported", "session", sessionID, "proposal", proposal.ID)
		return proposal.DecisionPublication
	}
	publication.Attempts++
	publication.LastAttemptAt = nowRFC3339()
	publication.LastError = ""
	updated, err := receipts.UpdateProposalDecisionPublication(sessionID, proposal.ID, publication.EventID, publication)
	if err != nil || !updated {
		if err != nil {
			slog.Warn("agentsessions: proposal decision attempt receipt failed", "session", sessionID, "proposal", proposal.ID, "err", err)
		}
		return proposal.DecisionPublication
	}
	if s.decisionPublisher == nil {
		err = fmt.Errorf("proposal decision event publisher is unavailable")
	} else if availability, ok := s.decisionPublisher.(interface{ Enabled() bool }); ok && !availability.Enabled() {
		err = fmt.Errorf("proposal decision event publisher is disabled")
	} else {
		err = s.decisionPublisher.PublishDomainEvent(ctx, publication.Event)
	}
	if err != nil {
		publication.LastError = truncateRunes(err.Error(), 500)
		if _, receiptErr := receipts.UpdateProposalDecisionPublication(sessionID, proposal.ID, publication.EventID, publication); receiptErr != nil {
			slog.Warn("agentsessions: proposal decision failure receipt failed", "session", sessionID, "proposal", proposal.ID, "err", receiptErr)
		}
		slog.Warn("agentsessions: proposal decision event publish failed",
			"session", sessionID, "proposal", proposal.ID, "err", err)
		return &publication
	}
	publication.State = ProposalDecisionPublicationPublished
	publication.PublishedAt = nowRFC3339()
	publication.LastError = ""
	recorded, receiptErr := receipts.UpdateProposalDecisionPublication(sessionID, proposal.ID, publication.EventID, publication)
	if receiptErr != nil || !recorded {
		slog.Warn("agentsessions: proposal decision published receipt failed", "session", sessionID, "proposal", proposal.ID, "err", receiptErr)
		publication.State = ProposalDecisionPublicationPending
		publication.PublishedAt = ""
		publication.LastError = "event publish succeeded but the durable success receipt was not confirmed"
	}
	return &publication
}

// proposalDecisionEvent builds the immutable decision_requested event snapshot
// for a ready proposal. Retries publish the stored snapshot instead of rebuilding
// facts from possibly changed producer code.
// The correlation carries session_id/proposal_id verbatim so the resolution
// webhook locates the exact proposal through the SAME SubjectRef scheme the
// AttemptDecider already parses (session_id/proposal_id, subject kind
// agent-session-proposal). The occurrence time is the proposal's CreatedAt (a
// stable, proposal-scoped instant), so the event id is deterministic and a
// redelivery — or a re-raise of the same ready proposal — maps to the one ask
// instead of a duplicate notification.
func (s *Service) newLegacyProposalDecisionPublication(ctx context.Context, sessionID string, proposal Proposal) (*ProposalDecisionPublication, error) {
	occurred := time.Now().UTC()
	event := s.proposalDecisionEvent(ctx, sessionID, proposal, occurred)
	if details, ok := event.Payload["technical_details"].(string); ok {
		contextURL, _ := event.Payload["context_url"].(string)
		event.Payload["technical_details"] = boundProposalDecisionDetails("New review request for current proposal terms. Earlier request state remains unresolved; this action does not withdraw an earlier request.\n\n"+details, contextURL)
	}
	return &ProposalDecisionPublication{
		Version: 1,
		State:   ProposalDecisionPublicationPending,
		EventID: operatordecision.EventID(event),
		Event:   event,
	}, nil
}

func (s *Service) newProposalDecisionPublication(ctx context.Context, sessionID string, proposal Proposal) (*ProposalDecisionPublication, error) {
	parsed, err := time.Parse(time.RFC3339Nano, proposal.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("proposal decision created_at is invalid: %w", err)
	}
	termsSHA256, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		return nil, fmt.Errorf("proposal decision terms digest: %w", err)
	}
	event := s.proposalDecisionEvent(ctx, sessionID, proposal, parsed.UTC())
	return &ProposalDecisionPublication{
		Version:             1,
		State:               ProposalDecisionPublicationPending,
		EventID:             operatordecision.EventID(event),
		Event:               event,
		DecisionTermsSHA256: termsSHA256,
	}, nil
}

func (s *Service) proposalDecisionEvent(ctx context.Context, sessionID string, proposal Proposal, occurred time.Time) eventbus.DomainEvent {
	target := proposal.Target.Name
	if strings.TrimSpace(target) == "" {
		target = proposal.Target.Ref
	}
	contextURL := proposalDecisionContextURL(ctx, s.proposalNoticeBaseResolver, sessionID)
	mutations, mutationErr := decodeProposalMutations(proposal.PayloadJSON)
	facts := operatordecision.Facts{
		Subject:          "Mutation proposal for " + target + " needs review",
		Question:         proposalDecisionQuestion(mutations, target),
		BigPicture:       proposalDecisionBigPicture(mutations, target, mutationErr),
		TechnicalDetails: proposalDecisionTechnicalDetails(proposal, target, contextURL, mutations, mutationErr),
		ContextURL:       contextURL,
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Apply the proposed mutations"},
			{Key: "reject", Label: "Reject & discard"},
			{Key: "request_changes", Label: "Request changes"},
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

	return operatordecision.BuildEvent(ProposalDecisionSource, ProposalDecisionKind, occurred, facts)
}

func proposalDecisionContextURL(ctx context.Context, resolve func(context.Context, string) (string, error), sessionID string) string {
	if resolve == nil {
		return ""
	}
	base, err := resolve(ctx, "swarm-manager")
	if err != nil || strings.TrimSpace(base) == "" {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/sessions/" + strings.TrimSpace(sessionID)
	parsed.RawPath = ""
	parsed.RawQuery = "tab=proposals"
	parsed.Fragment = ""
	return parsed.String()
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
func proposalDecisionQuestion(mutations []mutationSummary, target string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Apply %s to %s?", pluralizeMutations(len(mutations)), target)
	for i, mutation := range mutations {
		line := mutationSummaryLine(mutation)
		if line == "" {
			continue
		}
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

// mutationSummary mirrors only the operator-relevant proposal fields and keeps
// the item wire shape local to this event path.
type mutationSummary struct {
	ID        string                       `json:"id"`
	Op        string                       `json:"op"`
	Target    string                       `json:"target"`
	Rationale string                       `json:"rationale"`
	Item      *mutationItemDecisionDetails `json:"item"`
}

type mutationItemDecisionDetails struct {
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        *int     `json:"priority"`
	Effort          string   `json:"effort"`
	AcceptanceAllow []string `json:"acceptance_allow"`
	AcceptanceDeny  []string `json:"acceptance_deny"`
}

func decodeProposalMutations(payloadJSON string) ([]mutationSummary, error) {
	var payload struct {
		Mutations []mutationSummary `json:"mutations"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, err
	}
	return payload.Mutations, nil
}

// mutationSummaryLine is shared by the concise question and the detailed
// decision context so the operation, target and rationale cannot drift.
func mutationSummaryLine(mutation mutationSummary) string {
	parts := make([]string, 0, 4)
	if id := strings.TrimSpace(mutation.ID); id != "" {
		parts = append(parts, "["+id+"]")
	}
	if op := strings.TrimSpace(mutation.Op); op != "" {
		parts = append(parts, op)
	}
	if target := strings.TrimSpace(mutation.Target); target != "" {
		parts = append(parts, target)
	}
	if rationale := strings.TrimSpace(mutation.Rationale); rationale != "" {
		parts = append(parts, "— "+rationale)
	}
	return strings.Join(parts, " ")
}

func proposalDecisionBigPicture(mutations []mutationSummary, target string, parseErr error) string {
	if parseErr == nil && len(mutations) > 0 {
		addItems := 0
		allAddItems := true
		for _, mutation := range mutations {
			if strings.EqualFold(strings.TrimSpace(mutation.Op), "add_item") {
				addItems++
			} else {
				allAddItems = false
			}
		}
		if allAddItems {
			itemLabel := "backlog items"
			if addItems == 1 {
				itemLabel = "backlog item"
			}
			return truncateRunes(fmt.Sprintf("Approving records %d proposed %s under %s for later implementation; it does not implement that work.", addItems, itemLabel, target), operatordecision.MaxBigPictureCodePoints)
		}
		changeLabel := "changes"
		if len(mutations) == 1 {
			changeLabel = "change"
		}
		return truncateRunes(fmt.Sprintf("Approving applies %d listed %s to %s; see technical details for their specific effects.", len(mutations), changeLabel, target), operatordecision.MaxBigPictureCodePoints)
	}
	return truncateRunes(fmt.Sprintf("Review the proposed changes to %s. The exact effects are listed in the technical details.", target), operatordecision.MaxBigPictureCodePoints)
}

func proposalDecisionTechnicalDetails(proposal Proposal, target, contextURL string, mutations []mutationSummary, parseErr error) string {
	var lines []string
	if parseErr != nil {
		lines = []string{"Mutation details could not be parsed from the saved proposal."}
	} else {
		lines = make([]string, 0, len(mutations)*4)
		for _, mutation := range mutations {
			if line := mutationSummaryLine(mutation); line != "" {
				lines = append(lines, line)
			}
			if item := mutation.Item; item != nil {
				if kind := strings.TrimSpace(item.Kind); kind != "" {
					lines = append(lines, "Item kind: "+kind)
				}
				name, title := strings.TrimSpace(item.Name), strings.TrimSpace(item.Title)
				if name != "" || title != "" {
					label := name
					if title != "" {
						if label != "" {
							label += " — "
						}
						label += title
					}
					lines = append(lines, "Item: "+label)
				}
				if description := strings.TrimSpace(item.Description); description != "" {
					lines = append(lines, "Description: "+description)
				}
				if item.Priority != nil || strings.TrimSpace(item.Effort) != "" {
					meta := make([]string, 0, 2)
					if item.Priority != nil {
						meta = append(meta, fmt.Sprintf("priority %d", *item.Priority))
					}
					if effort := strings.TrimSpace(item.Effort); effort != "" {
						meta = append(meta, "effort "+effort)
					}
					lines = append(lines, "Work sizing: "+strings.Join(meta, "; "))
				}
				if len(item.AcceptanceAllow) > 0 {
					lines = append(lines, "Acceptance criteria:")
					for _, criterion := range item.AcceptanceAllow {
						if criterion = strings.TrimSpace(criterion); criterion != "" {
							lines = append(lines, "- "+criterion)
						}
					}
				}
				if len(item.AcceptanceDeny) > 0 {
					lines = append(lines, "Must not:")
					for _, criterion := range item.AcceptanceDeny {
						if criterion = strings.TrimSpace(criterion); criterion != "" {
							lines = append(lines, "- "+criterion)
						}
					}
				}
			}
		}
		if len(lines) == 0 {
			lines = []string{"The saved proposal contains no readable mutation details."}
		}
	}
	return boundProposalDecisionDetails(fmt.Sprintf("Proposal: %s\nTarget: %s (%s)\n%s", strings.TrimSpace(proposal.Summary), target, proposal.Target.Ref, strings.Join(lines, "\n")), contextURL)
}

func boundProposalDecisionDetails(details, contextURL string) string {
	limit := operatordecision.MaxTechnicalDetailsCodePoints
	if utf8.RuneCountInString(details) <= limit {
		return details
	}
	suffix := "\n[Additional proposal details are available from the review link.]"
	if strings.TrimSpace(contextURL) == "" {
		suffix = "\n[Additional proposal details were omitted to fit this bounded summary.]"
	}
	suffixLength := utf8.RuneCountInString(suffix)
	return truncateRunes(details, limit-suffixLength) + suffix
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
// proposal decision this scenario raised. Apply/Reject route through the attempt
// decider and the single mutation authority. Request changes uses the owner's
// persisted, idempotent session-revision continuation; it introduces no mutation
// path.
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
// from emission. Apply/Reject for a proposal that is no longer ready are no-ops.
// A duplicate request-changes resolution is instead reconciled against its
// persisted Ask ID and retried with the same Agent Manager continuation key.
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
	parts := strings.Split(subjectRef, "/")
	if len(parts) != 2 {
		slog.Info("agentsessions: proposal decision resolution ignored (invalid subject reference)")
		return true, nil
	}
	refSessionID, refProposalID := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if refSessionID == "" || refProposalID == "" || subjectRef != refSessionID+"/"+refProposalID {
		slog.Info("agentsessions: proposal decision resolution ignored (non-canonical subject reference)")
		return true, nil
	}
	if (sessionID != "" && sessionID != refSessionID) || (proposalID != "" && proposalID != refProposalID) {
		slog.Info("agentsessions: proposal decision resolution ignored (correlation subject mismatch)",
			"session", sessionID, "proposal", proposalID, "subject_session", refSessionID, "subject_proposal", refProposalID)
		return true, nil
	}
	// Derive absent IDs only from the validated canonical subject_ref, and use
	// that same subject for both publication verification and the effect.
	sessionID, proposalID = refSessionID, refProposalID

	// Only a concrete operator answer acts; an expired/defaulted resolution leaves
	// the proposal in the queue for manual handling.
	if res.State != operatordecision.StateAnswered {
		slog.Info("agentsessions: proposal decision resolution ignored (no operator answer)",
			"state", res.State, "session", sessionID, "proposal", proposalID)
		return true, nil
	}
	if e.service == nil || sessionID == "" || proposalID == "" {
		slog.Info("agentsessions: proposal decision resolution is a no-op; current publication cannot be verified",
			"session", sessionID, "proposal", proposalID)
		return true, nil
	}
	// A decision can act only on an exact frozen current publication. Legacy
	// unresolved rows and publications without a requester binding remain
	// unresolved; they are never treated as the current request by default.
	current, err := e.service.matchesCurrentDecisionPublication(ctx, sessionID, proposalID, res.RequesterRef)
	if err != nil {
		return true, err
	}
	if !current {
		slog.Info("agentsessions: stale proposal decision generation is a no-op", "session", sessionID, "proposal", proposalID)
		return true, nil
	}

	if res.Answer == "request_changes" {
		if e.service == nil || sessionID == "" || proposalID == "" {
			return true, fmt.Errorf("request-changes resolution requires the proposal owner")
		}
		rationale := strings.TrimSpace(res.Note)
		if rationale == "" {
			rationale = strings.TrimSpace(res.Reason)
		}
		if _, err := e.service.RequestMutationProposalRevisionForDecision(ctx, sessionID, proposalID, rationale, res.AskID); err != nil {
			return true, fmt.Errorf("route proposal revision request: %w", err)
		}
		slog.Info("agentsessions: proposal revision requested from notification-hub",
			"session", sessionID, "proposal", proposalID, "ask", res.AskID)
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

// proposalDecisionFromAnswer maps terminal answers onto the attempt decision
// vocabulary: approve → accept (all candidate mutation ids accepted), reject →
// drop (no mutation ids accepted, which DecideMutationListProposal records as a
// rejection). request_changes is handled separately through the owning session's
// existing revision continuation and never reaches the apply/reject authority.
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
