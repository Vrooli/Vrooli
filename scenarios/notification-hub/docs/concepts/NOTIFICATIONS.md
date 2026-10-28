# Notifications — Notification Hub

This document is the canonical contract for first-class FYI notifications:
how any scenario raises one, how it is branded, how recipients configure
which ones reach them, and why delivery stays in one place.

## Purpose Of This Document

Use this document to answer:

- How does a scenario raise a notification without writing any hub code?
- What does the producer supply, and what does the hub own?
- How is a notification attributed and branded to its source scenario?
- How does a recipient mute, throttle, or re-route a category?
- Why is there exactly one delivery engine, and not one per scenario?

This is the FYI-notification counterpart to the operator-decision contract
in [`INTEGRATIONS.md`](INTEGRATIONS.md) (`*.decision_requested.v1` →
`decisionFacts`). Both follow the same rule: **the producer supplies facts,
the hub owns the copy.** Read this as the forward contract — it describes the
contract, not the Go internals that satisfy it.

## The `.notification.v1` Convention

A scenario raises a first-class notification by publishing an event whose
type **ends with the structural suffix `.notification.v1`**. The hub
auto-subscribes to `**.notification.v1` at reconcile, unconditionally and
without operator configuration — exactly the mechanism already used for
`**.decision_requested.v1`. There is no per-producer registration and no
hub-side special case: the suffix is the contract.

The event id is the idempotency key, so a redelivery or re-publish produces
no new notification. The producer-supplied `dedupe_key` collapses distinct
events that represent the same underlying fact.

## Neutral Producer Contract (`notificationFacts`)

The producer sends neutral facts only; the hub composes the copy, the title,
and the presentation. These fields are parallel to `decisionFacts`.

| Field | Required | Meaning |
|---|---|---|
| `title` (or `subject`) | no | Short headline, shown exactly as sent (no source prefix). When absent, the hub falls back to the source's display name |
| `body` | yes | The message shown to the recipient |
| `urgency` | no | `low`, `normal` (default), `high` or `critical`. Only `critical` can pass a quiet window, and only when that window allows it |
| `sensitivity` | no | `public`, `private` (default), `sensitive` or `critical`. **Producer-declared** — see below |
| `category` | no | Per-event-type key used for recipient preferences and the settings catalog; defaults to the event type when absent |
| `context_url` | no | Deep link into the source scenario |
| `dedupe_key` | no | Producer-supplied dedupe key; collapses repeats from the same producer |
| `correlation` | no | String map (at most 1 KB) echoed downstream |

### Writing the title and body

A phone shows roughly 30 to 40 title characters and three body lines, and on
iOS every notification carries the installed app's icon and name, not the
source's. Write for that:

- **The title says what happened, specifically.** Lead with the outcome and,
  for a problem, the reason: "Agent interrupted: model at capacity", not
  "Agent interrupted"; "Commit message ready · Vrooli", not "Update".
- **Do not name the source scenario or Notification Hub.** The hub sends the
  title as written, and source names spend the space the outcome needs.
- **The body's first line says which one.** Several sessions, runs, or jobs
  run at once, so name the subject in human terms: a session's tab name and
  agent, a run's task title, a proposal's commit subject.
- **The next line says what to do or adds the detail** (the agent's question,
  the error, "Open the session to resume or retry"). Separate lines with `\n`.
- **Never send ids or raw keys** when a human name exists; fall back to a short
  id only when nothing else names the subject.

Examples in the tree: `web-console/api/session_notifications.go`,
`agent-manager/api/internal/orchestration/run_outcome_events.go`,
`git-control-tower/api/gct_notifications.go`.

### Producer-declared sensitivity

`sensitivity` is now stated by the producer rather than derived. The old path
inferred sensitivity from an incident `severity`, which only worked because
incidents were the sole producer. Generalizing to any producer means privacy
is a first-class fact the source declares, because only the source knows
whether a notification's body is safe to show on a lock screen or must stay
behind an unlock. An absent value defaults to `private`.

### Incidents become one producer, not the model

The legacy incident renderer (reading `check_id`, `severity`, `message`, and
related fields) is **demoted to a fallback**. It is applied only to events
that arrive without neutral `notificationFacts` — matched by the
operator-configured `incident.**` pattern described in
[`INTEGRATIONS.md`](INTEGRATIONS.md). Existing incident producers keep working
unchanged. New producers, including incidents once they adopt the neutral
shape, use `notificationFacts` and inherit branding, catalog, and preference
behavior for free.

## Source-Scenario Branding

Every notification carries its **source scenario** and a resolved display
identity:

```
{ name, icon, deep_link }
```

The display identity is resolved in this order:

1. **Declared** — the source scenario's `notifications` manifest block (below).
2. **Observed** — the source scenario id seen on the inbound event, used to
   derive a readable name and a generic icon.
3. **Generic** — a neutral fallback when neither is available.

The resolved identity is rendered in the UI timeline and embedded in the push
payload, so a recipient sees which scenario spoke and clicking deep-links into
that source scenario. Branding resolution never blocks delivery: an
unbranded source still delivers, just generically.

## Branding Manifest (optional `notifications` block)

A producer MAY curate its branding and catalog by adding a `notifications`
block to its own `.vrooli/service.json`. The hub ingests it at reconcile.

```jsonc
{
  "notifications": {
    "displayName": "Agent Manager",
    "icon": "…",              // icon reference rendered in timeline + push
    "deepLinkBase": "…",      // base URL for context_url deep links
    "categories": [
      {
        "key": "run_review",
        "displayName": "Run reviews",
        "description": "A run is waiting for your review",
        "defaultEnabled": true,
        "defaultMinUrgency": "normal"
      }
    ]
  }
}
```

