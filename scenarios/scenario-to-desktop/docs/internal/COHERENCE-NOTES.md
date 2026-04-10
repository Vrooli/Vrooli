# Coherence Audit — 2026-03-19

## Responsive Layout

### Problem
The UI was designed desktop-first with no mobile breakpoints. On viewports < 768 px:
- Tab navigation overflowed horizontally (labels clipped on both edges)
- Sidebar (`w-72`) consumed 77% of a 375 px screen, crushing the main content
- `p-6` padding wasted 48 px of horizontal space
- Header `text-4xl` + subtitle consumed excessive vertical real estate

### Changes Made

| Component | Before | After |
|-----------|--------|-------|
| **Tab bar** (`App.tsx`) | Static `inline-flex`, no scroll | Scrollable container with `overflow-x-auto scrollbar-hide`; icon-only inactive tabs on mobile; ARIA `role="tab"` added |
| **Tab labels** | "Scenario Inventory", "Generate Desktop App" | "Inventory", "Generate" (shorter, fits mobile) |
| **Header** | `text-4xl`, `mb-8`, always shows subtitle | `text-2xl md:text-4xl`, `mb-4 md:mb-8`, subtitle hidden on mobile |
| **Outer padding** | `p-6` everywhere | `p-3 md:p-6` |
| **GeneratorLayout** | Always shows sidebar + main side-by-side | Desktop: sidebar + main. Mobile: `MobilePipelineSummary` bar + full-width main; sidebar in slide-out drawer |
| **PipelineSidebar** | Collapse toggle always visible | Toggle hidden on mobile (drawer handles open/close) |
| **Sidebar store** | `collapsed` only | Added `mobileDrawerOpen` state |

### New Components
- `useMediaQuery` / `useIsMobile` hook — single responsive breakpoint seam
- `MobilePipelineSummary` — compact pipeline status bar for mobile

### New CSS Utility
- `.scrollbar-hide` in `styles.css` — hides scrollbar on the tab bar

## State Architecture
- **Local-first**: `useMediaQuery` is local per-component; no global responsive state
- **Store-scoped drawer**: `mobileDrawerOpen` in sidebarStore is the only new store field; ephemeral (not persisted to localStorage)

## Styling Coherence
- All responsive classes use Tailwind's `md:` prefix (768 px breakpoint)
- No raw breakpoint values in TSX; all flow through Tailwind or `useIsMobile`
- No new design tokens introduced — existing palette reused

## Test Coverage
- `useMediaQuery.test.ts` — 8 tests covering initial match, change events, cleanup, useIsMobile delegation
- `GeneratorLayout.test.tsx` — 6 tests covering desktop sidebar, mobile summary bar, drawer open/close, section click closes drawer
- `App.test.tsx` — Updated to reflect shortened tab labels and ARIA roles

## Priority Actions (Future)
1. Add `useMediaQuery`-based responsive behavior to `ScenarioInventory` grid (currently uses `md:grid-cols-4` which is fine, but inventory search bar could stack better)
2. Consider converting modals to bottom sheets on mobile for better reachability
3. Add touch-friendly padding to sidebar navigation items in mobile drawer
