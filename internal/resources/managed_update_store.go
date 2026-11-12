package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

var (
	ErrManagedUpdateConflict      = errors.New("managed update conflicts with retained operation")
	ErrManagedUpdateRevision      = errors.New("managed update checkpoint revision changed")
	ErrManagedUpdateNotFound      = errors.New("managed update request is not recorded")
	ErrManagedUpdateCommitUnknown = errors.New("managed update commit durability is unconfirmed; reconcile the same request")
)

const (
	managedUpdateLedgerFile = "managed-updates.json"
	managedUpdateMaxRecords = 128
	managedUpdateMaxBytes   = 16 << 20
)

// LookupManagedServiceUpdate is a read-only checkpoint lookup. It does not
// recommit, renew capacity or perform lifecycle effects.
func LookupManagedServiceUpdate(resource, request string) (ManagedServiceUpdate, error) {
	if !managedUpdateResource.MatchString(resource) || !managedUpdateKey.MatchString(request) {
		return ManagedServiceUpdate{}, fmt.Errorf("invalid update selector")
	}
	store, err := newManagedUpdateStore(resource)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	return store.Lookup(request)
}

type managedUpdateEntry struct {
	State            ManagedServiceUpdate `json:"state"`
	LastInputDigest  string               `json:"last_input_digest,omitempty"`
	LastBaseRevision uint64               `json:"last_base_revision,omitempty"`
}
type managedUpdateLedger struct {
	SchemaVersion int                  `json:"schema_version"`
	Resource      string               `json:"resource"`
	Records       []managedUpdateEntry `json:"records"`
}

// managedUpdateStore is the resource domain's file repository. It uses the
// resource storage resolver, not a scenario database or a caller's cwd. State
// locking is cross-process and follows the lifecycle lock when both are needed.
// Store callbacks never acquire a lifecycle lock or perform lifecycle effects.
type managedUpdateStore struct {
	resource string
	path     string
	lock     func(context.Context) (func(), error)
	write    func(string, []byte, os.FileMode) error
	now      func() time.Time
}

func newManagedUpdateStore(resource string) (*managedUpdateStore, error) {
	if !managedUpdateResource.MatchString(resource) {
		return nil, fmt.Errorf("invalid managed update resource")
	}
	paths, err := resourceStoragePaths(resource)
	if err != nil {
		return nil, err
	}
	return &managedUpdateStore{
		resource: resource, path: filepath.Join(paths.StateDir, managedUpdateLedgerFile),
		lock: func(ctx context.Context) (func(), error) {
			if nativeOpenRGBUsesPrivateLocks(resource) {
				return acquireOpenRGBNativeOwnerLock(ctx, "resource-update-state:"+resource)
			}
			return artifactlock.AcquireContext(ctx, "resource-update-state:"+resource)
		},
		write: writeResourceOwnedFileDurable, now: time.Now,
	}, nil
}

