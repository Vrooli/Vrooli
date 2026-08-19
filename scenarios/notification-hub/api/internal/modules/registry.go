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
	"notification-hub/internal/module"

	capsH "notification-hub/handlers/capabilities"
	conversationH "notification-hub/handlers/conversations"
	deliveryH "notification-hub/handlers/delivery"
	notificationH "notification-hub/handlers/notifications"
	recipientsH "notification-hub/handlers/recipients"
	routingH "notification-hub/handlers/routing"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	deliveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery"
	notificationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	recipientsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing"
	healthH "notification-hub/handlers/health"
	conversationSchema "notification-hub/internal/conversations"
	localdb "notification-hub/internal/database"
	deliverySchema "notification-hub/internal/delivery"
	notificationSchema "notification-hub/internal/notifications"
	recipientSchema "notification-hub/internal/recipients"
	routingSchema "notification-hub/internal/routing"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, conversationH.Endpoints...)
	out = append(out, deliveryH.Endpoints...)
	out = append(out, notificationH.Endpoints...)
	out = append(out, recipientsH.Endpoints...)
	out = append(out, routingH.Endpoints...)
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
		{Module: "conversations", File: conversationv1.File_notification_hub_v1_conversations_conversations_proto},
		{Module: "delivery", File: deliveryv1.File_notification_hub_v1_delivery_delivery_proto},
		{Module: "notifications", File: notificationv1.File_notification_hub_v1_notifications_notifications_proto},
		{Module: "recipients", File: recipientsv1.File_notification_hub_v1_recipients_recipients_proto},
		{Module: "routing", File: routingv1.File_notification_hub_v1_routing_routing_proto},
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
		apidb.SchemaProviderFunc(recipientSchema.Schema),
		apidb.SchemaProviderFunc(notificationSchema.Schema),
		apidb.SchemaProviderFunc(routingSchema.Schema),
		apidb.SchemaProviderFunc(deliverySchema.Schema),
		apidb.SchemaProviderFunc(conversationSchema.Schema),
	}
}
