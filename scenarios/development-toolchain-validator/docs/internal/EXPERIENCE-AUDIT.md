# Experience Audit — Development Toolchain Validator

This file is the per-scenario index and memory layer for UX, navigation,
and experience findings. Detailed source-of-truth contracts live in their
canonical homes (`ui/flow/navigation.json` for navigation, etc.); this
file points to them and records audit history.

Per `navigation-integrity-audit` §8: detailed routes / containers /
affordances / invariants belong in `ui/flow/navigation.json` and pass
through `flow-verifier`. This doc is the lightweight index above it.

## Navigation

### Surfaces

| Flow ID | Surface | Spec | Maturity | Reconcile | Verify | Last audited |
|---|---|---|---|---|---|---|
| `development-toolchain-validator.app.ui` | App UI navigation | `path:ui/flow/navigation.json` | Level 4 (behavioral conformance) | passes | passes | 2026-05-19 |

### Unmodeled Surfaces

(none — Skills index/detail, Manifests index/editor, and Tuple drill-down all routed and wired 2026-05-19 once their Connect-RPC clients landed)

### Accepted Exceptions

| Finding | Why accepted | Cleanup trigger |
|---|---|---|
| Diff tab on Tuple detail renders the workspace-sandbox `diff_hash` as text instead of the diff content | Diff blob lives in workspace-sandbox; bridge endpoint not yet exposed to the UI. Same rationale as plan §4 Phase 4 — the surface area is real, the data is honest about being a hash | Wire the workspace-sandbox bridge to fetch diff content by hash; replace the placeholder paragraph with the `DiffViewer` composite |
| `app-monitor` proxy basename uses `BrowserRouter basename="."` | Required to make the same build work locally, behind the app-monitor proxy, and via Cloudflare tunnel | Revisit if app-monitor's URL contract changes |

### Audit Notes

- **2026-05-18 — DTV UI Revamp (Claude):** Replaced the placeholder centered-card UI with a real layered architecture: `path:src/shared/{theme,ui/primitives,ui/composites,components,hooks,stores,lib}` + `path:src/surfaces/`. Authored `ui/flow/navigation.json` declaring 5 routes (goldens index/detail, skills index, manifests index, settings), 3 containers (sidebar, top header, mobile bottom nav), 6 affordances (4 global nav, 1 row-open, 1 detail-back), 4 keyboard shortcuts (`g g`/`g s`/`g m`/`g .`), and 3 reachability invariants. `flow-verifier flows validate && reconcile && verify run` all pass. Component-tier behavioral conformance test (`path:src/__tests__/navigation/affordances.test.tsx`) asserts every nav affordance renders with declared label and resolves to declared destination, viewport-keyed for sidebar vs mobile bottom nav. Phases 4–6 (tuple detail, skills, manifests surfaces) deferred until their backend Connect-RPC clients land; placeholders ship for skills + manifests so the navigation graph stays whole.
- **2026-05-19 — DTV UI Phases 3–7 completion (Claude):** Added 6 Connect-RPC UI clients (`path:src/api/{skillCatalog,manifest,validationRun,validationRecord,staleness,report}.ts`). Extended `ui/flow/navigation.json` with `/goldens/:slug/:tupleKind/:subjectId` (tuple detail), `/skills/:id` (skill detail), `/manifests/:skillId/:goldenSlug` (manifest editor) plus 6 new affordances and 3 new return paths; re-ran `flow-verifier flows codegen/validate/reconcile && verify run` — all PASS. Shipped real surfaces: `path:src/surfaces/goldens/TupleDetail.tsx` (Diff/Manifest/History tabs), `path:src/surfaces/skills/{SkillsIndex,SkillDetail}.tsx`, `path:src/surfaces/manifests/{ManifestsIndex,ManifestEditor}.tsx`. Wired per-row verdict-summary chips on `path:src/surfaces/goldens/GoldensIndex.tsx` via `report.getGoldenSummary` (`useQueries` fanout); populated Skills + Tools `VerdictGrid` on `path:src/surfaces/goldens/GoldenDetail.tsx` with click-through to tuple detail. Settings sync + watcher cards now call `skill_catalog.sync` and `staleness.listStale`. TopHeader convergence + stale chips now read real aggregates from `report` and `staleness`. Deleted `path:src/surfaces/placeholders/`. 151 tests pass (was 140).

## Other Experience Findings

(none recorded yet — add sections here as audit lenses fire)
