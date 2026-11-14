package interventions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// SignalClassifier answers the signal question for operator messages.
type SignalClassifier interface {
	ClassifySignals(ctx context.Context, messages []Message) ([]SignalAnswer, error)
}

// SignalJob defaults. The job never scans run_events by time: it works from
// operator_interventions rows, which the intervention job already derived from
// verified operator messages (agent, teammate and injected prompts are not
// recorded there), and reads a run's events only to get a message's preceding
// agent text. Until the back-fill marker for SignalVersion exists it walks the
// pending rows newest first, PassMessages at a time; afterwards it handles only
// interventions recorded within Tail, plus retries.
const (
	DefaultSignalInterval = time.Minute
	DefaultSignalPass     = 128
	DefaultSignalTail     = 6 * time.Hour
)

// SignalJob classifies operator-message signals in the background.
type SignalJob struct {
	Store      *Store
	Events     EventSource
	Classifier SignalClassifier
	// Enabled is the off switch, asked before every pass; nil means on. While it
	// is false nothing is read from run events and nothing is sent to a model.
	Enabled func() bool
	Now     func() time.Time
	Log     *slog.Logger

	Interval     time.Duration
	PassMessages int
	Tail         time.Duration
}

// SignalStats reports one pass.
type SignalStats struct {
	Pending, Classified, Unclassified int
	BackfillDone                      bool
	Disabled                          bool
}

// Start prunes signals of retired versions and runs passes until ctx ends.
func (j *SignalJob) Start(ctx context.Context) {
	if j == nil || j.Store == nil || j.Events == nil || j.Classifier == nil {
		return
	}
	j.defaults()
	go func() {
		if j.enabled() {
			if pruned, err := j.Store.PruneSignalVersions(ctx, SignalVersion); err != nil {
				j.Log.Warn("pruning old operator signals failed", "error", err.Error())
			} else if pruned > 0 {
				j.Log.Info("pruned operator signals of retired signal versions", "rows", pruned)
			}
		}
		ticker := time.NewTicker(j.Interval)
		defer ticker.Stop()
		for {
			stats, err := j.Pass(ctx)
			if err != nil && ctx.Err() == nil {
				j.Log.Warn("operator signal pass degraded", "error", err.Error(), "classified", stats.Classified, "unclassified", stats.Unclassified)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (j *SignalJob) defaults() {
	if j.Now == nil {
		j.Now = time.Now
	}
	if j.Log == nil {
		j.Log = slog.Default()
	}
	if j.Interval <= 0 {
		j.Interval = DefaultSignalInterval
	}
	if j.PassMessages <= 0 {
		j.PassMessages = DefaultSignalPass
	}
	if j.Tail <= 0 {
		j.Tail = DefaultSignalTail
	}
}

func (j *SignalJob) enabled() bool { return j.Enabled == nil || j.Enabled() }

// Pass classifies at most PassMessages pending messages.
func (j *SignalJob) Pass(ctx context.Context) (SignalStats, error) {
	j.defaults()
	var stats SignalStats
	if !j.enabled() {
		stats.Disabled = true
		return stats, nil
	}
	now := j.Now()
	done, err := j.Store.SignalBackfillDone(ctx, SignalVersion)
	if err != nil {
		return stats, err
	}
	var since time.Time
	if done {
		since = now.Add(-j.Tail)
	}
	pending, err := j.Store.PendingSignals(ctx, SignalVersion, since, now, j.PassMessages)
	if err != nil {
		return stats, err
	}
	stats.Pending = len(pending)
	byRun := map[string][]SignalCandidate{}
	var order []string
	for _, c := range pending {
		if _, seen := byRun[c.RunID]; !seen {
			order = append(order, c.RunID)
		}
		byRun[c.RunID] = append(byRun[c.RunID], c)
	}
	runs := passRuns{}
	var firstErr error
	for _, runID := range order {
		if ctx.Err() != nil {
			break
		}
		records, err := j.processRun(ctx, runID, byRun[runID], runs, now)
		if saveErr := j.Store.SaveSignals(ctx, SignalVersion, records, now); saveErr != nil {
			return stats, saveErr
		}
		for _, r := range records {
			if r.Answer.Status == StatusClassified {
				stats.Classified++
			} else {
				stats.Unclassified++
			}
		}
		if err != nil {
			firstErr = err
			break // the gateway is down: the rest wait for the next pass
		}
	}
	if firstErr == nil && !done && len(pending) < j.PassMessages {
		if err := j.Store.MarkSignalBackfilled(ctx, SignalVersion, now); err != nil {
			return stats, err
		}
		stats.BackfillDone = true
	}
	return stats, firstErr
}

func (j *SignalJob) processRun(ctx context.Context, runID string, candidates []SignalCandidate, runs passRuns, now time.Time) ([]SignalRecord, error) {
	retry := func(c SignalCandidate) time.Time {
		return now.Add(min(time.Duration(c.Attempts+1)*retryBackoff, 24*time.Hour))
	}
	id, err := uuid.Parse(runID)
	if err != nil {
		return nil, nil
	}
	var need int64
	for _, c := range candidates {
		need = max(need, c.Sequence)
	}
	run, err := turnsThrough(ctx, j.Events, runs, id, need)
	if err != nil {
		return nil, err
	}
	records := make([]SignalRecord, 0, len(candidates))
	var (
		messages []Message
		pending  []int
	)
	for _, c := range candidates {
		turn, ok := run.turns[c.EventID]
		if !ok || turn.initial {
			records = append(records, SignalRecord{SignalCandidate: c, NextAttemptAt: retry(c),
				Answer: SignalAnswer{Status: StatusUnclassified, Reason: "event_unavailable: the message is no longer a recorded operator turn"}})
			continue
		}
		pending = append(pending, len(records))
		messages = append(messages, Message{EventID: c.EventID, OperatorText: turn.text, PrecedingAgentText: turn.precedingAgentText})
		records = append(records, SignalRecord{SignalCandidate: c})
	}
	if len(messages) == 0 {
		return records, nil
	}
	answers, classifyErr := j.Classifier.ClassifySignals(ctx, messages)
	for k, i := range pending {
		answer := SignalAnswer{Status: StatusUnclassified, Reason: fmt.Sprintf("the classifier answered %d of %d messages", len(answers), len(messages))}
		if k < len(answers) {
			answer = answers[k]
		}
		records[i].Answer = answer
		if answer.Status == StatusUnclassified {
			records[i].NextAttemptAt = retry(records[i].SignalCandidate)
		}
	}
	return records, classifyErr
}
