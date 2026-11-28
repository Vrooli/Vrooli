# Component Library Gaps


Record what Research Desk needed from `react-component-library`
and could not link as-is. Every ejection and every missing asset is a debt the
library can pay back only if it is written down here; a silent local fork is
not. Keep this file short and factual, and raise each entry with the library
(`scenarios/react-component-library/docs/reference/`).

## Ejections

Components copied into this scenario instead of linked, with the reason. One row per
ejection.

| Component | Version | Reason | Revisit when |
|---|---|---|---|
| _none yet_ | | | |

## Missing Assets

Surfaces this scenario built locally because no library asset fit. Name the
shape the library would need, not the local file.

| Need | Nearest library asset | Why it did not fit |
|---|---|---|
| An icon in a soft rounded tile, with an optional state dot | `Chip`, `Icon` | No asset owns the 32–48 px tinted tile that carries a line icon and a state dot; local `ui/src/components/daylight.tsx` (`IconTile`). Ingest when a second scenario needs it. |
| A ring meter with the number in the hole | `BoundedMeter`, `ScoreGauge` | The published meters are linear or score-shaped; a conic ring with a quiet track and a semantic arc is the recurring Daylight shape. Local `ui/src/components/daylight.tsx` (`Ring`). Ingest when a second scenario needs it. |
| A seven-day pulse strip (date circles plus rounded bars) | `Chart`, `CartesianCharts` | A chart asset is heavier than the fixed seven-column strip and its `Research writes` caption; local `ui/src/components/daylight.tsx` (`WeekStrip`, `MiniBars`). Ingest when a second scenario needs it. |
| Render research-home Markdown (headings, lists, quotes, code, pipe tables, links into the desk) on a phone | `markdown-renderer` | Needs `react-markdown`, `remark-gfm`, `shiki` and `mermaid` as peers; installing them through `scenario-dependency-analyzer deps install` was refused by disk admission (2026-11-11). Local `ui/src/components/Markdown.tsx` builds React elements, never HTML. Revisit when the install is admitted. |
| A value with its evidence (source, reason, time read) one tap away, or `unknown` with the reason | none | No library asset pairs a fact with provenance. Local `ui/src/components/Fact.tsx`. |
| Confirm an outward action by showing the exact text and where it goes, with an optional required reason | `AlertDialog/2` | Used as the base; the exact-text block and required reason are composed locally in `ui/src/components/Confirm.tsx`. |
| AppShell/2 shell floors: the `menu-tabs` menu button at 44 px, and the sidebar navigation kept inside a landscape phone viewport (844x390) | `AppShell/2` | The menu IconButton renders 40 px and the sidebar nav ends 2 px below a 390 px viewport (experience claims `floor-tap-target-size`, `floor-chrome-pinned`). Patched with three scoped rules at the end of `ui/src/styles.css`; remove them when the library fixes both. |
| A 44 px tap floor at phone width regardless of the pointer the browser reports | `Button/2`, `IconButton/3`, `CopyIconButton/1`, `Checkbox/1` | The library grows buttons to 44 px only under `(pointer: coarse)`, so a headless phone-width capture measures 40 px and fails `floor-tap-target-size`. `ui/src/styles.css` applies the floor at phone width (and pads path and in-text links). `Checkbox/1` and `Switch/1` shrink the native input to 1x1 although the whole label row is the target, so the claim measures 1 px; `ui/src/styles.css` stretches the invisible input over its row. Remove both rules when the library does this. |

## Resolved in the library

| Gap | Resolution |
|---|---|
| A product card without the hairline border, with raw-child padding and a deliberate tint | `Card 1.4.0`: `variant="surface"`, `variant="inset"`, `variant="pastel"` with `tone`, `padding`, `interactive`. |
| A segmented control whose track and selected thumb read in light and dark | `Tabs 1.3.8`: sunken track, raised selected thumb; `Tabs 1.3.7` also fixed per-instance tab/panel ids. |
| `Section` could not take a grid-area class | `page-section 0.1.1` forwards `className`; the scenario now imports it. |
| A versioned asset with a legacy entry filename and a `story.tsx` could not be resolved by the dev/test resolver | The resolver now honours `component.json.entry` before its legacy fallbacks. |

## Cross-References

- `path:scenarios/research-desk/docs/concepts/UI-ARCHITECTURE.md` — where local components live
