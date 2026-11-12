package capacity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	ErrResidencyTransitionConflict     = errors.New("residency transition revision conflict")
	ErrResidencyTransitionOpen         = errors.New("claim already has an open residency transition")
	ErrResidencyResourceTransitionOpen = errors.New("resource/model has an unresolved residency transition")
	ErrResidencyModelMismatch          = errors.New("residency model identity does not match the transition")
	ErrResidencyActiveWork             = errors.New("residency transition is blocked while owner work is active or protected")
	ErrResidencyReleaseUnverified      = errors.New("residency release lacks verified unload evidence")
)

// ResidencyTransitionRecord is the durable replay view. Revision is an
// optimistic-concurrency token; claim identity is pinned to one captured
// capacity-claim generation for the full transition lifetime.
type ResidencyTransitionRecord struct {
	ClaimID         string              `json:"claim_id"`
	ClaimGeneration int64               `json:"claim_generation"`
	Revision        int64               `json:"revision"`
	Transition      ResidencyTransition `json:"transition"`
}

// ResidencyRepository persists owner transition reports separately from
// request demand. Implementations must preserve incomplete and unknown facts.
type ResidencyTransitionEvent struct {
	EventID    int64                     `json:"event_id"`
	EventKind  string                    `json:"event_kind"`
	RecordedAt time.Time                 `json:"recorded_at"`
	Record     ResidencyTransitionRecord `json:"record"`
}

type ResidencyRepository interface {
	BeginResidencyTransition(ctx context.Context, claimID string, claimGeneration int64, transition ResidencyTransition) (ResidencyTransitionRecord, error)
	CompleteResidencyTransition(ctx context.Context, claimID string, claimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error)
	ReconcileResidencyTransition(ctx context.Context, claimID string, claimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error)
	RecoverResidencyTransition(ctx context.Context, ownerID, claimID string, claimGeneration, currentClaimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error)
	ListResidencyTransitionEvents(ctx context.Context, transitionID string) ([]ResidencyTransitionEvent, error)
	GetResidencyTransition(ctx context.Context, transitionID string) (ResidencyTransitionRecord, error)
	GetLatestResidencyTransitionForClaim(ctx context.Context, claimID string) (ResidencyTransitionRecord, error)
	GetUnresolvedResidencyTransitionForOwner(ctx context.Context, ownerID string) (ResidencyTransitionRecord, error)
	ReleaseClaimAfterResidency(ctx context.Context, ownerID, claimID string, claimGeneration int64, transitionID string, expectedRevision int64, model ModelIdentity) (CapacityClaim, error)
}

// BeginResidencyTransition starts a durable transition under the exact claim
// generation that requested it. Open transitions are unique per claim, and a
// stale/restarted owner cannot begin work against a replaced claim generation.
func (s *SQLiteStore) BeginResidencyTransition(ctx context.Context, claimID string, claimGeneration int64, transition ResidencyTransition) (ResidencyTransitionRecord, error) {
	claimID = strings.TrimSpace(claimID)
	if claimID == "" || claimGeneration <= 0 {
		return ResidencyTransitionRecord{}, fmt.Errorf("%w: claim identity and positive generation are required", ErrInvalidClaim)
	}
	if err := validateResidencyStart(transition); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	record := ResidencyTransitionRecord{ClaimID: claimID, ClaimGeneration: claimGeneration, Revision: 1, Transition: transition}
	payload, err := json.Marshal(record)
	if err != nil {
		return ResidencyTransitionRecord{}, fmt.Errorf("encode residency transition: %w", err)
	}
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE capacity_claims SET claim_id=claim_id WHERE claim_id=?`, claimID); err != nil {
			return fmt.Errorf("acquire residency claim write authority: %w", err)
		}
		claim, err := getClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if claim.Generation != claimGeneration || !IsActiveClaimStatus(claim.Status) {
			return ErrStaleGeneration
		}
		if err := validateClaimResidencyModel(claim, transition.Model); err != nil {
			return err
		}
		if claim.ActivityState == ActivityActive || claim.Protected {
			return ErrResidencyActiveWork
		}
		if err := rejectUnresolvedResourceTransition(ctx, tx, claim, transition, s.now()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO capacity_residency_transitions
(transition_id, claim_id, claim_generation, revision, state, transition_json, started_at, completed_at)
VALUES (?, ?, ?, 1, 'started', ?, ?, NULL)`, transition.ID, claimID, claimGeneration, string(payload), formatTime(transition.StartedAt))
		if err != nil {
			if isResidencyOpenUniqueConflict(err) {
				return ErrResidencyTransitionOpen
			}
			return fmt.Errorf("persist residency transition start: %w", err)
		}
		return appendResidencyEvent(ctx, tx, "started", record, s.now())
	})
	if err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

