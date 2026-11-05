package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/certification"

	"github.com/vrooli/api-core/receiptsigning"
)

// RecordSchemaVersion is stamped on every evidence record.
const RecordSchemaVersion = 1

// Binding is the producer/operation/artifact/target identity every record
// and receipt carries. A consumer compares each field with what it expected;
// a missing field is a refusal, never a wildcard.
type Binding struct {
	ProducerRef   string `json:"producer_ref"`
	DeploymentID  string `json:"deployment_id"`
	ReleaseDigest string `json:"release_digest"`
	TargetKey     string `json:"target_key"`
	OperationID   string `json:"operation_id,omitempty"`
	ObservationID string `json:"observation_id,omitempty"`
}

// Validate refuses a binding without producer, deployment, release and
// target identity. Operation or observation must name the producing act.
func (b Binding) Validate() error {
	if strings.TrimSpace(b.ProducerRef) == "" {
		return fmt.Errorf("binding requires producer_ref")
	}
	if strings.TrimSpace(b.DeploymentID) == "" || strings.TrimSpace(b.ReleaseDigest) == "" || strings.TrimSpace(b.TargetKey) == "" {
		return fmt.Errorf("binding requires deployment_id, release_digest and target_key")
	}
	if strings.TrimSpace(b.OperationID) == "" && strings.TrimSpace(b.ObservationID) == "" {
		return fmt.Errorf("binding requires an operation_id or observation_id")
	}
	return nil
}

// Expect is what a consumer knows; empty fields are not checked so a caller
// may bind to a subset, but ProducerRef is always compared.
type Expect struct {
	ProducerRef   string
	DeploymentID  string
	ReleaseDigest string
	TargetKey     string
	OperationID   string
}

// Check compares the binding with the consumer's expectation.
func (b Binding) Check(expect Expect) error {
	want := expect.ProducerRef
	if want == "" {
		want = ProducerRef
	}
	if b.ProducerRef != want {
		return fmt.Errorf("producer_ref %q is not the cloud service principal %q", b.ProducerRef, want)
	}
	if expect.DeploymentID != "" && b.DeploymentID != expect.DeploymentID {
		return fmt.Errorf("record is bound to deployment %q, expected %q", b.DeploymentID, expect.DeploymentID)
	}
	if expect.ReleaseDigest != "" && !strings.EqualFold(b.ReleaseDigest, expect.ReleaseDigest) {
		return fmt.Errorf("record is bound to release %q, expected %q", b.ReleaseDigest, expect.ReleaseDigest)
	}
	if expect.TargetKey != "" && b.TargetKey != expect.TargetKey {
		return fmt.Errorf("record is bound to target %q, expected %q", b.TargetKey, expect.TargetKey)
	}
	if expect.OperationID != "" && b.OperationID != expect.OperationID {
		return fmt.Errorf("record is bound to operation %q, expected %q", b.OperationID, expect.OperationID)
	}
	return nil
}

// Record is one append-only evidence record for one cell. A rerun never
// edits a record: it appends a new one, and a failed record stays in force
// until an owner appends a withdrawal that names it and says why.
type Record struct {
	SchemaVersion int                `json:"schema_version"`
	ID            string             `json:"id"`
	ProfileID     string             `json:"profile_id"`
	CaseID        string             `json:"case_id"`
	Lane          certification.Lane `json:"lane"`
	Disposition   Disposition        `json:"disposition"`
	Binding       Binding            `json:"binding"`
	// ConfigurationDigest is recorded so evidence for the same release digest
	// under a different configuration is still visible.
	ConfigurationDigest string `json:"configuration_digest,omitempty"`
	// ReceiptRefs are opaque owner-scoped references (target receipts,
	// operation step receipts, observation ids). Never local paths.
	ReceiptRefs []string `json:"receipt_refs,omitempty"`
	Assertions  []string `json:"assertions,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	// Withdraws names a failed record this record withdraws. Withdrawal is an
	// explicit owner act with a reason; it never happens because a later run
	// passed.
	Withdraws  string    `json:"withdraws,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
	RecordedAt time.Time `json:"recorded_at"`
	// Signature is the producer attestation over CanonicalJSON. Absent when
	// no signer was configured; consumers may require it.
	Signature *receiptsigning.SignatureEnvelope `json:"signature,omitempty"`
}

