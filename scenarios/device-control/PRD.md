# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario device-control`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Device Control is the permanent capability for driving owner-trusted devices — physical phones, emulators and simulators, and host machines — through pluggable control strategies. It separates *what you want done on a device* from *how a particular device can be driven*, so one automation flow runs unchanged across an Android phone over ADB, an iOS simulator over simctl, an iPhone over accessibility-grade XCUITest, an iPhone over pixel-grade mirroring, or the local desktop. Vision-based screen understanding and control are built once against a minimal capability floor, so they work on every strategy — including strategies that expose no semantic accessibility tree at all.

- **Primary users/verticals**: The owner operating their own devices; the mobile delivery ramps (`scenario-to-ios`, `scenario-to-android`) which need conformance journeys on emulators and on physical hardware; agents performing goal-directed device automation; and any future scenario that must observe or actuate a device whose software it does not own.

- **Deployment surfaces**: Go API over Connect-RPC; Go CLI, which is the complete control surface because it is what the agent skill drives; React + Vite operator UI for device inventory, live session view, flow authoring and run review; and a durable evidence surface conforming to the shared `common/v1` evidence contract.

- **Value promise**: Devices become a first-class governed capability rather than a per-scenario side effect. A new control strategy is a small declared adapter, not a fork of the automation engine. An automation written once keeps working when the device, transport, or strategy changes. And an agent that works a task out once can have that run promoted into a deterministic flow that replays at no inference cost.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Capability floor and probed strategy contract | Every strategy implements identity and describe() and may add optional modalities — observe/frames, input actuation, media, properties, sensors, logs, semantic trees, app lifecycle, permissions, network control, orientation, clipboard, file transfer, recording, and WebView attach. Declarations are verified by probing at inventory time; an unprovable capability reports unavailable with the exact missing prerequisite and a next action, never silently present. Screenless devices are valid strategies and do not claim observe or screenshot.
- [x] OT-P0-002 | Strategy conformance suite | `device-control strategy verify <id>` runs a fixed suite against any registered strategy and reports which capability tiers it satisfies and therefore which flow step kinds it can execute. Adding a strategy means implementing the floor and passing the suite — no engine changes.
- [x] OT-P0-003 | Bridge-backed device inventory and reachability | Devices are enumerated from the vrooli-bridge fleet, which owns identity, pairing, trust, scopes and audit; this scenario owns device-specific probing and health. An attached device whose host node is offline reports unreachable with that exact reason rather than disappearing.
- [x] OT-P0-004 | Session leases and single-session safety | No verb executes without a held lease. Concurrent claims on the same device are refused rather than interleaved, because several strategies are physically single-session. Leases expire, are visible, and are releasable.
- [x] OT-P0-005 | Flow schema, bounded waits, and pre-execution capability gap report | One flow envelope with composable step vocabularies: device verbs, AI-assisted steps, control flow, bounded readiness and settle policies, and delegation to a browser-automation-studio flow. Steps declare required capabilities so a flow is checked against a strategy before execution and yields a capability gap report instead of a mid-run failure. Waits are named bounded policies with upper bounds; an exceeded bound is an evidence-visible failure, never a longer sleep.
- [x] OT-P0-006 | Target resolution ladder | A step names its target by intent. The resolver satisfies it at the highest rung the strategy supports: semantic match against an accessibility or view tree; visual-anchor match against a captured reference; or vision through ai-gateway. The chosen rung and its confidence are recorded in evidence, so a reviewer can see whether a result was deterministic or inferred.
- [x] OT-P0-007 | All inference routed through ai-gateway | Every AI-assisted step routes through ai-gateway. No direct provider HTTP client, no concrete model slug, and no provider secret exists in this scenario. Visual understanding uses the `locate.visual` role through the generated ai-gateway Connect client; unavailable routes are typed and never bypass the boundary.
- [x] OT-P0-008 | Evidence references with verified redaction | Captures are stored as references carrying checksum, size and creation time, conforming to common/v1 EvidenceRef. Screen frames can contain secrets, so redaction status is verified before any capture leaves the producer, and raw bytes are never handed to a consumer.
- [x] OT-P0-009 | Audit trail and immediate session kill | Every verb dispatched to a device is audited with actor, device, lease and outcome. A live session is visible in the operator surface and can be killed immediately from CLI or UI.
- [x] OT-P0-010 | CLI as the complete control surface | Every capability is reachable from the CLI with report-shaped JSON output, because the CLI is what the agent skill drives. A capability reachable only from the API or UI is treated as incomplete.
- [x] OT-P0-011 | android-adb strategy | First full-tier strategy, serving emulator and physical Android devices identically through one adapter: semantic view hierarchy, app lifecycle, permission control, input, screenshot and screen recording. Proves the floor, the ladder and the conformance suite against a real device class.
- [x] OT-P0-012 | Honest recording where a frame modality exists | A strategy that declares native recording uses it; a strategy that exposes frames but not native recording may synthesize a labeled recording from its observe stream. Screenless transports declare screenshot and recording unavailable with a reason. Evidence records the method and effective frame rate, and never labels a synthesized or low-rate capture as native.
- [x] OT-P0-013 | Secure authentication profiles and verified unlock | Device-control stores only authentication-profile metadata and credential-authority references, performs bounded Android numeric unlock attempts without secret-bearing commands, and reports success only after a fresh live keyguard probe proves the device is unlocked. Unsupported and human-gated methods fail closed with typed outcomes, and flows refuse locked or unknown visible surfaces.
- [ ] OT-P0-014 | Measurable agent reuse and improvement | When an agent controls an authorized device, device-control shall resolve its durable identity, reuse persisted validated flows, support evidence-backed promotion and version-preserving repair, and expose attributable learning and agent-effort measurements.

