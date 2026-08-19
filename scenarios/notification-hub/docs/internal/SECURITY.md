# Security — Notification Hub

Threat model, secret handling, and the authorization boundary.

## Purpose Of This Document

This scenario can be told anything by any scenario in the fleet, and it
sends what it is told to a human's locked screen. That makes two questions
security questions rather than product questions: what may appear on that
screen, and who may cause something to appear there.

## Data Sensitivity

Notification bodies carry whatever a caller puts in them. The scenario
does not inspect them and does not guess.

`sensitivity_label` is required at ingress and is never inferred. Only the
caller knows whether a body is safe on a locked screen or in a third
party's logs. A channel is treated as unapproved for a label unless it
declares approval; an unapproved channel receives a content-free pointer
to the console instead of the body, and the notification is never dropped
silently (OT-P0-010).

Web Push payloads are encrypted end to end under RFC 8291 with keys held
only by this scenario and the recipient's browser. No push service can
read a body. This property is why the self-hosted relay alternative was
rejected: that design kept bodies on an owner-run server but required the
phone to reach it at delivery time, degrading to a contentless placeholder
when it could not.

Delivery error reasons appear in the timeline and the UI. An adapter must
never place body content into an error reason.

## Auth And Authorization

**Identity is not owned here.** There is no password, no profile table,
and no scenario-issued API key (OT-P0-007). Owner tokens are verified
locally against `scenario-authenticator`'s published JWKS through
`api-core/owneridentity`, with no per-request callback. A recipient row is
a projection of a verified external identity.

Authenticated surfaces fail closed. When the authenticator is unreachable,
the API rejects requests it cannot attribute to a verified identity.
Already-accepted notifications continue to drain, because their recipient
was resolved at acceptance time.

**Trust posture selects the recipient model.** `trust_posture` in
`.vrooli/operator-state.json` is `personal`, `shared`, or `hosted`.
Multi-user routing is a data consequence of keying recipients by
authenticator identity, not a separate feature: in `personal` there is one
recipient and the UI shows no picker; in `shared` or `hosted` the same
tables hold several.

**Cross-node authority belongs to `vrooli-bridge`.** This scenario never
decides whether it may reach a machine. It asks; the bridge checks the
verb against the derived scope catalog and the node's granted scopes, and
refuses or carries the call.

## Secrets

| Secret | Where it lives | Rules |
|---|---|---|
| VAPID private key | Scenario-owned storage, never in the repository | Rotating it invalidates nothing, but the public half is baked into every existing subscription, so rotation requires re-subscribing every browser. Treat rotation as a migration. |
| Push subscription material | `push_subscriptions` table | Credential-equivalent. Anyone holding an endpoint plus its `p256dh` and `auth` keys can push to that browser. Never logged, never exported, never in a support bundle. |
| SMTP credentials (OT-P1-004) | Credential authority, referenced by descriptor | Not stored by this scenario. |
| Twilio credentials (OT-P2-002) | The `twilio` resource's credential descriptors | Not stored by this scenario. |

The scenario holds no owner password and mints no token.

## Threat Model

| Threat | Impact | Control |
|---|---|---|
| A caller sends a sensitive body to a channel that shows it on a locked screen | Content disclosure to anyone holding the device | Required `sensitivity_label` with default-deny per channel and a content-free fallback |
| A stolen `push_subscriptions` row | Attacker pushes arbitrary notifications to that browser | Treat the table as credential storage; never export; delete on `410 Gone` |
| An expired subscription that still looks healthy | Every send silently succeeds into nothing, and the owner believes they are covered | OT-P0-014: delete on gone, renew on change, and report the channel unavailable rather than healthy |
| The public origin lapses | Every subscription on every device dies at once | OT-P0-015: core-tier exposure, never auto-expired |
| A compromised scenario floods the spine | Notification fatigue, the owner mutes everything, and every downstream promise is void | Rate limiting per caller, duplicate suppression, and quiet hours |
| A notification body reaches a remote machine that should not see it | Content disclosure across the fleet | Sensitivity policy is evaluated before dispatch, not on the remote side |
| Widening the bridge dispatch vocabulary widens every node at once | A node can be asked to run commands the operator never intended | The vocabulary refactor must ship with narrowed per-node grants; a bare effect scope must stop meaning "every verb of that effect" |

## Security Gaps

Recorded honestly rather than deferred silently.

- **No rate limiting at ingress yet.** Notification fatigue is named in the
  PRD as the real failure mode, and duplicate suppression only covers
  repeats of the same key. A caller emitting distinct notifications in a
  loop is currently unbounded.
- **No per-caller authorization.** Any authenticated caller may notify any
  recipient on this install. Acceptable at `personal` trust posture;
  it is a gap the moment the posture changes.
- **Escalation chains can leak across sensitivity tiers.** An unanswered
  critical ask escalates to the next channel in the chain, and that
  channel may have a lower approval level than the first. Escalation must
  re-evaluate the sensitivity policy at each step rather than inherit the
  first decision.
- **The bridge grant model is currently coarse.** A bare
  `vrooli-bridge:write` grant satisfies any write verb in the dispatch
  vocabulary. This is bounded today only because that vocabulary is small.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy notes.
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency trust boundaries.
- [`PROBLEMS.md`](PROBLEMS.md) — open defects.
- [`../../PRD.md`](../../PRD.md) — OT-P0-007, OT-P0-010, OT-P0-014, OT-P0-015.
