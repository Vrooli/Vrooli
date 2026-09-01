# Problems

Known defects and divergences, newest first. This file is the honest record of where the code differs from the documents around it.

**Standing note (2026-09-01):** the documentation in this scenario was rewritten to the instrument design before implementation. The core reading, read-surface, room, provenance, input, state-mode, and component-adoption divergences are now addressed; remaining rows describe upstream data ownership, evidence provenance, or validator/dependency infrastructure limitations.

---

## Defects in this scenario

### Resolved — Five room scenes now draw deterministic nonblank compositions

**Observed** 2026-09-01 against a running instance, headless Chrome with software WebGL at 1600×1000.

The room surface now draws composition-specific deterministic geometry in `AmbientCanvas`: lattice, current, river, transmitter arcs, and orbiting panorama nodes. The first-frame probe checks a center pixel and writes a still fallback if the canvas is blank.

The BAS scene cases now evaluate actual canvas pixels rather than only checking for a mounted element.

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
