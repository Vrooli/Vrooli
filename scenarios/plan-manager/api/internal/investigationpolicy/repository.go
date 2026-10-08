package investigationpolicy

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Incident struct {
	ID                  string       `json:"incidentId"`
	ExecutionID         string       `json:"executionId"`
	FamilyID            string       `json:"familyId,omitempty"`
	SubjectExecutionIDs []string     `json:"subjectExecutionIds,omitempty"`
	PhaseID             string       `json:"phaseId"`
	PhaseGeneration     string       `json:"phaseGeneration"`
	PolicyVersion       string       `json:"policyVersion"`
	IncidentFingerprint string       `json:"incidentFingerprint"`
	Mode                Mode         `json:"mode"`
	State               string       `json:"state"`
	Decision            Decision     `json:"decision"`
	InvestigationID     string       `json:"investigationId,omitempty"`
	ProgramID           string       `json:"programId,omitempty"`
	ProgramStatus       string       `json:"programStatus,omitempty"`
	DispatchError       string       `json:"dispatchError,omitempty"`
	DispatchClaimKey    string       `json:"-"`
	DispatchStartedAt   string       `json:"-"`
	Occurrences         []Occurrence `json:"occurrences,omitempty"`
	OccurrenceCount     int          `json:"occurrenceCount"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}

type Repository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// dispatchClaimLease bounds the period for which a dispatching claim may be
// considered owned by a live Plan Manager process. The dispatch call itself
// is synchronous and the nested program has its own finite wait budget; a
// much longer lease prevents a slow but healthy caller from being stolen,
// while still allowing a restarted owner to recover a claim left by a crash.
const dispatchClaimLease = 10 * time.Minute

var (
	ErrIncidentNotFound = errors.New("plan investigation incident not found")
	ErrPolicyNotFound   = errors.New("plan investigation policy not found")
	ErrPolicyConflict   = errors.New("plan investigation policy revision conflict")
)

func NewRepository(db SQLExecutor, clock schedule.Clock) *Repository {
	if clock == nil {
		clock = schedule.System()
	}
	return &Repository{db: db, clock: clock}
}

func (r *Repository) PutPolicy(ctx context.Context, policy Policy, active bool) error {
	return r.PutPolicyIfVersion(ctx, policy, active, "")
}

func (r *Repository) PutPolicyIfVersion(ctx context.Context, policy Policy, active bool, expectedVersion string) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	if policy.Scope.Key() != "global" {
		return r.putOverride(ctx, policy, active, expectedVersion, string(raw), now)
	}
	if expectedVersion = strings.TrimSpace(expectedVersion); expectedVersion != "" {
		var currentVersion string
		err := r.db.QueryRowContext(ctx, `SELECT policy_version FROM plan_investigation_policies WHERE active=1 LIMIT 1`).Scan(&currentVersion)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: expected active version %q, current version is unset", ErrPolicyConflict, expectedVersion)
			}
			return err
		}
		if currentVersion != expectedVersion {
			return fmt.Errorf("%w: expected active version %q, current version %q", ErrPolicyConflict, expectedVersion, currentVersion)
		}
	}
	var existing string
	if err := r.db.QueryRowContext(ctx, `SELECT document FROM plan_investigation_policies WHERE policy_version=?`, policy.Version).Scan(&existing); err == nil && existing != string(raw) {
		return fmt.Errorf("%w: policy version %q is immutable", ErrPolicyConflict, policy.Version)
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if active {
		if _, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_policies SET active=0,updated_at=? WHERE active=1`, now); err != nil {
			return err
		}
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO plan_investigation_policies (policy_id,policy_version,document,active,created_at,updated_at) VALUES (?,?,?,?,?,?) ON CONFLICT(policy_version) DO UPDATE SET active=excluded.active,updated_at=excluded.updated_at`, uuid.NewString(), policy.Version, string(raw), active, now, now)
	return err
}

func (r *Repository) putOverride(ctx context.Context, policy Policy, active bool, expectedVersion, document, now string) error {
	scopeKey := policy.Scope.Key()
	var currentVersion, currentDocument string
	var currentActive int
	err := r.db.QueryRowContext(ctx, `SELECT policy_version,document,active FROM plan_investigation_policy_overrides WHERE scope_key=?`, scopeKey).Scan(&currentVersion, &currentDocument, &currentActive)
	if expectedVersion = strings.TrimSpace(expectedVersion); expectedVersion != "" {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: expected active version %q for %s, but no override exists", ErrPolicyConflict, expectedVersion, scopeKey)
		}
		if err != nil {
			return err
		}
		if currentActive == 0 || currentVersion != expectedVersion {
			return fmt.Errorf("%w: expected version %q for %s, current version %q", ErrPolicyConflict, expectedVersion, scopeKey, currentVersion)
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && currentVersion == policy.Version && currentDocument != document {
		return fmt.Errorf("%w: policy version %q is immutable for %s", ErrPolicyConflict, policy.Version, scopeKey)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO plan_investigation_policy_overrides (scope_key,policy_version,document,active,created_at,updated_at) VALUES (?,?,?,?,?,?) ON CONFLICT(scope_key) DO UPDATE SET policy_version=excluded.policy_version,document=excluded.document,active=excluded.active,updated_at=excluded.updated_at`, scopeKey, policy.Version, document, active, now, now)
	return err
}

