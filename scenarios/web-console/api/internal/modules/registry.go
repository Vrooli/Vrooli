// Package modules is the single registration point for the scenario's
// migrated API modules' static metadata. Both api/main.go (eventually)
// and api/cmd/gen-endpoints/main.go import this package to enumerate
// migrated domains uniformly.
//
// Web Console is mid-migration to the proto + Connect-RPC contract.
// During the migration window, only domains that have been moved out of
// the flat api/ package and into handlers/<domain>/ appear here. The
// remaining HandleFunc routes in main.go are not registered through
// this registry; they continue to work but are not validated by
// cmd/gen-endpoints.
//
// Adding a migrated domain: add one line to AllEndpoints below for the
// domain's exported Endpoints slice. When the domain owns a proto
// service (i.e. it is a true Connect-RPC domain rather than a tagged
// REST exception), also add an entry to AllProtoFiles so the parity
// test catches missing handler registrations.
package modules

import (
	"web-console/internal/module"

	aiH "web-console/handlers/ai"
	capabilitiesH "web-console/handlers/capabilities"
	conversationH "web-console/handlers/conversation"
	eventsH "web-console/handlers/events"
	healthH "web-console/handlers/health"
	hooksH "web-console/handlers/hooks"
	metricsH "web-console/handlers/metrics"
	sessionsH "web-console/handlers/sessions"
	settingsH "web-console/handlers/settings"
	shortcutsH "web-console/handlers/shortcuts"
	terminalH "web-console/handlers/terminal"
	workspaceH "web-console/handlers/workspace"
)

// AllEndpoints returns every migrated domain's static endpoint
// descriptors in a stable order (system endpoints first, then domains
// alphabetically). The stable order is what makes the diff-exit-code
// CI check on .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, aiH.Endpoints...)
	out = append(out, capabilitiesH.Endpoints...)
	out = append(out, conversationH.Endpoints...)
	out = append(out, eventsH.Endpoints...)
	out = append(out, hooksH.Endpoints...)
	out = append(out, metricsH.Endpoints...)
	out = append(out, sessionsH.Endpoints...)
	out = append(out, settingsH.Endpoints...)
	out = append(out, shortcutsH.Endpoints...)
	out = append(out, terminalH.Endpoints...)
	out = append(out, workspaceH.Endpoints...)
	return out
}
