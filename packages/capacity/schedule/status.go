package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// StatusOptions tunes a status projection.
type StatusOptions struct {
	// Policy supplies the mode, budgets and windows being reported. A zero
	// policy reports queue state only.
	Policy Policy
	// At overrides the store clock. Zero uses the store clock.
	At time.Time
}

// Status is the operator-facing view of the scheduler. Every field answers a
// question an operator actually asks: is it running, what is waiting, why is it
// waiting, how much budget is left, and what did the last run really cost.
type Status struct {
	Mode      Mode         `json:"mode"`
	At        time.Time    `json:"at"`
	Queue     QueueStatus  `json:"queue"`
	Deferrals []DeferCount `json:"deferrals,omitempty"`
	Budget    BudgetStatus `json:"budget"`
	Window    WindowStatus `json:"window"`
	Idle      IdleStatus   `json:"idle"`
	LastRun   *RunSummary  `json:"lastRun,omitempty"`
	Running   []RunSummary `json:"running,omitempty"`
}

// QueueStatus is the waiting set, sliced the ways operators slice it. Depth
// counts queued + deferred jobs; Running counts leased jobs separately.
type QueueStatus struct {
	Depth   int            `json:"depth"`
	Running int            `json:"running"`
	ByState map[string]int `json:"byState"`
	ByKind  map[string]int `json:"byKind"`
	ByModel map[string]int `json:"byModel"`
	ByOwner map[string]int `json:"byOwner"`
	// OldestDue is the earliest NotBefore among waiting jobs: when the next
	// waiting job becomes eligible.
	OldestDue *time.Time `json:"oldestDue,omitempty"`
}

// DeferCount counts jobs waiting for one reason.
type DeferCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// BudgetStatus is rolling-hour consumption against the configured caps. A zero
// limit means unlimited, and the matching Remaining field is then zero.
type BudgetStatus struct {
	WindowStart         time.Time `json:"windowStart"`
	SecondsUsed         float64   `json:"secondsUsed"`
	SecondsLimit        float64   `json:"secondsLimit"`
	SecondsRemaining    float64   `json:"secondsRemaining"`
	JobsUsed            int       `json:"jobsUsed"`
	JobsLimit           int       `json:"jobsLimit"`
	JobsRemaining       int       `json:"jobsRemaining"`
	ModelLoads          int       `json:"modelLoads"`
	ModelLoadsLimit     int       `json:"modelLoadsLimit"`
	ModelLoadsRemaining int       `json:"modelLoadsRemaining"`
}

// WindowStatus reports whether the schedule window is open and when it next
// opens.
type WindowStatus struct {
	Configured bool       `json:"configured"`
	Open       bool       `json:"open"`
	NextOpen   *time.Time `json:"nextOpen,omitempty"`
}

// IdleStatus reports the host-idle gate.
type IdleStatus struct {
	Required          bool    `json:"required"`
	Observed          bool    `json:"observed"`
	Idle              bool    `json:"idle"`
	Load1             float64 `json:"load1"`
	AvailableRAMBytes uint64  `json:"availableRamBytes"`
	Reason            string  `json:"reason,omitempty"`
}

// RunSummary is one execution record. Estimated and actual are both present so
// an operator can see whether the cost model is drifting.
type RunSummary struct {
	JobID            string     `json:"jobId"`
	Kind             string     `json:"kind"`
	Owner            string     `json:"owner"`
	Model            string     `json:"model,omitempty"`
	StartedAt        time.Time  `json:"startedAt"`
	FinishedAt       *time.Time `json:"finishedAt,omitempty"`
	EstimatedSeconds float64    `json:"estimatedSeconds"`
	ActualSeconds    float64    `json:"actualSeconds"`
	Items            int        `json:"items"`
	Tokens           int        `json:"tokens"`
	Outcome          string     `json:"outcome"`
}

