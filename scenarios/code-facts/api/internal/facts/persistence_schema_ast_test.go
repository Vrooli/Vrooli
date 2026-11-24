package facts

import "testing"

func TestExactSchemaBindingUsesActualGoDirectiveAndDirectReturn(t *testing.T) {
	source := []byte("package catalog\nimport _ \"embed\"\n//go:embed schema.sql\nvar schemaSQL string\nfunc Schema() string { return schemaSQL }\n")
	line, ok := exactSchemaBinding(source)
	if !ok || line != 3 {
		t.Fatalf("valid schema binding = (%d, %t)", line, ok)
	}
}

func TestExactSchemaBindingRejectsDirectiveAndFunctionTextInsideRawString(t *testing.T) {
	source := []byte("package catalog\nimport _ \"embed\"\nconst fixture = `\n//go:embed schema.sql\nvar schemaSQL string\nfunc Schema() string { return schemaSQL }\n`\n")
	if line, ok := exactSchemaBinding(source); ok {
		t.Fatalf("raw string text must not establish source lineage: line=%d", line)
	}
}

func TestExactSchemaBindingRejectsMultipleActualEmbedBindings(t *testing.T) {
	source := []byte("package catalog\nimport _ \"embed\"\n//go:embed schema.sql\nvar firstSchema string\n//go:embed schema.sql\nvar secondSchema string\nfunc Schema() string { return firstSchema }\n")
	if line, ok := exactSchemaBinding(source); ok {
		t.Fatalf("multiple embedded schema variables must remain ambiguous: line=%d", line)
	}
}
