# Coherence Audit - 2026-02-07

## State
- Current pattern: local-first + React Query for server state.
- App-wide stores: none (`create<...>` usage not found in `ui/src`).
- State hotspots:
  - `ui/src/surfaces/explorer/components/HealthPanel.tsx` (458 lines, 5 local `useState` values, healing workflow orchestration + rendering combined).
  - `ui/src/surfaces/viewer/components/ResetPanel.tsx` and `ui/src/surfaces/viewer/components/PreviewView.tsx` have light local state and appear bounded.

## Duplication
- Duplicate components: no duplicate base primitives found; `Button` is reused broadly from `ui/src/shared/ui/button.tsx`.
- Duplicate hooks/services: no obvious duplicate fetch/auth/form hooks from regex audit.
- Consolidation candidates:
  - Extract healing-flow state transitions from `HealthPanel` into a feature-local hook/controller (keep presentational component smaller).
  - Standardize any modal keyboard handling behind one shared helper if more modal surfaces add key listeners.

## Styling System
- Token coverage: partial-to-good.
  - Positive: global `ko-*` token variables are established in `ui/src/styles.css` and used across component classes.
  - Gap: no `ui/src/shared/theme/` ownership boundary yet (tokens currently live in a single root stylesheet).
- Primitive variant coverage: partial.
  - Positive: `Button` uses `cva` semantic variants (`primary`, `secondary`, `outline`, `danger`).
  - Gap: primitives are not yet organized under `shared/ui/primitives/` and other primitives (`Card`, `Input`, `Badge`) remain class-contract based in CSS.
- Surface-level style debt:
  - Significant styling contract still centralized in `styles.css` with many utility clusters; migration to explicit primitive/composite ownership remains incomplete.

## Theme Refresh Readiness
- Status: needs foundation work before a broad refresh.
- Required prerequisites:
  1. Establish `shared/theme` token ownership and move token definitions out of monolithic root styling.
  2. Define/organize primitives under `shared/ui/primitives` (at minimum: Button/Card/Input/Badge contracts).
  3. Reduce `HealthPanel` complexity to separate view composition from healing side-effect orchestration.

## Architecture Alignment
- `shared/ui` exists and is used, but `shared/theme` is absent.
- Surfaces mostly assemble shared building blocks; no major evidence of per-surface primitive reinvention.
- Service/controller boundaries are present (`shared/services`, `shared/controllers`) and used by hooks.

## Priority Actions
1. Introduce `shared/theme` ownership (`tokens.css` + exports) and keep `styles.css` as a temporary compatibility layer.
2. Move `Button` to `shared/ui/primitives` with a compatibility re-export path, then follow with `Input`/`Card` primitives.
3. Split `HealthPanel` into:
   - `useHealthHealingFlow` (state + actions)
   - `HealthPanelView` (pure render)
4. Add a shared modal keyboard helper if more than one modal needs `Escape` handling to avoid listener drift.

## UI Migration Loop – 2026-02-07

### What moved in this loop
- Replaced KO token palette from neon-green hacker aesthetic to a cleaner blue/slate system.
- Restyled shell and primitives (`ko-app-shell`, `ko-app-header`, `ko-panel`, `ko-card`, `ko-tab`, buttons, inputs, pills, badges, focus, scrollbars) to modern control-tower styling.
- Kept selector/test contracts stable by retaining existing class/test-id usage and surface composition.
- Updated header copy to match the new product framing.

### Legacy vs new contract notes
- Legacy contract retained temporarily: `ko-*` class namespace remains in use to avoid selector churn.
- New styling direction is now token-first inside `styles.css` and significantly less tied to hardcoded brand tone.
- Remaining debt: dedicated `shared/theme` file ownership is still not established; primitives still live in mixed locations (`shared/ui/button.tsx` + class contracts).

### Convergence scorecard
- Surface files with raw palette values: 0
- Legacy class contract references in migrated surfaces: non-zero (`ko-*` compatibility retained intentionally)
- Core primitives migrated to semantic variants: partial (`Button` only)
- High-traffic surfaces migrated: 100% visual refresh via shared classes/tokens
- Deprecated contract backlog with no owner: 0 (tracked here)
