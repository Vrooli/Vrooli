# Workbench implementation program

Status: **phase 0 complete; phase 1 complete; phase 2–5 vertical slice
implemented; deletion/parity and the capture sweep remain.** This is the program
to move React Component Library from a visually strong but structurally split
workbench to one professional system: five destinations, one workspace aggregate,
one search, one health/stats owner, one authoring transport, and an explicit
deletion path.

It treats the target as **greenfield design** and the delivery as a **strangler
migration**: define the ideal domain, contracts, schema and UI first; build new
vertical slices beside the old ones; migrate data once; switch routes; delete the
old owners. Published URL redirects and one-release read-only adapters are the only
compatibility we keep.

Related decision artifacts:

- [WORKBENCH-MATURATION.md](../concepts/WORKBENCH-MATURATION.md) — five destinations and workspace model
- [SEARCH-AND-COMPOSITION.md](../concepts/SEARCH-AND-COMPOSITION.md) — search findings and target
- [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md) — historical D · Nocturne record
- [DEBT.md](../studio/DEBT.md), [PROBLEMS.md](PROBLEMS.md) — dated debt and gaps
- [WORKBENCH-MATURATION.html](../mockups/workbench-maturation.html) and [search-options.html](../mockups/search-options.html) — rendered prototypes

## 0. Delivery status (2026-11-19)

**Implemented and verified**

- Five destinations with redirects: Start `/`, Library `/library`, Design
  `/design`, Review `/review`, Stats `/stats`; legacy `/components`, `/catalog`,
  `/coverage`, `/activity`, `/capabilities` preserve their query context.
- `workspace.proto` + `internal/workspaces`: pages → displays → states,
  revision snapshots, revision-guarded save, fork-on-use, validation, and a
  one-time `studio_designs` migration guarded by `workspace_migrations`.
- `search.proto` + `internal/search`: FTS5 over catalog, page types,
  workspaces/pages/displays/states, commands and findings; ranked golden
  queries, kind/availability facets and honest status.
- `review.proto`, `activity.proto`, `libraryhealth.proto`: server-owned findings
  with durable dispositions, an append-only event log, and a read-composed
  health snapshot with per-section availability.
- One `SearchSurface` shared by Start and ⌘K; workspace studio; Design workspace
  index; Stats Overview/Quality/Activity/Services; server-backed Review and
  Activity.
- Removed the browser findings/activity reconstructions, the duplicate Home
  page, 126 unused catalog strings and write-path `make check`.
- **Second slice (2026-11-19):** the workspace studio renders the mature studio
  surfaces (layers, system tokens, canvas with zoom/guides, inspector,
  status bar, present mode, undo/redo, autosave with conflict stop); Review is a
  server-filtered paginated queue; workspaces/search/review/activity/health are
  promoted to the CLI with no omissions; search links address the workspace
  studio and its projection version forces a rebuild on shape changes; the seven
  published-byte drifts are covered by a reviewed repair ledger and the release
  hash check now detects unexplained drift; all five destinations and the
  workspace studio pass structural axe.

**Evidence**

- Go: `go test ./...` passes across every package; `golangci-lint` is clean for
  `api` and `cli`.
- UI: 132 files / 797 tests, lint 0 errors, type-check, experience guard,
  catalog conformance and production build pass.
- Served API: legacy designs and uploads migrated; 703 search documents indexed;
  eight curated templates; Review total 13,926 in ~14 ms; all five routes 200;
  no horizontal overflow at 320; reduced-motion and forced-colors checked.

**Third slice (2026-11-20):** workspace reference assets (upload, byte route,
region crop, place as layer, legacy uploads migrated); `AssistWorkspace` + the
studio AI tab + `workspaces assist` in the CLI; HTML import as a new page; eight
reviewed curated templates seeded at boot with immutable provenance and a
read-only fork flow; the Design shelf renders real template compositions. The
first edit on a blank stored composition now writes what was on screen instead
of silently doing nothing. `golangci-lint` was rebuilt with Go 1.26, 30 dead CLI
functions were deleted, and both Go lints are clean. 320 px overflow was removed
from Start and Design; reduced motion and forced colors were measured.

**Remaining, in program order**

1. Delete the legacy design/template stores (phase 2/6). Two flows still keep the
   old studio alive: component-context authoring (`/design?template=…&component=…`
   needs the live component slot stored on a workspace) and experience pages with
   no workspace binding. Bound pages already open the workspace studio.
2. Render a composed state through the BAS evidence pipeline and write
   `WorkspaceState.status`/`capture_id` from it (phase 2). Region capture from a
   reference image is already implemented.
