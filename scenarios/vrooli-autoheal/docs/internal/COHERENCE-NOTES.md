# Coherence Audit - 2026-02-17

## State
- Current pattern: local-first with React Query for server state.
- App-wide stores: none.
- State hotspots:
  - `ui/src/App.tsx` holds cross-surface UI state (tabs, auto-refresh, modal selection, collapsed groups).
  - `ui/src/shared/components/SettingsDialog.tsx` has high local state density for config editing flow.
  - `ui/src/shared/components/CheckDetailModal.tsx` has high local state density for action execution + history display.

## Duplication
- Duplicate components:
  - Repeated tab trigger class clusters in `ui/src/App.tsx`, `ui/src/shared/components/SettingsDialog.tsx`, and `ui/src/shared/components/CheckDetailModal.tsx`.
  - Repeated Escape-key listener lifecycle code in `ui/src/shared/components/SettingsDialog.tsx` and `ui/src/shared/components/CheckDetailModal.tsx`.
  - Repeated custom toggle markup for enabled/auto-heal/critical states across settings and modal flows.
- Duplicate hooks/services:
  - No duplicate service-layer fetch helpers found.
- Consolidation completed:
  - Added shared `TabTrigger` composite (`ui/src/shared/ui/composites/tab-trigger.tsx`) and migrated all three call sites.
  - Added shared `useEscapeKey` hook (`ui/src/shared/hooks/useEscapeKey.ts`) and migrated modal/dialog escape handling.
  - Added shared `Switch` primitive (`ui/src/shared/ui/primitives/switch.tsx`) and migrated toggle controls in `SettingsDialog` and `CheckDetailModal`.

## Styling System
- Token coverage: partial-good.
  - Global semantic tokens exist in `ui/src/shared/theme/tokens.css`.
  - Core primitives use variants (`Button`, `Card`, `Badge`, `Switch`).
- Primitive variant coverage: improving.
  - Tabs were consolidated into shared composite.
  - Toggle controls now use shared variant primitive.
- Surface-level style debt:
  - `ui/src/shared/components/CheckDetailModal.tsx` and `ui/src/shared/components/SettingsDialog.tsx` were migrated off direct palette utility classes to semantic token classes for primary status/shell styles.
  - Remaining debt is mostly in other surfaces/components that still rely on non-semantic utility clusters.

## Theme Refresh Readiness
- Foundation is improved; still needs additional extraction before full refresh.
- Required prerequisites:
  1. Continue extracting repeated status/info panels into shared composites.
  2. Expand semantic token coverage for additional nuanced states (currently represented via alpha-tinted accent usage).
  3. Audit remaining surfaces for non-semantic utility patterns and migrate incrementally.

## Priority Actions
1. Extract reusable semantic panel/notice composites for repeated status/info block patterns.
2. Continue migrating remaining surfaces to semantic token classes and shared primitives.
3. Consider splitting `ui/src/App.tsx` into an app-shell surface + per-tab shell component to reduce cross-surface orchestration density.
