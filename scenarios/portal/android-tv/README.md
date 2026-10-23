# Portal Android TV launcher

## Daily17 qualification — 22 October 2026

Daily `com.vrooli.portal.tv.home` 0.3.9 (code 17) is installed through the
stable signing identity. SmartTube and YouTube entry plus actual Home returns
pass. The native pause control opens Google TV, leaves Home there and stops the
foreground keep-alive; resuming restores native Home. A screen-off readback
showed Google TV underneath, and wake returned to the native launcher without
Home or launch input. The single restart incremented boot count 66 → 67, but its first foreground
was Google TV's profile chooser and Android reported the service waiting to
bind. Confirming the already highlighted profile still left Google TV open. Explicit
owner launch restored the native screen, but its status showed permission and
request true, connected/foreground false and boot_seen still 66. Reboot
persistence has not passed; do not infer it from standby. Android Settings
independently disabled and re-enabled permission, but binding still failed.
One maintenance reinstall of the exact same signed17 artifact restored
connected/foreground true and boot_seen67. That is recovery, not a reboot fix.
Do not depend on reinstalling from the host for daily startup. Closing recovery
confirmed SmartTube entry, actual Home return and a native frame; the TV was
left on the restored native Home. The full-reboot activation phase stays open. Preview remains installed as 0.3.4 (code 12).

Two concrete defects were repaired. Device Control's stale Settings window
pointer overwrote the correctly resumed legacy activity; assertions now prefer
resumed activity and retain independent window disagreement. Android 11 adds
accessibility package filters cumulatively: setting `packageNames = null` does
not clear a previously declared own-package filter. The service XML now omits
the filter entirely; pause gates event types instead of adding another filter.
A build-owned regression checks the actual resource and prohibited capabilities.
The service receives only window-state metadata and retains no screen text.

Native captures work in the latest qualification; historical failed captures
remain failed and establish no particular security mechanism. Read-only owner
queries distinguish configured permission, live binding, app-owned status,
activity/window facts and bounded process-exit history. The app-owned status
provider requires Android's `DUMP` permission, refuses writes and exposes only
Home metadata. No Portal URL, account data or private screen content is returned.

The immutable signed daily artifact is
`android-release:870f1d185fd3193f8c824d92ec326d7ab8e101992a7d87fdc282b56dc87a2abb`.
APK+AAB hash, public certificate continuity and compiled no-content/no-key/no-gesture
capabilities were independently verified. The passing SmartTube/YouTube Home
flow is saved in Device Control under context
`hisense:api30:daily0.3.9:home-override`; retrieve its exact version before replay.
The saved flow ID is `6d276722-a5d1-45dd-bcf4-aa3a83eb20db`, version `1`.
Replay it only on the exact device/context after the operator returns the TV:

```bash
device-control flow replay \
  --id 6d276722-a5d1-45dd-bcf4-aa3a83eb20db --version 1 \
  --device-id android-0df87078c18abfb8 \
  --context-key hisense:api30:daily0.3.9:home-override \
  --actor tv-home-qualification --json
```

Private protocols and receipts are under
`vrooli-experiments/tv-capabilities-20261022T144942Z/home-wake-20261022` and
`home17-qualification-20261022` beside it. Focused owner regressions and native
build checks pass; earlier broad Device Control unit failures remain unresolved.

## Standby repair target — 22 October 2026

Active-user runtime evidence marks the enabled override as disconnected/crashed.
The newest application exit is signal termination, not a recorded Java exception;
the sender and precise kill policy remain unknown. Use Android's standard
foreground-service lifetime while override is requested and its permission is
allowed, with a low-priority ongoing notification opening native Settings. Pause,
permission revocation and unbind must stop this keep-alive. No extra microphone,
screen-content, gesture or key-filtering permission is needed.

Add a bounded first/new-boot entry and deferred wake handling. Window-state
metadata is used transiently to keep one boolean: whether stock Home is the
current window. Receiving other window-state metadata clears that boolean so
a delayed wake callback cannot take focus from another app; never read or store
other apps' screen text. Each wake gets at most two deferred checks, not a
polling watchdog. A normal service reconnect in the same boot must not force
Home over another app. Native Home arrival acknowledges successful routing,
so repeated healthy Home presses do not trip the failure burst limiter.

