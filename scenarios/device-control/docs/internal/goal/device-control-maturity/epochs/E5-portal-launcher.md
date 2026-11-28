# E5 — Portal TV Launcher Stability and App Lifecycle

## Outcome
Diagnose and resolve evidenced custom launcher fallback, improve useful diagnostics, and let TV users operate Vrooli-owned apps through intuitive app lifecycle controls.

## Work
- Inspect Portal startup/default-home configuration, foreground changes, process exits, crashes/ANRs, logs and restart/boot behavior. Capture enough diagnostics to distinguish crash, OS policy, intent/configuration and explicit user navigation; do not assume crash.
- Fix evidenced root cause and add deterministic lifecycle/launch regressions plus controlled TV soak/restart qualification.
- Extend long-press/menu behavior from system app-info/force-stop/uninstall to Vrooli-owned app actions start/restart/stop using “app” terminology. When an owned app is a scenario and not running, offer start. Show available lifecycle state and handle failure/recovery truthfully.
- Keep host scenario lifecycle and repair in the control plane; Portal calls stable owned operations rather than implementing private process repair.

## Proof
Portal-owned tests and native build validation, exact UI/action screenshots where useful, current TV launcher foreground/crash evidence and repeatable own-app lifecycle outcomes. Do not uninstall or force-stop unrelated system apps.

## Current implementation and qualification gap (2026-11-02)

- Source inspection found the requested owned-app menu already in `scenarios/portal/android-tv/app/src/main/java/com/vrooli/portal/tv/MainActivity.java`: long-press offers Start/Restart/Stop, restart/stop require confirmation, a not-running owned app offers Start, and actions refresh the catalog and report unconfirmed/errors. Portal's lifecycle handler owns authentication/operation-scope checks and invokes the Vrooli control plane.
- `PortalClient` requires trusted HTTPS and a signed-in operator for lifecycle RPCs. The currently documented direct-LAN HTTP connection therefore permits browsing only; action success on the TV is not established. Do not weaken this boundary to make HTTP appear functional. The last recorded daily installation is 0.4.19/code37, but current installed version remains unknown: fresh Device Control inventory at 19:40 UTC found SmartTV Cast ID `google-cast:3bc013548481e743aa769f04a9a9ba0b` and an available Android TV Remote transport, neither of which provides screen/foreground access.
- Added a dependency-free action policy seam and owner-build regression for app action vocabulary, scenario self-exclusion, selection mapping and destructive-action confirmation. Scenario-to-Android native preview build produced debug artifact `android-debug:f5b3246c7038ac391d812746196db6564bc62ade528cb49f15e5ccf1ebfd1413` (package `com.vrooli.portal.tv`, version 0.4.19/code37). Portal API Go packages and Test Genie unit run `20261102-174054-978141dc` pass; unit maturity remains L1 due existing CLI advisories. The native artifact was not installed. Current launcher fallback/foreground evidence, trusted HTTPS operator action and TV soak are still required.

## Follow-up — current Android TV Remote route (2026-11-02)

- Fresh Device Control inventory and exact `device describe` at 18:59:51 UTC report SmartTV Cast ID `google-cast:3bc013548481e743aa769f04a9a9ba0b` available. The Cast endpoint is `192.168.1.158:8009` and exposes media/property/sensor, with no input, screenshot or semantic tree. The separate Android TV Remote mDNS endpoint is `192.168.1.158:6466`; the owner reports it available with input/media/pairing and no screenshot/semantic capability. The earlier 18:40 handoff mislabeled the Cast port 8009 as the Remote endpoint; current evidence does not establish why prior Remote discovery was absent.
- A direct lease-owned `HOME` key over `mdns` returned success at 19:00:41 UTC, audit `dab954cb-10c4-46dd-aa19-294f46de5622`; the lease was released immediately. Treat this as Remote transport dispatch evidence only. The post-action Cast state still reports empty app/player values and cannot report input or physical power; no screen/foreground readback proves that TV Home became visible.
- A separate bounded five-second Device Control DNS-SD browse found four IPv4/IPv6 rows for `_androidtvremote2._tcp` on the same Remote identity, marked `paired=true` and `pairing_available=true`; it returned no ADB service row during that browse. This confirms current Remote pairing for this observation, not ADB being disabled or the cause of the earlier discovery gap.
- The Remote transport availability does not qualify Portal's trusted-HTTPS lifecycle actions. The built preview 0.4.19/code37 remains uninstalled, the direct-LAN HTTP route remains unsuitable for operator actions, and no current TV launcher or Portal app state is verified. A screen-bearing observation route and trusted signed-in Portal action path remain necessary.
- Memory work records: `26ca4404-2332-499d-8437-beee9751fdb1` (input dispatch), `c3d96a09-3199-40db-9452-35f503961aef` (paired browse), and `d8191cfd-8e45-43ff-8631-6b485b94876a` (service-type correction).

