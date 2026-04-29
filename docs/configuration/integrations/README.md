# Integrations

This folder describes external integrations Vrooli can connect to — third-party APIs, OAuth providers, cloud services, coding-agent sign-ins. Each page describes one integration: what the operator needs to provide, how the system stores and consumes it, and what the wizard surfaces.

## What goes in this folder

Pages are added when an integration is *actually wired*. Speculative pages for "things we might support someday" do not belong here — they rot. The discipline is:

- A new integration is on a backlog or being scoped → no page yet.
- A scenario or resource ships that consumes the integration → add a page describing the operator-facing surface.
- The wizard adds a step for the integration → already documented here, by definition.

## Existing pages

- [`external-auth.md`](external-auth.md) — concept page for OAuth / device-flow / coding-agent sign-ins. Schema is intentionally deferred until the first concrete integration lands; this page describes the framework and the gap.
- [`video-providers.md`](video-providers.md) — Seedance, Veo, Sora and the like. Worked example of wiring a pay-per-use AI video model via API key.

Other integrations (Cloudflare tunnels, LLM providers, GitHub, Slack, etc.) will get pages as they are wired. The absence of a page means the integration is either not yet wired or wired ad-hoc; flag the gap as a backlog item rather than writing a speculative page.

## Two integration patterns

Integrations split into two patterns based on credential shape:

### Paste-string (API key)

Operator obtains a string from a provider's web UI and pastes it. The system stores it in Vault and reads it via `packages/api-core/secrets`. The full flow lives in [`../secrets.md`](../secrets.md); the integration page describes provider-specifics (where to obtain, naming conventions, gotchas).

### External-auth (OAuth / device flow / sign-in)

Operator performs an interactive flow (browser-based OAuth, device code, CLI sign-in) and a token is returned. The system stores the token, but the obtain-flow is *interactive*, not paste-and-paste. Schema is currently deferred — see [`external-auth.md`](external-auth.md).

Coding-agent sign-ins (Claude Code, Codex, etc.) are usually external-auth: the package is installed during setup, then the operator runs an external sign-in command outside the wizard. The wizard's role is to detect whether sign-in succeeded (a probe) and surface the result, not to handle the auth flow itself.

## See also

- [`../secrets.md`](../secrets.md) — paste-string credential layering
- [`../architecture.md`](../architecture.md) — source-of-truth tables and resolution order
