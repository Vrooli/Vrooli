package planworkshop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type SubjectLoader func(Subject) (version, planID, planHash string, err error)

// ReconciliationStarter starts the single bounded internal continuation for
// a response that accepts non-direct proposals. Its idempotency key is the
// durable response ID, not a request attempt.
type ReconciliationStarter func(context.Context, Session, Response) (WorkflowProvenance, error)

type (
	ReviewStarter    func(context.Context, Session) (WorkflowProvenance, error)
	ReviewCollector  func(context.Context, Session, WorkflowProvenance) (ReviewResult, error)
	ProposalRecorder func(context.Context, Session, WorkflowProvenance, []ProposalDraft) (agentSessionID string, refs []ProposalRef, err error)
)

// DirectProposalApplier is deliberately narrow: it may only apply a proposal
// that the immutable packet classified as direct, after the operator accepts
// it. The composed Agent Session mutation flow remains the mutation authority.
type (
	DirectProposalApplier   func(context.Context, ProposalRef, string) error
	ReconciliationCollector func(context.Context, Session, Response, WorkflowProvenance) (ReconciliationResult, error)
	CandidateCreator        func(context.Context, Session, Response, WorkflowProvenance, json.RawMessage) (CandidateReference, error)
	CandidateApplier        func(context.Context, Session, CandidateReference, bool) error
	CandidateDiscarder      func(context.Context, Session, CandidateReference, string) error
)

type Service struct {
	store                         *Store
	load                          SubjectLoader
	now                           func() time.Time
	start                         ReconciliationStarter
	startReview                   ReviewStarter
	collectReview                 ReviewCollector
	recordProposals               ProposalRecorder
	applyDirect                   DirectProposalApplier
	collectReconciliation         ReconciliationCollector
	applyReviewTransition         func(context.Context, Session, WorkflowProvenance) error
	applyReconciliationTransition func(context.Context, Session, Response, WorkflowProvenance) error
	createCandidate               CandidateCreator
	applyCandidate                CandidateApplier
	discardCandidate              CandidateDiscarder
}

func NewService(store *Store, loader SubjectLoader) *Service {
	return &Service{store: store, load: loader, now: time.Now}
}

func (s *Service) SetReconciliationStarter(starter ReconciliationStarter) { s.start = starter }
func (s *Service) SetReviewStarter(starter ReviewStarter)                 { s.startReview = starter }
func (s *Service) SetReviewCollector(collector ReviewCollector)           { s.collectReview = collector }
func (s *Service) SetProposalRecorder(recorder ProposalRecorder)          { s.recordProposals = recorder }
func (s *Service) SetDirectProposalApplier(applier DirectProposalApplier) { s.applyDirect = applier }
func (s *Service) SetReconciliationCollector(collector ReconciliationCollector) {
	s.collectReconciliation = collector
}
func (s *Service) SetCandidateCreator(creator CandidateCreator)       { s.createCandidate = creator }
func (s *Service) SetCandidateApplier(applier CandidateApplier)       { s.applyCandidate = applier }
func (s *Service) SetCandidateDiscarder(discarder CandidateDiscarder) { s.discardCandidate = discarder }
func (s *Service) SetReviewTransitionApplier(applier func(context.Context, Session, WorkflowProvenance) error) {
	s.applyReviewTransition = applier
}

func (s *Service) SetReconciliationTransitionApplier(applier func(context.Context, Session, Response, WorkflowProvenance) error) {
	s.applyReconciliationTransition = applier
}

func WorkshopID(subject Subject) string {
	sum := sha256.Sum256([]byte(string(subject.Kind) + ":" + subject.Ref))
	return "pw_" + hex.EncodeToString(sum[:8])
}

