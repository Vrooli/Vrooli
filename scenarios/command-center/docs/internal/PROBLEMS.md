# Problems

Known defects and divergences, newest first. This file is the honest record of where the code differs from the documents around it.

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
| Ledger has no live revenue readings; monetization exposes no revenue or subscription surface yet. | monetization | [SOURCE-MAP.md](../concepts/SOURCE-MAP.md), `CC-P2-003` |
| Broadcast has no readable source; `marketing-crew` declares no aggregator. | marketing-crew | [SOURCE-MAP.md](../concepts/SOURCE-MAP.md), `CC-P2-004` |
| Governed local-package installation currently emits a registry install command for the approved `@vrooli/react-component-library` file record and fails with npm 404; the manifest and lock entry are retained from the approved record. | scenario-dependency-analyzer | Phase 9 |
| `AmbientDisplayShell` and `CycleController` remain scenario-local (`ui/src/components/AmbientShell.tsx`, `BoardController.tsx`): they bind to react-router and the board API, and the library's ingest has no seam for host-provided routing yet. The four primitives beneath them are library assets. | command-center / react-component-library | Decision 10 |
| The contrast floor was measured under software WebGL (SwiftShader) at 1600×1000, ten frames per room, digit rows only. Every room clears 14:1. A GPU run and a portrait run are still owed. | command-center | Phase 12 |
| Test Genie’s DOM exporter does not provide the `data-render-ms` artifact, so frame-budget evidence records deterministic tier draw budgets and the runtime measurement seam but not numeric browser timings. | test-genie / BAS | Phase 12 |

---

## Blocked on other teams

Neither is scenario work. Both are tracked as source bindings in [SOURCE-MAP.md](../concepts/SOURCE-MAP.md).

- **Ledger has no live revenue readings.** The monetization instrument is `live` but exposes no revenue or subscription surface. Coverage is `IN-REACH`, not `MISSING` — the substrate exists. (`CC-P2-003`)
- **Broadcast has no readable source at all.** `marketing-crew` declares no aggregator, and the social-scheduling capability "has no scenario at all — that capability is unowned, not merely unaggregated." This is one finding with one owner, not six pipeline gaps. (`CC-P2-004`)

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
