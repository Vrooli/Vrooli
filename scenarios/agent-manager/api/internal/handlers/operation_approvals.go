package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type operationSettingsReader interface {
	GetOrchestrationSettings(context.Context) (*config.OrchestrationSettings, error)
	GetRun(context.Context, uuid.UUID) (*domain.Run, error)
	GetTask(context.Context, uuid.UUID) (*domain.Task, error)
}

type configuredOperationAuthority struct {
	base     context.Context
	settings config.OperationApprovalSettings
	reader   operationSettingsReader
}

// AgentOperationPolicy marks this as the ordinary agent policy, not an exact
// signed-action authority. Only trusted ingress installs this authority.
func (configuredOperationAuthority) AgentOperationPolicy() {}

func operationPolicyAllowsWithoutGrant(ctx context.Context, operation, target string) bool {
	authority, ok := authn.RequestOperationAuthorityFromContext(ctx)
	a, ok := authority.(configuredOperationAuthority)
	if !ok {
		return false
	}
	required, known := a.settings.RequiresApproval(operation)
	if !known || required {
		return false
	}
	_, err := authn.RequireOperation(ctx, "agent-manager:write", operation, target)
	return err == nil
}

// ownerScopedStopAll admits a bulk request only with an explicit owner filter;
// it never turns owner-scoped policy into wildcard authority over other runs.
func ownerScopedStopAll(ctx context.Context) (string, bool) {
	authority, ok := authn.RequestOperationAuthorityFromContext(ctx)
	a, ok := authority.(configuredOperationAuthority)
	if !ok || a.settings.RunManagement {
		return "", false
	}
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || p.Kind != identity.ActorAgent || !p.IsVerified() || strings.TrimSpace(p.Owner) == "" || (!p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt)) {
		return "", false
	}
	if _, err := authn.RequireCapability(ctx, "agent-manager:write"); err != nil {
		return "", false
	}
	if _, err := authn.RequireOperation(a.base, "agent-manager:write", "agent.manage", "agent-manager:run/*"); err == nil {
		return "", false
	}
	return p.Owner, true
}

func (a configuredOperationAuthority) AuthorizeOperation(_ context.Context, p identity.Principal, capability, operation, target string) error {
	required, known := a.settings.RequiresApproval(operation)
	if known && !required && p.Kind == identity.ActorAgent && p.IsVerified() &&
		(p.ExpiresAt.IsZero() || time.Now().Before(p.ExpiresAt)) &&
		a.ownsTarget(p, operation, target) &&
		capability == "agent-manager:write" {
		_, err := authn.RequireCapability(a.base, capability)
		return err
	}
	_, err := authn.RequireOperation(a.base, capability, operation, target)
	return err
}

func (a configuredOperationAuthority) ownsTarget(p identity.Principal, operation, target string) bool {
	if strings.TrimSpace(p.Owner) == "" {
		return false
	}
	if operation == "run.create" {
		return target == RunCreationTarget
	}
	if operation == "effort.delivery" || operation == "effort.reconcile" {
		return strings.HasPrefix(target, "agent-manager:effort/") // Effort owner is checked by the service.
	}
	if strings.HasPrefix(target, "agent-manager:task/") {
		id, err := uuid.Parse(strings.TrimPrefix(target, "agent-manager:task/"))
		if err != nil {
			return false
		}
		task, err := a.reader.GetTask(a.base, id)
		if err != nil || task == nil {
			return false
		}
		creator, err := uuid.Parse(task.CreatedBy)
		if err != nil {
			return false
		}
		run, err := a.reader.GetRun(a.base, creator)
		return err == nil && run != nil && run.OwnerSubject == p.Owner
	}
	id, err := uuid.Parse(strings.TrimPrefix(target, "agent-manager:run/"))
	if err != nil || !strings.HasPrefix(target, "agent-manager:run/") {
		return false
	}
	run, err := a.reader.GetRun(a.base, id)
	return err == nil && run != nil && run.OwnerSubject == p.Owner
}

// OperationApprovalMiddleware applies the operator's policy to verified agents.
// Human decisions and service standing grants retain their separate authorities.
func OperationApprovalMiddleware(settings operationSettingsReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := identity.PrincipalFromContext(r.Context())
			if settings != nil && ok && p.Kind == identity.ActorAgent && p.IsVerified() {
				current, err := settings.GetOrchestrationSettings(r.Context())
				if err == nil && current != nil {
					a := configuredOperationAuthority{base: r.Context(), settings: current.OperationApprovals, reader: settings}
					next.ServeHTTP(w, r.WithContext(authn.WithRequestOperationAuthority(r.Context(), a)))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
