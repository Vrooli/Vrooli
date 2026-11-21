package interventions

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

var dbSeq atomic.Int64

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("file:interventions-%d?mode=memory&cache=shared", dbSeq.Add(1)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// The run-event high-water triggers (eventlog schema) read runs.id; the
	// runs table itself is created by the database package, not these schemas.
	runs := "CREATE TABLE IF NOT EXISTS runs (id TEXT PRIMARY KEY);"
	for _, schema := range []string{runs, domain.Schema(), eventlog.Schema(), Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// textClassifier stands in for the decision model: a category per message
// text, or a gateway outage.
type textClassifier struct {
	categories map[string]Category
	down       bool
	calls      int
	classified []string
	// short answers this many fewer messages than it was asked.
	short int
}

func (c *textClassifier) Classify(_ context.Context, messages []Message) ([]Answer, error) {
	c.calls++
	answers := make([]Answer, len(messages))
	for i, m := range messages {
		c.classified = append(c.classified, m.OperatorText)
		if c.down {
			answers[i] = unclassified("unavailable: ollaya is not running")
			continue
		}
		answers[i] = Answer{Status: StatusClassified, Category: c.categories[m.OperatorText], Probabilities: map[Category]float64{c.categories[m.OperatorText]: 0.9}, Method: "DECISION_MODEL", Model: "fake:decide"}
	}
	if c.down {
		return answers, fmt.Errorf("%w: connection refused", ErrGatewayUnavailable)
	}
	return answers[:max(0, len(answers)-c.short)], nil
}

type runBuilder struct {
	t      *testing.T
	store  *event.SQLiteStore
	runID  uuid.UUID
	at     time.Time
	events []*domain.RunEvent
}

func (b *runBuilder) add(ev *domain.RunEvent) *domain.RunEvent {
	b.at = b.at.Add(time.Minute)
	ev.Timestamp = b.at
	b.events = append(b.events, ev)
	return ev
}

func (b *runBuilder) operator(text string) *domain.RunEvent {
	return b.add(domain.NewProviderMessageEvent(b.runID, "user", text, domain.MessageEventData{ProviderOrigin: "claude"}))
}

func (b *runBuilder) teammate(text string) *domain.RunEvent {
	return b.add(domain.NewProviderMessageEvent(b.runID, "user", text, domain.MessageEventData{ProviderOrigin: "codex", Source: domain.MessageSourceTeammate}))
}

func (b *runBuilder) agent(text string, toolCalls int, tokens int) {
	b.add(domain.NewProviderMessageEvent(b.runID, "assistant", text, domain.MessageEventData{}))
	for i := 0; i < toolCalls; i++ {
		b.add(domain.NewToolCallEvent(b.runID, "Bash", uuid.NewString(), map[string]any{"command": "make test"}))
	}
	if tokens > 0 {
		b.add(&domain.RunEvent{RunID: b.runID, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, InputTokens: tokens, OutputTokens: 10, MessageID: uuid.NewString()}})
	}
}

func (b *runBuilder) save() {
	if err := b.store.Append(context.Background(), b.runID, b.events...); err != nil {
		b.t.Fatal(err)
	}
}

func readRows(t *testing.T, db *sqlx.DB) map[string]map[string]any {
	t.Helper()
	rows, err := db.Queryx(`SELECT * FROM operator_interventions`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]map[string]any{}
	for rows.Next() {
		row := map[string]any{}
		if err := rows.MapScan(row); err != nil {
			t.Fatal(err)
		}
		out[fmt.Sprint(row["event_id"])] = row
	}
	return out
}

func TestJobClassifiesOperatorMessagesWithPrecedingWorkAndRepeats(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-10 * 24 * time.Hour)}
	opening := run.operator("Make the message list wrap long lines")
	run.agent("Done: the message list now wraps.", 3, 1000)
	first := run.operator("The wrapping is broken on mobile, long lines overflow")
	run.agent("Fixed the mobile wrapping.", 2, 500)
	report := run.teammate("Handoff received: 12:04Z exactT3response verified")
	repeat := run.operator("Still broken: long lines overflow on mobile")
	run.agent("Checking.", 0, 0)
	thanks := run.operator("thanks")
	run.save()
	// A forked rollout replays the first correction with the same timestamp.
	fork := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-9 * 24 * time.Hour)}
	replayed := domain.NewProviderMessageEvent(fork.runID, "user", "The wrapping is broken on mobile, long lines overflow", domain.MessageEventData{ProviderOrigin: "codex"})
	replayed.Timestamp = first.Timestamp
	fork.events = append(fork.events, replayed)
	fork.save()

	classifier := &textClassifier{categories: map[string]Category{
		first.Data.(*domain.MessageEventData).Content: QualityMiss, repeat.Data.(*domain.MessageEventData).Content: QualityMiss, "thanks": ApprovalOrInfo,
	}}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }}
	stats, err := job.Pass(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Classified != 3 || stats.Initial != 1 || stats.Duplicate != 1 || stats.Unclassified != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	rows := readRows(t, db)
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want opening, two corrections, thanks and the fork replay (teammate report excluded)", len(rows))
	}
	if _, ok := rows[report.ID.String()]; ok {
		t.Fatal("a teammate report must not be classified as an operator message")
	}
	check := func(ev *domain.RunEvent, status Status, category Category, toolCalls, requests int64, repeatOf string) {
		t.Helper()
		row := rows[ev.ID.String()]
		if row["status"] != string(status) || row["category"] != string(category) || row["work_tool_calls"] != toolCalls || row["work_requests"] != requests || row["repeat_of_event_id"] != repeatOf || row["classifier_version"] != ClassifierVersion {
			t.Fatalf("row %v, want status=%s category=%s tool_calls=%d requests=%d repeat_of=%q", row, status, category, toolCalls, requests, repeatOf)
		}
	}
	check(opening, StatusInitial, "", 0, 0, "")
	check(first, StatusClassified, QualityMiss, 3, 1, "")
	check(repeat, StatusClassified, QualityMiss, 2, 1, first.ID.String())
	check(thanks, StatusClassified, ApprovalOrInfo, 0, 0, "")
	check(replayed, StatusDuplicate, "", 0, 0, "")
	if rows[first.ID.String()]["work_tokens"] != int64(1010) {
		t.Fatalf("work tokens = %v, want the 1010 tokens of the agent turn before the correction", rows[first.ID.String()]["work_tokens"])
	}

	// Idempotent: a later walk over the same window classifies nothing again.
	now = now.Add(2 * time.Hour)
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if classifier.calls != 1 {
		t.Fatalf("classifier calls = %d, want 1", classifier.calls)
	}
}

