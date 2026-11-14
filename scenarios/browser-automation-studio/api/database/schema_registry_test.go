package database

import (
	"strings"
	"testing"
)

func TestSchemaRegistryHasStableDomainOrderAndSources(t *testing.T) {
	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		t.Fatalf("resolveScenarioRoot() error = %v", err)
	}

	schemas, statements, err := loadSchemaStatements(scenarioRoot)
	if err != nil {
		t.Fatalf("loadSchemaStatements() error = %v", err)
	}
	wantDomains := []string{"core", "recording", "billing", "uxmetrics", "lifecycle"}
	if len(schemas) != len(wantDomains) || len(statements) != len(wantDomains) {
		t.Fatalf("schema registry sizes = (%d, %d), want %d", len(schemas), len(statements), len(wantDomains))
	}
	for index, domain := range wantDomains {
		if schemas[index].Domain != domain {
			t.Fatalf("schema %d domain = %q, want %q", index, schemas[index].Domain, domain)
		}
		if len(strings.TrimSpace(string(statements[index]))) == 0 {
			t.Fatalf("schema %q is empty", domain)
		}
	}

	if !strings.Contains(string(statements[0]), "CREATE TABLE IF NOT EXISTS projects") {
		t.Fatal("core schema must create projects before dependent domains")
	}
	if !strings.Contains(string(statements[0]), "CREATE TABLE IF NOT EXISTS project_assets") {
		t.Fatal("core schema must create project_assets for file-tree and sync consumers")
	}
	if !strings.Contains(string(statements[len(statements)-1]), "CREATE TRIGGER IF NOT EXISTS") {
		t.Fatal("lifecycle schema must install updated_at triggers after tables")
	}
}

func TestSchemaProvidersExposeCompleteEmbeddedSQL(t *testing.T) {
	providers, err := SchemaProviders("")
	if err != nil {
		t.Fatalf("SchemaProviders() error = %v", err)
	}
	if len(providers) != len(SchemaRegistry) {
		t.Fatalf("provider count = %d, want %d", len(providers), len(SchemaRegistry))
	}
	for index, provider := range providers {
		schema := provider.Schema()
		if strings.Contains(schema, "package ") || strings.Contains(schema, "//go:embed") {
			t.Fatalf("embedded %q provider returned Go source instead of SQL", SchemaRegistry[index].Domain)
		}
		if !strings.Contains(schema, "CREATE TABLE") && !strings.Contains(schema, "CREATE TRIGGER") {
			t.Fatalf("embedded %q provider contains no executable schema", SchemaRegistry[index].Domain)
		}
	}
}
