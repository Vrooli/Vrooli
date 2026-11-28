# Integrations — Client Workflow Rehearsal

## Current slice

No external resource is required for ordinary local rehearsal. An opt-in `task_scoped` request uses the configured loopback Workspace Sandbox API. Client Rehearsal mints the logical task ID; Workspace Sandbox resolves the task directory and owner. OpenAPI catalogs and sanitized recorded exchanges can export bounded exact routes into a task-scoped companion. The native n8n path uses the HTTPS `virtual-api` companion; the generic simulator path retains the HTTP-mock companion. OpenAPI native service routes require explicit node/resource/operation bindings and a pinned provider host; those bindings remain owner-only and are omitted from the Sandbox declaration. Unbound native service operations fail closed before runtime start. Both companions serve fixed synthetic responses and do not forward requests upstream. Route-hit receipts are retained in the handoff. Source tests exercise the contract, but actual API-to-task-network DNS/TLS routing and deny-by-default egress remain unverified.

## Planned owner integrations

| Capability | Owner | Contract required before activation |
|---|---|---|
| Managed n8n engine | Resource owner | Managed lifecycle; secret-free typed import/trigger/evidence broker; per-task synthetic credential objects; short-lived API-key provisioning/expiry/revocation and redaction; deny-by-default execution egress; license boundary preserved. |
| Simulator bridge | Rehearsal + resource/sandbox owners | Route exact documented service hosts to description-driven simulator; no fallback forwarding; test pinned-image HTTP, TLS, aliases, redirects and external DNS denial. |
| Built-in app nodes | Rehearsal + resource owner | Prove Gmail, Sheets and CRM node behavior with synthetic executable credentials and task-scoped routing. HTTP Request coverage alone is insufficient. |
| Native protocol nodes | Rehearsal + resource/sandbox owners | Protocol-aware virtual service (for example Postgres) or explicit fail-closed unsupported result. HTTP routing is insufficient. |
| Engagement-scoped companion services | Workspace Sandbox | Typed declaration, lifecycle, teardown, service isolation, and deny-by-default external egress. |

Do not declare an unverified resource dependency in `.vrooli/service.json`. No container may be started or pulled until the host admission owner has measured capacity and the lifecycle/network isolation path has been proven.

The owner must not return an n8n API key in `Start` or CLI/control-plane JSON. n8n documents API-key creation through Settings with an explicit expiration and the `X-N8N-API-KEY` header; a public REST key-minting operation is not established. The pinned self-hosted runtime needs a verified owner-controlled bootstrap path before the typed broker is available. Manual, webhook, scheduled, polling, chat, subworkflow, and error-trigger execution each need native-runtime proof with sanitized execution evidence. The local graph simulator and its HTTP test handler do not provide this proof.

Workflow import remains capped at 512 KiB. The simulator streams and discards HTTP request bodies up to 16 MiB without storing payload bytes; larger requests are rejected before fixture matching. This provides bounded large-request handling only: larger workflow exports and end-to-end/native large-payload replay remain open. Any higher limit or retained payload evidence needs an explicit storage, retention, cleanup, and replay policy.
