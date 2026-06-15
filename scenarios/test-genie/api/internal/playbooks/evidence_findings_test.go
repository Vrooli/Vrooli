package playbooks

import (
	"strings"
	"testing"

	"test-genie/internal/playbooks/execution"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func netArtifact(url string, status *int, failure string) *bastimeline.TimelineArtifact {
	payload := map[string]*commonv1.JsonValue{
		"url": {Kind: &commonv1.JsonValue_StringValue{StringValue: url}},
	}
	if status != nil {
		payload["status"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(*status)}}
	}
	if failure != "" {
		payload["failure"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: failure}}
	}
	return &bastimeline.TimelineArtifact{Type: basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT, Payload: payload}
}

func obsKinds(obs []Observation) (errors, warnings int) {
	for _, o := range obs {
		switch o.Type {
		case ObservationError:
			errors++
		case ObservationWarning:
			warnings++
		}
	}
	return
}

// TestEvidenceObservations_CleanTimelineNoFindings proves an all-green timeline
// surfaces no additive observations.
func TestEvidenceObservations_CleanTimelineNoFindings(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Proto: &bastimeline.ExecutionTimeline{},
		Logs:  []execution.ParsedLog{{Level: "info", Message: "ok"}},
	}
	obs := evidenceObservations("wf.json", tl)
	if len(obs) != 0 {
		t.Fatalf("expected no observations for a clean timeline, got %+v", obs)
	}
}

// TestEvidenceObservations_ConsoleErrorsAndWarningsFold proves console
// errors/warnings fold into error/warning observations.
func TestEvidenceObservations_ConsoleErrorsAndWarningsFold(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Proto: &bastimeline.ExecutionTimeline{},
		Logs: []execution.ParsedLog{
			{Level: "error", Message: "boom"},
			{Level: "error", Message: "bang"},
			{Level: "warn", Message: "careful"},
		},
	}
	obs := evidenceObservations("wf.json", tl)
	errs, warns := obsKinds(obs)
	if errs != 1 || warns != 1 {
		t.Fatalf("expected 1 error obs + 1 warning obs, got errors=%d warnings=%d (%+v)", errs, warns, obs)
	}
	// The error observation should report the count (2 console errors).
	found := false
	for _, o := range obs {
		if o.Type == ObservationError && strings.Contains(o.Message, "2 console error") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an error observation reporting 2 console errors, got %+v", obs)
	}
}

// TestEvidenceObservations_NetworkFailuresFold proves failed network requests
// surface as an error observation.
func TestEvidenceObservations_NetworkFailuresFold(t *testing.T) {
	notFound := 404
	tl := &execution.ParsedTimeline{
		Proto: &bastimeline.ExecutionTimeline{
			Entries: []*bastimeline.TimelineEntry{
				{Aggregates: &bastimeline.TimelineEntryAggregates{Artifacts: []*bastimeline.TimelineArtifact{
					netArtifact("http://err.test/x", &notFound, ""),
				}}},
			},
		},
	}
	obs := evidenceObservations("wf.json", tl)
	errs, _ := obsKinds(obs)
	if errs != 1 {
		t.Fatalf("expected 1 network error observation, got %d (%+v)", errs, obs)
	}
	if !strings.Contains(obs[0].Message, "HTTP 404") {
		t.Fatalf("expected network failure detail in observation, got %q", obs[0].Message)
	}
}

// TestEvidenceObservations_NilTimelineNoFindings proves a missing timeline does
// not manufacture noisy observations (the workflow's own error is the signal).
func TestEvidenceObservations_NilTimelineNoFindings(t *testing.T) {
	obs := evidenceObservations("wf.json", nil)
	if len(obs) != 0 {
		t.Fatalf("expected no additive observations for a nil timeline, got %+v", obs)
	}
}
