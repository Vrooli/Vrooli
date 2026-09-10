# Bringing the next scenario onto the cloud ramp

A scenario reaches a VPS through **declarations and configuration only**. The
cloud ramp (closure, executable plan, durable operations, target owner, edge,
credentials, health) reads what the scenario and its resources declare; it has
no product-specific branch and no cloud source is edited to adopt a new
scenario. The executable example is the `newcomer-service` fixture:

> [CODE: fixtures/workloads/newcomer-service] — declaration tree, seed, oracle
> [CODE: api/generality] — the in-process journey that proves it (`go test ./generality/`)
> [DOC: reference/closure.md] · [DOC: reference/executable-plan.md] · [DOC: reference/credential-lifecycle.md] · [DOC: guides/manifest-reference.md]

## 1. What the scenario declares (`.vrooli/service.json`)

Everything below is read by the closure resolver. Values are from
`fixtures/workloads/newcomer-service/declarations/scenarios/newcomer-service/.vrooli/service.json`.

| Block | Field | Example | Why the ramp needs it |
|---|---|---|---|
| `dependencies.scenarios` | `<id>: {enabled, required, startup_policy}` | `ledger-service: {enabled: true, required: true, startup_policy: must_start}` | Transitive closure with a `declared_by` reason; started before the workload |
| `dependencies.resources` | `<id>: {enabled, required, startup_policy}` | `ledger-store: {…, must_start}` | Resource artifacts, credentials and persistent data join the closure |
| `credentials.descriptors[]` | `logical_id`, `field`, `env`, `required`, `label` | `fixture/newcomer` / `webhook-signing-key` / `NEWCOMER_WEBHOOK_SIGNING_KEY` | Binding planned per deployment; a required descriptor without a value is a `needs_input` handoff, never a prompt in the plan |
| `ports` | `<name>: {env_var, range}` | `api`, `callback` | Listener names the `deployment.listeners` block refers to; the manifest assigns the numbers |
| `hostTools[]`, `hostSafeguards[]` | `name`, `required`, `reason`, `platforms`, `privilege` | `curl`, `firewall` (elevated) | Host requirements and privileges in the closure; `host.prepare` and `edge.firewall.allow` actions |
| `tier_feasibility.tiers.tier-1-local.requirements` | `ram_mb`, `disk_mb`, `cpu_cores` | 256 / 512 / 0.25 | Capacity fit and transient update headroom |

### The `deployment` block

```json
"deployment": {
  "supported_targets": [
    { "os": "linux", "distribution": "ubuntu", "version": "24.04", "architectures": ["amd64"] }
  ],
  "listeners": [
    { "id": "public-api", "port": "api", "visibility": "public_via_edge",
      "readiness": { "type": "http", "path": "/health", "timeout_ms": 5000 } },
    { "id": "provider-callback", "port": "callback", "visibility": "private",
      "readiness": { "type": "port_open" } }
  ],
  "persistent_data": [
    { "id": "newcomer-orders", "owner": "ledger-store", "binding": "schema:newcomer",
      "backup_provider": "data-backup-manager", "migration_owner": "scenario" }
  ],
  "recovery": { "code_rollback": "compatible_predecessor_only", "schema_strategy": "expand_contract" }
}
```

| Field | Values | Effect |
|---|---|---|
| `supported_targets[].architectures` | `amd64`, `arm64` | A platform outside the list is `unsupported_platform` in the closure and the plan refuses with `unsupported_capability` |
| `listeners[].visibility` | `public_via_edge`, `private` | Only `public_via_edge` listeners get an edge route (`ui` at the apex, any other port name at `<port>.<domain>`); every other port, database and management listener stays private. A scenario that declares no listeners keeps the legacy contract: its `ui` port is the one public route |
| `listeners[].readiness` | `http` (+`path`), `port_open` | What `verify.readiness` and the edge wait on |
| `persistent_data[]` | `id`, `owner`, `binding` (`schema:<name>`, `database:<name>`, `dir:<relative>`, `bucket:<name>`), `backup_provider`, `migration_owner` | Recovery point before every activation, data bound outside release trees, rollback admission |
| `recovery` | `code_rollback`: `compatible_predecessor_only` \| `any_predecessor` \| `none`; `schema_strategy`: `expand_contract` \| `explicit_restore` \| `none` | Maintenance vs side-by-side strategy, rollback eligibility on `release.retain_predecessor` |

