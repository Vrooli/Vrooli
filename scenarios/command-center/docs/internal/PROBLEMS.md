# Problems

Known defects and divergences, newest first. This file is the honest record of where the code differs from the documents around it.

## Broadcast / LPBS production gate — 2026-09-15

| Finding | Evidence | Disposition |
|---|---|---|
| Production LPBS is unavailable and Broadcast cannot be promoted to `NOW`. | `evidence/after/production-health-2026-09-15.txt` records HTTP 502 from `https://vrooli.com/health`; Plan Manager finding `f84cc56b-c89a-40bf-8d1b-331005276cce`. | Open, owned by the concurrent deployment workflow. Keep LPBS-bound readings `IN-REACH`; do not fabricate promotion evidence. |
| Local override now supplies typed LPBS digest, traffic, revenue, funnel and leaderboard readings. | `evidence/phase-12/broadcast-mark.json`, `broadcast-hide.json`, and `evidence/phase-13/direct-dom-summary.txt`. | Implemented locally. Override origin is explicitly labeled and is not production proof. |
| Unknown countries and update checks are not equivalent to complete geographic or install attribution. | `SOURCE-MAP.md`; current producer contracts. | Open limitation; retain `Unknown` and `update checks` wording in the UI. |
| Test Genie workflow health reports legacy case failures and unavailable experience-profile bindings. | Runs `20260915-193610-310ce0e3` and `20260915-193847-155d9d84`; Plan Manager finding `731677e8-9eb6-49b0-8031-1e523d776438`. | Dispositioned as validation infrastructure/legacy debt; new local UI behavior has focused unit/build and direct DOM evidence. |

## Broadcast presentation density — 2026-09-15

| Finding | Evidence | Disposition |
|---|---|---|
| Broadcast rendered the entire room's 23 readings during every beat, overwhelming the desktop supporting strip. | `RoomPage` previously selected only the beat hero; the supplied desktop capture showed `20 measured · 23 signals` and layout overflow. | Fixed: Broadcast beats now declare `readingIds`; hero, scene, supporting strip, counts, and reading-time calculations use only the active group. Landscape paging has an eight-tile safety ceiling; focused UI, fit, registry, type-check, build, and managed-runtime checks pass. |

## Performance audit — 2026-09-15

