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
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

const sessionBindingProofVersion = 1
const sessionBindingProofTimeout = 3 * time.Second
const sessionBindingProofMaxBytes = 4096

// SessionBindingProof is the small control-plane result required to renew an
// attached root's ordinary identity. The owner command returns it only after
// exact lease, scope, kernel-incarnation, Web Console and reread fences pass.
type SessionBindingProof struct {
	ProofVersion        int                    `json:"proof_version"`
	RunID               string                 `json:"run_id"`
	HarnessSessionID    string                 `json:"harness_session_id"`
	WebConsoleSessionID string                 `json:"web_console_session_id"`
	HarnessKind         string                 `json:"harness_kind"`
	OwnerKind           string                 `json:"owner_kind"`
	Binding             cliutil.ProcessBinding `json:"binding"`
}

// SessionBindingProofReader is deliberately narrow and read-only.
type SessionBindingProofReader interface {
	ReadSessionBinding(context.Context, uuid.UUID, string, string, string) (SessionBindingProof, error)
}

type SessionBindingProofReaderFunc func(context.Context, uuid.UUID, string, string, string) (SessionBindingProof, error)

func (f SessionBindingProofReaderFunc) ReadSessionBinding(ctx context.Context, runID uuid.UUID, harnessSessionID, webConsoleSessionID, harnessKind string) (SessionBindingProof, error) {
	return f(ctx, runID, harnessSessionID, webConsoleSessionID, harnessKind)
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

func readCLISessionBindingProof(parent context.Context, runID uuid.UUID, harnessSessionID, webConsoleSessionID, harnessKind string) (SessionBindingProof, error) {
	ctx, cancel := context.WithTimeout(parent, sessionBindingProofTimeout)
	defer cancel()
	args := []string{"agent", "identity-binding", "--harness-session-id", harnessSessionID, "--harness-kind", harnessKind, "--json"}
	if runID != uuid.Nil {
		args = append(args, "--run-id", runID.String())
	}
	if webConsoleSessionID != "" {
		args = append(args, "--web-console-session-id", webConsoleSessionID)
	}
	cmd := exec.CommandContext(ctx, "vrooli", args...)
	cmd.WaitDelay = 100 * time.Millisecond
	cmd.Env = withoutAgentIdentity(os.Environ())
	var stdout limitedProofOutput
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return SessionBindingProof{}, errors.New("session binding proof unavailable")
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

// resolveAttachedBaselineRenewal rotates only an ownerless zero-scope attached
// root. The inherited token is expired when this path is entered; any returned
// replacement remains subject to ordinary expiry-strict API verification.
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
	proof, err := o.sessionBindingProofReader.ReadSessionBinding(ctx, run.ID, run.HarnessSessionID, run.WebConsoleSessionID, run.HarnessKind)
	if err != nil || !validSessionBindingProof(proof, run, peerPID) {
		return "", false, identityRefreshUnavailable(err)
	}
	now := o.now().UTC()
	if run.OwnerExpiresAt != nil && !now.Before(*run.OwnerExpiresAt) {
		return "", false, nil
	}

	// A retry after an Agent Manager restart reconstructs the exact current
	// generation from the signed immutable anchor and persisted issue time.
	if run.IdentityTokenIssuedAt != nil {
		current, expiry, makeErr := attachedRenewalGeneration(anchorClaims, run.IdentityTokenIssuedAt.UTC(), run.OwnerExpiresAt, o.identitySecret)
		if makeErr != nil || identity.HashToken(current) != run.IdentityTokenHash {
			return "", false, identityRefreshUnavailable(makeErr)
		}
		if now.Unix() < expiry.Unix() {
			return current, true, nil
		}
	} else if run.IdentityRenewalAnchorHash != "" {
		// Once the original anchor has been rotated away from IdentityTokenHash,
		// an absent active-generation timestamp means the current token cannot be
		// reconstructed. Never guess a replacement generation from corrupt state.
		return "", false, identityRefreshUnavailable(fmt.Errorf("renewed identity issue time missing"))
	}

	issuedAt := time.Unix(now.Unix(), 0).UTC()
	current, expiry, err := attachedRenewalGeneration(anchorClaims, issuedAt, run.OwnerExpiresAt, o.identitySecret)
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
	current, expiry, err = attachedRenewalGeneration(anchorClaims, latest.IdentityTokenIssuedAt.UTC(), latest.OwnerExpiresAt, o.identitySecret)
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
	return fmt.Errorf("%w: %v", refusal, cause)
}

func renewableAttachedRoot(run *domain.Run, anchorHash string, claims *identity.Claims) bool {
	if run == nil || claims == nil || run.ID == uuid.Nil || run.ID != claims.RunID || run.TaskID != claims.TaskID ||
		run.ExecutionMode.Normalized() != domain.ExecutionModeAttached || run.ParentRunID != nil ||
		(run.Status != domain.RunStatusStarting && run.Status != domain.RunStatusRunning) || run.CancelRequestedAt != nil ||
		run.IdentityTokenRevokedAt != nil || run.RunnerPID <= 1 || run.WebConsoleSessionID == "" ||
		run.HarnessKind == "" || run.HarnessSessionID == "" || run.OwnerSubject != "" || len(run.OwnerScopes) != 0 ||
		claims.Purpose != "" || claims.Subject != "" || claims.WorkspaceID != "" || len(claims.Scopes) != 0 ||
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
		len(a.OwnerScopes) != 0 || len(b.OwnerScopes) != 0 {
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
		proof.HarnessSessionID != harnessSessionID || proof.HarnessKind != harnessKind ||
		proof.OwnerKind == "" || proof.Binding != binding || binding.PID <= 1 || binding.UID < 0 ||
		binding.Start == "" || binding.Boot == "" || binding.Cgroup == "" {
		return false
	}
	if expectedWebConsoleSessionID != "" && proof.WebConsoleSessionID != expectedWebConsoleSessionID {
		return false
	}
	return proof.WebConsoleSessionID == "" || proof.OwnerKind == "web-console-session"
}

func attachedRenewalGeneration(anchor *identity.Claims, issuedAt time.Time, ownerExpiresAt *time.Time, secret []byte) (string, time.Time, error) {
	if anchor == nil || issuedAt.IsZero() {
		return "", time.Time{}, fmt.Errorf("attached identity renewal claims unavailable")
	}
	claims := *anchor
	claims.IssuedAt = issuedAt.Unix()
	claims.ExpiresAt = issuedAt.Add(identity.DefaultTTL).Unix()
	expires := time.Unix(claims.ExpiresAt, 0).UTC()
	if ownerExpiresAt != nil && expires.After(*ownerExpiresAt) {
		expires = time.Unix(ownerExpiresAt.Unix(), 0).UTC()
		claims.ExpiresAt = expires.Unix()
	}
	token, err := identity.GenerateToken(&claims, secret)
	return token, expires, err
}
