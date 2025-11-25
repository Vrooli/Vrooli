package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vrooli/browser-automation-studio/database"
)

// workflowFileSnapshot captures the parsed state of a workflow file prior to reconciling it with the database.
type workflowFileSnapshot struct {
	Workflow         *database.Workflow
	Nodes            []any
	Edges            []any
	RelativePath     string
	AbsolutePath     string
	RawJSON          []byte
	FileVersion      int
	ChangeDesc       string
	SourceLabel      string
	NeedsWriteBack   bool
	FileHash         string
	LastModifiedTime time.Time
}

func (s *WorkflowService) readWorkflowFile(ctx context.Context, project *database.Project, absPath string) (*workflowFileSnapshot, error) {
	rel, err := filepath.Rel(s.projectWorkflowsDir(project), absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine relative workflow path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat workflow file: %w", err)
	}
	rawBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	// Allow empty files to be treated as a blank workflow shell.
	if len(rawBytes) == 0 {
		rawBytes = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(rawBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON in %s: %w", rel, err)
	}

	idValue := anyToString(payload["id"])
	var workflowID uuid.UUID
	var generatedID bool
	if idValue == "" {
		workflowID = uuid.New()
		generatedID = true
	} else {
		parsed, parseErr := uuid.Parse(idValue)
		if parseErr != nil {
			return nil, fmt.Errorf("workflow file %s has invalid id: %w", rel, parseErr)
		}
		workflowID = parsed
	}

	name := strings.TrimSpace(anyToString(payload["name"]))
	if name == "" {
		base := filepath.Base(absPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	folderPath := normalizeFolderPath(anyToString(payload["folder_path"]))
	if folderPath == defaultWorkflowFolder {
		// If the file lives in a nested directory and did not specify folder_path, derive it from the path.
		dir := filepath.Dir(rel)
		if dir != "." {
			folderPath = normalizeFolderPath(dir)
		}
	}

	description := strings.TrimSpace(anyToString(payload["description"]))
	tags := pq.StringArray(stringSliceFromAny(payload["tags"]))
	changeDesc := strings.TrimSpace(anyToString(payload["change_description"]))
	sourceLabel := strings.TrimSpace(anyToString(payload["source"]))
	if changeDesc == "" {
		changeDesc = fileSyncChangeDesc
	}
	if sourceLabel == "" {
		sourceLabel = fileSourceFileSync
	}

	versionHint := parseFlexibleInt(payload["version"])

	var flowCandidate map[string]any
	if rawFlow, ok := payload["flow_definition"].(map[string]any); ok {
		flowCandidate = rawFlow
	} else {
		flowCandidate = make(map[string]any)
		if nodes, ok := payload["nodes"]; ok {
			flowCandidate["nodes"] = nodes
		}
		if edges, ok := payload["edges"]; ok {
			flowCandidate["edges"] = edges
		}
	}

	// Ensure we always have a nodes/edges tuple even if empty.
	if _, hasNodes := flowCandidate["nodes"]; !hasNodes {
		flowCandidate["nodes"] = []any{}
	}
	if _, hasEdges := flowCandidate["edges"]; !hasEdges {
		flowCandidate["edges"] = []any{}
	}

	normalized, normErr := normalizeFlowDefinition(flowCandidate)
	if normErr != nil {
		return nil, fmt.Errorf("workflow file %s has invalid flow definition: %w", rel, normErr)
	}

	nodes := toInterfaceSlice(normalized["nodes"])
	edges := toInterfaceSlice(normalized["edges"])

	workflow := &database.Workflow{
		ID:                    workflowID,
		ProjectID:             &project.ID,
		Name:                  name,
		FolderPath:            folderPath,
		Description:           description,
		Tags:                  tags,
		FlowDefinition:        normalized,
		Version:               versionHint,
		CreatedBy:             strings.TrimSpace(anyToString(payload["created_by"])),
		LastChangeDescription: changeDesc,
		LastChangeSource:      sourceLabel,
	}

	snapshot := &workflowFileSnapshot{
		Workflow:         workflow,
		Nodes:            nodes,
		Edges:            edges,
		RelativePath:     filepath.ToSlash(rel),
		AbsolutePath:     absPath,
		RawJSON:          rawBytes,
		FileVersion:      versionHint,
		ChangeDesc:       changeDesc,
		SourceLabel:      sourceLabel,
		NeedsWriteBack:   generatedID,
		FileHash:         hashWorkflowDefinition(normalized),
		LastModifiedTime: info.ModTime(),
	}

	return snapshot, nil
}
