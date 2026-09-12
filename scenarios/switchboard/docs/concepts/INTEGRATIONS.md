# Integrations — Switchboard

## Purpose Of This Document

This document is the canonical dependency contract for resources, other
scenarios, and third-party services used by this scenario.

Use it to answer:

- What does this scenario depend on, and which dependencies are required?
- Which domain uses each dependency, and under what contract?
- What happens when a dependency is unavailable?
- Where is each dependency declared, so that this document and
  `.vrooli/service.json` can never disagree?

Every dependency named here is declared in `.vrooli/service.json` under
`dependencies.scenarios`. Every dependency declared there is named here. A
disagreement between the two is a defect, not a documentation lag.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | every persisting domain | resolved by `api-core/storage` from the scenario identity | API reports unhealthy; no inbound message is acknowledged, so transports redeliver rather than lose |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Must be started through lifecycle commands; never by running a binary directly |
| `scenario-authenticator` | scenario | yes | every authenticated surface | RS256 tokens verified locally against the published JWKS via `api-core/owneridentity` | Authenticated surfaces fail closed. Adapters keep receiving and recording; no turn dispatches for an unattributable sender |
| `prompt-manager` | scenario | yes | `agents`, `trust` | Agent descriptors and capability grants read by reference over Connect | Existing bindings resolve from the cached identifier; no new binding can be created, no descriptor detail renders |
| `agent-manager` | scenario | yes | `turns` | `StartRun` over Connect with a resolved scope; attenuation and approvals enforced there | Turns cannot execute. Messages are still de-duplicated and recorded; the sender gets a stated reason, never silence |
| `tunnel-manager` | scenario | no (yes once webhook ingress activates) | `channels` | Public origin request for descriptors declaring `requires: public_origin` | Webhook-ingress channels report unavailable with the missing origin named. Poll and in-app channels unaffected |
| `vrooli-bridge` | scenario | no | `channels` | Durable dispatch of this scenario's own cataloged CLI verb to a named machine | Host-bound channels marked unroutable with a reason. Host-local channels unaffected |
| `notification-hub` | scenario | no | `channels` | Inbound caller into this scenario's adapter surface for shared channels | Notifications keep using notification-hub's own senders. Nothing here degrades |
| `program-runtime` | scenario | no | `turns` (via `agent-manager`) | Reached only when a capability grant names a program binding | The granted step refuses with a stated reason; the rest of the turn proceeds |
| `audio-tools` | scenario | no | `channels` (call mode) | WebSocket capture and Connect transcription/synthesis | Voice and call-mode channels report unavailable. Text and media channels unaffected |
| `persona` | scenario | no | `channels` (provisioning) | One-time-code retrieval and typed operator handoff | Channels needing verification stay in a stated pending state. Provisioned channels unaffected |
| Telegram Bot API | third-party | no | `channels` | HTTPS long-poll or webhook, bot token from the credential authority | The channel reports unavailable with the transport error; other channels keep running |

## Vrooli Resources

**This scenario declares no resource dependency at P0, and that is a capability
decision rather than a default left untouched.**

Both are rejected on macOS support, not on a container runtime. Per
`path:docs/reference/platform-support.md`, `redis` is recorded
not `supported` on macOS, and `postgres` is only `build-verified` there —
staged from checksum-pinned upstream archives with **no macOS hardware run
performed**. Depending on either would put this scenario on unproven or
unsupported footing on the Mac fleet node that the iMessage lane exists to
reach, which is the single most valuable channel in the product.

Note that neither resource is Docker-backed in the start path any more: both
declare `driver: managed-service`, and the Linux `postgres` path stages a
digest-pinned filesystem tree with no container runtime. Both are still
*acquired* as OCI images (`managed_service.acquisition.kind: "oci-image"`) —
the image is a distribution format here, not something that gets run. The
rejection rests on macOS support and evidence, not on Docker. Queue state, retry schedules, budget counters, and
rate-limit windows are therefore in-process and SQLite-backed.

The first two channels — the in-app adapter and Telegram — need no resource at
all. Later channels arrive as optional `cloud-api` resources, which are
credential-and-reachability only and carry no local process.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `postgres` | rejected | `managed-service`, not Docker. `build-verified` on macOS with no hardware run performed, so unproven on the exact platform the Mac lane needs. | A macOS hardware run, plus a reason SQLite no longer suffices |
| `redis` | rejected | Recorded `unsupported` on macOS, which rules it out for the Mac lane by itself. Budgets and rate-limit counters are in-process and SQLite-backed. | macOS support landing, and this scenario ever running replicated — not planned |
| `twilio` | available, unused | Existing `cloud-api` resource with typed Go credential and provider diagnostics. | Adopt when OT-P2-001 (SMS) activates |
| `whisper`, `kyutai-stt`, `kokoro` | available, indirect | Consumed through `audio-tools`, never mounted directly. Declaring them here would create a second owner for speech. | Never directly; `audio-tools` owns them |

