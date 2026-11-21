package runannotations

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/toolfacts"
)

const messageLabels = `[
  {"id":"operator.mood","group":"operator","scope":"message","method":"decision","population":"human_operator",
   "question":{"type":"score","instructions":"how does the operator feel?","levels":["-2","-1","0","+1","+2"],"offset":-2},
   "evidence":[{"source":"operator_messages","field":"text"}],
   "gate":{"precision":0.7,"recall":0.5,"min_gold":2},
   "search":{"facet":"label:operator.mood","min_probability":0.6},
   "friction":{"enabled":true,"min_probability":0.6,"max_score":-1,"cost_unit":"message","importance":"medium"},
   "compatibility":{"change":"breaking"}},
  {"id":"run.humans","group":"run","scope":"run","method":"deterministic","population":"human_operator",
   "rule":"human_message_count > 0 ? 'human' : 'brief'","evidence":[{"source":"workload","field":"kind"}],
   "gate":{"precision":0.7,"recall":0.5,"min_gold":2},
   "search":{"facet":"label:run.humans","min_probability":0.6},"compatibility":{"change":"breaking"}}
]`

var messageBase = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func conversation(id, kind string, texts ...string) RunFacts {
	facts := runFact(id)
	facts.Workload = WorkloadFacts{Kind: kind}
	facts.OperatorMessages = nil
	for index, text := range texts {
		message := MessageFact{
			EventID: id + "-m" + string(rune('1'+index)), At: messageBase.Add(time.Duration(index) * time.Hour), Text: text,
		}
		if index == 0 {
			message.Seed = true
		}
		facts.OperatorHistory = append(facts.OperatorHistory, message)
	}
	facts.OperatorMessages = facts.OperatorHistory
	facts.AgentText = []MessageFact{
		{EventID: id + "-a1", At: messageBase.Add(30 * time.Minute), Text: "I changed the file"},
		{EventID: id + "-a2", At: messageBase.Add(90 * time.Minute), Text: "now it compiles"},
	}
	return facts
}

func TestProvenanceSeparatesAPersonFromAWorkflowBrief(t *testing.T) {
	policy := Provenance{BriefWorkloads: []string{"workflow_node", "scheduled"}, SeedBriefWorkloads: []string{"adhoc"}}
	cases := []struct {
		name         string
		facts        RunFacts
		human, brief int
	}{
		{"a workflow node run is all brief", conversation("w", "workflow_node", "implement the task"), 0, 1},
		{"a scheduled run is all brief however many turns", conversation("s", "scheduled", "tick", "tick", "tick"), 0, 3},
		{"an ad hoc run's seed is a brief but later turns are a person", conversation("a", "adhoc", "agent delegated this", "no, wrong file", "thanks"), 2, 1},
		{"an interactive run is a person throughout", conversation("i", "interactive", "hi", "fix it", "great"), 3, 0},
		{"an imported session with no workload kind is a person", conversation("x", "", "hello"), 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := tc.facts
			policy.Apply(&facts)
			if len(facts.HumanMessages) != tc.human || facts.BriefMessages != tc.brief {
				t.Fatalf("human %d brief %d, want %d/%d", len(facts.HumanMessages), facts.BriefMessages, tc.human, tc.brief)
			}
			features := FeaturesFromFacts(facts)
			if features.HumanMessageCount != tc.human || features.BriefMessageCount != tc.brief {
				t.Fatalf("features = %d/%d", features.HumanMessageCount, features.BriefMessageCount)
			}
			for _, message := range facts.HumanMessages {
				if !message.Human {
					t.Fatalf("a human message is not flagged: %+v", message)
				}
			}
		})
	}
}

