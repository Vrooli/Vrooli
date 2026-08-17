# Asset graph workspace — measured brief

> Companion to `2026-08-13-asset-graph-workspace.html` in this folder. The HTML
> is the visual version with UI mockups (open it in a browser); this file carries
> the same facts as terminal-readable text. Both were produced by the
> investigation that authored plan
> `react-component-library-asset-graph-workspace`.
>
> Every figure was measured on 2026-08-13 against
> `scenarios/react-component-library/catalog/assets/**`,
> `scenarios/react-component-library/library/**`, and the working tree on branch
> `agi`.

## The question this brief answers

Is the "hierarchy" of react-component-library assets — tokens, then primitives,
then components, then sections, then pages — a conceptual model we wrote down,
or is it encoded in code? And if it is encoded, what consumes it?

**Answer: it is encoded and enforced, and almost nothing consumes it.** The
catalog validates a six-rung dependency ladder across 410 assets and 848 edges
on every gate run. No UI surface, CLI command, or report renders any of it.

## Measured baseline

| Fact | Value |
|---|---|
| Catalog assets (`catalog/assets/**/*.json`) | 410 |
| `requires` edges | 848 |
| `suggests` edges | 60 |
| Assets declaring at least one `requires` | 366 |
| Assets declaring no dependencies | 44 |
| Rank rungs enforced by `rank()` | 6 (0–5) |
| Domains declared in `catalog/config.json` | 20, with `order` 10–200 |
| Go code that parses `domains[]` | none — `catalogConfig` has only `Gates` |
| Library version manifests (`library/**/component.json`) | 193 |
| Version manifests declaring `dependencies[]` | 72 |
| Manifest dependency edges | 143 |
| Version manifests pinning nothing | 121 |
| Reconciliation between the two graphs | none exists |
| Experience specs (`experience/components/*.json`) | 30 |
| Canonical `experience-contract.json` files in `library/**` | 182 |
| Experience specs using `extends` | 0 |
| Port-facet capabilities in the registry | 23 |
| Promises-facet capabilities | 59 |
| Catalog assets declaring `satisfies` | 23 |
| Asset detail tabs today | 7 (no relationships/graph tab) |
| Sidebar tree depth today | 1 level, grouped by `slot \|\| category`, `localeCompare` |
| Library manifests declaring a `slot` | 42 of 193 |

## The enforced ladder

`api/internal/catalogvalidate/validate.go:259`:

```go
func rank(kind string) int {
	return map[string]int{
		"foundation": 0,
		"runtime-hook": 1, "runtime-service": 1, "adapter": 1, "generator": 1,
		"primitive": 2,
		"component": 3,
		"pattern": 4, "navigation": 4,
		"page-template": 5,
		"fixture": -1,
	}[kind]
}
```

`validate.go:210` rejects any `requires` edge pointing at a higher rank with
finding code `catalog.dependency_rank`. `cycleFindings` rejects cycles with
`catalog.requires_cycle`. Both are `severity: error` and both run today.

Population by rung:

```
rung 5  page-template                                    21
rung 4  pattern (29) + navigation (11)                   40
rung 3  component                                       209
rung 2  primitive                                        35
rung 1  runtime-hook (38) + adapter (17)
        + generator (16) + runtime-service (16)          87
rung 0  foundation                                         9
        fixture (rank -1)                                  9
                                                        ---
                                                         410
```

**Known defect in the ladder.** `rank()` is a bare map lookup with no default.
Go returns the zero value for a missing key, so an asset whose `kind` is
misspelled, or a kind added later without updating the map, ranks 0 —
foundation — and silently passes every downward check. Guard this before
building surfaces that trust the ladder.

## Blast radius — transitive dependents

Computed over the 848-edge `requires` graph:

| Asset | Kind | Direct dependents | Transitive dependents |
|---|---|---|---|
| `foundations.tokens` | foundation | 90 | 303 |
| `hooks.use-reduced-motion` | runtime-hook | 22 | 187 |
| `foundations.contracts` | foundation | 22 | 155 |
| `primitives.text` | primitive | 18 | 127 |
| `foundations.icon-registry` | foundation | — | 119 |
| `primitives.icon` | primitive | — | 118 |
| `hooks.use-controllable-state` | runtime-hook | 18 | 108 |
| `primitives.status-indicator` | component | 21 | 99 |
| `feedback.async-boundary` | component | 28 | — |

303 of 410 assets sit downstream of `foundations.tokens`. Nothing in the product
says so.

## Worked example — `data-display.data-table`

Closure of 28 assets (root included), stratified by rung:

