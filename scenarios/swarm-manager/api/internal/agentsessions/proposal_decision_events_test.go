package agentsessions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"swarm-manager/internal/attempt"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// capturingPublisher records every decision event the service raises, so a test
// can assert exactly-one-decision and inspect the facts/correlation.
type capturingPublisher struct {
	events    []eventbus.DomainEvent
	err       error
	onPublish func(eventbus.DomainEvent) error
}

type disabledDecisionPublisher struct{}

func (disabledDecisionPublisher) Enabled() bool { return false }
func (disabledDecisionPublisher) PublishDomainEvent(context.Context, eventbus.DomainEvent) error {
	return nil
}

func (p *capturingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	if p.onPublish != nil {
		return p.onPublish(event)
	}
	return p.err
}

func newProposalDecisionService(t *testing.T) (*Service, *capturingPublisher, *fakeMutationProposalProcessor) {
	t.Helper()
	processor := &fakeMutationProposalProcessor{application: MutationProposalApplication{Outcomes: []MutationOutcome{{MutationID: "m1", Applied: true}}}}
	service := newTestService(t, &fakeSessionSpawner{})
	service.SetMutationProposalProcessor(processor)
	publisher := &capturingPublisher{}
	service.SetDecisionEventPublisher(publisher)
	return service, publisher, processor
}

const readyMutationPayload = `{"form":"mutation_list","rationale":"close the gate gap","mutations":[{"id":"m1","op":"add_item","target":"goal/quality-gates","rationale":"add missing coverage work"}]}`

func recordReadyMutationProposal(t *testing.T, service *Service, sessionID string) Proposal {
	t.Helper()
	proposal, err := service.RecordProposal(context.Background(), sessionID, Proposal{
		Kind:        ProposalMutationList,
		Status:      ProposalStatusReady,
		Summary:     "Mutation proposal for Quality Gates",
		PayloadJSON: readyMutationPayload,
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	})
	if err != nil {
		t.Fatalf("RecordProposal: %v", err)
	}
	return proposal
}

// recordLegacyReadyMutationProposal models a durable row written before the
// immutable event-snapshot rollout. It deliberately writes the legacy marker
// directly and does not manufacture a historical event identity.
func recordLegacyReadyMutationProposal(t *testing.T, service *Service, sessionID string) Proposal {
	t.Helper()
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatalf("storeFor: %v", err)
	}
	proposal := Proposal{
		ID:          "prop_legacy_review",
		Kind:        ProposalMutationList,
		Status:      ProposalStatusReady,
		Summary:     "Mutation proposal for Quality Gates",
		PayloadJSON: readyMutationPayload,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
		DecisionPublication: &ProposalDecisionPublication{
			Version:   1,
			State:     ProposalDecisionPublicationLegacyUnresolved,
			LastError: "legacy ready proposal has no frozen event snapshot; publication cannot be inferred",
		},
	}
	if err := store.SaveProposal(sessionID, proposal); err != nil {
		t.Fatalf("SaveProposal(legacy): %v", err)
	}
	return proposal
}

func TestLegacyProposalDecisionReissuePreservesHistoryAndFencesOldAnswers(t *testing.T) {
	service, publisher, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
	proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
	termsDigest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		t.Fatalf("proposalDecisionTermsSHA256: %v", err)
	}

	updated, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, termsDigest)
	if err != nil {
		t.Fatalf("ReissueLegacyProposalDecision: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("decision events = %d, want one new current-terms event", len(publisher.events))
	}
	current, ok := findProposal(updated, proposal.ID)
	if !ok || current.PreviousDecisionPublication == nil || current.DecisionPublication == nil {
		t.Fatalf("reissued proposal did not retain both publication generations: %+v", current)
	}
	if current.PreviousDecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved || current.PreviousDecisionPublication.EventID != "" {
		t.Fatalf("historical legacy marker was rewritten: %+v", current.PreviousDecisionPublication)
	}
	if current.DecisionPublication.State != ProposalDecisionPublicationPublished {
		t.Fatalf("current publication = %+v, want published new generation", current.DecisionPublication)
	}
	newEventID := operatordecision.EventID(publisher.events[0])
	if newEventID == "" || newEventID != current.DecisionPublication.EventID {
		t.Fatalf("current event identity %q does not match frozen snapshot %q", newEventID, current.DecisionPublication.EventID)
	}
	if got, _ := publisher.events[0].Payload["technical_details"].(string); !strings.HasPrefix(got, "New review request for current proposal terms.") {
		t.Fatalf("current request did not identify itself as a new terms review: %q", got)
	}
	if _, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, termsDigest); err == nil {
		t.Fatal("second reissue replaced an already frozen current request")
	}
	if len(publisher.events) != 1 {
		t.Fatalf("second reissue published another event: %d", len(publisher.events))
	}

	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	for _, answer := range []string{"approve", "reject", "request_changes"} {
		for _, requesterRef := range []string{"", "unknown-legacy-request-ref"} {
			stale := answeredResolution(session.ID, proposal.ID, answer)
			stale.RequesterRef = requesterRef
			handled, applyErr := effect.Apply(context.Background(), stale)
			if applyErr != nil || !handled {
				t.Fatalf("Apply(stale answer=%q requester_ref=%q) = handled %t, err %v", answer, requesterRef, handled, applyErr)
			}
		}
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("old or missing requester_ref applied mutations: %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)

	currentAnswer := answeredResolution(session.ID, proposal.ID, "approve")
	currentAnswer.RequesterRef = newEventID
	handled, err := effect.Apply(context.Background(), currentAnswer)
	if err != nil || !handled {
		t.Fatalf("Apply(current generation answer) = handled %t, err %v", handled, err)
	}
	if len(processor.accepted) != 1 || processor.accepted[0] != "m1" {
		t.Fatalf("current generation did not apply the explicitly accepted mutation: %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)
}

func TestLegacyProposalDecisionReissueRejectsChangedTermsWithoutPublishing(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
	proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
	oldDigest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		t.Fatalf("proposalDecisionTermsSHA256: %v", err)
	}
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proposal.Summary = "Updated current terms"
	proposal.UpdatedAt = nowRFC3339()
	if err := store.SaveProposal(session.ID, proposal); err != nil {
		t.Fatalf("SaveProposal(updated terms): %v", err)
	}
	if _, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, oldDigest); err == nil {
		t.Fatal("ReissueLegacyProposalDecision accepted a stale terms digest")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("stale terms request published %d events", len(publisher.events))
	}
	stored, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := findProposal(stored, proposal.ID)
	if !ok || current.DecisionPublication == nil || current.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved || current.PreviousDecisionPublication != nil {
		t.Fatalf("stale digest changed legacy publication state: %+v", current)
	}
}

func TestLegacyUnknownDecisionNeverActsBeforeOrAfterExplicitReissue(t *testing.T) {
	for _, answer := range []string{"approve", "reject", "request_changes"} {
		t.Run(answer, func(t *testing.T) {
			service, publisher, processor := newProposalDecisionService(t)
			session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
			proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
			digest, err := proposalDecisionTermsSHA256(proposal)
			if err != nil {
				t.Fatal(err)
			}
			effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
			oldAnswer := answeredResolution(session.ID, proposal.ID, answer)
			oldAnswer.RequesterRef = "historical-unresolved-request"
			if answer == "request_changes" {
				oldAnswer.AskID = "ask-race-fixture"
				oldAnswer.Note = "Please narrow the proposal."
			}
			if handled, applyErr := effect.Apply(context.Background(), oldAnswer); applyErr != nil || !handled {
				t.Fatalf("unknown legacy answer handled=%t err=%v", handled, applyErr)
			}
			if len(processor.accepted) != 0 || service.spawner.(*fakeSessionSpawner).continueCalls != 0 {
				t.Fatal("unknown legacy answer caused an effect before reissue")
			}
			assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)

			if _, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, digest); err != nil {
				t.Fatalf("explicit reissue after ignored legacy answer: %v", err)
			}
			if len(publisher.events) != 1 {
				t.Fatalf("explicit reissue published %d events, want one", len(publisher.events))
			}
			if handled, applyErr := effect.Apply(context.Background(), oldAnswer); applyErr != nil || !handled {
				t.Fatalf("pre-generation legacy answer after reissue handled=%t err=%v", handled, applyErr)
			}
			if len(processor.accepted) != 0 || service.spawner.(*fakeSessionSpawner).continueCalls != 0 {
				t.Fatal("old legacy answer acted on the newly frozen generation")
			}
			assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)
		})
	}
}

