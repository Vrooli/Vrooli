# Configuration Architecture

This page is the load-bearing reference for *where each configuration decision lives*. Every other page in this folder is a consumer of the table below.

## The three categories

Every configuration value in Vrooli fits exactly one category. Mixing them is the failure mode this layering prevents.

- **Declarative manifest** — what something *is*. Hand-edited, source-controlled, slow-changing. Lives in `service.json` / `resource.json` / `tool.json` / `safeguard.json`.
- **Computed analysis** — what an analyzer *figured out* about something. Tool output, regenerable. Lives in the `service.deployment` block and in generated schema artifacts.
- **Operator state** — what this install *chose*. Wizard- or hand-edited per machine. Lives in `.vrooli/operator-state.json`.

These don't reduce to each other. A scenario's manifest declares `runtime.kind = "long_running"`; the operator's state says `auto_restart: true`; an analyzer's output says `tier-3-mobile fitness 0.4`. Same scenario, three independent statements, three independent files.

## Source-of-truth table

For each operator-visible decision, exactly one file is the source of truth. Other files may reference or derive from it; they never override it.

| Decision | Source of truth | Surfaced by |
|---|---|---|
| Whether a scenario is enabled | `.vrooli/operator-state.json` → `scenarios.<name>.enabled` (overrides manifest) | onboarding scenarios step |
| Whether a scenario is system-required (uneditable) | `scenarios/<name>/.vrooli/service.json` → `service.system_required` | onboarding scenarios step (renders as locked-on) |
| Scenario's recommended runtime shape | `scenarios/<name>/.vrooli/service.json` → `runtime.kind` | onboarding (filter/sort), lifecycle orchestrator |
| Scenario's recommended auto-restart default | `scenarios/<name>/.vrooli/service.json` → `runtime.auto_restart_default` | onboarding (per-scenario toggle default) |
| Scenario's actual auto-restart on this install | `.vrooli/operator-state.json` → `scenarios.<name>.auto_restart` (override of default) | onboarding operating-mode step |
| Scenario → scenario dependencies | `scenarios/<name>/.vrooli/service.json` → `dependencies.scenarios` | onboarding scenarios step (cascade hint) |
| Whether a resource is enabled | `.vrooli/operator-state.json` → `resources.<name>.enabled` | onboarding resources step |
| Resource → resource dependencies | `resources/<name>/resource.json` → `dependencies` | onboarding resources step (cascade) |
| Resource credential descriptors | `resources/<name>/resource.json` → `credentials.env[]` (each item: bare string OR `secretDescriptor`) | onboarding secrets step |
| Where a credential value is stored | Vault path declared at `resources/<name>/resource.json` → `credentials.secret_ref` | secrets-manager scenario; runtime via `packages/api-core/secrets` |
| Host tool opt-in | `.vrooli/operator-state.json` → `host_tools.<name>.opted_in` (override of manifest `required`) | onboarding host step |
| Host safeguard opt-in | `.vrooli/operator-state.json` → `host_safeguards.<name>.opted_in` (override of manifest `required`) | onboarding host step |
| Safeguard risk indicator | `internal/safeguards/<name>/safeguard.json` → `risk` | onboarding host step (risk column) |
| What host tools/safeguards exist | filesystem: `internal/tools/<name>/tool.json`, `internal/safeguards/<name>/safeguard.json` (drift-protected by `internal/runtime/manifests_test.go`) | onboarding host step (registry source) |
| What integration connector types exist | filesystem: `scenarios/integration-hub/connectors/<id>/connector.json` (deferred) | integration-hub UI; onboarding integrations step |
| Connection instances (OAuth tokens, API keys for connectors) | Vault under `secret/vrooli/integrations/<connector>/<connection_id>` + integration-hub state (deferred) | integration-hub UI |
| Which integrations a scenario needs | `scenarios/<name>/.vrooli/service.json` → `integrations[]` (declared connector + scopes + purpose; deferred) | onboarding integrations step |
| Which connection a scenario actually uses | `.vrooli/operator-state.json` → `integrations.<scenario>.<connector>` (deferred) | onboarding integrations step |
| Connector-level secrets (e.g. OAuth client_secret) | Vault under `secret/vrooli/connectors/<connector_id>` (deferred) | integration-hub setup, not user-facing |
| Active profile | `.vrooli/operator-state.json` → `active_profile` | reserved for future use; profiles deferred |

