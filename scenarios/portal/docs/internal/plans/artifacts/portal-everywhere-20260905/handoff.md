
## Native activation observer interfaces — 2026-09-06

Device Control's native host backend now implements the activation observer,
activation image observer, and companion window/process verifier interfaces used
by the authenticated helper owner path. Windows probes foreground HWND/process
and virtual-screen geometry through bounded PowerShell and verifies HWND
ownership before/after activation. macOS probes the front Accessibility window
and verifies that the companion process remains present. Image evidence is
cropped to the exact probed window rectangle and rejected when the frame cannot
contain it, preventing full-screen bytes from being mislabeled as window
context.

Focused race tests, host/helper/strategy integration tests, vet, and Darwin
non-cgo and Windows cross-compiles pass. The Apple cgo/SDK build and live
Windows/macOS permission journeys remain unavailable, so NAT-03/NAT-04 and the
primary platform rows remain unverified.

## Governed helper-provider extension contract — 2026-09-06

The Scenario-to-Desktop native extension contract now carries optional
`helper_providers` metadata. Admission accepts only the closed
`device-control` / `desktop.session` provider, rejects duplicate or arbitrary
owner/capability values, and propagates the declaration through protobuf and
pipeline conversion. Generated `native-extension.json` includes the metadata;
no executable path or import authority is accepted. Scenario-to-Desktop
Generation/Pipeline tests and the build-tools extension suite pass. Helper
bytes still require ramp-owned packaging and live platform lifecycle evidence.

## Native extension generation refresh — 2026-09-06

The installed Scenario-to-Desktop CLI and build-tools dist were rebuilt against the current helper-provider proto/validator. Portal generate-only pipeline `29062494-af36-2843-a0de-d4346903d777` completed successfully; generated Electron `main.ts` and `preload.ts` now embed the governed `device-control` / `desktop.session` provider metadata. This closes the local generation propagation gap. It does not create live Windows, macOS, GNOME Wayland, or KDE Wayland evidence.

## Refreshed Portal companion artifact — 2026-09-06

Portal was regenerated, built, and smoke-tested through Scenario-to-Desktop pipeline `fd250e21-5c80-b22d-53d7-ec037f94743d` for Linux X11 staging. The AppImage is 108454052 bytes with SHA-256 `ace7ec108d7c049570c8c58de44d0b3ac1a75dac78b2f32c0e63851794e50eb0`; smoke capture `8c6d2dae-2662-446e-afcb-d10a66056681` passed. This confirms refreshed Linux package identity and launch behavior only; primary-platform lifecycle evidence remains open.

## Final-regression reference reconciliation — 2026-09-06

The acceptance ledger now carries partial DEL-16 producer and artifact references for the focused changed-owner regression and refreshed Linux companion receipt. DEL-16 and `finalRegression` remain unpassed until the whole collection and live platform gates are complete.

## Rechecked macOS host availability — 2026-09-06

A fresh `host inventory` run `c2d0e0ba-69db-4d54-aee3-fee34ea87e05` confirms the only macOS Bridge node remains darwin/amd64 with no attached display or interactive user session; xcodebuild readiness also fails. The required macOS arm64 Accessibility/ScreenCaptureKit journey remains unavailable.

## Fresh configured Linux X11 helper observation — 2026-09-06

Device Control was restarted through the lifecycle with the protected owner/helper bootstrap. Authenticated owner Open(control=false), Observe, and Stop completed on X11 session 2; capture was 1920×1080 and postconditions showed zero active grants, no helper lease, and no cleanup pending. Receipt `device-control-live-x11-helper-observation-receipt-20260906.json` records the result. This strengthens Linux helper evidence; the DEL-03 multi-platform gate remains open.

## Fresh physical X11 pointer acceptance — 2026-09-06

The installed Device Control CLI drove a disposable X11 fixture through the lifecycle-managed helper. One primary press/release pair was observed at the expected coordinates, the duplicate command returned the same receipt without a second effect, the pointer was restored, and Stop succeeded. Receipt `device-control-live-x11-pointer-refresh-receipt-20260906.json` records the run.

## Fresh physical X11 keyboard and stop-release acceptance — 2026-09-06

The installed Device Control CLI sent a key press and a held modifier only to a disposable focused X11 fixture. Duplicate suppression, Stop release, no-held-key, and focus-restoration assertions passed. Receipt `device-control-live-x11-keyboard-refresh-receipt-20260906.json` records the run.