// With the decision capability down, messages are recorded unclassified
// (coverage falls), never as "no friction", and retried once due.
func TestJobDegradesToUnclassifiedAndRetries(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2 * time.Hour)}
	run.operator("Fix the build")
	run.agent("Done.", 1, 100)
	correction := run.operator("Keep going, you stopped early")
	run.save()
	classifier := &textClassifier{categories: map[string]Category{"Keep going, you stopped early": EarlyStop}, down: true}
	store := NewStore(db)
	job := &Job{Store: store, Events: events, Classifier: classifier, Now: func() time.Time { return now }}
	if _, err := job.Pass(context.Background()); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("pass err = %v, want the gateway failure", err)
	}
	row := readRows(t, db)[correction.ID.String()]
	if row["status"] != string(StatusUnclassified) || row["category"] != "" || row["attempts"] != int64(1) || !strings.Contains(fmt.Sprint(row["unclassified_reason"]), "ollaya is not running") {
		t.Fatalf("row = %v, want unclassified with the reason", row)
	}
	summary, err := store.Summarize(context.Background(), ClassifierVersion, Gate{"fake:decide": {EarlyStop: 0.5}}, now.Add(-time.Hour*24), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if coverage := summary.Coverage(); coverage == nil || *coverage != 0 || summary.Unclassified != 1 {
		t.Fatalf("coverage = %v unclassified = %d, want 0 of 1", coverage, summary.Unclassified)
	}

	classifier.down = false
	now = now.Add(30 * time.Minute) // walk done, not yet due
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := readRows(t, db)[correction.ID.String()]["status"]; got != string(StatusUnclassified) {
		t.Fatalf("status = %v before the retry is due", got)
	}
	now = now.Add(2 * time.Hour) // a new walk, attempt due
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if row := readRows(t, db)[correction.ID.String()]; row["status"] != string(StatusClassified) || row["category"] != string(EarlyStop) {
		t.Fatalf("row = %v, want classified on retry", row)
	}
}

