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
| SQLite | embedded storage | yes | API, every persisting domain | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `scenario-authenticator` | scenario | yes | Every authenticated surface | JWKS verified locally via `api-core/owneridentity` | Authenticated surfaces fail closed; accepted notifications keep draining. |
| `vrooli-bridge` | scenario | no | Cross-node delivery relay | Durable dispatch + node capability registry | Host-bound channels are marked unroutable with a reason. |
| `vrooli-events` | scenario | no | Event-driven ingress | Inbound subscription webhook to this scenario's receiver | Event ingress unavailable; direct requests unaffected. |
| Push provider | third-party | conditional | Push channel adapter | `cloud-api` resource supplying credentials | Push deliveries fail retryably, then terminally with a stated reason. |

## Vrooli Resources

The core carries **no resource dependency at all**, and this is a
capability decision rather than a default left untouched. PostgreSQL and
Redis are still Docker-backed and are recorded `unsupported` on macOS and
Windows in `path:docs/reference/platform-support.md`, so depending on
either would make this scenario unable to run on the Mac fleet node that
the relay lane exists to reach. See OT-P0-009.

Channel credentials arrive through optional `cloud-api` resources, which
are credential-and-reachability only and carry no local process.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `postgres` | rejected | Docker-backed; `unsupported` on macOS and Windows. Would forfeit cross-node delivery. | Revisit only if the resource earns a portable managed-service acquisition path. |
| `redis` | rejected | Same portability ceiling. Queue, retry schedule, and rate-limit counters are in-process and SQLite-backed instead. | Revisit if this scenario ever runs replicated, which is not planned. |
| `ntfy` | **not yet created** | Intended push provider for OT-P0-001. No resource and no blueprint exists today; the nearest artifact is the `pushover` blueprint at `status: candidate`. | Blocks OT-P0-001. Create as `cloud-api`, promote to `managed-service` when self-hosted (OT-P1-008). |
| `twilio` | available, unused | Existing `cloud-api` resource with `account-sid` and `auth-token` descriptors. | Adopt when OT-P2-002 (SMS) activates. |

## Scenario Dependencies

Declared in `.vrooli/service.json` under `dependencies.scenarios`. All
three are `runtime_only`: this scenario imports no Go package from any of
them, so no import-graph edge should be expected as evidence.

| Scenario | Status | Startup Policy | Contract |
|---|---|---|---|
| `scenario-authenticator` | required | `try_start` | Identity for every authenticated surface. Tokens are RS256 and verified locally against the published JWKS through `api-core/owneridentity`; there is no per-request authorization callback. This scenario issues no API key, stores no password, and owns no profile table (OT-P0-007). |
| `vrooli-bridge` | optional | `ignore` | Node registry (`OS`, `Arch`, `Capabilities`, `LastSeenAt`) for capability-aware routing, plus durable dispatch for the delivery itself. Durable dispatch is chosen over the synchronous relay call so a delivery survives a node that is briefly offline (OT-P1-001). |
| `vrooli-events` | optional | `ignore` | Inbound webhook delivery from durable subscriptions to this scenario's REST receiver (OT-P1-003). |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Push provider (ntfy or equivalent) | planned, P0 | Only path to the owner's iPhone that needs neither an Apple Developer account nor a Mac to build and sign an app. | HTTPS POST to a topic. Body content is governed by the sensitivity label (OT-P0-010), because a public topic exposes its body to anyone holding the topic name. |
| SMTP relay | planned, P1 | Email channel. | Operator-supplied credentials (OT-P1-004). |
| Twilio | planned, P2 | SMS channel. | Credentials from the `twilio` resource (OT-P2-002). |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `scenario-authenticator` | JWKS fetch fails or signature does not verify | Authenticated surfaces reject with `unauthenticated`. The delivery worker keeps draining already-accepted work. | identity middleware tests |
| `vrooli-bridge` | Dispatch rejected, or no node advertises the channel capability | Delivery is marked **unroutable** with a stated reason and surfaces in the timeline. It is never left silently `pending` (OT-P0-011). | routing tests, capability-selection tests |
| `vrooli-events` | Webhook receiver never called | Event ingress is silently absent, which is acceptable: direct ingress is the P0 path. | receiver contract tests |
| Push provider | Non-2xx, or topic rejected | Retry with backoff, then terminal failure with an actionable reason (OT-P0-008). | channel adapter tests |

## Known Upstream Gap

`vrooli-events` stores subscriptions and can deliver a webhook when
triggered by hand, but nothing fans them out automatically: on ingest it
publishes to the SSE broker only, and `WebhookDeliverer.Deliver` is
reachable solely from a manual trigger endpoint. There is no matcher, no
retry queue, and no delivery engine. Event ingress therefore depends on
work in a scenario this one does not own, which is why OT-P1-003 is P1
and not P0. Do not plan P0 work that assumes this gap is closed.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
