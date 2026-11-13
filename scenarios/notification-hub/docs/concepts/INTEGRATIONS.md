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
| `agent-manager` | scenario | no | Native consent projection in Needs You | Lifecycle-resolved loopback status GET; source approval page remains authoritative | Current consent stays visibly unverified. The Hub does not start the source or grant permissions. |
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
| `scenario-authenticator` | required | `try_start` | Identity for every authenticated surface. Tokens are RS256 and verified locally against the published JWKS through `api-core/owneridentity`; there is no per-request authorization callback. This scenario issues no API key, stores no password, and owns no profile table (OT-P0-007). The installed web app signs in through the hub's same-origin `IdentityService` facade, which forwards to the authenticator and keeps the access and 7-day refresh tokens in HttpOnly cookies; `GetSession` renews an expired session, and browser calls never receive a token in a body. |
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

The API continuously reconciles named subscriptions in `vrooli-events`,
all delivered to `POST /api/v1/integrations/events`:

| Name | Pattern | Purpose |
|---|---|---|
| `notification-hub-events` | operator-configured (default `incident.**`, `PUT /api/v1/config/event-integration`) | Fallback-only ingress for events that carry no neutral `notificationFacts`. The incident renderer (check_id/severity/message/…) stays the translator for this pattern and for autoheal remediation approvals |
| `notification-hub-decisions` | `**.decision_requested.v1` | Operator decisions from any scenario (Agent Manager first) |
| `notification-hub-decision-withdrawals` | `**.decision_withdrawn.v1` | A producer closing its own open decision (see "Decision subjects"). Agent Manager's typed run and investigation withdrawals arrive here too; reconciliation widens the earlier exact-type subscription in place and deletes `notification-hub-investigation-withdrawals`, so no withdrawal is delivered twice |
| `notification-hub-notifications` | `**.notification.v1` | First-class FYI notifications from any scenario, carrying neutral `notificationFacts` (the hub owns the copy) |

The `**.notification.v1`, `**.decision_requested.v1` and
`**.decision_withdrawn.v1` subscriptions are
**structural**: they are reconciled unconditionally from the event-type suffix,
never operator-configured, exactly parallel to one another. The operator-configured
`incident.**` pattern and its incident renderer remain only as the **fallback** for
events that do not carry neutral facts, so existing incident producers keep working
unchanged. Reconciliation is idempotent. The events scenario owns matching, durable
queueing, retries, signing, and subscription health; this scenario owns only
the receiver and the translation into a durable notification or ask. If
reconciliation or the upstream scenario is unavailable, direct Connect-RPC
and CLI ingress remain unaffected.

For Agent Manager decisions, the answer outbox retains a resolution until
Events confirms the exact answer event is queued for the named Agent Manager
subscription. A bounded, restart-persistent sweep continues past failed
resolutions. This repair requires the Events missing-delivery reconciliation
endpoint; previously published resolutions are not retrospectively certified.
Queued and delivered are separate states, and dead letters retain their retry
history.

**Signature.** vrooli-events signs every delivery with HMAC-SHA256 over the
exact body (`X-Vrooli-Events-Signature: sha256=<hex>`). The receiver verifies
with the same secret the publisher uses, resolved the same way:
`VROOLI_EVENTS_WEBHOOK_SECRET` when set, otherwise the credential authority
entry `vrooli/vrooli-events` field `agent-manager-webhook-secret`. The value is
resolved lazily, cached after the first success, retried at most every 30 s
while unavailable, and never logged. An unavailable secret fails closed (401).

### Decision requests (`*.decision_requested.v1`)

The producer sends facts; the hub owns the copy. The event id is the
idempotency key, so a redelivery or re-publish returns the same ask and
sends nothing new.

