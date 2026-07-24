# UI-Health Professional UI Rebuild Plan

**Date:** 2026-05-20
**Scope:** Replace the template-baseline UI in `scenarios/ui-health/ui/` with a professional, polished UI for the validation / search / reindex / inventory domains.
**Reference shapes:** `scenarios/flow-verifier/ui/` (target shape), `scenarios/git-control-tower/ui/` (polish patterns to selectively borrow).

---

## Required Reading

Executing agent must load before starting:

```
prompt-manager skill read ux ui-health test seam-discovery-and-enforcement
```

Then read in this repo:
- `scenarios/ui-health/PRD.md`
- `scenarios/ui-health/DESIGN.md`
- `scenarios/ui-health/docs/internal/SEAMS.md`
- `scenarios/ui-health/docs/internal/TESTING.md`
- `scenarios/flow-verifier/ui/src/App.tsx`
- `scenarios/flow-verifier/ui/src/routes.generated.ts`
- `scenarios/flow-verifier/ui/flow/navigation.json`
- `scenarios/git-control-tower/ui/src/components/` (skim for primitive patterns: badges, modals, status pills)

---

## Greenfield Declaration

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. The existing template scaffolding in `scenarios/ui-health/ui/src/` is fair game to delete or rewrite. `features/health/HealthCard.tsx` is the only piece worth carrying forward — its health-status logic moves into the TopBar pill. **`features/notes/*` is template boilerplate with no role in ui-health and must be fully removed**, along with `pages/NotesPage.tsx`, any notes-related routes, i18n keys, tests, and selectors. Notes also has API/proto scaffolding (`api/handlers/notes/`, `packages/proto/schemas/ui-health/v1/notes/`, generated TS/Go clients) — flag for removal in a follow-up; this plan removes UI-side only and leaves the unused backend until a separate cleanup pass.

---

## Goals & Operational Targets

From `PRD.md` and `DESIGN.md`:

- **Domain surfaces:** validation results (manifest drift, missing slots, contracts), semantic search over UI surfaces, reindex job control, scenario/surface inventory.
- **Look:** Vrooli Operational Console — dense, calm, technical. Slate neutrals; blue (`#2563eb`) primary; cyan/green/amber/red semantic colors. Inter body, JetBrains Mono code.
- **A11y:** WCAG 2.1 AA, full keyboard nav, screen-reader friendly, ≥44px touch targets.
- **Responsive:** Desktop split with right-side InspectorPanel; mobile collapses sidebar → drawer + bottom nav, inspector becomes full-screen route.
- **Testability:** Stable `data-testid` selectors via a `ui/src/constants/selectors.ts` registry (per the ux skill).

---

## Approach (selected by user)

1. **Shape:** Flow-verifier (lazy pages, generated routes, feature folders, InspectorPanel).
2. **Routes:** Dashboard, Validation, Search, Inventory & Reindex (+ Settings, NotFound).
3. **Design system:** Bare CSS + `design-tokens.css` + hand-rolled primitives in `components/ui/`.
4. **Sequence:** Scaffold first, then features end-to-end.

---

## Current Technical Context

- API: Connect-RPC services under `packages/proto/schemas/ui-health/v1/{validation,search,reindex,inventory,contracts}` with Go handlers in `scenarios/ui-health/api/handlers/`. **Reuse generated TS clients in `packages/proto/gen/typescript/ui-health/v1/`** — do not hand-roll fetch wrappers.
- API port: resolved via lifecycle; `/health` + `/api/v1/health` exposed (see `.vrooli/endpoints.json`).
- Existing UI: 36 .tsx files, mostly template. Custom = `features/health/HealthCard.tsx`, `features/notes/{NotesCard,AttachmentUpload}.tsx`. Routes hardcoded in `app/routes.tsx`.
- Tokens: `ui/src/design-tokens.css` already present and matches DESIGN.md.
- CLI surface for deterministic ops: `vrooli scenario {start,stop,restart,test,logs} ui-health`. Validation/search/reindex CLI verbs live in `scenarios/ui-health/cli/` — assume reuse; add new verbs only if a search across `cli-health search "<op>"` (e.g. `cli-health search "ui-health validate"`) returns no hit and the UI genuinely needs a new programmatic surface.

---

## Phase 0 — Prep

