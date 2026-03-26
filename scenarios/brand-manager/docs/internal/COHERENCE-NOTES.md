# Coherence Audit - 2026-03-26

## State
- Current pattern: local-first (useState in pages, React Query for server state)
- App-wide stores: none (no zustand/context stores)
- State hotspots: none identified — state scope matches usage well

## Duplication
- Duplicate components: none — each component serves a distinct purpose
- Duplicate hooks: none — useRouter is the only custom hook
- Consolidation candidates: none at current size

## Styling System
- Token coverage: **good** — CSS custom properties defined in styles.css for text, surface, border, feedback, radius, motion
- Token integration: **improved** — tokens now wired into tailwind.config.ts as `brand-*` color utilities
- Primitive variant coverage: **partial** — CVA used for Button and Input only; Section and ErrorAlert use cn() with static strings
- Surface-level style debt: some raw slate-* classes used in pages; could migrate to brand-text-* tokens over time

## Theme Refresh Readiness
- Ready for incremental migration
- Required prerequisites:
  - Migrate page-level slate-* classes to brand-* token utilities (non-breaking, incremental)
  - Add CVA variants to Section component (low priority)

## Architecture Alignment
- Visual contracts partially owned in `styles.css` (tokens) + `components/ui/` (primitives)
- Surfaces assemble from primitives correctly — no primitive re-invention in pages
- Controllers/services: API layer in `lib/api.ts` is centralized and typed

## Priority Actions
1. [Done] Wire CSS tokens into Tailwind config as brand-* utilities
2. [Done] Add active nav state for navigation coherence
3. [Future] Migrate remaining raw palette classes to token utilities
4. [Future] Add CVA variants to Section and ErrorAlert
