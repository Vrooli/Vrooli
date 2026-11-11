// Package apiauth owns capability admission at public lighting API boundaries.
// Internal policy evaluation and domain calls retain their existing owner paths.
package apiauth

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
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
func RequireWrite(ctx context.Context) error { return RequireCapability(ctx, "lighting-control:write") }
