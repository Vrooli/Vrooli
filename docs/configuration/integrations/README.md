# Integrations

This folder describes how Vrooli connects to external services — third-party APIs, OAuth providers, cloud services, coding-agent sign-ins, social-media accounts. The integration model is a connector + connection split, owned by the deferred `integration-hub` scenario.

## Read order

If you're new to the integration model, read in this order:

1. [`connectors.md`](connectors.md) — what a connector is (the *type* definition).
2. [`connections.md`](connections.md) — what a connection is (an *instance* of a connector for one operator).
3. [`external-auth.md`](external-auth.md) — catalog of auth patterns connectors can declare.

The model intentionally mirrors the well-trodden integration-platform pattern (Zapier, n8n, Pipedream, Nango, Composio): one connector definition, many connection instances, scenarios reference instances by id.

## What goes in this folder

- **`connectors.md`** — the connector concept, manifest sketch, capability declarations.
- **`connections.md`** — connection instances, bound vs unbound, Vault layout, lifecycle CLI.
- **`external-auth.md`** — catalog of supported auth patterns (`api_key`, `oauth_web`, `oauth_device`, `external_sign_in_command`, `app_password`).
- **Per-integration pages** — one file per *wired* connector or per concrete integration that doesn't fit the connector model. Examples below.

The discipline is: **a page lives here only when the integration is actually wired.** Speculative pages for "things we might support someday" rot. New pages are added when a scenario or resource ships that consumes the integration.

## Existing per-integration pages

- [`video-providers.md`](video-providers.md) — Seedance, Veo, Sora and the like. Worked example of wiring a pay-per-use AI video model via API key. (Will likely become a `fal-api` connector when integration-hub ships.)
- [`buf-bsr.md`](buf-bsr.md) — Buf Schema Registry sign-in. Optional after the proto-codegen pipeline switched to local plugins; only needed for refreshing vendored proto modules. Worked example of an `external_sign_in_command` integration that's wired in advance of integration-hub.

Other integrations (Cloudflare tunnels, GitHub, Slack, Telegram, Gmail, TikTok-account, Instagram-account, etc.) get pages as they are wired. The absence of a page means the integration is either not yet wired or wired ad-hoc; flag the gap as a backlog item rather than writing a speculative page.

## Why integrations are not resources

Some integrations look resource-shaped (a model API, a database service). The dividing line is documented in [`connectors.md`](connectors.md#connector-vs-resource--when-to-use-which). Short version:

- **Resource** — a runtime capability the system depends on; lifecycle-managed (start/stop/health), typically singleton.
- **Connector** — access to an external account or service; not lifecycle-managed by Vrooli; can have many instances per install.

A pay-per-use AI API is a borderline case. Today these live as resources because the credential plumbing was already there; some may migrate to `api_key`-pattern connectors as integration-hub matures.

## Status

The integration-hub scenario does not exist yet. Until it does:

- Paste-string credentials live as resource credentials (the existing pattern). See [`../secrets.md`](../secrets.md).
- Loose / scratch keys for ad-hoc testing have no clean home. See the [scratch case](connections.md#unbound-connections-the-scratch-case) for the eventual story.
- Multi-instance integrations (the persona-actor case) are not supported. The marketing-crew persona accounts will land alongside or after integration-hub.

## See also

- [`../secrets.md`](../secrets.md) — paste-string credential layering for resources (the existing pattern)
- [`../architecture.md`](../architecture.md) — source-of-truth tables; integration-hub is on the deferred-scenarios list
