package secrets

import (
	"strings"
	"testing"
)

func TestSchemaContainsResourceSecretsWithoutInlineMigration(t *testing.T) {
	schema := Schema()
	if !strings.Contains(schema, "CREATE TABLE IF NOT EXISTS resource_secrets") {
		t.Fatal("schema is missing the resource_secrets table")
	}
	if strings.Contains(strings.ToUpper(schema), "ALTER TABLE") {
		t.Fatal("schema must not contain compatibility ALTER TABLE statements")
	}
}
