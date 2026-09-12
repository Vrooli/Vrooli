package tasks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// LearningResult binds an opaque receipt to the exact artifact and attempt.
// The receipt grants feedback authority, not authority to execute that artifact.
type LearningResult struct {
	Owner               string         `json:"owner,omitempty"`
	Artifact            map[string]any `json:"artifact,omitempty"`
	Name                string         `json:"name,omitempty"`
	FeedbackRef         string         `json:"feedback_ref"`
	TaskID              string         `json:"task_id"`
	AttemptID           string         `json:"attempt_id"`
	StepAttemptID       string         `json:"step_attempt_id"`
	Scope               string         `json:"scope"`
	Verb                string         `json:"verb"`
	StepKey             string         `json:"step_key"`
	FragmentHash        string         `json:"fragment_hash"`
	StepName            string         `json:"step_name"`
	CompatibilityDigest string         `json:"compatibility_digest"`
	Provenance          string         `json:"provenance"`
}
type Feedback struct {
	ObserverProvenance string   `json:"observer_provenance,omitempty"`
	ObservationID      string   `json:"observation_id"`
	FeedbackRef        string   `json:"feedback_ref"`
	Disposition        string   `json:"disposition"`
	Dimension          string   `json:"dimension"`
	Evidence           []string `json:"evidence"`
	Correction         string   `json:"correction"`
}
type FindingTransition struct {
	State    string   `json:"state"`
	Evidence []string `json:"evidence"`
	At       string   `json:"at"`
}
type Finding struct {
	TaskID         string              `json:"task_id,omitempty"`
	AttemptID      string              `json:"attempt_id,omitempty"`
	Artifact       map[string]any      `json:"artifact,omitempty"`
	ClaimExpiresAt string              `json:"claim_expires_at,omitempty"`
	ClaimRef       string              `json:"claim_ref,omitempty"`
	History        []FindingTransition `json:"history,omitempty"`
	ID             string              `json:"finding_id"`
	Owner          string              `json:"owner"`
	State          string              `json:"state"`
	FeedbackRef    string              `json:"feedback_ref"`
	ObservationID  string              `json:"observation_id"`
	Dimension      string              `json:"dimension"`
	Correction     string              `json:"correction"`
	Evidence       []string            `json:"evidence"`
	UpdatedAt      string              `json:"updated_at"`
}

