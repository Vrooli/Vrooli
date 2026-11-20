package runannotations

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildDigestIsValidBoundedJSON(t *testing.T) {
	facts := RunFacts{
		RunID:    "run-1",
		Workload: WorkloadFacts{Kind: "workflow_node", Key: strings.Repeat("k", 500)},
		OperatorMessages: []MessageFact{
			{EventID: "e1", At: time.Unix(1, 0), Text: strings.Repeat("operator ", 1000)},
			{EventID: "e2", At: time.Unix(2, 0), Text: strings.Repeat("again ", 1000)},
			{EventID: "e3", At: time.Unix(3, 0), Text: strings.Repeat("last ", 1000)},
		},
		AgentText: []MessageFact{
			{EventID: "a1", At: time.Unix(1, 0), Text: strings.Repeat("agent ", 1000)},
			{EventID: "a2", At: time.Unix(2, 0), Text: strings.Repeat("agent ", 1000)},
		},
		ToolSummary: ToolSummaryFacts{
			Calls: 50, Failures: 7, FailedCommands: 3, RepeatedPaths: 21,
			FailedTools: []string{"bash", "read", "bash", "read", "bash"},
			LastError:   strings.Repeat("error ", 400),
		},
		DeterministicFlags: DeterministicFacts{
			StepbackTriggers: []string{"objective-lost", "unit-per-turn", "short-workers"},
			ToolFindings:     []string{"blocked-by-permission", "retry-loop"},
			CompactionCount:  3,
		},
		ExistingSignals: map[string]SignalFact{
			"frustration": {Label: "frustration", Probability: floatPointer(0.71), LatestEventID: "s1"},
		},
		Timings:  TimingFacts{DurationMinutes: 42.5},
		Terminal: TerminalFacts{Status: "completed", Class: "goal_complete"},
	}
	digest := BuildDigest(facts)
	if len(digest) > DefaultDigestLimit {
		t.Fatalf("digest = %d chars, want <= %d", len(digest), DefaultDigestLimit)
	}
	var decoded Digest
	if err := json.Unmarshal([]byte(digest), &decoded); err != nil {
		t.Fatalf("digest is not JSON: %v\n%s", err, digest)
	}
	if decoded.RunID != "run-1" || decoded.Tool.FailedCommands != 3 {
		t.Fatalf("decoded = %+v", decoded)
	}
	if len(decoded.OperatorText) == 0 || len(decoded.OperatorText[0]) > 300 {
		t.Fatalf("operator text not bounded: %d", len(decoded.OperatorText))
	}
	if len(digest) > 2000 {
		t.Fatalf("digest is not the promised ~1-2k size: %d", len(digest))
	}
}

func TestBuildDigestShrinksToAConfiguredLimit(t *testing.T) {
	facts := RunFacts{
		RunID:    "run-1",
		Workload: WorkloadFacts{Kind: "adhoc"},
		OperatorMessages: []MessageFact{
			{EventID: "e1", Text: strings.Repeat("x", 5000)},
			{EventID: "e2", Text: strings.Repeat("y", 5000)},
		},
		AgentText: []MessageFact{{EventID: "a1", Text: strings.Repeat("z", 5000)}},
	}
	for _, limit := range []int{1200, 800, 500, 300} {
		digest, err := BuildDigestBounded(facts, limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(digest) > limit {
			t.Fatalf("limit %d: digest %d chars", limit, len(digest))
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(digest), &decoded); err != nil {
			t.Fatalf("limit %d: invalid JSON: %v", limit, err)
		}
	}
}

func TestBoundTextAndListUseRunes(t *testing.T) {
	if got := BoundText("héllo world", 5); got != "héll…" {
		t.Fatalf("BoundText = %q", got)
	}
	if got := BoundList([]string{"a", "b", "c"}, 2, 10); len(got) != 2 {
		t.Fatalf("BoundList = %#v", got)
	}
}

func TestReviewDigestCarriesMoreEvidenceThanTheModelDigest(t *testing.T) {
	facts := longRunFact("run-review")
	facts.ToolSummary.FailedCommands = 3
	facts.ToolSummary.LastError = strings.Repeat("tool error detail ", 60)
	model, err := BuildDigestBounded(facts, DefaultDigestLimit)
	if err != nil {
		t.Fatal(err)
	}
	review, err := BuildReviewDigest(facts, DefaultReviewDigestLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(review) <= len(model) {
		t.Fatalf("review digest %d chars is not wider than the model digest %d", len(review), len(model))
	}
	if len(review) > DefaultReviewDigestLimit {
		t.Fatalf("review digest = %d chars, want <= %d", len(review), DefaultReviewDigestLimit)
	}
	var modelBody, reviewBody Digest
	if err := json.Unmarshal([]byte(model), &modelBody); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(review), &reviewBody); err != nil {
		t.Fatal(err)
	}
	if len(reviewBody.OperatorText) <= len(modelBody.OperatorText) {
		t.Fatalf("review operator text = %d, model = %d", len(reviewBody.OperatorText), len(modelBody.OperatorText))
	}
	if len(reviewBody.Tool.LastError) <= len(modelBody.Tool.LastError) {
		t.Fatalf("review last error = %d chars, model = %d", len(reviewBody.Tool.LastError), len(modelBody.Tool.LastError))
	}
}
