package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/receiptsigning"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
)

func testRotationCommitAckSigner(t *testing.T, authority *ca.Authority) *credentialauthoritysigning.Signer {
	t.Helper()
	identity, err := ca.ParseIdentity("scenario/treasury/wallet-recovery-commit-ack-test")
	require.NoError(t, err)
	signer, err := credentialauthoritysigning.New(credentialauthoritysigning.Config{
		Identity: identity, Field: "commit-ack-key-ring", PublicField: "commit-ack-public-key-ring", Store: authority,
		AllowedPurposes: []receiptsigning.Purpose{receiptsigning.PurposeWalletRecoveryCommitAcknowledgement},
	})
	require.NoError(t, err)
	_, err = signer.Rotate(context.Background())
	require.NoError(t, err)
	return signer
}

type failDeleteOnceCustody struct {
	Custody
	deleteErr bool
}

func (c *failDeleteOnceCustody) Delete(id ca.Identity, field string) error {
	if c.deleteErr {
		c.deleteErr = false
		return errors.New("fixture scalar delete interruption")
	}
	deleter, ok := c.Custody.(interface {
		Delete(ca.Identity, string) error
	})
	if !ok {
		return errors.New("underlying test authority does not support delete")
	}
	return deleter.Delete(id, field)
}

func TestPortableRotationRetryAfterServiceRestartRetainsExactTombstoneUntilVerifiedReceipt(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	signer, verifier := testRotationReceiptSigner(t, authority)
	source.WithRotationReceiptVerifier(verifier)
	const secret = "operator-held test recovery secret with 40 bytes"
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	_, err = source.PersistPortableRecoveryBundle(ctx, old.ID, old.Network, old.OperatorIdentity, secret)
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)

	first, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	require.True(t, authority.Status(identity, keyField).Configured, "prepare must retain the old scalar")

	// A process restart reuses durable intent, successor and exact signed bytes.
	restarted := NewService(source.repository, source.books, authority, source.clock, source.recoveryPath).WithRotationReceiptVerifier(verifier)
	second, err := restarted.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	require.Equal(t, first.Tombstone, second.Tombstone)
	require.Equal(t, first.SHA256, second.SHA256)
	require.Equal(t, first.Successor.Address, second.Successor.Address)
	require.True(t, authority.Status(identity, keyField).Configured, "retry must not delete the old scalar")

	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: second.Successor.ID,
		DestinationID: "operator-qualified-db", DestinationBackend: "s3", PlanID: "manual-plan", RunID: "run-1",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "snapshot-2", LatestSnapshotID: "snapshot-2",
		TombstoneSHA256: second.SHA256, RestoredRevocationSHA256: second.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: "2026-11-15T12:00:00Z",
	}
	receipt = signTestRotationReceipt(t, signer, receipt)
	stale := receipt
	stale.LatestSnapshotID = "snapshot-1"
	_, err = restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, second.Successor.ID, stale)
	require.ErrorIs(t, err, ErrMismatch)
	require.True(t, authority.Status(identity, keyField).Configured, "a stale snapshot receipt must preserve the old scalar")
	replayed := receipt
	replayed.SnapshotID = "snapshot-replayed"
	replayed.LatestSnapshotID = "snapshot-replayed"
	_, err = restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, second.Successor.ID, replayed)
	require.ErrorIs(t, err, ErrMismatch, "an attestation cannot be replayed for a different snapshot")
	require.True(t, authority.Status(identity, keyField).Configured, "a replayed attestation must preserve the old scalar")
	expired := receipt
	leaseNow := time.Now().UTC()
	expired.VerifiedAt = leaseNow.Add(-2 * time.Minute).Format(time.RFC3339Nano)
	expired.ReservationExpiresAt = leaseNow.Add(-time.Second).Format(time.RFC3339Nano)
	expired = signExactTestRotationReceipt(t, signer, expired)
	_, err = restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, second.Successor.ID, expired)
	require.ErrorIs(t, err, ErrMismatch, "an expired reservation must not authorize scalar deletion")
	require.True(t, authority.Status(identity, keyField).Configured, "an expired reservation must preserve the old scalar")

	committed, err := restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, second.Successor.ID, receipt)
	require.NoError(t, err)
	require.Equal(t, second.Successor.Address, committed.Address)
	require.False(t, authority.Status(identity, keyField).Configured, "validated off-host tombstone permits scalar deletion")

	// A client retry after an interrupted response is idempotent.
	retried, err := restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, second.Successor.ID, receipt)
	require.NoError(t, err)
	require.Equal(t, committed.Address, retried.Address)
	got := sha256.Sum256(second.Tombstone)
	require.Equal(t, hex.EncodeToString(got[:]), receipt.TombstoneSHA256)
}

