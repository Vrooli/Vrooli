package runannotations

import (
	"path/filepath"
	"strings"
	"testing"
)

// defaultPackFile is the pack the service loads. These tests pin the shape of
// the real labels, not fixtures, so a pack edit that changes a label's meaning,
// scope or population fails here before it reaches a running service.
var defaultPackFile = filepath.Join("..", "..", "..", ".vrooli", "annotations", "default.json")

func loadDefaultPack(t *testing.T) *CompiledPack {
	t.Helper()
	pack, err := LoadCompiled(defaultPackFile)
	if err != nil {
		t.Fatalf("load default pack: %v", err)
	}
	return pack
}

func defaultLabel(t *testing.T, pack *CompiledPack, id string) Label {
	t.Helper()
	compiled, ok := pack.Label(id)
	if !ok {
		t.Fatalf("default pack has no label %s", id)
	}
	return compiled.label
}

func TestDefaultPackDeclaresTheCurrentVocabulary(t *testing.T) {
	pack := loadDefaultPack(t)
	want := []string{
		"run.type", "run.outcome",
		"agent.discovery_slow", "work.blocked_by_tool",
		"operator.dissatisfied", "operator.pleased", "operator.impatient", "operator.valence",
		"operator.dissatisfied_run", "operator.pleased_run", "operator.impatient_run",
	}
	if got := pack.Order(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	// Superseded labels must not linger as unmeasurable vocabulary: their
	// populations or evidence could not support a gold hold-out. agent.spinning
	// is here because no counted tool fact separated it from gold and two
	// labellers agreed at kappa 0.23; it returns only as a redefined label
	// (see DECISIONS.md, WS-J1).
	for _, gone := range []string{
		"agent.spinning", "operator.frustration", "operator.satisfaction", "operator.emotion",
		"operator.frustrated", "operator.frustrated_run",
		"agent.objective_lost", "agent.scope_drift", "interaction.misunderstanding",
	} {
		if _, ok := pack.Label(gone); ok {
			t.Fatalf("superseded label %s is still declared", gone)
		}
	}
}

func TestDefaultPackMoodLabelsAreMessageScopeOverHumanText(t *testing.T) {
	pack := loadDefaultPack(t)
	for _, id := range []string{"operator.dissatisfied", "operator.pleased", "operator.impatient"} {
		label := defaultLabel(t, pack, id)
		if label.Scope != ScopeMessage || label.Method != MethodDecision {
			t.Fatalf("%s is %s/%s, want a message-scope decision", id, label.Scope, label.Method)
		}
		if label.Population != PopulationHumanOperator {
			t.Fatalf("%s population = %q, want %q", id, label.Population, PopulationHumanOperator)
		}
		if label.Question == nil || label.Question.Type != QuestionNoul {
			t.Fatalf("%s must be a noul question", id)
		}
		if !strings.Contains(label.Require, "human_message_count > 0") {
			t.Fatalf("%s require = %q, want a human-text guard", id, label.Require)
		}
		if label.Optional {
			t.Fatalf("%s must be active: it is the mood series", id)
		}
		if label.Gate.MinGold < 40 {
			t.Fatalf("%s min_gold = %d", id, label.Gate.MinGold)
		}
	}
	// Small decision models see a 512-token window. The three mood questions
	// travel in one call, so their text is bounded: it is what decides how much
	// of the message the digest budget leaves room for.
	total := 0
	for _, id := range []string{"operator.dissatisfied", "operator.pleased", "operator.impatient"} {
		q := defaultLabel(t, pack, id).Question
		total += len(id) + len(q.Instructions) + len(q.TrueDescription) + len(q.FalseDescription)
	}
	if total > 560 {
		t.Fatalf("mood question text is %d characters; it leaves a 512-token model no room for the message", total)
	}
}

func TestDefaultPackMoodRequireAdmitsOnlyPeopleWritingWords(t *testing.T) {
	pack := loadDefaultPack(t)
	compiled, _ := pack.Label("operator.dissatisfied")
	evaluator := celEvaluator{CostLimit: defaultCostLimit}
	cases := []struct {
		name     string
		humans   int
		text     string
		admitted bool
	}{
		{"a person's sentence", 3, "Keep going, I don't know why you keep stopping.", true},
		{"a short answer", 1, "No", false},
		{"an attached image", 2, `<image name=[Image #1] path="x.png"> </image> [Image #1]`, false},
		{"a delegated lineage prompt", 2, "Agent Manager lineage: you are run 1, depth 1 of max 3", false},
		{"no human text in the run", 0, "Thanks, this looks great", false},
	}
	for _, tc := range cases {
		features := RunFeatures{HumanMessageCount: tc.humans, MessageText: tc.text, MessageIndex: 1, MessageCount: tc.humans}
		got, err := evaluator.evalBool(compiled.require, features.Activation())
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.admitted {
			t.Fatalf("%s: require = %t, want %t", tc.name, got, tc.admitted)
		}
	}
}

func TestDefaultPackProvenanceSeparatesPeopleFromAgentsByLineage(t *testing.T) {
	pack := loadDefaultPack(t)
	policy := pack.Pack.Runtime.Provenance
	message := func(id, text string) MessageFact { return MessageFact{EventID: id, Text: text} }
	person := message("p", "Keep going, but try to speed things up if possible.")
	steer := message("s", "Read E1 D34 and continue.")
	cases := []struct {
		name      string
		facts     RunFacts
		wantHuman int
	}{
		{"a workflow brief", RunFacts{Workload: WorkloadFacts{Kind: "workflow_node"}, OperatorHistory: []MessageFact{person}}, 0},
		{"a scheduled brief", RunFacts{Workload: WorkloadFacts{Kind: "scheduled"}, OperatorHistory: []MessageFact{person}}, 0},
		{"a run another run spawned", RunFacts{Lineage: LineageFacts{ParentRunID: "parent", Depth: 1}, OperatorHistory: []MessageFact{steer, steer}}, 0},
		{"a run that opens with the delegation preamble", RunFacts{Lineage: LineageFacts{Opening: "Agent Manager lineage: you are run 1, depth 1 of max 3"}, OperatorHistory: []MessageFact{steer}}, 0},
		{"an imported session a person typed into", RunFacts{Lineage: LineageFacts{Imported: true}, OperatorHistory: []MessageFact{person}}, 1},
		{"an imported session that was linked under a parent", RunFacts{Lineage: LineageFacts{Imported: true, ParentRunID: "parent", Depth: 1}, OperatorHistory: []MessageFact{person}}, 1},
		{"an interactive run with no parent", RunFacts{Workload: WorkloadFacts{Kind: "interactive"}, OperatorHistory: []MessageFact{person}}, 1},
	}
	for _, tc := range cases {
		facts := tc.facts
		policy.Apply(&facts)
		if len(facts.HumanMessages) != tc.wantHuman {
			t.Fatalf("%s: human = %d, want %d", tc.name, len(facts.HumanMessages), tc.wantHuman)
		}
	}
	runtime := pack.Pack.Runtime
	if runtime.Budgets.MaxMessagesPerRun < 48 {
		t.Fatalf("max_messages_per_run = %d: the transcript reader keeps 48 messages and the series must read them all", runtime.Budgets.MaxMessagesPerRun)
	}
	if runtime.BackfillHorizonDays < 36 {
		t.Fatalf("backfill horizon = %d days: the corpus's person-written messages reach back 35 days", runtime.BackfillHorizonDays)
	}
}

func TestDefaultPackRunDecisionLabelsAreNoulQuestions(t *testing.T) {
	pack := loadDefaultPack(t)
	slow := defaultLabel(t, pack, "agent.discovery_slow")
	if slow.Question == nil || slow.Question.Type != QuestionNoul {
		t.Fatalf("agent.discovery_slow must be a noul question, got %+v", slow.Question)
	}
	// Positives are about 2 percent of candidate runs in the gold reading: the
	// label stays in the vocabulary but costs no inference until asked for.
	if !slow.Optional {
		t.Fatal("agent.discovery_slow is too rare to gate and must stay optional")
	}
	blocked := defaultLabel(t, pack, "work.blocked_by_tool")
	if blocked.Question == nil || blocked.Question.Type != QuestionNoul {
		t.Fatalf("work.blocked_by_tool must stay a noul question, got %+v", blocked.Question)
	}
	// Every measured model of 512 to 1024 tokens answers it at chance, so it
	// costs no decision capacity until a model that can read the agent's text
	// is measured (notes/J2-bakeoff.md).
	if !blocked.Optional {
		t.Fatal("work.blocked_by_tool answers at chance on every measured model and must stay optional")
	}
	if strings.TrimSpace(blocked.PriorRule) == "" {
		t.Fatal("work.blocked_by_tool lost its deterministic prior")
	}
	for _, label := range pack.Pack.Labels {
		if label.Question != nil && label.Question.Type == QuestionScore {
			t.Fatalf("%s is a score question; ordinal scales are compressed by small decision models", label.ID)
		}
	}
}

// Run mood is stored, not read through a side path: valence is a message-scope
// composite of the two message answers, and each run label is a run-scope
// composite over the run's own message observations. Nothing here is asked of a
// model, so none of them may carry a score question.
func TestDefaultPackMoodIsDerivedNotDuplicated(t *testing.T) {
	pack := loadDefaultPack(t)
	valence := defaultLabel(t, pack, "operator.valence")
	if valence.Scope != ScopeMessage || valence.Method != MethodComposite || valence.Population != PopulationHumanOperator {
		t.Fatalf("operator.valence is %s/%s/%q, want a human-operator message composite", valence.Scope, valence.Method, valence.Population)
	}
	for _, input := range []string{"operator.pleased", "operator.dissatisfied"} {
		if !strings.Contains(valence.Rule, "labels['"+input+"']") {
			t.Fatalf("operator.valence rule %q does not read %s", valence.Rule, input)
		}
	}
	for _, name := range []string{"dissatisfied", "pleased", "impatient"} {
		run := defaultLabel(t, pack, "operator."+name+"_run")
		if run.Scope != ScopeRun || run.Method != MethodComposite || run.Population != PopulationHumanOperator {
			t.Fatalf("%s_run is %s/%s/%q, want a human-operator run composite", name, run.Scope, run.Method, run.Population)
		}
		if !strings.Contains(run.Rule, "message_labels['operator."+name+"']") {
			t.Fatalf("%s_run rule %q does not read the message label", name, run.Rule)
		}
		if run.Question == nil || run.Question.Type != QuestionNoul {
			t.Fatalf("%s_run must yield a probability", name)
		}
		if !strings.Contains(run.Require, "human_message_count > 0") {
			t.Fatalf("%s_run require = %q", name, run.Require)
		}
		// The message label counts messages; only the run label feeds the
		// run-level friction measures, so friction is declared once.
		if defaultLabel(t, pack, "operator."+name).Friction.Enabled {
			t.Fatalf("operator.%s declares friction as well as its run composite", name)
		}
	}
}

// Two question wordings were corrected after a second labeller read the gold
// blind and disagreed with the first on exactly these points. The questions
// carry the meaning (small models follow the words, not the definitions), so
// the corrections are pinned.
func TestDefaultPackQuestionsCarryTheAdjudicatedReading(t *testing.T) {
	pack := loadDefaultPack(t)
	impatient := defaultLabel(t, pack, "operator.impatient").Question
	text := strings.ToLower(impatient.Instructions + " " + impatient.TrueDescription)
	if strings.Contains(text, "keep going") {
		t.Fatalf("impatient must not treat a plain 'keep going' as impatience: %q", text)
	}
	for _, want := range []string{"hurry", "wrap up", "stall"} {
		if !strings.Contains(text, want) {
			t.Fatalf("impatient question %q lost %q", text, want)
		}
	}
	blocked := defaultLabel(t, pack, "work.blocked_by_tool").Question
	text = strings.ToLower(blocked.Instructions + " " + blocked.TrueDescription)
	for _, want := range []string{"error", "refusal", "rate limit", "outage", "required step"} {
		if !strings.Contains(text, want) {
			t.Fatalf("blocked_by_tool question %q lost %q", text, want)
		}
	}
	if !strings.Contains(strings.ToLower(blocked.FalseDescription), "defect") {
		t.Fatalf("blocked_by_tool must exclude a test or review that reports a defect: %q", blocked.FalseDescription)
	}
}

func TestDefaultPackLoadsAndCompiles(t *testing.T) {
	pack := loadDefaultPack(t)
	if pack.Pack.Name != "default" || pack.Pack.Version == "" {
		t.Fatalf("identity = %s %s", pack.Pack.Name, pack.Pack.Version)
	}
	if pack.Pack.Runtime.Escalation.SameModelDuration() <= 0 {
		t.Fatal("default pack declares no same-model escalation ttl")
	}
	if len(pack.Pack.Labels) == 0 {
		t.Fatal("default pack declares no labels")
	}
	seen := map[string]bool{}
	for _, label := range pack.Pack.Labels {
		hash := pack.LabelHashes[label.ID]
		if len(hash) != 64 {
			t.Fatalf("%s hash = %q", label.ID, hash)
		}
		if seen[hash] {
			t.Fatalf("duplicate hash %s", hash)
		}
		seen[hash] = true
		if label.Gate.MinGold < 1 {
			t.Fatalf("%s has no gate", label.ID)
		}
		if strings.TrimSpace(label.Search.Facet) == "" {
			t.Fatalf("%s has no search facet", label.ID)
		}
	}
	if len(pack.PackHash) != 64 {
		t.Fatalf("pack hash = %q", pack.PackHash)
	}
}

// A run's mood is a share, not a maximum: a maximum over messages rises with how
// many messages a run has (0 of 6 runs with two messages or fewer scored
// positive against 15 of 19 with three or more), so it would measure how long
// the operator talked. The mean calibrated probability is the estimated share of
// messages that showed the mood, whatever their number.
func TestDefaultPackRunMoodIsAShareNotAMaximum(t *testing.T) {
	pack := loadDefaultPack(t)
	evaluator := celEvaluator{CostLimit: defaultCostLimit}
	value := func(name string, count int, probabilities map[string]float64) float64 {
		t.Helper()
		compiled, _ := pack.Label("operator." + name + "_run")
		features := RunFeatures{HumanMessageCount: count}
		activation := features.Activation()
		activation["labels"] = map[string]float64{}
		activation["present"] = map[string]bool{}
		activation["message_labels"] = map[string]map[string]float64{"operator." + name: probabilities}
		got, err := evaluator.evalNumber(compiled.composite, activation)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return got
	}
	for _, name := range []string{"dissatisfied", "pleased", "impatient"} {
		run := defaultLabel(t, pack, "operator."+name+"_run")
		if strings.Contains(run.Rule, ".max") || !strings.Contains(run.Rule, ".mean") {
			t.Fatalf("%s_run rule %q must read the mean over messages", name, run.Rule)
		}
		// The same share gives the same value at two messages and at forty.
		few := value(name, 2, map[string]float64{"count": 2, "mean": 0.1, "max": 0.1})
		many := value(name, 40, map[string]float64{"count": 40, "mean": 0.1, "max": 0.1})
		if few != many {
			t.Fatalf("%s_run depends on the message count: %v at 2, %v at 40", name, few, many)
		}
		// One dissatisfied message among many is a small share; the same message
		// alone in a run is the whole run.
		lone := value(name, 1, map[string]float64{"count": 1, "mean": 0.9, "max": 0.9})
		diluted := value(name, 20, map[string]float64{"count": 20, "mean": 0.9 / 20, "max": 0.9})
		if lone <= diluted {
			t.Fatalf("%s_run: one message of one scored %v, one of twenty scored %v", name, lone, diluted)
		}
		// The declared thresholds sit above a run that is only noise at the
		// message base rate and below a run that is clearly dominated by the mood.
		threshold := *run.Search.MinProbability
		if diluted >= threshold || lone < threshold {
			t.Fatalf("%s_run threshold %v does not separate a diluted run (%v) from a dominated one (%v)", name, threshold, diluted, lone)
		}
		if run.Friction.Enabled && (run.Friction.MinProbability == nil || *run.Friction.MinProbability != threshold) {
			t.Fatalf("%s_run friction and search thresholds differ", name)
		}
	}
}

// Most of what operators write when they are unhappy is mild pushback (34 of 49
// gold positives; 7 are plain anger). Typed decision models follow option names
// over definitions, so the label is named, asked and described for pushback; a
// name-led "frustrated" option read as anger and missed the gold.
func TestDefaultPackDissatisfactionIsNamedAndAskedForPushback(t *testing.T) {
	pack := loadDefaultPack(t)
	label := defaultLabel(t, pack, "operator.dissatisfied")
	if label.Question == nil || label.Question.Type != QuestionNoul {
		t.Fatalf("operator.dissatisfied must be a noul question")
	}
	if !strings.HasPrefix(label.Question.TrueDescription, "Dissatisfied") || !strings.HasPrefix(label.Question.FalseDescription, "Not dissatisfied") {
		t.Fatalf("option names must lead with the class: %q / %q", label.Question.TrueDescription, label.Question.FalseDescription)
	}
	for _, want := range []string{"pushes back", "mildly or strongly"} {
		if !strings.Contains(label.Question.TrueDescription, want) {
			t.Fatalf("true_description %q must say %q so mild pushback counts", label.Question.TrueDescription, want)
		}
	}
	for _, id := range []string{"operator.dissatisfied", "operator.dissatisfied_run", "operator.valence"} {
		compiled := defaultLabel(t, pack, id)
		text := compiled.Description
		if compiled.Question != nil {
			text += compiled.Question.Instructions + compiled.Question.TrueDescription + compiled.Question.FalseDescription
		}
		if strings.Contains(strings.ToLower(text), "frustrat") {
			t.Fatalf("%s still speaks of frustration: the label is for dissatisfaction", id)
		}
	}
}
