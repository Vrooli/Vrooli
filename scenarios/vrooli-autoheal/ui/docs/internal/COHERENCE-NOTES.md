# Coherence Audit - 2026-02-18

## State
- Current pattern: local-first with React Query for server state.
- App-wide stores: none (no Zustand/global reducer usage found).
- State hotspots: `src/surfaces/dashboard/components/SystemProtection.tsx`, `src/surfaces/docs/DocsPage.tsx`.

## Duplication
- Duplicate components: no duplicate `Button/Card/Input/Modal` function declarations detected.
- Duplicate hooks/services: no repeated `useForm/useFetch/useAuth/useModal` hook patterns detected.
- Consolidation completed:
  - Added shared primitives: `Button`, `Card`, `Badge`, `Switch`, `Input`, `ModalOverlay`, `ModalContent`.
  - Added shared composite primitives: `Panel`, `PanelHeader`, `PanelTitle`, `Notice`, `NoticeTitle`.
  - Moved cross-surface widgets into `src/shared/components/*` (`ErrorDisplay`, `StatusIcon`, `StatusBadge`, `CheckDetailModal`, `SettingsDialog`, `CodePreview`).
  - Moved dashboard-only widgets into `src/surfaces/dashboard/components/*`.
  - Moved trends-only widgets into `src/surfaces/trends/*` and `src/surfaces/trends/components/*`.
  - Moved docs implementation from legacy `src/pages/Docs/*` into `src/surfaces/docs/*`.
  - Promoted metadata context ownership to `src/shared/contexts/CheckMetadataContext.tsx`.

## Styling System
- Token coverage: improved; docs surfaces now use semantic token classes instead of ad-hoc `slate/white/blue` classes.
- Primitive variant coverage: improved to `Button`, `Card`, `Badge`, `Switch`, `Input`, and modal primitives.
- Composite coverage: introduced a reusable `Notice` contract for semantic info/warning/success/danger callouts.
- Surface-level style debt:
  - Large components still using direct utility clusters: `src/surfaces/dashboard/components/SystemProtection.tsx`, `src/surfaces/dashboard/components/CheckCard.tsx`.
  - Remaining `SystemProtection` debt is now mostly large flow density, not raw palette usage.

## Theme Refresh Readiness
- Status: foundation advanced; ready for staged migration, not for full theme refresh.
- Required prerequisites:
  - Continue migrating remaining callouts to `Notice` where class clusters are still repeated.
  - Continue reducing class-cluster repetition in settings/system-protection surfaces.

## Architecture Alignment
- Completed:
  - Introduced `src/surfaces/dashboard`, `src/surfaces/trends`, and `src/surfaces/docs`.
  - Added surface-local component ownership under `src/surfaces/dashboard/components` and `src/surfaces/trends/components`.
  - Extracted docs ownership to `src/surfaces/docs/DocsPage.tsx` and `src/surfaces/docs/components/*`.
  - Established `src/shared/components` + `src/shared/contexts` as cross-surface ownership layers.
  - Added shared form and modal ownership under `src/shared/ui/primitives/input.tsx` and `src/shared/ui/primitives/modal.tsx`.
  - Migrated modal shell usage in `src/shared/components/SettingsDialog.tsx` and `src/shared/components/CheckDetailModal.tsx` to shared modal primitives.
  - Added shared notice composite ownership to `src/shared/ui/composites/panel.tsx` and migrated high-traffic callouts in `src/shared/components/CheckDetailModal.tsx` and `src/shared/components/SettingsDialog.tsx`.
  - Replaced ad-hoc monitoring text inputs in `src/shared/components/SettingsDialog.tsx` with shared `Input` primitive usage.
  - Split monolithic settings tabs into module ownership under `src/shared/components/settings/*`, with `SettingsDialog` now focused on orchestration/state wiring.
  - Migrated `src/surfaces/dashboard/components/SystemProtection.tsx` off raw `white/slate/emerald/amber/red/blue` class clusters to semantic token classes and shared primitives/composites.
  - Added regression coverage in `src/shared/components/CheckDetailModal.test.tsx` for importance, dangerous-action confirmation, and post-action result notice states.
  - Migrated docs search control in `src/surfaces/docs/components/DocsSidebar.tsx` to `Input`.
  - Consolidated duplicated health-group rendering logic in `src/surfaces/dashboard/DashboardSurface.tsx` into a single mapped section component with shared config.
  - Added progressive-disclosure mobile nav behavior in `src/surfaces/docs/components/DocsSidebar.tsx` to reduce viewport contention on narrow screens.
- Remaining:
  - Continue reducing density in `src/surfaces/dashboard/components/SystemProtection.tsx` via smaller local subcomponents/hooks.
  - Consider extracting monitoring section checkbox control into a shared checkbox primitive to complete primitive consistency.

## Priority Actions
1. Split `SystemProtection` flows (status, install, uninstall) into smaller local units/hooks to reduce file density.
2. Add a shared checkbox primitive and migrate monitoring \"Critical\" control to avoid native-input styling drift.
3. Normalize keyboard shortcut handling through a single app-shell manager if additional global shortcuts are introduced.
