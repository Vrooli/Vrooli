package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

type workflowProtoSnapshot struct {
	Workflow     *basapi.WorkflowSummary
	AbsolutePath string
	RelativePath string
	NeedsWrite   bool
}

func ReadWorkflowSummaryFile(ctx context.Context, project *database.ProjectIndex, absPath string) (*workflowProtoSnapshot, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	_ = ctx

	rel, err := filepath.Rel(ProjectWorkflowsDir(project), absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine relative workflow path: %w", err)
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}

	// Preferred: protojson WorkflowSummary.
	var pb basapi.WorkflowSummary
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &pb); err == nil && pb.Id != "" {
		return &workflowProtoSnapshot{
			Workflow:     &pb,
			AbsolutePath: absPath,
			RelativePath: filepath.ToSlash(rel),
		}, nil
	}

	// Fallback: parse flexible JSON and convert to proto (handles non-protojson files).
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON in %s: %w", rel, err)
	}

	converted, needsWrite, err := flexibleWorkflowPayloadToProto(project, payload)
	if err != nil {
		return nil, fmt.Errorf("workflow file %s: %w", rel, err)
	}

	return &workflowProtoSnapshot{
		Workflow:     converted,
		AbsolutePath: absPath,
		RelativePath: filepath.ToSlash(rel),
		NeedsWrite:   needsWrite,
	}, nil
}

// flexibleWorkflowPayloadToProto converts a loose JSON workflow map to proto format.
// This handles workflow files that aren't strict protojson (e.g., missing IDs, different casing).
func flexibleWorkflowPayloadToProto(project *database.ProjectIndex, payload map[string]any) (*basapi.WorkflowSummary, bool, error) {
	if project == nil {
		return nil, false, errors.New("project is nil")
	}
	if payload == nil {
		return nil, false, errors.New("payload is nil")
	}

	idStr, _ := payload["id"].(string)
	var id uuid.UUID
	needsWrite := false
	if strings.TrimSpace(idStr) == "" {
		id = uuid.New()
		needsWrite = true
	} else {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			return nil, false, fmt.Errorf("invalid workflow id: %w", err)
		}
		id = parsed
	}

	name := strings.TrimSpace(anyToString(payload["name"]))
	if name == "" {
		name = "workflow"
	}
	folderPath := normalizeFolderPath(anyToString(payload["folder_path"]))
	description := strings.TrimSpace(anyToString(payload["description"]))
	tags := stringSliceFromAny(payload["tags"])
	version := int32(parseFlexibleInt(payload["version"]))
	if version <= 0 {
		version = 1
	}

	// Parse definition from definition_v2 or flow_definition field.
	// Use buildFlowDefinition which normalizes subflow args and other proto-incompatible values.
	var def *basworkflows.WorkflowDefinitionV2
	var defErr error
	if v2Raw, ok := payload["definition_v2"].(map[string]any); ok {
		def, defErr = marshalToFlowDefinition(v2Raw)
	} else if flowDef, ok := payload["flow_definition"].(map[string]any); ok {
		def, defErr = marshalToFlowDefinition(flowDef)
	} else {
		// Try to build from top-level nodes/edges (fallback for legacy format)
		def, defErr = buildFlowDefinition(payload)
	}
	if defErr != nil {
		return nil, false, fmt.Errorf("build flow definition: %w", defErr)
	}

	if def == nil {
		def = &basworkflows.WorkflowDefinitionV2{}
		needsWrite = true
	}

	now := autocontracts.NowTimestamp()
	createdAt := now
	updatedAt := now
	if ts, ok := payload["created_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			createdAt = autocontracts.TimeToTimestamp(parsed)
		}
	}
	if ts, ok := payload["updated_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updatedAt = autocontracts.TimeToTimestamp(parsed)
		}
	}

	return &basapi.WorkflowSummary{
		Id:             id.String(),
		ProjectId:      project.ID.String(),
		Name:           name,
		FolderPath:     folderPath,
		Description:    description,
		Tags:           tags,
		Version:        version,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		FlowDefinition: def,
	}, needsWrite, nil
}

// WriteWorkflowSummaryFile persists a WorkflowSummary as a .workflow.json
// file under the project root. preferredRel, when non-empty, is treated
// as a project-root-relative path (matching WorkflowIndex.FilePath and
// the relPath produced by sync). Returning to ProjectWorkflowsDir-relative
// here causes path doubling on every round-trip (workflows/workflows/...
// recursion), so the two relativity spaces are kept aligned in this
// function and in desiredWorkflowSummaryFilePath.
func WriteWorkflowSummaryFile(project *database.ProjectIndex, wf *basapi.WorkflowSummary, preferredRel string) (absPath string, relPath string, err error) {
	if project == nil {
		return "", "", errors.New("project is nil")
	}
	if wf == nil {
		return "", "", errors.New("workflow is nil")
	}

	abs, rel := desiredWorkflowSummaryFilePath(project, wf)
	targetAbs := abs
	targetRel := rel
	if trimmed := strings.TrimSpace(preferredRel); trimmed != "" {
		// preferredRel is project-root-relative; honor it verbatim.
		cleanRel := filepath.ToSlash(filepath.Clean(trimmed))
		targetAbs = filepath.Join(project.FolderPath, filepath.FromSlash(cleanRel))
		targetRel = cleanRel
	}

	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create workflow directory: %w", err)
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(wf)
	if err != nil {
		return "", "", fmt.Errorf("marshal workflow summary: %w", err)
	}

	indented := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err == nil {
		indented = buf.Bytes()
	}

	tmp := targetAbs + ".tmp"
	if err := os.WriteFile(tmp, indented, 0o644); err != nil {
		return "", "", fmt.Errorf("failed to write workflow temp file: %w", err)
	}
	if err := os.Rename(tmp, targetAbs); err != nil {
		return "", "", fmt.Errorf("failed to finalize workflow file write: %w", err)
	}
	return targetAbs, filepath.ToSlash(targetRel), nil
}

// desiredWorkflowSummaryFilePath returns the canonical absolute path and
// project-root-relative path for a workflow summary. The relative path
// always includes the leading "workflows/" segment so it round-trips
// cleanly through WorkflowIndex.FilePath, which is project-root-relative
// per sync.go's filepath.Rel(projectRoot, path) contract.
func desiredWorkflowSummaryFilePath(project *database.ProjectIndex, wf *basapi.WorkflowSummary) (absPath string, relPath string) {
	folder := normalizeFolderPath(wf.FolderPath)
	subdir := workflowsSubdir(folder)
	slug := sanitizeWorkflowSlug(wf.Name)
	id, _ := uuid.Parse(wf.Id)
	fileName := fmt.Sprintf("%s--%s%s", slug, shortID(id), workflowFileExt)
	baseDir := ProjectWorkflowsDir(project)
	rootRelDir := workflowDirectoryName // "workflows"
	if subdir != "" {
		return filepath.Join(baseDir, subdir, fileName), filepath.ToSlash(filepath.Join(rootRelDir, subdir, fileName))
	}
	return filepath.Join(baseDir, fileName), filepath.ToSlash(filepath.Join(rootRelDir, fileName))
}
