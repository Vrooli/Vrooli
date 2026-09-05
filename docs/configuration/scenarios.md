# Configuring Scenarios

Scenarios are the user-facing capability layer. An operator selects which scenarios they want, and the rest of the system (resources, host tools, secrets) follows from that selection.

## What lives where

| Concern | File | Field |
|---|---|---|
| Whether the operator wants this scenario | `.vrooli/operator-state.json` | `scenarios.<name>.enabled` |
| Whether the scenario is part of Vrooli's own operation | `scenarios/<name>/.vrooli/service.json` | `service.system_required` |
| Recommended runtime shape (long-running, on-demand, one-shot) | `scenarios/<name>/.vrooli/service.json` | `runtime.kind` |
| Recommended default for auto-restart | `scenarios/<name>/.vrooli/service.json` | `runtime.auto_restart_default` |
| Operator override for auto-restart | `.vrooli/operator-state.json` | `scenarios.<name>.auto_restart` |
| Other scenarios this one depends on | `scenarios/<name>/.vrooli/service.json` | `dependencies.scenarios.<name>` |
| Integrations this scenario needs (deferred) | `scenarios/<name>/.vrooli/service.json` | `integrations[]` |
| Which connection satisfies each integration on this install (deferred) | `.vrooli/operator-state.json` | `integrations.<scenario>.<connector>` |

