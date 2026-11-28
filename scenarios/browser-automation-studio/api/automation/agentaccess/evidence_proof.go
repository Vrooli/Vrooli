package agentaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/principalrole"
)

const (
	BrowserEvidenceVerifierAudience = "scenario-authenticator:browser-automation-studio"
	BrowserEvidenceVerifierScope    = "browser-automation-studio:browser-evidence.verify"
	BrowserEvidenceVerifierSuffix   = "-browser-evidence-verifier"
	BrowserEvidenceReporterAudience = "scenario-authenticator:agent-manager"
	BrowserEvidenceReporterScope    = "agent-manager:write"
	BrowserEvidenceReporterSuffix   = "-agent-manager"
	BrowserEvidenceProofTTL         = 5 * time.Minute
)

// BrowserEvidenceVerifierRole is the narrowly scoped Agent Manager identity
// used only for callbacks that inspect or consume a closed BAS broker proof.
var BrowserEvidenceVerifierRole = principalrole.NewRole(
	BrowserEvidenceVerifierAudience,
	BrowserEvidenceVerifierScope,
	"browser-automation-studio",
	BrowserEvidenceVerifierSuffix,
	"browser evidence verifier",
)

// BrowserEvidenceReporterRole is the existing BAS→Agent Manager service role
// declared in BAS .vrooli/service-principals.json.
var BrowserEvidenceReporterRole = principalrole.NewRole(
	BrowserEvidenceReporterAudience,
	BrowserEvidenceReporterScope,
	"agent-manager",
	BrowserEvidenceReporterSuffix,
	"Agent Manager browser evidence reporter",
)

func NewBrowserEvidenceTokenVerifier() authn.TokenVerifier {
	return BrowserEvidenceVerifierRole.NewVerifier()
}

type BrowserEvidenceObservation struct {
	Kind           string                  `json:"kind"`
	SessionID      string                  `json:"session_id"`
	State          string                  `json:"state"`
	SessionMinutes int64                   `json:"session_minutes"`
	Actions        int64                   `json:"actions"`
	Bytes          int64                   `json:"bytes"`
	Runtime        *BrowserRuntimeEvidence `json:"runtime,omitempty"`
}

type closedBrowserEvidenceProof struct {
	Reference    string
	RunID        uuid.UUID
	GrantReceipt string
	SessionID    string
	LeaseID      string
	OwnerID      string
	CallerID     string
	ProfileID    string
	Backend      string
	ClosedAt     time.Time
	ExpiresAt    time.Time
	Observation  BrowserEvidenceObservation
	Digest       string
	Consumed     bool
}

var ErrBrowserEvidenceProof = errors.New("browser evidence proof is missing, expired, or not bound to this closed session")

func evidencePayloadSize(value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(data)
}

func (b *Broker) issueClosedEvidenceProof(e *entry, now time.Time) error {
	if e == nil || !e.closed || e.closedAt.IsZero() || e.sessionID == "" || e.leaseID == "" || e.callerID == "" || e.grantReceipt == "" || e.permissionRef == "" {
		return errors.New("closed browser session lacks broker-owned proof bindings")
	}
	runID, err := browserRunID(e.callerID)
	if err != nil {
		return err
	}
	reference, err := randomBridgeToken()
	if err != nil {
		return err
	}
	minutes := int64(now.Sub(e.openedAt) / time.Minute)
	if minutes < 0 {
		minutes = 0
	}
	observation := BrowserEvidenceObservation{
		Kind: "session", SessionID: e.sessionID, State: "closed",
		SessionMinutes: minutes, Actions: e.evidenceActions, Bytes: e.evidenceBytes,
		Runtime: e.runtimeEvidence,
	}
	proof := &closedBrowserEvidenceProof{
		Reference: reference, RunID: runID, GrantReceipt: e.grantReceipt,
		SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID,
		CallerID: e.callerID, ProfileID: e.permissionRef, Backend: e.backend,
		ClosedAt: now, ExpiresAt: now.Add(BrowserEvidenceProofTTL), Observation: observation,
	}
	proof.Digest = closedEvidenceDigest(proof)
	b.mu.Lock()
	e.evidenceProof = proof
	b.mu.Unlock()
	time.AfterFunc(BrowserEvidenceProofTTL, func() {
		e.opMu.Lock()
		defer e.opMu.Unlock()
		b.mu.Lock()
		if b.entries[e.sessionID] == e && e.closed && e.evidenceProof == proof && !time.Now().Before(proof.ExpiresAt) {
			delete(b.entries, e.sessionID)
		}
		b.mu.Unlock()
	})
	return nil
}