3. Add the `packages/ai-go/search` semantic leg and Ask citations (phase 3).
   `AssistWorkspace` covers design proposals, not retrieval.
4. Run a full keyboard journey sweep across the five destinations (phase 4/6).
   Captures, overflow, structural axe, reduced motion and forced colors are done.

**Closed since the second slice (2026-11-20):** workspace assets and assist;
curated template seeds; HTML import; Go lint runnable and clean; 320 px,
reduced-motion and forced-colors checks.

## 1. Definition of done

### Product

- Five destinations: **Start, Library, Design, Review, Stats**; the phone bottom
  navigation uses the same five.
- Start is the search/browse front door; Library is assets; Design is workspaces,
  pages, displays and states; Review is the findings queue; Stats is quality,
  activity and services.
- A curated template is a workspace snapshot that forks on use; there is no
  separate template entity.
- Search returns workspaces, pages, displays, states, page types, assets, commands
  and evidence, with honest availability and a working no-match/index-empty
  distinction.
- Every Home, Coverage, Activity and Capabilities module is reachable from its new
  home (the relocation ledger is complete).
- Desktop, 390 and 320 are designed; loading, empty, partial, stale, error,
  permission and offline states are deliberate.

### Architecture

- One owner per concept; no second store for the same thing.
- One Workspace aggregate; one composition store; one search index; one findings
  owner; one activity owner; one health composition.
- One authoring transport (Connect-RPC) for JSON; REST only for binary assets.
- One error envelope and one code mapping.
- One schema owner per table; versioned migrations; no inline production DDL.
- No silent local fallback that pretends to be durable.
- No god service: a service has a coherent consumer job, not “everything design”
  or “everything catalog”.

### Evidence

- `pnpm test`, lint, type-check, catalog check and build pass.
- `go test ./...` passes, including the three failures that exist today.
- Contract parity, migration dry-run, search ranking/p95, axe, reduced motion,
  forced colors, phone captures and overflow checks pass.
- No requirement is marked complete without evidence at the current revision.

### Debt

- The deletion ledger in §7 is executed, with a net-negative line report.
- No dead exports, dead CSS, duplicate device metrics, duplicate verdict renderers
  or unreferenced legacy pages.
- One canonical documentation set; no broken references.
- File budgets: no page component over 400 lines, no hook over 350, no CSS file
  over 1,500 unless the exception is recorded.

## 2. Current state (measured 2026-11-19)

### 2.1 Surfaces and scale

| Area | Current |
|---|---|
| Served UI routes | Home `/`, Components `/components`, Component `/assets/:id`, Preview, Design `/design` (+ scenario/page/sketch redirects), Review, Coverage, Capabilities, Activity, Settings, Create `/new` |
| Primary rail | 7 destinations + Settings + New |
| Workbench UI | ~22,300 lines of TS/TSX/CSS under `ui/src/workbench` |
| Largest files | `DesignStudioPage.tsx` 1,246; `useStudio.ts` 930; `templates/library.ts` 816; `ComponentPage.tsx` 790; `CoveragePage.tsx` 443; `HomePage.tsx` 569; `WorkbenchShell.tsx` 453 |
| API modules | 16 domain modules; 13 proto-backed Connect domains; designs and templates are REST exceptions |
| Largest API surfaces | `CatalogService` 16 RPCs; `SketchService` 37 RPCs; `ComponentsService` 22 RPCs |
| Catalog | 478 desired assets, 275 published manifests, 19 page types |
| Experience | 14 RCL experience pages, 54 experience components, 3 journeys |
| Studio data | 2 designs, 3 frames each; 0 template rows; 8 code-defined templates |

### 2.2 The central problem: one concept, five owners

A “page/flow design” is represented five ways:

| Representation | Owner / path | Shape | Problem |
|---|---|---|---|
| Page type / archetype | `catalog/assets/templates/*.json` (19) | Intent: regions, states, capabilities | Not a rendering; often `asset_unresolved` |
| Per-scenario experience page | `scenarios/*/experience/pages/*.json`, `internal/sketch` | `sketch.Document`: intent + template + placements + regions + notes | Duplicates composition inside a contract file; file-based, per-scenario |
| RCL design | `studio_designs`, `internal/designs` | Multi-frame doc, theme, options, binding, revision | No pages/states/provenance; frames are treated as viewports |
| RCL template | `studio_templates`, `internal/templates` | One `TemplateNode` tree, palette, radius | Separate CRUD, no revision/binding/assets, forks only by copy |
| Code + browser shelf | `templates/library.ts` (8), localStorage | JS constants and local copies | Hardcoded content, browser-local, no server ownership |

