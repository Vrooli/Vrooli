# Search and composition — findings and target model

Status: **proposal, not implemented**. This document records what the workbench does
today, why “landing page” fails, and the model and migration needed for one search
system across workspaces, pages, displays, states, page types, assets and commands.
The visual options are in [search-options.html](../mockups/search-options.html);
the matured IA and workspace model are in
[workbench-maturation.html](../mockups/workbench-maturation.html).

Behavioral claims below are current only where a source path or check is named.
The recommendation is design intent.

## What exists today

There is one server search and four UI search surfaces. They do not share a result
model, a ranking rule or a store.

| Surface | What it searches | How | Limit |
|---|---|---|---|
| Home search form | 275 published assets | Navigates to `/components?q=` | Name, id, description only |
| Components search | 275 published assets | Client-side substring filter over `useCatalog()` | No server search, no props/tags/stories despite placeholder copy |
| ⌘K command palette | 275 published assets + design scenario names | Client-side substring filter | No workspaces, templates, page types, commands or evidence |
| Design studio template shelf | 8 code-defined templates | A `<select>` dropdown | No search, no designs |
| `CatalogService.SearchAssets` | 478 authored catalog assets | In-memory token index in `api/internal/catalogsearch/index.go` | Not called by the UI; `searchDesignAssets()` is defined in `ui/src/api/catalog.ts` and called nowhere |

### What “landing page” actually does

- `Components?q=landing page` → **0 results**. The query is a single substring, and
  no published asset is named exactly that.
- `⌘K → landing` → **one design scenario**, no assets. The palette loads published
  assets, not the catalog’s `templates.landing-page`.
- `CatalogService.SearchAssets("landing page")` → `templates.landing-page` first,
  then every other page archetype with score `0.5`, because each contains the token
  `page`. There is no IDF, phrase boost, AND semantics or type prior.

The catalog already knows the answer. The UI cannot reach it.

### The structural duplication

| Thing | Store | Shape | Missing |
|---|---|---|---|
| Page archetype | `catalog/assets/templates/*.json` (19) | Intent: regions, states, capabilities, required assets | Not searchable or previewable in the workbench; `availabilityState` is often `asset_unresolved` |
| Studio template | `/api/templates` + `studio_templates` | One `TemplateNode` tree, palette, display, radius | No revision, pages, displays, states, binding, assets or conflict handling; a duplicate store |
| Design | `/api/designs` + `studio_designs` | Multiple frames, theme, options, revision, binding, assets | No workspace wrapper, pages, states, provenance or `realizes`; frames are viewports, not pages |
| Experience page | `experience/pages/*.json` (14) | Per-scenario intent contract | No concrete composition link; `Design.scenario/page` is the only join |
| Design page / scenario | `SketchService.ListDesignPages` | Scenario page inventory | Used by ⌘K as a stand-in for “pages” |

The search problem is downstream of this: there is no first-class, searchable
**workspace/page/display/state** object for a flow that composes assets. Templates
were an attempt to create one, but they live beside the design model instead of
inside it.

## The target model

Search should traverse four layers that already exist:

```
Intent ──realizes──▶ Composition ──composes──▶ Source ──proven by──▶ Evidence
Page type            Workspace           Asset               Story, contract,
regions, states      Page / Display      version, source     test, capture
                     State
```

Current mapping: `DesignDocument` → Page; `DesignFrame` → Display; a
`studio_designs` row becomes a one-page Workspace; `studio_templates` becomes
`provenance=curated` workspaces. States are new.

### Workspace, page, display, state

**Operator direction, 2026-11-19.** The design-studio object is a **Workspace**,
not a page. It contains multiple **Pages**; each Page has multiple **Displays**
(desktop, tablet, phone); each Display has multiple **States** (props and/or UI
state). This supersedes the earlier single-page-first recommendation in this
document.

```
Workspace  (Harbor, curated template, onboarding flow)
  Page     (Welcome, Profile, Preferences, Done)
    Display  (Desktop, Tablet, Phone)
      State  (Default, Filled, Validation error, Loading)
```

