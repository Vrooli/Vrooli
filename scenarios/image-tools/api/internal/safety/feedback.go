package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// FeedbackLabel is the operator's judgement of one reviewer verdict. It is
// calibration data: recording it never restores a deleted output or changes a
// job's stored verdict.
type FeedbackLabel string

const (
	// FeedbackFalsePositive means the reviewer was wrong: the output is
	// legitimate (a refusal that should not have happened, or a flag on content
	// that is not actually sensitive).
	FeedbackFalsePositive FeedbackLabel = "false_positive"
	// FeedbackConfirmed means the reviewer was right.
	FeedbackConfirmed FeedbackLabel = "confirmed"
	// FeedbackUncertain is kept out of the false-positive rate.
	FeedbackUncertain FeedbackLabel = "uncertain"
)

// FeedbackNoteLimit bounds an operator note; a judgement is a remark, not a
// report.
const FeedbackNoteLimit = 1000

// FeedbackCase is one labelled reviewer verdict, with the evidence snapshotted
// at record time so the corpus survives job retention.
type FeedbackCase struct {
	ID              int64
	JobID           string
	OutputRef       string
	Label           FeedbackLabel
	Action          string
	ReasonCode      string
	SafetyReview    string
	Score           float64
	Operation       string
	ModelID         string
	ProfileName     string
	ReviewerVersion string
	Note            string
	Actor           string
	RecordedAt      time.Time
	// Evidence is the reviewer's whole verdict JSON; Finding is the one image
	// finding when OutputRef names a variation.
	Evidence string
	Finding  string
}

// FeedbackObservation is the job-side input a feedback case is built from. The
// handler fills it from the job record and the engine's safety_* result meta, so
// the storage layer never needs the jobs package.
type FeedbackObservation struct {
	JobID         string
	OutputRef     string
	Operation     string
	ModelID       string
	Action        string
	ReasonCode    string
	SafetyReview  string
	Score         float64
	Evidence      string
	ReviewedRefs  []string
	RefusedRefs   []string
	ShadowRefs    []string
	SensitiveRefs []string
}

// Feedback errors callers translate to a status.
var (
	ErrNoSafetyVerdict   = errors.New("the job has no reviewer refusal or flag to label")
	ErrFeedbackRefAbsent = errors.New("the output ref was not part of the reviewed batch")
	ErrFeedbackLabel     = errors.New("label must be false_positive, confirmed or uncertain")
	ErrFeedbackNote      = fmt.Errorf("note is longer than %d characters", FeedbackNoteLimit)
)