## Follow-up — app-management guidance (2026-11-03 UTC)

- Portal TV Settings now explains an unset address, HTTP browse-only use, HTTPS certificate verification before sign-in, an in-memory token that may need renewed sign-in, and lifecycle permission. Policy regressions cover all four states without claiming a trusted connection, valid session, or authorized action. Two reviews caught and corrected scheme/trust and token-state overclaims; two post-fix reviews found no material defect. Owner preview build produced `android-debug:2919d8d67b362fe1c09528a8cbdb2c976721d1c6c0d38b6debfb8da9c1dd67f4` (source digest `b6a4192a46d2e09fcf9a319857f76af3290418169e63c0c4c1d9fcd559390064`); Portal Test Genie `20261103-011957-e3798407` passed unit/docs. Docs L0 `broken_command_snippet` finding remains (79 warnings, 14 infos). APK not installed; live TV foreground/app-action qualification remains open. Memory `5119b363-ce70-403f-bc59-a43c59ebe302`.


## Follow-up — fresh preview build and current transport inventory (2026-11-02)

- Current owner inventory at 19:40 UTC reports Moto G7 `android-6affcac615a281a2` / serial `ZY226TG2BM` available, with live state screen-off/locked and foreground `com.motorola.launcher3`; the state read reports USB despite the inventory's wireless transport selection. Galaxy A03s `android-024665203bca17fa` / `R9TT608Q6MH` is reachable and identity verified over wireless at `192.168.1.179:34483`, but Device Control's current capability probes are unavailable. SmartTV 4K Cast ID `google-cast:3bc013548481e743aa769f04a9a9ba0b` is available at `192.168.1.158:8009`; its Android TV Remote profile is available at `192.168.1.158:6466` with input/media/pairing only. Neither TV transport exposes screenshot or semantic state.
- The Portal source continues to require trusted HTTPS and a signed-in operator for lifecycle RPCs. `device-control device discover --service _vrooli-portal._tcp.local --timeout-seconds 5 --json` returned `health=degraded` because Cast discovery timed out and an empty service list; it is not evidence that the Portal advertisement is absent.
- Scenario-to-Android owner build succeeded at 19:43 UTC for preview artifact `android-debug:f5b3246c7038ac391d812746196db6564bc62ade528cb49f15e5ccf1ebfd1413`, built from source digest `8cddf9cf84fb921d2878c91d002237b4a8163880b89af967ded810d6685f1754`, package `com.vrooli.portal.tv`, version 0.4.19/code37, target SDK 36, debug-only. The APK was not installed. The build is a current native source/build proof; it does not prove TV launcher state or Portal action success.
- Portal Test Genie unit run `20261102-194228-279512af` passed at L1, but its owner command log says it was a cache hit to `20261102-182312-30345356`; count it as reused evidence only. Run artifact `artifact_0fb6d0fa1ed4900a5c62615837cf42bc` was fetched and inspected. Current TV launcher foreground, own-app lifecycle over authenticated HTTPS, and physical restart/soak remain unverified. The README's earlier operator review decision for DNS/certificate/firewall setup remains in force; no host network configuration was changed.

