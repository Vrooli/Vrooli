package interventions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// EventSource loads a run's events in sequence order.
type EventSource interface {
	Get(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error)
}

// MessageClassifier answers the category of operator messages.
type MessageClassifier interface {
	Classify(ctx context.Context, messages []Message) ([]Answer, error)
}

// Job defaults: classify new operator messages every minute, back-fill the
// last 30 days in six-hour slices, and bound each pass in time and messages so
// the walk never starves the API. The job runs while ProvenGate is empty on
// purpose: it collects the classifications the accuracy gate is measured on
// and that are shown per category once the operator admits one. A pass scans
// at most DefaultPassMessages user-role messages, recorded ones included
// (their payload unread), so it sends at most that many to the classifier, and
// starts no new run after DefaultPassBudget: the overrun is at most the one
// classifier call in flight.
const (
	DefaultInterval     = time.Minute
	DefaultWindow       = 30 * 24 * time.Hour
	DefaultSlice        = 6 * time.Hour
	DefaultPageSize     = 200
	DefaultPassMessages = 128
	DefaultPassBudget   = 45 * time.Second
	// DefaultRewalk is the rest between full walks of the window. A walk picks
	// up messages imported late with old timestamps and retries unclassified
	// ones whose attempt is due.
	DefaultRewalk = time.Hour
	// retryBackoff is the wait before an unclassified message is attempted
	// again, per previous attempt, capped at a day.
	retryBackoff = time.Hour
	// repeatWindow is how many earlier operator messages a repeated report may
	// look back over.
	repeatWindow = 3
)

// Job classifies operator messages in the background: new ones as they are
// stored, and the last Window back-filled within a per-pass budget. When
// ai-gateway or its decision models are unavailable the messages are recorded
// unclassified (lowering coverage) and retried, never recorded as no friction.
type Job struct {
	Store      *Store
	Events     EventSource
	Classifier MessageClassifier
	Now        func() time.Time
	Log        *slog.Logger

	Interval, Window, Slice, PassBudget, Rewalk time.Duration
	PageSize, PassMessages                      int

	cursor      Cursor
	walkStarted time.Time
	walkDone    bool
}

// PassStats reports one pass.
type PassStats struct {
	Scanned, Classified, Unclassified, Initial, Duplicate int
	WalkDone                                              bool
}

