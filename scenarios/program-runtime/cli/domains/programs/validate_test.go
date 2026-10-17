package programs

import (
	"encoding/json"
	"strings"
	"testing"
)

// resolveSchemaPath walks up for scenarios/program-runtime/schemas; the test
// runs inside that tree, so a contract with no $schema still resolves.
func TestResolveSchemaPathWalksUpToTheRepo(t *testing.T) {
	path := resolveSchemaPath(".", map[string]any{})
	if path == "" || !strings.HasSuffix(path, "program-contract.schema.json") {
		t.Fatalf("resolveSchemaPath walk-up = %q, want the canonical schema", path)
	}
}

func TestValidateAgainstSchemaAcceptsScaffoldAndRejectsViolations(t *testing.T) {
	schemaPath := resolveSchemaPath(".", map[string]any{})
	if schemaPath == "" {
		t.Skip("canonical schema not resolvable from the test tree")
	}

	// A scaffolded contract must pass the authoritative schema.
	res, err := buildScaffold("program-runtime.unit-probe", "A unit test contract that exercises the authoritative schema validation path.", "vrooli/scenario/status")
	if err != nil {
		t.Fatalf("buildScaffold: %v", err)
	}
	var good map[string]any
	if err := json.Unmarshal([]byte(res.ContractJSON), &good); err != nil {
		t.Fatalf("scaffold JSON: %v", err)
	}
	if errs := validateAgainstSchema(schemaPath, good); len(errs) != 0 {
		t.Fatalf("scaffolded contract should validate, got: %v", errs)
	}

	// Empty errors.classes violates minItems:1 and must be reported with its location.
	good["outputs"].(map[string]any)["errors"].(map[string]any)["classes"] = []any{}
	errs := validateAgainstSchema(schemaPath, good)
	if len(errs) == 0 {
		t.Fatal("empty errors.classes should fail schema validation")
	}
	joined := strings.Join(errs, "\n")
	if !strings.Contains(joined, "classes") {
		t.Fatalf("violation should name the classes location, got: %v", errs)
	}
}