func (s *managedUpdateStore) read() (managedUpdateLedger, error) {
	ledger := managedUpdateLedger{SchemaVersion: managedUpdateSchemaVersion, Resource: s.resource, Records: []managedUpdateEntry{}}
	file, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return ledger, nil
	}
	if err != nil {
		return managedUpdateLedger{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, managedUpdateMaxBytes+1))
	if err != nil {
		return managedUpdateLedger{}, err
	}
	if len(data) > managedUpdateMaxBytes {
		return managedUpdateLedger{}, fmt.Errorf("managed update ledger exceeds size bound")
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return managedUpdateLedger{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ledger); err != nil {
		return managedUpdateLedger{}, fmt.Errorf("decode managed update ledger: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return managedUpdateLedger{}, fmt.Errorf("managed update ledger has trailing content")
	}
	if ledger.SchemaVersion != managedUpdateSchemaVersion || ledger.Resource != s.resource || len(ledger.Records) > managedUpdateMaxRecords {
		return managedUpdateLedger{}, fmt.Errorf("invalid managed update ledger identity or capacity")
	}
	ids, requests := map[string]bool{}, map[string]bool{}
	active := 0
	for index := range ledger.Records {
		entry := &ledger.Records[index]
		entry.State.Target.markLegacyFileConfig()
		entry.State.Rollback.markLegacyFileConfig()
		u := entry.State
		if err := u.Validate(); err != nil {
			return managedUpdateLedger{}, fmt.Errorf("invalid retained update: %w", err)
		}
		if u.Resource != s.resource || ids[u.ID] || requests[u.RequestID] {
			return managedUpdateLedger{}, fmt.Errorf("managed update ledger has conflicting identities")
		}
		ids[u.ID], requests[u.RequestID] = true, true
		if !u.Phase.Terminal() {
			active++
		}
		if u.Revision == 1 {
			if entry.LastBaseRevision != 0 || entry.LastInputDigest != "" {
				return managedUpdateLedger{}, fmt.Errorf("initial update has unexpected mutation identity")
			}
		} else if entry.LastBaseRevision != u.Revision-1 || !updateDigest(entry.LastInputDigest) {
			return managedUpdateLedger{}, fmt.Errorf("update mutation identity is incomplete")
		}
	}
	if active > 1 {
		return managedUpdateLedger{}, fmt.Errorf("managed update ledger has competing active operations")
	}
	return ledger, nil
}

// Durable control documents must have one interpretation. encoding/json's
// last-duplicate-key-wins behavior could otherwise turn a consumed exec fence
// back into an apparently prepared one. Keep syntax checks in this domain;
// the typed decoder below still owns the actual schema.
func validateManagedUpdateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 128 {
			return fmt.Errorf("managed update document exceeds nesting bound")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		close := json.Delim(']')
		if delim == '{' {
			close = '}'
		} else if delim != '[' {
			return fmt.Errorf("invalid managed update document delimiter")
		}
		seen := map[string]bool{}
		for decoder.More() {
			if delim == '{' {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("managed update document has duplicate or invalid object fields")
				}
				seen[name] = true
			}
			if err := value(depth + 1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != close {
			return fmt.Errorf("managed update document has an incomplete compound value")
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("managed update document has trailing content")
	}
	return nil
}

func (s *managedUpdateStore) save(ledger managedUpdateLedger) error {
	data, err := json.Marshal(ledger)
	if err != nil {
		return err
	}
	if len(data) > managedUpdateMaxBytes {
		return fmt.Errorf("managed update ledger capacity exhausted; preserve existing records")
	}
	if err := s.write(s.path, data, 0o600); err != nil {
		return fmt.Errorf("%w: %w", ErrManagedUpdateCommitUnknown, err)
	}
	return nil
}

func (s *managedUpdateStore) Lookup(requestID string) (ManagedServiceUpdate, error) {
	ledger, err := s.read()
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	for _, entry := range ledger.Records {
		if entry.State.RequestID == requestID {
			return entry.State, nil
		}
	}
	return ManagedServiceUpdate{}, ErrManagedUpdateNotFound
}

func (s *managedUpdateStore) Active() (ManagedServiceUpdate, error) {
	ledger, err := s.read()
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	for _, entry := range ledger.Records {
		if !entry.State.Phase.Terminal() {
			return entry.State, nil
		}
	}
	return ManagedServiceUpdate{}, ErrManagedUpdateNotFound
}

// The caller holds the resource lifecycle lock. Ordinary lifecycle actions
// cannot interleave between durable update checkpoints, including after the
// update caller exits. Recovery uses its frozen owner path instead.
func ensureManagedUpdateLifecycleAvailable(resource, action string) error {
	s, err := newManagedUpdateStore(resource)
	if err != nil {
		return err
	}
	u, err := s.Active()
	if errors.Is(err, ErrManagedUpdateNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read managed update ownership before %s: %w", action, err)
	}
	return fmt.Errorf("%w: operation %s in %s owns %s; reconcile it before ordinary %s", ErrManagedUpdateConflict, u.ID, u.Phase, resource, action)
}

// Confirm makes a reconstructed checkpoint durable before the effect owner
// resumes it. Lookup/Active are inspection only: a visible rename may have lost
// its durability acknowledgement in the previous process. Reconfirm the exact
// revision under the state lock, then reconcile native identity before acting.
func (s *managedUpdateStore) Confirm(ctx context.Context, requestID string, revision uint64) (ManagedServiceUpdate, error) {
	release, err := s.lock(ctx)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return ManagedServiceUpdate{}, err
	}
	ledger, err := s.read()
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	for _, entry := range ledger.Records {
		if entry.State.RequestID != requestID {
			continue
		}
		if entry.State.Revision != revision {
			return ManagedServiceUpdate{}, ErrManagedUpdateRevision
		}
		if err := s.save(ledger); err != nil {
			return ManagedServiceUpdate{}, err
		}
		return entry.State, nil
	}
	return ManagedServiceUpdate{}, ErrManagedUpdateNotFound
}

// Create admits at most one operation. A request ID is never automatically
// forgotten: bounded capacity fails closed rather than allowing an old request
// to dispatch again. Explicit retry is a new request, not an overwrite of the
// failed release or an inferred permission from a previous admission.
func (s *managedUpdateStore) Create(ctx context.Context, initial ManagedServiceUpdate, explicitRetry bool) (ManagedServiceUpdate, bool, error) {
	if err := initial.Validate(); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	if initial.Resource != s.resource || initial.Phase != UpdatePrepared || initial.Revision != 1 || initial.RollbackProbe.Intent != nil || initial.RollbackQualifiedAt != nil || initial.RollbackProbeQuiescent || initial.Candidate.Intent != nil || initial.Primary.Intent != nil || initial.Restored.Intent != nil || len(initial.RetiredRestores) != 0 || initial.PrimaryStopVerified || initial.PrimaryQuiescent || initial.CandidateQuiescent || initial.AbortRequested || initial.Error != "" || initial.RecoveryFrom != "" {
		return ManagedServiceUpdate{}, false, fmt.Errorf("update admission must precede all effects")
	}
	release, err := s.lock(ctx)
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	ledger, err := s.read()
	if err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	for _, entry := range ledger.Records {
		old := entry.State
		if old.RequestID == initial.RequestID {
			if old.RollbackOf != initial.RollbackOf {
				return ManagedServiceUpdate{}, false, ErrManagedUpdateConflict
			}
			if old.SourceRevision != initial.SourceRevision || old.Target.Digest != initial.Target.Digest || old.Rollback.Digest != initial.Rollback.Digest || old.RollbackQualificationRequired != initial.RollbackQualificationRequired || !reflect.DeepEqual(old.Original, initial.Original) || !reflect.DeepEqual(old.OriginalScope, initial.OriginalScope) || !reflect.DeepEqual(old.Credentials, initial.Credentials) || !reflect.DeepEqual(old.Capacity, initial.Capacity) {
				return ManagedServiceUpdate{}, false, ErrManagedUpdateConflict
			}
			// A lost fsync acknowledgement can leave the rename visible. Recommit
			// the same ledger durably before returning a successful replay.
			if err := s.save(ledger); err != nil {
				return ManagedServiceUpdate{}, false, err
			}
			return old, true, nil
		}
	}
	for _, entry := range ledger.Records {
		old := entry.State
		if old.ID == initial.ID || !old.Phase.Terminal() {
			return ManagedServiceUpdate{}, false, ErrManagedUpdateConflict
		}
		if !explicitRetry && old.Phase != UpdateSucceeded && old.Target.Artifact.SHA256 == initial.Target.Artifact.SHA256 {
			return ManagedServiceUpdate{}, false, fmt.Errorf("%w: failed release requires an explicit new retry", ErrManagedUpdateConflict)
		}
	}
	if len(ledger.Records) >= managedUpdateMaxRecords {
		return ManagedServiceUpdate{}, false, fmt.Errorf("managed update ledger capacity exhausted; existing recovery receipts retained")
	}
	if initial.Target.Artifact.SHA256 == initial.Rollback.Artifact.SHA256 {
		return ManagedServiceUpdate{}, false, fmt.Errorf("target is already the retained release")
	}
	ledger.Records = append(ledger.Records, managedUpdateEntry{State: initial})
	if err := s.save(ledger); err != nil {
		return ManagedServiceUpdate{}, false, err
	}
	return initial, false, nil
}

// Commit applies observations to an exact checkpoint revision and advances one
// legal event. Callers supply the previously read checkpoint plus observations;
// they cannot replace launch contracts or existing process ownership. The exact
// same input replays after a lost response, while a different stale input fails.
func (s *managedUpdateStore) Commit(ctx context.Context, observed ManagedServiceUpdate, event ManagedUpdateEvent) (ManagedServiceUpdate, error) {
	data, err := json.Marshal(struct {
		State ManagedServiceUpdate `json:"state"`
		Event ManagedUpdateEvent   `json:"event"`
	}{observed, event})
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	sum := sha256.Sum256(data)
	inputDigest := hex.EncodeToString(sum[:])
	release, err := s.lock(ctx)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return ManagedServiceUpdate{}, err
	}
	ledger, err := s.read()
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	for i, entry := range ledger.Records {
		old := entry.State
		if old.ID != observed.ID || old.RequestID != observed.RequestID {
			continue
		}
		if entry.LastBaseRevision == observed.Revision && entry.LastInputDigest == inputDigest {
			if err := s.save(ledger); err != nil {
				return ManagedServiceUpdate{}, err
			}
			return old, nil
		}
		if old.Revision != observed.Revision {
			return ManagedServiceUpdate{}, ErrManagedUpdateRevision
		}
		validateObservation := validManagedUpdateObservation
		if event == UpdateFinishDisabled && old.Resource == openRGBResourceName {
			validateObservation = validOpenRGBUpdateRetirementObservation
		}
		if err := validateObservation(old, observed); err != nil {
			return ManagedServiceUpdate{}, err
		}
		next, err := transitionManagedUpdate(observed, event, s.now().UTC())
		if err != nil {
			return ManagedServiceUpdate{}, err
		}
		ledger.Records[i] = managedUpdateEntry{State: next, LastBaseRevision: old.Revision, LastInputDigest: inputDigest}
		if err := s.save(ledger); err != nil {
			return ManagedServiceUpdate{}, err
		}
		return next, nil
	}
	return ManagedServiceUpdate{}, ErrManagedUpdateNotFound
}

