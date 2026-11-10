// Package measures records per-operation runtime observability (IMG-P0-012):
// op latency (p50/p95), throughput, queue-wait, and terminal-state mix. It is a
// thin SQLite-backed recorder fed by the durable job Manager's OnComplete hook,
// kept decoupled from internal/jobs (the Manager calls a plain func, not this
// package) so the job system stays generic.
//
// Per-model latency and fallback-tier usage need facts only the AI runner knows
// (the resolved model id + provider tier); those are recorded via Record with an
// explicit Sample. The job-driven path (Observe) captures the op-level latency +
// queue-wait + outcome that every job carries for free.
package measures

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SQLExecutor is the narrow DB surface the recorder needs. Both *sql.DB (tests)
// and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Sample is one recorded operation execution.
type Sample struct {
	Operation    string
	ModelID      string // "" when not model-backed (deterministic ops)
	Tier         string // provider tier (local-gpu/local-cpu/byok); "" when n/a
	State        string // terminal state: succeeded/failed/canceled
	DurationMS   int64  // run wall-clock (StartedAt → FinishedAt)
	QueueWaitMS  int64  // CreatedAt → StartedAt
	FallbackUsed bool   // true when the chosen tier was not the preferred local-gpu
}

// JobTrace is one terminal job trace row. It is intentionally denormalized for
// operator/debug reads: a single row answers what ran, which model/backend/tier
// was selected, and how much time was spent queued vs running.
type JobTrace struct {
	JobID       string
	Operation   string
	ModelID     string
	Backend     string
	Tier        string
	Lane        string
	State       string
	DurationMS  int64
	QueueWaitMS int64
	ResultRef   string
	Error       string
}

// ModelProfile is an owner-collected cold, warm, or peak-memory observation.
// Unavailable profiles are retained with an explanation, so consumers can
// distinguish an unsupported measurement from missing recorder data.
type ModelProfile struct {
	Operation   string
	ModelID     string
	Profile     string // cold, warm, peak-memory, or worker-residency-peak-memory
	Status      string // measured, unavailable, or failed
	DurationMS  int64
	PeakMemoryB int64
	Device      string
	Evidence    string
}

// ModelProfileStat summarizes owner-collected observations for one profile.
type ModelProfileStat struct {
	Operation      string `json:"operation"`
	ModelID        string `json:"model_id"`
	Profile        string `json:"profile"`
	Status         string `json:"status"`
	Samples        int    `json:"samples"`
	LatencyP50MS   int64  `json:"latency_p50_ms"`
	LatencyP95MS   int64  `json:"latency_p95_ms"`
	PeakMemoryB    int64  `json:"peak_memory_bytes"`
	Device         string `json:"device"`
	LastEvidence   string `json:"last_evidence"`
	LastRecordedAt string `json:"last_recorded_at"`
}

// Recorder persists samples and answers aggregate queries.
type Recorder struct {
	db  SQLExecutor
	now func() time.Time
}

// NewRecorder constructs a Recorder over db.
func NewRecorder(db SQLExecutor) *Recorder { return &Recorder{db: db, now: time.Now} }

