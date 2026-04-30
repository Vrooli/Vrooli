# Connections

A **connection** is one authenticated instance of a connector — one operator's actual signed-in session for one integration. Where a connector defines *how* to talk to GitHub, a connection is *this operator's specific GitHub token, with these scopes, refreshed last Tuesday*.

This page describes the connection model. Connections are owned by the deferred `integration-hub` scenario; this page settles the shape so the scenario knows what to build.

## What a connection is

```
connection_id:  github-default
connector:      github-oauth
bound_to:       null                    # not yet attached to a scenario
scopes_granted: ["repo:read", "issues:write"]
created_at:     2026-04-29T16:30:00Z
last_probed:    2026-04-29T16:31:14Z   # last successful health probe
status:         healthy                 # healthy | needs_refresh | revoked | unknown
```

The token itself lives in Vault under a structured prefix; the metadata above lives in integration-hub's state. The split is the same as resources: secret values in Vault, descriptors in the manifest.

## Vault layout

```
secret/vrooli/integrations/<connector_id>/<connection_id>
```

Examples:

```
secret/vrooli/integrations/github-oauth/github-default
secret/vrooli/integrations/tiktok-account/persona-amy
secret/vrooli/integrations/tiktok-account/persona-bob
secret/vrooli/integrations/fal-api/scratch-2026-04-29
```

Each path holds the auth-pattern-specific keys. For OAuth: `access_token`, `refresh_token`, `expires_at`, `scopes`. For `api_key`: a single `value`. For `external_sign_in_command`: usually nothing in Vault — the agent stores its own token, integration-hub stores only metadata.

This sits parallel to:

