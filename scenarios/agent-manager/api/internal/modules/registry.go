// Package modules is the explicit registry of Agent Manager's domain-owned
// SQLite schemas. Registration order follows foreign-key dependencies.
package modules

import (
	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/modelhealth"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/policy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runmodel"
	"agent-manager/internal/runnerhealth"
	"agent-manager/internal/stats"
	"agent-manager/internal/workflowruntime"

	"github.com/vrooli/api-core/database"
)

func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(domain.Schema),
		database.SchemaProviderFunc(artifact.Schema),
		database.SchemaProviderFunc(workflowruntime.Schema),
		database.SchemaProviderFunc(runmodel.Schema),
		database.SchemaProviderFunc(eventlog.Schema),
		database.SchemaProviderFunc(policy.Schema),
		database.SchemaProviderFunc(permissionpolicy.Schema),
		database.SchemaProviderFunc(pricing.Schema),
		database.SchemaProviderFunc(modelhealth.Schema),
		database.SchemaProviderFunc(runnerhealth.Schema),
		database.SchemaProviderFunc(stats.Schema),
	}
}