const insertSampleSQL = `
INSERT INTO op_measure (operation, model_id, tier, state, duration_ms, queue_wait_ms, fallback_used, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

// Record persists one sample.
func (r *Recorder) Record(ctx context.Context, s Sample) error {
	if s.Operation == "" {
		return fmt.Errorf("measures: operation is required")
	}
	_, err := r.db.ExecContext(ctx, insertSampleSQL,
		s.Operation, s.ModelID, s.Tier, s.State, s.DurationMS, s.QueueWaitMS,
		boolToInt(s.FallbackUsed), r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("measures: record %q: %w", s.Operation, err)
	}
	return nil
}

// RecordModelProfile stores one cold/warm/memory observation, including honest
// unavailable results. Measured numeric values are never inferred from absent
// data.
func (r *Recorder) RecordModelProfile(ctx context.Context, p ModelProfile) error {
	if p.Operation == "" || p.ModelID == "" {
		return fmt.Errorf("measures: profile operation and model id are required")
	}
	switch p.Profile {
	case "cold", "warm", "peak-memory", "worker-residency-peak-memory":
	default:
		return fmt.Errorf("measures: unsupported model profile %q", p.Profile)
	}
	switch p.Status {
	case "measured", "unavailable", "failed":
	default:
		return fmt.Errorf("measures: unsupported profile status %q", p.Status)
	}
	if p.Status != "measured" && (p.DurationMS != 0 || p.PeakMemoryB != 0) {
		return fmt.Errorf("measures: non-measured profile cannot contain numeric results")
	}
	if p.Status == "measured" {
		if (p.Profile == "cold" || p.Profile == "warm") && p.DurationMS <= 0 {
			return fmt.Errorf("measures: measured %s profile requires positive duration", p.Profile)
		}
		if (p.Profile == "peak-memory" || p.Profile == "worker-residency-peak-memory") && p.PeakMemoryB <= 0 {
			return fmt.Errorf("measures: measured %s profile requires positive memory", p.Profile)
		}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO model_profile (operation, model_id, profile, status, duration_ms, peak_memory_b, device, evidence, recorded_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Operation, p.ModelID, p.Profile, p.Status, p.DurationMS, p.PeakMemoryB, p.Device, p.Evidence, r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("measures: record %s profile for %q: %w", p.Profile, p.ModelID, err)
	}
	return nil
}

// RecordModelJobProfiles derives only facts available from a successful durable
// model job. Warm/cold classification requires explicit worker-residency
// metadata; absent or malformed metadata never becomes an inferred profile.
// Peak process memory is measured only when the provider supplies an explicit
// per-job child-process RSS value and its source. Persistent-worker high-water
// observations use a separately named profile so they cannot be confused with
// isolated per-job peak RSS.
func (r *Recorder) RecordModelJobProfiles(ctx context.Context, sample Sample, jobID string, meta map[string]string) error {
	if sample.ModelID == "" || sample.Operation == "" {
		return nil
	}
	reused, ok := meta["warm_reused"]
	profile := ""
	if ok {
		switch reused {
		case "true":
			profile = "warm"
		case "false":
			profile = "cold"
		default:
			return fmt.Errorf("measures: job %q has malformed warm_reused metadata %q", jobID, reused)
		}
	}

	peakRSSBytes, peakRSSMeasured, peakRSSEvidence, err := processPeakRSSMetadata(meta)
	if err != nil {
		return fmt.Errorf("measures: job %q process peak RSS metadata: %w", jobID, err)
	}
	workerRSSBytes, workerRSSMeasured, workerRSSEvidence, workerRSSPresent, err := workerResidencyPeakRSSMetadata(meta, sample.ModelID, jobID)
	if err != nil {
		return fmt.Errorf("measures: job %q worker residency peak RSS metadata: %w", jobID, err)
	}
	evidencePrefix := "durable job " + jobID
	if profile != "" {
		evidencePrefix += "; residency classification from warm_reused=" + reused
	}
	status := "measured"
	duration := sample.DurationMS
	if sample.State != "succeeded" {
		status = "failed"
		duration = 0
	} else if duration <= 0 {
		status = "unavailable"
		duration = 0
	}
	var errs []error
	if profile != "" {
		if err := r.RecordModelProfile(ctx, ModelProfile{
			Operation: sample.Operation, ModelID: sample.ModelID, Profile: profile,
			Status: status, DurationMS: duration, Device: sample.Tier,
			Evidence: evidencePrefix + "; duration from durable job timestamps",
		}); err != nil {
			errs = append(errs, fmt.Errorf("record %s profile: %w", profile, err))
		}
	}
	peakStatus, peakMemory := "unavailable", int64(0)
	if sample.State != "succeeded" {
		peakStatus = "failed"
	} else if peakRSSMeasured {
		peakStatus, peakMemory = "measured", peakRSSBytes
	}
	if err := r.RecordModelProfile(ctx, ModelProfile{
		Operation: sample.Operation, ModelID: sample.ModelID, Profile: "peak-memory",
		Status: peakStatus, PeakMemoryB: peakMemory, Device: sample.Tier,
		Evidence: evidencePrefix + "; " + peakRSSEvidence,
	}); err != nil {
		errs = append(errs, fmt.Errorf("record peak-memory profile: %w", err))
	}
	if workerRSSPresent {
		workerStatus, workerMemory := "unavailable", int64(0)
		if sample.State != "succeeded" {
			workerStatus = "failed"
		} else if workerRSSMeasured {
			workerStatus, workerMemory = "measured", workerRSSBytes
		}
		if err := r.RecordModelProfile(ctx, ModelProfile{
			Operation: sample.Operation, ModelID: sample.ModelID, Profile: "worker-residency-peak-memory",
			Status: workerStatus, PeakMemoryB: workerMemory, Device: sample.Tier,
			Evidence: evidencePrefix + "; " + workerRSSEvidence,
		}); err != nil {
			errs = append(errs, fmt.Errorf("record worker-residency-peak-memory profile: %w", err))
		}
	}
	return errors.Join(errs...)
}

const (
	processPeakRSSBytesMetadataKey  = "process_peak_rss_bytes"
	processPeakRSSSourceMetadataKey = "process_peak_rss_source"
	processPeakRSSStatusMetadataKey = "process_peak_rss_status"
	processPeakRSSReasonMetadataKey = "process_peak_rss_reason"
	processPeakRSSWait4Source       = "child_wait4_rusage_maxrss"
)

func processPeakRSSMetadata(meta map[string]string) (int64, bool, string, error) {
	bytesText, hasBytes := meta[processPeakRSSBytesMetadataKey]
	source, hasSource := meta[processPeakRSSSourceMetadataKey]
	status, hasStatus := meta[processPeakRSSStatusMetadataKey]
	reason, hasReason := meta[processPeakRSSReasonMetadataKey]
	if !hasBytes && !hasSource && !hasStatus && !hasReason {
		return 0, false, "per-job process peak RSS is unavailable: provider supplied no attributable child-process measurement", nil
	}
	if hasStatus && status == "unavailable" {
		if hasBytes || hasSource || !hasReason || strings.TrimSpace(reason) == "" {
			return 0, false, "", fmt.Errorf("unavailable RSS metadata requires a reason and cannot include a measurement")
		}
		return 0, false, "per-job process peak RSS is unavailable: " + strings.TrimSpace(reason), nil
	}
	if hasStatus || hasReason || !hasBytes || !hasSource {
		return 0, false, "", fmt.Errorf("measured RSS metadata requires only %s and %s", processPeakRSSBytesMetadataKey, processPeakRSSSourceMetadataKey)
	}
	if source != processPeakRSSWait4Source {
		return 0, false, "", fmt.Errorf("unsupported measurement source %q", source)
	}
	bytes, err := strconv.ParseInt(bytesText, 10, 64)
	if err != nil || bytes <= 0 {
		return 0, false, "", fmt.Errorf("%s must be a positive base-10 byte count", processPeakRSSBytesMetadataKey)
	}
	return bytes, true, "measured per-job child-process peak RSS from " + source, nil
}

const (
	workerResidencyIDKey       = "worker_residency_id"
	workerResidencyModelKey    = "worker_residency_model"
	workerResidencyPIDKey      = "worker_residency_pid"
	workerResidencyBytesKey    = "worker_residency_peak_rss_bytes"
	workerResidencySourceKey   = "worker_residency_peak_rss_source"
	workerResidencyStatusKey   = "worker_residency_peak_rss_status"
	workerResidencyReasonKey   = "worker_residency_peak_rss_reason"
	workerResidencyLinuxSource = "linux_proc_status_vm_hwm_residency"
)

func workerResidencyPeakRSSMetadata(meta map[string]string, modelID, jobID string) (int64, bool, string, bool, error) {
	id, hasID := meta[workerResidencyIDKey]
	model, hasModel := meta[workerResidencyModelKey]
	pidText, hasPID := meta[workerResidencyPIDKey]
	bytesText, hasBytes := meta[workerResidencyBytesKey]
	source, hasSource := meta[workerResidencySourceKey]
	status, hasStatus := meta[workerResidencyStatusKey]
	reason, hasReason := meta[workerResidencyReasonKey]
	if !hasID && !hasModel && !hasPID && !hasBytes && !hasSource && !hasStatus && !hasReason {
		return 0, false, "", false, nil
	}
	if !hasID || strings.TrimSpace(id) == "" || !hasModel || model != modelID {
		return 0, false, "", true, fmt.Errorf("worker RSS metadata requires a residency id and matching model id")
	}
	warmReused := meta["warm_reused"]
	if warmReused != "true" && warmReused != "false" {
		return 0, false, "", true, fmt.Errorf("worker RSS metadata requires explicit warm_reused classification")
	}
	if warmReused == "false" && id != "image-tools-worker-residency:"+jobID {
		return 0, false, "", true, fmt.Errorf("cold worker residency id must be tied to its first durable job")
	}
	if status == "unavailable" {
		if hasBytes || hasSource || !hasReason || strings.TrimSpace(reason) == "" {
			return 0, false, "", true, fmt.Errorf("unavailable worker RSS metadata requires a reason and cannot contain a measurement")
		}
		if hasPID {
			pid, err := strconv.Atoi(pidText)
			if err != nil || pid < 0 {
				return 0, false, "", true, fmt.Errorf("%s must be a non-negative process id", workerResidencyPIDKey)
			}
		}
		return 0, false, fmt.Sprintf("resident worker high-water RSS unavailable for model %s, residency %s: %s; not isolated per-job RSS", modelID, id, strings.TrimSpace(reason)), true, nil
	}
	if status != "measured" || hasReason || !hasPID || !hasBytes || !hasSource {
		return 0, false, "", true, fmt.Errorf("measured worker RSS metadata requires status, PID, bytes, source, and no reason")
	}
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		return 0, false, "", true, fmt.Errorf("%s must be a positive process id", workerResidencyPIDKey)
	}
	if source != workerResidencyLinuxSource {
		return 0, false, "", true, fmt.Errorf("unsupported worker residency measurement source %q", source)
	}
	bytes, err := strconv.ParseInt(bytesText, 10, 64)
	if err != nil || bytes <= 0 {
		return 0, false, "", true, fmt.Errorf("%s must be a positive base-10 byte count", workerResidencyBytesKey)
	}
	evidence := fmt.Sprintf("resident worker process high-water RSS from %s for model %s, residency %s, PID %d; not isolated per-job RSS", source, modelID, id, pid)
	return bytes, true, evidence, true, nil
}

// ModelProfiles returns model-level summaries, optionally filtered by model
// and operation. The latest status/evidence are paired with latency percentiles
// and maximum observed peak memory from measured rows.
func (r *Recorder) ModelProfiles(ctx context.Context, modelID, operation string) ([]ModelProfileStat, error) {
	query := `SELECT operation, model_id, profile, status, duration_ms, peak_memory_b, device, evidence, recorded_at FROM model_profile`
	where := []string{}
	args := []any{}
	if modelID != "" {
		where = append(where, "model_id = ?")
		args = append(args, modelID)
	}
	if operation != "" {
		where = append(where, "operation = ?")
		args = append(args, operation)
	}
	if len(where) != 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY operation, model_id, profile, recorded_at"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("measures: query model profiles: %w", err)
	}
	defer rows.Close()
	type aggregate struct {
		stat      ModelProfileStat
		durations []int64
	}
	byKey := map[string]*aggregate{}
	order := []string{}
	for rows.Next() {
		var op, id, profile, status, device, evidence, recorded string
		var duration, memory int64
		if err := rows.Scan(&op, &id, &profile, &status, &duration, &memory, &device, &evidence, &recorded); err != nil {
			return nil, fmt.Errorf("measures: scan model profile: %w", err)
		}
		key := op + "\x00" + id + "\x00" + profile
		a := byKey[key]
		if a == nil {
			a = &aggregate{stat: ModelProfileStat{Operation: op, ModelID: id, Profile: profile}}
			byKey[key] = a
			order = append(order, key)
		}
		if status == "measured" {
			a.stat.Samples++
			if duration > 0 {
				a.durations = append(a.durations, duration)
			}
			if memory > a.stat.PeakMemoryB {
				a.stat.PeakMemoryB = memory
			}
		}
		a.stat.Status, a.stat.Device, a.stat.LastEvidence, a.stat.LastRecordedAt = status, device, evidence, recorded
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("measures: iterate model profiles: %w", err)
	}
	out := make([]ModelProfileStat, 0, len(order))
	for _, key := range order {
		a := byKey[key]
		a.stat.LatencyP50MS = percentile(a.durations, 50)
		a.stat.LatencyP95MS = percentile(a.durations, 95)
		out = append(out, a.stat)
	}
	return out, nil
}

const insertJobTraceSQL = `
INSERT INTO job_trace (job_id, operation, model_id, backend, tier, lane, state, duration_ms, queue_wait_ms, result_ref, error, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(job_id) DO UPDATE SET
  operation=excluded.operation,
  model_id=excluded.model_id,
  backend=excluded.backend,
  tier=excluded.tier,
  lane=excluded.lane,
  state=excluded.state,
  duration_ms=excluded.duration_ms,
  queue_wait_ms=excluded.queue_wait_ms,
  result_ref=excluded.result_ref,
  error=excluded.error,
  recorded_at=excluded.recorded_at`

// RecordJobTrace persists one terminal job trace. It upserts by job id so replay
// or recovery paths can safely re-observe a final job without duplicating rows.
func (r *Recorder) RecordJobTrace(ctx context.Context, tr JobTrace) error {
	if tr.JobID == "" {
		return fmt.Errorf("measures: job trace id is required")
	}
	if tr.Operation == "" {
		return fmt.Errorf("measures: job trace operation is required")
	}
	_, err := r.db.ExecContext(ctx, insertJobTraceSQL,
		tr.JobID, tr.Operation, tr.ModelID, tr.Backend, tr.Tier, tr.Lane, tr.State,
		tr.DurationMS, tr.QueueWaitMS, tr.ResultRef, tr.Error, r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("measures: record job trace %q: %w", tr.JobID, err)
	}
	return nil
}

// JobLike is the subset of a finalized job the Observe path reads. internal/jobs'
// Job satisfies it structurally via ObserveJob in the wiring layer.
type JobLike struct {
	Operation  string
	State      string
	DurationMS int64
	QueueMS    int64
}

// Observe records the op-level facts of a finalized job (no model/tier). Use it
// from the Manager's OnComplete hook.
func (r *Recorder) Observe(ctx context.Context, j JobLike) error {
	return r.Record(ctx, Sample{
		Operation:   j.Operation,
		State:       j.State,
		DurationMS:  j.DurationMS,
		QueueWaitMS: j.QueueMS,
	})
}

// JobTraceByID returns the structured trace for one finalized job.
func (r *Recorder) JobTraceByID(ctx context.Context, jobID string) (JobTrace, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT job_id, operation, model_id, backend, tier, lane, state, duration_ms, queue_wait_ms, result_ref, error FROM job_trace WHERE job_id = ?`, jobID)
	if err != nil {
		return JobTrace{}, fmt.Errorf("measures: query job trace %q: %w", jobID, err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return JobTrace{}, fmt.Errorf("measures: iterate job trace %q: %w", jobID, err)
		}
		return JobTrace{}, sql.ErrNoRows
	}
	var tr JobTrace
	if err := rows.Scan(&tr.JobID, &tr.Operation, &tr.ModelID, &tr.Backend, &tr.Tier, &tr.Lane, &tr.State, &tr.DurationMS, &tr.QueueWaitMS, &tr.ResultRef, &tr.Error); err != nil {
		return JobTrace{}, fmt.Errorf("measures: scan job trace %q: %w", jobID, err)
	}
	return tr, rows.Err()
}

