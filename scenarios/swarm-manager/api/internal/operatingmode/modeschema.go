package operatingmode

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// schemaBytes is a synced copy of the canonical operating-mode contract at
// .vrooli/schemas/operating-mode.schema.json. The package embeds its own copy
// (go:embed cannot reach the repo-root schema registry from here); the drift
// test asserts the two are byte-identical so the runtime validator and the
// registry never diverge.
//
//go:embed operating-mode.schema.json
var schemaBytes []byte

// SchemaID is the canonical $id of the operating-mode schema.
const SchemaID = "operating-mode/v1"

// Document kind discriminators. One schema covers both document kinds; the
// top-level `kind` field selects the oneOf branch.
const (
	DocumentKindMode       = "operating-mode"
	DocumentKindExampleRun = "operating-mode-example-run"
)

var (
	compiledSchema     *jsonschema.Schema
	compiledSchemaErr  error
	compiledSchemaOnce sync.Once
)

// compileSchema compiles the embedded schema once. A compilation failure is a
// programming error (a malformed embedded schema), surfaced on first use.
func compileSchema() (*jsonschema.Schema, error) {
	compiledSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.Draft = jsonschema.Draft2020
		if err := compiler.AddResource(SchemaID, bytes.NewReader(schemaBytes)); err != nil {
			compiledSchemaErr = fmt.Errorf("add operating-mode schema resource: %w", err)
			return
		}
		schema, err := compiler.Compile(SchemaID)
		if err != nil {
			compiledSchemaErr = fmt.Errorf("compile operating-mode schema: %w", err)
			return
		}
		compiledSchema = schema
	})
	return compiledSchema, compiledSchemaErr
}

// ValidateDocumentBytes validates a raw mode or example-run document against
// the embedded JSON Schema. It returns a typed, actionable error describing the
// first structural violation (unknown field, wrong type, missing required key),
// or nil when the document is structurally valid. Semantic cross-references
// (phase targets, profile ownership, guard-field consistency) are checked
// separately by the loader/validator.
func ValidateDocumentBytes(raw []byte) error {
	schema, err := compileSchema()
	if err != nil {
		return err
	}
	var instance any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&instance); err != nil {
		return fmt.Errorf("decode document JSON: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}
