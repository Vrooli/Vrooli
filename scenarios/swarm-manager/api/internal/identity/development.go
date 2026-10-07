package identity

import "encoding/json"

// DevelopmentStanding is retained under the existing effort owner lock. It is
// product disposition and replay evidence, not another commissioning authority.
// AuthorityDigest excludes this runtime state; stale ordinary Save cannot change it.
type DevelopmentStanding struct {
	EvidenceReceipts        map[string]DevelopmentEvidenceReceipt `json:"evidence_receipts,omitempty"`
	History                 map[string]DevelopmentHistoryRecord   `json:"history,omitempty"`
	DispositionVersion      uint64                                `json:"disposition_version"`
	AcceptedReferenceDigest string                                `json:"accepted_reference_digest,omitempty"`
	AcceptedProductDigest   string                                `json:"accepted_product_digest,omitempty"`
	EvidenceSetDigest       string                                `json:"evidence_set_digest,omitempty"`
	Criteria                json.RawMessage                       `json:"criteria,omitempty"`
	Decisions               map[string]DevelopmentDecisionRecord  `json:"decisions,omitempty"`
}

type DevelopmentDecisionRecord struct {
	RequestDigest string `json:"request_digest"`
	ReceiptDigest string `json:"receipt_digest"`
	Receipt       []byte `json:"receipt_bytes"`
}

// Exact immutable bytes remain base64 fields through canonical JSON formatting.
// These are retained references, never a second approval or restored access.
type DevelopmentHistoryRecord struct {
	Reference      []byte            `json:"reference_bytes"`
	Contract       []byte            `json:"contract_bytes"`
	ArtifactBytes  map[string][]byte `json:"artifact_bytes"`
	SnapshotDigest string            `json:"snapshot_digest"`
}

type DevelopmentEvidenceReceipt struct {
	Producer, Digest string
	Binding, Bytes   []byte
}
