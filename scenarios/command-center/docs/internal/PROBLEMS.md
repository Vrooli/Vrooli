# Problems

Known defects and divergences, newest first. This file is the honest record of where the code differs from the documents around it.

**Standing note (2026-09-01):** the documentation in this scenario was rewritten to the instrument design before any code changed. Everything under "Divergence from the design contract" is expected and dated, not a regression. The code still implements the 2026-04 read-only kiosk aggregator.

---

## Defects in this scenario

### P1 — Five of six scenes mount and draw nothing

**Observed** 2026-09-01 against a running instance, headless Chrome with software WebGL at 1600×1000.

The canvas region on `/hive` contains exactly two colours: 587,159 pixels of `rgb(2,16,13)` — the theme's own background — and one stray pixel. The DOM confirms `<canvas data-engine="three.js r170" width="1174" height="510">` mounted, no Suspense fallback showing, and the scene chunk built and served. Unchanged at a 20-second budget. The same is true of `/forge`, `/ledger`, `/broadcast` and `/panorama`, which share the placeholder scene.

`/mission-control` renders correctly under identical conditions. Everything visible in it is emissive or unlit; the placeholder scene uses a lit material with an ambient and a directional light. That points at the lighting path, but **the cause was not proven.**

**Not yet confirmed on a GPU.** Root-cause this before building any new scene (`CC-P1-003`, and the 2026-09-01 decision). If the cause lives in the canvas wrapper or the renderer version rather than in the placeholder scene, it follows into all six new rooms.

### P1 — The test suite would pass all five blank rooms again

The BAS cases for scene rendering wait for the wrapper to be visible and assert that a `<canvas>` exists inside it, then screenshot. An empty canvas satisfies every assertion. The suite proves the component mounted, which was never in doubt.

Fixed by pixel-nonblank assertions (`CC-P1-003`), which must land before the rooms rather than after.

### P2 — Kiosk hooks exist and are wired to nothing

`useKeyboardShortcuts`, `useSpatialNav`, `useFullscreen`, `useWakeLock` and `useGamepad` are written and unit-tested. Grep across `ui/src/pages`, `ui/src/components`, `App.tsx` and `main.tsx` finds no caller. There is no navigation of any kind: routes are reachable only by typing a URL, and the keyboard map for rooms 1–6 is defined but never registered.

Superseded by `CC-P1-009` and the shared `GamepadAction` vocabulary; the scenario-local hooks are replaced rather than wired.

### P2 — The scenario runs on ungoverned dependencies

`three`, `@react-three/fiber`, `@react-three/drei`, `@types/three` and `framer-motion` are declared in `ui/package.json` and none appears in `.vrooli/dependencies/approved-dependencies.json` in any state. Three scenarios use three.js at all.

This must go through `scenario-dependency-analyzer` before any dependency change. The intended end state: govern the existing three stack, add selective postprocessing behind the capability probe, and drop `framer-motion` and `recharts` — the first is covered by the library's motion primitives and the second charts nothing this design draws.

---

## Divergence from the design contract

Dated 2026-09-01. Each is expected until the corresponding requirement lands.

| Divergence | Contract | Requirement |
|---|---|---|
| The payload has no value field; `MetricEntry` carries six fields and no reading | [DATA.md](../concepts/DATA.md) | `CC-P0-001` |
| One merged `dataSource` field instead of two independent axes | [COVERAGE-MODEL.md](../concepts/COVERAGE-MODEL.md) | `CC-P0-002` |
| No sample values exist; non-live metrics render no figure at all | [PROVENANCE-MODEL.md](../concepts/PROVENANCE-MODEL.md) | `CC-P0-003` |
| No setpoint | [DATA.md](../concepts/DATA.md) § The setpoint | `CC-P0-005` |
| Registry has no schema version, id policy, tombstones or migration path | [OUTCOME-TAXONOMY.md](../concepts/OUTCOME-TAXONOMY.md) | `CC-P0-006` |
| Room list is six literal routes in `App.tsx`; metric list is hand-authored | [OUTCOME-TAXONOMY.md](../concepts/OUTCOME-TAXONOMY.md) | `CC-P0-007` |
| Source-team instrument state is not read at all | [SOURCE-MAP.md](../concepts/SOURCE-MAP.md) | `CC-P0-008` |
| No ranked surface; gaps are six per-room lists | [api-endpoints.md](../reference/api-endpoints.md) | `CC-P0-010` |
| Gaps are undated; nothing ages, nothing counts unregistered outcomes | [COVERAGE-MODEL.md](../concepts/COVERAGE-MODEL.md) | `CC-P0-011` |
| No describe endpoint; CLI exposes only `status`, `configure`, `help`, `version`. The surfaces are now declared in `.vrooli/endpoints.json` and `cli/manifest.json`, marked `planned` | [cli-commands.md](../reference/cli-commands.md) | `CC-P0-012` |
| No empirical axis; the payload carries no prediction verdict | [COVERAGE-MODEL.md](../concepts/COVERAGE-MODEL.md) § Axis 3 | `CC-P0-014` |
| Gap badges render red; the design contract's gap tone is violet and, as amended, material-primary | [PROVENANCE-MODEL.md](../concepts/PROVENANCE-MODEL.md) | `CC-P1-001` |
| Six rooms are one layout in six palettes — explicitly a `DESIGN.md` "Don't" | [UI-ARCHITECTURE.md](../concepts/UI-ARCHITECTURE.md) | `CC-P1-002` |
| No portrait compositions; below 768px the layout stacks but does not compose | [UI-ARCHITECTURE.md](../concepts/UI-ARCHITECTURE.md) | `CC-P1-004` |
| No capability probe; one scene tier, no fallback frame | [UI-ARCHITECTURE.md](../concepts/UI-ARCHITECTURE.md) | `CC-P1-012` |

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
| `experience-manager spec validate command-center` | **FAILED**, 16 findings, all `experience.ref_unresolved`. |

Re-run 2026-09-01 after the review corrections: `business-health` still **PASSED** with 18 targets and 32 requirements; `experience-manager` still reports exactly the same 16 `ref_unresolved` findings and no others, so the retiering and the added `prd_refs` introduced no new errors.

The experience failures are a **fleet-wide pre-existing condition, not a defect in these specs.** Every region that pins the library's `experience-surface` component reports "pins library component ... without a canonical experience contract," at every version. `infrastructure-manager` reports 27 of the same finding and `offer-desk` reports 2; no scenario with an experience contract currently passes. The library holds `ExperienceSurface` only at 1.0.3, while 40 pins across the fleet name 1.0.0 — these specs pin 1.0.3, the version that actually exists.

Closing this needs a canonical experience contract promoted into the component library. That is `react-component-library` work, not scenario work.

Every schema error in these specs was fixed: static regions now declare `states: ["static"]` as the schema requires.

**Claim tiers.** Every claim is `aspirational`, not `manual`. `manual` means human-attested with expiry, and no surface exists to attest against — asserting a review that cannot have happened is the same class of dishonesty this whole design exists to prevent. Each claim carries `x-intended-tier`, and the 30 that are deterministically checkable carry `x-check-plan`, so promotion to `machine` is mechanical once the check is written. `room-never-blank` is promoted first: it is the defect the current build shipped, and it is the reason the existing BAS cases proved nothing.

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
