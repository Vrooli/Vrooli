package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// SchemaVersion tracks whether this workflow was stored in V1 or V2 format.
	// V1 = legacy React Flow nodes/edges, V2 = unified proto-based definition.
	SchemaVersion SchemaVersion
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
	tags := database.StringArray(stringSliceFromAny(payload["tags"]))
	changeDesc := strings.TrimSpace(anyToString(payload["change_description"]))
	sourceLabel := strings.TrimSpace(anyToString(payload["source"]))
	if changeDesc == "" {
		changeDesc = fileSyncChangeDesc
	}
	if sourceLabel == "" {
		sourceLabel = fileSourceFileSync
	}

	versionHint := parseFlexibleInt(payload["version"])

	// Detect schema version (V1 or V2)
	schemaVersion := DetectSchemaVersion(payload)

	var normalized database.JSONMap
	var nodes, edges []any

	if schemaVersion == SchemaV2 {
		// V2 format: parse definition_v2 and convert to V1 for internal use
		v2Def := ParseV2Definition(payload)
		if v2Def != nil {
			v1Nodes, v1Edges, convErr := V2ToV1Definition(v2Def)
			if convErr != nil {
				return nil, fmt.Errorf("workflow file %s: failed to convert V2 definition: %w", rel, convErr)
			}
			nodes = v1Nodes
			edges = v1Edges
			normalized = database.JSONMap{
				"nodes": nodes,
				"edges": edges,
			}
		} else {
			// Fallback to V1 if V2 parsing failed
			schemaVersion = SchemaV1
		}
	}

	if schemaVersion == SchemaV1 {
		// V1 format: use legacy nodes/edges format
		var flowCandidate map[string]any
		if rawFlow, ok := payload["flow_definition"].(map[string]any); ok {
			flowCandidate = rawFlow
		} else {
			flowCandidate = make(map[string]any)
			if n, ok := payload["nodes"]; ok {
				flowCandidate["nodes"] = n
			}
			if e, ok := payload["edges"]; ok {
				flowCandidate["edges"] = e
			}
		}

		// Ensure we always have a nodes/edges tuple even if empty.
		if _, hasNodes := flowCandidate["nodes"]; !hasNodes {
			flowCandidate["nodes"] = []any{}
		}
		if _, hasEdges := flowCandidate["edges"]; !hasEdges {
			flowCandidate["edges"] = []any{}
		}

		var normErr error
		normalized, normErr = normalizeFlowDefinition(flowCandidate)
		if normErr != nil {
			return nil, fmt.Errorf("workflow file %s has invalid flow definition: %w", rel, normErr)
		}

		nodes = ToInterfaceSlice(normalized["nodes"])
		edges = ToInterfaceSlice(normalized["edges"])
	}

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
		SchemaVersion:    schemaVersion,
	}

	return snapshot, nil
}
