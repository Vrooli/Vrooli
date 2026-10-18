// This file verifies public run-creation callers and plans read-only profile admission.
package orchestration

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
)

// authenticateCreateRunCaller accepts no body-provided identity projection.
// Every offered channel must validate; a failed channel never becomes absence.
func (o *Orchestrator) authenticateCreateRunCaller(ctx context.Context, req *CreateRunRequest) error {
	refuse := func(message string) error {
		return domain.NewValidationErrorWithCode("authorization", message, domain.ErrCodePolicyScope)
	}
	verifiedOwner, hasVerifiedOwner := coreidentity.PrincipalFromContext(ctx)
	// A verified service principal carries its identity in the Authorization
	// bearer that the authn middleware already verified into this principal. That
	// bearer is NOT a human owner token, so it must not enter owner verification;
	// a service caller is authorized solely by its operator-approved standing
	// grant. Recognize it before owner/run routing and attribute the run to the
	// grant's approver — never the service, never the anonymous path.
	serviceCaller := hasVerifiedOwner && verifiedOwner.Kind == coreidentity.ActorService && verifiedOwner.IsVerified()
	if serviceCaller && req.RunIdentityToken == "" {
		owner, err := o.standingGrantOwner(ctx, verifiedOwner.Subject)
		if err != nil {
			return err
		}
		if owner != "" {
			req.OwnerToken = ""
			req.OwnerSubject = owner
			req.caller = &domain.CreateRunCaller{Kind: "service-standing", Subject: owner}
			return narrowCreateRunScopes(req)
		}
		// A verified service with no covering standing grant has no run-creation
		// authority. The middleware already refuses this, so reaching here means
		// the grant was revoked between the gate and admission: refuse, never fall
		// back to the anonymous path on the strength of a service bearer.
		return refuse("service caller has no standing grant for run creation")
	}
	// A top-level run-creation grant is a separate proof from the baseline run
	// identity. Attribute the new run to the grant's verified approver and use
	// only the approved run-creation scopes as its ceiling; never promote or union
	// baseline scopes. Exact-parent child creation keeps its separate path below.
	if !serviceCaller && req.OwnerToken == "" && req.ParentRunID == nil {
		handled, err := o.authenticateGrantedRunCreate(ctx, req)
		if handled || err != nil {
			return err
		}
	}
	ownerOffered := req.OwnerToken != "" || (hasVerifiedOwner && verifiedOwner.IsHuman())
	runOffered := req.RunIdentityToken != ""
	if !ownerOffered && !runOffered {
		// With AUTH-01 off (P-18) an unattended caller keeps the pre-AUTH-01
		// anonymous path; createRun resolves it without a caller record.
		if !owneridentity.CreateRunCallerEnforced() {
			return nil
		}
		return refuse("create-run requires verified caller identity")
	}
	if (ownerOffered && strings.TrimSpace(req.OwnerToken) == "") || (runOffered && strings.TrimSpace(req.RunIdentityToken) == "") {
		return refuse("offered caller identity is blank")
	}
	// Owner verification does not get to narrow until parent proof has also
	// passed. Keep the actual caller narrowing for the final intersection.
	narrowing, expected := req.RequestedScopes, req.ExpectedOwnerSubject
	req.RequestedScopes, req.ExpectedOwnerSubject = nil, ""
	if ownerOffered {
		if err := o.resolveCreateRunIdentity(ctx, req); err != nil {
			return err
		}
		// Use the scenario's declared existing human write capability. Being a
		// verified human alone is not permission to reserve or dispatch a run.
		if !scopecatalog.MatchCapability(req.OwnerScopes, "agent-manager:write") {
			return refuse("verified owner lacks agent-manager:write authority")
		}
		req.caller = &domain.CreateRunCaller{Kind: "human", Subject: req.OwnerSubject}
	}
	if runOffered {
		if o.runs == nil || len(o.identitySecret) == 0 {
			return refuse("run identity verifier unavailable")
		}
		verified, err := o.VerifyIdentityToken(ctx, strings.TrimSpace(req.RunIdentityToken))
		if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
			return refuse("run identity is invalid, expired, revoked or unavailable")
		}
		claims := verified.Claims
		if claims.RunID == uuid.Nil || req.ParentRunID == nil || *req.ParentRunID != claims.RunID {
			return refuse("run caller may create only its authenticated exact-parent child")
		}
		parent, err := o.runs.Get(ctx, claims.RunID)
		if err != nil || parent == nil || parent.TaskID != claims.TaskID || parent.OwnerSubject != strings.TrimSpace(claims.Subject) {
			return refuse("run claims do not match the retained parent identity")
		}
		expires := time.Unix(claims.ExpiresAt, 0)
		if !expires.After(o.now()) {
			return refuse("run identity is expired")
		}
		if ownerOffered && req.OwnerSubject != strings.TrimSpace(claims.Subject) {
			return refuse("owner and run identities do not name the same owner")
		}
		parentScopes := append([]string{}, claims.Scopes...)
		if ownerOffered {
			req.OwnerScopes = identity.IntersectScopes(req.OwnerScopes, parentScopes, nil)
			if req.OwnerExpiresAt.Before(expires) {
				expires = *req.OwnerExpiresAt
			}
		} else {
			req.OwnerSubject = strings.TrimSpace(claims.Subject)
			req.OwnerScopes = parentScopes
		}
		req.OwnerExpiresAt = &expires
		req.caller = &domain.CreateRunCaller{Kind: "run", Subject: req.OwnerSubject, RunID: claims.RunID}
	}
	req.RequestedScopes, req.ExpectedOwnerSubject = narrowing, expected
	return narrowCreateRunScopes(req)
}

