// Package dbschema is the single ordering point for Ecosystem Manager's SQLite
// schema providers. Each domain package owns its schema.sql + Schema() function;
// this registry only enumerates them (in dependency order) so server bootstrap
// can apply them all with one database.EnsureSchemas call.
//
// Adding a domain: add one line to AllSchemas. There is no other central
// registry mutation — the schema text lives next to the code that reads it.
package dbschema

import (
	"github.com/vrooli/api-core/database"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/steering"
)

// AllSchemas returns every domain's SQLite schema provider in a stable order.
// The order is not load-bearing today (no cross-table foreign keys span
// domains), but is kept deterministic so EnsureSchemas behaviour is reproducible.
func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(autosteer.Schema),
		database.SchemaProviderFunc(steering.Schema),
	}
}
