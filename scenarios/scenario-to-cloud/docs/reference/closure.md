# Deployment Closure

The closure is the complete, versioned set of components a target must hold to
run a scenario, derived from component declarations rather than from
product-specific rules. Every component carries the reasons it is included,
every platform gap is a typed `unsupported` entry, and a catalog that cannot
be read produces a typed `closure_unavailable` error rather than an empty
closure.

> [CODE: api/closure/service.go] — resolver, catalogs, selection mapping
> [CODE: api/domain/closure.go] — wire types
> [CODE: api/handlers_closure.go] — `GET /api/v1/closure`, `POST /api/v1/closure/explain`

## Inputs

| Input | Source | Notes |
|---|---|---|
| Scenario declarations | `scenarios/<id>/.vrooli/service.json` | dependencies, runtime, hostTools/hostSafeguards, credentials, ports, components, tier requirements, `deployment` |
| Resource declarations | `resources/<id>/resource.json` | requirements, dependencies, credentials, `deployment.profiles`, `managed_service.artifact`, provider policy, `deployment.{persistent_data,listeners,recovery}` |
| Analyzer | `scenario-dependency-analyzer` `GET /api/v1/analyze/{scenario}` | detected edges join the declared ones; an outage is `closure_unavailable`, never a silent fallback |
| Host requirements | `packages/hostreq` (control plane) | tools, safeguards, privilege and bundling per platform; cloud carries no private copy |
| Operator overrides | `closure.Request.overrides` | select/deselect optional dependencies, auto-restart, supervision membership, operating modes |
| Target platform | `os` + `arch` | both required; artifact eligibility is per architecture |
| Target capacity | optional | cpu, memory, disk; when absent `capacity.fit` is `unknown` |

## Rules

1. **Transitive over scenarios and resources.** Each edge yields a
   `declared_by` reason naming the parent; an edge further than one hop from
   the selected scenario adds `transitive_via` with the full path.
2. **Cycles.** A cycle made only of required (`must_start`) edges is a
   typed `closure_cycle` error whose `details.path` names the loop. A cycle
   containing an optional (`try_start`) edge is a declared degradation
   pattern and resolves normally. Contradictory operating modes (a `one_shot`
   scenario with auto-restart on, or a requested provider mode the resource
   forbids) are `closure_conflict`.
3. **Scope.** `bundle` (default) includes only what the selected scenario
   reaches, plus the shared `internal` and `packages` trees the mini-Vrooli
   bundle profile ships unminimised. `repository` also includes every
   `service.system_required` scenario with reason `system_required`.
4. **Optional stays optional.** An optional dependency the manifest enables is
   included with `optional_selected: true` and can be deselected; a disabled
   one can be selected. A required dependency can never be deselected, and
   requiredness only propagates over required edges.
5. **Supervision and auto-restart are independent.** Scenario components carry
   `supervision.member` (from the edge's `startup_policy`; `ignore` edges are
   not members), `supervision.startup_policy`, `supervision.runtime_kind`,
   `supervision.auto_restart` (from `runtime.auto_restart_default`) and
   `auto_restart_source` (`declared` or `override`). Overriding one never
   changes the other.
6. **Host tools and safeguards** come from `packages/hostreq` scoped to the
   included scenario paths and resources; each carries `safeguard_of` reasons
   naming the declaring component. Privileges are listed under `privileges`
   using the existing `none|user|elevated` effect vocabulary.
7. **Credential descriptors** are composed from every included component and
   keyed by the shared address `logical_id:field`; `logical_id` and `field`
   are never normalised. The same address declared by two owners is one
   component with both `credential_of` reasons.
8. **Native artifacts.** The `vrooli` control plane appears as a
   `native_artifact` (buildable on linux amd64/arm64); each bundled-service
   resource contributes `<resource>:server` with the digest-pinned artifact
   for the platform. A resource with no artifact for the target architecture
   yields `unsupported[{reason_code: missing_platform_artifact}]` naming the
   dependency; a declared `unsupported` target yields `unsupported_platform`.
9. **Capacity** sums declared `requirements` (resources) and tier
   requirements (scenarios, `tier-4-saas` then `tier-1-local`). Transient
   update headroom is two copies of the largest release artifact plus a
   256 MiB staging allowance; without known artifact sizes the largest
   declared disk footprint is the proxy (`headroom_basis` says which). A
   target that cannot hold cpu, memory, or disk plus headroom is
   `unsupported[{reason_code: insufficient_capacity}]`.
