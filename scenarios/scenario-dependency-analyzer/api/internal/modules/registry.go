package modules

import (
	localdb "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/database"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/store"

	apidb "github.com/vrooli/api-core/database"
)

// AllSchemas returns every SQL schema provider in stable boot order.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(store.Schema),
	}
}
