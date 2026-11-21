// Package app is the single registration point for the scenario's API
// modules' static metadata (package module holds the Module seam itself). Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call because they need live deps. This package holds
// the static side: each handler's Endpoints for codegen and each Connect
// domain's proto file for the parity test.
//
// Adding a domain: one line in AllEndpoints, one in AllProtoFiles, and the
// constructor in main.go.
package app

import (
	"research-desk/internal/module"

	"research-desk/handlers/activityapi"
	"research-desk/handlers/capabilitiesapi"

	"research-desk/handlers/healthapi"
	"research-desk/handlers/launchapi"
	"research-desk/handlers/operatorapi"

	"google.golang.org/protobuf/reflect/protoreflect"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthapi.Endpoints...)
	out = append(out, capabilitiesapi.Endpoints...)
	out = append(out, activityapi.Endpoints...)
	out = append(out, launchapi.Endpoints...)
	out = append(out, operatorapi.Endpoints...)
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
// REST-exception routes (health, capabilities, the home file route) are not listed here;
// gen-endpoints enforces their RESTException tags.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "activity", File: activityv1.File_research_desk_v1_activity_activity_proto},
		{Module: "launch", File: launchv1.File_research_desk_v1_launch_launch_proto},
		{Module: "operator", File: operatorv1.File_research_desk_v1_operator_operator_proto},
	}
}
