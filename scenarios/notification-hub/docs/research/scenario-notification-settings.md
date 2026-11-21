# Scenario notification settings

Updated: 2026-11-16. This document supersedes the 2026-10 extraction proposal.
The current contract and recipient behavior live in
[`../concepts/NOTIFICATIONS.md`](../concepts/NOTIFICATIONS.md) §Preference Model
and §Endpoints.

## Current boundary

Notification Hub remains the only delivery engine. Producer manifests declare
their categories; opted-in scenario servers mount the shared `/vrooli-notify`
app-delivery routes from `@vrooli/api-base/server`. Those routes derive the
producer slug from server configuration and overwrite preference writes with
that slug. Their default Hub address uses lifecycle peer discovery and bounded
CLI re-resolution; a deployment may provide an explicit `hubBaseUrl`.

The Hub Settings page manages every installed source. A producer settings page
manages only that producer's categories. Web Console's Go facade carries its
existing owner bearer/device/Cloudflare headers, pins catalog and preference
reads to `web-console`, rejects a write that names another source, and leaves
owner verification to Notification Hub. A browser-supplied source is never
authority.

Hub preference reads use the configured installation recipient. Writes must
resolve to that same recipient when one is configured; mismatched verified
bearer/cookie identities receive `403`. Cloudflare Access maps a verified human
to the configured recipient. This avoids saving a value under one subject and
then reading another subject's value. Reads retain the existing unauthenticated
catalog/bootstrap behavior; writes require a verified owner.

## Shared browser and UI implementation

`@vrooli/api-base/notifications` owns the REST DTOs, response validation,
source-filtered client, and pure category/default helpers. Its request function
is supplied by the caller, which owns URL resolution and credential headers.
`@vrooli/api-base/notifications/react` owns the React Query key, cache, optimistic
save, rollback, and refetch behavior. It does not select credentials. A source
list requests a scoped view; an omitted list requests the Hub's intentional
all-source view. Scoped results are filtered again in the client to protect
against an older Hub that ignores the query.

`NotificationPreferencesPanel` in the RCL `AppNotificationSettings` asset is
presentation-only. Callers pass translated group/row models and callbacks; it
does not fetch data or know credential policy. The same panel renders Hub's
multi-source settings, Web Console's one-source settings, and the app-delivery
category rows inside `AppNotificationSettings`. Session-scoped `scope_key`
overrides remain owned by Web Console's session mute control and are excluded
from global category rows.

The release candidate is `AppNotificationSettings@1.0.6-draft.1`. Its focused
catalog type/lint checks and component source preflight pass. Publication and
runtime adoption remain pending a cleared BAS story-capture route; the last
capture attempt returned HTTP 503 because lifecycle admission was closed.

## Manifest and UI disposition

Five production manifests intentionally declare `appDelivery`: agent-manager,
web-console, git-control-tower, system-monitor, and data-backup-manager. The
first three expose app settings; the latter two are headless producers by
design. Six unrelated scenarios had generic initializer/settings imports
without a manifest or app-delivery producer; those inert imports, controls,
test mocks, and boilerplate comments were removed with scenario-local
rationales. Research Desk's API decision courier is separate and remains
enabled. See the E03 execution record for verification and remaining gaps.

## Security and failure guarantees

- Web Console's producer-scoped facade forces the source on reads and rejects
  cross-source preference writes before forwarding.
- Notification Hub verifies write ownership, checks the configured-recipient
  match, validates JSON shape and field size, and caps preference bodies.
- App-delivery routes pin producer identity from server configuration and
  forward only the allowlisted auth/device/request headers.
- The shared client distinguishes a 503 from an empty catalog and rejects
  malformed success bodies. Its hook rolls an optimistic write back and
  invalidates the source-scoped query after settlement.

## Deferred questions resolved

- Keep preferences as the documented REST exception; notification event and
  app-delivery RPC contracts remain separately typed.
- Keep the per-producer facade. Direct cross-origin Hub calls would weaken the
  same-origin credential boundary and require CORS/token changes.
- Keep observed undeclared categories visible with a readable fallback label;
  producers should still declare curated metadata.
- Keep `channel_override` in the shared wire model but do not expose a control
  until the Hub defines supported channel choices.
