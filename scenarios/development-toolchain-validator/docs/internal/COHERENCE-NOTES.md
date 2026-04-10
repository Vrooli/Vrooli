# Coherence Audit - 2026-03-11

Last updated by: Ecosystem Manager (React Coherence phase)

## State

- **Current pattern**: Server-state only (React Query for health check)
- **App-wide stores**: None (appropriate for this minimal template)
- **State hotspots**: None identified
- **Local state usage**: 0 useState hooks found

**Assessment**: Minimal state architecture is appropriate for the current template stage. As features are added, follow scope-driven state decisions:
1. Local `useState` for component-local ephemeral state
2. Feature-local hooks/stores for feature-scoped state
3. App-wide stores only when truly cross-surface
4. Server state via React Query (already in place)

## Duplication

- **Duplicate components**: None found
- **Duplicate hooks/services**: None found
- **Consolidation candidates**: None needed

**Assessment**: No duplication issues. Template is minimal and clean.

## Styling System

- **Token coverage**: None (no CSS variables defined)
- **Primitive variant coverage**: Partial (Button uses CVA with 2 variants)
- **Surface-level style debt**: App.tsx uses Tailwind utility classes directly (acceptable for template)

**Design Token Gaps:**
1. Color tokens (text, surface, accent) not extracted
2. Spacing tokens not defined
3. Border radius tokens not defined
4. Motion tokens not defined

**Assessment**: Template styling is functional but not theme-ready. Add design tokens when the scenario moves beyond template stage.

## Architecture Alignment

**Current structure:**
```
ui/src/
├── components/
│   ├── ui/             # Primitives (Button)
│   └── ErrorBoundary.tsx
├── consts/
│   └── selectors.ts    # Automation selectors
├── lib/
│   ├── api.ts          # API client (centralized)
│   └── utils.ts        # CN helper
├── App.tsx             # Main app component
├── main.tsx            # Entry point with providers
└── styles.css          # Tailwind imports
```

**Questions answered:**
- [x] Are visual contracts owned in shared/theme + shared/ui? **Partial** - Button in ui/, no theme folder
- [x] Are surfaces assembling rather than inventing primitives? **Yes** - App.tsx uses Button, doesn't recreate
- [x] Do controllers/services follow stable boundaries? **Yes** - api.ts is clean, single-purpose

## Iframe-Safe Layout

**Current issue**: App.tsx uses `min-h-screen` which compiles to `height: 100vh`. Per UI Interop §4.5, this is iframe-unsafe because `100vh` can refer to the outer window's viewport height, not the iframe's actual dimensions.

**Fix required**: Replace `min-h-screen` with `h-full` and establish proper height chain in CSS.

## Theme Refresh Readiness

- **Ready now / needs foundation work**: Needs foundation work
- **Required prerequisites**:
  1. Define semantic design tokens in CSS variables
  2. Document token categories (color, surface, border, radius, space, motion)
  3. Consider theme switching infrastructure when needed

## Priority Actions

1. **[COMPLETED 2026-03-11]** Fix `min-h-screen` to `h-full` with proper height chain for iframe safety
2. **[COMPLETED 2026-03-11]** Add basic design tokens to styles.css as foundation (6 categories)
3. **[DEFERRED]** Expand ui/primitives as scenario grows (Card, Input, Badge)
4. **[DEFERRED]** Add theme switching when product requirements call for it

## Notes for Future Agents

- This is a template UI - coherence improvements should be proportional to actual feature complexity
- The selector system (`consts/selectors.ts`) is already well-structured for automation
- API client (`lib/api.ts`) follows centralized resolution pattern per UI Interop skill
- ErrorBoundary is in place at app root; add component-level boundaries as UI grows