func TestPortableRotationCommitAcknowledgementBindsExactReservationAndIsRetryStable(t *testing.T) {
	ctx := context.Background()
	service, authority, in := setup(t)
	publicationSigner, publicationVerifier := testRotationReceiptSigner(t, authority)
	service.WithRotationReceiptVerifier(publicationVerifier)
	ackSigner := testRotationCommitAckSigner(t, authority)
	service.WithRotationCommitAckSigner(ackSigner)

	old, err := service.Create(ctx, in)
	require.NoError(t, err)
	const secret = "operator-held test recovery secret for signed acknowledgement"
	_, err = service.PersistPortableRecoveryBundle(ctx, old.ID, old.Network, old.OperatorIdentity, secret)
	require.NoError(t, err)
	oldIdentity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	prepared, err := service.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "ack-successor")
	require.NoError(t, err)
	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "fixture-db", DestinationBackend: "s3", PlanID: "fixture-plan", RunID: "fixture-run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "fixture-snapshot", LatestSnapshotID: "fixture-snapshot",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true,
	}
	receipt = signTestRotationReceipt(t, publicationSigner, receipt)

	committed, envelope, err := service.CommitPortableRotationWithAcknowledgement(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, "treasury-target", receipt)
	require.NoError(t, err)
	require.Equal(t, prepared.Successor.ID, committed.ID)
	require.Equal(t, prepared.Successor.Address, committed.Address)
	require.Equal(t, receiptsigning.PurposeWalletRecoveryCommitAcknowledgement, envelope.Purpose)
	ack := receiptsigning.WalletRecoveryCommitAcknowledgement{
		Version: 1, OperationID: receipt.ReservationID, ReservationID: receipt.ReservationID,
		TargetID: "treasury-target", DestinationID: receipt.DestinationID, SnapshotID: receipt.SnapshotID,
		TombstoneSHA256: receipt.TombstoneSHA256, SuccessorID: committed.ID,
		SuccessorAddress: committed.Address, SuccessorPublicKey: committed.PublicKey,
	}
	canonical, err := receiptsigning.CanonicalJSON(ack)
	require.NoError(t, err)
	require.NoError(t, ackSigner.Verify(ctx, envelope, canonical), "signature must authenticate every exact reservation and successor field")
	require.False(t, authority.Status(oldIdentity, keyField).Configured, "validated tombstone commit revokes the old scalar before DBM releases its pin")

	retried, retryEnvelope, err := service.CommitPortableRotationWithAcknowledgement(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, "treasury-target", receipt)
	require.NoError(t, err)
	require.Equal(t, committed.ID, retried.ID)
	require.Equal(t, committed.Address, retried.Address)
	require.Equal(t, envelope, retryEnvelope, "a lost response must reproduce the exact acknowledgement")
}

func TestPortableRotationCommitAcknowledgementUnavailableKeepsOldScalarActive(t *testing.T) {
	ctx := context.Background()
	service, authority, in := setup(t)
	publicationSigner, publicationVerifier := testRotationReceiptSigner(t, authority)
	service.WithRotationReceiptVerifier(publicationVerifier)
	old, err := service.Create(ctx, in)
	require.NoError(t, err)
	_, err = service.PersistPortableRecoveryBundle(ctx, old.ID, old.Network, old.OperatorIdentity, "operator-held test recovery secret for fail-closed acknowledgement")
	require.NoError(t, err)
	oldIdentity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	prepared, err := service.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "ack-successor")
	require.NoError(t, err)
	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "fixture-db", DestinationBackend: "s3", PlanID: "fixture-plan", RunID: "fixture-run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "fixture-snapshot", LatestSnapshotID: "fixture-snapshot",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true,
	}
	receipt = signTestRotationReceipt(t, publicationSigner, receipt)

	_, _, err = service.CommitPortableRotationWithAcknowledgement(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, "treasury-target", receipt)
	require.ErrorIs(t, err, ErrUnavailable)
	require.True(t, authority.Status(oldIdentity, keyField).Configured, "without Treasury's exact acknowledgement signer, the old scalar stays active")
	metadata, err := service.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.Empty(t, metadata.RevokedAt, "an unavailable acknowledgement path must not commit revocation")
}

