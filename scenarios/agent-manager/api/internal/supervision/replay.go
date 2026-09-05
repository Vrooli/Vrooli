package supervision

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func finite(v float64) bool                           { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func (s *PolicyStore) SetReplayEvaluator(e Evaluator) { s.replay = e }

// BindEvaluator freezes executable identity on first use, including pre-existing
// policy records. Concurrent callers can only agree on the same artifact.
func (s *PolicyStore) BindEvaluator(ctx context.Context, version, digest string) error {
	if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
		return errors.New("a SHA-256 evaluator digest is required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO supervision_policy_artifacts(version,evaluator_digest) VALUES (?,?) ON CONFLICT(version) DO NOTHING`, version, digest)
	if err != nil {
		return err
	}
	var stored string
	if err = s.db.QueryRowContext(ctx, `SELECT evaluator_digest FROM supervision_policy_artifacts WHERE version=?`, version).Scan(&stored); err != nil {
		return err
	}
	if stored != digest {
		return errors.New("supervision evaluator changed under an immutable policy")
	}
	return nil
}

func sameOutcome(a, b SupervisionOutcome) bool {
	a.ID = ""
	b.ID = ""
	a.CreatedAt = time.Time{}
	b.CreatedAt = time.Time{}
	a.ExpiresAt = time.Time{}
	b.ExpiresAt = time.Time{}
	return reflect.DeepEqual(a, b)
}

// Assessment references must resolve to the same durable decision and subject.
// An empty observed class is explicitly unassessed, not a negative label.
func (s *PolicyStore) validateOutcome(ctx context.Context, o SupervisionOutcome) error {
	if o.WatchID == "" || o.ChildRunID == "" || len(o.EvidenceIDs) == 0 || !finite(o.CompletionImpact) {
		return errors.New("watch, child, evidence and finite impact are required")
	}
	var family, policy, predicted string
	err := s.db.QueryRowContext(ctx, `SELECT w.family_execution_id,w.spec_json,d.decision_json FROM cohort_watches w JOIN cohort_watch_decisions d ON d.watch_id=w.watch_id JOIN cohort_watch_subjects c ON c.watch_id=w.watch_id WHERE w.watch_id=? AND d.decision_id=? AND c.run_id=?`, o.WatchID, o.DecisionID, o.ChildRunID).Scan(&family, &policy, &predicted)
	if err != nil {
		return fmt.Errorf("outcome must reference a durable watched child and decision: %w", err)
	}
	var spec domainpb.WatchSpec
	var decision domainpb.WatchDecision
	if err = protojson.Unmarshal([]byte(policy), &spec); err != nil {
		return err
	}
	if err = protojson.Unmarshal([]byte(predicted), &decision); err != nil {
		return err
	}
	if family != o.FamilyExecutionID || spec.GetPolicyVersion() != o.PolicyVersion || decision.GetClassification() != o.PredictedClass {
		return errors.New("outcome attribution does not match decision")
	}
	if o.ActionID != "" {
		var n int
		if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cohort_watch_actions WHERE action_id=? AND watch_id=? AND decision_id=?`, o.ActionID, o.WatchID, o.DecisionID).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return errors.New("outcome action does not belong to decision")
		}
	}
	if o.ObservedClass != "" && !knownClass(o.ObservedClass) {
		return errors.New("unknown observed supervision class")
	}
	if o.Supersedes != "" {
		var watch, decision, child string
		if err = s.db.QueryRowContext(ctx, `SELECT watch_id,decision_id,child_run_id FROM supervision_outcomes WHERE outcome_id=?`, o.Supersedes).Scan(&watch, &decision, &child); err != nil {
			return err
		}
		if watch != o.WatchID || decision != o.DecisionID || child != o.ChildRunID {
			return errors.New("assessment supersession crosses its subject")
		}
	}
	return nil
}
func knownClass(c string) bool {
	switch c {
	case "quiet", "progress", "stalled", "blocked", "failed", "completed", "deadline", "quiet_time":
		return true
	}
	return false
}

// EvaluateCandidate executes the candidate against the exact retained input of
// each labelled decision. Caller counts and old predicted labels are not proof.
func (s *PolicyStore) EvaluateCandidate(ctx context.Context, version string, claimedRollout int, thresholds ReplayThresholds) (ReplayReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	report := ReplayReport{Version: version}
	if claimedRollout != 0 {
		return report, errors.New("rollout_samples is owner-derived; omit the caller count")
	}
	if s.replay == nil {
		return report, errors.New("supervision replay evaluator unavailable")
	}
	policy, err := s.Get(ctx, version)
	if err != nil {
		return report, err
	}
	if policy.State != "candidate" {
		return report, errors.New("only candidate policies can be evaluated")
	}
	thresholds.MinSamples = max(20, thresholds.MinSamples)
	thresholds.MinRolloutSamples = max(5, thresholds.MinRolloutSamples)
	if !finite(thresholds.MaxFalsePositiveRate) || !finite(thresholds.MaxFalseNegativeRate) || thresholds.MaxFalsePositiveRate < 0 || thresholds.MaxFalsePositiveRate > .1 || thresholds.MaxFalseNegativeRate < 0 || thresholds.MaxFalseNegativeRate > .1 {
		return report, errors.New("error-rate ceilings must be finite and within [0,0.1]")
	}
	var assessmentRevision int64
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM supervision_corpus_revision WHERE singleton=1`).Scan(&assessmentRevision); err != nil {
		return report, err
	}
	rows, err := s.db.QueryxContext(ctx, `SELECT o.outcome_id,o.decision_id,o.observed_class,o.safety_violation,o.completion_impact,i.input_json FROM supervision_outcomes o JOIN supervision_evaluation_inputs i ON i.decision_id=o.decision_id WHERE o.observed_class<>'' AND o.expires_at>? AND NOT EXISTS(SELECT 1 FROM supervision_outcomes newer WHERE newer.supersedes_outcome_id=o.outcome_id) ORDER BY o.created_at DESC,o.outcome_id LIMIT 200`, formatTime(s.now().UTC()))
	if err != nil {
		return report, err
	}
	type sample struct {
		id, decision, observed, raw string
		safety                      bool
		impact                      float64
	}
	samples := []sample{}
	for rows.Next() {
		var v sample
		if err = rows.Scan(&v.id, &v.decision, &v.observed, &v.safety, &v.impact, &v.raw); err != nil {
			rows.Close()
			return report, err
		}
		samples = append(samples, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	// Replay deliberately covers at most the newest 200 assessed records.
	// Unknown and expired records are excluded at the authority query.
	labels := map[string]string{}
	for _, sample := range samples {
		if prior, ok := labels[sample.decision]; ok && prior != sample.observed {
			return report, errors.New("conflicting child assessments for one cohort decision require review")
		}
		labels[sample.decision] = sample.observed
	}
	// Revoke prior gates before any potentially failing evaluation.
	if _, err = s.db.ExecContext(ctx, `DELETE FROM supervision_policy_gates WHERE version=?`, version); err != nil {
		return report, err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM supervision_replay_evidence WHERE version=?`, version); err != nil {
		return report, err
	}
	positives, negatives := 0, 0
	seen := map[string]bool{}
	for _, sample := range samples {
		if seen[sample.decision] {
			continue
		}
		seen[sample.decision] = true
		var input EvaluationInput
		if err = json.Unmarshal([]byte(sample.raw), &input); err != nil {
			return report, err
		}
		if input.Watch == nil || input.Watch.Spec == nil {
			return report, errors.New("replay input missing watch")
		}
		input.Watch = proto.Clone(input.Watch).(*domainpb.CohortWatch)
		input.Watch.Spec.PolicyVersion = version
		t := input.Watch.Spec.Triggers
		if t == nil {
			t = &domainpb.WatchTriggers{}
			input.Watch.Spec.Triggers = t
		}
		t.EventCount = policy.Policy.EventCount
		t.QuietTime = durationpb.New(time.Duration(policy.Policy.QuietSeconds) * time.Second)
		t.FrictionScore = policy.Policy.FrictionThreshold
		t.Terminal = policy.Policy.Terminal
		decision, evalErr := s.replay.Evaluate(ctx, input)
		if evalErr != nil {
			return report, evalErr
		}
		if decision == nil || decision.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE {
			return report, errors.New("candidate replay abstained or was unavailable; no promotion evidence")
		}
		observedSignal := isSignalClass(sample.observed)
		predictedSignal := isSignalClass(decision.GetClassification())
		if observedSignal {
			positives++
			if !predictedSignal {
				report.FalseNegatives++
			}
		} else {
			negatives++
			if predictedSignal {
				report.FalsePositives++
			}
		}
		// Binary signal accuracy alone misses false completion and unsafe actions.
		unsafe := false
		if decision.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL {
			if len(input.Subjects) == 0 || sample.observed != "completed" && sample.observed != "failed" {
				unsafe = true
			}
			for _, subject := range input.Subjects {
				unsafe = unsafe || !subject.Terminal
			}
		}
		permitted := false
		for _, action := range policy.Policy.AllowedActions {
			permitted = permitted || actionFromProgram(action) == decision.GetRecommendedAction()
		}
		if !permitted {
			unsafe = true
		}
		if !observedSignal && !predictedSignal && sample.observed != decision.GetClassification() {
			report.FalsePositives++
		}
		if unsafe {
			report.SafetyViolations++
		}
		report.SampleCount++
		if sample.safety {
			report.SafetyViolations++
		}
		pinned, err := s.Get(ctx, version)
		if err != nil {
			return report, err
		}
		if pinned.Policy.EvaluatorDigest == "" {
			return report, errors.New("replay did not pin evaluator identity")
		}
		_, err = s.db.ExecContext(ctx, `INSERT INTO supervision_replay_evidence(version,decision_id,outcome_id,evaluator_digest,predicted_class,observed_class,disposition,evaluated_at) VALUES (?,?,?,?,?,?,?,?)`, version, sample.decision, sample.id, pinned.Policy.EvaluatorDigest, decision.GetClassification(), sample.observed, int32(decision.GetDisposition()), formatTime(s.now().UTC()))
		if err != nil {
			return report, err
		}
	}
	// A rollout sample is an assessed decision actually executed under the candidate,
	// not a replay, fixture count, or caller assertion. Unknown outcomes never qualify.
	var impact sql.NullFloat64
	replaySafety := report.SafetyViolations
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT o.decision_id),AVG(CASE WHEN m.completion_impact_observed=1 THEN o.completion_impact END),COALESCE(SUM(o.safety_violation),0) FROM supervision_outcomes o JOIN supervision_evaluation_inputs i ON i.decision_id=o.decision_id LEFT JOIN supervision_outcome_measurements m ON m.outcome_id=o.outcome_id WHERE o.policy_version=? AND o.observed_class<>'' AND o.expires_at>? AND NOT EXISTS(SELECT 1 FROM supervision_outcomes newer WHERE newer.supersedes_outcome_id=o.outcome_id)`, version, formatTime(s.now().UTC())).Scan(&report.RolloutSamples, &impact, &report.SafetyViolations)
	if err != nil {
		return report, err
	}
	report.CompletionImpact = impact.Float64
	report.SafetyViolations += replaySafety
	report.ReplayPassed = report.SafetyViolations == 0 && report.SampleCount >= thresholds.MinSamples && positives > 0 && negatives > 0 && float64(report.FalsePositives)/float64(negatives) <= thresholds.MaxFalsePositiveRate && float64(report.FalseNegatives)/float64(positives) <= thresholds.MaxFalseNegativeRate
	report.RolloutPassed = report.RolloutSamples >= thresholds.MinRolloutSamples && impact.Valid && report.CompletionImpact >= 0 && report.SafetyViolations == 0
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return report, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	var currentRevision int64
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM supervision_corpus_revision WHERE singleton=1`).Scan(&currentRevision); err != nil {
		return report, err
	}
	if currentRevision != assessmentRevision {
		return report, errors.New("assessment corpus changed during replay; evaluate again before promotion")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO supervision_policy_gates(version,sample_count,false_positives,false_negatives,safety_violations,completion_impact,rollout_samples,replay_passed,rollout_passed,evaluated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, version, report.SampleCount, report.FalsePositives, report.FalseNegatives, report.SafetyViolations, report.CompletionImpact, report.RolloutSamples, report.ReplayPassed, report.RolloutPassed, formatTime(s.now().UTC()))
	if err != nil {
		return report, err
	}
	return report, tx.Commit()
}

func (s *PolicyStore) BindInference(ctx context.Context, version string, identity map[string]any) error {
	provider, _ := identity["provider"].(string)
	model, _ := identity["model"].(string)
	if provider == "" || model == "" || identity["applied"] == nil {
		return errors.New("gateway inference identity is required")
	}
	raw, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	if len(raw) > 2048 {
		return errors.New("gateway inference identity exceeds bound")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	_, err = s.db.ExecContext(ctx, `INSERT INTO supervision_inference_identity(version,identity_json,identity_digest) VALUES (?,?,?) ON CONFLICT(version) DO NOTHING`, version, string(raw), digest)
	if err != nil {
		return err
	}
	var pinned string
	if err = s.db.QueryRowContext(ctx, `SELECT identity_digest FROM supervision_inference_identity WHERE version=?`, version).Scan(&pinned); err != nil {
		return err
	}
	if pinned != digest {
		return errors.New("gateway inference identity changed under immutable policy")
	}
	return nil
}
