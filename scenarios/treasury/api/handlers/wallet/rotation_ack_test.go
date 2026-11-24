package wallet

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/receiptsigning"
	walletv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/wallet"
	"treasury/internal/operatorauth"
	domain "treasury/internal/wallet"
)

type rotationAckAuthorizer struct{}

func (rotationAckAuthorizer) Authorize(context.Context, http.Header) (operatorauth.Identity, error) {
	return operatorauth.Identity{Subject: "operator:test"}, nil
}

type rotationAckWallets struct {
	calls       int
	targetID    string
	publication domain.RotationPublicationReceipt
	wallet      domain.Wallet
	ack         receiptsigning.SignatureEnvelope
}

func (s *rotationAckWallets) Create(context.Context, domain.CreateInput) (domain.Wallet, error) {
	return domain.Wallet{}, nil
}
func (s *rotationAckWallets) Rotate(context.Context, string, string, string, string) (domain.Wallet, error) {
	return domain.Wallet{}, nil
}
func (s *rotationAckWallets) Get(context.Context, string, string, string) (domain.Wallet, error) {
	return domain.Wallet{}, nil
}
func (s *rotationAckWallets) Prove(context.Context, string, string, string, string) (domain.Proof, error) {
	return domain.Proof{}, nil
}
func (s *rotationAckWallets) QualifyRecovery(context.Context, string, string, string) (domain.Wallet, error) {
	return domain.Wallet{}, nil
}
func (s *rotationAckWallets) Recover(context.Context, string, string, string) (domain.Wallet, error) {
	return domain.Wallet{}, nil
}
func (s *rotationAckWallets) PreparePortableRotation(context.Context, string, string, string, string) (domain.RotationPreparation, error) {
	return domain.RotationPreparation{}, nil
}
func (s *rotationAckWallets) CommitPortableRotationWithAcknowledgement(_ context.Context, _, _, _, _, targetID string, receipt domain.RotationPublicationReceipt) (domain.Wallet, receiptsigning.SignatureEnvelope, error) {
	s.calls++
	s.targetID = targetID
	s.publication = receipt
	return s.wallet, s.ack, nil
}

var _ Wallets = (*rotationAckWallets)(nil)
var _ RotationPhases = (*rotationAckWallets)(nil)

func TestCommitPortableRotationReturnsSignedExactDBMAcknowledgement(t *testing.T) {
	ack := receiptsigning.WalletRecoveryCommitAcknowledgement{
		Version: 1, OperationID: "reservation-1", ReservationID: "reservation-1", TargetID: "target-1",
		DestinationID: "destination-1", SnapshotID: "snapshot-1", TombstoneSHA256: "tombstone-sha256",
		SuccessorID: "successor-1", SuccessorAddress: "0xsuccessor", SuccessorPublicKey: "successor-public-key",
	}
	envelope := receiptsigning.SignatureEnvelope{
		Version: receiptsigning.EnvelopeVersionV1, Purpose: receiptsigning.PurposeWalletRecoveryCommitAcknowledgement,
		Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519, KeyID: "treasury-test-key",
		Digest: "sha256:ack-digest", Signature: "signed-test-envelope",
	}
	wallets := &rotationAckWallets{wallet: domain.Wallet{ID: "successor-1", Network: "eip155:84532", Address: "0xsuccessor", PublicKey: "successor-public-key"}, ack: envelope}
	handler := NewConnectHandler(rotationAckAuthorizer{}, wallets, nil, nil).(*connectHandler)
	dbmAttestation := receiptsigning.SignatureEnvelope{
		Version: receiptsigning.EnvelopeVersionV1, Purpose: receiptsigning.PurposeWalletRecoveryPublicationReceipt,
		Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519, KeyID: "dbm-test-key",
		Digest: "sha256:publication-digest", Signature: "signed-publication-envelope",
	}
	attestationJSON, err := json.Marshal(dbmAttestation)
	require.NoError(t, err)
	request := connect.NewRequest(&walletv1.CommitPortableResearchWalletRotationRequest{
		Id: "root-1", Network: "eip155:84532", SuccessorId: "successor-1", TargetId: "target-1",
		DestinationId: "destination-1", PlanId: "plan-1", RunId: "run-1", TargetName: "treasury-wallet-recovery-root-1",
		SnapshotId: "snapshot-1", LatestSnapshotId: "snapshot-1", TombstoneSha256: "tombstone-sha256",
		RestoredRevocationSha256: "tombstone-sha256", DbmVerifyPassed: true, RestoredTombstoneMatches: true,
		ReservationId: "reservation-1", DbmAttestationJson: string(attestationJSON),
	})

	response, err := handler.CommitPortableResearchWalletRotation(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, wallets.calls)
	require.Equal(t, "target-1", wallets.targetID)
	require.Equal(t, "destination-1", wallets.publication.DestinationID)
	require.Equal(t, "snapshot-1", wallets.publication.SnapshotID)
	require.Equal(t, "tombstone-sha256", wallets.publication.TombstoneSHA256)
	require.Equal(t, dbmAttestation, *wallets.publication.Attestation)
	require.Equal(t, wallets.wallet.ID, response.Msg.GetWallet().GetId())

	var returned struct {
		Acknowledgement receiptsigning.WalletRecoveryCommitAcknowledgement `json:"acknowledgement"`
		Signature       receiptsigning.SignatureEnvelope                   `json:"signature"`
	}
	require.NoError(t, json.Unmarshal([]byte(response.Msg.GetCommitAckJson()), &returned))
	require.Equal(t, ack, returned.Acknowledgement)
	require.Equal(t, envelope, returned.Signature)

	missingTarget := connect.NewRequest(&walletv1.CommitPortableResearchWalletRotationRequest{
		Id: "root-1", Network: "eip155:84532", SuccessorId: "successor-1", DbmAttestationJson: string(attestationJSON),
	})
	_, err = handler.CommitPortableResearchWalletRotation(context.Background(), missingTarget)
	require.Error(t, err)
	require.Equal(t, 1, wallets.calls, "Treasury must not commit a rotation without the target binding DBM will reserve")
}