| Finding | Evidence | Disposition |
|---|---|---|
| Every room surface re-rendered on each 250 ms cycle tick | `progress` sat in the controller context value, so every `useBoardController` consumer committed four times a second. | Fixed: `BoardProgressContext`; regression test `advances the cycle rail without re-rendering the room surfaces`. |
| The cycle rail stuttered and a held beat looked frozen | The 250 ms `setInterval` and the rail's 250 ms `transition` raced, so the transition finished before the next target and the fill stalled 100–180 ms then jumped (live rAF probe showed `d=0` runs); a hold froze the fill short of the segment end and marked nothing. | Fixed 2026-09-15: `tickCycle` drives progress from `requestAnimationFrame`; holds wait at the segment end and set `data-held`; the CSS transition is a 120 ms jank buffer. See PROGRESS 2026-09-15; regression tests in `cycle.test.ts` and `AmbientDisplayShell.test.tsx`. |
| `samples=hide` can loop forever and peg the tab | `RoomPage` advanced hidden-hero beats with `(beatIndex + 1) % beats.length`. When no beat's hero is measured but a supporting reading is (`visible.length > 0`), it advances every render without converging. | Fixed 2026-09-15: `nextMeasuredBeat` jumps once to a measured-hero beat and stops at `-1`; regression tests in `hero.test.ts`; live `forge?beat=3&samples=hide` settles instead of looping. See PROGRESS 2026-09-15. |
| A scene draw exception killed the animation loop | `frame()` allowed a negative `t`/`dt`, so a scene could call `arc`/`drawImage` with a negative radius (observed `arc(...-0.8...)` in `signalConstellation`); the throw escaped the rAF loop, so the canvas stopped animating. | Fixed 2026-09-15: clamp the scene clock to non-negative, guard `drawGlow`, and catch a failed draw in `AmbientCanvas` to fall back to the composed still; regression test in `engine.test.ts`. |
| A whole tab could hang on a weak/software-rendered display | `probeTier` chose `full` from `hardwareConcurrency` and WebGL presence alone, so a TV with software WebGL ran the heaviest animated scene. Headless (SwiftShader) at 3840×2160 with CPU throttling showed intermittent 2–8 s main-thread blocks, worst on the release-ladder (`funnel-cascade`) beat. | Fixed 2026-09-15: detect software/virtualised renderers and keep the still; route few-core / low-memory / very large panels to `reduced`; cap the canvas at 3 MP; paint `reduced` at 30 fps. See PROGRESS 2026-09-15; `?tier=still` remains a forced escape hatch. Root cause on the operator's TV unconfirmed (no console access). |
| An auto-scroll pass could stall when its measured geometry churned | The step effect depended on the measured stop count, so a re-measure that changed it cleared the pending step timer; recurring churn cleared it every frame and the pass never advanced. This is the failure class behind the 90 s panel hold and a possible cause of a release-ladder hold. | Fixed 2026-09-15: read the stop count from a ref (not a dependency), add a watchdog that completes the pass after its worst case, and a controller backstop that releases and skips a beat stuck past `reading time + MAX_HOLD_MS + 30 s`. See PROGRESS 2026-09-15; 300 s live soak shows no stall. |
| An auto-scrolling panel held the room for the full 90 s | The optional counter line rendered only when a row was wholly visible; its appearance shrank the viewport it was measured from, so `stops`/viewport flipped every frame (3↔4, 91↔104), the step timer was cleared before it fired, and `read` never became true. Observed live on Forge beat 3 (`goal_progress`) at 1280×560. | Fixed 2026-09-15: reserve the counter line whenever the list is scrolling (nbsp when no range is named) and tolerate a pixel of churn in `sameGeometry`. See PROGRESS 2026-09-15; 180 s live run shows no hold; regression in `AutoScroll.test.tsx`. |
| The Forge release-ladder details looped forever and held the beat | `AutoScroll`'s step effect ignored its `read` flag, so it restarted from the top after one pass. Separately, `AUTOSCROLL_TIMING`/`STRIP_PAGE_MS` are fixed milliseconds while beat durations scale with `?cycle`, so a short cycle left the pass longer than the beat: the list never reached `read`, the beat held every visit, and each remount restarted the pass. | Fixed 2026-09-15: the list rests at the top after one pass, and `cycleScale` scales auto-scroll and strip-paging timing with the beat. See PROGRESS 2026-09-15; live stall at 1280×560 dropped 9.9 s → 4.0 s (cycle 20) and 8.4 s → 1.1 s (cycle 30); regressions in `AutoScroll.test.tsx` and `cycle.test.ts`. |
| Per-frame layout, style and reading resolution in the scene loop | `AmbientCanvas` called `getBoundingClientRect` ×3, `getComputedStyle` and `sceneData()` in every frame, and twice per frame during a crossfade. | Fixed: 250 ms layout/palette cadence; readings resolved on change. Not isolated in a measurement; the expected saving is small. |
| Colour-string parsing and per-segment strokes in scene loops | orbitalField issued about 2,750 stroke calls a frame; hive-lattice parsed about 600 `rgba()` strings a frame. | Fixed; see PROGRESS 2026-09-15 for A/B numbers. |
| orbitalField re-stroked 110 static orbit rings every frame | Layer breakdown: removing the rings took the frame from 18.5 to 5.2 ms. | Fixed with a cached ring layer; see PROGRESS 2026-09-15. Remaining orbital, hive and flow cost is moving content (stars, glows, cells, sparks). |
| meridianArc was the costliest scene and drew stray rectangles | 31.7 ms/frame in the eight-scene survey. Missing `beginPath()` re-stroked coastlines and the prior frame's clip path (quiet-zone boxes and canvas border). | Fixed; see PROGRESS 2026-09-15. |
| `frameIsBlank` reads the whole canvas once per scene mount | Measured 11.2 ms at 2880×1800 (software raster), once per first visit to a room, behind the transition veil. | Accepted. Many small reads can trigger Chrome's readback heuristic, which moves the canvas to CPU rendering. Downscaled or sparser sampling weakens a check that guards against blank rooms. Neither is worth about 11 ms per room change. |
| Canvas density is the largest remaining cost | Raster dominates orbital, hive and globe. At 1.5× instead of 2× density: orbital 17.5→12.6 ms, hive 15.0→7.6, flow 8.1→4.9, globe 19.6→14.2. | Decided 2026-09-15: the operator keeps 2× density and 60 fps. Further work must be pixel-preserving. A 30 fps cap would halve the cost if the display hardware ever needs it. |
| No performance-health baseline | BAS shares one browser, and CDP tracing is browser-wide, so concurrent sessions fail `Tracing.start`. QA `knw-1789451555082051158`. | Not ours to fix. Re-audit with `performance-health audit run command-center --workflow perf-room-cycle` when BAS is quiet. |
| drawn-fps is unusable | The analyzer reports `drawn-fps=0.0` with a 1.1e9 ms frame duration. QA `knw-1789452282548236326`. | Do not set a `drawn_fps_min` budget until it is fixed. No per-flow budget is set yet: one after-sample from a contended host is too noisy to ratchet. |

