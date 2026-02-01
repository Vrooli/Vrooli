package smoketest

import "strings"

// DefaultOutputParser implements OutputParser using configurable markers.
type DefaultOutputParser struct {
	config Config
}

// NewOutputParser creates a new output parser with the given config.
func NewOutputParser(config Config) *DefaultOutputParser {
	return &DefaultOutputParser{config: config}
}

// ParseResult analyzes smoke test output and returns the result.
func (p *DefaultOutputParser) ParseResult(output string) OutputResult {
	result := OutputResult{
		Passed:               strings.Contains(output, p.config.SuccessMarker),
		TelemetryUploaded:    strings.Contains(output, p.config.UploadSuccessMarker),
		TelemetryUploadError: strings.Contains(output, p.config.UploadErrorMarker),
		InitComplete:         strings.Contains(output, p.config.InitMarker),
		CleanShutdown:        strings.Contains(output, p.config.ExitMarker),
		Warnings:             []string{},
	}

	// Check for potential issues and add warnings
	if result.Passed && !result.InitComplete && p.config.InitMarker != "" {
		result.Warnings = append(result.Warnings, "success marker found but init marker missing")
	}
	if result.Passed && !result.CleanShutdown && p.config.ExitMarker != "" {
		result.Warnings = append(result.Warnings, "success marker found but clean exit marker missing")
	}

	return result
}

// ValidateSequence performs detailed validation of the smoke test output sequence.
func (p *DefaultOutputParser) ValidateSequence(output string) SequenceValidation {
	validation := SequenceValidation{
		Valid:            true,
		Stages:           []SequenceStage{},
		MissingStages:    []string{},
		OutOfOrderStages: []string{},
		Errors:           []string{},
	}

	lines := strings.Split(output, "\n")

	// Define expected sequence order
	type markerDef struct {
		name   string
		marker string
	}
	expectedOrder := []markerDef{
		{"init", p.config.InitMarker},
		{"ready", p.config.ReadyMarker},
		{"passed", p.config.SuccessMarker},
		{"exit", p.config.ExitMarker},
	}

	// Track which stages we've seen and their line numbers
	stageLines := make(map[string]int)

	// Scan output for markers
	for lineNum, line := range lines {
		for _, m := range expectedOrder {
			if m.marker != "" && strings.Contains(line, m.marker) {
				if _, seen := stageLines[m.name]; !seen {
					stageLines[m.name] = lineNum + 1 // 1-based line numbers
					validation.Stages = append(validation.Stages, SequenceStage{
						Name:       m.name,
						LineNumber: lineNum + 1,
					})
				}
			}
		}
	}

	// Check for missing stages (only if marker is configured)
	for _, m := range expectedOrder {
		if m.marker == "" {
			continue
		}
		if _, found := stageLines[m.name]; !found {
			// init and exit are optional for backwards compatibility
			if m.name == "init" || m.name == "exit" || m.name == "ready" {
				continue
			}
			validation.MissingStages = append(validation.MissingStages, m.name)
			validation.Valid = false
			validation.Errors = append(validation.Errors, "missing required stage: "+m.name)
		}
	}

	// Check sequence order
	lastLine := 0
	for _, m := range expectedOrder {
		if m.marker == "" {
			continue
		}
		if line, found := stageLines[m.name]; found {
			if line < lastLine {
				validation.OutOfOrderStages = append(validation.OutOfOrderStages, m.name)
				validation.Valid = false
				validation.Errors = append(validation.Errors, "stage out of order: "+m.name)
			}
			lastLine = line
		}
	}

	return validation
}
