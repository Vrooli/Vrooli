# React Coherence Notes

## Last Updated
2026-04-06 (Phase 18 iter3)

## State Management Patterns

- **Global state**: None (no Redux, Zustand, or Context providers beyond React Query). Health status is fetched via React Query and shared by query key.
- **Server state / cache**: `@tanstack/react-query` for all API data (health, events, policies, subscriptions, violations). Polling intervals defined in `lib/constants.ts`.
- **Component-local state**: `useState` for UI state (filters, paused flag, event list, form state). SSE stream managed via `useEffect` + `useRef` in `StreamPage`.
- **Routing**: react-router-dom HashRouter with `Routes`/`Route` components. `NAV_ITEMS` in `lib/router.ts` is single source of truth for navigation. URL query params used for cross-page linking (Phase 18) and full filter persistence (Phase 18 iter3: StreamPage syncs `?type=&source=`, EventLogPage syncs `?type=&source=&cid=&limit=`).

## Design Token System (Phase 10 → Phase 18)

CSS custom properties defined in `:root` in `styles.css`:
- **Surface tokens**: `--surface-base`, `--surface-sidebar`, `--surface-elevated`, `--surface-overlay`, `--surface-inset`, `--surface-code`
- **Border tokens**: `--border-default`, `--border-subtle`
- **Text tokens**: `--text-primary`, `--text-secondary`, `--text-muted`, `--text-faint`, `--text-accent`, `--text-accent-light`
- **Status tokens**: `--status-healthy`, `--status-healthy-bright`, `--status-degraded`, `--status-unhealthy`, `--status-unhealthy-bright`, `--status-unknown`
- **Error tokens** (Phase 18): `--error-text`, `--error-icon`, `--error-bg`, `--error-border`, `--error-link`, `--error-link-hover`
- **Radius tokens**: `--radius-md`, `--radius-lg`, `--radius-xl`

Usage: `text-[var(--text-muted)]`, `bg-[var(--surface-elevated)]`, etc. via Tailwind arbitrary value syntax.

## Composite Components

- **`PageHeader`** — Standardized page header with icon, title, and optional right-aligned actions. Used by all pages.
- **`Panel`** — Standardized card/panel with `elevated` and `overlay` variants.
- **`StatCard`** — Metric display card wrapping Panel.
- **`StatusBadge`** (Phase 18) — On/off toggle badge with semantic token colors. Used by PoliciesPage, CircuitBreakerPage, SubscriptionsPage.
- **`ErrorAlert`** — API error display with categorized messages and optional retry. Uses error tokens.
- **`ErrorBoundary`** — React class component catching render errors. Uses error tokens.

## Duplication Audit

- **Page headers**: Consolidated into `PageHeader` composite.
- **Card/panel styling**: Consolidated into `Panel` composite.
- **Status badges**: Consolidated into `StatusBadge` (Phase 18, was duplicated 3x with identical inline styles).
- **Filter inputs**: `StreamPage` and `EventLogPage` both have inline filter inputs. Could extract to a shared `EventFilters` component if more pages need filtering.
- **API base resolution**: `API_BASE` and `API_ROOT` are defined once in `lib/api.ts`.
- **formatBytes/formatTimestamp/safeStringify**: Shared utilities in `lib/utils.ts`.

## Styling Patterns

- **Approach**: Tailwind CSS utility classes + CSS custom property design tokens.
- **Class merging**: `cn()` utility (clsx + tailwind-merge) used in Layout nav buttons, Panel, status indicators.
- **Theme**: Dark theme using semantic design tokens. Phase 18 migrated ~25 remaining hardcoded Tailwind color classes to semantic tokens.
- **CVA**: Used by `Button` primitive for variant management (default, outline).
- **INPUT_CLASS**: Shared input styling constant in `lib/constants.ts`, now using tokens instead of hardcoded colors.

## Token Coverage Status (Phase 18 iter3)

**100% design token coverage** — zero hardcoded Tailwind color classes remain in any component or page file.
- `button.tsx`: Migrated to `--btn-default-bg`, `--btn-default-text`, `--btn-default-hover-bg`, `--btn-outline-border`, `--btn-outline-text`, `--btn-outline-hover-bg`, `--btn-focus-ring` CSS custom properties.
- All surfaces, text, borders, status indicators, error states, and button variants use semantic tokens.

## Shared UI Components (Phase 18 iter2)

- **EmptyState** (`components/EmptyState.tsx`): Icon + title + description + optional action link. Used in PoliciesPage, SubscriptionsPage, ScenarioMetricsPage.
- **Spinner** (`components/Spinner.tsx`): Animated SVG spinner + label. Used in all 10 data-fetching pages.
- **StatusBadge** (`components/StatusBadge.tsx`): On/Off badge using semantic tokens. Used in PoliciesPage, CircuitBreakerPage, SubscriptionsPage.
- **PageHeader** (`components/PageHeader.tsx`): Reusable page header with title, subtitle, and action slot.
- **Panel** (`components/Panel.tsx`): Card/panel wrapper with consistent spacing and border.
- **StatCard** (`components/StatCard.tsx`): Metric card with icon, label, and value.
- **ErrorAlert** (`components/ErrorAlert.tsx`): Error display with retry button.
- **ErrorBoundary** (`components/ErrorBoundary.tsx`): React error boundary for crash recovery.

## Remaining Opportunities

1. **Extract EventFilters** — if a third page needs event filtering, extract from StreamPage/EventLogPage.
2. ~~**Loading skeletons** — replace "Loading..." text with skeleton components.~~ Done (Phase 18 iter2): Spinner component applied.
3. ~~**Button variant tokens** — Button CVA uses direct Tailwind colors; could map to design tokens for full theme flexibility.~~ Done (Phase 18 iter3): Button CVA now uses `--btn-*` CSS custom properties. Zero hardcoded colors remain.
