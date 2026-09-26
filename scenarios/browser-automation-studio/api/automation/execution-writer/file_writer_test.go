package executionwriter

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/storage"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
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

type captureArtifactStorage struct {
	*storage.MemoryStorage
	objects map[string][]byte
}

func (s *captureArtifactStorage) StoreArtifact(ctx context.Context, name string, data []byte, contentType string) (*storage.ArtifactInfo, error) {
	s.objects[name] = append([]byte(nil), data...)
	return s.MemoryStorage.StoreArtifact(ctx, name, data, contentType)
}

func TestNetworkEvidenceRedactsSecretsBeforePublishingInlineAndStoredArtifacts(t *testing.T) {
	const (
		urlToken  = "synthetic-url-secret"
		headerKey = "synthetic-header-secret"
		bodyToken = "synthetic-body-secret"
	)
	dir := t.TempDir()
	store := &captureArtifactStorage{MemoryStorage: storage.NewMemoryStorage(), objects: map[string][]byte{}}
	writer := NewFileWriter(noopRepo{}, store, nil, NewStaticRoot(dir))
	executionID := uuid.New()
	plan := contracts.ExecutionPlan{ExecutionID: executionID, WorkflowID: uuid.New()}
	outcome := contracts.StepOutcome{
		ExecutionID: executionID, StepIndex: 1, Attempt: 1, NodeID: "request", StepType: "navigate",
		StartedAt: time.Now(), Success: true,
		FinalURL: "https://example.test/final?token=" + urlToken,
		Network: []contracts.NetworkEvent{{
			Type: "request", URL: "https://example.test/data?token=" + urlToken + "&safe=kept",
			Method: "POST", Timestamp: time.Now(),
			RequestHeaders:      map[string]string{"Authorization": "Bearer " + headerKey, "Accept": "application/json"},
			RequestBodyPreview:  `{"access_token":"` + bodyToken + `","safe":"kept"}`,
			ResponseBodyPreview: "opaque response preview",
		}},
	}
	if _, err := writer.RecordStepOutcome(context.Background(), plan, outcome); err != nil {
		t.Fatalf("record outcome: %v", err)
	}
	result := writer.getOrCreateResult(plan)
	var network ArtifactData
	for _, artifact := range result.Artifacts {
		if artifact.ArtifactType == "network" {
			network = artifact
			break
		}
	}
	if network.ArtifactID == "" {
		t.Fatal("network replay artifact was not persisted")
	}
	inline, err := json.Marshal(network.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(inline), urlToken) {
		t.Fatalf("inline network artifact retained query secret: %s", inline)
	}
	if len(store.objects) != 1 {
		t.Fatalf("stored network artifacts = %d, want 1", len(store.objects))
	}
	for _, stored := range store.objects {
		for _, secret := range []string{urlToken, headerKey, bodyToken, "opaque response preview"} {
			if strings.Contains(string(stored), secret) {
				t.Fatalf("stored network evidence retained %q: %s", secret, stored)
			}
		}
	}
	rawTimeline, err := os.ReadFile(filepath.Join(dir, executionID.String(), protoTimelineFileName))
	if err != nil {
		t.Fatal(err)
	}
	var timeline bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawTimeline, &timeline); err != nil {
		t.Fatal(err)
	}
	if got := timeline.Entries[0].Telemetry.GetUrl(); strings.Contains(got, urlToken) {
		t.Fatalf("timeline URL retained query secret: %s", got)
	}
}

