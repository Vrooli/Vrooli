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

### completion

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

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| PASSIVE mode | Search enriches the conversation but never delays the completion path. | `search`, `integrations`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| scenario embeds | Embeds are plan-2 scope. | Scenario embed plan begins. |
| FULL retrieval gate | Pre-LLM short-circuiting is plan-3 scope. | Retrieval eval harness and FULL policy are implemented. |
| voice input | Voice is out of v0. | Voice plan begins. |

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
