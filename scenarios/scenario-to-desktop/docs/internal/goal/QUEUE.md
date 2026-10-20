# Desktop packaging queue

## Handoff

Changed: Reconciled A192's bounded owner read, the latest `ROOT-20261019-DESKTOP-EXISTING-PEER-READ` feedback, current goal files, retained E3/SDA/BAS checkpoints and current source. The stale claims that no supported peer/session read exists and that RCL still owns the lane are superseded. FEEDBACK remains operator-owned and unchanged; its earlier text is superseded by the later root entry. No worker/task/leader was created and the old worker was not mutated.

Verified: A192's exact read `vrooli host desktop-session --session-id 2 --display 0 --json` at `2026-10-19T16:40:34.884294889Z` returned session 2, display-0 X11 peer PID 3224 / UID 1000, `matched=true`, active, unlocked, local, non-remote, reason `session_peer_matched`. The supported command obtains the selected socket's PID/UID through kernel `SO_PEERCRED` and checks logind/process/session-cgroup identity; it does not need a guessed or supplied PID. This result covers that selected display/session observation only; it does not prove other sockets, helper/capture/input readiness, current D7 normal/no-alert, matrix readiness, G8 launches, or acceptance. Evidence: `docs/internal/delivery-reliability-research/sources/A192logs/desktop-current-read-20261019/{README.md,review.md,probe.argv.json,probe.stdout.json,probe.exit}`.

The same safe owner projections report the original E3 run `41a8e793-3817-4f66-a8a7-c85ff50891a0` / task `6041722d-aa98-4cd7-b179-b61310c2bf7f` as `NEEDS_REVIEW` / `TASK_STATUS_QUEUED`, with no current task run. The run's exact handoff says its Web Console session is gone and retained goal evidence exceeds bounded replay; completion remains unproven. Do not change its state or infer completion. The original orchestrator projection is `COMPLETE`/CODEC_PIPE and its task is queued with no current run; when a continuing owner action is needed, use the supported same-run Continue route after its fresh preflight, not Wake (the safe projection has no await handle). Evidence is in the four `*.safe.json` projections in the A192 directory.

Lane reconciliation: the older FEEDBACK sentence that RCL owns the lane, E3 D10's corresponding hold, and the stale QUEUE wording are superseded by `ROOT-20261019-DESKTOP-EXISTING-PEER-READ` and A192's review: current coordination places Nooch's bounded slice ahead of Planner. RCL is no longer identified as owner. This does not establish that Desktop has the heavy lane or release any work; this handling used no heavy operation. FEEDBACK and the E3 plan-of-record were not edited; this handoff records the newer owner state.

Artifact/source comparison is read-only and anchored to the original worker's private `codex/tmp` checkpoint directory; no artifact adoption or transfer is recorded. The 2026-10-14 E3 checkpoint has 18/18 payloads and the SDA exact checkpoint 22/22; against current checkout, 1/18 and 2/22 entries respectively have matching hashes. The snapshot-hook checkpoint has 10/12 payloads; its missing entries are generated `scripts/build-v8-snapshot.js` and `scripts/startup-snapshot.js`, while current `.js.template` generator inputs are present. The 2026-10-14 fleet-final checkpoint has 3,077 manifest entries and 2,668 payloads; 2,661 payload-backed current files match and 7 differ: approved-dependencies.json, BAS `README.md` and `src/main.ts`, current E3.md, template-generator.ts/test.ts, and secrets-manager Electron `.gitignore`. Another 409 manifest entries have neither retained payload nor current path: 51 each under seven generated-shell roots and 52 under `vrooli-onboarding`. These absences cannot establish whether files were intentionally removed or merely unavailable in the capture. The separate pre-refresh BAS shell archive holds 1,361 files. These checkpoints are dated evidence, not an adopted current source diff.

