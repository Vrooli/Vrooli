# API Reference

Complete REST API endpoint documentation for Scenario-to-Cloud.

> [CODE: api/main.go `setupRoutes()`] — canonical route registration

## Base URL

All endpoints are prefixed with `/api/v1`.

## Authentication and authorization

> [CODE: api/authz/routes.go] · [DOC: reference/authorization-matrix.md] · [DOC: reference/configuration.md#authentication]

Every route except `GET /health` and `GET /api/v1/health` requires a verified principal (loopback OS user in `personal_local` mode, or a bearer token from a configured shared provider) holding the scope listed in the [Authorization Matrix](authorization-matrix.md). Refusals use the typed envelope with codes `unauthenticated` (401), `forbidden_scope`, `forbidden_target`, `forbidden_revoked`, `forbidden_origin`, `forbidden_host` (403), `request_too_large` (413) and `too_many_requests` (429).

### GET /authz/matrix

Return the route/effect/scope/target table the boundary enforces (read scope). The response has `schema_version`, `scenario`, `scopes` and `routes[]` with `method`, `path`, `effect`, `scope`, `target_bound`, `body_target`, `remote_reach`, `service_allowed`, `no_store` and `description`.

## Health

### GET /health

Check API health status. Also available at `/api/v1/health`.

**Response:**
```json
{
  "status": "ok",
  "service": "scenario-to-cloud",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Closure

> [CODE: api/handlers_closure.go] · [DOC: reference/closure.md]

### GET /closure?scenario=&environment=&os=&arch=[&scope=]

Derive the deployment closure for a scenario on a concrete platform. `scenario`,
`os` and `arch` are required; `environment` defaults to `production`; `scope` is
`bundle` (default) or `repository`. The response body is the closure document
(see the closure reference); its `reasons` and `unsupported` entries are the
explanation every surface shows.

### POST /closure/explain

Same result as GET with operator inputs in the body:

```json
{
  "scenario": "app", "environment": "production", "os": "linux", "arch": "arm64",
  "overrides": { "select_optional": [], "deselect_optional": ["optional-helper"], "auto_restart": { "app": true }, "supervision_member": {}, "operating_mode": { "postgres": "managed-private" } },
  "target_capacity": { "cpu": 2, "memory_bytes": 4294967296, "disk_bytes": 42949672960 },
  "release_artifact_bytes": [734003200]
}
```

**Errors** use the typed `{"error":{"code","message","retryable","details"}}` envelope:
`invalid_request` (400), `closure_unavailable` (424, catalog or analyzer unreadable — never an empty closure),
`closure_cycle` (422, `details.path`), `closure_conflict` (422).

---

## Releases

> [CODE: api/handlers_release.go] · [CODE: api/releasesvc] · [DOC: reference/release-identity.md]

### POST /releases/build

Build (or return the identical existing) release artifact set for a manifest
and target platform. Body: `{"manifest": {…}, "closure_digest": "sha256:…",
"goos": "linux", "goarch": "amd64", "trust_mode": "", "verify_reproducible": false}`.
Response: `{"schema_version": "1", "release": {release_digest, dir, complete, manifest, inputs}}`.

### GET /releases/{digest}

Read one complete stored release. A missing or incomplete release is
`release_verification_failed` with `details.reason` `release_not_found` or
`release_incomplete`.

### POST /releases/{digest}/verify

Run every cloud-side trust check without writing. Optional body
`{"trust_mode": "production", "goos": "linux", "goarch": "arm64"}`. Response:
`{"schema_version": "1", "report": {release_digest, verified, checks[], policy, trust_mode, signer_key_id}}`.

**Errors:** `release_verification_failed` (422, `details.check` names the failed
step and `details.reason` the cause; the partial report is in `details.report`),
`unsupported_capability` (501, wrong platform, `details.artifact`),
`closure_unavailable` (424, no closure digest), `manifest_invalid` (400),
`invalid_request` (400, malformed digest or trust mode).

### GET /releases/{digest}/evidence

> [CODE: api/handlers_publication.go] · [CODE: api/evidence] · [DOC: reference/governance-binding.md]

Per-cell dispositions of the `cloud-launch-v1` capability profile for one
release (`?deployment_id=` narrows to that deployment's target). Response:
`{"schema_version": "1", "evidence": {profile_id, release_digest, target_key,
cells[{case_id, lane, disposition, required, record_id, operation_id,
observation_id, receipt_refs[], observed_at, reason}], required_cells, passed,
blocking_cells[], producer_ref, evaluated_at}}`. `disposition` is one of
`passed|failed|skipped|unsupported|unavailable|missing`; `passed` is true only
when every required cell is `passed` for that exact release and target.

### GET /deployments/{id}/evidence

Same summary for the release the deployment runs (`?release_digest=`
overrides; `invalid_request` when the deployment runs no stored release).

### POST /deployments/{id}/publication/request

Body `{"request_key", "identity": {scenario_id, profile_id, candidate_commit,
artifact_digest, release_digest, configuration_digest, target_set[],
environment, channel, policy_version, candidate_id, destination_revision_id,
authorization_epoch}}`. Evaluates the profile, reports the coverage verdict to
Deployment Manager, prepares the exact review and stores `review_ref`.
Response `{"schema_version": "1", "publication": {...}, "evidence": {...}}`.
**Errors:** `request_key_conflict` (409, same key with another identity),
`governance_unavailable` (503, retryable), `publication_refused` (409).

### POST /deployments/{id}/publication/apply

Body `{"request_key", "review_ref", "plan_digest"}`. Re-reads the review
immediately before the effect, refuses on any mismatch, admits the reviewed
plan and reconciles against the target's `active-release.json`.
**Errors:** `publication_refused` (409, `details.refusal` ∈
`review_identity_mismatch | review_not_approved | review_revoked |
review_unavailable | evidence_incomplete | plan_digest_mismatch`),
`plan_stale`, `plan_digest_mismatch`, `needs_input`.

### GET /deployments/{id}/publication

Reads (and reconciles) the latest publication or `?request_key=`. States:
`requested | review_prepared | approved | activating | published | refused |
failed`. `activated_release_digest` and `predecessor_release_digest` come from
the target receipt. `operation_not_found` (404) when none exists.

### GET /deployments/{id}/receipt

Signed deployment receipt bound to `{producer_ref, deployment_id,
release_digest, configuration_digest, target_key, operation_id}`. Optional
`?release_digest=&bundle_sha256=&target_key=` state the caller's expectation;
a mismatch or a bad signature is `receipt_invalid` (422).

### POST /deployments/{id}/recovery (governed binding)

Rollback and forward repair additionally take `review_ref` (required),
`release_digest` and `preview_ref`. A dry run returns `preview_ref`; execution
must present it (`preview_required`, 412). Unbound or unsupported targets are
`publication_refused` with `details.refusal` `recovery_unbound` /
`rollback_target_unsupported`. Receipts carry `review_key`, `release_digest`,
`route_kind` and `preview_ref`.

---

## Scenarios

> [CODE: api/scenarios.go]

### GET /scenarios

List all locally available scenarios with their service.json metadata.

**Response:**
```json
{
  "scenarios": [
    {
      "id": "my-scenario",
      "name": "My Scenario",
      "description": "...",
      "services": ["api", "ui"],
      "ports": { "api": 8080, "ui": 3000 }
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /scenarios/{id}/ports

Get port mappings for a specific scenario.

**Response:**
```json
{
  "scenario_id": "my-scenario",
  "ports": { "api": 8080, "ui": 3000 },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /scenarios/{id}/dependencies

Get resource and scenario dependencies for a specific scenario.

**Response:**
```json
{
  "scenario_id": "my-scenario",
  "resources": ["postgres", "redis"],
  "scenarios": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Validation

### POST /validate/reachability

Validate that a VPS host is reachable via SSH.

**Request Body:**
```json
{
  "host": "192.168.1.100",
  "port": 22,
  "user": "root"
}
```

**Response:**
```json
{
  "ok": true,
  "message": "Host reachable",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Manifest

> [CODE: api/manifest/validator.go]

### POST /manifest/validate

Validate a deployment manifest.

**Request Body:**
```json
{
  "scenario": { "id": "my-scenario" },
  "target": { "vps": { "host": "example.com" } },
  "edge": { "domain": "app.example.com" }
}
```

**Response:**
```json
{
  "valid": true,
  "issues": [],
  "manifest": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Bundles

> [CODE: api/bundle/handlers.go]

### POST /bundle/build

Build a deployment bundle.

**Request Body:** Same as `/manifest/validate`

**Response:**
```json
{
  "artifact": {
    "path": "/path/to/bundle.tar.gz",
    "sha256": "abc123...",
    "size_bytes": 15000000
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /bundles

List all locally cached bundles.

**Response:**
```json
{
  "bundles": [
    {
      "filename": "vrooli-bundle-my-scenario-20240115.tar.gz",
      "scenario_id": "my-scenario",
      "sha256": "abc123...",
      "size_bytes": 15000000,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /bundles/stats

Get aggregate bundle statistics.

**Response:**
```json
{
  "stats": {
    "total_bundles": 5,
    "total_size_bytes": 75000000,
    "total_size_kb": 73242,
    "by_scenario": {
      "my-scenario": { "count": 3, "total_size_bytes": 45000000 }
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /bundles/{sha256}

Delete a specific local bundle by its SHA256 hash.

**Response:**
```json
{
  "ok": true,
  "freed_bytes": 15000000,
  "message": "Bundle deleted",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /bundles/cleanup

Clean up old bundles locally and optionally on VPS.

**Request Body:**
```json
{
  "scenario_id": "optional-filter",
  "keep_latest": 3,
  "clean_vps": false,
  "host": "example.com",
  "port": 22,
  "user": "root",
  "workdir": "~/Vrooli"
}
```

**Response:**
```json
{
  "ok": true,
  "local_deleted": ["..."],
  "local_freed_bytes": 30000000,
  "vps_deleted": 0,
  "vps_freed_bytes": 0,
  "vps_error": "",
  "message": "Cleaned up 2 bundles",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/bundles/vps

List bundles present in the VPS bundle cache directory for a specific deployment target.

This is the recommended list endpoint for operators because it is **deployment-scoped**
(no need to pass SSH parameters in the request body). It uses the deployment’s stored
manifest + SSH identity to connect to the VPS.

**Response:** same shape as `GET /deployments/{id}/bundles/vps`.

### POST /deployments/{id}/bundles/vps/gc

Garbage-collect old bundles in the VPS bundle cache directory for a specific deployment target.

This is designed to prevent repeated redeploys from accumulating unbounded disk usage in:
`<workdir>/.vrooli/cloud/bundles`.

Default policy (if omitted in request):
- Keep `keep_latest=2` bundles for the deployment’s scenario (newest by `mod_time`)
- Also protect the deployment’s currently recorded `bundle_sha256` (if present)

**Request Body:**
```json
{
  "scenario_id": "landing-page-business-suite",
  "keep_latest": 2,
  "protect_sha256": ["<optional extra sha>"],
  "dry_run": true
}
```

**Response:**
```json
{
  "ok": true,
  "dry_run": true,
  "deleted_count": 12,
  "deleted_bytes": 1200000000,
  "total_before_bytes": 3500000000,
  "total_after_bytes": 300000000,
  "deleted": [{"filename":"mini-vrooli_...","sha256":"...","size_bytes":123,"mod_time":"..."}],
  "kept": [{"filename":"mini-vrooli_...","sha256":"...","size_bytes":123,"mod_time":"..."}],
  "message": "Would delete 12 VPS bundle(s)",
  "timestamp": "2026-02-10T21:19:55Z"
}
```

---

## Preflight

> [CODE: api/vps/preflight/handlers.go]

### GET /preflight/requirements

Return canonical VPS requirements used by runtime preflight checks.

**Response (shape):**
```json
{
  "vps": {
    "os": {
      "required_id": "ubuntu",
      "recommended_version": "24.04",
      "compatible_versions": ["22.04", "20.04"]
    },
    "resources": {
      "min_disk_free_kb": 5242880,
      "min_ram_kb": 524288,
      "recommended_ram_kb": 2097152
    },
    "network": {
      "required_inbound_ports": [22, 80, 443]
    },
    "authentication": {
      "required_method": "bridge_enrollment_or_ssh_key_binding",
      "bootstrap_flow": "vrooli-bridge onboard (or authorise the operator key with ssh-copy-id; the cloud reaches it through the credential binding vrooli/scenario-to-cloud:ssh-key)"
    }
  }
}
```

### POST /preflight

Run preflight checks against a VPS.

**Request Body:**
```json
{
  "manifest": { "..." }
}
```

**Response:**
```json
{
  "ok": true,
  "checks": [
    {
      "id": "ssh_connectivity",
      "title": "SSH Connectivity",
      "status": "pass",
      "details": "Connected successfully"
    }
  ],
  "issues": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/fix/firewall

Allow the edge ports through the target owner (`cloud-target host repair
--action edge.ufw.allow`, ports 80 and 443 only; the broker never enables or
reloads the firewall). The connection is explicit (no deployment record yet)
and runs through the bounded SSH adapter as argv.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "ports": [80, 443]
}
```

**Response:**
```json
{
  "ok": true,
  "message": "Firewall ports opened",
  "ports": [80, 443],
  "status": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/fix/stop-processes

Stop one scenario through the lifecycle owner (`process.stop.scoped`) or,
without `scenario_id`, every vrooli-managed process through the top-level
`vrooli stop` verb. There is no process-pattern kill.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "workdir": "/root/Vrooli",
  "scenario_id": "optional"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "stop_scenario",
  "message": "Stopped scenario processes",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/disk/usage

Read root filesystem usage and the largest directories under a fixed root set (`/var`, `/home`, `/root`, `/opt`, `/tmp`). Every read is a `df`/`du` observation program through the bound reach; no caller-supplied path reaches the target. The target is named by its locator only: the transport authenticates with the deployment's credential binding (`vrooli/scenario-to-cloud:ssh-key`) or the operator's ambient SSH identity.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root"
}
```

**Response:**
```json
{
  "ok": true,
  "free_space": "10 GB",
  "free_bytes": 10737418240,
  "total_space": "50 GB",
  "total_bytes": 53687091200,
  "used_percent": 80,
  "largest_dirs": [
    { "path": "/var/log", "size": "2.5 GB", "bytes": 2684354560 }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/disk/cleanup

Free disk space through owner operations only. Each action is a privilege-broker action run by `vrooli cloud-target host repair` on the target: `journal_vacuum` (`journald.vacuum`, keeps 100 MiB), `docker_prune` (`docker.prune.unused-images`), `docker_prune_volumes` (`docker.prune.unused-volumes`). An action with no owner (`apt_clean`, `tmp_clean`, anything else) is refused before anything runs with the typed error `unsupported_capability`, naming `internal/privilegebroker` as the owner to extend. Omitting `actions` runs `journal_vacuum`.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "actions": ["journal_vacuum", "docker_prune"]
}
```

**Response:**
```json
{
  "ok": false,
  "space_freed": "1.2 GB",
  "space_freed_kb": 1258291,
  "message": "Freed 1.2 GB of disk space",
  "actions_run": ["journal_vacuum"],
  "actions_failed": ["docker_prune"],
  "action_results": [
    {
      "action": "journal_vacuum",
      "ok": true,
      "exit_code": 0,
      "summary": "{\"ok\":true}"
    },
    {
      "action": "docker_prune",
      "ok": false,
      "exit_code": 2,
      "summary": "broker_unavailable: docker is not installed",
      "hint": "Ensure Docker is installed and running, or drop docker_prune from actions."
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Refusal (no owner):**
```json
{
  "error": {
    "code": "unsupported_capability",
    "message": "disk cleanup action has no owner operation: apt_clean",
    "next_action": { "owner": "internal/privilegebroker", "kind": "capability", "reference": "apt_clean" },
    "details": { "reason": "no privilege-broker action owns apt cache cleaning (apt.packages.ensure only installs)", "supported_actions": ["docker_prune", "docker_prune_volumes", "journal_vacuum"] }
  }
}
```

### GET /secrets/{scenario}

Get local secrets defined for a scenario.

**Response:**
```json
{
  "secrets": { "KEY": "value" },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## VPS Plan/Apply Operations

> [CODE: api/handlers_vps_operations.go] · [DOC: reference/executable-plan.md]

These endpoints compile one executable plan (see the Executable Plan
reference) and execute exactly the plan the caller reviewed.

### POST /vps/setup/plan

Compile the install-scope plan for a manifest and local bundle. No records,
no target effects.

**Request Body:** `{ "manifest": {...}, "bundle_path": "/path/to/bundle.tar.gz" }`

**Response:**
```json
{
  "plan": [ { "id": "host.prepare", "title": "Ensure system packages: ...", "description": "host.prepare · effect host_write · ...", "command": "ssh ... (display only)" } ],
  "plan_digest": "sha256:...",
  "executable_plan": { "schema_version": "1", "actions": [ ... ] },
  "preview": { "target": "...", "changes": [ ... ], "data_effects": [ ... ], "downtime": { ... }, "recovery_strategy": "...", "shell_preview": [ ... ] },
  "issues": [],
  "timestamp": "2026-09-09T10:30:00Z"
}
```

`plan[].id` is the action id; `plan[].command` is a derived shell preview that
execution never consumes.

### POST /vps/setup/apply

Execute the reviewed install plan.

**Request Body:** `{ "manifest": {...}, "bundle_path": "...", "plan_digest": "sha256:...", "deployment_id"?: "...", "request_key"?: "..." }`

The plan is recompiled and compared with `plan_digest`: a missing digest is
`invalid_request` (400), a different digest is `plan_digest_mismatch` (409),
an unsatisfied input is `needs_input` (428). Nothing reaches the target
before these checks pass. With `deployment_id` and `request_key` a durable
`cloud_operations` row is admitted first.

**Response:** `{ "result": VPSSetupResult, "plan_digest", "operation_id", "issues", "timestamp" }` where `result.actions[]` are per-action receipts and `result.failed_step` is the failing action id.

### POST /vps/deploy/plan

Compile the runtime-scope plan (`workload.stop`, `edge.route.apply`,
`credentials.provision`, `runtime.start_dependencies`, `workload.start`,
`verify.readiness`, ...).

**Request Body:** `{ "manifest": {...} }`

**Response:** Same shape as `/vps/setup/plan`.

### POST /vps/deploy/apply

Execute the reviewed runtime plan. Same digest rules as `/vps/setup/apply`.

**Request Body:** `{ "manifest": {...}, "plan_digest": "sha256:...", "deployment_id"?: "...", "request_key"?: "..." }`

**Response:** `{ "result": VPSDeployResult, "plan_digest", "operation_id", "issues", "timestamp" }`.

### POST /vps/inspect/plan

Generate an inspection execution plan.

**Request Body:** Deployment manifest JSON.

**Response:** Same shape as `/vps/setup/plan`.

### POST /vps/inspect/apply

Execute VPS inspection (check processes, logs, health).

**Request Body:** Deployment manifest JSON.

**Response:** `VPSInspectResult` with scenario status, logs, and resource state.

---

## Deployments

> [CODE: api/handlers_deployment.go]
> [CODE: api/handlers_identity.go]
> [CODE: api/deployment/orchestrator.go]
> [DOC: docs/reference/identity-and-selectors.md]

Identity, selector forms and the typed error envelope used by these routes are
defined in `identity-and-selectors.md`. A typed Connect surface,
`DeploymentsService` (`ResolveDeployment`, `GetDeployment`, `ListDeployments`),
is mounted beside the REST routes at
`/vrooli.scenario_to_cloud.v1.deployments.DeploymentsService/`.

### GET /deployments/resolve

Resolve one selector to a stable deployment identity. Exactly one selector
form is accepted: `id`; `scenario` + `environment`; `scenario` + `domain`;
`scenario` + `host`.

**Query Parameters:**
- `id` — Deployment ID (alone)
- `scenario` — Scenario ID, paired with exactly one of the facets below
- `environment` — Environment name (`production` is the default environment)
- `domain` — Edge domain from the manifest
- `host` — Bound target locator host

**Response:**
```json
{
  "schema_version": "1",
  "ref": {
    "id": "uuid",
    "scenario_id": "my-scenario",
    "environment": "production",
    "target": {
      "transport": "ssh",
      "locator": {"host": "192.168.1.100", "port": 22, "user": "root", "workdir": "/root/Vrooli"}
    }
  },
  "timestamp": "2026-09-09T10:30:00Z"
}
```

**Errors:** `deployment_selector_invalid` (400), `deployment_not_found`
(404), `deployment_selector_ambiguous` (409, `details.candidates` lists every
match).

### GET /deployments

List all deployments.

**Query Parameters:**
- `status` — Filter by deployment status
- `scenario_id` — Filter by scenario ID
- `environment` — Filter by environment

**Response:**
```json
{
  "deployments": [
    {
      "id": "uuid",
      "name": "my-scenario @ example.com",
      "scenario_id": "my-scenario",
      "status": "deployed",
      "error_message": "",
      "progress_step": "",
      "progress_percent": 100,
      "created_at": "2024-01-15T10:30:00Z",
      "last_deployed_at": "2024-01-15T10:35:00Z",
      "domain": "app.example.com",
      "host": "192.168.1.100"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments

Create (or update) a deployment.

**Request Body:**
```json
{
  "manifest": { "..." },
  "name": "Optional custom name",
  "bundle_path": "optional",
  "bundle_sha256": "optional",
  "bundle_size_bytes": 0
}
```

**Response:**
```json
{
  "deployment": { "..." },
  "created": true,
  "updated": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}

Get full deployment details.

**Response:**
```json
{
  "deployment": {
    "id": "uuid",
    "name": "my-scenario @ example.com",
    "status": "deployed",
    "manifest": { "..." },
    "setup_result": { "..." },
    "deploy_result": { "..." },
    "created_at": "2024-01-15T10:30:00Z",
    "last_deployed_at": "2024-01-15T10:35:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/receipt

Return the owner-produced receipt for a completed VPS deployment. The endpoint
returns `409 receipt_unavailable` until the persisted deployment status,
successful deploy result, exact destination and bundle digest all agree. A
reachable host or an HTTP success response alone cannot produce this receipt.

**Response:**
```json
{
  "receipt": {
    "schema_version": 1,
    "deployment_id": "uuid",
    "scenario_id": "my-scenario",
    "scenario_version": "1.2.3",
    "target_kind": "vps",
    "destination_id": "sha256:...",
    "destination_host": "192.168.1.100",
    "destination_workdir": "/root/Vrooli",
    "destination_domain": "app.example.com",
    "bundle_sha256": "...",
    "outcome": "deployed",
    "health": "healthy",
    "external_receipt": "scenario-to-cloud:uuid",
    "observed_at": "2024-01-15T10:35:00Z"
  },
  "timestamp": "2024-01-15T10:35:00Z"
}
```

### POST /deployments/{id}/recovery

Request an owner-routed recovery action. `halt` stops the current deployment
after explicit confirmation. `rollback` and `forward_repair` require
`expected_bundle_sha256` to match the persisted current bundle,
`repair_bundle_sha256` to identify an exact retained local bundle, and
`data_compatibility=compatible`. `dry_run=true` validates those identities and
returns a preview without creating an operation or changing deployment state.
Execution is idempotent by `idempotency_key` and returns a pending or running
operation projection until the owner has observed a healthy deployment.

```json
{
  "action": "rollback",
  "expected_bundle_sha256": "<current-sha256>",
  "repair_bundle_sha256": "<retained-predecessor-sha256>",
  "data_compatibility": "compatible",
  "idempotency_key": "release-1:rollback:<current>:<repair>",
  "confirmation": "rollback deployment-1"
}
```

The effect receipt has `outcome=rolled_back` or `forward_repaired`,
`health=healthy`, the repaired `bundle_sha256`, and an operation-scoped
`external_receipt`. A failed operation includes an error and does not include
an effect receipt.

### GET /deployments/{id}/recovery/{operation_id}

Read the durable recovery operation. `pending` and `running` are in progress;
`succeeded` is represented by the owner effect receipt and `failed` preserves
the refusal or execution error. Callers must verify the deployment and
operation identities before accepting a terminal receipt.

### DELETE /deployments/{id}

Delete a deployment record.

**Query Parameters:**
- `stop=true` — Also stop the scenario on VPS
- `cleanup=true` — Also cleanup bundle files

**Response:**
```json
{
  "deleted": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/plan

> [CODE: api/handlers_plan.go] · [DOC: reference/executable-plan.md]

Compile and preview the executable plan for a stored deployment. No target
effects. The release archive is a cloud-local input of the plan: a deployment
without a recorded bundle is built first (and `force_bundle_build` rebuilds an
existing one) so the preview pins the same release digest the execute path
would, instead of a `pending:` placeholder.

**Request Body (optional):** `{ "scope": "full" | "install" | "runtime" | "start" | "stop" | "retire", "force_bundle_build"?: false }`

**Response:** `{ "schema_version": "1", "plan": ExecutablePlan, "plan_digest": "sha256:...", "preview": Preview, "steps": [...], "closure_status": "derived" | "unavailable: <reason>" }`

### POST /deployments/{id}/plan/apply

Admit the reviewed plan as a durable operation.

**Request Body:** `{ "plan_digest": "sha256:...", "request_key": "<idempotency key>", "scope"?: "...", "run_preflight"?: false, "plan"?: ExecutablePlan }`

| Result | Status | Code |
|---|---|---|
| Admitted | 202 | `{ "operation_id", "plan_digest", "state": "admitted" }` |
| Desired state already satisfied | 202 | `{ "plan_digest", "state": "no_op" }` (no operation) |
| Digest differs from the plan compiled now | 409 | `plan_digest_mismatch` |
| Reviewed `plan` supplied and a material precondition changed | 409 | `plan_stale` with `details.reasons[]` |
| Required operator input missing | 428 | `needs_input` with the onboarding handoff as `next_action` |
| Same `request_key` with a different digest | 409 | `request_key_conflict` |
| Another operation holds the deployment's lease | serialised by the owner (the operation stays `admitted` until acquired) |

The Connect `vrooli.scenario_to_cloud.v1.plans.PlansService` (`CompilePlan`,
`ApplyPlan`) mirrors these two endpoints.

### POST /deployments/{id}/execute

Admit a durable operation for the full plan (bundle build, preflight, setup,
deploy) and hand it to the operation owner. The response carries the
operation id clients wait on; a client may disconnect at any time and attach
later by that id. See [operation-lifecycle.md](operation-lifecycle.md).

**Request Body** (optional):
```json
{
  "run_preflight": true,
  "force_bundle_build": false,
  "provided_secrets": { "KEY": "value" },
  "request_key": "<idempotency key; equal replays return the same operation>"
}
```

**Response (202):**
```json
{
  "schema_version": "1",
  "operation_id": "uuid",
  "plan_digest": "sha256:...",
  "state": "admitted",
  "replayed": false,
  "wait": "/api/v1/operations/<operation_id>/wait",
  "deployment": { "..." },
  "message": "Operation admitted. ...",
  "timestamp": "2026-09-09T10:30:00Z"
}
```

| Result | Status | Code |
|---|---|---|
| Desired state already satisfied | 200 | `state: no_op` (no operation) |
| Bundle build failed | 400 | `invalid_request` with `details.step: bundle_build` |
| Required operator input missing | 428 | `needs_input` |
| Same `request_key` with a different digest | 409 | `request_key_conflict` |

Provided secrets are never persisted: after an owner restart a plan that still
needs them parks in `waiting_input` with a resume `next_action`.

### GET /deployments/{id}/progress

**SSE (Server-Sent Events) stream** of deployment progress. Add
`?operation_id=<id>` to key the stream on one durable operation: a terminal
operation replays its outcome (`completed` or `deployment_error`) and closes;
a live one streams hub events plus `operation` events until terminal.

Each event is a JSON object:
```json
{
  "step": "release.stage",
  "status": "running",
  "message": "Extracting bundle...",
  "percent": 45,
  "error_category": "",
  "retryable": false,
  "hint": ""
}
```

> Connect with `EventSource`. The stream closes when the operation (or, without
> `operation_id`, the deployment projection) completes or fails.

### POST /deployments/{id}/inspect

Inspect deployment status on the VPS.

**Response:**
```json
{
  "result": {
    "ok": true,
    "scenario_status": { "..." },
    "scenario_logs": "...",
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/stop

Record `desired_state: stopped` and stop the deployment's scenario through
the target owner's scoped lifecycle stop (`cloud-target host repair --action
process.stop.scoped`). Only this deployment's demand on shared resources is
released. Observation never restarts a stopped deployment (see
[activation-and-reconciliation.md](activation-and-reconciliation.md)).

**Response:**
```json
{
  "success": true,
  "error": "",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/start

Admit a durable operation over the `start` scope plan for a previously
stopped deployment (status `stopped` or `setup_complete`).

**Request Body** (optional): `{ "provided_secrets": {...}, "request_key": "..." }`

**Response (202):** same shape as `/execute` (`operation_id`, `plan_digest`,
`state`, `replayed`, `wait`). A deployment in another status is refused with
`operation_conflict`.

## Reconciliation, retirement and data bindings

> [CODE: api/handlers_reconcile.go] · [CODE: api/reconcile] · [DOC: reference/activation-and-reconciliation.md]

### GET /deployments/{id}/desired-state

The desired-state contract observers read before touching a workload.

```json
{ "schema_version": "1", "deployment_id": "uuid", "desired_state": "running|stopped|retired", "desired_revision": 4, "release_digest": "sha256:...", "recorded_at": "...", "reboot_policy": { "decision": "supervisor_restarts|operator_start_required|leave_stopped", "auto_restart": true, "startup_policy": "always", "source": "service.json" }, "observation_may_restart": false }
```

### PUT /deployments/{id}/desired-state

Body `{ "desired_state": "running" | "stopped" }`. Records intent only
(`retired` goes through `/retire/apply`). Returns the GET shape.

### POST /deployments/{id}/reconcile

Body (optional) `{ "plan_digest", "request_key" }`. Compares desired and
observed state into a typed report `{ outcome: unchanged|changed|blocked|unknown, reasons[], desired{desired_revision,...}, observed{observed_release_digest, observed_at, freshness, ...}, correction{kind, scope}?, reboot_policy }`
and, when a correction exists, the compiled plan it proposes (`correction`).
Without `plan_digest` nothing is applied (200). With the reviewed digest the
plan is admitted as a durable operation (202, `operation`); a different
digest is `plan_digest_mismatch` (409). A stopped intent is never corrected
into a start.

### POST /deployments/{id}/retire/plan

Body (optional) `{ "retention_policy": "retain" | "delete" }`. Returns
`{ retirement: { order[], deleted[], retained[], blocked[], outcome }, plan?, plan_error?, target? }`.
A deployment holding data without a policy is `blocked`.

### POST /deployments/{id}/retire/apply

Body `{ "retention_policy", "plan_digest", "request_key" }`. Refuses a
blocked plan or a different digest; otherwise admits the retire-scope plan
(202) and records `desired_state: retired`.

### POST /deployments/{id}/data-bindings/inventory

Read-only. Runs `cloud-target data inventory` per bundled scenario and
returns `{ declared_bindings[], recorded_mappings[], inventory[], unmapped[] }`.

### POST /deployments/{id}/data-bindings/adopt

Body `{ "mappings": [{ "binding_id", "scenario", "path" }] }`. Records the
legacy conversion on the deployment (`persistent_data`); binding ids must be
declared by the closure; no data moves. The next `release.activate` binds the
recorded paths and adopts their content by rename.

## Durable operations

> [CODE: api/handlers_operations.go] · [CODE: api/operations] · [DOC: reference/operation-lifecycle.md]

### GET /operations/{id}

Typed standing of one operation.

```json
{
  "schema_version": "1",
  "operation_id": "uuid",
  "deployment_id": "uuid",
  "request_key": "...",
  "plan_digest": "sha256:...",
  "state": "running",
  "terminal": false,
  "fence": 3,
  "worker_id": "host-1234-abcd",
  "lease_expires_at": "...",
  "cancel_requested": false,
  "active_step": "release.activate",
  "completed_steps": ["host.prepare", "release.stage"],
  "step_receipts": [{ "step": "release.stage", "outcome": "succeeded", "fence": 3, "source": "worker", "completed_at": "..." }],
  "unknown_effects": [],
  "result": null,
  "error": null,
  "next_action": { "owner": "scenario-to-cloud", "kind": "wait", "reference": "/api/v1/operations/uuid/wait", "label": "Wait once; the owner holds progress" },
  "reattach_command": "scenario-to-cloud operation wait uuid",
  "created_at": "...", "updated_at": "..."
}
```

### GET /operations/{id}/wait?timeout=

Block server-side until the operation is terminal or `timeout` (seconds or a
Go duration; capped by the owner's `ObserverTimeout`, default 5m) elapses. A
timed-out wait returns 200 with the live standing plus `still_pending: true`
and `recommended_next_check_seconds`; it never changes the record.

### POST /operations/{id}/cancel

Record a cancellation intent (202 with the standing). Honoured at the next
declared cancel point; a terminal operation is refused with
`operation_conflict`.

### GET /deployments/{id}/operations

`{ "schema_version": "1", "deployment_id": "...", "operations": [Standing, ...] }`, newest first.

### POST /operations/reconcile

Run one owner reconciliation pass now (destructive scope). Returns
`{ "worker_id", "acquired": ["<operation ids reacquired>"] }`.

The Connect `vrooli.scenario_to_cloud.v1.operations.OperationsService`
(`GetOperation`, `WaitOperation`, `CancelOperation`,
`ListDeploymentOperations`, `ReconcileOperations`) mirrors these endpoints.

---

## Live State Inspection

> [CODE: api/handlers_live_state.go]
> [CODE: api/vps/live_state.go]

### GET /deployments/{id}/live-state

Get comprehensive live state from the VPS (runs ~15 SSH commands in parallel).

**Response:**
```json
{
  "result": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

The result includes processes, ports, disk, memory, CPU, uptime, scenario status, resource status, Caddy config, and more. See [architecture.md](../concepts/architecture.md#live-state-parallel-inspection) for the full collection.

### GET /deployments/{id}/files

List files on the remote VPS.

**Query Parameters:**
- `path` — Directory path (defaults to deployment workdir)

**Response:**
```json
{
  "ok": true,
  "path": "/root/Vrooli",
  "entries": [
    {
      "name": "scripts",
      "size": "4096",
      "permissions": "drwxr-xr-x",
      "owner": "root",
      "modified": "2024-01-15"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/files/content

Read a file's content from the VPS.

**Query Parameters:**
- `path` (required) — File path on the VPS

**Response:**
```json
{
  "ok": true,
  "path": "/root/Vrooli/scenarios/my-scenario/.env",
  "size_bytes": 256,
  "content": "KEY=value\n...",
  "truncated": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/drift

Detect drift between expected and actual state on the VPS.

**Response:**
```json
{
  "result": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/health

Get the composed deployment health report. HTTP 200 means the report was produced, not that the deployment is healthy. The response carries the typed observation in `observation` (see [Health Contract](health-contract.md)); consumers that decide readiness must read `observation.status` and `observation.freshness`, never `ok`/`health` alone. A failed SSH inspection reports `health: unknown`.

**Response:**
```json
{
  "ok": true,
  "checks": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Live State Actions

> [CODE: api/handlers_live_state.go]

### GET /deployments/{id}/health/observation

Get only the typed health observation (`vrooli.scenario_to_cloud.v1.health.HealthObservation`) in its versioned envelope. Also served as Connect `HealthService.GetHealthObservation`. Every call makes a fresh inspection; `observed_at` is the producer time of the live-state inspection.

**Response:**
```json
{
  "schema_version": "1",
  "observation": {
    "deployment_id": "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11",
    "target_id": "host:203.0.113.10",
    "observed_release_digest": "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
    "observed_configuration_digest": "sha256:…",
    "observed_at": "2026-09-09T12:00:00Z",
    "status": "HEALTH_STATUS_UNHEALTHY",
    "checks": [
      {"id": "host_presence", "status": "CHECK_STATUS_PASSED", "reason_code": "", "detail": "Target answered the live-state inspection"},
      {"id": "application_readiness", "status": "CHECK_STATUS_FAILED", "reason_code": "process_not_running", "detail": "landing-page stopped"}
    ],
    "freshness": "FRESHNESS_CURRENT",
    "producer_ref": "scenario-to-cloud:health:v1",
    "partial": false,
    "missing_dependencies": [],
    "next_actions": [{"owner": "scenario-to-cloud", "kind": "command", "reference": "scenario-to-cloud process control 3c1c9a1e-… restart", "label": "landing-page stopped"}]
  }
}
```

Status and freshness are independent: a report can be current and unhealthy, or healthy and stale. `HEALTH_STATUS_UNKNOWN` with `FRESHNESS_UNKNOWN` means the target could not be inspected. Semantics and consumer rules: [Health Contract](health-contract.md).

### POST /deployments/{id}/actions/kill

Kill a process on the VPS by PID.

**Request Body:**
```json
{
  "pid": 12345,
  "signal": "TERM"
}
```

**Response:**
```json
{
  "ok": true,
  "pid": 12345,
  "signal": "TERM",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/restart

Restart a scenario or resource on the VPS.

**Request Body:**
```json
{
  "type": "scenario",
  "id": "my-scenario"
}
```

**Response:**
```json
{
  "ok": true,
  "type": "scenario",
  "id": "my-scenario",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/process

Generic process control (start/stop/restart/setup).

**Request Body:**
```json
{
  "action": "restart",
  "type": "resource",
  "id": "postgres"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "restart",
  "type": "resource",
  "id": "postgres",
  "message": "Restarted",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/vps

VPS-level management actions through the target owner's typed verbs.
`stop_vrooli` runs scoped stops for the scenario and its dependent
scenarios plus `resource stop` (kept while another deployment demands the
resource) and records `desired_state: stopped`; `cleanup` level 3 runs the
privilege broker's `docker.prune.unused-images` / `docker.prune.unused-volumes`;
`reboot` and cleanup levels 1, 2, 4, 5 are refused with `unsupported_capability`
naming the missing owner (`privilegebroker host.reboot`, the retirement
plan at `/deployments/{id}/retire/plan`).

**Request Body:**
```json
{
  "action": "reboot",
  "cleanup_level": 3,
  "confirmation": "REBOOT"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "reboot",
  "message": "VPS rebooting",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## History & Logs

> [CODE: api/handlers_history.go]

### GET /deployments/{id}/history

Get deployment event history.

**Response:**
```json
{
  "ok": true,
  "history": [
    {
      "type": "deploy_started",
      "timestamp": "2024-01-15T10:30:00Z",
      "message": "Deployment started",
      "details": "...",
      "success": true,
      "step_name": "setup"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/history

Add a custom history event.

**Request Body:**
```json
{
  "type": "custom_event",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Manual intervention",
  "details": "...",
  "success": true
}
```

**Response:**
```json
{
  "ok": true,
  "event": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/logs

Get deployment logs with filtering.

**Query Parameters:**
- `source` (default: `all`) — Filter by log source
- `level` (default: `all`) — Filter by level: `INFO`, `WARN`, `ERROR`, `DEBUG`
- `tail` (default: `200`, max: `2000`) — Number of lines
- `search` — Text search filter

**Response:**
```json
{
  "ok": true,
  "logs": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "source": "scenario",
      "level": "INFO",
      "message": "Server started on port 8080"
    }
  ],
  "total": 500,
  "filtered": 200,
  "sources": ["scenario", "caddy", "system"]
}
```

---

## Deployment Credentials

> [CODE: api/handlers_credentials.go] · [CODE: api/credentials/lifecycle.go] · [DOC: reference/credential-lifecycle.md]

Versioned credential bindings, rotation, revocation, break-glass and recovery.
Every response is metadata only: bindings, versions (number + opaque
`content_ref`), consumer acknowledgements and operation receipts. No route ever
returns a value. Binding ids are `cb_<24 hex>`, a digest of (deployment id, logical_id,
field); the binding record carries the descriptor for reading. All routes are `secret` effects.

### GET /deployments/{id}/credentials

**Response:**
```json
{
  "schema_version": "1",
  "bindings": [
    {
      "binding": {
        "id": "cb_3f9a1c...",
        "deployment_id": "7e49...",
        "descriptor": { "logical_id": "vrooli/app", "field": "db-password" },
        "class": "generated_database_password",
        "source_class": "per_install_generated",
        "target": { "type": "env", "name": "POSTGRES_PASSWORD" },
        "version": { "number": 2, "content_ref": "cref_...", "created_at": "..." },
        "consumer_refs": ["scenario:app", "resource:postgres"],
        "state": "materialized"
      },
      "acks": [ { "binding_id": "...", "consumer": "scenario:app", "version": 2, "verified_at": "..." } ]
    }
  ]
}
```

### POST /deployments/{id}/credentials/{binding}/rotate

**Request Body:** `{ "value": "<replacement, external classes only>", "request_key": "optional" }`.
The value is inbound only; it is never persisted or echoed.

**Response:** `{ "schema_version": "1", "operation": { ...CredentialRotation } }` with
`state` one of `planned | new_version_created | provider_prepared | consumers_updated | verified | pending_operator_input | old_version_revoked | complete | recovering | failed`.
A non-terminal standing (`consumers_updated` with `unreached[]`,
`pending_operator_input`) is returned with the typed error carrying the
operation in `error.details.operation`.

### POST /deployments/{id}/credentials/{binding}/revoke

**Response:** operation with state `complete` (online target) or
`revocation_incomplete` with `unreached[]` (target unreachable; typed error
`revocation_incomplete`, HTTP 202). Receipts state what revocation cannot
prove.

### POST /deployments/{id}/credentials/{binding}/break-glass

**Request Body:** `{ "scope": "...", "window_seconds": 3600, "confirmation": "BREAK-GLASS <binding-id>", "operator": "..." }`.
Refused with `break_glass_confirmation_required` unless the confirmation matches exactly; window at most 4h.

### POST /deployments/{id}/credentials/recover

**Request Body:** `{ "bundle_ref": "/path/on/replacement/host.bundle", "passphrase": "..." }`.
The passphrase is in-memory only and travels to the replacement host on standard input.
A locked store answers `credential_store_locked` with the next action.

### GET /deployments/{id}/credentials/rotations/{rotation}

One operation with its receipts and per-consumer standing.

### POST /deployments/{id}/credentials/rotations/{rotation}/resume

**Request Body:** `{ "operator_confirmed": true }` continues a
`pending_operator_input` operation, retries unreached consumers, or re-runs an
incomplete revocation.

Typed codes: `credential_descriptor_collision`, `credential_binding_ambiguous`,
`credential_binding_not_found`, `credential_rotation_not_found`,
`credential_rotation_conflict`, `credential_rotation_refused`,
`credential_store_locked`, `credential_distribution_failed`,
`revocation_incomplete`, `pending_operator_input`,
`credential_verification_failed`, `break_glass_confirmation_required`,
`credential_recovery_failed`, `forbidden_revoked`.

---

## Credential Lifecycle

> [CODE: api/handlers_credentials.go] · [CODE: api/credentialsvc/service.go] · [DOC: docs/reference/credential-lifecycle.md]

Versioned credential bindings, rotation, revocation, recovery and break-glass.
Metadata only on the way out; `value` (rotate) and `passphrase` (recover) are
inbound only and never persisted, logged or echoed. Every route is
`Cache-Control: no-store` and requires the destructive scope except the two
reads. The Connect `CredentialsService` is mounted beside these routes with
the same lifecycle.

### GET /deployments/{id}/credentials

`{ "schema_version": "1", "bindings": [ { "binding": {…}, "acks": [ {…} ] } ] }`.

### POST /deployments/{id}/credentials/{binding}/rotate

Body `{ "value": "…", "request_key": "…" }` (`value` only for external
classes; generated classes mint). `200` complete, `202` incomplete
(`state`, `unreached`, `pending_operator_input`, `resume_after`), typed
refusals carry `details.operation`.

### POST /deployments/{id}/credentials/{binding}/revoke

Body `{ "request_key": "…" }`. `200` complete; `202 revocation_incomplete`
with `unreached` when the target was not reached.

### POST /deployments/{id}/credentials/{binding}/break-glass

Body `{ "scope", "operator", "window_seconds" (≤ 14400), "confirmation": "BREAK-GLASS <binding>" }`.

### POST /deployments/{id}/credentials/recover

Body `{ "bundle_ref": "/path/on/replacement/host", "passphrase": "…" }`.

### GET /deployments/{id}/credentials/rotations/{rotation}

One lifecycle operation with consumers, receipts, handoff and error.

### POST /deployments/{id}/credentials/rotations/{rotation}/resume

Body `{ "operator_confirmed": true|false }`. Continues an operation waiting on
an operator step, an overlap window, missing acknowledgements or an unreachable
target.

## VPS Secrets Management

> [CODE: api/secrets/handlers_management.go] · [DOC: docs/reference/credential-lifecycle.md]

Post-deployment management of operator-supplied credentials on the target,
expressed on the credential binding model: create materialises a version,
update rotates, delete revokes. Values travel in only; no response carries a
value.

### GET /deployments/{id}/secrets

List the deployment's credential bindings as secret entries (metadata only).

**Response:**
```json
{
  "secrets": [
    { "key": "MAILER_TOKEN", "masked": true, "source": "credential-binding", "binding_id": "cb_…", "descriptor": "fixture/mailer:api-token", "class": "external_api_credential", "version": 2, "state": "materialized", "last_updated": "2026-09-09T12:00:00Z" }
  ],
  "metadata": { "environment": "credential-binding", "last_updated": "…", "scenario_id": "my-scenario", "generated_by": "scenario-to-cloud" },
  "timestamp": "…"
}
```

### GET /deployments/{id}/secrets/{key}

Get one binding's metadata. `?reveal=true` is refused with `forbidden_scope`;
credential values are not a management API surface.

### POST /deployments/{id}/secrets

Bind a new key and materialise its first version from the value in the body.

**Request Body:**
```json
{ "key": "MY_API_KEY", "value": "…", "restart_scenario": false }
```

**Response (201):** `{ "ok": true, "key": "MY_API_KEY", "action": "created", "binding_id": "cb_…", "version": 1, "message": "…", "timestamp": "…" }`.
An active binding for the key is `credential_rotation_conflict` (409).

### PUT /deployments/{id}/secrets/{key}

Rotate the key to the value in the body. Runs the full lifecycle
(provider prepare, distribute, consumer acknowledgement, predecessor
revocation). `200` when complete, `202` when the rotation is truthfully
incomplete (`operation_state`, `rotation_id` name what is outstanding).

**Request Body:**
```json
{ "value": "…", "restart_scenario": false }
```

### DELETE /deployments/{id}/secrets/{key}

Revoke the key's active version on the target. Requires
`"confirmation": "DELETE"`. An unreachable target returns
`revocation_incomplete` (202) with the operation.

## Terminal

### GET /deployments/{id}/terminal

**WebSocket** endpoint — opens an interactive SSH terminal session to the VPS.

Connect via WebSocket. Sends/receives raw terminal data. The server proxies stdin/stdout/stderr over SSH.

---

## Edge/TLS Management

> [CODE: api/handlers_edge.go]

### GET /deployments/{id}/edge/dns-check

Check if DNS records point to the deployment's VPS.

**Response:**
```json
{
  "ok": true,
  "vps_host": "192.168.1.100",
  "vps_ips": ["192.168.1.100"],
  "domains": [
    {
      "domain": "app.example.com",
      "role": "apex",
      "ok": true,
      "domain_ips": ["192.168.1.100"],
      "points_to_vps": true,
      "proxied": false,
      "message": "DNS configured correctly",
      "hint": "",
      "hint_data": {
        "type": "A",
        "name": "app.example.com",
        "value": "192.168.1.100",
        "ttl": 3600
      }
    }
  ],
  "message": "All DNS records OK",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/edge/dns-records

Get raw DNS records for the deployment's domains.

**Response:**
```json
{
  "ok": true,
  "domains": [
    {
      "domain": "app.example.com",
      "records": {
        "a": ["192.168.1.100"],
        "aaaa": [],
        "cname": "",
        "mx": [],
        "txt": []
      }
    }
  ],
  "message": "",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/edge/caddy

Control the Caddy web server on the VPS.

**Request Body:**
```json
{
  "action": "restart"
}
```

Actions: `start`, `stop`, `restart`, `reload`.

**Response:**
```json
{
  "ok": true,
  "action": "restart",
  "message": "Caddy restarted",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/edge/tls

Get TLS certificate information for the deployment's domain.

**Response:**
```json
{
  "ok": true,
  "domain": "app.example.com",
  "valid": true,
  "validation": "valid",
  "issuer": "Let's Encrypt",
  "subject": "app.example.com",
  "not_before": "2024-01-01T00:00:00Z",
  "not_after": "2024-04-01T00:00:00Z",
  "days_remaining": 76,
  "serial_number": "...",
  "sans": ["app.example.com"],
  "alpn": {
    "supported": true,
    "protocols": ["h2", "http/1.1"]
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/edge/tls/renew

Force TLS certificate renewal on the VPS.

**Response:**
```json
{
  "ok": true,
  "domain": "app.example.com",
  "message": "Certificate renewal initiated",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Investigation (Legacy)

> [CODE: api/investigation/service.go] — Legacy endpoints, kept for backward compatibility. Prefer the unified [Tasks](#tasks) endpoints.

### POST /deployments/{id}/investigate

Start an investigation using the agent-manager.

**Request Body:**
```json
{
  "focus": { "harness": true, "subject": true },
  "note": "Deployment failing after deploy step",
  "effort": "inspect"
}
```

**Response:**
```json
{
  "investigation": { "..." }
}
```

### GET /deployments/{id}/investigations

List investigations for a deployment.

**Response:**
```json
{
  "investigations": [{ "..." }]
}
```

### GET /deployments/{id}/investigations/{invId}

Get a specific investigation.

**Response:**
```json
{
  "investigation": { "..." }
}
```

### POST /deployments/{id}/investigations/{invId}/stop

Stop a running investigation.

**Response:**
```json
{
  "success": true,
  "message": "Investigation stopped"
}
```

### POST /deployments/{id}/investigations/{invId}/apply-fixes

Apply fixes recommended by an investigation.

**Response:**
```json
{
  "success": true,
  "applied": ["..."]
}
```

### GET /agent-manager/status

Check if the agent-manager integration is available.

**Response:**
```json
{
  "available": true,
  "profile": "scenario-to-cloud-investigator",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Tasks (Unified)

> [CODE: api/handlers_tasks.go]
> [CODE: api/tasks/service.go]

Unified task endpoints that replace the legacy investigation API.

### POST /deployments/{id}/tasks

Create a new task (investigate or fix).

**Request Body:**
```json
{
  "task_type": "investigate",
  "focus": { "harness": true, "subject": true },
  "note": "Deployment failing after deploy step",
  "effort": "inspect",
  "permissions": {
    "immediate": false,
    "permanent": false,
    "prevention": false
  },
  "include_contexts": [],
  "source_investigation_id": "",
  "max_iterations": 5
}
```

**Response:**
```json
{
  "task": { "..." }
}
```

### GET /deployments/{id}/tasks

List tasks for a deployment.

**Query Parameters:**
- `limit` (default: `10`) — Maximum number of tasks

**Response:**
```json
{
  "tasks": [{ "..." }]
}
```

### GET /deployments/{id}/tasks/{taskId}

Get a specific task.

**Response:**
```json
{
  "task": { "..." }
}
```

### POST /deployments/{id}/tasks/{taskId}/stop

Stop a running task.

**Response:**
```json
{
  "success": true,
  "message": "Task stopped"
}
```

## Documentation

### GET /docs/manifest

Get the documentation navigation manifest.

**Response:**
```json
{
  "version": "1.0.0",
  "title": "Scenario-to-Cloud Documentation",
  "defaultDocument": "QUICKSTART.md",
  "sections": [{ "..." }]
}
```

### GET /docs/content

Get a document's content.

**Query Parameters:**
- `path` — Document path (e.g., `QUICKSTART.md`)

**Response:**
```json
{
  "path": "QUICKSTART.md",
  "content": "# Quick Start Guide\n..."
}
```

---

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable message",
    "hint": "Optional suggestion"
  }
}
```

Common status codes:
- `400` — Bad Request (invalid input)
- `404` — Not Found
- `422` — Unprocessable Entity (validation failed)
- `500` — Internal Server Error

SSH-related endpoints include structured error metadata. See [configuration.md](configuration.md#error-categories) for the full error category taxonomy.

## SSH Identity Health Semantics

> [CODE: api/vps/health.go]
> [CODE: api/vps/live_state.go]

`GET /deployments/{id}/health` evaluates `ssh_key_auth` from canonical identity:

- `pass`: `auth_mode=explicit_key` and `verification_state=authorized`
- `warn`: `auth_mode=agent|default_ssh|unknown` (connected, but unpinned transport)
- `fail`: `auth_mode=explicit_key` with `verification_state=unauthorized|unknown`, or SSH unreachable

`GET /deployments/{id}/live-state` includes SSH identity fields under `result.system.ssh`:

```json
{
  "auth_mode": "explicit_key",
  "verification_state": "authorized",
  "key_path": "~/.ssh/id_ed25519",
  "public_key_fingerprint": "SHA256:...",
  "last_verified_at": "2026-02-08T18:30:00Z"
}
```

## TLS-ALPN Health Semantics

> [CODE: api/vps/health.go]
> [CODE: api/tlsinfo/alpn.go]

`GET /deployments/{id}/health` evaluates `tls_alpn` relative to certificate validity window:

- `pass`: `acme-tls/1` negotiated successfully, or ALPN probe fails while cert remains healthy (`days_remaining >= 30`)
- `warn`: ALPN probe fails and cert is in renewal window (`14 <= days_remaining < 30`)
- `fail`: ALPN probe fails with near-expiry cert (`days_remaining < 14`)

Operational guidance:
- ALPN warning with a healthy cert is informational and should not be treated as an outage by itself.
- Investigate immediately when ALPN is `warn`/`fail` inside renewal windows.
