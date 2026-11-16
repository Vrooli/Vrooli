package executionwriter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/services/evidence"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const evidenceManifestFileName = "evidence.proto.json"

func (r *FileWriter) evidenceManifestFilePath(ctx context.Context, executionID uuid.UUID) (string, error) {
	if r == nil || r.root == nil {
		return "", fmt.Errorf("execution artifact root is unavailable")
	}
	root, err := r.root.Root(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve execution artifact root: %w", err)
	}
	return filepath.Join(root, executionID.String(), evidenceManifestFileName), nil
}

// writeEvidenceManifest persists the renderer-neutral package inventory beside
// result.json. It is deliberately generated from safe metadata; neither
// capture paths nor object-store keys can enter this document.
func (r *FileWriter) writeEvidenceManifest(ctx context.Context, executionID uuid.UUID, result *ExecutionResultData, timeline *executionTimelineData) error {
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
		sha := strings.TrimSpace(artifact.SHA256)
		if sha == "" {
			return fmt.Errorf("artifact %s (%s) has no byte-derived SHA-256", artifact.ArtifactID, artifact.ArtifactType)
		}
		size := int64(0)
		if artifact.SizeBytes != nil {
			size = *artifact.SizeBytes
		}
		inputs = append(inputs, evidence.ArtifactInput{ID: artifact.ArtifactID, Kind: artifact.ArtifactType, MediaType: evidenceMediaType(artifact), SizeBytes: size, SHA256: sha, Producer: "execution-writer", CapturedAt: time.Now().UTC(), Provenance: evidence.ArtifactProvenanceInput{ArtifactType: artifact.ArtifactType}})
	}

	entries := []*bastimeline.TimelineEntry(nil)
	if timeline != nil {
		timeline.mu.Lock()
		if timeline.pb != nil {
			entries = make([]*bastimeline.TimelineEntry, 0, len(timeline.pb.Entries))
			for _, entry := range timeline.pb.Entries {
				if entry != nil {
					portable := proto.Clone(entry).(*bastimeline.TimelineEntry)
					sanitizeReplayEntryForEvidence(portable)
					entries = append(entries, portable)
				}
			}
		}
		timeline.mu.Unlock()
	}
	packageData, err := evidence.BuildReplayPackage(executionID.String(), workflowID, evidence.DefaultPolicy(), inputs, entries, evidence.ReplayPresentationInput{}, time.Now().UTC())
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
	path, err := r.evidenceManifestFilePath(ctx, executionID)
	if err != nil {
		return err
	}
	if err := corestorage.WriteFileAtomic(path, raw, 0o644); err != nil {
		return fmt.Errorf("write evidence manifest: %w", err)
	}
	r.root.RecordWrite(ctx)
	return nil
}

// sanitizeReplayEntryForEvidence keeps typed artifact identity and display
// metadata while removing storage locations and payload contents. The evidence
// manifest is portable: object keys, local paths and download URLs stay behind
// BAS's authorized artifact resolver.
func sanitizeReplayEntryForEvidence(entry *bastimeline.TimelineEntry) {
	if entry == nil {
		return
	}
	if aggregates := entry.GetAggregates(); aggregates != nil {
		for _, artifact := range aggregates.Artifacts {
			if artifact == nil {
				continue
			}
			artifact.StorageUrl = ""
			artifact.ThumbnailUrl = nil
			artifact.Payload = map[string]*commonv1.JsonValue{}
		}
	}
	telemetry := entry.GetTelemetry()
	if telemetry == nil {
		return
	}
	if screenshot := telemetry.GetScreenshot(); screenshot != nil {
		screenshot.Url = ""
		screenshot.ThumbnailUrl = ""
		screenshot.Path = nil
	}
	for _, artifact := range []*basdomain.TelemetryArtifact{
		telemetry.GetDomSnapshot(), telemetry.GetConsoleLogArtifact(), telemetry.GetNetworkEventArtifact(),
	} {
		if artifact == nil {
			continue
		}
		artifact.StorageUrl = ""
		artifact.Path = nil
	}
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