## Fresh physical X11 Unicode semantic input — 2026-09-06

The installed Device Control CLI inserted Japanese, Arabic, accented, combining-mark, and emoji text into a disposable GTK fixture exactly once. Duplicate receipt equality and Stop succeeded. Receipt `device-control-live-x11-unicode-refresh-receipt-20260906.json` records the run.

## Fresh physical X11 saved-flow replay — 2026-09-06

A two-insertion Unicode desktop flow with a terminal assertion passed, was promoted, replayed against a fresh GTK process, and produced equal duplicate records. Receipt `device-control-live-x11-saved-flow-refresh-receipt-20260906.json` records the run.

## Fresh physical X11 resolution and wheel acceptance — 2026-09-06

Live owner CLI resolution found the exact GTK application, uniquely resolved its editable field, and returned explicit absent resolution for a missing selector. Wheel input delivered both axes to a disposable X11 fixture with duplicate suppression and pointer restoration.

## Fresh physical X11 Stop benchmark — 2026-09-06

Fifty local owner CLI Stop trials passed after completed captures. Every durable lease cleared and post-stop input was denied; p95 latency was 8.229ms and maximum latency 8.626ms against a 250ms target.

## Fresh local companion activation binding — 2026-09-06

Authenticated owner IPC captured activation context for an unmapped native X window, refused a wrong-window capture, preserved a newer context after old deletion, rejected the deleted context, and released the session on Stop.

## Fresh physical X11 inline flow — 2026-09-06

A two-step Unicode inline flow passed against a live GTK fixture with duplicate suppression, application discovery, unique window resolution, and clean Stop.

## Selected desktop context integration — 2026-09-06

Focused integration `TestStartPortalCompanionContextOnSelectedDesktop` passed against the refreshed Linux staging Electron companion. The run retained prior-app focus, refused overlapping context capture, published a ready activation with an image, preserved capture/bounds/display provenance and source annotation geometry, removed the preview during cleanup, and restored the same draft node. Receipt `portal-selected-desktop-context-receipt-20260906.json` records the assertions. This is Linux X11 validation only; Windows, macOS, GNOME Wayland, and KDE Wayland live evidence remain open.

## Nested GNOME Wayland runtime recheck — 2026-09-06

A controlled nested GNOME Shell 46.2 session created a Wayland socket and AT-SPI registry, and the GNOME portal implementation activated with a Screenshot request object. The request remained pending because this host did not provide a usable portal-approved capture stream; no capture or input effect is counted. Receipt `gnome-wayland-runtime-probe-receipt-20260906-r2.json` records the bounded result. The focused native package regression (`wayland`, `host`, `desktophelper`, and `hostdesktop`) passed with race detection.

## Cross-target desktop-helper artifacts — 2026-09-06

The lifecycle helper now has retained cross-target build artifacts for Linux amd64, Windows amd64, and Darwin arm64 with ELF/PE/Mach-O identity and SHA-256 digests in `desktop-helper-cross-target-build-receipt-20260906.json`. The Darwin artifact is the CGO-disabled compatibility build because Apple SDK tooling is unavailable here; these artifacts do not replace live user-session and permission evidence.

## Remaining native acceptance seam regression — 2026-09-06

The named producer tests for Windows semantic invocation, macOS stale-capture revoke, GNOME Wayland portal flow, and KDE Wayland portal flow all passed with the race detector. Receipt `remaining-native-acceptance-seam-regression-receipt-20260906.json` records the results. These fixtures validate the implementation seams only; the required live platform rows remain unverified.

## Semantic process identity revalidation — 2026-09-06

Native Windows/macOS semantic actions now recheck that the process which produced the cached tree is still present immediately before an asserted, text, or invoke effect. A replacement-process regression refuses the action before the native effect; focused and package-wide race tests plus vet pass. Receipt `native-semantic-process-revalidation-receipt-20260906.json` records the guard and its limits.

A final process identity check now also occurs immediately before semantic text mutation, closing the gap between stale-value read and the native setter. The host race suite remains green.

## macOS frontmost-window verification — 2026-09-06

The macOS activation verifier now requires the expected Accessibility process to remain frontmost and expose at least one window, instead of accepting process existence alone. Regression tests cover both the accepted and refused paths; native package race tests and vet pass. Receipt `native-window-process-verification-receipt-20260906.json` records the seam evidence. Live macOS permission and session evidence remains unavailable.