- `secret/vrooli/<resource_name>` — paste-string secrets attached to a resource (existing).
- `secret/vrooli/scratch/<name>` — *not used*; scratch credentials live as unbound connections instead. See [unbound connections](#unbound-connections-the-scratch-case) below.

## Bound vs unbound

A connection is **bound** when it satisfies a specific scenario's integration requirement; **unbound** otherwise.

### Bound connections

Bound connections power the live system. Each binding is operator-decided and recorded in `operator-state.json`:

```jsonc
"integrations": {
  "swarm-manager": {
    "github-oauth": "github-default"
  },
  "marketing-crew": {
    "tiktok-account": [
      { "context": "persona-amy", "connection_id": "tiktok-amy" },
      { "context": "persona-bob", "connection_id": "tiktok-bob" }
    ]
  }
}
```

Single-instance scenarios bind one connection per connector; `multi: true` scenarios bind many, each tagged with a `context` the scenario understands (a persona slug, a customer id, etc.).

The scenario manifest declares the *requirement* (`integrations[].connector` + `scopes`); operator-state declares the *binding* (which concrete `connection_id` satisfies the requirement). This is the same separation as scenarios-and-their-resources: manifest declares need, operator-state declares choice.

### Unbound connections (the scratch case)

When the operator wants to test a third-party API before any scenario uses it, they create a connection with `bound_to: null`. The connection lives in the same store, gets the same management UI, the same probe-and-rotate behavior — it's just not yet referenced by any scenario.

Worked example: testing fal.ai for AI-UGC video generation.

1. Operator creates a `fal-api` connection in integration-hub: `connection_id: fal-scratch-2026-04-29`, `bound_to: null`.
2. Operator pastes the API key; it lives at `secret/vrooli/integrations/fal-api/fal-scratch-2026-04-29`.
3. Operator runs ad-hoc test scripts that read this connection.
4. Once a video-studio scenario ships and declares `integrations: [{ connector: "fal-api", scopes: [...], required: true }]`, the operator binds the existing connection: `integration-hub bind fal-scratch-2026-04-29 --to video-studio`.
5. The connection metadata gains `bound_to: ["video-studio"]`. Vault path doesn't move; no key rotation needed.

This is the entire scratch-keys story. There is no separate scratch namespace, no "loose secrets" file. Unbound connections are the canonical home.

## How scenarios bind to connections

The flow when a new scenario is enabled:

1. Operator selects scenario in onboarding (or runs `vrooli scenario enable`).
2. Wizard reads the scenario's `service.json#/integrations[]` declarations.
3. For each declared connector, integration-hub lists existing connections of that type and their `bound_to` state.
4. Operator picks an existing connection to bind, or creates a new one (which kicks off the connector's auth flow).
5. The choice is written to `operator-state.json#/integrations.<scenario>.<connector>`.
6. Scenario reads its bindings at runtime via integration-hub's CLI: `integration-hub get --scenario swarm-manager --connector github-oauth`.

Scenarios never touch Vault directly. The integration-hub CLI is the only seam.

## Lifecycle operations

Once `integration-hub` ships, these are the operations on a connection. The CLI surface is sketched here as the contract; the actual flag shapes get refined when the scenario is built.

```bash
# Create — runs the connector's auth flow and stores the result.
integration-hub create --connector github-oauth --id github-default

# List — what connections exist; `--connector` filters; `--unbound` shows scratch only.
integration-hub list
integration-hub list --connector tiktok-account
integration-hub list --unbound

# Probe — runs the connector's health check; updates last_probed and status.
integration-hub probe github-default

# Refresh — runs the connector's refresh flow (OAuth refresh-token exchange, etc.).
integration-hub refresh github-default

# Bind — attach an existing connection to a scenario. Validates scopes match.
integration-hub bind fal-scratch-2026-04-29 --to video-studio
integration-hub bind tiktok-amy --to marketing-crew --context persona-amy

# Unbind — detach without deleting the connection.
integration-hub unbind fal-scratch-2026-04-29 --from video-studio

# Revoke — call provider's revoke endpoint and delete from Vault.
integration-hub revoke github-default
```

All of these run through the connector's Go handler, so per-provider quirks are handled in one place.

## What the wizard surfaces

When the operator is in the integrations step of onboarding, integration-hub provides:

- For each `integrations[]` requirement on a selected scenario, a card showing connector + required scopes + purpose.
- Below each card, the list of existing connections of that connector type that satisfy the scopes (and aren't already bound for single-instance scenarios).
- A "Create new" action that runs the connector's auth flow (or paste-key form, for `api_key` connectors).
- For `multi: true` requirements, a binding table with one row per `context` the scenario knows about (e.g. one row per persona slug).

The wizard never invents connections silently — the operator picks or creates each one explicitly. This is the same discipline as the secrets step: nothing is assumed.

## Probe and rotation

Integration-hub probes connections on a schedule (per-connector cadence; OAuth tokens with short lives get probed more often). Probe results update the connection's `status`. The wizard's validation step in onboarding shows current status; if a connection is `needs_refresh` or `revoked`, the wizard surfaces an actionable error rather than masking it.

Rotation is operator-initiated for `api_key` and `app_password` patterns (the operator generates a new key, pastes it, integration-hub updates Vault and probes). For OAuth patterns, refresh is automatic via the connector's handler; revoke + reauth is operator-initiated.

## Why this layer exists separately

A scenario could in principle store its own tokens. But:

- **Multi-instance scenarios** (the persona-actor case) need a binding layer between requirement and instance.
- **Sharing across scenarios** (the same GitHub connection used by swarm-manager *and* a future scenario) requires a third party.
- **Probe and rotation** is a per-connector concern; centralizing avoids per-scenario reinvention.
- **The unbound/scratch case** needs a home that's not tied to any scenario.

These are the same reasons every integration-platform product converges on a hub-and-spoke shape. We're not inventing; we're matching the well-trodden pattern.

## Why this is deferred

The `integration-hub` scenario doesn't exist yet. Building it is a non-trivial chunk of work (probably 2–4 weeks of scenario development when scoped) and is on the [open work items](../architecture.md#open-work-items) list for that reason. Until then:

- API-key credentials for resources continue to use the existing `resource.credentials.env` + Vault path pattern. See [`../secrets.md`](../secrets.md).
- Loose / scratch keys for ad-hoc testing have no clean home today; either store them in Vault under an ad-hoc path and document the path manually, or delay the testing until integration-hub ships. The former is the pragmatic choice; the latter avoids accreting cleanup debt.

When integration-hub ships, scratch keys created during the gap migrate by being recreated as unbound connections. There is no automatic migration path; operator runs `integration-hub create` once per scratch key.

## See also

- [`connectors.md`](connectors.md) — connector definitions, the type-side of the model
- [`../secrets.md`](../secrets.md) — paste-string secrets attached to resources (the existing pattern)
- [`../architecture.md`](../architecture.md) — source-of-truth tables; `integration-hub` is on the deferred-scenarios list
- [`../scenarios.md`](../scenarios.md) — the `integrations` field a scenario manifest declares