For the resolution order between manifest defaults and operator overrides, see [`architecture.md`](architecture.md#resolution-order).

## Runtime policy is separate from lifecycle configuration

`service.json` is the declarative lifecycle and dependency contract. It does
not hold mutable application policy. `.vrooli/operator-state.json` records
which scenarios the operator selected and how the installation should run
them. A running scenario owns its live runtime settings in its API-backed
configuration store (and exposes the corresponding schema, CLI, and UI
surfaces). For example, notification-hub persists event patterns, templates,
and severity-to-sensitivity mapping in its own SQLite store. Updating those
settings changes behavior without rewriting `service.json`, changing operator
selection, or restarting the scenario.

## System-required vs user-application

Two classes of scenario, distinguished by `service.system_required`:

- **System-required (`service.system_required: true`)** — part of Vrooli's own operation. Examples: `vrooli-onboarding`, `secrets-manager`, `web-console`. Onboarding renders these as enabled and not user-toggleable. They are always selected, regardless of `operator-state.json`.
- **User-application (`service.system_required: false`, the default)** — opt-in capabilities the operator chooses to install. Examples: `swarm-manager`, `landing-page-business-suite`, `browser-automation-studio`. Onboarding renders these with a checkbox; selection cascades resource and scenario dependencies.

A scenario that flips from user-application to system-required must update its `service.json` and submit a PR; this is not an onboarding-time choice.

## Runtime kind

`runtime.kind` describes how the scenario is *meant* to run, not how the operator chooses to run it on a particular install. Values:

- **`long_running`** — process is intended to stay up. API servers, watchers, dashboards. Onboarding's "keep running" toggle defaults to `runtime.auto_restart_default` (typically `true` for system-required, `false` for user-application).
- **`on_demand`** — started when invoked, stopped when idle. CLI tools, request-handlers that don't run as standalone services. Default for new scenarios that don't explicitly opt in.
- **`one_shot`** — runs to completion and exits. Migrations, audits, batch jobs. The auto-restart toggle is meaningless here and onboarding hides it.

Pick the value that matches the scenario's actual shape, not what you wish it were. A scenario that's intended to stay up but currently exits early is still `long_running`; the bug is in the scenario, not the manifest.

## Scenario → scenario dependencies

Already supported by the existing `service.dependencies.scenarios` block (no schema change needed for this bundle). Each entry uses the `scenarioDependency` shape:

```json
"dependencies": {
  "scenarios": {
    "agent-manager": {
      "required": true,
      "startup_policy": "must_start",
      "description": "swarm-manager dispatches work through agent-manager's sandboxed runs"
    },
    "workspace-sandbox": {
      "required": false,
      "startup_policy": "try_start",
      "degraded_behavior": "Without sandboxing, swarm-manager refuses to dispatch destructive operations"
    }
  }
}
```

Onboarding consumes this to display a cascade hint when a scenario is selected:
> Selecting `swarm-manager` will also enable `agent-manager` (required) and try to start `workspace-sandbox` (degrades without it).

When you add a new scenario dependency to a manifest, no onboarding change is needed — the cascade is data-driven.

## Integrations a scenario needs (binding contract in progress)

When a scenario depends on an external service (GitHub, Slack, a TikTok account, a paid API), it declares the requirement in `service.json` under an `integrations` array. The Integration Hub owns connection metadata today; onboarding and manifest validation will consume this binding shape as additional connectors land.

```jsonc
"integrations": [
  {
    "connector": "github-oauth",
    "scopes": ["repo:read"],
    "purpose": "fetch issues for swarm-manager initiative tracking",
    "required": true
  },
  {
    "connector": "tiktok-account",
    "scopes": ["post:write"],
    "purpose": "publish AI-UGC videos for marketing-crew personas",
    "required": false,
    "multi": true
  }
]
```

Field intent:

- **`connector`** — id of a connector defined in `scenarios/integration-hub/connectors/<id>/connector.json`. The scenario manifest declares the *type*; the operator picks the actual connection in onboarding.
- **`scopes`** — minimum scopes the connection must have granted. Validated by integration-hub when the operator binds a connection.
- **`required`** — `true` if the scenario refuses to start without a satisfying connection; `false` if the integration gates a feature rather than the whole scenario.
- **`multi`** — `true` for scenarios that use many connections of the same type, each tagged with a `context` the scenario understands (e.g. one TikTok account per persona).
- **`purpose`** — operator-facing string surfaced in the wizard so the operator understands *why* the integration is needed.

The binding (which concrete `connection_id` satisfies the requirement) is operator state, not manifest data. See [`integrations/connections.md`](integrations/connections.md#how-scenarios-bind-to-connections) for the binding mechanics.

**Until integration-hub ships**, scenarios that need external services either:

1. Use a resource (for paste-string API keys that fit the existing `resource.credentials` shape), or
2. Block on integration-hub and document the dependency in their backlog.

Don't pre-populate `integrations[]` on existing scenarios; the field is a no-op until the consumer scenario exists.

## When to populate the new fields

These fields are optional. Existing scenarios are valid without them. Population guidance:

- **System-required scenarios** (`vrooli-onboarding`, `secrets-manager`, `web-console`) — set `service.system_required: true`, `runtime.kind: "long_running"`, `runtime.auto_restart_default: true`. Do this in a small dedicated PR.
- **Long-running user applications** — set `runtime.kind: "long_running"`, `runtime.auto_restart_default: false` (operators choose when they want it kept running).
- **One-shot tools** (migrations, audits) — set `runtime.kind: "one_shot"`, leave `auto_restart_default` unset or `false`.
- **Everything else** — leave the runtime block off; the schema default `on_demand` applies.

Do **not** mass-update via script. Per repo policy, walk scenarios individually.

## Adding a new scenario

When standing up a new scenario, decide upfront:

1. Is it system-required? Almost always no. Only flip to true if the scenario is part of Vrooli's own operating loop.
2. What's its runtime kind? Pick the most accurate of `long_running` / `on_demand` / `one_shot`.
3. What scenarios does it depend on? Declare them in `dependencies.scenarios` with `startup_policy` reflecting orchestration intent.

The wizard surfaces all of this automatically once the manifest is correct. There is no separate onboarding registration step.
## Schema-backed scenario settings

Scenario tunables belong in `.vrooli/config.json` under `settings`, with their
types and required defaults declared by the scenario's configuration schema.
The shared `scenarioconfig-go` loader resolves values in this order:

1. The scenario's `.vrooli/config.json` value.
2. The setting's default in the scenario schema.

An absent config file therefore has deterministic behavior, while an unknown
setting or wrong type fails with the setting name. Resource credentials,
dynamic ports, and peer addresses remain lifecycle/discovery concerns and do
not belong in scenario settings.

`browser-automation-studio` is the reference conversion at
`.vrooli/config.json`; its sidecar timing settings use this contract.
