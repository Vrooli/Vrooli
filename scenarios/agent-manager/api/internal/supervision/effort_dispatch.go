package supervision

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/scopecatalog"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const SupervisorDispatchPurpose = "agent-manager:supervisor-dispatch:v1"
const SupervisorDispatchScope = "agent-manager:supervise"

var dispatchName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]{0,127}$`)
var ErrDispatchAuthority = errors.New("supervisor dispatch authorization unavailable, mismatched, revoked or expired")

// Issuance is operator-authenticated. Expose only the failed fixed predicate,
// never principal data, held scopes, signer material or provider errors. Keep
// bearer admission and child verification errors deliberately unqualified.
func dispatchIssuanceRefusal(predicate string) error {
	return fmt.Errorf("%w: issue-dispatch predicate=%s", ErrDispatchAuthority, predicate)
}

// ConfigureDispatch reuses the AM signer and canonical credential authority.
func (s *EffortService) ConfigureDispatch(secret []byte, provision func(string) error, profile func(context.Context, string) error) {
	s.dispatchSecret, s.dispatchProvision, s.dispatchProfile = slices.Clone(secret), provision, profile
}

func (s *EffortService) IssueDispatch(ctx context.Context, req *api.IssueSupervisorDispatchRequest, actor EffortActor) (*pb.EffortEnrollment, error) {
	if req == nil || !actor.Operator || actor.ID == "" || !dispatchName.MatchString(req.TeamId) || !dispatchName.MatchString(req.MemberId) || !dispatchName.MatchString(req.ProfileKey) || !dispatchName.MatchString(req.IdempotencyKey) || req.MaximumRuns < 1 || req.MaximumRuns > 1000 || req.MinimumIntervalSeconds < 60 || req.MinimumIntervalSeconds > 86400 || req.ExpiresAt == nil || !req.ExpiresAt.IsValid() {
		return nil, errors.New("owner issuance requires exact team/member/profile, key, 1–1000 runs, 60–86400 second interval and expiry")
	}
	if len(s.dispatchSecret) == 0 {
		return nil, dispatchIssuanceRefusal("signer_unavailable")
	}
	if s.dispatchProvision == nil {
		return nil, dispatchIssuanceRefusal("credential_provisioner_unavailable")
	}
	if s.dispatchProfile == nil {
		return nil, dispatchIssuanceRefusal("profile_verifier_unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, observation, err := s.repo.GetEffort(ctx, req.EffortRef)
	if err != nil {
		return nil, err
	}
	if e.Withdrawn {
		return nil, dispatchIssuanceRefusal("enrollment_withdrawn")
	}
	if e.AuthorizedBy != actor.ID {
		return nil, dispatchIssuanceRefusal("enrollment_owner_mismatch")
	}
	if e.SupervisorOwnerSubject != actor.ID {
		return nil, dispatchIssuanceRefusal("supervisor_owner_mismatch")
	}
	if e.SupervisorScope != SupervisorDispatchScope {
		return nil, dispatchIssuanceRefusal("enrollment_scope_mismatch")
	}
	if !scopecatalog.MatchCapability(actor.Scopes, e.SupervisorScope) {
		return nil, dispatchIssuanceRefusal("owner_scope_missing")
	}
	key, digest := "dispatch-issue:"+req.IdempotencyKey, effortDigest(req)+actor.ID
	replay := &pb.EffortEnrollment{}
	replayed, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay)
	if err != nil {
		return nil, err
	}
	if replayed {
		if replay.GetDispatchAuthorization().GetAuthorizationId() != e.GetDispatchAuthorization().GetAuthorizationId() {
			return nil, dispatchIssuanceRefusal("replay_authorization_superseded")
		}
		if err = s.validateDispatchEnrollment(e, e.GetDispatchAuthorization().GetAuthorizationId()); err != nil {
			return nil, dispatchIssuanceRefusal("replay_authorization_inactive")
		}
	} else {
		now := s.now().UTC()
		if e.Revision != req.ExpectedRevision {
			return nil, ErrConflict
		}
		if !req.ExpiresAt.AsTime().After(now) {
			return nil, dispatchIssuanceRefusal("expiry_not_future")
		}
		if req.ExpiresAt.AsTime().After(now.Add(30 * 24 * time.Hour)) {
			return nil, dispatchIssuanceRefusal("expiry_exceeds_maximum")
		}
		if e.AuthorityExpiresAt != nil && req.ExpiresAt.AsTime().After(e.AuthorityExpiresAt.AsTime()) {
			return nil, dispatchIssuanceRefusal("expiry_exceeds_enrollment")
		}
		if err = s.dispatchProfile(ctx, req.ProfileKey); err != nil {
			return nil, errors.New("supervisor profile is not available")
		}
		e.DispatchAuthorization = &pb.SupervisorDispatchAuthorization{AuthorizationId: uuid.NewString(), OwnerSubject: actor.ID, TeamId: req.TeamId, MemberId: req.MemberId, ProfileKey: req.ProfileKey, Scopes: []string{e.SupervisorScope}, IssuedAt: timestamppb.New(now), ExpiresAt: req.ExpiresAt, TargetRevision: e.TargetRevision, IssuanceKey: req.IdempotencyKey, MaximumRuns: req.MaximumRuns, MinimumIntervalSeconds: req.MinimumIntervalSeconds}
		token, mintErr := s.dispatchToken(e)
		if mintErr != nil {
			return nil, dispatchIssuanceRefusal("signing_failed")
		}
		e.DispatchAuthorization.CredentialHash = identity.HashToken(token)
		e.Revision++
		e.UpdatedAt = timestamppb.New(now)
		if err = s.repo.SaveEffort(ctx, e, observation, req.ExpectedRevision, key, digest); err != nil {
			return nil, err
		}
	}
	// Save authorization first: failure never exposes an unrecorded capability.
	// A retry reconstructs the same signed credential; no bearer is journaled.
	token, err := s.dispatchToken(e)
	if err != nil {
		return nil, dispatchIssuanceRefusal("signing_failed")
	}
	if err = s.dispatchProvision(token); err != nil {
		return nil, errors.New("dispatcher credential provisioning failed; retry the same issuance request after repairing the canonical authority")
	}
	return e, nil
}

func (s *EffortService) dispatchToken(e *pb.EffortEnrollment) (string, error) {
	a := e.GetDispatchAuthorization()
	return identity.GenerateToken(&identity.Claims{Purpose: SupervisorDispatchPurpose, DispatchEffortRef: e.EffortRef, DispatchAuthorizationID: a.AuthorizationId, Subject: a.OwnerSubject, IssuedAt: a.IssuedAt.AsTime().Unix(), ExpiresAt: a.ExpiresAt.AsTime().Unix(), Scopes: []string{}}, s.dispatchSecret)
}

func (s *EffortService) validateDispatchEnrollment(e *pb.EffortEnrollment, id string) error {
	a := e.GetDispatchAuthorization()
	if a == nil || id == "" || a.AuthorizationId != id || e.Withdrawn || a.RevokedAt != nil || a.OwnerSubject != e.AuthorizedBy || a.OwnerSubject != e.SupervisorOwnerSubject || a.TargetRevision != e.TargetRevision || len(a.Scopes) != 1 || a.Scopes[0] != e.SupervisorScope || a.ExpiresAt == nil || !a.ExpiresAt.IsValid() || !a.ExpiresAt.AsTime().After(s.now()) || len(a.CredentialHash) != 64 {
		return ErrDispatchAuthority
	}
	return nil
}

// CheckDispatchIdentity is also called by active child identity verification.
func (s *EffortService) CheckDispatchIdentity(ctx context.Context, claims *identity.Claims) error {
	if claims == nil || claims.DispatchEffortRef == "" || claims.DispatchAuthorizationID == "" {
		return ErrDispatchAuthority
	}
	e, _, err := s.repo.GetEffort(ctx, claims.DispatchEffortRef)
	if err != nil || s.validateDispatchEnrollment(e, claims.DispatchAuthorizationID) != nil {
		return ErrDispatchAuthority
	}
	a := e.DispatchAuthorization
	if claims.Subject != a.OwnerSubject || claims.ExpiresAt > a.ExpiresAt.AsTime().Unix() || claims.ProfileKey != a.ProfileKey {
		return ErrDispatchAuthority
	}
	for _, scope := range claims.Scopes {
		if !slices.Contains(a.Scopes, scope) {
			return ErrDispatchAuthority
		}
	}
	return nil
}

func (s *EffortService) AdmitDispatch(ctx context.Context, req *api.CreateSupervisorRunRequest, token string) (*pb.SupervisorDispatchAuthorization, error) {
	if req == nil || len(s.dispatchSecret) == 0 || !dispatchName.MatchString(req.IdempotencyKey) || len(req.WorkReferences) > 20 {
		return nil, ErrDispatchAuthority
	}
	if _, err := uuid.Parse(req.TaskId); err != nil {
		return nil, errors.New("supervisor task identity required")
	}
	claims, err := identity.VerifyToken(strings.TrimSpace(token), s.dispatchSecret)
	if err != nil || claims.Purpose != SupervisorDispatchPurpose || claims.DispatchEffortRef != req.EffortRef || claims.DispatchAuthorizationID != req.AuthorizationId || len(claims.Scopes) != 0 || claims.RunID != uuid.Nil {
		return nil, ErrDispatchAuthority
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, observation, err := s.repo.GetEffort(ctx, req.EffortRef)
	if err != nil || s.validateDispatchEnrollment(e, req.AuthorizationId) != nil {
		return nil, ErrDispatchAuthority
	}
	a := e.DispatchAuthorization
	if a.TeamId != req.TeamId || a.MemberId != req.MemberId || a.OwnerSubject != claims.Subject || subtle.ConstantTimeCompare([]byte(a.CredentialHash), []byte(identity.HashToken(token))) != 1 {
		return nil, ErrDispatchAuthority
	}
	key, digest := "dispatch-admit:"+a.AuthorizationId+":"+req.IdempotencyKey, effortDigest(req)
	replay := &pb.EffortEnrollment{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		if err != nil {
			return nil, err
		}
		return proto.Clone(a).(*pb.SupervisorDispatchAuthorization), nil
	}
	if a.DispatchedRuns >= a.MaximumRuns || (a.LastDispatchedAt != nil && s.now().Before(a.LastDispatchedAt.AsTime().Add(time.Duration(a.MinimumIntervalSeconds)*time.Second))) {
		return nil, errors.New("supervisor dispatch run allowance exhausted or minimum interval pending")
	}
	expected := e.Revision
	a.DispatchedRuns++
	a.LastDispatchedAt = timestamppb.New(s.now().UTC())
	e.Revision++
	e.UpdatedAt = a.LastDispatchedAt
	if err = s.repo.SaveEffort(ctx, e, observation, expected, key, digest); err != nil {
		return nil, err
	}
	return proto.Clone(a).(*pb.SupervisorDispatchAuthorization), nil
}

func (s *EffortService) RevokeDispatch(ctx context.Context, req *api.RevokeSupervisorDispatchRequest, actor EffortActor) (*pb.EffortEnrollment, error) {
	if req == nil || !actor.Operator || actor.ID == "" || !dispatchName.MatchString(req.IdempotencyKey) || strings.TrimSpace(req.Reason) == "" || len(req.Reason) > 512 {
		return nil, ErrDispatchAuthority
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, observation, err := s.repo.GetEffort(ctx, req.EffortRef)
	if err != nil {
		return nil, err
	}
	if e.AuthorizedBy != actor.ID || e.GetDispatchAuthorization().GetOwnerSubject() != actor.ID || e.GetDispatchAuthorization().GetAuthorizationId() != req.AuthorizationId {
		return nil, ErrDispatchAuthority
	}
	key, digest := "dispatch-revoke:"+req.IdempotencyKey, effortDigest(req)+actor.ID
	replay := &pb.EffortEnrollment{}
	if ok, err := s.repo.ReplayEffortOperation(ctx, key, digest, replay); ok || err != nil {
		return replay, err
	}
	if e.Revision != req.ExpectedRevision {
		return nil, ErrConflict
	}
	e.DispatchAuthorization.RevokedAt = timestamppb.New(s.now().UTC())
	e.Revision++
	e.UpdatedAt = e.DispatchAuthorization.RevokedAt
	if err = s.repo.SaveEffort(ctx, e, observation, req.ExpectedRevision, key, digest); err != nil {
		return nil, err
	}
	return e, nil
}
