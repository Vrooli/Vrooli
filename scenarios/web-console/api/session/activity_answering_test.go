package session

import (
	"testing"

	"web-console/internal/backend"
)

func TestActivityReportsWhetherThePromptCanBeAnswered(t *testing.T) {
	// [REQ:P0-017i] The detector asks the host's answering policy about the
	// prompt it reports, so the published and the current activity both say
	// whether Messages may answer it and which harness version decided that.
	h := newActivityHarness(t)
	var asked []string
	h.d.cfg.Answering = func(harness string, prompt backend.PendingPrompt) (string, bool) {
		asked = append(asked, harness+":"+prompt.Kind)
		return "2.1.268", true
	}
	h.d.OnFrame()
	prompt := &backend.PendingPrompt{Kind: "question", Text: "Which color?", Options: []backend.PromptOption{{Key: "1", Label: "Red", Selected: true}, {Key: "2", Label: "Blue"}}}
	h.detector.analysis = backend.PromptAnalysis{Prompt: prompt, Confidence: 0.85}
	h.clock.Advance(activityDebounce)
	assertStates(t, h, ActivityWorking, ActivityWaiting)
	if got := h.published[1]; !got.Answerable || got.HarnessVersion != "2.1.268" {
		t.Fatalf("published waiting activity = %+v, want answerable on 2.1.268", got)
	}
	if got := h.d.Current(); !got.Answerable || got.HarnessVersion != "2.1.268" {
		t.Fatalf("current activity = %+v, want answerable on 2.1.268", got)
	}
	if len(asked) == 0 || asked[0] != "claude:question" {
		t.Fatalf("policy asked %v, want claude:question", asked)
	}
	if h.published[0].Answerable {
		t.Fatal("the working activity was reported answerable")
	}
}

func TestActivityWithoutAnAnsweringPolicyIsReadOnly(t *testing.T) {
	// [REQ:P0-017i] With no policy (answering off) a waiting prompt is read-only.
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.detector.analysis = backend.PromptAnalysis{Prompt: &backend.PendingPrompt{Kind: "permission", Text: "Proceed?", Options: []backend.PromptOption{{Key: "1", Label: "Yes"}}}, Confidence: 0.85}
	h.clock.Advance(activityDebounce)
	if got := h.d.Current(); got.State != ActivityWaiting || got.Answerable {
		t.Fatalf("current activity = %+v, want waiting and read-only", got)
	}
}