```
rung 3  11  data-table(root) table async-boundary empty-state error-state
            loading-state offline-state permission-state button pressable
            status-indicator
rung 2   7  icon scroll-area skeleton slot spinner stack text
rung 1   7  use-announce use-focus-visible use-network-status use-press
            use-reduced-motion create-scoped-store selection-store
rung 0   3  contracts icon-registry tokens
```

Upward: 5 direct dependents (`data-grid`, `resource-collection`,
`import-workflow`, `collection-page`, `data-table-generator`), 9 transitive.

The catalog declares `requires: [table, selection-store, async-boundary]`. The
implementation manifest `library/components/DataTable/component.json` pins
`[Table@1.0.0, AsyncBoundary@1.0.0]`. `selection-store` is declared and not
pinned. Nothing detects that.

## What already consumes the hierarchy

| Consumer | Location | What it uses |
|---|---|---|
| Gate applicability | `catalog/config.json` `gates[].appliesTo` | `asset.kind` selects which gates apply |
| Rank + cycle validation | `catalogvalidate/validate.go:210,259,263` | `rank()` over `requires` edges |
| Dependency closure | `components/dependency_closure.go` | manifest pins, dependencies-first order, cycle rejection |
| Adoption path resolution | `adoptions/pathresolver.go` | `slot` decides on-disk placement |
| Port precedence | `dependency_closure.go:45` | rank tiebreak — reachable, but the one production caller passes `adopted = nil` |

## What does not consume it

| Surface | Current behavior | Location |
|---|---|---|
| Catalog sidebar | one level, `slot \|\| category`, alphabetical | `ui/src/features/catalog/CatalogBrowser.tsx:230-237` |
| Asset detail tabs | preview, overview, files, tests, versions, progression, adoptions | `ui/src/routes.ts` |
| Asset overview | flat list of direct manifest dependencies, one hop, downward only | `ui/src/pages/ComponentDetailPage.tsx:437` |
| Reverse edges | none anywhere in the product | — |
| Dashboard | four flat sections: attention, evolved, adoption, moves | `ui/src/pages/DashboardPage.tsx` |
| CLI | `catalog` exposes `coverage`, `next`, `gate` only | `cli/domains/catalog/register.go` |
| Domain `order` | declared 10–200, parsed by nothing | `catalog/config.json` |

## Experience stacking — what exists and what does not

The machinery for hierarchical experience obligations is scaffolded in three
places and wired in none.

1. **Capability facets.** `scenarios/experience-manager/capabilities/capabilities/*.json`
   splits every capability into `promises` (59) and `port` (23). A `port` is
   documented as a host obligation: `reduced-motion` is the canonical case — the
   component promises correct reduced behavior, the host must supply the
   preference.
2. **`satisfies`.** 23 catalog assets declare which ports they provide
   (`adapters.*`, `foundations.tokens` → `design-tokens`,
   `foundations.ui-provider` → `ui-provider, reduced-motion, density-preference`,
   `services.layer-manager` → `layer-manager`).
3. **`extends`.** The experience schema supports `component.extends` pinning a
   canonical RCL contract, and `checkWrapperExtension`
   (`scenarios/experience-manager/api/internal/spec/claim_validation.go:158`)
   enforces additive-only wrapping. 182 canonical contracts exist. Zero specs
   use `extends`.

**The design correction.** Claims should not stack downward. Re-proving Button's
contrast inside DataTable inside a page template is combinatorial and re-tests
the same pixels. What genuinely composes upward is the **unmet port**: union the
port-facet `requiredCapabilities` over an asset's closure, subtract what the
closure provides, and the remainder is the contract a consuming scenario must
honor.

Worked example — `templates.collection-page`, closure of 47:

```
demanded ports:      reduced-motion
provided in closure: design-tokens (foundations.tokens)
                     icon-registry (foundations.icon-registry)
UNMET:               reduced-motion
```

`reduced-motion` is demanded by 7 of the 47 assets in that closure —
`collection-page`, `button`, `loading-state`, `status-indicator`, `skeleton`,
`spinner`, `use-reduced-motion` — and satisfied by none of them.
`foundations.ui-provider` declares `satisfies: reduced-motion` and is not a
dependency of anything in the closure. The host must mount it, and no surface
says so.

The same unmet port appears on `data-display.data-table` (closure 28) and
`patterns.resource-collection` (closure 46).

## Sequence of recommended work

1. Reverse index and a Relationships tab — projection over existing rows.
2. Sidebar becomes a real tree: domain → rung → asset, ordered by domain `order`.
3. Dashboard Structure panel: census, invariants, cannot-change-cheaply list.
4. Catalog ↔ manifest graph reconciliation.
5. Unmet ports over the closure.

Each step reuses the data the previous one landed. Step 1 requires no schema
change and no new authoring burden.
