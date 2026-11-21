# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](/scenarios/template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

This section records the intended `ui/` structure for the Nooch redesign and the
current state it replaces. The design language is [`../../DESIGN.md`](../../DESIGN.md);
surfaces and their composition are in [`EXPERIENCE.md`](EXPERIENCE.md) and
[`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) §4 and §6.

### Current state (2026-09-22 audit)

Nine routes with ~85 KB of TSX packed into ~570 physical lines; 261 raw Tailwind
palette classes against 7 token classes; page-level `useEffect` fetches although
`@tanstack/react-query` is installed; hard-coded English; UTC dates; no breakpoint
hook; no images; RCL `AppShell/2` in sidebar density with six items. Treat the
existing feature files as prototypes to be rebuilt, not extended. Details:
[`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) §3.

### Intended structure

```text
ui/src/
  app/            routes, providers (query client, i18n, appearance), error boundary
  layout/         AppShell configuration, header, PrimaryNavigation, bottom tabs, focused-cooking shell
  theme/          appearance provider + pre-paint boot script, token bridge, useBreakpoint
  components/     the shared components below (domain-neutral presentation)
  features/
    today/  week/  meals/  explore/  recipe/  cooking/
    groceries/  kitchen/  onboarding/  settings/
  media/          presentation chooser client, asset manifest types, <MealImage>
  api/            generated Connect clients + typed query/mutation hooks per domain
  i18n/           catalogs and formatters (local dates, quantities, money)
  offline/        outbox (IndexedDB), replay, pending-state hooks
```

- **Routes** follow R03.1: `/today`, `/week`, `/meals`, `/meals/explore`,
  `/meals/:id`, `/cook/:sessionId`, `/groceries`, `/kitchen` (On hand, Equipment,
  Preferences as tab state), `/setup`, `/settings`. `/nutrition` and `/transfer`
  retire (D-031). URL state carries only harmless UI context (week, view, tab);
  Back restores filters, scroll, date, and slot.
- **Shell:** the RCL `AppShell/2` is kept only if it can render the header
  navigation and five-tab phone bar of `DESIGN.md`; otherwise configure or
  contribute the needed variant through the library's draft workflow rather than
  forking chrome locally. Cooking renders a focused shell without destinations.
- **Shared components (R05.3):** AppShell, PrimaryNavigation, PageHeader,
  SegmentedControl, FilterChip, MealCard, MealHero, WeekSelector, MealSlotCard,
  ServingControl, EvidenceValue, IngredientRow, ShoppingRow, InventoryRow,
  EquipmentTile, EquipmentScene, RecipeMap, CookingStep, PersistentTimer,
  EmptyState, SaveStatus, ImpactPreview, EntityEditor, SourceDetails,
  GenerationJobStatus. Built once; used by every surface; each has preview states
  for long names, missing photos, partial numbers, blocked actions, and both
  appearances. Library-versus-local ownership is tracked in
  [`../reference/component-library-gaps.md`](../reference/component-library-gaps.md).
- **Composition by medium:** `useBreakpoint` (reuse a library hook if one exists)
  chooses distinct component trees for Today, Week, Cooking, Groceries, and
  Equipment; collection grids and forms reflow with CSS. Decisions use container
  width bands from `DESIGN.md`.
- **Server state:** react-query hooks per domain with query keys that include the
  relevant entity revisions; mutations carry idempotency keys and expected
  revisions and reconcile the returned revision (R22). The workspace resolves once
  per session (fixes B11).
- **Appearance:** a pre-paint script reads the stored preference and sets the
  resolved appearance before React mounts; the provider persists changes locally
  and to the account (AT-002).
- **Media:** `<MealImage>` renders a presentation descriptor from the API, reserves
  dimensions, uses `srcset` renditions for the current appearance and composition
  only, lazy-loads below the fold, and falls back through the treatment chain on
  error without moving controls.
- **Strings, dates, test ids:** every string via i18n keys; one local-date helper
  bound to the workspace timezone; every experience-contract element renders its
  bound `data-testid` from `experience/pages/*.json`.
- **Assets:** curated artwork lives under the UI public assets root in
  `assets/scenes/`, `assets/meals/`, `assets/equipment/`, `assets/ingredients/`
  with a versioned manifest whose ids are independent of filenames (R19.2,
  [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) §7).
