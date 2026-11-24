package receiptsigning

// WalletRecoveryCommitAcknowledgement is Treasury's signed proof that one
// exact DBM recovery reservation may be released. OperationID is stable across
// retries and equals the reservation ID; successor identity is the public key
// identity, never secret key material.
type WalletRecoveryCommitAcknowledgement struct {
	Version            int    `json:"version"`
	OperationID        string `json:"operation_id"`
	ReservationID      string `json:"reservation_id"`
	TargetID           string `json:"target_id"`
	DestinationID      string `json:"destination_id"`
	SnapshotID         string `json:"snapshot_id"`
	TombstoneSHA256    string `json:"tombstone_sha256"`
	SuccessorID        string `json:"successor_id"`
	SuccessorAddress   string `json:"successor_address"`
	SuccessorPublicKey string `json:"successor_public_key"`
}
