package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const workflowVersionsDirName = ".bas/versions"

func workflowVersionsDir(projectFolderPath string, workflowID uuid.UUID) string {
	return filepath.Join(projectFolderPath, workflowVersionsDirName, workflowID.String())
}

func workflowVersionFilePath(projectFolderPath string, workflowID uuid.UUID, version int32) string {
	return filepath.Join(workflowVersionsDir(projectFolderPath, workflowID), fmt.Sprintf("%d.json", version))
}

func persistWorkflowVersionSnapshot(project *database.ProjectIndex, wf *basapi.WorkflowSummary) error {
	if project == nil {
		return errors.New("project is nil")
	}
	if wf == nil {
		return errors.New("workflow is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(wf.Id))
	if err != nil {
		return fmt.Errorf("invalid workflow id: %w", err)
	}
	if wf.FlowDefinition == nil {
		return fmt.Errorf("missing flow_definition")
	}
	if wf.Version <= 0 {
		return fmt.Errorf("invalid workflow version")
	}

	dir := workflowVersionsDir(project.FolderPath, workflowID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure versions dir: %w", err)
	}

	versionMsg := &basapi.WorkflowVersion{
		WorkflowId:        wf.Id,
		Version:           wf.Version,
		FlowDefinition:    wf.FlowDefinition,
		ChangeDescription: wf.LastChangeDescription,
		CreatedBy:         wf.CreatedBy,
		CreatedAt:         wf.UpdatedAt,
	}
	if versionMsg.CreatedAt == nil {
		versionMsg.CreatedAt = timestamppb.New(time.Now().UTC())
	}

	target := workflowVersionFilePath(project.FolderPath, workflowID, wf.Version)
	// Don't clobber an existing snapshot for the same version.
	if _, err := os.Stat(target); err == nil {
		return nil
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(versionMsg)
	if err != nil {
		return fmt.Errorf("marshal workflow version: %w", err)
	}
	indented := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err == nil {
		indented = buf.Bytes()
	}

	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, indented, 0o644); err != nil {
		return fmt.Errorf("write workflow version tmp: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("finalize workflow version: %w", err)
	}
	return nil
}

func listWorkflowVersions(ctx context.Context, project *database.ProjectIndex, workflowID uuid.UUID) ([]*basapi.WorkflowVersion, error) {
	_ = ctx
	if project == nil {
		return nil, errors.New("project is nil")
	}
	dir := workflowVersionsDir(project.FolderPath, workflowID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*basapi.WorkflowVersion{}, nil
		}
		return nil, fmt.Errorf("read versions dir: %w", err)
	}

	type fileItem struct {
		version int32
		path    string
	}
	items := make([]fileItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		v, err := strconv.Atoi(base)
		if err != nil || v <= 0 {
			continue
		}
		items = append(items, fileItem{version: int32(v), path: filepath.Join(dir, name)})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].version > items[j].version })
	out := make([]*basapi.WorkflowVersion, 0, len(items))

	for _, item := range items {
		raw, err := os.ReadFile(item.path)
		if err != nil {
			continue
		}
		var pb basapi.WorkflowVersion
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &pb); err != nil {
			continue
		}
		out = append(out, &pb)
	}

	return out, nil
}

func getWorkflowVersion(ctx context.Context, project *database.ProjectIndex, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error) {
	_ = ctx
	if project == nil {
		return nil, errors.New("project is nil")
	}
	if version <= 0 {
		return nil, fmt.Errorf("invalid version")
	}
	path := workflowVersionFilePath(project.FolderPath, workflowID, version)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWorkflowVersionNotFound
		}
		return nil, err
	}
	var pb basapi.WorkflowVersion
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &pb); err != nil {
		return nil, fmt.Errorf("parse workflow version: %w", err)
	}
	return &pb, nil
}

