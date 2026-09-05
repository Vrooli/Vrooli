package workflow

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/evidence"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *WorkflowService) readExecutionTimeline(resultPath string) (*bastimeline.ExecutionTimeline, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, err
	}
	var result bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result JSON: %w", err)
	}
	return &result, nil
}

// GetExecutionReplayPackage loads the writer-owned, versioned evidence package.
// It is the only supported consumer-facing replay handoff; callers never
// reconstruct it from BAS result or artifact filesystem paths.
func (s *WorkflowService) GetExecutionReplayPackage(ctx context.Context, executionID uuid.UUID) (*basevidence.ReplayPackage, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if strings.TrimSpace(execution.ResultPath) == "" {
		return nil, fmt.Errorf("replay package unavailable: execution has no result")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(execution.ResultPath), "evidence.proto.json"))
	if err != nil {
		return nil, fmt.Errorf("read replay package: %w", err)
	}
	var pack basevidence.ReplayPackage
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &pack); err != nil {
		return nil, fmt.Errorf("parse replay package: %w", err)
	}
	return &pack, nil
}

// GetExecutionScreenshots reads screenshots from the execution result file and returns proto types.
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if execution.ResultPath == "" {
		return []*basexecution.ExecutionScreenshot{}, nil
	}

	resultData, err := s.readExecutionTimeline(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*basexecution.ExecutionScreenshot{}, nil
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	screenshots := make([]*basexecution.ExecutionScreenshot, 0, len(resultData.Entries))
	for _, entry := range resultData.Entries {
		if entry == nil || entry.Telemetry == nil || entry.Telemetry.Screenshot == nil {
			continue
		}
		stepIndex := int32(0)
		if entry.StepIndex != nil {
			stepIndex = *entry.StepIndex
		}
		nodeID := ""
		if entry.NodeId != nil {
			nodeID = *entry.NodeId
		}
		screenshot := &basexecution.ExecutionScreenshot{
			Screenshot: entry.Telemetry.Screenshot,
			StepIndex:  stepIndex,
			NodeId:     nodeID,
		}
		if entry.Action != nil && entry.Action.Metadata != nil && entry.Action.Metadata.Label != nil {
			label := strings.TrimSpace(*entry.Action.Metadata.Label)
			if label != "" {
				screenshot.StepLabel = &label
			}
		}
		if entry.Timestamp != nil {
			screenshot.Timestamp = entry.Timestamp
		}
		screenshots = append(screenshots, screenshot)
	}
	sort.Slice(screenshots, func(i, j int) bool {
		if screenshots[i].StepIndex != screenshots[j].StepIndex {
			return screenshots[i].StepIndex < screenshots[j].StepIndex
		}
		return screenshots[i].NodeId < screenshots[j].NodeId
	})
	return screenshots, nil
}

// ExecutionVideoArtifact captures recorded video artifact metadata.
type ExecutionVideoArtifact struct {
	ArtifactID  string         `json:"artifact_id"`
	StorageURL  string         `json:"storage_url,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Label       string         `json:"label,omitempty"`
	SizeBytes   *int64         `json:"size_bytes,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// ExecutionFileArtifact captures trace/HAR artifact metadata.
type ExecutionFileArtifact struct {
	ArtifactID  string         `json:"artifact_id"`
	StorageURL  string         `json:"storage_url,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Label       string         `json:"label,omitempty"`
	SizeBytes   *int64         `json:"size_bytes,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// GetExecutionVideoArtifacts reads recorded video artifacts from the execution result file.
func (s *WorkflowService) GetExecutionVideoArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionVideoArtifact, error) {
	files, err := s.listExecutionArtifacts(ctx, executionID, "videos")
	if err != nil {
		return nil, err
	}
	videos := make([]ExecutionVideoArtifact, 0, len(files))
	for _, file := range files {
		videos = append(videos, ExecutionVideoArtifact(file))
	}
	return videos, nil
}

// GetExecutionTraceArtifacts reads trace artifacts from the execution result file.
func (s *WorkflowService) GetExecutionTraceArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return s.listExecutionArtifacts(ctx, executionID, "traces")
}

