# Configuring Scenarios

Scenarios are the user-facing capability layer. An operator selects which scenarios they want, and the rest of the system (resources, host tools, secrets) follows from that selection.

## What lives where

| Concern | File | Field |
|---|---|---|
| Whether the operator wants this scenario | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `scenarios.<name>.enabled` |
| Whether the scenario is part of Vrooli's own operation | `scenarios/<name>/.vrooli/service.json` | `service.system_required` |
| Recommended runtime shape (long-running, on-demand, one-shot) | `scenarios/<name>/.vrooli/service.json` | `runtime.kind` |
| Recommended default for auto-restart | `scenarios/<name>/.vrooli/service.json` | `runtime.auto_restart_default` |
| Operator override for auto-restart | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `scenarios.<name>.auto_restart` |
| Other scenarios this one depends on | `scenarios/<name>/.vrooli/service.json` | `dependencies.scenarios.<name>` |
| Integrations this scenario needs (deferred) | `scenarios/<name>/.vrooli/service.json` | `integrations[]` |
| Which connection satisfies each integration on this install (deferred) | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `integrations.<scenario>.<connector>` |

For the resolution order between manifest defaults and operator overrides, see [`architecture.md`](architecture.md#resolution-order).

## Runtime policy is separate from lifecycle configuration

`service.json` is the declarative lifecycle and dependency contract. It does
not hold mutable application policy. `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` records
which scenarios the operator selected and how the installation should run
them. A running scenario owns its live runtime settings in its API-backed
configuration store (and exposes the corresponding schema, CLI, and UI
surfaces). For example, notification-hub persists event patterns, templates,
and severity-to-sensitivity mapping in its own SQLite store. Updating those
settings changes behavior without rewriting `service.json`, changing operator
selection, or restarting the scenario.

## Restart and maintenance of an execution owner

### Live API replacement fence

When a restart replaces an authoritative live scenario that declares `API_PORT`,
the control plane asks its API owner for lifecycle `Status`, closes request
admission with `Prepare`, and waits for admitted work to drain before stopping
the pinned process. Standard `api-core/server.Run` now installs the shared
provider, loopback-only lifecycle endpoint, and ingress accounting by default.
An API with work that outlives its HTTP request must register owner snapshot and
quiesce/resume hooks. Services with richer domain inventories can install their
own provider and admission gate through the same lifecycle protocol.

This is a replacement safety check, not an authorization check. If `Status` is
unsupported (`NotFound` or `Unimplemented`), the control plane logs a warning
and continues without a provider fence for compatibility with older owners.
Other status errors, malformed standings, or an incomplete drain proof still
fail closed during a normal restart. An operator can explicitly accept that
risk for a restart with
`vrooli scenario restart <name> --force-lifecycle
--lifecycle-override-reason '<reason>'`. The control plane records the reason
and warns that in-flight or detached owner work may be interrupted. This option
does not permit replacing a process whose runtime identity is non-authoritative.
Use it for a deliberate recovery when the work-loss risk is understood.

An ordinary operator-authorized `vrooli scenario restart agent-manager` uses a
separate recovery-first path. The control plane stages a stale replacement before
stopping the serving process, preserves detached Agent Manager API executors,
and lets the new owner recover durable runs and Web Console sessions. This path
does not require Agent Manager to close admission or drain itself. After restart,
check the original run identities and recovery progress; process health alone is
not proof that each run reconnected. See
[runner transcript recovery](../../scenarios/agent-manager/docs/runner-transcript-recovery.md).

The stricter maintenance proof below applies to operations such as standalone
setup and dependency refresh, not to that ordinary restart path. Normal stop and
explicit run cancellation also have different effects.

Agent Manager owns admission and the inventory of accepted execution work. The
control plane owns Stop, setup, replacement and the per-instance lifecycle lock.
An empty run listing is not maintenance authorization: workflow lifetimes,
admission in progress, finalization and physical executors can still be active.

The bounded AM integration uses its existing read-only
`GET /api/v1/maintenance/admission` contract. A replacement requires a closed
positive revision, `admitting: 0`, `remaining: 0`, `drained: true`, and an
`inventory` with explicit zero `remaining` and no work, executors or unknowns
for the ordinary drained path. The accounting-only path below retains cancelled
workflow records after separately proving their original children are terminal.
The owner must also advertise `lifecycleInterlock: "scenario-lock-v1"` only after
its explicit resume operation takes the same nonblocking lock as the lifecycle:
`<home>/.vrooli/state/locks/scenario-agent-manager.lock` (variant suffixes follow
the existing instance-key convention). Acquire that lock before the admission
mutex. Hold it until resume persistence completes. A busy lock refuses resume.
Status reads do not acquire this lock, so the lifecycle can read them while it
holds the lock. Restart must retain the closed admission revision on disk.

For operations requiring maintenance proof, the control plane reads this
precondition under the target lock before setup or teardown and rechecks it
where applicable. Missing
fields, unsupported interlocks, unknown inventory and unavailable owner reads
refuse the operation. For these maintenance-gated operations, an identified
agent executor cannot maintain its own AM owner even if an inventory incorrectly
reports zero. Dependency refresh follows
the same precondition. When freshness arbitration degrades
`restart_when_stale` to `rebuild_only` while owner admission is open, the
control plane MUST reduce the action to `reuse_running`, retain the healthy
existing process, and publish `EventDependencyStalePolicy` with reason
`execution owner admission open; serving stale build`. A busy lifecycle lock
uses the same `reuse_running` result with reason `dependency lock busy; serving
stale build`. An automatic lifecycle invocation never enters maintenance,
resumes admission or exchanges a local owner credential.

Accounting-only maintenance (operator implementation amendment, 2026-09-13)
uses the same closed revision and lifecycle interlock. A complete inventory with
no executors or unknowns may retain cancelling workflows whose original child
runs are terminal and whose finalization is inactive. The control plane MUST
verify each exact workflow through the owner's bounded trace and original run
reads. Every attempt MUST be bound to an observed terminal run; incomplete,
duplicate, nested or uncertain dispatch bindings refuse this narrow path.
The inventory counts and attempt counts MUST agree with the complete responses.
The proof is repeated under the lifecycle lock immediately before teardown.
This permits replacement while preserving original accounting records and
unknown charges. It MUST NOT report those workflows as drained, settle their
accounting, create a replacement child, or allow a live executor to be stopped.

For operations requiring maintenance proof, current-executor protection
recognizes an identity token, `VROOLI_RUN_ID`, or a nonempty `*_AGENT_TAG`;
legacy runners do not gain destructive-maintenance authority merely because
they predate identity-token wiring. This check does not prohibit the ordinary
recovery-first restart described above.

For an operation requiring this proof, an authorized operator uses AM's explicit
maintenance begin/drain operations, runs the lifecycle operation, then explicitly
resumes the retained revision.
An unavailable owner requires its startup/recovery protocol; a failed inventory
read cannot authorize destructive recovery. This contract covers standalone AM
setup and applicable dependency adoption; ordinary AM restart follows the
recovery-first path above. Other execution owners require their own qualified
precondition; this is not a new manifest hook framework.

`start` has one non-destructive stopped-owner recovery path: when the AM owner
read is unavailable and its registry instance is non-authoritative, the control
plane reads every active instance, managed PID/group, legacy process record and
claimed listener under the lifecycle lock. Only positive managed-scope absence
permits registry finalization and fresh startup. Unknown or live managed scope
refuses. This path does not call Stop, clean fixed-port orphans, or sweep host
processes on failure. Independent run-labelled executors outside the recorded
service groups remain untouched. A partial failed restoration may require
subsequent explicit maintenance; this read does not grant fresh run dispatch or
resume admission. Stop, restart and explicit setup do not use this exception.

Dependency serving health is separate from authored-source identity. A healthy
optional dependency is reused without walking source freshness. Required
dependencies apply their declared freshness policy to source drift, including
`reuse_running`; they do not reinterpret dirty source as a failed health probe.
Optional back-edges to an ancestor currently starting are reported as deferred;
required cycles remain errors. Explicit target adoption still checks identity.

### Physical executor exclusion

The root maintenance controller owns the shared, read-only physical projection
used by owner drain and fresh recovery. Invoke `vrooli runtime executor-scope
--json` with JSON on stdin (8 MiB maximum):

```json
{"executors":[{"runId":"run-id","tag":"current-tag","legacyTag":"legacy-tag","pid":123,"pgid":123,"startedAt":"2026-09-12T12:00:00Z","endedAt":"2026-09-12T13:00:00Z"}]}
```

Requests contain 1–16,384 unique runs; timestamps are optional durable execution
times. Gather the bounded owner inventory into one request instead of starting
one host scan per row or database page. Database pagination does not limit the
physical inventory to one page. Owners must match the root request and response
limits; over-capacity inventories refuse rather than silently omit history.
The response has `schemaVersion: "executor-scope-v1"`, `complete`, and an
`executors` array containing each `runId`, `state` (`absent`, `present`, or
`unknown`), bounded `pids` and `reasons`. Consumers must verify the version,
complete requested-ID coverage, `complete: true`, and explicit `absent` for
every candidate. Exit success alone or an empty/missing result is not exclusion.

The response is limited to 8 MiB and never truncates requested identities. PID
samples are capped at `min(64, floor(65536 / referenceCount))` per entry; they
are evidence samples, not executor counts. Every `present` result retains at
least one PID. Unknown reasons are bounded, and incomplete evidence remains
fail-closed. Run/tag/PID/group indexes make matching linear in the request and
host inventory, with a fixed sample budget even when many runs share a tag.
The four-second deadline includes matching and wire processing; expired or
oversized output fails without emitting usable partial JSON. A successful
small request against an older installed CLI does not qualify its bulk capacity.

The implementation reuses the canonical maintenance process table, native
supervision birth-time evidence and stable platform process handles. Matching
run identity and non-inherited current/legacy labels override terminal status. A numeric PID/group alone
is unknown; stable birth outside the durable execution interval (with a
two-second precision allowance) proves reuse. Group reuse requires leader
evidence, not merely a later-born child. Exited historical roots with no group
or labelled descendant can therefore be excluded without deleting history.

For an ended run only, inherited labels can be attributed to an independently
registered scenario service. Proof requires a read-only registry entry for the
current boot, a running lifecycle/supervisor instance and process registration,
exact PID/PGID, stable native birth within two seconds of the registered start,
the canonical service scope, and matching instance, scenario and nonempty
variant. A descendant must have stable birth and an observed parent chain to
that root within the same service scope and instance. Any explicit run identity
breaks inheritance. Only exact labels shared with the registered root qualify;
independent child labels, exact run identities and numeric PID/group ambiguity
remain executor evidence. Missing or mismatched proof retains the original
positive or unknown result. Classification precedes sampling, so an excluded
service cannot hide a real executor beyond the sample limit. This observation
does not stop a process, settle accounting, or authorize scenario replacement.
Unreadable same-user identities in potential execution scopes, unstable observations, cancellation and
unsupported platforms remain unknown. Environment contents, credentials and
command lines are not returned or logged. The read has a four-second host
observation ceiling and never signals a process or exchanges owner authority.
Kernel-attested unrelated native service scopes and the user manager's
`init.scope` are outside managed execution scope, even when their credential
environments are intentionally unreadable. This exception does not cover
Vrooli slices, terminal scopes, or missing cgroup evidence; numeric PID/group
ambiguity still requires the independent birth-time proof.

This proof concerns local-user, identity-labelled managed execution. Callers
must hold their admission/continuation fence across observation and dispatch;
it is not a transferable authorization receipt or proof against deliberately
identity-stripped executors. AM adapters must reuse this projection rather than
maintaining private `/proc` walkers or treating PID liveness as run identity.

## Platform identity provider versus scenario policy

`scenario-authenticator` is a platform identity provider, not a reason for
every scenario to force a login. It should be enabled as a system-required
dependency when the installation needs local multi-user, remote, shared
provider, or explicit commercial account linking. A scenario still declares
its own capabilities and resource/object policy; it must not invent a second
identity database or treat the authenticator's existence as permission to do
everything.

Desktop deployments default to `personal_local`: local use is private to the
operator and the supervisor's loopback token is a process credential, not a
human identity. The operator can enable another mode through the deployment or
scenario settings UI. That choice must be persisted as typed configuration,
shown in status, and fail closed if its required provider or mapping is not
available. The LPBS website session and commercial entitlement lease remain
separate from local identity and local authorization.

Use the project [Identity and Authentication contract](../concepts/IDENTITY-AND-AUTHENTICATION.md)
and [Authorization contract](../concepts/AUTHORIZATION.md) when adding a
scenario authentication profile.

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

## Branding declaration

A scenario declares the brand it ships and the icon target profiles it wants in
an optional `branding` block. brand-manager `apply` reads it and writes the
files. The block is additive; scenarios without it stay valid.

```json
"branding": {
  "brand": "aquila",
  "targets": ["web-public-v1", "electron-v1"]
}
```

Fields:

- **`brand`** — the brand-manager brand slug (`^[a-z0-9][a-z0-9-]*$`).
- **`targets`** — one or more target profile ids. Today: `web-public-v1` (the
  `/public/*` PWA set under `ui/public/public/`) and `electron-v1` (the
  `platforms/electron/assets/` desktop set, applied only when that directory
  exists).

Sizes and wiring live in brand-manager, not here; see
`scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`.

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

Detached resource services and their companions receive their resource lifecycle
identity. Their launch environment excludes the requesting agent's identity token,
run ID, and runner agent tags. This prevents a long-lived service from retaining
an obsolete agent's authority or being counted as that agent's live executor.
Existing services adopt this boundary on their next owner-managed restart.
