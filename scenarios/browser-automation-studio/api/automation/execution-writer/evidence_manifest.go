package executionwriter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const evidenceManifestFileName = "evidence.proto.json"

func (r *FileWriter) evidenceManifestFilePath(executionID uuid.UUID) string {
	return filepath.Join(r.dataDir, executionID.String(), evidenceManifestFileName)
}

// writeEvidenceManifest persists the renderer-neutral package inventory beside
// result.json. It is deliberately generated from safe metadata; neither
// capture paths nor object-store keys can enter this document.
func (r *FileWriter) writeEvidenceManifest(executionID uuid.UUID, result *ExecutionResultData, timeline *executionTimelineData) error {
	if result == nil {
		return nil
	}
	result.mu.Lock()
	workflowID := result.WorkflowID
	artifacts := append([]ArtifactData(nil), result.Artifacts...)
	result.mu.Unlock()

	inputs := make([]evidence.ArtifactInput, 0, len(artifacts))
	for _, artifact := range artifacts {
		if evidence.KindFor(artifact.ArtifactType).String() == "ARTIFACT_KIND_UNSPECIFIED" {
			continue
		}
		payload, err := json.Marshal(artifact.Payload)
		if err != nil {
			return fmt.Errorf("marshal artifact %s metadata: %w", artifact.ArtifactID, err)
		}
		descriptor := evidence.Describe(artifact.ArtifactType, evidenceMediaType(artifact), payload, evidence.DefaultPolicy())
		sha := strings.TrimSpace(artifact.SHA256)
		if sha == "" {
			sha = descriptor.SHA256
		}
		inputs = append(inputs, evidence.ArtifactInput{ID: artifact.ArtifactID, Kind: artifact.ArtifactType, MediaType: evidenceMediaType(artifact), SizeBytes: valueOr(descriptor.SizeBytes, artifact.SizeBytes), SHA256: sha, Producer: "execution-writer", CapturedAt: time.Now().UTC(), Provenance: map[string]any{"artifact_type": artifact.ArtifactType}})
	}

	entries := []*bastimeline.TimelineEntry(nil)
	if timeline != nil {
		timeline.mu.Lock()
		if timeline.pb != nil {
			entries = make([]*bastimeline.TimelineEntry, 0, len(timeline.pb.Entries))
			for _, entry := range timeline.pb.Entries {
				if entry != nil {
					entries = append(entries, proto.Clone(entry).(*bastimeline.TimelineEntry))
				}
			}
		}
		timeline.mu.Unlock()
	}
	packageData, err := evidence.BuildReplayPackage(executionID.String(), workflowID, evidence.DefaultPolicy(), inputs, entries, map[string]any{}, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("build replay package: %w", err)
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(packageData)
	if err != nil {
		return fmt.Errorf("marshal evidence manifest: %w", err)
	}
	var indented bytes.Buffer
	if json.Indent(&indented, raw, "", "  ") == nil {
		raw = indented.Bytes()
	}
	path := r.evidenceManifestFilePath(executionID)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write evidence manifest: %w", err)
	}
	return nil
}

func evidenceMediaType(artifact ArtifactData) string {
	if contentType := strings.TrimSpace(artifact.ContentType); contentType != "" {
		return contentType
	}
	switch evidence.KindFor(artifact.ArtifactType).String() {
	case "ARTIFACT_KIND_SCREENSHOT":
		return "image/png"
	case "ARTIFACT_KIND_VIDEO":
		return "video/webm"
	case "ARTIFACT_KIND_TRACE":
		return "application/zip"
	default:
		return "application/json"
	}
}

func valueOr(fallback int64, value *int64) int64 {
	if value != nil {
		return *value
	}
	return fallback
}