The retained cross-target helper binaries were rebuilt after the macOS frontmost-window verifier change; the helper receipt now carries current digests for all three targets.

## macOS semantic element identity revalidation — 2026-09-06

The macOS Accessibility adapter uses helper-generated ordinal runtime IDs. Cached process presence alone could allow a reordered or replaced window element to receive an action. Semantic validation now re-enumerates the current tree and compares the selected element's name, editability, role, parent, and window ancestry before validation and again immediately before the effect. `TestMacOSSemanticInvokeRefusesReplacedElement` proves a changed element is refused before invocation; the native host, Wayland, desktop-helper, and hostdesktop race suites plus vet pass. Receipt `native-semantic-element-revalidation-receipt-20260906.json` records the evidence. Live macOS Accessibility remains unavailable.

## Wayland held-input cleanup — 2026-09-06

The portal backend now tracks helper-owned key and pointer-button presses, suppresses repeated down/click effects, releases held input during cleanup and before portal session close, and retains failed releases for retry. Focused and native race/vet suites pass; receipt `wayland-held-input-release-receipt-20260906.json` records the seam evidence. Live GNOME and KDE Wayland capture/input journeys remain unavailable.

## Native host held-key ownership — 2026-09-06

Windows and macOS native host backends now distinguish key press from key down/up, use bounded native key events for held keys, suppress duplicate down effects, and release helper-owned keys during cleanup and close. Receipt `native-host-held-key-release-receipt-20260906.json` records focused and package-wide race/vet evidence. Live Windows and macOS user-session evidence remains unavailable.

## Native host held-pointer cleanup — 2026-09-06

Windows pointer down/up actions now track helper-owned buttons, suppress duplicate presses, and release them without moving the cursor during cleanup. macOS retains its explicit click-only contract. Native race and vet suites pass; live Windows/macOS validation remains unavailable.

## Native host pointer lifecycle — 2026-09-06

The host backend now emits Quartz pointer move/down/up/click events on macOS, retains Windows pointer handling, tracks helper-owned buttons on both platforms, and releases held buttons at the current cursor location during cleanup. Deterministic native race/vet tests pass; live platform acceptance remains pending.

## Stale semantic text refusal — 2026-09-06

The native host text mutation path now returns `ErrUnavailable` when the element value changes between validation and the setter, instead of treating a nil mismatch error as success. A regression covers the race; native race and vet suites pass.

## 2026-09-06 15:54Z — comprehensive regression terminal evidence

A fresh supported `vrooli scenario test portal` run (`20260906-152415-970b4a42`) reached a terminal failure after 1663.7 seconds. Sixteen phases passed, nine failed, and three were provider-unavailable. The exact phase summary is retained in `execution-evidence/portal-test-genie-comprehensive-regression-receipt-20260906.json` and is attached to `finalRegression` and DEL-16 without promoting either to passed. Focused native race/vet validation and Windows/macOS cross-target compilation remain green. Live Windows, macOS, GNOME Wayland, and KDE Wayland acceptance remains unavailable.

## 2026-09-06 15:55Z — macOS Bridge recheck

Bridge inventory run `4c569f8c-a92b-449a-830f-384703d80034` passed for `minimouse`, but the node remains Darwin amd64 with no attached display or interactive user session and no `xcodebuild`. Receipt `execution-evidence/macos-node-availability-probe-receipt-20260906-r3.json` is attached to NAT-04 and the macOS platform row; both remain `not_run`.

## 2026-09-06 15:58Z — focused changed-owner regression rerun

The bounded changed-owner regression is green after the latest native host test cleanup. Device Control native/helper/strategy/X11/AT-SPI packages passed; Portal targeted UI tests passed 33/33; scenario-to-desktop vanilla lifecycle/recovery passed 8/8; Portal Program Runtime contract passed 6/6. Receipt `execution-evidence/focused-final-regression-receipt-20260906-r2.json` records the current source fingerprint. It remains insufficient for DEL-16 because the comprehensive run failed and live native platform rows remain unverified.

## 2026-09-06 16:35Z — Device Control comprehensive regression