// ReconcileResidencyTransition refreshes only the observed result for a
// completed transition whose release sample aged out. Immutable request/model
// facts remain identical; a new verified sample is required before release.
func (s *SQLiteStore) ReconcileResidencyTransition(ctx context.Context, claimID string, claimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error) {
	claimID = strings.TrimSpace(claimID)
	if claimID == "" || claimGeneration <= 0 || expectedRevision <= 0 {
		return ResidencyTransitionRecord{}, fmt.Errorf("claim identity, generation, and positive expected revision are required")
	}
	if transition.CompletedAt == nil || !transition.CompletedAt.After(transition.StartedAt) {
		return ResidencyTransitionRecord{}, fmt.Errorf("residency reconciliation requires a fresh completion time")
	}
	if err := validateResidencyCompletion(transition); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	var record ResidencyTransitionRecord
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var oldPayload, state, startedAt string
		var completedAt sql.NullString
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT transition_json, revision, state, started_at, completed_at FROM capacity_residency_transitions WHERE transition_id=? AND claim_id=? AND claim_generation=?`, transition.ID, claimID, claimGeneration).Scan(&oldPayload, &revision, &state, &startedAt, &completedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("read residency transition before reconciliation: %w", err)
		}
		if revision != expectedRevision || state != "completed" {
			return ErrResidencyTransitionConflict
		}
		var old ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(oldPayload), &old); err != nil {
			return fmt.Errorf("decode persisted residency transition: %w", err)
		}
		if old.ClaimID != claimID || old.ClaimGeneration != claimGeneration || old.Revision != revision || old.Transition.ID != transition.ID {
			return fmt.Errorf("persisted residency transition row and payload identity disagree")
		}
		if err := validateResidencyRowTimes(old.Transition, state, startedAt, completedAt); err != nil {
			return err
		}
		if !sameResidencyStart(old.Transition, transition) || (old.Transition.CompletedAt != nil && old.Transition.CompletedAt.After(*transition.CompletedAt)) {
			return fmt.Errorf("residency reconciliation changed immutable transition start facts or moved completion backward")
		}
		claim, err := getClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if err := validateClaimResidencyModel(claim, transition.Model); err != nil {
			return err
		}
		if claim.Generation != claimGeneration || !IsActiveClaimStatus(claim.Status) {
			return ErrStaleGeneration
		}
		record = ResidencyTransitionRecord{ClaimID: claimID, ClaimGeneration: claimGeneration, Revision: revision + 1, Transition: transition}
		payload, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("encode reconciled residency transition: %w", err)
		}
		result, err := tx.ExecContext(ctx, `UPDATE capacity_residency_transitions SET revision=?, transition_json=?, completed_at=? WHERE transition_id=? AND claim_id=? AND claim_generation=? AND revision=? AND state='completed'`, record.Revision, string(payload), formatTime(*transition.CompletedAt), transition.ID, claimID, claimGeneration, expectedRevision)
		if err != nil {
			return fmt.Errorf("persist reconciled residency transition: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrResidencyTransitionConflict
		}
		return appendResidencyEvent(ctx, tx, "reconciled", record, s.now())
	})
	if err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

func rejectUnresolvedResourceTransition(ctx context.Context, tx *sql.Tx, claim CapacityClaim, candidate ResidencyTransition, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT t.transition_id, t.claim_id, t.claim_generation, t.revision, t.state, t.transition_json, t.started_at, t.completed_at, c.status FROM capacity_residency_transitions t JOIN capacity_claims c ON c.claim_id=t.claim_id WHERE c.owner_kind=? AND c.owner_id=? AND c.claim_id<>?`, claim.OwnerKind, claim.OwnerID, claim.ClaimID)
	if err != nil {
		return fmt.Errorf("read resource residency transition history: %w", err)
	}
	defer rows.Close()
	var latest *ResidencyTransitionRecord
	var latestState, latestClaimStatus string
	var ambiguous bool
	for rows.Next() {
		var transitionID, claimID, state, payload, startedAt, claimStatus string
		var claimGeneration, revision int64
		var completedAt sql.NullString
		if err := rows.Scan(&transitionID, &claimID, &claimGeneration, &revision, &state, &payload, &startedAt, &completedAt, &claimStatus); err != nil {
			return err
		}
		var current ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(payload), &current); err != nil {
			return fmt.Errorf("decode resource residency transition history: %w", err)
		}
		if current.Transition.ID != transitionID || current.ClaimID != claimID || current.ClaimGeneration != claimGeneration || current.Revision != revision {
			return fmt.Errorf("persisted resource residency transition row and payload identity disagree")
		}
		if err := validateResidencyRowTimes(current.Transition, state, startedAt, completedAt); err != nil {
			return err
		}
		if latest == nil || current.Transition.StartedAt.After(latest.Transition.StartedAt) {
			copy := current
			latest = &copy
			latestState = state
			latestClaimStatus = claimStatus
			ambiguous = false
		} else if current.Transition.StartedAt.Equal(latest.Transition.StartedAt) && current.Transition.ID != latest.Transition.ID {
			ambiguous = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if latest == nil {
		return nil
	}
	if ambiguous || latestState == "started" || !latest.Transition.hasFreshVerifiedObservation(now) {
		return ErrResidencyResourceTransitionOpen
	}
	if latest.Transition.Desired.Backend == ResidencyBackendUnloaded && latest.Transition.Observed.Known && latest.Transition.Observed.Backend == ResidencyBackendUnloaded {
		if IsActiveClaimStatus(latestClaimStatus) || !latest.Transition.verifiedUnloadAt(now) {
			return ErrResidencyResourceTransitionOpen
		}
	} else if !latest.Transition.Observed.Known {
		return ErrResidencyResourceTransitionOpen
	}
	return nil
}

