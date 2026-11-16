// Package walletrecovery owns DBM's fail-closed proof boundary for portable
// wallet recovery. It deliberately contains no credential creation code.
package walletrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/api-core/receiptsigning"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("wallet recovery attestation signer is not configured")

// Evidence is the independently collected DBM record set. The collector must
// read DBM's persisted run/outcome, Kopia's immutable snapshot inventory, and
// the terminal restore record plus its actual materialized-tree digest.
type Evidence struct {
	RequestID, TargetID, DestinationID                               string
	WalletID, Network, Address, SuccessorID                          string
	DestinationBackend, PlanID, TargetName                           string
	RunID, RunTrigger, RunStatus                                     string
	OutcomeTargetID, OutcomeDestinationID, OutcomeStatus, SnapshotID string
	SnapshotNewest, SnapshotImmutable                                bool
	SnapshotGeneration                                               int64
	RestoreID, RestoreStatus, RestoreSnapshotID                      string
	RestoredTombstoneSHA256                                          string
	RestoredRevocationSHA256                                         string
	DBMVerifyPassed, RestoredTombstoneMatches                        bool
	WalletBundlePresent                                              bool
}

type EvidenceReader interface {
	ReadWalletRecoveryEvidence(context.Context, string, string, string) (Evidence, error)
}
type Signer interface {
	Sign(context.Context, receiptsigning.Purpose, []byte) (receiptsigning.SignatureEnvelope, error)
}

// Receipt is the canonical signed statement consumed by Treasury. JSON field
// order is fixed by the struct; the digest covers the canonical JSON body.
type Receipt struct {
	Version                  int    `json:"version"`
	WalletID                 string `json:"wallet_id"`
	Network                  string `json:"network"`
	Address                  string `json:"address"`
	SuccessorID              string `json:"successor_id"`
	DestinationID            string `json:"destination_id"`
	DestinationBackend       string `json:"destination_backend"`
	PlanID                   string `json:"plan_id"`
	RunID                    string `json:"run_id"`
	TargetName               string `json:"target_name"`
	SnapshotID               string `json:"snapshot_id"`
	LatestSnapshotID         string `json:"latest_snapshot_id"`
	TombstoneSHA256          string `json:"tombstone_sha256"`
	RestoredRevocationSHA256 string `json:"restored_revocation_sha256"`
	BundlePresent            bool   `json:"bundle_present"`
	DBMVerifyPassed          bool   `json:"dbm_verify_passed"`
	RestoredTombstoneMatches bool   `json:"restored_tombstone_matches"`
	VerifiedAt               string `json:"verified_at"`
	ReservationID            string `json:"reservation_id"`
	ReservationExpiresAt     string `json:"reservation_expires_at"`
}

type SignedReceipt struct {
	Receipt     Receipt                          `json:"receipt"`
	Attestation receiptsigning.SignatureEnvelope `json:"attestation"`
}

type Service struct {
	Evidence     EvidenceReader
	Reservations ReservationStore
	Signer       Signer
	Now          func() time.Time
	TTL          time.Duration
}

