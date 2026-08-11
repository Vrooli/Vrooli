# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario scenario-to-android`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Scenario to Android is the delivery ramp that turns any Vrooli scenario into a distributable Android application and proves — with evidence a reviewer can inspect — that the generated app behaves like a Vrooli app on Android. It owns exactly one layer: build, package, sign, distribute, and gate. It does not drive devices (`device-control` does), does not automate web content (`browser-automation-studio` does), does not own device trust (`vrooli-bridge` does), and does not reimplement validation machinery (`packages/delivery-ramp-go` does). The ramp supplies four adapters to that spine — Prober, Builder, Driver, Distributor — and nothing else.

- **Primary users/verticals**: The owner shipping a scenario to a phone; `deployment-manager`, which consumes this ramp's `common.v1.TargetVerdict` to compute a release gate; scenario authors who need their existing BAS flows to replay unchanged inside a generated Android app; and agents that need a reproducible path from scenario source to an installable artifact.

- **Deployment surfaces**: Go API over Connect-RPC exposing the target inventory, validation matrix, and readiness ladder; Go CLI as the complete control surface; React + Vite operator UI for the target matrix, run review, and the release-readiness walkthrough; and generated Capacitor Android projects as the product output.

- **Value promise**: A scenario becomes an installable Android app without its UI being rewritten, because the Capacitor shell wraps the same web bundle the browser and desktop ramps already serve. The same BAS flow an author wrote once replays on Android with no mobile-specific authoring. And the path from "nothing" to "a person can install this" is a probed, walked-through ladder rather than tribal knowledge — including the Play Console registration, developer verification, signing, and target-API obligations that gate real distribution.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Capacitor shell generation from any scenario UI | Generate a complete, buildable Capacitor Android project that wraps the target scenario's existing web bundle without modifying that scenario's UI source. The shell is a pluggable template family so a second shell technology can be added later without re-architecting the ramp; only the Capacitor template ships at P0. Generation is reproducible: the same scenario at the same revision produces the same project.
- [ ] OT-P0-002 | Gradle build to APK and AAB with asserted target API | Produce both a debug-signable APK for direct install and an AAB for Play distribution. The build asserts `targetSdk` is at or above the current Play floor (API 36) as a machine assertion recorded in evidence, not as a comment in a Gradle file. A build below the floor fails closed with the required level named.
- [ ] OT-P0-003 | Prober: probed Android target inventory | Implement the spine `Prober` seam so target availability is decided by probing, never by operating-system name. Reports the emulator target on this host (KVM presence and writability, SDK, platform-tools, system image, AVD), emulator targets on bridge nodes, and physical devices surfaced through device-control. A capability that cannot be proven reports `unavailable` with the exact missing prerequisite and a next action.
- [ ] OT-P0-004 | Driver: one chaptered timeline composing device-control and BAS | Implement the spine `Driver` seam by delegating native shell operations (install, launch, permissions, background, rotate, uninstall) to device-control verbs and web-content steps to a browser-automation-studio flow, interleaved on a single chaptered timeline. The ramp implements no device driving of its own and holds a device-control lease for the duration of a run.
- [ ] OT-P0-005 | Android conformance journey suite | A registered, scenario-agnostic capability plan covering the Android platform contract: install and cold start, permission-denial grace, background and resume, process-death restore, rotation, keyboard avoidance, offline transition, deep link, notification tap, back navigation, update migration, and clean uninstall. Each chapter carries purpose, bounded readiness and settle policies, expected versus observed, and capture references. The runner has no scenario-name branch.
- [ ] OT-P0-006 | android-sdk resource consumption and AVD lifecycle | Consume the governed `android-sdk` resource for SDK, platform-tools, emulator, and system images rather than acquiring toolchain bytes in this scenario. Own the AVD lifecycle needed for a validation run — create, boot to a bounded readiness signal, run, and tear down — and report an absent or unusable toolchain as `unavailable` with the acquisition next action.
- [ ] OT-P0-007 | Signing with no key material in the repository | Generate and store the upload key through secrets-manager and sign build outputs by reference. No keystore, password, or key material exists in the repository, in a generated project, or in any evidence artifact. A missing signing identity reports `unavailable` with the ladder rung that would resolve it.
- [ ] OT-P0-008 | Distributor: honest distribution channel model | Implement the spine `Distributor` seam over three distinct channels with different identity and evidence requirements: Play (verified developer, AAB, review), verified sideload (developer verification, direct APK), and ADB internal (no verification, host-tethered). Each channel reports its own availability and reason. A channel is never presented as available because a sibling channel is.
- [ ] OT-P0-009 | Release readiness ladder for Google | A probed, ordered rung model from no account to production listing: Play Console registration, developer verification, signing key and Play App Signing, target API compliance, internal testing track, production listing. Each rung reports state, who can complete it, whether the ramp can automate it, and the next action. The same model backs the CLI, the UI walkthrough, and the generated documentation, so documentation cannot drift from code.
- [ ] OT-P0-010 | Fail-closed release gate emitting reference-only verdicts | Compute the Android release gate from the spine's immutable matrix and emit `common.v1.TargetVerdict` with `EvidenceRef` entries to deployment-manager. Capture bytes, recordings, keys, and endpoints stay producer-owned. `unavailable` and `unsupported` are terminal states that can never be promoted to a pass, and a missing required cell fails the gate rather than being omitted from it.
- [ ] OT-P0-011 | CLI as the complete control surface | Every capability — generate, build, probe, run a matrix, wait, rerun, compare, inspect readiness, distribute — is reachable from the CLI with report-shaped JSON output. A capability reachable only from the API or UI is treated as incomplete, because the CLI is what agents and CI drive.
- [ ] OT-P0-012 | hello-mobile fixture proves the ramp end to end | The `hello-mobile` fixture scenario is generated, built, installed, driven, and recorded on the local emulator, producing a complete conformance journey with useful decoded video and verified redaction. This is the proof that the ramp works without depending on any product scenario's correctness.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Physical-device conformance at semantic tier | Run the same conformance journey unchanged against a physical Android phone through the device-control `android-adb` strategy, with the adapter changing and the chapter set staying identical. Supports both USB and wireless transports, defaulting release-relevant runs to USB so evidence cannot fail for network reasons.
- [ ] OT-P1-002 | Bridge-dispatched validation on a remote node | Dispatch an Android validation cell to a bridge node and receive evidence references back, proving the transport symmetry: the same ramp binary serves a local request here and a dispatched request on `minimouse`, with neither side being the client.
- [ ] OT-P1-003 | Play internal testing track upload | Automate AAB upload to the Play internal testing track once the ladder rungs it depends on are satisfied, including release notes derived from scenario metadata and an evidence record of the upload identity.
- [ ] OT-P1-004 | Operator UI for matrix, runs, and readiness | A dense operational surface showing the probed target matrix with honest dispositions, run review with chaptered evidence and video, and the release-readiness ladder as a walkthrough that names the next action at every rung.
- [ ] OT-P1-005 | Update and data-migration journey across versions | Install version N, exercise state, install version N+1 over it, and assert user data survived — the mobile analog of the desktop updater journey, with the prior version's artifact retained as evidence.
- [ ] OT-P1-006 | Performance budget chapters | Bounded, evidence-visible budgets for cold start time and artifact size, reported as assertions with observed values so a regression is a failed chapter rather than a subjective judgement.
- [ ] OT-P1-007 | Play production listing asset generation | Generate store listing assets, screenshots from journey evidence, and a privacy policy from scenario metadata, leaving the data-safety declaration as an explicit owner action.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Additional Android form factors | Extend to Wear OS, Android TV, and Android XR targets, which carry their own Play target-API floors and their own conformance chapters, using the same Prober and Driver seams rather than new engines.
- [ ] OT-P2-002 | Play Integrity and attestation evidence | Record device and app integrity verdicts as evidence so a release gate can distinguish a run on an attested device from a run on an unverified one.
- [ ] OT-P2-003 | Alternative shell template | Add a second shell technology to the template family — a lower-memory native shell — for scenarios whose UX warrants it, without changing the ramp's adapter contract.
- [ ] OT-P2-004 | Multi-scenario bundle applications | Package several scenarios into one Android application with a shared shell and per-scenario routing, so an owner ships a suite rather than one app per scenario.

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go API with Connect-RPC and screaming architecture; Go CLI; React + Vite + TypeScript UI on the `vrooli-default` design kit. The generated product is a Capacitor Android project built with Gradle and the Android Gradle Plugin on JDK 17.

