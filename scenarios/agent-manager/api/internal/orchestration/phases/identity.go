// Identity-token generation and revocation.
//
// Identity tokens give agent processes a way to authenticate back to
// agent-manager (or other Vrooli services) on behalf of the run. The
// token is HMAC-signed with a server-side secret and revoked on run
// completion (RevokeIdentityToken in result.go).
//
// Generation is non-fatal: on signing error the run proceeds without a
// token. The hash is persisted on the run record so verification can
// happen at the API boundary without storing the plaintext token.

package phases

import (
	"context"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/vrooli/api-core/scopecatalog"
)

// GenerateIdentityTokenInput is the explicit input to GenerateIdentityToken.
type GenerateIdentityTokenInput struct {
	Deps    Deps
	Run     *domain.Run
	Profile *domain.AgentProfile
	Task    *domain.Task
	Secret  []byte
	// Meta is server-provided execution context. It is included in signed claims
	// so downstream attribution never accepts caller-provided workflow identity.
	Meta map[string]string
	// AccountScopes are the verified owner grants. Nil means the caller did
	// not provide an account narrowing layer; the resulting token remains
	// explicit and can still be narrowed by profile/request scopes.
	AccountScopes   []string
	RequestedScopes []string
	// ConcreteScopes is the already-built governance catalog. Passing it lets
	// minting expand wildcard account grants and drop human-only commands
	// without making token generation perform filesystem I/O.
	ConcreteScopes []scopecatalog.Scope
}

// GenerateIdentityToken creates a signed identity token for this run and
// persists its hash. Returns the plaintext token (empty when generation
// is skipped). Non-fatal: on failure the run proceeds without a token.
func GenerateIdentityToken(ctx context.Context, in GenerateIdentityTokenInput) string {
	if len(in.Secret) == 0 {
		return ""
	}

	now := in.Deps.Now()
	profileKey := ""
	if in.Profile != nil {
		profileKey = in.Profile.ProfileKey
	}
	scopePath := ""
	if in.Task != nil {
		scopePath = in.Task.ScopePath
	}

	subject := strings.TrimSpace(in.Meta["subject"])
	if subject == "" && in.Run != nil {
		subject = strings.TrimSpace(in.Run.OwnerSubject)
	}
	if subject == "" && len(in.Run.Subject) > 0 {
		subject = strings.TrimSpace(in.Run.Subject[0])
	}
	accountScopes := in.AccountScopes
	if accountScopes == nil && in.Run != nil {
		accountScopes = in.Run.OwnerScopes
	}
	if len(in.ConcreteScopes) > 0 {
		accountScopes = scopecatalog.Materialize(in.AccountScopes, in.ConcreteScopes, false)
	}
	claims := &identity.Claims{
		RunID:      in.Run.ID,
		TaskID:     in.Run.TaskID,
		Subject:    subject,
		Scopes:     identity.IntersectScopes(accountScopes, profileScopes(in.Profile), in.RequestedScopes),
		ProfileKey: profileKey,
		ScopePath:  scopePath,
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(identity.DefaultTTL).Unix(),
		Meta:       cloneMeta(in.Meta),
	}

	token, err := identity.GenerateToken(claims, in.Secret)
	if err != nil {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
			"failed to generate identity token: "+err.Error())
		return ""
	}

	in.Run.IdentityTokenHash = identity.HashToken(token)

	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist identity token hash: "+err.Error())
		}
	}

	return token
}

func profileScopes(profile *domain.AgentProfile) []string {
	if profile == nil {
		return nil
	}
	return append([]string(nil), profile.DeclaredScopes...)
}

func cloneMeta(meta map[string]string) map[string]string {
	if len(meta) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(meta))
	for key, value := range meta {
		cloned[key] = value
	}
	return cloned
}
