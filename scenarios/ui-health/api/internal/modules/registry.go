// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate domains
// uniformly.
package modules

import (
	"ui-health/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "ui-health/handlers/health"
	reindexH "ui-health/handlers/reindex"
	searchH "ui-health/handlers/search"
	validationH "ui-health/handlers/validation"
	visualhealthH "ui-health/handlers/visualhealth"
	localdb "ui-health/internal/database"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
	visualhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, reindexH.Endpoints...)
	out = append(out, searchH.Endpoints...)
	out = append(out, validationH.Endpoints...)
	out = append(out, visualhealthH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
	// Services limits parity validation to services this module actually mounts.
	// A shared proto file may declare services owned by another scenario.
	Services []protoreflect.Name
}

func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "reindex", File: reindexv1.File_ui_health_v1_reindex_reindex_proto},
		{Module: "search", File: searchv1.File_ui_health_v1_search_search_proto},
		{Module: "validation", File: validationH.ProtoFile, Services: []protoreflect.Name{"ScenarioValidationService"}},
		{Module: "visualhealth", File: visualhealthv1.File_ui_health_v1_visualhealth_visualhealth_proto},
	}
}

func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
	}
}
