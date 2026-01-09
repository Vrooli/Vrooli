package parsing

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/parsing"
)

// DiscoveredFile represents a found requirement file.
// This is a local type to avoid tight coupling with discovery package.
type DiscoveredFile struct {
	AbsolutePath string
	RelativePath string
	IsIndex      bool
	ModuleDir    string
}

// Validator parses requirement modules and returns validation results.
type Validator interface {
	// Parse parses all discovered files and builds a module index.
	Parse(ctx context.Context, files []DiscoveredFile) ParsingResult
}

// validator wraps requirements/parsing with validation result handling.
type validator struct {
	parser    parsing.Parser
	logWriter io.Writer
}

// New creates a new parsing validator.
func New(logWriter io.Writer) Validator {
	return &validator{
		parser:    parsing.NewDefault(),
		logWriter: logWriter,
	}
}

// NewWithParser creates a validator with a custom parser (for testing).
func NewWithParser(parser parsing.Parser, logWriter io.Writer) Validator {
	return &validator{
		parser:    parser,
		logWriter: logWriter,
	}
}

// Parse implements Validator.
func (v *validator) Parse(ctx context.Context, files []DiscoveredFile) ParsingResult {
	logInfo(v.logWriter, "Parsing requirement files...")

	// Convert to discovery.DiscoveredFile
	discoveryFiles := make([]discovery.DiscoveredFile, len(files))
	for i, f := range files {
		discoveryFiles[i] = discovery.DiscoveredFile{
			AbsolutePath: f.AbsolutePath,
			RelativePath: f.RelativePath,
			IsIndex:      f.IsIndex,
			ModuleDir:    f.ModuleDir,
		}
	}

	index, err := v.parser.ParseAll(ctx, discoveryFiles)
	if err != nil {
		return ParsingResult{
			Result: FailSystem(
				err,
				"Fix malformed JSON in the module files and rerun the phase.",
			),
		}
	}

	var observations []Observation

	// Report parsing errors as warnings
	for _, parseErr := range index.Errors {
		observations = append(observations, NewWarningObservation(fmt.Sprintf("parsing issue: %v", parseErr)))
		logWarn(v.logWriter, "parsing issue: %v", parseErr)
	}

	// Count requirements
	requirementCount := 0
	for _, module := range index.Modules {
		requirementCount += len(module.Requirements)
	}

	logSuccess(v.logWriter, "Parsed %d requirement(s) across %d module(s)", requirementCount, len(index.Modules))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("parsed %d requirement(s)", requirementCount)))

	// Add observations for each module
	for _, module := range index.Modules {
		moduleName := module.EffectiveName()
		observations = append(observations, NewInfoObservation(fmt.Sprintf("module: %s (%d requirements)", moduleName, len(module.Requirements))))
	}

	result := OK()
	result.Observations = observations

	return ParsingResult{
		Result:           result,
		ModuleCount:      len(index.Modules),
		RequirementCount: requirementCount,
		Index:            index,
	}
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}

func logWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARN] ⚠️ %s\n", msg)
}
