# react-vite template divergence report — 2026-07-14

Reviewed deliverable from the fleet drift reconverge (RCL trust-hardening plan,
phase 6). The six adoption records under `templates/scenarios/react-vite` are
flagged `local=modified`, so the batch reconverge **did not** and **must not**
overwrite them — `templates/**` is out of the write boundary. This report is the
per-file review of what each local modification actually is and whether the
library should absorb it.

## How this was produced

For each modified copy, the on-disk file and its adopted library version were
diffed with the provenance/authoring header stripped from both, so the
comparison is body-only:

```
diff <(strip_header library/components/<C>/versions/<v>/<C>.tsx) \
     <(strip_header templates/scenarios/react-vite/ui/src/components/ui/<f>.tsx)
```

No file under `templates/**` was written.

## Summary

| Component | Adopted ver | Body vs library | Real local change | Upstream-worthy? | Disposition |
|-----------|-------------|-----------------|-------------------|------------------|-------------|
| StatusBadge | 1.1.0 | identical (¹) | none | n/a | header/whitespace artifact — safe to refresh |
| Select | 1.1.0 | identical (¹) | none | n/a | header/whitespace artifact — safe to refresh |
| Input | 1.1.0 | identical | none | n/a | header/whitespace artifact — safe to refresh |
| EmptyState | 1.1.0 | identical (¹) | none | n/a | header/whitespace artifact — safe to refresh |
| Card | 1.1.0 | identical (¹) | none | n/a | header/whitespace artifact — safe to refresh |
| DataTable | 1.1.0 (behind 1.1.2) | +`tableTestId` (²) | test-id + a11y/i18n hooks | **yes — already absorbed** | absorbed into library 1.1.1 + 1.1.2 this phase |

¹ The only body-level difference is a single trailing blank line present in the
library version and absent from the vendored copy. The component code is
byte-identical.

² The react-vite copy's own body delta vs library `1.1.0` is `tableTestId`. The
`searchableText`, `filterGroupLabel`, and `sortLabel` affordances were the same
DataTable `1.1.x` regression class surfaced by the template-manager and
experience-manager copies; all four are now in the library (1.1.1 + 1.1.2), so a
future reconverge of the react-vite copy to `1.1.2` lands clean.

## Per-file findings

### StatusBadge, Select, Input, EmptyState, Card (5 of 6)

The vendored bodies are identical to their adopted library `1.1.0` source. These
records read `local=modified` **not** because the code diverged but because the
on-disk file no longer matches the recorded apply snapshot — the difference is
confined to the provenance header block (and, for four of them, a trailing blank
line). There is **nothing template-specific to absorb**: the react-vite copies
are faithful library `1.1.0`.

Disposition: report-only (templates denied). These are safe to reconverge/refresh
whenever `templates/**` writes are in scope; a reconverge would simply re-stamp
the provenance header and clear the `modified` flag. No library change warranted.

### DataTable (6 of 6) — upstream-worthy, already absorbed

The vendored DataTable copy adds a generic, reusable prop the library `1.1.0`
source lacked:

```tsx
export interface DataTableProps<Row> {
  ...
  tableTestId?: string;          // added by the react-vite copy
}
// ...
<table data-testid={tableTestId} ...>   // applied to the rendered table
```

This is a test hook, not template-specific behavior — every adopter benefits from
being able to target the table without coupling to markup. Its absence in
`1.1.0` was an adopter-facing regression (the same prop existed in `1.0.0`).

**Disposition: absorbed upstream this phase.** The DataTable `1.1.x` line had
dropped four generic, reusable adopter-facing affordances that `1.0.0`-era copies
carried. All four were restored in the library and released across two version
cuts during phase 6:

- **v1.1.1** — `tableTestId` (`data-testid` passthrough on the `<table>`) and the
  `searchableText` node-search fallback (search a column by its rendered content
  when no explicit `searchValue` is given).
- **v1.1.2** — `filterGroupLabel` (`aria-label` for the filter button group,
  overriding `filterLabel`) and `sortLabel` (`(header) => string`, the `aria-label`
  formatter for each sortable column's sort button, default `` `Sort by ${header}` ``).
  These are i18n/a11y affordances surfaced when experience-manager's `FleetPage.tsx`
  was reconverged.

The scenario adopters (template-manager, cleanup-manager, react-component-library's
own copy, and experience-manager) were reconverged to `1.1.2` and are green. The
react-vite copy remains `behind 1.1.2 / modified` only because `templates/**` is
out of the write boundary; once templates writes are in scope, reconverging it to
`1.1.2` yields a clean result because its local enhancements are now the library's
own.

## Recommended follow-ups (out of phase-6 boundary)

1. Reconverge the six react-vite copies to current under a change that is allowed
   to write `templates/**`. Five re-stamp cleanly; DataTable lands on `1.1.2`
   with its own enhancements, so it converges clean.
2. The `components version-create` CLI dropped `designStyles`/`draft` from
   `component.json` when it round-tripped the file — observed on **both** the
   1.1.1 and 1.1.2 cuts this phase; manually restored each time. Filed as a bug
   (plan ledger `c71f56c0`). Worth a defect fix so future version cuts do not
   silently strip component metadata.

## Addendum 2026-07-15 — follow-ups closed

Both follow-ups above are done:

1. **Template reconverged.** DataTable's four EM mobile-floor affordances
   (44px `min-h-11` tap targets on filter chips and sort buttons, `table-fixed`
   layout, `break-words` cells) were absorbed upstream as **DataTable 1.2.0**,
   and all five adopters (template-manager, cleanup-manager, experience-manager,
   react-component-library, and the react-vite template copy) now sit at
   `1.2.0 current/clean`. The five header-artifact react-vite copies
   (StatusBadge, Select, Input, EmptyState, Card) were re-stamped clean at their
   current versions. The reviewed-divergence allowlist entry for data-table was
   removed; `TestReactViteTemplateVendoredComponentsMatchCatalogLatest` passes
   with an empty allowlist. Reapplying into the template required teaching the
   style-fit (`components.FSServiceJSONReader`) and dependency
   (`deps.FSPackageJSONReader`) readers the `../templates/scenarios/<id>`
   scenario-key form (traversal-guarded, unit-tested).
2. **`components version-create` metadata strip fixed** (separate bug-fix
   session): `designStyles`/`fileSlots` survive the round-trip; verified on the
   1.2.0 cut.

Remaining known local divergence fleet-wide: template-manager's `bottom-nav.tsx`
(`pb-[max(env(safe-area-inset-bottom,0px),1.75rem)]` floor instead of `pb-safe`)
— honest `local=modified`, a candidate for a future BottomNav absorption.
