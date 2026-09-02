# Integrations — Notification Hub

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, every persisting domain | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `scenario-authenticator` | scenario | yes | Every authenticated surface | JWKS verified locally via `api-core/owneridentity` | Authenticated surfaces fail closed; accepted notifications keep draining. |
| `vrooli-bridge` | scenario | no | Cross-node delivery | Durable dispatch of this scenario's cataloged `notifications relay` CLI verb to a remote machine | Host-bound channels are marked unroutable with a reason. |
| `vrooli-events` | scenario | no | Event-driven ingress | Inbound subscription webhook to this scenario's receiver | Event ingress unavailable; direct requests unaffected. |
| `tunnel-manager` | scenario | yes | Push subscription origin | Core-tier public hostname, never auto-expired | No new device can subscribe; existing subscriptions keep working while the origin resolves. |
| Browser push service | third-party | yes | Web Push adapter | RFC 8030 Web Push with VAPID; payload encrypted under RFC 8291 | Deliveries fail retryably, then terminally with a stated reason. `410 Gone` deletes the subscription. |
| Linux desktop session | host capability | optional | `linux_notification` sender | `notify-send` through the freedesktop notification interface; requires a graphical display and a reachable D-Bus session bus | The channel reports unavailable with the missing display, session bus, or sender binary. It is never advertised as a macOS-only channel. |
| macOS desktop session | host capability | optional | `macos_notification` / `imessage` sender | `osascript` through Notification Center or Messages | The channel is selected only on macOS and reports host-specific unavailability before dispatch. |

## Vrooli Resources

The core carries **no resource dependency at all**, and this is a
capability decision rather than a default left untouched. The reason is
macOS evidence, not Docker: both PostgreSQL and Redis now declare
`driver: managed-service` and neither needs a container runtime to start,
although both are still acquired as OCI images. Neither is `supported` on
macOS — `path:docs/reference/platform-support.md` records `postgres` as
`build-verified` with no macOS hardware run performed, and its two tables
disagree about `redis` there (generated matrix: `build-verified`;
narrative capability table: `unsupported`). Depending on either would put
this scenario on unproven footing on the Mac fleet node that the relay lane
exists to reach. See OT-P0-009.

The primary channel needs no resource at all. Web Push is served from
this scenario's own installed progressive web app: the service worker
already ships in the scaffold and the template already registers it, so
the adapter is a `push` listener rather than new infrastructure. Later
channels that need vendor credentials arrive through optional `cloud-api`
resources, which are credential-and-reachability only and carry no local
process.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `postgres` | rejected | `managed-service`, not Docker, but only `build-verified` on macOS with no hardware run performed. Would put cross-node delivery on unproven footing. | A macOS hardware run, plus a workload SQLite cannot carry. |
| `redis` | rejected | Same portability ceiling. Queue, retry schedule, and rate-limit counters are in-process and SQLite-backed instead. | Revisit if this scenario ever runs replicated, which is not planned. |
| `ntfy` | rejected, blueprinted | Evaluated as the push provider and rejected. A self-hosted relay still routes its wake-up through a third party to Apple, so it adds a server without removing a dependency, and it requires the phone to reach that server at delivery time or degrade to a contentless placeholder. Web Push needs no server and encrypts the body end to end. | Preserved at `.vrooli/resources/blueprints/ntfy.json`, `status: candidate`. Revisit if a `curl`-able endpoint for non-Vrooli scripts is needed, or if fully de-googled Android delivery becomes a goal. |
| `twilio` | available, unused | Existing `cloud-api` resource with `account-sid` and `auth-token` descriptors. | Adopt when OT-P2-002 (SMS) activates. |

## Scenario Dependencies

Declared in `.vrooli/service.json` under `dependencies.scenarios`. All
four are `runtime_only`: this scenario imports no Go package from any of
them, so no import-graph edge should be expected as evidence.

| Scenario | Status | Startup Policy | Contract |
|---|---|---|---|
| `scenario-authenticator` | required | `try_start` | Identity for every authenticated surface. Tokens are RS256 and verified locally against the published JWKS through `api-core/owneridentity`; there is no per-request authorization callback. This scenario issues no API key, stores no password, and owns no profile table (OT-P0-007). |
| `tunnel-manager` | required | `try_start` | Stable public origin for the installed progressive web app. A Web Push subscription is bound to an origin and dies with it, so this scenario is a core-seed member and its hostname is never auto-expired (OT-P0-015). |
| `vrooli-bridge` | optional | `ignore` | The **reach plane** only: which machines exist, whether they are online, and whether this caller may send them anything. Cross-node delivery dispatches this scenario's cataloged `notification-hub notifications relay` verb to a remote machine; the bridge vocabulary is derived from the shared scope catalog. Durable dispatch is chosen over a synchronous relay call so a delivery survives a node that is briefly offline (OT-P1-001). |
| `vrooli-events` | optional | `ignore` | Inbound webhook delivery from durable subscriptions to this scenario's REST receiver (OT-P1-003). |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Browser push service (Apple, Google, Mozilla) | required, P0 | The only way to wake an installed web app on a phone. Reached through the standard Web Push protocol with VAPID, so no Apple Developer account, no Firebase project, and no Mac to build and sign an app. | RFC 8030 with VAPID authentication. The payload is encrypted under RFC 8291 with keys held only by this scenario and the browser, so the push service cannot read a body. `410 Gone` means the subscription is dead and must be deleted. |
| SMTP relay | optional, P1 | Email channel. | Operator-supplied SMTP host/from plus the manifest-declared password credential (OT-P1-004). |
| Twilio | planned, P2 | SMS channel. | Credentials from the `twilio` resource (OT-P2-002). |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `scenario-authenticator` | JWKS fetch fails or signature does not verify | Authenticated surfaces reject with `unauthenticated`. The delivery worker keeps draining already-accepted work. | identity middleware tests |
| `vrooli-bridge` | Dispatch rejected, machine offline, or the remote instance reports a non-ready disposition | Delivery is marked **unroutable** carrying the remote instance's own stated reason, and surfaces in the timeline. It is never left silently `pending` (OT-P0-011). | routing tests, remote-dispatch tests |
| `vrooli-events` | Webhook receiver never called | Event ingress is silently absent, which is acceptable: direct ingress is the P0 path. | receiver contract tests |
| `tunnel-manager` | Origin not resolvable | Push reports unavailable with a stated reason. No new subscription can be created. | channel status tests |
| Browser push service | Non-2xx | Retry with backoff, then terminal failure with an actionable reason (OT-P0-008). `410 Gone` deletes the subscription instead of retrying (OT-P0-014). | Web Push adapter tests |
| Linux desktop session | No display, D-Bus session, or `notify-send` | `linux_notification` is reported unavailable with a reason before dispatch. | Linux transport availability tests |
| macOS desktop session | Non-macOS host or missing `osascript` | macOS channels are not selected on Linux and unavailable reasons are exposed before dispatch. | Desktop sender selection tests |

## Event Subscription Reconciliation

When `VROOLI_EVENTS_API_BASE`, `VROOLI_NOTIFICATION_EVENTS_WEBHOOK_URL`,
and `VROOLI_NOTIFICATION_EVENTS_PATTERN` are configured, startup performs
an idempotent subscription reconciliation against `vrooli-events`. The
events scenario owns matching, durable queueing, retries, signing, and
subscription health; this scenario owns only the receiver and translation
into a durable notification. If reconciliation or the upstream scenario is
unavailable, direct Connect-RPC and CLI ingress remain unaffected.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
