# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario notification-hub`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose** — Notification Hub is the owner's notification spine. It accepts an event or a direct request, decides who should be told, on which device, through which channel, and at what time, then delivers it and records what happened. When this host cannot reach a target device itself, it relays the delivery through another machine in the owner's fleet.

The permanent capability it adds is *reaching a human reliably*. Every scenario and agent in Vrooli can produce something worth telling someone about; none of them should own retry logic, quiet hours, device addresses, or channel credentials. Those live here once.

The traffic runs both ways. Once a notification can carry a decision back (OT-P1-009 through OT-P1-011), this stops being an outbound pipe and becomes the human-in-the-loop gate for the whole fleet: an agent that needs permission, or that has stalled, asks through here and waits for an answer. That is the capability neither a hosted notification platform nor a self-hosted push pipe provides — the former has no fleet to route across, the latter has no routing layer at all. Restraint is part of the same job: a spine that fires too often gets muted, and a muted spine voids every promise downstream of it.

**Primary users**

- The machine owner, who registers devices, sets quiet hours, and reads the delivery timeline.
- Agents and scenarios that need to reach a human and should not implement delivery themselves.
- Additional authenticated people on the same install when the operator runs a `shared` or `hosted` trust posture.

**Deployment surfaces**

- Go API over Connect-RPC — the contract every scenario calls.
- `notification-hub` CLI — operator and agent access, including `--node` targeting for fleet operations.
- React UI — device and rule management, plus the delivery timeline.
- Webhook receiver — a REST exception for third-party and `vrooli-events` callbacks.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Real delivery to a personal device | The system shall deliver a notification to the owner's registered iPhone through Web Push from the installed progressive web app, and shall record a delivery receipt for it. The delivery shall depend on no Vrooli resource, and the body shall be encrypted so no push service can read it.
- [x] OT-P0-002 | Direct notification request | When an agent, scenario, or operator submits a notification request, the system shall persist it and shall return a durable notification id.
- [x] OT-P0-003 | Device and channel registry | The system shall record the owner's devices, the channels each device accepts, and the address used for each channel.
- [x] OT-P0-004 | Observable delivery state | The system shall record every delivery attempt with its outcome and shall expose the state of any notification through the API, the CLI, and the UI.
- [x] OT-P0-005 | Quiet hours | When a notification falls inside a configured quiet window, the system shall hold it until the window closes unless its urgency is marked critical.
- [x] OT-P0-006 | Duplicate suppression | When repeated requests carry the same dedupe key inside its window, the system shall collapse them into one delivery.
- [x] OT-P0-007 | Identity without local accounts | The system shall resolve identity from scenario-authenticator tokens and shall key recipients by that identity, so no password, profile table, or scenario-issued API key exists here.
- [x] OT-P0-008 | Retry with terminal failure | When a delivery attempt fails retryably, the system shall retry with backoff, and shall mark the delivery failed with a stated reason once the retry budget is spent.
- [x] OT-P0-009 | Runs without Docker | The system shall declare no resource dependency that requires Docker, so the same build runs on a Linux host and on a macOS fleet node.
- [x] OT-P0-010 | Sensitivity labelling with default-deny | The system shall require a sensitivity label on every notification at the ingress contract, and shall treat a channel as unapproved for a label unless that channel declares approval. When a channel is unapproved, the system shall deliver a content-free pointer to the console instead of the body, and shall never drop the notification silently.
- [x] OT-P0-011 | Unroutable delivery is visible | When no approved channel and no reachable node can serve a notification, the system shall mark it unroutable with a stated reason and shall surface it through the API, the CLI, and the UI. It shall never leave such a notification pending, because a silently pending notification is indistinguishable from one that was never requested.
- [x] OT-P0-012 | Request idempotency | When a caller retries a request carrying the same idempotency key, the system shall return the original notification id and shall not create a second notification. This is distinct from OT-P0-006: it protects a caller whose request timed out in flight, regardless of any dedupe window.
- [x] OT-P0-013 | Bounded delivery history | The system shall enforce a stated retention rule on notification and delivery-attempt records, and shall record that rule in the data documentation. Retry multiplies attempt rows, so an unbounded history is a defect rather than a deferred feature.
- [x] OT-P0-014 | Push subscription lifecycle | The system shall store one push subscription per browser and per recipient, shall remove a subscription when the push service reports it gone, and shall renew one when the browser reports it changed. A subscription that expired without replacement shall make that channel report unavailable rather than appear healthy, because a silently dead subscription is the failure mode that makes a notification spine untrustworthy.
- [ ] OT-P0-015 | Stable public origin | The scenario shall hold a core-tier public hostname so its origin never expires. A push subscription is bound to an origin and dies with it, so an expiring hostname would silently unsubscribe every device. The system shall report push unavailable with a stated reason when its origin is unreachable.

### 🟠 P1 – Should have post-launch

> Numbering is identity, not order. The acknowledgement trio — OT-P1-009 through OT-P1-011 — leads this tier and ships ahead of digest collapsing, scheduled delivery, and analytics.