func (s *Service) Open(subject Subject, packet ReviewPacket) (Session, error) {
	if err := subject.Validate(); err != nil {
		return Session{}, err
	}
	if err := packet.Validate(); err != nil {
		return Session{}, err
	}
	version, planID, planHash, err := s.load(subject)
	if err != nil {
		return Session{}, err
	}
	id := WorkshopID(subject)
	if current, err := s.store.Load(id); err == nil {
		if current.SubjectVersion == version && current.PlanContentHash == planHash && reviewPacketsEqual(current.Packet, packet) {
			return current, nil
		}
		now := s.now().UTC().Format(time.RFC3339Nano)
		current.Packet = packet
		current.SubjectVersion = version
		current.PlanID = planID
		current.PlanContentHash = planHash
		current.PacketHistory = append(current.PacketHistory, ReviewPacketVersion{ID: packetID(id, len(current.PacketHistory)+1), SubjectVersion: version, PlanContentHash: planHash, CreatedAt: now, Packet: packet})
		current.UpdatedAt = now
		return current, s.store.Save(current)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Session{}, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	session := Session{ID: id, Subject: subject, SubjectVersion: version, PlanID: planID, PlanContentHash: planHash, Packet: packet, PacketHistory: []ReviewPacketVersion{{ID: packetID(id, 1), SubjectVersion: version, PlanContentHash: planHash, CreatedAt: now, Packet: packet}}, CreatedAt: now, UpdatedAt: now}
	return session, s.store.Save(session)
}

// OpenOrGet opens a new session when absent. A nil packet is an operator
// request to revisit the current session and must never replace its latest
// review packet with an empty object.
func (s *Service) OpenOrGet(subject Subject, packet *ReviewPacket) (Session, error) {
	if packet != nil {
		return s.Open(subject, *packet)
	}
	if err := subject.Validate(); err != nil {
		return Session{}, err
	}
	if current, err := s.store.Load(WorkshopID(subject)); err == nil {
		return current, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Session{}, err
	}
	return s.Open(subject, ReviewPacket{})
}

func (s *Service) Get(id string) (Session, error) { return s.store.Load(id) }

// AttachFinding records externally-produced evidence (such as a research
// conclusion) in the same operator-visible workshop packet without creating a
// second proposal or decision store. Repeating the same finding is idempotent.
func (s *Service) AttachFinding(subject Subject, finding Finding) (Session, error) {
	if err := (ReviewPacket{Findings: []Finding{finding}}).Validate(); err != nil {
		return Session{}, err
	}
	current, err := s.OpenOrGet(subject, nil)
	if err != nil {
		return Session{}, err
	}
	for _, existing := range current.Packet.Findings {
		if existing.ID == finding.ID {
			return current, nil
		}
	}
	packet := current.Packet
	packet.Findings = append(packet.Findings, finding)
	return s.Open(subject, packet)
}

// AttachProposal adds a reference to the existing Agent Session proposal
// store. It is idempotent so observer retries cannot create duplicate operator
// actions for one evidence-producing terminal result.
func (s *Service) AttachProposal(subject Subject, proposal ProposalRef) (Session, error) {
	if err := (ReviewPacket{Proposals: []ProposalRef{proposal}}).Validate(); err != nil {
		return Session{}, err
	}
	current, err := s.OpenOrGet(subject, nil)
	if err != nil {
		return Session{}, err
	}
	for _, existing := range current.Packet.Proposals {
		if existing.SessionID == proposal.SessionID && existing.ProposalID == proposal.ProposalID {
			return current, nil
		}
	}
	packet := current.Packet
	packet.Proposals = append(packet.Proposals, proposal)
	return s.Open(subject, packet)
}

// StartReview creates the one bounded review workflow for the current
// immutable subject snapshot. Repeating a click returns its existing running
// correlation; a later review can start only after the prior result applies.
func (s *Service) StartReview(ctx context.Context, id string) (Session, ReviewRun, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if session.Review != nil && (session.Review.State == ReviewPending || session.Review.State == ReviewRunning) {
		return session, *session.Review, nil
	}
	version, _, hash, err := s.load(session.Subject)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if version != session.SubjectVersion || hash != session.PlanContentHash {
		return session, ReviewRun{State: ReviewStale, Error: "subject or canonical plan changed; reopen the Plan Workshop before reviewing"}, nil
	}
	if s.startReview == nil {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review is not configured")
	}
	// Durable pending state prevents a transport retry from creating a second
	// workflow. The starter uses the workshop ID and version as its own key.
	pending := ReviewRun{State: ReviewPending}
	session.Review = &pending
	session.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	if err := s.store.Save(session); err != nil {
		return Session{}, ReviewRun{}, err
	}
	workflow, err := s.startReview(ctx, session)
	if err != nil {
		pending.State, pending.Error = ReviewFailed, err.Error()
		session.Review = &pending
		session.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		if saveErr := s.store.Save(session); saveErr != nil {
			return Session{}, ReviewRun{}, saveErr
		}
		return session, pending, nil
	}
	run := ReviewRun{State: ReviewRunning, Workflow: workflow}
	session.Review = &run
	session.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	if err := s.store.Save(session); err != nil {
		return Session{}, ReviewRun{}, err
	}
	return session, run, nil
}

// ApplyReview projects a terminal, typed workflow result into the workshop and
// the existing Agent Session proposal store. No agent result mutates a plan or
// backlog object directly.
func (s *Service) ApplyReview(ctx context.Context, id string) (Session, ReviewRun, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if session.Review == nil {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review has not started")
	}
	if session.Review.State == ReviewApplied || session.Review.State == ReviewStale {
		return session, *session.Review, nil
	}
	if session.Review.State != ReviewRunning {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review is %s", session.Review.State)
	}
	if s.applyReviewTransition != nil {
		if err := s.applyReviewTransition(ctx, session, session.Review.Workflow); err != nil {
			return Session{}, ReviewRun{}, err
		}
		updated, err := s.store.Load(id)
		if err != nil || updated.Review == nil {
			return Session{}, ReviewRun{}, err
		}
		return updated, *updated.Review, nil
	}
	if s.collectReview == nil || s.recordProposals == nil {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review application is not configured")
	}
	version, _, hash, err := s.load(session.Subject)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if version != session.SubjectVersion || hash != session.PlanContentHash {
		session.Review.State = ReviewStale
		session.Review.Error = "subject or canonical plan changed while review was running"
		session.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		if err := s.store.Save(session); err != nil {
			return Session{}, ReviewRun{}, err
		}
		return session, *session.Review, nil
	}
	result, err := s.collectReview(ctx, session, session.Review.Workflow)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	return s.ApplyReviewResult(ctx, id, result)
}

// ApplyReviewResult projects a validated terminal result into the workshop.
// It is used by the shared transition runner; callers never receive authority
// to mutate a plan or backlog object from workflow output.
func (s *Service) ApplyReviewResult(ctx context.Context, id string, result ReviewResult) (Session, ReviewRun, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if session.Review == nil {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review has not started")
	}
	if session.Review.State == ReviewApplied || session.Review.State == ReviewStale {
		return session, *session.Review, nil
	}
	if session.Review.State != ReviewRunning {
		return Session{}, ReviewRun{}, fmt.Errorf("plan workshop review is %s", session.Review.State)
	}
	version, _, hash, err := s.load(session.Subject)
	if err != nil {
		return Session{}, ReviewRun{}, err
	}
	if version != session.SubjectVersion || hash != session.PlanContentHash {
		session.Review.State, session.Review.Error, session.UpdatedAt = ReviewStale, "subject or canonical plan changed while review was running", s.now().UTC().Format(time.RFC3339Nano)
		if err := s.store.Save(session); err != nil {
			return Session{}, ReviewRun{}, err
		}
		return session, *session.Review, nil
	}
	if err := result.Validate(); err != nil {
		return Session{}, ReviewRun{}, err
	}
	packet := ReviewPacket{Findings: result.Findings, Questions: result.Questions}
	if result.Outcome == "packet" && len(result.Proposals) > 0 {
		agentSessionID, refs, err := s.recordProposals(ctx, session, session.Review.Workflow, result.Proposals)
		if err != nil {
			return Session{}, ReviewRun{}, err
		}
		session.Review.AgentSessionID = agentSessionID
		packet.Proposals = refs
	}
	if result.Outcome != "packet" {
		packet.Findings = append(packet.Findings, Finding{ID: "review-attention", Severity: "attention", Summary: result.Reason})
	}
	if err := packet.Validate(); err != nil {
		return Session{}, ReviewRun{}, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	session.Packet = packet
	session.PacketHistory = append(session.PacketHistory, ReviewPacketVersion{ID: packetID(session.ID, len(session.PacketHistory)+1), SubjectVersion: session.SubjectVersion, PlanContentHash: session.PlanContentHash, CreatedAt: now, Packet: packet})
	session.Review.State, session.Review.AppliedAt, session.Review.Error = ReviewApplied, now, ""
	session.UpdatedAt = now
	if err := s.store.Save(session); err != nil {
		return Session{}, ReviewRun{}, err
	}
	return session, *session.Review, nil
}

// SubmitResponse records one operator decision exactly once. It never applies
// plan changes. A packet with accepted proposals needs one reconciliation
// workflow; an answer-only packet is safely resolved without agent work.
func (s *Service) SubmitResponse(ctx context.Context, id string, response Response) (Session, Resolution, error) {
	if strings.TrimSpace(response.Actor) == "" {
		return Session{}, Resolution{}, fmt.Errorf("response actor is required")
	}
	if strings.TrimSpace(response.IdempotencyKey) == "" {
		return Session{}, Resolution{}, fmt.Errorf("response idempotency_key is required")
	}
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	for _, existing := range session.Responses {
		if existing.IdempotencyKey == response.IdempotencyKey {
			for _, resolution := range session.Resolutions {
				if resolution.ResponseID == existing.ID {
					if resolution.State == ResolutionPending && len(existing.Accepted) > 0 {
						if err := s.applyDirectProposals(ctx, existing); err != nil {
							resolution.State = ResolutionUnavailable
							resolution.Error = err.Error()
							return s.replaceResolution(session, resolution)
						}
						if hasReconciliationProposal(existing.Accepted) {
							return s.startReconciliation(ctx, session, existing, resolution)
						}
						resolution.State = ResolutionDirectApplied
						resolution.Error = ""
						return s.replaceResolution(session, resolution)
					}
					return session, resolution, nil
				}
			}
		}
	}
	version, _, _, err := s.load(session.Subject)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	if response.SubjectVersion != session.SubjectVersion || version != session.SubjectVersion {
		return session, Resolution{State: ResolutionStale}, nil
	}
	if err := response.ValidateAgainst(session.Packet); err != nil {
		return Session{}, Resolution{}, err
	}
	accepted, err := canonicalAcceptedProposals(session.Packet, response.Accepted)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	response.Accepted = accepted
	if strings.TrimSpace(response.ID) == "" {
		response.ID = responseID(response.IdempotencyKey)
	}
	response.SubmittedAt = s.now().UTC().Format(time.RFC3339Nano)
	session.Responses = append(session.Responses, response)
	resolution := Resolution{ResponseID: response.ID, State: ResolutionDirectApplied, AppliedAt: response.SubmittedAt}
	if len(response.Accepted) > 0 {
		// Persist the intent before starting Agent Manager. A retry after a
		// transport interruption resumes this exact response rather than opening
		// a second reconciliation engagement.
		resolution.State = ResolutionPending
		resolution.ReconciliationID = reconciliationID(session.ID, response.ID)
	}
	session.Resolutions = append(session.Resolutions, resolution)
	session.UpdatedAt = response.SubmittedAt
	if err := s.store.Save(session); err != nil {
		return Session{}, Resolution{}, err
	}
	if len(response.Accepted) == 0 {
		return session, resolution, nil
	}
	if err := s.applyDirectProposals(ctx, response); err != nil {
		resolution.State = ResolutionUnavailable
		resolution.Error = err.Error()
		return s.replaceResolution(session, resolution)
	}
	if !hasReconciliationProposal(response.Accepted) {
		resolution.State = ResolutionDirectApplied
		resolution.Error = ""
		return s.replaceResolution(session, resolution)
	}
	return s.startReconciliation(ctx, session, response, resolution)
}

func canonicalAcceptedProposals(packet ReviewPacket, accepted []ProposalRef) ([]ProposalRef, error) {
	canonical := make(map[string]ProposalRef, len(packet.Proposals))
	for _, proposal := range packet.Proposals {
		canonical[proposal.SessionID+"/"+proposal.ProposalID] = proposal
	}
	result := make([]ProposalRef, 0, len(accepted))
	seen := make(map[string]bool, len(accepted))
	for _, proposal := range accepted {
		key := strings.TrimSpace(proposal.SessionID) + "/" + strings.TrimSpace(proposal.ProposalID)
		stored, ok := canonical[key]
		if !ok || seen[key] {
			return nil, fmt.Errorf("response accepts an invalid or duplicate proposal")
		}
		seen[key] = true
		result = append(result, stored)
	}
	return result, nil
}

func hasReconciliationProposal(accepted []ProposalRef) bool {
	for _, proposal := range accepted {
		if proposal.ApplyMode != "direct" {
			return true
		}
	}
	return false
}

func (s *Service) applyDirectProposals(ctx context.Context, response Response) error {
	for _, proposal := range response.Accepted {
		if proposal.ApplyMode != "direct" {
			continue
		}
		if s.applyDirect == nil {
			return fmt.Errorf("direct Plan Workshop proposal application is not configured")
		}
		if err := s.applyDirect(ctx, proposal, response.Actor); err != nil {
			return fmt.Errorf("apply direct proposal %s: %w", proposal.ProposalID, err)
		}
	}
	return nil
}

func (s *Service) startReconciliation(ctx context.Context, session Session, response Response, resolution Resolution) (Session, Resolution, error) {
	if resolution.Workflow != nil || resolution.State == ResolutionReconciliation {
		return session, resolution, nil
	}
	if s.start == nil {
		resolution.State = ResolutionUnavailable
		resolution.Error = "plan workshop reconciliation is not configured"
		return s.replaceResolution(session, resolution)
	}
	workflow, err := s.start(ctx, session, response)
	if err != nil {
		resolution.State = ResolutionUnavailable
		resolution.Error = err.Error()
		return s.replaceResolution(session, resolution)
	}
	resolution.State = ResolutionReconciliation
	resolution.Workflow = &workflow
	return s.replaceResolution(session, resolution)
}

func (s *Service) replaceResolution(session Session, resolution Resolution) (Session, Resolution, error) {
	for index := range session.Resolutions {
		if session.Resolutions[index].ResponseID == resolution.ResponseID {
			session.Resolutions[index] = resolution
			session.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
			if err := s.store.Save(session); err != nil {
				return Session{}, Resolution{}, err
			}
			return session, resolution, nil
		}
	}
	return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q is missing", resolution.ResponseID)
}

// ApplyReconciliation turns one terminal bounded workflow result into either
// a Plan Manager candidate reference or an explicit attention state. It never
// applies the candidate to the canonical plan.
func (s *Service) ApplyReconciliation(ctx context.Context, id, responseID string) (Session, Resolution, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	var response *Response
	for index := range session.Responses {
		if session.Responses[index].ID == responseID {
			response = &session.Responses[index]
			break
		}
	}
	if response == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop response %q is missing", responseID)
	}
	var resolution *Resolution
	for index := range session.Resolutions {
		if session.Resolutions[index].ResponseID == responseID {
			resolution = &session.Resolutions[index]
			break
		}
	}
	if resolution == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q is missing", responseID)
	}
	if resolution.State == ResolutionCandidateReady || resolution.State == ResolutionNeedsAttention || resolution.State == ResolutionStale {
		return session, *resolution, nil
	}
	if resolution.State != ResolutionReconciliation || resolution.Workflow == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop reconciliation is %s", resolution.State)
	}
	if s.applyReconciliationTransition != nil {
		if err := s.applyReconciliationTransition(ctx, session, *response, *resolution.Workflow); err != nil {
			return Session{}, Resolution{}, err
		}
		updated, err := s.store.Load(id)
		if err != nil {
			return Session{}, Resolution{}, err
		}
		for _, item := range updated.Resolutions {
			if item.ResponseID == responseID {
				return updated, item, nil
			}
		}
		return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q disappeared after transition apply", responseID)
	}
	if s.collectReconciliation == nil || s.createCandidate == nil {
		return Session{}, Resolution{}, fmt.Errorf("plan workshop reconciliation application is not configured")
	}
	version, _, hash, err := s.load(session.Subject)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	if version != session.SubjectVersion || hash != session.PlanContentHash {
		resolution.State = ResolutionStale
		resolution.Error = "subject or canonical plan changed while reconciliation was running"
		return s.replaceResolution(session, *resolution)
	}
	result, err := s.collectReconciliation(ctx, session, *response, *resolution.Workflow)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	return s.ApplyReconciliationResult(ctx, id, responseID, result)
}

