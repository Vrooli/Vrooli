// Package content provides content validation for scenario structure.
package content

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ConfigFile represents a .vrooli config file to validate.
type ConfigFile struct {
	// Name is the filename (e.g., "service.json").
	Name string
	// Required indicates if the file must exist.
	Required bool
	// SchemaFile is the schema filename (e.g., "service.schema.json").
	SchemaFile string
	// DocPath is the relative path to documentation for this config.
	DocPath string
}

// DefaultConfigFiles returns the list of .vrooli config files to validate.
func DefaultConfigFiles() []ConfigFile {
	return []ConfigFile{
		{
			Name:       "service.json",
			Required:   true,
			SchemaFile: "service.schema.json",
			DocPath:    "docs/scenarios/service-config.md",
		},
		{
			Name:       "testing.json",
			Required:   false,
			SchemaFile: "testing.schema.json",
			DocPath:    "docs/scenarios/testing-config.md",
		},
		{
			Name:       "endpoints.json",
			Required:   false,
			SchemaFile: "endpoints.schema.json",
			DocPath:    "docs/scenarios/endpoints-config.md",
		},
		{
			Name:       "lighthouse.json",
			Required:   false,
			SchemaFile: "lighthouse.schema.json",
			DocPath:    "docs/scenarios/lighthouse-config.md",
		},
	}
}

// SchemaError represents a schema validation error with context.
type SchemaError struct {
	File       string   // Config file path
	SchemaPath string   // Schema file path
	DocPath    string   // Documentation path
	Errors     []string // Validation error messages
}

func (e *SchemaError) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("❌ %s failed schema validation\n", e.File))
	for _, err := range e.Errors {
		b.WriteString(fmt.Sprintf("   • %s\n", err))
	}
	b.WriteString(fmt.Sprintf("\n   📋 Schema: %s\n", e.SchemaPath))
	if e.DocPath != "" {
		b.WriteString(fmt.Sprintf("   📖 Docs: %s\n", e.DocPath))
	}
	return b.String()
}

// SchemaValidatorInterface defines the interface for schema validation.
type SchemaValidatorInterface interface {
	// Validate checks all .vrooli config files against their schemas.
	Validate() Result
}

// SchemaValidator validates .vrooli config files against JSON schemas.
type SchemaValidator struct {
	scenarioDir string
	schemasDir  string
	configFiles []ConfigFile
	compiler    *jsonschema.Compiler
	logWriter   io.Writer
}

// NewSchemaValidator creates a new schema validator.
// scenarioDir is the path to the scenario being validated.
// schemasDir is the path to the directory containing schema files.
func NewSchemaValidator(scenarioDir, schemasDir string, logWriter io.Writer) *SchemaValidator {
	return &SchemaValidator{
		scenarioDir: scenarioDir,
		schemasDir:  schemasDir,
		configFiles: DefaultConfigFiles(),
		compiler:    jsonschema.NewCompiler(),
		logWriter:   logWriter,
	}
}

// WithConfigFiles overrides the default config files to validate.
func (v *SchemaValidator) WithConfigFiles(files []ConfigFile) *SchemaValidator {
	v.configFiles = files
	return v
}

// SchemaValidateResult contains the result of schema validation for a single file.
type SchemaValidateResult struct {
	// File is the config file that was validated.
	File string
	// Valid indicates if the file passed validation.
	Valid bool
	// Skipped indicates if the file was skipped (not found and not required).
	Skipped bool
	// Error contains validation error details if Valid is false.
	Error *SchemaError
}

// Validate implements SchemaValidatorInterface.
// It validates all .vrooli config files against their schemas and returns a Result.
func (v *SchemaValidator) Validate() Result {
	vrooliDir := filepath.Join(v.scenarioDir, ".vrooli")
	var observations []Observation
	var validCount int
	var allErrors []string

	for _, cf := range v.configFiles {
		configPath := filepath.Join(vrooliDir, cf.Name)
		schemaPath := filepath.Join(v.schemasDir, cf.SchemaFile)

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if cf.Required {
				errMsg := fmt.Sprintf("%s: required file not found", cf.Name)
				allErrors = append(allErrors, errMsg)
				observations = append(observations, NewErrorObservation(errMsg))
				logError(v.logWriter, "%s", errMsg)
			} else {
				observations = append(observations, NewSkipObservation(
					fmt.Sprintf("%s (optional, not found)", cf.Name),
				))
				logInfo(v.logWriter, "%s: optional file not found, skipping", cf.Name)
			}
			continue
		}

		// Check if schema file exists
		if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
			return FailSystem(
				fmt.Errorf("schema file not found: %s", schemaPath),
				"Ensure test-genie schemas directory is properly installed.",
			)
		}

		// Validate the config file
		if err := v.validateFile(configPath, schemaPath); err != nil {
			schemaErr := &SchemaError{
				File:       configPath,
				SchemaPath: schemaPath,
				DocPath:    cf.DocPath,
				Errors:     extractValidationErrors(err),
			}
			allErrors = append(allErrors, schemaErr.Error())
			observations = append(observations, NewErrorObservation(schemaErr.Error()))
			logError(v.logWriter, "%s failed schema validation", cf.Name)
		} else {
			validCount++
			observations = append(observations, NewSuccessObservation(
				fmt.Sprintf("%s: valid", cf.Name),
			))
			logSuccess(v.logWriter, "%s: schema valid", cf.Name)
		}
	}

	if len(allErrors) > 0 {
		return FailMisconfiguration(
			fmt.Errorf("schema validation failed for %d file(s)", len(allErrors)),
			"Fix the configuration files according to their schemas.",
		).WithObservations(observations...)
	}

	result := OKWithCount(validCount)
	result.Observations = observations
	return result
}

