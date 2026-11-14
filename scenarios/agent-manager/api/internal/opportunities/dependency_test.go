package opportunities

import (
	"fmt"
	"strings"
	"testing"
)

// lookupFlow is a three-step flow whose last step takes a per-run skill name:
// status → search → skill read <skill>. printed is what the search step
// printed in that run.
func lookupFlow(id, skill string) *session {
	s := newSession(id).
		vrooli("vrooli", "vrooli scenario status", "vrooli scenario status x").
		vrooli("search-hub", "search-hub query", "search-hub query intent").
		vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read "+skill)
	for i := range s.calls {
		s.calls[i].OutputEventID = fmt.Sprintf("%s-out-%d", id, i)
	}
	s.calls[1].Features.Command = "search-hub query intent" // constant across runs
	return s
}

// outputsFor serves the text each run's search step printed and records every
// read, so a test can bound how much output a report touches.
func outputsFor(printed map[string]string) (Outputs, *[]string) {
	var asked []string
	return func(ids []string) (map[string]string, error) {
		asked = append(asked, ids...)
		out := map[string]string{}
		for _, id := range ids {
			if text, ok := printed[id]; ok {
				out[id] = text
			}
		}
		return out, nil
	}, &asked
}

func lookupRuns(n int, echo bool) ([]*session, map[string]string) {
	return lookupRunsWhere(n, func(int) string {
		if echo {
			return "hit"
		}
		return ""
	})
}

// lookupRunsWhere prints, per run, how the search output names the skill read
// next: "hit" as a result, "hint" in a backticked suggestion, "body" inside a
// document body, anything else not at all.
func lookupRunsWhere(n int, how func(i int) string) ([]*session, map[string]string) {
	printed := map[string]string{}
	runs := repeat(n, func(i int) *session {
		id, skill := fmt.Sprintf("l%d", i), fmt.Sprintf("skill-%d", i)
		printed[id+"-out-0"] = "scenario running"
		printed[id+"-out-1"] = "top hits: other-skill, scenario-work-ladder"
		switch how(i) {
		case "hit":
			printed[id+"-out-1"] = "top hits: " + skill + ", other-skill"
		case "hint":
			printed[id+"-out-1"] = "top hits: other-skill. Optional reading: `prompt-manager skill read " + skill + "`"
		case "body":
			printed[id+"-out-1"] = "top hits: other-skill <![CDATA[ see " + skill + " ]]>"
		}
		return lookupFlow(id, skill)
	})
	return runs, printed
}

const lookupSubject = "vrooli scenario status → search-hub query → prompt-manager skill read"

// A journal note closes a slice of work: the steps of the next task are not
// part of the previous one's motif, which may end at the note.
func TestJournalNoteEndsASliceOfWork(t *testing.T) {
	calls := flatten(repeat(4, func(i int) *session {
		return newSession(fmt.Sprintf("b%d", i)).
			vrooli("vrooli", "vrooli scenario status", "vrooli scenario status x").
			vrooli("vrooli", "vrooli scenario logs", "vrooli scenario logs x").
			vrooli("vrooli-memory", "vrooli-memory journal note", "vrooli-memory journal note").
			vrooli("search-hub", "search-hub query", "search-hub query intent").
			vrooli("prompt-manager", "prompt-manager skill read", "prompt-manager skill read idea-workshop").
			vrooli("vrooli", "vrooli scenario restart", "vrooli scenario restart x")
	})...)
	r := Analyze(Input{Calls: calls, Library: Library{}})
	seen := 0
	for _, o := range r.Opportunities {
		if o.Kind != KindSequence {
			continue
		}
		steps := strings.Split(o.Subject, " → ")
		for i, path := range steps {
			if path == sliceBoundary && i != len(steps)-1 {
				t.Errorf("a motif runs on past the journal note: %q", o.Subject)
			}
		}
		seen++
	}
	if seen == 0 {
		t.Errorf("the slices on either side of the note are not motifs: %v", subjects(r))
	}
}

// Steps agents run together but choose independently are not a program: the
// later step's argument never appears in what the earlier step printed.
func TestIndependentStepsAreNotAProgramCandidate(t *testing.T) {
	runs, printed := lookupRuns(5, false)
	outputs, _ := outputsFor(printed)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, lookupSubject)
	if o.Verdict != VerdictIndependent || o.CallsSaved != 0 || o.TokensSaved != 0 || o.MeasuredSteps != 5 || o.DependentSteps != 0 {
		t.Fatalf("independent steps = %+v", o)
	}
	if !strings.Contains(o.Action, "choose each step independently (0 of 5 sampled later steps used an earlier step's result, 0%") || strings.Contains(o.Action, "write a program") {
		t.Fatalf("action = %q", o.Action)
	}
}

// A later step that takes its argument from the earlier output is one a
// program can run: it stays a candidate and states the rate.
func TestStepConsumingEarlierOutputStaysACandidate(t *testing.T) {
	runs, printed := lookupRuns(5, true)
	outputs, _ := outputsFor(printed)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, lookupSubject)
	if o.Verdict != VerdictCandidate || o.CallsSaved != 5*2 || o.DependentSteps != 5 || o.MeasuredSteps != 5 || !strings.Contains(o.Action, "100% of 5 later steps used an earlier step's result, at least") {
		t.Fatalf("dependent steps = %+v", o)
	}
}

// With no output text the dependency cannot be measured: say so and claim no
// savings.
func TestUnreadableOutputsClaimNoProgramSavings(t *testing.T) {
	runs, _ := lookupRuns(5, true)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}}), KindSequence, lookupSubject)
	if o.Verdict != VerdictUnmeasured || o.CallsSaved != 0 || !strings.Contains(o.Action, "no program savings are claimed") {
		t.Fatalf("unmeasured = %+v", o)
	}
}

