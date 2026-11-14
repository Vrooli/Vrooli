// Identity-renewal responsibility: prove an attached session binding before rotating its ordinary identity.
package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	sessionBindingProofVersion  = cliutil.AgentSessionBindingProofVersion
	sessionBindingProofTimeout  = 3 * time.Second
	sessionBindingProofMaxBytes = 4096
)

var (
	errBindingProofTimeout     = errors.New("session binding proof timed out")
	errBindingProofUnavailable = errors.New("session binding proof unavailable")
	errBindingProofMismatch    = errors.New("session binding proof mismatch")
)

// SessionBindingProof shares its wire contract with the control-plane owner.
type SessionBindingProof = cliutil.AgentSessionBindingProof

type SessionBindingRequest struct {
	RunID                                              uuid.UUID
	HarnessSessionID, WebConsoleSessionID, HarnessKind string
	ProcessID                                          int
}

// SessionBindingProofReader is deliberately narrow and read-only.
type SessionBindingProofReader interface {
	ReadSessionBinding(context.Context, SessionBindingRequest) (SessionBindingProof, error)
}

type SessionBindingProofReaderFunc func(context.Context, SessionBindingRequest) (SessionBindingProof, error)

func (f SessionBindingProofReaderFunc) ReadSessionBinding(ctx context.Context, request SessionBindingRequest) (SessionBindingProof, error) {
	return f(ctx, request)
}

// NewCLISessionBindingProofReader uses the installed control-plane owner command
// without a shell. Identity credentials are removed from the child environment.
func NewCLISessionBindingProofReader() SessionBindingProofReader {
	return SessionBindingProofReaderFunc(readCLISessionBindingProof)
}

type limitedProofOutput struct{ bytes.Buffer }

func (b *limitedProofOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > sessionBindingProofMaxBytes {
		return 0, errors.New("session binding proof output exceeds limit")
	}
	return b.Buffer.Write(p)
}

// Only fixed control-plane categories are retained. Never return or log raw
// subprocess stderr, which may contain environment or caller data.
type bindingProofReason struct{ category string }

func (e bindingProofReason) Error() string { return "owner proof " + e.category }

func classifiedBindingProofFailure(stderr string) error {
	allowed := map[string]bool{
		"kernel_binding_unverified": true, "launch_owner_unverified": true,
		"lease_changed_during_proof": true, "lease_identity_mismatch": true,
		"lease_time_unverified": true, "lease_unavailable": true,
		"process_changed_during_proof": true, "process_incarnation_unverified": true,
		"process_started_after_lease": true, "proof_cancelled": true,
		"proof_reader_unavailable": true, "run_binding_missing": true,
		"run_process_mismatch": true, "scope_membership_unverified": true,
		"selector_missing": true, "session_scope_unverified": true,
		"web_console_session_unverified": true, "runtime_home": true,
		"runtime_registry": true,
	}
	const prefix = "identity binding proof unavailable: "
	// The control plane is invoked with --json, so its errors normally arrive
	// as a JSON envelope on stderr. Decode only its documented fields and keep
	// the error text private; it is used solely for an exact allowlist match.
	if strings.HasPrefix(strings.TrimSpace(stderr), "{") {
		keys := json.NewDecoder(strings.NewReader(stderr))
		if first, err := keys.Token(); err != nil || first != json.Delim('{') {
			return errBindingProofUnavailable
		}
		seen := make(map[string]bool)
		for keys.More() {
			token, err := keys.Token()
			key, ok := token.(string)
			if err != nil || !ok || seen[key] {
				return errBindingProofUnavailable
			}
			seen[key] = true
			var value json.RawMessage
			if err := keys.Decode(&value); err != nil {
				return errBindingProofUnavailable
			}
		}
		if last, err := keys.Token(); err != nil || last != json.Delim('}') {
			return errBindingProofUnavailable
		}
		var envelope struct {
			Success  *bool  `json:"success"`
			Error    string `json:"error"`
			Code     string `json:"code"`
			Category string `json:"category"`
		}
		decoder := json.NewDecoder(strings.NewReader(stderr))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&envelope); err != nil || envelope.Success == nil || *envelope.Success || envelope.Code == "" || envelope.Category == "" || strings.ContainsAny(envelope.Error, "\r\n") {
			return errBindingProofUnavailable
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errBindingProofUnavailable
		}
		stderr = envelope.Error
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Error: "))
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		reason := strings.TrimPrefix(line, prefix)
		if allowed[reason] {
			return fmt.Errorf("%w: %w", errBindingProofUnavailable, bindingProofReason{category: reason})
		}
	}
	return errBindingProofUnavailable
}