// BuildFeedbackCase validates a judgement and snapshots the reviewer evidence.
// It is pure so the mapping from job meta to a corpus case is unit-testable
// without a database.
func BuildFeedbackCase(obs FeedbackObservation, label FeedbackLabel, note, actor string) (FeedbackCase, error) {
	if !validFeedbackLabel(label) {
		return FeedbackCase{}, ErrFeedbackLabel
	}
	jobID := strings.TrimSpace(obs.JobID)
	if jobID == "" {
		return FeedbackCase{}, errors.New("job id is required")
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > FeedbackNoteLimit {
		return FeedbackCase{}, ErrFeedbackNote
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "api"
	}

	var verdict MinorVerdict
	rawEvidence := strings.TrimSpace(obs.Evidence)
	if rawEvidence == "" {
		return FeedbackCase{}, ErrFeedbackNoEvidence
	}
	if err := json.Unmarshal([]byte(rawEvidence), &verdict); err != nil {
		return FeedbackCase{}, fmt.Errorf("safety: feedback evidence is not a reviewer verdict: %w", err)
	}
	verdict.Normalize()

	outputRef := strings.TrimSpace(obs.OutputRef)
	action := strings.TrimSpace(obs.Action)
	reason := strings.TrimSpace(obs.ReasonCode)
	score := obs.Score
	findingJSON := ""
	finding := MinorImageFinding{}

	if outputRef != "" {
		index := -1
		for i, ref := range obs.ReviewedRefs {
			if ref == outputRef {
				index = i
				break
			}
		}
		if index < 0 {
			if len(obs.ReviewedRefs) == 0 {
				// A legacy job recorded no reviewed-ref list; the aggregate
				// verdict is the only evidence available.
				index = 0
			} else {
				return FeedbackCase{}, ErrFeedbackRefAbsent
			}
		}
		// A per-output judgement must come from that output's finding or from
		// the ref lists. The job-level action describes the whole batch, so it
		// must not be projected onto one variation.
		action, reason, score = "", "", 0
		if index < len(verdict.Images) {
			finding = verdict.Images[index]
			action = string(finding.Action)
			reason = string(finding.ReasonCode)
			score = finding.NSFWScore
			if encoded, err := json.Marshal(finding); err == nil {
				findingJSON = string(encoded)
			}
		}
		if action == "" {
			switch {
			case containsString(obs.RefusedRefs, outputRef), containsString(obs.ShadowRefs, outputRef):
				action = string(ActionRefuse)
			case containsString(obs.SensitiveRefs, outputRef):
				action = string(ActionFlag)
			}
		}
	}
	if action == "" {
		action = actionForReview(obs.SafetyReview)
	}
	// The job-level meta uses compound actions for a partial or shadow refusal
	// (partial_refused, shadow_refused); those still describe a refused output.
	if action != string(ActionRefuse) && action != string(ActionFlag) && action != string(ActionAllow) {
		action = actionForReview(obs.SafetyReview)
	}
	if action != string(ActionRefuse) && action != string(ActionFlag) {
		return FeedbackCase{}, ErrNoSafetyVerdict
	}
	// A job-level case still snapshots the finding that justified the verdict,
	// so the corpus line is replayable without naming a variation.
	if outputRef == "" && findingJSON == "" && len(verdict.Images) > 0 {
		if primary, ok := primaryFinding(verdict.Images, action); ok {
			finding = primary
			if score == 0 {
				score = primary.NSFWScore
			}
			if reason == "" {
				reason = string(primary.ReasonCode)
			}
			if encoded, err := json.Marshal(primary); err == nil {
				findingJSON = string(encoded)
			}
		}
	}
	if reason == "" || strings.Contains(reason, ",") {
		// The job-level record joins every distinct code; a corpus case wants
		// one primary code so the per-reason stats do not grow compound keys.
		if primary := string(verdict.ReasonCode); primary != "" {
			reason = primary
		}
	}
	if score == 0 {
		if finding.NSFWScore > 0 {
			score = finding.NSFWScore
		} else {
			_, score = verdict.Sensitive()
		}
	}

	return FeedbackCase{
		JobID:           jobID,
		OutputRef:       outputRef,
		Label:           label,
		Action:          action,
		ReasonCode:      reason,
		SafetyReview:    strings.TrimSpace(obs.SafetyReview),
		Score:           score,
		Operation:       strings.TrimSpace(obs.Operation),
		ModelID:         strings.TrimSpace(obs.ModelID),
		ProfileName:     strings.TrimSpace(verdict.ProfileName),
		ReviewerVersion: strings.TrimSpace(verdict.ReviewerVersion),
		Note:            note,
		Actor:           actor,
		Evidence:        rawEvidence,
		Finding:         findingJSON,
	}, nil
}

// ErrFeedbackNoEvidence is returned when a job carries no reviewer evidence to
// snapshot. It is separate from ErrNoSafetyVerdict because it means the job
// predates the structured evidence record.
var ErrFeedbackNoEvidence = errors.New("the job has no reviewer evidence to snapshot")

// primaryFinding picks the image finding that best explains a job-level verdict:
// the first refused image for a refusal, the first flagged/sexual image for a
// flag, or the strongest score when neither is recorded.
func primaryFinding(images []MinorImageFinding, action string) (MinorImageFinding, bool) {
	preferRefuse := action == string(ActionRefuse)
	for _, image := range images {
		if preferRefuse && image.Refuse {
			return image, true
		}
		if !preferRefuse && (image.Action == ActionFlag || image.Sexual) {
			return image, true
		}
	}
	var best MinorImageFinding
	found := false
	for _, image := range images {
		if !found || image.NSFWScore > best.NSFWScore {
			best, found = image, true
		}
	}
	return best, found
}

func validFeedbackLabel(label FeedbackLabel) bool {
	switch label {
	case FeedbackFalsePositive, FeedbackConfirmed, FeedbackUncertain:
		return true
	default:
		return false
	}
}

func actionForReview(review string) string {
	switch review {
	case "refused", "partial_refused", "shadow_refused":
		return string(ActionRefuse)
	case "flagged":
		return string(ActionFlag)
	default:
		return ""
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// FeedbackStore persists operator judgements about reviewer verdicts.
type FeedbackStore struct {
	db  SQLExecutor
	now func() time.Time
}

// NewFeedbackStore constructs the store over db.
func NewFeedbackStore(db SQLExecutor) *FeedbackStore {
	return &FeedbackStore{db: db, now: time.Now}
}

// Record stores or replaces the judgement for one job output.
func (s *FeedbackStore) Record(ctx context.Context, item FeedbackCase) (FeedbackCase, error) {
	if s == nil || s.db == nil {
		return FeedbackCase{}, errors.New("safety: feedback store is not configured")
	}
	item.Label = FeedbackLabel(strings.ToLower(strings.TrimSpace(string(item.Label))))
	if !validFeedbackLabel(item.Label) {
		return FeedbackCase{}, ErrFeedbackLabel
	}
	item.JobID = strings.TrimSpace(item.JobID)
	item.OutputRef = strings.TrimSpace(item.OutputRef)
	item.Note = strings.TrimSpace(item.Note)
	if item.JobID == "" {
		return FeedbackCase{}, errors.New("job id is required")
	}
	if len([]rune(item.Note)) > FeedbackNoteLimit {
		return FeedbackCase{}, ErrFeedbackNote
	}
	if strings.TrimSpace(item.Actor) == "" {
		item.Actor = "api"
	}
	item.RecordedAt = s.now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO safety_feedback (job_id, output_ref, label, action, reason_code, safety_review, score,
  operation, model_id, profile_name, reviewer_version, note, actor, evidence, finding, recorded_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(job_id, output_ref) DO UPDATE SET
  label=excluded.label, action=excluded.action, reason_code=excluded.reason_code,
  safety_review=excluded.safety_review, score=excluded.score, operation=excluded.operation,
  model_id=excluded.model_id, profile_name=excluded.profile_name,
  reviewer_version=excluded.reviewer_version, note=excluded.note, actor=excluded.actor,
  evidence=excluded.evidence, finding=excluded.finding, recorded_at=excluded.recorded_at`,
		item.JobID, item.OutputRef, string(item.Label), item.Action, item.ReasonCode, item.SafetyReview,
		item.Score, item.Operation, item.ModelID, item.ProfileName, item.ReviewerVersion, item.Note,
		item.Actor, item.Evidence, item.Finding, item.RecordedAt.Format(time.RFC3339))
	if err != nil {
		return FeedbackCase{}, fmt.Errorf("safety: save feedback: %w", err)
	}
	// An upsert reports no useful last-insert id (it may have updated), so read
	// the row back; the response then always carries the stored id and timestamp.
	stored, found, err := s.Get(ctx, item.JobID, item.OutputRef)
	if err != nil {
		return FeedbackCase{}, err
	}
	if found {
		return stored, nil
	}
	return item, nil
}

// Clear removes one judgement. It reports whether a row existed.
func (s *FeedbackStore) Clear(ctx context.Context, jobID, outputRef string) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("safety: feedback store is not configured")
	}
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM safety_feedback WHERE job_id = ? AND output_ref = ?`,
		strings.TrimSpace(jobID), strings.TrimSpace(outputRef))
	if err != nil {
		return false, fmt.Errorf("safety: clear feedback: %w", err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

const feedbackColumns = `id, job_id, output_ref, label, action, reason_code, safety_review, score,
  operation, model_id, profile_name, reviewer_version, note, actor, evidence, finding, recorded_at`

func scanFeedback(scan func(...any) error) (FeedbackCase, error) {
	var out FeedbackCase
	var label, at string
	if err := scan(&out.ID, &out.JobID, &out.OutputRef, &label, &out.Action, &out.ReasonCode,
		&out.SafetyReview, &out.Score, &out.Operation, &out.ModelID, &out.ProfileName,
		&out.ReviewerVersion, &out.Note, &out.Actor, &out.Evidence, &out.Finding, &at); err != nil {
		return FeedbackCase{}, err
	}
	out.Label = FeedbackLabel(label)
	out.RecordedAt, _ = time.Parse(time.RFC3339, at)
	return out, nil
}

// Get returns one job output's judgement.
func (s *FeedbackStore) Get(ctx context.Context, jobID, outputRef string) (FeedbackCase, bool, error) {
	if s == nil || s.db == nil {
		return FeedbackCase{}, false, nil
	}
	item, err := scanFeedback(s.db.QueryRowContext(ctx,
		`SELECT `+feedbackColumns+` FROM safety_feedback WHERE job_id = ? AND output_ref = ?`,
		strings.TrimSpace(jobID), strings.TrimSpace(outputRef)).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return FeedbackCase{}, false, nil
	}
	return item, err == nil, err
}

// ForJobs returns every judgement for the given jobs, newest first, keyed by
// job id. The job list rides on it, so one query covers a page.
func (s *FeedbackStore) ForJobs(ctx context.Context, jobIDs []string) (map[string][]FeedbackCase, error) {
	out := make(map[string][]FeedbackCase, len(jobIDs))
	if s == nil || s.db == nil || len(jobIDs) == 0 {
		return out, nil
	}
	const batch = 400
	for start := 0; start < len(jobIDs); start += batch {
		end := min(start+batch, len(jobIDs))
		args := make([]any, 0, end-start)
		marks := make([]string, 0, end-start)
		for _, id := range jobIDs[start:end] {
			args = append(args, id)
			marks = append(marks, "?")
		}
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+feedbackColumns+` FROM safety_feedback WHERE job_id IN (`+strings.Join(marks, ",")+`)
			 ORDER BY recorded_at DESC, id DESC`, args...)
		if err != nil {
			return nil, fmt.Errorf("safety: read feedback: %w", err)
		}
		for rows.Next() {
			item, err := scanFeedback(rows.Scan)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("safety: scan feedback: %w", err)
			}
			out[item.JobID] = append(out[item.JobID], item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return out, nil
}

// FeedbackFilters narrows a corpus query. An empty field does not filter.
type FeedbackFilters struct {
	Label      FeedbackLabel
	ReasonCode string
	ModelID    string
	Operation  string
	JobID      string
	Since      time.Time
}

func (f FeedbackFilters) where() (string, []any) {
	var clauses []string
	var args []any
	if f.Label != "" {
		clauses = append(clauses, "label = ?")
		args = append(args, string(f.Label))
	}
	if f.ReasonCode != "" {
		clauses = append(clauses, "reason_code = ?")
		args = append(args, f.ReasonCode)
	}
	if f.ModelID != "" {
		clauses = append(clauses, "model_id = ?")
		args = append(args, f.ModelID)
	}
	if f.Operation != "" {
		clauses = append(clauses, "operation = ?")
		args = append(args, f.Operation)
	}
	if f.JobID != "" {
		clauses = append(clauses, "job_id = ?")
		args = append(args, f.JobID)
	}
	if !f.Since.IsZero() {
		clauses = append(clauses, "recorded_at >= ?")
		args = append(args, f.Since.UTC().Format(time.RFC3339))
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// List returns cases newest first and the total number matching the filters.
func (s *FeedbackStore) List(ctx context.Context, filters FeedbackFilters, limit, offset int) ([]FeedbackCase, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	where, args := filters.where()
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM safety_feedback`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("safety: count feedback: %w", err)
	}
	query := `SELECT ` + feedbackColumns + ` FROM safety_feedback` + where + ` ORDER BY recorded_at DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("safety: list feedback: %w", err)
	}
	defer rows.Close()
	var out []FeedbackCase
	for rows.Next() {
		item, err := scanFeedback(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("safety: scan feedback: %w", err)
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

// FeedbackLabelCount is one label's case count.
type FeedbackLabelCount struct {
	Label FeedbackLabel
	Cases int64
}

// FeedbackBucket is the false-positive record for one reason code or profile.
type FeedbackBucket struct {
	Key               string
	Cases             int64
	FalsePositives    int64
	Confirmed         int64
	FalsePositiveRate float64
}

// FeedbackStats summarises the labelled corpus.
type FeedbackStats struct {
	Total             int64
	FalsePositives    int64
	Confirmed         int64
	Uncertain         int64
	FalsePositiveRate float64
	ByLabel           []FeedbackLabelCount
	ByReason          []FeedbackBucket
	ByProfile         []FeedbackBucket
	Since             time.Time
}

// Stats aggregates the corpus, optionally since a time.
func (s *FeedbackStore) Stats(ctx context.Context, since time.Time) (FeedbackStats, error) {
	if s == nil || s.db == nil {
		return FeedbackStats{Since: since}, nil
	}
	filters := FeedbackFilters{Since: since}
	where, args := filters.where()
	out := FeedbackStats{Since: since}

	rows, err := s.db.QueryContext(ctx,
		`SELECT label, COUNT(*) FROM safety_feedback`+where+` GROUP BY label`, args...)
	if err != nil {
		return out, fmt.Errorf("safety: feedback label stats: %w", err)
	}
	for rows.Next() {
		var label string
		var cases int64
		if err := rows.Scan(&label, &cases); err != nil {
			rows.Close()
			return out, err
		}
		out.ByLabel = append(out.ByLabel, FeedbackLabelCount{Label: FeedbackLabel(label), Cases: cases})
		out.Total += cases
		switch FeedbackLabel(label) {
		case FeedbackFalsePositive:
			out.FalsePositives = cases
		case FeedbackConfirmed:
			out.Confirmed = cases
		case FeedbackUncertain:
			out.Uncertain = cases
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()
	out.FalsePositiveRate = rate(out.FalsePositives, out.Confirmed)

	reason, err := s.bucketStats(ctx, "reason_code", where, args)
	if err != nil {
		return out, err
	}
	out.ByReason = reason
	profile, err := s.bucketStats(ctx, "profile_name", where, args)
	if err != nil {
		return out, err
	}
	out.ByProfile = profile
	return out, nil
}

func (s *FeedbackStore) bucketStats(ctx context.Context, column, where string, args []any) ([]FeedbackBucket, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+column+`, label, COUNT(*) FROM safety_feedback`+where+` GROUP BY `+column+`, label`, args...)
	if err != nil {
		return nil, fmt.Errorf("safety: feedback %s stats: %w", column, err)
	}
	defer rows.Close()
	buckets := map[string]*FeedbackBucket{}
	for rows.Next() {
		var key, label string
		var cases int64
		if err := rows.Scan(&key, &label, &cases); err != nil {
			return nil, err
		}
		bucket := buckets[key]
		if bucket == nil {
			bucket = &FeedbackBucket{Key: key}
			buckets[key] = bucket
		}
		bucket.Cases += cases
		switch FeedbackLabel(label) {
		case FeedbackFalsePositive:
			bucket.FalsePositives = cases
		case FeedbackConfirmed:
			bucket.Confirmed = cases
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]FeedbackBucket, 0, len(buckets))
	for _, bucket := range buckets {
		bucket.FalsePositiveRate = rate(bucket.FalsePositives, bucket.Confirmed)
		out = append(out, *bucket)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cases != out[j].Cases {
			return out[i].Cases > out[j].Cases
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func rate(falsePositives, confirmed int64) float64 {
	decisive := falsePositives + confirmed
	if decisive == 0 {
		return 0
	}
	return float64(falsePositives) / float64(decisive)
}

// corpusLine is the stable JSON Lines record the export emits. Evidence and
// finding stay objects (not quoted strings) so a calibration script can read
// them directly.
type corpusLine struct {
	SchemaVersion int             `json:"schema_version"`
	ID            int64           `json:"id"`
	JobID         string          `json:"job_id"`
	OutputRef     string          `json:"output_ref,omitempty"`
	Label         string          `json:"label"`
	RecordedAt    string          `json:"recorded_at"`
	Note          string          `json:"note,omitempty"`
	Actor         string          `json:"actor,omitempty"`
	Safety        corpusSafety    `json:"safety"`
	Evidence      json.RawMessage `json:"evidence,omitempty"`
	Finding       json.RawMessage `json:"finding,omitempty"`
}

type corpusSafety struct {
	Action          string  `json:"action"`
	ReasonCode      string  `json:"reason_code"`
	SafetyReview    string  `json:"safety_review,omitempty"`
	Score           float64 `json:"score"`
	Operation       string  `json:"operation,omitempty"`
	ModelID         string  `json:"model_id,omitempty"`
	ProfileName     string  `json:"profile_name,omitempty"`
	ReviewerVersion string  `json:"reviewer_version,omitempty"`
}

// Export renders the labelled corpus as JSON Lines. limit defaults to 5000 and
// is capped at 20000.
func (s *FeedbackStore) Export(ctx context.Context, filters FeedbackFilters, limit int) (string, int, error) {
	if limit <= 0 {
		limit = 5000
	}
	if limit > 20000 {
		limit = 20000
	}
	cases, _, err := s.List(ctx, filters, limit, 0)
	if err != nil {
		return "", 0, err
	}
	var b strings.Builder
	for _, item := range cases {
		line := corpusLine{
			SchemaVersion: 1,
			ID:            item.ID,
			JobID:         item.JobID,
			OutputRef:     item.OutputRef,
			Label:         string(item.Label),
			RecordedAt:    item.RecordedAt.UTC().Format(time.RFC3339),
			Note:          item.Note,
			Actor:         item.Actor,
			Safety: corpusSafety{
				Action:          item.Action,
				ReasonCode:      item.ReasonCode,
				SafetyReview:    item.SafetyReview,
				Score:           item.Score,
				Operation:       item.Operation,
				ModelID:         item.ModelID,
				ProfileName:     item.ProfileName,
				ReviewerVersion: item.ReviewerVersion,
			},
		}
		if json.Valid([]byte(item.Evidence)) {
			line.Evidence = json.RawMessage(item.Evidence)
		}
		if json.Valid([]byte(item.Finding)) {
			line.Finding = json.RawMessage(item.Finding)
		}
		encoded, err := json.Marshal(line)
		if err != nil {
			return "", 0, fmt.Errorf("safety: encode corpus case %d: %w", item.ID, err)
		}
		b.Write(encoded)
		b.WriteByte('\n')
	}
	return b.String(), len(cases), nil
}