// A classifier that answers fewer messages than it was asked never panics the
// job: the answered ones are recorded, the rest stay unclassified to retry.
func TestJobMarksMessagesAShortAnswerLeftOutUnclassified(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2 * time.Hour)}
	run.operator("Fix the build")
	run.agent("Done.", 1, 100)
	first := run.operator("Keep going, you stopped early")
	run.agent("Continuing.", 1, 100)
	second := run.operator("That claim is wrong")
	run.save()
	classifier := &textClassifier{categories: map[string]Category{"Keep going, you stopped early": EarlyStop, "That claim is wrong": WrongClaim}, short: 1}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }}
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	rows := readRows(t, db)
	if rows[first.ID.String()]["status"] != string(StatusClassified) {
		t.Errorf("answered message = %v", rows[first.ID.String()])
	}
	if row := rows[second.ID.String()]; row["status"] != string(StatusUnclassified) || !strings.Contains(fmt.Sprint(row["unclassified_reason"]), "answered 1 of 2 messages") {
		t.Errorf("unanswered message = %v, want unclassified with the reason", row)
	}
}

func TestSummarizeShowsOnlyGatedCategoriesPerAgentHour(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	p := func(v float64) map[Category]float64 { return map[Category]float64{QualityMiss: v, EarlyStop: v} }
	records := []Record{
		{Candidate: Candidate{EventID: "e1", RunID: "r1", OccurredAt: formatTime(at)}, Version: ClassifierVersion, MessageKey: "k1", Answer: Answer{Status: StatusClassified, Category: QualityMiss, Probabilities: p(0.9), Model: "laya:en"}, Work: Work{Tokens: 400}},
		{Candidate: Candidate{EventID: "e2", RunID: "r1", OccurredAt: formatTime(at.Add(time.Minute))}, Version: ClassifierVersion, MessageKey: "k2", Answer: Answer{Status: StatusClassified, Category: QualityMiss, Probabilities: p(0.3), Model: "laya:en"}, Work: Work{Tokens: 100}},
		{Candidate: Candidate{EventID: "e3", RunID: "r1", OccurredAt: formatTime(at.Add(2 * time.Minute))}, Version: ClassifierVersion, MessageKey: "k3", Answer: Answer{Status: StatusClassified, Category: QualityMiss, Probabilities: p(0.95), Model: "laya:en"}, Work: Work{Tokens: 200}, RepeatOf: "e1"},
		{Candidate: Candidate{EventID: "e4", RunID: "r1", OccurredAt: formatTime(at.Add(3 * time.Minute))}, Version: ClassifierVersion, MessageKey: "k4", Answer: Answer{Status: StatusClassified, Category: EarlyStop, Probabilities: p(0.99), Model: "laya:en"}},
		{Candidate: Candidate{EventID: "e5", RunID: "r1", OccurredAt: formatTime(at.Add(4 * time.Minute))}, Version: ClassifierVersion, MessageKey: "k5", Answer: unclassified("timeout"), NextAttemptAt: at.Add(time.Hour)},
		{Candidate: Candidate{EventID: "e6", RunID: "r1", OccurredAt: formatTime(at.Add(5 * time.Minute))}, Version: ClassifierVersion, MessageKey: "k6", Answer: Answer{Status: StatusInitial}},
	}
	if err := store.Save(context.Background(), records, at); err != nil {
		t.Fatal(err)
	}
	// Two runs active in three ten-minute slots: half an hour of agent time.
	for i, slot := range []struct {
		run string
		at  time.Time
	}{{"r1", at}, {"r1", at.Add(2 * time.Minute)}, {"r1", at.Add(15 * time.Minute)}, {"r2", at}} {
		if _, err := db.Exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at) VALUES (?, ?, 0, ?, ?)`, "f", i, slot.run, formatTime(slot.at)); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := store.Summarize(context.Background(), ClassifierVersion, Gate{"laya:en": {QualityMiss: 0.5}}, at.Add(-time.Hour), at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	byCategory := map[Category]CategorySummary{}
	for _, c := range summary.Categories {
		byCategory[c.Category] = c
	}
	if qm := byCategory[QualityMiss]; qm.Classified != 3 || qm.Shown != 2 || qm.Repeats != 1 || qm.MedianWorkTokens != 400 || qm.Work.Tokens != 600 {
		t.Fatalf("quality_miss = %+v, want 3 classified, 2 above the gate, 1 repeat, median 400", qm)
	}
	if es := byCategory[EarlyStop]; es.Classified != 1 || es.Shown != 0 {
		t.Fatalf("early_stop = %+v, want classified but not shown (not proven)", es)
	}
	if summary.Classified != 4 || summary.Unclassified != 1 || summary.AgentHours != 0.5 {
		t.Fatalf("summary = %+v, want 4 classified, 1 unclassified, 0.5 agent-hours", summary)
	}
}

func TestSameSymptomNeedsSharedDistinctiveWords(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"The wrapping is broken on mobile", "Still broken: wrapping on mobile", true},
		{"The notification never arrived", "I didn't receive a notification again", true},
		{"The wrapping is broken on mobile", "The login page is slow", false},
		{"It still does not work", "This still doesn't work", false},
	}
	for _, tc := range cases {
		if got := sameSymptom(tc.a, tc.b); got != tc.want {
			t.Errorf("sameSymptom(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPruneVersionsKeepsOnlyTheCurrentClassifier(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	records := []Record{
		{Candidate: Candidate{EventID: "e1", OccurredAt: formatTime(at)}, Version: "choice-retired", MessageKey: "k", Answer: Answer{Status: StatusInitial}},
		{Candidate: Candidate{EventID: "e1", OccurredAt: formatTime(at)}, Version: ClassifierVersion, MessageKey: "k", Answer: Answer{Status: StatusInitial}},
	}
	if err := store.Save(context.Background(), records, at); err != nil {
		t.Fatal(err)
	}
	if pruned, err := store.PruneVersions(context.Background(), ClassifierVersion); err != nil || pruned != 1 {
		t.Fatalf("pruned = %d err = %v, want 1", pruned, err)
	}
}

// Rows returns only what the gate admits, newest first, with the message
// redacted and shortened, the agent work before it and the repeat link; a run
// filter and the limit narrow it.
func TestRowsAreGatedRedactedAndLinked(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	store := NewStore(db)
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: at}
	first := run.operator("Still broken, api_key=sk-secret12345 " + strings.Repeat("long ", 100))
	second := run.operator("The notification never arrived")
	other := &runBuilder{t: t, store: events, runID: uuid.New(), at: at.Add(time.Hour)}
	hidden := other.operator("thanks")
	run.save()
	other.save()
	answer := func(c Category) Answer {
		return Answer{Status: StatusClassified, Category: c, Probabilities: map[Category]float64{c: 0.9}, Model: "fake:decide"}
	}
	record := func(e *domain.RunEvent, c Category, repeatOf string) Record {
		return Record{Candidate: Candidate{EventID: e.ID.String(), RunID: e.RunID.String(), OccurredAt: formatTime(e.Timestamp)}, Version: ClassifierVersion, MessageKey: e.ID.String(), Answer: answer(c), Work: Work{Requests: 3, ToolCalls: 2, Tokens: 900}, RepeatOf: repeatOf}
	}
	if err := store.Save(context.Background(), []Record{record(first, QualityMiss, ""), record(second, QualityMiss, first.ID.String()), record(hidden, ApprovalOrInfo, "")}, at); err != nil {
		t.Fatal(err)
	}
	gate := Gate{"fake:decide": {QualityMiss: 0.5}}
	window := func(limit int, runID string) []Row {
		rows, err := store.Rows(context.Background(), ClassifierVersion, gate, at.Add(-time.Hour), at.Add(2*time.Hour), runID, limit)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	rows := window(0, "")
	if len(rows) != 2 || rows[0].EventID != second.ID.String() || rows[0].RepeatOf != first.ID.String() || rows[0].Work != (Work{Requests: 3, ToolCalls: 2, Tokens: 900}) {
		t.Fatalf("rows = %+v, want the two quality_miss rows newest first, the repeat linked to the first", rows)
	}
	if text := rows[1].Text; strings.Contains(text, "sk-secret") || !strings.HasSuffix(text, "…") || len([]rune(text)) > exampleLength+1 {
		t.Fatalf("text = %q, want redacted and shortened", text)
	}
	if rows := window(1, ""); len(rows) != 1 {
		t.Fatalf("limit 1 returned %d rows", len(rows))
	}
	if rows, err := store.Rows(context.Background(), ClassifierVersion, Gate{}, at.Add(-time.Hour), at.Add(2*time.Hour), "", 0); err != nil || len(rows) != 0 {
		t.Fatalf("empty gate rows = %+v err = %v, want none", rows, err)
	}
	if rows := window(0, other.runID.String()); len(rows) != 0 {
		t.Fatalf("run filter returned %+v, want none (the only row of that run is not admitted)", rows)
	}
}

// Under a pooled gate, friction by probability mass that no proven category
// admits is counted once as "category unproven", with its most likely category
// as a hint; a message a proven category admits is not counted twice.
func TestPooledFrictionCountsUnprovenCategoriesOnceWithHints(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	answer := func(c Category, distribution map[Category]float64) Answer {
		return Answer{Status: StatusClassified, Category: c, Probabilities: distribution, Model: "fake:decide"}
	}
	record := func(id string, minute int, a Answer, repeatOf string) Record {
		return Record{Candidate: Candidate{EventID: id, RunID: "r1", OccurredAt: formatTime(at.Add(time.Duration(minute) * time.Minute))}, Version: ClassifierVersion, MessageKey: id, Answer: a, Work: Work{Requests: 2, ToolCalls: 1, Tokens: 100}, RepeatOf: repeatOf}
	}
	records := []Record{
		record("quality", 0, answer(QualityMiss, map[Category]float64{QualityMiss: 0.6, EarlyStop: 0.2, ApprovalOrInfo: 0.2}), ""),
		record("quality-again", 1, answer(QualityMiss, map[Category]float64{QualityMiss: 0.7, TooDense: 0.1, ScopeChange: 0.2}), "quality"),
		record("early", 2, answer(EarlyStop, map[Category]float64{EarlyStop: 0.5, WrongClaim: 0.3, ApprovalOrInfo: 0.2}), ""),
		record("approval", 3, answer(ApprovalOrInfo, map[Category]float64{ApprovalOrInfo: 0.9, ScopeChange: 0.05, EarlyStop: 0.05}), ""),
	}
	if err := store.Save(context.Background(), records, at); err != nil {
		t.Fatal(err)
	}
	window := func(gate Gate) (Summary, []Row) {
		summary, err := store.Summarize(context.Background(), ClassifierVersion, gate, at.Add(-time.Hour), at.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := store.Rows(context.Background(), ClassifierVersion, gate, at.Add(-time.Hour), at.Add(time.Hour), "", 0)
		if err != nil {
			t.Fatal(err)
		}
		return summary, rows
	}
	pooledOnly := Gate{"fake:decide": {PooledFriction: 0.5}}
	summary, rows := window(pooledOnly)
	if p := summary.Pooled; p.Shown != 3 || p.Repeats != 1 || p.Work != (Work{Requests: 6, ToolCalls: 3, Tokens: 300}) || len(p.Hints) != 2 || p.Hints[0] != (CategoryHint{EarlyStop, 1}) || p.Hints[1] != (CategoryHint{QualityMiss, 2}) {
		t.Fatalf("pooled = %+v, want the three friction messages, hinted early_stop 1 and quality_miss 2", p)
	}
	for _, c := range summary.Categories {
		if c.Shown != 0 {
			t.Fatalf("%s shown %d under a pooled-only gate", c.Category, c.Shown)
		}
	}
	if len(rows) != 3 || !rows[0].Pooled || rows[0].Category != EarlyStop {
		t.Fatalf("rows = %+v, want three pooled rows, newest first, hinted by their friction category", rows)
	}
	both := Gate{"fake:decide": {PooledFriction: 0.5, QualityMiss: 0.5}}
	summary, rows = window(both)
	if summary.Pooled.Shown != 1 || summary.Pooled.Hints[0].Category != EarlyStop {
		t.Fatalf("pooled with quality_miss proven = %+v, want only the early_stop message", summary.Pooled)
	}
	var proven, pooled int
	for _, r := range rows {
		if r.Pooled {
			pooled++
		} else {
			proven++
		}
	}
	if proven != 2 || pooled != 1 {
		t.Fatalf("rows proven %d pooled %d, want 2 and 1 (no double count)", proven, pooled)
	}
	if summary, _ := window(Gate{}); summary.Pooled.Shown != 0 {
		t.Fatalf("an empty gate pooled %+v", summary.Pooled)
	}
}

// One pass sends at most PassMessages operator messages to the classifier,
// even when a page could hold more; the next passes continue the walk.
func TestJobPassNeverClassifiesMoreThanPassMessages(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2 * time.Hour)}
	run.operator("Start the audit")
	for i := 0; i < 7; i++ {
		run.agent("Working.", 1, 100)
		run.operator(fmt.Sprintf("correction %d", i))
	}
	run.save()
	classifier := &textClassifier{categories: map[string]Category{}}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }, PassMessages: 3, PageSize: 200}
	for pass := 0; pass < 3; pass++ {
		before := len(classifier.classified)
		stats, err := job.Pass(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if stats.Scanned > 3 || len(classifier.classified)-before > 3 {
			t.Fatalf("pass %d scanned %d and classified %d messages, want at most 3", pass, stats.Scanned, len(classifier.classified)-before)
		}
	}
	if len(classifier.classified) != 7 {
		t.Fatalf("classified %d of the 7 corrections across the passes", len(classifier.classified))
	}
}

// clockClassifier advances the job's clock by step on every call, so a pass
// reaches its deadline after its first classification.
type clockClassifier struct {
	textClassifier
	now  *time.Time
	step time.Duration
}

func (c *clockClassifier) Classify(ctx context.Context, messages []Message) ([]Answer, error) {
	*c.now = c.now.Add(c.step)
	return c.textClassifier.Classify(ctx, messages)
}

// The pass deadline holds inside a page: once it passes, no further run is
// started, and the next pass resumes at the first message left rather than
// waiting for the next walk of the window.
func TestJobPassStopsAtItsDeadlineInsideAPageAndResumes(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	a := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2 * time.Hour)}
	b := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2*time.Hour + 30*time.Second)}
	for _, run := range []*runBuilder{a, b} {
		run.operator("Start")
		run.agent("Working.", 1, 100)
		run.operator("correction in " + run.runID.String()[:8])
		run.save()
	}
	classifier := &clockClassifier{textClassifier: textClassifier{categories: map[string]Category{}}, now: &now, step: time.Minute}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }, PassBudget: 30 * time.Second}
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(classifier.classified) != 1 {
		t.Fatalf("the first pass classified %v, want one run's message before its deadline", classifier.classified)
	}
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(classifier.classified) != 2 {
		t.Fatalf("the next pass classified %v, want the message the deadline left", classifier.classified)
	}
}

// countingEvents records the events a pass reads.
type countingEvents struct {
	EventSource
	calls int
	read  []int64
}

func (c *countingEvents) Get(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	c.calls++
	events, err := c.EventSource.Get(ctx, runID, opts)
	for _, ev := range events {
		c.read = append(c.read, ev.Sequence)
	}
	return events, err
}

// A run spread over several pages of one pass is read once, and only up to
// its last message in the pass: the agent's work after it is not read.
func TestJobReadsARunOncePerPassUpToItsLastMessage(t *testing.T) {
	db := newTestDB(t)
	store := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: store, runID: uuid.New(), at: now.Add(-3 * time.Hour)}
	run.operator("Start the audit")
	for i := 0; i < 4; i++ {
		run.agent("Working.", 2, 100)
		run.operator(fmt.Sprintf("correction %d", i))
	}
	for i := 0; i < 20; i++ {
		run.agent("Still working.", 3, 100)
	}
	run.save()
	last := run.events[len(run.events)-1-20*5]
	events := &countingEvents{EventSource: store}
	classifier := &textClassifier{categories: map[string]Category{}}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }, PageSize: 2}
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(classifier.classified) != 4 {
		t.Fatalf("classified %v, want the 4 corrections", classifier.classified)
	}
	seen := map[int64]bool{}
	for _, seq := range events.read {
		if seen[seq] {
			t.Fatalf("event %d was read twice in one pass (%d reads)", seq, events.calls)
		}
		if seq > last.Sequence {
			t.Fatalf("event %d after the last message (%d) was read", seq, last.Sequence)
		}
		seen[seq] = true
	}
}

// A re-walk passes over recorded messages without reading their payload, and
// they count against the pass's scan cap like new work.
func TestJobRewalkCountsRecordedMessagesWithoutReadingThem(t *testing.T) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-2 * time.Hour)}
	run.operator("Start")
	for i := 0; i < 5; i++ {
		run.agent("Working.", 1, 100)
		run.operator(fmt.Sprintf("correction %d", i))
	}
	run.save()
	classifier := &textClassifier{categories: map[string]Category{}}
	job := &Job{Store: NewStore(db), Events: events, Classifier: classifier, Now: func() time.Time { return now }, PassMessages: 3}
	for pass := 0; pass < 3; pass++ {
		if _, err := job.Pass(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(readRows(t, db)) != 6 {
		t.Fatalf("recorded %d of 6 messages", len(readRows(t, db)))
	}
	// A payload no JSON reader accepts: reading a recorded one fails the scan.
	if _, err := db.Exec(`UPDATE run_events SET data = '{' WHERE id IN (SELECT event_id FROM operator_interventions)`); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour) // a new walk of the window
	for pass, want := range []PassStats{{Scanned: 3}, {Scanned: 3}, {WalkDone: true}} {
		stats, err := job.Pass(context.Background())
		if err != nil {
			t.Fatalf("re-walk pass %d read a recorded message: %v", pass, err)
		}
		if stats != want {
			t.Fatalf("re-walk pass %d = %+v, want %+v: recorded messages count against the cap", pass, stats, want)
		}
	}
	if len(classifier.classified) != 5 {
		t.Fatalf("the re-walk classified again: %v", classifier.classified)
	}
}