// sessionBindingFailureNext maps each fixed control-plane proof category to what
// went wrong and what the caller can do next. Only these allow-listed categories
// reach a requester; raw subprocess output never does.
var sessionBindingFailureNext = map[string]string{
	"lease_unavailable":              "the host session lease could not be read, so the session may have ended; run `vrooli agent list` to confirm it is live, then retry",
	"lease_identity_mismatch":        "the host lease does not match the harness kind, session id or Web Console session in this request; run `vrooli agent list --json`, then retry naming that exact session",
	"launch_owner_unverified":        "the lease does not carry a launcher-recorded Web Console owner; this session was started before that was recorded and cannot request authority, so relaunch it",
	"web_console_session_unverified": "Web Console could not confirm the session is live (stopped, or its state is unreadable); retry once, and if it repeats check that Web Console is running",
	"scope_membership_unverified":    "the session's launcher process is neither inside its agent scope nor the direct parent of the one process that is, so the host cannot tie the session to its scope; an operator must reconcile the installed `vrooli` CLI with the launcher that started this session, or the session must be relaunched",
	"session_scope_unverified":       "the lease names an agent scope the host does not recognise; relaunch the session",
	"kernel_binding_unverified":      "the kernel's record of the session process disagrees with its lease (boot, cgroup or identity); relaunch the session",
	"process_incarnation_unverified": "the session process no longer matches the incarnation its lease recorded; relaunch the session",
	"process_started_after_lease":    "the process claiming this lease started after the lease was created; relaunch the session",
	"process_changed_during_proof":   "the session process or lease changed while it was being proved; retry once, and relaunch the session if it repeats",
	"lease_changed_during_proof":     "the session process or lease changed while it was being proved; retry once, and relaunch the session if it repeats",
	"lease_time_unverified":          "the lease timestamps are not consistent with the host clock; retry, and relaunch the session if it repeats",
	"run_binding_missing":            "the attached run's process binding is missing; relaunch the session",
	"run_process_mismatch":           "the attached run records a different process than the session's lease; relaunch the session",
	"proof_cancelled":                "the proof was cancelled before it finished; retry",
	"proof_reader_unavailable":       "the installed `vrooli` CLI could not run the proof; an operator must repair the control-plane install",
	"runtime_registry":               "the installed `vrooli` CLI could not open the runtime registry, usually because the CLI build and the registry schema disagree; an operator must rebuild the CLI or finish the registry migration (no session can request authority until then)",
	"runtime_home":                   "the installed `vrooli` CLI could not resolve its runtime home; an operator must repair the control-plane install",
	"selector_missing":               "the request did not name a session; run `vrooli agent list --json` and retry naming it",
}

// describeSessionBindingFailure says which component could not prove the live
// session and what to do about it. The result is shown to the requesting agent,
// so it is built only from fixed categories and fixed text.
func describeSessionBindingFailure(err error) string {
	const prefix = "live launcher session binding is unavailable"
	var reason bindingProofReason
	switch {
	case errors.Is(err, errBindingProofMismatch):
		return prefix + ": the control-plane proof was returned for a different session, process, run or Web Console session than this request names; run `vrooli agent list --json` and retry naming the exact session. If it repeats, the session's recorded launch details no longer match the live process, so relaunch it"
	case errors.Is(err, errBindingProofTimeout):
		return prefix + ": `vrooli agent identity-binding` did not answer within " + sessionBindingProofTimeout.String() + "; the host may be under load, so retry once. If it repeats an operator must check the control-plane CLI"
	case errors.As(err, &reason):
		if next, ok := sessionBindingFailureNext[reason.category]; ok {
			return prefix + " (owner proof " + reason.category + "): " + next
		}
		return prefix + " (owner proof " + reason.category + "); retry once, and ask the operator if it repeats"
	case errors.Is(err, errBindingProofUnavailable):
		return prefix + ": `vrooli agent identity-binding` failed without a recognised reason, which usually means the installed `vrooli` CLI is missing, outdated or unable to run it. An operator must check `vrooli agent identity-binding --help` on the host running Agent Manager"
	default:
		return prefix + ": the proof response was unusable; retry once, and ask the operator if it repeats"
	}
}

