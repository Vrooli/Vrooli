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
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	rel, err := filepath.Rel(projectWorkflowsDir(project), absPath)
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

	// Legacy fallback: parse flexible JSON and convert to proto.
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON in %s: %w", rel, err)
	}

	converted, needsWrite, err := legacyWorkflowPayloadToProto(project, payload)
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

func legacyWorkflowPayloadToProto(project *database.ProjectIndex, payload map[string]any) (*basapi.WorkflowSummary, bool, error) {
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

	// Parse V2 definition - V1 format is no longer supported for disk storage.
	var def *basworkflows.WorkflowDefinitionV2
	if v2Raw, ok := payload["definition_v2"].(map[string]any); ok {
		v2Bytes, _ := json.Marshal(v2Raw)
		var v2 basworkflows.WorkflowDefinitionV2
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(v2Bytes, &v2); err == nil {
			def = &v2
		}
	}

	if def == nil {
		def = &basworkflows.WorkflowDefinitionV2{}
		needsWrite = true
	}

	now := timestamppb.New(time.Now().UTC())
	createdAt := now
	updatedAt := now
	if ts, ok := payload["created_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			createdAt = timestamppb.New(parsed)
		}
	}
	if ts, ok := payload["updated_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updatedAt = timestamppb.New(parsed)
		}
	}

	return &basapi.WorkflowSummary{
		Id:          id.String(),
		ProjectId:   project.ID.String(),
		Name:        name,
		FolderPath:  folderPath,
		Description: description,
		Tags:        tags,
		Version:     version,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		FlowDefinition: def,
	}, needsWrite, nil
}

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
	if strings.TrimSpace(preferredRel) != "" {
		targetAbs = filepath.Join(projectWorkflowsDir(project), filepath.FromSlash(preferredRel))
		targetRel = preferredRel
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

func desiredWorkflowSummaryFilePath(project *database.ProjectIndex, wf *basapi.WorkflowSummary) (absPath string, relPath string) {
	folder := normalizeFolderPath(wf.FolderPath)
	subdir := workflowsSubdir(folder)
	slug := sanitizeWorkflowSlug(wf.Name)
	id, _ := uuid.Parse(wf.Id)
	fileName := fmt.Sprintf("%s--%s%s", slug, shortID(id), workflowFileExt)
	baseDir := projectWorkflowsDir(project)
	if subdir != "" {
		return filepath.Join(baseDir, subdir, fileName), filepath.Join(subdir, fileName)
	}
	return filepath.Join(baseDir, fileName), fileName
}