10. **Digest.** `digest` is sha256 over the canonical JSON of the closure with
    the digest field cleared. All lists are sorted, so equal inputs give equal
    digests. `sources` (catalog, scope, host requirement resolver, analyzer
    use) is part of the digest.

## Declarations the closure reads

The `deployment` block is additive and shared by `service.json` and
`resource.json` (`common.schema.json#/definitions/deploymentDeclaration`):

```json
"deployment": {
  "supported_targets": [{ "os": "linux", "distribution": "ubuntu", "version": "24.04", "architectures": ["amd64", "arm64"] }],
  "persistent_data": [{ "id": "application-records", "owner": "postgres", "binding": "schema:app", "backup_provider": "data-backup-manager", "migration_owner": "scenario" }],
  "listeners": [{ "id": "public-http", "port": "api", "visibility": "public_via_edge", "readiness": { "type": "http", "path": "/health" } }],
  "recovery": { "code_rollback": "compatible_predecessor_only", "schema_strategy": "expand_contract" }
}
```

Ports not named in `listeners` are `private`. A listener without its own
`readiness` inherits the owning component's `run.readiness`.

## Wire shape

```json
{
  "schema_version": "1",
  "scenario_id": "app",
  "environment": "production",
  "platform": { "os": "linux", "arch": "amd64" },
  "components": [
    { "id": "store", "kind": "resource", "required": true, "optional_selected": false,
      "reasons": [
        { "kind": "declared_by", "from": "records-service", "detail": "declared in service.json" },
        { "kind": "transitive_via", "from": "records-service", "detail": "path app -> records-service -> store" } ],
      "version": "16", "content_identity": "sha256:…",
      "artifact": { "platform": "linux-amd64", "name": "store_linux_amd64", "digest": "sha256:…", "mode": "bundled-service", "eligibility": "eligible" } }
  ],
  "persistent_data": [ { "id": "application-records", "owner": "store", "binding": "schema:app", "backup_provider": "data-backup-manager", "migration_owner": "scenario", "declared_by": "scenario:app" } ],
  "listeners": [ { "id": "app/api", "owner": "scenario:app", "port_name": "api", "visibility": "public_via_edge", "readiness": { "type": "http", "path": "/health" } } ],
  "privileges": [ { "effect": "elevated", "subject": "safeguard:firewall", "safeguard": "firewall", "reason": "public listener" } ],
  "capacity": { "cpu": 2.75, "memory_bytes": 2952790016, "disk_bytes": 10200547328, "transient_update_headroom_bytes": 17448304640, "headroom_basis": "largest_declared_disk_footprint", "fit": "unknown", "contributions": [] },
  "unsupported": [],
  "sources": { "catalog": "repo:Vrooli", "scope": "bundle", "host_requirements": "packages/hostreq", "analyzer_tool": "scenario-dependency-analyzer", "analyzer_used": true },
  "digest": "sha256:…"
}
```

Component kinds: `scenario`, `resource`, `package`, `native_artifact`,
`tool`, `safeguard`, `credential_descriptor`. Reason kinds: `declared_by`,
`transitive_via`, `system_required`, `platform_artifact`, `credential_of`,
`safeguard_of`, `selected_by`.

## Errors

| Code | HTTP | Meaning |
|---|---|---|
| `closure_unavailable` | 424 | a catalog could not be read (missing scenario/resource, unreadable manifest, analyzer or host-requirement outage); `details.component` / `details.declared_by` / `details.source` say which |
| `closure_cycle` | 422 | required-edge dependency cycle; `details.path` |
| `closure_conflict` | 422 | contradictory operating modes; `details.component` and the modes |
| `invalid_request` | 400 | missing scenario, os, or arch; unknown scope |

## Configuration handoff

`closure.ToSelection(closure, target, overrides)` maps a closure onto the
shared `setup/v1.Selection`: `scenarios`, `optional_resources` (operator-
selected optional resources only; required resources are derived by setup from
the manifests), `host_tools`, `host_safeguards`, `credential_addresses`,
`operating_mode`, `transient_headroom_reserve_bytes`, with explicit
`field_presence`. The same closure always produces identical Selection bytes.

## Manifest binding

With a closure source configured, the manifest refresher fills
`dependencies.scenarios`, `dependencies.resources`, `bundle.scenarios` and
`bundle.resources` from the closure and records `dependencies.closure_digest`.
A closure failure refuses the refresh; the manifest is never rebuilt from a
stale dependency snapshot.

## Explanations are the closure

The API returns the closure document itself. The CLI and UI render the same
`reasons` and `unsupported` entries; no surface adds prose of its own, so an
operator sees the same explanation everywhere.