func TestInlineTelemetryRemainsAttributableWhenSnapshotStorageFails(t *testing.T) {
	root := t.TempDir()
	backend := rejectingArtifactStorage{err: errors.New("synthetic telemetry snapshot failure")}
	writer := NewFileWriter(nil, backend, nil, NewStaticRoot(root))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	now := time.Now().UTC()

	result, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepType:    "click",
		Success:     true,
		ConsoleLogs: []contracts.ConsoleLogEntry{{Type: "log", Text: "fixture console entry", Timestamp: now}},
		Network:     []contracts.NetworkEvent{{Type: "response", URL: "https://fixture.invalid/data", Status: 200, Timestamp: now}},
	})
	require.NoError(t, err, "inline evidence remains the durable source when optional snapshots fail")
	require.NotNil(t, result.TimelineArtifactID)

	artifacts := writer.getOrCreateResult(plan).Artifacts
	inline := map[string]ArtifactData{}
	for _, artifact := range artifacts {
		if artifact.ArtifactType == "console" || artifact.ArtifactType == "network" {
			inline[artifact.ArtifactType] = artifact
		}
	}
	require.Len(t, inline, 2)
	for _, kind := range []string{"console", "network"} {
		artifact := inline[kind]
		require.Len(t, artifact.SHA256, 64, "%s artifact must be byte-attributable", kind)
		require.NotEmpty(t, artifact.ArtifactID, "%s artifact must remain attributable", kind)
		require.NotEmpty(t, artifact.Payload, "%s inline payload must survive snapshot failure", kind)
	}
	require.Equal(t, "fixture console entry", inline["console"].Payload["text"])
	require.Equal(t, "https://fixture.invalid/data", inline["network"].Payload["url"])

	manifestPath, err := writer.evidenceManifestFilePath(context.Background(), plan.ExecutionID)
	require.NoError(t, err)
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "integrity references must survive in the durable evidence manifest")
	var manifest basevidence.ReplayPackage
	require.NoError(t, (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(manifestBytes, &manifest))
	refs := map[string]*basevidence.ArtifactManifest{}
	for _, artifact := range manifest.GetEvidence().GetArtifacts() {
		if artifact.GetKind().String() == "ARTIFACT_KIND_CONSOLE" || artifact.GetKind().String() == "ARTIFACT_KIND_NETWORK" {
			refs[artifact.GetKind().String()] = artifact
		}
	}
	require.Len(t, refs, 2)
	for _, ref := range refs {
		require.Len(t, ref.GetSha256(), 64)
		require.NotEmpty(t, ref.GetId())
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
	delay time.Duration
}

func (s *screenshotReceiptStore) StoreScreenshot(ctx context.Context, executionID uuid.UUID, name string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	s.calls++
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.fault == "error" {
		return nil, errors.New("screenshot store failed")
	}
	if s.fault == "context-canceled" {
		return nil, context.Canceled
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

func TestRecordStepOutcomeReportsScreenshotStorageWait(t *testing.T) {
	const storageDelay = 30 * time.Millisecond
	store := &screenshotReceiptStore{MemoryStorage: storage.NewMemoryStorage(), delay: storageDelay}
	root := t.TempDir()
	writer := NewFileWriter(nil, store, nil, NewStaticRoot(root))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))))

	_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepType:    "screenshot",
		Success:     true,
		Screenshot:  &contracts.Screenshot{Data: encoded.Bytes()},
	})
	require.NoError(t, err)
	require.Equal(t, 1, store.calls)
	timelineData, err := os.ReadFile(filepath.Join(root, plan.ExecutionID.String(), "timeline.proto.json"))
	require.NoError(t, err)
	var timeline bastimeline.ExecutionTimeline
	require.NoError(t, protojson.Unmarshal(timelineData, &timeline))
	require.Len(t, timeline.Entries, 1)
	var storedSpan int64
	for _, artifact := range timeline.Entries[0].GetAggregates().GetArtifacts() {
		if artifact.GetType().String() != "ARTIFACT_TYPE_SCREENSHOT" {
			continue
		}
		storedSpan = artifact.GetPayload()["storage_duration_ns"].GetIntValue()
	}
	require.GreaterOrEqual(t, storedSpan, storageDelay.Nanoseconds(),
		"persisted span must include time blocked inside StoreScreenshot")
	t.Logf("controlled StoreScreenshot wait: requested=%s reported=%s", storageDelay, time.Duration(storedSpan))
	require.NotZero(t, storedSpan,
		"screenshot timeline artifact must retain the storage span for later owner analysis")
}