Add `fixtures` localStorage (`rcl.workbench.fixtures.*`), design fallback copies
(`rcl.designs.v1`), and preferences. The same composition can exist in several of
these at once, with different ids and no reconciliation.

### 2.3 Search

- Four UI surfaces: Home form (navigates with `?q=`), Components client filter,
  ⌘K client filter, studio template `<select>`.
- One server index: `internal/catalogsearch` over 478 catalog assets, token
  substring, no IDF/phrase/AND/type prior; not called by the UI.
- `searchDesignAssets()` is defined in the UI and unused.
- No workspace/page/state/command/evidence result kinds.
- `CatalogService.SearchAssets/Status/Reindex` exist but the UI ignores them.
- “landing page” on Components returns zero; ⌘K returns one scenario; the server
  index returns `LandingPage` plus every page type at 0.5.

### 2.4 API and contracts

- `CatalogService` mixes catalog, coverage, gates, readiness, health, capture and
  search in 16 RPCs.
- `SketchService` mixes page inventory, authored composition, candidates,
  inference, critiques, capture, acceptance, briefs, history, import and templates
  in 37 RPCs.
- `internal/sketch`, `internal/designassist`, `internal/designcapture`,
  `internal/designcritique`, `internal/designinference` total ~9,000 lines and
  overlap the design domain.
- Designs and templates are REST exceptions with hand-written clients; the rest is
  Connect. The same domain has two transports and two error shapes.
- `handlers/sketch` and `designcapture` mix plain-text errors with JSON envelopes.
- `reconcile_evidence` DDL is created inline in three production files, bypassing
  the schema seam.
- The `docs/internal/goal/TARGETS.md` contract is referenced by `DESIGN.md`, the
  PRD and 12+ requirement notes but **does not exist**.

### 2.5 Storage and migration

- SQLite stores components, versions, adoptions, evidence, version ledger,
  designs, templates, assets, workflows.
- Authored experience pages live as files inside each scenario.
- Migrations are a mix of `CREATE TABLE IF NOT EXISTS`, `internal/designs.Migrate`
  with `ensureColumn`, and domain-specific migration helpers. There is no single
  ordered migration ledger.
- Templates have their own table and API; designs have another.
- Local fallback stores are not a cache: they can become the only copy and cannot
  be promoted when an offline-created id reaches the server.

### 2.6 UI complexity and duplication

- `DesignStudioPage` owns route binding, templates, layers, canvas, inspector,
  assets, AI, capture, present, import and checks. `useStudio` owns documents,
  history, save queue, sync, conflicts, assets and template seeding.
- Four check/verdict renderers with overlapping vocabularies.
- Three device metric tables: preview, in-context, studio.
- Hand-built preview chrome in `PreviewPage` beside `DeviceFrame`.
- `findingsFromCoverage` reconstructs Review in the browser; `Activity` is derived
  from catalog timestamps; `Stats` would be a third reconstruction.
- `WorkbenchShell` mounts catalog, coverage and scenario queries on every route.
- `ListDesignPages({})` walks the whole fleet with no server cache.
- `useComponentBundle` is five requests per component; a design with N components
  is 5N requests.
- Dead CSS (294 lines + media queries), dead exports, duplicate helpers and
  duplicated “take into studio” URL construction.
- i18n is now wired (1,045 workbench leaves in `en.json`), but some labels and
  relative-time formatting still bypass it.

### 2.7 Measured quality baseline

Run on 2026-11-19, current tree:

| Gate | Result |
|---|---|
| UI tests | **123 files / 750 tests passed** |
| UI lint | **0 errors**, 5 catalog warnings |
| UI type-check + catalog conformance | pass (tokens, pins, exports, usage-contract types) |
| Go tests | **3 failures in 2 packages**: `internal/gates` (utility-class allowlist; release provenance for 5 releases) and `internal/reconcile` (selector fixture drift at `ComponentsPage.tsx`) |
| Go build | pass |
| Served capture/phone | 11 routes have phone/desktop captures and axe at 390/320 per PROGRESS; full interaction-state phone journeys remain incomplete for 10 |
| Requirements | 54 complete, 36 in progress, 16 planned, 16 not implemented, 3 pending; many P0 |

The three Go failures are content/fixture debt, not studio code, but they still
block an honest “all green” claim.

### 2.8 Documentation and requirements contradictions

- `PRD.md` OT-P0-014 still describes **eleven pages** including Coverage,
  Capabilities and Activity as first-class; the new direction is five destinations.
- `WORKBENCH-REDESIGN.md` records the historical 11-page implementation;
  `WORKBENCH-MATURATION.md` proposes the five-page target. Both are useful only if
  one is clearly current and the other historical.
