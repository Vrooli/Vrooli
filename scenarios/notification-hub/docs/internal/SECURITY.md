# Security — Notification Hub

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

This scenario has no customers and no payment flow, and it is still one
of the more privacy-sensitive scenarios in the fleet. It knows who you
are, which devices you own, how to reach each one, when you sleep, and
the text of every message anything in Vrooli sent you.

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Notification body and title | high | notifications | Free text written by any scenario or agent. It is the most likely place for a secret to leak by accident, because the caller controls it and the reader is a locked screen. |
| Channel addresses | high | recipients | A push topic name is a bearer secret: whoever holds it can send to the device, and on shared infrastructure can read what was sent. Email addresses and phone handles are personal data. |
| Quiet windows | medium | recipients | Discloses when the owner sleeps and when they are away. Low impact alone, useful to an attacker in aggregate. |
| Device inventory | medium | recipients | Discloses which machines and phones the owner has, and which are currently reachable. |
| Delivery history | medium | delivery | Reveals activity patterns across the whole fleet even when bodies are withheld — a timeline of what happened and when. |
| Provider credentials | high | channels (reference only) | Never stored in this scenario's database. Held by the native credential authority and resolved through resource credential descriptors. |
| Identity subject | low | recipients | An opaque authenticator subject, not a credential. |

## The Sensitivity Model

Every notification carries a `sensitivity` label. The column is NOT NULL
with no default, so a caller decides once rather than inheriting a
guess.

| Label | Meaning | Body handling |
|---|---|---|
| `public` | Nothing is disclosed by the text. "Build finished", "backup complete". | Body may be sent through any enabled channel, including a third-party push service. |
| `private` | The text discloses activity, names, or amounts the owner would not want on a shared screen. | Title is sent; body is withheld and replaced with a link back to the console. Only channels marked `body_ok` for `private` receive the full text. |
| `secret` | The text contains or could contain credentials, tokens, keys, or regulated data. | Never leaves the machine as text. Only a bare "you have a notification" plus a link is delivered, regardless of channel. |

Routing enforces this, not the channel adapter, so a new adapter cannot
introduce a leak by forgetting to check.

**The house convention that makes this workable:** a notification says
*what happened* and links back to the console for detail. It never
carries the record itself. That keeps the same sentence safe on a locked
screen, in a shared room, and in a third-party provider's request log,
and it means most notifications are honestly `public`.

## Auth And Authorization

This scenario owns no identity. It stores no password, issues no API
key, and runs no login flow. That is OT-P0-007 and it is deliberate: the
previous implementation carried per-profile bcrypt API keys and an admin
surface, which was both a maintenance burden and an unnecessary
credential store.

- **Authentication** — RS256 access tokens issued by
  `scenario-authenticator`, verified locally against its published JWKS
  through `api-core/owneridentity`. No password reaches this scenario
  and no per-request callback is made.
- **Failure mode** — cached keys keep verification working through a
  brief authenticator outage. Once the cache expires, authenticated
  calls fail closed. Never fail open; an open notification API is a
  spam vector aimed at the owner's phone.
- **Authorization** — enforced at the API and service layer only. The
  UI and CLI must not decide access locally.
- **Trust posture** — recipients are keyed by the token subject, so a
  `shared` or `hosted` install (`.vrooli/operator-state.json`) holds
  several subjects in the same table with no schema change. In the
  current `personal` posture there is exactly one.
- **Sending on behalf of a scenario** — a scenario calling the send API
  is identified by its own caller identity, and that identity is
  recorded on the notification. "Who told me this" must always be
  answerable.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Push provider token | `ntfy` resource credential descriptor | no | Optional for a public topic, required for an authenticated or self-hosted endpoint. Resolved by logical id and field; never read from a file in this tree. |
| SMTP credentials | credential descriptor on the email channel adapter | no | Required only when the email channel is enabled (OT-P1-004). |
| Twilio account SID and auth token | `twilio` resource credential descriptors | no | Already declared by the existing resource; adopted only if SMS ships (OT-P2-002). |
| Authenticator JWKS | fetched over HTTP from `scenario-authenticator` | yes | Public key material, not a secret. Cached locally. |

No secret is stored in this scenario's database, in
`.vrooli/service.json`, or in the repository. Channel configuration
holds a reference to a descriptor; the credential authority holds the
value.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| A push topic name leaks | An attacker sends notifications to the owner's phone, and on a public provider can read notifications sent to it. | Topic names are high-entropy and generated, never guessable or derived from the scenario name. They are treated as bearer secrets in `DATA.md` and are not shown in logs or in screenshots taken by evidence capture. `secret` bodies never transit the provider at all. | designed, P0 |
| A caller puts a credential in a notification body | The credential appears on a locked screen and in a third-party provider's logs. | The sensitivity label plus the link-not-payload convention. Adapters receive an already-redacted body from routing, so no adapter can bypass it. | designed, P0 |
| Third-party push provider reads message content | Body text transits infrastructure the owner does not control. | Only `public` bodies are sent as text. OT-P1-008 (self-hosted push endpoint) removes the exposure entirely for owners who want it. | designed; residual until OT-P1-008 |
| Unauthenticated send | Anyone reaching the API can spam the owner's devices, which trains the owner to ignore notifications and defeats the scenario. | Authenticated send only, failing closed. Rate limiting per caller identity. | designed, P0 |
| Webhook ingress forgery | A forged inbound webhook raises arbitrary notifications. | The ingress receiver requires a shared secret or signature and matches an explicit ingress rule; it never raises a notification from an unmatched payload. | designed, P1 |
| Relayed delivery leaks to the wrong node | A notification body is dispatched to a fleet node that should not see it. | Dispatch targets a node id resolved from the bridge registry and checked against the recipient's device ownership, not from caller input. `secret` notifications are never relayed as text. | designed, P1 |
| Compromised Mac node reads relayed content | A node holding the iMessage adapter sees every notification routed through it. | Only notifications addressed to a channel that node serves are dispatched to it. This is a real residual risk of any relay design and is the reason the relay lane is opt-in per channel. | accepted, documented |
| Notification history discloses fleet activity | The delivery timeline is a record of everything that happened, retained by default. | Retention window bounds the exposure; the timeline is behind the same authentication as everything else. | designed, P0 |
| Quiet-window enforcement bypassed by urgency | A caller marks everything `critical` and defeats quiet hours. | Urgency is recorded per caller identity and the analytics surface reports critical-rate by caller, so an abusive caller is visible rather than merely annoying. | designed, P1 |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Body content transits a third-party push provider for `public` notifications | medium | OT-P1-008 ships a self-hosted push endpoint. |
| No per-caller rate limit before the routing core exists | medium | Ships with the P0 routing core; until then the API is authenticated but unthrottled. |
| Relay exposes notification content to the target node | low, accepted | Inherent to relayed delivery. Revisit if a node is ever shared with a second person. |
| No signing on the ingress webhook receiver | medium | Ships with OT-P1-003; the receiver is not enabled before then. |
| macOS iMessage automation needs Full Disk Access on the node | medium | OT-P1-002. Granting it widens what a compromised node agent can read, which is part of why iMessage is best-effort rather than a gate. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`DECISIONS.md`](DECISIONS.md) — why identity is delegated
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