// authenticateGrantedRunCreate recognizes a top-level run.create authorization
// carried by a verified legacy grant token or a verified baseline/operation
// pair. It relies on the canonical operation evaluator for rule/target matching
// and derives the new run's ceiling only from the reviewed run-creation scopes.
func (o *Orchestrator) authenticateGrantedRunCreate(ctx context.Context, req *CreateRunRequest) (bool, error) {
	principal, ok := coreidentity.PrincipalFromContext(ctx)
	if !ok || principal.Kind != coreidentity.ActorAgent || !principal.IsVerified() || strings.TrimSpace(req.RunIdentityToken) == "" {
		return false, nil
	}
	if provenance.ForwardedIdentityToken(ctx) != strings.TrimSpace(req.RunIdentityToken) {
		return false, nil
	}
	if _, err := authn.RequireOperation(ctx, "agent-manager:write", "run.create", standingGrantRunCreateTarget); err != nil {
		return false, nil
	}

	var grant *coreidentity.OperationGrant
	var grantRunID uuid.UUID
	var grantScopes []string
	var grantExpires time.Time
	proof, proofErr := provenance.ReverifyOperationProof(ctx)
	if proofErr != nil {
		return true, domain.NewValidationErrorWithCode("authorization", "run-creation grant is unavailable", domain.ErrCodePolicyScope)
	}
	if proof != nil {
		grant = proof.Grant
		grantRunID, _ = uuid.Parse(strings.TrimSpace(proof.RunID))
		grantScopes = slices.Clone(proof.Scopes)
		grantExpires = proof.ExpiresAt
		if proof.GrantID == "" || grant == nil || proof.GrantID != grant.ID {
			return true, domain.NewValidationErrorWithCode("authorization", "run-creation grant is invalid", domain.ErrCodePolicyScope)
		}
	} else {
		// Grant-only identities retain their existing route. Reverify the exact
		// offered token here and during caller revalidation so revocation or
		// replacement cannot survive between admission and dispatch.
		verified, err := o.VerifyIdentityToken(ctx, strings.TrimSpace(req.RunIdentityToken))
		if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
			return true, domain.NewValidationErrorWithCode("authorization", "run-creation identity is invalid or unavailable", domain.ErrCodePolicyScope)
		}
		grantRunID = verified.Claims.RunID
		grant = provenance.OperationGrantFromMeta(verified.Claims.Meta)
		grantScopes = slices.Clone(verified.Claims.Scopes)
		grantExpires = time.Unix(verified.Claims.ExpiresAt, 0).UTC()
		if verified.Claims.Meta["authorization_id"] == "" || principal.Grant == nil || grant == nil || principal.Grant.ID != grant.ID {
			return true, domain.NewValidationErrorWithCode("authorization", "run-creation grant is invalid", domain.ErrCodePolicyScope)
		}
	}
	if grantRunID == uuid.Nil || principal.Owner == "" || principal.Owner != grant.Approver ||
		!grant.Allows("run.create", standingGrantRunCreateTarget, o.now()) {
		return true, domain.NewValidationErrorWithCode("authorization", "run.create grant does not match the approved operation", domain.ErrCodePolicyScope)
	}
	runCreateScopes, ok := approvedRunCreationScopes(grantScopes)
	if !ok {
		return true, domain.NewValidationErrorWithCode("authorization", "run.create grant has no scoped run-creation ceiling", domain.ErrCodePolicyScope)
	}
	if o.runs == nil {
		return true, domain.NewConfigMissingError("runs", "repository unavailable", nil)
	}
	grantRun, err := o.runs.Get(ctx, grantRunID)
	if err != nil {
		return true, err
	}
	if grantRun == nil || grantRun.Status.IsTerminal() || grantRun.IdentityTokenRevokedAt != nil ||
		(grantRun.OwnerSubject != "" && grantRun.OwnerSubject != grant.Approver) {
		return true, domain.NewValidationErrorWithCode("authorization", "run-creation owner record is unavailable", domain.ErrCodePolicyScope)
	}
	if !grantExpires.After(o.now()) {
		return true, domain.NewValidationErrorWithCode("authorization", "run-creation grant expired", domain.ErrCodePolicyScope)
	}
	if grantRun.OwnerExpiresAt != nil {
		if !grantRun.OwnerExpiresAt.After(o.now()) {
			return true, domain.NewValidationErrorWithCode("authorization", "run-creation owner authority expired", domain.ErrCodePolicyScope)
		}
		if grantRun.OwnerExpiresAt.Before(grantExpires) {
			grantExpires = *grantRun.OwnerExpiresAt
		}
	}
	req.OwnerSubject = grant.Approver
	// Multi-policy grant scopes are not unioned into the new identity.
	req.OwnerScopes = runCreateScopes
	req.OwnerExpiresAt = &grantExpires
	req.caller = &domain.CreateRunCaller{Kind: "operation-grant", Subject: grant.Approver, RunID: grantRunID, GrantID: grant.ID}
	return true, narrowCreateRunScopes(req)
}

