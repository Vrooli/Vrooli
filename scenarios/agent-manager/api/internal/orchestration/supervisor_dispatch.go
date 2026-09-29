package orchestration

import (
	"context"
	"errors"
	"slices"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"github.com/google/uuid"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type SupervisorDispatchAuthority interface {
	AdmitDispatch(context.Context, *api.CreateSupervisorRunRequest, string) (*pb.SupervisorDispatchAuthorization, error)
	BindDispatchRun(context.Context, *api.CreateSupervisorRunRequest, string, uuid.UUID) error
	CheckDispatchIdentity(context.Context, *identity.Claims) error
}

func (o *Orchestrator) SetSupervisorDispatch(authority SupervisorDispatchAuthority) {
	o.supervisorDispatch = authority
}

// CreateSupervisorRun is the only purpose-bound dispatcher credential consumer.
// No caller-supplied config, owner, environment or profile becomes a grant.
func (o *Orchestrator) CreateSupervisorRun(ctx context.Context, req *api.CreateSupervisorRunRequest, token string) (*domain.Run, error) {
	if o.supervisorDispatch == nil || req == nil {
		return nil, errors.New("supervisor dispatch authority unavailable")
	}
	grant, err := o.supervisorDispatch.AdmitDispatch(ctx, req, token)
	if err != nil {
		return nil, err
	}
	profile, err := o.profiles.GetByKey(ctx, grant.ProfileKey)
	if err != nil || profile == nil {
		return nil, errors.New("granted supervisor profile unavailable")
	}
	if !slices.Equal(identity.IntersectScopes(grant.Scopes, profile.IdentityScopeCeiling(), grant.Scopes), grant.Scopes) {
		return nil, errors.New("granted supervisor profile no longer permits its scope; operator must qualify the profile")
	}
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, errors.New("supervisor task identity invalid")
	}
	expires := grant.ExpiresAt.AsTime()
	create := CreateRunRequest{
		TaskID: taskID, AgentProfileID: &profile.ID,
		Tag: "supervision-" + req.IdempotencyKey, IdempotencyKey: "supervisor-dispatch:" + grant.AuthorizationId + ":" + req.IdempotencyKey,
		OwnerSubject: grant.OwnerSubject, OwnerScopes: slices.Clone(grant.Scopes), RequestedScopes: slices.Clone(grant.Scopes), OwnerExpiresAt: &expires,
		DispatchBinding: &domain.DispatchBinding{EffortRef: req.EffortRef, AuthorizationID: grant.AuthorizationId}, WorkReferences: req.WorkReferences,
		WorkloadKind: domain.WorkloadKindScheduled, WorkloadKey: req.EffortRef, WorkloadInstance: req.IdempotencyKey,
	}
	// The admission is durable beyond the ordinary one-hour idempotency
	// cache. Reconcile the original run after checking live authorization.
	if existing, err := o.runs.GetByIdempotencyKey(ctx, create.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		if err := o.supervisorDispatch.BindDispatchRun(ctx, req, token, existing.ID); err != nil {
			return nil, err
		}
		return o.getIdentityBoundRunReplay(ctx, existing.ID, create)
	}
	run, err := o.CreateRun(ctx, create)
	if err != nil {
		return nil, err
	}
	if err := o.supervisorDispatch.BindDispatchRun(ctx, req, token, run.ID); err != nil {
		// Admission is durable, so a run that cannot be attached to its owner
		// accounting identity must not continue outside that boundary.
		_ = o.StopRun(ctx, run.ID)
		return nil, err
	}
	return run, nil
}

func (o *Orchestrator) checkSupervisorBinding(ctx context.Context, run *domain.Run, claims *identity.Claims) error {
	if run.DispatchBinding == nil {
		if claims.DispatchAuthorizationID != "" || claims.DispatchEffortRef != "" {
			return errors.New("unexpected supervisor authorization binding")
		}
		return nil
	}
	if o.supervisorDispatch == nil || run.DispatchBinding.AuthorizationID != claims.DispatchAuthorizationID || run.DispatchBinding.EffortRef != claims.DispatchEffortRef {
		return errors.New("supervisor authorization binding unavailable")
	}
	return o.supervisorDispatch.CheckDispatchIdentity(ctx, claims)
}