// CompleteResidencyTransition records the owner's result with a revision CAS.
// Identity, desired placement, reason and pre-transition evidence are immutable
// across the start/completion boundary; the observation can remain unknown.
func (s *SQLiteStore) CompleteResidencyTransition(ctx context.Context, claimID string, claimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error) {
	claimID = strings.TrimSpace(claimID)
	if claimID == "" || claimGeneration <= 0 || expectedRevision <= 0 {
		return ResidencyTransitionRecord{}, fmt.Errorf("claim identity, generation, and positive expected revision are required")
	}
	if transition.ID == "" || transition.CompletedAt == nil || !transition.CompletedAt.After(transition.StartedAt) {
		return ResidencyTransitionRecord{}, fmt.Errorf("residency completion requires an ID and completion time after start")
	}
	var record ResidencyTransitionRecord
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var oldPayload, startedAt string
		var completedAt sql.NullString
		var revision int64
		var state string
		err := tx.QueryRowContext(ctx, `SELECT transition_json, revision, state, started_at, completed_at FROM capacity_residency_transitions WHERE transition_id=? AND claim_id=? AND claim_generation=?`, transition.ID, claimID, claimGeneration).Scan(&oldPayload, &revision, &state, &startedAt, &completedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read residency transition before completion: %w", err)
		}
		if revision != expectedRevision || state != "started" {
			return ErrResidencyTransitionConflict
		}
		var old ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(oldPayload), &old); err != nil {
			return fmt.Errorf("decode persisted residency transition: %w", err)
		}
		if old.ClaimID != claimID || old.ClaimGeneration != claimGeneration || old.Revision != revision || old.Transition.ID != transition.ID {
			return fmt.Errorf("persisted residency transition row and payload identity disagree")
		}
		if err := validateResidencyRowTimes(old.Transition, state, startedAt, completedAt); err != nil {
			return err
		}
		if !sameResidencyStart(old.Transition, transition) {
			return fmt.Errorf("residency completion changed immutable transition start facts")
		}
		if err := validateResidencyCompletion(transition); err != nil {
			return err
		}
		claim, err := getClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if err := validateClaimResidencyModel(claim, transition.Model); err != nil {
			return err
		}
		if claim.Generation != claimGeneration || !IsActiveClaimStatus(claim.Status) {
			return ErrStaleGeneration
		}
		record = ResidencyTransitionRecord{ClaimID: claimID, ClaimGeneration: claimGeneration, Revision: revision + 1, Transition: transition}
		payload, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("encode completed residency transition: %w", err)
		}
		result, err := tx.ExecContext(ctx, `UPDATE capacity_residency_transitions
SET revision=?, state='completed', transition_json=?, completed_at=?
WHERE transition_id=? AND claim_id=? AND claim_generation=? AND revision=? AND state='started'`,
			record.Revision, string(payload), formatTime(*transition.CompletedAt), transition.ID, claimID, claimGeneration, expectedRevision)
		if err != nil {
			return fmt.Errorf("persist residency transition completion: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrResidencyTransitionConflict
		}
		return appendResidencyEvent(ctx, tx, "completed", record, s.now())
	})
	if err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

// GetResidencyTransition is the restart-safe replay/read path for one owner
// transition identity.
func (s *SQLiteStore) GetResidencyTransition(ctx context.Context, transitionID string) (ResidencyTransitionRecord, error) {
	transitionID = strings.TrimSpace(transitionID)
	var rowTransitionID, claimID, state, payload, startedAt string
	var completedAt sql.NullString
	var claimGeneration, revision int64
	if err := s.db.QueryRowContext(ctx, `SELECT transition_id, claim_id, claim_generation, revision, state, transition_json, started_at, completed_at
FROM capacity_residency_transitions WHERE transition_id=?`, transitionID).Scan(&rowTransitionID, &claimID, &claimGeneration, &revision, &state, &payload, &startedAt, &completedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResidencyTransitionRecord{}, ErrNotFound
		}
		return ResidencyTransitionRecord{}, fmt.Errorf("read residency transition: %w", err)
	}
	var record ResidencyTransitionRecord
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return ResidencyTransitionRecord{}, fmt.Errorf("decode residency transition replay: %w", err)
	}
	if record.Transition.ID != rowTransitionID || record.ClaimID != claimID || record.ClaimGeneration != claimGeneration || record.Revision != revision {
		return ResidencyTransitionRecord{}, fmt.Errorf("persisted residency transition row and replay payload identity disagree")
	}
	if err := validateResidencyRowTimes(record.Transition, state, startedAt, completedAt); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

