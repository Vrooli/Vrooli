// Package agentops holds the canonical, provider-neutral contracts for the
// declarative agent-operations model (EXECUTION-MODES.md D1–D8): operation
// contracts, target capabilities, layered bindings, execution provenance,
// durable domain workflow instances, transition policies over a closed
// domain-action registry, and initiative member-item strategy configuration.
//
// This package is CONTRACTS-only. It defines the vocabulary and the semantic
// validators that fail closed BEFORE any agent run starts; the generic runner
// that consumes these contracts is built in a later phase. Nothing here
// executes an operation, spawns an agent, or mutates domain state.
//
// The schemas live beside this code under schemas/*.json and are embedded (a
// single canonical copy per contract; unlike the operating-mode schema they
// are swarm-manager-local with exactly one consumer, so there is no repo-root
// copy and no drift test to maintain). JSON Schema pins the structural shape;
// the Go validators enforce what JSON Schema cannot (registry membership,
// precedence determinism, capability compatibility, digest well-formedness).
package agentops

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas/*.json
var schemaFS embed.FS

// Schema $id constants — the canonical identity of each contract schema.
const (
	SchemaTargetCapability    = "agentops/target-capability/v1"
	SchemaOperationContract   = "agentops/operation-contract/v1"
	SchemaOperationBinding    = "agentops/operation-binding/v1"
	SchemaExecutionProvenance = "agentops/execution-provenance/v1"
	SchemaWorkflowInstance    = "agentops/workflow-instance/v1"
	SchemaTransitionPolicy    = "agentops/transition-policy/v1"
	SchemaMemberItemStrategy  = "agentops/member-item-strategy/v1"
)

// schemaFiles maps each schema $id to its embedded file.
var schemaFiles = map[string]string{
	SchemaTargetCapability:    "schemas/target-capability.schema.json",
	SchemaOperationContract:   "schemas/operation-contract.schema.json",
	SchemaOperationBinding:    "schemas/operation-binding.schema.json",
	SchemaExecutionProvenance: "schemas/execution-provenance.schema.json",
	SchemaWorkflowInstance:    "schemas/workflow-instance.schema.json",
	SchemaTransitionPolicy:    "schemas/transition-policy.schema.json",
	SchemaMemberItemStrategy:  "schemas/member-item-strategy.schema.json",
}

var (
	compiled     = map[string]*jsonschema.Schema{}
	compiledErr  = map[string]error{}
	compiledOnce sync.Once
)

func compileAll() {
	compiledOnce.Do(func() {
		for id, path := range schemaFiles {
			raw, err := schemaFS.ReadFile(path)
			if err != nil {
				compiledErr[id] = fmt.Errorf("read embedded schema %s: %w", path, err)
				continue
			}
			c := jsonschema.NewCompiler()
			c.Draft = jsonschema.Draft2020
			if err := c.AddResource(id, bytes.NewReader(raw)); err != nil {
				compiledErr[id] = fmt.Errorf("add schema %s: %w", id, err)
				continue
			}
			schema, err := c.Compile(id)
			if err != nil {
				compiledErr[id] = fmt.Errorf("compile schema %s: %w", id, err)
				continue
			}
			compiled[id] = schema
		}
	})
}

// SchemaBytes returns the raw embedded bytes of a contract schema by $id. Used
// by tests and by any tooling that needs the canonical schema source.
func SchemaBytes(schemaID string) ([]byte, error) {
	path, ok := schemaFiles[schemaID]
	if !ok {
		return nil, fmt.Errorf("unknown agentops schema id %q", schemaID)
	}
	return schemaFS.ReadFile(path)
}

// SchemaIDs returns the sorted-free set of registered schema ids.
func SchemaIDs() []string {
	ids := make([]string, 0, len(schemaFiles))
	for id := range schemaFiles {
		ids = append(ids, id)
	}
	return ids
}

// ValidateDocument validates a raw JSON document against the named contract
// schema. It returns a typed, actionable error describing the first structural
// violation, or nil when the document conforms. Semantic checks beyond JSON
// Schema (registry membership, digests, precedence) live in the per-contract
// validators that call this first.
func ValidateDocument(schemaID string, raw []byte) error {
	compileAll()
	if err := compiledErr[schemaID]; err != nil {
		return err
	}
	schema, ok := compiled[schemaID]
	if !ok {
		return fmt.Errorf("unknown agentops schema id %q", schemaID)
	}
	var doc any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("decode document for %s: %w", schemaID, err)
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("%s validation: %w", schemaID, err)
	}
	return nil
}