// ApplyReconciliationResult projects a terminal reconciliation result into a
// candidate or attention resolution. Candidate application remains explicit.
func (s *Service) ApplyReconciliationResult(ctx context.Context, id, responseID string, result ReconciliationResult) (Session, Resolution, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	var response *Response
	for index := range session.Responses {
		if session.Responses[index].ID == responseID {
			response = &session.Responses[index]
			break
		}
	}
	if response == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop response %q is missing", responseID)
	}
	var resolution *Resolution
	for index := range session.Resolutions {
		if session.Resolutions[index].ResponseID == responseID {
			resolution = &session.Resolutions[index]
			break
		}
	}
	if resolution == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q is missing", responseID)
	}
	if resolution.State == ResolutionCandidateReady || resolution.State == ResolutionNeedsAttention || resolution.State == ResolutionStale {
		return session, *resolution, nil
	}
	if resolution.State != ResolutionReconciliation || resolution.Workflow == nil {
		return Session{}, Resolution{}, fmt.Errorf("workshop reconciliation is %s", resolution.State)
	}
	version, _, hash, err := s.load(session.Subject)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	if version != session.SubjectVersion || hash != session.PlanContentHash {
		resolution.State, resolution.Error = ResolutionStale, "subject or canonical plan changed while reconciliation was running"
		return s.replaceResolution(session, *resolution)
	}
	if err := result.Validate(); err != nil {
		return Session{}, Resolution{}, err
	}
	if result.Outcome == "candidate" {
		candidate, err := s.createCandidate(ctx, session, *response, *resolution.Workflow, result.CandidatePlan)
		if err != nil {
			return Session{}, Resolution{}, err
		}
		if strings.TrimSpace(candidate.ID) == "" || candidate.PlanID != session.PlanID || candidate.ExpectedBaseContentHash != session.PlanContentHash {
			return Session{}, Resolution{}, fmt.Errorf("candidate does not match the immutable workshop frontier")
		}
		resolution.State, resolution.Candidate, resolution.Error = ResolutionCandidateReady, &candidate, ""
	} else {
		resolution.State, resolution.Error = ResolutionNeedsAttention, result.Reason
	}
	resolution.AppliedAt = s.now().UTC().Format(time.RFC3339Nano)
	return s.replaceResolution(session, *resolution)
}

