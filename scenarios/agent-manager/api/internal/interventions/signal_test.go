package interventions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func signalDecision(label Signal, p float64) *inferencev1.DecideResponse {
	answer := &inferencev1.DecisionAnswer{QuestionId: signalQuestionID, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Choice: string(label)}
	rest := (1 - p) / float64(len(signalTaxonomy)-1)
	for _, l := range Signals() {
		probability := rest
		if l == label {
			probability = p
		}
		answer.Probabilities = append(answer.Probabilities, &inferencev1.DecisionProbability{Option: string(l), Probability: probability})
	}
	return &inferencev1.DecideResponse{Answers: []*inferencev1.DecisionAnswer{answer}, Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL, Provider: "ollaya", Model: "winnow:e4b", ResourceRole: "decide.local.accurate"}
}

// The signal question is a sibling of the category question: its own version,
// the local-only profile (message text never reaches a hosted decision model)
// and the agent's preceding text as context.
func TestClassifySignalsIsLocalOnlyAndCarriesContext(t *testing.T) {
	gateway := &fakeGateway{answer: func(s decodedState) *inferencev1.DecideResponse {
		if strings.Contains(s.OperatorMessage, "still broken") {
			return signalDecision(ReportedFailure, 0.9)
		}
		return signalDecision(Neutral, 0.8)
	}}
	mux := http.NewServeMux()
	mux.Handle(inferenceconnect.NewInferenceServiceHandler(gateway))
	server := httptest.NewServer(mux)
	defer server.Close()
	classifier := NewClassifier()
	classifier.ResolveURL = func(context.Context) (string, error) { return server.URL, nil }
	answers, err := classifier.ClassifySignals(context.Background(), []Message{
		{EventID: "a", OperatorText: "it is still broken", PrecedingAgentText: "I fixed the wrapping."},
		{EventID: "b", OperatorText: "please continue", PrecedingAgentText: "Done."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if answers[0].Signal != ReportedFailure || *answers[0].Probability() != 0.9 || answers[1].Signal != Neutral || answers[0].Model != "winnow:e4b" {
		t.Fatalf("answers = %+v", answers)
	}
	request := gateway.requests[0]
	if request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		t.Fatalf("profile = %v, want local-only", request.GetProfile())
	}
	if len(request.GetQuestions()) != 1 || request.GetQuestions()[0].GetId() != signalQuestionID || len(request.GetQuestions()[0].GetOptions()) != 5 {
		t.Fatalf("questions = %v, want the five-label signal question", request.GetQuestions())
	}
	var state decodedState
	_ = json.Unmarshal([]byte(request.GetItems()[0].GetStateJson()), &state)
	if state.PrecedingAgentText != "I fixed the wrapping." || state.OperatorMessage != "it is still broken" {
		t.Fatalf("state = %+v, want the message with the agent's preceding text", state)
	}
	if SignalVersion == ClassifierVersion || !strings.HasPrefix(SignalVersion, "signal-") {
		t.Fatalf("signal version %q must differ from the category classifier's %q", SignalVersion, ClassifierVersion)
	}
}

func TestSignalAnswerWithoutDistributionAndInvalidLabel(t *testing.T) {
	fallback := signalAnswerFrom(&inferencev1.DecideResponse{Answers: []*inferencev1.DecisionAnswer{{QuestionId: signalQuestionID, Choice: "correction"}}, Method: inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM})
	if fallback.Status != StatusClassified || fallback.Signal != Correction || fallback.Probability() != nil || fallback.ExplicitNegativeMass() != nil {
		t.Fatalf("fallback = %+v", fallback)
	}
	if bad := signalAnswerFrom(&inferencev1.DecideResponse{Answers: []*inferencev1.DecisionAnswer{{QuestionId: signalQuestionID, Choice: "angry"}}}); bad.Status != StatusUnclassified {
		t.Fatalf("an unknown label must be unclassified, got %+v", bad)
	}
}

type fakeSignals struct {
	labels     map[string]Signal
	down       bool
	calls      int
	seen       []Message
	probablity float64
}

func (f *fakeSignals) ClassifySignals(_ context.Context, messages []Message) ([]SignalAnswer, error) {
	f.calls++
	answers := make([]SignalAnswer, len(messages))
	for i, m := range messages {
		f.seen = append(f.seen, m)
		if f.down {
			answers[i] = SignalAnswer{Status: StatusUnclassified, Reason: "unavailable: ollaya is not running"}
			continue
		}
		label := f.labels[m.OperatorText]
		if label == "" {
			label = Neutral
		}
		p := f.probablity
		if p == 0 {
			p = 0.9
		}
		answers[i] = SignalAnswer{Status: StatusClassified, Signal: label, Probabilities: map[Signal]float64{label: p}, Method: "DECISION_MODEL", Model: "winnow:e4b"}
	}
	if f.down {
		return answers, fmt.Errorf("%w: connection refused", ErrGatewayUnavailable)
	}
	return answers, nil
}

type signalFixture struct {
	t        *testing.T
	store    *Store
	events   *event.SQLiteStore
	now      time.Time
	failed   *domain.RunEvent
	thanks   *domain.RunEvent
	opening  *domain.RunEvent
	teammate *domain.RunEvent
}

// signalFixture records one run (opening message, a failure report, a teammate
// report and thanks) and lets the intervention job derive its rows.
func newSignalFixture(t *testing.T) (*signalFixture, *Job) {
	db := newTestDB(t)
	events := event.NewSQLiteStore(db, logrus.New())
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	run := &runBuilder{t: t, store: events, runID: uuid.New(), at: now.Add(-3 * time.Hour)}
	f := &signalFixture{t: t, store: NewStore(db), events: events, now: now}
	f.opening = run.operator("Make the list wrap")
	run.agent("Done: the list wraps now.", 2, 500)
	f.failed = run.operator("it is still broken on mobile")
	run.agent("Fixed again.", 1, 300)
	f.teammate = run.teammate("Handoff received: 12:04Z exactT3response verified")
	f.thanks = run.operator("great, that works")
	run.save()
	job := &Job{Store: f.store, Events: events, Classifier: &textClassifier{categories: map[string]Category{}}, Now: func() time.Time { return now }}
	if _, err := job.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f, job
}

func signalRows(t *testing.T, f *signalFixture) map[string]map[string]any {
	t.Helper()
	rows, err := f.store.db.QueryxContext(context.Background(), `SELECT * FROM operator_signals`)
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

func TestSignalJobLabelsOnlyOperatorMessagesWithContextAndStoresNoText(t *testing.T) {
	f, _ := newSignalFixture(t)
	classifier := &fakeSignals{labels: map[string]Signal{"it is still broken on mobile": ReportedFailure, "great, that works": Satisfaction}}
	job := &SignalJob{Store: f.store, Events: f.events, Classifier: classifier, Now: func() time.Time { return f.now }}
	stats, err := job.Pass(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Classified != 2 || stats.Unclassified != 0 || !stats.BackfillDone {
		t.Fatalf("stats = %+v, want the two operator messages that followed agent work, and the back-fill done", stats)
	}
	rows := signalRows(t, f)
	if len(rows) != 2 || rows[f.failed.ID.String()]["label"] != "reported_failure" || rows[f.thanks.ID.String()]["label"] != "satisfaction" {
		t.Fatalf("rows = %v", rows)
	}
	for _, id := range []string{f.opening.ID.String(), f.teammate.ID.String()} {
		if _, ok := rows[id]; ok {
			t.Fatalf("event %s must not carry an operator signal (opening message / teammate report)", id)
		}
	}
	for _, m := range classifier.seen {
		if strings.Contains(m.OperatorText, "Handoff received") {
			t.Fatalf("a teammate message was sent to the signal classifier: %q", m.OperatorText)
		}
		if m.OperatorText == "it is still broken on mobile" && m.PrecedingAgentText != "Done: the list wraps now." {
			t.Fatalf("context = %q, want the agent's preceding text", m.PrecedingAgentText)
		}
	}
	for column := range rows[f.failed.ID.String()] {
		if strings.Contains(column, "text") && column != "unclassified_reason" {
			t.Fatalf("operator_signals must store no message text, found column %q", column)
		}
	}
	if rows[f.failed.ID.String()]["signal_version"] != SignalVersion {
		t.Fatalf("row version = %v, want %s", rows[f.failed.ID.String()]["signal_version"], SignalVersion)
	}
}

// The back-fill is bounded per pass and marker-gated: once the marker exists
// only recently recorded interventions are classified.
func TestSignalJobBackfillIsBoundedAndMarkerGated(t *testing.T) {
	f, _ := newSignalFixture(t)
	classifier := &fakeSignals{}
	job := &SignalJob{Store: f.store, Events: f.events, Classifier: classifier, Now: func() time.Time { return f.now }, PassMessages: 1}
	first, err := job.Pass(context.Background())
	if err != nil || first.Classified != 1 || first.BackfillDone {
		t.Fatalf("first pass = %+v err %v, want one message and the back-fill still open", first, err)
	}
	second, err := job.Pass(context.Background())
	if err != nil || second.Classified != 1 || second.BackfillDone {
		t.Fatalf("second pass = %+v err %v", second, err)
	}
	third, err := job.Pass(context.Background())
	if err != nil || third.Classified != 0 || !third.BackfillDone {
		t.Fatalf("third pass = %+v err %v, want nothing left and the marker written", third, err)
	}
	done, _ := f.store.SignalBackfillDone(context.Background(), SignalVersion)
	if !done {
		t.Fatal("the back-fill marker was not recorded")
	}
	// A new intervention recorded long ago is outside the live tail now.
	if _, err := f.store.db.ExecContext(context.Background(), `INSERT INTO operator_interventions (event_id, classifier_version, run_id, sequence, occurred_at, message_key, status, recorded_at)
VALUES ('old', ?, 'run-old', 1, '2026-01-01T00:00:00Z', 'k', 'classified', '2026-01-01T00:00:00Z')`, ClassifierVersion); err != nil {
		t.Fatal(err)
	}
	before := classifier.calls
	if stats, err := job.Pass(context.Background()); err != nil || stats.Pending != 0 || classifier.calls != before {
		t.Fatalf("after the marker a stale pending row was scanned: %+v err %v", stats, err)
	}
}

func TestSignalJobOffSwitchReadsNothingAndSendsNothing(t *testing.T) {
	f, _ := newSignalFixture(t)
	classifier := &fakeSignals{}
	on := false
	job := &SignalJob{Store: f.store, Events: f.events, Classifier: classifier, Now: func() time.Time { return f.now }, Enabled: func() bool { return on }}
	stats, err := job.Pass(context.Background())
	if err != nil || !stats.Disabled || classifier.calls != 0 || len(signalRows(t, f)) != 0 {
		t.Fatalf("switched off: stats %+v err %v calls %d rows %d", stats, err, classifier.calls, len(signalRows(t, f)))
	}
	on = true
	if stats, _ := job.Pass(context.Background()); stats.Classified != 2 {
		t.Fatalf("switched on: %+v", stats)
	}
}

func TestSignalsEnabledReadsTheEnvironment(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
	for value, want := range map[string]bool{"": true, "on": true, "off": false, "OFF": false, "0": false, "false": false, "disabled": false} {
		t.Setenv(SignalsSwitchEnv, value)
		if got := SignalsEnabled(); got != want {
			t.Errorf("%s=%q: enabled = %v, want %v", SignalsSwitchEnv, value, got, want)
		}
	}
}

// A gateway outage records unclassified rows with a retry time; they are never
// read as neutral, and are retried when due.
func TestSignalJobGatewayOutageIsUnclassifiedThenRetried(t *testing.T) {
	f, _ := newSignalFixture(t)
	classifier := &fakeSignals{down: true}
	job := &SignalJob{Store: f.store, Events: f.events, Classifier: classifier, Now: func() time.Time { return f.now }}
	if _, err := job.Pass(context.Background()); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("pass err = %v, want the gateway failure", err)
	}
	row := signalRows(t, f)[f.failed.ID.String()]
	if row["status"] != "unclassified" || row["label"] != "" || row["attempts"] != int64(1) || fmt.Sprint(row["next_attempt_at"]) == "" {
		t.Fatalf("row = %v, want unclassified with a retry time", row)
	}
	classifier.down = false
	job.Now = func() time.Time { return f.now.Add(3 * time.Hour) }
	if stats, err := job.Pass(context.Background()); err != nil || stats.Classified != 2 {
		t.Fatalf("retry pass = %+v err %v", stats, err)
	}
	if row := signalRows(t, f)[f.failed.ID.String()]; row["status"] != "classified" {
		t.Fatalf("row after retry = %v", row)
	}
}

func observation(label Signal, p float64, model, actual, mode string) SignalObservation {
	o := SignalObservation{Status: StatusClassified, Signal: label, Probability: &p, Distribution: map[Signal]float64{label: p}, SignalModel: model, ActualModel: actual}
	if mode != "" {
		o.ModeID, o.ModeRelation = mode, "applied"
	}
	return o
}

func repeatObservations(n int, o SignalObservation) []SignalObservation {
	out := make([]SignalObservation, n)
	for i := range out {
		out[i] = o
	}
	return out
}

func rateFor(cell SignalCell, label Signal) SignalRate {
	for _, r := range cell.Rates {
		if r.Signal == label {
			return r
		}
	}
	return SignalRate{}
}

// A proven label gets a rate with a range per (mode period, model) cell, set
// against the operator's own prior period outside any mode; an unproven label
// reports no count or rate, only its measured accuracy.
func TestSummarizeSignalsRatesRangesBaselineAndUnprovenLabels(t *testing.T) {
	gate := SignalGate{"winnow:e4b": {ReportedFailure: 0.5, ExplicitNegative: 0.5}}
	measured := SignalMeasurements{"winnow:e4b": {ReportedFailure: {TP: 18, Pred: 20, Gold: 24}, Correction: {TP: 5, Pred: 10, Gold: 19}}}
	var current, prior []SignalObservation
	current = append(current, repeatObservations(12, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	current = append(current, repeatObservations(28, observation(Neutral, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	current = append(current, repeatObservations(5, observation(Correction, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	current = append(current, repeatObservations(30, observation(Neutral, 0.9, "winnow:e4b", "sonnet", ""))...) // no mode, other model
	current = append(current, SignalObservation{Status: StatusUnclassified, ActualModel: "haiku", ModeID: "mode-1", ModeRelation: "applied"})
	prior = append(prior, repeatObservations(4, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", ""))...)
	prior = append(prior, repeatObservations(46, observation(Neutral, 0.9, "winnow:e4b", "haiku", ""))...)
	prior = append(prior, repeatObservations(50, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "mode-0"))...) // another mode: not the baseline

	cells := SummarizeSignals(current, prior, gate, measured)
	if len(cells) != 2 {
		t.Fatalf("cells = %d, want mode-1/haiku and no_mode/sonnet", len(cells))
	}
	mode := cells[0]
	if mode.ModeID != "mode-1" || mode.Model != "haiku" || mode.Messages != 46 || mode.Classified != 45 || mode.Coverage == nil || *mode.Coverage < 0.97 || *mode.Coverage > 0.98 {
		t.Fatalf("mode cell = %+v", mode)
	}
	failure := rateFor(mode, ReportedFailure)
	if failure.State != StateProven || failure.Count != 12 || failure.Rate == nil || *failure.Rate < 0.26 || *failure.Rate > 0.27 || failure.Low == nil || failure.High == nil || *failure.Low >= *failure.Rate || *failure.High <= *failure.Rate {
		t.Fatalf("reported failure = %+v, want 12 of 45 with a range around it", failure)
	}
	if failure.Baseline == nil || *failure.Baseline != 0.08 || failure.BaselineN != 50 || failure.Comparison != ComparisonHigher || failure.DifferenceLow == nil || *failure.DifferenceLow <= 0 {
		t.Fatalf("baseline comparison = %+v, want 4 of 50 outside any mode and a clear rise", failure)
	}
	pooled := rateFor(mode, ExplicitNegative)
	if pooled.State != StateProven || pooled.Count != 17 {
		t.Fatalf("pooled = %+v, want the 12 failures and 5 corrections, each with mass 0.9 on reported_failure or correction", pooled)
	}
	correction := rateFor(mode, Correction)
	if correction.State != StateUnproven || correction.Count != 0 || correction.Rate != nil || correction.Comparison != "" && correction.Comparison != ComparisonNotEnough || correction.Precision == nil || *correction.Precision != 0.5 {
		t.Fatalf("unproven correction = %+v, want no count or rate and its measured precision 0.5", correction)
	}
	quiet := cells[1]
	if quiet.ModeID != NoMode || quiet.Model != "sonnet" || rateFor(quiet, ReportedFailure).Baseline != nil || rateFor(quiet, ReportedFailure).Comparison != ComparisonNotEnough {
		t.Fatalf("no-baseline cell = %+v, want no comparison without a prior period of that model", quiet)
	}
}

func TestSummarizeSignalsSmallCellReportsNoRate(t *testing.T) {
	gate := SignalGate{"winnow:e4b": {ReportedFailure: 0.5}}
	cells := SummarizeSignals(repeatObservations(10, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "")), nil, gate, nil)
	rate := rateFor(cells[0], ReportedFailure)
	if rate.State != StateProven || rate.Count != 10 || rate.Rate != nil {
		t.Fatalf("rate = %+v, want the count but no rate below %d classified messages", rate, MinSignalCell)
	}
}

// A proven label for one model never admits another model's answers.
func TestSignalGateIsPerModelAndPerThreshold(t *testing.T) {
	gate := SignalGate{"winnow:e4b": {ReportedFailure: 0.6}}
	p := func(v float64) *float64 { return &v }
	if !gate.Shown("winnow:e4b", ReportedFailure, p(0.6)) || gate.Shown("winnow:e4b", ReportedFailure, p(0.59)) || gate.Shown("laya:en", ReportedFailure, p(0.99)) ||
		gate.Shown("winnow:e4b", Correction, p(0.99)) || gate.Shown("winnow:e4b", ReportedFailure, nil) {
		t.Fatal("gate admitted an answer it should not")
	}
	if !gate.Proven(ReportedFailure) || gate.Proven(Correction) {
		t.Fatal("Proven disagrees with the gate")
	}
}

// Only the explicit_negative bucket, proven, for the mode's own cell, with a
// clear rise over the operator's baseline, makes a quality finding.
func TestQualityFindingNeedsAProvenRiseInTheModesOwnCell(t *testing.T) {
	gate := SignalGate{"winnow:e4b": {ExplicitNegative: 0.4}}
	measured := SignalMeasurements{"winnow:e4b": {ExplicitNegative: {TP: 33, Pred: 41, Gold: 35}}}
	var current, prior []SignalObservation
	current = append(current, repeatObservations(15, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	current = append(current, repeatObservations(30, observation(Neutral, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	current = append(current, repeatObservations(40, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "mode-2"))...) // another mode
	prior = append(prior, repeatObservations(3, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", ""))...)
	prior = append(prior, repeatObservations(57, observation(Neutral, 0.9, "winnow:e4b", "haiku", ""))...)
	finding, ok, _ := qualityFinding(SummarizeSignals(current, prior, gate, measured), "mode-1")
	if !ok || finding.Model != "haiku" || finding.Messages != 45 || finding.BaselineMessages != 60 || finding.Rate < 0.33 || finding.Baseline != 0.05 || finding.DifferenceLow <= 0 {
		t.Fatalf("finding = %+v ok %v", finding, ok)
	}
	if finding.ClassifierPrecision < 0.80 || finding.ClassifierPrecision > 0.81 {
		t.Fatalf("precision = %v, want the measured 33/41", finding.ClassifierPrecision)
	}
	if _, ok, _ := qualityFinding(SummarizeSignals(current, prior, SignalGate{}, measured), "mode-1"); ok {
		t.Fatal("an unproven bucket made a finding")
	}
	if _, ok, _ := qualityFinding(SummarizeSignals(current, prior, gate, measured), "mode-9"); ok {
		t.Fatal("a mode with no cell made a finding")
	}
	flat := append(repeatObservations(5, observation(ReportedFailure, 0.9, "winnow:e4b", "haiku", "mode-1")), repeatObservations(40, observation(Neutral, 0.9, "winnow:e4b", "haiku", "mode-1"))...)
	if _, ok, _ := qualityFinding(SummarizeSignals(flat, prior, gate, measured), "mode-1"); ok {
		t.Fatal("a rise inside chance made a finding")
	}
}

// The model was chosen on the hold-out it was scored on, so the gate's interval is
// re-read at a level that allows for the two models tried. It must still clear the
// bar, and the single-label operating points must be the ones the precision was
// measured at, for models that have a measured precision.
func TestGateSurvivesTheSelectionAllowanceAndOperatingPointsAreTheMeasuredOnes(t *testing.T) {
	measured := SignalHoldout["winnow:e4b"][ExplicitNegative]
	if measured.TP != 33 || measured.Pred != 41 {
		t.Fatalf("the report states 33 of 41: %+v", measured)
	}
	const z = 2.2414 // two-sided 97.5%: 5% over two models
	k, n := float64(measured.TP), float64(measured.Pred)
	p := k / n
	denominator := 1 + z*z/n
	lower := (p + z*z/(2*n) - z*math.Sqrt(p*(1-p)/n+z*z/(4*n*n))) / denominator
	if lower < signalMinLower || p < signalMinPrecision {
		t.Fatalf("precision %.3f, adjusted lower bound %.3f: the gate no longer passes and the sender must go off", p, lower)
	}
	// One message fewer fails: the margin the report states.
	if (k-1)/n >= signalMinPrecision {
		t.Fatalf("the report says 32 of 41 fails the bar")
	}
	if !ProvenSignalGate.Proven(ExplicitNegative) {
		t.Fatal("the gate passes, so it must stay on")
	}
	low, at := 0.14, 0.15
	if SignalOperatingPoints.Shown("winnow:e4b", ReportedFailure, &low) || !SignalOperatingPoints.Shown("winnow:e4b", ReportedFailure, &at) {
		t.Fatal("reported_failure reads at 0.15")
	}
	high := 0.99
	if SignalOperatingPoints.Shown("laya:en", ReportedFailure, &high) {
		t.Fatal("a model with no recorded operating point is not read")
	}
}
