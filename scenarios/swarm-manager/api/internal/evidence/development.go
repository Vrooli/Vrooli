package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/effortauthority"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	runs "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"google.golang.org/protobuf/proto"
	"swarm-manager/internal/backlog"
)

var ErrDevelopmentProductFenceUnavailable = errors.New("current mutable product has no qualified commit-held publication fence")

// This narrow read is implemented by the ACTUAL generated RunsService client.
// It cannot start runs, refresh credentials or grant retention pins.
type DevelopmentRunReader interface {
	GetRun(context.Context, *connect.Request[runs.GetRunRequest]) (*connect.Response[runs.GetRunResponse], error)
}

// Bindings are trusted installed setup, never an HTTP receipt/actor/satisfied
// claim. Exact full producer response bytes and its source algorithm are bound.
type DevelopmentReceiptBinding struct {
	Reference                                                                                   *api.DevelopmentReference
	CriterionID, ReceiptID, Target, RunID, ReceiptDigest                                        string
	TreeDigest, SourceScope, ConfigurationFingerprint, DescriptorSnapshotDigest, PhaseSetDigest string
	RequiredPhases                                                                              []string
}
type DevelopmentEvidenceReader struct {
	ledger   *Ledger
	producer DevelopmentRunReader
	bindings map[string][]DevelopmentReceiptBinding
}

func evidenceDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func evidenceValueDigest(v any) string { return "sha256:" + effortauthority.Digest(v) }
func evidenceSHA(v string) bool {
	raw, e := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return e == nil && len(raw) == 32 && strings.ToLower(v) == v
}
func NewDevelopmentEvidenceReader(ledger *Ledger, producer DevelopmentRunReader, bindings []DevelopmentReceiptBinding) (*DevelopmentEvidenceReader, error) {
	if ledger == nil || ledger.db == nil || producer == nil || len(bindings) == 0 || len(bindings) > 256 {
		return nil, backlog.ErrDevelopmentRefused
	}
	raw, err := json.Marshal(bindings)
	if err != nil {
		return nil, err
	}
	var frozen []DevelopmentReceiptBinding
	if json.Unmarshal(raw, &frozen) != nil {
		return nil, backlog.ErrDevelopmentRefused
	}
	out := &DevelopmentEvidenceReader{ledger: ledger, producer: producer, bindings: map[string][]DevelopmentReceiptBinding{}}
	seen := map[string]bool{}
	for _, b := range frozen {
		if b.Reference == nil || b.Reference.EffortId == "" || b.CriterionID == "" || b.ReceiptID == "" || b.Target == "" || b.RunID == "" || !evidenceSHA(b.ReceiptDigest) || !strings.HasPrefix(b.TreeDigest, "td:") || !evidenceSHA(strings.TrimPrefix(b.TreeDigest, "td:")) || b.SourceScope == "" || b.ConfigurationFingerprint == "" || b.DescriptorSnapshotDigest == "" || b.PhaseSetDigest == "" || len(b.RequiredPhases) == 0 || len(b.RequiredPhases) > 64 {
			return nil, backlog.ErrDevelopmentRefused
		}
		key := evidenceValueDigest(b.Reference)
		unique := key + "/" + b.CriterionID
		if seen[unique] {
			return nil, backlog.ErrDevelopmentRefused
		}
		seen[unique] = true
		phases := map[string]bool{}
		for _, phase := range b.RequiredPhases {
			if phase == "" || phases[phase] {
				return nil, backlog.ErrDevelopmentRefused
			}
			phases[phase] = true
		}
		out.bindings[key] = append(out.bindings[key], b)
	}
	return out, nil
}
func validateDevelopmentProducer(raw []byte, b DevelopmentReceiptBinding) (*runs.GetRunResponse, error) {
	if len(raw) == 0 || len(raw) > 512*1024 || evidenceDigest(raw) != b.ReceiptDigest {
		return nil, backlog.ErrDevelopmentRefused
	}
	response := new(runs.GetRunResponse)
	if proto.Unmarshal(raw, response) != nil || response.Run == nil || response.TerminalSnapshotSchemaVersion < 1 || len(response.DegradedReasons) > 0 {
		return nil, backlog.ErrDevelopmentRefused
	}
	r := response.Run
	if r.RunId != b.RunID || r.Target != b.Target || r.Status != "passed" || !r.SourceStable || r.TreeDigest != b.TreeDigest || r.SourceScope != b.SourceScope || r.ExecutionConfigurationFingerprint != b.ConfigurationFingerprint || r.DescriptorSnapshotDigest != b.DescriptorSnapshotDigest || r.PhaseSetDigest != b.PhaseSetDigest || r.DescriptorSnapshotSchemaVersion < 1 || r.DescriptorSnapshot == nil || r.DescriptorSnapshot.Digest != b.DescriptorSnapshotDigest {
		return nil, backlog.ErrDevelopmentRefused
	}
	start, se := time.Parse(time.RFC3339Nano, r.StartedAt)
	end, ee := time.Parse(time.RFC3339Nano, r.CompletedAt)
	if se != nil || ee != nil || end.Before(start) || end.After(time.Now()) {
		return nil, backlog.ErrDevelopmentRefused
	}
	planned := map[string]bool{}
	for _, name := range r.PlannedPhases {
		if planned[name] {
			return nil, backlog.ErrDevelopmentRefused
		}
		planned[name] = true
	}
	actual := map[string]bool{}
	for _, phase := range r.Phases {
		if phase == nil || actual[phase.Name] || !planned[phase.Name] || phase.Status != "passed" || !phase.Comparable || phase.Advisory || !phase.ArtifactBacked {
			return nil, backlog.ErrDevelopmentRefused
		}
		actual[phase.Name] = true
	}
	if len(planned) != len(actual) {
		return nil, backlog.ErrDevelopmentRefused
	}
	for _, required := range b.RequiredPhases {
		if !planned[required] || !actual[required] {
			return nil, backlog.ErrDevelopmentRefused
		}
	}
	return response, nil
}