// ValidateDetailed validates all config files and returns detailed results.
func (v *SchemaValidator) ValidateDetailed() ([]SchemaValidateResult, error) {
	vrooliDir := filepath.Join(v.scenarioDir, ".vrooli")
	results := make([]SchemaValidateResult, 0, len(v.configFiles))

	for _, cf := range v.configFiles {
		configPath := filepath.Join(vrooliDir, cf.Name)
		schemaPath := filepath.Join(v.schemasDir, cf.SchemaFile)

		result := SchemaValidateResult{
			File: configPath,
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if cf.Required {
				result.Error = &SchemaError{
					File:       configPath,
					SchemaPath: schemaPath,
					DocPath:    cf.DocPath,
					Errors:     []string{"required file not found"},
				}
			} else {
				result.Skipped = true
				result.Valid = true
			}
			results = append(results, result)
			continue
		}

		// Check if schema file exists
		if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("schema file not found: %s", schemaPath)
		}

		// Validate the config file
		if err := v.validateFile(configPath, schemaPath); err != nil {
			result.Error = &SchemaError{
				File:       configPath,
				SchemaPath: schemaPath,
				DocPath:    cf.DocPath,
				Errors:     extractValidationErrors(err),
			}
		} else {
			result.Valid = true
		}

		results = append(results, result)
	}

	return results, nil
}

// validateFile validates a single config file against a schema.
func (v *SchemaValidator) validateFile(configPath, schemaPath string) error {
	// Read and compile the schema
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Create a fresh compiler for each validation to avoid caching issues
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	schemaID := "file://" + schemaPath
	if err := compiler.AddResource(schemaID, strings.NewReader(string(schemaData))); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Read the config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Parse as JSON first to catch syntax errors
	var doc interface{}
	if err := json.Unmarshal(configData, &doc); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate against schema
	return schema.Validate(doc)
}

// extractValidationErrors extracts human-readable error messages from a validation error.
func extractValidationErrors(err error) []string {
	if err == nil {
		return nil
	}

	// Try to extract detailed validation errors
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		return flattenValidationErrors(ve, "")
	}

	// Fall back to the error message
	return []string{err.Error()}
}

// flattenValidationErrors recursively extracts error messages from nested validation errors.
func flattenValidationErrors(ve *jsonschema.ValidationError, prefix string) []string {
	var errors []string

	// Build the JSON path
	path := prefix
	if ve.InstanceLocation != "" {
		if path != "" {
			path = ve.InstanceLocation
		} else {
			path = ve.InstanceLocation
		}
	}

	// Add the error message if present
	if ve.Message != "" {
		location := path
		if location == "" {
			location = "(root)"
		}
		errors = append(errors, fmt.Sprintf("%s: %s", location, ve.Message))
	}

	// Recursively process child errors
	for _, cause := range ve.Causes {
		errors = append(errors, flattenValidationErrors(cause, path)...)
	}

	return errors
}

// HasErrors returns true if any validation result has errors.
func HasErrors(results []SchemaValidateResult) bool {
	for _, r := range results {
		if r.Error != nil {
			return true
		}
	}
	return false
}

// FormatResults formats validation results as a human-readable string.
func FormatResults(results []SchemaValidateResult) string {
	var b strings.Builder

	for _, r := range results {
		if r.Skipped {
			b.WriteString(fmt.Sprintf("⏭️  %s (optional, not found)\n", filepath.Base(r.File)))
		} else if r.Valid {
			b.WriteString(fmt.Sprintf("✅ %s\n", filepath.Base(r.File)))
		} else if r.Error != nil {
			b.WriteString(r.Error.Error())
		}
	}

	return b.String()
}