## Work ladder — cycle rail freeze and jitter (2026-09-15)

- Rung: W3
- Evidence: `CC-P1-008` ("the cycle rail visibly stops and restarts") and `CC-P1-017` ("a beat does not end until its strip has shown every page … the rail waits at the segment's end so a held beat reads as reading, not frozen") already define the expected behavior. Live Chromium probe against the running scenario (`/forge`, 1280×720) reproduced the defect: the active fill froze 5.7–7.7 s at `scaleX(0.974–0.992)` with `data-paused` empty (hold), and rAF sampling showed 100–180 ms `d=0` stalls between jumps (transition/tick race). Cause is implementation, not contract.
- Blocker: none; repaired and validated under the scoped UI suite.
- Measured: 2026-09-15

## Morning walk friction review — 2026-09-06

The operator authorized live-path repair and friction analysis. This review separates
program execution, source truth, publication, and operator usefulness.

| Finding | Evidence and ownership | Disposition |
|---|---|---|
| Prep cannot publish its declared unavailable result | The skill requires publication; `Publish` accepted only ok/partial. | Fixed: preserve unavailable envelopes under the same phase, time, checkpoint, channel, and receipt checks. Failed/malformed programs remain rejected. |
| Formatting forces manual payload repair | Prior attempt `vision-walk-prep-20260905-01` required `jq -c` before publication. | Fixed: the owner checks compact JSON size. Whitespace-only changes retain replay identity. |
| Pending work falsely appears empty | v3 requested `review` and `blocked`; the owner's real review state is `in_review`, with three live records. | Fixed in prep v4. Behavioral fixtures now honor requested statuses instead of returning every fake row. Owner filter validation filed as QA `knw-1788673385526353410`. |
| Retrospective misses archived completed work | v3 left the owner archive filter unspecified, excluding archived completions. | Fixed in v4: include all completed records before ordering and bounding. A regression checks an archived recent completion against an older retained item. |
| Runtime interruption loses diagnostic context | Program `prog_3f203ec7-56f0-4f56-a908-b165fc6176a7` started 05:20:15Z; runtime restarted 05:20:31Z; caller got EOF and storage remained RUNNING. | Program Runtime now returns acceptance identity before waiting and reconciles interrupted records at startup. See its PROBLEMS entry. |
| Repeated input flags silently drop earlier selectors | `--input channel=test --input limit=3` produced channel=operator. The attempt was read-only; it did not publish test data into the operator channel. | Fixed in Program Runtime, with a real CLI-parser regression. |
| Preparation requires an extra fleet-health operation | Exact `heartbeat-fleet-health` result is still CLI-only. | Existing owner obligation `idea/prompt-manager-governed-fleet-health-read`; retain the explicit supplement until the binding exists. |
| Relevant evidence requires repeated manual inspection | Handoffs are missing, team notes are clipped prose or serialized wrappers, and an alphabetically bounded goal list does not prioritize decisions. | Next capability work: owner-defined briefing rows with stable identities, status validation, evidence references, selection rationale, and typed changes. Do not parse private state inside the program. |
| Empirical sample can mislead | Live friction digest `prog_eb109fc2-42c4-4609-87cf-82ff1e249c50` read 850 episodes across 40 runs, zero attributed to program-runtime, with a capped window. A prior scheduled prep attempt records operator provenance. | Zero matching episodes is not zero friction. Audit runtime caller attribution and learning provenance before measuring improvement. No causal usefulness claim. |
| Validation queue is opaque | Unit runs `20260906-053049-1625a740` and `20260906-053413-659794ee` stayed queued beyond their estimates and exposed no admission constraint. | Durable waits attached; QA `knw-1788673270905679782`. Do not treat queue delay as a code failure. |
| Toolchain read transiently fails | A Go command reported installed standard packages absent; subsequent stat and identical targeted test succeeded. | Cause unverified; control-plane/toolchain QA `knw-1788673271819878597`. No host repair was implemented in a scenario. |