// GetLatestResidencyTransitionForClaim returns the exact durable owner record
// used to verify the result of an unload callback.
func (s *SQLiteStore) GetLatestResidencyTransitionForClaim(ctx context.Context, claimID string) (ResidencyTransitionRecord, error) {
	claimID = strings.TrimSpace(claimID)
	rows, err := s.db.QueryContext(ctx, `SELECT transition_id, claim_id, claim_generation, revision, state, transition_json, started_at, completed_at
FROM capacity_residency_transitions WHERE claim_id=? ORDER BY started_at DESC, transition_id DESC`, claimID)
	if err != nil {
		return ResidencyTransitionRecord{}, fmt.Errorf("find latest claim residency transition: %w", err)
	}
	defer rows.Close()
	var latest *ResidencyTransitionRecord
	for rows.Next() {
		var transitionID, rowClaimID, state, payload, startedAt string
		var claimGeneration, revision int64
		var completedAt sql.NullString
		if err := rows.Scan(&transitionID, &rowClaimID, &claimGeneration, &revision, &state, &payload, &startedAt, &completedAt); err != nil {
			return ResidencyTransitionRecord{}, err
		}
		record, err := decodeResidencyRow(transitionID, rowClaimID, claimGeneration, revision, state, payload, startedAt, completedAt)
		if err != nil {
			return ResidencyTransitionRecord{}, err
		}
		if latest == nil {
			latest = &record
		} else if latest.Transition.StartedAt.Equal(record.Transition.StartedAt) && latest.Transition.ID != record.Transition.ID {
			return ResidencyTransitionRecord{}, fmt.Errorf("claim has ambiguous residency transitions with equal start times")
		}
	}
	if err := rows.Err(); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	if latest == nil {
		return ResidencyTransitionRecord{}, ErrNotFound
	}
	return *latest, nil
}

