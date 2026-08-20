package modules

import (
	internaldatabase "deployment-manager/internal/database"
	internaldeployments "deployment-manager/internal/deployments"
	internalevidence "deployment-manager/internal/evidence"
	internalprofiles "deployment-manager/internal/profiles"
	internalreleases "deployment-manager/internal/releases"

	coredb "github.com/vrooli/api-core/database"
)

// AllSchemas is the single boot-time schema registry. Empty system SQL is kept
// in the registry to make cross-domain ownership explicit.
func AllSchemas() []coredb.SchemaProvider {
	return []coredb.SchemaProvider{
		coredb.SchemaProviderFunc(internaldatabase.Schema),
		coredb.SchemaProviderFunc(internalprofiles.Schema),
		coredb.SchemaProviderFunc(internaldeployments.Schema),
		coredb.SchemaProviderFunc(internalreleases.Schema),
		coredb.SchemaProviderFunc(internalevidence.Schema),
	}
}
