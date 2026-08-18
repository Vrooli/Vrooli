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
| 2026-08-17 | Regenerate the scenario from `react-vite` rather than repair the 2025 implementation. | The previous scenario carried none of the eight current structural markers — no proto contracts, no `bas/`, no `experience/`, a static HTML UI served by Express, an empty requirements registry — and 14 of its 20 business endpoints returned `[]` or `"Not implemented"`. Its `.vrooli/testing.json` had been swept to declare a `react-vite` policy profile it could not satisfy. | Roughly 5,000 lines of API, UI, and test scaffolding discarded. `tunnel-manager` set the precedent by regenerating rather than migrating. | Not expected. The old tree is gone. |
| 2026-08-17 | Delete multi-tenancy. No profiles, no scenario-issued API keys, no billing quotas, no provider marketplace. | The 2025 PRD specified a multi-tenant notification SaaS with per-profile API keys and GDPR unsubscribe flows, aimed at customers that do not exist and integrating with scenarios that do not exist. | Removes about half the surface area and most of the remaining complexity. Identity moves to `scenario-authenticator`. | The owner decides to sell notification capability to third parties. That is a different product and should be a different scenario. |
| 2026-08-17 | Own no identity. Verify `scenario-authenticator` tokens locally and key recipients by the token subject. | Multi-user support should follow the project trust posture (`personal`, `shared`, `hosted` in `.vrooli/operator-state.json`), which is currently `personal`. | Multi-user routing (OT-P2-003) becomes a data consequence rather than a rewrite: the same recipient table holds one subject or several. No password or key is ever stored here. | Never. Storing credentials here would be a regression. |
| 2026-08-17 | Declare zero resource dependencies for the core; persist to SQLite. | `resource-postgres` and `resource-redis` acquire as OCI images and are recorded `unsupported` on macOS and Windows. The scenario must run on a macOS fleet node to serve Apple-only channels. | The relay lane becomes buildable. `vrooli scenario start` needs no Docker. The queue, retry schedule, and rate-limit counters are in-process. | Postgres completes a portable acquisition path AND measured volume outgrows SQLite. Both conditions, not either. |
| 2026-08-17 | Delivery providers stay resources, but only as thin `cloud-api` resources. | Credentials and reachability health belong in a resource manifest; the send call does not. `resource-twilio` establishes the pattern with a single `provider-check` command. | A provider resource never grows a send API. Channel adapters live in `api/internal/channels/`. | A provider is self-hosted, at which point it is promoted to `managed-service` with a portable acquisition target. |
| 2026-08-17 | Reach the iPhone through ntfy rather than APNs. | Sending push to an iPhone does not require a Mac or an Apple Developer account; an App Store relay client receives push from a plain HTTPS POST originating on this Linux host. A first-party APNs app needs a paid developer account, a Mac to sign, and `scenario-to-ios`. | First real delivery is reachable in the P0 slice. The APNs path is not blocked, just not owned here. | OT-P2-001 (first-party web push) or a decision to ship a real iOS app. |
| 2026-08-17 | Use `vrooli-bridge` durable dispatch for cross-node delivery, not the synchronous relay call. | The bridge exposes both. `RelayService.Call` is bounded and synchronous; `dispatch` creates a durable run with an audit record and an allowlist check. | A delivery survives a node that is briefly offline. `deliveries.node_id` exists in the P0 schema so P1 needs no migration. | Measurement shows dispatch latency makes time-critical channels unusable. |
| 2026-08-17 | Sequence event ingress after direct requests, and file the fan-out gap upstream. | `vrooli-events` documents this scenario as its primary webhook consumer, but on ingest it publishes only to its SSE broker. `WebhookDeliverer.Deliver` is reachable solely from a manual trigger endpoint — there is no matcher, retry queue, or delivery engine. | Event ingress is OT-P1-003. The receiver is built and tested against a synthetic caller so it is ready when upstream lands. The engine gap is filed against `vrooli-events`, not absorbed here. | The upstream fan-out engine ships. |
| 2026-08-17 | Treat iMessage as a best-effort secondary that does not gate the release. | The owner has a paired Mac mini node (`minimouse`) and wants iMessage delivery. macOS automation for Messages needs Full Disk Access, an unlocked session, and a signed-in account, and Apple has broken the surface before. | OT-P1-002 is attempted properly and validated where possible, but no release gate depends on it. | Apple ships a supported automation surface, or the path proves stable across several macOS updates. |
| 2026-08-17 | Quiet hours and duplicate suppression are P0, not polish. | A notification spine that fires at 3am or repeats forty times gets muted, after which every downstream promise it makes is void. | Both ship in the first release, and both are pure functions with table-driven tests. | Never. These are what distinguish the scenario from a `curl` wrapper. |
| 2026-08-17 | Every notification carries a sensitivity label, and bodies link rather than contain. | A public push topic exposes body content to anyone holding the topic name, and a notification body is visible on a locked screen. | `notifications.sensitivity` is NOT NULL with no default, so a caller must decide once. The house convention is that a body says what happened and links back to the console for detail. | Never relax the NOT NULL. A default would make the safe case invisible. |
| 2026-08-17 | Remove the unused `fileRootPath` template seam from `api/main.go`. | The generated scaffold ships it as a mandatory file-store seam, but no v1 domain stores files, and `golangci-lint` failed the generation hook on it. `device-control` and `money-ledger` resolved it the same way. | The generation hook passes. Re-add the helper when a domain genuinely stores blobs. | A domain needs the BlobStore. |
| 2026-08-17 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs carry maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-08-17 | Multi-tenant profiles with per-profile bcrypt API keys, provider registries, and billing quotas (2025 PRD). | Single-owner spine with identity delegated to `scenario-authenticator`. | The original contract described a SendGrid-shaped product. It was never completed and its incomplete parts were exactly the parts a single owner does not need. |
| 2026-08-17 | PostgreSQL and Redis as required resources (2025 `service.json`). | Embedded SQLite, no resources. | Both are OCI-acquired and unsupported on macOS/Windows; the dependency would have prevented running on the Mac node the relay lane targets. |
| 2026-08-17 | n8n as the multi-channel delivery router (2025 PRD). | Channel adapters behind an in-process routing core. | n8n was optional, never wired, and would add an external workflow engine to a decision that fits in a pure function. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts these decisions produced
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