// GetUnresolvedResidencyTransitionForOwner finds the latest transition that
// still fences an exact resource owner, including completed records whose
// measured evidence has aged out. Recovery never guesses between same-time
// records or replays a terminal record that no longer owns the device.
func (s *SQLiteStore) GetUnresolvedResidencyTransitionForOwner(ctx context.Context, ownerID string) (ResidencyTransitionRecord, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return ResidencyTransitionRecord{}, fmt.Errorf("residency recovery requires a resource owner")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.transition_id, t.claim_id, t.claim_generation, t.revision, t.state, t.transition_json, t.started_at, t.completed_at, c.status
FROM capacity_residency_transitions t
JOIN capacity_claims c ON c.claim_id=t.claim_id
WHERE c.owner_kind=? AND c.owner_id=?
ORDER BY t.started_at DESC, t.transition_id DESC`, OwnerKindResource, ownerID)
	if err != nil {
		return ResidencyTransitionRecord{}, fmt.Errorf("find unresolved resource residency transition: %w", err)
	}
	defer rows.Close()
	var latest *ResidencyTransitionRecord
	var latestState, latestClaimStatus string
	var ambiguous bool
	for rows.Next() {
		var transitionID, claimID, state, payload, startedAt, claimStatus string
		var completedAt sql.NullString
		var claimGeneration, revision int64
		if err := rows.Scan(&transitionID, &claimID, &claimGeneration, &revision, &state, &payload, &startedAt, &completedAt, &claimStatus); err != nil {
			return ResidencyTransitionRecord{}, err
		}
		if state != "started" && state != "completed" {
			return ResidencyTransitionRecord{}, fmt.Errorf("unexpected open residency state %q", state)
		}
		var current ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(payload), &current); err != nil {
			return ResidencyTransitionRecord{}, fmt.Errorf("decode unresolved residency transition: %w", err)
		}
		if current.Transition.ID != transitionID || current.ClaimID != claimID || current.ClaimGeneration != claimGeneration || current.Revision != revision {
			return ResidencyTransitionRecord{}, fmt.Errorf("persisted residency transition row and payload identity disagree")
		}
		if err := validateResidencyRowTimes(current.Transition, state, startedAt, completedAt); err != nil {
			return ResidencyTransitionRecord{}, err
		}
		if latest == nil || current.Transition.StartedAt.After(latest.Transition.StartedAt) {
			copy := current
			latest = &copy
			latestState, latestClaimStatus, ambiguous = state, claimStatus, false
		} else if current.Transition.StartedAt.Equal(latest.Transition.StartedAt) && current.Transition.ID != latest.Transition.ID {
			ambiguous = true
		}
	}
	if err := rows.Err(); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	if latest == nil {
		return ResidencyTransitionRecord{}, ErrNotFound
	}
	if ambiguous || latestState == "started" || !latest.Transition.hasFreshVerifiedObservation(s.now()) || !latest.Transition.Observed.Known {
		return *latest, nil
	}
	if latest.Transition.Desired.Backend == ResidencyBackendUnloaded && latest.Transition.Observed.Backend == ResidencyBackendUnloaded && (IsActiveClaimStatus(latestClaimStatus) || !latest.Transition.verifiedUnloadAt(s.now())) {
		return *latest, nil
	}
	return ResidencyTransitionRecord{}, ErrNotFound
}

// RecoverResidencyTransition reconciles a started transition after its owner
// process or control-plane caller restarts. The request is fenced by the exact
// resource owner, claim generation, transition revision, and immutable model
// identity. Unknown placement is durably recorded but remains state=started.
func (s *SQLiteStore) RecoverResidencyTransition(ctx context.Context, ownerID, claimID string, claimGeneration, currentClaimGeneration, expectedRevision int64, transition ResidencyTransition) (ResidencyTransitionRecord, error) {
	ownerID, claimID = strings.TrimSpace(ownerID), strings.TrimSpace(claimID)
	if ownerID == "" || claimID == "" || claimGeneration <= 0 || currentClaimGeneration <= 0 || expectedRevision <= 0 {
		return ResidencyTransitionRecord{}, fmt.Errorf("resource owner, transition claim identity, current claim generation, and positive revision are required")
	}
	if err := validateResidencyRecovery(transition, s.now()); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	var record ResidencyTransitionRecord
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var oldPayload, storedClaimID, startedAt string
		var completedAt sql.NullString
		var revision, storedGeneration int64
		var state string
		err := tx.QueryRowContext(ctx, `SELECT transition_json, claim_id, claim_generation, revision, state, started_at, completed_at FROM capacity_residency_transitions WHERE transition_id=?`, transition.ID).Scan(&oldPayload, &storedClaimID, &storedGeneration, &revision, &state, &startedAt, &completedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read residency transition before recovery: %w", err)
		}
		if storedClaimID != claimID {
			return fmt.Errorf("recovery claim does not own the exact transition")
		}
		if storedGeneration != claimGeneration {
			return ErrStaleGeneration
		}
		if revision != expectedRevision || state != "started" {
			return ErrResidencyTransitionConflict
		}
		var old ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(oldPayload), &old); err != nil {
			return fmt.Errorf("decode persisted residency transition: %w", err)
		}
		if old.ClaimID != claimID || old.ClaimGeneration != claimGeneration || old.Revision != revision || old.Transition.ID != transition.ID || !sameResidencyStart(old.Transition, transition) {
			return fmt.Errorf("%w: recovery changed transition, claim, resource, or model identity", ErrResidencyModelMismatch)
		}
		if err := validateResidencyRowTimes(old.Transition, state, startedAt, completedAt); err != nil {
			return err
		}
		claim, err := getClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if claim.OwnerKind != OwnerKindResource || claim.OwnerID != ownerID {
			return fmt.Errorf("recovery resource owner does not match the claim")
		}
		if err := validateClaimResidencyModel(claim, transition.Model); err != nil {
			return err
		}
		if claim.Generation != currentClaimGeneration {
			return ErrStaleGeneration
		}
		activeRecovery := IsActiveClaimStatus(claim.Status) && currentClaimGeneration >= claimGeneration
		terminalRecovery := !IsActiveClaimStatus(claim.Status) && currentClaimGeneration == claimGeneration+1 && (claim.Status == StatusReleased || claim.Status == StatusExpired || claim.Status == StatusPreempted)
		if !activeRecovery && !terminalRecovery {
			return ErrStaleGeneration
		}
		state = "started"
		kind := "recovery_unknown"
		if transition.Observed.Known {
			if transition.CompletedAt == nil || !transition.CompletedAt.After(transition.StartedAt) {
				return fmt.Errorf("known recovery requires a completion time after the original start")
			}
			if terminalRecovery {
				if transition.Desired.Backend == ResidencyBackendUnloaded && transition.verifiedUnloadAt(s.now()) {
					state, kind = "completed", "recovered"
				} else {
					kind = "recovery_terminal_observed"
				}
			} else {
				state, kind = "completed", "recovered"
			}
		}
		record = ResidencyTransitionRecord{ClaimID: claimID, ClaimGeneration: claimGeneration, Revision: revision + 1, Transition: transition}
		payload, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("encode recovered residency transition: %w", err)
		}
		var result sql.Result
		if state == "completed" {
			result, err = tx.ExecContext(ctx, `UPDATE capacity_residency_transitions SET revision=?, state='completed', transition_json=?, completed_at=? WHERE transition_id=? AND claim_id=? AND claim_generation=? AND revision=? AND state='started'`, record.Revision, string(payload), formatTime(*transition.CompletedAt), transition.ID, claimID, claimGeneration, expectedRevision)
		} else {
			result, err = tx.ExecContext(ctx, `UPDATE capacity_residency_transitions SET revision=?, transition_json=? WHERE transition_id=? AND claim_id=? AND claim_generation=? AND revision=? AND state='started'`, record.Revision, string(payload), transition.ID, claimID, claimGeneration, expectedRevision)
		}
		if err != nil {
			return fmt.Errorf("persist residency recovery: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return ErrResidencyTransitionConflict
		}
		return appendResidencyEvent(ctx, tx, kind, record, s.now())
	})
	if err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

// ListResidencyTransitionEvents returns the append-only event snapshots for
// one transition in commit order, including recovery observations.
func (s *SQLiteStore) ListResidencyTransitionEvents(ctx context.Context, transitionID string) ([]ResidencyTransitionEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id, transition_id, claim_id, claim_generation, revision, event_kind, recorded_at, transition_json FROM capacity_residency_transition_events WHERE transition_id=? ORDER BY event_id`, strings.TrimSpace(transitionID))
	if err != nil {
		return nil, fmt.Errorf("read residency transition audit history: %w", err)
	}
	defer rows.Close()
	var events []ResidencyTransitionEvent
	for rows.Next() {
		var event ResidencyTransitionEvent
		var transitionID, claimID, recorded, payload string
		var claimGeneration, revision int64
		if err := rows.Scan(&event.EventID, &transitionID, &claimID, &claimGeneration, &revision, &event.EventKind, &recorded, &payload); err != nil {
			return nil, err
		}
		record, err := decodeResidencyRecord(payload)
		if err != nil {
			return nil, err
		}
		if record.Transition.ID != transitionID || record.ClaimID != claimID || record.ClaimGeneration != claimGeneration || record.Revision != revision {
			return nil, fmt.Errorf("residency audit event row and snapshot identity disagree")
		}
		event.RecordedAt, err = time.Parse(time.RFC3339Nano, recorded)
		if err != nil {
			return nil, fmt.Errorf("decode residency event timestamp: %w", err)
		}
		event.Record = record
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func appendResidencyEvent(ctx context.Context, tx *sql.Tx, kind string, record ResidencyTransitionRecord, at time.Time) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode residency audit snapshot: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO capacity_residency_transition_events (transition_id, claim_id, claim_generation, revision, event_kind, transition_json, recorded_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, record.Transition.ID, record.ClaimID, record.ClaimGeneration, record.Revision, kind, string(payload), formatTime(at))
	if err != nil {
		return fmt.Errorf("append residency transition audit event: %w", err)
	}
	return nil
}

