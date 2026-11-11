// Package apiauth owns admission at public lighting API boundaries.
// Host setup retains its separate capability and OS permission checks.
package apiauth

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

func RequireCapability(ctx context.Context, capability string) error {
	principal, err := authn.RequireCapability(ctx, capability)
	if err != nil || !principal.IsVerified() || (!principal.ExpiresAt.IsZero() && !time.Now().Before(principal.ExpiresAt)) {
		if err == nil {
			err = authn.ErrCapability
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

// Ordinary lighting is available to current verified project agents without
// an operation grant or capability scope. The principal must agree with the
// verified Agent Manager request provenance; a source label alone is no proof.
// Human and service callers retain their existing standing capability policy.
func RequireWrite(ctx context.Context) error {
	p, _ := identity.PrincipalFromContext(ctx)
	if p.Kind != identity.ActorAgent {
		return RequireCapability(ctx, "lighting-control:write")
	}
	agent := provenance.FromContext(ctx)
	now := time.Now()
	if p.IsVerified() && p.Source == identity.SourceAgentProvenance && agent.IsVerifiedAgent() &&
		p.Subject == agent.RunID && p.Owner == agent.Subject &&
		!p.ExpiresAt.IsZero() && now.Before(p.ExpiresAt) &&
		!agent.ExpiresAt.IsZero() && now.Before(agent.ExpiresAt) {
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, authn.ErrCapability)
}