// Validate checks the record before it is appended.
func (r Record) Validate(profile *Profile) error {
	if r.SchemaVersion != RecordSchemaVersion {
		return fmt.Errorf("record %s: unsupported schema_version %d", r.ID, r.SchemaVersion)
	}
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("record requires an id")
	}
	if profile != nil {
		if r.ProfileID != profile.ID {
			return fmt.Errorf("record %s: profile %q is not %q", r.ID, r.ProfileID, profile.ID)
		}
		if !profile.Contains(CellID{CaseID: r.CaseID, Lane: r.Lane}) {
			return fmt.Errorf("record %s: cell %s/%s is not in profile %s", r.ID, r.CaseID, r.Lane, profile.ID)
		}
	}
	if !r.Disposition.Recordable() {
		return fmt.Errorf("record %s: disposition %q cannot be recorded", r.ID, r.Disposition)
	}
	if err := r.Binding.Validate(); err != nil {
		return fmt.Errorf("record %s: %w", r.ID, err)
	}
	if r.Disposition == DispositionPassed && len(r.ReceiptRefs) == 0 {
		return fmt.Errorf("record %s: a passed cell requires target-owned receipt references", r.ID)
	}
	if r.Disposition != DispositionPassed && strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("record %s: disposition %s requires a reason", r.ID, r.Disposition)
	}
	if r.Withdraws != "" && strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("record %s: withdrawal requires a reason", r.ID)
	}
	if r.ObservedAt.IsZero() || r.RecordedAt.IsZero() {
		return fmt.Errorf("record %s: observed_at and recorded_at are required", r.ID)
	}
	return nil
}

// CanonicalJSON is the signing payload: every field except the signature,
// with sorted keys. encoding/json sorts map keys, so the record is marshalled
// through a map to make the order independent of struct layout.
func (r Record) CanonicalJSON() ([]byte, error) {
	r.Signature = nil
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return json.Marshal(generic)
}

// ContentDigest is sha256 over the canonical JSON.
func (r Record) ContentDigest() (string, error) {
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Sign attaches the producer attestation.
func (r *Record) Sign(ctx context.Context, signer receiptsigning.ReceiptSigner) error {
	if signer == nil {
		return fmt.Errorf("receipt signer is unavailable")
	}
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return err
	}
	envelope, err := signer.Sign(ctx, receiptsigning.PurposeCloudEvidenceReceipt, canonical)
	if err != nil {
		return err
	}
	r.Signature = &envelope
	return nil
}

// VerifySignature checks the attestation. A record without a signature is
// reported as unsigned so the consumer decides whether that is acceptable.
func (r Record) VerifySignature(ctx context.Context, signer receiptsigning.ReceiptSigner) error {
	if r.Signature == nil {
		return ErrUnsigned
	}
	if signer == nil {
		return fmt.Errorf("receipt signer is unavailable")
	}
	if r.Signature.Purpose != receiptsigning.PurposeCloudEvidenceReceipt {
		return fmt.Errorf("record %s: signature purpose %q is not %q", r.ID, r.Signature.Purpose, receiptsigning.PurposeCloudEvidenceReceipt)
	}
	canonical, err := r.CanonicalJSON()
	if err != nil {
		return err
	}
	return signer.Verify(ctx, *r.Signature, canonical)
}

// ErrUnsigned marks a record or receipt that carries no attestation.
var ErrUnsigned = fmt.Errorf("evidence carries no producer signature")

// SortRecords orders records by observed time then id so evaluation is
// deterministic.
func SortRecords(records []Record) {
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].ObservedAt.Equal(records[j].ObservedAt) {
			return records[i].ObservedAt.Before(records[j].ObservedAt)
		}
		return records[i].ID < records[j].ID
	})
}
