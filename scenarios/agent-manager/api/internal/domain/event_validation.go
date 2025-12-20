// Package domain provides core domain types and business logic.
//
// This file provides event validation helpers that log warnings when
// events are missing expected data. This helps detect parsing issues
// and incomplete event capture from runners.
package domain

import (
	"log"
)

// ValidateEvent validates a RunEvent and logs warnings for missing data.
// This is a convenience function that calls the appropriate validator.
func ValidateEvent(evt *RunEvent) {
	if evt == nil {
		return
	}

	switch evt.EventType {
	case EventTypeToolCall:
		validateToolCallEvent(evt)
	case EventTypeToolResult:
		validateToolResultEvent(evt)
	case EventTypeMessage:
		validateMessageEvent(evt)
	case EventTypeMetric:
		validateMetricEvent(evt)
	}
}

// validateToolCallEvent logs warnings for tool_call events with missing data.
func validateToolCallEvent(evt *RunEvent) {
	data, ok := evt.Data.(*ToolCallEventData)
	if !ok {
		log.Printf("[WARN] tool_call event has wrong data type: %T (runID=%s)", evt.Data, evt.RunID)
		return
	}

	if data.ToolName == "" {
		log.Printf("[WARN] tool_call event has empty toolName (runID=%s)", evt.RunID)
	} else if data.ToolName == "unknown_tool" {
		log.Printf("[WARN] tool_call event has unknown toolName - parsing may have failed (runID=%s)", evt.RunID)
	}

	if data.Input == nil || len(data.Input) == 0 {
		log.Printf("[WARN] tool_call event has empty input (runID=%s, tool=%s)", evt.RunID, data.ToolName)
	}
}

// validateToolResultEvent logs warnings for tool_result events with missing data.
func validateToolResultEvent(evt *RunEvent) {
	data, ok := evt.Data.(*ToolResultEventData)
	if !ok {
		log.Printf("[WARN] tool_result event has wrong data type: %T (runID=%s)", evt.Data, evt.RunID)
		return
	}

	if data.ToolName == "" {
		log.Printf("[WARN] tool_result event has empty toolName (runID=%s)", evt.RunID)
	}

	// Output can legitimately be empty for some tool results (e.g., file write)
	// Only warn if both output and error are empty on a failed result
	if !data.Success && data.Output == "" && data.Error == "" {
		log.Printf("[WARN] failed tool_result event has no error details (runID=%s, tool=%s)", evt.RunID, data.ToolName)
	}
}

// validateMessageEvent logs warnings for message events with missing content.
func validateMessageEvent(evt *RunEvent) {
	data, ok := evt.Data.(*MessageEventData)
	if !ok {
		log.Printf("[WARN] message event has wrong data type: %T (runID=%s)", evt.Data, evt.RunID)
		return
	}

	if data.Role == "" {
		log.Printf("[WARN] message event has empty role (runID=%s)", evt.RunID)
	}

	if data.Content == "" {
		log.Printf("[WARN] message event has empty content (runID=%s, role=%s)", evt.RunID, data.Role)
	}
}

// validateMetricEvent logs warnings for metric events with suspicious data.
func validateMetricEvent(evt *RunEvent) {
	// Handle CostEventData (the primary metric type)
	if data, ok := evt.Data.(*CostEventData); ok {
		// Zero tokens might indicate a parsing issue
		if data.InputTokens == 0 && data.OutputTokens == 0 {
			log.Printf("[WARN] metric event has zero tokens (runID=%s)", evt.RunID)
		}
		return
	}

	// Handle legacy MetricEventData
	if data, ok := evt.Data.(*MetricEventData); ok {
		if data.Name == "" {
			log.Printf("[WARN] metric event has empty name (runID=%s)", evt.RunID)
		}
		return
	}

	log.Printf("[WARN] metric event has unexpected data type: %T (runID=%s)", evt.Data, evt.RunID)
}

// EventValidationStats tracks validation statistics for a set of events.
type EventValidationStats struct {
	TotalEvents      int
	ToolCallCount    int
	ToolResultCount  int
	MessageCount     int
	MetricCount      int
	ErrorCount       int
	LogCount         int
	WarningCount     int
	EmptyToolNames   int
	EmptyInputs      int
	EmptyMessages    int
}

// ValidateEvents validates a slice of events and returns statistics.
// This is useful for batch validation and debugging.
func ValidateEvents(events []*RunEvent) EventValidationStats {
	stats := EventValidationStats{
		TotalEvents: len(events),
	}

	for _, evt := range events {
		if evt == nil {
			continue
		}

		switch evt.EventType {
		case EventTypeToolCall:
			stats.ToolCallCount++
			if data, ok := evt.Data.(*ToolCallEventData); ok {
				if data.ToolName == "" || data.ToolName == "unknown_tool" {
					stats.EmptyToolNames++
					stats.WarningCount++
				}
				if data.Input == nil || len(data.Input) == 0 {
					stats.EmptyInputs++
					stats.WarningCount++
				}
			}
		case EventTypeToolResult:
			stats.ToolResultCount++
		case EventTypeMessage:
			stats.MessageCount++
			if data, ok := evt.Data.(*MessageEventData); ok {
				if data.Content == "" {
					stats.EmptyMessages++
					stats.WarningCount++
				}
			}
		case EventTypeMetric:
			stats.MetricCount++
		case EventTypeError:
			stats.ErrorCount++
		case EventTypeLog:
			stats.LogCount++
		}
	}

	return stats
}