- `DESIGN.md`, the PRD and `requirements/25-rehabilitation` reference a missing
  `docs/internal/goal/TARGETS.md`.
- `docs/studio/DEBT.md` is a dated snapshot; some entries are fixed, some remain.
  It must be re-baselined, not treated as current line-by-line truth.
- Experience contracts still describe the old route set and page claims.

### 2.9 Known gaps and bugs to close

- Template shelf is still a separate store and browser-local in places.
- Silent local fallback can look like a successful save; offline-created designs
  cannot be promoted.
- Fixture arrangement is a third local store, not tied to a version.
- Activity is index-derived; human attribution is not durable.
- Review has no server-owned dispositions or assignment.
- No per-pane error boundaries; drag listeners and timers can outlive components.
- Canvas/in-context accessibility: `role="img"` composition, non-native
  interactive wrappers, small-text contrast, duplicate Escape handlers.
- Token leak proof for scoped Nocturne tokens is missing.
- Legacy page components remain in source after route replacement.
- No `TARGETS.md`; no single goal ledger.
- Search has no ranking fixtures, p95 evidence, or command index.
- No migration test from designs/templates/sketch to the workspace model.

## 3. Target architecture (greenfield design)

### 3.1 Domain model and ownership

```
AssetRegistry ── adopted by ──▶ Workspaces ── bound to ──▶ Experience intent
published assets                composition                per-scenario contract
Catalog / PageType ── realized by ──▶ Workspace.Page
Coverage / Gates ──▶ Findings ──▶ Review disposition
Every mutation ──▶ Activity events
Coverage + Findings + Activity + Capabilities ──▶ Library health (read model)
All searchable objects ──▶ Search documents
```

| Concept | Owner | Not owned by |
|---|---|---|
| Published asset, versions, stories, adoption | existing `components` / `versions` / `adoptions` | Workspaces, Search |
| Desired catalog, page types, gates, coverage | existing `catalogcoverage` / `catalogbuild` | UI, Workspaces |
| Workspace, Page, Display, State, revision, assets | new `workspaces` | Experience files, Templates, UI |
| Per-scenario intent/claims/regions | existing `experience` reader (extracted from sketch) | Workspaces (composition is separate) |
| Findings + dispositions | new `findings` projection | UI `findingsFromCoverage` |
| Activity | new `activity` append-only event log | Catalog timestamps |
| Capabilities | existing `capabilities` | UI |
| Library health | new read composition | A new source of truth |
| Search | new `search` index | Catalog-specific search |
| Commands | static generated manifest + owning action handlers | A second command registry |

**Composition vs intent.** The RCL database owns the composition (workspace,
pages, displays, states). The scenario repository owns the intent contract
(`experience/pages/*.json`: purpose, claims, states, regions, bindings). A page
binds to a scenario/page; an explicit reconcile/export step may write a
composition snapshot back, but the experience file is not a second composition
store. `sketch.Document` is split: intent/claims reader vs composition/placement
logic; the latter moves into `workspaces`.

### 3.2 Service boundaries

Target service sizes are bounded by consumer job, not by domain adjectives.

| Service | RPCs | Owns |
|---|---|---|
| `WorkspaceService` | ~12–16 | list/get/create/update/delete, pages, displays, states, fork, import, assets, present/export |
| `WorkspaceAssistService` | ~4 | propose/refine/accept, inference status |
| `WorkspaceCaptureService` | ~4 | capture page/state, status, screenshot, retry |
| `ReviewService` | ~5 | findings list/get, dispositions, assign, acceptance/critique |
| `CatalogService` | ~8 | assets, graph, relationships, ports, structure |
| `PageTypeService` | ~3 | page types, regions, required states |
| `CoverageService` | ~4 | coverage, next work, run/record gate |
| `SearchService` | ~4 | query, suggest, status, reindex |
| `ActivityService` | ~2 | list/stream events |
| `LibraryHealthService` | ~2 | typed health snapshot by section |
| `ExperienceService` | ~3 | list/read/validate intent contracts |
| Existing | unchanged, except a source/authoring split if it blocks the target | versions, adoptions, deps, themes, preview, workflows, health, inventory |
| `ComponentsService` | ~22 now | read projections, authoring, source, stories, experience; split read vs authoring if it blocks the target |

Rules:

- No service over 16 RPCs without an explicit ADR.
- No REST exception for JSON authoring. Use Connect + `bytes spec_json` /
  `bytes args_json` when the client owns the shape.
- REST only for binary asset/preview bytes; failures still use the JSON error
  envelope.
- One error mapping table; `InvalidArgument`, `NotFound`, `Conflict`, `Internal`,
  `Unavailable`; no plain-text errors from authoring routes.
