package executionwriter

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestRecordExecutionArtifactsPublishesSanitizedHarMetadata(t *testing.T) {
	dir := t.TempDir()
	harPath := filepath.Join(dir, "capture.har")
	if err := os.WriteFile(harPath, []byte(`{"log":{"entries":[{"request":{"url":"https://example.test/?token=secret","headers":[{"name":"Authorization","value":"Bearer secret"}]}}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, nil, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "har", Path: harPath, ContentType: "application/json"}}); err != nil {
		t.Fatal(err)
	}
	result := writer.getOrCreateResult(plan)
	if len(result.Artifacts) != 1 {
		t.Fatalf("artifacts = %d", len(result.Artifacts))
	}
	artifact := result.Artifacts[0]
	if _, ok := artifact.Payload["path"]; ok {
		t.Fatal("local path leaked into artifact payload")
	}
	if _, ok := artifact.Payload["source_path"]; ok {
		t.Fatal("source path leaked into artifact payload")
	}
	if artifact.AccessPolicy != "ACCESS_POLICY_PROTECTED_STORAGE_ONLY" || !artifact.Redacted || len(artifact.SHA256) != 64 {
		t.Fatalf("unexpected protected HAR metadata: %#v", artifact)
	}
	encoded, _ := artifact.Payload["sanitized_base64"].(string)
	if strings.Contains(encoded, "secret") || encoded == "" {
		t.Fatalf("HAR derivative missing or unsafe: %q", encoded)
	}
	sanitized, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode sanitized HAR: %v", err)
	}
	digest := sha256.Sum256(sanitized)
	if artifact.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("HAR digest does not match published sanitized bytes: got %s", artifact.SHA256)
	}
}

func TestRecordExecutionArtifactsRedactsSourcePathPayload(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "capture.zip")
	if err := os.WriteFile(tracePath, []byte("trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	if err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: tracePath, ContentType: "application/zip", Payload: map[string]any{"source_path": "/private/capture.zip"}}}); err != nil {
		t.Fatal(err)
	}
	if _, leaked := writer.getOrCreateResult(plan).Artifacts[0].Payload["source_path"]; leaked {
		t.Fatal("source path leaked into execution artifact")
	}
}

type rejectingArtifactStorage struct {
	storage.StorageInterface
	err     error
	receipt *storage.ArtifactInfo
}

func (s rejectingArtifactStorage) StoreArtifactFromFile(context.Context, uuid.UUID, string, string, string) (*storage.ArtifactInfo, error) {
	return s.receipt, s.err
}

func (s rejectingArtifactStorage) StoreArtifact(context.Context, string, []byte, string) (*storage.ArtifactInfo, error) {
	return s.receipt, s.err
}

func TestExternalArtifactsRejectMissingOrUncommittedEvidence(t *testing.T) {
	failure := errors.New("synthetic artifact storage failure")
	for _, tc := range []struct {
		name, kind, data string
		backend          storage.StorageInterface
		missing          bool
	}{
		{name: "missing source", kind: "trace", missing: true},
		{name: "malformed HAR", kind: "har", data: "invalid HAR"},
		{name: "trace without storage", kind: "trace", data: "trace bytes"},
		{name: "video store fails", kind: "video", data: "video bytes", backend: rejectingArtifactStorage{err: failure}},
		{name: "trace store fails", kind: "trace", data: "trace bytes", backend: rejectingArtifactStorage{err: failure}},
		{name: "HAR store fails", kind: "har", data: `{"log":{"entries":[]}}`, backend: rejectingArtifactStorage{err: failure}},
		{name: "empty storage receipt", kind: "video", data: "video bytes", backend: rejectingArtifactStorage{}},
		{name: "storage receipt without object", kind: "video", data: "video bytes", backend: rejectingArtifactStorage{receipt: &storage.ArtifactInfo{URL: "/stored", SizeBytes: 11}}},
		{name: "storage receipt without URL", kind: "video", data: "video bytes", backend: rejectingArtifactStorage{receipt: &storage.ArtifactInfo{ObjectName: "object", SizeBytes: 11}}},
		{name: "storage byte count mismatch", kind: "video", data: "video bytes", backend: rejectingArtifactStorage{receipt: &storage.ArtifactInfo{URL: "/stored", ObjectName: "object", SizeBytes: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "capture")
			if !tc.missing {
				require.NoError(t, os.WriteFile(path, []byte(tc.data), 0o600))
			}
			writer := NewFileWriter(nil, tc.backend, nil, NewStaticRoot(dir))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
			require.Error(t, writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: tc.kind, Path: path}}))
			require.Empty(t, writer.getOrCreateResult(plan).Artifacts)
			if !tc.missing {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, tc.data, string(data))
			}
		})
	}
}

func TestExternalTraceBytesSurviveSourceRemoval(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "trace.zip")
	require.NoError(t, os.WriteFile(source, []byte("complete trace bytes"), 0o600))
	backend := storage.NewMemoryStorage()
	writer := NewFileWriter(nil, backend, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	require.NoError(t, writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "trace", Path: source, ContentType: "application/zip"}}))
	require.NoError(t, os.Remove(source))
	artifacts := writer.getOrCreateResult(plan).Artifacts
	require.Len(t, artifacts, 1)
	require.NotEmpty(t, artifacts[0].StorageURL)
	object, ok := artifacts[0].Payload["storage_object"].(string)
	require.True(t, ok)
	reader, _, err := backend.GetArtifact(context.Background(), object)
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "complete trace bytes", string(data))
	digest := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(digest[:]), artifacts[0].SHA256)
}

func TestExternalArtifactFailureRetainsValidSibling(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "trace.zip")
	require.NoError(t, os.WriteFile(source, []byte("valid trace"), 0o600))
	writer := NewFileWriter(nil, storage.NewMemoryStorage(), nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	err := writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{
		{ArtifactType: "trace", Path: filepath.Join(dir, "missing.zip")},
		{ArtifactType: "trace", Path: source},
	})
	require.Error(t, err)
	require.Len(t, writer.getOrCreateResult(plan).Artifacts, 1)
	require.NotEmpty(t, writer.getOrCreateResult(plan).Artifacts[0].StorageURL)
}

func TestStoredHARDerivativeContainsNoRawSecret(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "capture.har")
	raw := []byte(`{"log":{"entries":[{"request":{"url":"https://fixture.invalid/?token=SYNTHETIC-TOKEN","headers":[{"name":"Authorization","value":"SYNTHETIC-BEARER"}]}}]}}`)
	require.NoError(t, os.WriteFile(source, raw, 0o600))
	backend := storage.NewMemoryStorage()
	writer := NewFileWriter(nil, backend, nil, NewStaticRoot(dir))
	plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New()}
	require.NoError(t, writer.RecordExecutionArtifacts(context.Background(), plan, []ExternalArtifact{{ArtifactType: "har", Path: source}}))
	artifact := writer.getOrCreateResult(plan).Artifacts[0]
	reader, _, err := backend.GetArtifact(context.Background(), artifact.Payload["storage_object"].(string))
	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NotContains(t, string(data), "SYNTHETIC-TOKEN")
	require.NotContains(t, string(data), "SYNTHETIC-BEARER")
	digest := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(digest[:]), artifact.SHA256)
	preserved, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, raw, preserved)
}
