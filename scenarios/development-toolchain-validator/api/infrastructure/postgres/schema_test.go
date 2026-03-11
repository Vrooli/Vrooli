// DOC: initialization/postgres/schema.sql
// [REQ:REQ-P0-001] Reference Scenario Database Schema - Schema validation tests
//
// This file validates that the PostgreSQL schema has all required tables and columns.
// These tests verify the schema definition matches the domain model requirements.
package postgres

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestSchemaHasRequiredTables verifies all required tables exist in the schema.
func TestSchemaHasRequiredTables(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	requiredTables := []struct {
		name        string
		description string
		category    string
	}{
		{
			name:        "reference_scenarios",
			description: "Stores reference scenarios with template associations",
			category:    "core",
		},
		{
			name:        "skill_connections",
			description: "Links skills to reference scenarios with version tracking",
			category:    "core",
		},
		{
			name:        "structural_expectations",
			description: "Defines expected folders, files, and content patterns",
			category:    "validation",
		},
		{
			name:        "cli_assertions",
			description: "Defines CLI tool commands and JSONPath assertions",
			category:    "validation",
		},
		{
			name:        "validation_runs",
			description: "Records validation run history",
			category:    "history",
		},
		{
			name:        "structural_results",
			description: "Records structural validation results",
			category:    "history",
		},
		{
			name:        "cli_results",
			description: "Records CLI assertion results",
			category:    "history",
		},
	}

	for _, tc := range requiredTables {
		t.Run(tc.name, func(t *testing.T) {
			// Match CREATE TABLE statement for this table
			pattern := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + tc.name + `\s*\(`)
			if !pattern.MatchString(schemaSQL) {
				t.Errorf("required table %q not found in schema.sql; %s", tc.name, tc.description)
			}
		})
	}
}

// TestReferenceScenarioColumns verifies the reference_scenarios table has all required columns.
func TestReferenceScenarioColumns(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	// Extract the CREATE TABLE statement for reference_scenarios
	tableSQL := extractTableDefinition(schemaSQL, "reference_scenarios")
	if tableSQL == "" {
		t.Fatal("could not extract reference_scenarios table definition")
	}

	requiredColumns := []struct {
		name       string
		dataType   string
		notNull    bool
		unique     bool
		primaryKey bool
		category   string
	}{
		{name: "id", dataType: "UUID", primaryKey: true, category: "identity"},
		{name: "slug", dataType: "VARCHAR", notNull: true, unique: true, category: "identity"},
		{name: "name", dataType: "VARCHAR", notNull: true, category: "core"},
		{name: "template", dataType: "VARCHAR", notNull: true, category: "core"},
		{name: "path", dataType: "VARCHAR", notNull: true, category: "core"},
		{name: "description", dataType: "TEXT", category: "optional"},
		{name: "created_at", dataType: "TIMESTAMP", category: "audit"},
		{name: "updated_at", dataType: "TIMESTAMP", category: "audit"},
	}

	for _, col := range requiredColumns {
		t.Run(col.name, func(t *testing.T) {
			// Check column exists with correct type (look for "column_name TYPE")
			pattern := regexp.MustCompile(`(?im)^\s*` + col.name + `\s+` + col.dataType)
			if !pattern.MatchString(tableSQL) {
				t.Errorf("column %q with type %s not found in reference_scenarios", col.name, col.dataType)
			}

			// Check NOT NULL constraint (look for "column_name ... NOT NULL" on same line)
			if col.notNull {
				notNullPattern := regexp.MustCompile(`(?im)^\s*` + col.name + `.*NOT\s+NULL`)
				if !notNullPattern.MatchString(tableSQL) {
					t.Errorf("column %q missing NOT NULL constraint", col.name)
				}
			}

			// Check UNIQUE constraint (look for "column_name ... UNIQUE" on same line)
			if col.unique {
				uniquePattern := regexp.MustCompile(`(?im)^\s*` + col.name + `.*UNIQUE`)
				if !uniquePattern.MatchString(tableSQL) {
					t.Errorf("column %q missing UNIQUE constraint", col.name)
				}
			}

			// Check PRIMARY KEY constraint (look for "column_name ... PRIMARY KEY" on same line)
			if col.primaryKey {
				pkPattern := regexp.MustCompile(`(?im)^\s*` + col.name + `.*PRIMARY\s+KEY`)
				if !pkPattern.MatchString(tableSQL) {
					t.Errorf("column %q missing PRIMARY KEY constraint", col.name)
				}
			}
		})
	}
}

