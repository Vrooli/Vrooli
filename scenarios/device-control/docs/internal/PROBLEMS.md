# Problems — Device Control

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-17 — Shared workflow provider still cannot resolve the self route

**Symptom:** The fresh Phase 9 run `20260817-112516-aad4c0fe` passes the
device-control implementation phases but fails its BAS workflow cases when
`@scenario/self` is resolved for API port discovery.

**Root cause:** The shared workflow provider sends the literal placeholder to
runtime discovery without binding it to the current Test Genie target
scenario. This is the same provider defect recorded below, reproduced after
the device-control UI and performance findings were repaired.

**Workaround:** Use targeted API, UI, Lighthouse, and device-control evidence
checks while Test Genie/BAS is under repair. Keep the portable `@scenario/self`
token in authored cases; do not hard-code a runtime port in this scenario.

**Real fix:** workflow-health/Test Genie must bind `@scenario/self` to the run
target before querying `API_PORT`.

**Owner:** workflow-health/Test Genie maintainers.

**Refs:** run `20260817-112516-aad4c0fe`; `bas/cases/02-operator/`;
`performance-health lighthouse run device-control`.

### 2026-08-17 — Product pages are specified ahead of the current router

**Symptom:** Experience reconciliation cannot join bindings for the fleet,
device-detail, strategies, session, run-review, agent, or settings pages.

**Root cause:** Those documents describe the intended device-control product,
but the current UI router implements only dashboard, flows, evidence, and
settings presentation. Treating the documents as active made the capture
provider report unjoined bindings as implementation failures.

**Workaround:** Keep the not-yet-routed page and journey entries in `draft`
status. The implemented flows page remains active; promote each page only when
its route and stable selector evidence land together.

**Real fix:** Implement the remaining routes and join their bindings against
live accessibility captures, then promote them from draft to active.

**Owner:** Device-control UI owner.

**Refs:** `experience/index.json`, `ui/src/app/routes.tsx`, Test Genie run
`20260817-043745-9c1d041b`, phase `experience`.

### 2026-08-17 — Test Genie self-route resolution is not reliable in this run

**Symptom:** The workflow phase cannot open device-control BAS cases because
`@scenario/self` is passed to runtime discovery without a scenario name.

**Root cause:** The shared workflow provider resolves the placeholder as a
literal scenario key instead of binding it to the run's target scenario. This
is outside device-control's route or workflow content.

**Workaround:** Use targeted local UI/API validation and inspect BAS assets
directly while the provider is being repaired. Do not replace the placeholder
with a hard-coded port in scenario-owned workflows.

**Real fix:** workflow-health/Test Genie must bind `@scenario/self` before
calling runtime port discovery and preserve the provider run's target scenario.

**Owner:** workflow-health/Test Genie maintainers.

**Refs:** Run `20260817-043745-9c1d041b`, workflow artifact; related prior
finding `durable provider terminal run has no shared validation response`.

### 2026-08-17 — Phase 9 device-control run remains queued

**Symptom:** Fresh run `20260817-090402-bb45848c` remained queued with no
progress since creation for 45 minutes; durable status reported the run active
but not started.

**Root cause:** Shared Test Genie caller-queued admission/worker capacity is
saturated while Test Genie is under parallel development. This does not
indicate a device-control assertion failure.

**Workaround:** Retain the run identity, use targeted device-control checks
and the physical Galaxy A03s evidence, and do not call `findings` until the
server-owned run is terminal.

**Real fix:** Restore Test Genie worker scheduling and reconcile this retained
run's findings manifest.

**Owner:** Test Genie maintainers.

**Refs:** run `20260817-090402-bb45848c`; plan
`mobile-delivery-ramps-implement-scenario-to-ios-complete`.

### 2026-08-17 — Fresh integrated visual/performance findings need capture reconciliation

**Symptom:** Fresh run `20260817-134652-0046615c` reports repeated
`visual_status_bar_color_mismatch` findings across its UI-health captures and
the performance phase reports Lighthouse `0.54`, below the `0.75` threshold.

**Root cause:** The earlier targeted repair and Lighthouse evidence are green,
but the current server-owned capture still observes a different top-strip color
and performance profile. The exact capture colors are not exposed in the
terminal findings summary, so this cannot be safely closed by changing the
theme declaration blindly.

**Workaround:** Keep the shared theme declarations aligned with the actual
design tokens, use the passing targeted UI/build/quality checks, and retain the
integrated run as an open finding until a fresh capture exposes the measured
colors and timing evidence.

**Real fix:** Re-run UI-health/performance with the repaired Test Genie capture
path, compare the measured status strip with the declared theme token, and
repair the owning CSS or capture provider at the cause.

**Owner:** Device-control UI owner for CSS; ui-health/performance-health
maintainers for capture fidelity.

**Refs:** Run `20260817-134652-0046615c`; `ui-health` and `performance`
artifacts; targeted Lighthouse evidence recorded in `PROGRESS.md`.

Follow-up runs `20260817-144148-3bac8ee7`, `20260817-145412-e5861f38`, and
`20260817-150319-4b2d8e82` executed all 22 phases but retained the same
`visual_status_bar_color_mismatch` error. The scenario now aligns the body
safe-area pixels with the top-bar surface and exposes media-specific chrome
colors; the remaining mismatch is not closed because the shared capture path
does not publish the measured pixel evidence needed to distinguish its
desktop/mobile viewport handling from the scenario render.

### 2026-08-15 — Physical Android proof requires a configured unlock profile

**Symptom:** The Galaxy A03s is discoverable and accepts device-control
operations, but the conformance `observe` step reports
`visible surface unavailable` while the phone remains locked.

**Root cause:** Android credential unlock is intentionally authority-backed.
The prior stored profiles were revoked by an explicit cleanup operation, not
by device loss or an Android failure. A new active profile is now provisioned
and verified for the Galaxy A03s; its credential remains authority-held.

**Workaround:** Set `ANDROID_AUTH_PROFILE_ID` to the active profile id when
running the physical Android matrix. The journey invokes device-control's
profile-backed unlock endpoint and verifies the postcondition before install.
Do not treat wake-only or keyguard-visible recordings as product-surface proof.

**Real fix:** Keep the profile active and rotate it through the credential
authority when the device PIN changes. The conformance flow now supports this
profile-backed unlock path without receiving the secret.

**Owner:** Device owner / device-control authority owner.

**Refs:** `api/internal/control/auth_service.go`, `api/strategy/androidadb/androidadb.go`, Android delivery ramp phase 10.

### 2026-08-13 — Galaxy A03s native screenrecord can capture a black display body

**Symptom:** `adb screenrecord` can return a valid H.264/MP4 file while the
decoded display body is uniformly near black. During the reproduced run,
`screencap` showed the visible launcher and the power probe reported awake, but
the recording body measured YAVG=16 and YMAX=16. A separate attempt recorded
zero frames when SurfaceFlinger reported the display power mode off.

**Root cause:** The Samsung Android 13 image's native screenrecord path is not
currently correlated with the actual display composition. `mWakefulness=Awake`
and the power-manager state are insufficient evidence that screenrecord can
capture visible pixels.

**Workaround:** Device Control now samples decoded recording content outside
the status/navigation bands and fails the recording before publishing an
evidence reference when the body is uniformly near black. Use `device state`
and `screencap` as separate diagnostics; do not treat a valid MP4 container as
proof of a visible recording.

**Real fix:** Repair or replace the Android native capture path on this image,
or add an explicit screenshot-backed fallback that is labelled synthesized and
subject to its claim-class threshold. The fallback must remain within the
Device Control evidence path and must not silently relabel a degraded capture
as native.

**Owner:** device-control strategy owner.

**Refs:** `api/internal/evidence/recording.go`,
`api/strategy/androidadb/androidadb.go`,
`DVC-P0-012`, live flow `8f60f7cb-57c2-4739-ae88-eb3074035fb4`.

### 2026-08-16 — Evidence redaction painted a false quarter-screen bar

**Symptom:** Android screenshots and review recordings showed a large black
band across the top of the portrait frame. The raw stored evidence already
contained the band, so the web-console viewer was not the source of the issue.