## Follow-up — current Portal endpoint and TV readback boundary (2026-11-02 UTC)

- `vrooli scenario status portal --json` at `21:57Z` reported Portal healthy/running with build identity `sha256:e1e22e47c09418a206c510b49ede90cd55e2b1c653925d0bcb17d4f546e3587f`, API port 17476 and UI port 24965. Portal-owned `GET /health` returned HTTP 200 on both app ports. HTTPS requests to `https://localhost:17476/health` and `:24965/health` failed TLS with `wrong version number`; these direct app listeners were serving HTTP in this observation. Listener bindings were API `127.0.0.1:17476` and UI `0.0.0.0:24965`. This does not rule out a separate reverse proxy. The documentation example hostname `portal.home.example` did not resolve from this host; it is not evidence of the configured Portal Home hostname.
- Fresh Device Control inventory at `21:57:32Z` confirms SmartTV 4K ID `google-cast:3bc013548481e743aa769f04a9a9ba0b` available over Cast at `192.168.1.158:8009` with media/property/sensor but no input, screenshot or semantics. Its separate Android TV Remote profile at `192.168.1.158:6466` supports input/media/pairing but has no screenshot or semantic readback.
- The five-second Portal DNS-SD browse at this time returned `health=degraded`, `google-cast` discovery `context deadline exceeded`, and no services. This remains inconclusive about the Portal advertisement. No TV control, pairing, certificate, DNS or firewall change was made. Current TV Home foreground and Portal-owned app lifecycle action remain unverified.
- `scenarios/portal/android-tv/README.md` documents the owner path: direct HTTPS uses `PORTAL_TLS_CERT_FILE`, `PORTAL_TLS_KEY_FILE` and `PORTAL_HOME_HOST`, with a certificate trusted by the TV and a hostname resolving to the home gateway. It assigns certificate issuance/local DNS and network-admission setup to operator review through the existing lifecycle/setup safeguard. The operator has authorized HTTPS preference and an explicit HTTP fallback, but the exact trusted hostname/certificate/network route is not yet selected. Do not bypass the setup owner with private firewall rules.
- Memory work record: `b92e9c11-12c3-43f1-be04-dc1bc3170236`.

## Follow-up — native Portal preview build and unit gate (2026-11-02 UTC)

- The first Scenario-to-Android build invocation omitted the required native-project selector and produced generic package `com.vrooli.generated.app`; that artifact is not Portal evidence and was not installed. The corrected owner command set `ANDROID_PROJECT_KIND=native` and `ANDROID_NATIVE_PROFILE=preview` and returned immutable artifact `android-debug:f5b3246c7038ac391d812746196db6564bc62ade528cb49f15e5ccf1ebfd1413`, source digest `sha256:8cddf9cf84fb921d2878c91d002237b4a8163880b89af967ded810d6685f1754`, package `com.vrooli.portal.tv`, version 0.4.19/code37, target SDK 36, debug-only. The native project's `preBuild` depends on `verifyApplicationActions`, which compiles and executes the dependency-free menu policy regression.
- Fresh Portal Test Genie run `20261102-220506-0e93aab3` passed the `unit` phase at L1 with 53 observations and 49 warnings; findings identify existing test architecture debt (`TEST_UTIL_MISSING`, injectable seams) and do not claim Android/TV execution. No artifact was installed and no TV state changed.
- Current TV launcher foreground, trusted HTTPS operator action, and physical app start/restart/stop outcomes remain unverified. Memory work record: `2050d857-290f-45ca-8272-33b5df155349`.

## Follow-up — fresh TV readback capability check (2026-11-02 UTC)

