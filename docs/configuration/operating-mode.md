# Operating Mode

This page covers the operator's runtime choices: which scenarios stay running, what auto-restarts on failure, and how the install behaves at host startup.

## What lives where

| Concern | File | Field |
|---|---|---|
| Scenario's recommended runtime shape | `scenarios/<name>/.vrooli/service.json` | `runtime.kind` |
| Scenario's recommended auto-restart default | `scenarios/<name>/.vrooli/service.json` | `runtime.auto_restart_default` |
| Operator's auto-restart override | `.vrooli/operator-state.json` | `scenarios.<name>.auto_restart` |
| Operator's enabled selection | `.vrooli/operator-state.json` | `scenarios.<name>.enabled`, `resources.<name>.enabled` |
| Named configuration preset | control-plane configuration document | expands to desired scenario, resource, host, credential, and safeguard selections |
| Global startup behavior | (deferred — see "open work" below) | — |

## The runtime block

Per-scenario, declared in the scenario's own `service.json`:

```json
"runtime": {
  "kind": "long_running",
  "auto_restart_default": true
}
```

`runtime.kind` says how the scenario is *meant* to run. Three values, picked by the scenario author based on the scenario's actual shape:

- **`long_running`** — process is intended to stay up. The scenario is something the operator probably wants restarted on failure if they enabled it at all.
- **`on_demand`** — started when invoked (CLI, request handler, ad-hoc tool). Auto-restart usually doesn't apply; the wizard hides the toggle.
- **`one_shot`** — runs to completion and exits (migrations, audits). Auto-restart is meaningless; the wizard hides the toggle.

`runtime.auto_restart_default` is the *recommended* default for the auto-restart toggle. Onboarding pre-fills the toggle from this; the operator can override.

## Auto-restart resolution

For any specific scenario, the effective auto-restart setting is computed from this order:

```
1. .vrooli/operator-state.json → scenarios.<name>.auto_restart   (if present, use it)
2. scenarios/<name>/.vrooli/service.json → runtime.auto_restart_default   (if present, use it)
3. schema default (false)
```

Onboarding writes only to layer 1. Manifests own layers 2 and 3.

A scenario absent from `operator-state.json` falls through to the manifest default. This is how new scenarios installed after onboarding has run still pick up sensible behavior without re-running the wizard.

## What "auto-restart" actually means

The lifecycle orchestrator (in `path:internal/runtime/`) uses the resolved value to decide whether to relaunch a scenario after it exits. It does not turn on monitoring, alerting, or escalation — those are separate concerns handled by the auto-heal scenario and infra-health team. Auto-restart is the local "did the process die? bring it back" behavior.

The scope is intentionally narrow: a scenario can be "enabled, not auto-restarting" (operator wants it but is fine starting it manually) or "enabled, auto-restarting" (operator wants it always up). Anything more sophisticated — exponential backoff, max retry counts, alert escalation — belongs to whatever scenario provides that capability, not the runtime block.

## Profile-driven defaults

When profiles land (currently deferred — see [`profiles.md`](profiles.md)), a profile may declare a different `auto_restart_default` for specific scenarios than the manifest does. Until that lands, profiles influence operator-state directly via the wizard's profile-pre-selection step.

## Open work items

These are intentionally not in the current schema bundle:

- **Global startup behavior** — "start enabled scenarios when the host boots" vs "start nothing, operator does it manually." Currently implicit in the lifecycle orchestrator; no operator-state field. Lands when the operator surfaces a clear use case for the manual-only mode.
- **Per-resource auto-restart** — resources have `health_checks` but no operator-overridable auto-restart toggle. The current assumption is that resources always auto-restart when the host runs; if that assumption ever needs an escape hatch, it will mirror the scenario shape.
- **Restart policies** (max retries, backoff) — out of scope. If needed, becomes a feature of the auto-heal scenario rather than the runtime block.

## See also

- [`scenarios.md`](scenarios.md) — `runtime.kind` per scenario
- [`architecture.md#resolution-order`](architecture.md#resolution-order) — full resolution rules
- [`profiles.md`](profiles.md) — how profiles will eventually influence runtime defaults (deferred)
