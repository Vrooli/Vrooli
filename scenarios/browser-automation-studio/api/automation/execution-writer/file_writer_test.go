package executionwriter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/storage"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

type noopRepo struct{}

func (noopRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, nil
}

func (noopRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return nil
}

func (noopRepo) UpdateExecutionResultPath(ctx context.Context, id uuid.UUID, resultPath string, updatedAt time.Time) error {
	return nil
}

type statusRecordingRepo struct {
	statusUpdates int
	execution     *database.ExecutionIndex
}

func (r *statusRecordingRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return r.execution, nil
}

func (r *statusRecordingRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	r.statusUpdates++
	return nil
}

func (r *statusRecordingRepo) UpdateExecutionResultPath(ctx context.Context, id uuid.UUID, resultPath string, updatedAt time.Time) error {
	return nil
}

func TestRecordStepOutcomeDoesNotSetTerminalExecutionStatus(t *testing.T) {
	dataDir := t.TempDir()
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	repo := &statusRecordingRepo{execution: &database.ExecutionIndex{ID: plan.ExecutionID, Status: database.ExecutionStatusRunning}}
	writer := NewFileWriter(repo, storage.NewMemoryStorage(), nil, NewStaticRoot(dataDir))

	_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepIndex:   0,
		NodeID:      "recoverable-step",
		StepType:    "click",
		StartedAt:   time.Now().UTC(),
		Success:     false,
		Failure:     &contracts.StepFailure{Message: "expected recoverable failure"},
	})
	if err != nil {
		t.Fatalf("RecordStepOutcome() error = %v", err)
	}
	if repo.statusUpdates != 0 {
		t.Fatalf("RecordStepOutcome() changed terminal status %d times; terminal status belongs to executor/service", repo.statusUpdates)
	}
}

func TestRecordExecutionArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	artifactPath := filepath.Join(dataDir, "video.webm")
	if err := os.WriteFile(artifactPath, []byte("fake-video"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	memStore := storage.NewMemoryStorage()
	writer := NewFileWriter(noopRepo{}, memStore, nil, NewStaticRoot(dataDir))
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	}

	err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{
		{
			ArtifactType: "video_meta",
			Label:        "video-1",
			Path:         artifactPath,
			Payload: map[string]any{
				"page_index": 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("record artifacts: %v", err)
	}

	resultPath, err := writer.resultFilePath(context.Background(), plan.ExecutionID)
	if err != nil {
		t.Fatalf("result file path: %v", err)
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}

	var result bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ExecutionId != plan.ExecutionID.String() {
		t.Fatalf("expected execution id %s, got %s", plan.ExecutionID, result.ExecutionId)
	}
	if memStore.ObjectCount() != 1 {
		t.Fatalf("expected 1 stored artifact, got %d", memStore.ObjectCount())
	}
}

func TestScreenshotMetadataUsesEncodedPixelDimensions(t *testing.T) {
	for _, format := range []string{"png", "jpeg"} {
		t.Run(format, func(t *testing.T) {
			var encoded bytes.Buffer
			pixels := image.NewRGBA(image.Rect(0, 0, 780, 1688))
			var err error
			if format == "png" {
				err = png.Encode(&encoded, pixels)
			} else {
				err = jpeg.Encode(&encoded, pixels, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			outcome := contracts.StepOutcome{Screenshot: &contracts.Screenshot{Data: encoded.Bytes(), Width: 390, Height: 844, MediaType: "image/png"}}
			got := sanitizeOutcomeWithLimits(outcome, encoded.Len()+1, 1024, 1024, 1024)
			if got.Screenshot.Width != 780 || got.Screenshot.Height != 1688 || got.Screenshot.MediaType != "image/"+format {
				t.Fatalf("incorrect encoded image metadata: %dx%d %s", got.Screenshot.Width, got.Screenshot.Height, got.Screenshot.MediaType)
			}
			if !bytes.Equal(got.Screenshot.Data, encoded.Bytes()) {
				t.Fatal("image bytes changed")
			}
		})
	}
}

type outcomeIndexFault struct {
	noopRepo
	failure error
	writes  int
}

func (r *outcomeIndexFault) UpdateExecutionResultPath(context.Context, uuid.UUID, string, time.Time) error {
	r.writes++
	return r.failure
}

func TestRecordStepOutcomeRequiresDurableWrites(t *testing.T) {
	for _, stage := range []string{"timeline", "manifest", "index", "success"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			repo := &outcomeIndexFault{}
			writer := NewFileWriter(repo, storage.NewMemoryStorage(), nil, NewStaticRoot(root))
			var sentinel string
			if stage == "timeline" || stage == "manifest" {
				name := protoTimelineFileName
				if stage == "manifest" {
					name = resultFileName
				}
				barrier := filepath.Join(root, plan.ExecutionID.String(), name)
				require.NoError(t, os.MkdirAll(barrier, 0o700))
				sentinel = filepath.Join(barrier, "original")
				require.NoError(t, os.WriteFile(sentinel, []byte("retained"), 0o600))
			}
			if stage == "index" {
				repo.failure = errors.New("index storage failed")
			}
			result, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{ExecutionID: plan.ExecutionID, StepIndex: 0, Attempt: 1, NodeID: "action", StepType: "navigate", StartedAt: time.Now(), Success: true})
			if stage == "success" {
				require.NoError(t, err)
				require.NotNil(t, result.TimelineArtifactID)
				require.FileExists(t, filepath.Join(root, plan.ExecutionID.String(), resultFileName))
				require.FileExists(t, filepath.Join(root, plan.ExecutionID.String(), protoTimelineFileName))
				require.Equal(t, 1, repo.writes)
				return
			}
			require.Error(t, err)
			require.Nil(t, result.TimelineArtifactID, "failure cannot publish a completed receipt")
			if stage == "index" {
				require.ErrorIs(t, err, repo.failure)
				require.FileExists(t, filepath.Join(root, plan.ExecutionID.String(), resultFileName), "partial evidence stays owned")
			} else {
				require.Zero(t, repo.writes, "index cannot acknowledge missing evidence")
				data, readErr := os.ReadFile(sentinel)
				require.NoError(t, readErr)
				require.Equal(t, "retained", string(data))
			}
		})
	}
}

type screenshotReceiptStore struct {
	*storage.MemoryStorage
	fault string
	calls int
}

func (s *screenshotReceiptStore) StoreScreenshot(ctx context.Context, executionID uuid.UUID, name string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	s.calls++
	if s.fault == "error" {
		return nil, errors.New("screenshot store failed")
	}
	if s.fault == "nil-receipt" {
		return nil, nil
	}
	result, err := s.MemoryStorage.StoreScreenshot(ctx, executionID, name, data, contentType)
	switch s.fault {
	case "empty-url":
		result.URL = ""
	case "empty-object":
		result.ObjectName = ""
	case "wrong-size":
		result.SizeBytes++
	}
	return result, err
}

// [REQ:BAS-RH-J08] A required image cannot acknowledge absent/corrupt/truncated
// bytes or an invalid storage receipt. Optional capture and explicit discard
// keep their own verdict policy; all shapes preserve the caller's capture.
func TestScreenshotEvidenceRequiresValidReceipt(t *testing.T) {
	for _, stepType := range []string{"screenshot", "navigate"} {
		for _, fault := range []string{"success", "error", "nil-receipt", "empty-url", "empty-object", "wrong-size", "missing-storage", "missing-image", "invalid-image", "truncated-image", "over-budget", "disabled"} {
			t.Run(stepType+"/"+fault, func(t *testing.T) {
				var pngData bytes.Buffer
				require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 3))))
				data := pngData.Bytes()
				if fault == "invalid-image" {
					data = []byte("not an encoded image")
				}
				if fault == "truncated-image" {
					data = data[:len(data)-10]
				}
				imageData := &contracts.Screenshot{Data: data, MediaType: "image/jpeg", Width: 390, Height: 844}
				original := *imageData
				outcome := contracts.StepOutcome{ExecutionID: uuid.New(), StepIndex: 1, Attempt: 1, NodeID: "capture", StepType: stepType, StartedAt: time.Now(), Success: true, Screenshot: imageData, Notes: map[string]string{"caller": "preserved"}}
				if fault == "missing-image" {
					outcome.Screenshot = nil
				}
				originalNotes := map[string]string{"caller": "preserved"}
				store := &screenshotReceiptStore{MemoryStorage: storage.NewMemoryStorage(), fault: fault}
				var storageInterface storage.StorageInterface = store
				if fault == "missing-storage" {
					storageInterface = nil
				}
				dir := t.TempDir()
				writer := NewFileWriter(nil, storageInterface, nil, NewStaticRoot(dir))
				cfg := config.DefaultArtifactSettings()
				if fault == "over-budget" {
					cfg.MaxScreenshotBytes = len(data) - 1
				}
				if fault == "disabled" {
					cfg.CollectScreenshots = false
				}
				writer.SetArtifactConfig(&cfg)
				plan := contracts.ExecutionPlan{ExecutionID: outcome.ExecutionID, WorkflowID: uuid.New()}
				result, err := writer.RecordStepOutcome(context.Background(), plan, outcome)
				requiredFailure := stepType == "screenshot" && fault != "success" && fault != "disabled"
				if requiredFailure {
					require.Error(t, err)
					require.Nil(t, result.TimelineArtifactID)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, original, *imageData, "shaping cannot mutate caller-owned capture")
				require.Equal(t, originalNotes, outcome.Notes)
				manifest, readErr := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), resultFileName))
				require.NoError(t, readErr, "capture failures must retain a readable outcome")
				var timeline bastimeline.ExecutionTimeline
				require.NoError(t, (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(manifest, &timeline))
				require.Len(t, timeline.Entries, 1)
				require.Equal(t, !requiredFailure, timeline.Entries[0].Context.GetSuccess())
				if requiredFailure {
					require.Contains(t, timeline.Entries[0].Context.GetError(), "screenshot")
				}
				if fault != "success" {
					require.Empty(t, timeline.Entries[0].Telemetry.GetScreenshot().GetUrl(), "no image URL without a valid receipt")
					return
				}
				names, listErr := store.ListExecutionScreenshots(context.Background(), plan.ExecutionID)
				require.NoError(t, listErr)
				require.Len(t, names, 1)
				reader, _, getErr := store.GetScreenshot(context.Background(), names[0])
				require.NoError(t, getErr)
				stored, readErr := io.ReadAll(reader)
				require.NoError(t, readErr)
				require.NoError(t, reader.Close())
				require.Equal(t, data, stored)
				decoded, _, decodeErr := image.Decode(bytes.NewReader(stored))
				require.NoError(t, decodeErr)
				require.Equal(t, image.Rect(0, 0, 2, 3), decoded.Bounds())
			})
		}
	}
}

