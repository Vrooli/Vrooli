package agentsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
)

// ExactReapProof binds a human decision to one owner-gone session scope and
// the exact process incarnation set observed inside it. It is a decision
// binding, not authority by itself; ApprovalVerifier must validate the
// approval at the owning authority before a stop is allowed.
type ExactReapProof struct {
	Operation     string             `json:"operation"`
	Target        string             `json:"target"`
	SessionID     string             `json:"session_id"`
	BindingID     string             `json:"binding_id,omitempty"`
	LeaseStatus   string             `json:"lease_status,omitempty"`
	OwnerKind     string             `json:"owner_kind"`
	OwnerRef      string             `json:"owner_ref"`
	OwnerEvidence string             `json:"owner_evidence_sha256"`
	CgroupPath    string             `json:"cgroup_path"`
	Processes     []ExactProcessFact `json:"processes"`
	Digest        string             `json:"digest"`
}

// ExactReapAuthorizationTarget encodes both the exact systemd scope and the
// reviewed process-incarnation digest. The grant therefore expires with this
// single observed scope generation and cannot be replayed for a newer one.
func ExactReapAuthorizationTarget(proof ExactReapProof) string {
	if proof.Target == "" || proof.Digest == "" || proof.Digest != exactReapDigest(proof) {
		return ""
	}
	return "agent-session:reap/" + proof.Target + "/proof/" + proof.Digest
}

type ExactProcessFact struct {
	PID       int    `json:"pid"`
	StartTime string `json:"start_time"`
	Name      string `json:"name"`
}

// ExactScopeFacts contains only read-only host observations needed to prove
// the containment identity immediately before an exact stop.
type ExactScopeFacts interface {
	ScopeCgroupPath(scopeUnit string) (string, error)
	ScopeMembers(scopeUnit string) ([]int, error)
	Process(pid int) (ProcessFact, error)
	ProcessName(pid int) (string, error)
}

// NewExactScopeFacts exposes the read-only host proofs needed by the exact
// remediation path without widening the general session observation API.
func NewExactScopeFacts() ExactScopeFacts { return &hostFacts{} }

func (h *hostFacts) ScopeCgroupPath(scopeUnit string) (string, error) {
	return ScopeCgroupPath(scopeUnit)
}

// CaptureExactReapProof requires the exact scope string, a held orphaned
// scope, a known owner authority that confirmed it gone, and a non-empty,
// readable current process set. The result can be shown to the operator for
// an exact-target decision. Unknown Web Console origin is allowed to reach
// this review stage; it does not make the candidate automatically reapable.
func CaptureExactReapProof(session Session, exactScope string, facts ExactScopeFacts) (ExactReapProof, error) {
	if facts == nil {
		return ExactReapProof{}, errors.New("exact reap: host facts are unavailable")
	}
	unit := session.ScopeUnit()
	if unit == "" || exactScope != unit || session.Scope != exactScope {
		return ExactReapProof{}, errors.New("exact reap: selector must equal the session's complete scope unit")
	}
	if !session.ScopeHeld || session.State != StateOrphaned || session.Liveness.State != OwnerGone || !session.Owner.Known() {
		return ExactReapProof{}, errors.New("exact reap: requires a held orphaned scope and owner authority confirming gone")
	}
	if session.Frozen != "no" {
		return ExactReapProof{}, errors.New("exact reap: scope must be positively observed unfrozen")
	}
	path, err := facts.ScopeCgroupPath(unit)
	if err != nil {
		return ExactReapProof{}, fmt.Errorf("exact reap: canonical cgroup path unavailable: %w", err)
	}
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "..") || !strings.HasSuffix(path, "/"+agentscope.Slice+"/"+unit) {
		return ExactReapProof{}, errors.New("exact reap: canonical cgroup path is invalid")
	}
	members, err := facts.ScopeMembers(unit)
	if err != nil {
		return ExactReapProof{}, fmt.Errorf("exact reap: scope membership is unreadable: %w", err)
	}
	if len(members) == 0 {
		return ExactReapProof{}, errors.New("exact reap: scope membership is empty")
	}
	sort.Ints(members)
	proof := ExactReapProof{
		Operation: "agent-session.reap", Target: exactScope, SessionID: session.SessionID,
		BindingID: session.BindingID, LeaseStatus: session.LeaseStatus,
		OwnerKind: session.Owner.Kind, OwnerRef: session.Owner.Ref, CgroupPath: path,
	}
	ownerBytes, err := json.Marshal(session.Owner)
	if err != nil {
		return ExactReapProof{}, fmt.Errorf("exact reap: encode owner evidence: %w", err)
	}
	ownerHash := sha256.Sum256(ownerBytes)
	proof.OwnerEvidence = hex.EncodeToString(ownerHash[:])
	for i, pid := range members {
		if pid <= 0 || (i > 0 && members[i-1] == pid) {
			return ExactReapProof{}, errors.New("exact reap: scope membership contains an invalid or duplicate pid")
		}
		fact, factErr := facts.Process(pid)
		if factErr != nil {
			return ExactReapProof{}, fmt.Errorf("exact reap: process incarnation for pid %d is unreadable: %w", pid, factErr)
		}
		if fact.PID != pid || fact.StartTime == "" {
			return ExactReapProof{}, fmt.Errorf("exact reap: process incarnation for pid %d is invalid", pid)
		}
		name, nameErr := facts.ProcessName(pid)
		if nameErr != nil {
			return ExactReapProof{}, fmt.Errorf("exact reap: process name for pid %d is unreadable: %w", pid, nameErr)
		}
		if strings.TrimSpace(name) == "" {
			return ExactReapProof{}, fmt.Errorf("exact reap: process name for pid %d is empty", pid)
		}
		if isCodingAgentProcess(name) {
			return ExactReapProof{}, fmt.Errorf("exact reap: scope contains protected coding-agent process %q (pid %d)", name, pid)
		}
		proof.Processes = append(proof.Processes, ExactProcessFact{PID: pid, StartTime: fact.StartTime, Name: name})
	}
	proof.Digest = exactReapDigest(proof)
	return proof, nil
}