- [ ] OT-P0-015 | Portable desktop user-session control | When an authorized desktop session is selected, Device Control shall verify native observation, Unicode input, semantic actions, and geometry on Windows, macOS, Linux X11, GNOME Wayland, and KDE Wayland, with explicit unavailable or denied results for unsupported operations.
- [ ] OT-P0-016 | Destination authority and durable action receipts | When desktop control is admitted, Device Control shall bind actor, target, user session, capabilities, expiry, and epoch; reject stale or conflicting commands; reconcile uncertain effects; and release held input on stop, lock, takeover, or helper loss.
- [ ] OT-P0-017 | Verified portable desktop procedures | When a desktop procedure is promoted or repaired, Device Control shall require complete outcome assertions and exact replay, preserve acceptance and version identity, and bound vision and demonstration authoring through governed operations.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | ios-simctl strategy | Drive iOS simulators on a macOS node through simctl and XCUITest: install, launch, input, semantic tree, screenshot and video capture.
- [ ] OT-P1-002 | ios-xcuitest strategy | Drive physical iPhones through devicectl and WebDriverAgent at semantic tier. Gated on Apple Developer Program enrollment and code signing; reports unavailable with that next action until enrollment exists.
- [ ] OT-P1-003 | ios-mirror strategy | Drive physical iPhones through iPhone Mirroring with synthetic HID events and OCR. Floor tier only and requires no developer account, which makes it the first usable physical-iPhone path. Explicitly non-promotable for release evidence because OCR reads pixels rather than meaning.
- [x] OT-P1-004 | host-desktop strategy | Drive the local machine as a device, reusing existing headless display and input tooling so desktop automation composes with the same flow vocabulary.
- [x] OT-P1-005 | Agent mode via agent-manager and a prompt-manager skill | `device-control agent run --device <id> --goal "<goal>"` uses the prompt-manager device-control skill and routes any inference through ai-gateway. Runs build a typed capability/state world model, are bounded by lease scope, fully audited, abortable, and remain valid when the target has no frame modality.
- [x] OT-P1-006 | Promote an agent run to a deterministic flow | Every agent action is recorded as a flow step, so a successful goal-directed run exports as a deterministic flow that replays without inference cost. This is the compounding loop: infer once, replay free.
- [ ] OT-P1-007 | Browser-automation-studio delegation step | A flow step attaches to an application WebView and hands control to a named BAS flow, so native shell automation and web content automation compose on a single timeline with one evidence record.
- [ ] OT-P1-008 | Operator UI | Live device view with session controls, flow authoring and validation, run review with chaptered evidence, capability-composed television/phone/property panels, LAN discovery and pairing, and a strategy conformance matrix showing what each device can actually prove right now.
- [ ] OT-P1-009 | Visual-anchor library | Captured reference images with stable identity so common targets resolve deterministically at the middle rung of the ladder, without inference cost and without a semantic tree.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Multi-device flows | A single flow spanning two or more devices for cross-device handoff, with per-device leases held for the flow duration.
- [ ] OT-P2-002 | Scheduled and event-triggered flows | Run a flow on a schedule or in response to an event, with the same lease, audit and evidence guarantees as an interactive run.
- [ ] OT-P2-003 | Additional device classes | Extend beyond phones and hosts as new device classes become trusted, using the same floor and conformance suite rather than new engines.
- [ ] OT-P2-004 | Flow sharing with carried capability requirements | Export and import flows between owners so the declared capability requirements travel with the flow and are checked against the importing fleet before a run is attempted.

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go API with Connect-RPC and screaming architecture; Go CLI; React + Vite + TypeScript UI on the `vrooli-default` design kit. Strategy adapters are Go packages behind one interface plus a declared manifest.
- **Data + storage**: SQLite for flow definitions, run history, capability snapshots, lease records and audit. Capture artifacts are producer-owned files referenced by checksum — never inlined into the database or into a consumer payload.
- **Integration strategy**: `vrooli-bridge` for device reach, trust, scopes and durable remote dispatch. `ai-gateway` for all inference including visual understanding. `agent-manager` for goal-directed runs. `prompt-manager` for the operator skill. `browser-automation-studio` for web-content delegation. `deployment-manager` consumes evidence indirectly through the delivery ramps rather than directly.
- **Non-goals / guardrails**: This scenario does not own device identity, pairing or trust — bridge does. It does not own web-content automation — BAS does. It does not own release gating — the delivery ramps do. It does not implement its own inference client. It does not move files between devices — `device-sync-hub` does. It does not accept a control request without a lease.

