package modules

import (
	localdb "prompt-manager/internal/database"
	"prompt-manager/internal/skills"
	"prompt-manager/internal/store"
	"prompt-manager/internal/tags"
	"prompt-manager/internal/testing"

	"github.com/vrooli/api-core/database"
)

func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(skills.Schema),
		database.SchemaProviderFunc(tags.Schema),
		database.SchemaProviderFunc(testing.Schema),
		database.SchemaProviderFunc(store.ExperimentSchema),
	}
}
