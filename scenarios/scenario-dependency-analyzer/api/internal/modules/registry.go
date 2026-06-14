package modules

import (
	localdb "scenario-dependency-analyzer/internal/database"
	"scenario-dependency-analyzer/internal/store"

	apidb "github.com/vrooli/api-core/database"
)

// AllSchemas returns every SQL schema provider in stable boot order.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(store.Schema),
	}
}