// GetExecutionHarArtifacts reads HAR artifacts from the execution result file.
func (s *WorkflowService) GetExecutionHarArtifacts(ctx context.Context, executionID uuid.UUID) ([]ExecutionFileArtifact, error) {
	return s.listExecutionArtifacts(ctx, executionID, "har")
}

func (s *WorkflowService) listExecutionArtifacts(ctx context.Context, executionID uuid.UUID, folder string) ([]ExecutionFileArtifact, error) {
	_ = ctx
	if strings.TrimSpace(s.executionDataRoot) == "" {
		return []ExecutionFileArtifact{}, nil
	}
	base := filepath.Join(s.executionDataRoot, executionID.String(), "artifacts", folder)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExecutionFileArtifact{}, nil
		}
		return nil, fmt.Errorf("read artifacts directory: %w", err)
	}

	items := make([]ExecutionFileArtifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(base, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size := info.Size()
		sizeBytes := &size
		contentType := mime.TypeByExtension(filepath.Ext(name))
		if contentType == "" {
			if data, readErr := os.ReadFile(path); readErr == nil {
				contentType = http.DetectContentType(data)
			}
		}
		descriptor, describeErr := evidence.DescribeFile(folder, contentType, path, evidence.DefaultPolicy())
		if describeErr != nil {
			return nil, fmt.Errorf("describe %s artifact %q: %w", folder, name, describeErr)
		}
		// Public metadata never exposes the execution's local artifact path.
		payload := map[string]any{
			"size_bytes":      size,
			"sha256":          descriptor.SHA256,
			"classification":  descriptor.Classification.String(),
			"retention_class": descriptor.Retention.String(),
			"access_policy":   descriptor.Access.String(),
			"redacted":        descriptor.Redacted,
		}
		items = append(items, ExecutionFileArtifact{
			ArtifactID:  name,
			StorageURL:  artifactURLForPolicy(s, executionID, path, descriptor.Access),
			ContentType: contentType,
			Label:       strings.TrimSuffix(name, filepath.Ext(name)),
			SizeBytes:   sizeBytes,
			Payload:     payload,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ArtifactID < items[j].ArtifactID
	})
	return items, nil
}

func artifactURLForPolicy(s *WorkflowService, executionID uuid.UUID, path string, access basevidence.AccessPolicy) string {
	// INVARIANT: protectedEvidenceHasNoPublicLocation
	// Protected artifacts (currently raw HAR) may be described to operators but
	// must not gain a browser-accessible URL through an execution listing.
	if access == basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY {
		return ""
	}
	return s.assetURLForPath(executionID, path)
}

func (s *WorkflowService) assetURLForPath(executionID uuid.UUID, path string) string {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(s.executionDataRoot) == "" {
		return ""
	}
	base := filepath.Join(s.executionDataRoot, executionID.String())
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	rel = filepath.ToSlash(rel)
	return fmt.Sprintf("/api/v1/recordings/assets/%s/%s", executionID.String(), rel)
}

// UpdateExecutionResultPath updates the DB execution index with the given result file path.
// This is used by the recorder when it first writes the result file.
func (s *WorkflowService) UpdateExecutionResultPath(ctx context.Context, executionID uuid.UUID, resultPath string, updatedAt time.Time) error {
	return s.repo.UpdateExecutionResultPath(ctx, executionID, resultPath, updatedAt)
}

// Ensure our workflow service satisfies the recorder's minimal interface when used as ExecutionIndexRepository.
var _ executionwriter.ExecutionIndexRepository = (*WorkflowService)(nil)

func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return s.repo.GetExecution(ctx, id)
}

func (s *WorkflowService) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	return s.repo.UpdateExecutionStatus(ctx, id, status, errorMessage, completedAt, updatedAt)
}

func (s *WorkflowService) UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error {
	return s.repo.UpdateExecution(ctx, execution)
}