Current bounded G2/G3 verification (2026-10-19): `cd packages/proto && make verify-committed-gen` exited 0 (`GOFLAGS=-mod=mod go run ./cmd/protogen verify`); no generated drift was reported and no generation ran. `cd scenarios/scenario-to-desktop/api && GOMAXPROCS=2 GOFLAGS=-p=1 go test -v ./generation ./build` exited 0. A retained JSON-result invocation of the same packages reported 187 test events passed, 0 failed, 0 skipped; package results were `scenario-to-desktop-api/generation` PASS (0.387s) and `scenario-to-desktop-api/build` PASS (0.618s). Its exact command was `GOMAXPROCS=2 GOFLAGS=-p=1 go test -json ./generation ./build`. Focused Jest command `./node_modules/.bin/jest --runInBand --runTestsByPath template-generator.test.ts optimization-profile.test.ts --testNamePattern='startup snapshot build files|keeps electron-sidecar as the all-off default|projects each optimized toggle independently|passes Electron fuse selections into the generated Electron Builder config' --verbose` exited 0: 4 passed, 21 filtered/skipped, 2 suites passed. Selected cases: `includes startup snapshot build files only when the profile enables snapshots`; `keeps electron-sidecar as the all-off default`; `projects each optimized toggle independently`; `passes Electron fuse selections into the generated Electron Builder config`.

Using the existing `DesktopTemplateGenerator` owner methods `getTemplateFiles` and `processTemplateFile`, the two snapshot hook outputs were rendered in a temporary directory and read back. `scripts/build-v8-snapshot.js` was 1,767 bytes, SHA256 `03757da590fa0a97222323a43e30a5ed6a284e38762a8d56be35263db7d91569`; `scripts/v8-startup-snapshot.js` was 207 bytes, SHA256 `55f9e868e1193169a269a77e412f2e304fbc9a126b808eaaf9f2438123455e76`. Each equals the hash of its current `.js.template` input because neither contains substitution tokens. Temporary output was removed. This inspects rendered JavaScript only; Electron/mksnapshot was not installed or executed and no binary snapshot/build artifact was produced. Relevant current source identities retained by SHA256 include `template-generator.ts` `959953f335abedf9bcd3ea067def4028ef33d4f73723ff23c15d397de6a5c894`, `template-generator.test.ts` `1e672d2530edf3f7dc47bb916f75b91eca9411f9d72a65d9e96ae81e7ff78b3a`, `optimization-profile.test.ts` `8526e58b58d8de6edf78e75e200ed43536ac0bb7b55f026120d8a17d475165c1`, `generation/strategy.go` `74d1cf001e3c523f27c64d782f72aa2b677a4759ef922bfbda28c25a1ac81f11`, `generation/strategy_test.go` `6b4751c6bdbd75830291760d5353d0b599d91fa4025a94d81a75d8070c1473ad`, and `build/service.go` `17479636f6e89c0596bee61edcf1f8860d6c330260c0b63e98b5faa510f954fb`. Checks of these listed source identities before and after validation matched. No product code, generated project output or dependency registry was changed.

### G1–G7 remaining work and evidence