func validManagedUpdateObservation(old, next ManagedServiceUpdate) error {
	// Replace every observable field with its old value; everything left must
	// match exactly, including the phase/revision and frozen source contracts.
	immutable := next
	immutable.OriginalProof = old.OriginalProof
	immutable.Candidate, immutable.Primary, immutable.Restored = old.Candidate, old.Primary, old.Restored
	immutable.RollbackProbe = old.RollbackProbe
	immutable.RollbackProbeQuiescent = old.RollbackProbeQuiescent
	immutable.Error = old.Error
	immutable.CandidateQuiescent, immutable.PrimaryStopVerified, immutable.PrimaryQuiescent = old.CandidateQuiescent, old.PrimaryStopVerified, old.PrimaryQuiescent
	if !reflect.DeepEqual(immutable, old) {
		return fmt.Errorf("%w: immutable update checkpoint changed", ErrManagedUpdateConflict)
	}
	for _, pair := range []struct {
		old, next               ManagedUpdateProcess
		intentPhase, statePhase ManagedUpdatePhase
	}{
		{old.RollbackProbe, next.RollbackProbe, UpdatePrepared, UpdateRollbackProbeStarting},
		{old.Candidate, next.Candidate, UpdatePrepared, UpdateCandidateStarting},
		{old.Primary, next.Primary, UpdateCandidateReady, UpdatePrimaryStarting},
		{old.Restored, next.Restored, UpdateRestoring, UpdateRestoreStarting},
	} {
		if pair.old.Intent != nil && !reflect.DeepEqual(pair.old.Intent, pair.next.Intent) {
			return fmt.Errorf("%w: launch identity changed", ErrManagedUpdateConflict)
		}
		if pair.old.Intent == nil && pair.next.Intent != nil && old.Phase != pair.intentPhase {
			if !(pair.old == old.Primary && old.Phase == UpdatePrepared && exclusiveStopStart(old)) {
				return fmt.Errorf("launch intent was not recorded before its dispatch phase")
			}
		}
		if pair.old.State != nil && !reflect.DeepEqual(pair.old.State, pair.next.State) {
			return fmt.Errorf("%w: owned process identity changed", ErrManagedUpdateConflict)
		}
		if pair.old.State == nil && pair.next.State != nil && old.Phase != pair.statePhase {
			return fmt.Errorf("process observed outside its launch reconciliation phase")
		}
		if pair.old.NativeRef != nil && !reflect.DeepEqual(pair.old.NativeRef, pair.next.NativeRef) {
			return fmt.Errorf("%w: native service identity changed", ErrManagedUpdateConflict)
		}
		if pair.old.NativeRef == nil && pair.next.NativeRef != nil && old.Phase != pair.statePhase {
			return fmt.Errorf("native service observed outside its launch reconciliation phase")
		}
	}
	if old.RollbackQualifiedAt != nil && !reflect.DeepEqual(old.RollbackProbe.Proof, next.RollbackProbe.Proof) {
		return fmt.Errorf("rollback qualification proof is immutable after verification")
	}
	if old.RollbackQualifiedAt != nil && exclusiveStopStart(old) && !reflect.DeepEqual(old.OriginalProof, next.OriginalProof) {
		return fmt.Errorf("exclusive rollback qualification proof is immutable after verification")
	}
	if old.RollbackProbeQuiescent && !next.RollbackProbeQuiescent {
		return fmt.Errorf("rollback qualification quiescence cannot be revoked")
	}
	if !old.RollbackProbeQuiescent && next.RollbackProbeQuiescent && old.Phase != UpdateCommitting && old.Phase != UpdatePreserveCommitting && old.Phase != UpdateRestoreCommitting {
		return fmt.Errorf("rollback qualification cleanup outside a verified serving phase")
	}
	if len(next.Error) > 4096 {
		return fmt.Errorf("update error exceeds receipt bound")
	}
	if old.CandidateQuiescent && !next.CandidateQuiescent || old.PrimaryStopVerified && !next.PrimaryStopVerified || old.PrimaryQuiescent && !next.PrimaryQuiescent {
		return fmt.Errorf("quiescence evidence cannot be revoked to reuse an identity")
	}
	if !old.CandidateQuiescent && next.CandidateQuiescent && old.Phase != UpdatePreserveCommitting && old.Phase != UpdateRestoreCommitting && old.Phase != UpdateCommitting {
		return fmt.Errorf("candidate cleanup outside a verified preservation/promotion phase")
	}
	if !old.PrimaryStopVerified && next.PrimaryStopVerified && old.Phase != UpdatePrimaryStopping && old.Phase != UpdateRestoring {
		return fmt.Errorf("original stop outside a recorded promotion/restoration intent")
	}
	if !old.PrimaryQuiescent && next.PrimaryQuiescent && old.Phase != UpdateRestoring {
		return fmt.Errorf("failed primary cleanup outside restoration")
	}
	return nil
}