Validation: unavailable-publication and archived/review-selection regressions failed
before repair and passed after it. All 16 prep behavioral cases and focused owner
operation tests pass. Program Runtime repository/handler/CLI tests pass, including
focused race tests. Live prep `prog_409ec470-44ee-4794-9772-1b9a5c780545` completed in
57.7 seconds with 12 phases, 10 readable sources, three source gaps, and one
stale/undated section. Final v4 program
`prog_bf77efc3-a408-40a4-bb93-b9e72bea115e` completed in 51.175 seconds,
selected all three real in_review records, and retained 12 phases, 10 readable
sources, three unavailable sources, and two stale/undated sources. Publication
receipt `6b6d1b3e-7c4d-4d04-896b-8eb826fc7bdb` was independently read back:
the full envelope, phase-aligned briefing and exact fleet aggregate match.
Formatted JSON was accepted without caller compaction; no checkpoint was created.
Learning receipt `3bf6807f-6a98-4989-a17b-1136c3901cd5` records preparation
success in a repair-assisted cohort, omitting unobserved effort counts.
Full unit runs remain queued; the additional programs/skill-set submission was
refused because the existing Command Center run owns its slot. Focused passing
checks do not substitute for these pending suite verdicts.
No operator conversation or usefulness verdict has occurred during this repair.

Work ladder: W0 compares the authorized `command-center-morning-walk` goal with
OT-P0-009; both require bounded honest evidence and durable continuity. Business
and requirement gates pass. These defects are W3 implementation/fixture repairs.

**Standing note (2026-09-02):** the immersive display and core reading surfaces are implemented. The integration registry, typed feature state, source-time qualification, Prompt Manager transmitter, CLI parity, and confirmed scenario lifecycle action seam are now implemented; remaining rows describe upstream data ownership, visual evidence limitations, or validator/dependency infrastructure limitations.

---

## Defects in this scenario

### Resolved — Five room scenes now draw deterministic nonblank compositions

**Observed** 2026-09-01 against a running instance, headless Chrome with software WebGL at 1600×1000.

