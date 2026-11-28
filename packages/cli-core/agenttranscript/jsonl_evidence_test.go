package agenttranscript

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestExtractTranscriptEvidencePreservesGenericImportRules(t *testing.T) {
	injected := "<user_instructions> " + strings.Repeat("shared setup ", 6)
	input := strings.Join([]string{
		`malformed json is ignored`,
		`{"type":"metadata","nested":{"type":"ai-title","aiTitle":"  Harness task  "}}`,
		`{"events":[{"role":"user","content":"` + injected + `"}]}`,
		`{"type":"unknown-dialect","nested":{"type":"user_message","prompt":"  fix\tthis  bug "}}`,
		`{"type":"agent_message","message":"` + strings.Repeat("é", 700) + `"}`,
		`{"role":"assistant","content":"` + strings.Repeat("é", 700) + `"}`,
		`{"type":"unknown-dialect","nested":{"type":"agent_message","content":"second reply"}}`,
	}, "\n")
	options := TranscriptEvidenceOptions{
		ExtractLabels:        true,
		MaxLineBytes:         1 << 20,
		TitleSourceBytes:     6 * 1024,
		TitleItemBytes:       600,
		UserPromptBytes:      140,
		IsInjectedContext:    func(text string) bool { return strings.HasPrefix(text, "<user_instructions>") },
		GoalPreambleMinBytes: 40,
	}
	evidence, err := ExtractTranscriptEvidence(strings.NewReader(input), options)
	if err != nil {
		t.Fatal(err)
	}
	wantUser := "fix this bug"
	if evidence.HarnessTitle != "Harness task" || evidence.UserPrompt != wantUser {
		t.Fatalf("title/user prompt = %q/%q", evidence.HarnessTitle, evidence.UserPrompt)
	}
	expectedAgent := "Agent: " + strings.Repeat("é", 300)
	if evidence.TitleSource != "User: fix this bug\n"+expectedAgent+"\nAgent: second reply\n" {
		t.Fatalf("title source = %q", evidence.TitleSource)
	}
	if strings.Count(evidence.TitleSource, "Agent: "+strings.Repeat("é", 300)) != 1 {
		t.Fatalf("duplicate or untruncated agent entry: %q", evidence.TitleSource)
	}
	if !utf8.ValidString(evidence.TitleSource) {
		t.Fatal("title source is not valid UTF-8")
	}
	if !strings.Contains(evidence.GoalPreamble, "shared setup") {
		t.Fatalf("goal preamble did not preserve first user record: %q", evidence.GoalPreamble)
	}
}

func TestExtractTranscriptEvidenceUsesFirstGoalMarkerAfterPreamble(t *testing.T) {
	lines := []string{
		`{"role":"user","content":"This is a sufficiently long first objective from a transcript."}`,
		`malformed marker one`,
		`marker two`,
	}
	seen := []string{}
	evidence, err := ExtractTranscriptEvidence(strings.NewReader(strings.Join(lines, "\n")), TranscriptEvidenceOptions{
		MaxLineBytes:         1024,
		GoalMaxLineBytes:     1024,
		GoalPreambleMinBytes: 40,
		ParseGoalStatusLine: func(line string) (string, string, bool) {
			seen = append(seen, line)
			if line == "malformed marker one" {
				return "objective-one", "met", true
			}
			if line == "marker two" {
				return "objective-two", "unmet", true
			}
			return "", "", false
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.GoalPreamble == "" || !evidence.GoalMarkerFound || evidence.GoalMarkerObjective != "objective-one" || evidence.GoalMarkerStatus != "met" {
		t.Fatalf("evidence = %+v", evidence)
	}
	if len(seen) != 2 {
		t.Fatalf("goal marker callback calls = %d, want stop after first marker", len(seen))
	}
}

func TestExtractTranscriptEvidenceGoalLineLimitDoesNotStopLabelScan(t *testing.T) {
	oversizedForGoal := `{"role":"user","content":"` + strings.Repeat("x", 80) + `"}`
	lateMarkerCalls := 0
	input := oversizedForGoal + "\nmarker after goal limit"
	evidence, err := ExtractTranscriptEvidence(strings.NewReader(input), TranscriptEvidenceOptions{
		ExtractLabels:        true,
		MaxLineBytes:         256,
		GoalMaxLineBytes:     32,
		TitleSourceBytes:     256,
		TitleItemBytes:       100,
		UserPromptBytes:      140,
		GoalPreambleMinBytes: 40,
		ParseGoalStatusLine: func(line string) (string, string, bool) {
			lateMarkerCalls++
			return "", "", false
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.UserPrompt == "" || !strings.HasPrefix(evidence.TitleSource, "User: ") {
		t.Fatalf("label evidence lost after goal line limit: %+v", evidence)
	}
	if evidence.GoalPreamble != "" || lateMarkerCalls != 0 {
		t.Fatalf("goal scan should stop at its smaller line limit: %+v marker calls=%d", evidence, lateMarkerCalls)
	}
}

func TestExtractTranscriptEvidenceKeepsLabelAndGoalNormalizersDistinct(t *testing.T) {
	input := `{"role":"user","content":{"message":"A sufficiently long nested task name"}}`
	evidence, err := ExtractTranscriptEvidence(strings.NewReader(input), TranscriptEvidenceOptions{
		ExtractLabels:        true,
		MaxLineBytes:         1024,
		TitleSourceBytes:     1024,
		TitleItemBytes:       600,
		UserPromptBytes:      140,
		GoalPreambleMinBytes: 40,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.UserPrompt != "A sufficiently long nested task name" || !strings.HasPrefix(evidence.TitleSource, "User: ") {
		t.Fatalf("label extractor lost nested message fallback: %+v", evidence)
	}
	if evidence.GoalPreamble != "" {
		t.Fatalf("goal normalizer accepted nested message fallback: %q", evidence.GoalPreamble)
	}
}

func TestExtractTranscriptEvidenceReturnsPartialEvidenceWithScannerError(t *testing.T) {
	input := `{"role":"user","content":"` + strings.Repeat("g", 45) + `"}` + "\n" + strings.Repeat("x", 100)
	evidence, err := ExtractTranscriptEvidence(strings.NewReader(input), TranscriptEvidenceOptions{
		ExtractLabels:        true,
		MaxLineBytes:         80,
		GoalPreambleMinBytes: 40,
	})
	if err == nil {
		t.Fatal("expected scanner size error")
	}
	if evidence.GoalPreamble == "" {
		t.Fatalf("partial goal evidence was discarded: %+v", evidence)
	}
}
