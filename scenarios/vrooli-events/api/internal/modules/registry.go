// Package modules is the single registration point for the event store's
// durable schemas. Keeping the list here lets the lifecycle install the same
// schemas into the primary pool and every Test Genie routed pool.
package modules

import (
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"

	apidb "github.com/vrooli/api-core/database"
)

// AllSchemas returns every durable domain schema in dependency order.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(store.Schema),
		apidb.SchemaProviderFunc(policy.Schema),
		apidb.SchemaProviderFunc(subscription.Schema),
	}
}
