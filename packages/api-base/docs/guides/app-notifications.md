# App Notifications

A scenario's own installed app can show that scenario's notifications on a
device instead of Chimesail (Notification Hub's app). Notification Hub still
decides what is sent, when, and to which device; this package only gives every
scenario web server the browser-facing pieces. The delivery rule and the hub
API are in `scenarios/notification-hub/docs/concepts/NOTIFICATIONS.md`
§"App delivery".

## Producer and manifest parity

Every production category emitted for a scenario must appear exactly once in
that scenario's `.vrooli/notifications.json`; every manifest category must have
a producer. Quality Health's contract scan discovers Go API files that publish
typed `*.notification.v1` events, then checks that discovered emitters and
producer category declarations are represented in manifests. It checks
category-only manifests too; declaring categories does not turn on
`appDelivery`. This runs as part of the existing interop readiness gate:

```sh
(cd scenarios/quality-health/api && GOWORK=off go test ./internal/contractscan -run '^(TestNotificationManifest|TestNotificationProducer)' -count=1)
vrooli scenario test quality-health --phases unit
(cd scenarios/quality-health/api && GOWORK=off go run ./cmd/interop-graph --check)
```

When adding or removing a producer category, update its producer declaration
and manifest together. The gate discovers source/category declarations from
production API code instead of maintaining a producer roster. An emitter found
without a matching category manifest, or a category manifest without a
discovered emitter, fails the gate.

The current producer implementation covers Go scenario APIs. The readiness
check also blocks recognized non-Go publisher patterns until their language
has an equivalent source/category extractor. Before adding a publisher in a
new language or surface, extend the Quality Health contract scan and its
independent source fixtures; do not treat a passing Go-only scan as proof for
that new producer.

## Turning it on

1. In the scenario's `.vrooli/notifications.json`:

   ```json
   { "displayName": "Git Control Tower", "appDelivery": true, "categories": [ … ] }
   ```

2. In the scenario's settings UI:

   ```tsx
   import { AppNotificationSettings } from "@vrooli/react-component-library/AppNotificationSettings/1";

   <AppNotificationSettings />
   ```

3. Once at start-up (`ui/src/main.tsx`):

   ```ts
   import { initAppNotifications } from "@vrooli/react-component-library/AppNotificationSettings/1";

   initAppNotifications();
   ```

For category settings, use the shared typed browser client from
`@vrooli/api-base/notifications` and the optional React Query hook from
`@vrooli/api-base/notifications/react`. Supply a request function that uses the
scenario's same-origin facade and its existing owner authentication. The shared
client validates the REST response, distinguishes Notification Hub unavailability,
and filters scoped reads defensively; requests and response bodies are bounded
to 15 seconds and 1 MiB by default (both limits can be adjusted per client).
Oversized responses fail with a typed 502 error, while timeouts remain a visible
unavailable state. The client never selects credentials or a server origin.
`NotificationPreferencesPanel` from the RCL owns only category controls
and receives data and change callbacks from the caller.

Notification Hub's own Settings page intentionally requests the all-source
catalog. A producer's settings page requests only its source; its server facade
must pin that source from server configuration and reject writes that claim a
different source. Do not expose Hub credentials or accept a browser-supplied
source as authority.

Restart the scenario. Nothing else is needed: `createScenarioServer` and
`startScenarioServer` read the flag and mount the routes below.

The scenario must have an always-on (CORE) route in tunnel-manager. An
installed app is bound to its web address; the hub refuses app delivery for a
temporary address and the panel explains why.

## What the server mounts

When `appDelivery` is true (or `appNotifications: true` is passed), the server
mounts `/vrooli-notify` before its body parser:

| Route | Purpose |
|---|---|
| `GET /vrooli-notify/sw.js` | The shared push worker. Its scope is `/vrooli-notify/`, so it never controls a page and leaves the scenario's own `sw.js` untouched. |
| `POST /vrooli-notify/rpc/{GetAppState,EnableApp,ConfirmApp,DisableApp}` | Forwarded to Notification Hub's `AppDeliveryService`. |
| `GET /vrooli-notify/catalog` | This scenario's notification types. |
| `GET`, `PUT /vrooli-notify/preferences` | The operator's choices for them. |

Every forwarded call carries `x-vrooli-app-scenario: <serviceName>` from the
server's own configuration (a browser value is dropped), and the preference
routes force `source_scenario` to this scenario. Only sign-in headers are
forwarded (`Authorization`, Cloudflare Access headers, device id), never
cookies. The default target uses Vrooli's current lifecycle peer record and
bounded CLI re-resolution after a failure. For an explicitly configured
deployment target, pass the full URL through `appNotifications.hubBaseUrl`; do
not capture a local port in a long-lived process environment variable.

## Options

```ts
createScenarioServer({
  …,
  appNotifications: false,                       // force off
  // appNotifications: true,                     // force on
  // appNotifications: { scenario: "custom-slug", hubBaseUrl: "http://127.0.0.1:17736", timeoutMs: 15000 },
})
```

By default the flag in `../.vrooli/notifications.json` (relative to the UI
directory) decides.

## Why a separate worker

Most scenarios have customized their own `sw.js`, and nothing re-syncs template
files into existing scenarios. A second worker with a narrow scope runs beside
the app's own one without either knowing about the other; on iOS 26 a
subscription made from it shows the app's own name and icon.