// VerifyBrowserEvidenceProof re-resolves a short-lived proof reference only
// from the existing Broker entry map. Its request carries no session ID or
// counters: run and grant must match the bindings captured at close.
func (b *Broker) VerifyBrowserEvidenceProof(_ context.Context, runID uuid.UUID, grantReceipt, reference string) (BrowserEvidenceObservation, error) {
	e, proof := b.findEvidenceProof(reference)
	if e == nil || proof == nil {
		return BrowserEvidenceObservation{}, ErrBrowserEvidenceProof
	}
	e.opMu.Lock()
	defer e.opMu.Unlock()
	b.mu.RLock()
	owned := b.entries[e.sessionID] == e
	b.mu.RUnlock()
	if !owned || !validClosedEvidenceProof(e, proof, runID, grantReceipt, reference, time.Now()) {
		return BrowserEvidenceObservation{}, ErrBrowserEvidenceProof
	}
	if e.session != nil {
		ownerID, leaseID := e.session.OwnerIdentity()
		if ownerID != proof.OwnerID || leaseID != proof.LeaseID {
			return BrowserEvidenceObservation{}, ErrBrowserEvidenceProof
		}
	}
	return proof.Observation, nil
}

// ConsumeBrowserEvidenceProof marks successful AM ingestion. It is idempotent
// for the same still-retained proof/run so an AM response loss can be retried.
func (b *Broker) ConsumeBrowserEvidenceProof(_ context.Context, runID uuid.UUID, reference string) error {
	e, proof := b.findEvidenceProof(reference)
	if e == nil || proof == nil {
		return ErrBrowserEvidenceProof
	}
	e.opMu.Lock()
	defer e.opMu.Unlock()
	b.mu.RLock()
	owned := b.entries[e.sessionID] == e
	b.mu.RUnlock()
	if !owned || !validClosedEvidenceProof(e, proof, runID, proof.GrantReceipt, reference, time.Now()) {
		return ErrBrowserEvidenceProof
	}
	proof.Consumed = true
	return nil
}

func (b *Broker) findEvidenceProof(reference string) (*entry, *closedBrowserEvidenceProof) {
	if strings.TrimSpace(reference) == "" || len(reference) > 256 {
		return nil, nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, e := range b.entries {
		if e.evidenceProof != nil && e.evidenceProof.Reference == reference {
			return e, e.evidenceProof
		}
	}
	return nil, nil
}

func validClosedEvidenceProof(e *entry, p *closedBrowserEvidenceProof, runID uuid.UUID, grantReceipt, reference string, now time.Time) bool {
	if e == nil || p == nil || !e.closed || p.Reference != reference || p.RunID != runID || p.GrantReceipt != grantReceipt ||
		p.SessionID != e.sessionID || p.LeaseID != e.leaseID || p.OwnerID != e.ownerID || p.CallerID != e.callerID ||
		p.ProfileID != e.permissionRef || p.Backend != e.backend || !p.ExpiresAt.After(now) || p.ClosedAt.IsZero() ||
		p.Observation.Kind != "session" || p.Observation.State != "closed" || p.Observation.SessionID != e.sessionID ||
		p.Observation.SessionMinutes < 0 || p.Observation.Actions < 0 || p.Observation.Bytes < 0 {
		return false
	}
	return p.Digest == closedEvidenceDigest(p)
}

func closedEvidenceDigest(p *closedBrowserEvidenceProof) string {
	if p == nil {
		return ""
	}
	material, _ := json.Marshal(struct {
		Reference, RunID, GrantReceipt, SessionID, LeaseID, OwnerID, CallerID, ProfileID, Backend string
		ClosedAt, ExpiresAt                                                                       time.Time
		Observation                                                                               BrowserEvidenceObservation
	}{
		Reference: p.Reference, RunID: p.RunID.String(), GrantReceipt: p.GrantReceipt, SessionID: p.SessionID,
		LeaseID: p.LeaseID, OwnerID: p.OwnerID, CallerID: p.CallerID, ProfileID: p.ProfileID,
		Backend: p.Backend, ClosedAt: p.ClosedAt, ExpiresAt: p.ExpiresAt, Observation: p.Observation,
	})
	digest := sha256.Sum256(material)
	return hex.EncodeToString(digest[:])
}

func browserRunID(callerID string) (uuid.UUID, error) {
	const prefix = "run:"
	value, ok := strings.CutPrefix(strings.TrimSpace(callerID), prefix)
	if !ok {
		return uuid.Nil, fmt.Errorf("browser caller is not bound to a run principal")
	}
	runID, err := uuid.Parse(value)
	if err != nil || runID == uuid.Nil {
		return uuid.Nil, errors.New("browser caller run principal is invalid")
	}
	return runID, nil
}
