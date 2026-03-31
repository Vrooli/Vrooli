# Fix: Capture Button Position and Z-Index in Swarm-Manager Sidebar

## Required Reading

- `prompt-manager skill read implementation-plan-authoring` — Plan structure and quality gates

## Problem Statement

Two related CSS bugs with the swarm-manager capture FAB (floating action button):

1. **Position too high**: The capture button appears too high on the page. Currently positioned at `bottom: calc(5rem + env(safe-area-inset-bottom))` — needs to be lower.
2. **Z-index collision with sidebar**: Both the capture FAB and the sidebar container use `z-30`. When the sidebar is open, the capture button remains visible instead of being covered by the sidebar.

### Root Cause

In `floating-action-button.tsx` (line 28), the FAB uses `z-30`. The sidebar container in `Sidebar.tsx` also uses `z-30`. Since they share the same z-index, the render order determines which appears on top — and the FAB renders after the sidebar in the DOM, so it floats above.

The position issue is a simple CSS adjustment — the `bottom` value needs to be reduced to move the button lower on the page.

## Accepted Decisions

### D1 (Round 1): Fix strategy — Raise sidebar to z-40

The sidebar `<aside>` element will be raised from `z-30` to `z-40`. This means it will naturally cover the FAB (z-30) when open.

**Risk noted:** FloatingPanel (capture panel, detail overlay, help panel) also uses `z-40`. When the sidebar is open alongside a floating panel, they'll share the same z-index. This is acceptable because:
- On mobile, the sidebar goes full-screen so no overlap occurs
- On desktop, the sidebar is `md:relative` and participates in normal flow for its column, while FloatingPanel is absolutely/fixed positioned in a different area — spatial separation prevents visual conflict

### D2 (Round 1): Button position — Reduce 5rem to 3rem

The FAB's inline `bottom` style will change from `calc(5rem + env(safe-area-inset-bottom))` to `calc(3rem + env(safe-area-inset-bottom))`. This preserves the safe-area-inset pattern for mobile devices while moving the button ~32px lower.

### D3 (Round 2): Verification approach — Manual visual verification only

Since these are purely visual CSS changes (z-index and position), automated tests would just assert class names which is brittle and low-value. Manual visual verification across desktop/mobile viewports is sufficient.

### D4 (Round 2): Remove FAB right-offset when sidebar is open

The FAB currently shifts right via `md:right-[21.5rem]` (passed as className in `GraphWorkspace.tsx`) when the sidebar is open. After raising sidebar to z-40, the sidebar covers the FAB anyway, making this offset unnecessary. Remove the conditional right-offset class to simplify the code.

## Acceptance Criteria

- [ ] Capture button is positioned lower on the page (bottom changes from 5rem to 3rem base)
- [ ] When sidebar is open, the sidebar fully covers/hides the capture button
- [ ] No regressions to mobile sidebar behavior (backdrop at z-20 still works)
- [ ] No regressions to capture panel (z-40) or detail overlay (z-40) stacking
- [ ] FloatingPanel still appears correctly when sidebar is open on desktop
- [ ] FAB no longer shifts right when sidebar opens (unnecessary with z-40 sidebar)

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `scenarios/swarm-manager/ui/src/components/ui/floating-action-button.tsx` | Change `bottom` from `5rem` to `3rem` in inline style |
| `scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/Sidebar.tsx` | Change sidebar `<aside>` from `z-30` to `z-40` |
| `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx` | Remove conditional `md:right-[21.5rem]` className on FloatingActionButton |

### Out of Scope

- Capture panel (CapturePanel.tsx) — already at z-40, not affected
- Mobile-specific sidebar behavior changes beyond z-index
- Any backend changes
- Other z-index consumers (dialogs at z-50, popovers at z-50, etc.)
- Automated tests — manual visual verification per D3

## Current → Target Z-Index Hierarchy

| Z-Index | Component | Change? |
|---------|-----------|---------|
| `z-10` | Graph canvas elements | No |
| `z-20` | HUD bar, mobile backdrop, toggle buttons | No |
| `z-30` | **Capture FAB** | No change (stays z-30) |
| `z-40` | FloatingPanel, Detail overlay, **Sidebar** ← moved up | **Sidebar raised from z-30** |
| `z-50` | Popover menus, dialogs | No |

## Implementation Plan

### Phase 1: CSS Changes (3 files, ~3 lines changed)

**Step 1:** In `floating-action-button.tsx`, change the inline style:
```tsx
// Before
bottom: "calc(5rem + env(safe-area-inset-bottom))",
// After
bottom: "calc(3rem + env(safe-area-inset-bottom))",
```

**Step 2:** In `Sidebar.tsx`, change the sidebar `<aside>` className:
```tsx
// Before
"fixed inset-0 z-30 flex w-full flex-col ..."
// After
"fixed inset-0 z-40 flex w-full flex-col ..."
```

**Step 3:** In `GraphWorkspace.tsx`, remove the sidebar-open right-offset from the FAB:
```tsx
// Before
<FloatingActionButton
  ...
  className={cn(!sidebarCollapsed && "md:right-[21.5rem]")}
/>
// After
<FloatingActionButton
  ...
/>
```

### Phase 2: Verification (Manual)

- Visual check: FAB position lower on page
- Open sidebar → FAB should be hidden behind it
- Open capture panel with sidebar open → capture panel still visible
- Test on mobile viewport → sidebar covers full screen, FAB hidden
- Verify FAB no longer shifts right when sidebar opens

## Testing Strategy

Manual visual verification only (per D3). No automated tests for this change.

- Verify button position on desktop and mobile viewports
- Verify sidebar covers capture button when opened
- Verify capture panel (z-40) still appears above sidebar (both z-40, but capture panel is positioned independently)
- Verify detail overlay still works correctly
- Verify popovers and dialogs (z-50) still layer correctly above sidebar

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Sidebar z-40 conflicts with FloatingPanel z-40 | Low | Spatial separation on desktop; full-screen sidebar on mobile eliminates overlap |
| Bottom position change looks wrong on specific devices | Low | `env(safe-area-inset-bottom)` handles device-specific offsets; 3rem is a standard FAB distance |
| Other components at z-30 affected by sidebar change | None | Only the sidebar is changing; other z-30 consumers (sticky headers) are in different stacking contexts |
| Removing right-offset causes flash of wrong position | None | Sidebar at z-40 covers FAB immediately; no visible intermediate state |
