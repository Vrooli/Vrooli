// Package provenance verifies Agent Manager identity tokens at an API boundary
// and carries the resulting bounded, server-verified claims through context.
// Invocation headers are preserved only as unverified channel observations.
package provenance

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
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
	Actor               string                   `json:"actor"`
	VerificationStatus  string                   `json:"verification_status"`
	RunID               string                   `json:"run_id,omitempty"`
	TaskID              string                   `json:"task_id,omitempty"`
	Subject             string                   `json:"subject,omitempty"`
	WorkspaceID         string                   `json:"workspace_id,omitempty"`
	Scopes              []string                 `json:"scopes,omitempty"`
	ProfileKey          string                   `json:"profile_key,omitempty"`
	ScopePath           string                   `json:"scope_path,omitempty"`
	WorkflowExecutionID string                   `json:"workflow_execution_id,omitempty"`
	WorkflowNodeID      string                   `json:"workflow_node_id,omitempty"`
	Attempt             uint32                   `json:"attempt,omitempty"`
	SessionID           string                   `json:"session_id,omitempty"`
	SessionKind         string                   `json:"session_kind,omitempty"`
	Source              string                   `json:"source,omitempty"`
	Invocation          Invocation               `json:"invocation,omitempty"`
	Grant               *identity.OperationGrant `json:"grant,omitempty"`
	// OperationGrant and its claims stay separate from the ordinary principal.
	// Only authn.RequireOperation may consume this independently verified proof.
	OperationGrant     *identity.OperationGrant `json:"-"`
	OperationScopes    []string                 `json:"-"`
	OperationExpiresAt time.Time                `json:"-"`
	OperationRunID     string                   `json:"-"`
	OperationGrantID   string                   `json:"-"`
}

// OperationGrantFromMeta interprets signed metadata only after the authority
// has verified the current grant and run. Malformed metadata grants nothing.
func OperationGrantFromMeta(meta map[string]string) *identity.OperationGrant {
	var grant identity.OperationGrant
	if err := json.Unmarshal([]byte(meta["operation_grant"]), &grant); err != nil || grant.ID == "" || grant.Approver == "" || !time.Now().Before(grant.ExpiresAt) {
		return nil
	}
	return &grant
}

// ScopeDeniedError is returned when a verified agent token lacks a required
// capability. It is deliberately distinct from absent or invalid identity so
// handlers can audit the refusal without treating an unverified request as an
// authorization decision.
type ScopeDeniedError struct {
	Required string
}

func (e ScopeDeniedError) Error() string { return "agent identity lacks required scope: " + e.Required }

// RequireScope is the relying-party authorization seam. Callers should record
// the returned typed refusal in their domain audit stream before translating it
// to PermissionDenied at the transport boundary.
func (p Provenance) RequireScope(required string) error {
	if p.HasScope(required) {
		return nil
	}
	return ScopeDeniedError{Required: strings.TrimSpace(required)}
}

func (p Provenance) IsVerifiedAgent() bool {
	return p.Actor == ActorAgent && p.VerificationStatus == VerificationVerified && strings.TrimSpace(p.RunID) != ""
}

func (p Provenance) IsAgent() bool { return p.Actor == ActorAgent }