| Gate | Current-source observation | Retained evidence and exact remainder |
|---|---|---|
| G1 | No E3 feature gap was identified in this source-only review. The E3/SDA/BAS checkpoints do not include `packages/delivery-ramp-go` source, so no checkpoint hash comparison was available for that module. | E3 records the exact `cd packages/delivery-ramp-go && GOWORK=off go test ./...` as passing on 2026-10-14. This is historical only. Re-run that exact gate in a later authorized validation window; no test was run here. |
| G2 | All eight generated Electron shell manifests currently declare Electron 44.7.2. Since the 2026-10-14 fleet checkpoint, `scenarios/scenario-to-desktop/templates/build-tools/template-generator.ts` and its tests changed: template-specific defaults now merge before derived values and port entries are sorted. BAS `platforms/electron/src/main.ts` gained `playwright_driver_frames`; its README changed the template label from `advanced` to `basic`. | Current exact generation/build package scope now passes: 187 test events, 0 failed/skipped, plus current Proto freshness exit 0; commands and package timings are recorded above. This is a package-test result, not a new regeneration or parity comparison of all retained generated shells. No BAS regeneration or generated-shell comparison was done. |
| G3 | Current source in `scenarios/scenario-to-desktop/api/generation/strategy.go`, `templates/build-tools/template-generator.ts`, `templates/vanilla/main.ts` and runtime helpers contains four independent profile toggles and keeps `electron-sidecar` default. SDA exact-version handling is in `scenarios/scenario-dependency-analyzer/api/internal/installgateway/{gateway.go,template.go}`; current vanilla `package-lock.json` and approved-dependency records pin Electron and electron-mksnapshot to 44.7.2. | Current focused tests pass: sidecar default, independent toggle projection, Electron Builder fuse/snapshot config, and conditional hook selection (four selected tests; 21 filtered). Both current hook templates rendered through the owner generator into temporary output with identities recorded above. This does not execute Electron/electron-mksnapshot, produce or inspect a binary startup snapshot, or establish generated shell runtime behavior. Keep the snapshot toggle; no demonstrated G2/G3 cause required a source fix. |
| G4 | Packaged API capture/compare/validation source and tests are present in `scenarios/performance-health/api/internal/packaged/{service.go,types.go,service_test.go}`. | E3 records `cd scenarios/performance-health/api && go test ./internal/packaged/...` passing on 2026-10-14. This handling did not run it, and the retained checkpoints do not provide a dated comparison of this package against current source. Preserve the historical result; obtain a current exact receipt later. |
| G5 | Packaged CLI capture/compare source and tests are present in `scenarios/performance-health/cli/domains/packaged/{handlers.go,handlers_test.go}`, with independent toggle flags mapped into the request. | E3 records `cd scenarios/performance-health/cli && go test ./domains/packaged/...` passing on 2026-10-14. This handling did not run it; preserve it as historical and obtain a current exact receipt later. |
| G6 | `ExperimentManifest.Validate` in `scenarios/performance-health/api/internal/packaged/types.go` checks schema/version, Linux/macOS platform, four unique reference subjects and journeys, six required cells, toggle combinations, cold/warm modes, sample count 10, environment fields, and paired artifact path/digest; validator regressions are in `service_test.go`. `scenarios/performance-health/.vrooli/program-runtime/packaged-experiment.{json,py}` declares validation, 24 builds, sequential cell capture, journey replay and report bindings. | E3 reports the G6 owner suite passed historically; the program was prepared as `prog_608078be-62e9-4bfa-bac6-bb805b6cd675` but not submitted. No current validation/test receipt was created in this restricted review. |
| G7 | Current report code in `scenarios/performance-health/api/internal/packaged/service.go` groups by subject/profile/condition and computes metrics/medians; it carries toggle attribution, journey receipts/parity, reliability, artifact digests, host/display/profiler/power metadata and explicit missing-data status into JSON/Markdown. Related tests are in `service_test.go` and CLI `handlers_test.go`. | E3 reports the report suite passed historically. There is no current E3 comparison report because the prepared program was not submitted and no G8 measurement ran. Preserve the code-level historical pass; do not imply current measured results. |

The last recorded G1–G7 passes remain historical E3 slice-log evidence, not newly accepted gate states. The scoped current-source G2/G3 package checks now pass as recorded above. Remaining source/artifact questions are all-shell/BAS generated-output parity and binary snapshot production/loading. The E3 cumulative growth budget remains unproven because the original pre-E3 line baseline was not captured. No gate definition or prior pass claim was changed.

Remaining: E3 is unaccepted. Keep G8 at 0/480 and G9 unearned. D7 must be freshly read as `memory_pressure.level=normal` with no open `memory_pressure` alert before any later G8/G9 work. The A192 peer result is one bounded selected display/session association only. No current display-wide readiness, G8, G9, or E3 acceptance is established.

Unverified: the intent behind the 409 unavailable generated-shell manifest paths; all-shell/BAS generated-output parity and runtime behavior; Electron/mksnapshot binary snapshot production/loading; current G1/G4/G5 execution results; current G6/G7 tests and program validation; current D7; and the E3 cumulative growth budget. The source/package portions of G2/G3 are now checked, but these observations do not accept the E3 gates or epoch.