## Scenario Dependencies

Declared in `.vrooli/service.json` under `dependencies.scenarios`. **All are
`runtime_only`**: this scenario imports no Go package from any of them, so no
import-graph edge should be expected as evidence. This is deliberate — a shared
Go package's new import breaks the build of every scenario module that replaces
it, and this scenario adds no such coupling.

| Scenario | Status | Startup Policy | Contract |
|---|---|---|---|
| `scenario-authenticator` | required | `try_start` | Owner identity for every authenticated surface. This scenario issues no API key, stores no password, and owns no profile table. |
| `prompt-manager` | required | `try_start` | The source of truth for agent profiles and capability grants. Held by reference; a copy would fork on first edit (OT-P0-008). |
| `agent-manager` | required | `try_start` | Run execution and one-way scope attenuation. This scenario names scopes; it never defines a second policy vocabulary (OT-P0-012). |
| `tunnel-manager` | optional now, required on webhook ingress | `ignore` | Stable public origin. Only descriptors declaring `requires: public_origin` consult it (OT-P1-012). |
| `vrooli-bridge` | optional | `ignore` | The reach plane only. Cross-node work dispatches this scenario's own verb; the bridge never learns what a channel is (OT-P1-010). |
| `notification-hub` | optional | `ignore` | A second caller into the same adapter registry so a channel is never implemented twice (OT-P1-015). |
| `program-runtime` | optional | `ignore` | A grantable capability, reached only through an `agent-manager` run. |
| `audio-tools` | optional | `ignore` | Speech for call mode, consumed only (OT-P2-002). |
| `persona` | optional | `ignore` | Declared identity and the operator handoff for verification no machine may complete (OT-P2-001). |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Telegram Bot API | planned, P0 | The correct first external channel: free, official, bidirectional, native images, files and voice notes, no account provisioning, no public origin required, and it works from a Linux host. | HTTPS long-poll or webhook. Bot token held by the credential authority and referenced, never stored here. |
| Slack API | planned, P1 | Maps onto the same adapter contract as Telegram almost field for field. | OAuth app install, socket-mode ingress, Web API egress. |
| Apple Messages | planned, P1 | Reached through the owner's own Mac, not a vendor. There is no supported inbound interface, so ingress reads the local message store and is fragile across macOS releases by construction. | `osascript` egress and local message-store ingress, on a Mac fleet node via `vrooli-bridge`. Must degrade to a stated unavailable reason, never a silent failure. |
| Twilio | planned, P2 | SMS and voice on a real number. | Credentials from the existing `twilio` resource. A2P 10DLC registration is an operator handoff through `persona`, not an automatable step. |
| Hosted iMessage relay | rejected as default, optional at P2 | Routing private conversations through a vendor forfeits the custody promise that justifies this scenario. | If ever offered, only as an explicitly labelled fallback that states its trade-off at the point of purchase. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` reports unhealthy. Inbound messages are **not** acknowledged, so the transport redelivers rather than dropping. | health handler tests |
| `scenario-authenticator` | JWKS fetch fails or signature does not verify | Authenticated surfaces reject as unauthenticated. Ingress and recording continue. | identity middleware tests |
| `prompt-manager` | descriptor lookup fails | Existing bindings resolve from the cached agent identifier. Binding creation refuses with a stated reason. | binding resolution tests |
| `agent-manager` | `StartRun` unavailable | The sender receives a stated unavailable reason on the same thread. The turn is not silently dropped. | turn dispatch tests |
| `vrooli-bridge` | node offline or dispatch rejected | The host-bound channel is marked unroutable, with the machine and the reason both named. | channel availability tests |
| `tunnel-manager` | no origin available | Webhook-ingress channels report unavailable; the funnel renders a prompt to satisfy the requirement rather than a selectable dead option. | availability computation tests |
| Adapter transport | connect or send failure | Retry with the backoff the descriptor declares, then terminal with a stated reason recorded on the thread. | conformance phase, per adapter |

## Cross-References

- `.vrooli/service.json` — the machine-readable declaration this document mirrors
- `docs/concepts/DOMAINS.md` — the domains that consume each dependency
- `docs/concepts/ARCHITECTURE.md` — where these dependencies sit in the shape
- `docs/internal/DECISIONS.md` — why each dependency was chosen or rejected
- `docs/internal/SECURITY.md` — the trust boundary each dependency sits on
- `PRD.md` — the operational targets referenced above
