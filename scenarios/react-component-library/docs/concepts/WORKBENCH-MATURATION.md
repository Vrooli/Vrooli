# Workbench maturation — five places, one model

Status: **implemented slice; legacy-owner deletion, BAS state capture and the
semantic search leg remain.** The five destinations, workspace aggregate, unified search,
server-owned Review/Activity, a paginated Review queue, the CLI surface,
workspace assets/assist/HTML import, curated template seeds and composed Stats
are implemented and served. The workspace studio renders the
mature studio surfaces (page tabs, layers, system, canvas, inspector, status bar,
present, undo/redo, autosave). Evidence and remaining gaps are in
[WORKBENCH-PROGRAM.md](../internal/WORKBENCH-PROGRAM.md#0-delivery-status-2026-11-19).
This is the information architecture and design model after the D · Nocturne
workbench. It responds to the operator’s 2026-11-19 direction: keep the
Start/search index and the phone composition, re-home Home’s modules instead of
deleting them, consolidate Coverage, Activity and Capabilities into one Stats
page, and make the design studio a workspace with pages, displays and states.

Visual prototype: [workbench-maturation.html](../mockups/workbench-maturation.html)
Search decision record: [SEARCH-AND-COMPOSITION.md](SEARCH-AND-COMPOSITION.md)

Current behavior is asserted only by the checked behavior register. Everything below
is design intent until its checks and served evidence exist.

## Decision summary

| Area | Direction |
|---|---|
| Primary destinations | **Start, Library, Design, Review, Stats** — five, matching the phone bottom navigation |
| Home | Becomes **Start**: search/browse, continue, curated templates, compact pulse |
| Coverage + Activity + Capabilities | Become **Stats** views: Overview, Quality, Activity, Services |
| Design | Becomes a **Workspace** index plus a studio with pages, displays and states |
| Template | A curated **workspace snapshot**; fork on use; never a second store |
| Create | A global **New** action, not a primary destination |
| Review | Stays a separate work queue |
| Settings / Component / Preview | Secondary or object routes, not primary destinations |

Rationale: three of the seven destinations answer facets of one question — “is the
library healthy?”. Keeping them separate duplicates their summaries on Home, grows
the rail, and forces the phone to hide two of them. Review is different: it is a
queue with actions, not a measurement.

## Information architecture

### Primary

| Destination | Question | Contains |
|---|---|---|
| **Start** | What can I make or find? | Atlas/Spotlight/Ask search, browse by intent, continue, curated templates, compact library pulse, shortcuts |
| **Library** | What is it built from? | Assets, recently published, featured asset, collections, verification state |
| **Design** | What am I composing? | Workspace index, templates, page/display/state studio, flow view |
| **Review** | What needs a person? | Severity board, blockers, decisions, evidence debt, triage |
| **Stats** | Is the library healthy? | Overview, Quality (Coverage), Activity, Services (Capabilities) |

### Secondary and routes

- **New** is a global action. `/new` keeps the Create / Extract / Adopt routes.
- **Settings** stays secondary.
- **Component** (`/assets/:id`) and **Preview** (`/assets/:id/preview`) remain object
  routes.
- Redirects: `/coverage` → `/stats?view=quality`, `/activity` →
  `/stats?view=activity`, `/capabilities` → `/stats?view=services`, `/catalog` →
  `/components`.

## Home relocation ledger

Nothing is deleted. Every current module is re-homed.

| Current module | Was on | New home | Why |
|---|---|---|---|
| Search box + browse | Home | Start → Atlas | Search is the front door |
| Recently touched assets | Home | Library → Recent | Asset work belongs with assets |
| Continue designing | Home | Start → Continue + Design | Workspace continuation is a launchpad job |
| Library health ring | Home | Stats → Overview | Measurement belongs in Stats |
| Signals (capabilities, drafts, recency) | Home | Stats → Services / Activity | Split by what the signal measures |
| New in collection | Home | Library → Recently published | Browsing additions is a library task |
| Live specimen | Home | Library → Featured asset | Showcase where the asset is inspectable |
| KPI strip | Home | Stats → Overview; compact pulse on Start | Keep one number in reach, not a second dashboard |
| Maturity matrix, definition of ready, next work | Coverage | Stats → Quality | All quality measures in one scroll |
| Change trail, actors, durability note | Activity | Stats → Activity | Audit trail is a view of health |
| Service status + recovery commands | Capabilities | Stats → Services | Readiness is part of health |
| Findings queue + triage | Review | Review | A queue with actions is not a measurement |
| Create / Extract / Adopt | Create | Global New action | An action is not a destination |
| Settings | Settings | Settings | Secondary |
| Component + Preview | Routes | Routes | Deep links are not navigation |

The prototype’s Start, Library and Stats pages show each destination still has a
surface, not just a redirect.

## Workspace model

A design-studio object is a **Workspace**:

```
Workspace  (Harbor, template, onboarding)
  Page     (Welcome, Profile, Preferences, Done)
    Display  (Desktop 1180×780, Tablet 834×900, Phone 390×844)
      State  (Default, Filled, Validation error, Loading)
```

| Level | Meaning | Current equivalent |
|---|---|---|
| **Workspace** | The authored object and template unit. A flow is a workspace with ordered pages. | No wrapper today; `studio_designs` is a page |
| **Page** | One screen or route; realizes a page type from the catalog. | `DesignDocument` |
| **Display** | One device frame for that page. | `DesignFrame` |
| **State** | A named prop/state variant of one display; addressable and capturable. | Not modelled |

### Template

A template is a **curated workspace snapshot**. It is immutable; “use” forks it into
a private workspace and keeps the origin link. A single-page template is a workspace
with one page; an onboarding template is a workspace with four pages. The separate
`studio_templates` store disappears; the shelf becomes
`workspace.provenance = curated`.

### Page type

The catalog page type remains the intent/blueprint (regions, states, capabilities,
required assets), not a rendering. A workspace page declares `realizes`; the page
type projects `pages[]` / `workspaces[]` back. That is what makes “show me every
onboarding page and the workspaces that realize it” honest.

### States

A state is a first-class object, not a screenshot variant:

- `Harbor / Profile / Phone / Validation error` is a stable address.
- It carries args/props, a scenario/data fixture, design status and capture status.
- It can be searched, captured, tested and linked from Review.
- Missing states are visible (for example, `Profile / Loading`).
- A State is a variant of a Display, not a duplicate Display. The canvas may render
  state copies side by side; the stored object is one Display with N States.

This is the operator’s correction to the earlier single-page recommendation: a flow
needs multiple pages, and a display needs multiple states.

## Search implications

The search model from [SEARCH-AND-COMPOSITION.md](SEARCH-AND-COMPOSITION.md) stays;
the result kinds become workspace, page, display, state, page type, asset, command
and evidence.

| Query | First | Then | Then |
|---|---|---|---|
| `onboarding flow` | **Harbor** workspace · 4 pages · 12 displays · 12 states | Pages and state coverage | OnboardingPage page type, OnboardingFlow pattern, assets, commands |
| `landing page` | **LandingPage** page type | Workspaces/templates tagged or bound | Assets used, commands |
| `validation error` | **Harbor / Profile / Phone / Validation error** | The containing page and workspace | Assets in the state, capture command |
| `button` | **Button** asset | Workspaces/pages where it appears | Adopt/open/compare commands |

The empty state adds **Onboarding flows** as a category. A state result is only
possible if the data model names the display and state; anonymous viewport frames
cannot answer it.

The semantic leg uses the shared `packages/ai-go/search` engine (hybrid dense +
sparse, Qdrant, Ollama, reranker, lexical fallback, `.vrooli/search.json` tuning),
not a scenario-local pipeline.

## Migration

| Phase | Change | Exit |
|---|---|---|
| 0 | Unified search over existing stores, Atlas + Spotlight, command registry | “landing page” and “onboarding flow” return typed results; no capped client list |
| 1 | Workspace wrapper; rename `DesignDocument` to Page in the UI; one-page workspaces | Existing designs open as workspaces; routes and composition preserved |
| 2 | Multiple pages, ordering and flow view | Harbor holds Welcome → Profile → Preferences → Done |
| 3 | States per display, state switcher, capture and search indexing | Missing states are visible and citable |
| 4 | Migrate `studio_templates` to curated workspaces; fork on use; delete duplicate CRUD | One store, one shelf, one search path |

Schema sketch:

```text
Workspace { id, name, provenance, tags[], pages[] }
Page      { id, name, route?, realizes?, displays[], order? }
Display   { id, device, width, height, states[] }
State     { id, name, args, scenario?, status, captureId? }
```

## Acceptance evidence

- Five destinations at 1440, 390 and 320; the phone bottom navigation matches.
- Every Home, Coverage, Activity and Capabilities module is reachable from Start,
  Library or Stats; the ledger is complete.
- A workspace opens with pages, displays and states; switching a page, display and
  state changes the canvas and the inspector.
- A template forks into a workspace; the curated source is unchanged.
- Stats covers Overview, Quality, Activity and Services, including the cold-start
  “preparing” and unavailable states.
- “onboarding flow” returns the workspace first, then pages, then the page type,
  pattern, assets and commands.
- Missing states and missing captures never render as complete.
- No horizontal overflow at 390; no hidden primary destination.

## Open decisions

- State authoring: explicit args per state, captured scenario data, or both?
- Can a page appear in more than one flow/workspace, or is it owned by one workspace?
- Does Stats keep four tabs or become one long scroll with anchors?
- Do page types become user-facing “page types” in the catalog, or stay internal
  archetypes?

## Evidence limits

The prototype is static HTML/CSS in the Nocturne tokens, not the running app. It is
not wired to the API or SQLite, and its counts are illustrative where the model does
not exist. No product code or test suite changed. The current workbench is dark-only.
