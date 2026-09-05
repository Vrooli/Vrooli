package evidence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/proto"
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
	Provenance                            ArtifactProvenanceInput
}

// ArtifactProvenanceInput is the typed, portable provenance supplied by the
// writer. It cannot carry storage locations or arbitrary raw capture data.
type ArtifactProvenanceInput struct {
	Source       string
	ArtifactType string
}

// ReplayPresentationInput contains the bounded renderer hints accepted by a
// replay package.
type ReplayPresentationInput struct {
	Theme string
}

// BuildReplayPackage makes the renderer-neutral handoff consumed by preview
// and export code. It rejects incomplete identifiers and unknown artifact kinds
// rather than emitting a package that later consumers must guess how to read.
func BuildReplayPackage(executionID, workflowID string, policy *basevidence.EvidencePolicy, artifacts []ArtifactInput, timeline []*bastimeline.TimelineEntry, presentation ReplayPresentationInput, now time.Time) (*basevidence.ReplayPackage, error) {
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
		if input.Provenance.Source != "" || input.Provenance.ArtifactType != "" {
			artifact.Provenance = &basevidence.ArtifactProvenance{
				Source:       optionalString(input.Provenance.Source),
				ArtifactType: optionalString(input.Provenance.ArtifactType),
			}
		}
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Id < manifest.Artifacts[j].Id })
	timelineValues := make([]*bastimeline.TimelineEntry, 0, len(timeline))
	for _, entry := range timeline {
		if entry == nil {
			continue
		}
		timelineValues = append(timelineValues, proto.Clone(entry).(*bastimeline.TimelineEntry))
	}
	pack := &basevidence.ReplayPackage{Id: uuid.NewString(), SchemaVersion: ReplaySchemaVersion, ExecutionId: executionID, Evidence: manifest, Timeline: timelineValues, Presentation: &basevidence.ReplayPresentation{Theme: optionalString(presentation.Theme)}, CreatedAt: timestamppb.New(now)}
	if strings.TrimSpace(workflowID) != "" {
		pack.WorkflowId = &workflowID
	}
	return pack, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
