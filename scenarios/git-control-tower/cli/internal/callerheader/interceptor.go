// Package callerheader is the client-side Connect interceptor that
// advertises the caller kind on outbound git-control-tower RPCs.
//
// The server's policygate interceptor records these headers as attribution
// metadata only. They do not establish a human principal or authorize a
// repository mutation; that comes from authenticator verification and a
// server-issued intent.
package callerheader

import (
	"context"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"
)

const (
	// HeaderAuthorized mirrors api/internal/policygate.HeaderAuthorized.
	HeaderAuthorized = "X-Vrooli-Authorized"
	// EnvAuthorized is retained for compatibility with older clients. The
	// server treats the resulting header as non-authoritative metadata.
	EnvAuthorized = "VROOLI_GCT_AUTHORIZED"
)

// New returns a unary client interceptor that stamps every outbound request
// with attribution metadata.
func New() connect.Interceptor {
	return NewWithEnvLookup(os.Getenv)
}

// NewWithEnvLookup returns an interceptor that reads compatibility metadata
// through getenv. Production uses New, which wires os.Getenv; tests can supply
// deterministic values without changing process-wide environment state.
func NewWithEnvLookup(getenv func(string) string) connect.Interceptor {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	return interceptor{Getenv: getenv}
}

type interceptor struct {
	Getenv func(string) string
}

func (i interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Only stamp client-side requests; the same interceptor would
		// no-op on the server side (Connect calls WrapUnary on both
		// client and handler, but Spec.IsClient distinguishes).
		if req.Spec().IsClient {
			req.Header().Set(cliutil.HeaderCaller, cliutil.DetectCallerKind().String())
			if isAuthorized(i.Getenv) {
				req.Header().Set(HeaderAuthorized, "true")
			}
		}
		return next(ctx, req)
	}
}

func (i interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(cliutil.HeaderCaller, cliutil.DetectCallerKind().String())
		if isAuthorized(i.Getenv) {
			conn.RequestHeader().Set(HeaderAuthorized, "true")
		}
		return conn
	}
}

func (interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func isAuthorized(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(EnvAuthorized))) {
	case "true", "1", "yes":
		return true
	}
	return false
}
