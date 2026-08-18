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
| SQLite | embedded storage | yes | API, all persisting domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy; no notification is accepted, so nothing is silently lost. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `scenario-authenticator` | scenario | yes | recipients, all authenticated surfaces | RS256 tokens verified locally against its JWKS through `api-core/owneridentity` | Cached JWKS keeps verification working through a brief outage; once the cache expires, authenticated calls fail closed. |
| `ntfy` (push provider) | cloud-api resource | no | channels, delivery | Credential descriptors and endpoint from the resource manifest | The push channel reports unavailable and routing falls through to the next channel. |
| `vrooli-bridge` | scenario | no | relay (P1) | Node registry read plus durable dispatch | Channels this host cannot serve are reported unavailable rather than failing; local channels are unaffected. |
| `vrooli-events` | scenario | no | ingress (P1) | Inbound webhook to the ingress receiver | Event-raised notifications stop; direct requests are unaffected. |
| `twilio` | cloud-api resource | no | channels (P2) | Credential descriptors from the resource manifest | SMS channel reports unavailable. |

## Vrooli Resources

**The core declares no resource dependency.** This is the single most
consequential decision in the scenario, and it is a capability
requirement rather than a preference.

`resource-postgres` and `resource-redis` both acquire through
`acquisition.kind: "oci-image"` and both record `macos: unsupported` and
`windows: unsupported`. The fleet-wide matrix in
[`docs/reference/platform-support.md`](../../../../docs/reference/platform-support.md)
states it directly: *"PostgreSQL and Redis remain on the existing
docker-service driver; native macOS/Windows migration is a separate
plan."* This scenario must run on a macOS node to serve Apple-only
channels, so a dependency on either would make OT-P1-001 and OT-P1-002
unbuildable. Everything Redis was proposed for — the queue, retry
scheduling, rate-limit counters — is a table and a ticker at one owner's
volume.

Delivery providers are a different case. They stay resources, because a
resource is where credential descriptors and reachability health belong,
but they are `cloud-api` resources, which are `host-required` bundling
and supported on all three platforms. The division of labour follows
`resource-twilio`, whose entire CLI surface is `provider-check`: the
resource owns credentials and reachability, the scenario owns the send
call.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `postgres` | rejected | OCI-acquired and `unsupported` on macOS/Windows; would strand the scenario on Linux. SQLite serves the access pattern. | Postgres completes a portable acquisition path AND the data outgrows SQLite. Both, not either. |
| `redis` | rejected | Same portability ceiling. The queue and counters are in-process. | Delivery volume makes an in-process queue the measured bottleneck. |
| `ntfy` | to create | `cloud-api` template. Owns the push endpoint, an optional access token, and provider reachability health. | Created during the P0 push work. Promote to `managed-service` when the owner self-hosts (OT-P1-008). |
| `twilio` | available, unused | Already implemented as `cloud-api` with credential descriptors. Adopt only when SMS is actually wanted. | OT-P2-002. |
| SMTP/email | no resource | There is no SMTP resource in the fleet, and one endpoint plus one credential does not justify creating one. Credentials bind through a descriptor on the channel adapter. | A second scenario needs the same SMTP credentials. |

A note the `pushover` blueprint already records, and which applies to
every provider added here: *"shallow wrappers often duplicate simple
HTTP calls without much value."* Keep provider resources thin. A
provider resource that grows a send API has taken work that belongs to
the `channels` domain.

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `scenario-authenticator` | required | Supplies identity. This scenario deliberately owns no accounts, no passwords, and no API keys of its own — that is OT-P0-007. | RS256 access tokens verified locally against `/.well-known/jwks.json` through `api-core/owneridentity`. No per-request callback. Recipients are keyed by the token subject, which is what makes multi-user routing (OT-P2-003) a data consequence rather than a rewrite. |
| `vrooli-bridge` | optional, P1 | Supplies the fleet node registry and the durable dispatch path used to deliver through another machine. | Read node `OS`, `Arch`, `Capabilities`, and `LastSeenAt` from the registry; submit deliveries as durable dispatch jobs rather than synchronous relay calls, so a delivery survives a node that is briefly offline. |
| `vrooli-events` | optional, P1 | Supplies event-driven ingress. | A durable webhook subscription targeting this scenario's ingress receiver. |
| `device-control` | not a dependency | It governs *acting on* devices under a lease. Notification delivery is not a device action and must not acquire a lease to send a message. | — |

### Known upstream gap: `vrooli-events` fan-out

`vrooli-events` documents this scenario as its primary consumer, and its
README shows a subscription targeting a hooks endpoint here. That path
does not work today, and the gap is on the events side.

On ingest, `vrooli-events` publishes to its SSE broker and nothing else.
`WebhookDeliverer.Deliver` is reachable only from a manual
"trigger this subscription" endpoint. There is no matcher, no retry
queue, and no delivery engine. Subscriptions are stored; they do not
fire.

This is why event ingress is OT-P1-003 and not a P0. Building the
receiver here is cheap; making anything call it is work in a scenario
this one does not own. That work should be filed against `vrooli-events`
rather than absorbed here, and the ingress receiver should be built and
tested against a synthetic caller so it is ready when the upstream
engine lands.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| ntfy | planned, P0 | The shortest path from nothing to a notification arriving on the owner's phone. Free, open source, self-hostable later, with an App Store client. | HTTPS POST to a topic endpoint. The topic name is a bearer secret. Bodies are subject to the sensitivity rules in `SECURITY.md`. |
| Apple Push Notification service | not used | Reaching an iPhone does not require APNs. A first-party APNs app needs an Apple Developer account, a Mac to sign, and `scenario-to-ios` — none of which this scenario should own. | — |
| SMTP provider | planned, P1 | Email is a universally reachable fallback channel. | Operator-supplied host, port, and credentials. |
| Twilio | deferred, P2 | SMS costs money per message and is rarely the right channel when push works. | Through the `twilio` resource's credential descriptors. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` reports unhealthy and the API refuses new notifications. Refusing is correct: accepting a notification the system cannot persist is a silent loss, which is the one outcome this scenario must never produce. | health handler tests |
| `scenario-authenticator` unreachable | JWKS fetch failure | Cached keys continue to verify tokens until they expire, then authenticated calls fail closed. Never fail open. | owneridentity client tests with an unreachable resolver |
| Push provider unreachable | HTTP error or timeout on send | The delivery is marked retryable and rescheduled under backoff. After the retry budget, it is marked failed with the provider's reason. The notification stays visible in the timeline as failed. | channel adapter contract tests against a fake transport |
| Push provider rejects the topic | 4xx from the provider | Terminal, not retryable. The delivery fails with a reason naming the device, so the operator knows to re-register it. | channel adapter contract tests |
| `vrooli-bridge` unreachable | Registry read or dispatch failure | Channels only reachable through a node are reported unavailable; routing falls through to a locally servable channel. Local delivery is never blocked by a bridge outage. | routing decision tests with an empty node capability set |
| Target node offline | Dispatch queued, never acknowledged | The delivery stays `delivering` until the dispatch times out, then fails with a reason naming the node. Durable dispatch is chosen over synchronous relay precisely so a brief absence is not an immediate failure. | relay tests with a stubbed dispatch seam |
| `vrooli-events` unreachable | No inbound webhooks | Event-raised notifications stop; nothing else degrades. Direct requests are the primary ingress for this reason. | ingress handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why these dependencies were chosen
