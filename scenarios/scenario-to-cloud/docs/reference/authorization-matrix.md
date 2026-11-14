# Authorization Matrix

> Generated from `api/authz/routes.go` (`authz.RenderMarkdown`). Do not edit by hand; `go test ./authz/ -run TestMatrixDocMatchesTable` fails when this file drifts.

Every management route of the scenario-to-cloud API is classified once. The enforcement middleware (`api/authz/middleware.go`) denies by default: a route absent from this table is refused, a public route must be on the explicit allowlist, and every other route needs a verified principal holding the listed scope.

Scopes are the shared coarse vocabulary (`scenario-to-cloud:read`, `scenario-to-cloud:write`, `scenario-to-cloud:destructive`). Effect classes decide the extra boundaries:

| Effect | Meaning | Extra boundary |
|---|---|---|
| `public` | Liveness metadata only | None (allowlist: `/health`, `/api/v1/health`) |
| `read` | Observe local or remote state | Scope only; service principals allowed where marked |
| `workload_mutation` | Change deployment records or run lifecycle steps | Human only, origin check, per-principal concurrency, effect recheck |
| `secret` | Credential material or its metadata | Human only, `Cache-Control: no-store`, effect recheck |
| `host` | Repair or reconfigure a host | Human only, origin check, effect recheck |
| `interactive` | Long-lived session on a target | Human only, browser Origin required, socket bound to credential expiry |

Columns: **target** = `id` (deployment resolved and policy-checked before the handler), `body` (target named in the request body; only unrestricted principals), or `-`; **reach** = the route opens SSH or another reach to a target; **service** = service principals and verified agents may call it; **no-store** = response is never cacheable.