- Device Control refreshed SmartTV 4K identity `google-cast:3bc013548481e743aa769f04a9a9ba0b` at `22:08:27.328Z`. Available transports remain Cast `192.168.1.158:8009` and Android TV Remote `192.168.1.158:6466`; neither exposes screenshot or semantic state, and no Android ADB identity was listed.
- A read-only state request at `22:08:51–22:08:54Z` returned empty Cast receiver application/player-state values plus Remote mute/volume properties. These do not identify the launcher foreground. A five-second ADB mDNS browse returned no rows but was degraded by Cast timeout, so it does not prove no ADB service exists. No TV action was sent.
- The current available transports cannot verify Portal Home foreground or lifecycle action outcomes. Next: continue investigating owner-supported TV readback and trusted HTTPS configuration, keeping operator-controlled hostname/certificate/network admission decisions intact. Memory work record: `429861d1-69a8-4e96-8468-58f6aa1ac379`.


## Follow-up — fresh TV state and discovery boundary (2026-11-02 UTC)

- Fresh SmartTV inventory at `22:27:21.672Z` retains Cast and Android TV Remote profiles, neither with screen or semantic readback. State at `22:27:28–31Z` returned empty Cast receiver/player identity and Remote mute/volume only; launcher foreground, physical power and input source remain unavailable.
- A focused `_adb-tls-connect._tcp` / `_adb._tcp` browse returned no rows while overall discovery health was degraded by Cast timeout. This does not prove ADB absence. No TV action or pairing was attempted. Memory work record `dbae07e1-c0f5-4151-bc21-9292e793204e`.

## Follow-up — current TV receiver readback and owner build (2026-11-02 UTC)

- Fresh inventory at `22:51:18Z` verified the SmartTV Cast identity `google-cast:3bc013548481e743aa769f04a9a9ba0b` at `192.168.1.158:8009`; `device state` at `22:51:24–27Z` returned empty receiver application and player-state values, Remote mute `true`, volume `0`, while input source and physical power were unavailable. It did not establish current launcher foreground. `device-control device inspect` explicitly refused because the selected transport does not support read-only runtime inspection. No TV action was sent.
- Scenario-to-Android owner preview build at `22:52:29Z` reproduced immutable artifact `android-debug:f5b3246c7038ac391d812746196db6564bc62ade528cb49f15e5ccf1ebfd1413` from source digest `sha256:8cddf9cf84fb921d2878c91d002237b4a8163880b89af967ded810d6685f1754`; package `com.vrooli.portal.tv`, version 0.4.19/code37, debug-only. The owner `preBuild` runs `verifyApplicationActions`; the artifact was not installed. Portal Test Genie unit run `20261102-225210-df1210db` returned PASS L1 in 0 seconds; it is reused evidence, not fresh test execution. Its findings retain `TEST_UTIL_MISSING` advisory debt.
- Source still includes owned-app Start/Restart/Stop menu actions, confirmation for restart/stop, action status refresh, and trusted-HTTPS plus signed-in operator checks. No physical action outcome is established. The exact operator-controlled trusted hostname/certificate/local DNS/network route is still missing; direct-LAN HTTP remains browsing-only.

## Follow-up — current TV screen-wake route refusal (2026-11-02 UTC)

- Fresh Device Control inventory at 23:28:19Z verified exact SmartTV Cast identity `google-cast:3bc013548481e743aa769f04a9a9ba0b`, Cast `192.168.1.158:8009`, Android TV Remote `192.168.1.158:6466`, both available. Cast declares media/property/sensor; Remote declares input/media/pairing. Neither declares screen-control; neither provides screen/frame/semantic readback or physical power state.
- Exact-ID `device screen --operation wake` returned 409 `screen_operation_failed`, status unavailable: “device has no available, state-readable screen-control transport.” The request was refused before actuation; no wake is claimed. Current readback remains Cast receiver application/player empty, Remote mute true/volume 0, input and physical power unavailable; exact-ID inspect is unsupported.
- Source inventory confirms Android TV Remote's declaration has only volume operations and no `ScreenController`; its wire client admits a bounded set of remote keycodes, but that is not an owned, state-verified wake contract. Adding a wake key without safe readback would risk an unverified or toggle-like action, so no strategy change was made. A supported power-state or screen-bearing observer remains needed before a verified wake journey can be delivered.


