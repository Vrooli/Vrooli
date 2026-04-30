# Wizard Flow (V2 Rework)

This doc captures the planned wizard flow for the v2 rework, the wireframe sketches, and the design rationale. It is the implementation reference when the rework is picked up. The configuration substrate the wizard reads from and writes to is documented in [`/docs/configuration/`](../../../docs/configuration/) — read that first.

## Why the v2 rework

The first wizard grouped operator choices around resources. The mental-model mismatch is that operators don't think in resources — they think in capabilities. The v2 inverts: operators select **scenarios** first (the things they want), and resources, secrets, host tools/safeguards, and integrations are derived from that selection.

This is also the only way the architecture in [`/docs/configuration/architecture.md`](../../../docs/configuration/architecture.md) reads cleanly: scenarios declare what they need; the wizard surfaces that derivation; the operator sees their use-case as the unit of selection rather than infrastructure components.

## Step sequence

Six real steps; one optional pre-step. Re-enterable from any step.

1. **(Optional) Goal intake** — "What are you here to do?" Pre-selects a profile. Skippable. Profiles themselves are deferred (see [`profiles.md`](../../../docs/configuration/profiles.md)) so this step is also deferred until the second profile is real. Until then, the wizard starts at step 2.
2. **Scenarios** — search/filter list; system-required scenarios locked-on; per-scenario "keep running" toggle with default from `runtime.auto_restart_default`; selecting a scenario cascades scenario→scenario and scenario→resource dependencies.
3. **Resources** — auto-derived required + optional from scenario selection; user toggles optionals; no manual-only path to enable a resource without a scenario that uses it (besides standalone selection).
4. **Secrets** — only the credentials actually needed by the selected stack. Renders `secretDescriptor` fields (label, description, obtain_url) for rich resources; bare label for legacy ones.
5. **Integrations** — for each `integrations[]` requirement on a selected scenario, render a connector card with required scopes + purpose, and let the operator pick an existing connection or create a new one (which kicks off the connector's auth flow per [`/docs/configuration/integrations/external-auth.md`](../../../docs/configuration/integrations/external-auth.md)). Multi-instance scenarios (the persona-actor case) get a binding table with one row per `context`. The connector + connection model itself is owned by the deferred `integration-hub` scenario; this step is empty in the v2 baseline and lights up when integration-hub ships.
6. **Host (tools + safeguards)** — declared by selected scenarios and resources. Required tools install automatically; non-required tools and safeguards are opt-in. Safeguards display `risk` (low/medium/high).
7. **Operating mode + final validation** — confirm auto-restart per scenario; commit to operator-state.json; run full probe pass; show green-light or actionable error list.

## Wireframes

These were sketched during the design conversation. Use them as guides, not pixel specs.

### Step 2 — Scenarios

```
┌─ Scenarios ────────────────────────────────────────────┐
│  [search: ___]  [filter: all | core | enabled]         │
│                                                         │
│  ☑ vrooli-onboarding         CORE   (locked)           │
│  ☑ secrets-manager           CORE   (locked)           │
│  ☑ web-console               CORE   (locked)           │
│  ─────────────────────────────────────────────────     │
│  ☑ swarm-manager                    [keep running ☑]   │
│      ↳ pulls in: agent-manager, workspace-sandbox      │
│  ☑ agent-manager                    [keep running ☑]   │
│  ☑ workspace-sandbox                [keep running ☐]   │
│  ☐ landing-page-business-suite      [keep running –]   │
│  ☐ browser-automation-studio                            │
│                                                         │
│  Required resources (derived): postgres, redis, vault  │
│  Optional resources (next step): ollama, qdrant, ...    │
└────────────────────────────────────────────────────────┘
```

Notes:
- "CORE (locked)" entries are scenarios with `service.system_required: true`. They cannot be unchecked.
- The cascade hint (`↳ pulls in: agent-manager, workspace-sandbox`) reads from `dependencies.scenarios` on the selected scenario's manifest. Required deps say "pulls in"; `try_start` deps say "tries: ..." with degraded-behavior tooltip.
- "Keep running" toggle defaults to `runtime.auto_restart_default`. Hidden for `runtime.kind: "on_demand"` and `"one_shot"`.
- The footer rolls up the implied resource selection so the operator sees the consequence before continuing.

### Step 3 — Resources

Same shape as Scenarios, but:
- Required resources (derived from scenarios) are locked-and-checked.
- Optional resources (declared via `optional_dependencies`) are toggleable.
- Standalone resources (not required by any selected scenario) are toggleable separately.

### Step 4 — Secrets

Per-resource list of credentials the selected stack needs. Each entry renders the `secretDescriptor`:

```
┌─ Gemini API Key  (resource: gemini, required)  ────────┐
│  Google Gemini multimodal LLM. Required for scenarios  │
│  that call Gemini directly.                            │
│  Hint: Starts with 'AIza...'                           │
│  [Get one →]                                           │
│  [_______________________________________________]      │
│  [ Save  ]                                              │
└────────────────────────────────────────────────────────┘
```

`[Get one →]` is the `obtain_url`. Save writes to Vault under `secret_ref`. Validation against `validation_pattern` is client-side and lightweight.

Bare-string `credentials.env` entries render with just the env-var name as label and no other metadata. Operators are encouraged (via doc) to enrich credentials with `secretDescriptor` over time.

### Step 5 — Integrations (deferred until integration-hub ships)

The integrations step consumes connector and connection state owned by the `integration-hub` scenario (see [`/docs/configuration/integrations/connectors.md`](../../../docs/configuration/integrations/connectors.md) and [`connections.md`](../../../docs/configuration/integrations/connections.md)). Each connector requirement on a selected scenario gets a card; the operator picks an existing connection or creates a new one.

Single-instance requirement (e.g. one GitHub account):

```
┌─ GitHub  (required by swarm-manager) ───────────────────┐
│  scopes: repo:read                                      │
│  purpose: fetch issues for initiative tracking          │
│                                                         │
│  Existing connections:                                  │
│    ⦿ github-default       healthy  (last probed 2m ago) │
│    ○ + Sign in with GitHub                              │
│                                                         │
│  [Bind selected]   [Probe again]                        │
└─────────────────────────────────────────────────────────┘
```

Multi-instance requirement (the persona-actor case):

```
┌─ TikTok account  (required by marketing-crew, multi) ──┐
│  scopes: post:write                                    │
│  purpose: publish AI-UGC videos per persona            │
│                                                        │
│  context           connection                          │
│  persona-amy       [tiktok-amy ▼]      healthy         │
│  persona-bob       [tiktok-bob ▼]      needs_refresh   │
│  persona-carol     [+ Sign in… ▼]      —               │
│                                                        │
│  [Bind all]                                            │
└────────────────────────────────────────────────────────┘
```

Loose / scratch credentials (the operator wants to test fal.ai before any scenario uses it) appear in integration-hub's standalone view, not this wizard step. They show up here only once a scenario declares them.

Auth-flow specifics per pattern (`api_key`, `oauth_web`, `oauth_device`, `external_sign_in_command`, `app_password`) live in [`/docs/configuration/integrations/external-auth.md`](../../../docs/configuration/integrations/external-auth.md). The wizard dispatches by `connector.auth.kind` and renders the right surface (paste-form vs sign-in button vs device-code panel vs sign-in-command instructions).

This step is empty in the v2 baseline and lights up when `integration-hub` ships. Until then, scenarios that need external services either use a resource (for paste-string API keys) or block on integration-hub.

### Step 6 — Host (tools + safeguards)

```
┌─ Host Tools ──────────────────────────────────────────┐
│  ☑ git              required by 8 scenarios       LOW │
│  ☑ docker           required by all                LOW │
│  ☐ cloudflared      required by deployment-...     LOW │
├─ Host Safeguards ─────────────────────────────────────┤
│  ☑ clock            verifies system clock          LOW │
│  ☐ kernel_config    enables high-perf networking   MED │
│      writes /etc/sysctl.d/99-vrooli.conf               │
│  ☐ nat-protection   prevents loopback bypass       MED │
│  ☐ docker-host-firewall                            MED │
└────────────────────────────────────────────────────────┘
```

Required tools (per `hostRequirement.required: true`) are locked-and-checked; non-required and all safeguards are opt-in. The `risk` column reads from `safeguard.json#/risk` (`low` / `medium` / `high`). The expanded text under medium/high entries is from the manifest's `description` and `notes`.

### Step 7 — Operating mode + final validation

```
┌─ Operating Mode ──────────────────────────────────────┐
│  Auto-heal:        ⦿ on   ○ off   ○ per-scenario only │
│  On host startup:  ⦿ start enabled scenarios          │
│                    ○ start nothing (manual)            │
│  Default profile:  [engineering ▼]   [Save current…]  │
│                                                        │
│  [Validate and commit]                                 │
└────────────────────────────────────────────────────────┘
```

Then the final validation report:

```
┌─ Validation ──────────────────────────────────────────┐
│  ✓  Postgres reachable                                │
│  ✓  Vault reachable                                   │
│  ✓  GEMINI_API_KEY validated                          │
│  ✗  swarm-manager auto-restart: agent-manager not yet │
│      started — waiting (10s)...                       │
│  ✓  Cloudflared tunnel attached                       │
│                                                        │
│  Status: 4/5 green; 1 transient                       │
│  [Recheck] [Continue with degraded]                    │
└────────────────────────────────────────────────────────┘
```

## What's locked in

These are settled by the design conversation; do not relitigate without strong evidence:

- **Source of truth = manifests + operator-state**, not onboarding internals or doc lists.
- **Scenarios → Resources → Secrets → Integrations → Host → Operating-mode → Validation** order.
- **System-required scenarios are locked-on**, declared per `service.system_required`.
- **Per-scenario auto-restart** is the operator's call; manifest provides the *recommendation* via `runtime.auto_restart_default`.
- **Re-enterable**: not a one-shot. Adding a scenario later re-enters the wizard at the relevant step with prior state pre-loaded.
- **Risk indicator on safeguards** is a column, not a separate step.

## What's deferred

Captured here so a future implementation conversation has the full picture without re-deriving:

- **Goal-intake step** — depends on profiles (deferred).
- **Profiles** — deferred until second concrete profile exists. Reserved field `active_profile` already in `operator-state.json`.
- **Integration-hub scenario** — owns connector definitions and connection instances; powers wizard step 5; covers both bound (scenario-attached) and unbound (scratch / testing) credentials. Probably 2–4 weeks of scenario work when scoped. Wizard step 5 is empty until then. See [`/docs/configuration/integrations/connectors.md`](../../../docs/configuration/integrations/connectors.md).
- **Schema-types unification** (separate plan) — `healthCheck` defined four times across schemas; consolidating into `common.schema.json`. Not blocking the v2 rework but should land before too many new schemas accrete.

## Implementation pointers

When the rework is picked up:

1. Read [`/docs/configuration/architecture.md`](../../../docs/configuration/architecture.md) — the resolution-order rules are the wizard's evaluator contract.
2. Read [`operator-state.schema.json`](../../../.vrooli/schemas/operator-state.schema.json) — that's the only file the wizard writes to.
3. The existing wizard's `StepSelectResources` is the closest analog for the new Scenarios step shape (search/filter/select pattern). Reuse the component skeleton.
4. The host step is new; no existing equivalent. Use `safeguard.json#/risk` for the risk column.
5. Validation step is partly built (resources health probe exists); extend to cover scenarios and secrets.

## See also

- [`/docs/configuration/`](../../../docs/configuration/) — the full configuration substrate this wizard implements
- [`/docs/configuration/architecture.md`](../../../docs/configuration/architecture.md) — source-of-truth tables and resolution order
- [`/docs/configuration/scenarios.md`](../../../docs/configuration/scenarios.md) — `system_required`, `runtime.kind`, scenario deps, `integrations[]` (deferred)
- [`/docs/configuration/host/safeguards.md`](../../../docs/configuration/host/safeguards.md) — `risk` field meaning
- [`/docs/configuration/integrations/connectors.md`](../../../docs/configuration/integrations/connectors.md) — connector model powering step 5
- [`/docs/configuration/integrations/connections.md`](../../../docs/configuration/integrations/connections.md) — connection instances and binding
- [`/.vrooli/schemas/operator-state.schema.json`](../../../.vrooli/schemas/operator-state.schema.json) — wizard's write target