Exact next action: preserve the old worker/task in review. At the next permitted validation window, continue only under the same original Desktop orchestrator lineage through the existing supported same-run Continue after fresh owner preflight. Ask BAS's owner to regenerate/compare the changed Electron shell outputs through its governed generator, then perform any runtime/browser/capture checks only when Planner and lane sequencing permit. Keep remaining G1/G4/G5/G6/G7 evidence, D7, the 480-launch G8 matrix, G9, and the unproven cumulative growth baseline outstanding; retain every original gate. Do not create a worker/task/leader from this handoff.

## Forecast

- Destination (GOAL.md, TARGETS.md quality bands). Current gap after E1:

  | Measure | Now | Target |
  |---|---|---|
  | Selectable strategies | 1 measured (`electron-sidecar`) | ≥ 3 measured |
  | Auto-selection | none (unchanged by F1) | live and explainable |
  | hello-desktop cold time to interactive p50 | 1.805 s (`packaged-launch.v1`; E1 current electron-sidecar baseline) | ≤ 1.0 s |
  | Real reference scenarios | 4 selected references have comparable cold/warm baselines | measured |

- Last five accepted epochs: F1 delivered 45/60 work units (75%); E1 delivered
  57/30 (190%); E2 delivered 19/30 (63%). The recent combined ratio is
  121/120 ≈1.01. E1 runtime growth was +2,938 against its +3,000 cap.
- Remaining queue sum: E3–E6 estimate 105 work units; at 1.01 this scales to
  about 106. The remaining destination gap is strategy selection and performance
  quality, not baseline availability. Calibrate step-back K and spend threshold
  after three accepted epochs.

## Next

F1, E1 and E2 remain accepted. E3 is unaccepted; original worker
`41a8e793-3817-4f66-a8a7-c85ff50891a0` / task
`6041722d-aa98-4cd7-b179-b61310c2bf7f` remains NEEDS_REVIEW with completion
unproven. A192 records the bounded display-0/session-2 peer match and the
original owner projections. The latest ROOT feedback supersedes the old RCL
lane ownership and no-peer-read statements: current coordination places
Nooch's bounded slice ahead of Planner. This is not a Desktop heavy-lane
release. Exact source/artifact reconciliation and G1–G7 evidence status are in
Handoff. Preserve G8=0/480 and G9 unearned; no worker/task/leader was added.
E4 remains queued.

Every slice is a feature epoch: it adds a strategy, a contract or a measured
capability. Each brief must also:
- keep shared types mobile-ready (design §7);
- leave each working strategy selectable;
- list the code it replaces, so duplicate owners are deleted rather than
  forked. Examples: `launch-trace.v1` is superseded by `packaged-launch.v1`, and
  `framework: electron`-only validation is replaced by the strategy registry.

1. **E2 — Strategy seam and explainable selection** — accepted; full brief and
   gates: `epochs/E2.md` (30 units).
2. **[in progress] E3 — Experiment runner and `electron-optimized`** — no successor
   is admitted. The prior worker remains NEEDS_REVIEW/completion-unproven. Preserve
   toggle attribution, mobile-neutral shared contracts, every original gate and
   DPF-FB-017 containment; DPF-FB-021's matched snapshot route and shell refresh
   are recorded as delivered historically. Full brief: `epochs/E3.md` (30 units).
3. **E4 — `wails-inprocess`** — parked until F1/E1/E2/E3 acceptance and operator
   WebKitGTK setup; full scope and gates: `epochs/E4.md` (30 units).
4. **E5 — `tauri-sidecar`** — full scope in the accepted queue design; 20 units.
5. **E6 — Decide, default and formalize** — full matrix, reliability soaks,
   evidence-based selection policy, decision records, synthesis report, skill
   and recurring program; 25 units.

## Waiting on others