The server-owned Device Control run `20260906-155014-0265bf1c` reached terminal failure after its long architecture phase: 18 phases passed, 7 failed, and 3 were provider-unavailable. Receipt `execution-evidence/device-control-comprehensive-regression-receipt-20260906.json` preserves the phase summary and is attached to finalRegression and DEL-16. The bounded native/helper regression remains green; the four live platform rows remain unverified.

## 2026-09-06 16:46Z — native helper readiness hardening

Bound Bridge HTTP clients with 30-second timeouts, switched helper retry logging to structured `slog`, and moved the helper retry loop out of `main` into `run`. Focused helper/control tests and vet pass. Fresh git-control-tower review job `c803365c-5bc7-4b7d-a0e8-dad7e80a2421` completed with yellow readiness and zero blocking standards violations; remaining warnings are advisory or repository-wide. Live native platform and final plan gates remain open.

## 2026-09-06 16:48Z — helper timeout/logging validation

The Bridge client timeout and structured helper logging changes passed regular tests, vet, and a recovered sequential race run (`go test -count=1 -race ./internal/control`). The helper command package compiles with no test files. The fresh readiness review remains yellow with zero blocking standards violations; platform and final plan gates remain open.

## 2026-09-06 16:54Z — Portal readiness provider gap

Fresh git-control-tower review job `4c126b58-a835-4f10-98a7-b0ba62e1c7fa` completed its rules check but could not run tests because all integrated providers were unavailable. It produced no readiness dimensions and is not treated as a code verdict. Focused Portal/native tests remain green and authoritative locally.

## 2026-09-06 17:04Z — routed helper storage and declaration cleanup

The desktop helper's two production SQLite bootstrap paths now use `api-core/database.Open` with a bounded single-connection pool, and the desktop repository accepts the small engine-neutral database surface so production retains RoutedDB test-pool routing while fixture tests keep ordinary pools. Sessions and helper tests pass with race detection and vet. Storage validation no longer reports `RAW_SQL_OPEN`, `SQL_DB_HANDLE_CAPTURE`, or an uncovered helper build cache: the generated cross-target cache was moved to `/tmp` after its digests were retained in the cross-target receipt, and the framework retention receipt is declared under the scenario's state storage. Existing direct filesystem-writer findings remain outside this focused change. A fresh Portal review job `d9fff415-fff0-4e8f-ab92-4506e29b3336` completed with tests failed and all integrated capabilities unavailable, so it remains non-assessable rather than a code verdict.

## 2026-09-06 17:09Z — post-storage-cleanup focused regression

After the routed helper and storage declaration changes, Device Control package-wide tests passed, Portal targeted tests passed 33/33, scenario-to-desktop lifecycle/recovery passed 8/8, and the Portal Program Runtime contract passed 6/6. Storage validation no longer reports raw SQL, captured SQL handles, or uncovered owner storage. Receipt `execution-evidence/focused-final-regression-receipt-20260906-r3.json` records the current source fingerprint and remains attached to DEL-16 and finalRegression; comprehensive and live-platform gates remain open.

## 2026-09-06 17:14Z — refreshed cross-target helper artifacts

Linux amd64, Windows amd64, and Darwin arm64 helper binaries were rebuilt from the current routed-database source. The retained ELF, PE, and Mach-O artifacts and SHA-256 digests in `execution-evidence/desktop-helper-cross-target-build-receipt-20260906.json` now point to `/tmp/device-control-cross-target-artifacts-20260906`; the Darwin binary remains the CGO-disabled compatibility build because Apple SDK tooling is unavailable here.

## 2026-09-06 17:19Z — current Linux X11 helper observation

After the lifecycle rebuild, a fresh Linux X11 owner observation passed through the current helper: authenticated Open(control=false), 1920×1080 PNG capture with matching signature and dimensions, and Stop cleanup. Receipt `execution-evidence/device-control-live-x11-helper-observation-receipt-20260906-r2.json` is attached to the Linux X11 platform row; other platform rows remain unverified.

The same current helper then passed fresh Unicode semantic text, pointer, keyboard, and horizontal/vertical wheel CLI journeys. Duplicate receipts matched, held input was released, focus and pointer were restored, and every Stop completed. Receipt `execution-evidence/device-control-live-x11-interaction-receipt-20260906-r2.json` records the current source fingerprint and is attached to the Linux X11 row.

