# System Monitor UI — React Coherence Audit

**Last updated:** 2026-02-16 (Audit #2)
**Scope:** `scenarios/system-monitor/ui/src/`
**Total files:** 51 (31 components, 8 hooks, 3 type files, 1 CSS, 1 API helper, 7 others)
**Overall coherence score:** 6/10 (good foundation, inconsistent execution)

---

## 1. State Architecture
[CODE: ui/src/features/monitoring/hooks/useSystemMonitor.ts]

### State Inventory (Audit #2)

| Location | Hook | Purpose | Count |
|---|---|---|---|
| App.tsx | useState | dashboardState, systemSettingsModalOpen | 2 |
| useSystemMonitor | useState | metrics, detailedMetrics, processMonitorData, infrastructureData, investigations, metricHistory, isLoading, error, healthStatus, healthError, uiBoostActive | 11 |
| useInvestigationAgents | useState | agents, isSpawningAgent, stoppingAgents, agentErrors, spawnAgentError | 5 |
| useScriptExecution | useState | modalState (composite object) | 1 |
| InvestigationScriptsPage | useState+useReducer | 7 useState + 1 useReducer | 8 |
| InvestigationsSection | useState | reportsExpanded, scriptsExpanded, autoFixEnabled, reportsSearch, scriptsSearch, agentNote, showNoteField | 7 |
| AutomaticTriggersSection | useState | 6 states for trigger form | 6 |
| DiskDetailView | useState | 6 states for disk form/fetch | 6 |
| SystemSettingsModal | useState | settings, originalSettings, loading, saving, error, successMessage | 5 |
| ScriptEditorModal | useState | 4 states for editor | 4 |
| ReportsPanel | useState | 4 states for reports | 4 |
| ProcessMonitor | useState | confirmDialog | 1 |
| Header | useState | agentsOpen | 1 |

**Total useState:** 61+
**Zustand stores:** 0
**Context providers:** 0

### Assessment
- **No global store.** All state is component-local or in custom hooks. Heavy prop drilling from App.tsx (~50 props passed down).
- **useSystemMonitor is a god hook** (11 useState, 482 lines) — manages 5 metric types, 2 health states, 3 UI states, and 2 polling intervals. Should be split into 4-5 focused hooks.
- **InvestigationScriptsPage still has 8 state vars** — candidate for `useScriptsLibrary` hook.
- **InvestigationsSection has 7 useState** — form states should combine into a single reducer.

### Bugs Found
[CODE: ui/src/features/investigations/hooks/useInvestigationAgents.ts]
- **Race condition** in `useInvestigationAgents` (lines 306-337): `pollAgentStatuses()` runs multiple concurrent fetches without dedup. State updates can race.
- **Missing setTimeout cleanup** in `SystemSettingsModal` (line 97): `setTimeout(..., 3000)` has no useEffect cleanup — can fire after unmount.
- **Unmounted state updates** in `useSystemMonitor`: Multiple `setState()` calls inside async callbacks without consistent `mountedRef` checks.

### Recommendations (Updated)
1. ~~Extract script execution logic from App.tsx~~ — **DONE** (Round 2)
2. ~~Consolidate StatusIndicator's health check~~ — **DONE** (Round 2)
3. **Split `useSystemMonitor`** into `useMetricsFetch`, `useProcessMonitor`, `useInfrastructureMonitor`, `useSystemHealth`
4. **Introduce AppContext** or zustand store to eliminate top-level prop drilling
5. **Fix bugs**: setTimeout cleanup, race condition, unmounted state updates
6. Extract InvestigationScriptsPage state into `useScriptsLibrary` hook

---

## 2. Duplication
[CODE: ui/src/shared/api/apiFetch.ts]

### Previously Fixed (Rounds 1-4)
- Utility functions consolidated to `shared/utils/formatters.ts`
- `<Modal>` and `<ModalHeader>` shared components created
- `useClickOutside` hook extracted
- JS hover effects replaced with CSS `:hover`
- `ScriptListItem` extracted for script rendering
- `ConfirmDialog` extracted

### Current Duplication Issues

| Pattern | Severity | Locations | Notes |
|---|---|---|---|
| **Ad-hoc fetch implementations** | CRITICAL | 29 instances vs. 2 uses of `useApiCall` | Each reimplements error handling, loading state, AbortController |
| **Metric detail view overlap** | HIGH | CpuDetailView, MemoryDetailView, DiskDetailView, NetworkDetailView, GpuDetailView | 60-70% identical code across 5 files (header, chart, grid structure) |
| **DetailRow component unused** | HIGH | Exists in `shared/components/DetailRow.tsx` but not imported in 6+ locations | MetricCard, CpuDetailView, MemoryDetailView, DiskDetailView, ProcessMonitor, GpuDetailView all use inline detail-row markup |
| **Response parsing helpers** | MEDIUM | `useInvestigationAgents.ts` (extractString/extractBoolean/extractNumber) duplicated in `useScriptExecution.ts` (readString/readNumber/readBoolean) | Should be shared `typeGuards.ts` |
| **Formatters scattered** | MEDIUM | `formatters.ts` + `metricHelpers.tsx` | `formatMbPerSecond` and `formatInteger` belong in `formatters.ts`; `metricHelpers.tsx` mixes formatters with data builders |
| **Modal state patterns** | MEDIUM | 4 different approaches: composite useState (useScriptExecution), individual useState (InvestigationsSection), prop-based (ScriptEditorModal), per-field (SystemSettingsModal) | No unified pattern |
| **Collapsible panel pattern** | LOW | MetricCard, InfrastructureMonitor, ProcessMonitor, InvestigationsSection | Same ChevronUp/ChevronDown toggle; no shared component |
| **Timeline data transforms** | LOW | `metricHelpers.tsx` (buildSingleSeriesData, combineDiskSeries) and `useSystemMonitor.ts` (appendHistoryPoint, cloneSeries, ensureHistoryBase) | Split between two files |

### Priority Actions
1. **Create API service layer** — eliminates 29 fetch duplications (biggest single win)
2. **Adopt DetailRow in 6+ locations** — component already exists, just unused
3. **Extract shared `typeGuards.ts`** from response parsing duplicates
4. **Consolidate formatters** — move `formatMbPerSecond`, `formatInteger` to `formatters.ts`
5. **Parameterize metric detail views** — factory/config pattern to reduce 5 files → 1 factory + 5 configs

---

## 3. Styling System

### Token Coverage
`matrix-theme.css` (1,426 lines) provides a comprehensive token system:
- **Colors:** 50+ CSS custom properties including 20+ alpha variants (`--alpha-accent-02` through `--alpha-accent-80`)
- **Spacing:** 6 levels (xs through xxl) — consistently used, zero arbitrary spacing values
- **Typography:** 7 variables (1 font family, 6 sizes)
- **Border radius:** 3 levels
- **Shadows:** 3 variants
- **Transitions:** 3 speeds
- **Overlays:** `--overlay-light` through `--overlay-solid`

### CSS Classes
75+ utility classes + 60+ component classes defined. Good naming conventions.

### Issues (Current)

| Issue | Severity | Details |
|---|---|---|
| **550+ inline style declarations** | HIGH | 20+ components; top offenders: MetricCard (100+), DiskDetailView (80+), SystemSettingsModal (50+), LoadingSkeleton (40+) |
| **Hardcoded colors** | MEDIUM | ~15 instances across 4 files: Header.tsx (`#000`, `rgba(7,25,16,*)`), ErrorBoundary.tsx (`#0a0a0a`, `#1a1a1a`, `rgba(255,0,0,*)`), ScriptEditorModal.tsx (`rgba(0,0,0,0.8)`), InvestigationScriptsPage.tsx (5 `rgba` instances) |
| **No CVA** | MEDIUM | Button variants, card states, status indicators all use inline conditional styles or template literal class concatenation |
| **Monolithic CSS** | LOW | 1,426 lines in single file; no modularization into tokens/globals/components |
| **Unused dependencies** | LOW | `clsx` installed but never imported; Tailwind installed but never used |

### Scoring
| Dimension | Rating |
|---|---|
| Token infrastructure | 5/5 |
| Token adherence | 3/5 |
| Inline style prevalence | 2/5 |
| Class organization | 4/5 |
| Arbitrary values | 5/5 |
| Theme modularity | 2/5 |

### Previous Fixes (Rounds 1-4)
- Deleted dead `App.css` and `index.css`
- Added overlay alpha tokens, `.input-field`, `.input-label`, badge classes, detail-row classes
- Replaced JS hover with CSS `:hover` across 7 components
- Replaced ~25 hardcoded `rgba()` values with tokens
- Migrated many inline styles to CSS classes across 10+ files

### Dead CSS (FIXED)
- `App.css` — Vite scaffold (`.logo`, `.read-the-docs`, `@keyframes logo-spin`). **Removed** (was not imported).
- `index.css` — Vite scaffold with hardcoded colors (`#242424`, `#646cff`, `#1a1a1a`). **Removed** (was not imported).

---

## 4. Theme Refresh Readiness

### Current State: Phase 1 (Token Stabilization — Partial)

| Criterion | Status |
|---|---|
| Token system defined | Yes — comprehensive CSS variables |
| Tokens used consistently | Partial — most inline styles reference `var(--...)` but ~15 hardcoded colors remain |
| No raw hex/rgb in TSX | No — down from ~20 to ~15 instances |
| CVA or class-based variants | No — all variants are inline conditional styles |
| Component primitives extracted | Partial — Modal, ConfirmDialog, DetailRow, SearchInput, StatusBadge exist; no Button, Input, Card primitives |
| Design system package | No — everything lives in the scenario's `src/` |

### Migration Path
1. ~~**Phase 1a — Token stabilization**~~ — **DONE** (overlay tokens, `.input-field`, `.modal-dialog` classes added)
2. **Phase 1b — Remaining token gaps:** Add `--color-button-idle`, `--color-code-bg` tokens. Replace 15 remaining hardcoded colors.
3. **Phase 2 — Primitive extraction:** Create `shared/ui/Button.tsx`, `shared/ui/Input.tsx` with CSS class variants (or CVA).
4. **Phase 3 — Inline style migration:** Systematically replace 550+ `style={{}}` objects. Extract repeated patterns like `display:flex; flex-direction:column; gap:var(--spacing-lg)` into utility classes.
5. **Phase 4 — CSS modularization:** Split 1,426-line `matrix-theme.css` into `tokens/`, `globals.css`, `components.css`, `utilities.css`.

---

## 5. Architecture Notes

### Oversized Components (>200 lines)

| Component | Lines | Primary Issue |
|---|---|---|
| MetricCard.tsx | 716 | Color logic, formatting, threshold config, full UI tree mixed |
| AutomaticTriggersSection.tsx | 685 | Form state + API calls + trigger config UI |
| DiskDetailView.tsx | 573 | Disk scanning, history charting, file browsing interdependent |
| useSystemMonitor.ts | 482 | God hook: 11 useState, 6 fetch functions, 2 polling intervals |
| Header.tsx | 475 | 170-line agent dropdown embedded |
| InvestigationsSection.tsx | 433 | Investigation filter, script panel, triggers, agent spawning merged |

### Keyboard Shortcuts — NOT CENTRALIZED
3 separate `keydown` listeners with no coordination:
- `Header.tsx` (lines 110-117): Escape closes agent dropdown
- `Modal.tsx` (lines 18-28): Escape closes modal
- `Terminal.tsx` (lines 57-64): Escape closes terminal

Multiple listeners fire on single keystroke. No combo key support. Should extract to `useKeyboardShortcuts` hook.

### Loading State Patterns — 3 COMPETING APPROACHES
1. `LoadingSkeleton` component (270 lines)
2. Inline `<Loader2 className="animate-spin" />` spinners
3. No loading indicator (component renders with null/undefined data)

No consistent rule for when to use which approach.

### Side-Effect Patterns
- **useSystemMonitor:** 5s polling (metrics), 60s polling (detailed). Uses `mountedRef` guard but inconsistently. No request dedup. **No AbortController**.
- **useInvestigationAgents:** 4s polling (active agents). Uses `agentsRef` to avoid stale closures. No spawn dedup.
- ~~**StatusIndicator:** 10s polling (health)~~ — **FIXED** (consolidated into useSystemMonitor)
- **AutomaticTriggersSection:** 1s cooldown countdown polling.
- **Only DiskDetailView** implements AbortController.

### Event Listener Inventory
| Component | Listener | Target | Cleanup |
|---|---|---|---|
| Header.tsx | keydown (Escape) | window | Yes |
| Modal.tsx | keydown (Escape) | window | Yes |
| Terminal.tsx | keydown (Escape) | window | Yes |
| Header.tsx | mousedown (via useClickOutside) | document | Yes |
| StatusIndicator.tsx | mousedown (via useClickOutside) | document | Yes |
| InvestigationScriptsPage.tsx | resize | window | Yes |

---

## 6. Changes Made (This Audit)

### Round 1 — Initial Fixes
1. **Created `shared/utils/formatters.ts`** — consolidated `formatBytes`, `formatMegabytes`, `formatPercentage`, `getUtilizationColor` from 3 files.
2. **Updated MetricCard.tsx** — imports formatters from shared module, removed 3 local duplicates.
3. **Updated MetricDetailViews.tsx** — imports formatters from shared module, removed 4 local duplicates.
4. **Updated InfrastructureMonitor.tsx** — imports `getUtilizationColor` from shared module, removed local duplicate.
5. **Deleted `App.css`** — dead Vite scaffold CSS (not imported anywhere).
6. **Deleted `index.css`** — dead Vite scaffold CSS (not imported anywhere).

### Round 2 — All Priority Actions Completed
7. **Deleted 7 unused types** from `types/ui.ts` (`ExpandableCardState`, `ScriptEditorData`, `MetricThresholds`, `WebSocketMessage`, `ThemeColors`, `ComponentSize`, `ComponentVariant`, `PerformanceMetrics`).
8. **Created `shared/hooks/useClickOutside.ts`** — shared hook used by Header.tsx and StatusIndicator.tsx; removed duplicate manual mousedown listeners.
9. **Added CSS tokens to `matrix-theme.css`** — overlay alpha variables (`--overlay-light` through `--overlay-solid`, `--color-error-bg`, `--color-error-border`), `.input-field` + `.input-label` classes.
10. **Replaced JS hover with CSS `:hover`** — added hover utility classes (`.hover-bg-dark`, `.hover-bg-accent`, etc.) and button classes (`.btn-retry`, `.btn-copy-error`, `.btn-reload`, `.btn-kill`, `.btn-terminate`). Removed all `onMouseEnter`/`onMouseLeave` handlers from 7 components.
11. **Consolidated StatusIndicator health polling** — enhanced `useSystemMonitor` with `SystemHealthStatus` type, `healthStatus`/`healthError` state, `toggleMonitoring`, and `refreshHealth`. Rewrote StatusIndicator.tsx to be props-driven (no self-polling). Updated Header.tsx and App.tsx to pass health data through.
12. **Extracted `useScriptExecution` hook** — moved `modalState`, `openScriptEditor`, `closeScriptEditor`, `closeScriptResults`, `executeScript`, and `saveScript` from App.tsx (~140 lines) into `features/investigations/hooks/useScriptExecution.ts`. App.tsx reduced from ~486 to ~341 lines.
13. **Created shared `<Modal>` and `<ModalHeader>` components** — `shared/components/Modal.tsx` provides Escape-key handling, backdrop-click-to-close, and consistent CSS class usage. Added `.modal-lg`, `.modal-md`, `.modal-sm` size variants to CSS. Updated ScriptEditorModal, ScriptResultsModal, and SystemSettingsModal to use the shared wrapper; SystemSettingsModal inputs now use `.input-field` class.

### Round 3 — Inline Style Migration + Component Extraction

**New CSS utility classes added to `matrix-theme.css`:**
- `.card-subtitle` — dim subtitle text (color-text-dim, letter-spacing 0.08em, font-size sm)
- `.detail-row` / `.detail-row-label` / `.detail-row-value` / `.detail-row-value-sm` — label/value pairs
- `.detail-grid` / `.detail-grid-sm` / `.detail-grid-md` / `.detail-grid-lg` — auto-fit grid layouts
- `.data-table` / `.data-table th` / `.data-table td` / `.data-table tr + tr` — table styling
- `.text-muted` / `.text-dim-sm` / `.text-dim-xs` — dim text helpers
- `.progress-bar` / `.progress-bar-lg` / `.progress-fill` — progress bar components
- `.section-card` — card with overlay-medium background
- `.icon-text` / `.icon-text-xs` — inline-flex icon+text pattern
- `.search-input` / `.search-input-wrapper` / `.search-input-icon` — search input styling
- `.badge` / `.badge-success` / `.badge-warning` / `.badge-error` / `.badge-info` — status badges

**New shared components:**
- `shared/components/DetailRow.tsx` — reusable label/value pair component
- `shared/components/SearchInput.tsx` — search input with icon wrapper
- `shared/components/StatusBadge.tsx` — status badge component
- `shared/components/ConfirmDialog.tsx` — shared confirmation dialog

14. **Split MetricDetailViews.tsx** (1412 lines) into 6 files:
    - `MetricDetailViews.tsx` — shared components (MetricDetailLayout, MetricLineChart), helper functions, interfaces, and barrel re-exports (~280 lines)
    - `CpuDetailView.tsx` — CPU detail view component
    - `MemoryDetailView.tsx` — memory detail view component
    - `NetworkDetailView.tsx` — network detail view component
    - `DiskDetailView.tsx` — disk detail view with local state and fetch logic (largest file)
    - `GpuDetailView.tsx` — GPU detail view component
    - All files use CSS classes (`card-subtitle`, `detail-row`, `detail-grid`, `data-table`, `text-muted`, `text-dim-xs`, `progress-bar`) instead of inline styles where applicable. Conditional colors (getUtilizationColor) and computed widths kept inline.

15. **Migrated InvestigationScriptsPage.tsx** — converted inline styles to CSS classes, extracted `ScriptListItem` component (`features/investigations/components/ScriptListItem.tsx`) to deduplicate script item rendering.

16. **Migrated InvestigationsSection.tsx** — extracted `SearchInput` shared component, added `useMemo` optimizations, converted inline styles to CSS classes.

17. **Migrated Header.tsx, ProcessMonitor.tsx, InfrastructureMonitor.tsx** — replaced inline styles with CSS classes, extracted `ConfirmDialog` shared component for ProcessMonitor kill confirmation.

18. **Migrated remaining files** — SystemSettingsModal, AutomaticTriggersSection, ScriptEditorModal, ScriptResultsModal, ReportsPanel, StatusIndicator converted to CSS classes.

### Round 4 — CSS Token Consolidation + Hook Extraction

**CSS tokens added to `matrix-theme.css`:**
- `--overlay-darker: rgba(0, 0, 0, 0.8)` — darker overlay for code editors
- `--color-warning-bg: rgba(255, 170, 0, 0.1)` — warning background tint
- `.pool-item-zombie` / `.pool-item-high-thread` / `.pool-item-leak` — process alert item variants

**Hardcoded `rgba()` replacements (~25 instances across 10 files):**
- `rgba(0, 0, 0, 0.2)` → `var(--overlay-light)` in LoadingSkeleton, InvestigationScriptsPanel
- `rgba(0, 0, 0, 0.3)` → `var(--overlay-medium)` in ScriptEditorModal, ScriptResultsModal, InvestigationsSection, AutomaticTriggersSection
- `rgba(0, 0, 0, 0.4)` → `var(--overlay-medium)` in InvestigationsSection
- `rgba(0, 0, 0, 0.5)` → `var(--overlay-heavy)` in LoadingSkeleton, AlertPanel, ErrorBoundary(3x), ReportsPanel, AutomaticTriggersSection, ProcessMonitor
- `rgba(0, 0, 0, 0.8)` → `var(--overlay-darker)` in ScriptEditorModal(3x), ScriptResultsModal
- `rgba(0, 0, 0, 0.9)` → `var(--overlay-solid)` in ErrorBoundary
- `rgba(255, 0, 64, ...)` → `var(--color-error-bg)` / `var(--color-error-border)` in ProcessMonitor, ScriptResultsModal
- `rgba(255, 170, 0, 0.1)` → `var(--color-warning-bg)` in ProcessMonitor, AutomaticTriggersSection

19. **Refactored ConfirmDialog to use Modal internally** — replaced duplicate overlay/close logic with `<Modal>` wrapper. Removed `.confirm-dialog-overlay` CSS class. Same public API (props unchanged).

20. **Extracted ProcessAlertItem component** (`features/monitoring/components/ProcessAlertItem.tsx`) — consolidated zombie/high-thread/leak-candidate card pattern from ProcessMonitor into shared component with CSS class variants.

21. **Deduplicated InvestigationScriptsPanel** — extracted shared `renderScriptsList()` helper to eliminate ~60 lines of duplicated script item JSX between embedded and panel modes. Now uses `ScriptListItem` component for all script rendering.

22. **Created shared `usePolling` hook** (`shared/hooks/usePolling.ts`) — replaces identical `setInterval` + `clearInterval` + mounted-ref pattern. Migrated:
    - `useSystemMonitor.ts` (2 intervals: 5s metrics, 60s detailed)
    - `useInvestigationAgents.ts` (4s agent status polling)
    - `AutomaticTriggersSection.tsx` (5s trigger refresh, 1s cooldown tick)

23. **Created shared `useApiCall` hook** (`shared/hooks/useApiCall.ts`) — consolidates fetch + error-parse + loading-state + AbortController pattern. Available for single-resource consumers to migrate incrementally.

---

## 7. Priority Actions (Audit #2)

### Immediate (Fix before production)

| # | Action | Impact | Effort |
|---|---|---|---|
| 1 | Fix setTimeout cleanup in SystemSettingsModal (line 97) | Bug fix — prevents unmounted state update | 15 min |
| 2 | Fix unmounted state updates in useSystemMonitor — expand `mountedRef` checks | Bug fix — prevents memory leaks | 1 hour |
| 3 | Add spawn dedup to useInvestigationAgents.spawnInvestigationAgent() | Bug fix — prevents double spawns | 30 min |
| 4 | Fix race condition in useInvestigationAgents.pollAgentStatuses() | Bug fix — prevents inconsistent state | 1 hour |

### Short-term (Next sprint)

| # | Action | Impact | Effort |
|---|---|---|---|
| 5 | Create API service layer (replace 29 ad-hoc fetch implementations) | CRITICAL — biggest single dedup win | 3-4 days |
| 6 | Split useSystemMonitor into 4-5 focused hooks | HIGH — reduces 482-line god hook | 5 hours |
| 7 | Split MetricCard.tsx (716 lines → 3 components) | HIGH — makes it testable | 3 hours |
| 8 | Extract AgentDropdown from Header.tsx (475 → 200 lines) | MEDIUM — makes Header testable | 2 hours |
| 9 | Adopt DetailRow component in 6+ locations (already exists, just unused) | MEDIUM — eliminates inline detail markup | 2 hours |
| 10 | Create `useKeyboardShortcuts` hook, consolidate 3 scattered listeners | MEDIUM — prevents keyboard conflicts | 2 hours |

### Medium-term (Next iteration)

| # | Action | Impact | Effort |
|---|---|---|---|
| 11 | Introduce AppContext or zustand for monitor state (eliminate prop drilling) | MEDIUM | 4 hours |
| 12 | Unify loading state pattern (skeleton vs spinner vs nothing) | MEDIUM | 4 hours |
| 13 | Parameterize 5 metric detail views into factory + config pattern | MEDIUM | 3 days |
| 14 | Consolidate formatters: move formatMbPerSecond, formatInteger to formatters.ts | LOW | 1 hour |
| 15 | Extract shared typeGuards.ts from response parsing duplicates | LOW | 1 hour |
| 16 | Migrate 550+ inline styles to CSS classes | LOW-MEDIUM | ongoing |
| 17 | Replace 15 remaining hardcoded colors with CSS tokens | LOW | 2 hours |
| 18 | Create Button/Input primitives with CVA variants | MEDIUM | 1 day |
| 19 | Modularize 1,426-line matrix-theme.css | LOW | 3 hours |
| 20 | Extract InvestigationScriptsPage state into `useScriptsLibrary` hook | MEDIUM | 3 hours |

### Estimated total effort to 9/10 coherence: ~8-10 days
