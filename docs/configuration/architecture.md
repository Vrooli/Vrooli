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
| External-auth tokens (OAuth, device flow) | Vault under reserved prefix; schema TBD | onboarding integrations step (deferred) |
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

These resolution orders are the contract. UIs and runtime code consume them; new surfaces should reuse the same evaluator rather than reimplementing.

## Open work items

Items intentionally deferred from the current schema bundle. Each is a future-conversation decision; do not resolve speculatively.

- **External-auth credential schema** — current `secretDescriptor` covers paste-string secrets. OAuth / device-flow / coding-agent sign-in credentials need their own shape but no concrete integration is wired today. Schema lands when the first integration ships. See [`integrations/external-auth.md`](integrations/external-auth.md).
- **Profiles** — bundled selections of scenarios + resources + secrets (e.g. "engineering", "marketing", "homelab"). `operator-state.json` reserves `active_profile` for this; `profile.schema.json` lands when the second concrete profile exists. See [`profiles.md`](profiles.md).
- **Schema-types unification (separate plan)** — `healthCheck` is currently defined four times across `tool.schema.json`, `safeguard.schema.json`, `deployment.schema.json`, and `resource.schema.json` `health_checks`. `dependencies` shapes overlap between `service.dependencies.scenarios` and `service.deployment.dependencies`. Consolidating these into shared `common.schema.json` defs is the follow-up plan after the current bundle lands.
- **Renaming the `service.deployment` block** — the name overlaps with `deployment.schema.json` (runtime config). One block is hand-edited config; the other is analyzer output. Rename to `analysis` or `feasibility` is on the table for the unification plan.