Fresh resolution, physical flow, saved-flow promotion/replay, and companion-binding journeys also passed against the current helper. Exact and absent target resolution, duplicate flow records, terminal assertions, stale-context refusal, newer-context preservation, and cleanup release were verified. Receipt `execution-evidence/device-control-live-x11-resolution-flow-companion-receipt-20260906-r2.json` is attached to the Linux X11 row.

## 2026-09-06 17:35Z — shared Flow proto layout and focused regression refresh

Moved Device Control `Flow` and `Step` into `v1/shared`, regenerated committed Go and TypeScript bindings, and updated consumers. `proto-health validate scenario device-control` and `make verify-committed-gen` pass. Fresh focused receipt `execution-evidence/focused-final-regression-receipt-20260906-r4.json` records Device Control package tests passing, Portal 33/33, vanilla lifecycle/recovery 8/8, Program Runtime 6/6, and storage validation passing with existing direct-writer advisories. Linux/Windows/Darwin helper artifacts were rebuilt with source fingerprint `36ef17f50eca6caf47f799e63a2e7020c395ab62d2d66952879f6b78ee358933`. Live Windows, macOS, GNOME Wayland, and KDE Wayland acceptance plus the whole-collection final regression remain open.

## 2026-09-06 17:50Z — current Linux X11 post-proto helper regression

Device Control was restarted through the lifecycle manager after the shared Flow proto move and became healthy. A fresh authenticated owner observation passed at 1920×1080. Sequential Unicode, pointer, keyboard, wheel, resolution, physical flow, saved-flow promotion/replay, and native companion-binding journeys passed; the initial concurrent attempt was discarded because the single control lease was already held. Current r3 receipts use source fingerprint `36ef17f50eca6caf47f799e63a2e7020c395ab62d2d66952879f6b78ee358933` and are attached to the Linux X11 platform row and relevant acceptance cases. Other platform rows and final collection gates remain open.

## 2026-09-06 17:52Z — focused Test Genie queue limitation

The attempted Device Control storage/proto-only Test Genie runs were queued behind other server-owned work with waits exceeding 17 minutes and were explicitly aborted. Direct `storage-manager validate scenario device-control --json`, `proto-health validate scenario device-control --json`, and `make verify-committed-gen` passed and remain the authoritative focused results; no completion claim relies on the aborted runs.

## 2026-09-07 01:54Z — GNOME Wayland portal environment probe

A fresh isolated nested GNOME Wayland attempt created the compositor socket and AT-SPI registry and launched PipeWire, but GNOME Shell reported an incompatible nested display and its screencast service failed. `xdg-desktop-portal-gnome` aborted, leaving Screenshot/ScreenCast unavailable. Receipt `execution-evidence/gnome-wayland-runtime-probe-receipt-20260907.json` is attached to NAT-06 and the GNOME support row; both remain explicitly `not_run`.

## 2026-09-07 01:57Z — GNOME portal interface probe refined

A second nested GNOME attempt showed the portal object advertising Screenshot, ScreenCast, and RemoteDesktop and returning a Request object. `org.gnome.Shell.Screencast` still could not open the nested display, so the backend failed before a response. Receipt `execution-evidence/gnome-wayland-runtime-probe-receipt-20260907-r2.json` is attached to NAT-06 and the GNOME platform row; both remain `not_run`.

## 2026-09-06 18:03Z — fresh macOS node lifecycle readiness probe

`minimouse` remains online, heartbeat-fresh, protocol-compatible, and dispatchable, but is darwin/amd64 rather than the required arm64. Device Control start still abandons during dependency health because `scenario-authenticator` is not ready and no API/UI ports bind; no display or interactive user session is observed. Receipt `execution-evidence/macos-node-availability-probe-receipt-20260907.json` is attached to NAT-04 and the macOS platform row, which remain `not_run`.

## 2026-09-07 02:10Z — Portal native Linux AppImage packaging

`npm run dist:linux` passed in `scenarios/portal/platforms/electron` with electron-builder 26.15.3 and Electron 33.4.11. The retained Linux x64 AppImage is `/tmp/portal-native-artifacts-20260907/Vrooli-Portal-0.0.1.AppImage` with SHA-256 `a9554aeeb6e3023e05945381069b605db78019c4b1287a27eaed43c0e3fabace`; receipt `execution-evidence/portal-native-linux-appimage-build-receipt-20260907.json` is attached to PKG-07 and the Linux X11 row. The artifact is unsigned, Electron smoke was not claimed because the host sandbox helper is not setuid-configured, and Windows/macOS packaging plus live installation remain unavailable.