| Field | Meaning |
|---|---|
| `displayName` | Curated source name shown in place of the derived id |
| `icon` | Icon reference for the timeline and push payload |
| `deepLinkBase` | Base used to resolve `context_url` deep links into the source |
| `categories[].key` | Category key matched against `notificationFacts.category` |
| `categories[].displayName` | Human label in the settings catalog |
| `categories[].description` | One-line explanation shown to the recipient |
| `categories[].defaultEnabled` | Whether the category delivers before any recipient choice |
| `categories[].defaultMinUrgency` | Lowest urgency that delivers by default |

The block is **entirely optional**. Its only job is to make the catalog and
branding *curated* instead of *derived*; nothing about delivery depends on it.

## Catalog — Observed Baseline + Declared Overlay

The settings catalog is what a recipient sees when configuring notifications.
It is built from two sources so that configuration works with zero producer
effort and improves when a producer opts in:

- **Observed (always).** Every inbound notification upserts its
  `(source_scenario, category)` pair into the catalog. A category a recipient
  has received is therefore always configurable, even for a source that
  declared nothing.
- **Declared (optional overlay).** A source's `notifications` manifest block
  supplies curated display names, descriptions, icons, and defaults for its
  categories. Where a declared category exists, its metadata and defaults take
  precedence over the observed baseline.

The observed baseline guarantees configurability; the declared overlay
guarantees quality.

## Preference Model

Preferences are **per recipient**, keyed by `(source_scenario, category?)`
— a source-wide rule when `category` is omitted, a category-specific rule
otherwise. Each preference carries:

| Field | Meaning |
|---|---|
| `enabled` | Whether this source/category delivers at all |
| `min_urgency` | Lowest urgency that is allowed through |
| `channel_override` | Optional: force a specific delivery channel for this source/category |

The delivery worker consults preferences **before routing**. A notification
that is muted (`enabled: false`) or below `min_urgency` is **suppressed with a
recorded reason** — it is never silently dropped, so the timeline and
delivery record always explain why nothing was sent. Suppression is a
delivery outcome, not an absence.

### Endpoints (contract level)

Three Connect endpoints serve configuration. They are described here at the
contract level; shapes live with the service definition.

| Endpoint | Purpose |
|---|---|
| `ListNotificationCatalog` | Return the catalog (observed + declared) a recipient can configure |
| `GetNotificationPreferences` | Return the recipient's current preferences |
| `SetNotificationPreference` | Upsert one `(source_scenario, category?)` preference |

These are served today as plain REST (`GET /api/v1/notifications/catalog`,
`GET|PUT /api/v1/notifications/preferences`). Both reads accept an optional
`source_scenario` filter (repeatable or comma-separated). Notification Hub's
own Settings page reads unfiltered; a producing scenario's settings page MUST
pass its own slug so it shows only the categories it declared. See
[scenario notification settings](../research/scenario-notification-settings.md).

## Centralized Delivery (and Why)

**Delivery stays in notification-hub.** There is one installed progressive web
app, one device registration, and one delivery engine. Other scenarios
participate only as clients of the hub API: they distribute **configuration and
presentation** (the branding manifest, catalog entries, preference reads and
writes), never delivery. A scenario is **never** expected to run its own push
stack.

Per-scenario delivery stacks are explicitly **rejected**, not merely
discouraged:

- **Web Push has no read receipt.** The push service does not tell the sender
  whether a message was shown. Two scenarios delivering the same notification
  in parallel therefore cannot dedupe against each other — there is no signal
  to dedupe on — so the recipient gets it twice.
- **One router keeps "deliver once" correct.** A single delivery engine holds
  the device registrations, the dedupe keys, the retry schedule, and the
  preference gate in one place, so "deliver this exactly once, to the right
  device, honoring the recipient's choices" is a property of one component
  rather than an emergent hope across many.
- **One subscription origin.** A Web Push subscription is bound to an origin and
  dies with it (see `tunnel-manager` in [`INTEGRATIONS.md`](INTEGRATIONS.md)).
  Centralizing delivery means one origin to keep alive, not one per producer.

The division of labor is therefore stable: **producers own facts and
configuration; the hub owns copy, branding resolution, the preference gate,
and the single act of delivery.**

## Maturity Path For A Scenario

| Tier | Status | What the scenario does |
|---|---|---|
| T0: publish | Available | Publish `*.notification.v1` facts (above) and declare categories in `.vrooli/notifications.json` |
| T1: own settings section | Web Console only; extraction planned | Show only its own categories in its settings page, filtered with `?source_scenario=<name>` |
| T2: own identity on web push | Research | Its installed PWA receives its notifications (own icon, name, tap target), still routed by the hub |
| T3: native push | Research; iOS blocked | Native app from `scenario-to-*` receives push via APNs, FCM, or a desktop channel |

T0 is all a scenario needs today. Research and plans for the later tiers:
[`../research/scenario-notification-settings.md`](../research/scenario-notification-settings.md)
(T1) and [`../research/scenario-owned-delivery.md`](../research/scenario-owned-delivery.md)
(T2 and T3). Until the hub ships per-device routing, a scenario must **not**
register its own push subscription with the hub: it would be delivered in
addition to Chimesail, not instead of it.

## Cross-References

- [`INTEGRATIONS.md`](INTEGRATIONS.md) — event subscription reconciliation,
  the `notificationFacts` field table, and the parallel decision contract
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership and retention
- [`FLOWS.md`](FLOWS.md) — ingress-to-delivery flow
