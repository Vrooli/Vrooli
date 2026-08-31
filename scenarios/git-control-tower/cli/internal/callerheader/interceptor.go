// Package callerheader is the client-side Connect interceptor that
// advertises the caller kind on outbound git-control-tower RPCs.
//
// The server's policygate interceptor (api/internal/policygate) reads
// X-Vrooli-Caller and X-Vrooli-Authorized to make the agent-access gate
// decision. Sending the header lets the server treat trusted callers
// (an explicit human, or an agent that holds an authorization flag)
// correctly even though the server's own env has no agent signals.
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
	// EnvAuthorized is the env var the CLI's `--i-was-explicitly-authorized`
	// flag sets to "true" before invoking the Connect client. The
	// interceptor copies it into the X-Vrooli-Authorized header so the
	// CLI command-runner can opt in without knowing the wire details.
	EnvAuthorized = "VROOLI_GCT_AUTHORIZED"
)

// New returns a unary client interceptor that stamps every outbound
// request with X-Vrooli-Caller (always) and X-Vrooli-Authorized (when
// VROOLI_GCT_AUTHORIZED=true).
func New() connect.Interceptor {
	return interceptor{}
}

type interceptor struct{}

func (interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Only stamp client-side requests; the same interceptor would
		// no-op on the server side (Connect calls WrapUnary on both
		// client and handler, but Spec.IsClient distinguishes).
		if req.Spec().IsClient {
			req.Header().Set(cliutil.HeaderCaller, cliutil.DetectCallerKind().String())
			if isAuthorized() {
				req.Header().Set(HeaderAuthorized, "true")
			}
		}
		return next(ctx, req)
	}
}

func (interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(cliutil.HeaderCaller, cliutil.DetectCallerKind().String())
		if isAuthorized() {
			conn.RequestHeader().Set(HeaderAuthorized, "true")
		}
		return conn
	}
}

func (interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func isAuthorized() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvAuthorized))) {
	case "true", "1", "yes":
		return true
	}
	return false
}