## 2026-09-06 18:11Z — packaged Linux AppImage smoke validation

The retained Portal Electron Linux x64 AppImage passed a bounded startup smoke run under disposable Xvfb against the live Portal UI: `SMOKE_TEST_READY=true`, `SMOKE_TEST_RESULT=passed`, and `SMOKE_TEST_EXIT=clean`. Artifact SHA-256 is `a9554aeeb6e3023e05945381069b605db78019c4b1287a27eaed43c0e3fabace`; receipt `execution-evidence/portal-native-linux-appimage-smoke-receipt-20260906.json` is attached to PKG-07, DEL-12, and the Linux X11 row. Xvfb proves packaged startup and server readiness, while native user-session interaction remains covered by the separate X11 helper receipts. The artifact is unsigned and Windows/macOS packaging plus live platform evidence remain unavailable.

## 2026-09-06 18:17Z — cross-target Windows and macOS package builds

`npm run dist:win` exited 0 and produced an unsigned Windows x64 NSIS installer and PE executable. Installer SHA-256: `7c932aa8bebb2face073db4543f3b8607955e9a2cb4028e719bc41a3dc553eb8`; receipt `execution-evidence/portal-native-windows-package-receipt-20260906.json` is attached to PKG-07, DEL-12, and the Windows support row. `npm run dist:mac` produced unsigned macOS x64 and arm64 zips with SHA-256 `6cf4d54af33fcb3368c73bf12edf52ffd49acf6dc7c2d5793a633e259ba1c990` and `140070ab336b9cff86cae2c340fb3116a2e0ed6cdd018fc650620156ffef3812`; extracted binaries identify as Mach-O x86_64 and arm64. Receipt `execution-evidence/portal-native-macos-package-receipt-20260906.json` is attached to PKG-07, DEL-12, and the macOS row. Cross-target package identity does not prove live installation, permission, or native interaction; live platform cases remain not_run.

## 2026-09-06 18:22Z — Windows packaged smoke environment boundary

Wine 11.0 is now available after the cross-target build. A bounded attempt to run the Windows x64 packaged executable under Wine and disposable Xvfb emitted no smoke readiness/result/exit markers and did not terminate within 90 seconds; it was interrupted. Receipt `execution-evidence/portal-native-windows-wine-smoke-receipt-20260906.json` records this as `failed_environment`. The Windows package identity receipt remains valid, but no Windows UI Automation, permission, capture, or semantic acceptance is claimed; NAT-03 and the Windows support row remain `not_run`.

## 2026-09-06 18:25Z — post-packaging bounded regression refresh

Fresh bounded validation passed after the package builds: Device Control Go package suite; Portal targeted UI 33/33; scenario-to-desktop lifecycle/recovery 8/8; Portal Program Runtime contract 6/6; proto-health with no findings; and committed proto generation verification. Storage validation exited 0 with 19 existing `FILESYSTEM_DIRECT_WRITER` advisories and zero `RAW_SQL_OPEN`, `SQL_DB_HANDLE_CAPTURE`, or `STORAGE_PATH_UNCOVERED` findings. Receipt `execution-evidence/focused-final-regression-receipt-20260906-r5.json` is attached to DEL-16 and finalRegression as bounded evidence. Comprehensive Test Genie and live-platform gates remain open.

## 2026-09-06 18:27Z — Electron source smoke dependency boundary

A fresh source-level `npm run smoke-test` under Xvfb still fails before launch because the local `node_modules/electron` installation reports “Electron failed to install correctly.” Packaged AppImage smoke remains green; this dependency failure does not change platform acceptance status.

## 2026-09-06 18:36Z — governed dependency surface cleanup

The accidental Electron dependency added to `scenarios/portal/ui` was removed and its lockfile restored. The required first-party `@vrooli/react-component-library` import was re-established through the approved SDA gateway, and the Portal UI lockfile now contains that intended local-package closure without an Electron importer. The Electron platform surface was separately installed through SDA. Its scripts-disabled install still cannot run source-level Electron smoke, so packaged AppImage smoke remains the validated startup path.

## 2026-09-06 18:35Z — acceptance evidence reference integrity audit

An independent ledger audit inspected 460 producer, assertion, and artifact references. Every referenced file exists at an absolute path, and all 398 JSON references parse successfully. Receipt `execution-evidence/ledger-reference-integrity-receipt-20260906.json` is attached to DEL-16 and finalRegression. This verifies evidence plumbing only and does not promote any not_run case or platform row.