func TestAnEmptyProvenancePolicyReadsEveryMessageAsAPerson(t *testing.T) {
	// The policy is entirely pack data; the engine carries no word list of its
	// own, so a pack that declares none treats operator text as typed.
	pack, err := ParsePack([]byte(`{"pack":{"name":"t","version":"1","runtime":{"mode":"passive"},"labels":` + messageLabels + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	facts := conversation("w", "workflow_node", "implement the task", "# E3 do it")
	pack.Runtime.Provenance.Apply(&facts)
	if len(facts.HumanMessages) != 2 || facts.BriefMessages != 0 {
		t.Fatalf("human %d brief %d, want every message a person's under an empty policy", len(facts.HumanMessages), facts.BriefMessages)
	}
}

func TestHumanProvenanceIsUsableFromPackCEL(t *testing.T) {
	facts := []RunFacts{conversation("w", "workflow_node", "implement the task"), conversation("i", "interactive", "fix it")}
	service, _ := testService(t, messageLabels, facts, &fakeDecision{}, Options{})
	ctx := context.Background()
	for _, id := range []string{"w", "i"} {
		if _, err := service.Annotate(ctx, id, AnnotateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for id, want := range map[string]string{"w": "brief", "i": "human"} {
		view, err := service.Run(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		for _, label := range view.Labels {
			if label.ID == "run.humans" {
				got = label.Value.Text
			}
		}
		if got != want {
			t.Fatalf("%s: run.humans = %q, want %q", id, got, want)
		}
	}
}

func TestMessageScopeAnswersOncePerHumanMessageWithItsTimestamp(t *testing.T) {
	facts := []RunFacts{
		conversation("chat", "interactive", "this keeps failing", "no, that is wrong", "ok that works, thanks"),
		conversation("flow", "workflow_node", "implement the feature per the brief"),
	}
	decision := &fakeDecision{}
	service, store := testService(t, messageLabels, facts, decision, Options{})
	ctx := context.Background()
	result, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"chat", "flow"}, TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	byRun := map[string]RunView{}
	for _, run := range result.Runs {
		byRun[run.RunID] = run
	}
	if len(byRun["flow"].Messages) != 0 {
		t.Fatalf("a workflow brief was answered as an operator message: %+v", byRun["flow"].Messages)
	}
	chat := byRun["chat"].Messages
	if len(chat) != 3 {
		t.Fatalf("messages = %d, want one per human message", len(chat))
	}
	for index, message := range chat {
		if message.At == nil || !message.At.Equal(messageBase.Add(time.Duration(index)*time.Hour)) || len(message.Labels) != 1 || message.Labels[0].ID != "operator.mood" {
			t.Fatalf("message %d = %+v", index, message)
		}
		if message.Labels[0].State != StateObserved || message.Labels[0].OccurredAt == nil {
			t.Fatalf("message %d label = %+v", index, message.Labels[0])
		}
	}
	// Every state a model read carries one message, not the whole conversation.
	seen := 0
	for _, call := range decision.calls {
		for _, item := range call.GetItems() {
			if strings.Contains(item.GetStateJson(), `"message":{"index"`) {
				seen++
				if strings.Contains(item.GetStateJson(), "that works, thanks") && strings.Contains(item.GetStateJson(), "this keeps failing") {
					t.Fatalf("a message state carried two messages: %s", item.GetStateJson())
				}
			}
		}
	}
	if seen != 3 {
		t.Fatalf("message states sent = %d, want 3", seen)
	}
	rows, err := store.LatestMessagesForRun(ctx, "chat")
	if err != nil || len(rows) != 3 || rows[0].RunID != "chat" || rows[0].ScopeKind != ScopeMessage || rows[0].OccurredAt.IsZero() {
		t.Fatalf("stored rows = %+v (%v)", rows, err)
	}
	if view, err := service.Run(ctx, "chat"); err != nil || len(view.Messages) != 3 {
		t.Fatalf("read view = %+v (%v), want the stored messages", view.Messages, err)
	}
	if _, err := service.Run(ctx, "flow"); err != nil {
		t.Fatal(err)
	}

	// A settled message answer is reused: no second model call.
	before := len(decision.calls)
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil || len(decision.calls) != before {
		t.Fatalf("settled message answers were re-asked: %d new calls (%v)", len(decision.calls)-before, err)
	}
}

func TestMessageLabelsDoNotInvalidateRunScopeObservations(t *testing.T) {
	runOnly := `[{"id":"run.humans","group":"run","scope":"run","method":"deterministic","population":"human_operator",
   "rule":"human_message_count > 0 ? 'human' : 'brief'","evidence":[{"source":"workload","field":"kind"}],
   "gate":{"precision":0.7,"recall":0.5,"min_gold":2},
   "search":{"facet":"label:run.humans","min_probability":0.6},"compatibility":{"change":"breaking"}}]`
	with, err := ParsePack([]byte(packWithLabels(messageLabels)))
	if err != nil {
		t.Fatal(err)
	}
	without, err := ParsePack([]byte(packWithLabels(runOnly)))
	if err != nil {
		t.Fatal(err)
	}
	withCompiled, err := with.Compile()
	if err != nil {
		t.Fatal(err)
	}
	withoutCompiled, err := without.Compile()
	if err != nil {
		t.Fatal(err)
	}
	if withCompiled.LabelHashes["run.humans"] != withoutCompiled.LabelHashes["run.humans"] {
		t.Fatal("adding a message-scope label changed a run-scope label's definition hash")
	}
}

func TestPopulationReportsCoverageAgainstHumanText(t *testing.T) {
	facts := []RunFacts{
		conversation("chat", "interactive", "fix it", "thanks"),
		conversation("flow1", "workflow_node", "brief one"),
		conversation("flow2", "workflow_node", "brief two"),
		runFact("silent"),
	}
	facts[3].OperatorMessages = nil
	service, _ := testService(t, messageLabels, facts, &fakeDecision{}, Options{})
	ctx := context.Background()
	if _, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"chat", "flow1", "flow2", "silent"}, TimeBudget: -1}); err != nil {
		t.Fatal(err)
	}
	coverage, err := service.Coverage(ctx, "7d")
	if err != nil {
		t.Fatal(err)
	}
	prov := coverage.Provenance
	if prov == nil || prov.RunsAnnotated != 4 || prov.RunsHumanText != 1 || prov.RunsBriefOnly != 2 || prov.RunsNoOperatorText != 1 || prov.HumanMessages != 2 || prov.BriefMessages != 2 {
		t.Fatalf("provenance = %+v", prov)
	}
	for _, label := range coverage.Labels {
		switch label.ID {
		case "operator.mood":
			if label.Population == nil || label.Population.Unit != "message" || label.Population.Eligible != 2 || label.Eligible != 2 || label.Observed != 2 {
				t.Fatalf("mood label = %+v, want coverage against the 2 human messages", label)
			}
		case "run.humans":
			if label.Population == nil || label.Population.Unit != "run" || label.Population.Eligible != 1 {
				t.Fatalf("run.humans population = %+v, want the one run a person wrote", label.Population)
			}
		}
	}
}

func TestSeriesBucketsMessagesByWhenTheyWereWritten(t *testing.T) {
	first := conversation("day1", "interactive", "broken again", "still broken")
	second := conversation("day2", "interactive", "works now")
	for index := range second.OperatorHistory {
		second.OperatorHistory[index].At = second.OperatorHistory[index].At.Add(24 * time.Hour)
	}
	second.OperatorMessages = second.OperatorHistory
	service, _ := testService(t, messageLabels, []RunFacts{first, second}, &fakeDecision{}, Options{
		Clock: func() time.Time { return messageBase.Add(72 * time.Hour) },
	})
	ctx := context.Background()
	if _, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"day1", "day2"}, TimeBudget: -1}); err != nil {
		t.Fatal(err)
	}
	series, err := service.Series(ctx, "operator.mood", "7d", "day")
	if err != nil {
		t.Fatal(err)
	}
	if series.Messages != 3 || len(series.Points) != 2 || series.Points[0].Messages != 2 || series.Points[1].Messages != 1 {
		t.Fatalf("series = %+v, want two messages on day one and one on day two", series)
	}
	if series.Points[0].Mean == nil || series.Points[0].ProvenMean != nil || series.ProvenMessages != 0 || !strings.Contains(series.Note, "no clearing hold-out proof") {
		t.Fatalf("series = %+v, want a mean that is labelled evidence, not proof", series)
	}
	if _, err := service.Series(ctx, "run.humans", "7d", "day"); err == nil {
		t.Fatal("a run-scope label produced a message series")
	}
	empty, err := service.Series(ctx, "operator.mood", "1h", "day")
	if err != nil || empty.Messages != 0 || !strings.Contains(empty.Note, "unknown, not neutral") {
		t.Fatalf("empty window = %+v (%v)", empty, err)
	}
}