func decodeResidencyRecord(payload string) (ResidencyTransitionRecord, error) {
	var record ResidencyTransitionRecord
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return record, fmt.Errorf("decode residency transition audit snapshot: %w", err)
	}
	return record, nil
}

// validateResidencyRowTimes keeps SQLite's indexed/queryable state in lockstep
// with the durable replay payload. Both representations participate in
// capacity release and recovery authority, so disagreement must fail closed.
func validateResidencyRowTimes(transition ResidencyTransition, state, startedAt string, completedAt sql.NullString) error {
	if startedAt != formatTime(transition.StartedAt) {
		return fmt.Errorf("persisted residency transition row and replay start time disagree")
	}
	switch state {
	case "started":
		if completedAt.Valid || transition.CompletedAt != nil {
			return fmt.Errorf("persisted started residency row and replay completion state disagree")
		}
	case "completed":
		if !completedAt.Valid || transition.CompletedAt == nil {
			return fmt.Errorf("persisted completed residency row and replay completion state disagree")
		}
		if completedAt.String != formatTime(*transition.CompletedAt) {
			return fmt.Errorf("persisted residency transition row and replay completion time disagree")
		}
	default:
		return fmt.Errorf("persisted residency transition row has unexpected state %q", state)
	}
	return nil
}

func decodeResidencyRow(transitionID, claimID string, claimGeneration, revision int64, state, payload, startedAt string, completedAt sql.NullString) (ResidencyTransitionRecord, error) {
	record, err := decodeResidencyRecord(payload)
	if err != nil {
		return ResidencyTransitionRecord{}, err
	}
	if record.Transition.ID != transitionID || record.ClaimID != claimID || record.ClaimGeneration != claimGeneration || record.Revision != revision {
		return ResidencyTransitionRecord{}, fmt.Errorf("persisted residency transition row and replay payload identity disagree")
	}
	if err := validateResidencyRowTimes(record.Transition, state, startedAt, completedAt); err != nil {
		return ResidencyTransitionRecord{}, err
	}
	return record, nil
}

