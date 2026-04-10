package middleware

import (
	"context"
	"net/http"
	"strings"
)

const (
	// BYOKHeaderName is the header name for the user's OpenRouter API key.
	BYOKHeaderName = "X-BYOK-OpenRouter-Key"
)

// byokContextKey is the context key for storing the BYOK API key.
type byokContextKey struct{}

// BYOKMiddleware extracts the BYOK OpenRouter API key from requests
// and makes it available to handlers via context.
type BYOKMiddleware struct{}

// NewBYOKMiddleware creates a new BYOK middleware.
func NewBYOKMiddleware() *BYOKMiddleware {
	return &BYOKMiddleware{}
}

// InjectBYOKKey extracts the BYOK key from the request header and injects it into context.
// This middleware should run early in the chain, before any AI handlers.
func (m *BYOKMiddleware) InjectBYOKKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		byokKey := strings.TrimSpace(r.Header.Get(BYOKHeaderName))

		// Add to context even if empty (handlers can check)
		ctx := WithBYOKKey(r.Context(), byokKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithBYOKKey adds a BYOK API key to the context.
func WithBYOKKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, byokContextKey{}, key)
}

// BYOKKeyFromContext retrieves the BYOK API key from context.
// Returns empty string if no key is present.
func BYOKKeyFromContext(ctx context.Context) string {
	if v := ctx.Value(byokContextKey{}); v != nil {
		return v.(string)
	}
	return ""
}

// HasBYOKKey checks if a BYOK API key is present in the context.
func HasBYOKKey(ctx context.Context) bool {
	return BYOKKeyFromContext(ctx) != ""
}
