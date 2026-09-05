package supervision

import (
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

	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

var (
	ErrPolicyNotFound = errors.New("supervision policy not found")
	ErrPromotionGate  = errors.New("supervision policy promotion gate failed")
)

type SupervisionPolicy struct {
	Version            string   `json:"version"`
	EventCount         uint32   `json:"event_count"`
	QuietSeconds       int64    `json:"quiet_seconds"`
	FrictionThreshold  float64  `json:"friction_threshold"`
	Terminal           bool     `json:"terminal"`
	AllowedActions     []string `json:"allowed_actions"`
	ClassifierRevision string   `json:"classifier_revision"`
	EvaluatorDigest    string   `json:"evaluator_digest,omitempty"`
}

type PolicyRecord struct {
	Policy          SupervisionPolicy
	State           string
	Digest          string
	Supersedes      string
	CreatedBy       string
	ReviewedBy      string
	RejectionReason string
}

type SupervisionOutcome struct {
	ID                       string
	IdempotencyKey           string
	PolicyVersion            string
	FamilyExecutionID        string
	WatchID                  string
	DecisionID               string
	ActionID                 string
	ChildRunID               string
	EvidenceIDs              []string
	PredictedClass           string
	ObservedClass            string
	Overridden               bool
	Counterexample           bool
	SafetyViolation          bool
	CompletionImpact         float64
	CompletionImpactObserved bool
	Supersedes               string
	CreatedAt                time.Time
	ExpiresAt                time.Time
}

type ReplayThresholds struct {
	MinSamples           int
	MaxFalsePositiveRate float64
	MaxFalseNegativeRate float64
	MinRolloutSamples    int
}

type ReplayReport struct {
	Version          string
	SampleCount      int
	FalsePositives   int
	FalseNegatives   int
	SafetyViolations int
	CompletionImpact float64
	RolloutSamples   int
	ReplayPassed     bool
	RolloutPassed    bool
}

type OutcomeLedger interface {
	AppendSupervisionOutcome(context.Context, SupervisionOutcome) (string, error)
}

type OutcomeWriteResult struct {
	Outcome     SupervisionOutcome
	Reused      bool
	LedgerID    string
	LedgerError error
}

type PolicyStore struct {
	db     sqlcompat.DB
	now    func() time.Time
	ledger OutcomeLedger
	replay Evaluator
}

func NewPolicyStore(db sqlcompat.DB, ledger OutcomeLedger) *PolicyStore {
	return &PolicyStore{db: db, ledger: ledger, now: time.Now}
}

func (s *PolicyStore) CreateCandidate(ctx context.Context, policy SupervisionPolicy, supersedes, createdBy string) (PolicyRecord, error) {
	canonical, raw, digest, err := canonicalPolicy(policy)
	if err != nil {
		return PolicyRecord{}, err
	}
	if strings.TrimSpace(createdBy) == "" {
		return PolicyRecord{}, errors.New("candidate creator is required")
	}
	now := s.now().UTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO supervision_policies (version,state,policy_json,policy_digest,supersedes,created_by,created_at) VALUES (?,'candidate',?,?,?,?,?)`, canonical.Version, string(raw), digest, strings.TrimSpace(supersedes), strings.TrimSpace(createdBy), formatTime(now))
	if err != nil {
		var existing PolicyRecord
		existing, getErr := s.Get(ctx, canonical.Version)
		if getErr == nil && existing.Digest == digest {
			return existing, nil
		}
		return PolicyRecord{}, fmt.Errorf("create immutable supervision policy: %w", err)
	}
	return PolicyRecord{Policy: canonical, State: "candidate", Digest: digest, Supersedes: strings.TrimSpace(supersedes), CreatedBy: strings.TrimSpace(createdBy)}, nil
}

func (s *PolicyStore) Get(ctx context.Context, version string) (PolicyRecord, error) {
	var state, raw, digest, supersedes, createdBy, reviewedBy, rejection string
	err := s.db.QueryRowContext(ctx, `SELECT state,policy_json,policy_digest,supersedes,created_by,reviewed_by,rejection_reason FROM supervision_policies WHERE version=?`, strings.TrimSpace(version)).Scan(&state, &raw, &digest, &supersedes, &createdBy, &reviewedBy, &rejection)
	if errors.Is(err, sql.ErrNoRows) {
		return PolicyRecord{}, ErrPolicyNotFound
	}
	if err != nil {
		return PolicyRecord{}, err
	}
	var policy SupervisionPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return PolicyRecord{}, err
	}
	var artifact string
	if err := s.db.QueryRowContext(ctx, `SELECT evaluator_digest FROM supervision_policy_artifacts WHERE version=?`, version).Scan(&artifact); err == nil {
		policy.EvaluatorDigest = artifact
	} else if !errors.Is(err, sql.ErrNoRows) {
		return PolicyRecord{}, err
	}
	return PolicyRecord{Policy: policy, State: state, Digest: digest, Supersedes: supersedes, CreatedBy: createdBy, ReviewedBy: reviewedBy, RejectionReason: rejection}, nil
}

func (s *PolicyStore) Active(ctx context.Context) (PolicyRecord, error) {
	var version string
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM supervision_policies WHERE state='active'`).Scan(&version); errors.Is(err, sql.ErrNoRows) {
		return PolicyRecord{}, ErrPolicyNotFound
	} else if err != nil {
		return PolicyRecord{}, err
	}
	return s.Get(ctx, version)
}

