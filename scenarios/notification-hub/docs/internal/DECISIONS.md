# Decisions — Notification Hub

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-17 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-17 | Regenerate from the template rather than repair the previous scenario in place. | The previous notification-hub predated the `react-vite` template and had none of its eight structural markers. Six of twenty business endpoints were implemented; every send path terminated in a log line; the requirements registry was empty. Its `.vrooli/testing.json` had been swept to declare a `react-vite` policy profile it could not satisfy. | ~5,000 lines of API, UI, and test scaffold discarded. Worth porting: the twelve-table schema design and the channel/delivery-state taxonomy. `tunnel-manager` set the precedent by regenerating from `react-vite` 1.1 rather than migrating. | None. Relitigating this wastes the migration. |
| 2026-08-17 | Declare **no resource dependencies**. SQLite through the `api-core` storage seam; queue, retry schedule, and rate-limit counters in-process. | The old scenario declared `postgres` and `redis` as `required: true`. `path:docs/reference/platform-support.md` records both as Docker-backed and `unsupported` on macOS and Windows; their manifests confirm `acquisition.kind: "oci-image"`. | The same build runs on a Linux host and on a macOS fleet node, which is what makes cross-node relay possible at all. No Docker in the start path. Rejected alternative: keep Redis for the queue — it would have made the scenario Linux-only and silently vetoed the relay architecture. | Revisit only if `postgres` or `redis` earns a portable managed-service acquisition path like the one `searxng` now uses. Adding either back forfeits macOS. |
| 2026-08-17 | Own no identity. Verify `scenario-authenticator` tokens locally via `api-core/owneridentity`. | The old scenario carried profiles, per-profile API keys, bcrypt verification, and an `/admin` surface — the scaffolding of a notification product sold to third parties. | No password, no profile table, no scenario-issued API key here. Roughly half the old surface area disappears. Recipients key off the authenticator's `user_id`. | Revisit if a consumer outside the Vrooli trust boundary ever needs to send through this hub. That is a different product; prefer a new scenario. |
| 2026-08-17 | Treat multi-user support as a consequence of `trust_posture`, not as a feature to build. | `.vrooli/operator-state.json` carries `trust_posture` (`personal` \| `shared` \| `hosted`); this install is `personal`. `packages/api-core/trustposture` already exists. | In `personal` there is exactly one identity and the UI shows no picker. In `shared`/`hosted` the same tables hold several and routing is per-identity (OT-P2-003). Multi-user is earned by using the identity seam rather than by writing tenancy code. | Revisit when the operator moves this install to `shared` or `hosted`. |
| 2026-08-17 | Use Bridge **durable dispatch** for cross-node delivery, not `RelayService.Call`. | Bridge exposes both: a bounded synchronous relay and a queued, allowlist-checked, audited dispatch that creates a durable run. | A delivery survives a node that is briefly offline, which a synchronous call cannot. Costs a slower path and an extra state to model. | Revisit if a channel ever needs a synchronous acknowledgement from the remote node inside the request. |
| 2026-08-17 | Ship one real channel end to end before building any abstraction, and make it push to the owner's iPhone. | Everything the previous scenario ever did was simulated. | OT-P0-001 is a delivery requirement, not an architecture requirement. Push was chosen over email because it needs neither an Apple Developer account nor a Mac to build and sign an app — correcting the assumption that reaching an iPhone requires a Mac. A Mac is needed only to sign a first-party APNs app and for genuinely host-bound Apple channels. | None for the sequencing. The provider choice has its own row. |
| 2026-08-17 | Start with a hosted push provider as a `cloud-api` resource; promote to `managed-service` only when self-hosted. | `twilio` is the working precedent: a `cloud-api` resource whose entire CLI surface is `provider-check`. The resource owns credentials, endpoint, and reachability health; the scenario owns the send call. The archived `pushover` blueprint warns that "shallow wrappers often duplicate simple HTTP calls without much value." | Keeps provider resources thin. **The `ntfy` resource does not exist yet** — no resource, no blueprint. OT-P0-001 has this as an unstated prerequisite until it is created. | Promote to `managed-service` when OT-P1-008 activates, using the `composed`/`url` acquisition path that made `searxng` portable. |
| 2026-08-17 | Sequence direct ingress before event ingress. | `vrooli-events` stores subscriptions but never fans them out: ingest publishes to the SSE broker only, and `Deliver` is reachable solely from a manual trigger endpoint. No matcher, no retry queue, no engine. | Direct Connect-RPC and CLI ingress is P0 because it is self-contained. Event ingress is P1 because it depends on work in a scenario this one does not own. | Revisit when the `vrooli-events` subscription fan-out engine exists. File the gap against that scenario, not this one. |
| 2026-08-17 | Promote acknowledgement and escalation from P2 to P1, and add a blocking `ask` primitive alongside them. | Market research found three adjacent markets: metered multi-tenant notification SaaS (Novu, Knock, Courier), free self-hosted push pipes (ntfy, Gotify, Pushover, Apprise), and agent-to-human attention, which has no established player. The first is the charter this scenario rejected; the second is transport we ride on. | Return traffic is what separates this from both. A pipe has no routing layer; a hosted platform has no fleet to route across. Promoting the trio makes the scenario the human-in-the-loop gate for the fleet rather than an outbound pipe, and it needs no second machine, so it sequences ahead of the bridge work. Costs: acknowledgement state, a callback path back to the requesting scenario, and a long-lived blocking call the API must hold open. Renumbered `OT-P2-004`/`OT-P2-005` to `OT-P1-009`/`OT-P1-011`; safe because neither was implemented or referenced by a test tag. | Revisit if acknowledgement ships and agents do not use it. Delivery without acknowledgement means the spine is being muted, and the oversight thesis fails before any pricing question matters. |
| 2026-08-17 | Never meter notifications. | Metering is how every SaaS competitor prices, and the obvious way to monetize this. | It penalizes exactly the high-frequency agent traffic the scenario exists to serve, and it contradicts OT-P0-009's promise that the thing runs on the owner's own hardware at zero marginal cost. Commercial role is bundle inclusion in the `business` base SKU, with an agent-oversight add-on gated on OT-P1-009 through OT-P1-011. | Revisit only for a hosted relay serving owners with no always-on machine — the one place a metered tier would be coherent. |
| 2026-08-17 | Sensitivity labelling is default-deny, and the label is carried on the ingress contract from day one. | A public push topic exposes its body to anyone holding the topic name. A label added after the proto exists would be a required-field retrofit across API, CLI, and UI. | Every notification carries a label at acceptance. A channel is **not** approved for a label unless it declares approval; the unapproved case sends a content-free pointer back to the console rather than the body. Costs a required field in v1 of the proto. | Revisit only to add labels, never to make the default permissive. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
