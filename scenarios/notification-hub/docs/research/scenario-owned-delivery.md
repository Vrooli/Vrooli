# Scenario-Owned Delivery (Research)

> **Status:** Research, 2026-10-28. Nothing in this document is built. It
> extends the design in
> [`../concepts/ALTERNATE_DELIVERY_CHANNELS.md`](../concepts/ALTERNATE_DELIVERY_CHANNELS.md)
> and keeps the rule in
> [`../concepts/NOTIFICATIONS.md`](../concepts/NOTIFICATIONS.md) §"Centralized
> Delivery (and Why)". Web facts marked **[unverified]** need a device spike
> before anyone builds on them.

## Question

How can a scenario's notifications move from "delivered through Chimesail" to
"delivered under the scenario's own identity"? That means its own icon, name
and tap target, and richer presentation. The first step is installed PWAs
(iOS, Android and desktop web push). The later step is native apps built by
`scenario-to-ios`, `scenario-to-android` and `scenario-to-desktop`.
Notification Hub must remain the easy default.

## Answer in one paragraph

Keep the hub as the **only router, preference gate and dedupe authority**, and
make *where the push lands* a property of each subscription instead of a
separate delivery stack. Each subscription row gains a **face**: either `hub`
(Chimesail) or `scenario:<slug>`. A scenario's own installed PWA subscribes
with the **hub's VAPID public key** and sends the subscription to the hub
through a same-origin facade. For a notification from scenario S, the hub
delivers to S's face subscription on every physical device that has one. It
delivers to Chimesail on every other device, so each device gets exactly one
copy. A face subscription *shadows* the device's Chimesail subscription and
never replaces it, so falling back is free. Native apps (APNs, FCM, Electron)
are further face kinds behind the same `deliverTarget` seam. The one new idea
the hub needs is a **physical-device identity**. Today it has none for web
push: the "device" row is the page origin. Without that identity, "one copy
per device" cannot be stated, let alone enforced.

## 1. The delivery model today

### Registration and VAPID

- The hub owns one VAPID key pair. The private scalar is stored through
  api-core storage (`api/internal/push/webpush.go:122-140`). Only the public
  key is published, in `ListDevices.vapid_public_key`
  (`api/handlers/recipients/module.go:82`; `api/internal/hub/service.go:351-363`).
- Chimesail's service worker subscribes and calls
  `RecipientsService/RegisterPushSubscription` with the owner session cookie.
  It also re-registers on `pushsubscriptionchange` (`ui/public/sw.js:117-139`).
- The handler authenticates the subject. It then accepts `origin` **as the
  client supplied it, without validation**
  (`api/handlers/recipients/module.go:41-50`).
- `RegisterPushSubscription` upserts a **device named after the origin** and a
  single `web_push` channel address for that device. It then upserts the
  subscription keyed `(recipient, endpoint)`
  (`api/internal/hub/service.go:1229-1247`; `api/internal/recipients/schema.sql:8-36`).
  The device has the natural key `(recipient, name)`
  (`api/internal/hub/recipients.go:102-140`).

**Consequence:** every phone, laptop and browser profile that installs
Chimesail collapses into one `devices` row whose name is the Chimesail origin.
Fan-out still works, because `channelTargets` joins every `push_subscriptions`
row that has that `device_id` (`api/internal/hub/recipients.go:43-60`; test
`service_push_fanout_test.go:35-60`). But the hub has **no notion of a
physical device** for web push.

### Routing, dedupe and preferences (`Process`, `service.go:772-948`)

The worker processes a notification in this order:

1. Linked ask resolved, then suppress.
2. Scheduled hold.
3. Digest collapse (low urgency only) (`:810-831`).
4. Dedupe-key collapse and saturation cap, keyed on `requested_by + dedupe_key`
   (`:832-851`).
5. Quiet window (`:852-858`).
6. Per-`(source, category, scope)` preference gate (`:859-861`, `:958-975`).
7. Target selection: all channel addresses, or failing that all live
   subscriptions (`:862-874`).
8. Delivery to **every** target with retries (`:892-943`).

Dedupe is about *notifications* (collapsing repeated events). Dedupe across
*targets* does not exist. The model is "send to every registered target".

