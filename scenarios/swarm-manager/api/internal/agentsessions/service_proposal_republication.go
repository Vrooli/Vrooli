package agentsessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"swarm-manager/internal/apierr"
)

// proposalDecisionTerms is the immutable proposal content an operator sees
// and may authorize. Lifecycle state, validation diagnostics, decisions,
// UpdatedAt, and publication receipts are intentionally excluded: they can
// change while the same proposal remains ready or as an answer is committed.
// Readiness and no-prior-decision checks are enforced separately under the
// store lock before generation creation and again by the mutation authority.
type proposalDecisionTerms struct {
	ID          string          `json:"id"`
	Kind        ProposalKind    `json:"kind"`
	Summary     string          `json:"summary"`
	PayloadJSON string          `json:"payload_json"`
	CreatedAt   string          `json:"created_at"`
	Attribution *Attribution    `json:"attribution,omitempty"`
	Target      *ProposalTarget `json:"target,omitempty"`
}

func proposalDecisionAuthorizationTarget(sessionID, proposalID string) string {
	return "swarm-manager:proposal/" + strings.TrimSpace(sessionID) + "/" + strings.TrimSpace(proposalID)
}

func proposalDecisionTermsSHA256(proposal Proposal) (string, error) {
	terms := proposalDecisionTerms{
		ID: proposal.ID, Kind: proposal.Kind, Summary: proposal.Summary,
		PayloadJSON: proposal.PayloadJSON, CreatedAt: proposal.CreatedAt,
		Attribution: proposal.Attribution, Target: proposal.Target,
	}
	encoded, err := json.Marshal(terms)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// ReissueLegacyProposalDecision explicitly freezes one new decision request for
// exact current terms. The legacy marker remains preserved; this is not a retry
// of a guessed historical event and does not withdraw an unknown old Ask.
func (s *Service) ReissueLegacyProposalDecision(ctx context.Context, sessionID, proposalID, expectedTermsSHA256 string) (Session, error) {
	expectedTermsSHA256 = strings.ToLower(strings.TrimSpace(expectedTermsSHA256))
	if len(expectedTermsSHA256) != 64 {
		return Session{}, apierr.BadRequest("expected_terms_sha256 must be a 64-character SHA-256 digest")
	}
	if _, err := hex.DecodeString(expectedTermsSHA256); err != nil {
		return Session{}, apierr.BadRequest("expected_terms_sha256 must be hexadecimal")
	}

	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok {
		return Session{}, apierr.NotFound("proposal not found")
	}
	if !eligibleProposalDecision(proposal) || len(proposal.Decisions) != 0 || proposal.DecisionPublication == nil || proposal.DecisionPublication.State != ProposalDecisionPublicationLegacyUnresolved || proposal.PreviousDecisionPublication != nil {
		return Session{}, apierr.Conflict("proposal is not an unchanged unresolved legacy decision")
	}
	currentDigest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		return Session{}, err
	}
	if currentDigest != expectedTermsSHA256 {
		return Session{}, apierr.Conflict("proposal terms changed; read the current proposal and confirm its digest")
	}
	publication, err := s.newLegacyProposalDecisionPublication(ctx, session.ID, proposal)
	if err != nil {
		return Session{}, err
	}
	publication.DecisionTermsSHA256 = currentDigest
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, err
	}
	owner, ok := store.(LegacyProposalDecisionGenerationStore)
	if !ok {
		return Session{}, apierr.Unavailable("proposal owner cannot freeze a new legacy decision generation")
	}
	changed, err := owner.BeginLegacyProposalDecisionGeneration(session.ID, proposal.ID, currentDigest, *publication)
	if err != nil {
		return Session{}, err
	}
	if !changed {
		return Session{}, apierr.Conflict("proposal changed or is no longer eligible for a new decision request")
	}
	proposal.DecisionPublication = publication
	// The immutable publication EventID and terms digest form a monotonic fence:
	// legacy/unknown rows have no acceptable requester binding, and a frozen
	// publication cannot be replaced with a different known generation.
	s.emitProposalDecisionRequested(ctx, session.ID, proposal)
	return s.loadSession(ctx, session.ID)
}

// matchesCurrentDecisionPublication makes the Events requester_ref (the
// original request EventID) and the immutable proposal terms digest the fence
// for every answerable generation. Unknown legacy publications are fail-closed;
// an operator must inspect current terms and explicitly reissue them first.
func (s *Service) matchesCurrentDecisionPublication(ctx context.Context, sessionID, proposalID, requesterRef string) (bool, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return false, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok {
		return false, nil
	}
	publication := proposal.DecisionPublication
	if publication == nil || publication.State == ProposalDecisionPublicationLegacyUnresolved ||
		strings.TrimSpace(publication.EventID) == "" || strings.TrimSpace(requesterRef) == "" || requesterRef != publication.EventID {
		return false, nil
	}
	if publication.DecisionTermsSHA256 == "" {
		return false, nil
	}
	digest, err := proposalDecisionTermsSHA256(proposal)
	if err != nil {
		return false, err
	}
	return digest == publication.DecisionTermsSHA256, nil
}
