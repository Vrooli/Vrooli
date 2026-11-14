package main

import "fmt"

// BindingError is safe to show inline in the settings editor and stable for
// callers that want to classify validation failures.
type BindingError struct {
	Code     string `json:"code"`
	SignalID string `json:"signalId"`
	Slot     string `json:"slot"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Column   string `json:"column,omitempty"`
	Message  string `json:"message"`
}

func (e BindingError) Error() string { return e.Message }

// ValidateSignalBinding enforces the four engine-level signal shapes. The
// validator runs before render so malformed custom instances fail in the
// editor/API rather than producing a misleading scene.
func ValidateSignalBinding(signal MetricEntry, expectedShape string, requiredColumns []string, slot string) error {
	if signal.Shape == "" {
		return BindingError{Code: "signal_shape_missing", SignalID: signal.ID, Slot: slot, Expected: expectedShape, Message: fmt.Sprintf("signal %q has no declared shape", signal.ID)}
	}
	if expectedShape != "" && signal.Shape != expectedShape {
		return BindingError{Code: "signal_shape_mismatch", SignalID: signal.ID, Slot: slot, Expected: expectedShape, Actual: signal.Shape, Message: fmt.Sprintf("signal %q has shape %q; slot %q requires %q", signal.ID, signal.Shape, slot, expectedShape)}
	}
	for _, column := range requiredColumns {
		spec, ok := signal.Columns[column]
		if !ok || spec.Optional {
			return BindingError{Code: "required_column_missing", SignalID: signal.ID, Slot: slot, Expected: "rows", Actual: signal.Shape, Column: column, Message: fmt.Sprintf("signal %q is missing required rows column %q", signal.ID, column)}
		}
	}
	return nil
}

func validateSignalShapes(reg *Registry) error {
	for _, signal := range reg.Metrics {
		if signal.Shape != "scalar" && signal.Shape != "series" && signal.Shape != "rows" && signal.Shape != "meta" {
			return fmt.Errorf("metric %q has invalid shape %q", signal.ID, signal.Shape)
		}
		if signal.Shape == "rows" && len(signal.Columns) == 0 {
			return fmt.Errorf("rows metric %q must declare columns", signal.ID)
		}
	}
	return nil
}