### Presentation

- `deliverTarget` builds the `PushMessage{Title, Body, Tag, URL, Icon}`
  (`service.go:55-57`, `:992-1006`). The title is prefixed with the source
  display name, because iOS shows Chimesail's icon whatever icon is sent
  (`service.go:181-219`). The icon is the source's PWA icon at its tunnel
  base URL (`service.go:221-281`).
- The wire payload is `{title, body, id, url, icon}` (`webpush.go:165-171`).
  The sender sets `TTL` but no `Urgency` or `Topic` header
  (`webpush.go:207-218`).
- Chimesail's service worker maps that payload to `showNotification`
  (`sw.js:71-84`). A tap stashes a pending-nav URL, because iOS ignores
  `openWindow` and `navigate` URLs from a backgrounded home-screen app
  (`sw.js:11-31`, `:86-115`).

### What must change for per-subscription face routing

| # | Change | Where |
|---|---|---|
| 1 | Add `face TEXT NOT NULL DEFAULT 'hub'` to `push_subscriptions` (`hub`, `scenario:<slug>`, later `apns:<bundle>`, `fcm:<app>`, `desktop:<slug>`). | `recipients/schema.sql:25-36`, `ensureAdditiveColumns` (`service.go:329`) |
| 2 | Add a **physical-device key** (`install_group`) on each subscription, so that a scenario-face subscription and a Chimesail subscription on the same phone are known to be one device (see §1a). | Same table; the `devices` row stops being "origin" |
| 3 | Make `channelTargets` source-aware: `channelTargets(ctx, recipient, sourceScenario)`. For each `install_group`, pick the healthy `scenario:<source>` face if there is one, otherwise the `hub` face. Never pick both. | `recipients.go:43-60`, `Process` `:862-874` |
| 4 | Bind the face on the server side. Look up the registration's origin against the scenario's tunnel host (`discovery.ResolveExternalURLForNotice`, `packages/api-core/discovery/external_url_tunnel.go:49-72`) and set `face=scenario:<slug>` from that. Do not trust a client-supplied face. As a bonus, this fixes the unvalidated `origin`. | `RegisterPushSubscription` handler/service |
| 5 | Make the payload depend on the face. A scenario face does **not** get the "Source —" title prefix, because the OS already shows the scenario's name. Its URL is relative to the scenario's own scope, not a hub ask path. | `deliverTarget` `:992-1006` |
| 6 | Same-pass fallback. If a face send returns `Gone` or exhausts its retries, remove or mark the face subscription and deliver to that `install_group`'s `hub` subscription in the same `Process` pass. | `Process` `:907-943` |
| 7 | Extend the payload with `badge`, `image`, `actions`, `app_badge` and a declarative-push envelope (§4). Add the `Urgency` header. | `webpush.go:165-218` |
| 8 | Show faces per device in Settings and in `ChannelsStatus`, so the operator can see "iPhone: Aquila app; laptop: Chimesail". | `ChannelsStatus` `service.go:1258` |

### 1a. The physical-device problem (the crux)