- One proto package per service; generated clients for UI and CLI; a parity test
  asserts every RPC has an endpoint descriptor.

### 3.3 Storage

Versioned migrations (`schema_migrations` or `PRAGMA user_version`), one provider
per domain, no inline DDL. Target tables:

```text
workspaces(id, name, provenance, tags_json, binding_scenario, binding_page,
           theme_json, options_json, revision, created_at, updated_at)
workspace_pages(id, workspace_id, name, route, page_type_id, sort_order, spec_json)
workspace_displays(id, page_id, device, width, height, sort_order)
workspace_states(id, display_id, name, args_json, scenario_data_json,
                 status, capture_id, sort_order)
workspace_assets(id, workspace_id, page_id, name, mime, bytes, width, height,
                 fidelity, data, created_at)
workspace_revisions(id, workspace_id, revision, snapshot_json, actor, created_at)
activity_events(id, actor, action, subject_kind, subject_id, before_json,
                after_json, at, trace_id)
search_docs(id, kind, subject_id, title, body, facets_json, relations_json,
            availability, updated_at)
search_fts(...)  -- FTS5
findings(id, subject_kind, subject_id, gate, severity, evidence_json,
         status, owner, created_at)
finding_dispositions(id, finding_id, decision, reason, actor, at)
```

Rules:

- Workspace aggregate revision is the optimistic-concurrency guard. A write
  replaces the aggregate and appends a revision snapshot; page/display/state
  mutations are ordered.
- States are addressable: `workspace/page/display/state`; capture and search use
  the same id.
- No `studio_designs`, `studio_templates`, or code-defined template shelf after
  migration.
- No localStorage as a source of truth. Preferences may stay local. If offline
  authoring is required, use an explicit IndexedDB draft queue with stable ids,
  visible “not saved” state and a promote flow; otherwise authoring is read-only
  when the API is unavailable.
- Seed curated workspaces from a reviewed JSON seed, not TypeScript constants.
- Retention: activity and revisions are bounded; capture payloads already bounded.
- One backup/export path for workspaces and revisions.

### 3.4 Search

- One index service over: workspaces, pages, displays, states, page types,
  assets (published + catalog), experience pages, commands, findings/evidence.
- Lexical: SQLite FTS5 with field boosts, IDF, phrase/prefix, AND with OR fallback.
- Semantic: the shared `packages/ai-go/search` hybrid engine (Qdrant + Ollama,
  optional reranker) behind the same `SearchService`; lexical fallback when the
  semantic leg is unavailable. `.vrooli/search.json` is the tuning SSOT.
- Ranking: outcome-first by default; asset-first for explicit asset queries;
  relation expansion (workspace → pages → states → assets; page type → pages);
  availability/freshness labels; dedupe curated templates and forks.
- Commands: generated static manifest (id, label, group, shortcut, scope,
  availability, route/action); SearchService indexes it; the UI uses RCL
  `CommandRegistry`/`CommandPalette`.
- Query planning: synonyms, plurals, quoted phrases, type filters, `is:` filters.
- Status: indexed count, indexedAt, stale, lastError, semantic leg state.
- Tests: golden queries (`landing page`, `onboarding flow`, `validation error`,
  `button`, no-match, index-empty), p95 under 100 ms for the lexical corpus,
  semantic-leg degradation returns lexical results with a truthful status.

### 3.5 Activity, Review, Stats

- **Activity** is an append-only event log with actor, action, subject, before/
  after, timestamp and trace. Services write events through one `activity.Record`
  seam (or an outbox) in the same transaction as the mutation where practical.
  The UI no longer derives “who changed what” from catalog timestamps.
- **Review** is a server projection over gate evidence, tests, provenance,
  captures and dispositions. The UI does not reconstruct findings. Dispositions
  are explicit writes with reason/actor; assignment and triage are durable.
- **Stats** is a read composition, not a store: `LibraryHealthService` returns
  typed sections (overview, quality, activity, services) with `unavailable`
  per section, not zero. Quality uses coverage; Activity uses events; Services
  uses capabilities; Overview composes them.
- **Capabilities** remains the registry; Stats is its view.

### 3.6 UI architecture

Routes:

```text
/                         Start
/library                  Library
/design                   Workspace index
/design/:workspaceId      Workspace studio
/design/:workspaceId/:pageId
/design/:workspaceId/flow
/review                   Review
/stats                    Stats (view=overview|quality|activity|services)
/assets/:id               Component
/assets/:id/preview       Preview
/new                      Create / Extract / Adopt
/settings                 Settings
```

