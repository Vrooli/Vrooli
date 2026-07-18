package pairing

import (
	"context"
	"crypto/ed25519"
)

// Repository is the persistence seam the pairing service depends on. Production
// wires the sqlite implementation; service unit tests wire mocks.FakeRepository.
type Repository interface {
	// CreateCode persists a new (hashed) pairing code. The implementation
	// populates ID/CreatedAt.
	CreateCode(ctx context.Context, c PairingCode) (PairingCode, error)

	// GetCodeByHash returns the code with the given hash, or ErrCodeNotFound.
	GetCodeByHash(ctx context.Context, codeHash string) (PairingCode, error)

	// BurnCode marks the code redeemed by nodeID at the current time. It is the
	// single-use guard: it MUST atomically fail (ErrCodeUsed) if the code was
	// already burned, so two concurrent redeems cannot both succeed.
	BurnCode(ctx context.Context, id, nodeID string) error

	// StoreCredential persists (or replaces) a node's Ed25519 public key.
	StoreCredential(ctx context.Context, c Credential) error

	// RevokeCredential severs a node's credential (idempotent). A no-op when no
	// credential exists.
	RevokeCredential(ctx context.Context, nodeID string) error

	// ActivePublicKey returns the node's public key iff it has a non-revoked
	// credential. This satisfies nodeauth.CredentialStore.
	ActivePublicKey(ctx context.Context, nodeID string) (ed25519.PublicKey, bool, error)

	// CreateRequest persists a pending pairing request.
	CreateRequest(ctx context.Context, r PairingRequest) (PairingRequest, error)

	// GetRequest returns the request by id, or ErrRequestNotFound.
	GetRequest(ctx context.Context, id string) (PairingRequest, error)

	// DecideRequest records an approve/reject decision (and the minted node id
	// on approval). It MUST fail (ErrRequestDecided) if already decided.
	DecideRequest(ctx context.Context, id string, status RequestStatus, nodeID string) error

	// ListRequests returns pending requests (newest-first); includeDecided adds
	// approved/rejected ones.
	ListRequests(ctx context.Context, includeDecided bool) ([]PairingRequest, error)
}

// EnrollmentRepository is implemented by durable pairing repositories. It is
// deliberately separate from Repository so small legacy test fakes remain
// focused; production requires it whenever a correlation is supplied.
type EnrollmentRepository interface {
	ClaimCode(context.Context, string) error
	PrepareEnrollmentSaga(context.Context, EnrollmentSaga) (EnrollmentSaga, error)
	GetEnrollmentSaga(context.Context, string) (EnrollmentSaga, error)
	UpdateEnrollmentSaga(context.Context, EnrollmentSaga) error
	ListIncompleteEnrollmentSagas(context.Context) ([]EnrollmentSaga, error)
	FinalizeClaimedCode(context.Context, string, string) error
}
