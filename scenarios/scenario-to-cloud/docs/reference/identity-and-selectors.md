# Identity, selectors and typed errors

This reference defines the stable identities scenario-to-cloud exposes, the
selector forms that resolve to them, how contract versions are negotiated, and
the typed error model every client consumes. The wire contracts live in
`packages/proto/schemas/scenario-to-cloud/v1/{identity,errors,deployments}`;
the Go mirrors live in `api/identity` and `api/apierrors`.

## Identities

| Identity | Fields | Mutability | Authority |
|---|---|---|---|
| `DeploymentRef` | `id`, `scenario_id`, `environment`, `target` | `id` never changes; the target binding changes only through an explicit rebind | Cloud (this scenario) |
| `TargetRef` | `machine_id`, `node_id`, `enrollment_generation`, `transport`, `locator` | Machine and node identity come from the Bridge; the locator is mutable reachability metadata | Bridge for identity, Cloud for the binding |
| `TargetLocator` | `host`, `port`, `user`, `workdir` | May change at any time; never identity | Cloud |
| `ReleaseRef` | `digest`, `provenance_ref`, `closure_digest`, `configuration_digest` | Immutable | Ramp / artifact owner |
| `OperationRef` | `id`, `request_key`, `plan_digest`, `fence` | Immutable once admitted | Cloud operation owner |

Rules:

- A deployment is one scenario installed in one environment on one target.
  The database enforces `UNIQUE (scenario_id, environment, target_key)`.
- `environment` defaults to `production`. Records that predate the column
  were converted with that default (or the manifest's `environment` when it
  was set).
- `target_key` is derived on every write: `machine:<machine_id>` when the
  target is Bridge-enrolled, otherwise `host:<locator.host>`. The prefixes keep
  the two namespaces from colliding. It is denormalised into a column because
  SQLite (the routed test engine) cannot index a JSON expression the way
  PostgreSQL can, and both engines must enforce the same rule.
- `transport` is `bridge` or `ssh`. A revoked Bridge enrollment is a typed
  `reach_unavailable`; there is no silent fallback to SSH.
- `fence` is a per-deployment counter incremented when an operation acquires
  the deployment. Every target-side effect carries the fence and refuses a
  lower one.
- The manifest's `target.vps` block (`host`, `port`, `user`, `workdir`) seeds
  the locator. It is not part of the target binding's identity. The SSH key
  is the credential binding `vrooli/scenario-to-cloud:ssh-key`, never a
  manifest field.

## Selectors

Exactly one of the four forms is accepted. A scenario name alone is never
enough, because one scenario may be installed many times.

| Form | Facets | REST query | Connect field |
|---|---|---|---|
| by id | `id` | `?id=` | `selector.id` |
| by scenario and environment | `scenario_id` + `environment` | `?scenario=&environment=` | `selector.scenario_id`, `selector.environment` |
| by scenario and domain | `scenario_id` + `domain` | `?scenario=&domain=` | `selector.scenario_id`, `selector.domain` |
| by scenario and host | `scenario_id` + `host` | `?scenario=&host=` | `selector.scenario_id`, `selector.host` |

Resolution outcomes:

| Matches | Result |
|---|---|
| 0 | `deployment_not_found` (404) |
| 1 | the `DeploymentRef` |
| more than 1 | `deployment_selector_ambiguous` (409) with `details.candidates` listing every match and a `next_action` pointing at selection by id |

A selector that is not one of the four forms (for example `scenario` alone,
`id` combined with another facet, or two facets beside `scenario`) is
`deployment_selector_invalid` (400) and never reaches storage.

Surfaces:

- REST: `GET /api/v1/deployments/resolve?...` returns
  `{"schema_version":"1","ref":{...},"timestamp":...}`.
- Connect: `DeploymentsService.ResolveDeployment` at
  `/vrooli.scenario_to_cloud.v1.deployments.DeploymentsService/ResolveDeployment`.
- Manifest submission (`POST /api/v1/deployments`) resolves
  `(scenario, environment, host)` from the manifest. One match updates the
  record in place with the same id; no match creates a record; more than one
  match is the ambiguous conflict.

## Version negotiation

Every typed response carries `schema_version` (currently `"1"`). A client
that receives a version it does not understand must treat the payload as
unsupported instead of reading fields positionally. A request that names a
mandatory field or capability the server does not support is refused with
`unsupported_schema_version` or `unsupported_capability`; the server never
silently ignores a mandatory field.

Proto messages evolve additively within `v1`. Breaking changes require a new
version directory (`v2`) and a deprecation cycle for the old one.

## Typed errors

Every non-2xx REST body is:

```json
{
  "error": {
    "code": "deployment_selector_ambiguous",
    "message": "Selector matches more than one deployment; select by id or add environment",
    "retryable": false,
    "next_action": {"owner": "scenario-to-cloud", "kind": "selector", "reference": "id", "label": "Select by deployment id"},
    "details": {"candidates": [{"id": "..."}]}
  }
}
```

Keys are proto field names (snake_case). The HTTP status is derived from the
code on the server (`apierrors.StatusFor`) and is never part of the body.
Connect RPC failures attach the same `errors.v1.Error` message as an error
detail, so REST and Connect clients read one code set.

Clients:

- Go: `apierrors.FromHTTP(status, body)` / `apierrors.FromResponse(resp)`.
- TypeScript: `parseApiError(status, body)` in `ui/src/lib/apiErrors.ts`,
  which parses through the generated `ErrorSchema`.
- CLI exit codes: `0` ok, `1` failed, `2` refused or conflict, `3` pending or
  needs input, `124` observer timeout (`apierrors.ExitCodeFor`).

Stable codes are constants in `api/apierrors/codes.go`. The ones defined in
this phase:

| Code | Status | Meaning |
|---|---|---|
| `invalid_json`, `invalid_request` | 400 | Malformed request |
| `deployment_selector_invalid` | 400 | Selector is not one of the four forms |
| `unsupported_schema_version` | 400 | Client named a version the server does not speak |
| `unauthenticated` | 401 | No principal |
| `forbidden_scope`, `forbidden_target` | 403 | Principal lacks the scope or the target grant |
| `deployment_not_found`, `operation_not_found` | 404 | No record |
| `deployment_selector_ambiguous` | 409 | More than one match |
| `deployment_identity_conflict` | 409 | Second record for one (scenario, environment, target) |
| `request_key_conflict` | 409 | Same request key, different plan digest |
| `plan_stale`, `plan_digest_mismatch`, `operation_conflict`, `fence_stale` | 409 | Concurrency refusals |
| `manifest_invalid`, `release_verification_failed` | 422 | Content refused |
| `closure_unavailable` | 424 | A required closure input is missing |
| `needs_input` | 428 | Operation is waiting for the operator |
| `unsupported_capability` | 501 | Target or server cannot perform the action |
| `reach_unavailable`, `health_unknown` | 503 | Target cannot be reached or observed |
| `internal` | 500 | Unexpected failure; `details.cause` carries the operator text |

## Request keys and operations

`cloud_operations` is keyed by `UNIQUE (deployment_id, request_key)`. Admitting
the same request key twice with the same `plan_digest` returns the existing
operation (idempotent replay). The same key with a different `plan_digest` is
`request_key_conflict` and nothing is written. Request keys are scoped per
deployment, so two deployments may reuse one key.