**Root cause:** `internal/evidence/redact.go` treated notification protection as
a fixed top-quarter paint operation for every image and video. The policy did
not first establish that an expanded notification surface existed.

**Historical fix (superseded 2026-10-31):** The producer formerly masked the status-bar band by default; that arbitrary mask was later removed. Caller-supplied sensitive regions remain supported; the producer does not detect
private screen content. Regression tests check that unspecified areas remain
unchanged and unsupported redaction is not reported as verified.

**Refs:** `api/internal/evidence/redact.go`,
`api/internal/evidence/redact_test.go`, user evidence from 2026-08-16.

### 2026-08-10 — Experience floor errors are inherited from the template shell

**Symptom:** `experience-manager spec validate device-control` reports FAILED.
The only SEVERITY_ERROR findings are two floor claims, on every page that has a
live capture:

- `floor_tap_target_size` — the Theme control measures 29px tall at 390×844,
  under the 44px minimum.
- `floor_safe_area_tap_targets` — the bottom-nav Dashboard item sits flush at
  y=790 h=54 in an 844px viewport, overlapping the mobile unsafe bottom area.

**Root cause:** Not this scenario's code. `ui/src/layout/AppShell.tsx` and
`ui/src/layout/BottomNav.tsx` are byte-identical to the `react-vite` template,
and the nav delegates to the canonical `ui/src/components/ui/bottom-nav.tsx`,
which already applies `pb-safe` and `touch-target`. The safe-area finding looks
like `env(safe-area-inset-bottom)` resolving to 0 in the headless capture
browser rather than a CSS defect; the 29px Theme control is a genuine
tap-target defect. Both surfaces are shared, so every scenario generated from
this template inherits them.

**Workaround:** None needed for spec work — the warnings-only findings
(`state_missing`, `capture_unavailable`) are unaffected and the specs are
complete regardless. But the experience phase gate stays red until these are
fixed.

**Real fix:** Owned by the template and the component canon, not here. Patching
`ui/src/layout/*` locally would fix one scenario, leave every other generated
scenario broken, and register as template drift.

**Owner:** template-manager / react-component-library. Filed to scenario-qa as
`knw-1786386038390145690`
(`bug-inbox/code-defect/react-vite-template-shell-fails-experience-floor-tap`),
flagged `speculative-cause` because the safe-area half is a hypothesis about
the capture environment rather than a confirmed CSS defect.

**Refs:** `ui/src/components/ui/bottom-nav.tsx`,
`templates/scenarios/react-vite/ui/src/layout/`, `experience/pages/fleet.json`.

---

### 2026-08-17 — Integrated Test Genie validation is queued without a terminal verdict

**Symptom:** Fresh run `20260817-035511-16157988` remains `queued/active`
beyond its estimated duration, and the waiter returns without terminal JSON.

**Root cause:** Shared Test Genie scheduling/worker state is degraded while
another agent changes the test-genie scenario; this is not a device-control
application failure.

**Workaround:** Use the passing targeted UI, API, build, and requirements checks
as current evidence. Do not infer an integrated PASS from the queued run.

**Real fix:** Restore Test Genie worker scheduling and complete the retained
server-owned run, then reconcile its findings.

**Owner:** Test Genie maintainers.

**Refs:** Run `20260817-035511-16157988`; plan mobile-delivery-ramps.

### 2026-08-17 — Agent-mode experience contract is ahead of its UI

**Symptom:** The agent experience page declared an active route and required
accessibility bindings, but device-control has no agent route or implementation.

**Root cause:** The P1 agent-mode requirements remain planned while the
experience index had been promoted to active prematurely.

**Workaround:** Keep the detailed `pages/agent.json` contract as a draft and
leave the operator surface out of the active experience set until the agent
runtime and route exist.

**Real fix:** Implement the bounded agent run, transcript, abort, and promotion
surface, add its BAS coverage, then promote the page back to active.

**Owner:** Device-control maintainers.

**Refs:** `experience/pages/agent.json`; `requirements/02-post-launch/module.json`
(`DVC-P1-005`, `DVC-P1-006`).

### 2026-08-17 — Settings experience contract is ahead of its UI

**Symptom:** The settings experience specification describes redaction,
consumer grants, retention, and bridge-connection controls, but the current UI
only implements theme and locale preferences. Keeping the page active caused
the validator to report unjoined bindings for every declared settings element.

**Root cause:** The detailed settings contract was promoted before the
corresponding controls and state-specific BAS coverage existed.

**Workaround:** Keep the detailed `pages/settings.json` contract in `draft`.
The existing `/settings` route remains a real preferences page and is covered
by the UI route tests.

**Real fix:** Implement the declared policy, grant, retention, and bridge
controls with their state fixtures and BAS coverage, then promote the page back
to active.

**Owner:** Device-control maintainers.

**Refs:** `experience/index.json`; `experience/pages/settings.json`;
`ui/src/pages/SettingsPage.tsx`; requirements `DVC-P1-003`, `DVC-P1-004`, and
`DVC-P1-010`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| API/domain ownership | The current architecture audit reports 22 requirement-to-domain ownership errors and 13 unowned-source-path errors. The in-flight implementation consolidates several product surfaces behind `api/handlers/control` and `api/internal/control`, while this document still describes split `devices`, `sessions`, `flows`, and `strategies` source paths. | Targeted API/UI/CLI validation is green, but the architecture phase cannot be treated as clean until the ownership map and the intentional façade boundary agree. | Complete the domain consolidation deliberately: align `DOMAINS.md` and the requirement ownership register with the actual source tree, declare approved cross-domain façade edges or split the handler, then rerun Architecture Cartographer. |

### 2026-10-28 — Android ADB cancellation error hides context expiry

**Symptom:** A timed-out ADB app launch is reported as `signal: killed`, without showing whether the effective Device Control step/run context ended.

**Root cause:** `commandRunner.Run` used `exec.CommandContext` but returned only `CombinedOutput`'s process error, dropping `ctx.Err()`. The TV launch cause remains unknown; retained flow and step budgets overlap as competing explanations.

**Workaround:** Preserve the configured flow and step limits in each receipt; treat a bare `signal: killed` as ambiguous and do not infer that Android killed the app.

**Real fix:** Implemented: when a command fails after its context ends, return an error that wraps the context error and retains the subprocess error. Add focused runner cancellation and step-versus-run deadline regressions.

**Owner:** Device Control maintainers.

**Refs:** `api/strategy/androidadb/androidadb.go`; `api/strategy/androidadb/androidadb_test.go`; `api/internal/control/service_test.go`; TV comparison `tv-portal-launch-success-failure-comparison-2238.json`; Test Genie run `20261028-224224-c92cebd4`.

## Work ladder