// Attest validates the exact manual run and terminal restore, reserves and pins
// the target×destination before signing. A valid receipt therefore never
// exists while a newer snapshot mutation can race Treasury's scalar deletion.
// Repeating the same request is idempotent; a different request or stale
// evidence fails closed.
func (s Service) Attest(ctx context.Context, requestID, targetID, destinationID string) (SignedReceipt, error) {
	requestID, targetID, destinationID = strings.TrimSpace(requestID), strings.TrimSpace(targetID), strings.TrimSpace(destinationID)
	if requestID == "" || targetID == "" || destinationID == "" {
		return SignedReceipt{}, errors.New("request, target, and destination identities are required")
	}
	if s.Evidence == nil || s.Reservations == nil {
		return SignedReceipt{}, fmt.Errorf("wallet recovery evidence and reservation stores are required: %w", ErrNotConfigured)
	}
	if s.Signer == nil {
		return SignedReceipt{}, ErrNotConfigured
	}
	e, err := s.Evidence.ReadWalletRecoveryEvidence(ctx, requestID, targetID, destinationID)
	if err != nil {
		return SignedReceipt{}, fmt.Errorf("read independent DBM evidence: %w", err)
	}
	if e.RequestID != requestID || e.TargetID != targetID || e.DestinationID != destinationID {
		return SignedReceipt{}, errors.New("request evidence does not bind the requested target and destination")
	}
	if e.RunID == "" || e.RunTrigger != "manual" || e.RunStatus != "succeeded" {
		return SignedReceipt{}, errors.New("evidence requires the exact successful terminal manual run")
	}
	if e.OutcomeTargetID != targetID || e.OutcomeDestinationID != destinationID || e.OutcomeStatus != "succeeded" || e.SnapshotID == "" {
		return SignedReceipt{}, errors.New("manual run lacks a successful outcome for the exact target×destination")
	}
	if !e.SnapshotNewest || !e.SnapshotImmutable {
		return SignedReceipt{}, errors.New("snapshot is not the newest verified immutable off-host snapshot")
	}
	if e.RestoreID == "" || e.RestoreStatus != "verified" || e.RestoreSnapshotID != e.SnapshotID {
		return SignedReceipt{}, errors.New("exact snapshot lacks a successful terminal verified restore")
	}
	if !isSHA256(e.RestoredTombstoneSHA256) {
		return SignedReceipt{}, errors.New("restored tombstone digest is missing or malformed")
	}
	if e.WalletBundlePresent {
		return SignedReceipt{}, errors.New("restored recovery tree still contains wallet.bundle")
	}
	if e.WalletID == "" || e.Network == "" || e.Address == "" || e.SuccessorID == "" || e.DestinationBackend == "" || e.PlanID == "" || e.TargetName == "" {
		return SignedReceipt{}, errors.New("evidence lacks wallet, destination, plan, successor, or target identity")
	}
	stableReservationID := ReservationID(e.WalletID, e.SuccessorID, destinationID)
	if requestID != stableReservationID {
		return SignedReceipt{}, errors.New("reservation request ID must be stable for this wallet, successor, and destination")
	}
	if !e.DBMVerifyPassed || !e.RestoredTombstoneMatches || !isSHA256(e.RestoredRevocationSHA256) {
		return SignedReceipt{}, errors.New("DBM verify, restored revocation digest, or exact tombstone comparison is incomplete")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	res := Reservation{RequestID: stableReservationID, TargetID: targetID, DestinationID: destinationID, SnapshotID: e.SnapshotID, TombstoneSHA256: strings.ToLower(e.RestoredTombstoneSHA256), SnapshotGeneration: e.SnapshotGeneration, ExpiresAt: now.Add(ttl)}
	if err := s.Reservations.Reserve(ctx, res, now); err != nil {
		return SignedReceipt{}, fmt.Errorf("reserve newest snapshot against concurrent writes: %w", err)
	}
	// The pin must precede signing. Otherwise a caller could bypass the normal
	// CLI ordering and send a valid latest-at-issue receipt directly to Treasury
	// while DBM still permits another snapshot mutation. If signing fails or the
	// process stops here, the same request can retry and renew this exact pin.
	if err := s.Reservations.Commit(ctx, stableReservationID, targetID); err != nil {
		return SignedReceipt{}, fmt.Errorf("pin exact newest snapshot before issuing its receipt: %w", err)
	}
	r := Receipt{Version: 1, WalletID: e.WalletID, Network: e.Network, Address: e.Address, SuccessorID: e.SuccessorID, DestinationID: destinationID, DestinationBackend: e.DestinationBackend, PlanID: e.PlanID, RunID: e.RunID, TargetName: e.TargetName, SnapshotID: e.SnapshotID, LatestSnapshotID: e.SnapshotID, TombstoneSHA256: strings.ToLower(e.RestoredTombstoneSHA256), RestoredRevocationSHA256: strings.ToLower(e.RestoredRevocationSHA256), BundlePresent: e.WalletBundlePresent, DBMVerifyPassed: e.DBMVerifyPassed, RestoredTombstoneMatches: e.RestoredTombstoneMatches, VerifiedAt: now.Format(time.RFC3339Nano), ReservationID: stableReservationID, ReservationExpiresAt: res.ExpiresAt.UTC().Format(time.RFC3339Nano)}
	body, err := json.Marshal(r)
	if err != nil {
		return SignedReceipt{}, err
	}
	envelope, err := s.Signer.Sign(ctx, receiptsigning.PurposeWalletRecoveryPublicationReceipt, body)
	if err != nil {
		return SignedReceipt{}, fmt.Errorf("sign wallet recovery receipt: %w", err)
	}
	if strings.TrimSpace(envelope.KeyID) == "" || strings.TrimSpace(envelope.Signature) == "" || strings.TrimSpace(envelope.Version) == "" {
		return SignedReceipt{}, errors.New("signer returned an incomplete signature")
	}
	if envelope.Purpose != receiptsigning.PurposeWalletRecoveryPublicationReceipt {
		return SignedReceipt{}, errors.New("signer returned an envelope for the wrong receipt purpose")
	}
	if err := envelope.Validate(); err != nil {
		return SignedReceipt{}, fmt.Errorf("signer returned invalid signature envelope: %w", err)
	}
	return SignedReceipt{Receipt: r, Attestation: envelope}, nil
}

// ReservationID is deterministic so a timeout/restart retry for the same
// wallet successor and destination rejoins the same DBM lease.
func ReservationID(walletID, successorID, destinationID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(walletID) + "\x00" + strings.TrimSpace(successorID) + "\x00" + strings.TrimSpace(destinationID)))
	return "wallet-recovery-" + hex.EncodeToString(sum[:])
}

func isSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32
}