func TestProposalDecisionRequiresExactKnownPublicationForEveryAnswer(t *testing.T) {
	for _, publicationCase := range []string{"no_publication", "legacy_unknown", "published_without_terms_digest", "missing_requester_ref", "wrong_requester_ref"} {
		for _, answer := range []string{"approve", "reject", "request_changes"} {
			t.Run(publicationCase+"/"+answer, func(t *testing.T) {
				service, _, processor := newProposalDecisionService(t)
				session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
				var proposal Proposal
				switch publicationCase {
				case "legacy_unknown":
					proposal = recordLegacyReadyMutationProposal(t, service, session.ID)
				case "published_without_terms_digest":
					proposal = recordPublishedProposalWithoutTermsDigest(t, service, session.ID)
				case "no_publication":
					proposal = Proposal{ID: "prop_unpublished", Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Mutation proposal for Quality Gates", PayloadJSON: readyMutationPayload, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339(), Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}}
					if err := service.store.SaveProposal(session.ID, proposal); err != nil {
						t.Fatalf("SaveProposal(unpublished legacy row): %v", err)
					}
				default:
					proposal = recordReadyMutationProposal(t, service, session.ID)
				}

				resolution := answeredResolution(session.ID, proposal.ID, answer)
				switch publicationCase {
				case "missing_requester_ref":
					// Leave requester_ref empty.
				case "wrong_requester_ref":
					resolution.RequesterRef = "different-request-event"
				case "published_without_terms_digest":
					resolution.RequesterRef = proposal.DecisionPublication.EventID
				case "legacy_unknown":
					resolution.RequesterRef = "unrecoverable-historical-event"
				case "no_publication":
					resolution.RequesterRef = "unbound-event"
				}
				if answer == "request_changes" {
					resolution.Note = "Please narrow the proposal."
				}
				if handled, err := NewProposalDecisionEffect(attemptRouterFor(t, service), service).Apply(context.Background(), resolution); err != nil || !handled {
					t.Fatalf("unknown/unbound answer handled=%t err=%v", handled, err)
				}
				if len(processor.accepted) != 0 || service.spawner.(*fakeSessionSpawner).continueCalls != 0 {
					t.Fatal("unknown/unbound publication produced an answer effect")
				}
				stored, err := service.store.LoadSession(session.ID)
				if err != nil {
					t.Fatal(err)
				}
				current, ok := findProposal(stored, proposal.ID)
				if !ok || current.Status != ProposalStatusReady || len(current.Decisions) != 0 {
					t.Fatalf("unknown/unbound answer changed proposal: %+v", current)
				}
			})
		}
	}
}

func recordPublishedProposalWithoutTermsDigest(t *testing.T, service *Service, sessionID string) Proposal {
	t.Helper()
	proposal := Proposal{
		ID: "prop_missing_terms_digest", Kind: ProposalMutationList, Status: ProposalStatusReady,
		Summary: "Mutation proposal for Quality Gates", PayloadJSON: readyMutationPayload,
		CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339(),
		Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}
	event := service.proposalDecisionEvent(context.Background(), sessionID, proposal, time.Now().UTC())
	proposal.DecisionPublication = &ProposalDecisionPublication{
		Version: 1, State: ProposalDecisionPublicationPublished,
		EventID: operatordecision.EventID(event), Event: event, PublishedAt: nowRFC3339(),
	}
	if err := service.store.SaveProposal(sessionID, proposal); err != nil {
		t.Fatalf("SaveProposal(legacy published event without digest): %v", err)
	}
	return proposal
}

func TestReissuedCurrentGenerationAnswersRemainUsable(t *testing.T) {
	for _, answer := range []string{"approve", "reject", "request_changes"} {
		t.Run(answer, func(t *testing.T) {
			service, publisher, processor := newProposalDecisionService(t)
			session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
			proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
			digest, err := proposalDecisionTermsSHA256(proposal)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, digest); err != nil {
				t.Fatalf("ReissueLegacyProposalDecision: %v", err)
			}
			if len(publisher.events) != 1 {
				t.Fatalf("reissue events=%d, want one", len(publisher.events))
			}
			resolution := answeredResolution(session.ID, proposal.ID, answer)
			resolution.RequesterRef = operatordecision.EventID(publisher.events[0])
			if answer == "request_changes" {
				resolution.AskID = "ask-current-generation"
				resolution.Note = "Please narrow the proposal."
			}
			if handled, applyErr := NewProposalDecisionEffect(attemptRouterFor(t, service), service).Apply(context.Background(), resolution); applyErr != nil || !handled {
				t.Fatalf("current answer handled=%t err=%v", handled, applyErr)
			}
			wantStatus := map[string]ProposalStatus{"approve": ProposalStatusApplied, "reject": ProposalStatusRejected, "request_changes": ProposalStatusSuperseded}[answer]
			assertProposalStatus(t, service, session.ID, proposal.ID, wantStatus)
			if answer == "approve" && (len(processor.accepted) != 1 || processor.accepted[0] != "m1") {
				t.Fatalf("current approve did not apply mutations: %v", processor.accepted)
			}
			if answer != "approve" && len(processor.accepted) != 0 {
				t.Fatalf("non-approve answer applied mutations: %v", processor.accepted)
			}
		})
	}
}