Web Push gives the sender **no device identity**. Two endpoints on one phone,
one for Chimesail and one for the scenario's app, look unrelated. On iOS the
home-screen apps do not share storage with each other or with Safari
([WebKit bug 181849](https://bugs.webkit.org/show_bug.cgi?id=181849)). So a
client-minted device id cannot be shared across apps either.

If the hub simply sent S's notifications to "all S faces, else all Chimesail
subscriptions", a laptop without S installed would silently lose S's
notifications whenever the phone had S installed. The alternative, "all S
faces *and* all Chimesail subscriptions", produces duplicates on the phone.
Both rules are wrong.

Candidate pairing mechanisms, cheapest first:

1. **Operator confirmation at enable time (recommended for the first version).**
   When the scenario's PWA enables push, the facade returns the recipient's
   Chimesail subscriptions. Each one is labelled with a device label derived
   from the user agent ("iPhone · iOS 18.6 · registered 10-20"), and the
   best UA match is preselected. The operator confirms "deliver here instead
   of Chimesail on: [this iPhone]". The hub then stores the shared
   `install_group`. This is cheap and explicit, and it suits a single-operator
   system.
2. **Push-tap pairing.** The hub sends a visible "Tap to pair <App> on this
   device" push to each Chimesail subscription. The subscription whose tap
   lands with the pairing nonce is the same device. This adds no UI choice,
   but it is clunky and costs a visible notification.
3. **Heuristic (UA plus push-service host plus time).** This is fine as the
   *default* inside option 1. It is not safe on its own: two iPhones with the
   same iOS version look the same.

When no pairing exists, an unpaired face subscription is its own
`install_group`. It receives S's notifications, and Chimesail on that same
phone also receives them. That is a duplicate, so the UI must not allow an
unpaired face to become active while Chimesail is subscribed on what looks
like the same device. The alternative is to let the operator choose "this
device has no Chimesail".

## 2. Origins, manifests and service workers

**Each scenario is its own origin on the phone.** tunnel-manager exposes one
route per subdomain at `https://<subdomain>.<domain>`
(`scenarios/tunnel-manager/docs/concepts/DATA.md:65-69`;
`docs/internal/DECISIONS.md:26`). Cloudflare Access gates each hostname with
its own audience (`docs/concepts/GATED-UI-AUTHENTICATION.md:9-35`). Option C
is therefore **per-origin**. Several manifests on one origin are not needed,
and the iOS behavior for same-origin multi-install ([WebKit
13878](https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/))
does not matter to us. Each origin has its own permission, its own
`PushManager` subscription and its own notification identity on every
platform.

**Every scenario already ships a manifest and a service worker.** The
react-vite template creates `ui/public/sw.js` (an app-shell cache only, with
no push handler: `templates/scenarios/react-vite/ui/public/sw.js:1-34`). It
registers that worker in production (`ui/src/main.tsx:20-22`) and links a
manifest with `"id": "."` and `"display": "standalone"`
(`ui/index.html:10`; `ui/public/public/site.webmanifest:2-8`). 74 scenarios
have a manifest, and about 40 have `sw.js`. **Only notification-hub's worker
handles `push`.**

**There is no shared push service-worker package. One is needed:**
`packages/ui` or a new `@vrooli/push-sw`. Service workers are static files,
so the package should ship a versioned script that each scenario's `sw.js`
loads with `importScripts("./public/vrooli-push-sw.<ver>.js")`. Template
regeneration or a codemod adds that one line. The script must contain:

- a `push` handler that **always** calls `showNotification` inside
  `event.waitUntil`. iOS revokes a subscription after 3 pushes that show
  nothing ([WebKit commit
  108ccb4](https://webkit.googlesource.com/WebKit/+/108ccb442e40e9705d212e74c1f213e3b5a0ea05)).
- `notificationclick` with the pending-nav stash pattern lifted from
  Chimesail (`sw.js:11-31`, `:86-115`), plus the matching app-side hook that
  reads the stash on foreground.
- `pushsubscriptionchange` re-registration through the scenario's
  same-origin facade (not through hub cookies).
- an optional receipt beacon (§5, unknown U4).

**Registration path:** add a same-origin **notifications facade** in each
scenario's API. Generalize `scenarios/web-console/api/notifications_proxy.go:13-24`
into `packages/api-core` (it forwards the owner's `Authorization` to the hub,
and returns a clean 503 when the hub is down). The same facade serves the T1
settings component, so T2 adds no new auth surface. The hub derives the face
from the facade's scenario identity together with the origin check (§1 row 4).

**Origin lifetime:** the hub took a core-seed grant because "a push
subscription is bound to an origin" (`docs/internal/DECISIONS.md:31`). A
scenario face carries the same obligation. **A scenario is eligible for T2
only while its tunnel route is CORE tier.** A leased route that expires leaves
taps opening a dead hostname, and the worker cannot update or re-register.
The hub should treat "route not CORE or not exposed" as an unhealthy face and
fall back to Chimesail.

## 3. The native path

| Ramp | How it packages today | Push it would need |
|---|---|---|
| `scenario-to-ios` | Capacitor shell that wraps the scenario's web bundle in an Xcode project (`scenarios/scenario-to-ios/PRD.md:18`, `:26`). Signing material comes from secrets-manager (`:32`). Readiness ladder from "no account" upward (`:34`). **No Apple Developer Program enrollment exists** (`docs/internal/PROBLEMS.md:180-182`). A WKWebView shell **cannot use web push**. | APNs through `@capacitor/push-notifications` (device token). Needs the Developer Program ($99/yr), an `aps-environment` entitlement, and a .p8 token key (Key ID plus Team ID; one key covers every bundle id in the team). Delivery is `POST /3/device/<token>` over HTTP/2 to `api.push.apple.com` with `apns-topic`=bundle id, which Go `net/http` can do ([Apple](https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns)). Rich images need a Notification Service Extension (`mutable-content: 1`). A sender avatar needs Communication Notifications with `INSendMessageIntent` ([Apple](https://developer.apple.com/documentation/usernotifications/implementing-communication-notifications)). Both are native targets the Capacitor template would have to generate. |
| `scenario-to-android` | Capacitor shell first, in a pluggable shell family (`scenarios/scenario-to-android/docs/internal/DECISIONS.md:25`). A `project_kind=native` build exists (`docs/internal/PROGRESS.md:43`). Upload key in secrets-manager (`PRD.md:30`). Play ladder (`:32`). | FCM HTTP v1: a Firebase project plus a service account (OAuth2), with `google-services.json` in the generated project ([Firebase](https://firebase.google.com/docs/cloud-messaging/migrate-v1)). Android 13+ needs the `POST_NOTIFICATIONS` permission. For de-Googled devices, UnifiedPush through the hub's preserved **ntfy** resource blueprint (`docs/internal/PROGRESS.md` 2026-08-18 row) is a natural second transport. FCM can also carry iOS if an APNs key is uploaded ([Courier](https://www.courier.com/docs/integrations/push/firebase-fcm)), but that adds a Google dependency to iOS. |
| `scenario-to-desktop` | Electron, either a bundled runtime or a thin client to an external server (`scenarios/scenario-to-desktop/README.md`). Templates already expose `window.electron.notification.show` (`PRD.md:32`; `docs/guides/desktop-integration.md:101-107`). | Electron has **no Push API**. Its `pushNotifications` module is macOS-APNs only and needs signing ([Electron](https://electronjs.org/docs/latest/api/push-notifications)). The practical channel is a **live connection**: the running app holds an SSE or Connect stream to the hub, and the hub shows the notification locally through `Notification`. The face is healthy only while the stream is connected. Otherwise the hub falls back to Chimesail web push or to the host `linux_notification`/`macos_notification` senders (`api/internal/hub/transports.go:108-231`). |

**How the hub delivers to native tokens:** these are new face kinds in the
same subscription and target model. `apns:<bundle>` stores a device token and
the environment (sandbox or production). `fcm:<app>` stores a registration
token. `desktop:<slug>` stores a stream id. Each one is a new case in
`deliverTarget` (`service.go:992-1021`), behind narrow sender interfaces like
`PushSender` (`service.go:59-70`), and each maps "token invalid"
(APNs 410 `Unregistered`, FCM `UNREGISTERED`) to the existing
`SubscriptionGone` path (`service.go:925-931`). APNs and FCM credentials
belong to the hub, in secrets-manager, never in the scenario. Device pairing
(§1a) works the same way, because a native app knows its own token and can
run the same "replace Chimesail on this device" confirmation.

This is the "the hub still owns the subscription" branch of the open question
in `ALTERNATE_DELIVERY_CHANNELS.md:93-96`, with one difference. The
*permission* is the scenario's own (its app), while the *subscription record
and the send* stay with the hub. That combination gives full branding without
a second delivery stack. It replaces the "scenario receiver endpoint" design
in `ALTERNATE_DELIVERY_CHANNELS.md:44-59`, which would make each scenario run
a push sender. That is the thing NOTIFICATIONS.md rejects.

## 4. Platform facts (web research, 2026-10-28)

| Fact | Status | Source |
|---|---|---|
| iOS 16.4+ web push works only from a Home Screen web app. Each app has its own permission and is shown with its own manifest name and icon. Manifest `id` plus the user-chosen name let one site be installed more than once. | verified | [WebKit 13878](https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/) |
| iOS ignores the per-notification `icon`; the app icon is always used. | verified (developer reports) | [Apple forums 740688](https://developer.apple.com/forums/thread/740688) |
| iOS ignores `actions`; only the default action is shown. | verified (developer reports) | [Apple forums 726793](https://developer.apple.com/forums/thread/726793) |
| iOS `image`, `requireInteraction`, `renotify`, monochrome `badge` | assume unsupported | [unverified] |
| iOS honors title, body, tag, silent, data and the Badging API (`setAppBadge`). | verified | WebKit 13878 |
| **Declarative Web Push** (iOS 18.4, macOS 15.5): a JSON payload with `"web_push": 8030` and `notification.{title, navigate, body, silent, app_badge, …}`. It is shown even if the worker fails, so the 3-strike risk does not apply, and an optional `mutable` flag lets the worker rewrite it. | verified | [WebKit 16535](https://webkit.org/blog/16535/meet-declarative-web-push/), [WWDC25 235](https://developer.apple.com/videos/play/wwdc2025/235/) |
| No web mechanism shows a per-notification image on iOS; only a native NSE can. | verified (absence) | as above |
| iOS 26 changes to web push | none found | [unverified] |
| iOS revokes the subscription after 3 pushes that show no notification. | verified | [WebKit commit](https://webkit.googlesource.com/WebKit/+/108ccb442e40e9705d212e74c1f213e3b5a0ea05) |
| Some iOS subscriptions become invalid over time; handle 404/410 and `pushsubscriptionchange`. | verified | [WebKit bug 273063](https://bugs.webkit.org/show_bug.cgi?id=273063) |
| Android Chrome supports icon, monochrome `badge`, `image` (big picture), up to 2 `actions`, and `tag`/`renotify`. | partly verified | [Chrome actions](https://developer.chrome.com/blog/notification-actions), [MDN maxActions](https://developer.mozilla.org/en-US/docs/Web/API/Notification/maxActions_static) |
| Android: notifications inside a WebAPK's scope are attributed to the WebAPK (its name and icon), not to Chrome. | verified | [Chromium change](https://chromium.googlesource.com/chromium/src.git/+/f3267daa0067ce785e944e8f778346a17b3f35c7), [web.dev WebAPKs](https://web.dev/articles/webapks) |
| Android: several WebAPKs on one origin with overlapping scopes are ambiguous. Moot here, because we use distinct origins. | verified | [pwa-dev thread](https://groups.google.com/a/chromium.org/g/pwa-dev/c/e_TLLm3ZR1E) |
| Desktop: Edge 105+ and Chrome 152+ on macOS attribute installed-PWA notifications to the PWA's name and icon. Chrome on Windows | Edge/macOS verified; Chrome on Windows unverified | [Chrome blog](https://developer.chrome.com/blog/notification-attribution-macos), [Edge](https://learn.microsoft.com/microsoft-edge/progressive-web-apps-chromium/whats-new/pwa) |

**Implication:** on iOS, T2's gain is identity (icon, name, a tap that opens
the right app, per-app Focus and permission), not richness. Richness on iOS
needs T3. On Android and desktop, T2 also adds image, actions and badge.

## 5. Maturity tiers

The switch at each tier is a per-`(recipient, source)` preference,
`delivery_face ∈ {hub, scenario}`. It is stored with the existing
preferences (`api/internal/preferences/`), so the single preference gate
still owns it. With `delivery_face=hub` (the default), nothing changes. With
`delivery_face=scenario`, every paired device that has a healthy face for this
scenario gets the notification there, **instead of** on its Chimesail
subscription. Every other device still gets it in Chimesail. Notifications
are never sent to both apps on one device. Mutes, quiet hours, dedupe,
digests and sensitivity labels apply exactly as before, because they run
before target selection (`service.go:832-861`).

### T0: Publish (exists)

- **Scenario author:** emits `*.notification.v1` events and declares
  `.vrooli/notifications.json` (display name, icon, categories and their
  defaults). Examples: `scenarios/web-console/.vrooli/notifications.json`;
  loader `api/internal/preferences/manifest.go:15-22`, `:122-157`.
- **Result:** delivered in Chimesail with a "Source —" title. Android and
  desktop show the source icon (`service.go:221-260`). iOS shows the Chimesail
  icon.
- **Gaps:** none blocking.

### T1: In-scenario settings (partly exists, in web-console only)

- **Scenario author:** mounts a shared `<NotificationSettings source="…"/>`
  component and the facade route.
- **Shared work:** move `web-console/api/notifications_proxy.go` into
  `packages/api-core` (for example `notificationsfacade`), and move the
  settings tab UI into the component library.
- **Hub:** none.
- **Risk:** low. This is the prerequisite for T2, because the same facade
  carries registration.

### T2: Scenario-identity web push (not built)

- **Scenario author:** (a) keep the route CORE tier, (b) add one
  `importScripts` line to `sw.js`, (c) the settings component gains an
  "Install the app and deliver here" control, which runs `subscribe` with the
  hub VAPID key and the pairing confirmation.
- **Shared work:** a `@vrooli/push-sw` script (push, click with pending-nav,
  subscription change, beacon), an app-side pending-nav hook, and
  facade endpoints for `GetVapidKey`, `RegisterPushSubscription` and
  `ListPairableDevices`.
- **Hub:** the table in §1 rows 1–8: face plus install_group columns,
  source-aware target selection, server-side face binding, payload per face,
  same-pass fallback, a declarative-push envelope, and face visibility.
- **Fallbacks:** if the face is `Gone`, failing, the route is not CORE, or
  the preference is off, the device's Chimesail subscription is used. If no
  Chimesail subscription exists for that device, the notification is
  delivered elsewhere or ends `unroutable`, as today.
- **Docs to update first:** `NOTIFICATIONS.md` §Centralized Delivery (state
  that a face is a hub-owned subscription, not a scenario stack) and
  `ALTERNATE_DELIVERY_CHANNELS.md` (replace the receiver-endpoint mechanism).

### T3: Native push through `scenario-to-*` (not built; iOS blocked)

- **Scenario author:** a `notifications.native` block in the ramp config,
  saying which faces to generate (APNs, FCM, desktop stream).
- **Ramp work:** Capacitor push plugin wiring; iOS entitlement plus an
  optional NSE and Communication Notification targets; Android Firebase
  config and `POST_NOTIFICATIONS`; Electron stream client plus
  `Notification`. Each generated app registers its token through the same
  hub registration contract.
- **Hub:** APNs and FCM senders with credentials in secrets-manager, a
  desktop stream endpoint plus a liveness-based health check, and payload
  mapping (rich image URL → NSE `mutable-content`).
- **Blocker:** no Apple Developer enrollment (`scenario-to-ios/docs/internal/PROBLEMS.md:180`).
  Android FCM needs a Firebase project.

## 6. Risks, unknowns and the spike for each

| # | Unknown or risk | Why it matters | Cheap spike |
|---|---|---|---|
| U1 | **Device pairing.** Can the operator reliably tell their devices apart, and does the UA label suffice? | It is the whole dedupe guarantee (§1a). | Log the UA plus the push-service host for every current Chimesail subscription. Mock the "replace Chimesail on…" picker with real rows. **Half a day.** |
| U2 | **End-to-end iOS behavior of a second-origin face.** A scenario PWA on its own CORE subdomain, behind Cloudflare Access, subscribed with the hub's VAPID key, and the hub sending to it. Does it show the scenario's icon and name? Does `pushsubscriptionchange` re-register through the facade under an Access session? | This decides whether T2 works at all on the operator's phone. | **First spike (§7).** |
| U3 | Is the declarative push envelope backward compatible? (Chrome and Android ignore `web_push: 8030` and fall through to the worker's `push` event with the JSON.) | Lets one payload serve every platform and removes the iOS 3-strike risk. | Send one declarative payload to Chimesail on iPhone and on Android Chrome; observe both. **1 hour once U2 harness exists.** |
| U4 | **No read receipt.** A face whose permission was revoked in iOS Settings can keep accepting pushes (201) and show nothing. | Silent loss without fallback. This is the NOTIFICATIONS.md concern. | Have the push worker POST a shown-ack beacon (a per-subscription capability token, sent to a hub path that does not need the Access cookie). Measure ack rate and whether iOS allows the fetch in `push`. If acks stop, mark the face degraded and fall back. |
| U5 | Origin churn: a subdomain rename or a lease expiry strands the face. | Dead taps; the worker cannot update. | Rule: T2 requires a CORE route. Add a hub health check against the tunnel route tier. No spike, just policy. |
| U6 | The template `id: "."` resolves per origin. Is it stable across template upgrades? A changed `id` means a new app on iOS. | The operator would have to reinstall, and the old subscription goes stale. | Record the resolved manifest `id` in the face row. Warn when it changes. |
| U7 | **iOS ignores the URL on a tap** (pending-nav pattern). | The tap target is the selling point. | Covered by the U2 spike: tap opens the scenario app at the stashed route. |
| U8 | Native credentials and cost (APNs $99/yr; Firebase project). | T3 gating. | Already tracked by the scenario-to-ios readiness ladder. Build the hub's APNs sender against the sandbox only after enrollment. |
| U9 | Electron has no push. A stream-based face only works while the app runs. | Desktop T3 is "live only". | Accept it. The health signal is "connected", and falling back to Chimesail on desktop is correct behavior. |
| U10 | Android `image` and `actions` need hub payload support, and actions need an answer path (asks). | Richness on Android. | After T2, map hub asks to `actions` (max 2) and answer through the facade. |

## 7. First spike to run (U2, plus U3 and U7)

**Goal:** prove or disprove that a non-hub scenario PWA on its own CORE
subdomain can receive hub-sent web push under its own identity on the
operator's iPhone, and that a tap opens that app at the intended route.

1. Pick one CORE-exposed scenario with a working UI (for example
   `personal-planner` or `web-console`). Do this in a scratch branch of its
   `ui/public/sw.js`, or better a throwaway route. Add an inline `push` and
   `notificationclick` handler: Chimesail's `sw.js:71-115`, copied.
2. In that scenario UI, call `pushManager.subscribe({userVisibleOnly: true,
   applicationServerKey: <hub VAPID public key from ListDevices>})`. Print the
   subscription JSON.
3. Send one push to that endpoint with the hub's sender. Use a test-only CLI
   path or a Go test harness with `push.NewFromValues` (`webpush.go:90-108`),
   and **do not** register the subscription with the hub, which would cause
   duplicates today (§1). Send one classic payload and one declarative
   (`web_push: 8030`) payload.
4. Observe on the iPhone: the app name and icon on the banner and in
   Settings → Notifications, whether the declarative payload shows with the
   worker disabled, and whether the tap lands on the stashed route. Repeat on
   Android Chrome.
5. Record the pass/fail results in this document, together with the UA and
   push-service host (input to U1).

If step 4 shows the scenario's identity, T2 is real, and the next work is the
hub schema (`face`, `install_group`) together with the pairing picker. If it
fails, iOS T2 is dead, and the only route to scenario identity on iOS is T3.

## Sources (web)

- https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/
- https://webkit.org/blog/16535/meet-declarative-web-push/
- https://developer.apple.com/videos/play/wwdc2025/235/
- https://developer.apple.com/forums/thread/740688
- https://developer.apple.com/forums/thread/726793
- https://bugs.webkit.org/show_bug.cgi?id=181849
- https://bugs.webkit.org/show_bug.cgi?id=273063
- https://webkit.googlesource.com/WebKit/+/108ccb442e40e9705d212e74c1f213e3b5a0ea05
- https://developer.chrome.com/blog/notification-actions
- https://developer.mozilla.org/en-US/docs/Web/API/Notification/maxActions_static
- https://chromium.googlesource.com/chromium/src.git/+/f3267daa0067ce785e944e8f778346a17b3f35c7
- https://web.dev/articles/webapks
- https://groups.google.com/a/chromium.org/g/pwa-dev/c/e_TLLm3ZR1E
- https://developer.chrome.com/blog/notification-attribution-macos
- https://learn.microsoft.com/microsoft-edge/progressive-web-apps-chromium/whats-new/pwa
- https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns
- https://developer.apple.com/documentation/usernotifications/implementing-communication-notifications
- https://firebase.google.com/docs/cloud-messaging/migrate-v1
- https://www.courier.com/docs/integrations/push/firebase-fcm
- https://electronjs.org/docs/latest/api/push-notifications