// approvedRunCreationScopes constrains the new run to the reviewed capability
// ceiling for run creation. RequireOperation already evaluated the exact
// operation/target rule; this never unions unrelated token scopes into the run.
func approvedRunCreationScopes(tokenScopes []string) ([]string, bool) {
	policy, ok := authn.InteractivePolicy("run-creation")
	if !ok {
		return nil, false
	}
	scopes := identity.IntersectScopes(tokenScopes, policy.Scopes, nil)
	return scopes, scopecatalog.Resolve(scopes, "agent-manager:write")
}

// revalidateCreateRunCaller closes validation-time credential changes before
// the first admission effect. It does not renew the retained scope or expiry.
func (o *Orchestrator) revalidateCreateRunCaller(ctx context.Context, req CreateRunRequest) error {
	if req.caller == nil {
		return nil
	}
	fresh := req
	fresh.caller = nil
	fresh.OwnerSubject, fresh.OwnerScopes, fresh.OwnerExpiresAt = "", nil, nil
	if err := o.authenticateCreateRunCaller(ctx, &fresh); err != nil {
		return err
	}
	// A service-standing caller carries no owner token or expiry — its authority
	// is the durable standing grant, which re-authentication above just
	// re-resolved against the live store (approved, not revoked, not expired, and
	// still covering run.create for this owner). So the anti-escalation guarantee
	// holds: a grant revoked, narrowed, or expired between admit and here yields a
	// different (or empty) owner and is refused. We therefore compare caller
	// identity + attributed owner, but NOT the human/run OwnerExpiresAt ceiling
	// (which is intentionally absent for this kind), and never treat a re-minted
	// service bearer with the same underlying grant as a change.
	if req.caller.Kind == "service-standing" {
		if fresh.caller == nil || *fresh.caller != *req.caller || fresh.OwnerSubject != req.OwnerSubject {
			return domain.NewValidationErrorWithCode("authorization", "caller authority changed during admission", domain.ErrCodePolicyScope)
		}
		return nil
	}
	if fresh.caller == nil || *fresh.caller != *req.caller || fresh.OwnerSubject != req.OwnerSubject || !sameCreateRunScopeCeiling(fresh.OwnerScopes, req.OwnerScopes) || fresh.OwnerExpiresAt == nil || req.OwnerExpiresAt == nil || fresh.OwnerExpiresAt.Before(*req.OwnerExpiresAt) {
		return domain.NewValidationErrorWithCode("authorization", "caller authority changed during admission", domain.ErrCodePolicyScope)
	}
	return nil
}

