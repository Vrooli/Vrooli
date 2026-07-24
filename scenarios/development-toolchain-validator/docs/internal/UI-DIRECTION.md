# DTV UI — Visual Direction Brief

This brief precedes code per `ui-health` It captures intent, references, and constraints for the DTV UI revamp. The detailed implementation plan lives at `~/.vrooli/plans/dtv-ui-revamp-layered-architecture-design-system-and-navigation-contract.md`.

## Intent

Developer-tool aesthetic. Data-dense. Calm under load. Verdict grids are the dominant visual element — they must read clearly at a glance even on a 360w phone. The operator (a single developer) is checking on the state of skill/manifest convergence and drilling into individual run outcomes. The UI should feel closer to an observability console than a marketing site.

## References

- `scenarios/git-control-tower/ui/` — sticky `StatusHeader`, mobile `BottomNav`, `FileStatsBadges` dual-display pattern (`src/components/FileStatsBadges.tsx`), CVA badge variants in `src/components/ui/badge.tsx`.
- `scenarios/web-console/ui/` — token system at `src/styles.css` (`--wc-*`) + `tailwind.config.ts` semantic mapping; `useAppViewport` hook for mobile viewport height (`src/hooks/useAppViewport.ts`); Zustand store pattern; `SessionSidebar.tsx`; CVA+Slot button (`ui/components/ui/button.tsx`).
- `templates/scenarios/react-vite/ui/` — the template's flagship source for `ui/flow/navigation.json` + `routes.generated.ts` + React Router v6/7 wiring + responsive `Layout.tsx`.

## Default theme

Dark slate base. Verdict colors:

| Status            | Token                       | Hex (base)  | Use                          |
|-------------------|-----------------------------|-------------|------------------------------|
| Pass              | `--color-status-pass`       | emerald-400 | Clean convergence            |
| Stale             | `--color-status-stale`      | amber-400   | Manifest pinning drift       |
| Unexpected mutation | `--color-status-unexpected` | red-400     | Validation found surprises   |
| Failure (run-side) | `--color-status-failure`   | rose-500    | The validation run itself errored |
| Neutral / unknown | `--color-status-neutral`    | slate-400   | Not yet validated            |

Accent: cyan-400 for primary actions and active-nav indicators. Matches web-console; differentiates DTV from git-control-tower's emerald-heavy palette.

## Light theme

Ships in P0 with mirrored tokens via `:root[data-theme="light"]`. Final polish is incremental — dark is the default and the polished surface.

## Constraints

- en / ja / aR locales must render. Every user-facing string flows through `consts/strings.generated.ts`.
- No horizontal scroll at 360w. Sidebar collapses to drawer below `md`; bottom nav appears with `pb-safe`.
- `data-testid` selectors on every interactive element via `selectors.ts`.
- Keyboard navigation reaches every primary affordance. One central `useGlobalKeydown` per `ui-health`
- No raw color values in TSX surface markup. Tokens only.
- WCAG AA contrast for core text/status pairs in both themes.
- Preserve the existing i18n pipeline; cimode-default tests; cross-locale parity test.

## Scope

Layered structure (`src/shared/{theme,ui/primitives,ui/composites,components,hooks,stores,lib}` + `src/surfaces/`) + token system + 12 primitives + 13 composites + navigation contract (`ui/flow/navigation.json` + `routes.generated.ts`) + Goldens surface + Settings surface. Skills/Manifests/Tuple-detail surfaces are deferred until their backend Connect-RPC clients land.

Token-only refresh extends to layout: the placeholder centered card is replaced by a real AppShell with sidebar + top header + mobile bottom nav.
