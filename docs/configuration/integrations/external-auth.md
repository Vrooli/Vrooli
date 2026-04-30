# External Authentication Patterns

Catalog of the auth patterns connectors can declare. This page describes *how each pattern works* and *what the connector handler must implement*. The pattern names appear on connector manifests as `auth.kind`; see [`connectors.md`](connectors.md) for the connector model itself.

## Why this is a catalog, not a schema

The connector schema dispatches by `auth.kind`. Each pattern has its own per-pattern fields and runtime obligations. Listing them in one place keeps the dispatch table honest: a new pattern is added here first, with the contract spelled out, before any connector ships against it.

Patterns we currently anticipate. Five is enough for the foreseeable future; new patterns are exceptional.

## `api_key`

**What it looks like.** Operator obtains a string from a provider's web UI, pastes it once, the system stores it in Vault.

**Examples.** fal.ai, OpenRouter, Gemini, OpenAI, most pay-per-use AI APIs.

**Connector manifest fields under `auth`.**

```jsonc
"auth": {
  "kind": "api_key",
  "obtain_url":      "https://fal.ai/dashboard/keys",
  "default_hint":    "Starts with 'fal_'",
  "validation_pattern": "^fal_[A-Za-z0-9_-]+$"
}
```

These fields mirror `secretDescriptor` because the auth surface is identical — a paste-string secret. (`api_key`-pattern connectors are effectively `secretDescriptor`-shaped under a connector wrapper, and the schema reuses the type.)

**Vault contents.** A single `value` key holding the string.

**Probe contract.** The handler calls a known authenticated read-only endpoint (e.g. `/v1/keys/me` or equivalent) and reports `healthy` / `revoked` / `unknown`.

**Refresh contract.** No automatic refresh. Operator generates a new key in the provider UI and re-pastes; integration-hub updates Vault.

## `oauth_web`

**What it looks like.** Operator clicks "Sign in with X", browser opens to the provider, operator authorizes, browser returns to a callback URL with an authorization code, system exchanges the code for tokens.

**Examples.** GitHub, Google, Slack, Discord, most consumer-facing OAuth.

**Connector manifest fields under `auth`.**

```jsonc
"auth": {
  "kind":           "oauth_web",
  "authorize_url":  "https://github.com/login/oauth/authorize",
  "token_url":      "https://github.com/login/oauth/access_token",
  "scopes_offered": ["repo:read", "repo:write", "issues:write"],
  "callback_path":  "/integrations/github-oauth/callback"
}
```

The connector handler also embeds the client_id; the client_secret lives in Vault under `secret/vrooli/connectors/<connector_id>` (a separate Vault prefix from connection instances; this is the *connector's* secret, not any operator's).

**Vault contents.** `access_token`, `refresh_token` (when provider supports it), `expires_at`, `scopes`.

**Probe contract.** Handler calls a low-cost authenticated endpoint (e.g. `/user`) and parses `401` / `403` to distinguish `revoked` from `needs_refresh`.

**Refresh contract.** Handler runs the provider's refresh-token exchange before token expiry. Failure transitions the connection to `needs_refresh`; the wizard surfaces this for re-authorization.

**Callback handling.** Integration-hub exposes the callback URL on its API server; the handler parses the code, exchanges it, writes Vault. The callback URL must be registered with the provider as part of the connector's app configuration (a one-time human step, documented per connector).

## `oauth_device`

**What it looks like.** System requests a device code from the provider, displays it to the operator with a URL, operator visits the URL on any device and enters the code, system polls until the code is approved.

**Examples.** GitHub (alt flow), Google (limited-input devices), most providers that support headless or CLI sign-ins.

**Connector manifest fields under `auth`.** Similar to `oauth_web` but with `device_code_url` instead of `authorize_url`, and no `callback_path`. Polling parameters (interval, max wait) are handler-defined defaults, overridable per-provider.

**Vault contents.** Same as `oauth_web`.

**Probe and refresh.** Same as `oauth_web`.

**When to prefer over `oauth_web`.** When the operator runs Vrooli in a context without a browser available to receive the callback (headless server install, remote SSH session). The wizard can fall back to device flow when web flow can't reach back to the host.

## `external_sign_in_command`

**What it looks like.** Operator runs a sign-in command in their terminal — `claude /login`, `codex login`, `gh auth login`, etc. The third-party tool stores its own token in its own config dir. Integration-hub does not own the token; it owns *detection* of the sign-in state.

**Examples.** Coding agents (Claude Code, Codex, OpenCode), GitHub CLI, Cloudflare's `cloudflared`, Stripe CLI.

**Connector manifest fields under `auth`.**

```jsonc
"auth": {
  "kind":          "external_sign_in_command",
  "sign_in_command": "claude /login",
  "probe_command":   "claude whoami",
  "expected_substring": "Signed in as"
}
```

The connector handler invokes `probe_command` and matches against `expected_substring` (or a structured-output parser for more elaborate cases).

**Vault contents.** Usually empty — the third-party tool owns its own storage. If the connector needs to track metadata (last-probed identity, selected workspace), that lives in integration-hub state, not Vault.

**Probe contract.** Run the probe command. `0` exit + matching output = `healthy`; non-zero or no match = `unknown`. There is no automatic distinction between revoked and never-signed-in for this pattern; the probe just reports the live state.

**Refresh contract.** None. The third-party tool refreshes itself, or fails and prompts the operator to re-run sign-in.

**Why model these as connectors at all.** Because scenarios still need to *check* whether the operator has signed in before relying on the tool. A swarm-manager run that calls `codex` should fail-fast at startup if `codex login` was never run, not deep into a job. Modeling the sign-in as a connector gives onboarding a step to surface "you need to run X" and a probe to confirm.

## `app_password`

**What it looks like.** Operator generates a long-lived secondary password from the provider's UI (separate from their main password), pastes it, system stores in Vault. Distinct from `api_key` because the obtain UX is different — the provider's UI calls it an "app password" or "app-specific password", and the operator's mental model differs.

**Examples.** Bluesky's app passwords, Apple's app-specific passwords, some self-hosted services with limited OAuth support.

**Connector manifest fields under `auth`.** Same shape as `api_key` plus an `obtain_instructions` field that walks the operator through the provider's specific path to generate the password (often buried 3 clicks deep in account settings).

**Vault contents.** A `value` key, or sometimes a `username` + `value` pair when the provider requires both.

**Probe and refresh.** Same as `api_key` — no automatic refresh; operator regenerates if needed.

**Why not just call this `api_key`.** Because the obtain UX is materially different and operators get confused if the wizard says "paste your API key" for what the provider's UI calls "app password." Naming the pattern correctly makes onboarding's per-pattern guidance accurate.

## What this catalog does *not* cover

- **mTLS, SAML, signed-JWT-bearer, custom HMAC.** None of these have a foreseeable use case yet. When one shows up, it gets added here first with the contract spelled out, before any connector ships against it.
- **Per-provider quirks.** GitHub's PAT vs OAuth, Slack's bot tokens vs user tokens, Google's service accounts vs user OAuth — these are connector-level decisions (which `auth.kind` to declare and which scopes/endpoints), not new patterns.

## See also

- [`connectors.md`](connectors.md) — the connector model that consumes these patterns
- [`connections.md`](connections.md) — connection instances and Vault layout
- [`../secrets.md`](../secrets.md) — paste-string secrets attached to resources (the existing pattern, structurally the same as `api_key` connectors)
