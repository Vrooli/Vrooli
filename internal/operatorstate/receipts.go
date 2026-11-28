package operatorstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const permissionReceiptField = "permission_receipts"
const maxPermissionReceipts = 1024
const permissionReceiptOutcomeUpdated = "updated"

var ErrMutationReceiptConflict = errors.New("operator-state mutation request id conflict")

// MutationReceiptBinding contains only identity and request metadata. The
// payload itself is never persisted in the receipt ledger.
type MutationReceiptBinding struct {
	Owner            string `json:"owner"`
	PermissionID     string `json:"permission_id"`
	ActorID          string `json:"actor_id"`
	Target           string `json:"target"`
	Subject          string `json:"subject"`
	RequestID        string `json:"request_id"`
	ExpectedRevision string `json:"expected_revision"`
	RequestDigest    string `json:"request_digest"`
	AuthorityDigest  string `json:"authority_digest,omitempty"`
	DecisionRef      string `json:"decision_ref"`
}

type MutationReceipt struct {
	ID               string    `json:"id"`
	Owner            string    `json:"owner"`
	PermissionID     string    `json:"permission_id"`
	ActorID          string    `json:"actor_id"`
	Target           string    `json:"target"`
	Subject          string    `json:"subject"`
	RequestID        string    `json:"request_id"`
	ExpectedRevision string    `json:"expected_revision"`
	ResultRevision   string    `json:"result_revision"`
	RequestDigest    string    `json:"request_digest"`
	AuthorityDigest  string    `json:"authority_digest,omitempty"`
	DecisionRef      string    `json:"decision_ref"`
	Outcome          string    `json:"outcome"`
	At               time.Time `json:"at"`
}

// ApplyAtRevisionWithReceipt atomically applies the patch and persists a
// secret-free receipt in the same operator-state document. An exact request
// retry returns the original receipt even when its expected revision is now
// stale; reusing an owner/permission/request ID with changed bindings refuses.
func (s *Service) ApplyAtRevisionWithReceipt(ctx context.Context, expectedRevision string, patch []byte, validate DocumentValidator, binding MutationReceiptBinding) (Document, MutationReceipt, bool, error) {
	return s.applyAtRevisionWithReceipt(ctx, expectedRevision, patch, validate, &binding)
}

// FindMutationReceipt returns a prior receipt for an owner-scoped request ID.
// The caller must compare its actor, target and request digest before replay.
func (s *Service) FindMutationReceipt(ctx context.Context, owner, permissionID, requestID string) (MutationReceipt, bool, error) {
	path, err := s.Path(ctx)
	if err != nil {
		return MutationReceipt{}, false, err
	}
	if err := s.lockContext(ctx); err != nil {
		return MutationReceipt{}, false, err
	}
	defer s.lock.Unlock()
	release, err := s.acquireStateFileLock(ctx, path)
	if err != nil {
		return MutationReceipt{}, false, err
	}
	defer release()
	doc, err := s.loadLocked(path)
	if err != nil {
		return MutationReceipt{}, false, err
	}
	ledger, err := readMutationReceipts(doc)
	if err != nil {
		return MutationReceipt{}, false, err
	}
	receipt, ok := ledger[mutationReceiptKey(owner, permissionID, requestID)]
	return receipt, ok, nil
}