// Public CreateRun can select an existing profile but cannot reconcile profile
// state before admission. Profile mutation stays with its existing owner route.
// This explicit compatibility refusal adds no credential/grant or bypass.
type profileAdmissionPlan struct{ profile *domain.AgentProfile }

func (o *Orchestrator) planProfileAdmission(ctx context.Context, ref *ProfileRef) (*profileAdmissionPlan, error) {
	if o.profiles == nil {
		return nil, domain.NewConfigMissingError("profiles", "repository unavailable", nil)
	}
	key := strings.TrimSpace(ref.ProfileKey)
	if key == "" {
		return nil, domain.NewValidationError("profileRef.profileKey", "field is required")
	}
	existing, err := o.profiles.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	refuse := func() (*profileAdmissionPlan, error) {
		return nil, domain.NewValidationErrorWithHint("profileRef", "CreateRun requires an existing profile and cannot mutate its owner state", "reconcile the profile through its existing authorized owner route before requesting a run")
	}
	if existing == nil {
		return refuse()
	}
	if ref.UpdateExisting {
		if ref.Defaults == nil {
			return refuse()
		}
		// An already reconciled identical declaration is a no-op, preserving
		// clients that supply authoritative defaults without changing state.
		data, err := json.Marshal(ref.Defaults)
		if err != nil {
			return nil, err
		}
		var candidate domain.AgentProfile
		if err := json.Unmarshal(data, &candidate); err != nil {
			return nil, err
		}
		candidate.ID, candidate.ProfileKey = existing.ID, key
		candidate.CreatedAt, candidate.UpdatedAt = existing.CreatedAt, existing.UpdatedAt
		if candidate.CreatedBy == "" {
			candidate.CreatedBy = existing.CreatedBy
		}
		if strings.TrimSpace(candidate.Name) == "" {
			candidate.Name = key
		}
		if err := normalizeProfileInput(&candidate); err != nil {
			return nil, err
		}
		before, err := json.Marshal(existing)
		if err != nil {
			return nil, err
		}
		after, err := json.Marshal(&candidate)
		if err != nil {
			return nil, err
		}
		if string(before) != string(after) {
			return refuse()
		}
	}
	// Resolver uses the inspected snapshot. No profile write can race the
	// qualification decision or overwrite another owner's concurrent work.
	return &profileAdmissionPlan{profile: existing}, nil
}