Each room binds a named composition from `ui/src/scenes/` — orbital field, hive lattice, flow current, ledger columns, signal constellation, panorama constellation — drawn by a shared 2D-canvas engine whose fields are the room's own readings (bodies per running scenario, cells per portfolio entry, sparks per created item, ring drop-off per funnel stage, node rings in each room's provenance). `AmbientCanvas` samples the whole first frame, not one pixel, and falls back to a composed still if nothing painted. Quiet zones under the hero and the supporting readings keep bright bodies out of the figure layer, and the figure layer composites above the scene.

The BAS scene cases now evaluate actual canvas pixels rather than only checking for a mounted element.

### Resolved — Every live metric reported UNAVAILABLE

**Observed** 2026-09-01. `GET /api/v1/rooms/mission-control` returned `value: null`, `trust: UNAVAILABLE` for all five `NOW` metrics with both sources healthy. Two causes: the Swarm Manager base URL fell back to a hardcoded port that was never assigned, and the value join (`findNumber`) walked the upstream JSON for keys that do not exist in either payload. Fixed by resolving source ports at fetch time through `api-core/discovery` (`api/directory.go`) and by naming an explicit selector per metric (`api/selectors.go`); an unknown selector is now a stated `trustReason`, never a guessed number. A cached value is now served as `CACHED` with its age when a fetch fails, instead of being dropped.

### Resolved — Provenance was rendered as alarm

The `.cc-badge-gap` red chip is gone. Provenance is a material on the figure itself — solid, dimmed, hollow, dotted — resolved in exactly one place (`ui/src/lib/provenance.ts`), and the greyscale companions in the evidence set stay unambiguous.

### Resolved — The scene tests reject blank canvases

The BAS cases now sample rendered canvas pixels and require a nonblank composition. An empty
canvas no longer satisfies the scene assertions (`CC-P1-003`).

### Resolved — Kiosk controls use the shared spatial vocabulary

`useKeyboardShortcuts`, `useSpatialNav`, `useFullscreen`, `useWakeLock` and `useGamepad` are written and unit-tested. Grep across `ui/src/pages`, `ui/src/components`, `App.tsx` and `main.tsx` finds no caller. There is no navigation of any kind: routes are reachable only by typing a URL, and the keyboard map for rooms 1–6 is defined but never registered.

Superseded by `CC-P1-009` and the shared `GamepadAction` vocabulary. The scenario-local hooks were removed and `BoardController` now owns keyboard, pointer, touch, gamepad, URL, pause, cycling, and fullscreen behavior.

### Resolved — Unused animation/chart declarations removed

The three.js stack, postprocessing packages, self-hosted font faces, and local React Component
Library package are governed; `framer-motion` and `recharts` are absent from the manifest and
production bundle. The analyzer has no removal verb, so only the direct importer declarations
were removed with a scoped lockfile patch after source/bundle verification.

---

## Remaining divergences from the design contract

Dated 2026-09-01. Delivered P0/P1 requirements are removed from this section. The
remaining items are either upstream data limitations or a dependency-tooling limitation.

| Divergence | Owner | Contract / requirement |
|---|---|---|
| Ledger revenue and subscription readings are implemented against LPBS, but production promotion remains pending while `vrooli.com` is unavailable. | monetization / deployment owner | [SOURCE-MAP.md](../concepts/SOURCE-MAP.md), `CC-P2-003`, `evidence/after/production-health-2026-09-15.txt` |
| Broadcast source plumbing is implemented against LPBS; social and SEO remain explicit unowned gaps, and production promotion is blocked by the 502 origin outage. | marketing-crew / deployment owner | [SOURCE-MAP.md](../concepts/SOURCE-MAP.md), `CC-P2-004`, `evidence/after/production-health-2026-09-15.txt` |
| Governed local-package installation currently emits a registry install command for the approved `@vrooli/react-component-library` file record and fails with npm 404; the manifest and lock entry are retained from the approved record. | scenario-dependency-analyzer | Phase 9 |
| `AmbientDisplayShell` and `CycleController` remain scenario-local (`ui/src/components/AmbientShell.tsx`, `BoardController.tsx`): they bind to react-router and the board API, and the library's ingest has no seam for host-provided routing yet. The four primitives beneath them are library assets. | command-center / react-component-library | Decision 10 |
| The contrast floor was measured under software WebGL (SwiftShader) at 1600×1000, ten frames per room, digit rows only. Every room clears 14:1. A GPU run and a portrait run are still owed. | command-center | Phase 12 |
| Test Genie’s DOM exporter does not provide the `data-render-ms` artifact, so frame-budget evidence records deterministic tier draw budgets and the runtime measurement seam but not numeric browser timings. | test-genie / BAS | Phase 12 |