func TestReissuedProposalFactsCannotChangeBetweenRequestAndAnswer(t *testing.T) {
	service, publisher, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
	proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
	digest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, digest)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := findProposal(updated, proposal.ID)
	if !ok || current.DecisionPublication == nil || current.DecisionPublication.DecisionTermsSHA256 == "" {
		t.Fatalf("current request lacks immutable terms binding: %+v", current.DecisionPublication)
	}
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Proposal){
		func(p *Proposal) { p.PayloadJSON = `{"form":"mutation_list","mutations":[{"id":"different"}]}` },
		func(p *Proposal) { p.Target.Name = "Different goal" },
	} {
		tampered := current
		tampered.DecisionPublication = current.DecisionPublication
		mutate(&tampered)
		if err := store.SaveProposal(session.ID, tampered); err == nil {
			t.Fatal("SaveProposal accepted changed facts after the current request was frozen")
		}
	}
	if len(publisher.events) != 1 {
		t.Fatalf("tampering attempts published %d additional events", len(publisher.events)-1)
	}
	answer := answeredResolution(session.ID, proposal.ID, "approve")
	answer.RequesterRef = current.DecisionPublication.EventID
	handled, err := NewProposalDecisionEffect(attemptRouterFor(t, service), service).Apply(context.Background(), answer)
	if err != nil || !handled {
		t.Fatalf("Apply(exact current terms) = handled %t, err %v", handled, err)
	}
	if len(processor.accepted) != 1 || processor.accepted[0] != "m1" {
		t.Fatalf("unchanged exact proposal facts did not reach existing decision authority: %v", processor.accepted)
	}
}