- **Workspace** is the authored object and the template unit. A flow is a workspace
  with ordered pages. `provenance` is `authored | curated | imported`; a curated
  template is immutable and forks on use.
- **Page** is one screen or route. It is the current `DesignDocument`; its frames
  become Displays. A page can declare `realizes` a catalog page type.
- **Display** is one device frame. It is the current `DesignFrame`.
- **State** is a named variant of one display. It is addressable, searchable,
  capturable and testable: `Harbor / Profile / Phone / Validation error`.
- **Template** is a curated workspace snapshot, not a separate table. A single-page
  template is a workspace with one page; an onboarding template has four.

**Keep the page type separate.** The catalog page type is the intent/blueprint
(regions, states, capabilities, required assets), not a rendering. The page points
at it with `realizes`; the page type exposes `pages[]` / `workspaces[]` in return.
That is what makes “show me every landing page and what it composes” honest.

See [WORKBENCH-MATURATION.md](WORKBENCH-MATURATION.md) for the full IA, relocation
ledger and migration.

## Target search system

### One result model

A union result with a stable `kind` and a shared envelope:

```ts
type SearchKind =
  | "workspace" | "page" | "display" | "state"
  | "page_type" | "asset" | "command" | "evidence";

interface SearchResult {
  id: string;
  kind: SearchKind;
  title: string;
  subtitle: string;
  description: string;
  score: number;
  matchedFields: string[];
  highlights: Array<{ field: string; text: string }>;
  relations: {
    realizes?: string;                 // page → page type
    workspaces?: string[];             // page/state → containing workspace
    pages?: string[];                  // workspace → ordered pages
    displays?: string[];               // page → displays
    states?: string[];                 // display → states
    composes?: string[];               // workspace/page → assets
    usedBy?: string[];                 // asset → workspaces/pages
    boundTo?: { scenario: string; page: string };
  };
  preview?: { url: string; alt: string };
  badges: Array<"template" | "workspace" | "verified" | "draft" | "blueprint" | "unresolved">;
  route: string;
}
```

### One index, several sources

Index the component registry and catalog assets, workspaces, pages, displays,
states, page types, experience pages, scenario design pages and a declarative
command registry. Index the same fields the UI shows, plus tags, regions,
capabilities, stories and usage contracts where they exist. States are documents:
`Harbor / Profile / Phone / Validation error` is a searchable address.

Do not keep the 275/600 client cap. Search the full authorized corpus on the server
before pagination and expose an honest end.

### Ranking

- Field-weighted BM25/IDF, not raw substring counts.
- Exact and prefix boosts; quoted phrase boost; singular/plural and synonyms
  (`landing page` ↔ `marketing page` ↔ `hero page`).
- AND semantics with an explicit OR fallback.
- Type priors: outcome-first for plain queries (workspace/flow, then page, then
  state, then page type/asset/command); asset-first when the query clearly names
  an asset (`button`, `data table`).
- Relation expansion: if a workspace matches, show its ordered pages and state
  coverage, then the assets it composes. If a page type matches, show the pages and
  workspaces that realize it.
- Availability and freshness affect confidence, not relevance alone: label
  `built`, `blueprint`, `unresolved` honestly.
- Deduplicate a curated template and its forks; show provenance on the row.

### Commands and shortcuts

Use the library’s existing `CommandRegistry`, `CommandPalette` and
`GlobalCommandSystem` assets. Register actions declaratively with id, label, group,
shortcut, scope, availability predicate and handler. Commands become search results
with shortcuts, so “reindex”, “new design”, “open review” and “copy package path”
are discoverable by the same omnibox.

### AI as a mode, not the engine

`Ask` is explicit. It sees only indexed documents and must cite them. Unanswerable
queries fall back to lexical results. Keep the local-model constraint from the
experience-engineering programme; no paid API dependency.

Use the shared `packages/ai-go/search` engine for the semantic leg: hybrid dense +
sparse retrieval (Qdrant), Ollama embeddings, optional reranking, a lexical
fallback when Ollama/Qdrant is down, and `.vrooli/search.json` as the tuning SSOT.
Do not build a scenario-local embedding pipeline. `scenarios/cli-health` (dense
commands) and `scenarios/knowledge-observatory` (hybrid docs) are the worked
adopters.

