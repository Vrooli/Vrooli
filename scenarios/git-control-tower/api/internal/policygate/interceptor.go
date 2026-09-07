package policygate

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	"git-control-tower/internal/config"

	"github.com/vrooli/cli-core/cliutil"
	auditorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auditor/auditor_v1connect"
	branchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch/branch_v1connect"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// HeaderAuthorized is retained as a compatibility marker for clients. It is
// never treated as proof of human intent; only the authenticated principal
// context populated by the authentication owner can grant authority.
const HeaderAuthorized = "X-Vrooli-Authorized"

// MutatingProcedures is the set of Connect procedure suffixes the
// interceptor treats as mutating. Other procedures pass through
// unchanged. Keep this list in sync with the manifest's `effect:
// write|destructive` entries.
//
// Methods whose handler owns request-bound intent consumption are listed in
// HandlerManagedIntentProcedures below. When a new mutating Connect method is
// added, register it in the appropriate map AND in the manifest.
var MutatingProcedures = map[string]string{
	worktreeconnect.WorktreeServiceCreateWorktreeProcedure: "write",
	worktreeconnect.WorktreeServiceRemoveWorktreeProcedure: "destructive",
	worktreeconnect.WorktreeServiceLockWorktreeProcedure:   "write",
	worktreeconnect.WorktreeServiceUnlockWorktreeProcedure: "write",
	worktreeconnect.WorktreeServiceMoveWorktreeProcedure:   "write",
	worktreeconnect.WorktreeServicePruneWorktreesProcedure: "destructive",
}

// HandlerManagedIntentProcedures are mutating methods whose handler consumes
// the request-bound intent. They still require a verified human principal at
// the transport boundary, but cannot use the generic interceptor intent check
// because the request carries the intent ID and the handler must bind it to
// the exact preview it prepares under the repository lock.
var HandlerManagedIntentProcedures = map[string]string{
	repoconnect.RepoServiceSetActiveRepositoryProcedure: "write",
	repoconnect.RepoServiceOpenRepositoryProcedure:      "write",
	repoconnect.RepoServiceCloneRepositoryProcedure:     "destructive",
	repoconnect.RepoServiceRemoveRepositoryProcedure:    "destructive",
	repoconnect.RepoServiceStageFilesProcedure:          "write",
	repoconnect.RepoServiceUnstageFilesProcedure:        "write",
	repoconnect.RepoServiceCreateCommitProcedure:        "write",
	repoconnect.RepoServiceDeletePathProcedure:          "write",
	repoconnect.RepoServiceSaveFileContentProcedure:     "write",
	repoconnect.RepoServiceDiscardFilesProcedure:        "write",
	repoconnect.RepoServiceIgnorePathProcedure:          "write",
	repoconnect.RepoServicePushToRemoteProcedure:        "destructive",
	repoconnect.RepoServicePullFromRemoteProcedure:      "write",
	repoconnect.RepoServiceRunUpstreamActionProcedure:   "write",
	repoconnect.RepoServiceSaveGroupingRulesProcedure:   "write",
	repoconnect.RepoServiceMoveGitignoreEntryProcedure:  "write",
	repoconnect.RepoServiceUntrackBinaryProcedure:       "write",
	repoconnect.RepoServiceSavePrecommitConfigProcedure: "write",
	repoconnect.RepoServiceSaveCredentialProcedure:      "write",
	repoconnect.RepoServiceDeleteCredentialProcedure:    "write",
	repoconnect.RepoServiceUpdateRemoteURLProcedure:     "write",
	repoconnect.RepoServiceGenerateSSHKeyProcedure:      "write",
	repoconnect.RepoServiceDeleteSSHKeyProcedure:        "destructive",
	auditorconnect.AuditorServiceApplyFixProcedure:      "write",
	branchconnect.BranchServiceCreateBranchProcedure:    "write",
	branchconnect.BranchServiceSwitchBranchProcedure:    "write",
	branchconnect.BranchServicePublishBranchProcedure:   "destructive",
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
// verified-principal and exact-intent gate on mutating procedures. Caller
// headers can describe attribution for diagnostics, but cannot satisfy either
// requirement.
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
		procedure := req.Spec().Procedure
		effect, genericIntent := MutatingProcedures[procedure]
		handlerEffect, handlerManaged := HandlerManagedIntentProcedures[procedure]
		if !genericIntent && !handlerManaged {
			return next(ctx, req)
		}
		if handlerManaged {
			effect = handlerEffect
		}
		caller, trusted := PrincipalFromContext(ctx)
		cmd := CommandSpec{Name: req.Spec().Procedure, Effect: effect}
		if handlerManaged {
			return i.wrapHandlerManaged(ctx, req, next, caller, trusted, cmd)
		}
		decision := DecisionDeny
		if trusted {
			decision, _ = DecideAuthenticated(ctx, cmd, i.policy)
		}
		if i.audit != nil {
			i.audit.Log(Event{
				Caller:     caller.Kind.String(),
				Procedure:  req.Spec().Procedure,
				Effect:     effect,
				Policy:     string(i.policy.AgentAccess),
				Decision:   decision.String(),
				Authorized: trusted && caller.Kind == cliutil.CallerKindHuman,
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

func (i *interceptor) wrapHandlerManaged(ctx context.Context, req connect.AnyRequest, next connect.UnaryFunc, caller Principal, trusted bool, cmd CommandSpec) (connect.AnyResponse, error) {
	decision := DecisionDeny
	if trusted && caller.Kind == cliutil.CallerKindHuman {
		decision = DecisionAllow
	}
	if i.audit != nil {
		i.audit.Log(Event{
			Caller: caller.Kind.String(), Procedure: req.Spec().Procedure, Effect: cmd.Effect,
			Policy: string(i.policy.AgentAccess), Decision: decision.String(), Authorized: decision == DecisionAllow,
		})
	}
	if decision != DecisionAllow {
		if !trusted {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human authority is required for this mutation"))
		}
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot perform this mutation"))
	}
	return next(ctx, req)
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
	switch strings.ToLower(strings.TrimSpace(h.Get(cliutil.HeaderCaller))) {
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
