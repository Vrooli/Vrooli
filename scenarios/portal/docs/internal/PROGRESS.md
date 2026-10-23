# Progress — Portal

## Native interaction polish — 2026-10-23

Implemented cached native tabs, background package/artwork snapshots, idle page
preparation, freshness-gated catalogue refresh and asynchronous disconnect.
Explicit tab key handling prevents restored Home focus from changing pages.
App-return state preserves the originating tab; one centered loading overlay
spans resolution and WebView startup. Both libraries alphabetize favorites before
other apps and share a corner star. Removed redundant modes and TV-control copy.

Desired regressions and all four native owner checks pass. Daily29 measured
Vrooli handler average 1.734 ms / max 1.885 ms versus baseline 165.223 ms / max 199.025 ms.
TV apps measured 1.558 ms / max 1.869 ms versus 235.397 ms / max 853.489 ms. These are native
handler samples after startup settling, not end-to-end frame/GPU measurements.
Physical run da99e57d-a92d-4512-8f54-716678b4d156 returns from SmartTube to
TV apps. Run 9bd3cca4-5025-4cd3-b5db-aae47bf51c29 captures the single loader,
unified favorites, and Back to Vrooli with search intact. Status confirms Vrooli,
browser/loading false, and active Home override. Aquila's child URL error is
reported as knw-1792728989890967531; child compatibility is not claimed.

Daily27's foreground-only passing flow failed visual tab acceptance; daily28's
selection guard failed before the native return step. Both receipts are retained.
Final review repaired the new cache's clear-recents invalidation and removed an
obsolete focus guard in daily 0.4.12/code30. The signed owner build and four
independent native checks pass, as do digest and signer continuity. Installed
artifact `ec96821e680ab8286854373b22a660a508188068d7a98688bd1e99ca12f78772`.
Final smoke run 9abb1699-4d45-47eb-a2af-9c367e6b2607 passes, showing Nooch content
and Back to Vrooli. No user history was cleared, no reboot or Home permission
change occurred, and leases are released. Evidence is in android-tv/README.md
and interaction-polish-20261023. This does not certify every child-app feature.

## Setup-managed direct access implementation — 2026-10-22

Implemented optional control-plane `local_service_admission` with typed operator
bindings, setup-installed root policy, exact UFW rule management, live runtime UI
resolution/process checks, stale-rule cleanup and persistent revocation. Added
Portal's host requirement and focused `vrooli setup broker` so the operator need
not apply unrelated pending safeguards. Managed `make install` build/vet passed;
the updated CLI is installed. Saved the TV binding through the existing operator
state owner; no address or port is compiled into the capability.

Broker/safeguard regressions, registry/schema/configuration checks, setup isolation
and CLI argument checks pass. A read-only live probe verified Portal UI24965,
expected process UID/scenario/variant and home interface. Portal dependencies run
20261023-030047-37a4b222 passes; proto owner freshness passed. An unrelated runtime
invariant reported agent-session-containment live-test configuration; filed
Scenario QA knw-1792724261714121445, with no claimed regression attribution.

The initial automated install stopped at terminal-only sudo authentication. The
operator authenticated setup, then live qualification exposed UFW's need for
writable temporary storage inside the restricted broker service. Added failing
regressions for missing TMPDIR and hidden broker failure codes; both pass after
repair. `TMPDIR=/run/vrooli` reuses the existing writable directory while keeping
host temporary roots read-only. The operator authenticated the corrected setup
at 23:18. The live unit matches, persistent UFW configuration changed, and the
safeguard verifies the exact rule as already present. The operator sees Vrooli
apps over the daily launcher's saved LAN connection. Automatic endpoint-change
recovery, trusted HTTPS/certificate/DNS lifecycle and authenticated travel remain
unqualified. Evidence: android-tv/README.md and setup-admission-20261022.

Daily 0.4.5/code23 fixes the operator-reported P's cutoff: the server returns 128
apps, but native rendering previously stopped at60. Append batches near the
scroll/focus boundary without replacing existing cards; filtering searches all
entries and deep return focus remains reachable. The signed owner build,
independent policy/navigation/catalogue checks, digest and signer continuity
pass. Installed artifact
`5cb4e42989830a489bb03fa4858bd20ed38870514fdd722d180e322e946bb945`;
physical run 77281ad6-515b-4f71-be9f-80f337708159 crosses the old cutoff and
visually reaches Workspace Sandbox. The scrolled frames exposed viewport bleed
over the filter bar. Daily 0.4.6/code24 enabled ScrollView clipping;
signed owner build and digest/signer checks pass for artifact
`393f4f0fd2d1be60660f242c7c0831b6d2cc993bd3c0b13f32e9b5ad8d1dcb4d`.
the physical frame still shows overlap. Its combined run
30d61961-a5b1-4bd4-8a7a-8ceb7dd2eaa6 reaches the last card and renders Workspace
Sandbox's child HTML/assets, with a loading indicator still visible. It times
out during the closing capture after the Home command; overall cancelled,
not PASS. The Back-return frame shows Home, so deep return is unqualified.
Separate state confirms native foreground and no leases. After closing the
five-flow budget, a bounded one-flow visual repair checks parent clipping in
daily 0.4.7/code25. No reboot or Home permission change is involved.

