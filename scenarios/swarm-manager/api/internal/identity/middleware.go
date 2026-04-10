package identity

import (
	"log/slog"
	"net/http"

	"github.com/vrooli/cli-core/cliutil"
)

const headerAgentIdentityToken = "X-Agent-Identity-Token"

// Verifier validates an agent identity token and returns verification results.
// Production code uses cliutil.IdentityEnv.VerifyIdentity(); tests use a stub.
type Verifier interface {
	Verify(token string) (*cliutil.VerifyResult, error)
}

// VerifierFunc adapts a plain function to the Verifier interface.
type VerifierFunc func(token string) (*cliutil.VerifyResult, error)

func (f VerifierFunc) Verify(token string) (*cliutil.VerifyResult, error) {
	return f(token)
}

// CLIUtilVerifier implements Verifier using cli-core's VerifyIdentity.
type CLIUtilVerifier struct{}

func (CLIUtilVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	env := cliutil.IdentityEnv{Token: token}
	return env.VerifyIdentity()
}

// Middleware extracts agent identity from the X-Agent-Identity-Token header,
// verifies it, and injects Provenance into the request context.
//
// Fail-open behavior: missing/invalid tokens result in operator provenance.
// Requests are never rejected due to identity issues.
func Middleware(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(headerAgentIdentityToken)
			if token == "" {
				ctx := NewContext(r.Context(), Provenance{Type: TypeOperator})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			result, err := verifier.Verify(token)
			if err != nil {
				slog.Warn("identity verification error, falling back to operator", "error", err)
				ctx := NewContext(r.Context(), Provenance{Type: TypeOperator})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if !result.Valid || result.Claims == nil {
				slog.Warn("identity token invalid, falling back to operator", "error", result.Error)
				ctx := NewContext(r.Context(), Provenance{Type: TypeOperator})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			p := Provenance{
				Type:       TypeAgent,
				RunID:      result.Claims.RunID,
				TaskID:     result.Claims.TaskID,
				ProfileKey: result.Claims.ProfileKey,
			}
			ctx := NewContext(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
