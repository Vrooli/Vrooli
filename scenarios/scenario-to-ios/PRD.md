# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario scenario-to-ios`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Scenario to iOS is the delivery ramp that turns any Vrooli scenario into a distributable iOS application and proves — with evidence a reviewer can inspect — that the generated app behaves like a Vrooli app on iOS. It owns exactly one layer: build, package, sign, distribute, and gate. It does not drive devices (`device-control` does), does not automate web content (`browser-automation-studio` does), does not own device trust (`vrooli-bridge` does), and does not reimplement validation machinery (`packages/delivery-ramp-go` does). The ramp supplies four adapters to that spine — Prober, Builder, Driver, Distributor — and nothing else.

- **Purpose, restated for the constraint that shapes everything**: unlike every other ramp, this one cannot build on the host it usually runs on. The Apple toolchain exists only on macOS, so an iOS build is always executed on a macOS node reached through the bridge, and "iOS build and signing host" is a **node role** rather than a named machine. That indirection is not incidental complexity; it is what keeps the ramp alive when the machine currently filling the role stops qualifying.

- **Primary users/verticals**: The owner shipping a scenario to an iPhone or iPad; `deployment-manager`, which consumes this ramp's `common.v1.TargetVerdict` to compute a release gate; scenario authors whose existing BAS flows must replay unchanged inside a generated iOS app; and agents that need a reproducible path from scenario source to an installable artifact.

- **Deployment surfaces**: Go API over Connect-RPC exposing the target inventory, validation matrix, and readiness ladder; Go CLI as the complete control surface; React + Vite operator UI for the target matrix, run review, and the release-readiness walkthrough; and generated Capacitor Xcode projects as the product output.

