package agentops

import (
	"encoding/json"
	"fmt"
)

// ExecutionProvenance is the typed shape of execution-provenance.schema.json —
// the immutable record pinned when an operation execution starts.
type ExecutionProvenance struct {
	Kind                  string            `json:"kind"`
	Operation             OperationID       `json:"operation"`
	OperationVersion      string            `json:"operation_version"`
	Binding               ProvenanceBinding `json:"binding"`
	Mode                  string            `json:"mode"`
	ModeRevision          string            `json:"mode_revision"`
	CompiledModeDigest    string            `json:"compiled_mode_digest"`
	PromptCatalogRevision string            `json:"prompt_catalog_revision"`
	PromptCatalogDigest   string            `json:"prompt_catalog_digest"`
	Target                ProvenanceTarget  `json:"target"`
	CallerInputDigest     string            `json:"caller_input_digest"`
	PolicyRevision        string            `json:"policy_revision"`
	WorkflowInstanceID    string            `json:"workflow_instance_id"`
}

type ProvenanceBinding struct {
	Layer     BindingLayer `json:"layer"`
	OwnerKind string       `json:"owner_kind"`
	OwnerID   string       `json:"owner_id"`
}

type ProvenanceTarget struct {
	Kind TargetKind `json:"kind"`
	ID   string     `json:"id"`
}

// ValidateProvenance validates a provenance document against the schema and the
// semantic completeness rules JSON Schema cannot express: the operation and
// target kind are registered, the binding layer is known, and a system-default
// binding is attributed to the `system` owner. Every field is required by the
// schema, so a partial provenance record can never authorize a run.
func ValidateProvenance(raw []byte) error {
	if err := ValidateDocument(SchemaExecutionProvenance, raw); err != nil {
		return err
	}
	var p ExecutionProvenance
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("decode execution provenance: %w", err)
	}
	if !IsValidOperationID(p.Operation) {
		return fmt.Errorf("provenance names unregistered operation %q", p.Operation)
	}
	if !IsValidTargetKind(p.Target.Kind) {
		return fmt.Errorf("provenance names unknown target kind %q", p.Target.Kind)
	}
	if _, ok := layerRank[p.Binding.Layer]; !ok {
		return fmt.Errorf("provenance binding has unknown layer %q", p.Binding.Layer)
	}
	if p.Binding.Layer == LayerSystemDefault && p.Binding.OwnerKind != "system" {
		return fmt.Errorf("system-default provenance binding owner kind must be %q (got %q)", "system", p.Binding.OwnerKind)
	}
	if p.Binding.Layer != LayerSystemDefault && p.Binding.OwnerKind == "system" {
		return fmt.Errorf("override provenance binding must not use the system owner kind")
	}
	return nil
}
