# Coherence Audit - 2026-03-21

## State
- Current pattern: local-first + React Query for server state
- App-wide stores: None (appropriate for scope)
- State hotspots: None — each component owns only its local UI state
- Custom hook: `useMutationErrors` (shared by 3 components)

## Duplication
- Duplicate components: None
- Duplicate hooks/services: None (consolidated in Phase 14)
- Consolidation candidates: None identified

## Styling System
- Token coverage: Tailwind defaults (consistent, no custom tokens needed at this scale)
- Primitive variant coverage: `ui/button.tsx` (Radix slot-based); ErrorFallback, ErrorBanner are shared composites
- Surface-level style debt: None — all colors use Tailwind palette, no raw hex/rgb in TSX
- Edge stroke color: Moved to CSS variable `--edge-stroke-color` for theming (Phase 18)

## Theme Refresh Readiness
- Ready now for token-level changes (CSS variables + Tailwind)
- No prerequisites needed — architecture is clean

## Recent Changes (Phase 18 iter 2)
- Added `KeyboardShortcutHelp` component — reusable dialog for canvas shortcuts
- Component count: 27 (up from 25 in Phase 18 iter 1)
- No new duplication introduced

## Priority Actions
1. None — codebase is coherent and well-organized