func (r *Repository) ActivePolicy(ctx context.Context) (Policy, error) {
	policy, _, err := r.ResolvePolicy(ctx, PolicyScope{})
	return policy, err
}

// ResolvePolicy chooses the most specific active override in the documented
// order phase > execution > family > global. The selected document is
// immutable once its version is reused, so an incident can retain the exact
// policy version that admitted it.
func (r *Repository) ResolvePolicy(ctx context.Context, scope PolicyScope) (Policy, string, error) {
	if err := scope.Validate(); err != nil {
		return Policy{}, "", err
	}
	for _, candidate := range []string{
		PolicyScope{PhaseID: scope.PhaseID}.Key(),
		PolicyScope{ExecutionID: scope.ExecutionID}.Key(),
		PolicyScope{FamilyID: scope.FamilyID}.Key(),
	} {
		if candidate == "global" {
			continue
		}
		var raw string
		if err := r.db.QueryRowContext(ctx, `SELECT document FROM plan_investigation_policy_overrides WHERE scope_key=? AND active=1`, candidate).Scan(&raw); err == nil {
			var policy Policy
			if err := json.Unmarshal([]byte(raw), &policy); err != nil {
				return Policy{}, "", fmt.Errorf("decode scoped policy: %w", err)
			}
			return policy, candidate, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return Policy{}, "", err
		}
	}
	var raw string
	if err := r.db.QueryRowContext(ctx, `SELECT document FROM plan_investigation_policies WHERE active=1 LIMIT 1`).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Policy{}, "global", ErrPolicyNotFound
		}
		return Policy{}, "global", err
	}
	var policy Policy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return Policy{}, "global", fmt.Errorf("decode policy: %w", err)
	}
	return policy, "global", nil
}