Keep permission, live binding, effective Home and package-manager default
distinct in Settings and evidence. This target is not a declaration that the
new lifetime or startup behavior has passed physical acceptance.

## Earlier wake diagnosis — operator report, 22 October 2026

Daily 0.3.6 (code 14) was installed with the optional Home override. The operator
enabled the system service, confirmed a physical Home press stayed in Vrooli,
then reported that power off/on returned Google TV. Fresh active-user readback
still shows the service configured/enabled. That proves permission persistence,
not runtime binding or successful Home return from another app. Wake and reboot
activation remain unqualified. No stock packages were disabled.

The earlier setup was interrupted for normal TV use; its incomplete experiment
is retained separately. The resumed bounded diagnosis checks service binding,
missed wake/startup events, and Device Control's activity/window observations
before further launcher changes. Its Settings foreground readings conflict with
operator-visible screens, and failed capture is not proof of any particular
security mechanism. Do not promote those failed checks or claim true default
Home from the permission flag alone.

## Active Home override experiment — operator direction, 22 October 2026

The operator approved proceeding with the custom override first to get daily
Home working soon. Implement a daily-only, opt-in accessibility service that
routes solely on the previously observed stock Home activity's window event.
Other window-state metadata only clears its transient stock-visible boolean.
It launches the native Home alias, preserves system packages and Assistant, and
does not request screen-content retrieval, gestures or key filtering. Android's
accessibility settings own enabling/disabling the service. Settings must show
its actual connected/requested state and offer a persistent pause-and-open-stock
escape. A burst limiter pauses the override on repeated redirects instead of
creating an unbounded crash/relaunch loop. No host watchdog or Portal connection
may be required. Preview must not expose this service.

Qualify stock recovery first, then Home from native pages and normal TV apps,
sleep/wake and one reboot. Android may continue reporting Google TV as the
resolved Home; describe the result truthfully as an effective Home override,
not a changed package-manager default. This is a new bounded routing experiment;
the earlier code12 campaign and its failures remain unchanged. No package
disabling is included in this route. Physical acceptance is pending.

## Earlier code12 delivery checkpoint — 22 October 2026

Daily `com.vrooli.portal.tv.home` version 0.3.4 (code 12) was installed with
the stable signing identity. Physical checks cover native app entry, root Back,
the bounded recent-app row and both end destinations, restored focus, the gear
button and preserved recency through an in-place signed update. The operator
confirmed Netflix visibly open when its screenshot could not be decoded. These
checks establish app entry, not playback or every account/streaming path.

The separate preview package was updated from code 7 to code 12 on the TV.
Both packages then reported 0.3.4; preview entry, unchanged stock Home routing,
daily entry and retained daily recency passed in an independent bounded flow.
This proves channel isolation while stock Home is selected. Isolation with the
daily launcher actually selected remains a later activation check.

**Persistent activation is pending.** The operator selected Vrooli in Android's
Home dialog, but independent Home resolution and an actual Home-button test
still selected Google TV. One checked owner `home-set` request also returned
Android's `Success` while readback stayed stock; it was not retried or accepted
as successful activation. Read-only inspection observed Home priorities 2 for
Google TV, 1 for Setup Wizard recovery, and 0 for Vrooli. The Home role holder
could not be read reliably and remains unknown.

Both system packages remain installed at their original DEFAULT enabled setting
for Android user 0. That earlier experiment stopped before package disabling or
an accessibility override. Those are separate rollout choices, each requiring
its own bounded qualification and recovery evidence; neither is delivered by
the successful launcher build. Reboot, sleep/wake persistence and host-disconnected
Home qualification remain pending until an effective routing method is chosen.

