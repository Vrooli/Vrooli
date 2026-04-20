# Research Conclusion: Extract Reusable Domain Signal From Legacy Notification Hub

## Research Question
Which domain concepts from the legacy Vrooli notify subsystem (deleted in commit `cca5b23eed` under `packages/server/src/notify/`) and the existing `scenarios/notification-hub/` should be preserved as the canonical domain model for the greenfield rewrite — and which should be dropped because they encode technical choices we are abandoning (n8n, Postgres/Redis, web-only WebSocket assumptions, deep coupling to the legacy Vrooli DB)?

The output is a domain model document for the greenfield rewrite, not an implementation plan. Concrete proto schemas, SQLite DDL, and provider integrations are downstream work for `execute/notification-hub-greenfield-core`.

## Summary
<!-- TBD — round 1 in progress; will firm up once direction decisions land -->

Initial scan confirms the legacy notify code (1,870 LoC across `notify.ts`, `notificationSettings.ts`, `formatters.ts`, `types.ts`) and the existing Go scenario contain three layers worth distinguishing: (a) **portable domain primitives** (priority, lifecycle, contact channels, push device schema, rate-limit semantics) that should carry forward, (b) **policy patterns** (online-vs-offline routing, silent + always-persist, daily-limit fail-silent) that should carry forward as documented behaviors even though the underlying infrastructure changes, and (c) **Vrooli-coupled abstractions** (Label deferred-resolution, named `pushFoo` helpers, ModelType/SubscribableObject enums, `notification_subscription` table, chat-participant routing) that should be dropped in favor of event-driven subscriptions on `vrooli-events`.

## Methodology
- Read deleted source from `git show cca5b23eed^:packages/server/src/notify/{notify.ts,notificationSettings.ts,formatters.ts,types.ts}` (1,870 LoC total).
- Read existing scenario: `scenarios/notification-hub/PRD.md` (989 lines), `api/main.go` (928 lines), `api/processor.go` (541 lines).
- Cross-referenced against initiative `notification-hub-greenfield` orchestration summary (already-settled architectural decisions: greenfield, multi-tenant, ntfy backbone, vrooli-events subscriptions, SQLite + file-based, 4-level priority).
- Catalogued enums, schemas, routing primitives, lifecycle states, rate-limit mechanics, and i18n/template patterns from both sources.
- Identified divergences between legacy TS code, existing Go scenario, PRD, and initiative summary.

## Findings

### Finding 1: Legacy `NotificationCategory` enum (17 values) is product-shaped, not infrastructure-shaped
The legacy `NotificationCategory` type in `notify.ts:117-135` enumerates: `AccountCreditsOrApi`, `Award`, `IssueStatus`, `Message`, `NewObjectInTeam`, `NewObjectInProject`, `ObjectActivity`, `Promotion`, `PullRequestStatus`, `ReportStatus`, `Run`, `Schedule`, `Security`, `Streak`, `SystemAlert`, `Transfer`, `UserInvite`. These map to actual product surfaces of the old Vrooli web app, not generic notification categories.

The initiative has settled on event-glob subscriptions (e.g., `agent-manager.run.completed`) rather than a fixed category enum. The category concept can survive as an **optional analytics tag** carried on a notification record, but it should not be a closed enum. New scenarios (greenfield notification-hub serving any vrooli-events publisher) cannot extend a closed category list without code changes.

### Finding 2: `NotificationDeliveryMode` (online/offline routing) encodes a real policy worth preserving
`notify.ts:75-94` defines three modes: `default` (push only if user offline), `force_push` (always push), `prefer_websocket` (websocket if online, push if offline). The greenfield has no WebSocket assumption, but the underlying policy — *don't double-notify a user who is actively viewing the app* — is universal. The greenfield analog is "user has the app focused or is actively connected to the in-app feed" vs "user is offline / app backgrounded."

Mechanism translation: instead of a server-side `roomHasOpenConnections(userId)` check (which required Socket.IO state), the greenfield can rely on **last-seen presence** signals from the in-app feed connection (e.g., periodic SSE/poll heartbeat) or simply make this a **per-subscription preference** ("always push" vs "push only when away") and skip server-side presence tracking entirely.

### Finding 3: Push device fields are Web Push standard and directly reusable
`notify.ts:870-910` and the `push_device` Prisma model store `endpoint`, `p256dh`, `auth`, `expires` — the canonical Web Push subscription shape (RFC 8030 + VAPID). These should port verbatim into the greenfield SQLite schema. APNs and FCM device tokens fit alongside as additional `kind` variants on the same device table.

### Finding 4: Rate-limit semantics (Redis INCR + TTL + fail-silent) are sound but not portable as written
`notify.ts:333-389` implements daily limits via `redisClient.incr("notification:{userId}:{category}")` with a 24h `expire`, falling back to **silent** (suppress) on Redis failure. The fail-silent default is a deliberate spam-prevention choice (better to drop than to flood when the limiter is down). 