func (r *Repository) RecordIncident(ctx context.Context, observation Observation, decision Decision) (*Incident, bool, error) {
	if strings.TrimSpace(decision.IncidentFingerprint) == "" {
		return nil, false, nil
	}
	now := r.clock.Now().UTC()
	if !decision.Eligible {
		if err := r.recordOccurrence(ctx, observation, decision, now); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, false, err
	}
	state := "eligible"
	if decision.Queued {
		state = "queued"
	}
	incident := &Incident{ID: uuid.NewString(), ExecutionID: observation.ExecutionID, FamilyID: observation.FamilyID, PhaseID: observation.PhaseID, PhaseGeneration: observation.PhaseGeneration, PolicyVersion: observation.RuleVersion, IncidentFingerprint: decision.IncidentFingerprint, Mode: decision.Mode, State: state, Decision: decision, CreatedAt: now, UpdatedAt: now}
	result, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO plan_investigation_incidents (incident_id,execution_id,family_id,phase_id,phase_generation,policy_version,incident_fingerprint,mode,state,decision_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, incident.ID, incident.ExecutionID, incident.FamilyID, incident.PhaseID, incident.PhaseGeneration, incident.PolicyVersion, incident.IncidentFingerprint, incident.Mode, incident.State, string(raw), formatTime(now), formatTime(now))
	if err != nil {
		return nil, false, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		_, reused, err := r.GetIncident(ctx, decision.IncidentFingerprint)
		if err != nil {
			return nil, false, err
		}
		if !reused {
			return nil, false, fmt.Errorf("incident %q disappeared after duplicate insert", decision.IncidentFingerprint)
		}
		if err := r.recordOccurrence(ctx, observation, decision, now); err != nil {
			return nil, false, err
		}
		if err := r.recordSubjectRelation(ctx, decision.IncidentFingerprint, observation); err != nil {
			return nil, false, err
		}
		loaded, _, err := r.GetIncident(ctx, decision.IncidentFingerprint)
		return loaded, true, err
	}
	if err := r.recordOccurrence(ctx, observation, decision, now); err != nil {
		return nil, false, err
	}
	if err := r.recordSubjectRelation(ctx, decision.IncidentFingerprint, observation); err != nil {
		return nil, false, err
	}
	loaded, _, err := r.GetIncident(ctx, decision.IncidentFingerprint)
	return loaded, false, err
}

func (r *Repository) GetIncident(ctx context.Context, fingerprint string) (*Incident, bool, error) {
	var incident Incident
	var mode, decisionJSON, createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `SELECT incident_id,execution_id,family_id,phase_id,phase_generation,policy_version,incident_fingerprint,mode,state,decision_json,investigation_id,program_id,program_status,dispatch_error,dispatch_claim_key,dispatch_started_at,created_at,updated_at FROM plan_investigation_incidents WHERE incident_fingerprint=?`, fingerprint).Scan(&incident.ID, &incident.ExecutionID, &incident.FamilyID, &incident.PhaseID, &incident.PhaseGeneration, &incident.PolicyVersion, &incident.IncidentFingerprint, &mode, &incident.State, &decisionJSON, &incident.InvestigationID, &incident.ProgramID, &incident.ProgramStatus, &incident.DispatchError, &incident.DispatchClaimKey, &incident.DispatchStartedAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	incident.Mode = Mode(mode)
	if err := json.Unmarshal([]byte(decisionJSON), &incident.Decision); err != nil {
		return nil, false, err
	}
	incident.CreatedAt, incident.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
	if err := r.loadOccurrences(ctx, &incident, 100); err != nil {
		return nil, false, err
	}
	if err := r.loadSubjectRelations(ctx, &incident); err != nil {
		return nil, false, err
	}
	return &incident, true, nil
}