---

## Blocked on other teams

Neither is scenario work. Both are tracked as source bindings in [SOURCE-MAP.md](../concepts/SOURCE-MAP.md).

- **Ledger has LPBS revenue and subscription readings, but production promotion is pending.** Coverage remains `IN-REACH` because the production origin returned HTTP 502. (`CC-P2-003`; evidence in the dated production-gate entry above)
- **Broadcast has an LPBS source for business analytics.** Social reach and SEO remain unowned capabilities, so those gaps remain explicit rather than being presented as LPBS measurements. (`CC-P2-004`)

---

## Validation state of this documentation

Recorded 2026-09-01, after the rewrite.

| Validator | Result |
|---|---|
| `business-health validate scenario command-center` | **PASSED**, no findings. 18 operational targets, 31 requirements, every `prd_ref` resolves and every target is covered. |
| `vrooli scenario validate` | **PASSED** — 121 manifests, 394 dependency edges. |
| `experience-manager spec validate command-center` | **PASSED**, no findings. |

Re-run 2026-09-01 after the review corrections: `business-health` **PASSED** with 18 targets and 32 requirements; `experience-manager` **PASSED** after the canonical version-scoped ExperienceSurface contract was added.

The resolver now finds the canonical version-scoped `ExperienceSurface@1.0.3` contract, so this scenario's experience gate is clear. The pixel BAS suite proves `room-never-blank`; experience-manager currently requires its unrecognized `render-nonblank` claim to remain aspirational.

Every schema error in these specs was fixed: static regions now declare `states: ["static"]` as the schema requires.

**Claim tiers.** Claims remain `aspirational` unless their named machine check is present in the
experience suite. The `room-never-blank` claim is now machine-checked by the pixel assertions;
the other claims retain their declared tiers until their checks are promoted.

## Work ladder

- Morning walk repair measured 2026-09-05: W3. Goal `command-center-morning-walk` requires bounded phase-aligned evidence; OT-P0-009 requires changes against a named prior snapshot. Business and requirements gates pass. Prep v3 now compares full handoff content; all 15 behavioral cases pass. Test Genie `20260905-042040-2eb1130a` passes programs and skill-set but reports API test execution failure. A follow-up Unit Health read selects port 2026 despite the lifecycle reporting 15317; an explicit API base also returns a downstream port-2026 error. The API failure cause is unverified; no broader green claim.
- Rung: W3
- Evidence: W0 goal/PRD comparison now agrees after reconciling the stale `command-center-dashboards` and `command-center-foundation` goal/milestone descriptions on 2026-09-03; `business-health validate scenario command-center` and `vrooli scenario requirements validate command-center` both pass with no findings; focused Command Center contracts and race tests pass; the rebuilt live dashboard reports Vrooli `active_scenarios` as `VALID` with a producer-owned timestamp.
- Remaining: full-plan baseline coverage is still partial. Plan Manager requires a re-anchor because source edits invalidated generation 7; the latest collection has 2 ready and 2 failed required members. Test Genie source-identity admission was repaired, but the collection retains in-progress-run conflicts and provider deadline failures. Broad inherited suite findings remain separately logged. Targeted Command Center Go/UI and LPBS validation pass, including the complete LPBS API package suite after repairing stale fixtures, endpoint metadata, approved checksum data, monetization expectations, and admin-session bearer fallback. The targeted Command Center Test Genie experience phase also passes, including browser provenance assertions. Live dashboard evidence is passing. Revenue semantics were corrected to exclude `past_due` subscriptions per Decision 8, and all required LPBS credentials plus the LPBS/cloud deployment descriptors are now declared for tier-4 SaaS. A new re-anchor was intentionally not launched because it would repeat the saturated heavy workload and would no longer be a pre-work baseline.
- Measured: 2026-09-03