// Status computes the current projection.
func (s *Store) Status(ctx context.Context, opts StatusOptions) (Status, error) {
	policy := opts.Policy
	at := opts.At
	if at.IsZero() {
		at = s.now()
	}
	status := Status{
		Mode: policy.Mode,
		At:   at,
		Queue: QueueStatus{
			ByState: map[string]int{},
			ByKind:  map[string]int{},
			ByModel: map[string]int{},
			ByOwner: map[string]int{},
		},
	}

	rows, err := s.db.QueryContext(ctx, `SELECT state, kind, model_affinity, owner, not_before_at
		FROM schedule_jobs WHERE state IN (?, ?, ?)`,
		string(StateQueued), string(StateDeferred), string(StateRunning))
	if err != nil {
		return Status{}, fmt.Errorf("read scheduler queue: %w", err)
	}
	var oldest time.Time
	for rows.Next() {
		var state, kind, model, owner, notBefore string
		if err := rows.Scan(&state, &kind, &model, &owner, &notBefore); err != nil {
			rows.Close()
			return Status{}, err
		}
		status.Queue.ByState[state]++
		if state == string(StateRunning) {
			status.Queue.Running++
			continue
		}
		status.Queue.Depth++
		status.Queue.ByKind[kind]++
		modelKey := model
		if modelKey == "" {
			modelKey = "(none)"
		}
		status.Queue.ByModel[modelKey]++
		status.Queue.ByOwner[owner]++
		due, err := parseTime(notBefore)
		if err != nil {
			rows.Close()
			return Status{}, err
		}
		if oldest.IsZero() || due.Before(oldest) {
			oldest = due
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Status{}, err
	}
	if !oldest.IsZero() {
		status.Queue.OldestDue = &oldest
	}
	if status.Queue.Running > 0 {
		status.Running, err = s.runningRuns(ctx)
		if err != nil {
			return Status{}, err
		}
	}

	status.Deferrals, err = s.deferCounts(ctx)
	if err != nil {
		return Status{}, err
	}
	status.Budget, err = s.budgetStatus(ctx, policy, at)
	if err != nil {
		return Status{}, err
	}
	status.Window = WindowStatus{
		Configured: len(policy.Windows) > 0,
		Open:       policy.windowsOpen(at),
		NextOpen:   policy.nextWindowOpen(at),
	}
	if status.Window.Open {
		status.Window.NextOpen = nil
	}
	status.LastRun, err = s.lastRun(ctx)
	if err != nil {
		return Status{}, err
	}
	return status, nil
}

func (s *Store) budgetStatus(ctx context.Context, policy Policy, at time.Time) (BudgetStatus, error) {
	snapshot, err := s.budgetSnapshot(ctx, at)
	if err != nil {
		return BudgetStatus{}, err
	}
	status := BudgetStatus{
		WindowStart:     snapshot.From,
		SecondsUsed:     snapshot.Seconds,
		JobsUsed:        snapshot.Jobs,
		ModelLoads:      snapshot.ModelLoads,
		SecondsLimit:    policy.MaxSecondsPerHour,
		JobsLimit:       policy.MaxJobsPerHour,
		ModelLoadsLimit: policy.MaxModelLoadsPerHour,
	}
	if policy.MaxSecondsPerHour > 0 {
		status.SecondsRemaining = policy.MaxSecondsPerHour - snapshot.Seconds
		if status.SecondsRemaining < 0 {
			status.SecondsRemaining = 0
		}
	}
	if policy.MaxJobsPerHour > 0 {
		status.JobsRemaining = policy.MaxJobsPerHour - snapshot.Jobs
		if status.JobsRemaining < 0 {
			status.JobsRemaining = 0
		}
	}
	if policy.MaxModelLoadsPerHour > 0 {
		status.ModelLoadsRemaining = policy.MaxModelLoadsPerHour - snapshot.ModelLoads
		if status.ModelLoadsRemaining < 0 {
			status.ModelLoadsRemaining = 0
		}
	}
	return status, nil
}

func (s *Store) deferCounts(ctx context.Context) ([]DeferCount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT last_reason, COUNT(*) FROM schedule_jobs
		WHERE state = ? GROUP BY last_reason ORDER BY COUNT(*) DESC, last_reason ASC`,
		string(StateDeferred))
	if err != nil {
		return nil, fmt.Errorf("read deferral reasons: %w", err)
	}
	defer rows.Close()
	var counts []DeferCount
	for rows.Next() {
		var count DeferCount
		if err := rows.Scan(&count.Reason, &count.Count); err != nil {
			return nil, err
		}
		if count.Reason == "" {
			count.Reason = "(unspecified)"
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}

func (s *Store) lastRun(ctx context.Context) (*RunSummary, error) {
	runs, err := s.recentRuns(ctx, 20)
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		if run.Outcome != runOutcomeRunning {
			summary := run
			return &summary, nil
		}
	}
	return nil, nil
}

func (s *Store) runningRuns(ctx context.Context) ([]RunSummary, error) {
	runs, err := s.recentRuns(ctx, 50)
	if err != nil {
		return nil, err
	}
	var running []RunSummary
	for _, run := range runs {
		if run.Outcome == runOutcomeRunning {
			running = append(running, run)
		}
	}
	return running, nil
}

func (s *Store) recentRuns(ctx context.Context, limit int) ([]RunSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT job_id, kind, owner, model_affinity, started_at,
		finished_at, estimated_seconds, actual_seconds, items, tokens, outcome
		FROM schedule_runs ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("read scheduler runs: %w", err)
	}
	defer rows.Close()
	var runs []RunSummary
	for rows.Next() {
		var (
			run        RunSummary
			startedAt  string
			finishedAt sql.NullString
		)
		if err := rows.Scan(&run.JobID, &run.Kind, &run.Owner, &run.Model, &startedAt,
			&finishedAt, &run.EstimatedSeconds, &run.ActualSeconds, &run.Items, &run.Tokens, &run.Outcome); err != nil {
			return nil, err
		}
		if run.StartedAt, err = parseTime(startedAt); err != nil {
			return nil, err
		}
		if run.FinishedAt, err = scanOptionalTime(finishedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// SortDeferrals orders counts by descending count, then reason, for stable
// reporting in callers that merge sources.
func SortDeferrals(counts []DeferCount) {
	sort.SliceStable(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Reason < counts[j].Reason
	})
}