Redirects: `/components` → `/library`, `/coverage` → `/stats?view=quality`,
`/activity` → `/stats?view=activity`, `/capabilities` → `/stats?view=services`,
`/catalog` → `/library`, legacy `/sketch/...` → `/design/...`.

Component boundaries:

- `SearchSurface` shared by Start and ⌘K; URL state; server search.
- `CommandPalette` from RCL assets + generated command manifest.
- `WorkspaceShell` → `WorkspaceSidebar` (pages), `PageCanvas` (displays),
  `StateSwitcher`, `Inspector`, `FlowView`, `AssetsPane`, `AssistPane`,
  `CapturePane`, `PresentMode`.
- Hooks: `useWorkspace`, `useWorkspaceSave`, `useWorkspaceAssets`,
  `useWorkspaceAssist`, `useWorkspaceCapture`, `useSearch`, `useLibraryHealth`,
  `useReview`, `useActivity`.
- One `DeviceFrame`, one `DeviceSwitch`, one `viewports.ts`, one `verdicts.ts`,
  one timestamp reader, one error envelope.
- Per-pane error boundaries; focus target for canvas/layer tree; unique Escape
  handling; axe per route; reduced motion; forced colors; 390/320.
- Route-level queries; no shell-wide catalog/coverage loading. Batch component
  bundles; server cache for page inventory.
- i18n: every user-visible string in the catalog; no `defaultValue`-only; Intl for
  relative time/plurals.

### 3.7 CLI

- Add `workspaces` (list/get/create/fork/page/display/state), `search`, `stats`,
  `review`, `pages` commands generated from the same proto.
- No direct REST clients; no duplicate logic.
- Auto-discover the scenario API port; one-line human errors; `--json` opt-in.
- Remove CLI surfaces for retired domains.

### 3.8 Documentation and contracts

- Canonical set: `DESIGN.md` (visual intent), `WORKBENCH-MATURATION.md` (current
  IA/model), `SEARCH-AND-COMPOSITION.md` (search), `WORKBENCH-REDESIGN.md`
  (historical), `docs/studio/PLAN.md` (studio), `docs/internal/PROBLEMS.md`,
  `DEBT.md`, `PROGRESS.md`.
- Restore or retire `docs/internal/goal/TARGETS.md`; all references must resolve.
- Update PRD/requirements: 5 destinations, workspace model, search, stats,
  review; retire the 11-page OT-P0-014 language.
- Update experience contracts and the route inventory check.
- ADRs: composition ownership, template-as-provenance, page type vs page,
  activity durability, findings disposition authority, search engine, no silent
  local fallback.

## 4. Migration strategy

1. **Freeze the target.** Approve the domain/contracts and deletion ledger before
   writing product code. No new feature work on the old surfaces except critical
   fixes.
2. **Build beside.** New `workspaces`, `search`, `findings`, `activity`, `health`
   packages and proto services; new Start/Library/Design/Review/Stats routes
   behind a flag.
3. **One-time data migration.** Designs → workspaces/pages/displays/states;
   templates → curated workspaces; experience bindings → page links; preserve
   ids where possible; record a migration report and checksums.
4. **Switch routes.** Redirects for published URLs; old pages still readable for
   one release; no dual-write.
5. **Delete.** Remove old tables, services, REST modules, code templates, local
   stores, dead CSS/exports and legacy pages after parity and a rollback window.
6. **Certify.** Run all gates, captures, migration dry-run and rollback drill;
   publish a net-negative line report.

## 5. Program phases

| Phase | Scope | Exit evidence | Deletions |
|---|---|---|---|
| **0. Truth and guardrails** | Re-baseline tests/gates; inventory; fix missing TARGETS; approve ADRs and target contracts; freeze old features | One status page; three Go failures assigned; docs links resolve; target schema/ADRs approved | Stale duplicate plan docs; broken references |
| **1. Contracts and schema** | workspace/search/findings/activity/health proto; migrations; generated clients; adapters; no UI | Proto parity; migration dry-run; repository tests; schema lint | REST design/template exceptions; inline DDL copies |
| **2. Workspace core + Design** | Workspace aggregate; pages/displays/states; revisions; assets; fork; migrate designs/templates/experience bindings; Design index + studio + flow | Harbor 4-page/12-display/12-state flow; template fork; legacy design opens; state evidence; 390/320 | `studio_templates`, `internal/templates`, code templates, local template store |
| **3. Search** | Unified index; FTS5 + `ai-go/search`; commands; Start + Spotlight + Ask; facets/status | Golden queries, p95, degradation, command shortcuts, no client cap | `catalogsearch` as the only index; 4 UI search paths; `searchDesignAssets` |
| **4. Start + Library** | Start Atlas/categories/continue/templates/pulse; Library recent/featured/collections; re-home Home modules | Relocation ledger complete; 5 routes; phone tabs | Home duplication, Coverage/Capabilities/Activity routes, Home-only widgets |
| **5. Review + Stats** | Findings projection/dispositions; activity events; health sections; Review queue; Stats 4 views | All four Stats views; durable attribution; disposition write; honest unavailable | `findingsFromCoverage`, UI activity derivation, 3 separate pages |
| **6. Cleanup and certification** | Delete legacy; fix 3 Go failures; one device/verdict/timestamp/error owner; a11y/performance; docs/experience; captures; net-negative report | All gates green; axe/phone/reduced/forced; no dead code; one canonical docs set | Legacy pages/CSS/exports, duplicate metrics/verdicts, write-path `make check` |

