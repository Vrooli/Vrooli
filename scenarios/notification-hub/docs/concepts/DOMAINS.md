# Domains — Notification Hub

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The shape of this scenario follows one sentence: *something happened,
decide who should hear about it and how, then make sure they did.* Each
domain owns one clause of that sentence.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/notification-hub/v1/shared/health.proto` |
| recipients | Know who can be told, how to reach them, and when not to. | Hold the addressing and preference facts every other domain reads. | Recipients, devices, channel addresses, quiet windows, per-channel preferences. | crud | policy | Recipient, Device, ChannelAddress, QuietWindow | `api/internal/recipients/`, `api/handlers/recipients/`, `cli/domains/recipients/`, `ui/src/features/recipients/` |
| notifications | Accept a request and own its lifecycle. | Give every caller one durable id and one observable state machine. | Notifications, state transitions. | workflow | crud | Notification, Urgency, Sensitivity, DedupeKey | `api/internal/notifications/`, `api/handlers/notifications/`, `cli/domains/notifications/`, `ui/src/features/notifications/` |
| routing | Decide which channels a notification takes and by which path. | Concentrate every "who, what channel, local or relayed" judgment in one testable place. | No tables; writes its reasoning onto the delivery row. | policy | query | RoutingDecision, DeliveryPath | `api/internal/routing/` |
| channels | Declare what this host can actually send, and bind provider credentials. | Make host capability an explicit fact rather than an inference from config. | Channel health and capability declarations. | integration | registry | Channel, ChannelAdapter, HostCapability | `api/internal/channels/`, `api/handlers/channels/`, `ui/src/features/channels/` |
| delivery | Perform the send, retry it, and record what happened. | Turn a routing decision into an auditable outcome. | Deliveries, attempts, receipts, retry schedule. | workflow | integration | Delivery, Attempt, Receipt | `api/internal/delivery/`, `api/handlers/delivery/`, `cli/domains/delivery/`, `ui/src/features/timeline/` |

Two further domains are committed at P1 and are listed here because they
shape the P0 schema and seams rather than because they exist yet:

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Target |
|---|---|---|---|---|---|
| relay | Forward a delivery to a fleet node that can serve the channel. | Reach devices this host cannot reach. | Node capability cache, relay correlation. | integration | OT-P1-001, OT-P1-002 |
| ingress | Turn inbound events and webhooks into notification requests. | Let the fleet raise notifications without calling the API directly. | Ingress rules, inbound receipts. | integration | OT-P1-003 |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/notification-hub/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: channel provider health, which belongs to `channels`.
  A reachable database does not mean a reachable push provider, and
  conflating the two hides the failure that actually matters here.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### recipients

- Purpose: hold every fact needed to address a human, and every rule
  about when addressing them is unwelcome.
- Primary archetype: CRUD / registry.
- Secondary traits: policy data read by `routing`.
- Owns: recipients keyed by scenario-authenticator identity subject;
  devices belonging to a recipient; the channel addresses each device
  accepts; quiet windows; per-channel mute and urgency floors.
- Does not own: identity itself. A recipient row is a local projection
  of an authenticator subject, never a credential store. It holds no
  password, no token, and no scenario-issued key.
- Does not own: the decision to skip a delivery. It states the rule;
  `routing` applies it.
- API: `api/internal/recipients/`, `api/handlers/recipients/`.
- CLI: `cli/domains/recipients/` — register a device, list devices,
  set quiet hours, mute a channel.
- UI: `ui/src/features/recipients/`.
- Storage: `api/internal/recipients/schema.sql`.
- Requirements: OT-P0-003 (device and channel registry), OT-P0-005
  (quiet hours), OT-P0-007 (identity without local accounts).
- Tests: repository, service, handler, CLI, UI, accessibility. Quiet
  window evaluation is a pure function and carries table-driven tests
  across timezone and midnight-crossing cases.

### notifications

- Purpose: accept a request, give it a durable identity, and make its
  progress observable.
- Primary archetype: workflow / entity.
- Secondary traits: CRUD for read surfaces.
- Owns: the notification record, its urgency and sensitivity labels, its
  dedupe key and window, its scheduled time, and its append-only state
  transitions.
- Does not own: how it gets delivered. A notification records intent; a
  delivery records an attempt. One notification can produce several
  deliveries and outlives all of them.
- Does not own: duplicate detection policy across recipients — the
  dedupe key is caller-supplied and scoped per recipient.
- API: `api/internal/notifications/`, `api/handlers/notifications/`.
- CLI: `cli/domains/notifications/` — send, list, show, cancel.
- UI: `ui/src/features/notifications/`.
- Storage: `api/internal/notifications/schema.sql`.
- Requirements: OT-P0-002 (direct request), OT-P0-004 (observable
  state), OT-P0-006 (duplicate suppression), OT-P0-010 (sensitivity
  labelling).
- Tests: state-machine transition tests are the centre of gravity here;
  every illegal transition has a negative test.

### routing

- Purpose: hold every judgment about *where a notification goes* in one
  place that can be tested without a network.
- Primary archetype: policy / decision service.
- Secondary traits: query over `recipients` and `channels`.
- Owns: channel selection from urgency, sensitivity, preference, and
  device capability; the local-versus-relay path decision; the ordered
  fallback list when a preferred channel is unavailable.
- Owns no tables. Its output is a value written onto the delivery row,
  including a stated reason, so any delivery can explain itself later.
- Does not own: sending. It never opens a socket, which is what makes it
  exhaustively unit-testable.
- API: `api/internal/routing/`.
- CLI: exposed only as an explain command (`notifications explain`)
  that shows the decision without performing it.
- UI: surfaced as the "why this channel" detail on a timeline entry.
- Storage: none.
- Requirements: OT-P0-005, OT-P0-010, OT-P1-001.
- Tests: table-driven decision tests with no I/O; the fixture set is the
  specification.

### channels

- Purpose: state what this host can actually send, and bind each channel
  to its credentials and health.
- Primary archetype: integration / registry.
- Secondary traits: capability declaration consumed by `routing` and,
  later, advertised to the fleet by `relay`.
- Owns: the adapter registry, per-channel enablement, credential
  references resolved through the resource credential descriptors, and
  the last observed provider health.
- Does not own: credential values. It holds a reference; the credential
  authority holds the secret.
- Does not own: what a given recipient prefers — that is `recipients`.
  This domain answers "is push possible from this machine at all".
- API: `api/internal/channels/`, `api/handlers/channels/`.
- CLI: `cli/domains/channels/` — list channels, check a provider.
- UI: `ui/src/features/channels/`.
- Storage: `api/internal/channels/schema.sql`.
- Requirements: OT-P0-001, OT-P0-009 (no Docker-bound dependency),
  OT-P1-004, OT-P1-008.
- Tests: each adapter has a contract test against a fake transport; no
  test performs a real send.

### delivery

- Purpose: carry out a routing decision, retry it under a budget, and
  record an auditable outcome.
- Primary archetype: workflow / integration.
- Secondary traits: scheduling.
- Owns: one row per attempt, its state, the provider message id, the
  error code and detail on failure, the retry schedule, and the receipt.
- Does not own: the retry *policy* shape for a channel, which the
  channel adapter declares.
- Does not own: cross-node forwarding, which `relay` adds at P1. The
  delivery row already carries a nullable node reference so that
  addition needs no migration.
- API: `api/internal/delivery/`, `api/handlers/delivery/`.
- CLI: `cli/domains/delivery/` — list attempts, retry, show receipt.
- UI: `ui/src/features/timeline/` — the scenario's primary screen.
- Storage: `api/internal/delivery/schema.sql`.
- Requirements: OT-P0-001, OT-P0-004, OT-P0-008 (retry with terminal
  failure).
- Tests: retry and backoff are tested against an injected clock; the
  scheduler never sleeps in a test.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Urgency | How much interruption a notification is worth: `low`, `normal`, `high`, `critical`. Only `critical` overrides a quiet window. | `notifications`. |
| Sensitivity | How much of the notification may leave the machine: `public`, `private`, `secret`. Governs whether a body is sent or withheld. | `notifications`; enforced by `routing`. |
| Channel | A way of reaching a device — push, iMessage, email, SMS, desktop. | `channels`. |
| Delivery path | `local` when this host sends it, `relayed` when a fleet node does. | `routing`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `templates` | The old scenario carried a template engine and a versioned template table, and neither earned its place — every real notification body so far is one sentence plus a link. | A caller needs the same notification rendered per-locale, or bodies start being assembled in three or more callers. |
| `analytics` | Delivery counts are derivable from the delivery table by query. A domain is warranted only when rollups outgrow that. | OT-P1-007 measurement shows query latency is unacceptable on the real table size. |
| `acknowledgement` | Return-path handling (OT-P2-004) needs a public callback surface and a response model that P0 does not settle. | A caller needs the human's answer, not just proof of delivery. |
| `escalation` | OT-P2-005 depends on acknowledgement existing first. | Acknowledgement ships. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- The in-process work queue — infrastructure owned by `delivery`, not a
  domain. If it ever grows a vocabulary of its own, that is the signal
  it wanted to be a resource instead.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