func readCLISessionBindingProof(parent context.Context, request SessionBindingRequest) (SessionBindingProof, error) {
	ctx, cancel := context.WithTimeout(parent, sessionBindingProofTimeout)
	defer cancel()
	args := []string{"agent", "identity-binding", "--harness-session-id", request.HarnessSessionID, "--harness-kind", request.HarnessKind, "--json"}
	if request.RunID != uuid.Nil {
		args = append(args, "--run-id", request.RunID.String(), "--run-pid", strconv.Itoa(request.ProcessID))
	}
	if request.WebConsoleSessionID != "" {
		args = append(args, "--web-console-session-id", request.WebConsoleSessionID)
	}
	cmd := exec.CommandContext(ctx, "vrooli", args...)
	cmd.WaitDelay = 100 * time.Millisecond
	cmd.Env = withoutAgentIdentity(os.Environ())
	var stdout limitedProofOutput
	var stderr limitedProofOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return SessionBindingProof{}, errBindingProofTimeout
		}
		return SessionBindingProof{}, classifiedBindingProofFailure(stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	dec.DisallowUnknownFields()
	var proof SessionBindingProof
	if err := dec.Decode(&proof); err != nil {
		return SessionBindingProof{}, errors.New("invalid session binding proof")
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return SessionBindingProof{}, errors.New("invalid session binding proof")
	}
	return proof, nil
}