Device Control now provides checked `home-set`, read-only `home-state`, a typed
one-shot restart request and checked `package-enabled-set`. The package setter
compares expected state and independently reads the result back. Its focused
tests prove exact DEFAULT restoration with fake device responses; they do not
prove physical restoration or authorize a system-package change. See
[the owner contract](../../device-control/docs/reference/capabilities.md#checked-home-selection-and-device-restart).

| Potential package-route change | Original setting | Exact recovery setting |
|---|---|---|
| `com.google.android.apps.tv.launcherx` | `default` (0), user 0 | `default`, expected `disabled-user` |
| `com.google.android.tungsten.setupwraith` | `default` (0), user 0 | `default`, expected `disabled-user` |

If that route is chosen, change one package at a time under the exact device
lease, inspect the outcome before proceeding, and keep recovery independent
of the launcher UI. Restore Setup Wizard first, then Google TV, and verify
actual stock Home routing. Do not uninstall, clear data, assume explicit
`enabled` equals the original setting, or retry an uncertain change. Qualify
Settings, normal TV apps, casting/network integration and boot before adoption.
An accessibility-based alternative would preserve these packages but needs a
new native implementation, explicit system permission, and its own input,
startup, failure and recovery experiments. Its effectiveness is unproven here.

Scenario to Android's scoped unit phase passes. Device Control's focused Home,
foreground, restart and checked-package regressions pass; its broader unit phase
remains failed. The readiness review is yellow and rules checks were unavailable.
This checkpoint does not certify whole-scenario readiness.

## Daily Home target — operator priority, 22 October 2026

The operator now wants this launcher active and persistent for daily TV use.
Reliable installed TV apps are the activation requirement; unfinished Portal
experiences may remain visibly experimental and improve while Home stays live.
Home uses a horizontal scrolling row of at most 12 shortcuts: recent launches
from Vrooli first, then pinned apps, then installed apps to fill remaining spaces.
The complete installed inventory remains under All TV apps. The last two cards are All TV
apps and All Vrooli apps. Settings uses a gear with no filled button background;
remote focus adds an outline and icon tint, and its accessible label is retained.
Persist recency and restore visible focus; do not
request usage-access permission to infer launches from other launchers.
This supersedes the earlier preview-only target for the next rollout. It does
not mean Home has already been selected or that activation checks have passed.

| Outcome | Acceptance boundary | Owner |
|---|---|---|
| Dependable native Home | System Home reaches native Home from every preview/library/WebView state; Back at Home stays there; app shortcuts and inputs remain usable without Portal, computer or ADB. | Portal android-tv |
| Usable TV apps | Exact native package/component launch, clear unavailable state, verified SmartTube/YouTube/Netflix/Prime entry and return, remote focus and persisted favorite order after restart. | Portal; Device Control supplies qualification evidence |
| Selected Home survives normal life | Verify actual OEM Home selection, sleep/wake and authorized reboot/cold start; retain stock launcher and a tested recovery route. Never depend on a host process continually reopening the app. | Android/OEM owns Home; Portal integrates; Device Control verifies |
| Sustainable updates | Stable signing identity, versioned immutable builds, distinct daily/preview channels or an equivalent verified isolation boundary, preserved preferences and a rehearsed rollback. Debug-to-release key migration is explicit. | Scenario to Android, secrets-manager and existing distribution owners |
| Professional project integration | Native lifecycle/navigation/app catalogue/connection components have clear boundaries; important regression cases run through existing build/test owners; docs, requirement evidence and reviewed changes match actual support. | Portal and adjacent owners |
| Experimental services cannot disrupt TV | Portal, WebView, authentication or host outage cannot block Home or native app launch; failures return to native navigation. | Portal android-tv and shared gateway |

The Home eligibility intent and system-role/default-app selection must be
implemented and qualified on this Hisense Android 11 TV. Prefer Android's
supported selection UI; discover an OEM-specific supported alternative only if
needed. Keep default-Home mutation, inspection and restoration generic in Device
Control if a new typed owner operation is required. Never put raw ADB in the
launcher or bypass a genuinely human-only system decision. Preserve Assistant,
stock launcher packages and existing account/network policy.

Release-core checks are bounded, assertion-bearing cases, not one long journey.
One passing command is not proof that an app opened; record its actual foreground
and visible/semantic entry. A cancelled long flow is not promoted. Exercise the
normal remote, then remove the experiment connection before claiming daily-use
independence. A Home app must recover when Android recreates its activity; an
always-running watchdog is not the persistence mechanism.

After activation, retain the daily Home while developing candidates separately.
Qualify updates and recovery before applying them. Portal home-first identity,
local TLS/auth, secure-domain travel fallback, full per-experience TV controls
and voice remain the next product work; none blocks a working native Home.
The existing Home/Portal design, no generated substitute branding, shared gateway
and owner boundaries remain in force.

## Build and iterate on the native launcher

The source belongs to Portal. Scenario to Android owns source snapshots, Gradle,
profile projection, signing and retained APK/AAB bytes. Device Control owns
device identity, leases, installation, input and qualification. Android owns
the selected Home; neither a host watchdog nor an ADB reverse route is needed
for installed TV apps. Keep release promotion separate from building.

From the repository root, build an ordinary preview with:

```bash
ANDROID_PROJECT_KIND=native \
ANDROID_SOURCE_REF="$PWD/scenarios/portal/android-tv" \
ANDROID_NATIVE_PROFILE=preview \
SCENARIO_TO_ANDROID_HTTP_TIMEOUT=360 \
scenario-to-android android build --json
```

Build the daily candidate with the stable existing signing identity:

```bash
ANDROID_PROJECT_KIND=native \
ANDROID_SOURCE_REF="$PWD/scenarios/portal/android-tv" \
ANDROID_NATIVE_PROFILE=daily \
ANDROID_SIGNING=required \
ANDROID_SIGNING_IDENTITY=vrooli/portal/tv-home-signing-key \
SCENARIO_TO_ANDROID_HTTP_TIMEOUT=360 \
scenario-to-android android build --json
```

Start the build owner with `vrooli scenario start scenario-to-android` if needed.
Never build this source tree in place. Every owner build runs the actual
dependency-free navigation model's JVM verification through Gradle `preBuild`.
Rollback must respect Android version ordering: retain the known-good owner
artifact/source and build a qualified higher-version restoration with the same
certificate. Do not uninstall the daily Home to force an older APK onto it.
Increment `version_code` and `version_name` in `vrooli-native.json` before an
in-place update. Preserve the daily signing identity; do not provision a new key
for each release. Metadata defaults to the ordinary preview profile so omitted
profile selection cannot accidentally produce a Home candidate.

The preview package is `com.vrooli.portal.tv`; the daily package is
`com.vrooli.portal.tv.home`. Both use the Java namespace `com.vrooli.portal.tv`.
Only the daily profile enables alias `com.vrooli.portal.tv.HomeActivity`.
Their preferences and installed package identities are separate. The build
result includes `native_profile`, package, version, signing identity and retained
paths. Verify those fields, the APK+AAB combined checksum and public signing
certificate before installation. A successful build is not device acceptance.

Use Device Control's existing short, lease-owned flows for install, launch,
exact foreground checks and redacted observations. Inspect the actual target
before clicking; semantic lookup is currently unqualified on this TV. Netflix
capture availability is separate from whether it opens normally. Keep account
screens and personal captures out of shared evidence bundles.

Choose Home through Settings only after the core TV-app and recovery checks.
With the optional override enabled, `Pause override · open Google TV` first
persists the paused state and stops its keep-alive, then opens the saved previous
Home. `Enable Home override` resumes routing; Android accessibility permission
is a separate state. Android accessibility settings remain an independent disable
route. On this TV the path is Settings → System → Accessibility → Vrooli
Home override → Enable. The right-hand two-pane preview and opened service
page have different positions; inspect the fresh frame rather than replaying
preview coordinates. Confirm the visible Stop/Use dialog and independently
read permission and runtime binding. The latest off/on recovered permission
but not the post-reboot binding failure. These controls do not change the
package-manager Home choice. Recovery
must also restore the system Home choice when that choice was changed.
Keep stock packages enabled. Android release downgrades are not a promised
rollback path: recover to stock, then build a corrected or prior known-good
source with a higher version code and the same key. Do not clear user data to
force a downgrade. Update preview independently and verify the daily Home and
its preferences stay selected before promoting a new daily build.

Portal origin, cookies and catalogue cache are optional. Daily builds use HTTPS;
the narrowly scoped `ANDROID_PREVIEW_CLEARTEXT_HOST=localhost` switch is for
debug preview experiments only. It does not implement a paired local-first
production connection or authenticated travel fallback.

## Delivered preview v0.2.1 — 22 October 2026

The ordinary preview now implements scenic native Home, four editable installed
app favorites, a visible Vrooli destination, native TV-app and experience grids,
search/favorites filters, applied product labels/artwork, bounded loading and
unavailable states, and focus/filter/scroll return state. The wallpaper brief
and exact generation prompt are in [the image brief](../docs/mockups/image-generation-brief.md).
One WebView opens the selected experience; local navigation stays native.

The installed debug artifact is
`android-debug:fc2fab95f9cb51ebf1b8b4a3713a0afe41de15a0bde44b0d514622ec46ef85d6`
(version code 7). The owner-built APK+AAB digest and size were verified before
installation. No dependencies were added to the Android project.

Physical evidence establishes Home/focus movement/TV apps/Back, then Nooch's
actual `Your day` rendering through the localhost Portal gateway, D-pad input,
Back to the focused Nooch wordmark card with the `Nooch` query preserved, and
Back to native Home. Trial `3f0e3263-ebf7-4b9d-9d96-fc07b34c0812` reached those
checks, then was **cancelled at its 90-second deadline** during second library
entry. It is not an overall passing flow. Recovery
`af7ee62f-8693-457c-b8f3-e65a171a8589` passed: saved HTTPS origin restored, stock
Home foreground, no sessions, owned reverse route restored. No food/account
writes, Home selection, Assistant selection or OS/network policy changes occurred.

The operator's gray icon was Nooch's published placeholder PNG. Portal now
suppresses explicitly marked template art and the card uses its text wordmark;
the operator explicitly chose this wordmark on 22 October. An approved
Nooch icon remains a future branding-owner task. Nooch launch uncovered
and repaired two real defects: shared gateway compressed HTML handling, then
Nooch's missing browser-router mount path. Direct phone URLs remain supported.

Validation: 32 affected api-base checks; ten Nooch route checks; seven Portal
catalogue/artwork checks; focused native navigation/ordering, CLI, gateway and
Device Control checks pass. Nooch Test Genie unit and Portal dependencies pass.
Portal and Device Control unit phases remain failed on existing desktop fixture
and coverage gates; no whole-scenario readiness claim is made.

Remaining qualification: repeated entry after the deadline, persisted favorites
editing/reordering on hardware, explicit offline/stale and large-text/reduced-
motion trials, complete remote web operation, isolated TV emulator, header
parity behind the privacy mask, daily home route with paired identity/TLS,
authenticated travel, voice and default-Home routing. These are separate from
one successful page render. Split future physical flows into focused ≤90-second
cases. Restart only the preview for a clean query; the device owner's text
operation appends and does not promise replacement of selected text.

Private receipts: `tv-capabilities-20261022T144942Z/polished-preview-20261022`.
The existing Drive plan's Experiments tab holds the shared evidence index.

## Earlier ordinary-preview delivery boundary — 22 October 2026

The operator authorized the proposed polished ordinary-app slice after the
design review. Work shape: bounded implementation and qualification, W3 against
the selected Horizon/library design and this README. Deliver native Home,
favorites, branded library, explicit connection/loading/error states and exact
Back/focus restoration. Keep the existing catalogue, build and device owners.
Scope includes adjacent owner fixes needed for repeatable observation; record
extensions and their evidence here. No HOME/Assistant selection, package removal,
new microphone permission, network policy, account or authentication-policy
change is included. A design review approval does not certify a working product.

Harness handoff: Work until the ordinary TV preview implements the selected
Horizon/library interaction and its affected checks pass, with native owner-build
artifacts, physical frame/focus/return/recovery receipts and remaining gates
recorded here and in the existing Drive experiment report. Read the selected
concepts, local-experimentation and docs/TESTING.md first. Keep going through
friction; step back and shrink or switch when evidence warrants it. Two fresh
review passes must find no material in-scope correctness or visual findings;
green tests alone do not prove design acceptance. Operator feedback is the
request for a professional finish and quick, proper feedback; the earlier test
app does not meet it. Missing decisions or credentials are reported precisely;
diagnosable implementation friction is not a permission request. Checkpoint
this bounded slice before expanding to travel auth, microphone or Home routing.

Adjacent owner extension: Portal's shared browser gateway is required to open
one app over the selected connection without separate per-app reverse routes.
Use the existing api-base engine and an opt-in catalogue resolution mode; retain
direct URL clients. Device Control's semantic assertions are read-only but the
existing retry classifier excludes them and drops failure-time frames. Qualify
bounded assertion retries and redacted failure frames at that owner before
repeated physical catalogue trials; input actions remain one-shot.

The operator independently reported a gray Nooch icon and a failed web launch
while the phone route worked. The published Nooch PNGs are placeholder artwork,
not a native decoding fault. The browser gateway also forwards gzip negotiation
to an HTML stream injector that reads encoded bytes as UTF-8 and retains the
encoding header. Fix that protocol defect in api-base's shared host engine and
prove streamed/cached gzip and Brotli behavior before another vertical trial.

Native Android Views own Horizon navigation and installed TV app shortcuts.
One system WebView is created only for a selected Vrooli experience. No HOME
intent filter, default launcher changes, accessibility service, boot receiver,
ADB dependency, or background server is part of ordinary use.

Build through Scenario to Android, with `ANDROID_PROJECT_KIND=native` and
`ANDROID_SOURCE_REF` pointing here. Its builder copies and fingerprints source
before using its existing Gradle, signing and artifact pipeline. Do not build
this project in place. `vrooli-native.json` is shared by the builder and Gradle.

The Vrooli library consumes Portal's ApplicationCatalogService Connect JSON
contract in `packages/proto/schemas/portal/v1/surfaces/surfaces.proto`. Runtime
inventory comes from the control plane; TV compatibility remains unverified
until tested. Selecting an application resolves its current browser URL.
Application sign-in remains in that application's WebView; this client never
mints credentials or bypasses authentication.

Use Settings to enter a Portal web origin. A `vrooli-portal://connect?server=…`
link opens a confirmation screen. It never silently changes the origin.
HTTPS is the default. LAN HTTP qualification must use an explicitly scoped
network security domain in the build; no global cleartext setting is enabled.

Qualification and known limitations are recorded in Portal's existing
`docs/internal/PROGRESS.md`. This preview is not a certified replacement Home.

## Design-readiness baseline before delivery — 22 October 2026

The selected visual source is Drive's **03 — Vrooli TV — Visual Directions**,
document `11W3kDDQ2MDPAyU3sWyJk0xrboyVMP56uBemWUT8iUL0`. Horizon Home
(`1UnkJDeZbD0rePTo1YMUf6s9UN8FyLP64`) and the shared Portal library
(`14uwqqWjS157hurs9PQoHcmsvI_XoX1kE`) are concept references. They are
generated images with illustrative branding, not screenshots or wallpaper assets.
The owner selected Horizon previously and now requests a professional finish.
The earlier v0.1.4 procedural scene, greeting and button-list catalogue were
technical preview treatments. The delivery status above supersedes those
implementation gaps; the selected visual target remains applicable.

| Surface | Target composition | Gap at design review / required proof |
|---|---|---|
| Home | Scenic still, restrained top navigation, clock and one familiar app row | Replace the placeholder scene with approved clean artwork; remove the unsolicited greeting. Keep Vrooli visible in the first row. |
| TV apps | Installed TV activities, actual package branding, explicit favorites editing | Current Home and TV apps both enumerate all activities. Favorites, persisted order and banner/icon fallback need implementation. |
| Vrooli | Shared destination from tile/tab; native three-column grid, authentic names/art and honest qualification | Current catalogue derives labels from scenario slugs and renders text buttons. Add metadata through Portal's existing catalogue owner. Aquila is web-console's product name; reuse its applied asset. |
| Inputs / Settings | Short native lists; clear connection route and recovery | Input enumeration exists; input switching, focus restoration and home/remote route selection remain unqualified. |
| Open experience | Full-screen app; cancellable loading, auth/unavailable states, exact return | A rendered Nooch page is evidence of rendering only. D-pad operation, authenticated travel use and return-to-card remain unqualified. |

Use the actual device's 960 × 540dp layout grid (1920 × 1080px at density 320).
Begin with 48dp horizontal and 27–30dp vertical safe areas, leaving additional
room for focus outlines. These are trial tokens, not measured overscan results.
Use 18sp labels, 16sp secondary text, a 28sp library title and 21–24sp clock.
Retain one immediate ivory focus outline with a dark keyline, optional 1.025×
scale and 80–120ms interpolation. Reduced motion removes scaling. Separate
selected-tab indication from keyboard focus. Keep geometry stable while loading.

The initial Home composition can fit four owner-selected installed apps plus
the library tile in five columns; avoid placing the library after every installed
utility. One possible fit is five 158dp tiles plus four 16dp gutters inside
the safe area with focus inset. Validate long labels, focused-edge clipping,
large font scale, bright/dark art and real sofa readability before freezing it.
Use the installed TV banner when suitable, then the actual icon and a readable
label. Never stretch or regenerate another app's accepted mark.

Design the following states before judging parity: focused/unfocused/pressed,
loading/cancel, no matches, offline/stale, sign-in required, app unavailable,
and return from a web experience. Voice states remain conditional on an actual
input path; the mockup must not imply a recording microphone. Keep zero Home
widgets as the proposed first slice; Ambient and automatic seasonal art can
follow later. These are scope recommendations, not additional owner decisions.

Keep the ordinary native Views implementation for the next slice. Extract the
theme/tile/navigation components, stable item identities, focus/history state,
and catalogue/connection state from the single Activity as those changes are
implemented. A Compose rewrite is not justified by the current evidence. Keep
daily navigation free of Device Control; keep network/auth logic out of paint
and focus handlers. No new repository or launcher-specific proxy is needed.

### Feedback loop: observed capability and missing qualification

One unchanged owner build of v0.1.4 completed in **11.646 seconds**, including
CLI/API overhead. APK and AAB bytes reproduce the previous combined artifact
identity `android-debug:a2262a69ab1771a94a71f9debf438d176cb93e2c50214f0f6fce3fc141cd0b7f`.
The checksum is SHA256(APK bytes + NUL + AAB bytes), not APK-only SHA256.
Total output is 169,753 bytes; source files remained unchanged. This is one
build sample with an already provisioned toolchain, not a cold-cache,
design-change, deployment or complete visual-cycle benchmark. Nothing was installed during that design-readiness benchmark; the later
delivery trials below are separate evidence.

Use three feedback stages:

1. Review the chosen mockups and real assets against a named state matrix.
2. Build through Scenario to Android, then use an isolated Android TV target
   and fixed catalogue fixtures for native focus/state captures. This target
   is still to be qualified: the two inspected local AVDs are generic Google
   API 36 images configured at 320 × 640, not TV profiles. The target owner
   reports emulator prerequisites ready while visual validation lacks a GUI
   session. Do not treat that result as TV-ready or boot an existing AVD as
   a substitute for a capacity-admitted TV profile.
3. Validate a small selected candidate on this physical Android 11 TV through
   Device Control's leased flows. Capture foreground, focus/semantics, image,
   failure-time diagnostics and exact recovery. Keep the stock launcher selected.
   A physical capture's status-bar privacy mask obscures the top 90 pixels;
   it is not evidence of clipped launcher navigation. Header parity needs a
   qualified native reference capture or an owner-supported scoped policy.

Before automating repeated visual trials, fix the observed text-append/filter
state friction, use exact semantic controls instead of assumed initial focus,
and capture failures before an assertion aborts the flow. Prefer bounded
condition readiness to longer fixed sleeps. Qualify WebView observation cleanup
at its Device Control owner before using CDP; tracked issue
`knw-1792696363648007859` remains open. Screenshot success alone does not prove
web navigation, frame latency, physical contrast or accessibility.

An Android ramp probe reported ADB enumeration timeout while an exact Device
Control read of this TV succeeded during this investigation. Preserve both
receipts; broad discovery failure is not proof the known TV is disconnected.
The TV was in stock ambient during that read-only design-readiness check.
Later delivery trials use separately recorded leases and recovery receipts.

Research record: private experiment
`tv-capabilities-20261022T144942Z/design-readiness-20261022` and the existing
Drive plan's Experiments tab. Google TV layout, focus and typography guidance:
[layouts](https://developer.android.com/design/ui/tv/guides/styles/layouts),
[focus](https://developer.android.com/design/ui/tv/guides/styles/focus-system),
[type](https://developer.android.com/design/ui/tv/guides/styles/typography).

For temporary development qualification, the debug builder also accepts the
literal `localhost` scope through `ANDROID_PREVIEW_CLEARTEXT_HOST=localhost`.
The origin validator accepts loopback HTTP only in a debuggable app. Use Device
Control's flow-scoped `reverse-tcp` operation for the required ports; the flow
must return a verified restoration receipt. This is a test route, not a daily
use dependency or proof of release networking. The APK does not request audio
recording permission or implement an Assistant service. Portal sessions and
AudioTools remain the intended host owners for future voice work.

## Local-first connection target (research, not yet implemented)

Keep a paired host identity and separate home and remote routes. These routes
are independent of Portal's Client / Local control / Automation capability
profiles. At home, choose the verified direct route before contacting the
remote domain. TV `localhost` denotes the television; it reaches the host only
while an explicit reverse tunnel exists. Ordinary home use needs a reachable
LAN/overlay origin or a host gateway. The current debug reverse route is a
qualification tool, not a permanent service.

Remember the user-approved home address, discover candidates through the
existing DNS-SD/Bridge discovery capability where useful, and use Android
network-change callbacks to invalidate stale probes. An SSID, private address,
service advertisement, TCP connection or HTTP 200 is not host identity. Check
the paired server identity and a valid Portal protocol response before using
an address. Start with a bounded probe of the known home route; contact the
saved HTTPS remote route only when home is unavailable and remote fallback
is enabled. Preserve a manual Remote action for travel. Identity/certificate
mismatch, captive-portal HTML, authentication-required and service-unavailable
are separate outcomes; a TLS failure must not downgrade to HTTP.

Show the chosen route and reconnect status. Keep a healthy route for the
current interaction. Do not retry chat submissions, uploads or tool commands
on another route automatically. Switch at an idle/reconnect boundary, with
existing session IDs and owner replay/resume rules. Keep cookies and device
credentials scoped to their validated origin; never copy Cloudflare cookies
to LAN addresses or add trusted-identity headers on the TV.

Use the same selected route consistently for catalogue resolution, scenario
pages, API calls and audio transport. The current URL resolver preserves
private-IP routes but cannot derive a sibling application from a two-label
`.local` hostname; its fallback is localhost. A stable home gateway or an
explicit canonical host-route model must address that case before `.local`
is offered as a working browser route. Bridge's current proxy accepts unary
Connect RPCs; it is not an HTML/assets/WebSocket gateway. App Monitor already
hosts `api-base`'s scenario proxy with browser-path and WebSocket handling;
reuse that shared engine in Portal's existing UI gateway for a single home
entrypoint, rather than creating a proxy implementation or depending on the
entire App Monitor administration UI. App Monitor was not running during this
research; its implementation is precedent, not a live TV qualification. The delivered preview opts into relative gateway resolution on its selected
Portal origin; existing direct URL clients retain their contract. Connection
settings still accept origin roots only. Child applications must support the
injected proxy mount path, as Nooch now does. Remote application exposure also
needs validation against Tunnel Manager rather than assuming every running
scenario has a public sibling subdomain.

For browser speech, plain HTTP on a LAN IP is not a secure context. Prefer
trusted local HTTPS, or qualify a native Android capture frontend that uses
the existing authenticated host audio service. A split-DNS HTTPS origin could
keep the same domain local at home and Cloudflare-backed away, but requires
explicit DNS, certificate, ingress and authentication work. An overlay/VPN is
another route option, not implicit authorization to install or enroll one.

The native RPC client deliberately does not follow login redirects. It needs
an explicit sign-in state and a cookie retry bridge into its own origin.
WebView permits HTTPS redirects; actual IdP/Cloudflare behavior must be tested.
Google OAuth cannot be assumed to support an embedded WebView. A system
browser or existing owner pairing/device flow may be necessary; none is
claimed implemented here. Microphone runtime permission, origin-scoped WebView
permission, audio input availability, provider health and Assistant-role/key
routing remain separate gates. Do not request all WebView permissions merely
to enable microphone capture.