func TestLegacyProposalDecisionTermsDigestIsReturnedAndFrozenRetryReusesEvent(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Review the prepared change.")
	proposal := recordLegacyReadyMutationProposal(t, service, session.ID)
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	session.ProposalTarget = proposal.Target
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession(target): %v", err)
	}

	request := httptest.NewRequest("GET", "/api/v1/proposal-sessions?target_type=goal&target_ref=quality-gates", nil)
	recorder := httptest.NewRecorder()
	NewHandler(service).ListProposalSessions(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("ListProposalSessions status = %d, body %q", recorder.Code, recorder.Body.String())
	}
	var listed struct {
		Sessions []proposalSessionRead `json:"sessions"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode ListProposalSessions: %v", err)
	}
	if len(listed.Sessions) != 1 || listed.Sessions[0].DecisionTermsSHA256[proposal.ID] == "" {
		t.Fatalf("eligible unresolved legacy terms digest missing from list: %+v", listed.Sessions)
	}
	digest := listed.Sessions[0].DecisionTermsSHA256[proposal.ID]
	if digest != mustDecisionTermsDigest(t, proposal) {
		t.Fatalf("listed terms digest = %q, want canonical proposal digest", digest)
	}

	publisher.err = errors.New("Events unavailable")
	if _, err := service.ReissueLegacyProposalDecision(context.Background(), session.ID, proposal.ID, digest); err != nil {
		t.Fatalf("ReissueLegacyProposalDecision: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("initial failed publication attempts = %d, want one", len(publisher.events))
	}
	firstEventID := operatordecision.EventID(publisher.events[0])
	stored, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := findProposal(stored, proposal.ID)
	if !ok || current.DecisionPublication == nil || current.DecisionPublication.State != ProposalDecisionPublicationPending {
		t.Fatalf("failed new-generation publish was not retained as pending: %+v", current)
	}

	publisher.err = nil
	if _, err := service.ReconcileProposalDecisionPage(context.Background(), ProposalDecisionCursor{}, 10); err != nil {
		t.Fatalf("ReconcileProposalDecisionPage: %v", err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("retry event attempts = %d, want two", len(publisher.events))
	}
	if got := operatordecision.EventID(publisher.events[1]); got != firstEventID {
		t.Fatalf("retry changed frozen generation event ID: got %q, want %q", got, firstEventID)
	}
	stored, err = store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, _ = findProposal(stored, proposal.ID)
	if current.DecisionPublication == nil || current.DecisionPublication.State != ProposalDecisionPublicationPublished || current.DecisionPublication.EventID != firstEventID {
		t.Fatalf("retried generation receipt = %+v, want published immutable event %q", current.DecisionPublication, firstEventID)
	}
}

func mustDecisionTermsDigest(t *testing.T, proposal Proposal) string {
	t.Helper()
	digest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		t.Fatalf("proposalDecisionTermsSHA256: %v", err)
	}
	return digest
}

// A ready mutation-list proposal raises exactly ONE decision on the rail. The
// durable successful publication receipt suppresses redundant re-raises.
func TestReadyProposalRaisesExactlyOneDecision(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)

	if len(publisher.events) != 1 {
		t.Fatalf("expected exactly one decision event, got %d", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != ProposalDecisionEventType {
		t.Fatalf("event type = %q, want %q", event.EventType, ProposalDecisionEventType)
	}
	if event.Source != ProposalDecisionSource {
		t.Fatalf("event source = %q, want %q", event.Source, ProposalDecisionSource)
	}
	correlation, _ := event.Payload["correlation"].(map[string]any)
	if got := correlation["subject_ref"]; got != session.ID+"/"+proposal.ID {
		t.Fatalf("subject_ref = %v, want %s", got, session.ID+"/"+proposal.ID)
	}
	if got := correlation["subject_kind"]; got != AttemptSubjectProposal {
		t.Fatalf("subject_kind = %v, want %s", got, AttemptSubjectProposal)
	}
	if got, _ := event.Payload["question"].(string); !strings.Contains(got, "m1") || !strings.Contains(got, "Quality Gates") {
		t.Fatalf("question missing mutation id or target: %q", got)
	}
	options, _ := event.Payload["options"].([]any)
	optionKeys := make(map[string]bool, len(options))
	for _, value := range options {
		option, _ := value.(map[string]any)
		optionKeys[option["key"].(string)] = true
	}
	for _, key := range []string{"approve", "reject", "request_changes"} {
		if !optionKeys[key] {
			t.Fatalf("decision options missing %q: %#v", key, options)
		}
	}

	// A successful event publication receipt prevents another publish attempt.
	service.emitProposalDecisionRequested(context.Background(), session.ID, proposal)
	if len(publisher.events) != 1 {
		t.Fatalf("published receipt should suppress another attempt; got %d events", len(publisher.events))
	}
	if proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationPublished {
		t.Fatalf("proposal has no successful publication receipt: %+v", proposal.DecisionPublication)
	}
}

func TestProposalDecisionTechnicalDetailsCarryReviewableItemContextBoundedly(t *testing.T) {
	criteria := []string{
		"Concurrent admissions allow exactly one current run.",
		"Active attempts reject another ordinary run.",
		"Sequential retry claims a fresh run ID.",
		"Duplicate failed recovery reuses its replacement.",
		"Stale runs cannot overwrite current task state.",
		"Setup failure settles only its matching claim.",
		"Automatic apply does not imply human acceptance.",
		"Human decision attribution remains distinct.",
		"Ambiguous legacy histories stay unresolved.",
	}
	payload, err := json.Marshal(map[string]any{
		"form": "mutation_list",
		"mutations": []any{map[string]any{
			"id": "capture-item-1", "op": "add_item", "target": "fix/agent-manager-task-current-run-projection",
			"rationale": "Keep task execution status tied to its current attempt.",
			"item": map[string]any{
				"kind": "fix", "name": "agent-manager-task-current-run-projection",
				"title":       "Fence task execution status to its current run",
				"description": "Atomically claim each attempt before sandbox effects and preserve human acceptance as a separate decision.",
				"priority":    8, "effort": "L", "acceptance_allow": criteria,
				"acceptance_deny": []string{"Do not treat a completed run as independent human acceptance."},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	proposal := Proposal{
		Summary:     "Task status must track the current run without claiming human approval.",
		PayloadJSON: string(payload),
		Target:      &ProposalTarget{Ref: "capture/cap-a5456e585ccb1150", Name: "Task lifecycle status"},
	}
	service, _, _ := newProposalDecisionService(t)
	service.SetProposalNoticeBaseResolver(func(context.Context, string) (string, error) { return "https://swarm.example.test", nil })
	event := service.proposalDecisionEvent(context.Background(), "sess-1", proposal, time.Now().UTC())
	details, ok := event.Payload["technical_details"].(string)
	if !ok {
		t.Fatalf("decision event technical_details = %#v, want string", event.Payload["technical_details"])
	}
	if event.Payload["context_url"] != "https://swarm.example.test/sessions/sess-1?tab=proposals" {
		t.Fatalf("decision event context_url = %v", event.Payload["context_url"])
	}
	bigPicture, _ := event.Payload["big_picture"].(string)
	if !strings.Contains(bigPicture, "records 1 proposed backlog item") || !strings.Contains(bigPicture, "does not implement that work") {
		t.Fatalf("capture add_item purpose/consequence is unclear: %q", bigPicture)
	}
	otherEffect := proposal
	otherEffect.PayloadJSON = `{"mutations":[{"id":"m2","op":"update_item","target":"fix/existing"}]}`
	otherEffect.Target = &ProposalTarget{Ref: "fix/existing", Name: "Existing backlog item"}
	otherEvent := service.proposalDecisionEvent(context.Background(), "sess-1", otherEffect, time.Now().UTC())
	otherBigPicture, _ := otherEvent.Payload["big_picture"].(string)
	if !strings.Contains(otherBigPicture, "applies 1 listed change to Existing backlog item") || strings.Contains(otherBigPicture, "does not implement") {
		t.Fatalf("non-add-item summary overstates its effect: %q", otherBigPicture)
	}
	for _, expected := range []string{
		"agent-manager-task-current-run-projection",
		"Fence task execution status to its current run",
		"Atomically claim each attempt before sandbox effects",
		"Item kind: fix", "priority 8", "effort L", "— Keep task execution status tied to its current attempt.", "Must not:",
		"Do not treat a completed run as independent human acceptance.",
	} {
		if !strings.Contains(details, expected) {
			t.Errorf("technical details omitted %q: %q", expected, details)
		}
	}
	for _, criterion := range criteria {
		if !strings.Contains(details, criterion) {
			t.Errorf("technical details omitted acceptance criterion %q", criterion)
		}
	}
	if got := utf8.RuneCountInString(details); got > operatordecision.MaxTechnicalDetailsCodePoints {
		t.Fatalf("technical details have %d code points, limit %d", got, operatordecision.MaxTechnicalDetailsCodePoints)
	}
	if question, _ := event.Payload["question"].(string); len(question) > operatordecision.MaxQuestion {
		t.Fatalf("concise question has %d bytes, limit %d", len(question), operatordecision.MaxQuestion)
	}

	malformed := proposal
	malformed.PayloadJSON = "{"
	malformedEvent := service.proposalDecisionEvent(context.Background(), "sess-1", malformed, time.Now().UTC())
	malformedDetails, _ := malformedEvent.Payload["technical_details"].(string)
	if !strings.Contains(malformedDetails, "could not be parsed") || strings.Contains(malformedDetails, "PayloadJSON") {
		t.Fatalf("malformed proposal did not produce a safe explicit fallback: %q", malformedDetails)
	}

	oversized := proposal
	oversized.PayloadJSON = `{"mutations":[{"id":"large","op":"add_item","item":{"name":"large-item","title":"Large item","description":"` + strings.Repeat("x", operatordecision.MaxTechnicalDetailsCodePoints*2) + `","acceptance_allow":["retained criterion"]}}]}`
	oversizedEvent := service.proposalDecisionEvent(context.Background(), "sess-1", oversized, time.Now().UTC())
	bounded, _ := oversizedEvent.Payload["technical_details"].(string)
	if got := utf8.RuneCountInString(bounded); got != operatordecision.MaxTechnicalDetailsCodePoints {
		t.Fatalf("oversized technical details have %d code points, want bound %d", got, operatordecision.MaxTechnicalDetailsCodePoints)
	}
	if !strings.HasSuffix(bounded, "[Additional proposal details are available from the review link.]") {
		t.Fatalf("bounded details do not explain where remaining detail lives: suffix=%q", bounded[len(bounded)-80:])
	}
	noLinkService, _, _ := newProposalDecisionService(t)
	noLinkEvent := noLinkService.proposalDecisionEvent(context.Background(), "sess-1", oversized, time.Now().UTC())
	withoutLink, _ := noLinkEvent.Payload["technical_details"].(string)
	if !strings.HasSuffix(withoutLink, "[Additional proposal details were omitted to fit this bounded summary.]") || strings.Contains(withoutLink, "available from the review link") {
		t.Fatalf("bounded details without a review URL made a false link claim: suffix=%q", withoutLink[len(withoutLink)-90:])
	}
}

func TestProposalDecisionPublicationRetriesFrozenEventAfterReload(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	publisher.err = errors.New("events temporarily unavailable")
	service.SetProposalNoticeBaseResolver(func(context.Context, string) (string, error) { return "https://swarm.example.test", nil })
	proposal := recordReadyMutationProposal(t, service, session.ID)
	if len(publisher.events) != 1 {
		t.Fatalf("first event attempts = %d, want one", len(publisher.events))
	}
	first := publisher.events[0]
	if proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationPending {
		t.Fatalf("failed publish was not retained as pending: %+v", proposal.DecisionPublication)
	}
	if got := first.Payload["context_url"]; got != "https://swarm.example.test/sessions/"+session.ID+"?tab=proposals" {
		t.Fatalf("context_url = %v", got)
	}
	if got := first.Payload["big_picture"]; got == "" {
		t.Fatal("decision omitted its practical purpose")
	}
	if got := first.Payload["technical_details"]; got == "" {
		t.Fatal("decision omitted known proposal details")
	}

	// A new Service instance over the same FileStore models the durable owner
	// after process restart; publication retries use the saved event snapshot.
	reloaded, err := NewService(ServiceConfig{Store: service.store, Spawner: &fakeSessionSpawner{}, ProjectRoot: "/repo", ProfileKey: "swarm-manager/default"})
	if err != nil {
		t.Fatal(err)
	}
	publisher.err = nil
	reloaded.SetDecisionEventPublisher(publisher)
	reloaded.SetProposalNoticeBaseResolver(func(context.Context, string) (string, error) { return "https://changed.example.test", nil })
	stored, err := reloaded.loadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Proposals) != 1 {
		t.Fatalf("reloaded proposals = %d, want one", len(stored.Proposals))
	}
	reloaded.emitProposalDecisionRequested(context.Background(), session.ID, stored.Proposals[0])
	if len(publisher.events) != 2 {
		t.Fatalf("retry event attempts = %d, want two", len(publisher.events))
	}
	if operatordecision.EventID(first) != operatordecision.EventID(publisher.events[1]) {
		t.Fatalf("retry changed immutable event ID: %s -> %s", operatordecision.EventID(first), operatordecision.EventID(publisher.events[1]))
	}
	if got := publisher.events[1].Payload["context_url"]; got != first.Payload["context_url"] {
		t.Fatalf("retry changed frozen context URL: %v -> %v", first.Payload["context_url"], got)
	}
	stored, err = reloaded.loadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Proposals[0].DecisionPublication; got == nil || got.State != ProposalDecisionPublicationPublished || got.Attempts != 2 {
		t.Fatalf("retry receipt = %+v, want published after two attempts", got)
	}
	reloaded.emitProposalDecisionRequested(context.Background(), session.ID, stored.Proposals[0])
	if len(publisher.events) != 2 {
		t.Fatalf("published receipt did not suppress duplicate retry: %d events", len(publisher.events))
	}
}

func TestProposalDecisionPublicationDoesNotOverwriteConcurrentDecision(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	publisher.onPublish = func(eventbus.DomainEvent) error {
		current, err := service.loadSession(context.Background(), session.ID)
		if err != nil {
			return err
		}
		current.Proposals[0].Status = ProposalStatusRejected
		current.Proposals[0].UpdatedAt = nowRFC3339()
		current.Proposals[0].Decisions = append(current.Proposals[0].Decisions, ProposalDecision{Kind: "reject", DecidedAt: nowRFC3339()})
		return service.saveProposal(context.Background(), session.ID, current.Proposals[0])
	}
	proposal := recordReadyMutationProposal(t, service, session.ID)
	if proposal.DecisionPublication == nil {
		t.Fatal("proposal lacks frozen publication")
	}
	stored, err := service.loadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := stored.Proposals[0]
	if got.Status != ProposalStatusRejected || len(got.Decisions) != 1 || got.DecisionPublication == nil || got.DecisionPublication.State != ProposalDecisionPublicationPublished {
		t.Fatalf("publication receipt overwrote or lost concurrent owner decision: %+v", got)
	}
}

func TestProposalDecisionReconcilerSkipsResolvedAndMarksLegacyUnresolved(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	legacy := Proposal{ID: "prop_legacy", Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Legacy", PayloadJSON: readyMutationPayload, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339(), Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}}
	if err := service.store.SaveProposal(session.ID, legacy); err != nil {
		t.Fatal(err)
	}
	resolved := legacy
	resolved.ID = "prop_resolved"
	resolved.Status = ProposalStatusRejected
	if err := service.store.SaveProposal(session.ID, resolved); err != nil {
		t.Fatal(err)
	}
	page, err := service.ReconcileProposalDecisionPage(context.Background(), ProposalDecisionCursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasMore {
		t.Fatal("unexpected second page")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("legacy/resolved proposals were published: %d events", len(publisher.events))
	}
	stored, err := service.loadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range stored.Proposals {
		if item.ID == legacy.ID && (item.DecisionPublication == nil || item.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved) {
			t.Fatalf("legacy ready proposal not marked unresolved: %+v", item.DecisionPublication)
		}
		if item.ID == resolved.ID && item.DecisionPublication != nil {
			t.Fatalf("resolved proposal received publication metadata: %+v", item.DecisionPublication)
		}
	}
}

func TestDisabledProposalDecisionPublisherDoesNotClaimPublication(t *testing.T) {
	service := newTestService(t, &fakeSessionSpawner{})
	service.SetDecisionEventPublisher(disabledDecisionPublisher{})
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	if proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationPending || proposal.DecisionPublication.LastError == "" {
		t.Fatalf("disabled publisher falsely claimed success: %+v", proposal.DecisionPublication)
	}
}

func TestProposalDecisionPagesAdvanceByStableOwnerKey(t *testing.T) {
	store := NewFileStore(t.TempDir())
	for _, id := range []string{"sess_page_a", "sess_page_b", "sess_page_c"} {
		session := validStoredSession(id)
		if err := store.CreateSession(session); err != nil {
			t.Fatal(err)
		}
		proposal := Proposal{ID: "prop_" + id[len("sess_"):], Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: id, PayloadJSON: readyMutationPayload, CreatedAt: testTimestamp, UpdatedAt: testTimestamp, Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}}
		if err := store.SaveProposal(id, proposal); err != nil {
			t.Fatal(err)
		}
	}
	cursor := ProposalDecisionCursor{}
	var got []string
	for pageNumber := 0; pageNumber < 3; pageNumber++ {
		page, err := store.ListProposalDecisionPage(cursor, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Candidates) != 1 {
			t.Fatalf("page %d candidates = %d, want one", pageNumber, len(page.Candidates))
		}
		got = append(got, page.Candidates[0].SessionID)
		if pageNumber < 2 && !page.HasMore {
			t.Fatalf("page %d reported end before all owner keys", pageNumber)
		}
		cursor = page.Next
	}
	if strings.Join(got, ",") != "sess_page_a,sess_page_b,sess_page_c" {
		t.Fatalf("stable page order = %v", got)
	}
	last, err := store.ListProposalDecisionPage(cursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Candidates) != 0 || last.HasMore {
		t.Fatalf("terminal page = %+v, want exhausted cursor", last)
	}
}

func TestProposalDecisionReconcilerAdvancesPastCorruptRecordAndRetriesLaterProposal(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	fileStore := service.store.(*FileStore)
	for _, sessionID := range []string{"sess_a_corrupt", "sess_b_valid"} {
		if err := fileStore.CreateSession(validStoredSession(sessionID)); err != nil {
			t.Fatal(err)
		}
	}
	corruptPath := filepath.Join(fileStore.sessionDir("sess_a_corrupt"), proposalsDirName, "aaa.json")
	if err := os.MkdirAll(filepath.Dir(corruptPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corruptPath, []byte("{malformed"), 0o600); err != nil {
		t.Fatal(err)
	}

	publisher.err = errors.New("events temporarily unavailable")
	proposal := recordReadyMutationProposal(t, service, "sess_b_valid")
	if proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationPending || len(publisher.events) != 1 {
		t.Fatalf("setup did not leave a frozen pending event: publication=%+v events=%d", proposal.DecisionPublication, len(publisher.events))
	}
	firstEvent := publisher.events[0]
	publisher.err = nil

	page, err := service.ReconcileProposalDecisionPage(context.Background(), ProposalDecisionCursor{}, 10)
	if err != nil {
		t.Fatalf("bounded page failed on one corrupt proposal record: %v", err)
	}
	if len(page.Issues) != 1 || page.Issues[0].SessionID != "sess_a_corrupt" || page.Issues[0].ProposalFile != "aaa.json" || page.Issues[0].Reason != "proposal_unreadable" {
		t.Fatalf("corrupt proposal was not surfaced as unresolved page issue: %+v", page.Issues)
	}
	if len(page.Candidates) != 1 || page.Candidates[0].SessionID != "sess_b_valid" || page.Candidates[0].Proposal.ID != proposal.ID || page.HasMore {
		t.Fatalf("valid later proposal was not visited within the page: %+v", page)
	}
	if len(publisher.events) != 2 || operatordecision.EventID(firstEvent) != operatordecision.EventID(publisher.events[1]) {
		t.Fatalf("valid later notification was not retried with its frozen event: %d events", len(publisher.events))
	}
	if _, err := os.Stat(corruptPath); err != nil {
		t.Fatalf("corrupt proposal must remain in place and unresolved: %v", err)
	}
	stored, err := service.loadSession(context.Background(), "sess_b_valid")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Proposals) != 1 || stored.Proposals[0].DecisionPublication == nil || stored.Proposals[0].DecisionPublication.State != ProposalDecisionPublicationPublished {
		t.Fatalf("later valid proposal did not receive a durable publication receipt: %+v", stored.Proposals)
	}
}

// A needs-revision proposal is not an operator decision and raises nothing; a
// no-change recommendation is reviewed on the keep-as-is path, not this rail.
func TestNonReadyMutationProposalRaisesNoDecision(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")

	if _, err := service.RecordProposal(context.Background(), session.ID, Proposal{
		Kind:          ProposalMutationList,
		Status:        ProposalStatusNeedsRevision,
		Summary:       "Revision requested for Quality Gates",
		NeedsRevision: true,
		PayloadJSON:   readyMutationPayload,
		Target:        &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}); err != nil {
		t.Fatalf("RecordProposal(needs_revision): %v", err)
	}
	if _, err := service.RecordProposal(context.Background(), session.ID, Proposal{
		Kind:        ProposalNoChangeRecommendation,
		Status:      ProposalStatusReady,
		Summary:     "Keep Quality Gates as-is",
		PayloadJSON: `{"form":"mutation_list","mutations":[]}`,
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}); err != nil {
		t.Fatalf("RecordProposal(no_change): %v", err)
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no decision events for non-ready/no-change proposals, got %d", len(publisher.events))
	}
}

// An APPROVE answer applies the proposal's candidate mutation ids through the
// single authority; the proposal ends applied.
func TestProposalDecisionApproveApplies(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	handled, err := effect.Apply(context.Background(), currentAnsweredResolution(t, service, session.ID, proposal.ID, "approve"))
	if err != nil {
		t.Fatalf("Apply(approve): %v", err)
	}
	if !handled {
		t.Fatal("approve resolution was not handled")
	}
	if len(processor.accepted) != 1 || processor.accepted[0] != "m1" {
		t.Fatalf("authority received accepted ids %v, want [m1]", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)
}

// A REJECT answer records a rejection (no mutation ids accepted); the proposal
// ends rejected and the authority applies nothing.
func TestProposalDecisionRejectRecordsRejection(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	handled, err := effect.Apply(context.Background(), currentAnsweredResolution(t, service, session.ID, proposal.ID, "reject"))
	if err != nil {
		t.Fatalf("Apply(reject): %v", err)
	}
	if !handled {
		t.Fatal("reject resolution was not handled")
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("authority should apply nothing on reject, got accepted %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusRejected)
}

// Request changes resumes the original proposal session with the operator's
// note, supersedes the reviewed proposal, and never sends an approval to the
// mutation authority.
func TestProposalDecisionRequestChangesContinuesOwnerWithoutApplying(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	spawner := service.spawner.(*fakeSessionSpawner)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	resolution := currentAnsweredResolution(t, service, session.ID, proposal.ID, "request_changes")
	resolution.Note = "Narrow this to the one missing acceptance gate."

	handled, err := effect.Apply(context.Background(), resolution)
	if err != nil {
		t.Fatalf("Apply(request_changes): %v", err)
	}
	if !handled {
		t.Fatal("request-changes resolution was not handled")
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("request changes must not apply mutations, accepted %v", processor.accepted)
	}
	if spawner.continueRunID != session.RunID || !strings.Contains(spawner.continueMessage, resolution.Note) {
		t.Fatalf("revision was not routed to the original session run with its note: run=%q message=%q", spawner.continueRunID, spawner.continueMessage)
	}
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, found := findProposal(updated, proposal.ID)
	if !found || current.Status != ProposalStatusSuperseded {
		t.Fatalf("reviewed proposal status = %+v, want superseded", current)
	}
}

func TestProposalDecisionRequestChangesRetryReusesPersistedContinuation(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	spawner := service.spawner.(*fakeSessionSpawner)
	spawner.continueErr = errors.New("agent-manager response lost after acceptance")
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	resolution := currentAnsweredResolution(t, service, session.ID, proposal.ID, "request_changes")
	resolution.AskID = "ask-stable-revision"
	resolution.Note = "Narrow this to the one missing acceptance gate."

	if _, err := effect.Apply(context.Background(), resolution); err == nil {
		t.Fatal("first delivery should report the lost continuation response")
	}
	firstMessage, firstKey := spawner.continueMessage, spawner.continueIdempotencyKey
	if firstKey == "" {
		t.Fatal("first continuation omitted its idempotency key")
	}
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	changed, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed.RunID = "replacement-run"
	if err := store.SaveSession(changed); err != nil {
		t.Fatal(err)
	}
	if _, err := effect.Apply(context.Background(), resolution); err != nil {
		t.Fatalf("retry same request-changes resolution: %v", err)
	}
	if spawner.continueCalls != 2 || spawner.continueRunID != session.RunID || spawner.continueMessage != firstMessage || spawner.continueIdempotencyKey != firstKey {
		t.Fatalf("retry did not replay identical continuation: calls=%d key=%q message=%q", spawner.continueCalls, spawner.continueIdempotencyKey, spawner.continueMessage)
	}
	if _, err := effect.Apply(context.Background(), resolution); err != nil {
		t.Fatalf("duplicate after acknowledged retry: %v", err)
	}
	if spawner.continueCalls != 2 {
		t.Fatalf("acknowledged duplicate issued another continuation, calls=%d", spawner.continueCalls)
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("request changes applied mutations: %v", processor.accepted)
	}
	updated, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, found := findProposal(updated, proposal.ID)
	if !found || current.Status != ProposalStatusSuperseded || len(current.Decisions) != 1 || current.Decisions[0].AskID != resolution.AskID || !current.Decisions[0].ContinuationAcked || current.Decisions[0].RunID != session.RunID {
		t.Fatalf("revision decision was not durably singular: %+v", current)
	}
	matches := 0
	for _, message := range updated.Messages {
		if message.ID == proposalDecisionMessageID(resolution.AskID) {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("stable revision message count = %d, want 1", matches)
	}
}

func TestProposalDecisionRequestChangesRepairsSavedDecisionWithoutMessage(t *testing.T) {
	service, _, _ := newProposalDecisionService(t)
	spawner := service.spawner.(*fakeSessionSpawner)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	resolution := currentAnsweredResolution(t, service, session.ID, proposal.ID, "request_changes")
	resolution.AskID = "ask-saved-before-message"
	resolution.Note = "Add the owner note."
	prompt := revisionPrompt(proposal, resolution.Note)
	proposal.Status = ProposalStatusSuperseded
	proposal.Decisions = append(proposal.Decisions, ProposalDecision{
		Kind: "request_changes", AskID: resolution.AskID, RunID: session.RunID,
		MessageID: proposalDecisionMessageID(resolution.AskID), Prompt: prompt,
		Note: resolution.Note, DecidedAt: nowRFC3339(),
	})
	if err := service.saveProposal(context.Background(), session.ID, proposal); err != nil {
		t.Fatal(err)
	}
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	if _, err := effect.Apply(context.Background(), resolution); err != nil {
		t.Fatalf("retry saved decision without message: %v", err)
	}
	if spawner.continueRunID != session.RunID || spawner.continueMessage != prompt {
		t.Fatalf("saved receiver identity changed: run=%q message=%q", spawner.continueRunID, spawner.continueMessage)
	}
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	matches := 0
	for _, message := range updated.Messages {
		if message.ID == proposalDecisionMessageID(resolution.AskID) && message.Content == prompt {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("saved decision recovery wrote %d matching revision messages, want 1", matches)
	}
}

func TestInitialProposalDecisionPublicationBindsCurrentTerms(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	if proposal.DecisionPublication == nil || proposal.DecisionPublication.DecisionTermsSHA256 == "" {
		t.Fatalf("initial ready proposal publication has no terms binding: %+v", proposal.DecisionPublication)
	}
	want, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.DecisionPublication.DecisionTermsSHA256 != want {
		t.Fatalf("initial publication terms digest = %q, want %q", proposal.DecisionPublication.DecisionTermsSHA256, want)
	}
	if len(publisher.events) != 1 || operatordecision.EventID(publisher.events[0]) != proposal.DecisionPublication.EventID {
		t.Fatalf("initial publication is not bound to its published event: publication=%+v events=%d", proposal.DecisionPublication, len(publisher.events))
	}
}

// A STALE proposal at resolution time (already decided) is a no-op: the authority
// is never called a second time and no error escapes, so vrooli-events does not
// retry a poison message.
func TestProposalDecisionStaleAtResolutionIsNoOp(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	// First answer applies the proposal (now applied — no longer ready).
	if _, err := effect.Apply(context.Background(), currentAnsweredResolution(t, service, session.ID, proposal.ID, "approve")); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	acceptedAfterFirst := len(processor.accepted)
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)

	// A redelivered resolution for the now-stale proposal must NOT re-apply.
	handled, err := effect.Apply(context.Background(), currentAnsweredResolution(t, service, session.ID, proposal.ID, "approve"))
	if err != nil {
		t.Fatalf("stale Apply returned error (would poison-retry): %v", err)
	}
	if !handled {
		t.Fatal("stale resolution should be acknowledged (handled=true)")
	}
	if len(processor.accepted) != acceptedAfterFirst {
		t.Fatalf("authority was called again for a stale proposal: accepted %v", processor.accepted)
	}
}

// An unanswered (expired/defaulted) resolution leaves the proposal untouched.
func TestProposalDecisionUnansweredIsNoOp(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	res := answeredResolution(session.ID, proposal.ID, "approve")
	res.State = operatordecision.StateExpired
	handled, err := effect.Apply(context.Background(), res)
	if err != nil {
		t.Fatalf("Apply(expired): %v", err)
	}
	if !handled {
		t.Fatal("expired resolution should be acknowledged")
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("no mutation should apply for an unanswered resolution, got %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)
}

func TestProposalDecisionEffectIgnoresMismatchedCorrelationSubjectBeforeAnyEffect(t *testing.T) {
	for _, answer := range []string{"approve", "reject", "request_changes"} {
		t.Run(answer, func(t *testing.T) {
			service, _, processor := newProposalDecisionService(t)
			spawner := service.spawner.(*fakeSessionSpawner)
			session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
			verifiedProposal := recordReadyMutationProposal(t, service, session.ID)
			otherProposal, err := service.RecordProposal(context.Background(), session.ID, Proposal{
				Kind: ProposalMutationList, Status: ProposalStatusReady,
				Summary: "A different proposal with separate terms", PayloadJSON: readyMutationPayload,
				Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
			})
			if err != nil {
				t.Fatalf("RecordProposal(other): %v", err)
			}
			resolution := currentAnsweredResolution(t, service, session.ID, verifiedProposal.ID, answer)
			resolution.Correlation["subject_ref"] = session.ID + "/" + otherProposal.ID

			handled, err := NewProposalDecisionEffect(attemptRouterFor(t, service), service).Apply(context.Background(), resolution)
			if err != nil || !handled {
				t.Fatalf("Apply(mismatched %s subject): handled=%t err=%v", answer, handled, err)
			}
			if len(processor.accepted) != 0 {
				t.Fatalf("mismatched %s correlation reached mutation authority: %v", answer, processor.accepted)
			}
			if spawner.continueCalls != 0 {
				t.Fatalf("mismatched %s correlation continued proposal revision %d times", answer, spawner.continueCalls)
			}
			assertProposalStatus(t, service, session.ID, verifiedProposal.ID, ProposalStatusReady)
			assertProposalStatus(t, service, session.ID, otherProposal.ID, ProposalStatusReady)
		})
	}
}

func TestProposalDecisionEffectDerivesIDsFromCanonicalSubjectRef(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	resolution := currentAnsweredResolution(t, service, session.ID, proposal.ID, "approve")
	delete(resolution.Correlation, "session_id")
	delete(resolution.Correlation, "proposal_id")

	handled, err := NewProposalDecisionEffect(attemptRouterFor(t, service), service).Apply(context.Background(), resolution)
	if err != nil || !handled {
		t.Fatalf("Apply(subject-only): handled=%t err=%v", handled, err)
	}
	if len(processor.accepted) != 1 || processor.accepted[0] != "m1" {
		t.Fatalf("canonical subject_ref did not route the matching proposal: accepted=%v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)
}

// A foreign source event type is not this effect's to handle.
func TestProposalDecisionEffectIgnoresForeignEventType(t *testing.T) {
	service, _, _ := newProposalDecisionService(t)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	if effect.Handles("agent_manager.run.decision_requested.v1") {
		t.Fatal("effect must not claim a foreign source event type")
	}
	res := answeredResolution("s", "p", "approve")
	res.SourceEventType = "agent_manager.run.decision_requested.v1"
	handled, err := effect.Apply(context.Background(), res)
	if err != nil {
		t.Fatalf("Apply(foreign): %v", err)
	}
	if handled {
		t.Fatal("effect claimed a foreign resolution")
	}
}

func attemptRouterFor(t *testing.T, service *Service) *attempt.Router {
	t.Helper()
	router := attempt.NewRouter()
	if err := router.Register(AttemptSubjectProposal, NewAttemptDecider(service)); err != nil {
		t.Fatalf("register attempt decider: %v", err)
	}
	return router
}

func answeredResolution(sessionID, proposalID, answer string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		AskID:           "ask-" + sessionID + "-" + proposalID,
		State:           operatordecision.StateAnswered,
		Answer:          answer,
		AnsweredBy:      "operator@example.test",
		Note:            "reviewed on the rail",
		SourceEventType: ProposalDecisionEventType,
		Correlation: map[string]string{
			"subject_kind": AttemptSubjectProposal,
			"subject_ref":  sessionID + "/" + proposalID,
			"session_id":   sessionID,
			"proposal_id":  proposalID,
		},
	}
}

func currentAnsweredResolution(t *testing.T, service *Service, sessionID, proposalID, answer string) operatordecision.DecisionResolution {
	t.Helper()
	resolution := answeredResolution(sessionID, proposalID, answer)
	session, err := service.loadSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load current answer publication: %v", err)
	}
	proposal, ok := findProposal(session, proposalID)
	if !ok || proposal.DecisionPublication == nil {
		t.Fatalf("proposal %q has no current decision publication", proposalID)
	}
	resolution.RequesterRef = proposal.DecisionPublication.EventID
	return resolution
}

func assertProposalStatus(t *testing.T, service *Service, sessionID, proposalID string, want ProposalStatus) {
	t.Helper()
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatalf("storeFor: %v", err)
	}
	session, err := store.LoadSession(sessionID)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	proposal, ok := findProposal(session, proposalID)
	if !ok {
		t.Fatalf("proposal %s not found", proposalID)
	}
	if proposal.Status != want {
		t.Fatalf("proposal status = %q, want %q", proposal.Status, want)
	}
}
