# Connectors

A **connector** is a declarative definition of an *integration type* — how to talk to GitHub, how to talk to Slack, how to talk to a TikTok account, how to call the fal.ai HTTP API. Connectors are reusable across scenarios and operators; they are *not* an authenticated session, just the recipe for creating one.

This page describes the connector model. The `integration-hub` runtime now ships the provider-neutral connection service and its first OpenRouter API-key pilot. The broader manifest and driver catalog remains incremental work.

## Why a separate concept

Without connectors, every scenario that wants to talk to GitHub has to redefine "GitHub": auth flow, scopes, base URL, token-refresh behavior. Multiplied across hundreds of plausible third-party integrations, this is unmanageable. Every product in the integration-platform space (Zapier, n8n, Pipedream, Nango, Composio) converges on the same factoring: **type definition (connector) is separate from instance (connection)**, and scenarios reference instances by ID.

Vrooli adopts the same pattern because:

- **Multi-instance is native to our use case.** The marketing AI-UGC strategy already calls for multiple persona-actor accounts on the same platform — many TikTok accounts, many Instagram accounts, each bound to a different persona. The connector/connection split handles this without special cases.
- **Wrap-not-use applies.** Scenarios should not roll their own GitHub HTTP client. They request a connection from the integration-hub and call through it. Same architectural principle that says "don't roll your own browser, use BAS."
- **Scenarios shouldn't carry the auth burden.** OAuth refresh, device-flow polling, token rotation — all per-provider runtime concerns. Centralizing them in the connector + integration-hub keeps scenarios simple.

## Connector vs connection (one-line definition)

| | Connector | Connection |
|---|---|---|
| What | Definition of *how to talk to provider X* | An instance of an authenticated session for one operator |
| Source-controlled | Yes (manifest in this repo) | No (lives in the credential authority + integration-hub state) |
| Multiplicity | One per integration *type* | Zero, one, or many per connector |
| Owns | Auth pattern, scopes, base URL, capabilities, runtime driver | Tokens, expiry, `bound_to` metadata, last-probed status |

See [`connections.md`](connections.md) for the connection side.

## Auth patterns the connector model supports

A connector declares which auth pattern it uses. Each pattern has a per-pattern Go driver (in the integration-hub) that handles the specifics; the connector manifest declares the *kind*, the driver runs the flow.

- **`api_key`** — operator pastes a string. Same shape as `secretDescriptor` (which `api_key`-pattern connectors compose). The simplest case; e.g. fal.ai, OpenRouter.
- **`oauth_web`** — operator clicks a button, gets redirected to the provider, authorizes, returns with a code; system exchanges code for token. Most consumer-facing OAuth providers; e.g. GitHub, Slack, Google.
- **`oauth_device`** — system displays a code; operator visits a URL on another device and enters it; system polls until authorized. Used when there's no browser; e.g. CLI installs.
- **`external_sign_in_command`** — operator runs a command in their terminal (`claude /login`, `codex login`); the agent persists tokens in its own config; integration-hub probes for sign-in state. Used for coding agents and similar tools that own their own auth state.
- **`app_password`** — operator generates a long-lived secondary password from the provider's UI (Bluesky's app password, Apple's app-specific passwords). Same paste shape as `api_key` but separate so the UI explains the obtain step correctly.

A connector picks exactly one. If a provider supports multiple (e.g. GitHub supports both OAuth and PAT), Vrooli ships them as separate connectors (`github-oauth`, `github-pat`) — clearer than a polymorphic single connector.

## Sketch of the connector manifest

This shape remains **provisional** for connector manifests. The stable provider-neutral lifecycle contract is in `common.v1.integrations`; individual manifest fields become contractual as concrete drivers are added.

```jsonc
{
  "$schema": "../../../.vrooli/schemas/connector.schema.json",
  "id": "github-oauth",
  "display_name": "GitHub",
  "description": "Read and write GitHub repositories on behalf of the operator.",
  "auth": {
    "kind": "oauth_web",
    "authorize_url": "https://github.com/login/oauth/authorize",
    "token_url":     "https://github.com/login/oauth/access_token",
    "scopes_offered": ["repo:read", "repo:write", "issues:write", "actions:read"],
    "callback_path":  "/integrations/github-oauth/callback"
  },
  "capabilities": [
    "list_repos",
    "read_issues",
    "write_issues",
    "read_actions"
  ],
  "rate_limits": {
    "requests_per_hour": 5000,
    "notes": "GitHub REST API: 5000/hour authenticated"
  },
  "obtain_url": "https://github.com/settings/applications",
  "icon_uri": "icons/github.svg",
  "handler": "github_oauth"
}
```