func withoutAgentIdentity(environment []string) []string {
	out := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if key == cliutil.EnvIdentityToken {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// resolveAttachedBaselineRenewal renews only an externally attached root. Each
// generation carries ordinary identity with zero capabilities; expired owner
// delegation is retained as attribution and never renewed as authority.
func (o *Orchestrator) resolveAttachedBaselineRenewal(ctx context.Context, peerPID int, inherited string, anchorClaims *identity.Claims) (string, bool, error) {
	if anchorClaims == nil || len(o.identitySecret) == 0 {
		return "", false, nil
	}
	store, ok := o.runs.(repository.AttachedIdentityRenewalRepository)
	if !ok || store == nil || o.sessionBindingProofReader == nil {
		return "", false, nil
	}
	anchorHash := identity.HashToken(inherited)
	run, err := o.runs.GetByTokenHash(ctx, anchorHash)
	if err != nil {
		return "", false, identityRefreshUnavailable(err)
	}
	if run == nil {
		run, err = store.GetByIdentityRenewalAnchorHash(ctx, anchorHash)
		if err != nil {
			return "", false, identityRefreshUnavailable(err)
		}
		if run == nil {
			return "", false, nil
		}
	}
	if !renewableAttachedRoot(run, anchorHash, anchorClaims) {
		return "", false, nil
	}
	proof, err := o.sessionBindingProofReader.ReadSessionBinding(ctx, SessionBindingRequest{RunID: run.ID, HarnessSessionID: run.HarnessSessionID, WebConsoleSessionID: run.WebConsoleSessionID, HarnessKind: run.HarnessKind, ProcessID: run.RunnerPID})
	if err != nil {
		return "", false, identityRefreshUnavailable(err)
	}
	if !validSessionBindingProof(proof, run, peerPID) {
		return "", false, identityRefreshUnavailable(errBindingProofMismatch)
	}
	now := o.now().UTC()

	// A retry after an Agent Manager restart reconstructs the exact current
	// generation from the signed immutable anchor and persisted issue time.
	if run.IdentityRenewalAnchorHash != "" {
		if run.IdentityTokenIssuedAt == nil {
			return "", false, identityRefreshUnavailable(fmt.Errorf("renewed identity issue time missing"))
		}
		current, expiry, makeErr := attachedRenewalGeneration(anchorClaims, run.IdentityTokenIssuedAt.UTC(), o.identitySecret)
		if makeErr != nil || identity.HashToken(current) != run.IdentityTokenHash {
			return "", false, identityRefreshUnavailable(makeErr)
		}
		if now.Unix() < expiry.Unix() {
			return current, true, nil
		}
	}

	issuedAt := time.Unix(now.Unix(), 0).UTC()
	current, expiry, err := attachedRenewalGeneration(anchorClaims, issuedAt, o.identitySecret)
	if err != nil || !now.Before(expiry) {
		return "", false, nil
	}
	rotatedHash := identity.HashToken(current)
	updated, err := store.RenewAttachedIdentity(ctx, run, anchorHash, rotatedHash, issuedAt)
	if err != nil {
		return "", false, identityRefreshUnavailable(err)
	}
	if updated {
		return current, true, nil
	}

	// Another concurrent request may have won the same generation CAS. Return
	// only its reconstructible active token after rechecking the same proof.
	latest, err := store.GetByIdentityRenewalAnchorHash(ctx, anchorHash)
	if err != nil || latest == nil || !renewableAttachedRoot(latest, anchorHash, anchorClaims) {
		return "", false, identityRefreshUnavailable(err)
	}
	// The owner proof already re-read lease, scope, process incarnation and
	// Web Console session after its first observation. Avoid a second bounded
	// CLI call on the CAS-loser path: the persisted row must still match that
	// exact proof and original run snapshot before reconstruction.
	if !sameRenewalBinding(run, latest) || !validSessionBindingProof(proof, latest, peerPID) || latest.IdentityTokenIssuedAt == nil {
		return "", false, identityRefreshUnavailable(nil)
	}
	current, expiry, err = attachedRenewalGeneration(anchorClaims, latest.IdentityTokenIssuedAt.UTC(), o.identitySecret)
	if err != nil || identity.HashToken(current) != latest.IdentityTokenHash || !now.Before(expiry) {
		return "", false, identityRefreshUnavailable(err)
	}
	return current, true, nil
}

func identityRefreshUnavailable(cause error) error {
	refusal := &domain.AuthorizationError{Code: "identity_refresh_unavailable"}
	if cause == nil {
		return refusal
	}
	return fmt.Errorf("%w: %w", refusal, cause)
}

// Log only bounded categories: proof errors and database failures can contain
// caller data, and the broker's external refusal must not reveal it.
func identityRefreshDiagnostic(err error) string {
	var reason bindingProofReason
	if errors.As(err, &reason) {
		return "owner_proof_" + reason.category
	}
	switch {
	case errors.Is(err, errBindingProofTimeout):
		return "owner_proof_timeout"
	case errors.Is(err, errBindingProofUnavailable):
		return "owner_proof_unavailable"
	case errors.Is(err, errBindingProofMismatch):
		return "owner_proof_mismatch"
	default:
		return "other"
	}
}

func renewableAttachedRoot(run *domain.Run, anchorHash string, claims *identity.Claims) bool {
	if run == nil || claims == nil || run.ID == uuid.Nil || run.ID != claims.RunID || run.TaskID != claims.TaskID ||
		run.ExecutionMode.Normalized() != domain.ExecutionModeAttached || run.ParentRunID != nil ||
		(run.Status != domain.RunStatusStarting && run.Status != domain.RunStatusRunning) || run.CancelRequestedAt != nil ||
		run.IdentityTokenRevokedAt != nil || run.RunnerPID <= 1 || run.WebConsoleSessionID == "" ||
		run.HarnessKind == "" || run.HarnessSessionID == "" ||
		claims.Purpose != "" || claims.Subject != run.OwnerSubject || claims.WorkspaceID != "" || !slices.Equal(claims.Scopes, run.OwnerScopes) ||
		claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt || claims.ExpiresAt > claims.IssuedAt+int64(identity.DefaultTTL/time.Second) ||
		claims.ProfileKey != run.HarnessKind || claims.ScopePath != "" ||
		claims.Meta["execution_mode"] != string(domain.ExecutionModeAttached) ||
		claims.Meta["harness_kind"] != run.HarnessKind || claims.Meta["harness_session"] != run.HarnessSessionID ||
		claims.Meta["authorization_id"] != "" || len(claims.Meta) != 3 {
		return false
	}
	if run.OwnerExpiresAt != nil && claims.ExpiresAt > run.OwnerExpiresAt.Unix() {
		return false
	}
	if run.IdentityRenewalAnchorHash != "" && run.IdentityRenewalAnchorHash != anchorHash {
		return false
	}
	if run.IdentityRenewalAnchorHash == "" && run.IdentityTokenHash != anchorHash {
		return false
	}
	return true
}

func sameRenewalBinding(a, b *domain.Run) bool {
	if a == nil || b == nil || a.ID != b.ID || a.TaskID != b.TaskID || a.Status != b.Status || a.LifecycleVersion != b.LifecycleVersion ||
		a.OwnerIdentity != b.OwnerIdentity || a.OwnerEpoch != b.OwnerEpoch || a.CancelRequestedAt != nil || b.CancelRequestedAt != nil ||
		a.ExecutionMode.Normalized() != b.ExecutionMode.Normalized() || a.RunnerPID != b.RunnerPID || a.RunnerPGID != b.RunnerPGID ||
		a.HarnessKind != b.HarnessKind || a.HarnessSessionID != b.HarnessSessionID || a.WebConsoleSessionID != b.WebConsoleSessionID ||
		a.OwnerSubject != b.OwnerSubject || a.IdentityTokenRevokedAt != nil || b.IdentityTokenRevokedAt != nil || a.ParentRunID != nil || b.ParentRunID != nil ||
		!slices.Equal(a.OwnerScopes, b.OwnerScopes) {
		return false
	}
	if (a.OwnerExpiresAt == nil) != (b.OwnerExpiresAt == nil) {
		return false
	}
	if a.OwnerExpiresAt != nil && !a.OwnerExpiresAt.Equal(*b.OwnerExpiresAt) {
		return false
	}
	return true
}

func validSessionBindingProof(proof SessionBindingProof, run *domain.Run, peerPID int) bool {
	return proof.ProofVersion == sessionBindingProofVersion && run != nil &&
		proof.RunID == run.ID.String() && proof.HarnessSessionID == run.HarnessSessionID &&
		(proof.LeaseSessionID == run.HarnessSessionID || proof.LeaseSessionID == run.ID.String()) &&
		proof.WebConsoleSessionID == run.WebConsoleSessionID && proof.OwnerKind == "web-console-session" &&
		proof.HarnessKind == run.HarnessKind && proof.Binding.PID == run.RunnerPID &&
		proof.Binding.PID > 1 && proof.Binding.UID >= 0 && proof.Binding.Start != "" &&
		proof.Binding.Boot != "" && proof.Binding.Cgroup != "" && cliutil.ProcessDescendsFrom(peerPID, proof.Binding)
}

func validSessionBindingTuple(proof SessionBindingProof, runID uuid.UUID, harnessSessionID, expectedWebConsoleSessionID, harnessKind string, binding cliutil.ProcessBinding) bool {
	expectedRunID := ""
	if runID != uuid.Nil {
		expectedRunID = runID.String()
	}
	if proof.ProofVersion != sessionBindingProofVersion || proof.RunID != expectedRunID ||
		proof.HarnessSessionID != harnessSessionID || (proof.LeaseSessionID != harnessSessionID && proof.LeaseSessionID != expectedRunID) || proof.LeaseSessionID == "" || proof.HarnessKind != harnessKind ||
		proof.OwnerKind == "" || proof.Binding != binding || binding.PID <= 1 || binding.UID < 0 ||
		binding.Start == "" || binding.Boot == "" || binding.Cgroup == "" {
		return false
	}
	if expectedWebConsoleSessionID != "" && proof.WebConsoleSessionID != expectedWebConsoleSessionID {
		return false
	}
	return proof.WebConsoleSessionID == "" || proof.OwnerKind == "web-console-session"
}

func attachedRenewalGeneration(anchor *identity.Claims, issuedAt time.Time, secret []byte) (string, time.Time, error) {
	if anchor == nil || issuedAt.IsZero() {
		return "", time.Time{}, fmt.Errorf("attached identity renewal claims unavailable")
	}
	claims := *anchor
	claims.Scopes = []string{}
	claims.IssuedAt = issuedAt.Unix()
	claims.ExpiresAt = issuedAt.Add(identity.DefaultTTL).Unix()
	expires := time.Unix(claims.ExpiresAt, 0).UTC()
	token, err := identity.GenerateToken(&claims, secret)
	return token, expires, err
}
