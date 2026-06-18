package modules

import (
	localdb "prompt-manager/internal/database"
	"prompt-manager/metrics"
	"prompt-manager/tags"
	"prompt-manager/testing"

	"github.com/vrooli/api-core/database"
)

func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(metrics.Schema),
		database.SchemaProviderFunc(tags.Schema),
		database.SchemaProviderFunc(testing.Schema),
	}
}