Workstreams can run in parallel after Phase 1:

- **A. Data/API** — contracts, schema, migration, workspace/health/findings/activity.
- **B. Search** — index, ranking, commands, semantic integration.
- **C. UI shell/Start/Library** — 5 routes, shared search, phone.
- **D. Workspace studio** — aggregate UI, flow, states, assets, assist/capture.
- **E. Review/Stats** — findings/activity/health, dispositions, Stats views.
- **F. Quality/evidence** — gates, migration tests, captures, docs, deletion.

Dependencies: A → B/D/E; C can build against contracts; F continuous.

## 6. Quality gates

### Contracts and data

- Proto parity: every RPC has an endpoint descriptor; generated clients compile.
- One error envelope; table tests for code mapping; no plain-text authoring errors.
- Migration dry-run on a copy of the real DB; counts/checksums; rollback works.
- One schema owner per table; no inline DDL; migration ledger idempotent.
- Workspace aggregate tests: create/read/update/delete, page/display/state order,
  revision conflict, fork, cascade, assets, import/export.
- Experience binding tests: page link survives, intent contract untouched.

### Search

- Golden queries and ranking order; no unrelated page types.
- p95 under 100 ms lexical on the full catalog; semantic degradation returns
  lexical results with a truthful status.
- Facets/counts, suggestions, no-match vs index-empty, stale/partial/error.
- Command index includes shortcuts and availability; commands execute.
- No client-side search over a capped asset list.

### UI and accessibility

- Page-level test per route; studio interaction test; command palette; search.
- Axe zero on every route at 1440/390/320.
- Keyboard: focus target for canvas/layer tree, roving tabs, unique Escape,
  focus restore.
- Reduced motion; forced colors; contrast (including 9 px labels and control
  borders); target sizes; text scale.
- Phone captures and no horizontal overflow at 390/320.
- Loading/empty/partial/stale/error/permission/offline states.

### Build and hygiene

- `pnpm test`, lint, type-check, catalog check, build pass.
- `go test ./...` passes; the three current failures fixed or explicitly retired
  with evidence.
- `make check` is read-only; formatters are separate.
- Deletion gates: no references to retired modules; no dead exports/CSS; no file
  over budget.
- Net-negative line report after cleanup.

## 7. Deletion and simplification ledger

| Remove / replace | Replaced by | Why |
|---|---|---|
| `internal/templates` + `/api/templates` + `studio_templates` | Workspace provenance `curated` + fork | One composition store |
| `ui/src/workbench/templates/local.ts` local shelf | WorkspaceService | No browser-only template ownership |
| `ui/src/workbench/templates/library.ts` 8 code templates | Reviewed JSON seed + DB | Data, not code |
| `internal/designs` (after migration) | `internal/workspaces` | One aggregate |
| `sketch` composition (`document`, `store`, candidates, placements, revisions, template render) | `workspaces` + `experience` reader | Intent vs composition split |
| `internal/sketch` god RPC surface | `WorkspaceService`, `Assist`, `Capture`, `Review`, `ExperienceService` | Bounded services |
| `CatalogService` coverage/search/health RPCs | `CoverageService`, `SearchService`, `LibraryHealthService` | Bounded services |
| `catalogsearch` as the only search | `internal/search` multi-source index | Search is not catalog-only |
| Home page duplication | Start + Library + Stats | One home per job |
| `CoveragePage`, `CapabilitiesPage`, `ActivityPage` routes | Stats views | One health surface |
| `findingsFromCoverage` UI projection | ReviewService | Server owns findings |
| `activityFromCatalog` derivation | Activity events | Durable attribution |
| Four verdict renderers | `verdicts.ts` | One vocabulary |
| Three device metric tables | `viewports.ts` | One device truth |
| Hand-built Preview chrome | `DeviceFrame`/`DeviceSwitch` | One chrome |
| 294+ dead CSS lines and dead `.wb-studio*` media rules | deleted | No shipping dead CSS |
| `searchDesignAssets`, unused exports, duplicate URL builders | deleted | No dead API |
| localStorage design/template/fixture stores | API + explicit offline queue or read-only | No silent split brain |
| Inline `reconcile_evidence` DDL (3 copies) | domain schema provider | One schema owner |
| 11-page PRD/experience claims | 5-destination contracts | No contradictory contract |

