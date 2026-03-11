# Coherence Audit - 2026-03-11 (Updated)

This document captures the React Coherence audit findings and recommendations for the reference-react-vite scenario.

## State

- **Current pattern**: Server state via React Query + minimal local state
- **App-wide stores**: None
- **State hotspots**: None

### State Architecture Assessment

The current architecture follows the minimal state principle correctly:
- Uses React Query (`@tanstack/react-query`) for all server state (tasks, projects, health)
- Uses `useState` for ephemeral UI state (form inputs, updating IDs tracking)
- No app-wide stores (not needed for current scope)

**State by Surface:**

| Surface | State Type | Implementation |
|---------|-----------|----------------|
| Dashboard | Server state | React Query for tasks/projects/health |
| Tasks | Server state + local | React Query for CRUD, useState for form/updating |
| Projects | Server state + local | React Query for CRUD, useState for form/updating |
| Layout | Server state | React Query for health indicator |

**Recommendation**: Continue following the state location decision table:
1. Local `useState` for component-specific ephemeral state (form inputs, loading states)
2. Feature-local hooks/stores only if state needs sharing within a surface
3. App-wide store only when state must span multiple surfaces

## Duplication

- **Duplicate components**: None detected
- **Duplicate hooks/services**: None detected
- **Consolidation candidates**: None

### Current Component Inventory

| Path | Purpose | Status |
|------|---------|--------|
| `components/Layout.tsx` | App shell with nav and health indicator | Correct location |
| `components/ErrorBoundary.tsx` | Error isolation with recovery UI | Correct location |
| `components/ui/button.tsx` | Primitive button with CVA variants | Correct location |
| `pages/Dashboard.tsx` | Dashboard overview with stats and recent tasks | Correct location |
| `pages/Tasks.tsx` | Task list with CRUD operations | Correct location |
| `pages/Projects.tsx` | Project grid with CRUD operations | Correct location |
| `hooks/useKeyboardShortcuts.ts` | Centralized keyboard handler | Correct location |
| `lib/api.ts` | Full API client with typed functions | Correct location |
| `lib/utils.ts` | Utility functions (cn for className merging) | Correct location |

## Styling System

- **Token coverage**: None (using Tailwind defaults - appropriate for reference)
- **Primitive variant coverage**: Good (Button uses CVA)
- **Surface-level style debt**: Low (clean, consistent patterns)

### Styling Assessment

The current approach:
- Uses Tailwind utility classes directly with consistent patterns
- Button primitive has CVA variants (`default`, `outline`, `sm` size)
- Consistent color palette: slate-950 background, slate-50 text, white/10 borders
- Loading skeletons use `animate-pulse` consistently
- Error states use red-500/red-400 consistently

**Observation**: The styling is coherent and consistent across all surfaces:
- Dashboard: stat cards with white/5 bg, bordered sections
- Tasks: form inputs, task rows with consistent styling
- Projects: card grid with color indicators

## Theme Refresh Readiness

- **Status**: Not required (reference implementation)
- **Readiness**: Foundation in place
- **Required prerequisites**: None

The styling system is coherent:
- CVA is set up for variants
- Consistent utility class patterns
- Height chain uses `h-full` pattern correctly

## Architecture Alignment

### File Organization

Current structure:
```
src/
├── components/
│   ├── Layout.tsx           # App shell with nav/health
│   ├── ErrorBoundary.tsx    # Shared error boundary
│   └── ui/
│       └── button.tsx       # Primitive with CVA variants
├── pages/
│   ├── Dashboard.tsx        # Dashboard with stats
│   ├── Tasks.tsx            # Task management
│   └── Projects.tsx         # Project management
├── hooks/
│   └── useKeyboardShortcuts.ts  # Central shortcut handler
├── lib/
│   ├── api.ts               # Full API client (slot [F])
│   └── utils.ts             # Utility functions
├── consts/
│   └── selectors.ts         # Test selectors registry
├── test-utils/              # Test infrastructure
├── App.tsx                  # Router setup (slot [E])
├── main.tsx                 # Entry point (slot [D])
└── styles.css               # Global styles
```

