package export

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/testutil/videofixture"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestResolveExecutionRecordingUsesTypedArtifactIdentity(t *testing.T) {
	id := uuid.New()
	root := t.TempDir()
	dir := filepath.Join(root, id.String())
	video := filepath.Join(dir, "artifacts", "videos", "source.webm")
	if err := os.MkdirAll(filepath.Dir(video), 0700); err != nil {
		t.Fatal(err)
	}
	bytes := []byte("actual recorded media")
	if err := os.WriteFile(video, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	artifactID := uuid.NewString()
	timeline := &bastimeline.ExecutionTimeline{ExecutionId: id.String(), Entries: []*bastimeline.TimelineEntry{{Aggregates: &bastimeline.TimelineEntryAggregates{Artifacts: []*bastimeline.TimelineArtifact{{Id: artifactID, Type: basbase.ArtifactType_ARTIFACT_TYPE_CUSTOM, ContentType: "video/webm", StorageUrl: "/api/v1/recordings/assets/" + id.String() + "/artifacts/videos/source.webm", Payload: contracts.ToJsonValueMap(map[string]any{"sha256": fmt.Sprintf("%x", sha256.Sum256(bytes))})}}}}}}
	raw, err := protojson.Marshal(timeline)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "timeline.proto.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	repo := timelineLoaderTestRepository{execution: &database.ExecutionIndex{ID: id, ResultPath: filepath.Join(dir, "result.json")}}
	resolved, err := ResolveExecutionRecording(t.Context(), repo, root, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ArtifactID != artifactID || resolved.Path != video {
		t.Fatalf("typed identity lost %+v", resolved)
	}
}

func TestTypedRecordingDeduplicatesAliasesAndRefusesDistinctContents(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.mp4")
	alias := filepath.Join(directory, "alias.mp4")
	second := filepath.Join(directory, "second.webm")
	for path, data := range map[string][]byte{first: videofixture.MP4(), alias: videofixture.MP4(), second: videofixture.WebM()} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	artifact := func(id, path string) executionwriter.ArtifactData {
		return executionwriter.ArtifactData{ArtifactID: id, ArtifactType: "video", Payload: map[string]any{"path": path}}
	}
	repeated := []executionwriter.ArtifactData{artifact("z-video", first), artifact("a-meta", first), artifact("b-copy", alias)}
	source, err := uniqueTypedRecording(t.Context(), repeated, nil)
	if err != nil {
		t.Fatal(err)
	}
	if source.ArtifactID != "a-meta" || source.Path != first {
		t.Fatalf("alias identity/order was not deterministic %+v", source)
	}
	if _, err = uniqueTypedRecording(t.Context(), []executionwriter.ArtifactData{artifact("a-first", first), artifact("b-second", second)}, nil); err == nil {
		t.Fatal("silently chose one distinct typed recording")
	}
	if _, err = uniqueTypedRecording(t.Context(), []executionwriter.ArtifactData{artifact("a-second", second), artifact("b-first", first)}, nil); err == nil {
		t.Fatal("timeline order changed ambiguity policy")
	}
}

func TestTypedRecordingMeasuresInlineAliasesBeforeDeduplication(t *testing.T) {
	inline := func(id string, data []byte) executionwriter.ArtifactData {
		return executionwriter.ArtifactData{ArtifactID: id, ArtifactType: "video", ContentType: "video/mp4", Payload: map[string]any{"inline": true, "base64": base64.StdEncoding.EncodeToString(data)}}
	}
	resolved, err := uniqueTypedRecording(t.Context(), []executionwriter.ArtifactData{inline("a", videofixture.MP4()), inline("b", videofixture.MP4())}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resolved.Cleanup()
	if _, err = uniqueTypedRecording(t.Context(), []executionwriter.ArtifactData{inline("a", videofixture.MP4()), inline("b", videofixture.WebM())}, nil); err == nil {
		t.Fatal("silently collapsed distinct inline recordings with empty locators")
	}
}
