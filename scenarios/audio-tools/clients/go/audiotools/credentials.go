package audiotools

import (
	"context"

	"connectrpc.com/connect"
)

// Header names used on the wire. Match the audio-tools API's
// internal/byok/envelope package — the two sides MUST agree on every
// header literal, so the wire constants live here as the canonical
// source for client code and tests share them with the server-side
// envelope package.
const (
	HeaderProvider     = "X-Audio-BYOK-Provider"
	HeaderKey          = "X-Audio-BYOK-Key"
	HeaderLPBSToken    = "X-Audio-LPBS-Token"
	HeaderUserIdentity = "X-Audio-User-Identity"
)

// Credentials is the typed shape carried over the wire as the four
// X-Audio-* metadata headers. The byok_key field is the only secret
// in this struct; callers MUST NOT log a Credentials value directly.
type Credentials struct {
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string
}

// HasAny reports whether any non-empty field is present.
func (c Credentials) HasAny() bool {
	return c.BYOKProvider != "" || c.BYOKKey != "" || c.LPBSToken != "" || c.UserIdentity != ""
}

type credentialsKey struct{}

// WithCredentials returns a new context that carries the given
// Credentials. The default WithCredentialsInterceptor getter reads
// from here.
func WithCredentials(ctx context.Context, creds Credentials) context.Context {
	return context.WithValue(ctx, credentialsKey{}, creds)
}

// FromContext returns the Credentials carried by ctx, if any.
func FromContext(ctx context.Context) (Credentials, bool) {
	c, ok := ctx.Value(credentialsKey{}).(Credentials)
	return c, ok
}

// CredentialsGetter resolves per-call credentials from context.
type CredentialsGetter func(ctx context.Context) Credentials

// DefaultCredentialsGetter reads from context using FromContext. Use
// WithCredentialsInterceptor(DefaultCredentialsGetter) when the caller
// pipeline always stashes credentials with WithCredentials.
var DefaultCredentialsGetter CredentialsGetter = func(ctx context.Context) Credentials {
	c, _ := FromContext(ctx)
	return c
}

// WithCredentialsInterceptor returns a Connect interceptor that
// populates the four X-Audio-* headers on every outbound unary and
// stream request, sourced from the per-call context via the supplied
// getter.
//
// Absent fields are not written. Server-side parsing tolerates absent
// headers and treats them as "not provided".
func WithCredentialsInterceptor(get CredentialsGetter) connect.Interceptor {
	if get == nil {
		get = DefaultCredentialsGetter
	}
	return &credentialsInterceptor{get: get}
}

type credentialsInterceptor struct {
	get CredentialsGetter
}

func (i *credentialsInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		applyHeaders(req.Header(), i.get(ctx))
		return next(ctx, req)
	}
}

func (i *credentialsInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		applyHeaders(conn.RequestHeader(), i.get(ctx))
		return conn
	}
}

func (i *credentialsInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	// Client-side interceptor; handler wrap is a no-op.
	return next
}

func applyHeaders(h interface{ Set(string, string) }, c Credentials) {
	if c.BYOKProvider != "" {
		h.Set(HeaderProvider, c.BYOKProvider)
	}
	if c.BYOKKey != "" {
		h.Set(HeaderKey, c.BYOKKey)
	}
	if c.LPBSToken != "" {
		h.Set(HeaderLPBSToken, c.LPBSToken)
	}
	if c.UserIdentity != "" {
		h.Set(HeaderUserIdentity, c.UserIdentity)
	}
}
