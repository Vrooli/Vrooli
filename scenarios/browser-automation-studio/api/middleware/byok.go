package middleware

import (
	"context"
	"net/http"
	"strings"
)

const (
	// BYOKHeaderName is a legacy source-selection hint. Its value is discarded.
	BYOKHeaderName = "X-BYOK-OpenRouter-Key"
)

// byokContextKey is a route-selection hint only. Raw API keys never enter
// request context or an AI Gateway call.
type byokContextKey struct{}

// BYOKMiddleware maps a legacy source-selection hint to a boolean context value.
type BYOKMiddleware struct{}

// NewBYOKMiddleware creates a new BYOK middleware.
func NewBYOKMiddleware() *BYOKMiddleware {
	return &BYOKMiddleware{}
}

// InjectBYOKKey accepts the legacy header as a source hint without retaining
// its value. Credential Authority owns the actual saved credential.
// This middleware should run early in the chain, before any AI handlers.
func (m *BYOKMiddleware) InjectBYOKKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithOwnKeySourceHint(r.Context(), strings.TrimSpace(r.Header.Get(BYOKHeaderName)) != "")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithOwnKeySourceHint records that the request selected the saved-key source. The
// supplied header value is deliberately discarded; Gateway uses a credential
// stored in Credential Authority.
func WithOwnKeySourceHint(ctx context.Context, present bool) context.Context {
	return context.WithValue(ctx, byokContextKey{}, present)
}

// HasOwnKeySourceHint reports whether the legacy header selected saved-key use.
func HasOwnKeySourceHint(ctx context.Context) bool {
	value, _ := ctx.Value(byokContextKey{}).(bool)
	return value
}
