package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// RotationPreparation is the public, non-secret handoff for an operator-owned
// DBM publication. The exact tombstone is persisted under Treasury's recovery
// root before it is returned; retrying the same successor returns the same
// signed bytes.
type RotationPreparation struct {
	Successor Wallet
	Tombstone []byte
	SHA256    string
}

// RotationPublicationReceipt is produced by the authenticated operator CLI
// after DBM verifies and scratch-restores the newest S3/SFTP snapshot. Treasury
// binds it to the exact staged tombstone before changing revocation metadata or
// deleting the old Credential Authority scalar.
type RotationPublicationReceipt struct {
	Version                  int                               `json:"version"`
	WalletID                 string                            `json:"wallet_id"`
	Network                  string                            `json:"network"`
	Address                  string                            `json:"address"`
	SuccessorID              string                            `json:"successor_id"`
	DestinationID            string                            `json:"destination_id"`
	DestinationBackend       string                            `json:"destination_backend"`
	PlanID                   string                            `json:"plan_id"`
	RunID                    string                            `json:"run_id"`
	TargetName               string                            `json:"target_name"`
	SnapshotID               string                            `json:"snapshot_id"`
	LatestSnapshotID         string                            `json:"latest_snapshot_id"`
	TombstoneSHA256          string                            `json:"tombstone_sha256"`
	RestoredRevocationSHA256 string                            `json:"restored_revocation_sha256"`
	BundlePresent            bool                              `json:"bundle_present"`
	DBMVerifyPassed          bool                              `json:"dbm_verify_passed"`
	RestoredTombstoneMatches bool                              `json:"restored_tombstone_matches"`
	VerifiedAt               string                            `json:"verified_at"`
	ReservationID            string                            `json:"reservation_id"`
	ReservationExpiresAt     string                            `json:"reservation_expires_at"`
	Attestation              *receiptsigning.SignatureEnvelope `json:"attestation,omitempty"`
}

