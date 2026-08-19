package pairing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

// Code/TTL policy. Codes are single-use and short-lived: a live code can enrol a
// rogue node (SECURITY.md), so the window is deliberately tight.
const (
	defaultCodeTTL = 15 * time.Minute
	maxCodeTTL     = 1 * time.Hour
	codeBytes      = 20 // 160 bits of entropy → 32 base32 chars
)

// Service is the pairing domain's application logic: it mints/burns codes,
// registers redeeming nodes (via the NodeRegistrar seam), stores their Ed25519
// public keys, and runs the request/approve fallback.
type Service struct {
	repo           Repository
	registrar      NodeRegistrar
	clock          schedule.Clock
	validateScopes func([]string) error
}

type Option func(*Service)

func WithGrantValidator(validate func([]string) error) Option {
	return func(s *Service) { s.validateScopes = validate }
}

// NewService constructs the pairing service.
func NewService(repo Repository, registrar NodeRegistrar, clk schedule.Clock, opts ...Option) *Service {
	s := &Service{repo: repo, registrar: registrar, clock: clk}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// IssuedCode is the result of IssueCode: the plaintext code (returned ONCE) and
// its expiry.
type IssuedCode struct {
	Code      string
	ExpiresAt time.Time
}

// IssueCode mints a single-use pairing code, stores only its hash, and returns
// the plaintext once. ttl<=0 uses the default; it is clamped to maxCodeTTL.
func (s *Service) IssueCode(ctx context.Context, name string, scopes []string, ttl time.Duration) (IssuedCode, error) {
	return s.issueCode(ctx, name, scopes, ttl, "")
}

// IssueCodeForEnrollment issues a code bound to one opaque EnrollmentAttempt
// correlation. The plaintext remains transient, while the correlation survives
// redemption and every later bootstrap failure.
func (s *Service) IssueCodeForEnrollment(ctx context.Context, name string, scopes []string, ttl time.Duration, correlationID string) (IssuedCode, error) {
	if strings.TrimSpace(correlationID) == "" {
		return IssuedCode{}, ErrInvalid{Field: "correlation_id", Reason: "is required"}
	}
	return s.issueCode(ctx, name, scopes, ttl, strings.TrimSpace(correlationID))
}

func (s *Service) issueCode(ctx context.Context, name string, scopes []string, ttl time.Duration, correlationID string) (IssuedCode, error) {
	if err := s.validateGrantScopes(scopes); err != nil {
		return IssuedCode{}, err
	}
	if ttl <= 0 {
		ttl = defaultCodeTTL
	}
	if ttl > maxCodeTTL {
		ttl = maxCodeTTL
	}
	code, err := generateCode()
	if err != nil {
		return IssuedCode{}, fmt.Errorf("generate pairing code: %w", err)
	}
	now := s.clock.Now().UTC()
	expires := now.Add(ttl)
	if _, err := s.repo.CreateCode(ctx, PairingCode{
		CodeHash:      hashCode(code),
		Name:          strings.TrimSpace(name),
		Scopes:        normalizeScopes(scopes),
		CorrelationID: correlationID,
		CreatedAt:     now,
		ExpiresAt:     expires,
	}); err != nil {
		return IssuedCode{}, err
	}
	return IssuedCode{Code: code, ExpiresAt: expires}, nil
}

// Redeem burns a valid code and enrols the redeeming node: it registers the
// durable node record (carrying the code's name/scopes merged with the node's
// self-reported facts), stores the node's Ed25519 public key, and marks the
// code used. Returns the new node id. An expired/unknown/already-used code is
// rejected before any node is created.
func (s *Service) Redeem(ctx context.Context, code, nodePublicKeyB64 string, facts NodeFacts) (nodeID string, err error) {
	if strings.TrimSpace(code) == "" {
		return "", ErrInvalid{Field: "code", Reason: "is required"}
	}
	if err := validatePublicKey(nodePublicKeyB64); err != nil {
		return "", err
	}

	stored, err := s.repo.GetCodeByHash(ctx, hashCode(code))
	if err != nil {
		return "", err // ErrCodeNotFound
	}
	if stored.Redeemed() {
		if stored.CorrelationID != "" {
			if repo, ok := s.repo.(EnrollmentRepository); ok {
				saga, sagaErr := repo.GetEnrollmentSaga(ctx, stored.CorrelationID)
				if sagaErr == nil && saga.State == "completed" {
					return saga.NodeID, nil
				}
			}
		}
		return "", ErrCodeUsed
	}
	if s.clock.Now().UTC().After(stored.ExpiresAt) {
		return "", ErrCodeExpired
	}
	if stored.CorrelationID != "" {
		return s.redeemCorrelated(ctx, stored, nodePublicKeyB64, facts)
	}
	if nodeID, ok, err := s.existingNodeByPublicKey(ctx, nodePublicKeyB64); err != nil {
		return "", err
	} else if ok {
		if err := s.repo.BurnCode(ctx, stored.ID, nodeID); err != nil {
			return "", err
		}
		return nodeID, nil
	}

	// The owner-assigned name/scopes from the code win; os/arch/endpoint/caps are
	// the node's self-report.
	regFacts := facts
	if stored.Name != "" {
		regFacts.Name = stored.Name
	}
	regFacts.Scopes = normalizeScopes(stored.Scopes)

	nodeID, err = s.registrar.RegisterNode(ctx, regFacts)
	if err != nil {
		return "", fmt.Errorf("register node: %w", err)
	}
	if err := s.repo.StoreCredential(ctx, Credential{NodeID: nodeID, PublicKey: nodePublicKeyB64, CreatedAt: s.clock.Now().UTC()}); err != nil {
		return "", fmt.Errorf("store credential: %w", err)
	}
	// Atomic single-use gate: a concurrent second redeem loses here.
	if err := s.repo.BurnCode(ctx, stored.ID, nodeID); err != nil {
		return "", err // ErrCodeUsed
	}
	return nodeID, nil
}

func (s *Service) redeemCorrelated(ctx context.Context, code PairingCode, publicKey string, facts NodeFacts) (string, error) {
	repo, ok := s.repo.(EnrollmentRepository)
	if !ok {
		return "", fmt.Errorf("correlated pairing requires enrollment-capable repository")
	}
	registrar, ok := s.registrar.(CorrelatedNodeRegistrar)
	if !ok {
		return "", fmt.Errorf("correlated pairing requires correlation-capable node registrar")
	}
	saga, err := repo.PrepareEnrollmentSaga(ctx, EnrollmentSaga{CorrelationID: code.CorrelationID, CodeID: code.ID, PublicKey: publicKey, Facts: facts})
	if err != nil {
		return "", err
	}
	if saga.CodeID != code.ID || saga.PublicKey != publicKey {
		return "", ErrInvalid{Field: "correlation_id", Reason: "does not match the original redemption"}
	}
	if saga.State == "completed" {
		return saga.NodeID, nil
	}
	if nodeID, ok, err := s.existingNodeByPublicKey(ctx, publicKey); err != nil {
		return "", err
	} else if ok && saga.State == "prepared" {
		// Reconciliation is still a normal correlated saga: claim and finalize
		// the newly issued code, but bind it to the existing credential instead
		// of creating a duplicate registry node.
		if err := repo.ClaimCode(ctx, code.ID); err != nil && err != ErrCodeUsed {
			return "", err
		}
		saga.NodeID, saga.State = nodeID, "claimed"
		if err := repo.UpdateEnrollmentSaga(ctx, saga); err != nil {
			return "", err
		}
		return s.continueEnrollment(ctx, repo, registrar, saga)
	}
	if saga.State == "prepared" {
		if err := repo.ClaimCode(ctx, code.ID); err != nil && err != ErrCodeUsed {
			return "", err
		}
		saga.State = "claimed"
		if err := repo.UpdateEnrollmentSaga(ctx, saga); err != nil {
			return "", err
		}
	}
	return s.continueEnrollment(ctx, repo, registrar, saga)
}

func (s *Service) existingNodeByPublicKey(ctx context.Context, publicKey string) (string, bool, error) {
	lookup, ok := s.repo.(interface {
		ActiveNodeByPublicKey(context.Context, string) (string, bool, error)
	})
	if !ok {
		return "", false, nil
	}
	nodeID, found, err := lookup.ActiveNodeByPublicKey(ctx, publicKey)
	if err != nil {
		return "", false, fmt.Errorf("lookup existing node credential: %w", err)
	}
	return nodeID, found, nil
}

func (s *Service) continueEnrollment(ctx context.Context, repo EnrollmentRepository, registrar CorrelatedNodeRegistrar, saga EnrollmentSaga) (string, error) {
	if saga.State == "completed" {
		return saga.NodeID, nil
	}
	nodeID := saga.NodeID
	if nodeID == "" {
		known, err := registrar.FindNodeByPairingCorrelation(ctx, saga.CorrelationID)
		if err == nil {
			nodeID = known
		} else {
			nodeID, err = registrar.RegisterNodeWithCorrelation(ctx, saga.Facts, saga.CorrelationID)
			if err != nil {
				saga.LastError = err.Error()
				_ = repo.UpdateEnrollmentSaga(ctx, saga)
				return "", fmt.Errorf("register correlated node: %w", err)
			}
		}
		saga.NodeID, saga.State, saga.LastError = nodeID, "node_registered", ""
		if err := repo.UpdateEnrollmentSaga(ctx, saga); err != nil {
			return "", err
		}
	}
	if err := s.repo.StoreCredential(ctx, Credential{NodeID: nodeID, PublicKey: saga.PublicKey, CreatedAt: s.clock.Now().UTC()}); err != nil {
		saga.LastError = err.Error()
		_ = repo.UpdateEnrollmentSaga(ctx, saga)
		return "", fmt.Errorf("store credential: %w", err)
	}
	if err := repo.FinalizeClaimedCode(ctx, saga.CodeID, nodeID); err != nil && err != ErrCodeUsed {
		return "", err
	}
	saga.State, saga.NodeID, saga.LastError, saga.CompletedAt = "completed", nodeID, "", s.clock.Now().UTC()
	if err := repo.UpdateEnrollmentSaga(ctx, saga); err != nil {
		return "", err
	}
	return nodeID, nil
}

// ReconcileEnrollments resumes every incomplete correlated pairing saga after a
// restart. Registry correlation makes the operation idempotent even if the
// prior process died after creating the Node and before recording its ID.
func (s *Service) ReconcileEnrollments(ctx context.Context) (int, error) {
	repo, ok := s.repo.(EnrollmentRepository)
	if !ok {
		return 0, nil
	}
	registrar, ok := s.registrar.(CorrelatedNodeRegistrar)
	if !ok {
		return 0, nil
	}
	sagas, err := repo.ListIncompleteEnrollmentSagas(ctx)
	if err != nil {
		return 0, err
	}
	for _, saga := range sagas {
		if saga.State == "prepared" {
			if err := repo.ClaimCode(ctx, saga.CodeID); err != nil && err != ErrCodeUsed {
				return 0, err
			}
			saga.State = "claimed"
			if err := repo.UpdateEnrollmentSaga(ctx, saga); err != nil {
				return 0, err
			}
		}
		if _, err := s.continueEnrollment(ctx, repo, registrar, saga); err != nil {
			return 0, err
		}
	}
	return len(sagas), nil
}

// ResolveEnrollment returns the Node associated with a correlated redemption.
// It is a typed control-plane result, not bootstrap output parsing.
func (s *Service) ResolveEnrollment(ctx context.Context, correlationID string) (string, bool, error) {
	repo, ok := s.repo.(EnrollmentRepository)
	if !ok {
		return "", false, nil
	}
	saga, err := repo.GetEnrollmentSaga(ctx, strings.TrimSpace(correlationID))
	if err == ErrCodeNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return saga.NodeID, saga.State == "completed" && saga.NodeID != "", nil
}

// RequestPairing records a pending join request (the no-pre-shared-code path).
func (s *Service) RequestPairing(ctx context.Context, nodePublicKeyB64 string, facts NodeFacts) (PairingRequest, error) {
	if err := validatePublicKey(nodePublicKeyB64); err != nil {
		return PairingRequest{}, err
	}
	return s.repo.CreateRequest(ctx, PairingRequest{
		PublicKey:    nodePublicKeyB64,
		Name:         strings.TrimSpace(facts.Name),
		OS:           facts.OS,
		Arch:         facts.Arch,
		Endpoint:     facts.Endpoint,
		Capabilities: facts.Capabilities,
		Status:       RequestPending,
		CreatedAt:    s.clock.Now().UTC(),
	})
}

// Approve approves (or rejects) a pending request. On approval it registers the
// node and stores its credential, then records the decision atomically.
func (s *Service) Approve(ctx context.Context, requestID string, approve bool, scopes []string) (status RequestStatus, nodeID string, err error) {
	if approve {
		if err := s.validateGrantScopes(scopes); err != nil {
			return "", "", err
		}
	}
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return "", "", err
	}
	if req.Status != RequestPending {
		return "", "", ErrRequestDecided
	}

	if !approve {
		if err := s.repo.DecideRequest(ctx, req.ID, RequestRejected, ""); err != nil {
			return "", "", err
		}
		return RequestRejected, "", nil
	}

	nodeID, err = s.registrar.RegisterNode(ctx, NodeFacts{
		Name:         req.Name,
		OS:           req.OS,
		Arch:         req.Arch,
		Endpoint:     req.Endpoint,
		Capabilities: req.Capabilities,
		Scopes:       normalizeScopes(scopes),
	})
	if err != nil {
		return "", "", fmt.Errorf("register node: %w", err)
	}
	if err := s.repo.StoreCredential(ctx, Credential{NodeID: nodeID, PublicKey: req.PublicKey, CreatedAt: s.clock.Now().UTC()}); err != nil {
		return "", "", fmt.Errorf("store credential: %w", err)
	}
	if err := s.repo.DecideRequest(ctx, req.ID, RequestApproved, nodeID); err != nil {
		return "", "", err
	}
	return RequestApproved, nodeID, nil
}

