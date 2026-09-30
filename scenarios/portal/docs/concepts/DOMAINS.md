# Domains — Portal

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Portal's real domains are listed below. Add to this map when a new product
capability owns data, proto methods, API behavior, UI state, or CLI commands.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Let lifecycle tools and operators see whether Portal is healthy. | No product data. | reporting | query | health, dependency | `api/handlers/health/`, `packages/proto/schemas/portal/v1/shared/health.proto` |
| chat | Own grouped conversations and branchable message-tree state. | Persist the operator's conversation workspace. | chats, chat_groups, messages, settings, usage, search attachments. | service | mutation, query | chat, group, message tree | `api/internal/chat/`, `api/handlers/chat/`, `api/handlers/message/`, `packages/proto/schemas/portal/v1/chat/`, `packages/proto/schemas/portal/v1/message/`, `ui/src/features/chat/`, `cli/domains/chat/`, `cli/domains/message/` |
| completion | Assemble and stream LLM completions. | Turn persisted conversation state into OpenRouter requests with honest degraded errors. | Usage records through chat repository. | orchestration | service | completion, OpenRouter, skill context | `api/internal/completion/`, `api/internal/integrations/openrouter/`, `api/handlers/message/` |
| agentchat | Bridge chat mode to agent-manager. | Let coding-agent conversations live in the same chat workspace. | Final agent transcript through chat repository. | orchestration | service | agent harness, activity | `api/internal/agentchat/`, `api/internal/integrations/agentmanager/`, `api/handlers/message/` |
| integrations | Measure optional dependency readiness and behavior mode. | Keep Portal usable and honest when dependencies degrade. | integration overrides and rolling observations. | service | reporting, classification | readiness, behavior mode, override | `api/internal/integrations/registry/`, `api/handlers/integrations/`, `packages/proto/schemas/portal/v1/integrations/`, `ui/src/features/integrations/`, `cli/domains/integrations/` |
| search | Mediate ecosystem suggestions and passive search attachments. | Surface Vrooli capabilities without gating chat completions. | Search attachments through chat repository. | query | aggregation | suggestion, attachment, passive search | `api/internal/search/`, `api/handlers/search/`, `packages/proto/schemas/portal/v1/search/`, `ui/src/features/search/`, `cli/domains/search/` |
| brief | Build one gated, trust-labelled context brief for every consumer. | Serve current-turn LLM, agent, and verified external-harness context from one producer. | briefs, brief_items, brief_uses. | scoring | query, rendering, telemetry | verdict, trust class, withheld reason | `api/internal/brief/`, `handlers/brief/`, `packages/proto/schemas/portal/v1/brief/`, `ui/src/features/brief/`, `cli/domains/brief/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: health status is surfaced through shell readiness indicators and lifecycle status.
- Storage: none; probes configured database reachability.
- Requirements: `PORTAL-P0-001`.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### chat

- Purpose: persist the chat workspace: grouped chats, message branches, active
  leaf selection, search attachments, settings, and usage records.
- API: `ChatService` and `MessageService` unary methods.
- CLI: `portal chats ...` and `portal messages ...`.
- UI: `ChatWorkspace`.
- Storage: declarative SQLite schema in `api/internal/chat/schema.sql`.
- Requirements: `PORTAL-P0-002`, `PORTAL-P1-001`, `PORTAL-P2-001`.

Chat repositories now support a server-bound owner context. `chat_owners` and
`chat_group_owners` are created atomically with their parent records; missing
ownership rows identify legacy records. Account scopes see only their records,
and unbound contexts see only legacy records. Message branches, search
attachments, and usage writes validate their parent conversation membership.
ChatService and MessageService bind validated bearer identities for unary and
streaming calls; missing credentials retain legacy access, while invalid
credentials are rejected. Agent admission owners are stored independently of
the source chat, so recovery and Stop remain account-scoped after chat deletion.
The UI still needs account-scoped chat requests before private attachments are enabled.

### completion

Completion supports a `MessageImageResolver` boundary for authenticated, live
message-to-rendered-image associations. It resolves only user messages on the
selected branch and aborts the completion when image resolution fails. The
production resolver is not wired yet: existing shared chat storage must acquire
account ownership before private context or derived replies enter it.

The OpenRouter adapter supports ordered text and private PNG image parts, with
four images and 32 MiB total image bytes per request. It validates PNGs, sends an
image-bearing request once, and omits provider error bodies from image errors.
Wire format follows [OpenRouter image inputs](https://openrouter.ai/docs/guides/overview/multimodal/image-understanding).


- Purpose: build OpenRouter requests from the active branch path and stream
  completion events back through `MessageService.StreamCompletion`.
- Owns: request assembly, selected-skill system-prompt injection, token events,
  assistant-message persistence, and usage capture.
- Does not own: OpenRouter credentials, chat persistence schema, or UI rendering.
- Requirements: `PORTAL-P1-001`.

### agentchat

- Purpose: map agent chat mode to agent-manager runs and stream normalized
  activity events.
- Owns: runner mapping, WebSocket event decoding, terminal transcript
  persistence.
- Does not own: terminal surfaces or agent-manager internals.
- Requirements: `PORTAL-P1-002`.

### integrations

- Purpose: maintain Portal's own measured readiness view for optional
  dependencies.
- Owns: rolling latency/error windows, hysteresis policy, override persistence,
  and status proto conversion.
- Requirements: `PORTAL-P2-002`.

### search

- Purpose: mediate calls to search-hub for suggestions and passive
  send-time attachments.
- Owns: budgets, result projection, degraded responses, readiness observations,
  and next-turn context material.
- Requirements: `PORTAL-P2-001`.

### brief

- Purpose: query Search Hub for the current prompt, classify result trust,
  apply consumer-specific policy and thresholds, and persist the verdict.
- Owns: `BriefService`, brief storage, Connect API, CLI, inspector data, and
  the integration seam used by LLM, agent, and external-harness flows.
- Does not own: Search Hub provider indexing, native hook installation, or
  command execution.
- API: `BriefService.Build`, `Get`, `List`, `RecordUse`.
- Storage: declarative SQLite schema in `api/internal/brief/schema.sql` with
  thirty-day retention and cascading item/use deletion.
- Requirements: `PORTAL-P2-003`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| PASSIVE mode | Search enriches the conversation but never delays the completion path. | `search`, `integrations`. |

## Portal Everywhere domains under implementation

These obligations are active under the Portal Everywhere plan. The paths below
describe intended placement; they are not claims that an implementation exists.
The requirement modules retain planned status until producer evidence earns it.

| Domain | Portal responsibility | Execution owner | Requirements |
|---|---|---|---|
| surfaces | Aggregate safe owner references, source failures, freshness, and target selection; host interactive surfaces. | Device Control for desktops/devices, BAS for browsers, Web Console for terminals, scenario owners for embeds. | `requirements/02-everywhere/module.json` CAT and EMB cases |
| taskrouting | Resolve equivalent authorized routes and reconcile uncertain results before retry. | Program Runtime composes typed owner operations; destination owners enforce grants. | OPT cases |
| companion | Present pill, palette, and the ordinary expanded workspace with shared identities. | Scenario-to-Desktop packages the shell; Device Control owns the user-session helper. | UI cases |
| context | Capture explicit bounded context and annotations with source geometry and privacy. | Device Control provides desktop capture; existing artifact owners retain evidence. | UI-03, UI-06, UI-12 |
| voice | Finalize optional speech input once and expose playback controls and permission recovery. | Audio Tools provides optional speech operations. | OPT-02, UI-11 |
| learning | Measure comparable effort and verified outcomes without storing raw screenshots as learning records. | Vrooli Memory retains bounded provenance; execution owners provide receipts. | LEARN cases |
| migration | Reconcile Assistant issue records and route replacement capture to current owners. | Existing reporting and agent owners receive tasks. | `assistantmigration/`, `cli/domains/assistantmigration/`; MIG cases |

The context domain's initial import boundary lives in
`api/internal/contextcapture/`. It validates original PNG pixels, source
provenance, bounded region/strokes, and an explicit retention duration before a
storage operation. Source metadata is declared provenance, not proof of native
capture or authority to act. The domain now includes a SQLite metadata repository and a service using
`api-core/blobstore` for bytes. Quota reservation precedes image writes; staging
records are unreadable, publication follows successful writes, and expired
records remain tracked until blob deletion succeeds. Reads enforce exact owner,
expiry, MIME, size, and image digest. `handlers/contextcapture/` now supplies account-authenticated Import, Read, and
Delete RPCs with no JSON owner field and no-store responses. Runtime registration uses a private storage-resolver directory and runs bounded
expiry cleanup at startup and every minute. Routed tests bind to the selected
test pool and separate in-memory blobs. `portal context import|read|delete` provides authenticated CLI access; image
exports use new private files and metadata-only output. Caller-supplied import request UUIDs resolve identical retries to the same
artifact and expiry; bounded 24-hour request receipts prevent revival after
deletion. Read-only request-ID reconciliation and chat attachment integration
remain pending.

Shared references live in `packages/api-core/targetmodel/`; this package is a
contract, not an inventory database. Native connection offers and grants remain
behind owner admission, never in embed URLs or catalog descriptors. The
companion's machine identity must be supplied explicitly, independently of the
Portal API host.

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| FULL retrieval gate | Pre-LLM short-circuiting is plan-3 scope. | Retrieval eval harness and FULL policy are implemented. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy

## Surface catalog implementation

The surfaces domain exposes read-only SurfaceCatalogService.List and Resolve
with CLI mirrors. Source paths: `api/internal/surfaces/`,
`api/handlers/surfaces/`, `cli/domains/surfaces/`, and
`packages/proto/schemas/portal/v1/surfaces/`. It owns no database. Production
currently consumes Web Console's typed target catalog, the optional Device
Control desktop owner, and Compute Manager's screenless instance catalog.
Compute instances are projected as device-panel capabilities with explicit
bridge host identity; they never become desktop surfaces. Additional provider
attestation and the interactive UI remain under implementation.