## 8. Risks and decisions

1. **Composition vs intent ownership.** Recommended: RCL DB owns composition;
   scenario `experience/` owns intent/claims; an explicit reconcile/export links
   them. The alternative keeps two composition stores and should be rejected.
2. **Workspace vs page reuse.** One owner per page for v1; cross-flow reuse later.
3. **Offline authoring.** Recommended: no silent fallback; read-only offline or an
   explicit, promotable IndexedDB draft queue. Do not keep the current fallback.
4. **Activity durability.** Recommended: append-only events with an outbox; if
   deferred, remove actor/attribution claims and keep Activity as an explicitly
   labeled projection.
5. **Findings dispositions.** Need an owner and authorization rules; Review must
   not fake triage.
6. **Search semantic leg.** Local `ai-go/search` only; lexical fallback; no paid
   provider.
7. **Scope control.** Out of scope for this program: multi-user presence/comments,
   native mobile apps, full AI chat/conversation, freeform in-context drag beyond
   the current placement, external asset imports. State them explicitly.

## 9. First ten working days

1. Create the program ledger and status table; fix or retire the missing
   `docs/internal/goal/TARGETS.md` references.
2. Re-run and record UI/Go/lint/type/catalog/build; assign the three Go failures.
3. Approve ADRs: composition ownership, workspace aggregate, template provenance,
   activity durability, findings authority, search engine, no silent fallback.
4. Draft `workspace.proto`, `search.proto`, `findings.proto`, `activity.proto`,
   `health.proto`; generate clients and parity tests.
5. Draft the migration mapping for designs/templates/experience; run it on a DB
   copy; record counts and checksums.
6. Implement the workspace repository and aggregate tests.
7. Implement the FTS5 search index and golden-query tests; add the semantic
   adapter behind the same interface.
8. Replace the shell's global queries with route-level queries; add per-pane error
   boundaries.
9. Build the Start/Library/Design skeleton against the new contracts; keep old
   routes redirecting.
10. Start the deletion ledger: remove dead CSS/exports and the write-path behavior
    of `make check`; do not delete the old design stores until migration parity.

## Appendix A — current inventory

### Routes and owners

| Route | Current owner | Target |
|---|---|---|
| `/` | HomePage | Start |
| `/components` | ComponentsPage | Library (redirect) |
| `/assets/:id` | ComponentPage | Component (kept) |
| `/assets/:id/preview` | PreviewPage | Preview (kept) |
| `/design` | DesignStudioPage | Design workspace index + studio |
| `/design/:scenario/:page` | DesignStudioPage | Workspace/page deep link |
| `/review` | ReviewPage | Review (kept, server findings) |
| `/coverage` | CoveragePage | `/stats?view=quality` |
| `/activity` | ActivityPage | `/stats?view=activity` |
| `/capabilities` | CapabilitiesPage | `/stats?view=services` |
| `/settings` | SettingsPage | Settings (kept) |
| `/new` | CreatePage | Global New action/route |
| `/sketch/...`, `/catalog` | redirects | Keep redirects |

### Stores

| Store | Current | Target |
|---|---|---|
| SQLite `studio_designs` | 2 rows | `workspaces`/`workspace_pages` |
| SQLite `studio_templates` | 0 rows | removed; curated workspaces |
| SQLite `studio_design_assets` | design-scoped uploads | workspace assets |
| Scenario `experience/pages/*.json` | intent + composition | intent only; composition linked |
| `localStorage` templates/designs/fixtures | fallback/drafts | removed; explicit offline only |
| `component_*`, `catalog_gate_evidence`, `version_ledger`, `assisted_workflows` | existing owners | kept |

### Evidence baseline

- UI: 123 files / 750 tests pass.
- Lint: 0 errors; 5 catalog warnings.
- Type-check/catalog: pass.
- Go: 3 failures in `internal/gates` and `internal/reconcile`.
- Captures: 11 routes have phone/desktop captures and axe at 390/320 per PROGRESS;
  full interaction-state phone journeys remain incomplete for 10.
- Requirements: 54 complete, 36 in progress, 16 planned, 16 not implemented,
  3 pending.
