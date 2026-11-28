package cliout

import "io"

const (
	EnvelopeKeySuccess = "success"
)

// FieldsWithSuccess builds the standard CLI JSON envelope with an explicit success value.
func FieldsWithSuccess(success bool, fields map[string]any) map[string]any {
	payload := make(map[string]any, len(fields)+1)
	payload[EnvelopeKeySuccess] = success
	for key, value := range fields {
		payload[key] = value
	}
	return payload
}

// SuccessFields builds the standard success envelope used by CLI JSON output.
func SuccessFields(fields map[string]any) map[string]any {
	return FieldsWithSuccess(true, fields)
}

// WriteSuccessJSON emits the standard success envelope with a single payload key.
func WriteSuccessJSON(w io.Writer, key string, value any) error {
	return WriteJSON(w, SuccessFields(map[string]any{key: value}))
}

// WriteSuccessFields emits the standard success envelope with additional fields.
func WriteSuccessFields(w io.Writer, fields map[string]any) error {
	return WriteJSON(w, SuccessFields(fields))
}

// WriteFieldsWithSuccess emits the standard envelope with an explicit success value.
func WriteFieldsWithSuccess(w io.Writer, success bool, fields map[string]any) error {
	return WriteJSON(w, FieldsWithSuccess(success, fields))
}
