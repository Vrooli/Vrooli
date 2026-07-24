package modules

import (
	localdb "prompt-manager/internal/database"
	internalskills "prompt-manager/internal/skills"
	internaltags "prompt-manager/internal/tags"
	internaltesting "prompt-manager/internal/testing"
	"prompt-manager/store"

	"github.com/vrooli/api-core/database"
)

func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(internalskills.Schema),
		database.SchemaProviderFunc(internaltags.Schema),
		database.SchemaProviderFunc(internaltesting.Schema),
		database.SchemaProviderFunc(store.ExperimentSchema),
	}
}