## 2026-09-06 18:38Z — fresh macOS dependency readiness retry

The `minimouse` Bridge node was rechecked and `scenario-authenticator` was stopped/restarted through the remote lifecycle. It still has two processes but no bound ports or health result; the start operation remains abandoned in its health step. The node is Darwin amd64 without an attached display or interactive user session, while the acceptance row requires macOS ARM64. Receipt `execution-evidence/macos-node-availability-probe-receipt-20260906-r2.json` is attached to NAT-04 and the macOS row; both remain `not_run`.

## 2026-09-06 18:40Z — GNOME headless Wayland capability boundary

A fresh GNOME Shell 46.0 headless Wayland probe started Mutter 46.2 and created `wayland-0`, but exposed no monitor. The `--virtual-monitor 1920x1080` option failed because this backend cannot create virtual monitors. No capture/input surface or user session was available, so no Device Control journey was attempted. Receipt `execution-evidence/gnome-wayland-headless-probe-receipt-20260906.json` is attached to NAT-06 and the GNOME row; both remain `not_run`.

## 2026-09-06 18:44Z — fresh nested GNOME Wayland monitor probe

A bounded `xvfb-run`/D-Bus nested GNOME Shell attempt launched GNOME Shell 46.0/Mutter 46.2 but failed to create a virtual monitor because the backend does not support virtual monitors. No monitor, capture stream, input surface, or user session was exposed. Receipt `execution-evidence/gnome-wayland-nested-probe-receipt-20260906.json` is attached to NAT-06 and the GNOME support row; both remain `not_run`.

## 2026-09-06 18:46Z — KDE Wayland environment probe

The current host has no `kwin_wayland` or `startplasmacompositor` executable. No KDE Wayland session can be provisioned here, so no capture/input journey was attempted. Receipt `execution-evidence/kde-wayland-environment-probe-receipt-20260906.json` is attached to NAT-07 and the KDE support row; both remain `not_run`.

## 2026-09-06 18:49Z — Device Control requirements evidence refresh

`vrooli scenario requirements sync device-control --json` refreshed the owner evidence timestamp and validation vocabulary. The subsequent requirements validation returned `PASSED` with no report findings; the sync reports 18/30 operational targets and 23/67 requirements complete. This resolves the stale-evidence warning without promoting any Portal Everywhere live-platform case.

## 2026-09-06 18:52Z — fresh native package regression

`cd scenarios/device-control/api && go test -count=1 ./internal/native/... ./internal/desktophelper/...` passed across AT-SPI, host, Wayland, X11, and desktop-helper packages. Receipt `execution-evidence/device-control-native-host-regression-receipt-20260906-r2.json` is attached to DEL-16 and finalRegression as focused evidence; whole-collection and live platform gates remain open.

## 2026-09-06 19:16Z — targeted Test Genie provider-readiness recovery

The narrow `performance,storage,workflow,proto` run `20260906-185359-4e9c559f` never left `preparing:provider_readiness`; after 732 seconds without progress, Test Genie was restarted through the lifecycle manager and the orphan run terminalized as `aborted`. Test Genie is healthy afterward. Receipt `execution-evidence/device-control-targeted-testgenie-provider-readiness-receipt-20260906.json` records the infrastructure boundary; direct focused package receipts remain authoritative and DEL-16/finalRegression remain open.

## 2026-09-06 19:20Z — isolated proto Test Genie retry

After the Test Genie restart, the isolated `proto` run `20260906-191706-71f15650` again froze in `preparing:provider_readiness` for 161.4 seconds against a 12-second estimate. The typed abort completed and the run is terminal `aborted`; no proto phase verdict was produced. Receipt `execution-evidence/device-control-proto-testgenie-readiness-receipt-20260906.json` records the infrastructure boundary. Direct proto-health and committed-generation checks remain passing evidence.

## 2026-09-06 19:23Z — Test Genie CLI rebuild boundary

A fresh isolated `proto` retry did not create a run because the Test Genie CLI rebuild failed while compiling managed Go 1.25.12, reporting standard-library packages unavailable under `/home/matthalloran8/.vrooli/opt/go/go`. A concurrent Git Control Tower compilation was observed and left undisturbed; direct Device Control Go tests still pass. Receipt `execution-evidence/device-control-testgenie-cli-rebuild-receipt-20260906.json` records the pre-admission infrastructure failure.