- Confirm generated TS Connect clients exist for all services; if missing, run the proto codegen first (don't proceed without typed clients).
- Extract the health-status logic from `features/health/HealthCard.tsx` into the new TopBar pill component, then delete the old file.
- Delete the entire notes surface: `features/notes/`, `pages/NotesPage.tsx`, notes routes, notes i18n keys, notes test files, notes selectors. Grep for `notes` after deletion to confirm zero residual references in `ui/src/`.
- Delete the placeholder `pages/{Dashboard,Settings}Page.tsx` once their replacements are ready.

---

## Phase 1 — Navigation source + route generation

Borrow flow-verifier's data-driven routing.

1. Create `scenarios/ui-health/ui/flow/navigation.json` declaring routes:
   - `dashboard` → `/`
   - `validation` → `/validation`, `validation_detail` → `/validation/:scenarioId`
   - `search` → `/search` (query in URL params)
   - `inventory` → `/inventory`, `surface_detail` → `/inventory/:surfaceId`
   - `reindex` → `/reindex` (job list + trigger), `reindex_job` → `/reindex/:jobId`
   - `settings` → `/settings`
   - `not_found` → `*`
2. Add a generator step that produces `ui/src/routes.generated.ts` (typed `ROUTE_PATTERNS`, path builders). Mirror `scenarios/flow-verifier/ui/src/routes.generated.ts` exactly in shape.
3. Wire generation into `make build` / scenario lifecycle so the file regenerates from `navigation.json`.

---

## Phase 2 — Shell, theme, and InspectorPanel

Rewrite `App.tsx`, `app/routes.tsx`, and `layout/`:

- `App.tsx` ≤80 lines: `Providers > Router > AppShell > <Routes>` with lazy-loaded page modules behind `Suspense` + a `RouteSkeleton` fallback (copy from flow-verifier).
- `layout/AppShell.tsx`: desktop grid = sidebar (collapsible) + main + optional right InspectorPanel; mobile = TopBar + main + BottomNav, sidebar → drawer.
- `layout/TopBar.tsx`: brand, breadcrumb, theme toggle, health pill (consume `features/health/HealthCard` logic), global search shortcut (`⌘K` opens search).
- `layout/Sidebar.tsx` / `layout/MobileNav.tsx` / `layout/BottomNav.tsx`: driven by `navigation.json`-derived nav config.
- `components/InspectorPanel.tsx`: slot-based right pane with header, body, close affordance, responsive collapse-to-route on mobile.
- Theme: light/dark/system via `data-theme` attribute on `<html>`; persist preference; respect `prefers-color-scheme`.

---

## Phase 3 — Shared UI primitives (`components/ui/`)

Build only what features need; do not pre-build a kitchen-sink library. Initial set:

- `Button.tsx` (primary/secondary/ghost/danger, loading state)
- `Card.tsx`, `CardHeader.tsx`, `CardBody.tsx`
- `Badge.tsx` (status: ok/warn/error/info/neutral)
- `StatusPill.tsx` (icon + label, for health/job states)
- `Table.tsx` (sortable, sticky header, empty/loading/error states)
- `EmptyState.tsx` (icon + heading + body + CTA — required for every list page)
- `Skeleton.tsx` + `RouteSkeleton.tsx`
- `Modal.tsx` + `ConfirmDialog.tsx` (focus trap, ESC, backdrop)
- `Toast.tsx` + `ToastHost.tsx` (port flow-verifier's pattern)
- `SearchInput.tsx`, `Select.tsx`, `Tabs.tsx`
- `CodeBlock.tsx` (JetBrains Mono, line numbers, copy button — for surface previews / manifest snippets)
- `ProgressBar.tsx` (reindex job progress)
- `Icon.tsx` — wrapper around **Lucide** icons (per ux skill: icons not emojis)

All primitives:
- Take `data-testid` props through a typed selector registry (`ui/src/constants/selectors.ts`).
- Use design-tokens CSS variables only — no hard-coded colors.
- Include `.test.tsx` + `.a11y.test.tsx` from the start.

---

## Phase 4 — Feature: Validation

`features/validation/`:

- `ValidationListPage.tsx` — table of scenarios with columns: name, manifest status, drift, missing slots count, last validated. Filter chips by status. Row click opens detail in InspectorPanel (desktop) or navigates (mobile).
- `ValidationDetailPanel.tsx` — manifest summary, drift section (diff via `CodeBlock`), missing-slot list with file paths (per ux skill: paths break-anywhere to prevent overflow), contracts/provenance section showing `ComponentProvenance` data.
- `useValidation.ts` — Connect-RPC client hook (TanStack Query) for ListValidation, GetValidation, RevalidateScenario.
- Empty state: "No scenarios validated yet — run `vrooli scenario validate ui-health` or trigger from Reindex."
- Tests: unit (hook + page render), a11y, BAS selector workflow validating the list + detail flow.

---

## Phase 5 — Feature: Search

`features/search/`:

- `SearchPage.tsx` — large `SearchInput` (autofocus, `⌘K`-bindable), filter chips (kind: component/page/feature/hook; scenario), ranked results list with snippet + provenance badge + score. Cursor pagination.
- `ResultCard.tsx` — title, snippet (highlight matches), scenario badge, provenance pill (CUSTOM / ADOPTED_UNMODIFIED / ADOPTED_MODIFIED / UNKNOWN), score, "Open in Inventory" link.
- `useSearch.ts` — debounced query (250ms), URL-synced (`?q=…&kind=…`), Connect-RPC SearchService.
- Empty / no-results / error states all distinct (per ux skill).
- Tests: unit + a11y + BAS workflow exercising query → result → drill-in.

---

## Phase 6 — Feature: Inventory & Reindex

`features/inventory/`:

- `InventoryPage.tsx` — filterable surface list (kind, scenario, provenance). Switch to grouped-by-scenario view.
- `SurfaceDetailPanel.tsx` — surface metadata, file path, provenance details, source preview (`CodeBlock`), related surfaces.

`features/reindex/`:

- `ReindexPage.tsx` — "Trigger reindex" primary button (with scenario scope selector), live job list (status: queued/running/done/failed) with ProgressBar.
- `JobDetailPanel.tsx` — job log tail (SSE or polling), retry/cancel actions.
- `useReindexJobs.ts` — Connect-RPC ReindexService; subscribe to job updates.
- Confirmation modal before destructive reindex of all scenarios.

---

## Phase 7 — Dashboard composition

`pages/DashboardPage.tsx`:

- Top row: 3 stat cards (Scenarios validated, Surfaces indexed, Open issues).
- Activity feed: recent reindex jobs + recent validation runs (merged, time-sorted).
- Quick actions: "Search surfaces" → SearchPage, "Validate a scenario" → Validation, "Reindex now" → Reindex.
- API status detail card (expands the TopBar pill).
- All sourced from existing endpoints; do not invent new ones.

---

## Phase 8 — i18n, a11y, responsive sweep

- Move every user-visible string into `i18n/locales.ts`.
- Run keyboard-only walkthrough of every page; ensure focus rings, skip-to-content, ARIA labels on icon buttons.
- Mobile sweep per `ux` skill §5: verify all pages in a 360×640 viewport with virtual keyboard simulated; check overflow on long paths/scenario IDs; verify ≥44px touch targets; consolidate desktop headers on mobile.
- Lighthouse: target ≥95 a11y, ≥90 best-practices on every route. Record results in `docs/internal/PERFORMANCE.md`.

---

## Phase 9 — Tests & validation

- **Unit:** vitest for every component and hook; coverage gate ≥80% for `features/*` and `components/ui/*`.
- **A11y:** `.a11y.test.tsx` per page using `@axe-core/react` assertions.
- **BAS workflows:** Add scenario workflows under `scenarios/ui-health/bas/` covering: dashboard load, search query→result→inventory, validation list→detail, reindex trigger→completion. Use selector registry.
- **Selectors:** `ui/src/constants/selectors.ts` registry; all new tests reference it (never hardcoded strings).

---

## Final: Cleanup & Health Verification

This is mandatory. Even if some issues appear pre-existing, fix them:

1. `cd scenarios/ui-health/ui && npm run lint` — fix **all** warnings/errors in modified files.
2. `cd scenarios/ui-health/ui && npx tsc --noEmit` — fix **all** type errors.
3. `cd scenarios/ui-health/ui && npm test -- --run` — all tests green.
4. `cd scenarios/ui-health/api && go build ./... && go test ./... -timeout 300s` — green (we shouldn't be touching API, but verify nothing broke through proto regen).
5. `gofumpt -w` and `golangci-lint run` on any Go files touched.
6. `vrooli scenario restart ui-health`.
7. Verify health: `vrooli scenario status ui-health` and `curl -s http://localhost:<API_PORT>/health` returns `"readiness":true`.
8. Open the UI in a browser and walk through Dashboard → Validation → Search → Inventory → Reindex → Settings. Confirm no console errors, no layout overflow on a mobile-sized viewport, theme toggle works, and InspectorPanel opens/closes correctly.
9. Run the BAS workflows from Phase 9 against the live scenario; all must pass.
10. Update `docs/internal/SEAMS.md` with any new UI seams added; update `docs/manifest.json` if new user-facing docs were created.

---

## Out of Scope

- New API endpoints (work strictly against existing proto services).
- Adding shadcn/ui, MUI, or any new UI dependency beyond Lucide icons (already permitted by ux skill).
- Removing notes API/proto/CLI scaffolding — UI removal only here; backend notes cleanup is a separate follow-up task.
- Multi-language i18n content beyond English baseline keys (structure only).

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Connect-RPC TS clients missing or stale | Phase 0 gate — regenerate before any feature work |
| Scope creep into API/proto changes | Out-of-scope clause; if a UI requirement needs a new endpoint, stop and surface to user |
| Visual drift from DESIGN.md tokens | Hard rule: no hex literals in feature code; only `var(--token)` |
| Mobile regressions | Phase 8 sweep + BAS workflows run in both viewport sizes |
| Test flake on reindex job streaming | Use deterministic mock job IDs in tests; real streaming only validated in BAS workflow |