- Rung: W3 attempted; W0-W2 passed.
- Evidence: W0 found the deterministic `swarm-manager` goal `generalized-device-control-portable-searxng-retirement` targeting `scenario/device-control`; the PRD now agrees that identity and declaration are mandatory while modalities are optional, so the goal/PRD contract comparison passes. W1 `business-health validate scenario device-control` and W2 `vrooli scenario requirements validate device-control` both passed. Targeted API and CLI Go suites pass; the UI suite passes with 43 files and 183 tests, type-check, lint, and coverage (97.24% lines, 85.21% branches, 90.21% functions). Identity claims, durable merge/split/audit restoration, multi-transport state composition, Cast endpoint binding, degraded multi-service discovery, canonical pairing IDs, frameless agent dry-run behavior, polling observation fallback, and CLI/UI claims rendering are covered by focused tests. Follow-up live witnesses now also pass: `packages/mdns-go` resolves the real Android TV Remote and Cast services over IPv4/IPv6, and the Cast protobuf transport reads state and exercises same-value volume/application launch against the real SmartTV receiver.
- Blocker: the server-owned W3 run `20260818-010813-555225d8` remained `in_progress` in `preparing:target_runtime` after the required single wait and one permitted durable-state read; it has no terminal verdict yet. The earlier run `20260818-005935-e7a75376` terminated with 16/22 phases passed and provider/pre-existing findings; its CLI failure was corrected by rebuilding the installed scenario CLI, and its UI coverage failure was corrected by the tests above. A normal scenario start was also blocked by an unrelated `vrooli-bridge` UI port collision. The package live mDNS check now passes on the operator LAN for both `_androidtvremote2._tcp` and `_googlecast._tcp`, resolving only the requested PTR instances and their SRV/TXT/A/AAAA data. The Cast live witness now passes for receiver state, absolute-volume command, and same-id application launch. Owner-present PIN pairing, lease-owned end-to-end CLI actuation, and externally caused event proof remain unproven. The required plan-specific baseline `reachable-non-mobile-devices-real-lan-discovery-google-tv-baseline` and historical `device-control-lan-reach-baseline` do not exist, so no clean baseline diff can be claimed.
- Workaround: preserve the server-owned run for provider recovery, use the targeted suites plus the package live mDNS witness for code evidence, and run live pairing/Cast checks on a host with owner-present hardware. At 2026-08-18 02:13 UTC, the running scenario's `device-control device discover --json --timeout-seconds 8` returned 503 because that server process was started at 01:08 before the latest source changes and its mDNS browse timed out; `device list --json` did preserve explicit unreachable rows for Android TV Remote and Google Cast. Independent `avahi-browse` and the package live test both witness the real TV/Cast services, but they do not substitute for the package/owner walkthrough. Plan Manager recorded a degraded partial handoff because execution began after the implementation changes and no trustworthy before-state could be reconstructed. A full `go test -race ./...` is not a clean signal because the pre-existing `strategy/androidadb` screenrecord fixture races while mutating its scripted runner; targeted race runs for control, strategy, Android TV Remote, and Google Cast remain the relevant clean evidence.
- Measured: 2026-08-18
- Current investigation: the Google Cast status parser previously converted an omitted `volume.level` or `muted` field into the zero values `0` and `false`, and the generic device-state JSON also emitted unknown mobile-only fields as `auto_rotate: false` and `charging: false`. The parser now tracks field presence, reports omitted values as unavailable, and preserves explicitly known false values. Regression tests cover both cases. The paired Android TV Remote and Google Cast records remain separate identities because they have different endpoints and the Remote transport does not expose readable state.
- Current evidence: after a managed restart on 2026-09-06, the live SmartTV Cast endpoint explicitly returned `volume: 0`, `muted: false`, `application: 2C6A6E3D`, and unavailable `input`/`power`; this is an explicit device response, not the former missing-field fallback. `go test ./...` passes for the device-control API. A fresh comprehensive Test Genie run could not be admitted because preview capacity was saturated; the prior server-owned run completed degraded with provider-unavailable and pre-existing maturity findings.
- Current blocker: a physical TV or owner-asserted transport merge is required to reconcile the Cast-reported zero with the operator-observed nonzero volume. The Android TV Remote endpoint cannot provide a readable volume state, so device-control must not infer or merge that state by address alone.
- Measured: 2026-09-06
- Current investigation: W3 genericity follow-up removed the program's room/device example, made volume transport selection depend on declared semantic operations and writable properties, and retained endpoint aliases with stable IPv4 preference for cross-transport discovery. Live discovery now finds the paired Remote and Cast services for the same TV host and exposes an unconfirmed relation candidate; no automatic identity merge is performed.
- Current evidence: API/strategy/device-store tests, CLI tests, 14 Program Runtime tests, endpoint generation, and Search Hub discovery all pass. The live Cast state read reports explicit receiver `volume=0` and `muted=false`; physical speaker output is still not verified. The server-owned Test Genie run `20260906-225040-8b75f4a0` was still `in_progress` after its single wait, so no terminal owner verdict is claimed.
- Measured: 2026-09-06
- Current investigation: the live Android TV Remote and Google Cast transports are both observed at `192.168.1.158`, and the relation service emits a same-host candidate at confidence `0.72`. It remains `unconfirmed` because a shared address is evidence of co-location, not proof that two transport identities belong to one logical device; auto-merging would be unsafe on networks with multiple services or stale discovery records.
- Current fix: transport state now declares its measurement domain. Google Cast exposes `volume=0` and `muted=false` as explicit `receiver_volume`/`receiver_mute` values, while marking physical-output volume and mute unavailable. Physical-output volume plans fail closed when only receiver-domain state is available, and the UI labels the controls as receiver volume/mute instead of TV volume/mute.
- Current evidence: after managed restart, the live Cast state still reports explicit receiver `volume=0` with `state_domain=receiver_volume`, plus `physical_output_volume` unavailable. A diagnostic physical-output absolute-volume request returned `409 unsupported_operation` before actuation. The focused API, UI, type-check, and 14 Program Runtime validations pass.
- Measured: 2026-09-06
- Research correction: Google Cast's documented `volume.controlType=MASTER` means the master volume of the TV or attached audio device; `ATTENUATION` means Cast-device/input attenuation. The captured live Cast status contains `controlType: "master"`, so the earlier blanket receiver-only classification was too conservative for this endpoint.
- Current fix: the Cast parser now retains `controlType` and maps `MASTER` volume/mute state to the physical-output domain. It retains receiver-only labeling for attenuation or unknown responses. The live API now reports the Cast value as `physical_output`, although it still reports the numeric level as `0`; that discrepancy with the operator-observed level remains a live-device validation issue, not a missing-field default.
- Research evidence: Google's Cast sender reference defines `MASTER` as TV or audio-device volume control, and Google's Cast media documentation distinguishes receiver/system volume from media-stream volume. The Android TV Remote protocol also defines `RemoteSetVolumeLevel` with volume level/max/mute fields, providing a second unsolicited readback path now parsed by device-control.
- Measured: 2026-09-06
- Current fix: Android TV Remote v2 now parses `RemoteSetVolumeLevel` frames, normalizes `volume_level / volume_max`, and publishes physical-output volume and mute state with provenance. When multiple state-bearing transports are composed, disagreements are recorded and the higher-confidence observation is retained for the aggregate state.
- Current evidence: after a managed restart on 2026-09-06, the paired Remote live read reported physical-output volume `0.08` and mute `false`; the Cast `MASTER` read reported physical-output volume `0` and mute `false`. This confirms the previously observed disagreement and provides a nonzero readback path. The two identities remain separate because their relation is still an unconfirmed same-host candidate.
- Current investigation: local inventory previously stamped the TV with the local hostname as `host_node_id`. The lease fence interpreted that as a Bridge-attached peripheral and rejected actuation because Bridge only retained the phone attachment. Local enumeration now leaves `host_node_id` empty; only Bridge attachment reconciliation sets it. The Android TV Remote client now negotiates the maintained basic feature set, refreshes its connection after volume actuation for a fresh state snapshot, and preserves command delivery separately from physical verification.
- Current evidence: before this correction the live operation failed during lease validation with `volume session was refreshed once`; after the ownership fix the operation acquired a lease and sent five volume-down keys, but the TV remained at `0.08`. A controlled retry after adding the response-witness requirement produced no `RemoteSetVolumeLevel` witness and returned typed `unavailable`; the state remained `0.08`. This proves the lease bug was real and separate from the remaining transport/output problem.
- Protocol correction: Android TV Remote normally publishes `RemoteSetVolumeLevel` on connection rather than after every key. Relative volume delivery therefore refreshes the session before verification instead of requiring a post-key event. Feature negotiation now uses the maintained basic mask (`611`) intersected with the TV-advertised mask, and volume parsing selects native or ARC/eARC fields according to the external-output flag, with a compatibility fallback for older frames.
- Current live evidence: after the protocol and mDNS fixes were deployed, one controlled five-key retry completed with `status=ok` and `verification_class=conflicted`; both the pre-action and fresh post-action Remote snapshots remained `0.08`. The paired endpoint is reachable through `192.168.1.158:6466`, while Cast independently reports `0`, so no physical volume change is proven.
- Discovery correction: a restart briefly failed to read the paired TV because mDNS attempted an administratively-up but non-running Docker bridge. `packages/mdns-go` now requires operational multicast interfaces; the live state read succeeds again through the physical LAN interface.
- Superseding live resolution: the Remote volume witness identified the active route as `arc-earc-amplifier` at `0.08`, while Android TV Remote `VOLUME_DOWN` keys left that physical level unchanged. A bounded Cast `MASTER` setpoint changed the Remote ARC/eARC readback from `0.08` to `0.04`, proving Cast is the working physical-output actuator for this TV. The two transport identities were then merged under the owner-asserted Cast identity, with the behavioral cross-transport change retained as evidence.
- Current fix: volume planning now selects a writable physical-output setter for observed ARC/eARC routes instead of ineffective directional keys; absolute physical-output readback is correctly classified as `verified`; durable transport profiles persist capability/operation/property metadata; stale restored profiles trigger one targeted inventory refresh. The agent-facing `device-control.volume` program now accepts explicit `confirm=true` and emits a compact bounded receipt so learning metadata cannot truncate the envelope.
- Final live evidence: after managed restarts, `device-control` remained healthy, a direct relative request selected Cast and verified `0.02 -> 0.01`, and the final same-value command restored/confirmed the requested target `0.04` (50% below the original `0.08`) through the Android Remote ARC/eARC readback. The confirmed program-runtime call `prog_3908fcec-09ca-46eb-8775-7dc975d17a43` returned envelope `status=ok`, `verification_class=verified`, `signals.outcome=verified_success`, and operation `76e7022f-40ba-4394-9f79-c849e0364215`.
- Measured: 2026-09-06