## Follow-up — current SmartTV and A03s state boundary (2026-11-03 UTC)

- At `01:49:43Z`, fresh inventory verified A03s `android-024665203bca17fa` / `R9TT608Q6MH` (Android 13), Moto G7 `android-6affcac615a281a2` / `ZY226TG2BM` (Android 10, USB and wireless `192.168.1.179:5555`), and SmartTV Cast `google-cast:3bc013548481e743aa769f04a9a9ba0b` at `192.168.1.158:8009` plus Remote at `192.168.1.158:6466`. A03s `device state` at `01:50Z` still failed on USB and wireless.
- Fresh SmartTV state at `01:50:14–18Z` reports Cast receiver application `AndroidNativeApp`, empty player state and volume `0.0`; Android TV Remote reports mute `false` and volume `0.11` (ARC/eARC source), so volume observations conflict. Input source and physical power remain unavailable. No foreground package, launcher state or Portal app action is established; inspect remains unsupported. No actuation was sent.


## Follow-up — current Portal process and TV readback (2026-11-03 UTC)

- `vrooli scenario status portal --json` reports healthy/running with API/UI listeners on 17476/24965, but the serving build is still `sha256:e1e22e47c09418a206c510b49ede90cd55e2b1c653925d0bcb17d4f546e3587f`; this does not deploy or qualify the newer native Portal source.
- Fresh TV read at 02:10Z identifies Cast receiver application `AndroidNativeApp`, empty player state, Cast volume `0.0`, and Android TV Remote mute false/volume `0.11` (ARC/eARC). Input, power, launcher foreground, and Portal lifecycle action outcomes remain unavailable or unverified. No TV action was sent. Trusted HTTPS hostname/certificate/DNS route remains an operator decision; no network setup was changed.

## Follow-up — Portal owner inventory recovered through the root API (2026-11-03 UTC)

- Portal `surfaces apps` and `app-url` initially failed because their control-plane owner call could not reach `vrooli-api` on port 8092. The root bootstrap marker was present; supported `vrooli develop --scenarios none --onboarding none` started only the project API and skipped the orchestrator. Portal's app inventory then returned 128 scenarios, and `app-url portal` resolved the live Portal UI to `http://localhost:24965`.
- The fresh `surfaces list` still reports only Web Console ready; Device Control, Bridge and Compute Manager sources are unavailable (`source_unreachable`). Portal serves the old build `sha256:e1e22e47c09418a206c510b49ede90cd55e2b1c653925d0bcb17d4f546e3587f`. The Portal lifecycle endpoint requires a signed-in human bearer token and exact lifecycle authorization; the owner CLI exposes no lifecycle action command. No app lifecycle action, TV action, TLS, DNS or network configuration change occurred. The returned direct app URL is HTTP and does not satisfy trusted-HTTPS action requirements.
- Memory work records `af93d3b2-9846-4b37-8167-6cffafd12376`, `7f34f5a2-bf03-4756-9722-4b69e3f8f8bc` and correction `644de28e-952b-43d1-b254-82a64ae23bd2` record root API recovery and Portal gateway diagnostics.
- Portal's same-origin gateway was exercised read-only: `portal surfaces app-url device-control --gateway` resolved `/apps/device-control/proxy/`; API proxy-target returned `UI_PORT=20698` and `API_PORT=16465`. `http://localhost:24965/` returned the Portal page title, and `http://localhost:24965/apps/device-control/proxy/` returned Device Control's page title. This verifies host-browser HTML launch over HTTP only; no TV/browser identity or trusted TLS is proven.


## Active outcome — lifecycle response truth regression