// OpStat is the aggregated measure for one operation.
type OpStat struct {
	Operation     string
	Count         int
	Succeeded     int
	Failed        int
	Canceled      int
	LatencyP50MS  int64
	LatencyP95MS  int64
	QueueWaitP50  int64
	FallbackCount int
}

// Stats returns aggregated measures per operation (optionally filtered to one).
func (r *Recorder) Stats(ctx context.Context, operation string) ([]OpStat, error) {
	query := "SELECT operation, state, duration_ms, queue_wait_ms, fallback_used FROM op_measure"
	var args []any
	if operation != "" {
		query += " WHERE operation = ?"
		args = append(args, operation)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("measures: query stats: %w", err)
	}
	defer rows.Close()

	type acc struct {
		durations []int64
		queues    []int64
		stat      OpStat
	}
	byOp := map[string]*acc{}
	order := []string{}
	for rows.Next() {
		var (
			op, state string
			dur, q    int64
			fb        int
		)
		if err := rows.Scan(&op, &state, &dur, &q, &fb); err != nil {
			return nil, fmt.Errorf("measures: scan stat: %w", err)
		}
		a := byOp[op]
		if a == nil {
			a = &acc{stat: OpStat{Operation: op}}
			byOp[op] = a
			order = append(order, op)
		}
		a.stat.Count++
		switch state {
		case "succeeded":
			a.stat.Succeeded++
		case "failed":
			a.stat.Failed++
		case "canceled":
			a.stat.Canceled++
		}
		if fb != 0 {
			a.stat.FallbackCount++
		}
		a.durations = append(a.durations, dur)
		a.queues = append(a.queues, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("measures: iterate stats: %w", err)
	}

	out := make([]OpStat, 0, len(order))
	for _, op := range order {
		a := byOp[op]
		a.stat.LatencyP50MS = percentile(a.durations, 50)
		a.stat.LatencyP95MS = percentile(a.durations, 95)
		a.stat.QueueWaitP50 = percentile(a.queues, 50)
		out = append(out, a.stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Operation < out[j].Operation })
	return out, nil
}

// percentile returns the p-th percentile (nearest-rank) of vs, or 0 when empty.
func percentile(vs []int64, p int) int64 {
	if len(vs) == 0 {
		return 0
	}
	s := append([]int64(nil), vs...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	rank := (p*len(s) + 99) / 100 // ceil(p/100 * n)
	if rank < 1 {
		rank = 1
	}
	if rank > len(s) {
		rank = len(s)
	}
	return s[rank-1]
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
