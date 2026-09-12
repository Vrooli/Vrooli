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
	"content-desk/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	artifactsH "content-desk/handlers/artifacts"
	campaignsH "content-desk/handlers/campaigns"
	claimsH "content-desk/handlers/claims"
	healthH "content-desk/handlers/health"
	ledgerH "content-desk/handlers/ledger"
	posttypesH "content-desk/handlers/posttypes"
	reviewH "content-desk/handlers/review"
	localdb "content-desk/internal/database"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	campaignsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/campaigns"
	claimsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/claims"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	posttypesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/posttypes"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, artifactsH.Endpoints...)
	out = append(out, campaignsH.Endpoints...)
	out = append(out, claimsH.Endpoints...)
	out = append(out, ledgerH.Endpoints...)
	out = append(out, posttypesH.Endpoints...)
	out = append(out, reviewH.Endpoints...)
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
		{Module: "artifacts", File: artifactsv1.File_content_desk_v1_artifacts_artifacts_proto},
		{Module: "campaigns", File: campaignsv1.File_content_desk_v1_campaigns_campaigns_proto},
		{Module: "claims", File: claimsv1.File_content_desk_v1_claims_claims_proto},
		{Module: "ledger", File: ledgerv1.File_content_desk_v1_ledger_ledger_proto},
		{Module: "posttypes", File: posttypesv1.File_content_desk_v1_posttypes_posttypes_proto},
		{Module: "review", File: reviewv1.File_content_desk_v1_review_review_proto},
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
		apidb.SchemaProviderFunc(artifactsH.Schema),
		apidb.SchemaProviderFunc(campaignsH.Schema),
		apidb.SchemaProviderFunc(claimsH.Schema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(ledgerH.Schema),
		apidb.SchemaProviderFunc(posttypesH.Schema),
		apidb.SchemaProviderFunc(reviewH.Schema),
	}
}
