package agentops

import (
	"encoding/json"
	"fmt"
)

// LegacyExecutionImport is the typed shape of
// legacy-execution-import.schema.json: the Phase-8 migrator's full-fidelity
// import snapshot of one pre-cutover execution-runs.json entry. The header
// correlates the import; Legacy carries the verbatim legacy entry. It is a
// dedicated kind — NOT an ExecutionProvenance — because provenance digests
// never existed for legacy runs and fabricating them would violate the
// no-fabricated-identity rule.
type LegacyExecutionImport struct {
	Kind               string          `json:"kind"`
	SchemaVersion      string          `json:"schema_version"`
	Operation          OperationID     `json:"operation"`
	ExecutionID        string          `json:"execution_id"`
	WorkflowInstanceID string          `json:"workflow_instance_id"`
	ImportedAt         string          `json:"imported_at"`
	Legacy             json.RawMessage `json:"legacy"`
}

// ValidateLegacyExecutionImport validates an import snapshot against the
// schema plus the semantic rule JSON Schema cannot express: the operation the
// legacy entry maps onto must be a registered operation contract.
func ValidateLegacyExecutionImport(raw []byte) error {
	if err := ValidateDocument(SchemaLegacyExecutionImport, raw); err != nil {
		return err
	}
	var imp LegacyExecutionImport
	if err := json.Unmarshal(raw, &imp); err != nil {
		return fmt.Errorf("decode legacy execution import: %w", err)
	}
	if !IsValidOperationID(imp.Operation) {
		return fmt.Errorf("legacy execution import %q maps to unregistered operation %q", imp.ExecutionID, imp.Operation)
	}
	return nil
}

// DecodeLegacyExecutionImport decodes a raw import snapshot document.
func DecodeLegacyExecutionImport(raw []byte) (LegacyExecutionImport, error) {
	var imp LegacyExecutionImport
	if err := json.Unmarshal(raw, &imp); err != nil {
		return LegacyExecutionImport{}, fmt.Errorf("decode legacy execution import: %w", err)
	}
	return imp, nil
}