// ListIncidents returns a bounded operator history. Empty filters mean all
// incidents; ordering is newest first with a stable fingerprint tie-breaker.
func (r *Repository) ListIncidents(ctx context.Context, executionID, familyID, state string, limit int) ([]*Incident, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT incident_id,execution_id,family_id,phase_id,phase_generation,policy_version,incident_fingerprint,mode,state,decision_json,investigation_id,program_id,program_status,dispatch_error,dispatch_claim_key,dispatch_started_at,created_at,updated_at FROM plan_investigation_incidents`
	args := []any{}
	where := []string{}
	if strings.TrimSpace(executionID) != "" {
		where = append(where, "(execution_id=? OR incident_fingerprint IN (SELECT incident_fingerprint FROM plan_investigation_incident_subjects WHERE execution_id=?))")
		args = append(args, strings.TrimSpace(executionID), strings.TrimSpace(executionID))
	}
	if strings.TrimSpace(familyID) != "" {
		where = append(where, "family_id=?")
		args = append(args, strings.TrimSpace(familyID))
	}
	if strings.TrimSpace(state) != "" {
		where = append(where, "state=?")
		args = append(args, strings.TrimSpace(state))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, incident_fingerprint DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	incidents := make([]*Incident, 0, limit)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, incident := range incidents {
		if err := r.loadOccurrences(ctx, incident, 100); err != nil {
			return nil, err
		}
		if err := r.loadSubjectRelations(ctx, incident); err != nil {
			return nil, err
		}
	}
	return incidents, nil
}

// ListOccurrences returns every bounded trigger evaluation, including
// suppressed evaluations that did not create an incident. This lets operators
// distinguish "nothing observed" from "observed and intentionally suppressed".
func (r *Repository) ListOccurrences(ctx context.Context, executionID string, eligibleOnly bool, limit int) ([]Occurrence, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT occurrence_id,incident_fingerprint,execution_id,family_id,shared_failure_ref,phase_id,phase_generation,policy_version,eligible,decision_json,observed_at,created_at FROM plan_investigation_occurrences`
	args := []any{}
	where := []string{}
	if strings.TrimSpace(executionID) != "" {
		where = append(where, "execution_id=?")
		args = append(args, strings.TrimSpace(executionID))
	}
	if eligibleOnly {
		where = append(where, "eligible=1")
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY observed_at DESC, occurrence_id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	occurrences := make([]Occurrence, 0, limit)
	for rows.Next() {
		occurrence, err := scanOccurrence(rows)
		if err != nil {
			return nil, err
		}
		occurrences = append(occurrences, occurrence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return occurrences, nil
}

func (r *Repository) FindOccurrence(ctx context.Context, observation Observation, decision Decision) (*Occurrence, bool, error) {
	key, err := observationKey(observation, decision)
	if err != nil {
		return nil, false, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT occurrence_id,incident_fingerprint,execution_id,family_id,shared_failure_ref,phase_id,phase_generation,policy_version,eligible,decision_json,observed_at,created_at FROM plan_investigation_occurrences WHERE occurrence_key=?`, key)
	occurrence, err := scanOccurrence(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &occurrence, true, nil
}

func scanIncident(row interface{ Scan(...any) error }) (*Incident, error) {
	incident := &Incident{}
	var mode, decisionJSON, createdAt, updatedAt string
	if err := row.Scan(&incident.ID, &incident.ExecutionID, &incident.FamilyID, &incident.PhaseID, &incident.PhaseGeneration, &incident.PolicyVersion, &incident.IncidentFingerprint, &mode, &incident.State, &decisionJSON, &incident.InvestigationID, &incident.ProgramID, &incident.ProgramStatus, &incident.DispatchError, &incident.DispatchClaimKey, &incident.DispatchStartedAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	incident.Mode = Mode(mode)
	if err := json.Unmarshal([]byte(decisionJSON), &incident.Decision); err != nil {
		return nil, err
	}
	incident.CreatedAt, incident.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
	return incident, nil
}

func (r *Repository) recordOccurrence(ctx context.Context, observation Observation, decision Decision, now time.Time) error {
	key, err := observationKey(observation, decision)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT OR IGNORE INTO plan_investigation_occurrences (occurrence_id,incident_fingerprint,occurrence_key,execution_id,family_id,shared_failure_ref,phase_id,phase_generation,policy_version,eligible,decision_json,observed_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, uuid.NewString(), decision.IncidentFingerprint, key, observation.ExecutionID, observation.FamilyID, observation.SharedFailureRef, observation.PhaseID, observation.PhaseGeneration, observation.RuleVersion, decision.Eligible, string(raw), formatTime(observation.Now), formatTime(now))
	return err
}

func (r *Repository) recordSubjectRelation(ctx context.Context, fingerprint string, observation Observation) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO plan_investigation_incident_subjects (incident_fingerprint,execution_id,phase_id,phase_generation,relation,created_at) VALUES (?,?,?,?,?,?)`, fingerprint, observation.ExecutionID, observation.PhaseID, observation.PhaseGeneration, "implicated", formatTime(r.clock.Now()))
	return err
}

func (r *Repository) loadSubjectRelations(ctx context.Context, incident *Incident) error {
	rows, err := r.db.QueryContext(ctx, `SELECT execution_id FROM plan_investigation_incident_subjects WHERE incident_fingerprint=? ORDER BY execution_id ASC`, incident.IncidentFingerprint)
	if err != nil {
		return err
	}
	defer rows.Close()
	incident.SubjectExecutionIDs = make([]string, 0)
	for rows.Next() {
		var executionID string
		if err := rows.Scan(&executionID); err != nil {
			return err
		}
		incident.SubjectExecutionIDs = append(incident.SubjectExecutionIDs, executionID)
	}
	return rows.Err()
}

func (r *Repository) loadOccurrences(ctx context.Context, incident *Incident, limit int) error {
	rows, err := r.db.QueryContext(ctx, `SELECT occurrence_id,incident_fingerprint,execution_id,family_id,shared_failure_ref,phase_id,phase_generation,policy_version,eligible,decision_json,observed_at,created_at FROM plan_investigation_occurrences WHERE incident_fingerprint=? AND eligible=1 ORDER BY created_at DESC,occurrence_id DESC LIMIT ?`, incident.IncidentFingerprint, limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	incident.Occurrences = make([]Occurrence, 0)
	for rows.Next() {
		var occurrence Occurrence
		var decisionJSON, observedAt, createdAt string
		var eligible int
		if err := rows.Scan(&occurrence.ID, &occurrence.IncidentFingerprint, &occurrence.ExecutionID, &occurrence.FamilyID, &occurrence.SharedFailureRef, &occurrence.PhaseID, &occurrence.PhaseGeneration, &occurrence.PolicyVersion, &eligible, &decisionJSON, &observedAt, &createdAt); err != nil {
			return err
		}
		occurrence.Eligible = eligible != 0
		if err := json.Unmarshal([]byte(decisionJSON), &occurrence.Decision); err != nil {
			return err
		}
		occurrence.ObservedAt, occurrence.CreatedAt = parseTime(observedAt), parseTime(createdAt)
		incident.Occurrences = append(incident.Occurrences, occurrence)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM plan_investigation_occurrences WHERE incident_fingerprint=?`, incident.IncidentFingerprint).Scan(&incident.OccurrenceCount)
}

func scanOccurrence(row interface{ Scan(...any) error }) (Occurrence, error) {
	var occurrence Occurrence
	var decisionJSON, observedAt, createdAt string
	var eligible int
	if err := row.Scan(&occurrence.ID, &occurrence.IncidentFingerprint, &occurrence.ExecutionID, &occurrence.FamilyID, &occurrence.SharedFailureRef, &occurrence.PhaseID, &occurrence.PhaseGeneration, &occurrence.PolicyVersion, &eligible, &decisionJSON, &observedAt, &createdAt); err != nil {
		return Occurrence{}, err
	}
	occurrence.Eligible = eligible != 0
	if err := json.Unmarshal([]byte(decisionJSON), &occurrence.Decision); err != nil {
		return Occurrence{}, err
	}
	occurrence.ObservedAt, occurrence.CreatedAt = parseTime(observedAt), parseTime(createdAt)
	return occurrence, nil
}

func (r *Repository) LinkInvestigation(ctx context.Context, fingerprint, investigationID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET investigation_id=?,state=CASE WHEN state='completed' THEN state ELSE 'dispatched' END,updated_at=? WHERE incident_fingerprint=?`, strings.TrimSpace(investigationID), formatTime(r.clock.Now()), fingerprint)
	return err
}

// MarkInvestigationCompleted records the nested Agent Manager execution only
// after the composition wrapper has returned a terminal diagnosis. A program
// handle alone is not evidence of diagnostic completion.
func (r *Repository) MarkInvestigationCompleted(ctx context.Context, fingerprint, investigationID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET investigation_id=?,state='completed',updated_at=? WHERE incident_fingerprint=?`, strings.TrimSpace(investigationID), formatTime(r.clock.Now()), fingerprint)
	return err
}

// LinkProgram records the durable composition handle before a caller returns.
// The nested Agent Manager request uses the incident fingerprint as its stable
// caller key, so a retry after acknowledgement loss cannot create a second
// logical investigation even if the wrapper program itself is recreated.
func (r *Repository) LinkProgram(ctx context.Context, fingerprint, programID, status string) error {
	state := "dispatched"
	if strings.Contains(strings.ToLower(status), "failed") || strings.Contains(strings.ToLower(status), "cancelled") {
		state = "dispatch_failed"
	}
	_, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET program_id=?,program_status=?,state=?,dispatch_error='',dispatch_claim_key='',dispatch_started_at='',updated_at=? WHERE incident_fingerprint=?`, strings.TrimSpace(programID), strings.TrimSpace(status), state, formatTime(r.clock.Now()), fingerprint)
	return err
}

func (r *Repository) MarkDispatchFailed(ctx context.Context, fingerprint, detail string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET state='dispatch_failed',dispatch_error=?,dispatch_claim_key='',dispatch_started_at='',updated_at=? WHERE incident_fingerprint=?`, strings.TrimSpace(detail), formatTime(r.clock.Now()), fingerprint)
	return err
}

// MarkDispatchRetryable releases a composition handle whose terminal wrapper
// proved that no nested investigation was admitted. The outer program remains
// queryable in Program Runtime, while the incident can retry with the same
// caller key after the unavailable owner recovers.
func (r *Repository) MarkDispatchRetryable(ctx context.Context, fingerprint, programID, detail string) error {
	detail = strings.TrimSpace(detail)
	if strings.TrimSpace(programID) != "" {
		detail = strings.TrimSpace(detail + " (program " + strings.TrimSpace(programID) + ")")
	}
	_, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET program_id='',state='dispatch_failed',dispatch_error=?,dispatch_claim_key='',dispatch_started_at='',updated_at=? WHERE incident_fingerprint=?`, detail, formatTime(r.clock.Now()), fingerprint)
	return err
}

// ClaimDispatch atomically elects one automatic-trigger caller to create the
// durable composition handle. A process crash leaves a dispatching claim
// visible for recovery; failed dispatches explicitly clear it for retry.
func (r *Repository) ClaimDispatch(ctx context.Context, fingerprint, claimKey string) (bool, error) {
	claimKey = strings.TrimSpace(claimKey)
	if claimKey == "" {
		return false, errors.New("dispatch claim key is required")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET state='dispatching',dispatch_claim_key=?,dispatch_started_at=?,updated_at=? WHERE incident_fingerprint=? AND program_id='' AND dispatch_claim_key='' AND state IN ('eligible','dispatch_failed')`, claimKey, formatTime(r.clock.Now()), formatTime(r.clock.Now()), fingerprint)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

// RecoverStaleDispatch releases a claim left by an owner that disappeared
// after the durable claim commit and before the external program handle was
// recorded. It is deliberately conditional on an expired timestamp and an
// absent program ID, so an active or already-acknowledged dispatch cannot be
// reclaimed by a concurrent observation.
func (r *Repository) RecoverStaleDispatch(ctx context.Context, fingerprint string) (bool, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return false, errors.New("incident fingerprint is required")
	}
	now := r.clock.Now().UTC()
	cutoff := formatTime(now.Add(-dispatchClaimLease))
	result, err := r.db.ExecContext(ctx, `UPDATE plan_investigation_incidents SET state='dispatch_failed',dispatch_error='dispatch claim expired after owner restart',dispatch_claim_key='',dispatch_started_at='',updated_at=? WHERE incident_fingerprint=? AND program_id='' AND state='dispatching' AND dispatch_claim_key<>'' AND dispatch_started_at<>'' AND dispatch_started_at<=?`, formatTime(now), fingerprint, cutoff)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func observationKey(observation Observation, decision Decision) (string, error) {
	raw, err := json.Marshal(struct {
		Observation Observation `json:"observation"`
		Decision    string      `json:"decision"`
	}{Observation: observation, Decision: decision.IncidentFingerprint})
	if err != nil {
		return "", fmt.Errorf("marshal trigger occurrence identity: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func parseTime(value string) time.Time {
	result, _ := time.Parse(time.RFC3339Nano, value)
	return result
}