## 🤝 Dependencies & Launch Plan

**Required resources**: none at launch beyond SQLite. The Android toolchain arrives as a separate governed `android-sdk` resource consumed by the `android-adb` strategy. Xcode is a host capability that is verified and instructed, never installed by us.

**Scenario dependencies**: `vrooli-bridge` (required — device reach and trust), `ai-gateway` (required for AI-assisted steps), `agent-manager` (required for agent mode), `prompt-manager` (required for the operator skill), `browser-automation-studio` (optional — the delegation step).

**Operational risks**:

1. *Vision requests cross the ai-gateway boundary.* The `locate.visual` role accepts image attachments and returns canonical bounds. The caller still owns downscaling and device-coordinate conversion; browser-automation-studio's existing direct-provider vision path remains outside this scenario and is a separate conformance concern. This scenario never repeats that shortcut.
2. *Capability blast radius.* A capability that can tap anything on a personal phone and read anything on its screen deserves credential-grade treatment. Without scoped grants, live-session visibility and a hard kill, it is not safe to leave running.
3. *Silent session collision.* Several strategies are physically single-session. A missing lease does not present as a clean error; it presents as two runs quietly corrupting each other's evidence.

**Launch sequencing**: capability floor and strategy contract first; then bridge-backed inventory and leases; then the `android-adb` strategy as the first full-tier proof; then the flow schema and executor with the resolution ladder; then the remaining strategies; then agent mode.

## 🎨 UX & Branding

- **Look and feel**: Vrooli Operational Console — dense, professional, information-first. The primary surface is a fleet view answering "which devices do I control, what can each actually do right now, and what is running on them."
- **Accessibility**: full keyboard operation, visible focus, semantic roles and names, and stable `data-testid` selectors. A live device session must be operable and its state perceivable without relying on the video frame alone — the frame is evidence, not the interface.
- **Voice and messaging**: state capability in terms of what is provable. Never claim a device supports something that was not probed. An unavailable capability always names the missing prerequisite and the next action.
- **Safety surfacing**: a live control session must be unmistakable, with the holding consumer, the lease expiry, and a one-click kill always visible.
- **Branding hooks**: standard `vrooli-default` kit; keep the seeded PWA manifest, service worker and safe-area tokens valid, and replace the generic icons when product branding exists.
