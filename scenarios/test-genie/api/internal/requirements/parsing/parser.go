// Package parsing parses requirement JSON files into domain objects.
package parsing

import (
	"context"
	"encoding/json"
	"path/filepath"

	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/types"
)

// Reader abstracts file reading for parsing operations.
type Reader interface {
	ReadFile(path string) ([]byte, error)
}

// Parser parses requirement JSON files.
type Parser interface {
	// Parse reads and normalizes a single requirement file.
	Parse(ctx context.Context, filePath string) (*types.RequirementModule, error)

	// ParseAll parses multiple files and builds an index.
	ParseAll(ctx context.Context, files []discovery.DiscoveredFile) (*ModuleIndex, error)
}

// parser implements Parser using file system operations.
type parser struct {
	reader     Reader
	normalizer *Normalizer
}

// New creates a Parser with the provided Reader.
func New(reader Reader) Parser {
	return &parser{
		reader:     reader,
		normalizer: NewNormalizer(),
	}
}

// NewDefault creates a Parser using the real file system.
func NewDefault() Parser {
	return &parser{
		reader:     &osReader{},
		normalizer: NewNormalizer(),
	}
}

// osReader implements Reader using the os package.
type osReader struct{}

func (r *osReader) ReadFile(path string) ([]byte, error) {
	return readFileFromOS(path)
}

// readFileFromOS reads a file from the OS filesystem.
// This is a separate function to avoid import issues.
func readFileFromOS(path string) ([]byte, error) {
	return osReadFile(path)
}

// Parse reads and normalizes a single requirement file.
func (p *parser) Parse(ctx context.Context, filePath string) (*types.RequirementModule, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := p.reader.ReadFile(filePath)
	if err != nil {
		return nil, types.NewParseError(filePath, err)
	}

	module, err := p.parseJSON(data)
	if err != nil {
		return nil, types.NewParseError(filePath, err)
	}

	module.FilePath = filePath
	isIndex := filepath.Base(filePath) == "index.json"
	module.IsIndex = isIndex

	// Set source file on all requirements
	for i := range module.Requirements {
		module.Requirements[i].SourceFile = filePath
		module.Requirements[i].SourceModule = module.EffectiveName()
	}

	// Normalize the module
	p.normalizer.NormalizeModule(module)

	return module, nil
}

// parseJSON unmarshals JSON data into a RequirementModule.
func (p *parser) parseJSON(data []byte) (*types.RequirementModule, error) {
	// First try parsing with standard structure
	var module types.RequirementModule
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, types.ErrInvalidJSON
	}

	return &module, nil
}

// ParseAll parses multiple files and builds a ModuleIndex.
func (p *parser) ParseAll(ctx context.Context, files []discovery.DiscoveredFile) (*ModuleIndex, error) {
	index := NewModuleIndex()

	for _, file := range files {
		select {
		case <-ctx.Done():
			return index, ctx.Err()
		default:
		}

		module, err := p.Parse(ctx, file.AbsolutePath)
		if err != nil {
			// Record error but continue parsing other files
			index.Errors = append(index.Errors, err)
			continue
		}

		module.RelativePath = file.RelativePath
		module.ModuleName = file.ModuleDir

		if err := index.AddModule(module); err != nil {
			index.Errors = append(index.Errors, err)
		}
	}

	// Build parent-child relationships
	index.BuildHierarchy()

	return index, nil
}

// rawModule is used for flexible JSON parsing.
type rawModule struct {
	Metadata     json.RawMessage   `json:"_metadata,omitempty"`
	Imports      []string          `json:"imports,omitempty"`
	Requirements []json.RawMessage `json:"requirements"`
}

// rawRequirement supports both "validation" and "validations" fields.
type rawRequirement struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Status      string          `json:"status"`
	PRDRef      string          `json:"prd_ref,omitempty"`
	Category    string          `json:"category,omitempty"`
	Criticality string          `json:"criticality,omitempty"`
	Description string          `json:"description,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Children    []string        `json:"children,omitempty"`
	DependsOn   []string        `json:"depends_on,omitempty"`
	Blocks      []string        `json:"blocks,omitempty"`
	Validation  []rawValidation `json:"validation,omitempty"`
	Validations []rawValidation `json:"validations,omitempty"`
}

// rawValidation supports flexible validation parsing.
type rawValidation struct {
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

// ParseFlexible parses JSON with support for legacy field names.
func ParseFlexible(data []byte) (*types.RequirementModule, error) {
	var raw rawModule
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, types.ErrInvalidJSON
	}

	module := &types.RequirementModule{
		Imports: raw.Imports,
	}

	// Parse metadata if present
	if len(raw.Metadata) > 0 {
		if err := json.Unmarshal(raw.Metadata, &module.Metadata); err != nil {
			// Ignore metadata parse errors
		}
	}

	// Parse requirements
	for _, reqData := range raw.Requirements {
		var rawReq rawRequirement
		if err := json.Unmarshal(reqData, &rawReq); err != nil {
			continue // Skip invalid requirements
		}

		req := types.Requirement{
			ID:          rawReq.ID,
			Title:       rawReq.Title,
			Status:      types.NormalizeDeclaredStatus(rawReq.Status),
			PRDRef:      rawReq.PRDRef,
			Category:    rawReq.Category,
			Criticality: types.NormalizeCriticality(rawReq.Criticality),
			Description: rawReq.Description,
			Tags:        rawReq.Tags,
			Children:    rawReq.Children,
			DependsOn:   rawReq.DependsOn,
			Blocks:      rawReq.Blocks,
		}

		// Support both "validation" and "validations" fields
		rawValidations := rawReq.Validation
		if len(rawValidations) == 0 {
			rawValidations = rawReq.Validations
		}

		for _, rawVal := range rawValidations {
			val := types.Validation{
				Type:       types.NormalizeValidationType(rawVal.Type),
				Ref:        rawVal.Ref,
				WorkflowID: rawVal.WorkflowID,
				Phase:      rawVal.Phase,
				Status:     types.NormalizeValidationStatus(rawVal.Status),
				Notes:      rawVal.Notes,
				Scenario:   rawVal.Scenario,
				Folder:     rawVal.Folder,
				Metadata:   rawVal.Metadata,
			}
			req.Validations = append(req.Validations, val)
		}

		module.Requirements = append(module.Requirements, req)
	}

	return module, nil
}
