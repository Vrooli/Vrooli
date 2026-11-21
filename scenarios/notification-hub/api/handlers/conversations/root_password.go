package conversations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
)

type rootPasswordAuthenticator interface {
	VerifyDecision(context.Context, string, string, string, string, string, string) (identity.DecisionProof, error)
}

// AnswerRootWithPassword authenticates freshly for this one answer. No token or
// standing permission is returned. The normal answer transaction compares the
// reviewed revision and makes concurrent/replayed answers fail.
func (h *handler) AnswerRootWithPassword(ctx context.Context, req *connect.Request[v1.AnswerRootWithPasswordRequest]) (*connect.Response[v1.AnswerRootWithPasswordResponse], error) {
	m := req.Msg
	answer := strings.TrimSpace(m.GetAnswer())
	if answer != "approve" && answer != "reject" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("choose approve or reject"))
	}
	record, err := h.service.GetAsk(ctx, m.GetAskId())
	if err != nil || !rootPasswordDecisionEligible(record, int(m.GetRevisionCount()), time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("decision unavailable or changed; reload it"))
	}
	if h.stepUp == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("sign-in unavailable"))
	}
	snapshot, err := h.service.RootEnrollmentSnapshot(ctx, record.Recipient)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("enrollment unavailable"))
	}
	challenge := m.GetMfaChallenge()
	digest := rootPasswordDigest(record.ID, int(m.GetRevisionCount()), answer)
	if challenge != "" {
		c, err := unwrapChallengeWithGrants(challenge, "root-answer:"+digest, record.Recipient)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
		}
		snapshot, challenge = c.Grants, c.Challenge
	}
	proof, err := h.stepUp.VerifyDecision(ctx, m.GetEmail(), m.GetPassword(), challenge, m.GetTotpCode(), m.GetRecoveryCode(), digest)
	if err != nil {
		if errors.Is(err, identity.ErrRateLimited) {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many sign-in attempts; try again later"))
		}
		if errors.Is(err, identity.ErrAuthUnavailable) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("sign-in unavailable; try again"))
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
	}
	if proof.MFARequired {
		challenge, err := wrapChallengeWithGrants(proof.MFAChallenge, "root-answer:"+digest, record.Recipient, int64(record.RevisionCount), snapshot)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("sign-in unavailable"))
		}
		return connect.NewResponse(&v1.AnswerRootWithPasswordResponse{MfaRequired: true, MfaChallenge: challenge}), nil
	}
	if strings.TrimSpace(proof.Subject) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
	}
	// Read the canonical deployment policy with the same parser and resource
	// filtering as other relying services. Never infer operator from an email.
	bindings, err := identity.OwnerBindings()
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("owner policy unavailable"))
	}
	direct := rootPasswordOwnerAllowed(bindings, proof.Subject, record.Recipient)
	owners := eligibleRootGrants(bindings, record.Recipient, snapshot)
	allowed := direct
	if !direct {
		allowed, err = h.service.RootEnrollmentActive(ctx, record.Recipient, proof.Subject, owners)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("enrollment unavailable"))
	}
	if !allowed {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
	}
	if !rootPasswordDecisionEligible(record, int(m.GetRevisionCount()), time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("decision unavailable or changed; reload it"))
	}
	revision := int(m.GetRevisionCount())
	if direct {
		owners = nil
	}
	result, err := h.service.AnswerAsk(ctx, hub.AnswerInput{RootEnrollmentOwners: owners, AskID: record.ID, Answer: answer, Actor: record.Recipient, ExpectedRevision: &revision, AnsweredBy: proof.Subject, AnsweredByIdentityKind: hub.AnswerIdentityKindHuman, AnsweredByIdentitySource: hub.AnswerIdentitySourcePassword})
	if errors.Is(err, hub.ErrEnrollmentChanged) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
	}
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.AnswerRootWithPasswordResponse{AnsweredAt: result.AnsweredAt}), nil
}

func rootPasswordDigest(askID string, revision int, answer string) string {
	data, _ := json.Marshal(struct {
		Domain   string `json:"domain"`
		AskID    string `json:"ask_id"`
		Revision int    `json:"revision"`
		Answer   string `json:"answer"`
	}{"notification-hub/root-rebaseline-answer/v1", askID, revision, answer})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func rootPasswordOwnerAllowed(bindings []authn.OwnerBinding, subject, recipient string) bool {
	for _, b := range bindings {
		if b.Source == apiidentity.SourceScenarioAuthenticator && b.Subject == subject && b.CanonicalSubject == recipient && scopecatalog.Resolve(b.Scopes, "root:rebaseline") {
			return true
		}
	}
	return false
}

func rootPasswordDecisionEligible(record hub.AskRecord, revision int, now time.Time) bool {
	deadline, err := time.Parse(time.RFC3339Nano, record.Deadline)
	return err == nil && now.Before(deadline) && record.Open() && record.RevisionCount == revision && record.Source == "root-rebaseline" && record.SourceEventType == "root.rebaseline.decision_requested.v1" && record.ProviderDecisionContext != nil && record.ProviderDecisionContext.SchemaVersion == 1 && record.ProviderDecisionContext.Scope == hub.ProviderDecisionScopeRootRebaseline
}

func cloudflareRootOwnerAllowed(bindings []authn.OwnerBinding, subject, recipient string) bool {
	if subject == "" || recipient == "" {
		return false
	}
	for _, b := range bindings {
		if b.Source == apiidentity.SourceCloudflareAccess && b.Subject == subject && b.CanonicalSubject == recipient && scopecatalog.Resolve(b.Scopes, "root:rebaseline") {
			return true
		}
	}
	return false
}

func (h *handler) checkRootPasswordAllowed(ctx context.Context, bindings []authn.OwnerBinding, subject, recipient string) (bool, error) {
	if rootPasswordOwnerAllowed(bindings, subject, recipient) {
		return true, nil
	}
	owners, err := h.service.RootEnrollmentOwners(ctx, recipient, subject)
	if err != nil {
		return false, err
	}
	for _, owner := range owners {
		if cloudflareRootOwnerAllowed(bindings, owner, recipient) {
			return true, nil
		}
	}
	return false, nil
}

func eligibleRootGrants(bindings []authn.OwnerBinding, recipient string, snapshot []hub.RootEnrollmentGrant) []hub.RootEnrollmentGrant {
	grants := []hub.RootEnrollmentGrant{}
	for _, grant := range snapshot {
		if cloudflareRootOwnerAllowed(bindings, grant.Owner, recipient) {
			grants = append(grants, grant)
		}
	}
	return grants
}