## Browser E2E validation notes

- 2026-08-12: BAS direct execution successfully completed the dashboard identity journey before the later transport drop and the onboarding/re-probe journey (`1475435b-13f5-4122-afed-9ae9c7b3e735`). The managed browser DOM showed the real Galaxy A03s stable ID, serial, model, Android version, USB transport, and capability inventory.
- 2026-08-12: The flow validation/run journey completed all six nodes after adding the missing flows-page root selector (`6934c56c-ecd1-4685-aa00-dfc39b36bfdd`). The journey now asserts that no operator-visible error is present after dispatch.
- BAS emits a 404 for its optional `__vrooli_recording_event__` callback during mutating browser steps. The device-control API calls themselves returned the expected 200 responses when the phone was attached; the callback warning is owned by the BAS capture environment.
- Standalone BAS CLI runs require explicit substitution of `@scenario/self` and `@selector/...` tokens. The authored cases retain those portable tokens for registry/workflow-health execution; direct CLI validation uses the selector manifest as the resolver.

## Android physical conformance status

- 2026-08-12: The scenario exposed a provider-neutral, twelve-chapter Android physical conformance plan and emitted a `common.v1.TargetVerdict` with `DEVICE_KIND_PHYSICAL`, evidence references, and a serial-bearing detail string. The API and CLI fail closed with `unavailable` when the fixture APK is absent.
- Superseding evidence on 2026-08-15: final governed matrix run `run-b556ab488cfe12f571be7379` built package `com.vrooli.hellomobile` at target API 36, used the active profile-backed unlock path, installed and drove the physical Galaxy A03s, and retained a 181.88s decoded review recording, logcat, and start/end clock evidence. The `offline_transition` chapter is typed unavailable because the target lacks `network-control`; the required-cell gate therefore remains closed.
- Current matrix-wide status on 2026-08-17: targeted run `run-b91a7dd7f47c66e93aaad4c7` closed the required-cell gate with the governed `vrooli-api36` emulator as the satisfying target. The Galaxy A03s remains correctly unavailable for the offline chapter; its cell in that run also failed closed because the currently reachable handset was locked at the visible-surface step. This does not invalidate the retained unlocked physical evidence above.
- The physical-target conformance gap remains open until network control is implemented or the chapter is explicitly excluded from the required physical profile. The matrix-wide gate is satisfied by the emulator, and all other executable chapters completed on hardware.
- Targeted validation is green: API/CLI Go suites, UI 36-file/145-test suite, type-check, lint, managed build/restart, JSON/endpoint generation, the hello-mobile APK build/package inspection, and requirement structure validation. Comprehensive Test Genie evidence is currently degraded: run `20260812-183608-51052a99` found the initial undeclared CLI commands, run `20260812-184226-7d007c1d` found and exposed manifest schema errors, and run `20260812-185005-fbd1e28e` aborted because its legacy runner predates canonical terminal snapshots. These are recorded as infrastructure/contract findings, not treated as passing baseline evidence.

### 2026-08-12 — Cross-platform honesty and wireless seams implemented

The cross-platform strategy declarations now distinguish unsupported host OSes
from unavailable prerequisites, and expose supported host platforms in the
strategy API and CLI. USB onboarding uses native inspection commands on Linux,
macOS, and Windows; bounded sampling reports link flaps and cable guidance.
Android ADB promotion preserves the stable device identity while switching to a
verified wireless endpoint. Lease listings and terminal session responses omit
bearer tokens, and the physical smoke fixture requires a changed frame.

The dashboard now exposes onboarding from the zero-device state. The Windows
host-desktop path remains explicitly unsupported until a real capture/input
implementation is verified; PowerShell presence is not treated as capability
proof.

**`network-control` is unavailable on the Galaxy A03s.** `adb shell svc wifi`
fails on this device image, which blocks the `offline_transition` conformance
chapter from running on the only physical device currently available.

**Owner:** unassigned.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues


## Agent reuse validation boundary — 2026-09-04

W0: the operator explicitly requested usage/improvement implementation; the PRD
now records the agent-reuse target. W1: the corresponding AGENT-REUSE requirement
links the owner and program regression tests. W2: targeted and live evidence is
retained in `.vrooli/program-runtime/tests/validation-evidence.json`; broad Test
Genie evidence still has provider failures. W3: owner APIs, persistence/programs,
and registered skills implement the target; no estimated speed floor is asserted.

Provider follow-up: Scenario QA `knw-1788561851141947801` records the missing UI
surface/command-execution discrepancies and the diagnostic localhost:2026 failure.
Device requirement-evidence follow-up: `knw-1788561878575592107` records the 21
older complete claims without requirements-sync snapshots. The new AGENT-REUSE
requirement remains in_progress until its provider evidence can be earned.
Do not lower acceptance gates or erase existing evidence to make these checks pass.

## Work ladder

- Rung: W3
- Evidence: `vrooli scenario requirements validate device-control --json` reports the contract and requirement registry at L3; targeted implementation and live workflow evidence cover only the currently available Linux path, while the plan still has unverified platform rows and certification cases.
- Blocker: Device Control implementation and acceptance evidence remain incomplete for the plan's cross-platform and final-regression outcomes. The current Plan Manager phase pointer is still Phase 4 after a killed control-plane validation.
- Measured: 2026-09-08

### 2026-09-17 — Android TV ADB is reachable outside the governed application path

**Symptom:** An owner-authorized SmartTV at `192.168.1.158:5555` was already
connected through ADB, and the owner successfully installed SmartTube with
`adb install -r`. Device-control inventory recognized an Android target, but
its live declaration did not expose `app-lifecycle`; `device state` returned
`device_state_unavailable`; and `device app --operation package-state` returned
`capability_unavailable`. Static `strategy verify android-adb` also reported a
degraded result because screenshot decoding returned `image: unknown format`.

**Root cause:** The Android adapter has typed install and lifecycle code, but
network-TV onboarding, live capability probing, and the public app surface are
not yet one coherent contract. A probe failure can currently obscure the
application capabilities that a reachable target may still support. The
manual ADB path bypasses stable target adoption, artifact provenance, lease,
audit, and post-install verification.

**Workaround:** The owner may use a separately authorized raw ADB command for
diagnosis or emergency setup, but it is not product evidence and must not be
used by flows or agents. Use Cast/Android TV Remote for the screenless
operations they actually prove.

**Real fix:** Implement governed Android TV network-ADB onboarding for classic
TCP and Wireless Debugging/TLS profiles; verify serial/model/endpoint
ownership; probe capabilities independently; expose app inventory, artifact
install, package state, launch, stop, uninstall, and clear-data through the
CLI/API under a lease; add package-scoped agent context; repair screenshot
format handling; and earn live phone and TV conformance evidence.