- **Shared machinery, not reimplemented**: `packages/delivery-ramp-go` owns the target inventory contract, the immutable validation matrix, local and bridge transports, the journey and evidence schemas, and the closed disposition vocabulary. This scenario implements only the exported `Prober`, `Builder`, `Driver`, and `Distributor` interfaces and never reaches into spine internals to express a platform difference.

- **Data + storage**: SQLite for generated-project records, build records, matrix runs, and readiness state. Build artifacts and captures are producer-owned files referenced by checksum — never inlined into the database or into a consumer payload. Signing material lives only in secrets-manager.

- **Integration strategy**: `device-control` for every device verb, holding a lease for the duration of a run; `browser-automation-studio` for web-content flows inside the app WebView; `vrooli-bridge` for remote target reach and durable dispatch; `deployment-manager` as the evidence consumer and release-gate authority; `android-sdk` as the governed toolchain resource; `secrets-manager` for signing identity.

- **Why Capacitor**: the scenario UI is already React and Vite on every other surface, so a WebView shell wraps the existing bundle with no per-scenario UI rewrite, and the same BAS flow replays unchanged. The known trade is a higher memory floor than a native shell, which is why the shell is a pluggable template family rather than a hard-coded choice.

- **Non-goals / guardrails**: This ramp does not drive devices, does not automate web content, does not own device identity or trust, does not own the release gate's consumer side, does not acquire toolchain bytes, and does not hold key material. It never infers platform support from `runtime.GOOS`, and it never converts a bridge dispatch into a pass without target-owned evidence.

