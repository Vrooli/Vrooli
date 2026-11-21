package runannotations

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/toolfacts"
)

// endOfARun is a long run whose operator briefs are long and whose end (the last
// tool error and the agent's closing words) is what a blocked-by-tool judgment
// reads.
func endOfARun() RunFacts {
	facts := RunFacts{
		RunID:    "11111111-2222-3333-4444-555555555555",
		Workload: WorkloadFacts{Kind: "workflow_node", Key: "build/step-3"},
		Terminal: TerminalFacts{Status: "failed", Class: "tool_blocked"},
		Timings:  TimingFacts{DurationMinutes: 31.5},
		ToolSummary: toolfacts.Facts{
			ToolCalls: 88, ToolFailures: 9, FailedCommands: 4,
			FailedTools: []string{"bash", "write", "bash", "read", "bash"},
			LastError:   "permission denied: cannot write /etc/hosts " + strings.Repeat("because the sandbox is read only ", 20),
		},
		DeterministicFlags: DeterministicFacts{
			StepbackTriggers: []string{"objective-lost", "retry-loop", "short-workers"},
			ToolFindings:     []string{"blocked-by-permission", "retry-loop"},
		},
		ExistingSignals: map[string]SignalFact{"frustration": {Label: "frustration", Probability: floatPointer(0.8)}},
	}
	for i := 0; i < 6; i++ {
		facts.OperatorMessages = append(facts.OperatorMessages, MessageFact{EventID: "op", At: time.Unix(int64(i), 0), Text: strings.Repeat("operator brief detail ", 40)})
		facts.AgentText = append(facts.AgentText, MessageFact{EventID: "ag", At: time.Unix(int64(i), 0), Text: "agent says step " + string(rune('A'+i)) + " " + strings.Repeat("filler ", 60)})
	}
	return facts
}

func decodeDigest(t *testing.T, body string) Digest {
	t.Helper()
	var digest Digest
	if err := json.Unmarshal([]byte(body), &digest); err != nil {
		t.Fatalf("digest is not JSON: %v\n%s", err, body)
	}
	return digest
}

