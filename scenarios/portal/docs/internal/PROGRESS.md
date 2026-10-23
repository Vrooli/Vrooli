# Progress — Portal

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log
| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-06 | Codex | done | Added Phase 9 CLI coverage for chat, messages, integrations, and search; refreshed Portal README and reference docs. |

| 2026-10-22 | codex | completed | Recovered and visually inspected the selected Horizon Home and shared Portal library from Drive. The technical preview differs in artwork, greeting, app-row treatment and text-button catalogue. android-tv/README.md now records the design gap matrix, proposed native tokens and first slice, state coverage, ownership and feedback qualification. One unchanged v0.1.4 owner build took 11.646 seconds; combined APK+NUL+AAB SHA256 and 169753-byte total verified and matched the earlier artifact. This is one provisioned-toolchain build sample, not a complete deployment or visual latency result. Existing AVDs are generic API36 google_apis 320x640 profiles, not TV. The ramp reported ADB enumeration timeout while exact TV state read succeeded; neither generic emulator readiness nor that discovery timeout proves TV preview readiness or disconnection. No deployment, TV foreground, default, network, permission or dependency changes. Evidence: private tv-capabilities-20261022T144942Z/design-readiness-20261022; existing Drive Experiments tab. |

| 2026-10-22 | codex | partial-qualification | Implemented native Horizon Home, editable app favorites, branded/searchable library, stateful return and bounded origin-scoped artwork/network reads. Installed verified v0.2.1 code 7, artifact fc2fab95f9cb51ebf1b8b4a3713a0afe41de15a0bde44b0d514622ec46ef85d6. Physical trial 1 passed Home/focus/TV apps/Back. Trial 5 (3f0e3263-ebf7-4b9d-9d96-fc07b34c0812) passed real Nooch Your day rendering, D-pad action, Back to focused Nooch/query and native Home; overall cancelled at 90s during second library entry. Recovery af7ee62f-8693-457c-b8f3-e65a171a8589 passed; saved HTTPS origin, stock Home, empty sessions and reverse cleanup verified. Earlier failed compression/routing/input-fixture trials retained. Fixed compressed HTML in shared api-base and missing Nooch router basename; direct route preserved. Nooch unit and Portal dependencies pass; existing Portal/Device Control unit fixture/coverage gates remain failed. See android-tv/README.md and Drive experiment report; no whole-product or TV-ready claim. |
## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Native TV preview implementation — 2026-10-22

Added an ordinary Android Views TV application under `android-tv`, with native Horizon navigation, installed TV app shortcuts, inputs, connection settings, and an on-demand system WebView. It has no HOME registration. Portal's ApplicationCatalogService reads the control-plane runtime inventory and resolves existing application URLs; it does not introduce a registry or start applications. Private LAN URL resolution and the shared UI proxy forwarded-host boundary were corrected and covered by focused regressions.

The native APK was built by Scenario to Android and installed through Device Control. Native Home and recovery to stock Home were observed. The Vrooli library fails on the physical TV in v0.1.2; host-side API success is not end-to-end evidence. The operator's 12:34 PM photo independently confirms this failure and the operator subsequently confirmed stock Home returned. Investigation remains active. No actual Vrooli experience, authentication, offline, sleep, or input-switch qualification is claimed.

Focused catalog, URL, CLI parity, proxy-origin, and native build preparation regressions passed. Test Genie `20261022-163320-300c2119` passed dependencies and failed unit; detailed attribution is being retained. This is a development preview, not launcher replacement readiness. Private experiment receipts are retained under the operator's `vrooli-experiments/tv-capabilities-20261022T144942Z/implementation-20261022` directory.

### Qualification checkpoint — 2026-10-22

The v0.1.3 ordinary APK displays native pages. An asserted connection-screen trial proves host UFW drops the TV's connection to the LAN Portal gateway; both native HTTP and WebView timeout. The existing HTTPS route is reachable by the TV and responds with HTTP 302, while a host-side probe received 403; these are different observations and neither is a catalog success. Inputs enumeration shows HDMI 1/2/3 and AV; switching was not attempted. Explicit sleep/wake verbs were accepted, but state remained on, so physical sleep is unverified. The preview was returned to stock Home and held leases released.

