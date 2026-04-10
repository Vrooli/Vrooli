# Fix: Snooze Selector Rendering on Backlog Item Cards

## Required Reading

- `prompt-manager skill read react-coherence` — React component patterns and portal usage
- `prompt-manager skill read ux` — UI/UX interaction patterns

## Problem Statement

The `SnoozePopover` component on backlog item cards renders its dropdown menu using CSS `absolute` positioning inside a parent container that has `overflow-x-auto`. This causes the dropdown to be clipped by the card's overflow boundary, making it partially or fully invisible to the user.

### Root Cause Analysis

**Observation:** The snooze popover menu (`SnoozePopover.tsx:49-72`) uses `absolute right-0 top-full z-50 mt-1 w-40` classes, positioned relative to its trigger button. However, the action row in `backlog-card.tsx:111` uses `overflow-x-auto`, which creates a new stacking context and clips any absolutely-positioned child that extends beyond its bounds.

**Comparison:** The `StatusChipPopover` component (`status-chip-popover.tsx`) already solves this exact problem by using the shared `Popover` component (`ui/popover.tsx`), which:
1. Uses `createPortal` to render at `document.body`
2. Uses `fixed` positioning with viewport clamping
3. Escapes all parent overflow/z-index constraints

## Scope

### In Scope
- Refactor `SnoozePopover` to use the existing `Popover` component from `ui/popover.tsx`
- Ensure click-outside dismissal still works (handled by `Popover` via `useModalBehavior`)
- Ensure snooze popover positions correctly relative to its trigger

### Out of Scope
- Changes to snooze store logic or presets
- Changes to the `Popover` component itself
- Backlog card layout changes (overflow-x-auto stays as-is)

## Approach

**Decision (d1):** Reuse existing `Popover` component — the proven pattern already used by `StatusChipPopover` in the same card context. Gets portal rendering, viewport clamping, and click-outside/Esc handling for free.

**Decision (d2):** Keep `overflow-x-auto` on the action row — minimal change, focused fix. The overflow serves a purpose for long action rows on narrow screens.

## Implementation Steps

### Step 1: Refactor `SnoozePopover` to use the `Popover` component

**File:** `scenarios/swarm-manager/ui/src/components/command-post/SnoozePopover.tsx`

Follow the `StatusChipPopover` pattern:

1. **Replace state/ref management:**
   - Remove the manual `handleClickOutside` effect and `popoverRef` — `Popover` handles this via `useModalBehavior`
   - Add `popoverPos` state (`{ x: number, y: number }`) for fixed positioning
   - Add a `buttonRef` on the trigger button

2. **Compute position on open:**
   - In the trigger button's `onClick`, call `buttonRef.current.getBoundingClientRect()` to get the trigger's screen position
   - Set `popoverPos` to `{ x: rect.left, y: rect.bottom + 4 }` (same offset as StatusChipPopover)

3. **Replace the dropdown div with `<Popover>`:**
   - Use `<Popover isOpen={open} onClose={() => setOpen(false)} x={popoverPos.x} y={popoverPos.y} className="w-40 p-1" testId="snooze-popover">`
   - Move the preset buttons inside `<Popover>` children
   - Remove the outer `<div className="relative">` wrapper — no longer needed since portal renders at body

4. **Update imports:**
   - Add: `import { Popover } from "../ui/popover"`
   - Remove: `useEffect`, `useCallback` (no longer needed for click-outside)
   - Keep: `useState`, `useRef`

### Step 2: Verify all usage sites

**File:** `scenarios/swarm-manager/ui/src/components/backlog/backlog-card.tsx`

The `SnoozePopover` is used in 3 positions within `backlog-card.tsx`. Since the component's external API (`itemKey`, `onSnooze`, `children`) does not change, no modifications are needed in the card. Verify that the wrapper element change (from `<div className="relative">` to a fragment or plain wrapper) doesn't break the card layout.

## Testing Strategy

1. **Visual verification:** Open backlog page, click snooze on a card, confirm dropdown renders fully visible above/below the card — not clipped by the action row
2. **Click-outside:** Confirm clicking outside dismisses the popover
3. **Escape key:** Confirm pressing Escape dismisses the popover (new behavior from `useModalBehavior`)
4. **Scroll behavior:** Confirm popover stays anchored to trigger position (fixed positioning)
5. **Multiple cards:** Confirm opening snooze on one card closes any other open snooze popover
6. **Snooze presets:** Confirm selecting a snooze preset still calls `onSnooze` and closes the popover

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Portal z-index conflicts | Low | `Popover` uses `z-50`, consistent with all other popovers |
| Viewport edge clipping | Low | `Popover` already handles viewport clamping |
| Wrapper element change breaks card layout | Low | Verify the trigger button still flows correctly in the action row without the `relative` div |

## Acceptance Criteria

- Snooze dropdown is fully visible when triggered from any backlog card position
- Click-outside and Escape dismissal work
- Snooze functionality (selecting a preset) works end-to-end
- No visual regressions on other card actions