Greenfield equivalents in SQLite-only world:
- Counter table `(profile_id, contact_id, category_or_channel, period_start, count)` with periodic GC.
- Same fail-silent semantic when storage I/O fails.
- Add the **cost-based** dimension explicitly (legacy did not have it; PRD calls for hourly/daily/monthly + cost caps).

### Finding 5: The `<Label|Type:ID>` deferred-resolution pattern is clever but heavy
`notify.ts:559-654` (function `replaceLabels`) scans variable values for `<Label|User:123>` placeholders, then batch-fetches DB rows in the user's preferred language and substitutes localized object names. This avoided sending stale or cross-language strings into the queue.

This pattern requires:
- Direct DB access from the notification service (breaks portability).
- A `ModelMap` registry of model types and `display().label.{select,get}` resolvers per type.
- i18next runtime in the worker.

For the greenfield (portable, multi-tenant, no Vrooli DB access), the recommended replacement is **caller-supplied resolved strings**: producers call `i18n.resolve()` themselves before publishing the event, and the notification-hub treats template variables as opaque strings. This sacrifices the "queue is stale-safe across language changes" property but is consistent with the event-driven boundary.

### Finding 6: Existing Go scenario already implements a usable subset; mostly needs hardening, not rewrite
`scenarios/notification-hub/api/main.go` and `processor.go` already model `Profile`, `Contact`, `Template`, `Notification`, `NotificationRequest`, plus per-channel send (`sendEmail`, `sendSMS`, `sendPushNotification`, `sendWebhook`) with a worker pool. The PRD lists DND/quiet hours, frequency limits, and unsubscribe management as P0-but-not-implemented. 

The greenfield rewrite is justified by the **architectural** changes (drop n8n+Postgres+Redis, adopt SQLite+ntfy+vrooli-events, define proto contract), not by a need to redesign the domain model from scratch. Most domain primitives in the existing Go scenario are sound and should be preserved with adjustments noted below.

### Finding 7: Lifecycle state granularity differs across sources
- Existing Go scenario: `pending → processing → delivered | failed` (3 transitions).
- PRD spec text + research item description: `pending → queued → sending → sent → delivered | failed` (5 transitions).
- Legacy TS code: implicit — uses a queue (BullMQ/Redis) and DB record creation; no explicit state enum on the notification.

The 5-state model is more useful for observability (separates "in our queue" from "in flight to provider" from "provider acknowledged"). Recommend adopting the 5-state model for the greenfield even though it costs one extra DB write per transition.

### Finding 8: Routing primitives are Vrooli-domain-specific
`NotifyResult` (notify.ts:541-720) exposes `toUser`, `toUsers`, `toTeam`, `toOwner`, `toSubscribers`, `toChatParticipants`, `toAll`. These mix two concerns: **recipient resolution** (who) and **fan-out semantics** (one or many). In the greenfield event-driven model, the user has already declared interest via subscription glob patterns — the hub fans out to whoever subscribed, period. Team/Owner/Subscriber/ChatParticipant abstractions belong in the *publisher* (or in vrooli-events itself), not in the notification hub.

Concretely: drop these primitives from the hub's API. The hub exposes `enqueue(event)`, and subscriptions decide who gets notified.

## Limitations
<!-- TBD -->
- Initial round; not all source files fully analyzed (e.g., legacy email/push/sms queue subdirectories under `packages/server/src/notify/` have not yet been read in full).
- Scope of "domain model doc" output format unconfirmed — could be a single markdown doc, or could include skeleton proto files; this is decision **D4**.
- Confidence on Finding 2 (delivery mode policy translation) is medium; depends on whether the greenfield will have any active-presence signal at all.
- Have not yet investigated the unsubscribe-scope model (all/per-channel/per-category) in depth — deferred to round 2.
- Have not yet investigated DND/quiet-hours timezone semantics in depth — deferred to round 2.
- Have not yet investigated template variable substitution edge cases (HTML escaping, channel-specific content blocks) — deferred to round 2.

## Actions
<!-- TBD — finalize once readiness is at 2+ across dimensions and all key findings are documented -->

Tentative actions (subject to revision):
- **Update document** — Author the canonical domain model doc at `scenarios/notification-hub/docs/DOMAIN_MODEL.md` (path subject to D4 outcome) covering: profile/contact/device schemas, priority+lifecycle state machine, rate-limit semantics, unsubscribe scopes, DND model, template structure, subscription glob model.
- **Create backlog item** — If proto schemas are in scope (D4 = include proto), create `execute/notification-hub-proto-contract` to author `packages/proto/schemas/notification-hub/v1/`. Otherwise that work folds into `execute/notification-hub-greenfield-core`.
