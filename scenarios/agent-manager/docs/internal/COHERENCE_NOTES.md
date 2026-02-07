# React Coherence Audit - 2026-02-06

## State Management

**Current Pattern**: Local React state + hooks, with no Zustand stores yet in `ui/src`.

**Stores Found**:
- None

**Observations**:
- Highest state density remains in `RunDetail.tsx` and page containers (`RunsPage.tsx`, `TasksPage.tsx`).
- Replaced stats `TimeWindow` Context with a centralized module-level store in `ui/src/features/stats/hooks/useTimeWindow.tsx`.
- `TimeWindowProvider` is now a lightweight compatibility wrapper that applies `defaultPreset` lifecycle behavior without Context coupling.
- `RunDetail.tsx` had duplicated collapsible/resizable panel logic that diverged from shared hooks.
- This pass consolidated `RunDetail.tsx` onto shared hooks to reduce bespoke state/effect management and align panel behavior.

## Duplication

**Resolved in this pass**:
- Consolidated duplicate pricing display logic (`sourceColor`, `sourceLabel`, `formatPrice`, timestamp formatting) from:
  - `ui/src/components/dialogs/EditPricingDialog.tsx`
  - `ui/src/components/dialogs/SettingsDialog/ModelPricingTab.tsx`
- Canonicalized into:
  - `ui/src/components/dialogs/pricingDisplay.ts`
- Further consolidated pricing source badge styling composition into shared `pricingSourceBadgeClass` utility in:
  - `ui/src/components/dialogs/pricingDisplay.ts`
  - consumed by `EditPricingDialog.tsx` and `SettingsDialog/ModelPricingTab.tsx`

## Styling

**Status**: No major token/CVA migration in this pass.

**Notes**:
- Existing arbitrary spacing remains in `ui/src/components/ui/scroll-area.tsx` (`p-[1px]`), inherited from Radix wrapper implementation.

## Priority Actions

1. ✅ Consolidated stats time-window shared state from React Context into a single store-style hook module
2. ✅ Reused shared panel hooks in `RunDetail.tsx` instead of local resize/collapse implementations
3. ✅ Extended shared panel hooks with optional persistence key + axis support for coherent reuse and legacy key compatibility
4. ✅ Extracted and reused shared pricing display helpers across pricing dialogs

## Follow-ups

1. Evaluate introducing focused shared stores (or reducer-backed modules) for cross-component state in `RunsPage.tsx` and `TasksPage.tsx` as complexity grows.
2. Reduce duplicate date/relative-time formatting between general UI utils and stats feature utils in a future coherence pass.

---

# React Coherence Update - 2026-02-07

## Completed This Pass

1. Extracted `RunsPage` list/investigation state into `ui/src/hooks/useRunsPageState.ts`:
   - Search/filter/sort state
   - Selection mode + selected IDs
   - Investigation modal loading/error state
   - Shared selection handlers (`toggleSelectionMode`, shift-range selection)

2. Extracted `TasksPage` run/profile dialog state into `ui/src/hooks/useTasksRunDialogState.ts`:
   - Run dialog + profile dialog visibility
   - Run config mode, selected profile, sandbox reuse ID
   - Inline run config + profile form state
   - Fallback runner list mutators and dialog reset helpers

3. Wired both pages to consume these shared hooks without changing workflows:
   - `ui/src/pages/RunsPage.tsx`
   - `ui/src/pages/TasksPage.tsx`

## Validation

- `pnpm lint` (UI): passes with warnings only (no new errors)
- `pnpm build` (UI): passes
- `make test` (scenario): passes all phases

---

# React Coherence Update - 2026-02-07

## Completed This Pass

1. Strengthened styling coherence in pricing table headers in `ui/src/components/dialogs/SettingsDialog/ModelPricingTab.tsx`:
   - Replaced dynamic Tailwind class interpolation (`text-${align}`) with explicit alignment/justify maps.
   - Composed classes via shared `cn` helper for consistent class composition patterns.

2. Improved settings form state coherence in `ui/src/components/dialogs/SettingsDialog/ModelPricingTab.tsx`:
   - Added `useEffect` synchronization so pricing settings inputs reliably reflect asynchronously loaded API settings.
   - Kept save behavior and API workflow unchanged.

## Validation

- `pnpm lint` (UI): passes with existing warnings only (no new errors)
- `pnpm build` (UI): passes
- `vrooli scenario test agent-manager`: passes (9/9 phases)
