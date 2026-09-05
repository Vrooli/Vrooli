package operations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/execution"
)

// ActivityLister is the seam into the agentactivity ledger. It is a
// trimmed slice of *agentactivity.Service so tests can supply a fake list
// without standing up the full service tree.
type ActivityLister interface {
	List(ctx context.Context, filters agentactivity.ListFilters) ([]agentactivity.Record, error)
}

// GovernanceReader exposes the per-lane caps + queue ceiling used to size
// the LaneStatus / QueueStatus payloads. The execution.Service satisfies
// this interface via GovernanceStatus(); tests pass a fixed-shape stub.
type GovernanceReader interface {
	GovernanceStatus() (*execution.GovernanceStatusResponse, error)
}

// AggregatorConfig wires the three readers Aggregator joins. Now is a
// time.Time seam so tests can pin GeneratedAt + window math against a
// fixed clock.
type AggregatorConfig struct {
	Activities ActivityLister
	Governance GovernanceReader
	Now        func() time.Time
}

// Aggregator joins the activity ledger, the round projection, and the
// governance lane caps into a single OperationsView. It owns no state of
// its own — all reads are "now"-bounded and the aggregator returns a
// fully-materialized view.
type Aggregator struct {
	cfg AggregatorConfig
}

// NewAggregator builds an Aggregator. Activities and Governance are
// required; Rounds may be nil (older test wirings without operating-mode
// service get an empty milestone map). Now defaults to time.Now if unset.
func NewAggregator(cfg AggregatorConfig) (*Aggregator, error) {
	if cfg.Activities == nil {
		return nil, fmt.Errorf("operations: AggregatorConfig.Activities is required")
	}
	if cfg.Governance == nil {
		return nil, fmt.Errorf("operations: AggregatorConfig.Governance is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Aggregator{cfg: cfg}, nil
}

// Aggregate produces the OperationsView for the given filter set.
//
// The filter is applied in two passes:
//  1. Window + ActiveOnly are pushed down to ActivityLister so the wire
//     payload from the activity store is bounded — no in-handler post-filter.
//  2. Status / Lane / Mode / OwnerType / Q are applied here in a single
//     pass over the bounded slice. None of these are persistence axes,
//     so doing them in-package keeps ActivityLister simple.
//
// Lane utilization comes from agentactivity.LaneActiveCounts over the
// fetched record set (live count); capacity / queue come from
// GovernanceStatus. The two views agree because GovernanceStatus already
// reads the same activity ledger via SetActivityLaneReader.
func (a *Aggregator) Aggregate(ctx context.Context, f Filters) (*OperationsView, error) {
	window := f.Window
	if window <= 0 {
		window = DefaultWindow
	}
	if window < MinWindow {
		window = MinWindow
	}
	if window > MaxWindow {
		window = MaxWindow
	}

	now := a.cfg.Now().UTC()
	since := now.Add(-window)

	records, err := a.cfg.Activities.List(ctx, agentactivity.ListFilters{
		ActiveOrFinishedSince: since,
	})
	if err != nil {
		return nil, fmt.Errorf("operations: list activities: %w", err)
	}

	gov, err := a.cfg.Governance.GovernanceStatus()
	if err != nil {
		return nil, fmt.Errorf("operations: governance status: %w", err)
	}

	active := make([]ActivityRow, 0, len(records))
	finished := make([]ActivityRow, 0, len(records))
	for _, rec := range records {
		row := buildRow(rec, nil, now)
		if !matchesFilter(row, f) {
			continue
		}
		if IsActiveStatus(row.Status) {
			active = append(active, row)
		} else {
			finished = append(finished, row)
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].RequestedAt > active[j].RequestedAt
	})
	sort.SliceStable(finished, func(i, j int) bool {
		return finished[i].FinishedAt > finished[j].FinishedAt
	})

	lanes := buildLaneStatuses(records, gov)
	queue := QueueStatus{
		Depth:    gov.QueueDepth,
		MaxDepth: gov.MaxQueueDepth,
	}

	return &OperationsView{
		Lanes:            lanes,
		Queue:            queue,
		Activities:       active,
		RecentlyFinished: finished,
		GeneratedAt:      now,
		WindowSeconds:    int(window.Seconds()),
	}, nil
}

// indexRoundsByRunID walks the active-rounds map and returns a runID →
// (milestoneName, round summary) lookup. RunID is stable across the
// activity record and the round payload, so this is the canonical join
// key — milestone name is not present on activity records.
type roundIndexEntry struct {
	milestoneName string
	round         struct {
		Mode, Phase string
		Round       int
	}
}

func buildRow(rec agentactivity.Record, roundByRunID map[string]roundIndexEntry, now time.Time) ActivityRow {
	row := ActivityRow{
		ActivityID:      rec.ActivityID,
		RunID:           rec.RunID,
		OwnerType:       string(rec.OwnerType),
		OwnerKind:       rec.OwnerKind,
		OwnerName:       rec.OwnerName,
		OwnerTitle:      rec.OwnerTitle,
		Purpose:         string(rec.Purpose),
		PhaseKind:       rec.PhaseKind,
		Status:          string(rec.Status),
		RequestedAt:     rec.RequestedAt,
		StartedAt:       rec.StartedAt,
		FinishedAt:      rec.FinishedAt,
		FailureReason:   rec.FailureReason,
		RequestedBy:     rec.RequestedBy,
		InteractionType: string(rec.InteractionType),
	}

	if lane, err := agentactivity.LaneOf(rec.Purpose, rec.PhaseKind); err == nil {
		row.Lane = string(lane)
	}

	if rec.OwnerType == agentactivity.OwnerMilestone {
		row.MilestoneName = rec.OwnerName
	}

	if entry, ok := roundByRunID[strings.TrimSpace(rec.RunID)]; ok {
		row.Mode = entry.round.Mode
		row.Phase = entry.round.Phase
		row.Round = entry.round.Round
		if row.MilestoneName == "" {
			row.MilestoneName = entry.milestoneName
		}
	}

	row.RuntimeSeconds = computeRuntimeSeconds(rec, now)
	return row
}

// computeRuntimeSeconds returns the elapsed-time the row should display.
// FinishedAt wins whenever it is set — even on otherwise-active statuses
// like needs_review, the agent has stopped working and continuing to count
// wall-clock until the operator decides would be misleading on the
// dashboard. When FinishedAt is empty and the record is still active,
// runtime counts up to now. Returns 0 when timestamps cannot be parsed —
// that's a UI-display concern, not an aggregation failure.
func computeRuntimeSeconds(rec agentactivity.Record, now time.Time) int64 {
	start := strings.TrimSpace(rec.StartedAt)
	if start == "" {
		start = strings.TrimSpace(rec.RequestedAt)
	}
	if start == "" {
		return 0
	}
	startT, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return 0
	}
	end := now
	if finished := strings.TrimSpace(rec.FinishedAt); finished != "" {
		if t, err := time.Parse(time.RFC3339, finished); err == nil {
			end = t
		}
	}
	delta := end.Sub(startT)
	if delta < 0 {
		return 0
	}
	return int64(delta.Seconds())
}