// EnsureInitialActive installs the repository-owned baseline exactly once.
// Later versions must use the measured, human-reviewed promotion workflow.
func (s *PolicyStore) EnsureInitialActive(ctx context.Context, policy SupervisionPolicy, createdBy string) (PolicyRecord, error) {
	if active, err := s.Active(ctx); err == nil {
		return active, nil
	} else if !errors.Is(err, ErrPolicyNotFound) {
		return PolicyRecord{}, err
	}
	canonical, raw, digest, err := canonicalPolicy(policy)
	if err != nil {
		return PolicyRecord{}, err
	}
	now := s.now().UTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO supervision_policies (version,state,policy_json,policy_digest,created_by,created_at,reviewed_by,reviewed_at) VALUES (?,'active',?,?,?,?,?,?)`, canonical.Version, string(raw), digest, createdBy, formatTime(now), createdBy, formatTime(now))
	if err != nil {
		if active, getErr := s.Active(ctx); getErr == nil {
			return active, nil
		}
		return PolicyRecord{}, err
	}
	return s.Get(ctx, canonical.Version)
}

func DefaultSupervisionPolicy() SupervisionPolicy {
	return SupervisionPolicy{Version: "supervision-v1", EventCount: 64, QuietSeconds: 30, FrictionThreshold: .7, Terminal: true, AllowedActions: []string{"observe", "park", "escalate", "wake_parent"}, ClassifierRevision: "supervision-evaluate-v1"}
}

func (s *PolicyStore) RecordOutcome(ctx context.Context, outcome SupervisionOutcome) (OutcomeWriteResult, error) {
	if strings.TrimSpace(outcome.IdempotencyKey) == "" || strings.TrimSpace(outcome.PolicyVersion) == "" || strings.TrimSpace(outcome.FamilyExecutionID) == "" || strings.TrimSpace(outcome.DecisionID) == "" {
		return OutcomeWriteResult{}, errors.New("idempotency key, policy, family, and decision are required")
	}
	if err := s.validateOutcome(ctx, outcome); err != nil {
		return OutcomeWriteResult{}, err
	}
	if outcome.ID == "" {
		outcome.ID = uuid.NewString()
	}
	if outcome.CreatedAt.IsZero() {
		outcome.CreatedAt = s.now().UTC()
	}
	if outcome.ExpiresAt.IsZero() {
		outcome.ExpiresAt = outcome.CreatedAt.Add(180 * 24 * time.Hour)
	}
	outcome.EvidenceIDs = boundedPolicyEvidence(outcome.EvidenceIDs)
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return OutcomeWriteResult{}, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return OutcomeWriteResult{}, err
	}
	defer tx.Rollback()
	evidence, _ := json.Marshal(outcome.EvidenceIDs)
	result, err := tx.ExecContext(ctx, `INSERT INTO supervision_outcomes (outcome_id,idempotency_key,policy_version,family_execution_id,watch_id,decision_id,action_id,child_run_id,evidence_json,predicted_class,observed_class,overridden,counterexample,safety_violation,completion_impact,supersedes_outcome_id,created_at,expires_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, outcome.ID, outcome.IdempotencyKey, outcome.PolicyVersion, outcome.FamilyExecutionID, outcome.WatchID, outcome.DecisionID, outcome.ActionID, outcome.ChildRunID, string(evidence), outcome.PredictedClass, outcome.ObservedClass, outcome.Overridden, outcome.Counterexample, outcome.SafetyViolation, outcome.CompletionImpact, outcome.Supersedes, formatTime(outcome.CreatedAt), formatTime(outcome.ExpiresAt))
	if err != nil {
		return OutcomeWriteResult{}, err
	}
	inserted, _ := result.RowsAffected()
	if inserted > 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO supervision_outcome_measurements(outcome_id,completion_impact_observed) VALUES (?,?)`, outcome.ID, outcome.CompletionImpactObserved); err != nil {
			return OutcomeWriteResult{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return OutcomeWriteResult{}, err
	}

	write := OutcomeWriteResult{Outcome: outcome, Reused: inserted == 0}
	if inserted == 0 {
		if err := s.scanOutcome(s.db.QueryRowContext(ctx, `SELECT outcome_id,idempotency_key,policy_version,family_execution_id,watch_id,decision_id,action_id,child_run_id,evidence_json,predicted_class,observed_class,overridden,counterexample,safety_violation,completion_impact,supersedes_outcome_id,created_at,expires_at,COALESCE((SELECT completion_impact_observed FROM supervision_outcome_measurements m WHERE m.outcome_id=supervision_outcomes.outcome_id),0) FROM supervision_outcomes WHERE idempotency_key=?`, outcome.IdempotencyKey), &write.Outcome); err != nil {
			return OutcomeWriteResult{}, err
		}
		if !sameOutcome(write.Outcome, outcome) {
			return OutcomeWriteResult{}, errors.New("outcome idempotency key reused with changed payload")
		}
	}
	// Projection retries retain the canonical outcome identity.
	if s.ledger != nil {
		write.LedgerID, write.LedgerError = s.ledger.AppendSupervisionOutcome(ctx, write.Outcome)
	}
	return write, nil
}

func (s *PolicyStore) ListOutcomes(ctx context.Context, version string, limit int, watchIDs ...string) ([]SupervisionOutcome, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT outcome_id,idempotency_key,policy_version,family_execution_id,watch_id,decision_id,action_id,child_run_id,evidence_json,predicted_class,observed_class,overridden,counterexample,safety_violation,completion_impact,supersedes_outcome_id,created_at,expires_at,COALESCE((SELECT completion_impact_observed FROM supervision_outcome_measurements m WHERE m.outcome_id=supervision_outcomes.outcome_id),0) FROM supervision_outcomes`
	args := []any{}
	if version = strings.TrimSpace(version); version != "" {
		query += ` WHERE policy_version=?`
		args = append(args, version)
	}
	if len(watchIDs) > 0 && strings.TrimSpace(watchIDs[0]) != "" {
		if len(args) == 0 {
			query += " WHERE "
		} else {
			query += " AND "
		}
		query += "watch_id=?"
		args = append(args, strings.TrimSpace(watchIDs[0]))
	}
	query += ` ORDER BY created_at DESC,outcome_id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	outcomes := make([]SupervisionOutcome, 0, limit)
	for rows.Next() {
		var outcome SupervisionOutcome
		if err := s.scanOutcome(rows, &outcome); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, outcome)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return outcomes, nil
}

func (s *PolicyStore) Promote(ctx context.Context, version, reviewedBy string) (PolicyRecord, error) {
	if _, err := s.PruneExpired(ctx); err != nil {
		return PolicyRecord{}, err
	}
	if strings.TrimSpace(reviewedBy) == "" {
		return PolicyRecord{}, errors.New("human reviewer is required")
	}
	// Gates must be earned by actual replay, not legacy aggregate-only rows.
	var evidence int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM supervision_replay_evidence WHERE version=?`, version).Scan(&evidence); err != nil || evidence < 20 {
		return PolicyRecord{}, ErrPromotionGate
	}
	var replay, rollout bool
	if err := s.db.QueryRowContext(ctx, `SELECT replay_passed,rollout_passed FROM supervision_policy_gates WHERE version=?`, version).Scan(&replay, &rollout); err != nil || !replay || !rollout {
		return PolicyRecord{}, ErrPromotionGate
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return PolicyRecord{}, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return PolicyRecord{}, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT replay_passed,rollout_passed FROM supervision_policy_gates WHERE version=?`, version).Scan(&replay, &rollout); err != nil || !replay || !rollout {
		return PolicyRecord{}, ErrPromotionGate
	}
	now := formatTime(s.now().UTC())
	if _, err := tx.ExecContext(ctx, `UPDATE supervision_policies SET state='retired' WHERE state='active'`); err != nil {
		return PolicyRecord{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE supervision_policies SET state='active',reviewed_by=?,reviewed_at=? WHERE version=? AND state='candidate'`, strings.TrimSpace(reviewedBy), now, version)
	if err != nil {
		return PolicyRecord{}, err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return PolicyRecord{}, ErrPolicyNotFound
	}
	if err := tx.Commit(); err != nil {
		return PolicyRecord{}, err
	}
	return s.Get(ctx, version)
}

