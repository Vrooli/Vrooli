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
| `development-toolchain-validator.app.ui` | App UI navigation | `path:ui/flow/navigation.json` | Level 4 (behavioral conformance) | passes | passes | 2026-05-18 |

### Unmodeled Surfaces

| Surface | Why it should be modeled | Current risk | Recommended next step |
|---|---|---|---|
| Skills index/detail | Deferred to Phase 5 of the UI revamp; placeholder route ships in the spec but the page is a "coming soon" stub | Low — destination is reachable and labeled honestly; users won't get a 404 | Land the skill-catalog Connect-RPC client; replace `path:ui/src/surfaces/placeholders/SkillsPlaceholder.tsx` with the real surface; add reachability invariants if Skills gates anything |
| Manifests index/editor | Same as Skills (Phase 6 deferred) | Low | Same as Skills, but for `path:ui/src/surfaces/placeholders/ManifestsPlaceholder.tsx` |
| Tuple drill-down (`/goldens/:slug/:tupleKind/:subjectId`) | Phase 4 of the UI revamp; planned but not yet routed | Low — the drill-down would render run summary + diff + manifest tabs that depend on verdicts/manifests APIs | Land those APIs, add the route, add a `golden_row_cell_open` affordance to the spec |

### Accepted Exceptions

| Finding | Why accepted | Cleanup trigger |
|---|---|---|
| Convergence chip on `TopHeader` renders a neutral `—` placeholder instead of the real convergence ratio | Verdicts API not shipped — placeholder is honest about absence; renders the chip area so the surface doesn't reshape when the API lands | Replace with real query once `verdicts.proto` ships convergence summary RPC |
| Stale-flag chip on `TopHeader` renders static `0` placeholder | Stale-flag API not shipped — same rationale | Replace with real query once stale-flag API lands |
| `app-monitor` proxy basename uses `BrowserRouter basename="."` | Required to make the same build work locally, behind the app-monitor proxy, and via Cloudflare tunnel | Revisit if app-monitor's URL contract changes |

### Audit Notes

- **2026-05-18 — DTV UI Revamp (Claude):** Replaced the placeholder centered-card UI with a real layered architecture: `path:src/shared/{theme,ui/primitives,ui/composites,components,hooks,stores,lib}` + `path:src/surfaces/`. Authored `ui/flow/navigation.json` declaring 5 routes (goldens index/detail, skills index, manifests index, settings), 3 containers (sidebar, top header, mobile bottom nav), 6 affordances (4 global nav, 1 row-open, 1 detail-back), 4 keyboard shortcuts (`g g`/`g s`/`g m`/`g .`), and 3 reachability invariants. `flow-verifier flows validate && reconcile && verify run` all pass. Component-tier behavioral conformance test (`path:src/__tests__/navigation/affordances.test.tsx`) asserts every nav affordance renders with declared label and resolves to declared destination, viewport-keyed for sidebar vs mobile bottom nav. Phases 4–6 (tuple detail, skills, manifests surfaces) deferred until their backend Connect-RPC clients land; placeholders ship for skills + manifests so the navigation graph stays whole.

## Other Experience Findings

(none recorded yet — add sections here as audit lenses fire)