Write operator-facing questions to the shared [operator legibility
standard](../../../../docs/agent-system/SWARM_MANAGER_WORK.md#operator-legibility).
The recipient may not know the sending scenario or its terms. Explain the
problem and its practical effect in the question, define necessary acronyms at
first use, and state what each option causes the owner to do. Put detailed
evidence in `big_picture` or a useful context link; do not make it necessary to
understand the choice. The recommendation reason should explain the benefit or
risk in ordinary language. Each event represents one choice and one next
action. Keep the question short and plain; explain its practical consequence in
`big_picture` and state each option's result. If the owner needs answers to
independent or multi-part decisions, publish separate events so the operator
can answer each one with a tap. Do not require an operator to compose JSON or
combine unrelated choices in the free-text `Other` answer.

| Field | Required | Meaning |
|---|---|---|
| `question` | yes | One sentence, at most 600 characters |
| `options` | yes | 2–8 `{key, label}`; the key is what the requester gets back |
| `recommended` | yes | One option key, shown first |
| `recommendation_reason` | no | At most 300 characters |
| `default_answer` | no | One option key; allowed only with `reversible: true` |
| `reversible` | with a default | The decision can be undone later |
| `deadline` | no | RFC 3339; defaults to 24 h after receipt |
| `urgency` | no | `low`, `normal`, `high` (default) or `critical`; only `critical` can pass a quiet window, and only when that window allows it |
| `subject` | no | Short name shown in the title ("Decision needed: BAS") |
| `context_url` | no | HTTPS/HTTP link, or a local artifact path opened in Web Console |
| `correlation` | no | String map (at most 1 KB) echoed verbatim in the resolution event |

Any scenario opts into operator asks by publishing a `*.decision_requested.v1`
event with this payload — no code changes here. For example, Agent Manager
publishes `agent_manager.run.decision_requested.v1` when a run enters
`needs_review`, with `options` `approve`/`reject`, a `context_url` to the run's
review page, and `correlation.run_id`; the resolution's `answer` then drives the
run's approve/reject back in Agent Manager. This is the general mechanism, not a
hub-side special case.

### Decision subjects: one open decision per situation

A decision that is about an ongoing situation (a blocked backup, a mode left on,
a pending grant) carries a **subject key**. The hub then keeps at most one open
decision per `(source, source_event_type, subject_key)`, the producer can
update or close it, and a stale decision closes itself. Producers use
`operatordecision.Facts.SubjectKey` and `operatordecision.Withdrawal`
(`packages/api-core/operatordecision`).

| Request field | Meaning |
|---|---|
| `subject_key` | 1–128 of `A-Z a-z 0-9 . _ : / @ -`, first a letter or digit, e.g. `backup-failure:<destination-id>` |
| `reassert_within_seconds` | Optional, 60 s – 30 days, needs `subject_key`. The hub closes the open decision when the producer has not re-sent it for this long |
| `asserted_at` | Optional RFC 3339 time the producer last saw the decision was still needed; orders a request against the subject's withdrawals |

What a request with a subject key does:

| State of the subject | Request terms | Result |
|---|---|---|
| No open decision | any | A new decision is created and delivered |
| Open decision | identical | Nothing is delivered; the hub records the re-assertion time |
| Open decision | same answer menu (option keys in order, recommended, default, reversible, urgency, recipient, sensitivity; option **labels** are text), other text or correlation changed | **Update in place**: the text, correlation and deadline become the new revision's, no new push, the previous text is kept as a revision |
| Open decision | different answer menu | **Replace**: the old decision closes as withdrawn, "Replaced by a newer version of this decision", linked to the new one, which is delivered |
| Last decision dismissed by the operator | identical to the dismissed terms | Suppressed: no new decision until the terms change |
| Last decision answered or defaulted under 15 minutes ago | identical | Returns that decision (the producer has not processed the answer yet) |
| Last withdrawal asserted after this request's `asserted_at` | any | Ignored as stale; the response says `stale: true` |

Details the table leaves implicit:

- A request with no `deadline` keeps the open decision's deadline; it does not
  extend it. A deadline-only change (the producer's explicit deadline moved)
  is applied without a new revision.
- A request carrying the event id of a revision the decision has already moved
  past (a redelivery) changes nothing, as does a re-send whose `asserted_at`
  is older than the decision's last confirmation. The same event id with
  different terms is `409`.
- At most 50 earlier revisions are kept per decision; `revision_count` keeps
  counting.
- The request response names the outcome: `created`, `reasserted`, `updated`,
  `replaced`, `suppressed` (a dismissed decision with identical terms) or
  `returned` (answered or defaulted under 15 minutes ago); `stale: true` and no
  `ask_id` for a stale request.

Every accepted re-send (an update in place, or identical terms under a new
event id) makes its event id the decision's `requester_ref`; every earlier id
is kept and still matches. A redelivered earlier
revision returns the current decision unchanged, and a request asserted
before the current revision is ignored, so an update never reverts. An answer
to a decision that was updated in place carries the latest revision's
`requester_ref` and correlation. A producer must still treat an answer that names an
older generation of the same subject as an answer to the current one, or as a
benign no-op.

**Closing (`*.decision_withdrawn.v1`).** The producer publishes
`<source>.<kind>.decision_withdrawn.v1` with `reason` (1–300 bytes, shown to the
operator, plain words) and either `subject_key` or `requester_ref` (the request
event id; an earlier revision's event id also matches). `source_event_type`
names the request type and defaults to the withdrawal type with its suffix
swapped. The producer is the event's source: a withdrawal closes only that
source's decision. Naming a `requester_ref` finds the decision by any of its
revisions' event ids, and follows a replaced decision to its successor. The hub
closes only an open decision, as `withdrawn` with the reason;
an answered, defaulted, expired or dismissed decision keeps its outcome. A
withdrawal that arrives before its request is recorded, so a delayed request
asserted earlier is not delivered. Agent Manager's typed run and investigation
withdrawals keep their fixed reasons (`verified_empty_diff` and the like are
codes, not free text); their event types are never read as generic ones.
Ordering uses `asserted_at` (the hub's receipt time when absent): a withdrawal
asserted before the decision's last confirmation does not close it, and a
withdrawal that names a `subject_key` is remembered per subject for the stale
rule above.

**Safety nets.**
- A subject with `reassert_within_seconds` whose producer stops re-sending
  closes as withdrawn: "No longer confirmed by <source> since <time>".
- An open decision whose default was withheld because it was never delivered
  expires 7 days after its deadline instead of staying open forever.

**Direct asks** (`notification-hub conversations ask`) take the same
`--subject-key`; `conversations withdraw --subject-key <k> --reason <r>` closes
the requester's own open ask for that key.

**Resolutions** carry `subject_key` and, on a replaced decision,
`superseded_by_ask_id`. `state: withdrawn` means no operator answer applies.

### Notifications (`*.notification.v1`)

The producer sends facts; the hub owns the copy. Any scenario raises a
first-class FYI notification by publishing an event whose type ends with the
structural suffix `.notification.v1` — no code changes here, exactly like
`*.decision_requested.v1`. The event id is the idempotency key, and
`dedupe_key` collapses repeats from the same producer. These neutral fields
are the `notificationFacts` contract, parallel to `decisionFacts`.

| Field | Required | Meaning |
|---|---|---|
| `title` (or `subject`) | no | Short headline; the hub supplies one from source branding + category when absent |
| `body` | yes | The message shown to the recipient |
| `urgency` | no | `low`, `normal` (default), `high` or `critical`; only `critical` can pass a quiet window, and only when that window allows it |
| `sensitivity` | no | `public`, `private` (default), `sensitive` or `critical`. **Producer-declared**: this replaces the old incident-only derivation of sensitivity from `severity`, so the producer states privacy directly |
| `category` | no | Per-event-type key used for recipient preferences and the settings catalog; defaults to the event type when absent |
| `context_url` | no | Deep link into the source scenario |
| `dedupe_key` | no | Producer-supplied dedupe key; collapses repeats. Replaces the old hardcoded `incident_id` extractor |
| `correlation` | no | String map (at most 1 KB) echoed downstream |

Every inbound notification also carries its **source scenario** and a resolved
display identity `{name, icon, deep_link}`, resolved declared manifest → observed
source → generic, rendered in the UI timeline and the push payload. Incidents
become one producer of this contract; the legacy incident renderer
(check_id/severity/message/…) is demoted to a fallback used only for events that
do not carry neutral facts, so nothing breaks. The full specification — the
branding manifest, the observed+declared catalog, the preference model, and the
centralized-delivery principle — lives in [`NOTIFICATIONS.md`](NOTIFICATIONS.md).

### Ask resolution (`notification_hub.ask.resolved.v1`)

Every terminal transition of an ask (answered, defaulted, expired, withdrawn,
dismissed, and a late answer after a default) publishes one event through Vrooli Events with source
`notification-hub`. The payload carries `ask_id`, `state`, `answer`,
`answer_label`, `note`, `answered_by` (`default` for a default),
`answered_at`, `by_default`, `late`, `default_answer`, `reason`, `source`,
`source_event_type`, `requester_ref` (the event id or idempotency key that
opened the ask), `subject_key`, `superseded_by_ask_id` (set on a decision a newer
one replaced), `correlation` (the latest revision's), and `resolved_at`. Publication uses an outbox:
a failed publish is retried by the one-minute sweep, and the event id is
derived from the content, so a retry is deduplicated by vrooli-events.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness

## Native human consent visibility

Agent Manager interactive authorization stays native. Its
`agent_manager.authorization_requested.notification.v1` FYI is delivery evidence,
not an answerable Hub ask. `ConversationsService.ListNativeConsents` projects
existing configured-recipient notifications whose source is `agent-manager`,
category is `authorization-requested`, canonical UUID dedupe identity matches the
original consent URL, and current source state is checked. Candidate pagination
is independent of the 20-row Updates window; even a closed-only page has a next
cursor when older candidates remain.

The optional source dependency is Agent Manager's lifecycle-resolved loopback
`GET /api/v1/authorizations/{id}/status`. Hub sends no owner credentials, follows
no redirects, never fetches a notification-provided URL, and never starts Agent
Manager. The reader returns only authorization identity, effective state, and
review expiry. Missing, invalid, unreachable, or in-flight source state stays
visibly unverified rather than clearing the queue. Approved, revoked, expired,
and superseded source states are omitted from Needs You. No grant state is cached
or written into Hub, and neither viewing nor opening the original review URL
answers an ask or approves a permission. The sole authorization action remains
Agent Manager's human-verified consent page. Updates also retains safe source
context links.

A push transport receipt means provider acceptance, not OS display. This repair
does not create requests, resend notifications, or claim that an iPhone displayed
an existing receipt. Device display verification remains a separate operator test.