Anything not in this table is out of scope for the wizard.

## Resolution order

To answer "what is the *effective* value of X right now?", the system applies a fixed resolution order per concern. Onboarding only ever writes to `operator-state.json`; runtime-readers walk this order:

### Per-scenario auto-restart

```
1. .vrooli/operator-state.json → scenarios.<name>.auto_restart   (if present)
2. scenarios/<name>/.vrooli/service.json → runtime.auto_restart_default   (if present)
3. schema default (false)
```

### Whether a scenario is enabled

```
1. scenarios/<name>/.vrooli/service.json → service.system_required = true   →   ALWAYS ENABLED (no operator override possible)
2. .vrooli/operator-state.json → scenarios.<name>.enabled   (if present)
3. default (disabled — operator must opt in)
```

### Whether a host safeguard is applied

```
1. .vrooli/operator-state.json → host_safeguards.<name>.opted_in   (if present)
2. fall back to manifest's `required` field (which mirrors hostRequirement defaults in service.json)
3. default (not applied — safeguards are opt-in by nature)
```

### What credential metadata is shown to the operator

```
1. resources/<name>/resource.json → credentials.env[]   — for each entry:
     a. if string  → render bare label (legacy form)
     b. if object  → render full secretDescriptor (label, description, obtain_url, default_hint)
2. fall back: empty (resource declares no credentials)
```

### Which connection a scenario uses for a given connector (deferred)

Lands when `integration-hub` ships. The intent:

```
1. .vrooli/operator-state.json → integrations.<scenario>.<connector>   (operator binding)
2. if absent and scenario marks the integration as required → wizard error: "no connection bound"
3. if absent and scenario marks the integration as optional → scenario runs in degraded mode
```

For `multi: true` scenarios (persona-actor case), the operator-state value is an array of `{ context, connection_id }` rather than a single `connection_id`. The scenario picks at runtime by `context`.

These resolution orders are the contract. UIs and runtime code consume them; new surfaces should reuse the same evaluator rather than reimplementing.

## Open work items

Items intentionally deferred from the current schema bundle. Each is a future-conversation decision; do not resolve speculatively.

- **`integration-hub` scenario** — the home for both connector definitions (declarative manifests of *how to talk to provider X*) and connection instances (operator's actual authenticated sessions). Owns the auth-flow drivers, the Vault layout for connection tokens, the bind/unbind/refresh CLI, and the storage for unbound/scratch credentials. Non-trivial work (probably 2–4 weeks when scoped); blocking the integrations wizard step beyond the current empty placeholder. First concrete connector is most likely `fal-api` (paste-string) followed by `github-oauth`. See [`integrations/connectors.md`](integrations/connectors.md) and [`integrations/connections.md`](integrations/connections.md) for the full design intent.
- **External-auth credential schema** — concrete schema dispatch for `oauth_web` / `oauth_device` / `external_sign_in_command` / `app_password` patterns. Catalog lives in [`integrations/external-auth.md`](integrations/external-auth.md); the schema lands as part of the integration-hub work above. The current `secretDescriptor` continues to cover paste-string resource secrets independently.
- **Profiles** — bundled selections of scenarios + resources + secrets (e.g. "engineering", "marketing", "homelab"). `operator-state.json` reserves `active_profile` for this; `profile.schema.json` lands when the second concrete profile exists. See [`profiles.md`](profiles.md).
- **Schema-types unification (separate plan)** — `healthCheck` is currently defined four times across `tool.schema.json`, `safeguard.schema.json`, `deployment.schema.json`, and `resource.schema.json` `health_checks`. `dependencies` shapes overlap between `service.dependencies.scenarios` and `service.deployment.dependencies`. Consolidating these into shared `common.schema.json` defs is the follow-up plan after the current bundle lands.
- **Renaming the `service.deployment` block** — the name overlaps with `deployment.schema.json` (runtime config). One block is hand-edited config; the other is analyzer output. Rename to `analysis` or `feasibility` is on the table for the unification plan.
