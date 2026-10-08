package modules

import (
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
)

// AllSchemas is the explicit boot registry for per-domain schemas.
func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{database.SchemaProviderFunc(investigations.Schema)}
}
