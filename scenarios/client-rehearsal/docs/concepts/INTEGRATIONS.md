# Integrations — Client Workflow Rehearsal

## Current slice

No external resource is required to build or use the local rehearsal adapter. It interprets a normalized synthetic workflow graph and resolves HTTP operations against OpenAPI descriptions or recorded exchanges. Native service nodes can select a declared profile operation by n8n node type, resource, and operation; the profile declares those mappings through `x-n8n-node-type`, `x-n8n-resource`, and `x-n8n-operation` extensions. This is reusable across workflows and never dispatches the node's real service code. OpenAPI catalogs and sanitized recorded exchanges can also export bounded exact routes into a Workspace Sandbox HTTP-mock profile. Recorded-exchange export requires successful responses on synthetic `.invalid` hosts with static paths; it rejects real hosts, credential-shaped response fields, and error responses. This produces a reusable mock profile, not proof that native n8n reached it. The tested local HTTP handler remains unattached to n8n.

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

The 512 KiB workflow request ceiling and simulator body policy are first-slice limits, not large-payload coverage. Before raising them, design bounded streaming/externalized evidence with explicit size, storage, retention, cleanup and replay behavior.
