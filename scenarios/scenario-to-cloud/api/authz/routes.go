// Package authz is the single management boundary of the scenario-to-cloud
// API: every registered route is classified once in the table below, and the
// enforcement middleware refuses anything the table does not admit. The
// vocabulary is the shared coarse scope grammar (<scenario>:<effect>) from
// api-core/scopecatalog; the finer effect classes here only decide which
// coarse scope a route needs and which extra boundaries (target binding,
// browser origin, service-principal exclusion, no-store) apply.
package authz

import (
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge/edgev1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence/evidencev1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health/healthv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations/operationsv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans/plansv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/releases/releasesv1connect"
)

// Scenario is the scope namespace of this service. Scope values are derived
// from it so the catalog grammar and this table can never drift apart.
const Scenario = "scenario-to-cloud"

// Coarse scopes in the shared catalog vocabulary.
const (
	ScopeRead        = Scenario + ":read"
	ScopeWrite       = Scenario + ":write"
	ScopeDestructive = Scenario + ":destructive"
)

// Scopes lists every scope this service understands, most restrictive last.
func Scopes() []string { return []string{ScopeRead, ScopeWrite, ScopeDestructive} }

// Effect is the management effect class of a route.
type Effect string

const (
	// EffectPublic routes disclose nothing target-specific and need no principal.
	EffectPublic Effect = "public"
	// EffectRead observes local or remote state without changing it.
	EffectRead Effect = "read"
	// EffectWorkloadMutation changes deployment records or runs a workload
	// lifecycle step (execute, stop, start, delete, tasks).
	EffectWorkloadMutation Effect = "workload_mutation"
	// EffectSecret reads or changes credential material or its metadata.
	EffectSecret Effect = "secret"
	// EffectHost repairs or reconfigures a host (processes, firewall, disk,
	// edge, bundles on the target, local instances).
	EffectHost Effect = "host"
	// EffectInteractive opens a long-lived interactive session on a target.
	EffectInteractive Effect = "interactive"
)

// Route is one classified management entry point.
type Route struct {
	// Method is the HTTP method; "*" for a prefix mount that serves any method.
	Method string `json:"method"`
	// Path is the gorilla/mux template (or the prefix for mounts).
	Path string `json:"path"`
	// Prefix marks a PathPrefix mount (Connect service, dev routing).
	Prefix bool `json:"prefix,omitempty"`
	// Effect is the management effect class.
	Effect Effect `json:"effect"`
	// Scope is the required coarse scope; empty only for public routes.
	Scope string `json:"scope,omitempty"`
	// TargetBound routes carry a deployment id in {id}; the deployment and its
	// target are resolved and policy-checked before the handler runs.
	TargetBound bool `json:"target_bound"`
	// BodyTarget routes address a target inside the request body (legacy
	// manifest-shaped calls). The boundary cannot resolve that target before
	// the handler, so only unrestricted principals may use them.
	BodyTarget bool `json:"body_target,omitempty"`
	// RemoteReach routes open SSH or other reach to a target.
	RemoteReach bool `json:"remote_reach,omitempty"`
	// ServiceAllowed rows may be called by service principals (and verified
	// agents); everything else is human-only.
	ServiceAllowed bool `json:"service_allowed"`
	// NoStore responses carry credential or identity material and must never
	// be cached.
	NoStore bool `json:"no_store,omitempty"`
	// Optional rows are registered only in some runtimes (development-only
	// mounts) and are not required to exist on the live router.
	Optional bool `json:"optional,omitempty"`
	// Description is the operator-facing explanation used by the matrix doc.
	Description string `json:"description"`
}

// Effectful reports whether the route changes state anywhere.
func (r Route) Effectful() bool {
	switch r.Effect {
	case EffectPublic, EffectRead:
		return false
	default:
		return true
	}
}

// Key is the lookup key used by the enforcer and the census.
func (r Route) Key() string { return Key(r.Method, r.Path) }

