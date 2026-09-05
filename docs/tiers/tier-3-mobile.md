# Tier 3 mobile delivery

Evidence and constraints below are current as of **2026-08-17**. They are
host/device observations, not claims that Apple hardware or release signing
exists.

Tier 3 mobile delivery turns a scenario's existing web bundle into an
installable application and proves its behavior on the target device. It is
two platform ramps sharing one delivery spine, one fixture, and one evidence
contract:

| Ramp | Shell/toolchain | Native target owner | Host constraint |
|---|---|---|---|
| `scenario-to-android` | Capacitor + Android SDK/Gradle | `device-control` Android strategies | Build is cross-platform; emulator/phone legs require a capable host |
| `scenario-to-ios` | Capacitor/WKWebView + Xcode | `device-control` iOS strategies | Build, simulator, and signing run on a trusted macOS host |

`packages/delivery-ramp-go` owns the provider-neutral `Prober`, `Builder`,
`Driver`, and `Distributor` seams, immutable validation matrix, journey
evidence, dispositions, transports, and reference-only verdicts. Each ramp
implements those seams and does not duplicate them.

Android journey evidence records explicit start and end host/device clock
samples through the device-control lease. Logcat `ActivityTaskManager` and
WebView events are correlated only after applying the measured offset; missing
launch/render events fail their assertions rather than becoming inferred
passes. Native action outcomes remain separately labelled as device-control
observations.

The retained Galaxy A03s proof is dated 2026-08-15; its `offline_transition`
chapter remains unavailable because the handset lacks `network-control`.

Physical Android journeys use one bounded recording per chapter. The same
device-control producer can finalize those retained chapter references into a
normalized, redacted review recording; the journey publishes both the
chapter-to-video mapping and the review reference without exposing capture
bytes or host paths to the ramp.

The ramp finds a serving host by probing capability. If the local host cannot
serve a device cell, `vrooli-bridge` discovers and dispatches to a trusted
node; a dispatch without target-owned evidence is never a pass. `device-control`
owns device verbs, leases, capability self-tests, and capture production.
Browser Automation Studio owns the scenario's web-content flow, attached to
the already-running mobile WebView. Native shell chapters and BAS steps are
interleaved on one monotonic `journey-evidence.v2` timeline, with recording
offsets and reference-only evidence sent to `deployment-manager`.

Android uses the governed `android-sdk` resource for platform-tools,
cmdline-tools, the emulator, and an API 36+ system image. iOS deliberately
declares no resource: Xcode is a licensed host capability and the iOS ramp
selects a macOS bridge node instead. Neither ramp creates an operator signing
identity during validation; release readiness reports the missing Google or
Apple signing/account rung explicitly.

## Android delivery-ramp proof status

The implementation has been exercised against the physical Galaxy A03s
(`ro.serialno=R9TT608Q6MH`) over wireless ADB. The device-control registry now
keys the device by hardware serial and retains the current endpoint separately;
reconnect waits for the transport before re-verifying identity. The Android
builder produced both APK and AAB outputs with `targetSdk=36`, and a real
`hello-mobile` build carried brand-manager assignment
`eb63f611-1be6-4801-9e06-104da7af8f09` into the generated project.

Targeted unit and package suites are green for device-control, the Android
journey/build/matrix code, and `packages/delivery-ramp-go`. The physical
matrix reached install, launch, logcat capture, and recording on the phone.
The complete executable fixture chapters are proven by matrix
`run-b556ab488cfe12f571be7379` using the active profile-backed unlock and the
Galaxy A03s evidence path. The required `offline_transition` chapter remains
honestly unavailable because this handset does not expose network-control;
the physical cell does not claim that coverage. This is an unavailable
hardware capability, not a passing or degraded journey verdict. A later
targeted matrix, `run-b91a7dd7f47c66e93aaad4c7`, used the governed
`vrooli-api36` emulator to satisfy the required journey/profile gate; the gate
records `android-04ab3fc382bfb33e` as the satisfying target while retaining the
physical cell's fail-closed result. Its review recording is
`/home/matthalloran8/.vrooli/data/vrooli/device-control/evidence/6d1e8272-e0c1-4779-ae8a-519b6e40c893.bin`
(`47.240000` seconds, SHA-256
`da0be80ff778b6d38422ca6b3de2f7fe14994fc86482715a0e2c5bbffcf350`). Sources: `scenarios/device-control/docs/internal/PROBLEMS.md`,
`scenarios/scenario-to-android/docs/internal/PROBLEMS.md`, and the retained
matrix identity above.

A later physical retry (`run-f22e9273e238472801819a9b`) failed closed at the
unlock boundary after the stored credential was rejected; no credential was
guessed or persisted. The emulator gate therefore remains the current
qualifying Android evidence, while the phone requires an authorized unlock
credential before another physical run.

Unsupported and unavailable are terminal dispositions, not degraded passes.
The mobile release gate fails closed when a required cell is missing, and
each channel (Play, verified sideload, and ADB internal on Android) reports
its own availability and next action.

The shared required-cell gate is matrix-wide: a required journey/profile is
satisfied by any available target that advertises the journey's required
capabilities. The gate records the satisfying target ID, so emulator coverage
cannot be mistaken for physical-device coverage. If no target can satisfy the
capability, the gate remains fail-closed.

## iOS Linux-testable frontier

`scenario-to-ios` generates a deterministic Capacitor/Xcode project on Linux
and exposes target, conformance, readiness, build, and distribution reports
through its API, CLI, and dashboard. Linux reports the native iOS simulator as
`unsupported`; a missing macOS bridge, Xcode toolchain, signing identity, or
Apple Developer enrollment is `unavailable` with a named next action. No
Apple-dependent path fabricates a pass. Runtime iOS evidence and the
`hello-mobile` bridge proof remains pending until a trusted macOS bridge is
registered. The prior Linux iOS suite `20260817-134040-f5c5e378` passed
21/21; the fresh current-checkout run `20260817-153455-3474647a` executed all
21 phases with 20 passing and only the shared `storage` phase failing without
a scenario finding. Native Apple execution, signing, and physical fixture
evidence remain unavailable. The current Phase 9 supporting evidence is
mixed: `device-control` run `20260817-151350-c3ef0da7` reached 22 phases with
20 passing and retained shared workflow/UI-health findings; `hello-mobile` run
`20260817-135349-efb7c62c` stopped before phase execution because the shared
lifecycle could not read `resources/sherpa-onnx/resource.json`; and the
no-regression desktop run `20260817-141430-da7c2997` reached 20/22 with
dependency/security findings. The fresh Android suite
`20260817-154407-ecdc014b` passed all 21/21 phases, providing the current
integrated Android counterpart.
These constraints are recorded in the owning scenario ledgers rather than
being treated as mobile capability passes.
Sources: `scenarios/scenario-to-ios/docs/reference/apple-hardware-activation.md`,
`scenarios/scenario-to-ios/docs/internal/PROBLEMS.md`, and the fresh run
identity above.