// TestSkillConnectionColumns verifies the skill_connections table has all required columns.
func TestSkillConnectionColumns(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	tableSQL := extractTableDefinition(schemaSQL, "skill_connections")
	if tableSQL == "" {
		t.Fatal("could not extract skill_connections table definition")
	}

	requiredColumns := []struct {
		name     string
		dataType string
		category string
	}{
		{name: "id", dataType: "UUID", category: "identity"},
		{name: "reference_id", dataType: "UUID", category: "foreign_key"},
		{name: "skill_id", dataType: "VARCHAR", category: "core"},
		{name: "skill_version", dataType: "VARCHAR", category: "version"},
		{name: "skill_content_hash", dataType: "VARCHAR", category: "version"},
		{name: "connected_at", dataType: "TIMESTAMP", category: "audit"},
		{name: "updated_at", dataType: "TIMESTAMP", category: "audit"},
	}

	for _, col := range requiredColumns {
		t.Run(col.name, func(t *testing.T) {
			pattern := regexp.MustCompile(`(?i)` + col.name + `\s+` + col.dataType)
			if !pattern.MatchString(tableSQL) {
				t.Errorf("column %q with type %s not found in skill_connections", col.name, col.dataType)
			}
		})
	}
}

// TestSchemaHasRequiredIndexes verifies important indexes exist.
func TestSchemaHasRequiredIndexes(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	requiredIndexes := []struct {
		name     string
		table    string
		column   string
		category string
	}{
		{name: "idx_reference_scenarios_slug", table: "reference_scenarios", column: "slug", category: "lookup"},
		{name: "idx_reference_scenarios_template", table: "reference_scenarios", column: "template", category: "filter"},
		{name: "idx_skill_connections_reference", table: "skill_connections", column: "reference_id", category: "foreign_key"},
		{name: "idx_skill_connections_skill", table: "skill_connections", column: "skill_id", category: "lookup"},
	}

	for _, idx := range requiredIndexes {
		t.Run(idx.name, func(t *testing.T) {
			pattern := regexp.MustCompile(`(?i)CREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?` + idx.name)
			if !pattern.MatchString(schemaSQL) {
				t.Errorf("required index %q on %s(%s) not found", idx.name, idx.table, idx.column)
			}
		})
	}
}

