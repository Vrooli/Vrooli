package policygate

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	"git-control-tower/internal/config"

	"github.com/vrooli/cli-core/cliutil"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// HeaderCaller is the request header set by the CLI to advertise its
// caller classification (human|agent|external-agent|...). When absent,
// the server falls back to its own DetectCallerKind() which on the
// server side will almost always be CallerKindUnknown.
const HeaderCaller = "X-Vrooli-Caller"

// HeaderAuthorized is the request header the CLI sets to "true" when
// the user passed the agent-override flag. Satisfies the `confirm`
// policy.
const HeaderAuthorized = "X-Vrooli-Authorized"

// MutatingProcedures is the set of Connect procedure suffixes the
// interceptor treats as mutating. Other procedures pass through
// unchanged. Keep this list in sync with the manifest's `effect:
// write|destructive` entries.
//
// Today only WorktreeService has Connect handlers (the other GCT
// domains are still REST). When a new mutating Connect method is
// added, register it here AND in the manifest.
var MutatingProcedures = map[string]string{
	worktreeconnect.WorktreeServiceCreateWorktreeProcedure: "write",
	worktreeconnect.WorktreeServiceRemoveWorktreeProcedure: "destructive",
	worktreeconnect.WorktreeServiceLockWorktreeProcedure:   "write",
	worktreeconnect.WorktreeServiceUnlockWorktreeProcedure: "write",
	worktreeconnect.WorktreeServiceMoveWorktreeProcedure:   "write",
	worktreeconnect.WorktreeServicePruneWorktreesProcedure: "destructive",
}

// AuditLogger is the minimal log seam the interceptor uses to record
// gate decisions. Production wires the GCT structured logger; tests
// can wire a slice-collector.
type AuditLogger interface {
	Log(event Event)
}

// Event is one gate decision record.
type Event struct {
	Caller     string
	Procedure  string
	Effect     string
	Policy     string
	Decision   string
	Authorized bool
}

// stdAuditLogger sends events to the package-level log.Default(). Good
// enough for development; production callers should swap in a
// structured logger.
type stdAuditLogger struct{ logger *log.Logger }

// StdAuditLogger returns an AuditLogger backed by the standard logger.
// Used by Server when no custom logger is wired.
func StdAuditLogger() AuditLogger { return stdAuditLogger{logger: log.Default()} }

func (s stdAuditLogger) Log(event Event) {
	s.logger.Printf("policygate event caller=%s procedure=%s effect=%s policy=%s decision=%s authorized=%t",
		event.Caller, event.Procedure, event.Effect, event.Policy, event.Decision, event.Authorized)
}

// NewInterceptor returns a Connect interceptor that enforces the
// agent-access gate on mutating procedures.
//
// Order of operations:
//
//  1. Read X-Vrooli-Caller header. If absent, fall back to
//     DetectCallerKind() on the server's own env (rarely informative
//     but covers the "agent → direct curl" path).
//  2. Read X-Vrooli-Authorized header. true → CallerOverrideFlags.AuthorizedByUser.
//  3. Look up the procedure in MutatingProcedures. Read-only procedures
//     pass through.
//  4. Decide() → record audit event → apply.
//
// The interceptor is unary-only; streaming procedures are not gated
// today (and GCT has none on the Connect surface yet). If you add
// streaming methods, extend WrapStreamingHandler.
func NewInterceptor(policy config.PolicyConfig, audit AuditLogger) connect.Interceptor {
	return &interceptor{policy: policy, audit: audit}
}

type interceptor struct {
	policy config.PolicyConfig
	audit  AuditLogger
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		effect, ok := MutatingProcedures[req.Spec().Procedure]
		if !ok {
			return next(ctx, req)
		}
		caller := callerFromHeader(req.Header(), i.policy.CallerDetection)
		flags := CallerOverrideFlags{AuthorizedByUser: strings.EqualFold(req.Header().Get(HeaderAuthorized), "true")}
		cmd := CommandSpec{Name: req.Spec().Procedure, Effect: effect}
		decision := Decide(caller, cmd, flags, i.policy)
		if i.audit != nil {
			i.audit.Log(Event{
				Caller:     caller.String(),
				Procedure:  req.Spec().Procedure,
				Effect:     effect,
				Policy:     string(i.policy.AgentAccess),
				Decision:   decision.String(),
				Authorized: flags.AuthorizedByUser,
			})
		}
		switch decision {
		case DecisionAllow:
			return next(ctx, req)
		case DecisionWarn:
			// Surface the warning via response trailers so the CLI can
			// display it. The mutation still runs.
			resp, err := next(ctx, req)
			if resp != nil {
				resp.Header().Set("X-Vrooli-Policy-Warning", RenderDenyMessage(cmd, i.policy))
			}
			return resp, err
		case DecisionDeny:
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New(RenderDenyMessage(cmd, i.policy)))
		default:
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("policygate: unknown decision"))
		}
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	// Streaming mutating methods do not exist in GCT today. If/when
	// they do, mirror WrapUnary here.
	return next
}

// callerFromHeader returns the CallerKind based on the X-Vrooli-Caller
// header. When the header is absent or unrecognized, falls back to the
// server-side detector chosen by callerDetection. Server-side fallback
// is rarely informative (the server's env has no agent signals) but
// covers the "agent runs curl directly" path.
func callerFromHeader(h interface {
	Get(string) string
}, detection config.CallerDetection,
) cliutil.CallerKind {
	switch strings.ToLower(strings.TrimSpace(h.Get(HeaderCaller))) {
	case "human":
		return cliutil.CallerKindHuman
	case "vrooli-agent":
		return cliutil.CallerKindVrooliAgent
	case "external-agent":
		return cliutil.CallerKindExternalAgent
	case "override-agent":
		return cliutil.CallerKindOverride
	}
	// Header missing or unknown → use server-side env detection.
	if detection == config.CallerDetectionStrict {
		if cliutil.IsAgentControlledContext() {
			return cliutil.CallerKindVrooliAgent
		}
		return cliutil.CallerKindUnknown
	}
	return cliutil.DetectCallerKind()
}