func BenchmarkScreenshotOutcomePersistence(b *testing.B) {
	pixels := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	for y := 0; y < 720; y++ {
		for x := 0; x < 1280; x++ {
			pixels.SetRGBA(x, y, color.RGBA{R: uint8(x / 8), G: uint8(y / 8), B: uint8((x + y) / 16), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, pixels); err != nil {
		b.Fatal(err)
	}
	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(b.TempDir()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
		_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{ExecutionID: plan.ExecutionID, StepIndex: i, Attempt: 1, NodeID: "capture", StepType: "screenshot", StartedAt: time.Now(), Success: true, Screenshot: &contracts.Screenshot{Data: encoded.Bytes(), MediaType: "image/png", Width: 1280, Height: 720}})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestScreenshotAndOutcomeWriteFailuresBothSurvive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "occupied")
	require.NoError(t, os.WriteFile(root, []byte("original"), 0o600))
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	writer := NewFileWriter(nil, &screenshotReceiptStore{MemoryStorage: storage.NewMemoryStorage(), fault: "error"}, nil, NewStaticRoot(root))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	result, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{ExecutionID: plan.ExecutionID, StepType: "screenshot", Success: true, Screenshot: &contracts.Screenshot{Data: data.Bytes()}})
	require.ErrorContains(t, err, "screenshot store failed")
	require.ErrorContains(t, err, "not a directory")
	require.Nil(t, result.TimelineArtifactID)
	original, readErr := os.ReadFile(root)
	require.NoError(t, readErr)
	require.Equal(t, "original", string(original))
}

// [REQ:BAS-RH-J24] Real durable timeline retains the JSON contract, not Go debug text.
func TestStructuredOutcomeSurvivesDiskProjection(t *testing.T) {
	type sample struct {
		Identity int64    `json:"identity"`
		Labels   []string `json:"labels"`
	}
	large := int64(9007199254740993)
	item := sample{Identity: large, Labels: []string{"first", "second"}}
	var absent *sample
	cases := []struct {
		name  string
		value any
	}{
		{"primitive", large}, {"map", map[string]any{"identity": large}},
		{"struct", item}, {"pointer", &item}, {"typed-list", []sample{item}},
		{"nil-pointer", absent}, {"proto-shaped-map", map[string]any{"string_value": "data"}},
		{"json-number", json.Number("9007199254740993")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			now := time.Date(2026, 9, 22, 10, 0, 0, 123, time.UTC)
			outcome := contracts.StepOutcome{SchemaVersion: contracts.StepOutcomeSchemaVersion, PayloadVersion: contracts.PayloadVersion,
				ExecutionID: plan.ExecutionID, CorrelationID: "attempt-3", StepIndex: 7, Attempt: 3, NodeID: "typed-evidence", StepType: "assert",
				StartedAt: now, CompletedAt: &now, Failure: &contracts.StepFailure{Kind: contracts.FailureKindUser, Code: "EXPECTED_FAILURE", Message: "fixture failed", Details: map[string]any{"value": tc.value}},
				Assertion:     &contracts.AssertionOutcome{Mode: "equals", Expected: tc.value, Actual: tc.value},
				ExtractedData: map[string]any{"value": tc.value}, Notes: map[string]string{"capture": "retained"},
			}
			result, err := writer.RecordStepOutcome(context.Background(), plan, outcome)
			require.NoError(t, err)
			require.NotNil(t, result.TimelineArtifactID)
			writer.ForgetExecution(plan.ExecutionID)
			data, err := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), "timeline.proto.json"))
			require.NoError(t, err)
			var timeline bastimeline.ExecutionTimeline
			require.NoError(t, protojson.Unmarshal(data, &timeline))
			require.Len(t, timeline.Entries, 1)
			entry := timeline.Entries[0]
			require.False(t, entry.Context.GetSuccess())
			require.Equal(t, "EXPECTED_FAILURE", entry.Context.GetErrorCode())
			var saved any
			for _, artifact := range entry.GetAggregates().GetArtifacts() {
				if value, ok := artifact.Payload["outcome"]; ok {
					saved = typeconv.JsonValueToAny(value)
				}
			}
			require.IsType(t, map[string]any{}, saved, "core outcome must remain a structured object")
			expectedJSON, err := json.Marshal(outcome)
			require.NoError(t, err)
			actualJSON, err := json.Marshal(saved)
			require.NoError(t, err)
			// UseNumber gives an independent JSON-contract oracle without rounding large integers.
			var expected, actual any
			expectedDecoder := json.NewDecoder(bytes.NewReader(expectedJSON))
			expectedDecoder.UseNumber()
			actualDecoder := json.NewDecoder(bytes.NewReader(actualJSON))
			actualDecoder.UseNumber()
			require.NoError(t, expectedDecoder.Decode(&expected))
			require.NoError(t, actualDecoder.Decode(&actual))
			require.Equal(t, expected, actual)
			valueJSON, err := json.Marshal(tc.value)
			require.NoError(t, err)
			for _, projected := range []any{typeconv.JsonValueToAny(entry.Context.Assertion.Expected), typeconv.JsonValueToAny(entry.Context.Assertion.Actual), typeconv.JsonValueToAny(entry.Aggregates.ExtractedDataPreview).(map[string]any)["value"]} {
				encoded, err := json.Marshal(projected)
				require.NoError(t, err)
				require.Equal(t, string(valueJSON), string(encoded), "assertion/extraction must retain exact values")
			}
		})
	}
}

