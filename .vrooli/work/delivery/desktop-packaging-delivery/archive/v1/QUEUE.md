# Desktop packaging queue

## Handoff

Sequence 6 safe-boundary checkpoint: verified orchestrator `f8d106ed-6a46-4c3a-854a-0031e789628e` (task `0e255154-e8d9-4268-9609-890b4e94bc2a`) resumed with no await handle; current child listing is empty. No child or managed validation is in flight. Canonical feedback requests an overnight pause and reports the Desktop team disabled. This run is not currently parked; it will park after this checkpoint.

E3 remains unaccepted. Original worker `41a8e793-3817-4f66-a8a7-c85ff50891a0` is NEEDS_REVIEW/goal_blocked under prior parent `8ad257a4-3096-4eca-9cd7-ff3af33db8c8`; the Web Console session is gone, completion is unproven, and task `6041722d-aa98-4cd7-b179-b61310c2bf7f` is QUEUED. G8 remains 0/480 and G9 unearned. Run-private checkpoint artifacts exist, but no shared transfer or owner-indexed bundle is established. The bounded G8/G9 remainder and its preconditions are recorded below in `## Next`.

Latest safety readings remain: D7 metric normal at 03:15:45.708Z, memory-pressure alert active at 03:15:39.199Z (D7 fails); display 0/session 2 matched at 03:15:35.178Z, qualifying only that pair. Current resource admission is not established. Do not dispatch a worker, run measurement or validation, or change acceptance gates during the requested pause.

Resume only on explicit owner instruction. After resume, reread FEEDBACK.md and the effort signal log, verify run and child state, refresh D7/peer/admission, and then continue only the approved E3 scope. The owner can wake this same parked run with `agent-manager run wake --key f8d106ed-6a46-4c3a-854a-0031e789628e`; the Desktop team is reported disabled, so team enablement remains with its authorized owner.

## Needs operator

- [waiting for explicit owner resume after overnight pause] The latest canonical supervisor feedback requests a pause at the safe boundary and reports the Desktop team disabled. This run has no await handle and no direct child; no child or managed validation is in flight. E3 remains unaccepted: original worker `41a8e793-3817-4f66-a8a7-c85ff50891a0` is NEEDS_REVIEW/goal_blocked, task `6041722d-aa98-4cd7-b179-b61310c2bf7f` is QUEUED, G8 is 0/480 and G9 is unearned. Latest D7 metric is normal at 03:15:45.708Z but the paired alert was active at 03:15:39.199Z, so D7 fails; peer PID 3224 matches display 0/session 2 only. Current admission is not established. Do not dispatch or run validation during the pause. After explicit owner resume, reread feedback/signals and run/child state, then refresh same-parent preflight, paired D7, complete peer/session evidence and Agent Manager admission before continuing E3. Same-run resume command: `agent-manager run wake --key f8d106ed-6a46-4c3a-854a-0031e789628e`.

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
unproven and its Web Console session gone. It belongs to prior parent
`8ad257a4-3096-4eca-9cd7-ff3af33db8c8`; the E4 worker also belongs to that
prior parent. This orchestrator's prior children await resolved with no active
or ended children. Existing checkpoint manifests and artifacts remain in the
old run-private state directory; no shared transfer or owner-indexed bundle is
established. Sequence 5 removed the unsupported Nooch-lane attribution; current
Desktop admission remains unproven, and no current capacity result is claimed.

Prepared E3 remainder for later admission: use only `epochs/E3.md` scope and
existing implementation; preserve the old worker/task evidence and every gate.
After fresh same-parent preflight, D7 normal with no open memory-pressure alert,
complete peer/session evidence and actual Agent Manager admission, finish G8's
480-launch Linux matrix (six cells × four subjects × cold/warm × 10) with
journey replay and D7 condition metadata per sample, then finish G9 with one
passing Test Genie run and one wait for each of scenario-to-desktop and
performance-health after DPF-WA-004 findings and D7 clear. Do not widen the
epoch or infer acceptance. E3 task remains QUEUED; no successor was admitted.
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
2. **[parked] E3 — Experiment runner and `electron-optimized`** — no successor
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
- **E3 G8/G9 display evidence:** A192 provides the supported bounded read `vrooli host desktop-session --session-id <session-id> --display <local-display> --json`; it obtains peer identity from the selected X11 socket with `SO_PEERCRED` and performs the control-plane session check. The historical 2026-10-19 read and the fresh 2026-10-21T03:15:35Z read both match display 0/session 2 (peer PID 3224, UID 1000, active, unlocked, local). This qualifies only that selected pair; it does not establish other sockets, capture/input readiness, G8 or G9. Evidence: `docs/internal/delivery-reliability-research/sources/A192logs/desktop-current-read-20261019/`. The old missing-capability wording and unsupported Nooch-lane attribution are superseded. At 03:15:45Z the memory-pressure level was normal, but the 03:15:39Z evaluation had the memory-pressure alert active, so D7 fails and G8/G9 remain held. Keep G8 at 0/480, G9 unearned, E4 queued and E3 unaccepted.
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
- 2026-10-20 park: a withdrawn scheduling reservation and empty suite queue do
  not establish host capacity; require current D7 and peer/session evidence before
  dispatch or measurement.
- 2026-10-21 park: a fresh normal D7 metric does not satisfy the gate while the
  paired alert remains open; empty Test Genie runs and unrelated GPU claims do
  not establish admission to desktop work.
- 2026-10-21 park: an empty child await is not an unresolved producer; refresh
  the paired D7/peer observations and keep any future E3 brief bounded to the
  unmet gates without inferring capacity from historical claims.
- 2026-10-21 pause: stop at the safe boundary, checkpoint exact acceptance
  gates and a same-run resume command, then park the existing owner without
  changing the old worker's state or admitting a successor.

## Done

- **E1 — packaged-launch baselines**, accepted 2026-10-12: 57/30 units;
  +2,938 runtime and +476 test lines; four comparable cohorts. Full gates and
  evidence: `epochs/E1.md`. Findings-index persistence degradation remains open.
- **E2 — strategy seam and explainable selection**, accepted 2026-10-12: 19/30
  units; +1,415 runtime and +72 test lines. G1–G9 evidence: `epochs/E2.md`.
- **F1 — packaging foundations**, accepted: 45/60 units; +47 runtime and +249
  test lines. Full gates/evidence and remaining owner returns: `epochs/F1.md`.