func (s *Service) PreparePortableRotation(ctx context.Context, id, network, operator, successorID string) (RotationPreparation, error) {
	if !validID.MatchString(successorID) || successorID == id {
		return RotationPreparation{}, ErrInvalid
	}
	if err := s.available(ctx); err != nil {
		return RotationPreparation{}, err
	}
	old, err := s.repository.Get(ctx, id)
	if err != nil {
		return RotationPreparation{}, err
	}
	if old.Network != network || old.OperatorIdentity != operator || !validReference(old) || (old.RevokedAt != "" && old.SuccessorID != successorID) {
		return RotationPreparation{}, ErrMismatch
	}
	if old.Network != "eip155:84532" {
		return RotationPreparation{}, fmt.Errorf("%w: root rotation is limited to Base Sepolia until balance migration is supported", ErrUnavailable)
	}
	if old.RevokedAt == "" {
		if _, err = s.binding(ctx, id, network, operator); err != nil {
			return RotationPreparation{}, err
		}
	}
	if old.RevokedAt == "" {
		err = s.repository.WithCustodyLock(ctx, old.ID, func(repository Repository) error {
			current, getErr := repository.Get(ctx, old.ID)
			if getErr != nil {
				return getErr
			}
			if current.RevokedAt != "" {
				return ErrMismatch
			}
			return repository.PrepareRotation(ctx, old.ID, successorID, s.clock.Now().UTC())
		})
		if err != nil {
			return RotationPreparation{}, err
		}
	}
	next, err := s.rotationSuccessor(ctx, old, successorID)
	if err != nil {
		return RotationPreparation{}, err
	}
	revocationTime := s.clock.Now().UTC()
	prepared := old
	if prepared.RevokedAt == "" {
		prepared.RevokedAt = revocationTime.Format(time.RFC3339Nano)
	}
	prepared.SuccessorID = successorID
	if s.recoveryPath != nil {
		dir, pathErr := s.recoveryPath(ctx, portableBackupName(old.ID))
		if pathErr != nil {
			return RotationPreparation{}, pathErr
		}
		if existingBytes, readErr := readBackup(filepath.Join(dir, "revocation.json")); readErr == nil {
			var existing PortableRecoveryRevocation
			if json.Unmarshal(existingBytes, &existing) != nil || !VerifyPortableRecoveryRevocation(existing) || existing.WalletID != old.ID || existing.Network != old.Network || existing.Address != old.Address || existing.SuccessorID != successorID {
				return RotationPreparation{}, ErrMismatch
			}
			parsedTime, parseErr := time.Parse(time.RFC3339Nano, existing.RevokedAt)
			if parseErr != nil {
				return RotationPreparation{}, ErrMismatch
			}
			revocationTime = parsedTime.UTC()
			prepared.RevokedAt = existing.RevokedAt
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return RotationPreparation{}, ErrMismatch
		}
	}
	tombstone, err := s.preparePortableRevocation(ctx, prepared)
	if err != nil {
		return RotationPreparation{}, err
	}
	digest := sha256.Sum256(tombstone)
	return RotationPreparation{Successor: next, Tombstone: tombstone, SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s *Service) CommitPortableRotation(ctx context.Context, id, network, operator, successorID string, receipt RotationPublicationReceipt) (Wallet, error) {
	if err := s.available(ctx); err != nil {
		return Wallet{}, err
	}
	old, err := s.repository.Get(ctx, id)
	if err != nil {
		return Wallet{}, err
	}
	if old.Network != network || old.OperatorIdentity != operator || old.SuccessorID != "" && old.SuccessorID != successorID {
		return Wallet{}, ErrMismatch
	}
	next, err := s.repository.Get(ctx, successorID)
	if err != nil || next.OperatorIdentity != operator || next.Network != network || next.BookID != old.BookID {
		return Wallet{}, ErrMismatch
	}
	verifiedNext, err := s.Get(ctx, successorID, network, operator)
	if err != nil || verifiedNext.Address != next.Address || verifiedNext.PublicKey != next.PublicKey {
		return Wallet{}, fmt.Errorf("%w: successor key is not available and address-bound; old scalar remains active", ErrMismatch)
	}
	intended, err := s.repository.RotationSuccessor(ctx, id)
	if err != nil || intended != successorID {
		return Wallet{}, ErrMismatch
	}
	if s.recoveryPath == nil {
		return Wallet{}, ErrUnavailable
	}
	dir, err := s.recoveryPath(ctx, portableBackupName(id))
	if err != nil {
		return Wallet{}, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return Wallet{}, fmt.Errorf("%w: rotation recovery directory must be an existing private directory", ErrUnavailable)
	}
	tombstone, err := readBackup(filepath.Join(dir, "revocation.json"))
	if err != nil {
		return Wallet{}, fmt.Errorf("%w: staged signed rotation tombstone is unavailable", ErrUnavailable)
	}
	var parsed PortableRecoveryRevocation
	if json.Unmarshal(tombstone, &parsed) != nil || !VerifyPortableRecoveryRevocation(parsed) || parsed.WalletID != id || parsed.Network != network || parsed.Address != old.Address || parsed.SuccessorID != successorID {
		return Wallet{}, ErrMismatch
	}
	if err := validateRotationReceipt(receipt, old, successorID, tombstone); err != nil {
		return Wallet{}, err
	}
	if s.rotationReceiptVerifier == nil {
		return Wallet{}, fmt.Errorf("%w: trusted DBM wallet-recovery attestation verifier is not configured; old scalar remains active", ErrUnavailable)
	}
	if receipt.Attestation == nil || receipt.Attestation.Purpose != receiptsigning.PurposeWalletRecoveryPublicationReceipt {
		return Wallet{}, ErrMismatch
	}
	verifiedAt, err := time.Parse(time.RFC3339Nano, receipt.VerifiedAt)
	if err != nil {
		return Wallet{}, ErrMismatch
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, receipt.ReservationExpiresAt)
	now := s.clock.Now().UTC()
	if err != nil || !now.Before(expiresAt) || verifiedAt.After(now.Add(15*time.Second)) || expiresAt.Sub(verifiedAt) <= 0 || expiresAt.Sub(verifiedAt) > 10*time.Minute {
		return Wallet{}, fmt.Errorf("%w: DBM snapshot reservation is expired, future-dated, or exceeds the bounded lease; old scalar remains active", ErrMismatch)
	}
	canonicalReceipt, err := rotationReceiptSigningBytes(receipt)
	if err != nil {
		return Wallet{}, err
	}
	if err := s.rotationReceiptVerifier.Verify(ctx, *receipt.Attestation, canonicalReceipt); err != nil {
		return Wallet{}, fmt.Errorf("%w: DBM rotation publication attestation did not verify; old scalar remains active", ErrMismatch)
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		return Wallet{}, err
	}
	receiptPath := filepath.Join(dir, "rotation-publication.json")
	if err := replacePrivateArtifact(receiptPath, receiptBytes); err != nil {
		return Wallet{}, fmt.Errorf("%w: persist DBM publication receipt: %v", ErrUnavailable, err)
	}
	if old.RevokedAt != "" {
		if old.RevokedAt != parsed.RevokedAt {
			return Wallet{}, ErrMismatch
		}
		if err := s.deleteRevokedScalar(old); err != nil {
			return Wallet{}, err
		}
		return s.Get(ctx, successorID, network, operator)
	}
	revokedAt, err := time.Parse(time.RFC3339Nano, parsed.RevokedAt)
	if err != nil {
		return Wallet{}, ErrMismatch
	}
	err = s.repository.WithCustodyLock(ctx, id, func(repository Repository) error {
		current, getErr := repository.Get(ctx, id)
		if getErr != nil {
			return getErr
		}
		if current.RevokedAt != "" {
			if current.SuccessorID == successorID {
				return nil
			}
			return ErrMismatch
		}
		return repository.SetRevoked(ctx, id, revokedAt, successorID)
	})
	if err != nil {
		return Wallet{}, err
	}
	old, err = s.repository.Get(ctx, id)
	if err != nil {
		return Wallet{}, err
	}
	if err = s.deleteRevokedScalar(old); err != nil {
		return Wallet{}, err
	}
	return s.Get(ctx, successorID, network, operator)
}

// CommitPortableRotationWithAcknowledgement commits the validated rotation and
// signs a replay-safe proof for the exact DBM reservation. A retry of an
// already committed rotation can produce the same bound acknowledgement.
func (s *Service) CommitPortableRotationWithAcknowledgement(ctx context.Context, id, network, operator, successorID, targetID string, receipt RotationPublicationReceipt) (Wallet, receiptsigning.SignatureEnvelope, error) {
	if s == nil || s.rotationCommitAckSigner == nil || strings.TrimSpace(targetID) == "" {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, fmt.Errorf("%w: Treasury commit acknowledgement signer or DBM target is not configured; DBM reservation remains pinned", ErrUnavailable)
	}
	committed, err := s.CommitPortableRotation(ctx, id, network, operator, successorID, receipt)
	if err != nil {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, err
	}
	ack := receiptsigning.WalletRecoveryCommitAcknowledgement{
		Version: 1, OperationID: receipt.ReservationID, ReservationID: receipt.ReservationID,
		TargetID: targetID, DestinationID: receipt.DestinationID, SnapshotID: receipt.SnapshotID,
		TombstoneSHA256: receipt.TombstoneSHA256, SuccessorID: committed.ID,
		SuccessorAddress: committed.Address, SuccessorPublicKey: committed.PublicKey,
	}
	canonical, err := receiptsigning.CanonicalJSON(ack)
	if err != nil {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, err
	}
	envelope, err := s.rotationCommitAckSigner.Sign(ctx, receiptsigning.PurposeWalletRecoveryCommitAcknowledgement, canonical)
	if err != nil {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, fmt.Errorf("rotation committed but Treasury could not sign DBM acknowledgement; retry the same receipt: %w", err)
	}
	if envelope.Purpose != receiptsigning.PurposeWalletRecoveryCommitAcknowledgement || envelope.Validate() != nil {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, errors.New("rotation committed but Treasury signer returned an invalid commit acknowledgement; retry the same receipt")
	}
	if err := s.rotationCommitAckSigner.Verify(ctx, envelope, canonical); err != nil {
		return Wallet{}, receiptsigning.SignatureEnvelope{}, fmt.Errorf("rotation committed but Treasury could not verify its commit acknowledgement; retry the same receipt: %w", err)
	}
	return committed, envelope, nil
}

func rotationReceiptSigningBytes(receipt RotationPublicationReceipt) ([]byte, error) {
	receipt.Attestation = nil
	return json.Marshal(receipt)
}

func (s *Service) rotationSuccessor(ctx context.Context, old Wallet, successorID string) (Wallet, error) {
	next, err := s.repository.Get(ctx, successorID)
	if err == nil {
		if next.BookID != old.BookID || next.Network != old.Network || next.OperatorIdentity != old.OperatorIdentity || !validReference(next) {
			return Wallet{}, ErrMismatch
		}
		// Create is deliberately idempotent for an exact persisted binding and
		// completes CA mint/readback if the previous process stopped mid-create.
		return s.Create(ctx, CreateInput{ID: successorID, BookID: old.BookID, Network: old.Network, OperatorIdentity: old.OperatorIdentity})
	}
	if !errors.Is(err, ErrNotFound) {
		return Wallet{}, err
	}
	return s.Create(ctx, CreateInput{ID: successorID, BookID: old.BookID, Network: old.Network, OperatorIdentity: old.OperatorIdentity})
}

func validateRotationReceipt(r RotationPublicationReceipt, old Wallet, successorID string, tombstone []byte) error {
	digest := sha256.Sum256(tombstone)
	expectedHash := hex.EncodeToString(digest[:])
	if r.Version != 1 || r.WalletID != old.ID || r.Network != old.Network || r.Address != old.Address || r.SuccessorID != successorID ||
		strings.TrimSpace(r.DestinationID) == "" || (r.DestinationBackend != "s3" && r.DestinationBackend != "sftp") || strings.TrimSpace(r.PlanID) == "" ||
		strings.TrimSpace(r.RunID) == "" || strings.TrimSpace(r.ReservationID) == "" || r.TargetName != "treasury-wallet-recovery-"+old.ID || strings.TrimSpace(r.SnapshotID) == "" ||
		r.LatestSnapshotID != r.SnapshotID || r.TombstoneSHA256 != expectedHash || r.RestoredRevocationSHA256 != expectedHash || r.BundlePresent ||
		!r.DBMVerifyPassed || !r.RestoredTombstoneMatches {
		return ErrMismatch
	}
	if _, err := time.Parse(time.RFC3339Nano, r.VerifiedAt); err != nil {
		return ErrMismatch
	}
	return nil
}

func replacePrivateArtifact(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".rotation-receipt-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func (s *Service) deleteRevokedScalar(w Wallet) error {
	identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
	if err != nil {
		return ErrMismatch
	}
	key, keyErr := s.key(w)
	if errors.Is(keyErr, credentialauthority.ErrUnconfigured) {
		return nil
	}
	if keyErr != nil {
		return keyErr
	}
	defer key.Zero()
	deleter, ok := s.custody.(interface {
		Delete(credentialauthority.Identity, string) error
	})
	if !ok {
		return fmt.Errorf("%w: credential authority does not support key revocation", ErrUnavailable)
	}
	if err = deleter.Delete(identity, keyField); err != nil {
		return fmt.Errorf("%w: revocation tombstone exists, but old key cleanup failed: %v", ErrUnavailable, err)
	}
	return nil
}
