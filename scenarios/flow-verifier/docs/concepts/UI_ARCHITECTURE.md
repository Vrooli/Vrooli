# UI Architecture

Companion to [`DESIGN.md`](../../DESIGN.md). DESIGN.md is the source of
truth for visual language; this file documents the **structural** shape
of the React app so a contributor can navigate it in minutes.

## Shell

Entry point: `ui/src/main.tsx`. It mounts `<ThemeProvider>`, the React
Query client, the i18n provider, the router, and a top-level
`<Profiler>` boundary.

The application shell is `ui/src/components/AppShell.tsx`. It owns:

- **Desktop layout**: resizable left `<Sidebar>` + main content. Sidebar
  width persists across reloads via `useResizablePanel` (storage key
  `flow-verifier.sidebar.width.v1`, range 260–480 px).
- **Mobile layout** (`< 768 px`): `<MobileHeader>` on top, `<MobileNav>`
  at the bottom, full-screen `<MobileDrawer>` for the same sidebar
  content. Safe-area insets honored via `.pt-safe` / `.pb-safe` helpers
  declared in `design-tokens.css`.
- **Persistent surfaces**: `<HealthPill>` (compact API health) and
  `<ThemeToggle>` (light/dark/system) live in the sidebar header.

The shell never wraps page-level content in a card. Cards are reserved
for repeated records (rows, run cards) per DESIGN.md.

## Routing

`ui/src/App.tsx` defines a single `<Routes>` table mounted under the
shell. All routes are lazy-loaded via `React.lazy` + `<Suspense>` and
wrapped per-route by `<ErrorBoundary>` so a render-time crash in one
route never dismounts the shell.

| Path | Page | Purpose |
|---|---|---|
| `/` | `pages/DashboardPage` | Health, recent runs, verification timeline |
| `/flows` | `pages/InventoryPage` | Browse / search / filter / sort all flows |
| `/flows/:flowId` | `features/flow-detail/FlowDetailPage` | Graph / Traces / History tabs |
| `/runs/:runId` | `features/run-detail/RunDetailPage` | Run metadata + counterexample tree |
| `/settings` | `pages/SettingsPage` | Theme, density, font scale, default root |
| `*` | `pages/NotFoundPage` | Fallback with home link |

`BrowserRouter` `basename` is normalized from `import.meta.env.BASE_URL`
so the SPA mounts cleanly behind any proxy path.

## Tokens & theming

`tailwind.theme.json` extends Tailwind with semantic colors backed by
CSS variables in `design-tokens.css`. Every component references tokens
like `bg-app-surface`, `text-app-foreground`, `border-app-border` —
hardcoded `slate-*`, `white/N`, `text-red-N`, `bg-black/N` literals are
forbidden in `src/` (enforced by token-sweep grep).

`ThemeProvider` watches `prefers-color-scheme` and mirrors the resolved
theme onto `<html>` as `class="dark"` plus `data-resolved-theme`. The
localStorage key `flow-verifier.theme.v1` is a first-paint cache only;
the backend `user_settings` row owns the canonical value.

## Settings persistence

`lib/preferences.ts` is the typed client for `/api/v1/settings`.
Settings are single-tenant (principal `'local'`). The Settings page
reads through `@tanstack/react-query` and writes optimistically via
`putSettings`. The same store powers the inventory filter defaults and
the theme.

## Cross-references

- [`DESIGN.md`](../../DESIGN.md) — visual contract.
- [`docs/concepts/ux-overview.md`](ux-overview.md) — route-by-route data
  dependencies and screenshot descriptions.
- [`docs/reference/api-endpoints.md`](../reference/api-endpoints.md) —
  JSON contract for every endpoint the UI consumes.
- `ui-health` skill (`prompt-manager skill read ui-health`)
  — scope-discipline rules every screen follows.
- `ui-health` skill — proxy basename + router slot expectations.
