# App Notifications

A scenario's own installed app can show that scenario's notifications on a
device instead of Chimesail (Notification Hub's app). Notification Hub still
decides what is sent, when, and to which device; this package only gives every
scenario web server the browser-facing pieces. The delivery rule and the hub
API are in `scenarios/notification-hub/docs/concepts/NOTIFICATIONS.md`
§"App delivery".

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
cookies. Notification Hub's port comes from `NOTIFICATION_HUB_API_PORT` or
`vrooli scenario port notification-hub API_PORT`.

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
