// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call — they need live deps (db handle, clock, logger)
// and abstracting them is needless ceremony. This package only handles
// the static side: the Endpoints slice each handler exports for
// codegen, and the Schema() function each handler re-exports for
// EnsureSchemas.
//
// Adding a domain: add two lines below — one in AllEndpoints, one in
// AllSchemas. The runtime constructor lands in main.go's server.New
// call as a third line. Three central lines per new domain, no other
// central registry mutations.
package modules

import (
	"device-studio/internal/module"

	capsH "device-studio/handlers/capabilities"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	devicesH "device-studio/handlers/devices"
	healthH "device-studio/handlers/health"
	libraryH "device-studio/handlers/library"
	motionsH "device-studio/handlers/motions"
	photographicH "device-studio/handlers/photographic"
	physicalruntimeH "device-studio/handlers/physicalruntime"
	preparedphotoH "device-studio/handlers/preparedphoto"
	projectsH "device-studio/handlers/projects"
	renderjobsH "device-studio/handlers/renderjobs"
	scenesH "device-studio/handlers/scenes"
	shotsH "device-studio/handlers/shots"
	localdb "device-studio/internal/database"

	devicesdomain "device-studio/internal/devices"
	librarydomain "device-studio/internal/library"
	motionsdomain "device-studio/internal/motions"
	scenesdomain "device-studio/internal/scenes"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	motionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	scenesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, projectsH.Endpoints...)
	out = append(out, photographicH.Endpoints...)
	out = append(out, preparedphotoH.Endpoints...)
	out = append(out, devicesH.Endpoints...)
	out = append(out, scenesH.Endpoints...)
	out = append(out, motionsH.Endpoints...)
	out = append(out, libraryH.Endpoints...)
	out = append(out, physicalruntimeH.Endpoints...)
	out = append(out, shotsH.Endpoints...)
	out = append(out, renderjobsH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test in registry_test.go walks every entry and asserts
// each rpc method in the FileDescriptor has exactly one matching
// EndpointDescriptor in AllEndpoints().
//
// Adding a Connect-RPC domain: append one line below. The global parity
// test then covers it automatically — there is no per-domain parity
// test to write.
//
// REST-exception-only domains (none in the template today) are simply
// not listed here; the global test never inspects them, and the
// gen-endpoints validateTransport pass enforces their RESTException
// tags at codegen time.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "projects", File: projectsv1.File_device_studio_v1_projects_projects_proto},
		{Module: "devices", File: devicesv1.File_device_studio_v1_devices_devices_proto},
		{Module: "scenes", File: scenesv1.File_device_studio_v1_scenes_scenes_proto},
		{Module: "motions", File: motionsv1.File_device_studio_v1_motions_motions_proto},
		{Module: "library", File: libraryv1.File_device_studio_v1_library_library_proto},
		{Module: "shots", File: shotsv1.File_device_studio_v1_shots_shots_proto},
		{Module: "physical", File: physicalv1.File_device_studio_v1_physical_physical_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → … (domains alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(projectsH.Schema),
		apidb.SchemaProviderFunc(librarydomain.Schema),
		apidb.SchemaProviderFunc(devicesdomain.Schema),
		apidb.SchemaProviderFunc(scenesdomain.Schema),
		apidb.SchemaProviderFunc(motionsdomain.Schema),
		apidb.SchemaProviderFunc(shotsH.Schema),
		apidb.SchemaProviderFunc(renderjobsH.Schema),
	}
}
