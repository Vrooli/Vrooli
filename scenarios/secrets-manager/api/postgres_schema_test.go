package main

import (
	"strings"
	"testing"

	"secrets-manager-api/internal/secrets"
)

func TestPostgresSchemaIsEmbeddedAndIdempotent(t *testing.T) {
	schema := secrets.Schema()
	if !strings.Contains(schema, "CREATE TABLE IF NOT EXISTS resource_secrets") {
		t.Fatal("embedded PostgreSQL schema does not declare resource_secrets")
	}
	if strings.Contains(schema, "ALTER TABLE") {
		t.Fatal("declarative schema must not contain compatibility ALTER statements")
	}
	if !strings.Contains(secrets.ResourceSecretMetadataMigration(), "ALTER TABLE resource_secrets") {
		t.Fatal("metadata migration is not embedded")
	}
}
