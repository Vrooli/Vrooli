// Request-local verified human proof for agent run creation; never durable credentials.
package agentmanager

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"strings"
	"time"
)

type createCallerKey struct{}
type createCaller struct {
	token      string
	principal  identity.Principal
	runOffered bool
}

var ErrCreateCaller = errors.New("verified human agent-manager:write caller required")

// WithVerifiedCreateCaller retains a proof already checked by Portal's account
// authenticator. It does not verify tokens or turn arbitrary input into authority.
func WithVerifiedCreateCaller(ctx context.Context, token string, principal identity.Principal, runOffered bool) context.Context {
	principal.Scopes = append([]string(nil), principal.Scopes...)
	return context.WithValue(ctx, createCallerKey{}, createCaller{token, principal, runOffered})
}

// RequireCreateCaller runs before Portal admission, uploads, or owner writes.
// Run-token delegation is held until this adapter has a qualified exact-parent route.
func RequireCreateCaller(ctx context.Context) error {
	p, ok := ctx.Value(createCallerKey{}).(createCaller)
	if !ok || p.runOffered || !p.principal.IsHuman() || !p.principal.ExpiresAt.After(time.Now()) || p.token == "" || strings.ContainsAny(p.token, " \t\r\n") || ctx.Err() != nil {
		return ErrCreateCaller
	}
	for _, scope := range p.principal.Scopes {
		if scopecatalog.MatchCapability([]string{scope}, "agent-manager:write") {
			return nil
		}
	}
	return ErrCreateCaller
}
func createCallerAuthorization(ctx context.Context) string {
	p, _ := ctx.Value(createCallerKey{}).(createCaller)
	return "Bearer " + p.token
}
func redactCreateCaller(ctx context.Context, text string) string {
	p, _ := ctx.Value(createCallerKey{}).(createCaller)
	if p.token != "" {
		text = strings.ReplaceAll(text, p.token, "[redacted]")
	}
	return text
}