func (s *PolicyStore) Reject(ctx context.Context, version, reviewedBy, reason string) error {
	if strings.TrimSpace(reviewedBy) == "" || strings.TrimSpace(reason) == "" {
		return errors.New("reviewer and rejection reason are required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE supervision_policies SET state='rejected',reviewed_by=?,reviewed_at=?,rejection_reason=? WHERE version=? AND state='candidate'`, reviewedBy, formatTime(s.now().UTC()), reason, version)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return ErrPolicyNotFound
	}
	return nil
}

func (s *PolicyStore) Rollback(ctx context.Context, activeVersion, reviewedBy string) (PolicyRecord, error) {
	active, err := s.Get(ctx, activeVersion)
	if err != nil || active.State != "active" || active.Supersedes == "" || reviewedBy == "" {
		return PolicyRecord{}, errors.New("active policy with a predecessor and reviewer is required")
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return PolicyRecord{}, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return PolicyRecord{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE supervision_policies SET state='rolled_back',reviewed_by=?,reviewed_at=? WHERE version=? AND state='active'`, reviewedBy, formatTime(s.now().UTC()), activeVersion); err != nil {
		return PolicyRecord{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE supervision_policies SET state='active' WHERE version=? AND state='retired'`, active.Supersedes)
	if err != nil {
		return PolicyRecord{}, err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return PolicyRecord{}, ErrPolicyNotFound
	}
	if err := tx.Commit(); err != nil {
		return PolicyRecord{}, err
	}
	return s.Get(ctx, active.Supersedes)
}

func (s *PolicyStore) SetDisabled(ctx context.Context, disabled bool, reason, actor string) error {
	if actor == "" || (disabled && strings.TrimSpace(reason) == "") {
		return errors.New("actor and disable reason are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO supervision_policy_control (singleton,disabled,reason,updated_by,updated_at) VALUES (1,?,?,?,?) ON CONFLICT(singleton) DO UPDATE SET disabled=excluded.disabled,reason=excluded.reason,updated_by=excluded.updated_by,updated_at=excluded.updated_at`, disabled, reason, actor, formatTime(s.now().UTC()))
	return err
}

func (s *PolicyStore) Disabled(ctx context.Context) (bool, string, error) {
	var disabled bool
	var reason string
	err := s.db.QueryRowContext(ctx, `SELECT disabled,reason FROM supervision_policy_control WHERE singleton=1`).Scan(&disabled, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	return disabled, reason, err
}

// PolicyControlledEvaluator turns the emergency switch into a replay-safe
// unavailable decision. It never falls back to an ungoverned classifier.
type PolicyControlledEvaluator struct {
	Store    *PolicyStore
	Delegate Evaluator
}

func (e PolicyControlledEvaluator) Evaluate(ctx context.Context, input EvaluationInput) (*domainpb.WatchDecision, error) {
	disabled, reason, err := e.Store.Disabled(ctx)
	if err != nil {
		return unavailableProgramDecision(input, "policy_control_unavailable"), nil
	}
	if disabled {
		classification := "supervision_disabled"
		if reason != "" {
			classification += ":" + reason
		}
		return unavailableProgramDecision(input, classification), nil
	}
	return e.Delegate.Evaluate(ctx, input)
}

func (s *PolicyStore) PruneExpired(ctx context.Context) (int64, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	cutoff := formatTime(s.now().UTC())
	// Expired assessments cannot continue to authorize promotion. Remove dependent
	// replay references before deleting their source outcomes.
	if _, err = tx.ExecContext(ctx, `DELETE FROM supervision_policy_gates WHERE version IN (SELECT r.version FROM supervision_replay_evidence r JOIN supervision_outcomes o ON o.outcome_id=r.outcome_id WHERE o.expires_at<=?)`, cutoff); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM supervision_replay_evidence WHERE outcome_id IN (SELECT outcome_id FROM supervision_outcomes WHERE expires_at<=?)`, cutoff); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM supervision_outcomes WHERE expires_at<=?`, cutoff)
	if err != nil {
		return 0, err
	}
	// Inputs share the assessment retention window; keep only those with a live outcome.
	if _, err = tx.ExecContext(ctx, `DELETE FROM supervision_evaluation_inputs WHERE NOT EXISTS (SELECT 1 FROM supervision_outcomes o WHERE o.decision_id=supervision_evaluation_inputs.decision_id)`); err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

func canonicalPolicy(policy SupervisionPolicy) (SupervisionPolicy, []byte, string, error) {
	// Normalize before hashing so replay and live watches use the same safety defaults.
	if policy.EventCount == 0 {
		policy.EventCount = 64
	}
	policy.Terminal = true
	policy.Version = strings.TrimSpace(policy.Version)
	policy.ClassifierRevision = strings.TrimSpace(policy.ClassifierRevision)
	if policy.Version == "" || len(policy.Version) > 64 || policy.EventCount > 64 || policy.QuietSeconds < 1 || policy.QuietSeconds > 86400 || !finite(policy.FrictionThreshold) || policy.FrictionThreshold < 0 || policy.FrictionThreshold > 1 {
		return SupervisionPolicy{}, nil, "", errors.New("invalid bounded supervision policy")
	}
	allowed := map[string]bool{"observe": true, "park": true, "nudge": true, "escalate": true, "wake_parent": true}
	seen := map[string]bool{}
	for _, action := range policy.AllowedActions {
		if !allowed[action] || seen[action] {
			return SupervisionPolicy{}, nil, "", fmt.Errorf("invalid or duplicate symbolic action %q", action)
		}
		seen[action] = true
	}
	sort.Strings(policy.AllowedActions)
	raw, err := json.Marshal(policy)
	if err != nil {
		return SupervisionPolicy{}, nil, "", err
	}
	digestBytes := sha256.Sum256(raw)
	return policy, raw, hex.EncodeToString(digestBytes[:]), nil
}

func boundedPolicyEvidence(values []string) []string {
	if len(values) > 20 {
		values = values[:20]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if len(value) > 128 {
			value = value[:128]
		}
		out = append(out, value)
	}
	return out
}

func isSignalClass(value string) bool {
	switch strings.ToLower(value) {
	case "stalled", "blocked", "failed", "deadline", "quiet_time":
		return true
	default:
		return false
	}
}

type outcomeScanner interface{ Scan(...any) error }

func (s *PolicyStore) scanOutcome(row outcomeScanner, out *SupervisionOutcome) error {
	var evidence, created, expires string
	if err := row.Scan(&out.ID, &out.IdempotencyKey, &out.PolicyVersion, &out.FamilyExecutionID, &out.WatchID, &out.DecisionID, &out.ActionID, &out.ChildRunID, &evidence, &out.PredictedClass, &out.ObservedClass, &out.Overridden, &out.Counterexample, &out.SafetyViolation, &out.CompletionImpact, &out.Supersedes, &created, &expires, &out.CompletionImpactObserved); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(evidence), &out.EvidenceIDs); err != nil {
		return err
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return nil
}