// HasScope reports whether a verified agent capability covers required. It
// never treats an unverified or absent provenance value as authorized.
func (p Provenance) HasScope(required string) bool {
	if !p.IsVerifiedAgent() || strings.TrimSpace(required) == "" {
		return false
	}
	return scopecatalog.Resolve(p.Scopes, required)
}

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
		// Interactive claims carry the human owner in Subject for account
		// boundaries. The acting agent is the independently identified run.
		if p.Grant != nil {
			actorID = strings.TrimSpace(p.RunID)
		} else {
			actorID = strings.TrimSpace(p.Subject)
		}
		if actorID == "" {
			actorID = strings.TrimSpace(p.ProfileKey)
		}
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
	provenance        Provenance
	token             string
	operationToken    string
	operationVerifier PairVerifier
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
	if !cliutil.TrustedIdentityDestination(request) && (request.Header.Get(cliutil.HeaderAgentIdentityToken) != "" || request.Header.Get(cliutil.HeaderAgentOperationToken) != "") {
		clone := request.Clone(request.Context())
		clone.Header = request.Header.Clone()
		clone.Header.Del(cliutil.HeaderAgentIdentityToken)
		clone.Header.Del(cliutil.HeaderAgentOperationToken)
		return base.RoundTrip(clone)
	}
	token := ForwardedIdentityToken(request.Context())
	if !cliutil.TrustedIdentityDestination(request) {
		return base.RoundTrip(request)
	}
	if token == "" {
		// A CLI client can run through this server-owned transport because the
		// CLI and API modules both use http.DefaultTransport. Preserve only the
		// pair cli-core resolved for this exact request; unverified headers keep
		// the existing stripping behavior below.
		if cliutil.HasResolvedAgentCredentialPair(request) {
			return base.RoundTrip(request)
		}
		if request.Header.Get(cliutil.HeaderAgentOperationToken) == "" {
			return base.RoundTrip(request)
		}
		clone := request.Clone(request.Context())
		clone.Header = request.Header.Clone()
		clone.Header.Del(cliutil.HeaderAgentOperationToken)
		return base.RoundTrip(clone)
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	if operationToken := ForwardedOperationToken(request.Context()); operationToken != "" {
		clone.Header.Set(cliutil.HeaderAgentOperationToken, operationToken)
	} else {
		clone.Header.Del(cliutil.HeaderAgentOperationToken)
	}
	return base.RoundTrip(clone)
}

var installDefaultForwardingTransport sync.Once

// Install before application goroutines can issue HTTP requests. Assigning
// http.DefaultTransport from server.Run races with clients already in flight;
// sync.Once serializes installers, but cannot synchronize net/http readers.
func init() { InstallDefaultForwardingTransport() }

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

// ForwardedOperationToken returns the supplemental proof only after the pair
// was verified for this active baseline identity.
func ForwardedOperationToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	identity, ok := ctx.Value(contextKey{}).(requestIdentity)
	if !ok || !identity.provenance.IsVerifiedAgent() || identity.provenance.OperationGrant == nil {
		return ""
	}
	return identity.operationToken
}

// Verifier is the narrow Agent Manager verification seam. It keeps Agent
// Manager as token authority and makes all server behavior testable.
type Verifier interface {
	Verify(token string) (*cliutil.VerifyResult, error)
}

// PairVerifier verifies an operation token against one already verified
// baseline token. It deliberately returns only the supplemental proof.
type PairVerifier interface {
	VerifyPair(context.Context, string, string) (*cliutil.VerifyPairResult, error)
}

type VerifierFunc func(token string) (*cliutil.VerifyResult, error)

func (f VerifierFunc) Verify(token string) (*cliutil.VerifyResult, error) { return f(token) }

// CLIUtilVerifier calls Agent Manager through cli-core's standard verifier.
type CLIUtilVerifier struct{}

func (CLIUtilVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	return (cliutil.IdentityEnv{Token: token}).VerifyIdentity()
}

func (CLIUtilVerifier) VerifyPair(ctx context.Context, baselineToken, operationToken string) (*cliutil.VerifyPairResult, error) {
	return (cliutil.IdentityEnv{Token: baselineToken}).VerifyIdentityPair(ctx, operationToken)
}

// VerifyOwnWebConsoleSession rechecks the verified inbound token with the
// authority's optional session-owner proof. Ambient environment is never used.
func VerifyOwnWebConsoleSession(ctx context.Context) (string, error) {
	p := FromContext(ctx)
	token := ForwardedIdentityToken(ctx)
	if !p.IsVerifiedAgent() || token == "" {
		return "", nil
	}
	result, err := (cliutil.IdentityEnv{Token: token}).VerifyIdentityWithSessionBinding(ctx)
	if err != nil {
		return "", err
	}
	if result == nil || !result.Valid || result.Claims == nil || result.Claims.RunID != p.RunID {
		return "", nil
	}
	return strings.TrimSpace(result.Claims.Meta["verified_web_console_session_id"]), nil
}

// OperationProof is a freshly owner-verified supplemental grant. It remains
// outside Provenance.Scopes and the ordinary identity.Principal.
type OperationProof struct {
	Grant     *identity.OperationGrant
	Scopes    []string
	ExpiresAt time.Time
	RunID     string
	GrantID   string
}