- **Value promise**: A scenario becomes an installable iOS app without its UI being rewritten, because the Capacitor shell wraps the same web bundle every other surface already serves. The same BAS flow an author wrote once replays on iOS with no mobile-specific authoring. And the notoriously opaque path from "nothing" to "a person can install this" — enrollment, certificates, provisioning, bundle identity, TestFlight, review — becomes a probed ladder that names the next action at every rung.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Capacitor shell generation into a buildable Xcode project | Generate a complete Xcode project that wraps the target scenario's existing web bundle without modifying that scenario's UI source, with a reproducible bundle identifier derived from scenario identity. The shell is a pluggable template family so a second shell technology can be added later; only the Capacitor template ships at P0.
- [x] OT-P0-002 | Builder producing simulator and device artifacts with an asserted SDK floor | Produce a simulator `.app` and a device-installable IPA through `xcodebuild` on a macOS node. The build asserts the Xcode and iOS SDK version meet the current App Store Connect floor (Xcode 26 and the iOS 26 SDK) as a machine assertion recorded in evidence. A toolchain below the floor fails closed with the required version named, rather than producing an artifact that will be rejected at upload.
- [x] OT-P0-003 | Prober distinguishing categorically unsupported from currently unavailable | Implement the spine `Prober` seam so a Linux host reports native iOS targets as `unsupported` — terminal and correct forever, since no Apple toolchain exists for Linux — while a macOS node missing a prerequisite reports `unavailable` with the exact missing item. Probes Xcode version, installed simulator runtimes including the x86_64 universal architecture variant required on Intel hosts, GUI session availability, and signing identity presence.
- [x] OT-P0-004 | Build-host role with explicit toolchain expiry awareness | Model the iOS build and signing host as a bridge-node role satisfied by any qualifying node, never as a specific machine. Record the fulfilling node's macOS and Xcode versions in evidence and report when a host can still validate but can no longer produce submittable builds, so the loss of release capability is a visible state rather than a surprise at upload time.
- [x] OT-P0-005 | Driver: one chaptered timeline composing device-control and BAS | Implement the spine `Driver` seam by delegating native shell operations to device-control verbs — selecting the `ios-simctl`, `ios-xcuitest`, or `ios-mirror` strategy by what the cell must prove — and web-content steps to a browser-automation-studio flow, interleaved on a single chaptered timeline. The ramp implements no device driving of its own and holds a device-control lease for the duration of a run.
- [x] OT-P0-006 | iOS conformance journey suite | A registered, scenario-agnostic capability plan covering the iOS platform contract: install and cold start, permission-denial grace, background and resume, process-death restore, rotation and size class, keyboard avoidance, offline transition, universal link, notification tap, swipe-back navigation, update migration, and clean uninstall. Each chapter carries purpose, bounded readiness and settle policies, expected versus observed, and capture references. The runner has no scenario-name branch.
- [x] OT-P0-007 | Signing and provisioning with no key material in the repository | Manage certificates, provisioning profiles, and App Store Connect API keys exclusively through secrets-manager, referenced by identity at build time. No key, profile, or password exists in the repository, in a generated project, or in any evidence artifact. An absent signing identity reports `unavailable` naming the ladder rung that resolves it.
- [x] OT-P0-008 | Distributor: honest distribution channel model | Implement the spine `Distributor` seam over channels with genuinely different requirements: simulator-only (no signing, no distribution), development device (signing identity, device registration), TestFlight (enrollment, App Store Connect record, upload), and App Store (review submission). Each channel reports its own availability and reason; a channel is never presented as available because a sibling channel is.
- [x] OT-P0-009 | Release readiness ladder for Apple | A probed, ordered rung model from no account to App Store submission: Apple Developer Program enrollment, a qualifying macOS node with Xcode, certificates and provisioning, bundle ID and App Store Connect record, TestFlight upload, App Review submission. Each rung reports state, who can complete it, whether the ramp can automate it, and the next action. The same model backs the CLI, the UI walkthrough, and the generated documentation, so documentation cannot drift from code.
- [x] OT-P0-010 | Fail-closed release gate emitting reference-only verdicts | Compute the iOS release gate from the spine's immutable matrix and emit `common.v1.TargetVerdict` with `EvidenceRef` entries to deployment-manager. Capture bytes, recordings, certificates, and endpoints stay producer-owned. `unavailable` and `unsupported` are terminal and can never be promoted to a pass, and a missing required cell fails the gate rather than being omitted from it.
- [x] OT-P0-011 | CLI as the complete control surface | Every capability — generate, build, probe, run a matrix, wait, rerun, compare, inspect readiness, distribute — is reachable from the CLI with report-shaped JSON output, including when the actual work executes on a remote macOS node. A capability reachable only from the API or UI is treated as incomplete.
- [x] OT-P0-012 | hello-mobile fixture proves the ramp end to end over the bridge | The `hello-mobile` fixture is generated, built, installed, driven, and recorded on an iOS simulator hosted on a macOS bridge node, producing a complete conformance journey with useful decoded video and verified redaction, initiated from a Linux host that cannot itself build for iOS. This is simultaneously the ramp proof and the transport-symmetry proof.