func TestFailedScreenshotStorageDurationSurvivesOutcome(t *testing.T) {
	const storageDelay = 30 * time.Millisecond
	for _, fault := range []string{"error", "context-canceled"} {
		t.Run(fault, func(t *testing.T) {
			root := t.TempDir()
			store := &screenshotReceiptStore{MemoryStorage: storage.NewMemoryStorage(), fault: fault, delay: storageDelay}
			writer := NewFileWriter(nil, store, nil, NewStaticRoot(root))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			var encoded bytes.Buffer
			require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))))

			result, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
				ExecutionID: plan.ExecutionID,
				StepType:    "screenshot",
				Success:     true,
				Screenshot:  &contracts.Screenshot{Data: encoded.Bytes()},
			})
			require.Error(t, err)
			require.Nil(t, result.TimelineArtifactID, "failed required screenshot must not return a completed receipt")
			require.Equal(t, 1, store.calls)

			timelineData, readErr := os.ReadFile(filepath.Join(root, plan.ExecutionID.String(), "timeline.proto.json"))
			require.NoError(t, readErr, "failed capture should retain its failure evidence")
			var timeline bastimeline.ExecutionTimeline
			require.NoError(t, protojson.Unmarshal(timelineData, &timeline))
			require.Len(t, timeline.Entries, 1)
			var savedOutcome map[string]any
			for _, artifact := range timeline.Entries[0].GetAggregates().GetArtifacts() {
				if value := artifact.GetPayload()["outcome"]; value != nil {
					savedOutcome, _ = typeconv.JsonValueToAny(value).(map[string]any)
				}
			}
			require.NotNil(t, savedOutcome)
			notes, ok := savedOutcome["notes"].(map[string]any)
			require.True(t, ok)
			nanosText, ok := notes["screenshot_storage_duration_ns"].(string)
			require.True(t, ok, "failed storage timing must be retained in core outcome notes")
			nanos, parseErr := strconv.ParseInt(nanosText, 10, 64)
			require.NoError(t, parseErr)
			require.GreaterOrEqual(t, nanos, storageDelay.Nanoseconds())
			require.Contains(t, notes, "screenshot_persistence_error")
		})
	}
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

func TestManagedFullPageScreenshotPersistsWithinDecodeBudget(t *testing.T) {
	data, err := os.ReadFile("testdata/full-page-1280x12800.png")
	require.NoError(t, err)
	config, format, weight, err := screenshotDecodeConfig(data)
	require.NoError(t, err)
	require.Equal(t, "png", format)
	require.Equal(t, 1280, config.Width)
	require.Equal(t, 12800, config.Height)
	require.Less(t, weight, maxActiveScreenshotDecodeBytes)
	require.Greater(t, weight*2, maxActiveScreenshotDecodeBytes,
		"measured Go heap retention makes two concurrent full-page raster decodes exceed the observed resident-memory envelope")

	store := storage.NewMemoryStorage()
	writer := NewFileWriter(nil, store, nil, NewStaticRoot(t.TempDir()))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	result, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepType:    "screenshot",
		Success:     true,
		Screenshot:  &contracts.Screenshot{Data: data, MediaType: "image/png"},
	})
	require.NoError(t, err)
	require.NotNil(t, result.TimelineArtifactID)
	names, err := store.ListExecutionScreenshots(context.Background(), plan.ExecutionID)
	require.NoError(t, err)
	require.Len(t, names, 1)
	reader, _, err := store.GetScreenshot(context.Background(), names[0])
	require.NoError(t, err)
	stored, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, data, stored, "bounded integrity validation must preserve every full-page pixel")
}

func TestScreenshotRasterOverBudgetFailsBeforeFullDecode(t *testing.T) {
	data := pngIHDR(10000, 10000)
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err, "DecodeConfig reads the bounded header without allocating the raster")
	require.Equal(t, "png", format)
	require.Equal(t, 10000, config.Width)
	require.Equal(t, 10000, config.Height)
	_, _, _, err = screenshotDecodeConfig(data)
	require.ErrorContains(t, err, "active screenshot decode budget")

	store := storage.NewMemoryStorage()
	writer := NewFileWriter(nil, store, nil, NewStaticRoot(t.TempDir()))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	_, err = writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepType:    "screenshot",
		Success:     true,
		Screenshot:  &contracts.Screenshot{Data: data, MediaType: "image/png"},
	})
	require.ErrorContains(t, err, "active screenshot decode budget")
	names, listErr := store.ListExecutionScreenshots(context.Background(), plan.ExecutionID)
	require.NoError(t, listErr)
	require.Empty(t, names, "an over-budget raster must never reach storage")
}

