# Polish Initiative Details Page Layout and Styling

## Required Reading

| Skill | Why |
|-------|-----|
| `ux` | UX improvement guidelines, responsive patterns, mobile-first design principles |
| `react-coherence` | React component patterns, Tailwind consistency, reusable component library |

```bash
prompt-manager skill read ux react-coherence
```

## Problem Statement

The Initiative Details Page (`InitiativeDetailsPage.tsx`) is the simplest of the four swarm-manager detail pages (Backlog, Scenario, Execution, Initiative). While functionally complete, it lacks the visual polish and layout sophistication of its siblings. Key gaps:

1. **Title duplication**: The initiative title appears in both the `DetailPageHeader` AND a separate title row inside the metadata card. This is redundant.
2. **Metadata layout**: Created/Updated timestamps use a simple inline layout instead of the structured grid pattern (`grid grid-cols-2 gap-3`) used by Scenario and Execution detail pages.
3. **Progress card lacks context**: The segmented progress bar has no percentage label or summary.
4. **Item chips lack status indicators**: Member items are displayed as colored chips but lack an icon or visual cue for item kind (fix, execute, idea, chore).
5. **Files card layout**: On mobile, the preview panel lacks a max-height constraint, potentially pushing content far down.
6. **No empty state**: When an initiative has zero items or zero files, there is no guidance or empty-state messaging.
7. **Inconsistent section header style**: The "Files" header has an icon while "Progress" and "Items" do not.
8. **Typography weight inconsistency**: Title in metadata card uses `text-xl font-semibold` while Backlog uses `text-xl font-bold`.

## Settled Decisions

### From Workshop Round 1

| Decision | Choice | Detail |
|----------|--------|--------|
| Duplicate title | **Remove from metadata card body** | Matches all other detail pages. Header is the sole title location. |
| Section header icons | **Add icons to all section headers** | Progress → `BarChart3`, Items → `Layers`. Matches Files having `FolderOpen`. |
| Item chip kind indicator | **Prefix chip with kind icon** | Wrench (fix), Play (execute), Lightbulb (idea), CheckSquare (chore). Clear visual signal without text overhead. |
| Empty states | **Add contextual empty states for both sections** | Muted message + subtle icon for zero items and zero files. Matches patterns elsewhere in the UI. |

### From Workshop Round 2

| Decision | Choice | Detail |
|----------|--------|--------|
| Progress summary format | **Fraction below bar: "3 of 7 completed"** | Clean, reads naturally as a caption below the progress bar. Best for small counts (<50 items). |
| Section header icons | **Progress: `BarChart3`, Items: `Layers`** | Both distinct from `FolderOpen`, commonly used in dashboard UIs. Visually balanced. |
| Chip truncation | **Show first 8, then "+N more" that expands** | Lower threshold keeps the card compact. Click-to-expand provides access without navigation. Requires local expanded/collapsed state. |
| Test approach | **Behavioral tests only** | Test presence and interaction (empty states render, kind icons present via test-id, progress summary text appears). Do not test CSS classes or grid structure. |

## Scope

### In Scope
- `scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx` — primary target
- `scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.test.tsx` — test updates
- Shared detail components in `scenarios/swarm-manager/ui/src/components/detail/` — minor tweaks if needed for consistency
- `scenarios/swarm-manager/ui/src/components/ui/` — if new small UI primitives needed (e.g., empty state)

### Out of Scope
- API changes or new endpoints
- New features (e.g., editing initiatives, creating items)
- Changes to other detail pages (Backlog, Scenario, Execution)
- State management or routing changes

## Current Technical Context

### Key Files
- **`InitiativeDetailsPage.tsx`** (379 lines): Main page component. Uses `DetailPageLayout`, queries initiative data and files, renders metadata card, progress card, items card, files card.
- **`InitiativeDetailsPage.test.tsx`** (244 lines): Tests rendering, chip clicks, error state, lens bar, fallback display.
- **`DetailPageHeader.tsx`** (74 lines): Shared header with close button, entity-type badge, title, status badge, actions slot.
- **`error-state.tsx`**: Auto-detects error variant (network, timeout, notFound, etc.) with retry button.

### Current Patterns
- Card styling: `bg-slate-900/45 border-slate-700/60 rounded-lg p-5`
- Section headers: `text-sm font-semibold uppercase tracking-wide text-slate-400`
- Timestamps: `formatRelativeTime()` with inline display (NOT grid)
- Item chips: `rounded-full px-2.5 py-1 text-xs font-medium` with `BACKLOG_STATUS_CHIP_COLORS`
- Progress bar: `h-2.5 rounded-full bg-slate-800` with colored segments
- File tree + preview: `grid-cols-1 lg:grid-cols-2`

### Cross-Page Comparison
| Pattern | Other Detail Pages | Initiative (Current) |
|---------|-------------------|---------------------|
| Timestamp layout | `grid grid-cols-2 gap-3` with small-caps labels | Inline, no grid |
| Section header icons | Consistent icons on all headers | Only Files has icon |
| Empty states | Contextual messages | Cards hidden if no data |
| Kind indicators on chips | N/A (not applicable) | No kind icons |

## Target End State

The Initiative Details Page should feel visually consistent with Scenario and Execution detail pages while maintaining its simpler content model. Specifically:

1. Single title in header only — no body duplication
2. Timestamps in 2-column grid with small-caps labels
3. All section headers have icon + label in same style (`BarChart3` for Progress, `Layers` for Items, `FolderOpen` for Files)
4. Progress bar shows completion fraction as caption: "3 of 7 completed"
5. Item chips prefixed with kind-specific Lucide icons (Wrench, Play, Lightbulb, CheckSquare)
6. Empty states displayed for zero items and zero files (muted text + icon)
7. File preview has `max-h-[60vh]` constraint on mobile
8. First 8 item chips visible, then "+N more" chip that expands to reveal all
9. Consistent typography weights matching cross-page conventions

## Implementation Strategy

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

### Phase 1: Layout & Structure Cleanup
1. Remove duplicate title row from metadata card body (the one with Target icon)
2. Restructure timestamps into `grid grid-cols-2 gap-3` with `text-xs text-slate-500 uppercase tracking-wider` labels
3. Normalize typography weight to match cross-page convention (`font-bold` for card titles)

### Phase 2: Visual Polish
1. Add `BarChart3` icon to Progress section header and `Layers` icon to Items section header (matching `FolderOpen` on Files)
2. Add completion summary text below progress bar: "X of Y completed" in `text-xs text-slate-400`
3. Add kind-specific Lucide icons to item chips: `{fix: Wrench, execute: Play, idea: Lightbulb, chore: CheckSquare}`
4. Refine card spacing for consistent visual rhythm

### Phase 3: Empty States & Edge Cases
1. Add empty state for zero member items: muted text + `Inbox` icon + "No items assigned to this initiative yet"
2. Add empty state for zero files: muted text + `FileX` icon + "No files associated with this initiative"
3. Add `max-h-[60vh] overflow-auto` to file preview panel on mobile viewports
4. Implement click-to-expand truncation for item chips: show first 8, then a "+N more" chip that toggles expanded state to reveal all items. Use local `useState` for expanded/collapsed state.

### Phase 4: Test Updates
1. Update existing tests to account for removed title row
2. Add test: empty state renders when initiative has zero items
3. Add test: empty state renders when initiative has zero files
4. Add test: kind icons render on item chips (via `data-testid` or accessible label)
5. Add test: progress summary text renders with correct counts
6. Add test: "+N more" chip renders when items > 8 and expands on click
7. Verify all existing tests still pass

### Final: Cleanup & Verification
- Run type checking (`npx tsc --noEmit`) and fix ALL errors, even pre-existing
- Run linter (`eslint`) and fix ALL warnings in modified files
- Run unit tests and fix any failures
- `vrooli scenario restart swarm-manager`
- Verify health: `curl -s http://localhost:<port>/health`

## Contract Decisions

No API or data model changes. All changes are presentational:
- Kind icon mapping: `{fix: Wrench, execute: Play, idea: Lightbulb, chore: CheckSquare}` — derived from item kind string
- Progress summary format: `"X of Y completed"` — computed from rollup totals
- Chip truncation threshold: **8 items** before showing "+N more" (click-to-expand)
- Empty state messages: static strings, no i18n needed for now

## Testing Plan

### Automated Tests (in `InitiativeDetailsPage.test.tsx`)
- All existing tests must pass (9 current tests)
- New behavioral tests for:
  - Empty state rendering (zero items)
  - Empty state rendering (zero files)
  - Kind icon presence on item chips (via `data-testid` or accessible label)
  - Progress summary text with correct fraction
  - "+N more" chip when items exceed 8, and click-to-expand behavior

### Manual Verification
- Visual check at 375px, 414px, 768px viewports
- Confirm no horizontal overflow in any card
- Confirm file preview max-height works on mobile

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Shared component changes affect other pages | Low | Medium | Only modify shared components with additive changes (new optional props) |
| Progress bar percentage edge cases | Low | Low | Handle zero-total case (guard already exists) |
| Item chip overflow with many items | Medium | Low | Implement "+N more" truncation at 8 items with expand toggle |
| Expanded state increases card height unpredictably | Low | Low | Items are small chips; even 20+ items won't overflow the page. Consider max-height if needed later. |
| Test breakage from removed title | Low | Low | Update assertions to match new structure |

## Non-goals / Prohibited Patterns

- Do NOT add new API endpoints or modify data models
- Do NOT change other detail pages (Backlog, Scenario, Execution)
- Do NOT add state management or routing changes
- Do NOT add i18n infrastructure for empty state messages
- Do NOT create new shared components unless absolutely necessary — prefer inline solutions
- Do NOT add snapshot tests — use behavioral tests only

## Definition of Done

- [ ] No duplicate title displayed (removed from metadata card body)
- [ ] Timestamps in `grid grid-cols-2 gap-3` layout with small-caps labels
- [ ] All section headers have consistent icon + label pattern (BarChart3, Layers, FolderOpen)
- [ ] Progress bar shows "X of Y completed" summary below the bar
- [ ] Item chips show kind-specific Lucide icons
- [ ] Empty states render for zero items and zero files
- [ ] File preview has max-height constraint on mobile
- [ ] First 8 chips shown, "+N more" chip expands to show all on click
- [ ] All existing tests pass
- [ ] New behavioral tests for empty states, kind icons, progress summary, chip truncation + expand
- [ ] No horizontal overflow at 375px viewport
- [ ] Consistent typography weights across sections
- [ ] Type checking passes with no errors
- [ ] Linter passes on modified files
- [ ] Scenario restarts and health check passes
