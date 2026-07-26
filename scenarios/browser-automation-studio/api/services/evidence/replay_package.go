package evidence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	EvidenceSchemaVersion = "bas-evidence/v1"
	ReplaySchemaVersion   = "bas-replay/v1"
)

// ArtifactInput contains only portable metadata. A caller resolves artifact
// bytes through an authorized storage adapter; no filesystem or object-store
// paths can enter a ReplayPackage through this API.
type ArtifactInput struct {
	ID, Kind, MediaType, SHA256, Producer string
	SizeBytes                             int64
	CapturedAt                            time.Time
	TimelineEntryID                       string
	Provenance                            map[string]any
}

// BuildReplayPackage makes the renderer-neutral handoff consumed by preview
// and export code. It rejects incomplete identifiers and unknown artifact kinds
// rather than emitting a package that later consumers must guess how to read.
func BuildReplayPackage(executionID, workflowID string, policy *basevidence.EvidencePolicy, artifacts []ArtifactInput, timeline []*bastimeline.TimelineEntry, presentation map[string]any, now time.Time) (*basevidence.ReplayPackage, error) {
	if _, err := uuid.Parse(strings.TrimSpace(executionID)); err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}
	if strings.TrimSpace(workflowID) != "" {
		if _, err := uuid.Parse(workflowID); err != nil {
			return nil, fmt.Errorf("invalid workflow ID: %w", err)
		}
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	manifest := &basevidence.EvidenceManifest{Id: uuid.NewString(), ExecutionId: executionID, SchemaVersion: EvidenceSchemaVersion, Policy: policy, CreatedAt: timestamppb.New(now)}
	for _, input := range artifacts {
		if _, err := uuid.Parse(strings.TrimSpace(input.ID)); err != nil {
			return nil, fmt.Errorf("invalid artifact ID: %w", err)
		}
		kind := KindFor(input.Kind)
		if kind == basevidence.ArtifactKind_ARTIFACT_KIND_UNSPECIFIED {
			return nil, fmt.Errorf("unknown artifact kind %q", input.Kind)
		}
		// INVARIANT: replayArtifactHasIntegrityDigest
		// A renderer receives portable integrity metadata, never a storage location.
		if len(input.SHA256) != 64 {
			return nil, fmt.Errorf("artifact %s has no SHA-256", input.ID)
		}
		classification, retention, access := ClassificationFor(kind, policy)
		artifact := &basevidence.ArtifactManifest{Id: input.ID, Kind: kind, MediaType: input.MediaType, SizeBytes: input.SizeBytes, Sha256: input.SHA256, Classification: classification, RetentionClass: retention, AccessPolicy: access, Redacted: kind == basevidence.ArtifactKind_ARTIFACT_KIND_HAR && policy.RedactHar, ExecutionId: executionID, Producer: input.Producer, CapturedAt: timestamppb.New(input.CapturedAt)}
		if strings.TrimSpace(input.TimelineEntryID) != "" {
			artifact.TimelineEntryId = &input.TimelineEntryID
		}
		if len(input.Provenance) > 0 {
			value, err := structpb.NewStruct(input.Provenance)
			if err != nil {
				return nil, fmt.Errorf("artifact provenance: %w", err)
			}
			artifact.Provenance = value
		}
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Id < manifest.Artifacts[j].Id })
	presentationStruct, err := structpb.NewStruct(presentation)
	if err != nil {
		return nil, fmt.Errorf("presentation: %w", err)
	}
	timelineValues := make([]*structpb.Struct, 0, len(timeline))
	for _, entry := range timeline {
		if entry == nil {
			continue
		}
		raw, err := protojson.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("marshal replay timeline entry: %w", err)
		}
		value := &structpb.Struct{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, value); err != nil {
			return nil, fmt.Errorf("convert replay timeline entry: %w", err)
		}
		timelineValues = append(timelineValues, value)
	}
	pack := &basevidence.ReplayPackage{Id: uuid.NewString(), SchemaVersion: ReplaySchemaVersion, ExecutionId: executionID, Evidence: manifest, Timeline: timelineValues, Presentation: presentationStruct, CreatedAt: timestamppb.New(now)}
	if strings.TrimSpace(workflowID) != "" {
		pack.WorkflowId = &workflowID
	}
	return pack, nil
}