func TestPortableRotationResumeAfterRevocationCommitButBeforeScalarDelete(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	signer, verifier := testRotationReceiptSigner(t, authority)
	source.WithRotationReceiptVerifier(verifier)
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	_, err = source.PersistPortableRecoveryBundle(ctx, old.ID, old.Network, old.OperatorIdentity, "operator-held fixture recovery secret 1234567890")
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	prepared, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "operator-qualified-db", DestinationBackend: "s3", PlanID: "manual-plan", RunID: "run-1",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "snapshot-2", LatestSnapshotID: "snapshot-2",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true,
	}
	receipt = signTestRotationReceipt(t, signer, receipt)
	fault := &failDeleteOnceCustody{Custody: authority, deleteErr: true}
	source.custody = fault
	_, err = source.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, receipt)
	require.ErrorContains(t, err, "fixture scalar delete interruption")
	require.True(t, authority.Status(identity, keyField).Configured, "failed deletion must retain the old scalar")
	metadata, err := source.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.NotEmpty(t, metadata.RevokedAt, "the committed tombstone intent must remain durable for restart retry")

	restarted := NewService(source.repository, source.books, fault, source.clock, source.recoveryPath).WithRotationReceiptVerifier(verifier)
	committed, err := restarted.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, receipt)
	require.NoError(t, err)
	require.Equal(t, prepared.Successor.Address, committed.Address)
	require.False(t, authority.Status(identity, keyField).Configured, "restart retry must finish deleting the revoked scalar")
	metadata, err = restarted.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.Equal(t, prepared.Successor.ID, metadata.SuccessorID)
}

func TestPortableRotationRejectsMissingOrMismatchedOffHostReceiptBeforeDeletingScalar(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	_, verifier := testRotationReceiptSigner(t, authority)
	source.WithRotationReceiptVerifier(verifier)
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	prepared, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	receipt := RotationPublicationReceipt{Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "local-only", DestinationBackend: "filesystem", PlanID: "plan", RunID: "run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "snap", LatestSnapshotID: "snap",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256, DBMVerifyPassed: true,
		RestoredTombstoneMatches: true, VerifiedAt: "2026-11-15T12:00:00Z"}
	_, err = source.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, receipt)
	require.ErrorIs(t, err, ErrMismatch)
	require.True(t, authority.Status(identity, keyField).Configured)
}

func TestPortableRotationRejectsCallerFabricatedDBMReceiptWithoutVerifier(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	prepared, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	fabricated := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "invented-off-host-db", DestinationBackend: "s3", PlanID: "invented-plan", RunID: "invented-run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "invented-snapshot", LatestSnapshotID: "invented-snapshot",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: time.Now().UTC().Format(time.RFC3339Nano),
		ReservationID: "fabricated-reservation", ReservationExpiresAt: time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339Nano),
	}
	_, err = source.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, fabricated)
	require.ErrorIs(t, err, ErrUnavailable, "caller booleans and hashes are not DBM attestations")
	require.True(t, authority.Status(identity, keyField).Configured, "a forged receipt must not delete the old scalar")
	stored, err := source.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.Empty(t, stored.RevokedAt, "a forged receipt must not mark the old root revoked")
}

func TestPortableRotationRejectsUnsignedCallerReceiptEvenWhenVerifierIsReady(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	_, verifier := testRotationReceiptSigner(t, authority)
	source.WithRotationReceiptVerifier(verifier)
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	prepared, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	fabricated := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "invented-off-host-db", DestinationBackend: "s3", PlanID: "invented-plan", RunID: "invented-run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "invented-snapshot", LatestSnapshotID: "invented-snapshot",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: time.Now().UTC().Format(time.RFC3339Nano),
		ReservationID: "unsigned-reservation", ReservationExpiresAt: time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339Nano),
	}
	_, err = source.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, fabricated)
	require.ErrorIs(t, err, ErrMismatch, "a ready verifier still rejects an unsigned caller assertion")
	require.True(t, authority.Status(identity, keyField).Configured, "an unsigned receipt must not delete the old scalar")
	stored, err := source.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.Empty(t, stored.RevokedAt, "an unsigned receipt must not mark the old root revoked")
}

func TestPortableRotationKeepsOldScalarWhenSuccessorBindingIsCorrupt(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	signer, verifier := testRotationReceiptSigner(t, authority)
	source.WithRotationReceiptVerifier(verifier)
	old, err := source.Create(ctx, in)
	require.NoError(t, err)
	oldIdentity, err := ca.ParseIdentity(old.CustodyReference)
	require.NoError(t, err)
	prepared, err := source.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, "stable-successor")
	require.NoError(t, err)
	nextIdentity, err := ca.ParseIdentity(prepared.Successor.CustodyReference)
	require.NoError(t, err)
	require.NoError(t, authority.Put(nextIdentity, keyField, "not-a-valid-scalar"))
	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "operator-qualified-db", DestinationBackend: "s3", PlanID: "manual-plan", RunID: "run-1",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "snapshot-1", LatestSnapshotID: "snapshot-1",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: "2026-11-15T12:00:00Z",
	}
	receipt = signTestRotationReceipt(t, signer, receipt)
	_, err = source.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, receipt)
	require.Error(t, err, "commit must prove the successor key before deleting the old scalar")
	require.True(t, authority.Status(oldIdentity, keyField).Configured)
	stored, err := source.repository.Get(ctx, old.ID)
	require.NoError(t, err)
	require.Empty(t, stored.RevokedAt)
}