// ApplyCandidate is the final explicit authorization boundary. It rechecks
// the current frontier and delegates only to Plan Manager's guarded candidate
// apply API; no workflow result can update a canonical plan by itself.
func (s *Service) ApplyCandidate(ctx context.Context, id, responseID string, acknowledgeQualityImpact bool) (Session, Resolution, error) {
	if !acknowledgeQualityImpact {
		return Session{}, Resolution{}, fmt.Errorf("acknowledge_quality_impact is required")
	}
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	for index := range session.Resolutions {
		resolution := &session.Resolutions[index]
		if resolution.ResponseID != responseID {
			continue
		}
		if resolution.State == ResolutionCandidateApplied {
			return session, *resolution, nil
		}
		if resolution.State != ResolutionCandidateReady || resolution.Candidate == nil {
			return Session{}, Resolution{}, fmt.Errorf("workshop candidate is %s", resolution.State)
		}
		if s.applyCandidate == nil {
			return Session{}, Resolution{}, fmt.Errorf("plan workshop candidate application is not configured")
		}
		version, _, hash, err := s.load(session.Subject)
		if err != nil {
			return Session{}, Resolution{}, err
		}
		if version != session.SubjectVersion || hash != session.PlanContentHash {
			resolution.State = ResolutionStale
			resolution.Error = "subject or canonical plan changed before candidate application"
			return s.replaceResolution(session, *resolution)
		}
		if err := s.applyCandidate(ctx, session, *resolution.Candidate, true); err != nil {
			return Session{}, Resolution{}, err
		}
		resolution.State, resolution.Error, resolution.AppliedAt = ResolutionCandidateApplied, "", s.now().UTC().Format(time.RFC3339Nano)
		return s.replaceResolution(session, *resolution)
	}
	return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q is missing", responseID)
}