func validateResidencyRecovery(transition ResidencyTransition, now time.Time) error {
	if transition.ID == "" || !transition.Model.Complete() || transition.Reason == "" || transition.StartedAt.IsZero() {
		return fmt.Errorf("residency recovery requires original transition and model identity")
	}
	if transition.Footprint.Source != FootprintSourceMeasured || !transition.Footprint.Verified || transition.Footprint.ObservedAt.IsZero() || transition.Footprint.ObservedAt.After(now) || now.Sub(transition.Footprint.ObservedAt) > maxResidencyFootprintAge {
		return fmt.Errorf("residency recovery requires a fresh measured physical footprint")
	}
	if transition.Footprint.Bytes < 0 {
		return fmt.Errorf("residency recovery footprint cannot be negative")
	}
	if !transition.Footprint.ObservedAt.After(transition.StartedAt) {
		return fmt.Errorf("residency recovery footprint must be measured after the original transition began")
	}
	if transition.CompletedAt != nil && transition.Footprint.ObservedAt.After(*transition.CompletedAt) {
		return fmt.Errorf("residency recovery footprint cannot follow its completion time")
	}
	if !transition.Observed.Known {
		if transition.CompletedAt != nil || transition.EffectsVerified {
			return fmt.Errorf("unknown recovery must remain incomplete and unverified")
		}
		return nil
	}
	if !validResidencyBackend(transition.Observed.Backend) || transition.CompletedAt == nil || !transition.CompletedAt.After(transition.StartedAt) || !transition.EffectsVerified {
		return fmt.Errorf("known recovery requires a valid backend, completion time, and verified owner observation")
	}
	return nil
}

// hasUnresolvedResidencyTransitionTx prevents generic claim mutations from
// invalidating an owner transition's generation fence or freeing capacity while
// its physical result is unknown or stale.
func hasUnresolvedResidencyTransitionTx(ctx context.Context, tx *sql.Tx, claimID string, now time.Time) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT t.transition_id, t.claim_id, t.claim_generation, t.revision, t.state, t.transition_json, t.started_at, t.completed_at, c.status FROM capacity_residency_transitions t JOIN capacity_claims c ON c.claim_id=t.claim_id WHERE t.claim_id=? ORDER BY t.started_at DESC, t.transition_id DESC`, claimID)
	if err != nil {
		return false, fmt.Errorf("read claim residency transition fence: %w", err)
	}
	defer rows.Close()
	var latest *ResidencyTransitionRecord
	var latestState, latestClaimStatus string
	var ambiguous bool
	for rows.Next() {
		var transitionID, rowClaimID, state, payload, startedAt, claimStatus string
		var claimGeneration, revision int64
		var completedAt sql.NullString
		if err := rows.Scan(&transitionID, &rowClaimID, &claimGeneration, &revision, &state, &payload, &startedAt, &completedAt, &claimStatus); err != nil {
			return false, err
		}
		record, err := decodeResidencyRow(transitionID, rowClaimID, claimGeneration, revision, state, payload, startedAt, completedAt)
		if err != nil {
			return false, err
		}
		if latest == nil || record.Transition.StartedAt.After(latest.Transition.StartedAt) {
			latest = &record
			latestState, latestClaimStatus, ambiguous = state, claimStatus, false
		} else if record.Transition.StartedAt.Equal(latest.Transition.StartedAt) && record.Transition.ID != latest.Transition.ID {
			ambiguous = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if latest == nil {
		return false, nil
	}
	if ambiguous {
		return true, fmt.Errorf("claim has ambiguous residency transitions with equal start times")
	}
	if latestState == "started" {
		return true, nil
	}
	if !latest.Transition.hasFreshVerifiedObservation(now) {
		return true, nil
	}
	if latest.Transition.Desired.Backend == ResidencyBackendUnloaded && latest.Transition.Observed.Known && latest.Transition.Observed.Backend == ResidencyBackendUnloaded {
		if IsActiveClaimStatus(latestClaimStatus) || !latest.Transition.verifiedUnloadAt(now) {
			return true, nil
		}
	}
	return false, nil
}

// ReleaseClaimAfterResidency is the proof-gated capacity path. The claim is
// released only when the exact transition revision has a fresh, verified
// measured unload and the original claim generation is still current.
func (s *SQLiteStore) ReleaseClaimAfterResidency(ctx context.Context, ownerID, claimID string, claimGeneration int64, transitionID string, expectedRevision int64, model ModelIdentity) (CapacityClaim, error) {
	if strings.TrimSpace(ownerID) == "" || strings.TrimSpace(claimID) == "" || claimGeneration <= 0 || strings.TrimSpace(transitionID) == "" || expectedRevision <= 0 || !model.Complete() {
		return CapacityClaim{}, fmt.Errorf("resource owner, claim identity, generation, transition ID, positive revision, and complete model identity are required")
	}
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var revision int64
		var state, payload, startedAt string
		var completedAt sql.NullString
		var storedClaimID string
		var storedGeneration int64
		if err := tx.QueryRowContext(ctx, `SELECT claim_id, claim_generation, revision, state, transition_json, started_at, completed_at
