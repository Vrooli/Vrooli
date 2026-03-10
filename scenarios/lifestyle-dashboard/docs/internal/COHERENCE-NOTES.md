# Coherence Audit - 2026-03-10

## State Architecture

- **Current pattern**: local-first with server state via React Query
- **App-wide stores**: None (correct for current scope)
- **State hotspots**: None identified (minimal local state usage)
- **Assessment**: Good - follows scope-driven state architecture

### State Inventory

| File | useState calls | Purpose |
|------|---------------|---------|
| EventsPage.tsx | 1 | Local filter state |
| All others | 0 | Pure server state via React Query |

No zustand stores, no excessive context usage. Server state correctly managed via `@tanstack/react-query`.

## Duplication

### Fixed Issues

1. **Type duplication resolved**: `Domain` interface was duplicated in both `lib/api.ts` and `DomainsPage.tsx`. Now imports from `lib/api.ts` as single source of truth.

### Remaining (Acceptable)

1. Domain card rendering pattern appears in both `DomainCard.tsx` (reusable) and inline in `DomainsPage.tsx` (page-specific layout). This is acceptable as they serve different purposes.

## Styling System

- **Token coverage**: Partial (using Tailwind color palette, no custom semantic tokens)
- **Primitive variant coverage**: Good (Button and Card use CVA)
- **Surface-level style debt**: Minimal

### CVA Primitives

| Component | Variants | Status |
|-----------|----------|--------|
| Button | `variant: default/outline`, `size: default/sm` | Good |
| Card | `interactive: true/false`, `padding: default/lg/none` | New - Phase 10 |

### Design Token Opportunities (Future)

If a theme refresh is planned, consider extracting:
- Card border/background colors to semantic tokens
- Status colors (emerald, amber, red for healthy/degraded/unhealthy) to tokens

## Code Organization

### Current Structure (Aligned)

```
src/
├── components/
│   ├── dashboard/     # Feature-specific (9 components)
│   ├── ui/            # Primitives (Button, Card)
│   ├── ErrorAlert.tsx # Shared error handling
│   └── Layout.tsx     # App shell
├── consts/selectors.ts # QA automation selectors
├── lib/
│   ├── api.ts         # Centralized API layer
│   ├── format.ts      # Formatting utilities
│   └── utils.ts       # General utilities (cn)
├── pages/             # Route-level components (5 pages)
└── main.tsx           # Entry point
```

### Assessment

- **Visual contracts**: Owned in `components/ui/` with CVA variants
- **Surfaces**: Pages correctly assemble primitives rather than inventing new ones
- **Services**: Centralized in `lib/api.ts` with proper error handling
- **Architecture**: Clean separation between pages, components, and utilities

## Theme Refresh Readiness

**Status**: Foundation ready, not urgent

**Prerequisites for refresh**:
1. Extract semantic color tokens if needed
2. Expand primitive set (Input, Badge, Modal if needed)
3. Current dark theme is consistent, no mixed styling contracts

## Priority Actions Completed (Phase 10)

1. Fixed type duplication (Domain interface)
2. Created Card primitive with CVA variants
3. Refactored DashboardPage, StatCard, DomainCard to use Card primitive
4. Created UI barrel export (`components/ui/index.ts`)

## Recommendations for Future Phases

1. **No immediate action needed** - Architecture is coherent
2. **If adding forms**: Create Input primitive with consistent styling
3. **If adding modals**: Create Modal primitive to avoid duplication
4. **If theme refresh**: Define semantic tokens in `styles.css` first