- **Mission:** Complete the Device Control and Portal maturity destination; app lifecycle actions must report the owner's observed terminal state and preserve uncertainty when a request is ambiguous.
- **Outcome:** Extract deterministic response interpretation for Start/Restart/Stop and cover confirmed, unconfirmed, and malformed/unknown results with the native owner regression. The activity may report completion only when the returned application state proves the requested action; ambiguous results must instruct refresh before retry.
- **References:** This epoch's `TV app lifecycle target`; Portal PRD/design; `scenarios/portal/android-tv/app/src/main/java/com/vrooli/portal/tv/MainActivity.java`, `ApplicationActionPolicy.java`, `PortalClient.java`, and `verification/ApplicationActionPolicyTest.java`.
- **Ownership:** Portal owns lifecycle UI and response wording. Scenario-to-Android owns the source snapshot and native preview build. Do not change lifecycle API semantics, HTTPS/sign-in gates, control-plane behavior, signing identity, or TV state. Preserve existing dirty changes.
- **Verification:** Native owner build with `ANDROID_PROJECT_KIND=native`, `ANDROID_SOURCE_REF` set to the launcher, preview profile, and JSON output; run Portal API package tests only if the response contract changes. Use the Test Genie phase appropriate to changed Portal source. No service restart, app install, or TV mutation in this slice.
- **Return:** Changed paths, focused test/build commands and terminal results, source/artifact identities, remaining and unverified.


### Accepted source/build slice — lifecycle response truth (2026-11-03 UTC)

- The Portal-owned native activity now calls a deterministic response policy. It requires both `outcome=completed` and the action's expected runtime state, keeps all other results uncertain, refreshes the catalogue, and tells the operator to refresh before retrying. Start-after-action opens only after confirmed Start.
- Desired regressions cover success for Start/Restart/Stop and mismatch, incomplete outcome, unknown runtime, invalid action, and null inputs. Existing HTTPS/sign-in and destructive confirmation checks remain in the owner suite.
- Scenario-to-Android preview owner build passed, including `verifyApplicationActions`: artifact `android-debug:c1a8b226fde3246e28144890dd1014eb1c7966c4db45d543a064dfdf52dc4d88`, source digest `sha256:507ca2d1eab7bec62dc397c40f4a9963cd4c2a2064325333ad344e7b3e0bcab9`, preview package `com.vrooli.portal.tv`, v0.4.19/code37. APK/AAB hashes are in LOG. Portal unit Test Genie `20261103-032846-9767d590` passed L1 with 48 existing warnings. No API contract changed, so no Portal API repair was needed.
- The artifact was not installed. Trusted HTTPS, operator sign-in on TV, successful lifecycle actions, exact TV foreground readback, and launcher soak remain unverified. This accepts the response-policy source/build slice, not E5 or the overall goal. Worker `/root/portal_lifecycle_response`; independent coordinator source/API review found the policy matches Portal's `Act` contract.

## Follow-up — fresh SmartTV read-only state (2026-11-03 UTC)

- Exact Device Control ID `google-cast:3bc013548481e743aa769f04a9a9ba0b` still resolves to Cast `192.168.1.158:8009` and Android TV Remote `192.168.1.158:6466`. Read at 04:31Z returned empty receiver application/player values, mute conflict (`google-cast=false`, Remote=true), Remote volume 0 with source `arc-earc-amplifier`, and input/physical power unavailable. Neither transport supports screenshot or semantics; inspect remains unsupported. No TV action occurred. TV-owned visual/power readback remains open. The operator clarified that HTTPS is not required for ordinary Device Control LAN/Remote actions; Portal app lifecycle calls retain their own trusted-HTTPS/sign-in requirement.

## Operator correction — HTTPS scope (2026-11-03 UTC)

- The operator corrected the earlier blanket HTTPS prerequisite: ordinary Device Control LAN reads and Android TV Remote actions have been used without HTTPS. Do not require a Portal hostname, certificate, or local DNS/network change for those Device Control routes.
- This correction does not alter Portal app policy: `PortalClient` still requires trusted HTTPS and a signed-in human for its lifecycle RPC, and the server still enforces verified human authorization. That app-specific path has not been qualified on the TV. Physical foreground/input/power readback and Portal action outcomes remain unavailable/unverified; no new TV action was sent.