P0 acceptance note: checked rows represent delivered capability contracts and
targeted Linux evidence. Apple-dependent build, simulator, signing, and
bridge-fixture execution remain explicitly `unavailable` until a trusted
macOS/Xcode bridge and Apple credentials are registered; Linux native iOS
targets remain terminally `unsupported`.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Physical iPhone conformance at semantic tier | Run the same conformance journey unchanged against a physical iPhone through the device-control `ios-xcuitest` strategy, with the adapter changing and the chapter set staying identical. Gated on Apple Developer Program enrollment and code signing; reports `unavailable` with that next action until enrollment exists.
- [ ] OT-P1-002 | Physical iPhone exploration via mirroring, explicitly non-promotable | Support the device-control `ios-mirror` strategy for physical iPhones, which requires no developer account and is therefore the first usable physical-iPhone path. Its runs are recorded as non-promotable for release purposes, because OCR reads pixels rather than meaning and cannot distinguish a correct rendering from a convincing image of one.
- [ ] OT-P1-003 | TestFlight upload automation | Automate IPA upload to TestFlight through the App Store Connect API once the ladder rungs it depends on are satisfied, with build notes derived from scenario metadata and an evidence record of the upload identity.
- [ ] OT-P1-004 | Operator UI for matrix, runs, and readiness | A dense operational surface showing the probed target matrix with honest dispositions including the terminal `unsupported` rows, run review with chaptered evidence and video, and the Apple readiness ladder as a walkthrough that names the next action at every rung.
- [ ] OT-P1-005 | Update and data-migration journey across versions | Install version N, exercise state, install version N+1 over it, and assert user data survived, with the prior version's artifact retained as evidence.
- [ ] OT-P1-006 | Performance budget chapters | Bounded, evidence-visible budgets for cold start time and artifact size, reported as assertions with observed values so a regression is a failed chapter rather than a subjective judgement.
- [ ] OT-P1-007 | App Store listing asset generation | Generate listing assets and screenshots from journey evidence at the required device sizes, leaving privacy nutrition labels and review correspondence as explicit owner actions.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Additional Apple form factors | Extend to iPadOS-specific layouts, visionOS, and watchOS targets, each carrying its own SDK floor and conformance chapters, using the same Prober and Driver seams rather than new engines.
- [ ] OT-P2-002 | Alternative shell template | Add a second shell technology to the template family for scenarios whose UX warrants a lower memory floor, without changing the ramp's adapter contract.
- [ ] OT-P2-003 | Multi-scenario bundle applications | Package several scenarios into one iOS application with a shared shell and per-scenario routing, so an owner ships a suite rather than one app per scenario.
- [ ] OT-P2-004 | Review submission and response tracking | Drive App Review submission through the App Store Connect API and track review state and rejection reasons as first-class records, so a rejection becomes an actionable state rather than an email.

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go API with Connect-RPC and screaming architecture; Go CLI; React + Vite + TypeScript UI on the `vrooli-default` design kit. The generated product is a Capacitor iOS project built with `xcodebuild` on a macOS node.

- **Shared machinery, not reimplemented**: `packages/delivery-ramp-go` owns the target inventory contract, the immutable validation matrix, local and bridge transports, the journey and evidence schemas, and the closed disposition vocabulary. This scenario implements only the exported `Prober`, `Builder`, `Driver`, and `Distributor` interfaces and never reaches into spine internals to express a platform difference.

- **Remote execution is the normal case, not the exception**: because no Apple toolchain runs on Linux, the bridge transport is this ramp's primary path rather than a fallback. The same ramp binary running on a macOS node serves a locally initiated request identically, so neither side is "the client." A dispatch that returns no target-owned evidence is never converted into a pass.

- **Data + storage**: SQLite for generated-project records, build records, matrix runs, and readiness state. Build artifacts and captures are producer-owned files referenced by checksum — never inlined into the database or into a consumer payload. Certificates, provisioning profiles, and App Store Connect keys live only in secrets-manager.

- **Integration strategy**: `device-control` for every device verb, selecting the `ios-simctl`, `ios-xcuitest`, or `ios-mirror` strategy by what a cell must prove; `browser-automation-studio` for web-content flows inside the app WKWebView; `vrooli-bridge` for macOS node reach and durable dispatch; `deployment-manager` as the evidence consumer and release-gate authority; `secrets-manager` for signing identity.

- **Xcode is a host capability, not a resource**: it cannot be installed, licensed, or versioned by us, so it is verified and instructed rather than acquired. Modelling it as a resource would place a permanently failing acquisition step in the resource fleet.

- **Non-goals / guardrails**: This ramp does not drive devices, does not automate web content, does not own device identity or trust, does not own the release gate's consumer side, does not acquire the Apple toolchain, and does not hold key material. It never infers platform support from `runtime.GOOS`, and it never treats mirror-derived evidence as release-gating.

## 🤝 Dependencies & Launch Plan

**Required resources**: none acquirable. Xcode and the macOS operating system are host capabilities on a bridge node, verified and instructed. SQLite otherwise.

