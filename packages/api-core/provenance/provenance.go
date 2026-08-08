// Package provenance verifies Agent Manager identity tokens at an API boundary
// and carries the resulting bounded, server-verified claims through context.
// Invocation headers are preserved only as unverified channel observations.
package provenance

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/cli-core/cliutil"
)

const (
	ActorOperator = "operator"
	ActorAgent    = "agent"

	VerificationVerified    = "verified"
	VerificationAbsent      = "absent"
	VerificationInvalid     = "invalid"
	VerificationUnavailable = "unavailable"
)

// Invocation is bounded client-channel metadata. It is intentionally kept
// distinct from Provenance because API clients can forge it.
type Invocation struct {
	Scenario         string `json:"scenario,omitempty"`
	Command          string `json:"command,omitempty"`
	InvocationID     string `json:"invocation_id,omitempty"`
	HarnessSessionID string `json:"harness_session_id,omitempty"`
	HarnessKind      string `json:"harness_kind,omitempty"`
}

// Provenance records the verification result for an inbound request. Only a
// value with ActorAgent and VerificationVerified may be used as evidence that
// an Agent Manager run performed an attributed mutation.
type Provenance struct {
	Actor               string     `json:"actor"`
	VerificationStatus  string     `json:"verification_status"`
	RunID               string     `json:"run_id,omitempty"`
	TaskID              string     `json:"task_id,omitempty"`
	ProfileKey          string     `json:"profile_key,omitempty"`
	ScopePath           string     `json:"scope_path,omitempty"`
	WorkflowExecutionID string     `json:"workflow_execution_id,omitempty"`
	WorkflowNodeID      string     `json:"workflow_node_id,omitempty"`
	Attempt             uint32     `json:"attempt,omitempty"`
	SessionID           string     `json:"session_id,omitempty"`
	SessionKind         string     `json:"session_kind,omitempty"`
	Source              string     `json:"source,omitempty"`
	Invocation          Invocation `json:"invocation,omitempty"`
}

func (p Provenance) IsVerifiedAgent() bool {
	return p.Actor == ActorAgent && p.VerificationStatus == VerificationVerified && strings.TrimSpace(p.RunID) != ""
}

func (p Provenance) IsAgent() bool { return p.Actor == ActorAgent }

// WriteFields returns the bounded attribution fields that a write seam may
// persist. A caller-supplied run id is never treated as proof: only the
// server-verified claim can populate actor_id/run_id. ProfileKey is the
// verified durable actor identity (the team member/profile), while RunID is
// the individual execution instance. The remaining fields
// preserve the verification outcome and the unverified invocation source so
// absence, invalid credentials, and verifier outages remain distinguishable.
func (p Provenance) WriteFields() (actorID, actorKind, sourceRuntime, verificationStatus, runID, workflowExecutionID string) {
	verificationStatus = strings.TrimSpace(p.VerificationStatus)
	if verificationStatus == "" {
		verificationStatus = VerificationAbsent
	}
	actorKind = strings.TrimSpace(p.Actor)
	if actorKind == "" {
		actorKind = ActorOperator
	}
	sourceRuntime = strings.TrimSpace(p.Invocation.Scenario)
	if sourceRuntime == "" {
		sourceRuntime = strings.TrimSpace(p.Source)
	}
	if p.IsVerifiedAgent() {
		actorID = strings.TrimSpace(p.ProfileKey)
		runID = strings.TrimSpace(p.RunID)
		workflowExecutionID = strings.TrimSpace(p.WorkflowExecutionID)
	}
	return actorID, actorKind, sourceRuntime, verificationStatus, runID, workflowExecutionID
}

func (p Provenance) WithSession(sessionID, sessionKind, source string) Provenance {
	p.SessionID = strings.TrimSpace(sessionID)
	p.SessionKind = strings.TrimSpace(sessionKind)
	p.Source = strings.TrimSpace(source)
	return p
}

// ObservationFields returns unverified harness-channel observations. A
// session id grants no authority and is intentionally separate from RunID.
func (p Provenance) ObservationFields() (sessionID, sessionKind string) {
	return strings.TrimSpace(p.Invocation.HarnessSessionID), strings.TrimSpace(p.Invocation.HarnessKind)
}

func (p Provenance) FormatStartedBy() string {
	if p.IsAgent() {
		return "agent:" + p.ProfileKey + "/" + p.RunID
	}
	return ActorOperator
}

type contextKey struct{}

type requestIdentity struct {
	provenance Provenance
	token      string
}

// ForwardingTransport copies a verified inbound Agent Manager identity to an
// outbound request. It intentionally ignores unverified request headers: only
// Middleware can place a forwardable token in the request context.
type ForwardingTransport struct {
	Base http.RoundTripper
}

