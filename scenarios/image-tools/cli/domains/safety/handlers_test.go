package safety

import (
	"strings"
	"testing"
	"time"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

func TestFeedbackLabelMapsTheCLIWords(t *testing.T) {
	for raw, want := range map[string]safetyv1.SafetyFeedbackLabel{
		"legitimate":     safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		"false-positive": safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		"":               safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		"confirmed":      safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
		"uncertain":      safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNCERTAIN,
	} {
		got, err := feedbackLabel(raw)
		if err != nil || got != want {
			t.Errorf("feedbackLabel(%q) = %v/%v, want %v", raw, got, err, want)
		}
	}
	if _, err := feedbackLabel("maybe"); err == nil {
		t.Fatal("an unknown label was accepted")
	}
}

func TestParseSinceAndLimit(t *testing.T) {
	if ts, err := parseSince(""); ts != nil || err != nil {
		t.Fatalf("empty since = %v/%v", ts, err)
	}
	ts, err := parseSince("2026-11-24T01:02:03Z")
	if err != nil || ts == nil || !ts.AsTime().Equal(time.Date(2026, 11, 24, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("parsed since = %v/%v", ts, err)
	}
	if _, err := parseSince("yesterday"); err == nil {
		t.Fatal("a non-RFC3339 since was accepted")
	}
	if got, err := parseLimit("", 100, 1000); err != nil || got != 100 {
		t.Fatalf("empty limit = %d/%v", got, err)
	}
	if got, err := parseLimit("5", 100, 1000); err != nil || got != 5 {
		t.Fatalf("limit 5 = %d/%v", got, err)
	}
	if got, _ := parseLimit("9999", 100, 1000); got != 1000 {
		t.Fatalf("limit above the cap = %d, want 1000", got)
	}
	if _, err := parseLimit("0", 100, 1000); err == nil {
		t.Fatal("zero limit was accepted")
	}
}

func TestFormatFeedbackCaseNamesTheEvidence(t *testing.T) {
	item := &safetyv1.SafetyFeedbackCase{
		JobId: "job-1", OutputRef: "out/a.png", Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		Action: "refuse", ReasonCode: "sexual_unassessable", SafetyReview: "refused", Score: 0.97,
		ModelId: "flux", ProfileName: "local-balanced", Note: "adult subject",
	}
	line := formatFeedbackCase(item)
	for _, want := range []string{"job-1", "out/a.png", "legitimate", "sexual_unassessable", "flux", "local-balanced", "adult subject"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q is missing %q", line, want)
		}
	}
}
