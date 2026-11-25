# Performance — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](TESTING.md#behavior-claim-register).

This document records performance budgets, current measurements, known
constraints, and regression procedures.


Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Lighthouse Home performance | 0.61 (error threshold 0.75) | Test Genie run `20261122-124253-457fa6e9`; the retained phase result/log contain no Lighthouse report or owner breakdown | 2026-11-22 |
| Existing Home startup asset closure (compressed) | 320,089 bytes gzip across entry, App, shared/error-boundary chunks and CSS | `ui/dist` assets stamped 09:49 America/New_York (14:49 UTC), after the 12:48 UTC Lighthouse phase; no source hash or run binding is available, so this is not a comparable before/after measure | 2026-11-22 |

The Test Genie performance artifact identifies only the failing page and score.
It does not retain the Lighthouse report, metric-level diagnostics, or a source
owner breakdown. The separate performance-health provider was unavailable
because no browser UI surface was resolved. The existing `ui/dist` measurement
is therefore useful only as a bounded asset-size observation; it does not
establish the cause of the 0.61 score or support a source-level fix. Its build
timestamp is later than the Lighthouse run and it is not bound to a source
revision, so it cannot serve as a before/after comparison.

### Source editor bundle audit

The existing production artifact separates the source editor from page and
component navigation. `SourceWorkbenchEditor` imports `CodeEditor` through
`React.lazy`; `CodeEditor` alone imports the Monaco bootstrap. The bootstrap
imports Monaco's editor API, the TypeScript language contribution, one custom
Nocturne theme, and two worker constructors. Vite emits worker scripts as
separate assets, and the bundled constructors create `Worker` instances only
when Monaco requests a language worker: JavaScript/TypeScript use the TypeScript
worker, while other labels use the editor worker.

| Editor asset | Uncompressed bytes | gzip bytes | Role |
|---|---:|---:|---|
| `CodeEditor-jtv3QZfD.js` | 2,569,560 | 667,460 | Monaco editor API, TypeScript contribution, custom theme and editor integration |
| `CodeEditor-D3a1iDRN.css` | 74,382 | 11,956 | Monaco editor styles |
| `ts.worker-Blsa1YQj.js` | 6,042,939 | 1,371,688 | TypeScript/JavaScript syntax service and compiler runtime |
| `editor.worker-b0HUMjXM.js` | 251,830 | 75,949 | Core editor worker |

The worker files are large but are not part of the page's initial module graph;
the editor chunk itself also stays behind the source-editing boundary. The
source configures TypeScript and JavaScript syntax diagnostics, maps CSS and
JSON extensions to Monaco language IDs, and uses completion/suggestion UI,
formatting, and a custom theme. Only the TypeScript language contribution is
explicitly loaded; dedicated CSS/JSON language contributions are not in this
closure, so their syntax support is not established by the source. No
additional theme packs or language services are imported. Disabling the
TypeScript contribution or worker would remove configured TS/JS diagnostics,
and splitting Monaco's editor API into hand-selected contributions has no
measured safe cut yet. The artifact therefore exposes no proven low-risk size
reduction. These asset measurements do not explain or attribute the Lighthouse
score above.

## Workbench redesign budgets (planned)

These are targets, not current measurements:

| Surface | Target | Why |
|---|---|---|
| Fixture bench first paint | Show the anatomy stage before secondary frames; no per-frame React state update while measuring frames | The bench mounts real harnesses, so measurement must not turn into a whole-page rerender on every pointer move |
| Home / Components first paint | No synthetic blocking work; real projections stream or show explicit loading | The current catalog already exposes pending previews |
| Drawer open/close | 120–200 ms transition, no layout thrash | The drawer is a frequent evidence surface |
| Design artboard resize | Stable geometry during pointer move; no per-frame full page rerender | Resize must not drop the selected component or reset scroll |
| Coverage / Review queues | Bounded render window; no unbounded DOM for findings | The queue can grow beyond the sample |

If a target cannot be met, record the measured cost and reduce the surface rather than hiding a spinner over a broken interaction. See [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md).

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
