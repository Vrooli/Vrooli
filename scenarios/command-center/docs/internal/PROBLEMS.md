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

- Rung: W3
- Evidence: W0 goal/PRD comparison now agrees after reconciling the stale `command-center-dashboards` and `command-center-foundation` goal/milestone descriptions on 2026-09-03; `business-health validate scenario command-center` and `vrooli scenario requirements validate command-center` both pass with no findings; focused Command Center contracts and race tests pass; the rebuilt live dashboard reports Vrooli `active_scenarios` as `VALID` with a producer-owned timestamp.
- Remaining: full-plan baseline coverage is still partial. Generation 7 has 3 ready, 3 pending, and 4 failed required members; Test Genie source-identity admission was repaired, but the collection retains admission saturation, in-progress-run conflicts, and deadline failures. Broad inherited suite findings remain separately logged. Targeted Command Center and Web Console contract runs pass; focused implementation, generated-contract, and live evidence is passing.
- Measured: 2026-09-03

## Component library defects found while designing

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
