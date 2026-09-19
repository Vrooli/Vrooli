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
	"persona/internal/module"

	accessH "persona/handlers/access"
	accountsH "persona/handlers/accounts"
	capsH "persona/handlers/capabilities"
	channelsH "persona/handlers/channels"
	documentsH "persona/handlers/documents"
	handoffsH "persona/handlers/handoffs"
	journalH "persona/handlers/journal"
	personasH "persona/handlers/personas"
	"persona/internal/access"
	"persona/internal/accounts"
	"persona/internal/channels"
	"persona/internal/documents"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/personas"

	apidb "github.com/vrooli/api-core/database"
	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/accounts"
	channelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/channels"
	documentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents"
	handoffsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/handoffs"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/journal"
	personasv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/personas"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "persona/handlers/health"
	localdb "persona/internal/database"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, accessH.Endpoints...)
	out = append(out, accountsH.Endpoints...)
	out = append(out, channelsH.Endpoints...)
	out = append(out, documentsH.Endpoints...)
	out = append(out, handoffsH.Endpoints...)
	out = append(out, personasH.Endpoints...)
	out = append(out, journalH.Endpoints...)
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
		{Module: "access", File: accessv1.File_persona_v1_access_access_proto},
		{Module: "accounts", File: accountsv1.File_persona_v1_accounts_accounts_proto},
		{Module: "channels", File: channelsv1.File_persona_v1_channels_channels_proto},
		{Module: "documents", File: documentsv1.File_persona_v1_documents_documents_proto},
		{Module: "handoffs", File: handoffsv1.File_persona_v1_handoffs_handoffs_proto},
		{Module: "personas", File: personasv1.File_persona_v1_personas_personas_proto},
		{Module: "journal", File: journalv1.File_persona_v1_journal_journal_proto},
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
		apidb.SchemaProviderFunc(personas.Schema),
		apidb.SchemaProviderFunc(access.Schema),
		apidb.SchemaProviderFunc(channels.Schema),
		apidb.SchemaProviderFunc(handoffs.Schema),
		apidb.SchemaProviderFunc(documents.Schema),
		apidb.SchemaProviderFunc(journal.Schema),
		apidb.SchemaProviderFunc(accounts.Schema),
	}
}