func (s *Service) applyAtRevisionWithReceipt(ctx context.Context, expectedRevision string, patch []byte, validate DocumentValidator, binding *MutationReceiptBinding) (Document, MutationReceipt, bool, error) {
	if !json.Valid(patch) {
		return Document{}, MutationReceipt{}, false, errors.New("operator-state merge patch is invalid JSON")
	}
	var patchValue map[string]json.RawMessage
	if err := json.Unmarshal(patch, &patchValue); err != nil || patchValue == nil {
		return Document{}, MutationReceipt{}, false, errors.New("operator-state merge patch must be a JSON object")
	}
	if binding != nil {
		if strings.TrimSpace(binding.Owner) == "" || strings.TrimSpace(binding.PermissionID) == "" || strings.TrimSpace(binding.ActorID) == "" || strings.TrimSpace(binding.RequestID) == "" || strings.TrimSpace(binding.ExpectedRevision) == "" || strings.TrimSpace(binding.RequestDigest) == "" {
			return Document{}, MutationReceipt{}, false, errors.New("operator-state receipt binding is incomplete")
		}
		if binding.ExpectedRevision != expectedRevision {
			return Document{}, MutationReceipt{}, false, errors.New("operator-state receipt revision does not match mutation revision")
		}
	}
	path, err := s.Path(ctx)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	if err := s.lockContext(ctx); err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	defer s.lock.Unlock()
	release, err := s.acquireStateFileLock(ctx, path)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	defer release()
	current, err := s.loadLocked(path)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	if binding != nil {
		key := mutationReceiptKey(binding.Owner, binding.PermissionID, binding.RequestID)
		ledger, err := readMutationReceipts(current)
		if err != nil {
			return Document{}, MutationReceipt{}, false, err
		}
		if prior, ok := ledger[key]; ok {
			if !receiptMatchesBinding(prior, *binding) {
				return Document{}, MutationReceipt{}, false, ErrMutationReceiptConflict
			}
			return current, prior, true, nil
		}
	}
	if expectedRevision != "" && Revision(current) != expectedRevision {
		return Document{}, MutationReceipt{}, false, fmt.Errorf("%w: expected %q, found %q", ErrRevisionConflict, expectedRevision, Revision(current))
	}
	currentBytes, err := marshalDocument(current)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	merged, err := mergePatch(currentBytes, patch)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	var next Document
	if err := unmarshalDocument(merged, &next); err != nil {
		return Document{}, MutationReceipt{}, false, fmt.Errorf("decode merged operator state: %w", err)
	}
	next.Schema = SchemaPath
	next.Version = documentVersion
	next.UpdatedAt = nextRevisionTimestamp(current.UpdatedAt, s.cfg.Now()).Format(time.RFC3339Nano)
	var receipt MutationReceipt
	if binding != nil {
		receipt = MutationReceipt{ID: mutationReceiptKey(binding.Owner, binding.PermissionID, binding.RequestID), Owner: binding.Owner, PermissionID: binding.PermissionID, ActorID: binding.ActorID, Target: binding.Target, Subject: binding.Subject, RequestID: binding.RequestID, ExpectedRevision: binding.ExpectedRevision, ResultRevision: Revision(next), RequestDigest: binding.RequestDigest, AuthorityDigest: binding.AuthorityDigest, DecisionRef: binding.DecisionRef, Outcome: permissionReceiptOutcomeUpdated, At: nextRevisionTimestamp(current.UpdatedAt, s.cfg.Now())}
		if parsed, parseErr := time.Parse(time.RFC3339Nano, next.UpdatedAt); parseErr == nil {
			receipt.At = parsed
		}
		ledger, err := readMutationReceipts(current)
		if err != nil {
			return Document{}, MutationReceipt{}, false, err
		}
		ledger[receipt.ID] = receipt
		trimMutationReceipts(ledger)
		next.PermissionReceipts = ledger
	}
	data, err := marshalDocument(next)
	if err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	if err := s.validate(data, next); err != nil {
		return Document{}, MutationReceipt{}, false, err
	}
	if validate != nil {
		if err := validate(next); err != nil {
			return Document{}, MutationReceipt{}, false, err
		}
	}
	if err := s.writeStateFile(path, append(data, '\n')); err != nil {
		return Document{}, MutationReceipt{}, false, fmt.Errorf("write operator state: %w", err)
	}
	if s.cfg.Roots != nil {
		s.cfg.Roots.RecordWrite(ctx)
	}
	return next, receipt, false, nil
}

func mutationReceiptKey(owner, permissionID, requestID string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{owner, permissionID, requestID}, "\x00")))
	return hex.EncodeToString(h[:])
}

func readMutationReceipts(doc Document) (map[string]MutationReceipt, error) {
	ledger := map[string]MutationReceipt{}
	for key, receipt := range doc.PermissionReceipts {
		ledger[key] = receipt
	}
	return ledger, nil
}

func receiptMatchesBinding(r MutationReceipt, b MutationReceiptBinding) bool {
	return r.Owner == b.Owner && r.PermissionID == b.PermissionID && r.ActorID == b.ActorID && r.Target == b.Target && r.Subject == b.Subject && r.RequestID == b.RequestID && r.ExpectedRevision == b.ExpectedRevision && r.RequestDigest == b.RequestDigest && r.AuthorityDigest == b.AuthorityDigest && r.DecisionRef == b.DecisionRef
}

func trimMutationReceipts(ledger map[string]MutationReceipt) {
	if len(ledger) <= maxPermissionReceipts {
		return
	}
	keys := make([]string, 0, len(ledger))
	for key := range ledger {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return ledger[keys[i]].At.Before(ledger[keys[j]].At) })
	for _, key := range keys[:len(keys)-maxPermissionReceipts] {
		delete(ledger, key)
	}
}