**Owner:** Device-control maintainers.

**Refs:** `api/strategy/androidadb/androidadb.go`,
`api/internal/control/app_lifecycle.go`,
`docs/guides/connecting-a-device.md`,
`docs/reference/capabilities.md`, and the 2026-09-17 SmartTV validation.

**Resolution (2026-09-17):** All four defects are fixed and verified live against
the SmartTV 4K at `192.168.1.158:5555` (serial `AE70A4D38B`).

1. *Screenshot decode.* This TV firmware prepends a 65-byte log preamble (`Init
   wrapper sys mutex successful...\nopen mma dev failed\n`) before the PNG
   signature, so `image.Decode` reported `image: unknown format`.
   `sanitizeCapturedImage` now strips any bytes before the first decodable image
   signature at every capture/evidence site (observe, recording samples, the
   screenshot action, and the capability probe). `strategy verify android-adb`
   now reports `status: available`, `failed: []`.
2. *Screenshot capability honesty.* The `Describe` screenshot probe now requires
   an actually-decodable frame (`probeScreenshot`), so it cannot declare
   screenshot support the observer path would then reject, and a screenshot
   failure no longer masks package/lifecycle/input/state capabilities (each is
   probed independently).
3. *Governed network onboarding.* `OnboardNetworkADB` (CLI `device
   onboard-network --endpoint`, `POST /api/v1/devices/onboard-network`) adopts a
   directly-addressable classic-TCP or Wireless-Debugging/TLS endpoint as a
   first-class governed device. `ConnectNetwork` validates the endpoint, runs
   `adb connect`, confirms authorization, and derives the durable identity from
   the hardware serial (never the address); the endpoint-bound adapter is
   persisted as the device's transport strategy. `strategyForFlow` now routes an
   unselected transport for a wireless-only device to its verified endpoint-bound
   strategy instead of the ambient base adapter (see DECISIONS.md 2026-09-17).
4. *package-state verification.* `AppLifecycle` package-state now runs
   `pm list packages` (exact-line match, no prefix collision) and `dumpsys
   package` and returns a verified `installed` boolean plus `version`, replacing
   the canned "observed through the next device-state probe" reply.

Live evidence: onboarded device `android-0df87078c18abfb8` reports every ADB
capability available; `device state` returns full state
(`foreground_package: org.smarttube.stable`); package-state reports SmartTube
`installed: true, version: 32.47` and a bogus package `installed: false`;
`app-launch` and `app-stop` complete under lease with persisted audit receipts
(`app-package-state`, `app-launch`, `app-stop`, all `outcome: success`). The raw
manual `adb install -r` path remains owner-diagnostic only and is not product
evidence.