- [x] OT-P1-001 | Cross-node delivery relay | When this host cannot serve a channel, the system should ask its own instance on the machine that can, through vrooli-bridge durable dispatch. The channel vocabulary should stay inside this scenario, so vrooli-bridge carries the call without learning what a channel is.
- [x] OT-P1-002 | iMessage delivery from a Mac node | When a recipient channel is iMessage, the system should deliver it through the paired Mac node and should record the same receipt shape as a local delivery.
- [x] OT-P1-003 | Event-driven ingress | The system should accept vrooli-events webhook deliveries and should raise notifications from subscription pattern matches.
- [x] OT-P1-004 | Email channel | The system should deliver notifications over SMTP using operator-supplied credentials.
- [x] OT-P1-005 | Digest collapsing | When several low-urgency notifications arrive inside a digest window, the system should combine them into one scheduled summary delivery.
- [x] OT-P1-006 | Scheduled delivery | The system should deliver a notification at a caller-specified future time and should survive a restart between acceptance and delivery.
- [x] OT-P1-007 | Delivery analytics | The system should report delivery counts, failure rates, and per-channel latency over a selectable window.
- [x] OT-P1-008 | Owner-controlled notification content | The system should keep notification content under owner control end to end. Web Push already satisfies this by encrypting the payload under keys held only by this scenario and the recipient's browser. Any future channel that cannot meet that bar should be treated as a lower sensitivity tier rather than adopted silently.
- [x] OT-P1-009 | Acknowledgement and response | The system should accept an acknowledgement or a chosen action from a delivered notification and should return it to the requesting scenario, so a notification can carry a decision back rather than only carrying information out.
- [x] OT-P1-010 | Blocking ask primitive | The system should expose a request that delivers a question and blocks until the recipient answers or a caller-supplied deadline expires, and should return the answer or a stated timeout to the caller. This is the calling convention an agent needs; OT-P1-009 is the data path underneath it.
- [x] OT-P1-011 | Escalation chains | When a notification marked critical is not acknowledged within its timeout, the system should escalate it to the next channel in the recipient's chain and should record each escalation step. Ships with OT-P1-009 and OT-P1-010, because an unanswered approval is the case that needs escalating.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-002 | SMS channel | The system may deliver notifications over SMS using credentials supplied by the twilio resource.
- [ ] OT-P2-003 | Multi-user routing | When the install runs a shared or hosted trust posture, the system may route notifications per authenticated identity.

## 🧱 Tech Direction Snapshot

**Preferred stack** — The `react-vite` template shape without deviation: a Go API on Connect-RPC with proto-defined contracts, a React and TypeScript UI on the `vrooli-default` design kit, and a Go CLI on `cli-core`. The webhook receiver is the one sanctioned REST exception.

**Data** — SQLite through the `api-core` storage seam. No PostgreSQL, no Redis, no Docker, and no resource dependency of any kind. The queue, the retry schedule, and the rate-limit counters are in-process and SQLite-backed. This is a capability decision, not a shortcut: PostgreSQL and Redis are still Docker-backed and are recorded `unsupported` on macOS and Windows, so depending on either would make the scenario unable to run on the Mac node the relay lane exists to reach. Retention is set before the schema is written, not after: retry multiplies delivery-attempt rows, and an unbounded attempt table is a known failure mode elsewhere in this repo.

**Ingress contract** — The sensitivity label and the idempotency key are required fields on the request from the first version of the proto. Both are cheap to specify now and expensive to retrofit, because adding a required field later is a breaking change across the API, the CLI, and the UI at once. The label is set by the caller, since only the caller knows whether the body is safe on a locked screen.

**Channel integration** — One routing core, many thin channel adapters. The first and primary channel is **Web Push** from this scenario's own installed progressive web app: the service worker already ships in the scaffold, the template already registers it, and 39 scenarios in the fleet already carry one, so the adapter is a `push` listener rather than new infrastructure. Payloads are encrypted under RFC 8291 with keys no push service holds, which answers the sensitivity requirement more completely than keeping bodies on a server the phone must then reach.

A self-hosted relay server was designed and rejected: it would still have routed its wake-up through a third party to Apple, so it added a server without removing a dependency, and it required the phone to reach that server at delivery time or degrade to a contentless placeholder. That design is preserved as the `ntfy` resource blueprint at `status: candidate`, with the trade-offs that would justify revisiting it. Later channels that need vendor credentials — SMS through `twilio` — arrive as optional `cloud-api` resources, following that resource's precedent where the entire CLI surface is `provider-check` and the send call belongs here.

**Cross-node delivery** — `vrooli-bridge` durable dispatch, not a synchronous relay call, so a delivery survives a node that is briefly offline. This scenario asks its own remote instance through the cataloged `notification-hub notifications relay` verb; `notification-hub channels status` remains the local/remote readiness probe. The channel vocabulary therefore never enters vrooli-bridge, which stays the reach plane — which machines exist, whether they are online, and whether this caller may send them anything. `device-control` is the worked precedent for a scenario dispatching its own verbs to a remote node.