## 🤝 Dependencies & Launch Plan

**Required resources**: `android-sdk` (governed resource, to be created) for SDK, platform-tools, emulator, and system images. SQLite otherwise. JDK 17 and `ffmpeg` are host prerequisites already present on the development host; `/dev/kvm` is required for usable emulator performance and is probed, not assumed.

**Scenario dependencies**: `device-control` (required — every device verb), `vrooli-bridge` (required — remote target reach and trust), `deployment-manager` (required — evidence consumer and gate), `browser-automation-studio` (required for scenario-owned web flows), `secrets-manager` (required for signing identity), `hello-mobile` (required as the conformance fixture).

**Operational risks**:

1. *Emulator performance collapse without KVM.* Hardware acceleration is the difference between a ~30-second boot and an unusable 3–5 FPS. Video evidence captured without acceleration is worthless, so KVM must be a probed capability whose absence yields `unavailable` rather than a slow pass.
2. *Dated external obligations.* Play requires target API 36 for new apps and updates from 31 August 2026, with an extension path to 1 November 2026. Google developer verification begins enforcement on certified devices in Brazil, Indonesia, Singapore, and Thailand on 30 September 2026 and globally through 2027 — and it gates sideloading, not just Play. Distribution assumptions that ignore these dates expire.
3. *Wireless ADB flakiness surfacing as evidence failure.* Wireless pairing expires on reboot and the link is less reliable than USB. Without defaulting release-relevant runs to USB, intermittent transport faults present as failed conformance chapters.
4. *Lease collisions on a shared device.* A deployment journey and a general-automation task targeting the same phone will corrupt each other's evidence silently unless every run holds a device-control lease for its full duration.

**Launch sequencing**: the four spine adapters against the local emulator first, with the `hello-mobile` fixture as the proof; then the conformance journey suite; then signing and the distribution channel model; then the readiness ladder; then physical-device and bridge-dispatched targets; then Play upload automation and the operator UI.

## 🎨 UX & Branding

- **Look and feel**: Vrooli Operational Console — dense, professional, information-first. The primary surface is the target matrix answering "which Android targets can I prove right now, what did the last run show, and what is standing between me and shipping this." Status is encoded in form as well as text so an `unavailable` cell reads at a glance.

- **Accessibility**: full keyboard operation, visible focus states, semantic roles and names, and stable `data-testid` selectors. Journey video is evidence, not the interface — every run verdict, chapter assertion, and readiness rung must be perceivable and operable without playing a recording. Colour never carries disposition alone.

- **Voice and messaging**: state capability in terms of what has been probed. Never claim a target supports something that was not proven. An `unavailable` disposition always names the missing capability and the next action; an `unsupported` disposition says plainly that it is terminal rather than implying a future fix.

- **Readiness walkthrough**: the release-readiness ladder is a first-class product surface, not documentation. It shows each rung's state, its cost, whether the ramp can complete it, and what the owner must do personally — because several rungs require a government ID or a payment method and no amount of automation removes that.

- **Branding hooks**: standard `vrooli-default` kit. Keep the seeded PWA manifest, service worker, maskable icons, and safe-area tokens valid; replace the generic icons when product branding exists. Generated Android applications carry the target scenario's branding, not this ramp's.
