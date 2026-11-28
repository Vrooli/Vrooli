package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/receiptsigning"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
)

func testRotationReceiptSigner(t *testing.T, authority *ca.Authority) (*credentialauthoritysigning.Signer, *credentialauthoritysigning.PublicVerifier) {
	t.Helper()
	identity, err := ca.ParseIdentity("scenario/data-backup-manager/wallet-recovery-test")
	require.NoError(t, err)
	signer, err := credentialauthoritysigning.New(credentialauthoritysigning.Config{
		Identity: identity, Field: "rotation-attestation-key-ring", PublicField: "rotation-attestation-public-key-ring", Store: authority,
		AllowedPurposes: []receiptsigning.Purpose{receiptsigning.PurposeWalletRecoveryPublicationReceipt},
	})
	require.NoError(t, err)
	_, err = signer.Rotate(context.Background())
	require.NoError(t, err)
	verifier, err := credentialauthoritysigning.NewPublicVerifier(identity, "rotation-attestation-public-key-ring", authority,
		[]receiptsigning.Purpose{receiptsigning.PurposeWalletRecoveryPublicationReceipt})
	require.NoError(t, err)
	return signer, verifier
}

func signTestRotationReceipt(t *testing.T, signer *credentialauthoritysigning.Signer, receipt RotationPublicationReceipt) RotationPublicationReceipt {
	t.Helper()
	now := time.Now().UTC()
	receipt.VerifiedAt = now.Format(time.RFC3339Nano)
	if receipt.ReservationID == "" {
		receipt.ReservationID = "test-reservation-" + receipt.WalletID + "-" + receipt.SuccessorID
	}
	receipt.ReservationExpiresAt = now.Add(2 * time.Minute).Format(time.RFC3339Nano)
	return signExactTestRotationReceipt(t, signer, receipt)
}

func signExactTestRotationReceipt(t *testing.T, signer *credentialauthoritysigning.Signer, receipt RotationPublicationReceipt) RotationPublicationReceipt {
	t.Helper()
	canonical, err := rotationReceiptSigningBytes(receipt)
	require.NoError(t, err)
	attestation, err := signer.Sign(context.Background(), receiptsigning.PurposeWalletRecoveryPublicationReceipt, canonical)
	require.NoError(t, err)
	receipt.Attestation = &attestation
	return receipt
}

func prepareAndCommitRotationFixture(t *testing.T, ctx context.Context, service *Service, old Wallet, successorID string) (Wallet, RotationPreparation) {
	t.Helper()
	authority, ok := service.custody.(*ca.Authority)
	require.True(t, ok, "rotation fixture requires the local credential-authority test store")
	signer, verifier := testRotationReceiptSigner(t, authority)
	service.WithRotationReceiptVerifier(verifier)
	prepared, err := service.PreparePortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, successorID)
	require.NoError(t, err)
	receipt := RotationPublicationReceipt{
		Version: 1, WalletID: old.ID, Network: old.Network, Address: old.Address, SuccessorID: prepared.Successor.ID,
		DestinationID: "fixture-off-host-db", DestinationBackend: "s3", PlanID: "fixture-manual-plan", RunID: "fixture-run",
		TargetName: "treasury-wallet-recovery-" + old.ID, SnapshotID: "fixture-snapshot", LatestSnapshotID: "fixture-snapshot",
		TombstoneSHA256: prepared.SHA256, RestoredRevocationSHA256: prepared.SHA256,
		DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: "2026-11-15T12:00:00Z",
	}
	receipt = signTestRotationReceipt(t, signer, receipt)
	committed, err := service.CommitPortableRotation(ctx, old.ID, old.Network, old.OperatorIdentity, prepared.Successor.ID, receipt)
	require.NoError(t, err)
	return committed, prepared
}