func (s *Service) validateGrantScopes(scopes []string) error {
	if len(scopes) == 0 {
		return nil
	}
	if s.validateScopes == nil {
		return ErrInvalid{Field: "scopes", Reason: "catalog grant validator is not configured"}
	}
	if err := s.validateScopes(scopes); err != nil {
		return ErrInvalid{Field: "scopes", Reason: err.Error()}
	}
	return nil
}

// ListRequests returns pending (or, with includeDecided, all) join requests.
func (s *Service) ListRequests(ctx context.Context, includeDecided bool) ([]PairingRequest, error) {
	return s.repo.ListRequests(ctx, includeDecided)
}

// RevokeCredential severs a node's credential. Called by the registry domain's
// atomic revoke (so a single RevokeNode kills durable identity AND auth).
func (s *Service) RevokeCredential(ctx context.Context, nodeID string) error {
	return s.repo.RevokeCredential(ctx, nodeID)
}

// SealingPublicKey returns the node-bound X25519 public key used to seal
// operator authorization envelopes. It is derived from the already-pinned
// Ed25519 identity; no additional private key is stored or transported.
func (s *Service) SealingPublicKey(ctx context.Context, nodeID string) ([]byte, error) {
	public, ok, err := s.repo.ActivePublicKey(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("node %q has no active credential", nodeID)
	}
	return sealing.PublicKeyFromEd25519(public)
}

// --- helpers ---

func generateCode() (string, error) {
	buf := make([]byte, codeBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Crockford-ish: standard base32, no padding, uppercase — human-typeable.
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// hashCode hashes a plaintext code with SHA-256; only the hash is stored. A code
// is a high-entropy random token (160 bits), so a plain hash is sufficient —
// there is nothing to brute-force as with a low-entropy password.
func hashCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}

func validatePublicKey(b64 string) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return ErrInvalid{Field: "node_public_key", Reason: "must be standard base64"}
	}
	if len(raw) != ed25519.PublicKeySize {
		return ErrInvalid{Field: "node_public_key", Reason: fmt.Sprintf("must be a %d-byte Ed25519 key", ed25519.PublicKeySize)}
	}
	return nil
}

func normalizeScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if v := strings.TrimSpace(s); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
