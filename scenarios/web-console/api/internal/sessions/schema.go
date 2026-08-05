package sessions

import _ "embed"

//go:embed schema.sql
var schemaSQL string

//go:embed seed.sql
var seedSQL string

//go:embed agent_type_migration.sql
var agentTypeMigrationSQL string

// Schema returns the session and workspace schema.
func Schema() string { return schemaSQL }

// Seed returns the default seed data for a new database.
func Seed() string { return seedSQL }

// AgentTypeMigrationSchema returns the table definition used when rebuilding
// a legacy sessions table whose agent_type CHECK constraint is too narrow.
func AgentTypeMigrationSchema() string { return agentTypeMigrationSQL }