Daily25 is now installed, artifact
`9044e6c3dd0811c369ab67d226c8535b9bcae1604eb737b34f375002b92f4920`.
Signed owner checks and independent digest/signer continuity pass. Physical
run 69eb4f1e-0b54-43de-b012-5e4ba6e77048 passes: the scrolled viewport no longer
covers the fixed controls; its closing frame and terminal assertion confirm
native Home. Sessions are released. The full 128-card traversal remains the
daily23/24 evidence; daily25's catalogue model is unchanged. No claim of complete
child API/socket behavior, deep Back restoration, endpoint-change recovery,
trusted TLS or authenticated travel is made.

## Favorite-first launcher refinement — 2026-10-22

Installed stable signed daily0.4.4/code22, artifact
`7a00f138603d26878467477d0d580667809250583394419e34e89266720504e9`.
Favorites now lead Home, with recent use ordering each group. Removed redundant
TV apps/Vrooli/Inputs copy and reorganized Settings into Home, Portal and
Preferences panes with a dedicated address dialog. The existing Home integration
and network preferences were preserved.

The desired-order regression failed before and passed after the fix. The signed
source's navigation, Home policy/resource and origin checks pass; artifact and
signer continuity verify. Physical frames confirm cleaner pages, Settings and
four leading favorites. Actual Netflix launcher use moved it first among those
favorites. Netflix and SmartTube foreground/Home-return checks pass. The initial
install/Home check failed on stock Home and required explicit native entry;
preserve it as an update activation limitation. The interrupted dialog-close
sequence did not capture Preferences, so it is excluded from Cancel/D-pad
qualification; a separate closing frame confirms Preferences without clearing
history. No reboot was performed. Closing Home status is connected and active,
native Home was left visible, and sessions are released. Passing terminal-assert
flow `615fa217-4686-44f0-b366-4f0104d4555b` v1 is saved for the exact
`hisense:api30:daily0.4.4:home-override` context. Private evidence:
`tv-capabilities-20261022T144942Z/launcher-refinement-20261022`.

Traced setup, host requirement declarations, parameterized safeguards, broker
policy and existing cloud ingress ownership. Replaced the superseded one-off
firewall instructions with a concrete comparison and recommended existing-gateway
HTTPS approach in android-tv/README.md. General local-service admission,
certificate lifecycle, home DNS, pairing and authenticated travel are proposed,
not implemented. No host networking or privileged state was changed. Full-reboot
Home binding remains an earlier open issue; no broad suite readiness claim.

## Daily launcher convenience update — 2026-10-22

Installed stable signed 0.4.3/code21, immutable artifact
`af5d0cdf41b89fcece739c648a0e6e2dd5b7ed4f46cfd05ab6276b2a0b0999d8`.
Added app-card long-press/Menu options, persistent favorite stars, exact Android
app-info/force-stop entry and system-confirmed uninstall entry for non-system
apps. Existing favorites survive updates. Native page tabs select on focus and
keep it during asynchronous library refresh. Inputs are sorted icon cards with
truthful standby/connected/disconnected metadata. Preserved Home override code
and accessibility capabilities. A scoped ignore exception exposes Java sources
previously hidden by the root `vrooli` executable-name pattern.

Portal now advertises its lifecycle-selected UI gateway with shared mdns-go.
Android NSD is bounded to eight seconds/eight services and serial resolution;
same-host port changes and HTTPS upgrades require a valid catalogue before
promotion. The UI gateway can serve TLS from configured certificate/key files
and fails closed on incomplete configuration. The operator explicitly chose
HTTPS preference with LAN HTTP fallback: a default-off preference permits only
the selected private IPv4 origin, with native/WebView policy and visible
unencrypted labeling. No automatic TLS/authentication downgrade or cookie copy.
Connection drafts survive activity resume, discovery and checkbox rerenders.

The daily app had no saved origin because its preferences are isolated from the
preview. Configured the operator-approved HTTP fallback and current home origin.
Fresh direct TV request timed out; host gateway independently returned HTTP200
with a valid 128-app catalogue. This is consistent with the earlier confirmed
UFW drop, not a fresh attribution proof or successful connection. Native DNS-SD
discovery/automatic port recovery, direct trusted local TLS, child pages and
authenticated travel remain physically unqualified. No firewall rule or ADB
reverse was applied. The current broker admits only Bridge's fixed port, and
passwordless sudo is unavailable. The later operator instruction superseded the
proposed manual administrator rules; see android-tv/README.md for the current
setup-managed options. Those rules were never applied.

Focused Go application/discovery checks and six UI gateway checks pass. Native
signed21 snapshot passes navigation, Home policy/resource and route policy
checks. Dependencies run 20261023-015135-45434432 passes after governed SDA tidy
repair; failed predecessor 20261023-014708-36075ae6 is retained. No broad unit
suite green claim. Physical long press, favorite add/remove and exact app info
were inspected; no force stop, uninstall or input-source switch was performed.
Final run b87a62f3-751e-4dd6-90e1-526fd18e8932 passed SmartTube entry/Home return.
Frames confirm focus stays on all three non-Home tabs through refresh, sorted
input cards and themed menu. The closing frame is native Home. Screensaver
receipts were retained and excluded from control-success claims; Home restored
the native form before the final direct connection test. Full reboot binding
remains the earlier open issue, with no reboot this session. Private receipts:
vrooli-experiments/tv-capabilities-20261022T144942Z/launcher-polish-20261022.

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
