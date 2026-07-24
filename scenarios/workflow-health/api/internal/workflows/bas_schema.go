package workflows

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// BASDocument separates the authored/catalog envelope from the V2 graph that
// BAS validates and executes. Persisted BAS workflows are WorkflowSummary-like
// documents with a nested flow_definition; reusable assets can be bare V2
// definitions. Workflow Health supports both without weakening V2 validation.
type BASDocument struct {
	Definition            map[string]any
	Catalog               map[string]any
	Envelope              bool
	UnknownEnvelopeFields []string
}

var knownEnvelopeFields = map[string]struct{}{
	"id": {}, "project_id": {}, "folder_path": {}, "created_at": {}, "updated_at": {},
	"name": {}, "description": {}, "tags": {}, "version": {}, "metadata": {},
	"flow_definition": {}, "definition_v2": {},
}

// ResolveBASDocument identifies the authored file shape. It never discards
// fields from the graph: the selected definition is decoded strictly later.
func ResolveBASDocument(doc map[string]any) (BASDocument, error) {
	for _, key := range []string{"flow_definition", "definition_v2"} {
		raw, exists := doc[key]
		if !exists {
			continue
		}
		definition, ok := raw.(map[string]any)
		if !ok || definition == nil {
			return BASDocument{}, fmt.Errorf("BAS workflow envelope field %q must be an object", key)
		}
		catalog := cloneObject(doc)
		outerMetadata, outerPresent, err := objectField(doc, "metadata")
		if err != nil {
			return BASDocument{}, err
		}
		innerMetadata, _, err := objectField(definition, "metadata")
		if err != nil {
			return BASDocument{}, fmt.Errorf("BAS %s: %w", key, err)
		}
		if outerPresent && innerMetadata != nil {
			if outerMode, innerMode := normalizedExecutionMode(outerMetadata), normalizedExecutionMode(innerMetadata); outerMode != "" && innerMode != "" && outerMode != innerMode {
				return BASDocument{}, fmt.Errorf("BAS envelope metadata.execution_mode %q conflicts with %s.metadata.execution_mode %q", outerMode, key, innerMode)
			}
		}
		if !outerPresent && innerMetadata != nil {
			catalog["metadata"] = cloneObject(innerMetadata)
		}
		unknown := make([]string, 0)
		for field := range doc {
			if _, known := knownEnvelopeFields[field]; !known {
				unknown = append(unknown, field)
			}
		}
		sort.Strings(unknown)
		return BASDocument{Definition: definition, Catalog: catalog, Envelope: true, UnknownEnvelopeFields: unknown}, nil
	}
	return BASDocument{Definition: doc, Catalog: doc}, nil
}

func objectField(doc map[string]any, key string) (map[string]any, bool, error) {
	raw, present := doc[key]
	if !present || raw == nil {
		return nil, false, nil
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("metadata must be an object")
	}
	return value, true, nil
}

func cloneObject(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func normalizedExecutionMode(metadata map[string]any) string {
	mode, _ := metadata["execution_mode"].(string)
	return strings.ToLower(strings.TrimSpace(mode))
}

// DecodeBASDefinition validates an authored workflow against BAS's canonical
// WorkflowDefinitionV2 schema. Unknown fields remain errors: workflow-health
// must report schema drift during static validation, not defer it to execution.
func DecodeBASDefinition(definition map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	document, err := ResolveBASDocument(definition)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(normalizeBASDefinitionAliases(document.Definition))
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	var out basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode BAS WorkflowDefinitionV2: %w", err)
	}
	return &out, nil
}

// DecodeBASDefinitionJSON is the static-validation entry point for authored
// workflow files. It shares the execution decoder exactly.
func DecodeBASDefinitionJSON(data []byte) (*basworkflows.WorkflowDefinitionV2, error) {
	var definition map[string]any
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("parse workflow JSON: %w", err)
	}
	return DecodeBASDefinition(definition)
}

// ResolveBASDocumentJSON parses an authored workflow once for callers that
// need both the catalog envelope and the V2 graph.
func ResolveBASDocumentJSON(data []byte) (BASDocument, error) {
	var definition map[string]any
	if err := json.Unmarshal(data, &definition); err != nil {
		return BASDocument{}, fmt.Errorf("parse workflow JSON: %w", err)
	}
	return ResolveBASDocument(definition)
}

func normalizeBASDefinitionAliases(definition map[string]any) map[string]any {
	out := make(map[string]any, len(definition))
	for key, value := range definition {
		out[key] = value
	}
	metadata, ok := definition["metadata"].(map[string]any)
	if !ok {
		return out
	}
	metadataOut := make(map[string]any, len(metadata))
	for key, value := range metadata {
		metadataOut[key] = value
	}
	if mode, ok := metadataOut["execution_mode"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "observer":
			metadataOut["execution_mode"] = "EXECUTION_MODE_OBSERVER"
		case "mutating":
			metadataOut["execution_mode"] = "EXECUTION_MODE_MUTATING"
		case "destructive":
			metadataOut["execution_mode"] = "EXECUTION_MODE_DESTRUCTIVE"
		}
	}
	out["metadata"] = metadataOut
	return out
}