func (s *Store) PutLearningResult(ctx context.Context, r LearningResult) error {
	if len(r.FragmentHash) == 64 {
		r.FragmentHash = "sha256:" + r.FragmentHash
	}
	if !strings.HasPrefix(r.FeedbackRef, "prt_feedback_v1_") || len(r.FeedbackRef) < 45 || len(r.FeedbackRef) > 128 || r.AttemptID == "" {
		return errors.New("bounded opaque feedback reference and attempt required")
	}
	body, err := json.Marshal(r)
	if err != nil || len(body) > 8192 {
		return errors.New("bounded learning result required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var old string
	err = s.db.QueryRowContext(ctx, "SELECT record FROM learning_results WHERE feedback_ref=?", r.FeedbackRef).Scan(&old)
	if err == nil {
		if old != string(body) {
			return errors.New("feedback reference identity is immutable")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO learning_results(feedback_ref,attempt_id,provenance,record) VALUES(?,?,?,?)", r.FeedbackRef, r.AttemptID, r.Provenance, string(body))
	return err
}

// ApplyFeedback atomically retains the observation, invalidates confirmed faulty
// code, and emits a durable finding. Retries cannot inflate contradictions.
func (s *Store) ApplyFeedback(ctx context.Context, f Feedback, provenance string) (*Finding, error) {
	f.ObserverProvenance = provenance
	if f.ObservationID == "" || len(f.ObservationID) > 128 || (f.Disposition != "supported" && f.Disposition != "contradicted") {
		return nil, errors.New("observation identity and disposition required")
	}
	switch f.Dimension {
	case "execution", "verification", "usefulness", "efficiency", "context":
	default:
		return nil, errors.New("invalid feedback dimension")
	}
	if len(f.Evidence) == 0 || len(f.Evidence) > 16 || len(f.Correction) > 2048 {
		return nil, errors.New("bounded feedback evidence required")
	}
	for _, e := range f.Evidence {
		if strings.TrimSpace(e) == "" || len(e) > 512 {
			return nil, errors.New("invalid feedback evidence")
		}
	}
	body, _ := json.Marshal(f)
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var raw string
	if err = tx.QueryRowContext(ctx, "SELECT record FROM learning_results WHERE feedback_ref=? AND (provenance=? OR (? IN ('agent','operator') AND provenance IN ('agent','operator')))", f.FeedbackRef, provenance, provenance).Scan(&raw); err != nil {
		return nil, err
	}
	var target LearningResult
	if err = json.Unmarshal([]byte(raw), &target); err != nil {
		return nil, err
	}
	var previous string
	err = tx.QueryRowContext(ctx, "SELECT record FROM learning_feedback WHERE observation_id=?", f.ObservationID).Scan(&previous)
	if err == nil {
		if previous != string(body) {
			return nil, errors.New("observation identity conflict")
		}
		if f.Disposition == "contradicted" {
			id := findingIdentity(target, f)
			var rawFinding string
			if err = tx.QueryRowContext(ctx, "SELECT record FROM learning_findings WHERE finding_id=?", id).Scan(&rawFinding); err != nil {
				return nil, err
			}
			var existing Finding
			if err = json.Unmarshal([]byte(rawFinding), &existing); err != nil {
				return nil, err
			}
			existing.ClaimRef = ""
			return &existing, nil
		}
		return nil, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, "INSERT INTO learning_feedback(observation_id,feedback_ref,record,created_at) VALUES(?,?,?,?)", f.ObservationID, f.FeedbackRef, string(body), now); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO learning_feedback_delivery(observation_id,next_attempt_at) VALUES(?,?)", f.ObservationID, now); err != nil {
		return nil, err
	}
	var finding *Finding
	if f.Disposition == "contradicted" {
		if (f.Dimension == "execution" || f.Dimension == "verification") && target.StepKey != "" && target.FragmentHash != "" {
			if _, err = tx.ExecContext(ctx, "UPDATE learning_fragments SET contradicted_since_edit=contradicted_since_edit+1 WHERE step_key=? AND fragment_hash=?", target.StepKey, target.FragmentHash); err != nil {
				return nil, err
			}
		}
		owner := strings.TrimSuffix(target.Scope, "-usage")
		if artifactOwner, ok := target.Artifact["owner"].(string); ok && artifactOwner != "" {
			owner = artifactOwner
		}
		if owner == "bas" {
			owner = "browser-automation-studio"
		}
		if target.Owner != "" {
			owner = target.Owner
		}
		if owner == "" {
			owner = "program-runtime"
		}
		id := findingIdentity(target, f)
		finding = &Finding{TaskID: target.TaskID, AttemptID: target.AttemptID, Artifact: target.Artifact, ID: id, Owner: owner, State: "observed", FeedbackRef: f.FeedbackRef, ObservationID: f.ObservationID, Dimension: f.Dimension, Correction: f.Correction, Evidence: f.Evidence, UpdatedAt: now}
		var existingRaw string
		existingErr := tx.QueryRowContext(ctx, "SELECT record FROM learning_findings WHERE finding_id=?", id).Scan(&existingRaw)
		if existingErr == nil {
			var existing Finding
			if err = json.Unmarshal([]byte(existingRaw), &existing); err != nil {
				return nil, err
			}
			finding.History = existing.History
			finding.State = existing.State
			finding.ClaimRef = existing.ClaimRef
			finding.ClaimExpiresAt = existing.ClaimExpiresAt
			if existing.State == "measured" {
				finding.State = "observed"
				finding.ClaimRef = ""
				finding.ClaimExpiresAt = ""
				finding.History = append(finding.History, FindingTransition{State: "observed", Evidence: f.Evidence, At: now})
			}
		} else if !errors.Is(existingErr, sql.ErrNoRows) {
			return nil, existingErr
		}
		encoded, _ := json.Marshal(finding)
		if _, err = tx.ExecContext(ctx, "INSERT INTO learning_findings(finding_id,owner,state,record,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(finding_id) DO UPDATE SET state=excluded.state,record=excluded.record,updated_at=excluded.updated_at", id, owner, finding.State, string(encoded), now); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if finding != nil {
		finding.ClaimRef = ""
	}
	return finding, nil
}

func (s *Store) ListFindings(ctx context.Context, owner string) ([]Finding, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT record FROM learning_findings WHERE (?='' OR owner=?) AND state<>'measured' ORDER BY updated_at,finding_id LIMIT 101", owner, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Finding{}
	for rows.Next() {
		var raw string
		var f Finding
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &f); err != nil {
			return nil, err
		}
		f.ClaimRef = ""
		f.FeedbackRef = ""
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *Store) TransitionFinding(ctx context.Context, id, expected, next, owner, claimRef string, evidence []string) (*Finding, error) {
	allowed := map[string]string{"observed": "routed", "routed": "claimed", "claimed": "repaired", "repaired": "validated", "validated": "measured"}
	if (allowed[expected] != next && !(expected == "claimed" && next == "claimed")) || next == "" || len(evidence) == 0 || len(evidence) > 16 {
		return nil, errors.New("ordered transition and evidence required")
	}
	for _, e := range evidence {
		if strings.TrimSpace(e) == "" || len(e) > 512 {
			return nil, errors.New("bounded transition evidence required")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var raw string
	if err := s.db.QueryRowContext(ctx, "SELECT record FROM learning_findings WHERE finding_id=?", id).Scan(&raw); err != nil {
		return nil, err
	}
	var f Finding
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, err
	}
	if owner != f.Owner {
		return nil, errors.New("finding owner must match")
	}
	// Repeating an acknowledged transition with the same capability and
	// evidence is observation, not a second transition.
	if f.State == next && len(f.History) > 0 && reflect.DeepEqual(f.History[len(f.History)-1].Evidence, evidence) && (next == "routed" || (claimRef != "" && claimRef == f.ClaimRef)) {
		f.FeedbackRef = ""
		return &f, nil
	}
	if f.State != expected {
		return nil, errors.New("finding state changed")
	}
	reclaim := expected == "claimed" && next == "claimed"
	if reclaim && claimRef != f.ClaimRef {
		expires, err := time.Parse(time.RFC3339Nano, f.ClaimExpiresAt)
		if err != nil || time.Now().Before(expires) {
			return nil, errors.New("finding claim lease has not expired")
		}
	}
	if !reclaim && expected != "observed" && expected != "routed" && (claimRef == "" || claimRef != f.ClaimRef) {
		return nil, errors.New("finding claim receipt required")
	}
	if next == "claimed" {
		if claimRef != "" {
			if !strings.HasPrefix(claimRef, "prt_claim_v1_") || len(claimRef) < 44 || len(claimRef) > 128 {
				return nil, errors.New("opaque claim receipt required")
			}
			f.ClaimRef = claimRef
		} else {
			var token [32]byte
			if _, err := rand.Read(token[:]); err != nil {
				return nil, err
			}
			f.ClaimRef = "prt_claim_v1_" + hex.EncodeToString(token[:])
		}
		f.ClaimExpiresAt = time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339Nano)
	}
	f.State = next
	f.History = append(f.History, FindingTransition{State: next, Evidence: evidence, At: time.Now().UTC().Format(time.RFC3339Nano)})
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	body, _ := json.Marshal(f)
	_, err := s.db.ExecContext(ctx, "UPDATE learning_findings SET state=?,record=?,updated_at=? WHERE finding_id=? AND state=?", next, string(body), f.UpdatedAt, id, expected)
	f.FeedbackRef = ""
	return &f, err
}

// EligibleFragment filters evidence before ranking, so stale high-count code
// cannot conceal a lower-count implementation that meets the caller's policy.
func (s *Store) EligibleFragment(ctx context.Context, key string, minVerified, minContexts int, maxAgeDays float64) (*Fragment, error) {
	if key == "" || len(key) > 512 || minVerified < 0 || minVerified > 100 || minContexts < 0 || minContexts > 100 || maxAgeDays < 0 || maxAgeDays > 36500 {
		return nil, errors.New("invalid fragment eligibility policy")
	}
	cutoff := ""
	if maxAgeDays > 0 {
		cutoff = time.Now().UTC().Add(-time.Duration(maxAgeDays * float64(24*time.Hour))).Format(time.RFC3339Nano)
	}
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT f.fragment_hash FROM learning_fragments f LEFT JOIN learning_fragment_evidence e ON e.step_key=f.step_key AND e.fragment_hash=f.fragment_hash WHERE f.step_key=? AND f.verified>=? AND f.verified>0 AND f.contradicted_since_edit=0 AND NOT EXISTS (SELECT 1 FROM learning_results r JOIN learning_feedback feedback ON feedback.feedback_ref=r.feedback_ref WHERE json_extract(r.record,'$.step_key')=f.step_key AND json_extract(r.record,'$.fragment_hash')=f.fragment_hash AND json_extract(feedback.record,'$.disposition')='contradicted' AND json_extract(feedback.record,'$.dimension') IN ('execution','verification')) GROUP BY f.fragment_hash HAVING COUNT(DISTINCT CASE WHEN e.outcome='verified_success' THEN e.input_digest END)>=? AND (?='' OR MAX(CASE WHEN e.outcome='verified_success' THEN e.created_at END)>=?) ORDER BY f.verified DESC,f.last_used_at DESC,f.created_at DESC,f.fragment_hash LIMIT 1`, key, minVerified, minContexts, cutoff, cutoff).Scan(&hash)
	if err != nil {
		return nil, err
	}
	return s.fragmentByHash(ctx, key, hash)
}

func (s *Store) RejectedFragmentHashes(ctx context.Context, key string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT fragment_hash FROM learning_fragments WHERE step_key=? AND contradicted_since_edit>0 UNION SELECT json_extract(r.record,'$.fragment_hash') FROM learning_results r JOIN learning_feedback f ON f.feedback_ref=r.feedback_ref WHERE json_extract(r.record,'$.step_key')=? AND json_extract(f.record,'$.disposition')='contradicted' AND json_extract(f.record,'$.dimension') IN ('execution','verification') ORDER BY fragment_hash LIMIT 101`, key, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var h string
		if err = rows.Scan(&h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
func (s *Store) ListLearningFindings(ctx context.Context, owner string) ([]Finding, error) {
	return s.ListFindings(ctx, owner)
}

// LearningResultStatus returns only eligibility and evidence identity. It never
// exposes other feedback capabilities or treats missing evidence as success.
func (s *Store) LearningResultStatus(ctx context.Context, ref string, artifact map[string]any, provenance string) (map[string]any, error) {
	if ref == "" && len(artifact) == 0 {
		return nil, errors.New("feedback reference or artifact required")
	}
	encoded, err := json.Marshal(artifact)
	if err != nil || len(encoded) > 4096 {
		return nil, errors.New("bounded artifact required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.record,f.record FROM learning_results r LEFT JOIN learning_feedback f ON f.feedback_ref=r.feedback_ref WHERE (r.provenance=? OR (? IN ('agent','operator') AND r.provenance IN ('agent','operator'))) AND ((?<>'' AND r.feedback_ref=?) OR (?='' AND json_extract(r.record,'$.artifact')=?)) ORDER BY f.created_at DESC LIMIT 101`, provenance, provenance, ref, ref, ref, string(encoded))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{"found": false, "eligible": true, "contradicted": false, "observations": []Feedback{}}
	observations := []Feedback{}
	for rows.Next() {
		var resultRaw string
		var feedbackRaw sql.NullString
		if err = rows.Scan(&resultRaw, &feedbackRaw); err != nil {
			return nil, err
		}
		var result LearningResult
		if err = json.Unmarshal([]byte(resultRaw), &result); err != nil {
			return nil, err
		}
		out["found"] = true
		out["attempt_id"] = result.AttemptID
		out["step_attempt_id"] = result.StepAttemptID
		out["scope"] = result.Scope
		out["step_key"] = result.StepKey
		out["fragment_hash"] = result.FragmentHash
		if feedbackRaw.Valid {
			var f Feedback
			if err = json.Unmarshal([]byte(feedbackRaw.String), &f); err != nil {
				return nil, err
			}
			f.FeedbackRef = ""
			observations = append(observations, f)
			if f.Disposition == "contradicted" && (f.Dimension == "execution" || f.Dimension == "verification" || f.Dimension == "usefulness") {
				out["eligible"] = false
				out["contradicted"] = true
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	var contradicted bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_results r JOIN learning_feedback f ON f.feedback_ref=r.feedback_ref WHERE (r.provenance=? OR (? IN ('agent','operator') AND r.provenance IN ('agent','operator'))) AND ((?<>'' AND r.feedback_ref=?) OR (?='' AND json_extract(r.record,'$.artifact')=?)) AND json_extract(f.record,'$.disposition')='contradicted' AND json_extract(f.record,'$.dimension') IN ('execution','verification','usefulness'))`, provenance, provenance, ref, ref, ref, string(encoded)).Scan(&contradicted)
	if err != nil {
		return nil, err
	}
	out["contradicted"] = contradicted
	out["eligible"] = !contradicted
	out["truncated"] = len(observations) > 100
	if len(observations) > 100 {
		observations = observations[:100]
	}
	out["observations"] = observations
	return out, nil
}

// RouteFindings is called by the server outbox worker, independently of the
// task agent. Routing assigns the already validated owning scenario queue.
func (s *Store) RouteFindings(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "SELECT record FROM learning_findings WHERE state='observed' ORDER BY updated_at,finding_id LIMIT 100")
	if err != nil {
		return err
	}
	findings := []Finding{}
	for rows.Next() {
		var raw string
		var f Finding
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal([]byte(raw), &f); err != nil {
			rows.Close()
			return err
		}
		findings = append(findings, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, finding := range findings {
		if finding.State != "observed" {
			continue
		}
		_, err = s.TransitionFinding(ctx, finding.ID, "observed", "routed", finding.Owner, "", []string{"program-runtime:owner-queue:" + finding.Owner})
		if err != nil {
			return err
		}
	}
	return nil
}

// DrainFeedback projects feedback into the original task's Memory scope. The
// runtime eligibility transaction is authoritative even while Memory is down.
func (s *Store) DrainFeedback(ctx context.Context, now time.Time, deliver Deliver) error {
	if deliver == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT f.observation_id,f.record,f.created_at,r.record,d.attempts FROM learning_feedback_delivery d JOIN learning_feedback f ON f.observation_id=d.observation_id JOIN learning_results r ON r.feedback_ref=f.feedback_ref WHERE d.state='pending' AND d.next_attempt_at<=? ORDER BY d.next_attempt_at LIMIT 32`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	type pending struct {
		id, feedback, created, result string
		attempts                      int
	}
	items := []pending{}
	for rows.Next() {
		var item pending
		if err = rows.Scan(&item.id, &item.feedback, &item.created, &item.result, &item.attempts); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		var f Feedback
		var result LearningResult
		if err = json.Unmarshal([]byte(item.feedback), &f); err != nil {
			return err
		}
		if err = json.Unmarshal([]byte(item.result), &result); err != nil {
			return err
		}
		record, deliveryErr := s.Get(ctx, result.AttemptID)
		if deliveryErr == nil && (record.State != "completed" || record.FinishInputs == nil || record.FinishDigest == "") {
			deliveryErr = errors.New("original task capture or Memory contract unavailable")
		}
		if deliveryErr == nil {
			// Get decodes a fresh record, so this projection cannot mutate its checkpoint.
			target := result.StepAttemptID
			if target == "" {
				target = result.AttemptID
			}
			record.FinishInputs["observations"] = []any{map[string]any{"attempt_id": target, "disposition": f.Disposition, "evidence_refs": f.Evidence, "method_revision": "learn.feedback/" + f.Dimension, "correction": f.Correction, "provenance": result.Provenance, "observed_at": item.created}}
			var response map[string]any
			response, deliveryErr = deliver(ctx, *record)
			signals, _ := response["signals"].(map[string]any)
			if deliveryErr == nil && (response["status"] != "ok" || signals["capture_status"] != "complete") {
				deliveryErr = errors.New("feedback capture not acknowledged")
			}
		}
		state := "delivered"
		lastError := ""
		if deliveryErr != nil {
			state = "pending"
			lastError = deliveryErr.Error()
			if len(lastError) > 512 {
				lastError = lastError[:512]
			}
		}
		next := now.Add(time.Second * time.Duration(1<<min(item.attempts+1, 12))).UTC().Format(time.RFC3339Nano)
		if _, err = s.db.ExecContext(ctx, "UPDATE learning_feedback_delivery SET state=?,attempts=attempts+1,next_attempt_at=?,last_error=? WHERE observation_id=?", state, next, lastError, item.id); err != nil {
			return err
		}
	}
	return nil
}

func findingIdentity(target LearningResult, feedback Feedback) string {
	key := any([]string{feedback.FeedbackRef, feedback.Dimension})
	if len(target.Artifact) > 0 || target.FragmentHash != "" {
		key = []any{target.Scope, target.Artifact, target.StepKey, target.FragmentHash, target.CompatibilityDigest, feedback.Dimension}
	}
	encoded, _ := json.Marshal(key)
	return fmt.Sprintf("learning-finding:%x", sha256.Sum256(encoded))
}

// ClaimFragmentCohort prevents a caller-supplied key from crossing the live,
// test and replay evidence boundaries even if the caller forges the key digest.
func (s *Store) ClaimFragmentCohort(ctx context.Context, key, provenance string) error {
	if key == "" || len(key) > 512 {
		return errors.New("bounded step key required")
	}
	cohort := "live"
	if provenance == "test" || provenance == "replay" {
		cohort = provenance
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO learning_fragment_cohorts(step_key,cohort) VALUES(?,?)", key, cohort); err != nil {
		return err
	}
	var existing string
	if err := s.db.QueryRowContext(ctx, "SELECT cohort FROM learning_fragment_cohorts WHERE step_key=?", key).Scan(&existing); err != nil {
		return err
	}
	if existing != cohort {
		return errors.New("fragment evidence cohort mismatch")
	}
	return nil
}

func (s *Store) FragmentRejected(ctx context.Context, key, hash string) (bool, error) {
	if len(hash) == 64 {
		hash = "sha256:" + hash
	}
	if key == "" || len(key) > 512 || hash == "" || len(hash) > 128 {
		return false, errors.New("bounded fragment identity required")
	}
	var rejected bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_fragments WHERE step_key=? AND fragment_hash=? AND contradicted_since_edit>0 UNION SELECT 1 FROM learning_results r JOIN learning_feedback f ON f.feedback_ref=r.feedback_ref WHERE json_extract(r.record,'$.step_key')=? AND json_extract(r.record,'$.fragment_hash')=? AND json_extract(f.record,'$.disposition')='contradicted' AND json_extract(f.record,'$.dimension') IN ('execution','verification'))`, key, hash, key, hash).Scan(&rejected)
	return rejected, err
}

func (s *Store) FeedbackDeliveryMetrics(ctx context.Context) (map[string]any, error) {
	var pending, delivered int
	var oldest string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN d.state='pending' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN d.state='delivered' THEN 1 ELSE 0 END),0),COALESCE(MIN(CASE WHEN d.state='pending' THEN f.created_at END),'') FROM learning_feedback_delivery d JOIN learning_feedback f ON f.observation_id=d.observation_id`).Scan(&pending, &delivered, &oldest)
	return map[string]any{"feedback_pending": pending, "feedback_delivered": delivered, "feedback_oldest_pending_at": oldest}, err
}

// RequeueUnpinned adopts a finish contract only for captures admitted before
// Memory was installed. Previously pinned implementations are never replaced.
func (s *Store) RequeueUnpinned(ctx context.Context, digest string) error {
	if digest == "" || len(digest) > 256 {
		return errors.New("bounded finish digest required")
	}
	rows, err := s.db.QueryContext(ctx, "SELECT record FROM learning_tasks WHERE delivery='blocked' AND COALESCE(parent_attempt_id,'')='' AND json_extract(record,'$.finish_digest')='' AND json_extract(record,'$.last_error')='memory_not_installed' LIMIT 100")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var body string
		var r Record
		if err = rows.Scan(&body); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal([]byte(body), &r); err != nil {
			rows.Close()
			return err
		}
		if r.FinishDigest == "" && r.LastError == "memory_not_installed" && r.FinishInputs != nil {
			ids = append(ids, r.AttemptID)
			if len(ids) >= 100 {
				break
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		_, err = s.Update(ctx, id, func(r *Record) error {
			if r.FinishDigest != "" || r.LastError != "memory_not_installed" {
				return nil
			}
			r.FinishDigest = digest
			r.LastError = ""
			r.Delivery = "pending"
			r.NextAttemptAt = time.Now().UTC().Format(time.RFC3339Nano)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