func TestUnrepresentableOutcomeDoesNotAcknowledgeOrPoisonNextWrite(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"nested-function", map[string]any{"unavailable": func() {}}},
		{"unsigned-overflow", uint64(math.MaxUint64)},
		{"infinite", math.Inf(1)},
		{"cyclic-map", cycle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			outcome := contracts.StepOutcome{ExecutionID: plan.ExecutionID, NodeID: "bad", StepType: "extract", Success: true, ExtractedData: map[string]any{"value": tc.value}}
			result, err := writer.RecordStepOutcome(context.Background(), plan, outcome)
			require.Error(t, err, "unsupported evidence cannot receive a success receipt")
			require.Nil(t, result.TimelineArtifactID)
			outcome.NodeID = "good"
			outcome.ExtractedData = map[string]any{"value": int64(9007199254740993)}
			_, err = writer.RecordStepOutcome(context.Background(), plan, outcome)
			require.NoError(t, err)
			data, err := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), "timeline.proto.json"))
			require.NoError(t, err)
			var timeline bastimeline.ExecutionTimeline
			require.NoError(t, protojson.Unmarshal(data, &timeline))
			require.Len(t, timeline.Entries, 1, "failed projection must not contaminate next write")
			require.Equal(t, "good", timeline.Entries[0].GetNodeId())
		})
	}
}