func TestScreenshotJPEGRasterOverBudgetFailsBeforeFullDecode(t *testing.T) {
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil))
	data := encoded.Bytes()
	sof := bytes.Index(data, []byte{0xff, 0xc0})
	require.NotEqual(t, -1, sof, "test JPEG should contain a baseline start-of-frame header")
	binary.BigEndian.PutUint16(data[sof+5:sof+7], 10000)
	binary.BigEndian.PutUint16(data[sof+7:sof+9], 10000)

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, 10000, config.Width)
	require.Equal(t, 10000, config.Height)
	_, _, _, err = screenshotDecodeConfig(data)
	require.ErrorContains(t, err, "active screenshot decode budget")

	store := storage.NewMemoryStorage()
	writer := NewFileWriter(nil, store, nil, NewStaticRoot(t.TempDir()))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	_, err = writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
		ExecutionID: plan.ExecutionID,
		StepType:    "screenshot",
		Success:     true,
		Screenshot:  &contracts.Screenshot{Data: data, MediaType: "image/jpeg"},
	})
	require.ErrorContains(t, err, "active screenshot decode budget")
	names, listErr := store.ListExecutionScreenshots(context.Background(), plan.ExecutionID)
	require.NoError(t, listErr)
	require.Empty(t, names, "an over-budget JPEG raster must never reach storage")
}

func pngIHDR(width, height uint32) []byte {
	data := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	chunk := make([]byte, 4+4+13)
	binary.BigEndian.PutUint32(chunk[:4], 13)
	copy(chunk[4:8], "IHDR")
	binary.BigEndian.PutUint32(chunk[8:12], width)
	binary.BigEndian.PutUint32(chunk[12:16], height)
	chunk[16] = 8
	chunk[17] = 2
	crc := crc32.ChecksumIEEE(chunk[4:])
	data = append(data, chunk...)
	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], crc)
	return append(data, checksum[:]...)
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

type solidRaster struct {
	bounds image.Rectangle
	color  color.Color
}

func (s solidRaster) ColorModel() color.Model { return color.RGBAModel }
func (s solidRaster) Bounds() image.Rectangle { return s.bounds }
func (s solidRaster) At(int, int) color.Color { return s.color }

// BenchmarkScreenshotOutcomePersistenceExpandedRaster keeps the compressed-byte
// versus decoded-pixel cost reproducible without allocating a source raster.
func BenchmarkScreenshotOutcomePersistenceExpandedRaster(b *testing.B) {
	const width, height = 1280, 12800
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, solidRaster{
		bounds: image.Rect(0, 0, width, height),
		color:  color.RGBA{R: 247, G: 247, B: 247, A: 255},
	}); err != nil {
		b.Fatal(err)
	}
	if encoded.Len() >= config.DefaultMaxScreenshotBytes {
		b.Fatalf("expanded-raster fixture is %d bytes; want less than screenshot limit %d", encoded.Len(), config.DefaultMaxScreenshotBytes)
	}

	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(b.TempDir()))
	b.ReportAllocs()
	b.SetBytes(int64(encoded.Len()))
	b.ResetTimer()
	b.ReportMetric(float64(width*height), "pixels/op")
	for i := 0; i < b.N; i++ {
		plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
		_, err := writer.RecordStepOutcome(context.Background(), plan, contracts.StepOutcome{
			ExecutionID: plan.ExecutionID,
			StepIndex:   i,
			Attempt:     1,
			NodeID:      "capture",
			StepType:    "screenshot",
			StartedAt:   time.Now(),
			Success:     true,
			Screenshot: &contracts.Screenshot{
				Data: encoded.Bytes(), MediaType: "image/png", Width: width, Height: height,
			},
		})
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
		{"primitive", large},
		{"map", map[string]any{"identity": large}},
		{"struct", item},
		{"pointer", &item},
		{"typed-list", []sample{item}},
		{"nil-pointer", absent},
		{"proto-shaped-map", map[string]any{"string_value": "data"}},
		{"json-number", json.Number("9007199254740993")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			now := time.Date(2026, 9, 22, 10, 0, 0, 123, time.UTC)
			outcome := contracts.StepOutcome{
				SchemaVersion: contracts.StepOutcomeSchemaVersion, PayloadVersion: contracts.PayloadVersion,
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
	outcome := contracts.StepOutcome{
		SchemaVersion: contracts.StepOutcomeSchemaVersion, PayloadVersion: contracts.PayloadVersion,
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
