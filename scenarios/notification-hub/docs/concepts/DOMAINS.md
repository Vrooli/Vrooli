# Domains — Notification Hub

Domain map for the scenario: the product capability boundaries, what each
owns, and what deliberately is not a domain.

## Purpose Of This Document

This document is the boundary authority. It decides where a capability
lives before any code is written, so a later agent adds behavior to the
domain that already owns the concept instead of creating a second owner
for it.

A domain here is a product capability boundary that should be easy to
find, easy to test, and easy to delete. Each one owns its own proto
package, SQLite schema, service and repository seams, handler module, CLI
group, and UI feature folder.

The map is deliberately small. Five product domains cover all 15 P0
targets. Splitting further would create boundaries that never vary
independently, and every extra boundary is a place where a future change
has to be applied twice.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| recipients | Know who can be told, on what, and when they want to be left alone. | Give every other domain one answer to "where does a person actually receive things". | Recipients, devices, channel addresses, push subscriptions, quiet windows, escalation chains. | registry | crud | Recipient, Device, ChannelAddress, QuietWindow | `api/internal/recipients/`, `cli/domains/recipients/`, `ui/src/features/recipients/` |
| notifications | Accept a request and own its lifecycle. | Give every caller a durable id immediately, and make the state of that id answerable forever after. | Notifications, idempotency keys, dedupe keys, state transitions. | service | entity | Notification, SensitivityLabel, IdempotencyKey, DedupeKey | `api/internal/notifications/`, `cli/domains/notifications/`, `ui/src/features/notifications/` |
| routing | Decide which channels serve a notification, on which machine, and at what time. | Concentrate every "should this be sent, here, now" judgement in one testable place. | Routing decisions, quiet-hour holds, suppression records. | policy | service | RoutingDecision, Hold, Suppression | `api/internal/routing/`, `ui/src/features/routing/` |
| delivery | Carry an accepted notification to a channel and record what happened. | Own every channel adapter and every attempt, so adding a channel touches one domain. | Delivery attempts, receipts, per-machine channel status cache. | worker | integration | DeliveryAttempt, Receipt, ChannelAdapter, ChannelStatus | `api/internal/delivery/`, `cli/domains/channels/`, `ui/src/features/delivery/` |
| conversations | Carry a decision back from the person to the caller. | Turn an outbound pipe into the fleet's human-in-the-loop gate. | Asks, answers, deadlines, escalation steps. | service | state-machine | Ask, Answer, Escalation | `api/internal/conversations/`, `cli/domains/ask/`, `ui/src/features/conversations/` |
| health | Report runtime readiness and dependency reachability. | Expose API and database readiness, and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/` |

## Domain Details

### recipients

- Purpose: give every other domain one answer to where a person actually
  receives things.
- Primary archetype: registry.
- Owns: recipient records keyed by `scenario-authenticator` identity;
  devices belonging to a recipient; one channel address per device and
  channel; push subscription material; quiet windows; per-channel mute;
  escalation chain order.
- Does not own: identity itself. There is no password, no profile table,
  and no scenario-issued API key here (OT-P0-007). A recipient row is a
  projection of a verified external identity, created on first sight.
- Why preferences live here and not in `routing`: a quiet window is
  meaningless without a recipient, the UI edits both on one screen, and
  the two never vary independently. `routing` reads preferences; it does
  not store them.
- Storage: domain-owned schema in `api/internal/recipients/schema.sql`.
- Requirements: OT-P0-003, OT-P0-007, OT-P0-014, OT-P2-003.

### notifications

- Purpose: give every caller a durable id immediately, and keep the state
  of that id answerable forever after.
- Primary archetype: service over an entity.
- Owns: the notification record, its sensitivity label, its idempotency
  key, its dedupe key, and its state machine. Owns every ingress adapter:
  Connect-RPC, CLI, and the webhook receiver that `vrooli-events` and
  third parties post to.
- Does not own: the decision about where a notification goes, or the act
  of sending it. It records that a decision was made and what came back.
- Why ingress is not its own domain: an ingress path is a different way
  to construct the same record. A separate domain would own no data and
  would force every intake change across two boundaries.
- Storage: domain-owned schema in `api/internal/notifications/schema.sql`,
  including the retention rule required by OT-P0-013.
- Requirements: OT-P0-002, OT-P0-004, OT-P0-010, OT-P0-012, OT-P0-013,
  OT-P1-003.

### routing

- Purpose: concentrate every "should this be sent, here, now" judgement in
  one testable place.
- Primary archetype: policy engine.
- Owns: channel selection by urgency and label, quiet-hour holds,
  duplicate suppression, the default-deny sensitivity rule and its
  content-free fallback, and the unroutable verdict with its stated
  reason.
- Does not own: preferences (read from `recipients`), channel reachability
  (read from `delivery`), or the send itself.
- Why this is a domain rather than a service inside `notifications`: it is
  the only part of the scenario where a wrong answer is silent rather than
  loud. It needs its own tests, its own fixtures, and a boundary that
  makes "what would this notification do" answerable without sending
  anything.
- Storage: holds and suppression records in
  `api/internal/routing/schema.sql`. A routing decision is recorded, never
  recomputed after the fact, so the timeline can explain a past choice.
- Requirements: OT-P0-005, OT-P0-006, OT-P0-010, OT-P0-011.

### delivery

- Purpose: own every channel adapter and every attempt, so adding a
  channel touches one domain.
- Primary archetype: worker.
- Owns: the adapter registry, the attempt record, retry scheduling and
  backoff, receipts, and the per-machine channel status cache that backs
  cross-node delivery. Owns the `notification-hub channels status` readiness
  verb and the `notification-hub notifications relay` verb used for durable
  remote delivery.
- Does not own: whether a notification should be sent. By the time
  `delivery` sees a notification, `routing` has already decided.
- Why cross-node delivery is not its own domain: from the routing
  engine's view, a channel on another machine is a channel with a
  different address. The remote hop is a delivery concern, and modelling
  it separately would split the adapter registry in two.
- Storage: attempts, receipts, and the channel status cache in
  `api/internal/delivery/schema.sql`.
- Requirements: OT-P0-001, OT-P0-004, OT-P0-008, OT-P1-001, OT-P1-002,
  OT-P1-004, OT-P2-002.

### conversations

- Purpose: turn an outbound pipe into the fleet's human-in-the-loop gate.
- Primary archetype: service over a state machine.
- Owns: an ask and its allowed answers, the deadline, the answer once
  given, and the escalation steps taken when no answer arrives. Owns the
  blocking call convention a caller uses to wait for a decision.
- Does not own: delivery of the question, which is an ordinary
  notification, or the recipient's escalation chain, which belongs to
  `recipients`.
- Why this is a domain and not a flag on a notification: the pending state
  must survive an API restart, it has its own deadline sweeper, and its
  failure mode — a caller blocked forever — is unrelated to any delivery
  failure mode.
- Storage: asks, answers, and escalation steps in
  `api/internal/conversations/schema.sql`. The pending-ask registry is
  durable, not in-memory, so a restart between question and answer does
  not strand a caller.
- Requirements: OT-P1-009, OT-P1-010, OT-P1-011.

### health

- Purpose: expose API and database readiness, and show the UI can read
  live backend state.
- Primary archetype: reporting.
- Owns: health response construction and dependency status mapping.
- Does not own: product data or business rules.
- Storage: none.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Channel | A way to reach one device: web push, email, SMS, or a host-bound facility such as iMessage. | `delivery`. |
| Sensitivity label | Caller-supplied statement about whether a body may appear on a locked screen or in a third party's logs. | `notifications` carries it; `routing` enforces it. |
| Disposition | A channel's answer to "can you serve this right now", with a stated reason. | `delivery`. |
| Machine | A durable fleet member. Distinct from a bridge node, which is one registration of a machine. | `vrooli-bridge` owns the identity; `delivery` addresses it. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| analytics | OT-P1-007 is a read model over `delivery` attempts. Building it before the attempt table has a real shape would fix that shape prematurely. | When delivery volume makes the timeline an inadequate answer to "what is failing". |
| digests | OT-P1-005 collapses low-urgency notifications into one scheduled send. It is a `routing` behavior until it needs its own composition rules and templates. | When a digest needs its own content assembly rather than a list of collapsed titles. |
| schedules | OT-P1-006 delivers at a future time. The hold mechanism `routing` already needs for quiet hours covers the simple case. | When callers need recurrence, calendars, or timezone-aware windows. |

## Non-Domains

These are important and should not become product domains.

- **Identity.** Owned by `scenario-authenticator`. This scenario verifies
  tokens locally against published JWKS and stores no credential.
- **Fleet reach.** Owned by `vrooli-bridge`: which machines exist, whether
  they are online, and whether this caller may send them anything.
  Modelling a second registry here would split the single answer to what
  the owner controls.
- **Event distribution.** Owned by `vrooli-events`. This scenario is a
  subscriber, not a bus.
- **Public exposure.** Owned by `tunnel-manager`.
- **Templates.** Notification bodies are composed by the caller. A
  template authoring surface is an explicit non-goal in the PRD.

## Cross-References

- [`FLOWS.md`](FLOWS.md) — how these domains interact per request.
- [`DATA.md`](DATA.md) — the tables each domain owns.
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — the scenarios this one depends on.
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — substitutable boundaries.
- [`../../PRD.md`](../../PRD.md) — the operational targets referenced above.