**Remaining (not blocking this entry's closure):** live install / uninstall /
clear-data against the TV were not exercised (destructive on the owner device
without a vetted artifact). The app-lifecycle audit now falls back to the durable
device-record transport when the adapter declaration carries none, so an
endpoint-bound TV records `transport: wireless` instead of an empty value.

## Work ladder

- Rung: W3 (localized implementation defect with established expected behavior),
  with W2 requirements-sync re-earned afterward.
- Evidence: reproduced all four defects live, then fixed at cause.
  `go test ./...` passes for the device-control api and cli; focused table-driven
  tests cover `sanitizeCapturedImage`, `probeScreenshot`, `ConnectNetwork`,
  `validateADBEndpoint`, package-state verification, and the corrected
  wireless-default transport routing. `strategy verify android-adb` →
  `available`. Live SmartTV onboarding/state/package/launch/stop/audit captured
  above. `vrooli scenario requirements validate device-control` and
  `business-health validate scenario device-control` both report 0 errors after
  `requirements sync` (0 statuses changed; snapshot recorded).
- Blocker: none for the four defects. DVC-P0-011 remains `in_progress` because
  its full-conformance-against-TV-class clause is not yet earned (only
  onboarding/state/app-lifecycle proven live, not the full conformance suite).
- Comprehensive Test Genie run `20260917-202453-b64493c7`: 19/28 phases passed,
  including every phase this change touches — structure, contracts, code-facts,
  go-code-graph, api, architecture, quality, business, security, proto,
  agent-conformance, storage, measures, experience, templates, branding,
  skill-set, performance, typescript-code-graph. The 9 non-passing phases are
  pre-existing scenario debt or provider/infrastructure issues unrelated to this
  change and not caused by it: `portability` (dependency-analyzer trusted-base
  RPC precondition), `ui-health` (53 template/UI findings; UI untouched here),
  `dependencies` (declared-without-import debt=5; no deps added here), `docs`
  (broken_command_snippet debt=27; the new onboard-network snippets are not among
  them), `unit` (coverage thresholds: UI functions 84.45% with all 120 tests
  passing, and `cmd/desktop-helper` 0%), `workflow` (the `@scenario/self`
  provider defect already logged above), `channel-conformance` (switchboard
  provider start killed), `tidiness` (duplication debt=50; the new ConnectNetwork
  /OnboardNetworkADB/sanitizeCapturedImage symbols are not flagged), and
  `programs` (pre-existing author-flow/prepare-task/replay-flow fixture envelope
  errors). Scoped re-run confirmed `api`, `quality`, and `contracts` pass after
  declaring `device app` in `cli/manifest.json`.
- Measured: 2026-09-17

### Scoped W3 inspection repair — 2026-10-22

- Rung: W3, localized runtime-inspection implementation/qualification under the authorized TV research; no W0–W2 gate or full readiness claim.
- Evidence: fixed read-only voice configuration facts were absent from InspectDevice. Additive proto/owner inspection now returns configuration and explicit unknowns; focused VoiceInspection/Inspection tests and live CLI/Connect delivery pass. Current hardware microphone feature is false; role-holder probe is unknown, not proof of absence.
- Remaining: reliable native/WebView readiness and input replacement are unqualified by two failed local trials. Physical audio access and Assistant role selection need separate bounded prototypes. WebView attachment cleanup source finding is tracked by `knw-1792696363648007859`; CDP attachment was not exercised.
- Measured: 2026-10-22. Unit run `20261022-192128-ae97fc18` retains unrelated desktop fixture and coverage failures; original issues remain unchanged.

### Scoped W3 Android ADB cancellation diagnostic — 2026-10-28

- Rung: W3, localized implementation diagnostic defect; no W0–W2 or full readiness claim.
- Evidence: Android `commandRunner` now preserves `context.Canceled`/`context.DeadlineExceeded` through its returned error. `TestCommandRunnerRunReportsContextCancellation` and the full `strategy/androidadb` package pass. `TestFlowReportsWhetherStepOrRunDeadlineStoppedLifecycle` passes both shorter-step and shorter-run cases and confirms the owner flow labels them distinctly.
- Validation limit: Test Genie run `20261028-224224-c92cebd4` reports `programs` passed and `unit` failed on unrelated desktop/AT-SPI/X11 Unix-socket bind/EOF failures under generated long temp paths, plus UI coverage command failure. The focused Android package and flow regressions pass directly. Do not repeat the unchanged broad suite until the Unit Health temp-path blocker changes.
- Remaining: actual cause of the two physical TV launch failures is unknown. Do not repeat a launch solely to wake DreamX; a later commissioned attempt can now retain context-aware failure evidence.
- Measured: 2026-10-28.

### Scoped W3 flow retention and lifecycle boundary — 2026-10-28

- Rung: W3; scoped owner retention and diagnostic repair, not full scenario readiness.
- A running process previously held ordinary run results and replay flows only in memory. Explicit, sensitive-data-acknowledged retention now stores a selected run in the flows domain with an operator-selected deadline; expired rows are deleted on access or list, without background deletion, and legacy rows retain their previous no-deadline status; inventory is point-in-time and does not fence admission. Clipboard-step flows are refused; this is not a general secret detector. Owner deletion removes the retained record, not live maps, evidence files, backups, or SQLite WAL remnants.
- Validation: focused restart/export, exact-device promotion, clipboard refusal, storage-failure and HTTP acknowledgment/refusal tests pass. Test Genie `20261028-235204-fec0848b` storage passed with 19 advisory findings; unit failed on three test commands. Source changed after admission, so the run is not a final-tree proof. `make endpoints` regenerated the endpoint catalog successfully.
- Remaining: Device Control does not implement the `common.v1.LifecycleMaintenanceService` provider contract. The control plane treats an absent provider as unguarded. A complete replacement fence must cover existing held-lease flow/actuation paths, observation leases, recordings, agent work and the separate desktop session manager; preserve volatile run/evidence state; retain fence identity through replacement; and resume through the existing control-plane owner. A flow-only admission switch cannot establish complete inventory or protect held leases. Research prototypes for process and helper fencing were removed after review; the owner boundary and test findings remain in the research receipt. The older serving process also cannot use the new inventory or retention APIs, so its unknown in-memory state remains unpreserved; do not infer safe replacement from this change.
- Evidence: research receipt `device-control-retention-specialist-2354.json` and scoped Test Genie findings in the current TV research run; no serving replacement or device effect was performed.

### Desktop helper cleanup receipt availability — 2026-10-29

- Rung: W3, localized cleanup-recovery defect. A failed native release was durably marked pending, but the next maintenance tick terminated the helper and made the exact cleanup receipt unavailable. The helper now keeps that read-only path available while retrying native release at a bounded two-second cadence. Storage and authority errors still terminate the helper; pending cleanup still blocks input admission.
- A focused owner/helper integration regression injects release failure, waits through a maintenance tick, observes revoked input and an exact `released=false` receipt, then confirms `released=true` after a successful destination retry. The full `internal/sessions` package passed with short test socket paths. Initial long-socket and pre-fix failures remain in the TV research evidence.
- This does not implement the lifecycle maintenance fence or qualify the serving process. Cross-actor desktop admission/forwarding inventory, in-flight native and WebRTC work, durable fence state and process-wide admission remain necessary before a complete drain can be claimed.
- Research exercised helper fencing, cross-actor admission pagination, and a peer-bound operation proof. Those unmounted prototypes were removed from the deliverable: they did not cover the API, recordings, volatile state, all admission surfaces, or control-plane integration. Their results remain in the research evidence; no complete lifecycle provider was mounted.

### Desktop WebRTC release uncertainty — 2026-10-29

- Rung: W3. A failed provider-wide input release revoked a session but made exact cleanup retry fail as already revoked; expiry and capture-failure revocation could also discard release errors. The manager now keeps cleanup pending, refuses a successor controller, and permits exact-session cleanup after in-flight input, clipboard and capture work settles. Observer close does not release another controller's held input. An incomplete capture that ignores cancellation remains a blocker until it actually returns.
- One manager currently owns one local native provider whose `ReleaseHeld` has no device argument. It therefore permits one controller across that provider, including different target IDs; separate manager/provider instances remain independent. This is a product-visible restriction until a provider exposes genuinely independent release scopes. Same-target takeover remains supported and failed takeover revokes the old controller pending cleanup.
- Focused pre-fix failure and desktop manager/handler race-test evidence are in the current TV research run. The point-in-time manager inventory is not a lifecycle admission fence, has no complete public recovery inventory, and does not cover historical failures from the currently serving binary. No live WebRTC trial or process replacement was made.


### External recording stop uncertainty — 2026-10-29

- A wrong-device stop previously erased another device's owner handle before checking it. A failed stop also erased the handle even though Android's recorder consumes its own handle before it can report encoder, artifact, or restoration failure. The owner now preserves an uncertain process-local marker and never replays a stop automatically. This marker is not durable and is not proof of destination cleanup.
- When the recorder returns video and an error together, the owner applies its existing evidence processing and returns the recovered reference alongside HTTP 409 `recording_stop_uncertain`; the marker and reference remain retrievable during this process lifetime with a valid device lease. Artifact processing failure leaves uncertainty without a reference. On Android the host wrapper being signaled or exiting does not establish that the remote encoder stopped; restoration commands are attempted but failure remains unresolved. No automatic device reconciliation or restart safety is claimed.
- Focused fake-based owner, HTTP, and Android recorder tests cover wrong-device refusal, failed stop without replay, recovered artifact with restoration failure, artifact-read failure and independent restoration attempts. No live recording was started.

### Screen readiness and verified control — 2026-10-31

- Rung: W3, scoped implementation. A `screen-on` flow condition accepted unknown screen state; the regression now requires positive on/off evidence. Screenshot-bearing flows with a state reader reject an unknown visible surface.
- Android ADB exposes explicit screen wake and sleep through a shared capability. The semantic screen operation acquires a lease and verifies the result; ready additionally checks the keyguard and may use the existing authentication profile. Screen state is device-reported, not independent physical panel evidence.
- The owner confirmed physical wake and sleep on the TV, and the semantic wake, sleep and ready operations subsequently changed the TV's reported state as requested. A screenshot and package-scoped log capture also passed on the TV. The saved Android phone endpoint timed out on reconnect.
- A requested multi-second TV recording produced one encoded frame; FPS alone misclassified it as useful. Recording evidence now requires multiple frames and, for bounded captures, an adequate duration. A failed or cancelled flow stops its active recorder once and retains an uncertainty marker on a failed stop. Screenshot fallback reports its sample rate rather than duplicated encoder FPS. An 8-second native TV trial produced 15 frames over 7.94 seconds (1.89 fps): the retained clip is degraded and the operation reports `no_evidence` for transition quality.
- Test Genie unit run `20261101-022717-f2e9da7b` failed. Follow-up isolated long Unix socket paths and UI coverage below thresholds; the full API and CLI Go suites pass with a short temporary directory. Workflow-program test fixture contract metadata and relative helper lookup were repaired. The operator authorized managed service restarts; the existing incomplete lifecycle-fence limitation remains as noted above.

### Android TV recording also disrupts playback — 2026-11-01

On the SmartTV 4K, YouTube's active SurfaceView presented about 60 fps for five
seconds without capture. During a simultaneous nine-second 1080p, 8 Mbps native
`screenrecord`, it presented about 1.3 fps and the encoder returned 12 frames.
A separate 640×360, 1 Mbps native recording returned 11 frames in eight seconds;
a short concurrent presentation sample was about 2.4 fps. These are individual
trials, not a cross-device performance guarantee. The video impact is separate
from the resulting file's frame rate. Lowering resolution and bitrate alone did
not fix this device.

A short repeated-PNG experiment averaged about 47 fps on the presentation
surface against a 60 fps baseline, but completed only one screenshot during the
seven-second trial and a later screenshot timed out. This is not a viable
high-frame-rate replacement. Protected media, device firmware, resource sharing
and sampling overhead remain possible contributors; these trials do not isolate
an underlying firmware cause. Device Control's `performance-state` reports
recent presentation timings and cumulative graphics counters, with their limits,
for repeatable paired investigations. The native recorder retains its prior
1 Mbps setting; there is no claim that this setting resolves the issue.

### Android recording ownership and display-state restoration — 2026-11-01

The native recorder previously sent a device-wide `pkill` for `screenrecord`,
which could interrupt another recording. It now looks up the encoder by the
unique output path and signals only a single matching PID; missing or ambiguous
process identity reports an uncertain stop. This is still not durable
reconciliation after process loss, and process listing and signaling are not an
atomic OS operation. Android recording now uses shell-owned temporary storage.

The old restoration inferred the saved stay-awake preference from the current
`mStayOn` state and conflated light and deep idle settings. It now snapshots
and restores the actual preference and each idle setting independently, including
unset preferences. A live TV recording restored stay-awake `0`, timeout `600000`,
and both idle modes enabled. Its visual fallback produced degraded evidence at
about one source frame per second; the successful restoration is not evidence
that recording performance is fixed.

Screenshot fallback previously encoded every sample as if it arrived at a fixed
2 fps, making a six-second capture play back in three seconds when screenshots
were slow. It now uses sample timestamps and the measured capture duration for
presentation while separately reporting the actual source rate. A live TV
six-second capture produced a 6.07-second file with about 0.5 source fps and
remained degraded; the duplicated output frames do not add motion evidence.

A further paired YouTube trial compared uncompressed ADB screenshots with PNG.
Normal playback presented 60 fps; during an eight-second raw capture loop it
presented about 25 fps with five completed captures. After a second 60 fps
baseline, a PNG loop presented about 45 fps with one completed capture. On the
launcher, raw screenshots took approximately 0.8 seconds each versus 1.5 for
PNG, but this speed comes with substantial workload and network differences.
These short, sequential trials do not prove a general ranking across devices.
Raw capture is not enabled by default on the strength of this TV experiment.

A follow-up device inventory found no video encoder marked hardware-accelerated
in this TV's Android `dumpsys media.player` listing. AVC, HEVC, VP8 and VP9
include Android software encoders. During an independent native screenrecord,
the device's codec logs identified the selected encoder as
`c2.android.avc.encoder`, which the inventory marks software-only. This is
consistent with CPU contention during capture, but does not isolate the full
cause of playback stutter. A different codec, encoder implementation, device,
or external capture path has not yet been qualified.

A scoped MediaCodec experiment using Android's existing build tools compared
software encoders at 640×360 under active YouTube playback. HEVC produced 11
frames in nine seconds and a concurrent presentation sample was about 2 fps.
In separate paired trials with 60 fps baselines, VP8 produced 10 frames in nine
seconds with approximately 12 fps presentation; AVC produced 15 frames with
approximately 12 fps presentation. A 320×180 AVC trial still produced 15 frames
and approximately 9.5 fps presentation. These are small sequential samples,
not an encoder benchmark; none provides acceptable capture or playback. The
experiment's temporary device code and media files were removed afterward.

The retained Galaxy A03s endpoint `192.168.1.179:34483` timed out on reconnect.
Its last known address answered ICMP, but identity and current ADB authorization
were not established. ADB's mDNS inventory advertised the TV only. Reconnect now
recognizes ADB's text-reported failures even with a successful process exit,
and advances to discovered endpoints without waiting on that failed transport;
the serial identity check remains required.

### Phone and playback-audio qualification — 2026-11-01

The owner-connected Moto G7 (Android 10, serial identity distinct from the
previous Galaxy) passed USB sleep, wake, credential-backed ready/Home, and
full-frame screenshot checks. Its credential was provisioned through the
existing host-bound credential authority. A moving Home-focus flow produced
88 native frames in 6.95 seconds, about 12.7 fps. A package-scoped launcher log
snapshot returned no matching records, which is not proof of no events.

Android playback capture was exercised with a temporary permissioned app on
both the Android 11 TV and Android 10 phone. Each required the system projection
prompt and a foreground capture service. Both recorded a non-silent 440 Hz test
tone; the TV also captured non-silent audio while YouTube played. A separate
phone prototype captured a video and PCM audio during one projection session;
its host-muxed result has approximate start alignment, not validated audio/video
synchronization. The temporary apps were uninstalled. These experiments do not
add audio capture to the Device Control recording path. Target-app capture
policy can also prevent particular playback from being captured.

Evidence now reports whether an audio stream exists, with unknown distinct from
absent. A stream alone proves neither audibility nor synchronization. Android
ADB screenrecord and its screenshot fallback are presently video-only.

Wireless promotion of the Moto exposed adbd's asynchronous restart: an immediate
connection could be refused but its text error was treated as a hardware serial.
Promotion now retries the listener, waits for the exact transport, verifies the
hardware serial and requires the network endpoint to be present; inventory uses
that verified endpoint rather than the old USB serial. Reconnect refreshed the
physical phone's network endpoint, and subsequent inventory saw USB and wireless
as available.

The phone was subsequently promoted to classic wireless ADB on its verified
serial and a wireless screenshot flow passed. Both USB and wireless transports
were available after inventory refresh. A system-UI package log capture also
passed. The previous empty launcher-log snapshot remains an observation gap.

Long Android fallback sessions now cap retained screenshot bytes and thin the
sample sequence when necessary, keeping original capture times and reporting
only retained samples as evidence. This limits memory use; it cannot raise the
underlying device capture frame rate.

A follow-up TV projection trial captured the synthetic playback tone and video
in one session, but the video contained only two frames over 0.89 seconds.
Host muxing used the recorded API start times and is approximate; it does not
establish audio/video synchronization. The test app was removed from the TV.

`device capture --kind recording --require-audio` now requires both an
executable audio recorder and a declared audio capability. A missing or
uninspectable output audio track degrades evidence. The option currently
returns `capability_gap` on the tested Android ADB transports: the experimental
projection owner is not integrated. An audio track may still contain silence.

### Review recording reference provenance — 2026-11-01

New recordings retain the device ID, lease ID and content digest alongside the
artifact. Review finalization requires each chapter to match the active lease
and retained bytes. A successor lease may stop an abandoned recording for
cleanup, but the result remains attributed to its originating lease. Existing
artifacts without this provenance remain readable,
but cannot be included in a newly finalized review because their origin cannot
be verified. These records are local producer metadata, not a hardware-signed
attestation of device output.

### Inventory refresh can delay lease operations — 2026-11-01

During concurrent full-suite activity, a controlled API trial read `/sessions`
in 0.02 seconds before issuing an explicit `/devices` inventory refresh; the
concurrent `/sessions` request and the refresh both exceeded 30 seconds, and a
subsequent session read took 9.56 seconds. Device Control's health endpoint
remained responsive during a similar occurrence. Inventory holds the shared
service mutex while it performs transport and attached-device probes, while
session reads and release also require that mutex, making lock contention a
confirmed contributor: an isolated, database-free regression with a delayed
strategy `Describe` call blocked `ListLiveSessions` until the probe completed
(the Android projection research run retains `inventory-lock-regression.log`). Database latency and
queued concurrent refreshes may add to the live delay. Lease state and active
run cancellation now use a separate mutex, preserving exclusive lease admission
without waiting for inventory probes. A regression held a probe open while
reading, validating, releasing and reacquiring a lease; it passed with the race
detector and the scenario unit phase. Inventory refresh now serializes identity
mutations with a dedicated lock but releases the general service lock around
transport and Bridge calls. Ordering tests ensure a forget completes after an
in-flight refresh, and the full control package passed with the race detector.
After managed restart, while a live refresh took 19.0 seconds, session reads
took 3 ms, cached-device reads under 1 ms and a retained screenshot read 4 ms;
both the TV and phone remained available. A subsequent trace found that
observer setup still described candidates synchronously under the service lock
and inventory described base strategies twice. Observer qualification now runs
in a cancellable, instance-owned task; a stale task cannot remove its
replacement, and duplicate base probes are removed while endpoint-specific and
device-scoped probes remain. Full control-package race checks and Test Genie
unit run `20261101-074733-143db975` passed. After restart, two explicit refreshes
each completed in approximately 10.0 seconds with both devices available;
before the observer fix, two comparable requests exceeded 35 seconds each.
The refresh still reaches its configured timeout, and identity mutations
intentionally wait for a refresh to preserve ordering.

### Lease-owned flow cancellation — 2026-11-01

A lease's active-run cancellation entry could be overwritten by a second flow,
and a flow could register after release between lease validation and execution.
Registration now checks the live lease and installs its cancellation atomically
under the lease mutex; a second active flow on the same lease is refused. Focused
regressions failed before the change and pass under the race detector, including
cancellation of the original run. External recordings can still surround a
single lease-owned flow; the restriction applies to overlapping flow runs.

### Visual capture contract and sync evidence — 2026-11-01

The prior P0 text claimed that all visual capture was verified redacted and
that consumers never receive capture bytes. That contradicted the operator's
full-frame-by-default request and the actual evidence endpoint. The target and
requirement now describe full-frame defaults, explicit supported redaction,
and honest metadata; unsupported video redaction is not presented as working.
Three named evidence regressions pass directly. The requirement remains in
progress pending trustworthy requirements-sync evidence. Full run
`20261101-065628-5d6abe1a` incorrectly promoted nonexistent named tests and
unearned targets; Scenario QA report `knw-1793517723391094766` tracks that
producer defect. The inaccurate promotions were reversed.

### Native recorder and screenshot-fallback overhead — 2026-11-01

The Android recorder samples screenshots while native video runs, so fallback
capture may add load even when native output succeeds. A paired SmartTV trial
with active YouTube playback measured 60 fps before each capture, then 3.02 fps
with native recording alone and 2.79 fps with concurrent PNG samples. The clips
contained 21 and 18 frames respectively, and only one screenshot completed in
the second trial. Those single measurements do not isolate or quantify the
incremental sampling cost; the native path itself was highly disruptive. Two
other setup attempts had no playing video and provide no performance evidence.
No fallback default was changed based on this inconclusive comparison. The
Android projection research run retains the full trial description and samples.

### Android screenshot integrity — 2026-11-01

The Android capability probe and observer previously accepted a valid image
header without checking the remaining compressed data; the direct screenshot
step did not decode it at all. A truncated PNG could therefore be reported as
available and retained as successful evidence. Both capture paths and the
capability probe now use the same full-image validation, with a decoded-size
bound, after removing any supported firmware preamble. The observer reports the
actual image format. A regression exercises a valid header with incomplete
image data, and a JPEG fixture checks format reporting. Device Control unit run
`20261101-155539-c25eeda5` passed; after managed restart, screenshot capture
succeeded on both the TV and phone. A refresh took 7.0 seconds versus 7.5
seconds in one pre-change sample, which is not a controlled performance result.

This validates encoded image integrity, not that every displayed layer was
captured or that a visually blank frame is wrong. Several blank-looking frames
were observed while the Hisense Device Assistant list was scrolling; their
cause and relation to actual display content remain unconfirmed.

### Recording setup cancellation — 2026-11-01

Android recording startup previously discarded the caller's context while it
read and changed display retention settings. A cancelled flow could therefore
continue starting a recorder or wait indefinitely on an ADB command. Startup
now uses the caller context for preparation and checks it before starting the
encoder. If preparation has changed settings, restoration uses an independent
eight-second deadline; stopping a recording also bounds restoration. Regression
tests cover pre-cancellation, cancellation during setup, independent bounded
restoration, and a failed restore. Android adapter race tests and Device Control
unit run `20261101-160543-af17216b` passed.

After deployment, a phone motion flow produced a 6.97-second native video at
about 13.35 fps; a TV motion flow produced a 13.28-second video at about 0.68
fps and was correctly marked degraded. Both devices restored their original
stay-awake, screen-off timeout, and idle settings. Four-second TV and six-second
phone recordings of static screens returned zero-duration video and were rejected;
the cause and whether a still-frame fallback is appropriate remain unverified.
Neither native recording contained audio. This change does not implement audio
capture or resolve the TV recording performance problem.
A later seven-second phone repeat yielded a playable screenshot-fallback clip
with two source samples (about 0.29 source fps), correctly marked degraded for
transition evidence. A separate direct seven-second native capture of the phone
produced seven decoded frames over about 5.88 seconds. These sequential trials
used different capture paths and do not establish why the earlier Device Control
recordings had zero duration or isolate the contribution of screenshot sampling.

### Dark recording evidence — 2026-11-01

A decodable near-black recording previously failed evidence creation and was
lost, even though the scene itself may have been dark. Such recordings are
now retained with `content_verified: false`, a degraded disposition and a
reason explaining that visible content could not be verified. A flow reports
`no_evidence` rather than claiming the visual result passed. The direct and
session recording paths both propagate degraded status; corrupt or undecodable
video still fails validation. A synthetic dark-video regression verifies the
retained bytes and flow disposition. Device Control unit run
`20261101-162015-a2988eea` and the evidence/control race tests passed. This does
not determine whether a particular dark device capture faithfully reflects the
screen, change the screenshot fallback selection, or add audio recording.

### Agent observation and decision-model qualification — 2026-11-01

Android semantic observations now expose bounded accessibility controls through
the shared lease owner and the CLI; live observations and semantic-target flows
worked on both the TV and phone. The agent can receive current semantics and
bounded action history, but its model receives no screenshot image. Its former
`observe` planning choice captured an image without passing it back to the
planner; that misleading choice is now withheld from model-backed planning.
Semantic targeting is limited to unique, enabled, actionable controls and checks
all visible identifiers against the existing policy gate. This heuristic cannot
classify every side effect. Unverified model claims cannot promote a flow.

A live TV dry run on the earlier build made an observation, then hit the
planning deadline; after removing that misleading choice, another dry run
rejected a key action without the required value. A phone dry run rejected an invented step while its current
ADB probes were unavailable. Neither is evidence of successful autonomous
navigation. Local synthetic decision-model trials found severe context
sensitivity in the fast route and better results in a small accurate-model
sample. They do not qualify either route for unattended actuation. The
[experiment and receipts](/home/matthalloran8/.vrooli/research/topics/tv-capabilities/runs/device-decision-prompt-20261101/README.md)
retain the tests and limits. Device Control still needs independently verified
agent goal assertions, visual planning support, and broader decision-model
evaluation before reusable automatic navigation is reliable.

Another dry run exposed an AI Gateway configuration mismatch: the local
`device-control.plan` candidate named an Ollama resource role that does not
exist. The candidate now uses the same `chat.default` resource role as the
gateway's structured extraction route, with gateway schema validation. After
managed restart, a synthetic local-only planning request returned a validated
answer from Ollama. A subsequent TV dry run exceeded the former ten-second
planner deadline; after extending the bounded deadline to 30 seconds, a live
dry run produced a semantic target but repeatedly planned against the unchanged
screen. Dry runs now stop at the first proposal, report `planned`, and do not
claim that the goal was achieved. This route repair does not establish
real-device agent accuracy or decision-model reliability.

An explicit `--expect-selected` outcome now lets the operator require one
exact accessibility control to be selected in the current complete snapshot
before an agent completion claim is verified. On the TV, an agent opened the
TV apps page and verified its selected tab; a separate check expecting Home
failed while TV apps was selected. This verifies only the supplied selected
control condition, not arbitrary goals or reusable-flow promotion. Android
selection and checked fields remain unknown when not reported.

Wireless reconnect for the phone returned no route to its saved endpoint, and
a fresh TLS service browse returned no services; neither result establishes why
the phone is absent. Promoted Android flows now preflight a known serial-bound
USB profile if implicit wireless preflight cannot satisfy the flow. Explicit
transport choices are retained and no effect is retried after actuation. State
reads can fill missing core fields from a secondary transport. The fallback is
covered by fixtures; it has not been physically tested with the disconnected
phone and does not yet cover every device operation.

After the Moto G7 (Android 10) was connected over USB, re-promotion refreshed
and verified its legacy ADB-over-TCP endpoint. Wireless debugging state reported
native mode unsupported and legacy TCP enabled. Following USB removal, one
screen-off assertion failed, the address stopped answering, and a subsequent
ADB shell read hung. The address answered and a wireless screenshot succeeded
after the phone was awake. This is one intermittent observation; sleep has not
been established as the cause. Android state and endpoint-identity reads now
have deadlines, and an all-failed state read is unavailable rather than a
successful empty observation. Tests cover those boundaries.

Native Android Wireless debugging has separate USB-only enable/disable and
assertion flow steps with setting readback. Android 10 reports the capability
gap and supports the separate legacy promotion route. The native flow remains
unvalidated on a physical Android 11+ device; setting readback does not prove
pairing or network reachability. Device health now distinguishes partial
availability from total loss when another transport has fresh evidence.

With USB reconnected, a bounded comparison produced one verified screen-off
transition; the phone answered network pings at five and fifteen seconds after
that transition and again after wake. A second screen-off assertion timed out,
so the second cycle is inconclusive. A separate USB wake passed, but a later
state read reported the screen off; durable restoration was not established.
These few observations do not determine why wireless ADB
became unreachable earlier or establish a causal relationship with sleep.