// ReverifyOperationProof checks the exact current pair again at the protected
// effect boundary. Unavailable or changed grants fail closed for that effect.
func ReverifyOperationProof(ctx context.Context) (*OperationProof, error) {
	if ctx == nil {
		return nil, nil
	}
	request, ok := ctx.Value(contextKey{}).(requestIdentity)
	if !ok || request.operationToken == "" || request.operationVerifier == nil || !request.provenance.IsVerifiedAgent() {
		return nil, nil
	}
	result, err := request.operationVerifier.VerifyPair(ctx, request.token, request.operationToken)
	if err != nil {
		return nil, err
	}
	if result == nil || !result.Valid || result.Operation == nil || !result.Operation.Valid || result.Operation.Claims == nil {
		return nil, nil
	}
	claims := result.Operation.Claims
	grant := OperationGrantFromMeta(claims.Meta)
	if grant == nil {
		return nil, nil
	}
	return &OperationProof{
		Grant: grant, Scopes: append([]string(nil), claims.Scopes...),
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(), RunID: strings.TrimSpace(claims.RunID), GrantID: grant.ID,
	}, nil
}

// VerifyPath is the Agent Manager endpoint that answers token verification.
//
// A request to it must never be verified by this middleware. Verification is
// itself an HTTP call to this path, and cli-core's forwarding transport
// re-attaches the ambient identity token to that call, so gating the endpoint
// behind verification makes every inbound verification issue another one.
// That recursion has no base case and amplifies until it saturates the host.
const VerifyPath = "/api/v1/identity/verify"

// isVerificationEndpoint reports whether the request targets the verification
// endpoint itself. Matching is on the cleaned path only; the token authority
// serves this route on every scenario that embeds api-core.
func isVerificationEndpoint(rawPath string) bool {
	return path.Clean(rawPath) == VerifyPath
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
			operationToken := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentOperationToken))
			// The verification endpoint verifies its own caller in the handler.
			// Verifying it here would re-enter this same endpoint without bound.
			if token == "" || isVerificationEndpoint(r.URL.Path) {
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
			provenance.Subject = strings.TrimSpace(result.Claims.Subject)
			provenance.WorkspaceID = strings.TrimSpace(result.Claims.WorkspaceID)
			provenance.Scopes = append([]string(nil), result.Claims.Scopes...)
			provenance.Grant = OperationGrantFromMeta(result.Claims.Meta)
			provenance.ProfileKey = strings.TrimSpace(result.Claims.ProfileKey)
			provenance.ScopePath = strings.TrimSpace(result.Claims.ScopePath)
			// Claims.Meta is part of the verified identity response. These optional
			// execution coordinates use Agent Manager's existing signed metadata keys;
			// its verifier checks retained workflow membership. Caller headers and
			// snake-case observation labels cannot establish this ownership.
			provenance.WorkflowExecutionID = strings.TrimSpace(result.Claims.Meta["workflowExecutionId"])
			provenance.WorkflowNodeID = strings.TrimSpace(result.Claims.Meta["workflowNodeId"])
			if raw := strings.TrimSpace(result.Claims.Meta["attempt"]); raw != "" {
				if attempt, parseErr := strconv.ParseUint(raw, 10, 32); parseErr == nil {
					provenance.Attempt = uint32(attempt)
				}
			}
			request := requestIdentity{provenance: provenance, token: token}
			if operationToken != "" {
				if pairVerifier, ok := verifier.(PairVerifier); ok {
					pair, pairErr := pairVerifier.VerifyPair(r.Context(), token, operationToken)
					if pairErr != nil {
						slog.Warn("agent operation credential verification unavailable", "error", pairErr)
					} else if pair != nil && pair.Valid && pair.Operation != nil && pair.Operation.Valid && pair.Operation.Claims != nil {
						operationClaims := pair.Operation.Claims
						provenance.OperationGrant = OperationGrantFromMeta(operationClaims.Meta)
						if provenance.OperationGrant != nil {
							provenance.OperationScopes = append([]string(nil), operationClaims.Scopes...)
							provenance.OperationExpiresAt = time.Unix(operationClaims.ExpiresAt, 0).UTC()
							provenance.OperationRunID = strings.TrimSpace(operationClaims.RunID)
							provenance.OperationGrantID = provenance.OperationGrant.ID
							request.operationToken = operationToken
							request.operationVerifier = pairVerifier
						}
					}
				}
			}
			request.provenance = provenance
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, request)))
		})
	}
}
