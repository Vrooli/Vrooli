// Package modules is the explicit registry of Agent Manager's domain-owned
// SQLite schemas. Registration order follows foreign-key dependencies.
package modules

import (
	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/findings"
	"agent-manager/internal/modelhealth"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/policy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runmodel"
	"agent-manager/internal/runnerhealth"
	"agent-manager/internal/stats"
	"agent-manager/internal/workflowruntime"

	"github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoFileEntry connects a mounted Connect domain to its generated source
// descriptor. It is intentionally explicit so parity tests can prevent
// orphaned proto methods as domains are added.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles lists every Agent Manager domain proto that owns a served
// Connect surface. New proto domains must be registered here.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{{Module: "episodes", File: domainpb.File_agent_manager_v1_domain_episode_proto}}
}

func AllSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(domain.Schema),
		database.SchemaProviderFunc(artifact.Schema),
		database.SchemaProviderFunc(workflowruntime.Schema),
		database.SchemaProviderFunc(runmodel.Schema),
		database.SchemaProviderFunc(eventlog.Schema),
		database.SchemaProviderFunc(findings.Schema),
		database.SchemaProviderFunc(policy.Schema),
		database.SchemaProviderFunc(permissionpolicy.Schema),
		database.SchemaProviderFunc(pricing.Schema),
		database.SchemaProviderFunc(modelhealth.Schema),
		database.SchemaProviderFunc(runnerhealth.Schema),
		database.SchemaProviderFunc(stats.Schema),
	}
}