// ExactReapApproval is an authority receipt. The digest, operation, and
// target must match the current proof exactly; the caller still must validate
// it with the canonical approval authority.
type ExactReapApproval struct {
	DecisionID  string    `json:"decision_id"`
	ApprovedBy  string    `json:"approved_by"`
	Operation   string    `json:"operation"`
	Target      string    `json:"target"`
	ProofDigest string    `json:"proof_digest"`
	VerifiedAt  time.Time `json:"verified_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ApprovalVerifier interface {
	VerifyExactApproval(context.Context, ExactReapProof) (ExactReapApproval, error)
}

// ExactSessionReader must freshly reconcile one exact scope and its owner
// authority. It cannot return a cached plan/session object.
type ExactSessionReader func(context.Context, string) (Session, error)

// ExecuteExactReap revalidates the captured identity after the authority
// approves the exact target, verifies the authority receipt, then captures
// the identity again immediately before using the existing contained stopper.
// It never selects by PID, prefix, age, or a wildcard.
func ExecuteExactReap(ctx context.Context, proof ExactReapProof, facts ExactScopeFacts, verifier ApprovalVerifier, refresh ExactSessionReader, stopper Stopper) (ExactReapApproval, error) {
	if verifier == nil || refresh == nil || stopper == nil {
		return ExactReapApproval{}, errors.New("exact reap: verifier, fresh session reader, and contained stopper are required")
	}
	if proof.Operation != "agent-session.reap" || proof.Target == "" || proof.Digest == "" || proof.Digest != exactReapDigest(proof) {
		return ExactReapApproval{}, errors.New("exact reap: invalid or altered proof")
	}
	approval, err := verifier.VerifyExactApproval(ctx, proof)
	if err != nil {
		return ExactReapApproval{}, fmt.Errorf("exact reap: approval authority rejected receipt: %w", err)
	}
	now := time.Now().UTC()
	if approval.DecisionID == "" || approval.ApprovedBy == "" || approval.Operation != proof.Operation || approval.Target != ExactReapAuthorizationTarget(proof) || approval.ProofDigest != proof.Digest || approval.VerifiedAt.IsZero() || approval.ExpiresAt.IsZero() || approval.VerifiedAt.After(now.Add(time.Minute)) || !approval.ExpiresAt.After(now) || !approval.ExpiresAt.After(approval.VerifiedAt) {
		return ExactReapApproval{}, errors.New("exact reap: approval is not bound to this operation, target, and proof")
	}
	currentSession, err := refresh(ctx, proof.Target)
	if err != nil {
		return ExactReapApproval{}, fmt.Errorf("exact reap: fresh session/owner reconciliation failed: %w", err)
	}
	current, err := CaptureExactReapProof(currentSession, proof.Target, facts)
	if err != nil {
		return ExactReapApproval{}, err
	}
	if current.Digest != proof.Digest {
		return ExactReapApproval{}, errors.New("exact reap: cgroup or process identity changed since approval")
	}
	// Stopper resolves the exact unit beneath the canonical Vrooli agent slice;
	// a user-provided cgroup path is never sent to the platform layer.
	if err := stopper.Stop(ctx, currentSession); err != nil {
		return ExactReapApproval{}, err
	}
	return approval, nil
}

func exactReapDigest(proof ExactReapProof) string {
	proof.Digest = ""
	data, _ := json.Marshal(proof)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func isCodingAgentProcess(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, agent := range []string{"codex", "claude", "opencode", "gemini", "aider", "cursor", "grok"} {
		if name == agent || strings.HasPrefix(name, agent+"-") || strings.HasPrefix(name, agent+"_") {
			return true
		}
	}
	return false
}