Because readiness is asked rather than pushed, freshness is this scenario's responsibility: cache the last channel status per machine with a stated lifetime, refresh it on a schedule, and re-check after a failed delivery before marking anything unroutable.

**Ingress order** — Direct Connect-RPC and CLI requests first, because they are self-contained. Event-driven ingress second, because it depends on the optional `vrooli-events` subscription fan-out and startup reconciliation.

**Non-goals** — Multi-tenant profiles, scenario-issued API keys, billing and usage quotas, a provider marketplace, marketing campaign sequencing, and a template authoring UI. Each belongs to a notification product sold to third parties. This scenario serves one owner and their fleet.

## 🤝 Dependencies & Launch Plan

**Resources** — **None, at any tier that ships first.** Web Push needs no resource, so `dependencies.resources` stays empty and OT-P0-009 is literally true rather than aspirational. This also keeps the scenario runnable on a macOS fleet node, which `postgres` and `redis` would prevent: both remain Docker-backed and are recorded `unsupported` on macOS and Windows in the platform matrix. The existing `twilio` resource is adopted only when SMS activates at P2, as an optional dependency.

**Scenario dependencies**

- `scenario-authenticator` — required. Supplies owner identity; tokens are verified locally against its JWKS through `api-core/owneridentity`, with no per-request callback.
- `tunnel-manager` — required. Supplies the stable public origin a push subscription is bound to (OT-P0-015). This scenario is a core-seed member, so its hostname is always-on and never auto-expired; a leased hostname would silently unsubscribe every device on expiry. Absent, previously registered subscriptions keep working until the browser next needs the origin, and no new device can subscribe.
- `vrooli-bridge` — optional. Supplies the node registry and durable dispatch for cross-node delivery. Absent, the hub delivers only what this host can reach and marks other channels unavailable rather than failing.
- `vrooli-events` — optional. Supplies event ingress. Absent, direct requests still work.

**Risks**

- The `vrooli-events` subscription fan-out engine does not exist. Reading that scenario's source rather than its documentation: on ingest it publishes to the SSE broker only, its subscription store is referenced by CRUD handlers and two manual endpoints, its webhook deliverer has no retry or signing, and its `subscription_health` table has no writer at all. Its own `PROBLEMS.md` reports persistent subscriptions as complete. Event ingress therefore depends on work in a scenario this one does not own, which is why it is P1 and not P0.
- iMessage automation on macOS is brittle. It needs Full Disk Access, an unlocked session, and a signed-in Messages account, and Apple has broken the automation surface before. Treat a working iMessage path as a bonus, never as a gate.
- A push subscription is bound to an origin and expires silently. An expired subscription, a hostname that lapsed, or a progressive web app removed from the Home Screen all look identical from here: nothing arrives and nothing errors. OT-P0-014 and OT-P0-015 exist because of this, and the delivery timeline must make a dead channel visible rather than quiet.
- Notification fatigue is the real failure mode. A spine that fires too often gets muted, after which every downstream promise is void. Quiet hours and duplicate suppression are P0 because of this, not as polish.

**Launch sequencing**

1. One real delivery to the owner's iPhone through Web Push, before any abstraction is built.
2. The routing core — recipients, devices, preferences, quiet hours, duplicate suppression, retry.
3. Acknowledgement, the blocking ask primitive, and escalation, as one slice. This is the differentiating capability and it needs no second machine, so it comes before the fleet work.
4. Cross-node delivery by asking this scenario's own remote instance through bridge dispatch, with the Mac node as its first consumer. This depends on a vrooli-bridge refactor that derives its dispatch vocabulary from the shared scope catalog instead of a hand-maintained verb list, so no bridge change is needed per scenario.
5. iMessage from that Mac node, as a best-effort secondary that does not gate the release.
6. Event ingress, once the upstream fan-out engine exists.

## 🎨 UX & Branding

**Design kit** — `vrooli-default`, the Vrooli Operational Console. The scenario is an instrument panel, not a marketing surface.

**Character** — Dense, status-first, and scannable. The primary screen is a delivery timeline that answers one question at a glance: did the thing I was promised would reach me actually reach me? Summary before detail; the newest and the failed rise to the top.

**State encoding** — Delivery state carries form as well as colour, so `pending`, `held`, `delivered`, and `failed` read without relying on hue. Semantic status colour stays separate from the design kit's accent.

**Accessibility** — WCAG 2.2 AA. Every control is keyboard reachable with a visible focus state, the timeline is navigable without a pointer, and no state is signalled by colour alone. Motion respects `prefers-reduced-motion`.

**Voice** — Factual and specific, in the reader's terms rather than the system's. A person manages *devices and quiet hours*, not webhook subscriptions and channel adapters. A notification body says what happened and links back to the console for detail. It never carries a secret, a token, or a full record, so the same sentence is safe on a locked screen and in a shared room.

**Failure text** — An undelivered notification states what failed and what to do about it. "Push rejected the topic — re-register this device" beats "delivery error".
