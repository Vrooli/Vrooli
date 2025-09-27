package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExecutionTrace represents a step-by-step trace of algorithm execution
type ExecutionTrace struct {
	AlgorithmID   string         `json:"algorithm_id"`
	Language      string         `json:"language"`
	Input         interface{}    `json:"input"`
	Steps         []TraceStep    `json:"steps"`
	Output        interface{}    `json:"output"`
	ExecutionTime int            `json:"execution_time_ms"`
	MemoryUsed    int            `json:"memory_used_bytes,omitempty"`
	Success       bool           `json:"success"`
}

// TraceStep represents a single step in algorithm execution
type TraceStep struct {
	StepNumber  int                    `json:"step_number"`
	Operation   string                 `json:"operation"`
	Variables   map[string]interface{} `json:"variables"`
	Description string                 `json:"description"`
	Timestamp   int64                  `json:"timestamp_ms"`
	LineNumber  int                    `json:"line_number,omitempty"`
}

// ExecutionTracer handles tracing of algorithm execution
type ExecutionTracer struct {
	localExecutor *LocalExecutor
}

// NewExecutionTracer creates a new execution tracer
func NewExecutionTracer() *ExecutionTracer {
	return &ExecutionTracer{
		localExecutor: NewLocalExecutor(5 * time.Second),
	}
}

// TraceExecution traces the execution of an algorithm with given input
func (t *ExecutionTracer) TraceExecution(algorithmID, language, code string, input interface{}) (*ExecutionTrace, error) {
	trace := &ExecutionTrace{
		AlgorithmID: algorithmID,
		Language:    language,
		Input:       input,
		Steps:       []TraceStep{},
		Success:     false,
	}

	startTime := time.Now()

	// Instrument the code based on language
	instrumentedCode, err := t.instrumentCode(language, code)
	if err != nil {
		return trace, fmt.Errorf("failed to instrument code: %v", err)
	}

	// Execute the instrumented code
	inputJSON, _ := json.Marshal(input)
	result, err := t.executeInstrumentedCode(language, instrumentedCode, string(inputJSON))
	if err != nil {
		return trace, fmt.Errorf("execution failed: %v", err)
	}

	// Parse the trace output
	if err := t.parseTraceOutput(result, trace); err != nil {
		return trace, fmt.Errorf("failed to parse trace output: %v", err)
	}

	trace.ExecutionTime = int(time.Since(startTime).Milliseconds())
	trace.Success = true

	return trace, nil
}

// instrumentCode adds tracing instrumentation to the code
func (t *ExecutionTracer) instrumentCode(language, code string) (string, error) {
	switch strings.ToLower(language) {
	case "python":
		return t.instrumentPython(code), nil
	case "javascript":
		return t.instrumentJavaScript(code), nil
	default:
		return "", fmt.Errorf("tracing not supported for language: %s", language)
	}
}

// instrumentPython adds tracing to Python code
func (t *ExecutionTracer) instrumentPython(code string) string {
	instrumented := `
import json
import sys
import time

_trace_steps = []
_step_counter = 0

def _trace(operation, variables, description="", line_number=0):
    global _step_counter, _trace_steps
    _step_counter += 1
    _trace_steps.append({
        "step_number": _step_counter,
        "operation": operation,
        "variables": variables,
        "description": description,
        "timestamp_ms": int(time.time() * 1000),
        "line_number": line_number
    })

# Original code with tracing
`
	// Add the original code with some basic instrumentation points
	// For a complete implementation, we'd parse the AST and add precise instrumentation
	instrumented += code + `

# Output the trace
print("__TRACE_START__")
print(json.dumps(_trace_steps))
print("__TRACE_END__")
`
	return instrumented
}

// instrumentJavaScript adds tracing to JavaScript code  
func (t *ExecutionTracer) instrumentJavaScript(code string) string {
	instrumented := `
const _trace_steps = [];
let _step_counter = 0;

function _trace(operation, variables, description = "", lineNumber = 0) {
    _step_counter++;
    _trace_steps.push({
        step_number: _step_counter,
        operation: operation,
        variables: variables,
        description: description,
        timestamp_ms: Date.now(),
        line_number: lineNumber
    });
}

// Original code with tracing
`
	instrumented += code + `

// Output the trace
console.log("__TRACE_START__");
console.log(JSON.stringify(_trace_steps));
console.log("__TRACE_END__");
`
	return instrumented
}

// executeInstrumentedCode runs the instrumented code and captures trace output
func (t *ExecutionTracer) executeInstrumentedCode(language, code, input string) (string, error) {
	switch strings.ToLower(language) {
	case "python":
		result, err := t.localExecutor.ExecutePython(code, input)
		if err != nil {
			return "", err
		}
		return result.Output, nil
	case "javascript":
		result, err := t.localExecutor.ExecuteJavaScript(code, input)
		if err != nil {
			return "", err
		}
		return result.Output, nil
	default:
		return "", fmt.Errorf("execution not supported for language: %s", language)
	}
}

// parseTraceOutput extracts trace steps from the execution output
func (t *ExecutionTracer) parseTraceOutput(output string, trace *ExecutionTrace) error {
	// Extract trace JSON from output between markers
	startMarker := "__TRACE_START__"
	endMarker := "__TRACE_END__"
	
	startIdx := strings.Index(output, startMarker)
	endIdx := strings.Index(output, endMarker)
	
	if startIdx == -1 || endIdx == -1 {
		// No trace markers found, try to extract basic output
		trace.Output = strings.TrimSpace(output)
		return nil
	}

	traceJSON := output[startIdx+len(startMarker):endIdx]
	traceJSON = strings.TrimSpace(traceJSON)

	// Parse the trace steps
	var steps []TraceStep
	if err := json.Unmarshal([]byte(traceJSON), &steps); err != nil {
		return fmt.Errorf("failed to parse trace JSON: %v", err)
	}

	trace.Steps = steps

	// Extract actual output (everything after trace end marker)
	actualOutput := strings.TrimSpace(output[endIdx+len(endMarker):])
	if actualOutput != "" {
		trace.Output = actualOutput
	}

	return nil
}

// GenerateVisualization creates a visual representation of the execution trace
func (t *ExecutionTracer) GenerateVisualization(trace *ExecutionTrace) map[string]interface{} {
	visualization := map[string]interface{}{
		"algorithm_id": trace.AlgorithmID,
		"language":     trace.Language,
		"input":        trace.Input,
		"output":       trace.Output,
		"total_steps":  len(trace.Steps),
		"execution_ms": trace.ExecutionTime,
		"timeline":     []map[string]interface{}{},
	}

	// Create timeline visualization
	for _, step := range trace.Steps {
		stepViz := map[string]interface{}{
			"step":        step.StepNumber,
			"operation":   step.Operation,
			"description": step.Description,
			"variables":   step.Variables,
			"time_ms":     step.Timestamp,
		}
		visualization["timeline"] = append(visualization["timeline"].([]map[string]interface{}), stepViz)
	}

	return visualization
}