## 2026-09-06 19:26Z — platform package identity completion

The acceptance ledger now records retained package digests for Windows x64 (`7c932aa8…` NSIS installer), macOS ARM64 (`140070ab…` zip), and Linux GNOME/KDE Wayland (`a9554aee…` Portal AppImage). Platform statuses remain `not_run` because package identity does not prove live installation, permissions, capture, or input; the verifier no longer reports missing `artifactDigest` errors.

## 2026-09-06 19:41Z — targeted Test Genie regression after CLI recovery

The rebuilt Test Genie CLI admitted an isolated `proto` run `20260906-193253-091d7cb7`, which completed with a server-owned PASS in 2 seconds. A subsequent targeted `performance,storage,workflow` run `20260906-193916-c6454ff3` completed in 128 seconds: storage passed; performance failed Lighthouse at 0.56 below the 0.75 threshold; workflow failed six BAS cases because the shared workflow-health resolver could not bind `@scenario/self` to the healthy lifecycle-running Device Control API. Receipts `execution-evidence/device-control-proto-testgenie-pass-receipt-20260906.json` and `execution-evidence/device-control-targeted-testgenie-regression-receipt-20260906.json` are attached to DEL-16/finalRegression. The integrated performance/workflow gates and live platform rows remain open.

## 2026-09-06 20:12Z — Device Control workflow UI contract fixes and rerun

The BAS compiler now normalizes authored `@scenario/self` against an absolute scenario root before lifecycle port resolution; its compiler/executor tests pass. Device Control now treats nullable `evidence` in a capability-gap flow response as an empty retained-evidence list, preserving the run-review surface, and its app shell adds scroll padding for fixed bottom navigation. Device Control UI validation passed (`pnpm test`: 154 tests, `pnpm type-check`, and `pnpm build`). The lifecycle-managed scenario restarted healthy. Test Genie workflow run `20260906-200826-e3a1650b` confirmed the route and run-review card now render, but two BAS cases still fail at `flow-evidence-image`: the selected live Android TV device has no screenshot capability and the API correctly returns `evidence:null`. Receipt `execution-evidence/device-control-workflow-ui-fix-testgenie-receipt-20260906.json` is attached to DEL-16 and finalRegression. Platform rows and finalRegression remain open; no retained capture is claimed without a screenshot-capable target.

## 2026-09-06 20:23Z — governed Android SDK readiness attempt

The governed `android-sdk` resource installed successfully under `/home/matthalloran8/.vrooli/resources/android-sdk`, exposing `adb`, emulator tooling, JDK 17, and Gradle. A local API 36 Google APIs x86_64 AVD was created, but the resource's bounded `avd-start` gate failed after three minutes because `sys.boot_completed` never became `1`; the emulator exited and `adb devices` remained empty. The retained Galaxy A03s endpoint also returned `No route to host` from this host. Receipt `execution-evidence/android-sdk-avd-readiness-receipt-20260906.json` is attached to DEL-16/finalRegression as environment evidence. No screenshot-capable target was fabricated and the two retained-capture workflow cases remain unpassed.

## 2026-09-06 20:30Z — second local Android AVD attempt

The pre-existing `vrooli-api36` AVD was also started through `resource-android-sdk avd-start`; it failed the same three-minute `sys.boot_completed` gate with QEMU thread hangs and exited without an `adb` device. The Android SDK toolchain is installed, but this host cannot provide a usable screenshot-capable Android target. The readiness receipt was extended with this second bounded attempt; no workflow case was promoted.

## 2026-09-06 20:40Z — Device Control initial-load performance gate recovered

Added a cached `/api/v1/devices?cached=true` path backed by the last observed actionable inventory and used it only for the Dashboard first paint; periodic refreshes and flow authoring retain the full transport probe. The cached endpoint measured ~6ms while the full probe remains ~6.1s. Device Control API package tests and UI validation remain green (`pnpm test`: 154, type-check, build). Test Genie performance run `20260906-203836-bdb0d920` passed all performance-health capabilities in 23 seconds. Receipt `execution-evidence/device-control-performance-testgenie-pass-receipt-20260906.json` is attached to DEL-16/finalRegression. Live platform and screenshot-capable-target gates remain open.