## UX states

- **Atlas** — the browse-first front door and empty query state. Categories derive
  from catalog domains, page types and workspaces with counts. “Onboarding flows”
  is a first-class category. Recents, curated templates and commands fill the page.
- **Spotlight** — the same engine invoked with ⌘K. Grouped results (workspace, page,
  state, page type, asset, command), preview pane on desktop, full-screen sheet on
  phone, roving focus and visible shortcuts.
- **Ask** — an explicit mode inside the omnibox. Natural language in, a cited
  shortlist and next action out.
- **Phone** — pinned search field, two-column category grid, one scroll of grouped
  results, best match expanded, FAB for the primary action, no hover-only controls.
  The bottom navigation is Start, Library, Design, Review, Stats.

Every state must cover loading, stale, index-empty, no-match, partial and
request-error, with a recovery path. The no-match state must distinguish “nothing
matches” from “nothing is indexed”.

## Migration plan

**Phase 0 — unify search over existing stores (no migration).** Add the result
model and index, index existing designs/templates, catalog assets, experience pages
and a command registry, then build Atlas + Spotlight. This delivers the visible win
without waiting for entity cleanup.

**Phase 1 — Workspace wrapper.** Promote each existing design to a page inside a
one-page workspace; rename the UI concept; keep old routes as compatibility reads.
`DesignDocument` becomes Page; `frames` become Displays.

**Phase 2 — multiple pages and flows.** Add ordered pages and the flow view. An
onboarding workspace holds Welcome → Profile → Preferences → Done.

**Phase 3 — states per display.** Add named states with args, scenario data,
design status and capture status. Index them; expose them in the studio switcher and
the evidence panel.

**Phase 4 — template = curated workspace.** Migrate `studio_templates` into
workspaces with `provenance=curated`; fork on use; delete the separate CRUD and the
duplicate search path. Page types link to pages/workspaces through `realizes`.

## Acceptance evidence

- “landing page” returns `LandingPage` first, then workspaces/templates tagged or
  bound to it, then composed assets — never every page type.
- “onboarding flow” returns the Harbor workspace first, then its pages and state
  coverage, then the page type, pattern, assets and commands.
- “validation error” returns the addressable state and its workspace/page/display.
- Empty-state categories come from the catalog and open real filtered results.
- Commands are searchable, show shortcuts and context, and execute.
- The full 478-asset corpus works on desktop and at 390 px; no capped client list,
  no hidden overflow.
- Loading, stale, index-empty, no-match, partial and request-error are distinct.
- Every AI answer cites an indexed workspace, page, state, page type or asset.
- The workbench consumes its own search and command assets.
- “Use template” forks a workspace; curated templates cannot be edited in place;
  provenance survives the fork.
- Missing states and missing captures never render as complete.

## Open decisions

1. Default ranking: outcome-first or asset-first? (Recommendation: outcome-first
   for plain queries, asset-first for explicit asset queries.)
2. How local must Ask be? (Recommendation: local model through AI Gateway, lexical
   fallback.)
3. State authoring: explicit args per state, captured scenario data, or both?
4. Can a page appear in more than one flow/workspace, or is it owned by one?

Answered on 2026-11-19: a template is a curated workspace; workspaces hold multiple
pages; each page holds displays; each display holds states. Curated templates fork
on use.

## Evidence

- Captures: `docs/mockups/assets/*.png`, 2026-11-19, 1440×900 and 390×844.
- Options: [search-options.html](../mockups/search-options.html).
- Maturing IA/model prototype: [workbench-maturation.html](../mockups/workbench-maturation.html) and [WORKBENCH-MATURATION.md](WORKBENCH-MATURATION.md).
- Sources: `catalog/assets/templates/`, `api/internal/catalogsearch/index.go`,
  `ui/src/workbench/WorkbenchShell.tsx`,
  `ui/src/workbench/pages/ComponentsPage.tsx`, `ui/src/workbench/designs.ts`,
  `api/internal/templates/types.go`, `experience/pages/`.