## Component library defects found while designing

## Work ladder

- Rung: W3
- Evidence: the active Traffic Instrument plan requires beats, panel readings, live scene data, and new compositions; Command Center API and TypeScript checks pass, but browser behavior and crossfade validation remain incomplete.
- Blocker: none; continue implementation and targeted validation.
- Measured: 2026-09-03

Reported from source reading on 2026-09-01, not from running the components. File against `react-component-library`.

- **`CartesianCharts` ignores its `kind` prop.** It accepts `line | area | bar | stacked-bar | scatter | histogram` and interpolates the value into the description string only; every kind renders a line chart.
- **`NetworkGraph` has two defects.** It sets `context.strokeStyle = "var(--color-border)"` on a canvas 2D context, which cannot resolve CSS variables, so edges draw in the default black; and each node `<button>` contains only a visually-hidden span, so the keyboard list renders as empty buttons.
- **`Meter` is vacuous.** It renders an empty `<section>`. `BoundedMeter` is the real implementation.
- **`CommandCenterShell` is misnamed.** It is a sidebar-nav operational console — the shell `DESIGN.md` lists under "Don't" for command displays. The name will keep drawing people here to the wrong component.

---

## Cross-references

- [DECISIONS.md](DECISIONS.md) — why the design is what it is
- [PROGRESS.md](PROGRESS.md) — what changed and when
- [../concepts/ARCHITECTURE.md](../concepts/ARCHITECTURE.md) — the target the divergences are measured against

## Morning walk work ladder (2026-09-04)

- W0: operator requested scenario-owned morning walk preparation and continuity; existing OT-P0-008 promised board reads but no walk workflow. Added OT-P0-009 under the authorized `command-center-morning-walk` goal.
- W1: adding linked preparation, ownership, continuity and evidence obligations before implementation.
- Scope: morning walk capability; existing display and provider gaps remain separately recorded above.


### Morning walk validation and remaining evidence (2026-09-04)

- W0/W1: authorized target OT-P0-009 and requirement CC-P0-015; business-health
  and requirement reference validation passed. Goal: `command-center-morning-walk`.
- W2/W3: Test Genie `20260904-231540-4a758c06` passed `programs` and `skill-set`.
  Ten behavioral fixtures cover phases, real zero, isolated/all outages, invalid
  inputs, checkpoint continuity, stale evidence, a false legacy heading, and output
  overflow. Thirty-one kernel handle tests pass, including serialized nested envelopes.
  Changed Command Center, Source Ledger and program-runtime Go packages pass.
- Live prep has twelve phases, eleven readable sources, two missing handoffs,
  two stale/undated source sections, and no active checkpoint. These gaps are
  evidence conditions, not zeros. A legacy prose mention of a checkpoint was
  initially misread; anchored heading extraction and a regression case repair it.
- Skill validation: registry ownership/discovery and declared program gates pass;
  normal, partial, invalid-input, continuity and overflow branches were exercised.
  No independent agent trial or operator-usefulness improvement is claimed.
- Broader suites are not green: Command Center unit run
  `20260904-231350-8aacbaba` reports UI coverage command failure; Source Ledger
  `20260904-225914-6955cdee` and program-runtime `20260904-231351-baa86e12`
  report missing UI discovery and additional execution findings. Source Ledger
  also reports generated replay companion duplication. Prior baselines/causes
  were not established. Scenario QA receipt: `knw-1788563800147690577`.
