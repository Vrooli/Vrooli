package interventions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

// corpusPath is the P4-W4a labelled operator-message corpus (split train/holdout);
// validationPath is the untouched fresh random validation set (split validation).
// Nothing may be tuned on the validation rows.
const (
	corpusPath     = "../runsignal/testdata/interventions/operator-messages.jsonl"
	validationPath = "../runsignal/testdata/interventions/validation-random.jsonl"
)

type corpusRow struct {
	ID                 string `json:"id"`
	Source             string `json:"source"`
	Category           string `json:"category"`
	Split              string `json:"split"`
	SampledBy          string `json:"sampled_by"`
	Text               string `json:"text"`
	PrecedingAgentText string `json:"preceding_agent_text"`
}

// TestLiveInterventionBenchmark runs the production classifier against the
// live ai-gateway and applies the P4-A4 proof route (amendment 2): thresholds
// are chosen on split=train only; a category is shown iff, at its threshold,
// precision >= 0.8 and recall >= 0.7 on split=holdout AND on the validation set,
// and the validation set has >= 10 gold rows for it (fewer cannot evidence
// recall, so such a category may only be a hint inside the pooled bucket). The
// pooled "operator friction (category unproven)" bucket is shown iff the
// probability mass on friction categories, at the train-picked threshold, meets
// the bars on both sets. ProvenGate must equal the measured result for the
// model that answered. Network: set AGENT_MANAGER_LIVE_DECIDE=1 (optionally
// AGENT_MANAGER_LIVE_DECIDE_ROLE); it is skipped otherwise.
func TestLiveInterventionBenchmark(t *testing.T) {
	if os.Getenv("AGENT_MANAGER_LIVE_DECIDE") == "" {
		t.Skip("set AGENT_MANAGER_LIVE_DECIDE=1 to benchmark against the live ai-gateway")
	}
	rows := append(loadRows(t, corpusPath), loadRows(t, validationPath)...)
	classifier := NewClassifier()
	classifier.Role = DefaultRole
	if role := os.Getenv("AGENT_MANAGER_LIVE_DECIDE_ROLE"); role != "" {
		classifier.Role = role
	}
	messages := make([]Message, len(rows))
	for i, row := range rows {
		messages[i] = Message{EventID: row.ID, OperatorText: row.Text, PrecedingAgentText: row.PrecedingAgentText}
	}
	started := time.Now()
	answers, err := classifier.Classify(context.Background(), messages)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	models := map[string]int{}
	answered := 0
	for _, answer := range answers {
		if answer.Status == StatusClassified {
			answered++
			models[answer.Model+" ("+answer.Method+")"]++
		}
	}
	t.Logf("role %q: %d/%d answered in %s (%.0f ms/message end to end, %d concurrent batches of <= %d); answering models %v",
		classifier.Role, answered, len(rows), elapsed.Round(time.Millisecond), float64(elapsed.Milliseconds())/float64(len(rows)), DefaultConcurrency, MaxBatch, models)
	if float64(answered) < 0.95*float64(len(rows)) {
		t.Fatalf("only %d of %d messages answered; the gateway or its decision models are degraded", answered, len(rows))
	}

	thresholds := map[Category]*float64{}
	var report strings.Builder
	fmt.Fprintf(&report, "\nTrain-picked thresholds (max precision with recall >= 0.7 on split=train):\n")
	for _, category := range frictionCategories() {
		thresholds[category] = pickThreshold(rows, answers, category)
		if thresholds[category] == nil {
			fmt.Fprintf(&report, "  %-20s unreachable (no threshold gives recall >= 0.7 on train)\n", category)
		} else {
			fmt.Fprintf(&report, "  %-20s p >= %.3f\n", category, *thresholds[category])
		}
	}
	measured := map[Category]bool{}
	pooledThreshold := pickPooledThreshold(rows, answers)
	if pooledThreshold == nil {
		fmt.Fprintf(&report, "  %-20s unreachable\n", "pooled friction")
	} else {
		fmt.Fprintf(&report, "  %-20s friction mass >= %.3f\n", "pooled friction", *pooledThreshold)
	}
	sets := []struct {
		name string
		keep func(corpusRow) bool
	}{
		{"holdout", func(r corpusRow) bool { return r.Split == "holdout" }},
		{"validation", func(r corpusRow) bool { return r.Split == "validation" }},
	}
	stats := map[string]map[Category]confusion{}
	for _, set := range sets {
		stats[set.name] = map[Category]confusion{}
		fmt.Fprintf(&report, "\n%s (argmax category; train threshold where reachable):\n%-20s %4s %4s %4s  %-18s %-18s\n", set.name, "category", "tp", "pred", "gold", "precision [95%]", "recall [95%]")
		for _, category := range append(frictionCategories(), "any_friction") {
			c := score(rows, answers, set.keep, category, thresholds)
			stats[set.name][category] = c
			fmt.Fprintf(&report, "%-20s %4d %4d %4d  %-18s %-18s\n", category, c.tp, c.pred, c.gold, interval(c.tp, c.pred), interval(c.tp, c.gold))
		}
		pooled := scorePooled(rows, answers, set.keep, pooledThreshold)
		stats[set.name]["pooled"] = pooled
		fmt.Fprintf(&report, "%-20s %4d %4d %4d  %-18s %-18s\n", "pooled friction", pooled.tp, pooled.pred, pooled.gold, interval(pooled.tp, pooled.pred), interval(pooled.tp, pooled.gold))
	}
	fmt.Fprintf(&report, "\nP4-A4 gate (amendment 2):\n")
	for _, category := range frictionCategories() {
		hold, validation := stats["holdout"][category], stats["validation"][category]
		shown := thresholds[category] != nil && hold.meets() && validation.meets() && validation.gold >= minValidationGold && (hold.gold >= 10 || wilsonLower(hold.tp, hold.pred) >= 0.6)
		measured[category] = shown
		verdict := "not shown"
		switch {
		case shown:
			verdict = "SHOWN"
		case validation.gold < minValidationGold:
			verdict = fmt.Sprintf("not shown individually (validation gold %d < %d; hint in the pooled bucket only)", validation.gold, minValidationGold)
		}
		fmt.Fprintf(&report, "  %-20s %s\n", category, verdict)
	}
	pooledShown := pooledThreshold != nil && stats["holdout"]["pooled"].meets() && stats["validation"]["pooled"].meets()
	fmt.Fprintf(&report, "  %-20s %v\n", "operator friction (category unproven)", pooledShown)
	t.Log(report.String())

	dominant, most := "", 0
	for _, answer := range answers {
		if answer.Status == StatusClassified && answer.Method == "DECISION_MODEL" {
			if n := countModel(answers, answer.Model); n > most {
				dominant, most = answer.Model, n
			}
		}
	}
	for _, category := range frictionCategories() {
		threshold, proven := ProvenGate[dominant][category]
		if proven != measured[category] {
			t.Errorf("ProvenGate[%q][%s] = %v but the benchmark measured shown=%v; update ProvenGate", dominant, category, proven, measured[category])
		}
		if proven && thresholds[category] != nil && math.Abs(threshold-*thresholds[category]) > 1e-9 {
			t.Errorf("ProvenGate[%q][%s] threshold %.3f but the train-picked threshold is %.3f", dominant, category, threshold, *thresholds[category])
		}
	}
	threshold, proven := ProvenGate[dominant][PooledFriction]
	if proven != pooledShown {
		t.Errorf("ProvenGate[%q] pooled bucket = %v but the benchmark measured shown=%v; update ProvenGate", dominant, proven, pooledShown)
	}
	if proven && pooledThreshold != nil && math.Abs(threshold-*pooledThreshold) > 1e-9 {
		t.Errorf("ProvenGate[%q] pooled threshold %.3f but the train-picked friction mass threshold is %.3f", dominant, threshold, *pooledThreshold)
	}
}

