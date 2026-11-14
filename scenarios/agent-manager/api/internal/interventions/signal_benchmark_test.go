package interventions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// signalCorpusPath is the operator-signal corpus: random turns from sessions in
// neither the intervention corpus nor its validation set, split by session.
const signalCorpusPath = "../runsignal/testdata/interventions/signal-turns.jsonl"

// Gate bars, fixed before any model was scored (docs/reference/operational-stats.md).
const (
	signalMinGold      = 10
	signalMinPrecision = 0.8
	signalMinRecall    = 0.7
	signalMinLower     = 0.6
	signalMinKappa     = 0.6
)

type signalRow struct {
	ID                 string `json:"id"`
	Session            string `json:"session"`
	Split              string `json:"split"`
	Label              string `json:"label"`
	LabelB             string `json:"label_b"`
	Text               string `json:"text"`
	PrecedingAgentText string `json:"preceding_agent_text"`
}

func loadSignalRows(t *testing.T) []signalRow {
	t.Helper()
	file, err := os.Open(signalCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var rows []signalRow
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var row signalRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

func cohenKappa(a, b []string) (agreement, kappa float64) {
	n := float64(len(a))
	counts := map[string][2]float64{}
	same := 0.0
	for i := range a {
		if a[i] == b[i] {
			same++
		}
		x := counts[a[i]]
		x[0]++
		counts[a[i]] = x
		y := counts[b[i]]
		y[1]++
		counts[b[i]] = y
	}
	expected := 0.0
	for _, c := range counts {
		expected += c[0] / n * c[1] / n
	}
	agreement = same / n
	return agreement, (agreement - expected) / (1 - expected)
}

// TestSignalCorpus checks the corpus the gate is scored on: at least 100
// labelled turns, a session-disjoint split, only known labels, and an
// independent second labelling that meets the kappa bar.
func TestSignalCorpus(t *testing.T) {
	rows := loadSignalRows(t)
	if len(rows) < 100 {
		t.Fatalf("%d signal turns, want at least 100", len(rows))
	}
	splits := map[string]string{}
	var a, b []string
	for _, row := range rows {
		if !knownSignal(Signal(row.Label)) || !knownSignal(Signal(row.LabelB)) {
			t.Fatalf("row %s has an unknown label %q / %q", row.ID, row.Label, row.LabelB)
		}
		if row.PrecedingAgentText == "" {
			t.Errorf("row %s has no context", row.ID)
		}
		if prior, ok := splits[row.Session]; ok && prior != row.Split {
			t.Errorf("session %s is in both splits", row.Session)
		}
		splits[row.Session] = row.Split
		a, b = append(a, row.Label), append(b, row.LabelB)
	}
	agreement, kappa := cohenKappa(a, b)
	t.Logf("%d turns, %d sessions; labeller agreement %.3f, kappa %.3f", len(rows), len(splits), agreement, kappa)
	if kappa < signalMinKappa {
		t.Errorf("kappa %.3f below the gate's %.2f", kappa, signalMinKappa)
	}
}

// signalPredicted is the production rule: a message counts as the label when
// the model's probability for that label reaches the threshold, whether or not
// it is the most likely label (chosen on split=train, where requiring the
// argmax capped reported_failure recall at 0.62). Below 0.5 two labels can both
// be predicted; a label's count is its own.
func loadSignalDump(t *testing.T, path string) map[string]SignalAnswer {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	out := map[string]SignalAnswer{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var row struct {
			ID    string             `json:"id"`
			Got   Signal             `json:"got"`
			P     map[Signal]float64 `json:"p"`
			Model string             `json:"model"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		out[row.ID] = SignalAnswer{Status: StatusClassified, Signal: row.Got, Probabilities: row.P, Model: row.Model, Method: "DECISION_MODEL"}
	}
	return out
}

func signalPredicted(answer SignalAnswer, label Signal, threshold float64) bool {
	if answer.Status != StatusClassified {
		return false
	}
	if label == ExplicitNegative {
		mass := answer.ExplicitNegativeMass()
		return mass != nil && *mass >= threshold
	}
	p, ok := answer.Probabilities[label]
	if len(answer.Probabilities) == 0 {
		return answer.Signal == label && threshold == 0
	}
	return ok && p >= threshold
}

func signalGold(row signalRow, label Signal) bool {
	if label == ExplicitNegative {
		return row.Label == string(ReportedFailure) || row.Label == string(Correction)
	}
	return row.Label == string(label)
}

func signalScore(rows []signalRow, answers []SignalAnswer, split string, label Signal, threshold float64) confusion {
	var c confusion
	for i, row := range rows {
		if row.Split != split {
			continue
		}
		hit, gold := signalPredicted(answers[i], label, threshold), signalGold(row, label)
		if hit {
			c.pred++
		}
		if gold {
			c.gold++
		}
		if hit && gold {
			c.tp++
		}
	}
	return c
}

// pickSignalThreshold chooses on split=train: among thresholds with recall >=
// the bar, the one with the highest precision (lowest on ties). ok is false
// when no threshold reaches the recall bar; the threshold is then the best F1.
func pickSignalThreshold(rows []signalRow, answers []SignalAnswer, label Signal) (threshold float64, ok bool) {
	bestPrecision, bestF1, f1Threshold := -1.0, -1.0, 0.0
	for step := 2; step <= 19; step++ {
		t := float64(step) * 0.05
		c := signalScore(rows, answers, "train", label, t)
		if c.pred == 0 || c.gold == 0 {
			continue
		}
		precision, recall := float64(c.tp)/float64(c.pred), float64(c.tp)/float64(c.gold)
		if f1 := 2 * precision * recall / (precision + recall + 1e-9); f1 > bestF1 {
			bestF1, f1Threshold = f1, t
		}
		if recall >= signalMinRecall && precision > bestPrecision+1e-9 {
			bestPrecision, threshold, ok = precision, t, true
		}
	}
	if !ok {
		return f1Threshold, false
	}
	return threshold, true
}

func (c confusion) signalMeets() bool {
	return c.gold >= signalMinGold && c.pred > 0 &&
		float64(c.tp)/float64(c.pred) >= signalMinPrecision && float64(c.tp)/float64(c.gold) >= signalMinRecall &&
		wilsonLower(c.tp, c.pred) >= signalMinLower
}

// TestLiveSignalBenchmark scores the production signal classifier against the
// live ai-gateway. AGENT_MANAGER_LIVE_SIGNAL=train prints train numbers only
// (for wording and threshold work); =full also scores the hold-out, once;
// =holdout is full with the train answers taken from an earlier train dump.
// AGENT_MANAGER_LIVE_DECIDE_ROLE picks the role (decide.accurate answers with
// Winnow-E4B, decide.fast with Laya). Skipped otherwise.
func TestLiveSignalBenchmark(t *testing.T) {
	mode := os.Getenv("AGENT_MANAGER_LIVE_SIGNAL")
	if mode == "" {
		t.Skip("set AGENT_MANAGER_LIVE_SIGNAL=train|full to benchmark against the live ai-gateway")
	}
	rows := loadSignalRows(t)
	if mode == "train" {
		var train []signalRow
		for _, row := range rows {
			if row.Split == "train" {
				train = append(train, row)
			}
		}
		rows = train // the hold-out is neither sent to a model nor read while tuning
	}
	if limit, _ := strconv.Atoi(os.Getenv("AGENT_MANAGER_LIVE_SIGNAL_LIMIT")); limit > 0 && limit < len(rows) {
		rows = rows[:limit] // smoke runs
	}
	classifier := NewClassifier()
	classifier.Role = DefaultRole
	classifier.Concurrency = 1
	classifier.HTTPClient = &http.Client{Timeout: 30 * time.Minute}
	if role := os.Getenv("AGENT_MANAGER_LIVE_DECIDE_ROLE"); role != "" {
		classifier.Role = role
	}
	// holdout mode reuses the train answers of an earlier train run of the same
	// signal version (AGENT_MANAGER_LIVE_SIGNAL_TRAIN_DUMP) and sends only the
	// hold-out rows to the model.
	cached := map[string]SignalAnswer{}
	if mode == "holdout" {
		cached = loadSignalDump(t, os.Getenv("AGENT_MANAGER_LIVE_SIGNAL_TRAIN_DUMP"))
	}
	all := make([]SignalAnswer, len(rows))
	var todo []int
	for i, row := range rows {
		if answer, ok := cached[row.ID]; ok && row.Split == "train" {
			all[i] = answer
		} else {
			todo = append(todo, i)
		}
	}
	messages := make([]Message, len(todo))
	for k, i := range todo {
		messages[k] = Message{EventID: rows[i].ID, OperatorText: rows[i].Text, PrecedingAgentText: rows[i].PrecedingAgentText}
	}
	started := time.Now()
	answers, err := classifier.ClassifySignals(context.Background(), messages)
	// A batch the gateway dropped (a restart, a model reload) is asked again,
	// only those messages, up to three times; the run is long on CPU.
	for attempt := 0; attempt < 3; attempt++ {
		var again []int
		for i, a := range answers {
			if a.Status != StatusClassified {
				again = append(again, i)
			}
		}
		if len(again) == 0 {
			break
		}
		retry := make([]Message, len(again))
		for k, i := range again {
			retry[k] = messages[i]
		}
		t.Logf("retrying %d unclassified messages after: %v", len(again), err)
		got, retryErr := classifier.ClassifySignals(context.Background(), retry)
		for k, i := range again {
			answers[i] = got[k]
		}
		err = retryErr
	}
	unclassified := 0
	for _, a := range answers {
		if a.Status != StatusClassified {
			unclassified++
		}
	}
	if unclassified > 0 {
		t.Fatalf("%d of %d messages stayed unclassified: %v", unclassified, len(answers), err)
	}
	for k, i := range todo {
		all[i] = answers[k]
	}
	answers = all
	models := map[string]int{}
	for _, a := range answers {
		if a.Status == StatusClassified {
			models[a.Model+" ("+a.Method+")"]++
		}
	}
	t.Logf("version %s role %q: %v in %s", SignalVersion, classifier.Role, models, time.Since(started).Round(time.Millisecond))
	if os.Getenv("AGENT_MANAGER_LIVE_SIGNAL_DUMP") != "" {
		dump, _ := os.Create(os.Getenv("AGENT_MANAGER_LIVE_SIGNAL_DUMP"))
		defer dump.Close()
		for i, a := range answers {
			body, _ := json.Marshal(map[string]any{"id": rows[i].ID, "split": rows[i].Split, "gold": rows[i].Label, "got": a.Signal, "p": a.Probabilities, "model": a.Model})
			fmt.Fprintln(dump, string(body))
		}
	}
	splits := []string{"train"}
	if mode == "full" || mode == "holdout" {
		splits = append(splits, "holdout")
	}
	labels := append(Signals(), ExplicitNegative)
	var report strings.Builder
	for _, label := range labels {
		threshold, reachable := pickSignalThreshold(rows, answers, label)
		fmt.Fprintf(&report, "\n%-18s train-picked threshold p >= %.2f (recall bar reachable on train: %v)\n", label, threshold, reachable)
		for _, split := range splits {
			c := signalScore(rows, answers, split, label, threshold)
			verdict := ""
			if split == "holdout" {
				verdict = "  gate: not proven"
				if reachable && c.signalMeets() {
					verdict = "  gate: PROVEN"
				}
			}
			fmt.Fprintf(&report, "  %-8s tp/pred/gold %3d/%3d/%3d  P %-18s R %-18s%s\n", split, c.tp, c.pred, c.gold, interval(c.tp, c.pred), interval(c.tp, c.gold), verdict)
		}
	}
	t.Log(report.String())
}
