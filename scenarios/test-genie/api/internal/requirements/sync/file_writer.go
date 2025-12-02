package sync

import (
	"context"
	"encoding/json"
	"time"

	"test-genie/internal/requirements/types"
)

// FileWriter writes requirement files.
type FileWriter struct {
	writer Writer
}

// NewFileWriter creates a new FileWriter.
func NewFileWriter(writer Writer) *FileWriter {
	return &FileWriter{writer: writer}
}

// WriteModule writes a requirement module to disk.
func (w *FileWriter) WriteModule(ctx context.Context, module *types.RequirementModule) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if module == nil || module.FilePath == "" {
		return nil
	}

	// Create a serializable copy
	output := serializableModule{
		Metadata:     serializeMetadata(module.Metadata),
		Imports:      module.Imports,
		Requirements: serializeRequirements(module.Requirements),
	}

	return w.WriteJSON(module.FilePath, output)
}

// WriteJSON writes any value as JSON to a file.
func (w *FileWriter) WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline
	data = append(data, '\n')

	return w.writer.WriteFile(path, data, 0644)
}

// serializableModule is the output format for requirement files.
type serializableModule struct {
	Metadata     map[string]any          `json:"_metadata,omitempty"`
	Imports      []string                `json:"imports,omitempty"`
	Requirements []serializableRequirement `json:"requirements"`
}

// serializableRequirement is the output format for requirements.
type serializableRequirement struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title"`
	Status      string                   `json:"status,omitempty"`
	PRDRef      string                   `json:"prd_ref,omitempty"`
	Category    string                   `json:"category,omitempty"`
	Criticality string                   `json:"criticality,omitempty"`
	Description string                   `json:"description,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
	Children    []string                 `json:"children,omitempty"`
	DependsOn   []string                 `json:"depends_on,omitempty"`
	Blocks      []string                 `json:"blocks,omitempty"`
	Validation  []serializableValidation `json:"validation,omitempty"`
}

// serializableValidation is the output format for validations.
type serializableValidation struct {
	Type       string         `json:"type"`
	Ref        string         `json:"ref,omitempty"`
	WorkflowID string         `json:"workflow_id,omitempty"`
	Phase      string         `json:"phase,omitempty"`
	Status     string         `json:"status"`
	Notes      string         `json:"notes,omitempty"`
	Scenario   string         `json:"scenario,omitempty"`
	Folder     string         `json:"folder,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// serializeMetadata converts ModuleMetadata to a map for JSON output.
func serializeMetadata(meta types.ModuleMetadata) map[string]any {
	result := make(map[string]any)

	if meta.Module != "" {
		result["module"] = meta.Module
	}
	if meta.ModuleName != "" {
		result["module_name"] = meta.ModuleName
	}
	if meta.Description != "" {
		result["description"] = meta.Description
	}
	if meta.PRDRef != "" {
		result["prd_ref"] = meta.PRDRef
	}
	if meta.Priority != "" {
		result["priority"] = meta.Priority
	}
	if meta.SchemaVersion != "" {
		result["schema_version"] = meta.SchemaVersion
	}
	if meta.AutoSyncEnabled != nil {
		result["auto_sync_enabled"] = *meta.AutoSyncEnabled
	}
	// Update last_validated_at
	result["last_validated_at"] = time.Now().Format(time.RFC3339)

	if len(result) == 0 {
		return nil
	}

	return result
}

// serializeRequirements converts requirements to serializable format.
func serializeRequirements(reqs []types.Requirement) []serializableRequirement {
	result := make([]serializableRequirement, len(reqs))

	for i, req := range reqs {
		result[i] = serializableRequirement{
			ID:          req.ID,
			Title:       req.Title,
			Status:      string(req.Status),
			PRDRef:      req.PRDRef,
			Category:    req.Category,
			Criticality: string(req.Criticality),
			Description: req.Description,
			Tags:        emptyToNil(req.Tags),
			Children:    emptyToNil(req.Children),
			DependsOn:   emptyToNil(req.DependsOn),
			Blocks:      emptyToNil(req.Blocks),
			Validation:  serializeValidations(req.Validations),
		}
	}

	return result
}

// serializeValidations converts validations to serializable format.
func serializeValidations(vals []types.Validation) []serializableValidation {
	if len(vals) == 0 {
		return nil
	}

	result := make([]serializableValidation, len(vals))

	for i, val := range vals {
		result[i] = serializableValidation{
			Type:       string(val.Type),
			Ref:        val.Ref,
			WorkflowID: val.WorkflowID,
			Phase:      val.Phase,
			Status:     string(val.Status),
			Notes:      val.Notes,
			Scenario:   val.Scenario,
			Folder:     val.Folder,
			Metadata:   emptyMapToNil(val.Metadata),
		}
	}

	return result
}

// emptyToNil converts empty slices to nil for cleaner JSON output.
func emptyToNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// emptyMapToNil converts empty maps to nil for cleaner JSON output.
func emptyMapToNil(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	return m
}

// WriteJSONL writes records as newline-delimited JSON.
func (w *FileWriter) WriteJSONL(path string, records []any) error {
	var data []byte

	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			continue // Skip invalid records
		}
		data = append(data, line...)
		data = append(data, '\n')
	}

	return w.writer.WriteFile(path, data, 0644)
}

// WriteSyncSnapshot writes a snapshot of the current sync state.
func (w *FileWriter) WriteSyncSnapshot(path string, index *types.RequirementModule) error {
	return w.WriteJSON(path, index)
}
