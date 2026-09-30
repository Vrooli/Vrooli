package executionwriter

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildResultManifestPayloadUsesStableFallbackWithoutTimeline(t *testing.T) {
	executionID := uuid.New()
	payload, err := buildResultManifestPayload(executionID, &ExecutionResultData{WorkflowID: "workflow-1"}, nil)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if payload["execution_id"] != executionID.String() || payload["workflow_id"] != "workflow-1" {
		t.Fatalf("unexpected fallback manifest: %#v", payload)
	}
}

func TestResultManifestPreservesTimelineWhileOmittingArtifactPayloads(t *testing.T) {
	id := uuid.New()
	source := &executionTimelineData{pb: &bastimeline.ExecutionTimeline{ExecutionId: id.String(), Entries: []*bastimeline.TimelineEntry{{
		Id: "entry", Aggregates: &bastimeline.TimelineEntryAggregates{Artifacts: []*bastimeline.TimelineArtifact{{Id: "durable", Payload: map[string]*commonv1.JsonValue{
			"value": {Kind: &commonv1.JsonValue_StringValue{StringValue: strings.Repeat("capture", 1024)}},
		}}}},
	}}}}
	before := proto.Clone(source.pb)
	payload, err := buildResultManifestPayload(id, nil, source)
	require.NoError(t, err)
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "capture")
	require.NotContains(t, string(encoded), "artifacts")
	require.Contains(t, string(encoded), "entry")
	require.True(t, proto.Equal(before, source.pb), "manifest projection cannot alter full timeline evidence")
}

func BenchmarkResultManifestProjection(b *testing.B) {
	id := uuid.New()
	timeline := &executionTimelineData{pb: &bastimeline.ExecutionTimeline{ExecutionId: id.String()}}
	for i := 0; i < 10; i++ {
		timeline.pb.Entries = append(timeline.pb.Entries, &bastimeline.TimelineEntry{Id: "entry", Aggregates: &bastimeline.TimelineEntryAggregates{
			Artifacts: []*bastimeline.TimelineArtifact{{Id: "dom", Payload: map[string]*commonv1.JsonValue{
				"html": {Kind: &commonv1.JsonValue_StringValue{StringValue: strings.Repeat("<div>capture</div>", 32768)}},
			}}},
		}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := buildResultManifestPayload(id, nil, timeline); err != nil {
			b.Fatal(err)
		}
	}
}
