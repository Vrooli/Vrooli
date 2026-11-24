# Configuration Architecture

This page is the load-bearing reference for *where each configuration decision lives*. Every other page in this folder is a consumer of the table below.

## The three categories

Every configuration value in Vrooli fits exactly one category. Mixing them is the failure mode this layering prevents.

- **Declarative manifest** — what something *is*. Hand-edited, source-controlled, slow-changing. Lives in `service.json` / `resource.json` / `tool.json` / `safeguard.json`.
- **Computed analysis** — what an analyzer *figured out* about something. Tool output is regenerable and lives in analyzer reports or generated schema artifacts, never in the authored scenario manifest's `tier_feasibility` input.
- **Operator state** — what this install *chose*. Wizard- or hand-edited per machine. Lives beneath the operator runtime home; see the [runtime-home contract](../repo-contract.md#operator-runtime-home-vrooli).

These don't reduce to each other. A scenario's manifest declares `runtime.kind = "long_running"`; the operator's state says `auto_restart: true`; an analyzer's output says `tier-3-mobile fitness 0.4`. Same scenario, three independent statements, three independent files.

Generated runtime state is outside this configuration model. Setup completion markers, resource-populated markers, runtime registries (including the SQLite port-claim registry), process records, logs, and telemetry are mutable operational artifacts and live under `~/.vrooli/`, not project `.vrooli/`. Project-specific setup/resource markers are rooted at `~/.vrooli/state/projects/<project-key>/`, where the project key is derived from the cleaned absolute project root.

## Source-of-truth table

For each operator-visible decision, exactly one file is the source of truth. Other files may reference or derive from it; they never override it.

| Decision | Source of truth | Surfaced by |
|---|---|---|
| Whether a scenario is enabled | `operator state` → `scenarios.<name>.enabled` (overrides manifest) | onboarding scenarios step |
| Whether a scenario is system-required (uneditable) | `scenarios/<name>/.vrooli/service.json` → `service.system_required` | onboarding scenarios step (renders as locked-on) |
| Scenario's recommended runtime shape | `scenarios/<name>/.vrooli/service.json` → `runtime.kind` | onboarding (filter/sort), lifecycle orchestrator |
| Scenario's recommended auto-restart default | `scenarios/<name>/.vrooli/service.json` → `runtime.auto_restart_default` | onboarding (per-scenario toggle default) |
| Scenario's actual auto-restart on this install | `operator state` → `scenarios.<name>.auto_restart` (override of default) | onboarding operating-mode step |
| Scenario → scenario dependencies | `scenarios/<name>/.vrooli/service.json` → `dependencies.scenarios` | onboarding scenarios step (cascade hint) |
| Whether a resource is enabled | `operator state` → `resources.<name>.enabled` | onboarding resources step |
| Resource → resource dependencies | `resources/<name>/resource.json` → `dependencies` | onboarding resources step (cascade) |
| Resource credential descriptors | `resources/<name>/resource.json` → `credentials.descriptors[]` | onboarding secrets step |
| Where an ordinary credential value is stored | probed native credential authority keyed by descriptor logical identity + field | control plane runtime injection |
| Host tool opt-in | `operator state` → `host_tools.<name>.opted_in` (override of manifest `required`) | onboarding host step |
| Host safeguard opt-in | `operator state` → `host_safeguards.<name>.opted_in` (override of manifest `required`) | onboarding host step |
| Safeguard risk indicator | `internal/safeguards/<name>/safeguard.json` → `risk` | onboarding host step (risk column) |
| What host tools/safeguards exist | filesystem: `internal/tools/<name>/tool.json`, `internal/safeguards/<name>/safeguard.json` (drift-protected by `internal/runtime/manifests_test.go`) | onboarding host step (registry source) |
| What integration connector types exist | filesystem: `scenarios/integration-hub/connectors/<id>/connector.json` (pilot currently uses an API-key connector) | integration-hub UI; onboarding integrations step |
| Connection instances (OAuth tokens, API keys for connectors) | Credential authority under identity `vrooli/integrations/<connector>/<connection_id>` + integration-hub metadata state | integration-hub UI |
| Which integrations a scenario needs | `scenarios/<name>/.vrooli/service.json` → `integrations[]` (declared connector + scopes + purpose; binding contract in progress) | onboarding integrations step |
| Which connection a scenario actually uses | `operator state` → `integrations.<scenario>.<connector>` (binding contract in progress) | onboarding integrations step |
| Connector-level secrets (e.g. OAuth client_secret) | Credential authority under identity `vrooli/integrations/connectors/<connector_id>` (needed as drivers are added) | integration-hub setup, not user-facing |
| Active profile | `operator state` → `active_profile` | onboarding purpose intake and profile evaluator |

Anything not in this table is out of scope for the wizard.

## Resolution order

To answer "what is the *effective* value of X right now?", the system applies a fixed resolution order per concern. Onboarding only ever writes to `operator-state.json`; runtime-readers walk this order:

### Per-scenario auto-restart

```
1. operator state → scenarios.<name>.auto_restart   (if present)
2. scenarios/<name>/.vrooli/service.json → runtime.auto_restart_default   (if present)
3. schema default (false)
```

### Whether a scenario is enabled

```
1. scenarios/<name>/.vrooli/service.json → service.system_required = true   →   ALWAYS ENABLED (no operator override possible)
2. operator state → scenarios.<name>.enabled   (if present)
3. default (disabled — operator must opt in)
```

### Whether a host safeguard is applied

```
1. operator state → host_safeguards.<name>.opted_in   (if present)
2. fall back to manifest's `required` field (which mirrors hostRequirement defaults in service.json)
3. default (not applied — safeguards are opt-in by nature)
```

### What credential metadata is shown to the operator

```
1. resources/<name>/resource.json → credentials.descriptors[] — render the
   logical identity, field, label, description, and obtain URL without a value
2. fall back: empty (resource declares no credentials)
```

### How scenario integration requirements and bindings work today

These are separate contracts:

- Integration Hub owns explicit connection bindings. The
  [`Connection.bindings[]` contract](../../packages/proto/schemas/common/v1/integrations.proto)
  records a scenario slug and optional context on the Hub connection.
  [`hub.go`](../../scenarios/integration-hub/api/hub.go) implements bind and
  unbind; bind checks the caller-supplied required scopes against the
  connection's granted scope metadata. The caller performs the binding.
- Onboarding reads `service.json#integrations[]` and reports each declared
  requirement as deferred. Its
  [`readiness.go`](../../scenarios/vrooli-onboarding/api/readiness.go) does not
  query Hub, compare manifest scopes with a connection, resolve a binding, or
  block scenario startup. It does not read or write an
  `operator-state.integrations` path.

A declaration may request multiple connections, but Onboarding does not resolve
those contexts to Hub bindings. Keep that behavior deferred until a tested
manifest-to-connection workflow defines scope checks, context matching, and
startup semantics. See the
[scenario declaration contract](scenarios.md#scenario-integration-declarations),
[Hub connection contract](integrations/connections.md), and Onboarding's
[integration ownership guide](../../scenarios/vrooli-onboarding/docs/concepts/INTEGRATIONS.md).

## Generated operator-control inventory

The operator-control catalog is generated from provider descriptors at runtime;
the onboarding wizard and the CLI consume the same inventory. Each descriptor
must name its owner, scope, purpose, sensitivity, applicability, disposition,
permission provenance, and supported lifecycle operations. A descriptor with a
`deferred` or `unsupported` disposition must also provide the reason, so a
required blocker cannot be mistaken for a missing input. Renderers must not
add a capability-specific control branch.

The current control census has these authoritative owners:

| Control family | Authority | Disposition | Durable source | Notes |
|---|---|---|---|---|
| scenario/resource enablement, capacity, update posture, trust posture, notifications | operator-state authority | configurable | `operator state` | Validated by `operatorstate` and manifest-backed resolvers. |
| host tools and safeguards | host-requirement owner | configurable or required | manifests plus operator state | Optional decline is distinct from an absent choice and survives restart. |
| credential-store access and escrow | credential authority | configurable | credential authority and capability evidence | Secret values are write-only for the action. |
| storage recovery standing approvals | storage-manager | configurable | storage-manager approval owner | Approval is provider-scoped and revocable; it is not an arbitrary command grant. |
| durable backup and recovery evidence | data-backup-manager | protected | data-backup-manager evidence | Onboarding can refresh evidence but cannot mutate the backup policy. |
| external integrations without an owner | named future owner | deferred or unsupported | no binding is written | Required cases remain blockers with an explicit reason. Profiles are owned by onboarding and are not deferred. |

For a machine-readable descriptor inventory, use `vrooli capability catalog
--json`; `vrooli capability status --json` returns the live status projection.

## Host update-control modes

The host update-control mode is operator state, not a manifest default. This
host records `guard` in `operator state`; the control plane owns
the guard, freeze, and coupled-set decisions. A scenario may observe and
report update state, but it must not repair host packages privately.

The effective mode resolves in this order:

1. `operator state.update_control`.
2. The control-plane default (`observe` when no operator choice exists).

`guard` preserves declared coupled sets and requires a maintenance window for
live replacement. Runtime checks report drift and missing evidence; they do
not silently change the mode or apply package updates.

## Scenario runtime configuration

Scenario runtime configuration is distinct from all three source-of-truth
categories above. A scenario's `.vrooli/config.json` and schema-backed API
settings hold live application policy such as notification templates,
sensitivity mapping, and event subscriptions. `service.json` declares the
scenario and lifecycle contract; `operator state` records
installation choices. Neither file is a substitute for a scenario's live
runtime configuration, and changing an API-backed setting must not require a
scenario restart.

## Naming conventions for shared types

When a definition in `common.schema.json` is referenced from multiple schemas, the **property name on each consumer should match the definition name** if the type plays a single semantic role at every call site. Divergence between property name and type name is reserved for the Kubernetes pattern — one type that genuinely plays multiple roles (e.g. one `Probe` shape used as `livenessProbe`, `readinessProbe`, `startupProbe`). In that case the property name encodes the role; the type name encodes only the shape.

Worked example: `verificationCheck` (in `common.schema.json`) is referenced from both `tool.schema.json` and `safeguard.schema.json` for the same role — one-shot post-action verification. The property is named `verificationCheck` in both manifests. There is no `installCheck` / `applyCheck` divergence because the role is identical.

Counter-example (planned): when `runtimeProbe` lands as a shared type and is referenced from the canonical service and resource schemas, both consumers use the property name `runtimeProbe` (or `runtimeProbes` for arrays). Splitting into `livenessProbe` / `readinessProbe` would only be justified if the same shape acquired multiple distinct roles within a single consumer.

## Open work items

Items intentionally deferred from the current schema bundle. Each is a future-conversation decision; do not resolve speculatively.

- **`integration-hub` expansion** — the Hub pilot owns metadata-only connection lifecycle and the OpenRouter API-key connector. Remaining work includes connector manifests, OAuth/device-flow drivers, onboarding binding, and multi-instance consumption. See [`integrations/connectors.md`](integrations/connectors.md) and [`integrations/connections.md`](integrations/connections.md).
- **External-auth credential schema** — concrete schema dispatch for `oauth_web` / `oauth_device` / `external_sign_in_command` / `app_password` patterns. Catalog lives in [`integrations/external-auth.md`](integrations/external-auth.md); the schema lands as part of the integration-hub work above. The current `secretDescriptor` continues to cover paste-string resource secrets independently.
- **Schema-types unification (Wave 2)** — Wave 1 landed: `verificationCheck` (one-shot post-action verification) was lifted to `common.schema.json` and is referenced by both `tool.schema.json` and `safeguard.schema.json`, replacing the two duplicate inline definitions. Wave 2 is the harder consolidation: continuous monitoring declarations in the canonical service and resource schemas need to fold into a shared `runtimeProbe` definition. The naming-convention reconciliation (snake_case vs camelCase) and the Go-consumer migration on both sides are real work; deferred to its own plan. `dependencies` shape overlap between `service.dependencies.scenarios` and `service.tier_feasibility.dependencies` is also Wave 2 territory.