// CaptureProducerReceipt is an unmounted owner import from the actual read-only
// producer. It selects one routed pool once and stores exact immutable bytes.
// No acceptance RPC imports a receipt or silently upgrades legacy observations.
func (o *DevelopmentEvidenceReader) CaptureProducerReceipts(ctx context.Context, ref *api.DevelopmentReference) error {
	if o == nil || o.ledger == nil || o.ledger.db == nil || ref == nil {
		return backlog.ErrDevelopmentRefused
	}
	bindings := o.bindings[evidenceValueDigest(ref)]
	if len(bindings) == 0 {
		return backlog.ErrDevelopmentRefused
	}
	pool, err := o.ledger.db.PoolForContext(ctx)
	if err != nil || pool == nil {
		return backlog.ErrDevelopmentRefused
	}
	type captured struct {
		b   DevelopmentReceiptBinding
		raw []byte
	}
	rows := make([]captured, 0, len(bindings))
	for _, b := range bindings {
		response, err := o.producer.GetRun(ctx, connect.NewRequest(&runs.GetRunRequest{Target: b.Target, RunId: b.RunID}))
		if err != nil || response == nil || response.Msg == nil {
			return backlog.ErrDevelopmentRefused
		}
		raw, err := (proto.MarshalOptions{Deterministic: true}).Marshal(response.Msg)
		if err != nil {
			return err
		}
		if _, err := validateDevelopmentProducer(raw, b); err != nil {
			return err
		}
		rows = append(rows, captured{b, raw})
	}
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, row := range rows {
		var existing, bind []byte
		err := tx.QueryRowContext(ctx, `SELECT receipt_bytes,binding_bytes FROM development_producer_receipts WHERE id=?`, row.b.ReceiptID).Scan(&existing, &bind)
		bindingBytes, e := json.Marshal(row.b)
		if e != nil {
			return e
		}
		if err == nil {
			if !bytes.Equal(existing, row.raw) || !bytes.Equal(bind, bindingBytes) {
				return backlog.ErrDevelopmentRefused
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO development_producer_receipts(id,producer,run_id,reference_digest,criterion_id,content_digest,binding_bytes,receipt_bytes) VALUES(?,?,?,?,?,?,?,?)`, row.b.ReceiptID, "test-genie", row.b.RunID, evidenceValueDigest(ref), row.b.CriterionID, row.b.ReceiptDigest, bindingBytes, row.raw); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (o *DevelopmentEvidenceReader) ReadDevelopmentEvidence(ctx context.Context, ref *api.DevelopmentReference, criteria []string) (backlog.DevelopmentEvidenceSnapshot, error) {
	var snapshot backlog.DevelopmentEvidenceSnapshot
	if o == nil || o.ledger == nil || o.ledger.db == nil || ref == nil || len(criteria) == 0 {
		return snapshot, backlog.ErrDevelopmentRefused
	}
	bindings := o.bindings[evidenceValueDigest(ref)]
	if len(bindings) != len(criteria) {
		return snapshot, backlog.ErrDevelopmentRefused
	}
	pool, err := o.ledger.db.PoolForContext(ctx)
	if err != nil || pool == nil {
		return snapshot, backlog.ErrDevelopmentRefused
	}
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return snapshot, err
	}
	defer tx.Rollback()
	required := map[string]bool{}
	for _, key := range criteria {
		if required[key] {
			return snapshot, backlog.ErrDevelopmentRefused
		}
		required[key] = true
	}
	var digests []string
	for _, b := range bindings {
		if !required[b.CriterionID] {
			return snapshot, backlog.ErrDevelopmentRefused
		}
		var raw, bindingBytes []byte
		var producer, runID, refDigest, criterion, contentDigest string
		if err := tx.QueryRowContext(ctx, `SELECT producer,run_id,reference_digest,criterion_id,content_digest,binding_bytes,receipt_bytes FROM development_producer_receipts WHERE id=?`, b.ReceiptID).Scan(&producer, &runID, &refDigest, &criterion, &contentDigest, &bindingBytes, &raw); err != nil {
			return snapshot, backlog.ErrDevelopmentRefused
		}
		expected, _ := json.Marshal(b)
		if producer != "test-genie" || runID != b.RunID || refDigest != evidenceValueDigest(ref) || criterion != b.CriterionID || contentDigest != b.ReceiptDigest || !bytes.Equal(expected, bindingBytes) {
			return snapshot, backlog.ErrDevelopmentRefused
		}
		response, err := validateDevelopmentProducer(raw, b)
		if err != nil {
			return snapshot, err
		}
		// Explicit algorithm/scope identity; never alias this to HEAD, a strict
		// BuildInputManifest, complete dependency closure or current checkout proof.
		product := evidenceValueDigest(struct{ Algorithm, Scope, Fingerprint string }{"treedigest.Compute/v1", response.Run.SourceScope, response.Run.TreeDigest})
		if snapshot.ProductDigest != "" && snapshot.ProductDigest != product {
			return snapshot, backlog.ErrDevelopmentRefused
		}
		snapshot.ProductDigest = product
		snapshot.Criteria = append(snapshot.Criteria, &api.DevelopmentCriterionEvidence{CriterionId: b.CriterionID, ReceiptIds: []string{b.ReceiptID}, ReceiptSetDigest: evidenceValueDigest([]string{b.ReceiptDigest}), TestedProductDigest: product, Satisfied: true})
		snapshot.Receipts = append(snapshot.Receipts, backlog.DevelopmentRetainedEvidence{ID: b.ReceiptID, Producer: producer, Digest: b.ReceiptDigest, Binding: append([]byte(nil), bindingBytes...), Bytes: append([]byte(nil), raw...)})
		digests = append(digests, b.ReceiptDigest)
	}
	if err := tx.Commit(); err != nil {
		return snapshot, err
	}
	sort.Strings(digests)
	sort.Slice(snapshot.Criteria, func(i, j int) bool { return snapshot.Criteria[i].CriterionId < snapshot.Criteria[j].CriterionId })
	snapshot.EvidenceSetDigest = evidenceValueDigest(digests)
	snapshot.Version = evidenceValueDigest(snapshot)
	return snapshot, nil
}

// No inspected owner excludes writes to the current shared dirty product. This
// concrete implementation REFUSES before invoking save. SourceStable, an SQL
// transaction, a run pin or another hash must never substitute for a product
// owner publication capability held through commit. Future supported writers
// must implement and qualify that concrete owner protocol before activation.
func (o *DevelopmentEvidenceReader) CommitDevelopmentEvidence(ctx context.Context, ref *api.DevelopmentReference, snapshot backlog.DevelopmentEvidenceSnapshot, save func(func() error) error) error {
	return fmt.Errorf("%w", ErrDevelopmentProductFenceUnavailable)
}

var _ backlog.DevelopmentEvidenceOwner = (*DevelopmentEvidenceReader)(nil)
