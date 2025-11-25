package validator

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/workflow.schema.json
var workflowSchemaBytes []byte

var (
	schemaOnce     sync.Once
	compiledSchema *jsonschema.Schema
	schemaErr      error
)

func loadSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("workflow.schema.json", bytes.NewReader(workflowSchemaBytes)); err != nil {
			schemaErr = fmt.Errorf("failed to load workflow schema: %w", err)
			return
		}
		compiledSchema, schemaErr = compiler.Compile("workflow.schema.json")
	})
	return compiledSchema, schemaErr
}

func convertSchemaErrors(err error) []Issue {
	var validationErr *jsonschema.ValidationError
	if errors.As(err, &validationErr) {
		return flattenValidationErrors(validationErr)
	}
	return []Issue{{
		Severity: SeverityError,
		Code:     "WF_SCHEMA_ERROR",
		Message:  err.Error(),
	}}
}

func flattenValidationErrors(err *jsonschema.ValidationError) []Issue {
	if err == nil {
		return nil
	}
	if len(err.Causes) == 0 {
		return []Issue{{
			Severity: SeverityError,
			Code:     "WF_SCHEMA_INVALID",
			Message:  err.Message,
			Pointer:  err.InstanceLocation,
		}}
	}
	var issues []Issue
	for _, cause := range err.Causes {
		issues = append(issues, flattenValidationErrors(cause)...)
	}
	return issues
}