**Scenario dependencies**: `device-control` (required — every device verb and all three iOS strategies), `vrooli-bridge` (required — macOS node reach; this ramp cannot function without it on a Linux host), `deployment-manager` (required — evidence consumer and gate), `browser-automation-studio` (required for scenario-owned web flows), `secrets-manager` (required for signing identity), `hello-mobile` (required as the conformance fixture).

**Operational risks**:

1. *The only available macOS node has a dated release capability.* `minimouse` is darwin/amd64 — Intel. macOS 26 Tahoe was the last Intel-supporting release and Xcode 27 dropped Intel entirely. It therefore satisfies the current App Store Connect floor (Xcode 26 and the iOS 26 SDK, mandatory since 28 April 2026) but will stop producing submittable builds once Apple raises that floor again, projected around April 2027 from their annual cadence. Mitigation is structural: the build host is a node role, so an Apple Silicon machine later is a node registration rather than a rewrite. Validation capability is unaffected and is tracked separately from release capability for exactly this reason.
2. *No Apple Developer Program enrollment exists.* Signing, physical-device provisioning, TestFlight, and the App Store are all gated behind it, and `ios-xcuitest` cannot sign WebDriverAgent without it. Every dependent capability must report `unavailable` with enrollment as the next action rather than failing obscurely. The `ios-mirror` strategy is the deliberate escape hatch that needs no enrollment.
3. *Simulator runtimes are not automatically available on Intel hosts.* The x86_64 universal architecture variant must be fetched explicitly, so runtime availability is a probe result rather than an assumption.
4. *Simulator work requires a logged-in GUI session.* The bridge agent already installs on macOS as a LaunchAgent bootstrapped into the `gui/<uid>` domain and documents that SSH-only sessions have no GUI bootstrap, so the prerequisite is understood — but it must be probed on the live node, not assumed from the code.
5. *Lease collisions on a shared device.* A deployment journey and a general-automation task targeting the same iPhone will corrupt each other's evidence silently unless every run holds a device-control lease for its full duration. The mirroring strategy is physically single-session, which makes this failure mode certain rather than merely likely.

**Launch sequencing**: the Prober first, since correctly reporting `unsupported` on Linux and probing a macOS node is the precondition for everything else; then the Builder and bridge-dispatched simulator validation with the `hello-mobile` fixture as the proof; then the conformance journey suite; then signing and the distribution channel model; then the readiness ladder; then physical-device targets as enrollment allows; then TestFlight automation and the operator UI.

## 🎨 UX & Branding

- **Look and feel**: Vrooli Operational Console — dense, professional, information-first. The primary surface is the target matrix answering "which iOS targets can I prove right now, what did the last run show, and what is standing between me and shipping this." Because this ramp has permanently terminal rows, the matrix must make `unsupported` visually distinct from `unavailable` — one is a fact of the world, the other is a task.

- **Accessibility**: full keyboard operation, visible focus states, semantic roles and names, and stable `data-testid` selectors. Journey video is evidence, not the interface — every run verdict, chapter assertion, and readiness rung must be perceivable and operable without playing a recording. Colour never carries disposition alone.

- **Voice and messaging**: state capability in terms of what has been probed. Never claim a target supports something that was not proven. An `unavailable` disposition always names the missing capability and the next action; an `unsupported` disposition says plainly that it is terminal rather than implying a future fix. Never imply that a mirror-derived run could gate a release.

- **Readiness walkthrough**: the Apple readiness ladder is a first-class product surface, not documentation. Apple's path is the most opaque part of shipping an iOS app, so the walkthrough shows each rung's state, its cost, whether the ramp can complete it, and what the owner must do personally — enrollment, identity verification, and payment are irreducibly human steps and the surface should say so plainly rather than implying automation that cannot exist.

- **Expiry surfacing**: when the fulfilling build host can validate but no longer produce submittable builds, that must read as a distinct, clearly explained state — not as a generic failure and not as a silent pass.

- **Branding hooks**: standard `vrooli-default` kit. Keep the seeded PWA manifest, service worker, maskable icons, and safe-area tokens valid; replace the generic icons when product branding exists. Generated iOS applications carry the target scenario's branding, not this ramp's.