// matchesFilter applies the in-package post-filter. Window is already
// applied by the activity store; everything here is a wire-shape filter.
// Lower-cased compares mirror the polling.go matching style.
func matchesFilter(row ActivityRow, f Filters) bool {
	if len(f.Statuses) > 0 && !containsLower(f.Statuses, row.Status) {
		return false
	}
	if len(f.Lanes) > 0 && !containsLower(f.Lanes, row.Lane) {
		return false
	}
	if len(f.Modes) > 0 && !containsLower(f.Modes, row.Mode) {
		return false
	}
	if len(f.OwnerTypes) > 0 && !containsLower(f.OwnerTypes, row.OwnerType) {
		return false
	}
	if q := strings.TrimSpace(strings.ToLower(f.Q)); q != "" {
		if !strings.Contains(strings.ToLower(row.OwnerTitle), q) &&
			!strings.Contains(strings.ToLower(row.OwnerName), q) &&
			!strings.Contains(strings.ToLower(row.RunID), q) {
			return false
		}
	}
	return true
}

func containsLower(values []string, value string) bool {
	target := strings.ToLower(strings.TrimSpace(value))
	for _, v := range values {
		if strings.ToLower(strings.TrimSpace(v)) == target {
			return true
		}
	}
	return false
}

// buildLaneStatuses returns a length-4 slice in canonical lane order. It
// computes Active live from the record set (so the same audited time
// window powers both the bars and the rows) and reads Capacity / Queue
// from GovernanceStatus, which the wiring layer already feeds with the
// same activity-lane reader.
func buildLaneStatuses(records []agentactivity.Record, gov *execution.GovernanceStatusResponse) []LaneStatus {
	counts := agentactivity.LaneActiveCounts(records)

	capByLane := map[string]int{}
	queueByLane := map[string]int{}
	if gov != nil {
		for _, l := range gov.Lanes {
			capByLane[l.Lane] = l.Capacity
			queueByLane[l.Lane] = l.Queue
		}
	}

	out := make([]LaneStatus, 0, 4)
	for _, lane := range agentactivity.Lanes() {
		key := string(lane)
		out = append(out, LaneStatus{
			Lane:     key,
			Active:   counts[lane],
			Capacity: capByLane[key],
			Queue:    queueByLane[key],
		})
	}
	return out
}