## 2. What a resource declares (`resource.json`)

From `declarations/resources/ledger-store/resource.json`:

- `deployment.profiles.desktop.<os>.architectures` and
  `managed_service.artifact.sha256_by_platform` — the artifact the closure binds
  per platform. A missing architecture is `missing_platform_artifact` naming
  the resource (P24-A03).
- `credentials.descriptors[]` — the resource's own credential (generated per
  install, materialised only to consumers that declared the resource).
- `deployment.persistent_data[]` and `deployment.backup` (`provider`,
  `consistency`, `dump`/`restore` argv) — the engine-native recovery point.

## 3. What the operator supplies (configuration, not code)

| Item | Where | Notes |
|---|---|---|
| Target | `scenario-to-cloud manifest init --scenario <id> --host <host> --domain <domain>`, then `deployment create <manifest>` | One deployment record per `(scenario, environment, target)`. Bridge-enrolled targets bind `transport: bridge` (enroll with `vrooli-bridge onboard`); otherwise the bounded SSH adapter with an explicit key |
| Environment | `environment` in the manifest (`staging`, `production`) | Two environments of one scenario are distinct deployments with distinct target keys, credential binding ids, edge domains and receipts (P24-A02) |
| Ports | `ports.<name>` | One number per declared listener name |
| Edge | `edge.domain`, `edge.caddy.{enabled,email}`, DNS records | Routes are derived from the listeners; DNS must point every public host at the target before the external claim |
| Credentials | Descriptor values through the onboarding handoff (`vrooli-onboarding://deployments/<id>/resume/…`) or the credentials API | Values never enter a plan, argv, receipt or log; the plan stays `needs_input` until every required descriptor is satisfied |

## 4. Preview, explain gaps, apply

```bash
# The closure with reasons and unsupported entries
curl "http://localhost:${API_PORT}/api/v1/closure?scenario=newcomer-service&environment=production&os=linux&arch=amd64"

# The executable plan (digest, changes, data effects, downtime, handoff)
scenario-to-cloud deployment plan --scenario newcomer-service --environment production

# Admit the reviewed digest as a durable operation and wait once
scenario-to-cloud deployment apply --scenario newcomer-service --environment production --plan-digest <digest> --request-key <key>
```

What the preview tells you:

- `outcome: needs_input` with one `handoff` — a required descriptor has no
  value. `handoff.missing` names the exact `logical_id:field` addresses.
- `unsupported_capability` naming `<component>:<reason_code>` — a dependency
  cannot be served on this platform (for example
  `ledger-store:missing_platform_artifact`); the closure's `unsupported[]`
  carries the detail.
- `closure_unavailable` — a declaration could not be read; nothing is
  guessed.
- `changes[]` and `data_effects[]` — every action with its owner operation,
  effect, retry class and recovery; every declared persistent-data binding
  the recovery point will cover.

## 5. What is proven for the fixture (package lane)

`go test ./generality/` in `api/` drives the ramp in-process against a fake
`cloud-target` owner: closure → plan → durable admission → worker execution →
target receipts → health observation → evidence cell, once, as `staging` and
`production` on two targets, across an update that keeps the declared public
route and private callback listener, and beside the `stateless-web` fixture
whose plan digest must equal its recorded golden. The real-VPS and QEMU
lanes remain pending the external inputs recorded in the certification
ledgers.

## 6. Checklist

1. Declare dependencies, credentials, ports and the `deployment` block; run
   `vrooli scenario validate <id>` (schema).
2. Declare artifacts, credentials, persistent data and backup hooks on any
   new resource.
3. Create the manifest and the deployment record for each environment.
4. Preview the closure and the plan; resolve every `needs_input` and
   `unsupported` entry through configuration or declarations.
5. Apply the reviewed digest; wait on the operation; read the health and
   edge observations.

No step edits `scenarios/scenario-to-cloud`.