func TestMessageDigestIsBoundedValidJSON(t *testing.T) {
	facts := conversation("chat", "interactive", "first", strings.Repeat("angry words ", 400))
	message := facts.OperatorHistory[1]
	for _, limit := range []int{1800, 900, 500} {
		body, err := BuildMessageDigest(facts, message, 1, 1, limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) > limit {
			t.Fatalf("limit %d produced %d chars", limit, len(body))
		}
		if !strings.Contains(body, "angry words") {
			t.Fatalf("limit %d lost the message: %s", limit, body)
		}
	}
	if body, _ := BuildMessageDigest(facts, message, 1, 1, 1800); !strings.Contains(body, "I changed the file") {
		t.Fatalf("the agent text before the message was dropped: %s", body)
	}
}

// authorshipPolicy is an inline authorship policy for engine tests, so they
// never depend on the live pack's lists (default_pack_test.go pins those).
func authorshipPolicy(t *testing.T) Provenance {
	t.Helper()
	body := `{"pack":{"name":"t","version":"1","runtime":{"mode":"passive","provenance":{
		"brief_workloads":["workflow_node","scheduled"],"seed_brief_workloads":["adhoc"],"delegated_by_lineage":true,
		"delegating_openings":["(?i)^\\s*agent manager lineage:"],
		"agent_markers":["^\\s*# E\\d+\\b","\\bRead E\\d+ D\\d+\\b","(?i)^\\s*(supervisor directive|coordination\\b)"]}},
		"labels":[` + labelJSON("a") + `]}}`
	pack, err := ParsePack([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	pack.normalize()
	return pack.Runtime.Provenance
}

func TestAuthorshipFollowsLineageNotTheWorkloadKind(t *testing.T) {
	policy := authorshipPolicy(t)
	spawned := func(facts RunFacts) RunFacts {
		facts.Lineage = LineageFacts{ParentRunID: "parent-run", Depth: 1}
		return facts
	}
	opened := func(facts RunFacts, opening string) RunFacts {
		facts.Lineage.Opening = opening
		return facts
	}
	imported := func(facts RunFacts) RunFacts {
		facts.Lineage = LineageFacts{Imported: true, ParentRunID: "parent-run", Depth: 1}
		return facts
	}
	cases := []struct {
		name         string
		facts        RunFacts
		human, brief int
	}{
		{"a person typing into a root interactive run is a person", conversation("p", "interactive", "hi", "fix it", "great"), 3, 0},
		{"an interactive run another run spawned is delegated throughout", spawned(conversation("s", "interactive", "do E3", "F4 look at this", "thanks")), 0, 3},
		{"an ad hoc run another run spawned is delegated throughout", spawned(conversation("a", "adhoc", "brief", "steer", "steer")), 0, 3},
		{"an imported session is a person even when linked under a parent", imported(conversation("i", "", "hello", "no, that is wrong")), 2, 0},
		{"a run opened by a lineage prompt is delegated with no recorded parent",
			opened(conversation("o", "interactive", "Agent Manager lineage: you are run o.", "Operator F15: make mockups"), "Agent Manager lineage: you are run o."), 0, 2},
		{"an epoch assignment is an agent's text wherever it appears", conversation("e", "interactive", "# E37 Settings and Reach", "this is wrong"), 1, 1},
		{"a person who says supervisor mid-sentence is still a person", conversation("m", "interactive", "tell the supervisor to stop"), 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := tc.facts
			policy.Apply(&facts)
			if len(facts.HumanMessages) != tc.human || facts.BriefMessages != tc.brief {
				t.Fatalf("human %d brief %d, want %d/%d", len(facts.HumanMessages), facts.BriefMessages, tc.human, tc.brief)
			}
		})
	}
	// The lineage rule is pack data: a pack may decline it.
	off := policy
	off.DelegatedByLineage = false
	facts := spawned(conversation("s", "interactive", "do E3", "thanks"))
	off.Apply(&facts)
	if len(facts.HumanMessages) != 2 {
		t.Fatalf("with delegated_by_lineage=false a spawned run's text is read as typed: human %d", len(facts.HumanMessages))
	}
}

func TestLineageIsACELFactAndTheSourceReadsIt(t *testing.T) {
	facts := conversation("s", "interactive", "hi")
	facts.Lineage = LineageFacts{ParentRunID: "p", Depth: 1}
	features := FeaturesFromFacts(facts)
	if !features.SpawnedByRun || features.ImportedSession {
		t.Fatalf("features = %+v", features)
	}
	env, err := featureEnv()
	if err != nil {
		t.Fatal(err)
	}
	program, err := compileProgram(env, "spawned_by_run && !imported_session", 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := celEvaluator{}.evalBool(program, features.Activation())
	if err != nil || !got {
		t.Fatalf("spawned_by_run in CEL = %v (%v)", got, err)
	}
	imported := FeaturesFromFacts(RunFacts{Lineage: LineageFacts{Imported: true, ParentRunID: "p"}})
	if imported.SpawnedByRun || !imported.ImportedSession {
		t.Fatalf("an imported child is not spawned: %+v", imported)
	}
}

func TestInvalidProvenancePatternsAreRefused(t *testing.T) {
	body := `{"pack":{"name":"t","version":"1","runtime":{"mode":"passive","provenance":{"agent_markers":["("]}},"labels":[` + labelJSON("a") + `]}}`
	if _, err := ParsePack([]byte(body)); err == nil || !strings.Contains(err.Error(), "agent_markers") {
		t.Fatalf("err = %v, want the bad pattern named", err)
	}
}

func TestToolFactsAreCELFactsOverTheWholeRun(t *testing.T) {
	facts := conversation("s", "interactive", "hi")
	facts.ToolSummary = toolfacts.Facts{Source: "transcript", ToolCalls: 120, UnchangedRepeats: 30, LongestUnchangedRun: 19}
	env, err := featureEnv()
	if err != nil {
		t.Fatal(err)
	}
	for expression, want := range map[string]bool{
		"tool_source == 'transcript' && longest_unchanged_run >= 19 && unchanged_repeats >= 30": true,
		"tool_source != 'none' && tool_calls == 120":                                            true,
		"longest_unchanged_run >= 20":                                                           false,
	} {
		program, err := compileProgram(env, expression, 0)
		if err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
		got, err := celEvaluator{}.evalBool(program, FeaturesFromFacts(facts).Activation())
		if err != nil || got != want {
			t.Fatalf("%s = %v (%v), want %v", expression, got, err, want)
		}
	}
	// The window facts are gone: a rule naming one fails to compile instead of
	// silently reading zero.
	for _, removed := range []string{"tool_window_calls > 0", "consecutive_identical_commands > 1"} {
		if _, err := compileProgram(env, removed, 0); err == nil {
			t.Fatalf("%q compiled, but the windowed fact no longer exists", removed)
		}
	}
}

// A recorded sender is authoritative: it outranks the lineage and marker rules
// in both directions, and the lineage rule still decides messages with none.
func TestARecordedSenderOutranksLineage(t *testing.T) {
	policy := authorshipPolicy(t)
	message := func(id, text, sender string) MessageFact { return MessageFact{EventID: id, Text: text, Sender: sender} }
	cases := []struct {
		name      string
		facts     RunFacts
		wantHuman []string
	}{
		{"a person typing into an agent-spawned run is a person",
			RunFacts{Lineage: LineageFacts{ParentRunID: "parent", Depth: 1}, OperatorHistory: []MessageFact{
				message("a", "do E3", ""), message("b", "no, that is wrong", domain.MessageSenderPerson)}},
			[]string{"b"}},
		{"a person who pastes an epoch heading is still a person",
			RunFacts{OperatorHistory: []MessageFact{message("a", "# E3 as you wrote it, this is wrong", domain.MessageSenderPerson)}},
			[]string{"a"}},
		{"an agent steering a root interactive run is not a person",
			RunFacts{OperatorHistory: []MessageFact{message("a", "please also check the tests", domain.MessageSenderAgent), message("b", "thanks", "")}},
			[]string{"b"}},
		{"a person in a workflow-node run is a person, an unsent message is a brief",
			RunFacts{Workload: WorkloadFacts{Kind: "workflow_node"}, OperatorHistory: []MessageFact{
				message("a", "implement it", ""), message("b", "stop, wrong direction", domain.MessageSenderPerson)}},
			[]string{"b"}},
		{"messages with no sender fall back to the lineage rule",
			RunFacts{Lineage: LineageFacts{ParentRunID: "parent", Depth: 1}, OperatorHistory: []MessageFact{message("a", "hello", "")}},
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := tc.facts
			policy.Apply(&facts)
			var got []string
			for _, human := range facts.HumanMessages {
				got = append(got, human.EventID)
			}
			if strings.Join(got, ",") != strings.Join(tc.wantHuman, ",") {
				t.Fatalf("human messages = %v, want %v", got, tc.wantHuman)
			}
		})
	}
}