func BenchmarkStructuredOutcomePersistence(b *testing.B) {
	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(b.TempDir()))
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	value := struct {
		Identity int64    `json:"identity"`
		Labels   []string `json:"labels"`
	}{9007199254740993, []string{"a", "b", "c"}}
	outcome := contracts.StepOutcome{SchemaVersion: contracts.StepOutcomeSchemaVersion, PayloadVersion: contracts.PayloadVersion,
		NodeID: "typed", StepType: "assert", Success: true, StartedAt: now, CompletedAt: &now, Attempt: 1,
		ExtractedData: map[string]any{"item": value}, Assertion: &contracts.AssertionOutcome{Expected: value, Actual: value, Success: true},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
		outcome.ExecutionID = plan.ExecutionID
		if _, err := writer.RecordStepOutcome(context.Background(), plan, outcome); err != nil {
			b.Fatal(err)
		}
		writer.ForgetExecution(plan.ExecutionID)
	}
}

func TestCheckpointCommitPreservesPreviousStateOnEncodingFailure(t *testing.T) {
	root := t.TempDir()
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(root))
	cp := Checkpoint{SchemaVersion: CheckpointVersion, ExecutionID: uuid.New(), WorkflowID: uuid.New(), LastStepIndex: 0, NodeID: "first", TotalSteps: 2, Store: map[string]any{"secret": "private-only", "nested": []any{true, nil, map[string]any{"value": 12.5}}}}
	require.NoError(t, writer.RecordCheckpoint(context.Background(), cp))
	path := filepath.Join(root, cp.ExecutionID.String(), CheckpointFileName)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	got, err := ReadCheckpoint(filepath.Join(filepath.Dir(path), "result.json"), cp.ExecutionID, cp.WorkflowID)
	require.NoError(t, err)
	require.Equal(t, cp, *got)
	cp.LastStepIndex = 1
	cp.Store = map[string]any{"not_json": math.NaN()}
	require.ErrorContains(t, writer.RecordCheckpoint(context.Background(), cp), "encode recovery state")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "failed commit must preserve previous cursor and store together")
}

func TestCheckpointCommitReportsRealFilesystemFailure(t *testing.T) {
	root := t.TempDir()
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(root))
	cp := Checkpoint{SchemaVersion: CheckpointVersion, ExecutionID: uuid.New(), WorkflowID: uuid.New(), LastStepIndex: 0, NodeID: "first", TotalSteps: 1, Store: map[string]any{}}
	path := filepath.Join(root, cp.ExecutionID.String(), CheckpointFileName)
	require.NoError(t, os.MkdirAll(path, 0o700))
	require.ErrorContains(t, writer.RecordCheckpoint(context.Background(), cp), "commit recovery state")
	_, err := ReadCheckpoint(filepath.Join(filepath.Dir(path), "result.json"), cp.ExecutionID, cp.WorkflowID)
	require.Error(t, err, "a failed filesystem commit cannot become usable recovery state")
}