FROM capacity_residency_transitions WHERE transition_id=?`, transitionID).Scan(&storedClaimID, &storedGeneration, &revision, &state, &payload, &startedAt, &completedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("read residency release proof: %w", err)
		}
		if storedClaimID != claimID || storedGeneration != claimGeneration || revision != expectedRevision || state != "completed" {
			return fmt.Errorf("%w: release expected claim=%s generation=%d revision=%d completed; found claim=%s generation=%d revision=%d state=%s", ErrResidencyTransitionConflict, claimID, claimGeneration, expectedRevision, storedClaimID, storedGeneration, revision, state)
		}
		var record ResidencyTransitionRecord
		if err := json.Unmarshal([]byte(payload), &record); err != nil {
			return fmt.Errorf("decode residency release proof: %w", err)
		}
		if record.Transition.ID != transitionID || record.ClaimID != claimID || record.ClaimGeneration != claimGeneration || record.Revision != revision {
			return fmt.Errorf("persisted residency release row and payload identity disagree")
		}
		if err := validateResidencyRowTimes(record.Transition, state, startedAt, completedAt); err != nil {
			return err
		}
		if record.Transition.Model != model {
			return fmt.Errorf("%w: release request does not match the transition artifact", ErrResidencyModelMismatch)
		}
		if !record.Transition.verifiedUnloadAt(s.now()) {
			return ErrResidencyReleaseUnverified
		}
		claim, err := getClaimTx(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if claim.OwnerKind != OwnerKindResource || claim.OwnerID != ownerID {
			return fmt.Errorf("residency release resource owner does not match the claim")
		}
		if err := validateClaimResidencyModel(claim, model); err != nil {
			return err
		}
		if claim.Generation != claimGeneration || !IsActiveClaimStatus(claim.Status) {
			return ErrStaleGeneration
		}
		result, err := tx.ExecContext(ctx, `UPDATE capacity_claims SET status=?, generation=generation+1, updated_at=?
WHERE claim_id=? AND generation=? AND status IN (?, ?, ?)`, StatusReleased, formatTime(s.now()), claimID, claimGeneration, StatusReserved, StatusGranted, StatusDegraded)
		if err != nil {
			return fmt.Errorf("release capacity claim after verified residency transition: %w", err)
		}
		if err := finishMutation(ctx, tx, result, claimID, &out); err != nil {
			return err
		}
		return appendResidencyEvent(ctx, tx, "claim_released", record, s.now())
	})
	return out, err
}

func validateResidencyStart(transition ResidencyTransition) error {
	if transition.ID == "" || !transition.Model.Complete() || transition.Reason == "" || transition.StartedAt.IsZero() {
		return fmt.Errorf("residency transition requires ID, complete model identity, reason, and start time")
	}
	if transition.CompletedAt != nil {
		return fmt.Errorf("new residency transition cannot already be completed")
	}
	if !transition.Desired.Known || !validResidencyBackend(transition.Desired.Backend) {
		return fmt.Errorf("residency transition requires a known, supported desired backend")
	}
	if transition.PreFootprint.Bytes < 0 || transition.Footprint.Bytes < 0 {
		return fmt.Errorf("residency footprint bytes cannot be negative")
	}
	return nil
}

func validateClaimResidencyModel(claim CapacityClaim, model ModelIdentity) error {
	if !model.Complete() || claim.DegradeProfile == nil || !claim.DegradeProfile.Model.Complete() || claim.DegradeProfile.Model != model {
		return fmt.Errorf("%w: transition model must match the complete model identity bound to the resource claim", ErrResidencyModelMismatch)
	}
	return nil
}

func validateResidencyCompletion(transition ResidencyTransition) error {
	if transition.CompletedAt == nil || !transition.CompletedAt.After(transition.StartedAt) {
		return fmt.Errorf("residency completion time must follow start time")
	}
	if transition.Observed.Known && !validResidencyBackend(transition.Observed.Backend) {
		return fmt.Errorf("residency observation has an unsupported known backend")
	}
	if transition.Footprint.Bytes < 0 {
		return fmt.Errorf("residency footprint bytes cannot be negative")
	}
	return nil
}

func sameResidencyStart(a, b ResidencyTransition) bool {
	a.CompletedAt, b.CompletedAt = nil, nil
	a.Observed, b.Observed = ResidencyPlacement{}, ResidencyPlacement{}
	a.Footprint, b.Footprint = ResidencyFootprint{}, ResidencyFootprint{}
	a.EffectsVerified, b.EffectsVerified = false, false
	return reflect.DeepEqual(a, b)
}

func isResidencyOpenUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_capacity_residency_one_open_per_claim") || strings.Contains(message, "unique constraint failed: capacity_residency_transitions.claim_id")
}

var _ ResidencyRepository = (*SQLiteStore)(nil)