func (t ForwardingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if request == nil {
		return base.RoundTrip(request)
	}
	token := ForwardedIdentityToken(request.Context())
	if token == "" {
		return base.RoundTrip(request)
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	return base.RoundTrip(clone)
}

var installDefaultForwardingTransport sync.Once

// InstallDefaultForwardingTransport enables token forwarding for standard
// net/http clients owned by an api-core server. A client with no explicit
// transport (the normal scenario pattern) uses http.DefaultTransport, so field
// scenarios inherit forwarding without per-scenario request code.
func InstallDefaultForwardingTransport() {
	installDefaultForwardingTransport.Do(func() {
		http.DefaultTransport = ForwardingTransport{Base: http.DefaultTransport}
	})
}

func NewContext(ctx context.Context, provenance Provenance) context.Context {
	return context.WithValue(ctx, contextKey{}, requestIdentity{provenance: provenance})
}

func FromContext(ctx context.Context) Provenance {
	if ctx != nil {
		if identity, ok := ctx.Value(contextKey{}).(requestIdentity); ok {
			return identity.provenance
		}
		if provenance, ok := ctx.Value(contextKey{}).(Provenance); ok {
			return provenance
		}
	}
	return Provenance{Actor: ActorOperator, VerificationStatus: VerificationAbsent}
}

// ForwardedIdentityToken returns the opaque caller token only when the inbound
// request was verified as an active Agent Manager identity. Shared API clients
// use it for an internal scenario hop; no unverified header is forwarded.
func ForwardedIdentityToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	identity, ok := ctx.Value(contextKey{}).(requestIdentity)
	if !ok || !identity.provenance.IsVerifiedAgent() {
		return ""
	}
	return identity.token
}

// Verifier is the narrow Agent Manager verification seam. It keeps Agent
// Manager as token authority and makes all server behavior testable.
type Verifier interface {
	Verify(token string) (*cliutil.VerifyResult, error)
}

type VerifierFunc func(token string) (*cliutil.VerifyResult, error)

func (f VerifierFunc) Verify(token string) (*cliutil.VerifyResult, error) { return f(token) }

// CLIUtilVerifier calls Agent Manager through cli-core's standard verifier.
type CLIUtilVerifier struct{}

func (CLIUtilVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	return (cliutil.IdentityEnv{Token: token}).VerifyIdentity()
}

// Middleware is deliberately fail-open for request handling. It never upgrades
// missing, invalid, or unavailable verification to an operator or verified
// agent claim: callers receive an explicit verification state instead.
func Middleware(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provenance := Provenance{
				Actor:              ActorOperator,
				VerificationStatus: VerificationAbsent,
				Invocation: Invocation{
					Scenario:         strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationScenario)),
					Command:          strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationCommand)),
					InvocationID:     strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationID)),
					HarnessSessionID: strings.TrimSpace(r.Header.Get(cliutil.HeaderHarnessSessionID)),
					HarnessKind:      strings.TrimSpace(r.Header.Get(cliutil.HeaderHarnessKind)),
				},
			}
			token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
			if token == "" {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestIdentity{provenance: provenance})))
				return
			}
			if verifier == nil {
				provenance.VerificationStatus = VerificationUnavailable
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestIdentity{provenance: provenance})))
				return
			}
			result, err := verifier.Verify(token)
			if err != nil {
				slog.Warn("agent identity verification unavailable", "error", err)
				provenance.VerificationStatus = VerificationUnavailable
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestIdentity{provenance: provenance})))
				return
			}
			if result == nil || !result.Valid || result.Claims == nil {
				provenance.VerificationStatus = VerificationInvalid
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestIdentity{provenance: provenance})))
				return
			}
			provenance.Actor = ActorAgent
			provenance.VerificationStatus = VerificationVerified
			provenance.RunID = strings.TrimSpace(result.Claims.RunID)
			provenance.TaskID = strings.TrimSpace(result.Claims.TaskID)
			provenance.ProfileKey = strings.TrimSpace(result.Claims.ProfileKey)
			provenance.ScopePath = strings.TrimSpace(result.Claims.ScopePath)
			// Claims.Meta is part of the verified identity response. These optional
			// execution coordinates are never read from caller-controlled headers.
			provenance.WorkflowExecutionID = strings.TrimSpace(result.Claims.Meta["workflow_execution_id"])
			provenance.WorkflowNodeID = strings.TrimSpace(result.Claims.Meta["workflow_node_id"])
			if raw := strings.TrimSpace(result.Claims.Meta["attempt"]); raw != "" {
				if attempt, parseErr := strconv.ParseUint(raw, 10, 32); parseErr == nil {
					provenance.Attempt = uint32(attempt)
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, requestIdentity{provenance: provenance, token: token})))
		})
	}
}