- Pending capability: `idea/prompt-manager-governed-fleet-health-read`. Until the
  owner exposes its attribution algorithm, the prep skill uses one exact CLI read.
- Pending measurement: `idea/command-center-walk-usefulness-measurement`.
  Learning scope `command-center-usage` exists; no actual-walk baseline or
  causal benefit is claimed. Preserve real feedback before establishing targets.

### Morning walk next sequence (2026-09-04)

W0: user selected complete-path validation, better evidence selection/change tracking, and owner publication/checkpoint operations. OT-P0-009 now covers those explicit writes; the prior prohibition on all upstream mutations has an owned-ledger exception. Other goals preserve display and outcome boundaries. Actual operator conversation benefit remains unmeasured; test rehearsals cannot supply it.


### Selected sequence implementation and evidence (2026-09-04)

- Preparation v2: oldest pending selection, latest completed selection, stable
  source identities, and explicit no/invalid/unavailable baseline states.
  Comparisons report newly observed, changed, refreshed, and not observed;
  capped absence never implies resolution. Meta-instrument gap ids are retained.
- Owner operations: State reads briefing and checkpoint independently; Publish
  validates envelope phase order, freshness, channel and fleet evidence;
  Checkpoint validates phase/state/identity and refuses finished-walk resurrection.
  Both use Source Ledger request-key replay and transactional predecessor checks,
  then verify durable receipts. Exact late retries return their original receipt.
  Source Ledger remains the storage authority; no local checkpoint database exists.
- Test channel separates rehearsals from operator records. Live program `prog_c072b481-e766-41a3-baef-25e86d45f22a`
  succeeded with partial source evidence and compared a named saved briefing.
  Latest rehearsal publication receipt: `9b8f2c06-1c7b-4d2a-a12a-99cbf078ea52`; completed checkpoint receipt:
  `5ad9bffb-de31-49ca-9705-15f78a182dc9`. Rehearsal checks covered duplicate publish, exact content readback,
  completion and delayed active-event retry. No operator conversation occurred.
- Learning reader is adapted from BAS. Test capture receipt
  `4a24a00b-387b-44a3-bf95-211b862db340` is excluded from eligible operator attempts:
  the live reader reports `unreliable:no_eligible_attempts`. Rehearsal timing covers
  the observed publication/checkpoint calls only; it is not total preparation
  effort or an improvement estimate. Recall advice efficacy was not exercised.
- Thirteen behavioral cases pass, plus owner operation and Source Ledger
  conditional/replay/concurrent-writer tests. Command Center CLI packages pass.
  Test Genie `20260904-234322-50487e65` programs and skill-set pass.
  Broad unit runs `20260904-233918-d0fcec2c` (Command Center) and
  `20260904-234220-8e83689d` (Source Ledger) still fail: CC UI coverage command;
  Source Ledger discovery, generated companion and other execution findings.
  These remain covered by QA `knw-1788563800147690577`; no gates were weakened.
- API transitive protobuf validation dependency was installed through the
  Dependency Analyzer gateway; approved-dependency validation passes with an
  existing local-module range advisory. Analyzer scans do not currently infer
  the typed Source Ledger client, so the explicit optional lifecycle declaration
  remains necessary. It is required for walk writes, not board availability.
- Owner validation establishes structure and continuity, not independent truth
  of an arbitrary caller-supplied program id or prose interpretation. The skill
  retains source verification and operator authority. Direct generic ledger writes
  can bypass domain transition rules; normal consumers now use the owner commands.
- Next empirical step: conduct an actual operator walk and record missing context,
  corrections, extra reads, duration and stated usefulness in comparable contexts.
  Do not count these implementation rehearsals as evidence of operator benefit.