// TestSchemaHasRequiredEnums verifies custom ENUM types exist.
func TestSchemaHasRequiredEnums(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	requiredEnums := []struct {
		name     string
		values   []string
		category string
	}{
		{
			name:     "expectation_type",
			values:   []string{"folder", "file", "content_snippet"},
			category: "structural",
		},
		{
			name:     "assertion_operator",
			values:   []string{"eq", "neq", "gt", "gte", "lt", "lte", "exists", "contains", "matches", "between"},
			category: "validation",
		},
		{
			name:     "validation_status",
			values:   []string{"pass", "fail", "error", "skip"},
			category: "results",
		},
	}

	for _, enum := range requiredEnums {
		t.Run(enum.name, func(t *testing.T) {
			// Check ENUM type exists
			pattern := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+` + enum.name + `\s+AS\s+ENUM`)
			if !pattern.MatchString(schemaSQL) {
				t.Errorf("required ENUM type %q not found", enum.name)
			}

			// Check all values exist
			for _, val := range enum.values {
				valPattern := regexp.MustCompile(`'` + val + `'`)
				if !valPattern.MatchString(schemaSQL) {
					t.Errorf("ENUM %q missing value %q", enum.name, val)
				}
			}
		})
	}
}

// TestSchemaHasForeignKeyConstraints verifies foreign key relationships.
func TestSchemaHasForeignKeyConstraints(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	foreignKeys := []struct {
		table      string
		column     string
		refTable   string
		onDelete   string
		category   string
	}{
		{table: "skill_connections", column: "reference_id", refTable: "reference_scenarios", onDelete: "CASCADE", category: "cascade"},
		{table: "structural_expectations", column: "connection_id", refTable: "skill_connections", onDelete: "CASCADE", category: "cascade"},
		{table: "cli_assertions", column: "connection_id", refTable: "skill_connections", onDelete: "CASCADE", category: "cascade"},
		{table: "validation_runs", column: "reference_id", refTable: "reference_scenarios", onDelete: "CASCADE", category: "cascade"},
	}

	for _, fk := range foreignKeys {
		t.Run(fk.table+"_"+fk.column, func(t *testing.T) {
			// Check REFERENCES clause exists
			pattern := regexp.MustCompile(`(?i)` + fk.column + `[^,)]+REFERENCES\s+` + fk.refTable)
			if !pattern.MatchString(schemaSQL) {
				t.Errorf("foreign key %s.%s -> %s not found", fk.table, fk.column, fk.refTable)
			}

			// Check ON DELETE action - look within the same line as the REFERENCES
			// The constraint may span multiple characters between the column and ON DELETE
			onDeletePattern := regexp.MustCompile(`(?is)` + fk.column + `[^;]+REFERENCES\s+` + fk.refTable + `[^;]+ON\s+DELETE\s+` + fk.onDelete)
			if !onDeletePattern.MatchString(schemaSQL) {
				t.Errorf("foreign key %s.%s missing ON DELETE %s", fk.table, fk.column, fk.onDelete)
			}
		})
	}
}

// TestSchemaHasUpdateTriggers verifies timestamp triggers exist.
func TestSchemaHasUpdateTriggers(t *testing.T) {
	schemaSQL, err := loadSchemaSQL(t)
	if err != nil {
		t.Fatalf("failed to load schema.sql: %v", err)
	}

	// Check the trigger function exists
	funcPattern := regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+update_updated_at`)
	if !funcPattern.MatchString(schemaSQL) {
		t.Error("update_updated_at trigger function not found")
	}

	triggers := []struct {
		name     string
		table    string
		category string
	}{
		{name: "reference_scenarios_updated_at", table: "reference_scenarios", category: "audit"},
		{name: "skill_connections_updated_at", table: "skill_connections", category: "audit"},
	}

	for _, trigger := range triggers {
		t.Run(trigger.name, func(t *testing.T) {
			pattern := regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+` + trigger.name)
			if !pattern.MatchString(schemaSQL) {
				t.Errorf("trigger %q for %s not found", trigger.name, trigger.table)
			}
		})
	}
}

// loadSchemaSQL loads the schema.sql file content for testing.
func loadSchemaSQL(t *testing.T) (string, error) {
	t.Helper()

	// Try multiple possible paths to find the schema file
	possiblePaths := []string{
		"../../../initialization/postgres/schema.sql",
		"../../../../initialization/postgres/schema.sql",
		"initialization/postgres/schema.sql",
	}

	// Also try from the scenario root
	if cwd, err := os.Getwd(); err == nil {
		// Navigate up to find the scenario root
		for i := 0; i < 5; i++ {
			schemaPath := filepath.Join(cwd, "initialization/postgres/schema.sql")
			if _, err := os.Stat(schemaPath); err == nil {
				possiblePaths = append([]string{schemaPath}, possiblePaths...)
				break
			}
			cwd = filepath.Dir(cwd)
		}
	}

	for _, path := range possiblePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
	}

	return "", os.ErrNotExist
}

// extractTableDefinition extracts a CREATE TABLE statement from the schema.
func extractTableDefinition(schemaSQL, tableName string) string {
	// Match from CREATE TABLE to the closing );
	pattern := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + tableName + `\s*\([^;]+\);`)
	match := pattern.FindString(schemaSQL)
	return match
}