// Key builds the route key from a method and a path template.
func Key(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

// DeploymentsServicePrefix is the Connect mount of the typed DeploymentsService.
const DeploymentsServicePrefix = "/" + deploymentsv1connect.DeploymentsServiceName + "/"

// HealthServicePrefix is the Connect mount of the typed HealthService (P16).
const HealthServicePrefix = "/" + healthv1connect.HealthServiceName + "/"

// ReleasesServicePrefix is the Connect mount of the typed ReleasesService.
const ReleasesServicePrefix = "/" + releasesv1connect.ReleasesServiceName + "/"

// OperationsServicePrefix is the Connect mount of the typed OperationsService (P07).
const OperationsServicePrefix = "/" + operationsv1connect.OperationsServiceName + "/"

// EvidenceServicePrefix is the Connect mount of the typed EvidenceService (P17).
const EvidenceServicePrefix = "/" + evidencev1connect.EvidenceServiceName + "/"

// PlansServicePrefix is the Connect mount of the typed PlansService (P06).
const PlansServicePrefix = "/" + plansv1connect.PlansServiceName + "/"

// EdgeServicePrefix is the Connect mount of the typed EdgeService (P14).
const EdgeServicePrefix = "/" + edgev1connect.EdgeServiceName + "/"

// DevRoutingServicePrefix is the development-only Test Genie routing mount
// registered by api-core/devrouting. It is absent in production.
const DevRoutingServicePrefix = "/vrooli.dev_routing.v1.routing.RoutingService/"

// PublicAllowlist is the only set of routes that may be classified public.
// Both health routes serve the api-core health handler, which reports service
// name, readiness, uptime and database connectivity/latency; it carries no
// deployment, target, host or credential data, so the metadata is safe to
// keep public for infrastructure probes.
var PublicAllowlist = map[string]struct{}{
	Key("GET", "/health"):        {},
	Key("GET", "/api/v1/health"): {},
}

const api = "/api/v1"

// Table is the complete route census. The census test walks the live router
// and fails when a registered route is missing here.
var Table = []Route{
	{Method: "GET", Path: "/health", Effect: EffectPublic, ServiceAllowed: true, Description: "Liveness metadata for infrastructure probes."},
	{Method: "GET", Path: api + "/health", Effect: EffectPublic, ServiceAllowed: true, Description: "Liveness metadata for clients."},
	{Method: "GET", Path: api + "/authz/matrix", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "This authorization matrix, for UI/CLI parity."},

	// Local scenario discovery.
	{Method: "GET", Path: api + "/scenarios", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "List deployable scenarios on this host."},
	{Method: "GET", Path: api + "/scenarios/{id}/ports", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Port declarations of one scenario."},
	{Method: "GET", Path: api + "/scenarios/{id}/dependencies", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Dependency closure of one scenario."},
	{Method: "POST", Path: api + "/validate/reachability", Effect: EffectRead, Scope: ScopeRead, Description: "Probe reachability of an operator-supplied address."},

	// Manifest authoring (local computation only).
	{Method: "GET", Path: api + "/manifest/schema", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Manifest JSON schema."},
	{Method: "GET", Path: api + "/manifest/template", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Manifest template."},
	{Method: "POST", Path: api + "/manifest/init", Effect: EffectRead, Scope: ScopeRead, Description: "Derive a manifest from a scenario (reads local secrets metadata)."},
	{Method: "POST", Path: api + "/manifest/doctor", Effect: EffectRead, Scope: ScopeRead, Description: "Diagnose a manifest."},
	{Method: "POST", Path: api + "/manifest/fix", Effect: EffectRead, Scope: ScopeRead, Description: "Return a corrected manifest."},
	{Method: "POST", Path: api + "/manifest/validate", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Validate and normalize a manifest."},

	// Local bundle store.
	{Method: "POST", Path: api + "/bundle/build", Effect: EffectWorkloadMutation, Scope: ScopeWrite, Description: "Build a bundle on this host."},
	{Method: "GET", Path: api + "/bundles", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "List local bundles."},
	{Method: "GET", Path: api + "/bundles/stats", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Local bundle store statistics."},
	{Method: "POST", Path: api + "/bundles/cleanup", Effect: EffectHost, Scope: ScopeDestructive, Description: "Delete local bundles."},
	{Method: "DELETE", Path: api + "/bundles/{sha256}", Effect: EffectHost, Scope: ScopeDestructive, Description: "Delete one local bundle."},
	{Method: "GET", Path: api + "/deployments/{id}/bundles/vps", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "List bundles cached on the deployment target."},
	{Method: "POST", Path: api + "/deployments/{id}/bundles/vps/gc", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Garbage-collect bundles on the deployment target."},

	// Preflight and host repair (legacy body-addressed target).
	{Method: "POST", Path: api + "/preflight", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, RemoteReach: true, Description: "Run preflight checks against a target named in the body."},
	{Method: "GET", Path: api + "/preflight/requirements", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Preflight requirement catalog."},
	{Method: "POST", Path: api + "/preflight/fix/firewall", Effect: EffectHost, Scope: ScopeDestructive, BodyTarget: true, RemoteReach: true, Description: "Open firewall ports on a target."},
	{Method: "POST", Path: api + "/preflight/fix/stop-processes", Effect: EffectHost, Scope: ScopeDestructive, BodyTarget: true, RemoteReach: true, Description: "Stop scenario processes on a target."},
	{Method: "POST", Path: api + "/preflight/disk/usage", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, RemoteReach: true, Description: "Read disk usage on a target."},
	{Method: "POST", Path: api + "/preflight/disk/cleanup", Effect: EffectHost, Scope: ScopeDestructive, BodyTarget: true, RemoteReach: true, Description: "Free disk space on a target."},

	// Credential material.
	{Method: "GET", Path: api + "/secrets/{scenario}", Effect: EffectSecret, Scope: ScopeDestructive, NoStore: true, Description: "Fetch bundle secret values for a scenario."},

	// Legacy VPS operations (target addressed in the manifest body).
	{Method: "POST", Path: api + "/vps/setup/plan", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, Description: "Plan a VPS setup."},
	{Method: "POST", Path: api + "/vps/setup/apply", Effect: EffectHost, Scope: ScopeDestructive, BodyTarget: true, RemoteReach: true, Description: "Apply a VPS setup."},
	{Method: "POST", Path: api + "/vps/deploy/plan", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, Description: "Plan a VPS deploy."},
	{Method: "POST", Path: api + "/vps/deploy/apply", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, BodyTarget: true, RemoteReach: true, Description: "Deploy a bundle to a VPS."},
	{Method: "POST", Path: api + "/vps/inspect/plan", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, Description: "Plan a VPS inspection."},
	{Method: "POST", Path: api + "/vps/inspect/apply", Effect: EffectRead, Scope: ScopeRead, BodyTarget: true, RemoteReach: true, Description: "Inspect a VPS."},

	// Local disposable instances.
	{Method: "POST", Path: api + "/instances/plan", Effect: EffectRead, Scope: ScopeRead, Description: "Plan a local disposable instance."},
	{Method: "GET", Path: api + "/instances/readiness", Effect: EffectRead, Scope: ScopeRead, Description: "Report local disposable instance lane readiness (tools, KVM, images, architectures)."},
	{Method: "POST", Path: api + "/instances", Effect: EffectHost, Scope: ScopeDestructive, Description: "Create a local disposable instance."},
	{Method: "POST", Path: api + "/instances/{id}/{action}", Effect: EffectHost, Scope: ScopeDestructive, Description: "Act on a local disposable instance."},

	// Deployment records and lifecycle.
	{Method: "GET", Path: api + "/deployments/resolve", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Resolve a deployment selector to its identity."},
	{Method: "GET", Path: api + "/deployments", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "List deployments."},
	{Method: "POST", Path: api + "/deployments", Effect: EffectWorkloadMutation, Scope: ScopeWrite, Description: "Create or resubmit a deployment record."},
	{Method: "GET", Path: api + "/deployments/{id}", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read one deployment."},
	{Method: "GET", Path: api + "/deployments/{id}/receipt", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read the deployment receipt."},
	{Method: "POST", Path: api + "/deployments/{id}/recovery", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Run a recovery action on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/recovery/{operation_id}", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read a recovery operation."},
	{Method: "DELETE", Path: api + "/deployments/{id}", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Delete a deployment and its target bundles."},
	{Method: "POST", Path: api + "/deployments/{id}/execute", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Execute the deployment pipeline on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/progress", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Stream deployment progress (SSE)."},
	{Method: "POST", Path: api + "/deployments/{id}/inspect", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "Inspect the target over SSH."},
	{Method: "POST", Path: api + "/deployments/{id}/stop", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Stop the workload on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/start", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Start the workload on the target."},

	// Executable plans (P06).
	{Method: "POST", Path: api + "/deployments/{id}/plan", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, Description: "Compile and preview the executable plan; no records, no target effects."},
	{Method: "POST", Path: api + "/deployments/{id}/plan/apply", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Admit the reviewed plan digest as a durable operation and hand off to the pipeline."},

	// Durable operations (P07). Operation routes are addressed by operation
	// id; the cancel handler re-checks the operation's deployment at the
	// effect boundary.
	{Method: "POST", Path: api + "/operations/reconcile", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, Description: "Run one operation-owner reconciliation pass (reacquire expired leases, resume from receipts)."},
	{Method: "GET", Path: api + "/operations/{id}", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Read the durable standing of an operation."},
	{Method: "GET", Path: api + "/operations/{id}/wait", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Block server-side until the operation is terminal or the observer bound elapses; never mutates."},
	{Method: "POST", Path: api + "/operations/{id}/cancel", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, Description: "Record a cancellation intent honoured at the next declared cancel point."},
	{Method: "GET", Path: api + "/deployments/{id}/operations", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "List the operations admitted against a deployment."},

	// Live state.
	{Method: "GET", Path: api + "/deployments/{id}/live-state", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Observe live target state."},
	{Method: "GET", Path: api + "/deployments/{id}/metrics-debug", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "Raw metrics for debugging."},
	{Method: "GET", Path: api + "/deployments/{id}/files", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "List files on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/files/content", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Read a file on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/drift", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Compare desired and observed state."},
	{Method: "GET", Path: api + "/deployments/{id}/health", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Legacy health report (embeds the typed observation)."},
	{Method: "GET", Path: api + "/deployments/{id}/health/observation", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Typed health observation."},
	{Method: "POST", Path: api + "/deployments/{id}/actions/kill", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Kill a process on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/actions/restart", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Restart a process on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/actions/process", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Control a process on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/actions/vps", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Run a VPS-level action on the target."},

	// History and logs.
	{Method: "GET", Path: api + "/deployments/{id}/history", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read deployment history."},
	{Method: "POST", Path: api + "/deployments/{id}/history", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Append a history event."},
	{Method: "GET", Path: api + "/deployments/{id}/logs", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Read workload logs from the target."},

	// Secrets on the target.
	{Method: "GET", Path: api + "/deployments/{id}/secrets", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "List secrets on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/secrets", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Create a secret on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/secrets/{key}", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Read a secret on the target."},
	{Method: "PUT", Path: api + "/deployments/{id}/secrets/{key}", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Update a secret on the target."},
	{Method: "DELETE", Path: api + "/deployments/{id}/secrets/{key}", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Delete a secret on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/expected-secrets", Effect: EffectSecret, Scope: ScopeRead, TargetBound: true, NoStore: true, Description: "Secret names the scenario expects (metadata only)."},
	// Credential lifecycle (P13): bindings, versions and operations are
	// metadata only; rotate/revoke/recover/break-glass change credential
	// material on the target.
	{Method: "GET", Path: api + "/deployments/{id}/credentials", Effect: EffectSecret, Scope: ScopeRead, TargetBound: true, NoStore: true, Description: "List credential bindings, versions and acknowledgements (metadata only)."},
	{Method: "POST", Path: api + "/deployments/{id}/credentials/recover", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Re-provision credentials onto a replacement host from a recovery bundle."},
	{Method: "GET", Path: api + "/deployments/{id}/credentials/rotations/{rotation}", Effect: EffectSecret, Scope: ScopeRead, TargetBound: true, NoStore: true, Description: "Read one credential lifecycle operation (metadata only)."},
	{Method: "POST", Path: api + "/deployments/{id}/credentials/rotations/{rotation}/resume", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Resume a credential operation waiting on an operator, a window or an unreachable target."},
	{Method: "POST", Path: api + "/deployments/{id}/credentials/{binding}/rotate", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Rotate a credential binding to a new version."},
	{Method: "POST", Path: api + "/deployments/{id}/credentials/{binding}/revoke", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Revoke a credential binding's active version on the target."},
	{Method: "POST", Path: api + "/deployments/{id}/credentials/{binding}/break-glass", Effect: EffectSecret, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "Issue a scoped, time-bounded, audited emergency credential version."},
	{Method: "*", Path: "/vrooli.scenario_to_cloud.v1.credentials.CredentialsService/", Prefix: true, Effect: EffectSecret, Scope: ScopeDestructive, RemoteReach: true, NoStore: true, Description: "Connect CredentialsService (bindings, rotate, revoke, recover, resume, break-glass)."},

	// Interactive terminal.
	{Method: "GET", Path: api + "/deployments/{id}/terminal", Effect: EffectInteractive, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, NoStore: true, Description: "WebSocket SSH terminal on the target."},

	// Edge and TLS.
	{Method: "GET", Path: api + "/deployments/{id}/edge/dns-check", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Check DNS for the deployment domain."},
	{Method: "GET", Path: api + "/deployments/{id}/edge/dns-records", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Expected DNS records."},
	{Method: "POST", Path: api + "/deployments/{id}/edge/caddy", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Control the edge proxy on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/edge/tls", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "TLS certificate observation."},
	{Method: "POST", Path: api + "/deployments/{id}/edge/tls/renew", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Renew TLS on the target."},
	{Method: "GET", Path: api + "/deployments/{id}/edge", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Typed edge observation: routes, private listeners, DNS binding, TLS lifecycle, ACME environment; never carries secrets."},

	// Documentation.
	{Method: "GET", Path: api + "/docs/manifest", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Documentation manifest."},
	{Method: "GET", Path: api + "/docs/content", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Documentation content."},

	// Investigations (legacy agent integration).
	{Method: "POST", Path: api + "/deployments/{id}/investigate", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Start an investigation agent."},
	{Method: "GET", Path: api + "/deployments/{id}/investigations", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "List investigations."},
	{Method: "GET", Path: api + "/deployments/{id}/investigations/{invId}", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read an investigation."},
	{Method: "POST", Path: api + "/deployments/{id}/investigations/{invId}/stop", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Stop an investigation."},
	{Method: "POST", Path: api + "/deployments/{id}/investigations/{invId}/apply-fixes", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Apply investigation fixes on the target."},
	{Method: "GET", Path: api + "/agent-manager/status", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Agent manager availability."},

	// Unified tasks.
	{Method: "POST", Path: api + "/deployments/{id}/tasks", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Create a task."},
	{Method: "GET", Path: api + "/deployments/{id}/tasks", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "List tasks."},
	{Method: "GET", Path: api + "/deployments/{id}/tasks/{taskId}", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Read a task."},
	{Method: "POST", Path: api + "/deployments/{id}/tasks/{taskId}/stop", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Stop a task."},

	// Closure (P05).
	{Method: "GET", Path: api + "/closure", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Read the dependency closure."},
	{Method: "POST", Path: api + "/closure/explain", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Explain a closure decision."},

	// Releases (verified artifacts on this host).
	{Method: "POST", Path: api + "/releases/build", Effect: EffectWorkloadMutation, Scope: ScopeWrite, Description: "Build a verified release on this host."},
	{Method: "GET", Path: api + "/releases/{digest}", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Read a release manifest by digest."},
	{Method: "POST", Path: api + "/releases/{digest}/verify", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Verify a release against its manifest."},
	{Method: "GET", Path: api + "/releases/{digest}/evidence", Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Per-cell capability-profile dispositions for a release (P17)."},
	{Method: "GET", Path: api + "/deployments/{id}/evidence", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Per-cell dispositions for the release a deployment runs (P17)."},
	{Method: "POST", Path: api + "/deployments/{id}/publication/request", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Report coverage to Deployment Manager and request the exact review (P17)."},
	{Method: "POST", Path: api + "/deployments/{id}/publication/apply", Effect: EffectHost, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Re-check the approval and activate the approved release on the target (P17)."},
	{Method: "GET", Path: api + "/deployments/{id}/publication", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, ServiceAllowed: true, Description: "Read and reconcile a publication against the target receipt (P17)."},

	// Data recovery (P12).
	{Method: "POST", Path: api + "/deployments/{id}/recovery-points", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Capture an encrypted recovery point of the declared data bindings."},
	{Method: "GET", Path: api + "/deployments/{id}/recovery-points", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "List recovery points (references and checksums only)."},
	{Method: "POST", Path: api + "/deployments/{id}/recovery-points/rollback-admission", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Schema-aware rollback admission; refuses with a typed forward-repair or restore plan."},
	{Method: "POST", Path: api + "/deployments/{id}/recovery-points/prune", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, Description: "Apply retention; protected recovery points are never deleted."},
	{Method: "POST", Path: api + "/deployments/{id}/recovery-points/{rp}/restore", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, Description: "Restore a recovery point into clean bindings; refuses restore_target_not_clean."},
	{Method: "GET", Path: api + "/deployments/{id}/recovery-points/{rp}/verify", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Verify a recovery point's checksums, key availability and invariants."},
	// Reconciliation, retirement and legacy data-binding conversion (P11/P15).
	{Method: "GET", Path: api + "/deployments/{id}/desired-state", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, ServiceAllowed: true, Description: "Desired state and reboot policy; the contract observers (autoheal) read before touching a workload."},
	{Method: "PUT", Path: api + "/deployments/{id}/desired-state", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, Description: "Record the operator's running/stopped intent (no target effect)."},
	{Method: "POST", Path: api + "/deployments/{id}/reconcile", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Observe drift and propose a correction; with the reviewed plan digest, admit it as a durable operation."},
	{Method: "POST", Path: api + "/deployments/{id}/retire/plan", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "Preview retirement: retained and deleted owned objects in owner order plus the executable plan."},
	{Method: "POST", Path: api + "/deployments/{id}/retire/apply", Effect: EffectWorkloadMutation, Scope: ScopeDestructive, TargetBound: true, RemoteReach: true, Description: "Admit the reviewed retirement plan as a durable operation."},
	{Method: "POST", Path: api + "/deployments/{id}/data-bindings/inventory", Effect: EffectRead, Scope: ScopeRead, TargetBound: true, RemoteReach: true, Description: "Read-only inventory of mutable directories versus declared and recorded data bindings."},
	{Method: "POST", Path: api + "/deployments/{id}/data-bindings/adopt", Effect: EffectWorkloadMutation, Scope: ScopeWrite, TargetBound: true, Description: "Record the legacy data-binding mapping on the deployment; no data moves."},

	// Typed Connect surface and development-only mounts.
	{Method: "*", Path: DeploymentsServicePrefix, Prefix: true, Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Connect DeploymentsService (resolve, get, list)."},
	{Method: "*", Path: HealthServicePrefix, Prefix: true, Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Connect HealthService (typed observations)."},
	{Method: "*", Path: OperationsServicePrefix, Prefix: true, Effect: EffectWorkloadMutation, Scope: ScopeDestructive, Description: "Connect OperationsService (get/wait/list are reads; cancel and reconcile mutate, so the mount requires the destructive scope)."},
	{Method: "*", Path: PlansServicePrefix, Prefix: true, Effect: EffectWorkloadMutation, Scope: ScopeDestructive, Description: "Connect PlansService (CompilePlan is a read; ApplyPlan admits an operation, so the mount requires the destructive scope)."},
	{Method: "*", Path: EdgeServicePrefix, Prefix: true, Effect: EffectRead, Scope: ScopeRead, ServiceAllowed: true, Description: "Connect EdgeService (typed edge observations)."},
	{Method: "*", Path: ReleasesServicePrefix, Prefix: true, Effect: EffectWorkloadMutation, Scope: ScopeWrite, Description: "Connect ReleasesService (build, get, verify)."},
	{Method: "*", Path: EvidenceServicePrefix, Prefix: true, Effect: EffectHost, Scope: ScopeDestructive, Description: "Connect EvidenceService (evidence reads are reads; ApplyPublication activates a release, so the mount requires the destructive scope)."},
	{Method: "*", Path: DevRoutingServicePrefix, Prefix: true, Effect: EffectWorkloadMutation, Scope: ScopeWrite, Optional: true, Description: "Test Genie routed-pool leases (development only)."},
}

var index = buildIndex()

func buildIndex() map[string]Route {
	out := make(map[string]Route, len(Table))
	for _, route := range Table {
		out[route.Key()] = route
	}
	return out
}

// Lookup resolves a registered route by method and mux template. Prefix
// mounts match any method.
func Lookup(method, template string) (Route, bool) {
	if route, ok := index[Key(method, template)]; ok {
		return route, true
	}
	if route, ok := index[Key("*", template)]; ok && route.Prefix {
		return route, true
	}
	return Route{}, false
}

// Sorted returns the table in a stable order for documentation and export.
func Sorted() []Route {
	out := append([]Route(nil), Table...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// Validate checks the table's internal invariants: unique keys, scopes in the
// vocabulary, public rows only from the allowlist, and every effectful or
// secret row carrying a scope.
func Validate() []string {
	var findings []string
	seen := map[string]struct{}{}
	for _, route := range Table {
		key := route.Key()
		if _, dup := seen[key]; dup {
			findings = append(findings, "duplicate route "+key)
		}
		seen[key] = struct{}{}
		switch route.Effect {
		case EffectPublic:
			if _, ok := PublicAllowlist[key]; !ok {
				findings = append(findings, "public route outside allowlist: "+key)
			}
			if route.Scope != "" {
				findings = append(findings, "public route with scope: "+key)
			}
		case EffectRead, EffectWorkloadMutation, EffectSecret, EffectHost, EffectInteractive:
			if !isKnownScope(route.Scope) {
				findings = append(findings, "route without a catalog scope: "+key)
			}
		default:
			findings = append(findings, "unknown effect on "+key)
		}
		if route.ServiceAllowed && (route.Effect == EffectSecret || route.Effect == EffectHost || route.Effect == EffectInteractive) {
			findings = append(findings, "service principals may not use secret/host/interactive routes: "+key)
		}
		if route.ServiceAllowed && route.Effectful() {
			findings = append(findings, "service principals may only use read routes: "+key)
		}
		if route.TargetBound && !strings.Contains(route.Path, "{id}") {
			findings = append(findings, "target-bound route without {id}: "+key)
		}
		if route.TargetBound && route.BodyTarget {
			findings = append(findings, "route cannot be both target-bound and body-target: "+key)
		}
		if route.Effect == EffectInteractive && route.Scope != ScopeDestructive {
			findings = append(findings, "interactive route must require the destructive scope: "+key)
		}
	}
	return findings
}

func isKnownScope(scope string) bool {
	for _, known := range Scopes() {
		if scope == known {
			return true
		}
	}
	return false
}
