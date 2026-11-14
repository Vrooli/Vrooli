// Package modules is the single registration point for this scenario's storage
// domains.
//
// Adding a domain is one line in AllSchemas plus one folder under
// internal/<domain>/. Removing a domain is deleting that folder plus the one
// line here — no central schema file has to be edited, and no orphaned table
// survives the removal.
package modules

import (
	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/alerts"
	localdb "knowledge-observatory/internal/database"
	"knowledge-observatory/internal/deepsearch"
	"knowledge-observatory/internal/docaccess"
	"knowledge-observatory/internal/dochealing"
	"knowledge-observatory/internal/graph"
	"knowledge-observatory/internal/ingest"
	"knowledge-observatory/internal/metadata"
	"knowledge-observatory/internal/preferences"
	"knowledge-observatory/internal/quality"
	"knowledge-observatory/internal/search"
)

// AllSchemas returns the system schema first, then every domain in a stable
// alphabetical order. Consumed by database.EnsureSchemas at boot.
//
// The system home runs first because cross-cutting objects (on PostgreSQL, the
// schema namespace and the shared trigger function) must exist before any
// domain references them.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(alerts.Schema),
		apidb.SchemaProviderFunc(deepsearch.Schema),
		apidb.SchemaProviderFunc(docaccess.Schema),
		apidb.SchemaProviderFunc(dochealing.Schema),
		apidb.SchemaProviderFunc(graph.Schema),
		apidb.SchemaProviderFunc(ingest.Schema),
		apidb.SchemaProviderFunc(metadata.Schema),
		apidb.SchemaProviderFunc(preferences.Schema),
		apidb.SchemaProviderFunc(quality.Schema),
		apidb.SchemaProviderFunc(search.Schema),
	}
}
