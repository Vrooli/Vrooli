package execution

import (
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/evidence"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// networkEventArtifact builds a network-event timeline artifact carrying the
// given url/status/failure payload, mirroring what BAS surfaces on aggregates.
func networkEventArtifact(url string, status *int, failure string) *bastimeline.TimelineArtifact {
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

func TestConsoleEntries_NormalizesLevels(t *testing.T) {
	tl := &ParsedTimeline{
		Logs: []ParsedLog{
			{Level: "ERROR", Message: "boom"},
			{Level: " Warn ", Message: "careful"},
			{Level: "info", Message: "fyi"},
		},
	}
	got := ConsoleEntries(tl)
	if len(got) != 3 {
		t.Fatalf("expected 3 console entries, got %d", len(got))
	}
	if got[0].Level != "error" || got[1].Level != "warn" || got[2].Level != "info" {
		t.Fatalf("levels not normalized: %+v", got)
	}

	// Analyzer must count these correctly.
	v := evidence.Analyze(evidence.Evidence{Loaded: true, Handshake: evidence.Handshake{Signaled: true}, Console: got})
	if v.ConsoleErrorCount != 1 || v.ConsoleWarningCount != 1 {
		t.Fatalf("unexpected counts: errors=%d warnings=%d", v.ConsoleErrorCount, v.ConsoleWarningCount)
	}
}

func TestConsoleEntries_NilTimeline(t *testing.T) {
	if got := ConsoleEntries(nil); got != nil {
		t.Fatalf("expected nil for nil timeline, got %+v", got)
	}
	if got := ConsoleEntries(&ParsedTimeline{}); got != nil {
		t.Fatalf("expected nil for empty logs, got %+v", got)
	}
}

func TestNetworkFailures_DetectsFailuresAndStatuses(t *testing.T) {
	ok := 200
	notFound := 404
	tl := &ParsedTimeline{
		Proto: &bastimeline.ExecutionTimeline{
			Entries: []*bastimeline.TimelineEntry{
				{Aggregates: &bastimeline.TimelineEntryAggregates{Artifacts: []*bastimeline.TimelineArtifact{
					networkEventArtifact("http://ok.test/a", &ok, ""),                         // healthy: ignored
					networkEventArtifact("http://err.test/b", &notFound, ""),                  // 404: failure
					networkEventArtifact("http://gone.test/c", nil, "ERR_CONNECTION_REFUSED"), // transport failure
				}}},
			},
		},
	}
	got := NetworkFailures(tl)
	if len(got) != 2 {
		t.Fatalf("expected 2 network failures, got %d: %+v", len(got), got)
	}
	if got[0].Status == nil || *got[0].Status != 404 {
		t.Fatalf("expected 404 failure, got %+v", got[0])
	}
	if got[1].ErrorText != "ERR_CONNECTION_REFUSED" {
		t.Fatalf("expected transport failure, got %+v", got[1])
	}
}

func TestNetworkFailures_NilTimeline(t *testing.T) {
	if got := NetworkFailures(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
	if got := NetworkFailures(&ParsedTimeline{}); got != nil {
		t.Fatalf("expected nil for proto-less timeline, got %+v", got)
	}
}

// TestToEvidence_CleanHappyFixtureYieldsCleanVerdict proves a clean timeline
// (the golden happy fixture) folds into a passing verdict with no spurious
// findings.
func TestToEvidence_CleanHappyFixtureYieldsCleanVerdict(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "timeline_happy.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	ev := ToEvidence(parsed, ToEvidenceOptions{Label: "happy.json"})
	v := evidence.Analyze(ev)

	if !v.Passed() {
		t.Fatalf("expected passing verdict for clean timeline, got %s: %s", v.Status, v.Message)
	}
	if v.NetworkFailureCount != 0 || v.PageErrorCount != 0 || v.ConsoleErrorCount != 0 || v.ConsoleWarningCount != 0 {
		t.Fatalf("expected zero findings on clean timeline, got %+v", v)
	}
	// The happy fixture carries a screenshot on the navigate frame; it should be
	// the surfaced end-state reference.
	if ev.ScreenshotRef == "" {
		t.Fatalf("expected a final screenshot reference, got empty")
	}
}

func TestToEvidence_NilTimelineIsNotLoaded(t *testing.T) {
	ev := ToEvidence(nil, ToEvidenceOptions{Label: "x"})
	if ev.Loaded {
		t.Fatalf("expected Loaded=false for nil timeline")
	}
	v := evidence.Analyze(ev)
	if v.Passed() {
		t.Fatalf("expected failing verdict for a workflow with no timeline")
	}
}

func TestToEvidence_BlankFinalDOMIsPageError(t *testing.T) {
	tl := &ParsedTimeline{
		Proto:           &bastimeline.ExecutionTimeline{},
		FinalDOM:        "<html><body>   </body></html>",
		FinalDOMPreview: "<body></body>",
		Frames:          []ParsedFrame{{FinalURL: "http://blank.test/"}},
	}
	ev := ToEvidence(tl, ToEvidenceOptions{Label: "blank.json"})
	if len(ev.PageErrors) != 1 {
		t.Fatalf("expected a blank-DOM page error, got %+v", ev.PageErrors)
	}
	if ev.URL != "http://blank.test/" {
		t.Fatalf("expected final-frame URL, got %q", ev.URL)
	}
	v := evidence.Analyze(ev)
	if v.Passed() || v.PageErrorCount != 1 {
		t.Fatalf("expected failing verdict with 1 page error, got %+v", v)
	}
}

func TestToEvidence_RenderedDOMIsNotBlank(t *testing.T) {
	tl := &ParsedTimeline{
		Proto:    &bastimeline.ExecutionTimeline{},
		FinalDOM: "<html><body><h1>Welcome</h1></body></html>",
	}
	ev := ToEvidence(tl, ToEvidenceOptions{})
	if len(ev.PageErrors) != 0 {
		t.Fatalf("expected no page errors for a rendered DOM, got %+v", ev.PageErrors)
	}
}

func TestToEvidence_MissingDOMSnapshotIsNotBlank(t *testing.T) {
	// No DOM snapshot at all must not be treated as blank.
	tl := &ParsedTimeline{Proto: &bastimeline.ExecutionTimeline{}}
	ev := ToEvidence(tl, ToEvidenceOptions{})
	if len(ev.PageErrors) != 0 {
		t.Fatalf("expected no page errors when no DOM was captured, got %+v", ev.PageErrors)
	}
}