| Method | Path | Effect | Scope | Target | Reach | Service | No-store | Description |
|---|---|---|---|---|---|---|---|---|
| GET | `/api/v1/agent-manager/status` | read | `scenario-to-cloud:read` | - | no | yes | no | Agent manager availability. |
| GET | `/api/v1/authz/matrix` | read | `scenario-to-cloud:read` | - | no | yes | no | This authorization matrix, for UI/CLI parity. |
| POST | `/api/v1/bundle/build` | workload_mutation | `scenario-to-cloud:write` | - | no | no | no | Build a bundle on this host. |
| GET | `/api/v1/bundles` | read | `scenario-to-cloud:read` | - | no | yes | no | List local bundles. |
| POST | `/api/v1/bundles/cleanup` | host | `scenario-to-cloud:destructive` | - | no | no | no | Delete local bundles. |
| GET | `/api/v1/bundles/stats` | read | `scenario-to-cloud:read` | - | no | yes | no | Local bundle store statistics. |
| DELETE | `/api/v1/bundles/{sha256}` | host | `scenario-to-cloud:destructive` | - | no | no | no | Delete one local bundle. |
| GET | `/api/v1/closure` | read | `scenario-to-cloud:read` | - | no | yes | no | Read the dependency closure. |
| POST | `/api/v1/closure/explain` | read | `scenario-to-cloud:read` | - | no | yes | no | Explain a closure decision. |
| GET | `/api/v1/deployments` | read | `scenario-to-cloud:read` | - | no | yes | no | List deployments. |
| POST | `/api/v1/deployments` | workload_mutation | `scenario-to-cloud:write` | - | no | no | no | Create or resubmit a deployment record. |
| GET | `/api/v1/deployments/resolve` | read | `scenario-to-cloud:read` | - | no | yes | no | Resolve a deployment selector to its identity. |
| DELETE | `/api/v1/deployments/{id}` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Delete a deployment and its target bundles. |
| GET | `/api/v1/deployments/{id}` | read | `scenario-to-cloud:read` | id | no | yes | no | Read one deployment. |
| POST | `/api/v1/deployments/{id}/actions/kill` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Kill a process on the target. |
| POST | `/api/v1/deployments/{id}/actions/process` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Control a process on the target. |
| POST | `/api/v1/deployments/{id}/actions/restart` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Restart a process on the target. |
| POST | `/api/v1/deployments/{id}/actions/vps` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Run a VPS-level action on the target. |
| GET | `/api/v1/deployments/{id}/bundles/vps` | read | `scenario-to-cloud:read` | id | yes | no | no | List bundles cached on the deployment target. |
| POST | `/api/v1/deployments/{id}/bundles/vps/gc` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Garbage-collect bundles on the deployment target. |
| GET | `/api/v1/deployments/{id}/credentials` | secret | `scenario-to-cloud:read` | id | no | no | yes | List credential bindings, versions and acknowledgements (metadata only). |
| POST | `/api/v1/deployments/{id}/credentials/recover` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Re-provision credentials onto a replacement host from a recovery bundle. |
| GET | `/api/v1/deployments/{id}/credentials/rotations/{rotation}` | secret | `scenario-to-cloud:read` | id | no | no | yes | Read one credential lifecycle operation (metadata only). |
| POST | `/api/v1/deployments/{id}/credentials/rotations/{rotation}/resume` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Resume a credential operation waiting on an operator, a window or an unreachable target. |
| POST | `/api/v1/deployments/{id}/credentials/{binding}/break-glass` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Issue a scoped, time-bounded, audited emergency credential version. |
| POST | `/api/v1/deployments/{id}/credentials/{binding}/revoke` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Revoke a credential binding's active version on the target. |
| POST | `/api/v1/deployments/{id}/credentials/{binding}/rotate` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Rotate a credential binding to a new version. |
| POST | `/api/v1/deployments/{id}/data-bindings/adopt` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Record the legacy data-binding mapping on the deployment; no data moves. |
| POST | `/api/v1/deployments/{id}/data-bindings/inventory` | read | `scenario-to-cloud:read` | id | yes | no | no | Read-only inventory of mutable directories versus declared and recorded data bindings. |
| GET | `/api/v1/deployments/{id}/desired-state` | read | `scenario-to-cloud:read` | id | no | yes | no | Desired state and reboot policy; the contract observers (autoheal) read before touching a workload. |
| PUT | `/api/v1/deployments/{id}/desired-state` | workload_mutation | `scenario-to-cloud:destructive` | id | no | no | no | Record the operator's running/stopped intent (no target effect). |
| GET | `/api/v1/deployments/{id}/drift` | read | `scenario-to-cloud:read` | id | yes | yes | no | Compare desired and observed state. |
| GET | `/api/v1/deployments/{id}/edge` | read | `scenario-to-cloud:read` | id | yes | yes | no | Typed edge observation: routes, private listeners, DNS binding, TLS lifecycle, ACME environment; never carries secrets. |
| POST | `/api/v1/deployments/{id}/edge/caddy` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Control the edge proxy on the target. |
| GET | `/api/v1/deployments/{id}/edge/dns-check` | read | `scenario-to-cloud:read` | id | no | yes | no | Check DNS for the deployment domain. |
| GET | `/api/v1/deployments/{id}/edge/dns-records` | read | `scenario-to-cloud:read` | id | no | yes | no | Expected DNS records. |
| GET | `/api/v1/deployments/{id}/edge/tls` | read | `scenario-to-cloud:read` | id | no | yes | no | TLS certificate observation. |
| POST | `/api/v1/deployments/{id}/edge/tls/renew` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Renew TLS on the target. |
| GET | `/api/v1/deployments/{id}/evidence` | read | `scenario-to-cloud:read` | id | no | yes | no | Per-cell dispositions for the release a deployment runs (P17). |
| POST | `/api/v1/deployments/{id}/execute` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Execute the deployment pipeline on the target. |
| GET | `/api/v1/deployments/{id}/expected-secrets` | secret | `scenario-to-cloud:read` | id | no | no | yes | Secret names the scenario expects (metadata only). |
| GET | `/api/v1/deployments/{id}/files` | read | `scenario-to-cloud:read` | id | yes | no | no | List files on the target. |
| GET | `/api/v1/deployments/{id}/files/content` | read | `scenario-to-cloud:read` | id | yes | no | yes | Read a file on the target. |
| GET | `/api/v1/deployments/{id}/health` | read | `scenario-to-cloud:read` | id | yes | yes | no | Legacy health report (embeds the typed observation). |
| GET | `/api/v1/deployments/{id}/health/observation` | read | `scenario-to-cloud:read` | id | yes | yes | no | Typed health observation. |
| GET | `/api/v1/deployments/{id}/history` | read | `scenario-to-cloud:read` | id | no | yes | no | Read deployment history. |
| POST | `/api/v1/deployments/{id}/history` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Append a history event. |
| POST | `/api/v1/deployments/{id}/inspect` | read | `scenario-to-cloud:read` | id | yes | no | no | Inspect the target over SSH. |
| POST | `/api/v1/deployments/{id}/investigate` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Start an investigation agent. |
| GET | `/api/v1/deployments/{id}/investigations` | read | `scenario-to-cloud:read` | id | no | yes | no | List investigations. |
| GET | `/api/v1/deployments/{id}/investigations/{invId}` | read | `scenario-to-cloud:read` | id | no | yes | no | Read an investigation. |
| POST | `/api/v1/deployments/{id}/investigations/{invId}/apply-fixes` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Apply investigation fixes on the target. |
| POST | `/api/v1/deployments/{id}/investigations/{invId}/stop` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Stop an investigation. |
| GET | `/api/v1/deployments/{id}/live-state` | read | `scenario-to-cloud:read` | id | yes | yes | no | Observe live target state. |
| GET | `/api/v1/deployments/{id}/logs` | read | `scenario-to-cloud:read` | id | yes | yes | no | Read workload logs from the target. |
| GET | `/api/v1/deployments/{id}/metrics-debug` | read | `scenario-to-cloud:read` | id | yes | no | no | Raw metrics for debugging. |
| GET | `/api/v1/deployments/{id}/operations` | read | `scenario-to-cloud:read` | id | no | yes | no | List the operations admitted against a deployment. |
| POST | `/api/v1/deployments/{id}/plan` | read | `scenario-to-cloud:read` | id | no | no | no | Compile and preview the executable plan; no records, no target effects. |
| POST | `/api/v1/deployments/{id}/plan/apply` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Admit the reviewed plan digest as a durable operation and hand off to the pipeline. |
| GET | `/api/v1/deployments/{id}/progress` | read | `scenario-to-cloud:read` | id | no | yes | no | Stream deployment progress (SSE). |
| GET | `/api/v1/deployments/{id}/publication` | read | `scenario-to-cloud:read` | id | yes | yes | no | Read and reconcile a publication against the target receipt (P17). |
| POST | `/api/v1/deployments/{id}/publication/apply` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Re-check the approval and activate the approved release on the target (P17). |
| POST | `/api/v1/deployments/{id}/publication/request` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Report coverage to Deployment Manager and request the exact review (P17). |
| GET | `/api/v1/deployments/{id}/receipt` | read | `scenario-to-cloud:read` | id | no | yes | no | Read the deployment receipt. |
| POST | `/api/v1/deployments/{id}/reconcile` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Observe drift and propose a correction; with the reviewed plan digest, admit it as a durable operation. |
| POST | `/api/v1/deployments/{id}/recovery` | host | `scenario-to-cloud:destructive` | id | yes | no | no | Run a recovery action on the target. |
| GET | `/api/v1/deployments/{id}/recovery-points` | read | `scenario-to-cloud:read` | id | no | yes | no | List recovery points (references and checksums only). |
| POST | `/api/v1/deployments/{id}/recovery-points` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Capture an encrypted recovery point of the declared data bindings. |
| POST | `/api/v1/deployments/{id}/recovery-points/prune` | workload_mutation | `scenario-to-cloud:destructive` | id | no | no | no | Apply retention; protected recovery points are never deleted. |
| POST | `/api/v1/deployments/{id}/recovery-points/rollback-admission` | read | `scenario-to-cloud:read` | id | no | yes | no | Schema-aware rollback admission; refuses with a typed forward-repair or restore plan. |
| POST | `/api/v1/deployments/{id}/recovery-points/{rp}/restore` | workload_mutation | `scenario-to-cloud:destructive` | id | no | no | no | Restore a recovery point into clean bindings; refuses restore_target_not_clean. |
| GET | `/api/v1/deployments/{id}/recovery-points/{rp}/verify` | read | `scenario-to-cloud:read` | id | no | yes | no | Verify a recovery point's checksums, key availability and invariants. |
| GET | `/api/v1/deployments/{id}/recovery/{operation_id}` | read | `scenario-to-cloud:read` | id | no | yes | no | Read a recovery operation. |
| POST | `/api/v1/deployments/{id}/retire/apply` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Admit the reviewed retirement plan as a durable operation. |
| POST | `/api/v1/deployments/{id}/retire/plan` | read | `scenario-to-cloud:read` | id | yes | no | no | Preview retirement: retained and deleted owned objects in owner order plus the executable plan. |
| GET | `/api/v1/deployments/{id}/secrets` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | List secrets on the target. |
| POST | `/api/v1/deployments/{id}/secrets` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Create a secret on the target. |
| DELETE | `/api/v1/deployments/{id}/secrets/{key}` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Delete a secret on the target. |
| GET | `/api/v1/deployments/{id}/secrets/{key}` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Read a secret on the target. |
| PUT | `/api/v1/deployments/{id}/secrets/{key}` | secret | `scenario-to-cloud:destructive` | id | yes | no | yes | Update a secret on the target. |
| POST | `/api/v1/deployments/{id}/start` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Start the workload on the target. |
| POST | `/api/v1/deployments/{id}/stop` | workload_mutation | `scenario-to-cloud:destructive` | id | yes | no | no | Stop the workload on the target. |
| GET | `/api/v1/deployments/{id}/tasks` | read | `scenario-to-cloud:read` | id | no | yes | no | List tasks. |
| POST | `/api/v1/deployments/{id}/tasks` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Create a task. |
| GET | `/api/v1/deployments/{id}/tasks/{taskId}` | read | `scenario-to-cloud:read` | id | no | yes | no | Read a task. |
| POST | `/api/v1/deployments/{id}/tasks/{taskId}/stop` | workload_mutation | `scenario-to-cloud:write` | id | no | no | no | Stop a task. |
| GET | `/api/v1/deployments/{id}/terminal` | interactive | `scenario-to-cloud:destructive` | id | yes | no | yes | WebSocket SSH terminal on the target. |
| GET | `/api/v1/docs/content` | read | `scenario-to-cloud:read` | - | no | yes | no | Documentation content. |
| GET | `/api/v1/docs/manifest` | read | `scenario-to-cloud:read` | - | no | yes | no | Documentation manifest. |
| GET | `/api/v1/health` | public | `-` | - | no | yes | no | Liveness metadata for clients. |
| POST | `/api/v1/instances` | host | `scenario-to-cloud:destructive` | - | no | no | no | Create a local disposable instance. |
| POST | `/api/v1/instances/plan` | read | `scenario-to-cloud:read` | - | no | no | no | Plan a local disposable instance. |
| GET | `/api/v1/instances/readiness` | read | `scenario-to-cloud:read` | - | no | no | no | Report local disposable instance lane readiness (tools, KVM, images, architectures). |
| POST | `/api/v1/instances/{id}/{action}` | host | `scenario-to-cloud:destructive` | - | no | no | no | Act on a local disposable instance. |
| POST | `/api/v1/manifest/doctor` | read | `scenario-to-cloud:read` | - | no | no | no | Diagnose a manifest. |
| POST | `/api/v1/manifest/fix` | read | `scenario-to-cloud:read` | - | no | no | no | Return a corrected manifest. |
| POST | `/api/v1/manifest/init` | read | `scenario-to-cloud:read` | - | no | no | no | Derive a manifest from a scenario (reads local secrets metadata). |
| GET | `/api/v1/manifest/schema` | read | `scenario-to-cloud:read` | - | no | yes | no | Manifest JSON schema. |
| GET | `/api/v1/manifest/template` | read | `scenario-to-cloud:read` | - | no | yes | no | Manifest template. |
| POST | `/api/v1/manifest/validate` | read | `scenario-to-cloud:read` | - | no | yes | no | Validate and normalize a manifest. |
| POST | `/api/v1/operations/reconcile` | workload_mutation | `scenario-to-cloud:destructive` | - | no | no | no | Run one operation-owner reconciliation pass (reacquire expired leases, resume from receipts). |
| GET | `/api/v1/operations/{id}` | read | `scenario-to-cloud:read` | - | no | yes | no | Read the durable standing of an operation. |
| POST | `/api/v1/operations/{id}/cancel` | workload_mutation | `scenario-to-cloud:destructive` | - | no | no | no | Record a cancellation intent honoured at the next declared cancel point. |
| GET | `/api/v1/operations/{id}/wait` | read | `scenario-to-cloud:read` | - | no | yes | no | Block server-side until the operation is terminal or the observer bound elapses; never mutates. |
| POST | `/api/v1/preflight` | read | `scenario-to-cloud:read` | body | yes | no | no | Run preflight checks against a target named in the body. |
| POST | `/api/v1/preflight/fix/firewall` | host | `scenario-to-cloud:destructive` | body | yes | no | no | Open firewall ports on a target. |
| POST | `/api/v1/preflight/fix/stop-processes` | host | `scenario-to-cloud:destructive` | body | yes | no | no | Stop scenario processes on a target. |
| GET | `/api/v1/preflight/requirements` | read | `scenario-to-cloud:read` | - | no | yes | no | Preflight requirement catalog. |
| POST | `/api/v1/releases/build` | workload_mutation | `scenario-to-cloud:write` | - | no | no | no | Build a verified release on this host. |
| GET | `/api/v1/releases/{digest}` | read | `scenario-to-cloud:read` | - | no | yes | no | Read a release manifest by digest. |
| GET | `/api/v1/releases/{digest}/evidence` | read | `scenario-to-cloud:read` | - | no | yes | no | Per-cell capability-profile dispositions for a release (P17). |
| POST | `/api/v1/releases/{digest}/verify` | read | `scenario-to-cloud:read` | - | no | yes | no | Verify a release against its manifest. |
| GET | `/api/v1/scenarios` | read | `scenario-to-cloud:read` | - | no | yes | no | List deployable scenarios on this host. |
| GET | `/api/v1/scenarios/{id}/dependencies` | read | `scenario-to-cloud:read` | - | no | yes | no | Dependency closure of one scenario. |
| GET | `/api/v1/scenarios/{id}/ports` | read | `scenario-to-cloud:read` | - | no | yes | no | Port declarations of one scenario. |
| GET | `/api/v1/secrets/{scenario}` | secret | `scenario-to-cloud:destructive` | - | no | no | yes | Fetch bundle secret values for a scenario. |
| POST | `/api/v1/validate/reachability` | read | `scenario-to-cloud:read` | - | no | no | no | Probe reachability of an operator-supplied address. |
| POST | `/api/v1/vps/deploy/apply` | workload_mutation | `scenario-to-cloud:destructive` | body | yes | no | no | Deploy a bundle to a VPS. |
| POST | `/api/v1/vps/deploy/plan` | read | `scenario-to-cloud:read` | body | no | no | no | Plan a VPS deploy. |
| POST | `/api/v1/vps/inspect/apply` | read | `scenario-to-cloud:read` | body | yes | no | no | Inspect a VPS. |
| POST | `/api/v1/vps/inspect/plan` | read | `scenario-to-cloud:read` | body | no | no | no | Plan a VPS inspection. |
| POST | `/api/v1/vps/setup/apply` | host | `scenario-to-cloud:destructive` | body | yes | no | no | Apply a VPS setup. |
| POST | `/api/v1/vps/setup/plan` | read | `scenario-to-cloud:read` | body | no | no | no | Plan a VPS setup. |
| GET | `/health` | public | `-` | - | no | yes | no | Liveness metadata for infrastructure probes. |
| * | `/vrooli.dev_routing.v1.routing.RoutingService/*` | workload_mutation | `scenario-to-cloud:write` | - | no | no | no | Test Genie routed-pool leases (development only). |
| * | `/vrooli.scenario_to_cloud.v1.credentials.CredentialsService/*` | secret | `scenario-to-cloud:destructive` | - | yes | no | yes | Connect CredentialsService (bindings, rotate, revoke, recover, resume, break-glass). |
| * | `/vrooli.scenario_to_cloud.v1.deployments.DeploymentsService/*` | read | `scenario-to-cloud:read` | - | no | yes | no | Connect DeploymentsService (resolve, get, list). |
| * | `/vrooli.scenario_to_cloud.v1.edge.EdgeService/*` | read | `scenario-to-cloud:read` | - | no | yes | no | Connect EdgeService (typed edge observations). |
| * | `/vrooli.scenario_to_cloud.v1.evidence.EvidenceService/*` | host | `scenario-to-cloud:destructive` | - | no | no | no | Connect EvidenceService (evidence reads are reads; ApplyPublication activates a release, so the mount requires the destructive scope). |
| * | `/vrooli.scenario_to_cloud.v1.health.HealthService/*` | read | `scenario-to-cloud:read` | - | no | yes | no | Connect HealthService (typed observations). |
| * | `/vrooli.scenario_to_cloud.v1.operations.OperationsService/*` | workload_mutation | `scenario-to-cloud:destructive` | - | no | no | no | Connect OperationsService (get/wait/list are reads; cancel and reconcile mutate, so the mount requires the destructive scope). |
| * | `/vrooli.scenario_to_cloud.v1.plans.PlansService/*` | workload_mutation | `scenario-to-cloud:destructive` | - | no | no | no | Connect PlansService (CompilePlan is a read; ApplyPlan admits an operation, so the mount requires the destructive scope). |
| * | `/vrooli.scenario_to_cloud.v1.releases.ReleasesService/*` | workload_mutation | `scenario-to-cloud:write` | - | no | no | no | Connect ReleasesService (build, get, verify). |

The same table is served at `GET /api/v1/authz/matrix` (read scope) for UI and CLI parity.