Field intent:

- **`id`** — globally unique among connectors. Stable; renames are breaking. Lowercase, hyphenated.
- **`auth`** — pattern + per-pattern parameters. Schema diverges by `auth.kind`.
- **`capabilities`** — declarative tags scenarios match against. Lets a scenario say "I need a connection that can `write_issues`" without naming GitHub specifically. Usually scenarios will name the connector directly, but capabilities support cross-connector substitution where it's possible.
- **`rate_limits`** — informational; integration-hub uses this to schedule and back-off. Not a security control.
- **`handler`** — name of the Go driver in `integration-hub` that implements the runtime calls and the auth flow specifics. Same shape as `tool.json#/handler` in the host-tools registry.

## Where connector manifests will live

```
scenarios/integration-hub/connectors/
  github-oauth/
    connector.json
    icons/github.svg
  slack-oauth/
    connector.json
  fal-api/
    connector.json
  tiktok-account/
    connector.json
  ...
```

Mirroring how `path:internal/tools/` and `path:internal/safeguards/` work today: one folder per connector, manifest plus assets, drift-protected by tests asserting manifest-handler invariants. The manifests are source-controlled; the connection *instances* (real tokens) are not.

## Connector vs resource — when to use which

Some integrations could plausibly be modeled either way. Guidance:

- **Resource** — when the integration provides a *runtime capability the system depends on*: a model server (Ollama), a database (Postgres), a workflow engine. Resources have lifecycle (start, stop, health), live in `resources/`, and are typically singletons per install.
- **Connector** — when the integration provides *access to an external account or service*: GitHub, Slack, a TikTok account, a paid API. Connections are not lifecycle-managed by Vrooli; they're authenticated sessions Vrooli holds on behalf of the operator.

A pay-per-use AI API (fal.ai, OpenAI) is a borderline case. Today `path:resources/openrouter/` models this as a resource because the credential is just a paste-string and resources already had the credential plumbing. As `integration-hub` matures, this may migrate to an `api_key`-pattern connector — but that's a future-conversation decision, not a current concern. Don't pre-migrate.

## Capabilities a scenario declares

When a scenario depends on an integration, it declares the requirement in its `service.json`. The proposed shape for scenario binding is:

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

- **`connector`** — the connector id this scenario expects.
- **`scopes`** — minimum scopes the connection must have granted. Onboarding/integration-hub validates the operator's connection meets these.
- **`required`** — whether the scenario refuses to start without a satisfying connection. Optional integrations gate features, not the whole scenario.
- **`multi`** — whether the scenario uses many connections of this type (the persona-actor case). When `true`, the operator binds each connection to a context (e.g. a persona) and the scenario picks at runtime.
- **`purpose`** — operator-facing string, surfaced in the wizard so they understand *why* this integration is needed.

The connection-binding (which actual `connection_id` satisfies the requirement) lives in `operator-state.json`, not the scenario manifest. See [`connections.md`](connections.md#how-scenarios-bind-to-connections).

## Remaining connector work

The first concrete connector is the OpenRouter API-key pilot in `scenarios/integration-hub`. Designing the complete schema before more drivers would lock in shapes that do not survive contact with reality. Per the discipline in [`../README.md`](../README.md): build for one, generalize after three.

The next useful drivers are likely `fal-api` (another paste-string API key) and `github-oauth` (OAuth web). Until they ship, this page is a design guide for those extensions rather than a claim that those providers are wired.

## See also

- [`connections.md`](connections.md) — connection instances, `bound_to`, scratch state
- [`../architecture.md`](../architecture.md) — remaining integration work
- [`../scenarios.md`](../scenarios.md) — the `integrations` field a scenario manifest will declare
- [`README.md`](README.md) — discipline for adding integration pages