// A motif whose steps never change between occurrences needs no output: the
// program embeds them.
func TestFixedStepsNeedNoOutputs(t *testing.T) {
	calls := flatten(repeat(4, func(i int) *session { return testLoop(fmt.Sprintf("t%d", i)) })...)
	o := find(t, Analyze(Input{Calls: calls, Library: Library{}}), KindSequence, "vrooli scenario test → test-genie runs wait → test-genie runs show → vrooli-memory journal note")
	if o.Verdict != VerdictCandidate || o.MeasuredSteps != 0 {
		t.Fatalf("fixed steps = %+v", o)
	}
}

// Only a capped sample of occurrences has its output read.
func TestDependencySampleIsCapped(t *testing.T) {
	runs, printed := lookupRuns(MaxDependencySamples*3, true)
	outputs, asked := outputsFor(printed)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, lookupSubject)
	if o.MeasuredSteps != MaxDependencySamples {
		t.Fatalf("measured %d later steps, want the %d sampled", o.MeasuredSteps, MaxDependencySamples)
	}
	if len(*asked) > MaxDependencySamples*2 {
		t.Fatalf("read %d output events, want at most %d (the earlier steps of the sample)", len(*asked), MaxDependencySamples*2)
	}
}

// A rate whose 95% interval spans the threshold decides nothing: the verdict
// says so, with the rate and the sample, and claims no savings.
func TestRateNearThresholdIsUndetermined(t *testing.T) {
	runs, printed := lookupRunsWhere(6, func(i int) string {
		if i%2 == 0 {
			return "hit"
		}
		return ""
	})
	outputs, _ := outputsFor(printed)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, lookupSubject)
	if o.Verdict != VerdictUndetermined || o.CallsSaved != 0 || o.TokensSaved != 0 || !strings.Contains(o.Action, "3 of 6 sampled later steps used an earlier step's result, 50%") || !strings.Contains(o.Action, "no program savings are claimed") {
		t.Fatalf("near-threshold = %+v", o)
	}
}

// A skill named only in a suggestion or inside a document body was not picked
// from the results: it is not a dependency.
func TestSuggestionsAndBodiesAreNotResults(t *testing.T) {
	for _, how := range []string{"hint", "body"} {
		runs, printed := lookupRunsWhere(8, func(int) string { return how })
		outputs, _ := outputsFor(printed)
		o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, lookupSubject)
		if o.Verdict != VerdictIndependent || o.DependentSteps != 0 {
			t.Errorf("%s: %+v", how, o)
		}
	}
}

// An argument the agent already typed in an earlier step is echoed by that
// step's output, not derived from it.
func TestEchoedArgumentIsNotADependency(t *testing.T) {
	runs := repeat(5, func(i int) *session {
		id := fmt.Sprintf("e%d", i)
		s := newSession(id).
			vrooli("vrooli", "vrooli scenario restart", fmt.Sprintf("vrooli scenario restart app-%d", i)).
			vrooli("vrooli", "vrooli scenario status", fmt.Sprintf("vrooli scenario status app-%d", i)).
			vrooli("vrooli", "vrooli scenario logs", "vrooli scenario logs fixed")
		for j := range s.calls {
			s.calls[j].OutputEventID = fmt.Sprintf("%s-out-%d", id, j)
		}
		return s
	})
	printed := map[string]string{}
	for i := range runs {
		printed[fmt.Sprintf("e%d-out-0", i)] = fmt.Sprintf("restarted app-%d", i)
	}
	outputs, asked := outputsFor(printed)
	o := find(t, Analyze(Input{Calls: flatten(runs...), Library: Library{}, Outputs: outputs}), KindSequence, "vrooli scenario restart → vrooli scenario status → vrooli scenario logs")
	if o.Verdict != VerdictCandidate || o.MeasuredSteps != 0 || len(*asked) != 0 {
		t.Fatalf("echo = %+v, read %v", o, *asked)
	}
}

// Suggested spans and document bodies are not results; everything else is.
// A stray backtick hides nothing after it, a span never crosses a line, and
// a CDATA body ends at its own close, not at a "]]>" printed before it.
func TestHitTextStripsOnlyMatchedSpansAndDocumentBodies(t *testing.T) {
	for name, tc := range map[string]struct{ in, keep, drop string }{
		"matched span":           {"hit alpha-skill; read `beta-skill` next", "alpha-skill", "beta-skill"},
		"stray backtick":         {"it's `odd\nhit alpha-skill here", "alpha-skill", ""},
		"stray after a span":     {"read `beta-skill` then ` alpha-skill", "alpha-skill", "beta-skill"},
		"span per line":          {"open ` alpha-skill\nclose ` gamma-skill", "gamma-skill", ""},
		"close before the start": {"]]> hit alpha-skill <![CDATA[ beta-skill ]]> gamma-skill", "alpha-skill", "beta-skill"},
		"body after a close":     {"]]> x <![CDATA[ beta-skill ]]> gamma-skill", "gamma-skill", "beta-skill"},
		"unterminated body":      {"hit alpha-skill <![CDATA[ beta-skill", "alpha-skill", "beta-skill"},
		"two bodies":             {"<![CDATA[ beta-skill ]]> alpha-skill <![CDATA[ delta-skill ]]>", "alpha-skill", "delta-skill"},
	} {
		got := hitText(tc.in)
		if !mentions(got, tc.keep) || (tc.drop != "" && mentions(got, tc.drop)) {
			t.Errorf("%s: hitText(%q) = %q, want %q kept and %q dropped", name, tc.in, got, tc.keep, tc.drop)
		}
	}
}