// DiscardCandidate is the explicit "ignore" decision for a candidate. It
// changes only Plan Manager's candidate state; the canonical plan is untouched.
func (s *Service) DiscardCandidate(ctx context.Context, id, responseID, reason string) (Session, Resolution, error) {
	session, err := s.store.Load(id)
	if err != nil {
		return Session{}, Resolution{}, err
	}
	for index := range session.Resolutions {
		resolution := &session.Resolutions[index]
		if resolution.ResponseID != responseID {
			continue
		}
		if resolution.State == ResolutionCandidateDiscarded {
			return session, *resolution, nil
		}
		if resolution.State != ResolutionCandidateReady || resolution.Candidate == nil {
			return Session{}, Resolution{}, fmt.Errorf("workshop candidate is %s", resolution.State)
		}
		if s.discardCandidate == nil {
			return Session{}, Resolution{}, fmt.Errorf("plan workshop candidate discard is not configured")
		}
		if err := s.discardCandidate(ctx, session, *resolution.Candidate, strings.TrimSpace(reason)); err != nil {
			return Session{}, Resolution{}, err
		}
		resolution.State, resolution.Error, resolution.AppliedAt = ResolutionCandidateDiscarded, "", s.now().UTC().Format(time.RFC3339Nano)
		return s.replaceResolution(session, *resolution)
	}
	return Session{}, Resolution{}, fmt.Errorf("workshop resolution %q is missing", responseID)
}

func packetID(sessionID string, ordinal int) string {
	return fmt.Sprintf("%s_p%d", sessionID, ordinal)
}

func reviewPacketsEqual(a, b ReviewPacket) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func responseID(key string) string {
	raw, _ := json.Marshal(key)
	sum := sha256.Sum256(raw)
	return "r_" + hex.EncodeToString(sum[:8])
}

func reconciliationID(sessionID, responseID string) string {
	sum := sha256.Sum256([]byte(sessionID + "\x00" + responseID))
	return "reconcile_" + hex.EncodeToString(sum[:8])
}