Portal unit run attribution: desktop tests hit the shared Unix-socket path limit; all 229 UI cases passed but branch coverage 84.78% missed the existing 85% threshold. Dependencies passed. No full-suite PASS is claimed. Actual Vrooli rendering and completed sign-in remain unverified.

### Final evidence and handoff — 2026-10-22

A native Home shortcut opened SmartTube, with matching captured frame and foreground state. Two Back presses remained within SmartTube, so return-to-preview behavior is unverified; Home recovered stock Google TV Home and the lease was released. The later HTTPS browser-opening attempt ended on native Home, not a verified sign-in page. No authenticated Vrooli render is claimed. Device Control's built UI now places frame capture and directional controls above detailed inventory; the deployed BAS capture confirms both are visible together.

The existing Drive refined-plan document's Experiments tab now contains the implementation findings and evidence link; its original tab was preserved. Curated ZIP includes protocols, compact receipts, native source, APK/AAB and checksums, excluding credentials, room photo and personalized recommendation images. ZIP SHA-256: `b67039df24256bd667f1262b398cab8b16fd5295ef2b37e0dba0ebaf1f23b620`. Full private evidence remains in the local experiment directory. A Swarm follow-up capture was refused by the exact `swarm.bookkeeping` / `swarm-manager:capture/intake` authorization gate; no grant was widened. The remaining findings are retained here and in Drive.

## Reverse-network qualification — 2026-10-22

The TV loaded Portal's actual catalogue and rendered Nooch's Your day screen using explicit loopback ADB reverse routes over its authorized wireless transport. Live run `b7d2101e-d15a-40e9-a118-987f341a4162` remains failed because the final exact Nooch accessibility assertion did not match; captured page rendering is confirmed separately. Earlier LAN UFW rejection remains valid for that path and does not imply general TV-to-host inability.

Device Control now owns bounded `reverse-tcp` setup/cleanup and exact-destination `reverse-tcp-remove` recovery. A typed local ADB server request is necessary because the CLI rejects explicit localhost addresses. Android 11 normalizes listed endpoint names; an early exact-name cleanup receipt was insufficient, corrected and reconciled by recovery run `9cc0de21-153e-409e-98ed-815e67457dd5` before final use. Corrected final cleanup verifies both routes removed. Stock Home/default component and empty closing sessions are confirmed; the preview's previous HTTPS origin was restored. No Assistant, firewall, VPN or microphone change.

Debug preview v0.1.4 supports only scoped debug localhost HTTP; native build artifact SHA-256 `a2262a69ab1771a94a71f9debf438d176cb93e2c50214f0f6fce3fc141cd0b7f`. Focused protocol/adapter/control/recovery/declaration and native build preparation tests pass. Test Genie `20261022-175436-c0f61748` programs pass; unit still fails on shared desktop fixtures and UI coverage, with its reverse declaration fixture subsequently repaired. No full-suite readiness claim.

Next work: reuse existing WebView/CDP observation to qualify semantics, choose a durable authenticated route, then test an explicit TV microphone path into existing Portal/AudioTools owners. Current APK has no recording permission or Assistant service. Hona's mesh repository does not establish the TV voice/role mechanism. Full private protocol, failures, primary sources and frames are retained under `vrooli-experiments/tv-capabilities-20261022T144942Z/reverse-network-20261022`. Work-record write again returned EOF without identity; repository/Drive records are the confirmed handoff.

Drive experiment addendum and 267,049-byte evidence ZIP verified in the existing folder. ZIP SHA-256 `6afd4dcad688f2be56332b2affd3764964d39b342415904c0ed76df4e91ac49e`; bundle link is retained in the private `drive-handoff.json`. Scenario to Android artifact checksum is for APK+NUL+AAB, independently verified; the APK alone is 87,292 bytes with SHA-256 `42947b4fc599d432fd0b8b10b8db9efbb08c5002d6fc23004815c7796308d12d`. The ZIP SHA256SUMS uses the correct individual APK digest.

## Home-first and remaining TV research — 2026-10-22

The operator requires direct home access with a saved secure-domain fallback for travel. Portal's existing TV README and decisions now specify separate routes, paired host verification, lightweight bounded home probing, visible remote sign-in and no automatic write replay. Reuse `api-base`'s existing `createScenarioProxyHost` in Portal's UI gateway; App Monitor is a source-backed precedent, not a running/qualified TV dependency. Gateway-aware application URL resolution and local HTTPS remain implementation work. No automatic fallback, DNS, firewall or VPN change was installed.