// minValidationGold is the fewest validation rows of a category from which
// recall can be evidenced.
const minValidationGold = 10

func loadRows(t *testing.T, path string) []corpusRow {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var rows []corpusRow
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var row corpusRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		if row.Source == "operator" {
			rows = append(rows, row)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

func frictionCategories() []Category {
	var out []Category
	for _, category := range Categories() {
		if category.Friction() {
			out = append(out, category)
		}
	}
	return out
}

func countModel(answers []Answer, model string) int {
	n := 0
	for _, answer := range answers {
		if answer.Model == model {
			n++
		}
	}
	return n
}

// predicted is the category an answer counts as under the thresholds: a
// friction category below its threshold (or without one) is not predicted.
func predicted(answer Answer, thresholds map[Category]*float64) Category {
	if answer.Status != StatusClassified {
		return ""
	}
	if !answer.Category.Friction() {
		return answer.Category
	}
	threshold := thresholds[answer.Category]
	if threshold == nil {
		return answer.Category // argmax when no threshold is reachable
	}
	if p := answer.Probability(); p != nil && *p < *threshold {
		return ""
	}
	return answer.Category
}

// pickThreshold returns the probability threshold with the highest train
// precision among those keeping train recall >= 0.7, or nil if none does.
func pickThreshold(rows []corpusRow, answers []Answer, category Category) *float64 {
	var best *float64
	bestPrecision := -1.0
	for step := 0; step <= 38; step++ {
		threshold := 0.05 + 0.025*float64(step)
		tp, pred, gold := 0, 0, 0
		for i, row := range rows {
			if row.Split != "train" {
				continue
			}
			p := answers[i].Probability()
			hit := answers[i].Category == category && (p == nil || *p >= threshold)
			isGold := Category(row.Category) == category
			if hit {
				pred++
			}
			if isGold {
				gold++
			}
			if hit && isGold {
				tp++
			}
		}
		if gold == 0 || pred == 0 || float64(tp)/float64(gold) < 0.7 {
			continue
		}
		if precision := float64(tp) / float64(pred); precision > bestPrecision {
			value := threshold
			best, bestPrecision = &value, precision
		}
	}
	return best
}

type confusion struct{ tp, pred, gold int }

func (c confusion) meets() bool {
	return c.pred > 0 && c.gold > 0 && float64(c.tp)/float64(c.pred) >= 0.8 && float64(c.tp)/float64(c.gold) >= 0.7
}

func score(rows []corpusRow, answers []Answer, keep func(corpusRow) bool, category Category, thresholds map[Category]*float64) confusion {
	var c confusion
	for i, row := range rows {
		if !keep(row) {
			continue
		}
		got := predicted(answers[i], thresholds)
		gold := Category(row.Category)
		var hit, isGold bool
		if category == "any_friction" {
			hit, isGold = got.Friction(), gold.Friction()
		} else {
			hit, isGold = got == category, gold == category
		}
		if hit {
			c.pred++
		}
		if isGold {
			c.gold++
		}
		if hit && isGold {
			c.tp++
		}
	}
	return c
}

func wilsonLower(k, n int) float64 { lo, _ := wilson(k, n); return lo }

func wilson(k, n int) (float64, float64) {
	if n == 0 {
		return math.NaN(), math.NaN()
	}
	const z = 1.96
	p, nf := float64(k)/float64(n), float64(n)
	d := 1 + z*z/nf
	c := (p + z*z/(2*nf)) / d
	h := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / d
	return math.Max(0, c-h), math.Min(1, c+h)
}

func interval(k, n int) string {
	if n == 0 {
		return "-"
	}
	lo, hi := wilson(k, n)
	return fmt.Sprintf("%.2f [%.2f,%.2f]", float64(k)/float64(n), lo, hi)
}

// frictionMass is the probability the answer puts on every friction category.
func frictionMass(answer Answer) (float64, bool) {
	if answer.Status != StatusClassified {
		return 0, false
	}
	mass := answer.FrictionMass()
	if mass == nil {
		return 0, false
	}
	return *mass, true
}

// pickPooledThreshold is pickThreshold for "any friction category" using the
// friction probability mass.
func pickPooledThreshold(rows []corpusRow, answers []Answer) *float64 {
	var best *float64
	bestPrecision := -1.0
	for step := 0; step <= 38; step++ {
		threshold := 0.05 + 0.025*float64(step)
		var c confusion
		for i, row := range rows {
			if row.Split != "train" {
				continue
			}
			mass, ok := frictionMass(answers[i])
			hit := ok && mass >= threshold
			if hit {
				c.pred++
			}
			if Category(row.Category).Friction() {
				c.gold++
				if hit {
					c.tp++
				}
			}
		}
		if c.gold == 0 || c.pred == 0 || float64(c.tp)/float64(c.gold) < 0.7 {
			continue
		}
		if precision := float64(c.tp) / float64(c.pred); precision > bestPrecision {
			value := threshold
			best, bestPrecision = &value, precision
		}
	}
	return best
}

func scorePooled(rows []corpusRow, answers []Answer, keep func(corpusRow) bool, threshold *float64) confusion {
	var c confusion
	for i, row := range rows {
		if !keep(row) || threshold == nil {
			continue
		}
		mass, ok := frictionMass(answers[i])
		hit := ok && mass >= *threshold
		if hit {
			c.pred++
		}
		if Category(row.Category).Friction() {
			c.gold++
			if hit {
				c.tp++
			}
		}
	}
	return c
}