**Assessment**: The structure follows design-system ready patterns:
- `components/` contains shared widgets (Layout, ErrorBoundary)
- `components/ui/` contains primitives (Button)
- `pages/` contains route-level views (Dashboard, Tasks, Projects)
- `hooks/` contains domain-agnostic hooks
- `lib/` contains services and utilities

**Expansion recommendations**:
1. Add `shared/theme/` if custom theming is needed
2. Add more primitives to `components/ui/` (Card, Input, Badge) following CVA pattern
3. Consider `surfaces/` structure if pages become complex with subcomponents

## Keyboard Shortcut Coherence

- **Status**: Compliant
- **Single root listener**: Yes (`useKeyboardShortcuts.ts`)
- **Input suppression**: Yes (checks INPUT, TEXTAREA, contentEditable, Monaco)
- **Iframe relay**: Yes (uses `emitShortcutIntent` from `@vrooli/iframe-bridge`)

## Interop Compliance

| Slot | File | Status |
|------|------|--------|
| [A] | package.json | Compliant - has @vrooli/api-base + @vrooli/iframe-bridge + react-router-dom |
| [B] | vite.config.ts | Compliant - base: './' with INTEROP-CRITICAL comment |
| [C] | server.js | Compliant - uses startScenarioServer() |
| [D] | main.tsx | Compliant - initIframeBridgeChild with guard + appId |
| [E] | App.tsx | Compliant - BrowserRouter with basename="." + INTEROP-CRITICAL comment |
| [F] | lib/api.ts | Compliant - uses resolveApiBase + buildApiUrl |
| [G] | hooks/useKeyboardShortcuts.ts | Compliant - uses emitShortcutIntent |

## React Stability Compliance

| Check | Status |
|-------|--------|
| `strict: true` | Enabled (via tsconfig.node.json) |
| `noUncheckedIndexedAccess: true` | Enabled |
| `react-hooks/rules-of-hooks` | Enforced via ESLint |
| `@typescript-eslint/no-non-null-assertion` | Enforced via ESLint |
| `import/no-cycle` | Enforced via ESLint |
| Error Boundary | Present - wraps all page content |
| No scrollIntoView | Compliant (none used) |
| No h-screen | Compliant (uses h-full) |

## Test Coverage

| Surface | Tests |
|---------|-------|
| Layout | 7 tests (header, nav, health indicator) |
| Dashboard | 9 tests (stats, recent tasks, quick actions) |
| Tasks | 9 tests (CRUD, states, form) |
| Projects | 8 tests (CRUD, states, form) |
| **Total** | **30 tests passing** |

## Priority Actions

### Completed (Phase 11)

1. ✅ Added routing with React Router and proxy-aware basename
2. ✅ Implemented Dashboard, Tasks, Projects pages
3. ✅ Full API integration with typed client
4. ✅ Loading/error/empty states for all data-fetching components
5. ✅ Test selectors for automation

### Future (If Needed)

1. **Add design tokens**: Create `shared/theme/tokens.css` for custom theming
2. **Expand primitives**: Add Card, Input, Badge to `components/ui/`
3. **Consider stores**: If cross-surface state is needed, add minimal Zustand store
4. **Add Notes UI**: Notes CRUD currently API-only, could add UI if needed

## Relationship to Other Skills

| Skill | Status | Notes |
|-------|--------|-------|
| react-stability | Applied | ESLint + TS safety rules in place |
| vrooli-ui-interop | Compliant | All 7 slots verified |
| navigation-integrity-audit | Compliant | 3 routes with consistent nav |
| react-coherence | Applied | State/styling/structure aligned |

---

*Last updated: 2026-03-11 by Scenario Improver (Phase 11: Progress - UI Implementation)*
