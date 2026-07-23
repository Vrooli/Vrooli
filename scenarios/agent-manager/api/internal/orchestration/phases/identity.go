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

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
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

	claims := &identity.Claims{
		RunID:      in.Run.ID,
		TaskID:     in.Run.TaskID,
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
