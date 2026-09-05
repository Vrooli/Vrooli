# UI Spec Reconciliation

The React Component Library treats the catalog and scenario experience documents as desired intent, and source plus adoption provenance as observed behavior. Asset reconciliation compares catalog declarations with versioned library source. Page reconciliation applies the same rule one level higher: it compares `scenarios/<scenario>/experience/pages/<page>.json` with the scenario UI files that implement its regions.

The authored page sketch belongs in the scenario's experience document. The library may read and update `page.sketch`, but it stores only derived observations such as scan provenance, region verdicts, and coverage history in its own database. Viewport and grid coordinates remain non-normative drawing hints. The region-to-implementation binding is normative and is what verification evaluates.

## Region verdicts

The verdict vocabulary is closed.

| Verdict | Meaning | Advisory blocking rule |
| --- | --- | --- |
| `matches` | The resolved file is an unmodified adoption of the placed built asset version. | Never |
| `drifted` | The resolved file is an adopted asset whose bytes differ from its recorded version. | Report now; eligible for later promotion |
| `missing` | A declared region has no resolvable implementation file. | Report now; required regions are eligible for later promotion |
| `extra` | A scanned file in a declared UI slot resolves to no page region. | Never |
| `unverifiable` | No declared manifest exists, no join rule resolves, or the placement is a placeholder. | Never |

Every region result records its authored owner and the evidence used to derive the verdict. Verification scans a scenario once and reuses that provenance for all regions.

## Three-tier decomposition

Page design exhausts reuse before invention:

1. Replace a local component with a compatible, implemented catalog asset.
2. Build or adopt a compatible catalog declaration or page-template region that is already designed but not implemented.
3. Record a genuinely new need as an intentional placeholder only after tiers 1 and 2 both ran and returned no match.

Raw local markup is not a fourth tier. A brief orders tier 1 before tier 2 before tier 3, preserves sketch notes as constraints, and names `sketch verify` as its closing gate.

## Region-to-file join

The resolver evaluates these rules in order and stops on the first hit:

1. Resolve a region's `bindings.elements` test ID through the scenario selector registry. This is proven evidence.
2. Resolve a placed catalog asset through the scenario's adoption record and adopted path. This is proven evidence.
3. Slug-match the region ID to a component file inside a slot declared by the template UI manifest. This is heuristic evidence and is labeled `proven: false`.

When none resolves, verification reports `unverifiable` with a reason. Provenance tagged as `UNKNOWN` is kept distinct from scenario-local `CUSTOM` source because the remediation differs.

## Coverage

Coverage reports one count for each supply state:

- `built`: an implemented catalog asset supplies the placement.
- `declared`: a catalog declaration or template region exists but lacks implementation.
- `invented`: the sketch intentionally carries a placeholder.

The workbench renders real catalog components where implementations exist and explicit wireframes elsewhere. Each region shows its rendering mode, verdict, join evidence, and authored note.

## Baseline 2026-09-03

The plan was authored on 2026-09-03. The execution re-measured the live shared tree on 2026-09-04 before plan-owned feature edits; these values are the effective before-state. The immutable collection was partial because unrelated source changed during capture and the React Component Library comprehensive run exceeded its deadline, so the measurements below are retained independently and final validation compares against them directly.

| Measure | Baseline value |
| --- | ---: |
| I19 — declared assets without implementation | 222 assets |
| I21 — adoption depth | 4.412621763383565% |
| I27 — machinery-to-asset line ratio | 0.947600477516912 |
| Go machinery lines | 57,925 |
| UI lines | 17,566 |
| Asset lines | 62,977 |
| Experience page documents | 209 |
| Documents with regions | 41 |
| Documents with bindings | 143 |
| Fleet library region references | 64 |
| Fleet local region references | 17 |

The visual before-state is captured at 1440×900 in `plan-artifacts/react-component-library-design-from-the-library/screenshots/before/` for the catalog, catalog coverage, and AmbientCanvas asset-detail surfaces. The coverage capture also records that the live coverage endpoint was unavailable before implementation.