// Start prunes rows of other classifier versions and runs passes until ctx
// ends. A nil job or one without a store, events or classifier does nothing.
func (j *Job) Start(ctx context.Context) {
	if j == nil || j.Store == nil || j.Events == nil || j.Classifier == nil {
		return
	}
	j.defaults()
	go func() {
		if pruned, err := j.Store.PruneVersions(ctx, ClassifierVersion); err != nil {
			j.Log.Warn("pruning old intervention classifications failed", "error", err.Error())
		} else if pruned > 0 {
			j.Log.Info("pruned intervention classifications of retired classifier versions", "rows", pruned)
		}
		ticker := time.NewTicker(j.Interval)
		defer ticker.Stop()
		for {
			stats, err := j.Pass(ctx)
			if err != nil && ctx.Err() == nil {
				j.Log.Warn("operator intervention pass degraded", "error", err.Error(), "classified", stats.Classified, "unclassified", stats.Unclassified)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (j *Job) defaults() {
	if j.Now == nil {
		j.Now = time.Now
	}
	if j.Log == nil {
		j.Log = slog.Default()
	}
	set := func(value *time.Duration, fallback time.Duration) {
		if *value <= 0 {
			*value = fallback
		}
	}
	set(&j.Interval, DefaultInterval)
	set(&j.Window, DefaultWindow)
	set(&j.Slice, DefaultSlice)
	set(&j.PassBudget, DefaultPassBudget)
	set(&j.Rewalk, DefaultRewalk)
	if j.PageSize <= 0 {
		j.PageSize = DefaultPageSize
	}
	if j.PassMessages <= 0 {
		j.PassMessages = DefaultPassMessages
	}
}

// Pass advances the walk by at most PassMessages scanned message rows or
// PassBudget, whichever comes first; a pass that reaches its deadline inside a
// page resumes at the first run it did not process. A gateway failure ends the
// pass after the failed messages are recorded unclassified.
func (j *Job) Pass(ctx context.Context) (PassStats, error) {
	j.defaults()
	var stats PassStats
	now := j.Now()
	deadline := now.Add(j.PassBudget)
	nowText := formatTime(now)
	// After a walk reaches the present the cursor follows new messages; a
	// fresh walk of the whole window starts once Rewalk has passed.
	if j.cursor.Timestamp == "" || (j.walkDone && now.Sub(j.walkStarted) >= j.Rewalk) {
		j.cursor, j.walkStarted, j.walkDone = Cursor{Timestamp: formatTime(now.Add(-j.Window))}, now, false
	}
	runs := passRuns{}
	for stats.Scanned < j.PassMessages && j.Now().Before(deadline) && ctx.Err() == nil {
		if j.cursor.Timestamp >= nowText {
			j.walkDone, stats.WalkDone = true, true
			return stats, nil
		}
		upper := nowText
		if sliceEnd, err := time.Parse(time.RFC3339Nano, j.cursor.Timestamp); err == nil && sliceEnd.Add(j.Slice).Before(now) {
			upper = formatTime(sliceEnd.Add(j.Slice))
		}
		// A page never takes the pass past PassMessages, so the cap is the
		// most messages one pass sends to the classifier.
		candidates, next, scanned, err := j.Store.Candidates(ctx, j.cursor, upper, ClassifierVersion, now, min(j.PageSize, j.PassMessages-stats.Scanned))
		if err != nil {
			return stats, err
		}
		stats.Scanned += scanned
		// The cursor moves past the page even when processing fails: what was
		// not recorded is found again on the next walk.
		j.cursor = next
		resume, err := j.process(ctx, candidates, runs, now, deadline, &stats)
		if resume != nil {
			j.cursor = *resume
		}
		if err != nil || resume != nil {
			return stats, err
		}
	}
	return stats, nil
}

// passRuns holds, per run, the events one pass has read and the operator
// turns derived from them, so a run spread over pages is read once.
type passRuns map[string]*runTurns

type runTurns struct {
	events  []*domain.RunEvent
	through int64 // the last sequence read; events after it are not held
	turns   map[string]operatorTurn
	order   []string
}

// turnsThrough returns a run's operator turns over its events up to sequence
// need, which is all a message's classification and its repeat check read.
// Only the events after those already held are read, and at most need minus
// the last sequence held of them: sequences are unique in a run.
func (j *Job) turnsThrough(ctx context.Context, runs passRuns, id uuid.UUID, need int64) (*runTurns, error) {
	run := runs[id.String()]
	if run == nil {
		run = &runTurns{through: -1}
		runs[id.String()] = run
	}
	if run.turns != nil && run.through >= need {
		return run, nil
	}
	events, err := j.Events.Get(ctx, id, event.GetOptions{AfterSequence: run.through, Limit: int(need - run.through), EventTypes: []domain.RunEventType{domain.EventTypeMessage, domain.EventTypeToolCall, domain.EventTypeMetric, domain.EventTypeStatus}})
	if err != nil {
		return nil, err
	}
	for _, ev := range events {
		if ev.Sequence <= need {
			run.events = append(run.events, ev)
		}
	}
	run.through = need
	run.turns, run.order = operatorTurns(run.events)
	return run, nil
}

// operatorTurn is one operator message in a run with what preceded it.
type operatorTurn struct {
	index              int
	text               string
	precedingAgentText string
	work               Work
	initial            bool
}

// process classifies one page of candidates, run by run, and saves them. At
// the deadline it starts no further run and returns the cursor just before
// the first candidate it left, so the next pass resumes there.
func (j *Job) process(ctx context.Context, candidates []Candidate, runs passRuns, now, deadline time.Time, stats *PassStats) (*Cursor, error) {
	byRun := map[string][]Candidate{}
	var order []string
	for _, c := range candidates {
		if _, seen := byRun[c.RunID]; !seen {
			order = append(order, c.RunID)
		}
		byRun[c.RunID] = append(byRun[c.RunID], c)
	}
	keys := make([]string, len(candidates))
	for i, c := range candidates {
		keys[i] = messageKey(c)
	}
	recorded, err := j.Store.RecordedKeys(ctx, ClassifierVersion, keys)
	if err != nil {
		return nil, err
	}
	for _, runID := range order {
		if !j.Now().Before(deadline) || ctx.Err() != nil {
			first := byRun[runID][0] // the earliest candidate left: runs go in order of first appearance
			return &Cursor{Timestamp: first.OccurredAt, RunID: first.RunID, Sequence: first.Sequence - 1}, nil
		}
		// A gateway failure still returns the run's records, unclassified, so
		// coverage reflects them; the pass then stops.
		records, err := j.processRun(ctx, runID, byRun[runID], runs, recorded, now, stats)
		if saveErr := j.Store.Save(ctx, records, now); saveErr != nil {
			return nil, saveErr
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (j *Job) processRun(ctx context.Context, runID string, candidates []Candidate, runs passRuns, recorded map[string]string, now time.Time, stats *PassStats) ([]Record, error) {
	id, err := uuid.Parse(runID)
	if err != nil {
		return nil, nil
	}
	var need int64
	for _, c := range candidates {
		need = max(need, c.Sequence)
	}
	run, err := j.turnsThrough(ctx, runs, id, need)
	if err != nil {
		return nil, err
	}
	turns, order := run.turns, run.order
	records := make([]Record, 0, len(candidates))
	var (
		toClassify []Message
		pending    []int // record index of each message to classify
	)
	for _, c := range candidates {
		turn, ok := turns[c.EventID]
		if !ok {
			continue // the event was deleted after the scan
		}
		record := Record{Candidate: c, Version: ClassifierVersion, MessageKey: messageKey(c), Work: turn.work}
		switch {
		case recorded[record.MessageKey] != "" && recorded[record.MessageKey] != c.EventID:
			record.Answer = Answer{Status: StatusDuplicate}
			stats.Duplicate++
		case turn.initial:
			record.Answer = Answer{Status: StatusInitial}
			stats.Initial++
		default:
			pending = append(pending, len(records))
			toClassify = append(toClassify, Message{EventID: c.EventID, OperatorText: turn.text, PrecedingAgentText: turn.precedingAgentText})
		}
		if recorded[record.MessageKey] == "" {
			recorded[record.MessageKey] = c.EventID
		}
		records = append(records, record)
	}
	if len(toClassify) == 0 {
		return records, nil
	}
	answers, classifyErr := j.Classifier.Classify(ctx, toClassify)
	for k, i := range pending {
		// A classifier that answers fewer messages than it was asked leaves the
		// rest unclassified, to retry, never a panic.
		answer := Answer{Status: StatusUnclassified, Reason: fmt.Sprintf("the classifier answered %d of %d messages", len(answers), len(toClassify))}
		if k < len(answers) {
			answer = answers[k]
		}
		records[i].Answer = answer
		if answer.Status == StatusUnclassified {
			records[i].NextAttemptAt = now.Add(min(time.Duration(records[i].Attempts+1)*retryBackoff, 24*time.Hour))
			stats.Unclassified++
		} else {
			stats.Classified++
		}
	}
	if err := j.markRepeats(ctx, records, turns, order); err != nil {
		return records, err
	}
	return records, classifyErr
}

// operatorTurns walks a run once and returns each operator message with the
// last agent text before it and the agent work since the previous operator
// message, plus the operator message ids in order.
func operatorTurns(events []*domain.RunEvent) (map[string]operatorTurn, []string) {
	account := runsignal.AccountUsage(events)
	turns := map[string]operatorTurn{}
	var (
		order     []string
		agentText string
		work      Work
		agentSeen bool
	)
	for i, ev := range events {
		if total := account.PerTurn[i].Total(); total > 0 {
			work.Requests++
			work.Tokens += total
		}
		switch data := ev.Data.(type) {
		case *domain.ToolCallEventData:
			work.ToolCalls++
			agentSeen = true
		case *domain.MessageEventData:
			if strings.EqualFold(data.Role, "assistant") {
				if text := strings.TrimSpace(data.Content); text != "" && !data.EvidenceOnly {
					agentText = text
					agentSeen = true
				}
				continue
			}
			if domain.MessageSource(ev) != domain.MessageSourceOperator {
				continue
			}
			id := ev.ID.String()
			turns[id] = operatorTurn{index: len(order), text: data.Content, precedingAgentText: agentText, work: work, initial: !agentSeen}
			order = append(order, id)
			work = Work{}
		}
	}
	return turns, order
}

// markRepeats links a quality_miss report to an earlier quality_miss in the
// previous repeatWindow operator messages of the run when the agent worked in
// between (it acted on the first report) and the two describe the same
// symptom. Deterministic over stored categories: the decision state holds one
// message, so the model cannot see the earlier report or the claimed fix.
func (j *Job) markRepeats(ctx context.Context, records []Record, turns map[string]operatorTurn, order []string) error {
	var earlier []string
	for _, r := range records {
		if r.Answer.Category != QualityMiss {
			continue
		}
		at := turns[r.EventID].index
		earlier = append(earlier, order[max(0, at-repeatWindow):at]...)
	}
	if len(earlier) == 0 {
		return nil
	}
	categories, err := j.Store.Categories(ctx, ClassifierVersion, earlier)
	if err != nil {
		return err
	}
	for _, r := range records {
		if r.Answer.Status == StatusClassified {
			categories[r.EventID] = r.Answer.Category
		}
	}
	for i := range records {
		r := &records[i]
		if r.Answer.Category != QualityMiss {
			continue
		}
		current := turns[r.EventID]
		toolCalls := current.work.ToolCalls
		for k := current.index - 1; k >= max(0, current.index-repeatWindow); k-- {
			previous := turns[order[k]]
			if categories[order[k]] == QualityMiss && toolCalls > 0 && sameSymptom(previous.text, current.text) {
				r.RepeatOf = order[k]
				break
			}
			toolCalls += previous.work.ToolCalls
		}
	}
	return nil
}

// sameSymptom reports whether two reports share a content word: at least a
// quarter of the shorter report's distinctive words (four letters or more,
// generic words removed).
func sameSymptom(a, b string) bool {
	wordsA, wordsB := symptomWords(a), symptomWords(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return false
	}
	shared := 0
	for word := range wordsA {
		if wordsB[word] {
			shared++
		}
	}
	return shared > 0 && float64(shared) >= 0.25*float64(min(len(wordsA), len(wordsB)))
}

var genericWords = map[string]bool{}

func init() {
	for _, word := range strings.Fields(`about again also because been before being could didn does doesn done even fixed from have here
	into just know like look looks make more need only please really said should still sure than that their them then there these they
	thing things think this those though want were what when where which while will with work working works would your`) {
		genericWords[word] = true
	}
}

func symptomWords(text string) map[string]bool {
	words := map[string]bool{}
	for _, word := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if len(word) >= 4 && !genericWords[word] {
			words[word] = true
		}
	}
	return words
}

// messageKey identifies a message across rollouts: a forked Codex rollout
// replays the parent's operator messages with identical timestamps and text.
func messageKey(c Candidate) string {
	sum := sha256.Sum256([]byte(c.OccurredAt + "\x00" + strings.TrimSpace(c.Text)))
	return hex.EncodeToString(sum[:16])
}