func TestASmallBudgetKeepsTheEndOfARunAndLosesItsBeginning(t *testing.T) {
	// 512 tokens leave about 700 characters for the state.
	body, err := BuildDigestBounded(endOfARun(), 706, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > 706 {
		t.Fatalf("digest = %d chars, want <= 706", len(body))
	}
	digest := decodeDigest(t, body)
	if !strings.HasPrefix(digest.Tool.LastError, "permission denied: cannot write /etc/hosts") || len(digest.Tool.LastError) < 80 {
		t.Fatalf("last_error did not survive: %q", digest.Tool.LastError)
	}
	if len(digest.AgentText) != 1 || !strings.HasPrefix(digest.AgentText[0], "agent says step F") {
		t.Fatalf("the agent's closing words did not survive: %q", digest.AgentText)
	}
	if len(digest.OperatorText) != 0 {
		t.Fatalf("early-run operator text outranked the end of the run: %q", digest.OperatorText)
	}
}

func TestALargeBudgetIsRenderedAsItAlwaysWas(t *testing.T) {
	whole, err := BuildDigestBounded(endOfARun(), 10000, nil)
	if err != nil {
		t.Fatal(err)
	}
	digest := decodeDigest(t, whole)
	if len(digest.OperatorText) != 3 || len(digest.AgentText) != 2 || len(digest.Flags.StepbackTriggers) != 3 || digest.ExistingSignals == nil {
		t.Fatalf("a budget that fits everything lost something: %+v", digest)
	}
	if got := len([]rune(digest.OperatorText[0])); got != 280 {
		t.Fatalf("operator text = %d chars, want the full 280", got)
	}
	if got := len([]rune(digest.Tool.LastError)); got != 200 {
		t.Fatalf("last_error = %d chars, want the full 200", got)
	}
	// The order of priority never matters while everything fits.
	same, err := BuildDigestBounded(endOfARun(), 10000, []string{sectionOperator})
	if err != nil {
		t.Fatal(err)
	}
	if same != whole {
		t.Fatal("a declared priority changed a digest that fits")
	}
}

func TestADeclaredPriorityKeepsWhatTheLabelJudgesAndShrinksTheRest(t *testing.T) {
	facts := endOfARun()
	byOperator, err := BuildDigestBounded(facts, 706, []string{sectionOperator})
	if err != nil {
		t.Fatal(err)
	}
	byDefault, err := BuildDigestBounded(facts, 706, nil)
	if err != nil {
		t.Fatal(err)
	}
	asked, fallback := decodeDigest(t, byOperator), decodeDigest(t, byDefault)
	if len(asked.OperatorText) == 0 {
		t.Fatalf("a label that names operator messages lost them: %s", byOperator)
	}
	if len(fallback.OperatorText) != 0 || len(fallback.Tool.LastError) == 0 {
		t.Fatalf("the default order kept operator text: %s", byDefault)
	}
	if asked.Tool.LastError != "" || len(asked.AgentText) != 0 {
		t.Fatalf("sections the label did not name outlasted the one it did: %s", byOperator)
	}
}

func TestDigestPriorityIsTheOrderALabelListsItsSections(t *testing.T) {
	got := DigestPriority([]EvidenceSelector{
		{Source: "tool_summary", Field: "failures"},
		{Source: "workload", Field: "kind"},
		{Source: "agent_text", Field: "last_text"},
		{Source: "tool_summary", Field: "last_error"},
		{Source: "deterministic_flags", Field: "tool_findings"},
	})
	want := []string{"tool_summary", "agent_text", "deterministic_flags"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("priority = %v, want %v (one entry per section that shrinks, first appearance first)", got, want)
	}
}

func TestEveryDigestSectionHasStepsAndADefaultRank(t *testing.T) {
	for section, steps := range sectionSteps {
		if steps[len(steps)-1] != (sectionStep{}) {
			t.Errorf("%s does not end dropped: %v", section, steps)
		}
		if !contains(defaultDigestPriority, section) {
			t.Errorf("%s has no default rank", section)
		}
		if !EvidenceSources[section] {
			t.Errorf("%s is not an evidence source a label can name", section)
		}
	}
	if len(defaultDigestPriority) != len(sectionSteps) {
		t.Errorf("default priority %v and steps for %d sections disagree", defaultDigestPriority, len(sectionSteps))
	}
}

// blockedLabel is a run-scope decision whose evidence is the end of the run.
func blockedLabel(id, evidence string) string {
	return strings.Replace(scoreLabel(id, ""), `"evidence":[{"source":"tool_summary","field":"calls"}]`, `"evidence":[`+evidence+`]`, 1)
}

const endOfRunEvidence = `{"source":"tool_summary","field":"last_error"},{"source":"agent_text","field":"last_text"}`

func TestASmallModelIsSentTheEndOfTheRunItsLabelsJudge(t *testing.T) {
	facts := endOfARun()
	decision := &fakeDecision{}
	limits := limitsFunc(func(string, string) (ModelLimits, bool) { return ModelLimits{ContextTokens: 512}, true })
	service, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast","digest_chars":1800}`,
		`[`+blockedLabel("blocked", endOfRunEvidence)+`]`, []RunFacts{facts, withRunID(facts, "run-2")}, decision, Options{Limits: limits, MaxDigestChars: 1800})
	ctx := context.Background()
	// The first call to a role is sized by the pack: no model has answered yet.
	if _, err := service.Annotate(ctx, facts.RunID, AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	decision.calls = nil
	if _, err := service.Annotate(ctx, "run-2", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 {
		t.Fatalf("calls = %d", len(decision.calls))
	}
	state := decision.calls[0].GetItems()[0].GetStateJson()
	if len(state) > 512*charsPerToken {
		t.Fatalf("state is %d chars, want it within the model's window", len(state))
	}
	digest := decodeDigest(t, state)
	if len(digest.Tool.LastError) < 80 || len(digest.AgentText) == 0 {
		t.Fatalf("the model was not sent the end of the run: %s", state)
	}
	if len(digest.OperatorText) != 0 {
		t.Fatalf("operator text the label does not judge outranked it: %s", state)
	}
}

func withRunID(facts RunFacts, id string) RunFacts {
	facts.RunID = id
	return facts
}

func TestAChangeToHowARunIsRenderedChangesTheDefinitionHash(t *testing.T) {
	hashOf := func(t *testing.T, label string) string {
		t.Helper()
		pack, err := ParsePack([]byte(`{"pack":{"name":"p","version":"1","runtime":{},"labels":[` + label + `]}}`))
		if err != nil {
			t.Fatal(err)
		}
		hash, err := LabelDefinitionHash(pack.Labels[0])
		if err != nil {
			t.Fatal(err)
		}
		return hash
	}
	forward := blockedLabel("x", `{"source":"tool_summary","field":"last_error"},{"source":"agent_text","field":"last_text"}`)
	reversed := blockedLabel("x", `{"source":"agent_text","field":"last_text"},{"source":"tool_summary","field":"last_error"}`)
	if hashOf(t, forward) == hashOf(t, reversed) {
		t.Fatal("ranking the sections differently asks the model something different and must change the hash")
	}
	// Only a label read from the run's digest is affected.
	message := func(evidence string) string {
		return strings.Replace(blockedLabel("m", evidence), `"scope":"run"`, `"scope":"message"`, 1)
	}
	if hashOf(t, message(`{"source":"operator_messages","field":"text"},{"source":"agent_text","field":"text"}`)) !=
		hashOf(t, message(`{"source":"agent_text","field":"text"},{"source":"operator_messages","field":"text"}`)) {
		t.Fatal("a message label does not read the run digest; its order must not change its hash")
	}
}
