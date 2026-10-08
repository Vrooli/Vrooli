// This file verifies public run-creation callers and plans read-only profile admission.
package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/scopecatalog"
)

// authenticateCreateRunCaller accepts no body-provided identity projection.
// Every offered channel must validate; a failed channel never becomes absence.
func (o *Orchestrator) authenticateCreateRunCaller(ctx context.Context, req *CreateRunRequest) error {
	refuse := func(message string) error {
		return domain.NewValidationErrorWithCode("authorization", message, domain.ErrCodePolicyScope)
	}
	if req.EffortProof != "" {
		return o.authenticateEffortCaller(ctx, req)
	}
	ownerOffered := req.OwnerToken != ""
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
		if parent.ResolvedConfig != nil && parent.ResolvedConfig.Admission != nil && parent.ResolvedConfig.Admission.Effort != nil {
			if ownerOffered {
				return refuse("effort child cannot replace its retained authority with a human session")
			}
			if err := o.inheritEffortCaller(ctx, req, parent); err != nil {
				return err
			}
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
		bindEffortCaller(req)
	}
	req.RequestedScopes, req.ExpectedOwnerSubject = narrowing, expected
	return narrowCreateRunScopes(req)
}

// revalidateCreateRunCaller closes validation-time credential changes before
// the first admission effect. It does not renew the retained scope or expiry.
func (o *Orchestrator) revalidateCreateRunCaller(ctx context.Context, req CreateRunRequest) error {
	if req.effort != nil {
		if _, err := o.effortAuthority.CheckBinding(ctx, req.effort.binding); err != nil {
			return err
		}
		if req.effort.serial {
			return o.revalidateSerialCaller(ctx, req)
		}
		if req.effort.recovery {
			return nil
		}
	}
	if req.caller == nil {
		return nil
	}
	fresh := req
	fresh.caller = nil
	fresh.OwnerSubject, fresh.OwnerScopes, fresh.OwnerExpiresAt = "", nil, nil
	if err := o.authenticateCreateRunCaller(ctx, &fresh); err != nil {
		return err
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