These never hold acceptance or other slices.
- **E3 G8/G9 display evidence:** A192 provides the supported bounded read `vrooli host desktop-session --session-id <session-id> --display <local-display> --json`; it obtains peer identity from the selected X11 socket with `SO_PEERCRED` and performs the control-plane session check. At `2026-10-19T16:40:34.884294889Z`, display 0/session 2 returned peer PID 3224, UID 1000, matched, active, unlocked, local, non-remote (`session_peer_matched`). This qualifies only that selected display/session observation. It does not establish other sockets, capture/input readiness, current D7, G8 or G9. Evidence: `docs/internal/delivery-reliability-research/sources/A192logs/desktop-current-read-20261019/`. The old missing-capability wording and RCL ownership claim are superseded by the latest ROOT feedback; coordination currently places Nooch's bounded slice ahead of Planner, without declaring a Desktop lane release. Keep G8 at 0/480, G9 unearned, E4 queued and E3 unaccepted.
- **F1 G8 — secrets-manager journey.** Artifact gate passes with retained current-source bundle and checksums; desktop pointer-click journey remains failed (`surface_unchanged`). Production authorization remains required. Evidence: `epochs/F1.md`.

- **E4 remainder — Wails in-process packaging** (`epochs/E4.md`). After F1/E1/E2/E3 acceptance, resolve BAS shell drift, operator setup, Test Genie retry conditions and supported WebKit automation attachment; complete its admitted gates. Never infer parity from a second browser or window visibility.
- **F1 BAS shell regeneration** — owner request is recorded through BAS's goal FEEDBACK route after the team-inbox refusal. Return when BAS accepts/completes it, regenerates through its owner generator, passes drift tests and reports source/output identities. Do not edit BAS-owned output.

- **F1 D3/D4 complete; G8 artifact gate passes, desktop smoke journey outstanding** — trust-mode
  guard changes, focused regressions, governed C3 deletions, D4 lifecycle
  operations and retained current-source artifact evidence are recorded in
  `epochs/F1.md`. The separate pointer-click smoke journey fails with
  `surface_unchanged`; retain it as an open validation finding.

- **M1 — Mobile packaging effort** (Android, then iOS) on the same contracts.
  - Returns when the operator starts it after E6.
  - Preconditions found at commissioning:
    - no real scenario's API is reachable from a mobile app (UI resolves
      `localhost`);
    - the iOS project generator writes a comment-only `project.pbxproj`;
    - the iOS build never routes to minimouse;
    - AVD `vrooli-api36` has `hw.ramSize=96M` (invalid for measurement);
    - Android launch timing (`am start -W`) and `dumpsys meminfo` are not
      implemented.
- **BAS WebDriver attach** — only if E4 shows the scenario-to-desktop-side
  adapter cannot satisfy journey parity. Returns on the BAS delivery
  orchestrator's answer to a FEEDBACK.md request.
- **Wails WebKit automation attach** — E4 G4/G5 remain unverified until Wails
  exposes a supported automation-controlled WebView hook or WebKitWebDriver
  supports constrained authenticated attachment to an existing WebKitWebView.
  Do not launch a second browser or infer `app_interactive` from window
  visibility.

## Qualification findings

(none)

## Lessons

- Prove current-source builds and root causes before adding measurements or
  strategies; preserve historical and current evidence separately.
- Estimate from measured scope. E1 attributed 99.98% of the historical 6.178 s
  waterfall; do not ship optimizations without current-build attribution.
- Isolate staging/cache paths and verify owner suites before costly cross-scenario
  bundle work. Artifact checksums do not prove export readiness.
- Treat generated shells and templates as separate owners. Classify root-cause
  epochs that retain capability as `feature`; record measured growth without padding.

## Done

- **E1 — packaged-launch baselines**, accepted 2026-10-12: 57/30 units;
  +2,938 runtime and +476 test lines; four comparable cohorts. Full gates and
  evidence: `epochs/E1.md`. Findings-index persistence degradation remains open.
- **E2 — strategy seam and explainable selection**, accepted 2026-10-12: 19/30
  units; +1,415 runtime and +72 test lines. G1–G9 evidence: `epochs/E2.md`.
- **F1 — packaging foundations**, accepted: 45/60 units; +47 runtime and +249
  test lines. Full gates/evidence and remaining owner returns: `epochs/F1.md`.
