# Component Library Gaps

## Purpose Of This Document

Record what Nutrition Planner needed from `react-component-library` (RCL)
and could not link as-is. Every ejection and every missing asset is a debt the
library can pay back only if it is written down here; a silent local fork is
not. Keep this file short and factual, and raise each entry with the library
(`scenarios/react-component-library/docs/reference/`).

**Current adoption (audit 2026-09-22).** The UI imports eight RCL assets:
`AppShell/2`, `PageHeader/2`, `Select/1`, `SettingsList/1`, `Button/2`, and the
unused `HealthCard/0` and `ExperienceSurface/1`. Feature pages are raw HTML with
hard-coded Tailwind palette classes and no RCL components. The v2.0
redesign (spec R05.3) asks for one shared component set built once; this file
tracks which of those come from the library and which are local.

### Redesign shared components — library candidates (to verify)

The candidates below are asset names **found in the library's package exports
on 2026-09-22**; whether each fits the Nooch design (tokens, anatomy, both
appearances, phone composition) is **not yet verified**. When a component is
adopted, built locally, or ejected, move it to the right table below.
Nutrition-specific business components always stay in this scenario.

| Nooch component (R05.3) | Library candidate(s) found | Expected home |
|---|---|---|
| AppShell, PrimaryNavigation | `AppShell`, `AppNavigation`, `NavLink`, `BottomNav` | Library, configured for a header-nav desktop shell and five labelled bottom tabs; to verify |
| PageHeader | `PageHeader`, `Heading` | Library; to verify serif title support through tokens |
| SegmentedControl (mode switches) | `ButtonGroup`, `SelectionControl`, `Tabs` | To verify; no asset named SegmentedControl exists |
| In-page tabs | `Tabs`, `ScrollableTabs` | Library; to verify underline style (D-027) |
| FilterChip | `Chip`, `FilterBar` | Library; to verify selected-token behavior |
| Search | `SearchInput` | Library; to verify |
| MealCard, MealHero, MealSlotCard | `Card`, `CardShell`, `CardGrid`, `ProgressiveImage`, `AspectRatio` as building blocks | Local composition (nutrition-specific) |
| WeekSelector | — | Local |
| ServingControl | `NumberField` | Local wrapper over a library field; to verify |
| EvidenceValue | `ProvenanceInk`, `Badge`, `StatusBadge` | Local (unknown-vs-zero semantics are domain rules) |
| IngredientRow, ShoppingRow, InventoryRow | `Checkbox`, `List`, `DataTable`, `SwipeActions` as building blocks | Local |
| EquipmentTile, EquipmentScene | `Toggle`, `SelectionControl` as building blocks | Local |
| RecipeMap | `Table`, `ScrollArea` | Local |
| CookingStep, PersistentTimer | `Progress`, `RollingNumber`, `LiveAnnouncer` | Local |
| EmptyState | `EmptyState`, `ErrorState`, `LoadingState`, `OfflineState` | Library; to verify |
| SaveStatus | `PendingSyncBadge`, `Toast`, `ToastManager`, `UndoBanner` | Library; to verify |
| ImpactPreview | `ConflictResolutionFlow`, `ResponsiveDialog` as building blocks | Local composition |
| EntityEditor | `Form`, `FormSection`, `FormField`, `FormActions`, `UnsavedChangesFlow`, `DirtyStateGuard` | Library building blocks; to verify |
| SourceDetails | `DescriptionList`, `CollapsibleRegion` | Local composition |
| GenerationJobStatus | `Progress`, `StatusBadge`, `BudgetBar` | Local composition |
| Sheets and dialogs | `BottomSheet`, `ResponsiveDialog`, `ResponsivePanel`, `Dialog` | Library; to verify keyboard-safe sticky actions (R06.4) |
| Breakpoint hook | `useMediaQuery`, `useViewportEnvironment` | Library hook preferred over a local `useBreakpoint` (D-036); to verify SSR safety |
| Motion and accessibility hooks | `useReducedMotion`, `useFocusTrap`, `useFocusReturn`, `useNetworkStatus` | Library; to verify |

Before building a shared component locally, check the library first
(`react-component-library` CLI and its docs). To change a library asset, use
`react-component-library components draft-begin <asset>`; never edit a release
directory.

## Ejections

Components copied into this scenario with `react-component-library adoptions
eject <component-id> nutrition-planner --reason "..."`. One row per ejection.

| Component | Version | Reason | Revisit when |
|---|---|---|---|
| _none yet_ | | | |

## Missing Assets

Surfaces this scenario built locally because no library asset fit. Name the
shape the library would need, not the local file.

| Need | Nearest library asset | Why it did not fit |
|---|---|---|
| _none recorded yet_ — the redesign will add rows here as local components are built | | |

## Cross-References

- [choosing-ui.md](../guides/choosing-ui.md) — when to eject and when to record a gap
- [UI-ARCHITECTURE.md](../concepts/UI-ARCHITECTURE.md) — where local components live
- [REDESIGN_PLAN.md](../internal/REDESIGN_PLAN.md) — section 4 (engineering bar) lists the shared components
- [DESIGN.md](../../DESIGN.md) — tokens and the control language every component must follow