Remote observation `4d740007-2c70-4bbd-abde-12abdd00b610` passed its operations and captured the real Cloudflare Access email-code sign-in page. No email/code was submitted; authenticated Portal access remains unqualified. Fresh local navigation trials failed at selection and then catalogue readiness; their exact causes are not fully established and earlier Nooch rendering is not a fresh replay PASS. Both owned routes were restored, the previous HTTPS setting restored, preview closed and stock Home/empty sessions verified.

Device Control now exposes fixed read-only voice facts through InspectDevice, with nullable/unknown fields and verified-system-user scope. Live facts: microphone feature false; selected Katniss voice service; OEM policy input names without an audio-delivery claim; Assistant role query unknown. No audio was recorded. AudioTools reports healthy general API, available Whisper batch serviceability and unavailable pinned Kyutai streaming. WebView forward teardown remains unqualified; source-audit bug `knw-1792696363648007859` filed without physical CDP attachment.

Focused voice/inspection/reverse-flow checks and governed proto generation/verification pass. Test Genie `20261022-192128-ae97fc18` unit fails on desktop socket/AT-SPI/X11 fixtures and existing coverage floors (all 125 UI cases pass). The research, failures, sources and 151,781-byte evidence bundle are verified in the existing Drive report/voice documents. ZIP SHA-256 `cdf0a9685920e256fbad99d457bbcbb3e653d2b5219d3f14ebc0b0a141775b89`. Private protocol/receipts: `vrooli-experiments/tv-capabilities-20261022T144942Z/local-first-20261022`. Work-record `0f7955e9-9c2a-4804-8771-e425d21502e5` persisted; attribution verification is invalid, so it is not a trusted caller-proof claim. No comparable learning baseline is claimed.


## Daily Home, standby and reboot diagnosis — 2026-10-22

Stable signed daily 0.3.9/code17 is installed; preview remains code12. SmartTube
and YouTube launch/actual Home return pass. Native pause persisted disabled,
stopped foreground keep-alive and left Home on stock; resume restored native.
Screen-off readback confirmed stock underneath, and wake reached native without
Home/launch input. Native frame capture now works. These are bounded physical
checks, not streaming/playback certification.

Device Control repaired stale window focus overriding resumed activity, including
this TV's legacy resumed format. Independent foreground, accessibility and
bounded process-exit queries retain unknowns. The optional app-owned status
provider has fixed target-scoped routing and strict metadata parsing; Portal's
provider is DUMP-protected and read-only. Focused owner regressions pass.

Android11 retains accessibility package filters additively; setting null did not
clear the initial own-package XML filter. Code17 removes that filter and gates
paused delivery with eventTypes. Actual-resource regression failed before and
passed after; the signed owner build verified no content, gesture or key-filter
capabilities. The service's requested foreground lifetime, first/new-boot policy
and bounded wake checks need no host watchdog or Portal connection.

The authorized single restart changed boot_count 66 to 67 but first resumed
Google TV's profile chooser. Confirming its highlighted current profile did not
restore native. Permission persisted but runtime reported binding, not bound;
after exact native launch the app reported connected=false, foreground=false,
boot_seen=66. Full reboot activation is still unmet and its root cause is not
established. Independent Android Settings service recovery is being checked;
its two-pane preview and actual service page have different switch coordinates. Do not substitute native launch for startup proof. All failed and
successful receipts are preserved under the home-wake and home17-qualification
experiments beside the earlier private TV evidence. The build/replay/recovery
runbook is in android-tv/README.md; broader Device Control unit failures remain.


Closing recovery: Android's own service page and visible confirmation dialogs
successfully disabled and re-enabled only Vrooli; app metadata independently
confirmed allowed=false then true. Binding remained disconnected. One maintenance
reinstall of the identical signed17 bytes restored connected/foreground=true and
boot_seen=67. This restores the previously working activation, not a reboot fix;
no more reboot or build trials were run. Preserve the startup failure for the
next bounded binding investigation. No additional dependency or permission
capability was introduced; stock packages remain enabled.

Closing SmartTube entry, Home return and native frame pass after exact-artifact
recovery. The TV is left in native Home; full reboot activation remains open.
