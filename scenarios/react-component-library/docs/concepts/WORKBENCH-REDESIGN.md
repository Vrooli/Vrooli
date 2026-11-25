# Workbench redesign — D · Nocturne

> **Status: implemented in source and served on 2026-11-19; governed capture evidence and full phone certification remain pending.** This document is not a current-behavior claim. Current behavior is asserted only in the [behavior claim register](../internal/TESTING.md#behavior-claim-register). The workbench implementation is in `ui/src/workbench/`. See [Implementation record](#implementation-record-2026-11-19) for what was proven and what is still blocked.

> **2026-11-19 maturation direction — supersedes the seven-destination IA below.** The operator selected a five-destination workbench (**Start, Library, Design, Review, Stats**), a unified Stats page absorbing Coverage/Activity/Capabilities, and a **Workspace → Page → Display → State** design model. The implementation record and behavior claims in this document remain current; the IA and model changes are proposals in [WORKBENCH-MATURATION.md](WORKBENCH-MATURATION.md), with the prototype at [workbench-maturation.html](../mockups/workbench-maturation.html).

## Decision

The operator selected **D · Nocturne** for every page of the React Component Library workbench after reviewing three initial directions (A · Atelier, B · Obsidian, C · Porcelain) and a two-theme expansion (B and D).

The implementation now lives in `ui/src/workbench/`: the shell, eleven page surfaces, the Fixtures and In-context component workspace, the Asset data drawer, the Design studio, and eight page templates built from RCL assets. The documentation below remains the contract and acceptance path; it is not a production-ready claim until the evidence, phone composition and data-contract gates pass.

## Implementation record (2026-11-19)

### Shipped and exercised

| Surface | What changed | How it was exercised |
|---|---|---|
| Shell | Nocturne rail, top bar, command palette, mobile bottom navigation | served at 1440, 390 and 320; no horizontal overflow on any of the eleven routes |
| Home | launchpad with real continuation, health, findings, recent/new assets and template entry | served against the live catalog and coverage projections |
| Components | kind tabs, search, domain/status filters, list/gallery/tree, preview-failure fallback | real 275-asset catalog; preview 404s now resolve through the manifest category |
| Component | Fixtures specimen bench grouped by story role with size-to-fit frames; In context with typed slots, ranked candidates and a visible fallback; props overrides; evidence | bench rendered IconButton and Card at measured size and collapsed redundant pins; typed slots scored action/metric/list and kept page health separate from component fit; prop override applied to the real harness |
| Asset data drawer | Overview, Source, Versions, Experience, Tests, Relationships with deep links | opened from the component metadata row and by `?tab=` |
| Design studio | scenario/page/layers, artboard, inspector, contract, compare, present, local templates, theme palette, add/move/duplicate/remove layers | edited text, added a chip, duplicated a layer, duplicated a template, changed palette, presented, accepted a revision |
| Review | severity board projected from real failed gates | 251 findings from the live coverage report |
| Coverage | maturity matrix, definition of ready, ranked next work | live report; cold-start warming state shown instead of zero |
| Capabilities | integration state, what it enables, recovery command | live registry |
| Activity | index-derived trail with an explicit durability boundary | 550 bounded events, 200 rendered |
| Settings | browser-local density, device, motion, freshness, viewports, locale | persisted locally; scenario policy boundary stated |
| Create | explained Create / Extract / Adopt routes | real `initializeComponent`, `ingestComponent`, `preflightAdoption`, `linkAdoption` wiring |
| Preview | chrome-free share surface | story, device, checks and back link |

### Validation performed

- `pnpm build` (experience route guard, library pins, token census, selector manifest, string registry, usage-contract types, Vite build): pass.
- `pnpm run catalog:check` (token check, export map, strict catalog TypeScript, catalog ESLint): pass.
- UI unit suite: 95 files / 462 tests pass.
- Go `handlers/preview`, `handlers/catalog`, `handlers/components`, `internal/components`, `internal/catalogcoverage`: pass; `internal/gates` retains only the two accepted released-artifact findings.
- Browser sweep: every route at 1440, 390 and 320; no horizontal document overflow, no page errors. The only console noise is two honest `404`s for asset previews that have no captured image (`TimeMention`, `RunLadder`), which the gallery renders as “Preview pending”.
- Governed source editor exercised in Chromium: opened a draft, appended and saved through `UpdateComponentContent` with the read digest, ran `UpdateComponentContent` → `CheckComponentVersion` to a passing draft, then restored the exact original bytes (SHA matched) and re-checked. No console errors. Monaco is served from the scenario bundle; the editor chunk is fetched only on first edit.
- Phone authoring: Source opens at 390/320 with a working Monaco surface, no overflow. Design studio at 390/320 exposes the layer tree and inspector as bottom sheets, and adding, editing, duplicating, removing and accepting a revision all worked. In-context defaults to the phone frame on a narrow viewport and the slot drag/resize stayed inside the frame.
- Coverage cold path: the first full gate pass is minutes, so the API now persists its report (`~/.vrooli/data/vrooli/react-component-library/coverage-report-v1.json`), serves stale while refreshing, coalesces cold callers and returns an explicit “being prepared” state. The coverage page polls and never renders zero while waiting.
- Governed capture: all eleven routes captured through BAS at phone (390) and desktop (1280), dark theme. Captures live in `~/.vrooli/data/react-component-library/ux-captures/workbench-20261119/<route>/` with the BAS bundle id, pixel SHA-256 and frozen page. Component and Preview use `[data-fixtures-ready]` / `[data-fixture-status="ready"]` wait selectors so the real harness is in frame.
- Accessibility: axe-core reports **zero violations** on every route at 1440 and 390. Fixed critical button names on icon-only segmented controls, tablist structure, decorative-landmark duplication, scrollable-region focusability, heading order, and the muted-metadata contrast token (3.2:1 → 4.9:1). Added a focus trap with Escape/restore for the command palette, Present mode and the phone studio sheets; Tabs now use a roving tab index with Arrow/Home/End.
- Reduced motion: with `prefers-reduced-motion: reduce`, no element in any route (or its harness iframes) has an animation or transition above 10 ms, and scroll behavior is `auto`. The Settings “Reduce motion” switch forces the same contract through `data-wb-reduce-motion`.
- Forced colors: every route was rendered under `forced-colors: active`; no text with matching foreground/background was found. Primary buttons were painting their label as a solid block, so they now opt out and keep a 12:1 pair. Chart swatches, meters, matrix cells and dividers gain explicit `CanvasText` outlines so graphical meaning survives the palette flattening.

### Still open

- **State breadth in governed captures** — each route has a default phone and desktop capture. Interaction states beyond the default (L6) are only captured when a `--step` is supplied; the workbench does not yet script every state for every route.
- **Lens partials** — outline captures exceed the 8000-token outline budget on dense pages (L2 partial), and the Nocturne token scale is not projected into the RCL token family the style census recognizes (L3 partial). Neither is a missing capture.
- **Phone state coverage** — all eleven routes were exercised at 390 and 320 for overflow/navigation; the source editor, studio sheets, in-context drag/resize and keyboard traps were exercised directly. Every loading/empty/partial/error state is not separately scripted.
- **Pre-existing corpus findings** — five releases have no provenance record, `AppNotificationSettings@1.0.6` still carries library-owned utility classes, and the published `CommandCenterShell@1.0.5` type omits the `metrics[].id` its contract now requires. Coverage surfaces them as findings; they are not redesign regressions.
- **Activity durability** — Activity remains an index-derived projection. A durable event store is still required before claiming human attribution.

D is a dark, warm editorial workbench:

- deep charcoal/plum surfaces;
- amber signal and selection, ice-blue secondary emphasis;
- serif display headings, sans-serif working UI, mono metadata;
- a gallery for Components, a severity board for Review, and a domain × maturity matrix for Coverage.

B · Obsidian remains the technical/dense reference and the comparison direction. It is not the implementation target.

The selected prototype is:

```text
~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/data/option-d-nocturne.html
```

The page-by-page decision review is:

```text
~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/data/rcl-workbench-redesign.html
```

## Two proposal artifacts, two jobs

The experiment produced two presentations. Keep both:

1. **Full workbench prototype** — one self-contained HTML file with all 11 pages, navigation and interactions. It mimics the actual UX and is the canonical design artifact. A reviewer can experience the proposed flow instead of reading it.
2. **Page-by-page review** — a comparison and decision artifact with screenshots, page-model differences, recommendations and the feature-preservation ledger. It is how a direction is chosen and how loss is audited.

The prototype proves the experience. The review proves the decision and preservation. Neither replaces the other.

## Why this redesign

The served workbench on 2026-11-18 had defects that are not cosmetic:

- `/coverage` returned “Coverage unavailable”; no maturity distribution or ranked next work rendered.
- Button 2.2.14 had no version history and no empty-state explanation in the Versions tab.
- Button 2.2.14 had no component-test evidence, while older evidence remained visible as `skipped` or `stale`.
- The phone preview toolbar clipped its appearance and viewport controls.
- TimeMention and RunLadder previews were pending.
- The usage-contract type map was stale.
- The “Main actions” dialog offered three unexplained text links.
- Seven peer asset tabs competed with the preview.

The redesign must make the workbench beautiful and usable, but it must also make these failures visible and recoverable rather than hiding them behind a cleaner surface.

## Design principles

1. **The artifact is the hero.** Every page starts from the rendered component, the real page, or the finding. Metadata and evidence support the artifact.
2. **One strong idea per page.** Home is a launchpad; Components is finding an asset; Review is what needs a person; Coverage is how mature the library is; Create is three governed routes.
3. **Evidence is ambient.** A check, ring, stamp or status chip lives on the thing it describes. Detail opens in context. There is no seven-tab evidence maze.
4. **Preserve before invent.** Every current capability has a new home before a new surface is added. The [feature ledger](#feature-preservation) is the guardrail.
5. **Use the library.** The redesigned workbench must be built from current RCL assets. A synthetic prototype is not an implementation model; missing shared parts are extracted, not copied.
6. **No unexplained actions.** Create, Extract and Adopt say what they do, what they need and what changes.
7. **Honest states.** Missing capture, stale evidence, an unavailable service and zero proven bindings never become a green check.
8. **Dark is a workbench, not a library theme.** Nocturne styles the workbench chrome. Published components keep the shared BaseStyles contract.

## Information architecture

### Target routes

| Route | Page | Current route | Migration |
|---|---|---|---|
| `/` | Home | `/` currently catalog workspace | Home becomes the index; catalog moves to `/components` |
| `/components` | Components | `/components`, `/catalog` | `/catalog` redirects to `/components` |
| `/assets/:id` | Component | `/assets/:id` | Preserved; query state for mode and drawer |
| `/assets/:id/preview` | Preview | Same | Preserved |
| `/design` | Design studio | `/design`, `/design/:scenario`, `/design/:scenario/:page` | One studio; legacy routes redirect with scenario/page context |
| `/review` | Review | Not present | New |
| `/coverage` | Coverage | Same | Preserved |
| `/capabilities` | Capabilities | Same | Preserved |
| `/activity` | Activity | Not present | New |
| `/settings` | Settings | Same | Preserved |
| `/new` | Create | Floating Main actions dialog | New route; the dialog becomes a menu that links here |

The route map is a target, not current behavior. The implementation must preserve old deep links or redirect them deliberately; no route may silently become a 404.

### Navigation

The desktop rail contains, in order:

1. Home
2. Components
3. Design
4. Review
5. Coverage
6. Capabilities
7. Activity
8. Settings

`New asset` is a primary action in the rail foot and the Components header. The rail collapses; the mobile composition is defined below, not inferred from the desktop rail.

## Page responsibilities

### Home

**Job:** help a maintainer resume work, understand library health, and start a component or a page.

**Hierarchy:** greeting and live status → primary search → continue designing → library health → recently verified → new in the collection → capture queue → design studio entry.

**Required content:** continue card with scenario/page, draft revision and checks; health ring and freshness; open findings; recent assets; new assets; capture entry. No anonymous catalogue wall.

**States:** default, loading, empty (new workspace), partial (health unavailable), error (catalog unavailable).

### Components

**Job:** find an asset and open it without losing the distinction between components, hooks and support assets.

**Hierarchy:** title and count → kind tabs → search and filters → result set → collections and coverage rail.

**Required content:** Components / Hooks / Support assets; search; domain, kind, status and sort filters; list, gallery and tree controls; adoption count; evidence freshness; pending/failed preview state; `New asset`.

**States:** default, loading, empty filters, request error, pending preview, stale evidence.

### Component

**Job:** understand one immutable asset version and see how it behaves in isolation and in a page.

**Two modes:**

- **Fixtures** — a free canvas of all declared stories. Drag to arrange; select to inspect; edit public props; re-run the selected fixture. Anatomy, axis and boundary roles remain visible. The canvas is a workspace, not a card grid.
- **In context** — the same component inside a real page. Switch desktop/tablet/phone; move and resize within the page; see fit and contract checks; take the composition into the Design studio.

**Asset data drawer:** Overview, Source, Versions, Experience, Tests, Relationships. This is the preservation path for the current seven tabs. The drawer is contextual; it does not compete with the preview.

**Inspector:** selected fixture or context placement; check ring; checks; props; usage; evidence links; run/re-run.

**States:** preview loading, ready, partial (one fixture failed while another works), error, unmeasured, stale evidence, no test evidence for the current release.

### Design

**Job:** compose and review a scenario page from real assets, and present the result as a credible design.

**Hierarchy:** scenario → page → layers; artboard; inspector; contract; compare; revisions and collaborators.

**Required content:** scenario/page tree; layer tree; device switch; brief; artboard; selection and handles; findings; page contract; before/after and fidelity label; present mode; accept composition / request review; revision and saved state.

The current `/design`, `/design/:scenario` and `/design/:scenario/:page` routes collapse into one studio. The existing brief → compose → review → adopt sequence remains the workflow spine.

### Review

**Job:** show what needs a person, ordered by blast radius.

**Required content:** open findings, failing assets, stale evidence, resolved-this-week; queue or severity board; finding title, asset, type, severity, age; selected finding with why-it-matters, evidence links, affected asset and triage/assign actions.

**Queue scope:** Review projects one finding per check that is failing on a published component's current release or open draft. The newest observation wins, so a later pass clears the finding. Passing, unmeasured, skipped, corpus and superseded-version evidence remain in Coverage; they are not review work. A refresh prunes open findings the projector no longer implies, while human decisions and their reasons survive.

**States:** default, empty, loading, request error, stale queue, permission-denied for writes.

### Coverage

**Job:** answer “how mature is the library, and what should we work on next?”

**Required content:** evidence freshness, asset count, production-ready count, next-work count; maturity distribution (D uses a domain × maturity matrix); definition of ready; ranked next work with impact, blast radius and effort; domain breakdown; measured time; refresh; filter.

**States:** ready, loading, unavailable (service did not complete), stale, empty ranking. Unavailable must never render as zero or “all good”.

### Capabilities

**Job:** show which integrations the library can rely on and how to recover a missing one.

**Required content:** available count, last checked, service cards with state, what it enables, recovery command; unavailable/degraded states; dependency map link.

### Activity

**Job:** show what changed and who changed it.

**Required content:** publishes, adoptions, evidence, resolved findings, retries; actor; time; asset/version; week summary; filters.

**States:** default, empty, loading, request error.

### Settings

**Job:** adjust browser-local workbench preferences without implying control over library policy.

**Required content:** appearance (theme, density, accent, preview text size); evidence defaults (freshness window, capture viewports, run on publish); notifications; language; local-data export/reset; explicit local-vs-scenario boundary.

### Create

**Job:** provide three explained, governed routes into the same lifecycle.

**Required content:** Create component / Extract from library / Adopt into scenarios; description of each; form fields; source provenance; draft version; “what happens next” steps (draft, derived artifacts, focused checks, publish); summary and primary action.

**States:** default, validation error, source analysis running, analysis failed, permission-denied.

### Preview

**Job:** a chrome-free share surface for a component story.

**Required content:** back to asset; story; device; checks; share; the rendered artifact. No application navigation.

## Component workspace details

### Fixtures canvas

- The canvas uses a dotted/grid surface and absolute fixture placement.
- Fixtures are cards with a name, role, rendered specimen, execution state and check summary.
- Dragging must not interfere with pressing the specimen. Selection and drag are separate intents.
- Selecting a fixture updates the inspector; it does not navigate away.
- Props edit public argument paths only, through the same validated effective-story model as the current editor.
- Reset restores the named baseline.
- The canonical saved arrangement is local; an unsaved arrangement must not imply a release change.

### In-context mode

- Device presets are desktop, tablet and phone; width is explicit and persisted for the session.
- The component sits in the template's declared slot with a visible slot label. The slot is live (the real harness), and the studio owns its placement. Freeform move/resize inside the page remains a future step and is not claimed.
- Fit checks evaluate the component against the page contract, not against the component alone.
- “Take into studio” carries the component, version, story, data and template into the Design studio.

### Template shelf

In-context and the Design studio share one declarative template shelf. A template is data, not JSX: a tree of layout, text, control and slot nodes with a palette and a display family. The studio edits the same tree it renders.

The shelf ships eight compositions built from RCL assets:

| Template | Category | Design idea |
|---|---|---|
| Aperture | Analytics | A calm signal room: one number leads, the rest support it |
| Marlow | Commerce | A shop that reads like a magazine: one object, one sentence, one invitation |
| Solstice | AI | A quiet intelligence: one prompt, a short answer, no dashboard noise |
| Verdant | Climate | Proof before poetry: the number is the hero, the method is visible |
| Atelier | Portfolio | The work speaks first; the index is a quiet second voice |
| Pulse | Health | Care, not compliance: a soft surface that asks one question at a time |
| Harbor | Onboarding | A threshold, not a funnel: one promise on the left, one step on the right |
| Nocturne Ops | Operations | Dense, legible, unromantic: what is on fire and what to do |

Each template declares a component slot with a hint (for example “Add-to-bag action on the featured product”). The slot renders the real component harness, never a mock. The studio can select any layer, edit text, variant, tone, alignment, gap, columns, padding and elevation, add or remove nodes, accept a revision, compare against the baseline, present the composition, and hand the same template and component into the studio from In context.

Templates are **samples**, not measured data. The `Stat` assets inside them use `provenance="sample"`; the art is CSS-only, not stock photography.

### Asset data drawer

The drawer tabs and their preserved content:

| Drawer tab | Current source | Required content |
|---|---|---|
| Overview | Overview tab | library ID, slot, category, tags, source hash, design affinities, shared assets |
| Source | Files tab | version-scoped read-only source; begin/open a draft, edit any version artifact in locally bundled Monaco, save with an optimistic-concurrency guard, run the focused version check, and publish through it |
| Versions | Versions tab | version list, current/latest, diff selectors, capture links |
| Experience | Experience tab | claims, state scope, tier, freshness, capture link |
| Tests | Tests tab | verdict, screenshot, accessibility tree, computed styles, layout, console, performance |
| Relationships | Relationships tab | dependency direction, closure rungs, direct/transitive consumers |

The drawer must deep-link (for example `/assets/:id?tab=versions`). The current query contract already supports `tab`; preserve it or map it deliberately.

**Source editing is governed, not a direct file save.** A released version is read-only. Beginning a draft calls `BeginComponentVersion`; editing writes one artifact through `UpdateComponentContent` with the digest captured by the read (`expected_sha256`), so a concurrent change surfaces as a conflict instead of an overwrite. `CheckComponentVersion` re-indexes just that manifest and returns stage verdicts, and `PublishComponentVersion` repeats the check, compiles the preview, and only then promotes the draft. Unsaved buffers are stashed in `sessionStorage` per asset + version + file, so switching versions or leaving the page does not silently lose edits. Monaco is bundled locally (worker + TypeScript language service) and fetched lazily on the first edit, so the editor adds no cost to ordinary browsing and no CDN dependency.

## Nocturne design system

Nocturne is workbench chrome. It is not a replacement theme for published components.

### Surfaces and color

| Role | Intent |
|---|---|
| Canvas | near-black with a faint warm tint |
| Panel | slightly lifted charcoal/plum |
| Border | low-contrast warm gray |
| Text | warm white |
| Muted text | warm gray, never below AA for body text |
| Signal | amber for selection, primary action and live state |
| Secondary signal | ice blue for technical emphasis and comparison |
| Success / warning / danger | green / amber / red, always paired with text or icon |
| Focus | visible amber ring with sufficient contrast on every surface |

Do not rely on color alone. Do not use the amber signal for warning semantics; warning uses its own amber-orange with a label.

### Typography

- Display headings: Iowan Old Style / Palatino / Georgia fallback stack, weight 600, tight tracking.
- Working UI: Inter or the platform sans stack.
- Metadata, code, hashes, paths and tabular metrics: monospace.
- Body text remains at least 14px in dense desktop panels; 16px on phone. Controls keep usable target sizes.
- Serif is a hierarchy tool, not body text. Do not set long prose in serif.

### Geometry

- 12–16px panel radius; 8–10px control radius; 1px borders.
- 4/8/12/16/24 spacing scale.
- Soft elevation on floating panels; no heavy drop shadows on every card.
- Motion is 120–200ms ease-out for selection, panel and drawer transitions; respect reduced motion.

### Token ownership

The workbench cannot silently author a second palette. The implementation must choose one of these explicitly:

1. Add the workbench chrome roles to BaseStyles and publish a new release, then generate consumers; or
2. Define a small, scoped workbench token set in `ui/src/workbench-tokens.css` for chrome-only surfaces, accent, and text, collision-checked against BaseStyles and explicitly forbidden from being imported by published components.

The preferred path is (2) for workbench chrome, because Nocturne is an application identity, not a shared component theme. The token check must be extended to prove the scoped set does not leak into the library. This is an implementation decision, not a current behavior claim.

## Data and contracts

The design is only honest if these owners exist:

| Need | Owner / contract | Current gap |
|---|---|---|
| Review findings queue | A findings projection over gate evidence, component tests, provenance and capture state | No dedicated queue; current tabs expose fragments |
| Coverage report | Catalog coverage service | Current page can return “Coverage unavailable” |
| Version history and diff | Versions service | Current Button 2.2.14 history is empty |
| Current-release test evidence | Component tests / BAS | Button 2.2.14 has none |
| Usage-contract type map | Generated contract types | Current map is stale |
| Experience page contracts | Experience Manager / RCL experience docs | New Review/Activity/Create pages need authored drafts |
| Capture links | BAS / UI Health | Evidence must remain inspectable, not just a verdict |

The UI must expose `unmeasured`, `unavailable`, `stale`, `partial` and `permission-denied` explicitly. A missing producer is not an empty result and is not a pass.

## Feature preservation

| Current surface | New home | Disposition |
|---|---|---|
| Catalog workspace and kinds/views | Components | Preserved + enhanced |
| Component Preview/Overview/Files/Versions/Experience/Tests/Relationships | Component + Asset data drawer + Preview | Preserved, re-grouped |
| Component harness | Machine iframe target | Preserved as infrastructure |
| Design chooser/workspace | Design studio | Reimagined, same jobs |
| Coverage | Coverage | Preserved + enhanced |
| Capabilities | Capabilities | Preserved + enhanced |
| Settings | Settings | Preserved + enhanced |
| Main actions dialog | Create | Reimagined |
| Preview popout | Preview | Preserved |
| Mobile bottom navigation | Not designed in the prototype | **Open gap; required before production** |

No current capability is intentionally dropped. The full ledger is in the experiment folder:

```text
~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/COMPONENT-LEDGER.md
```

## Migration plan

### Phase 0 — Contracts and documentation · **done**

- Record the D selection and the two proposal artifacts.
- Update the PRD target, requirements module and experience drafts.
- Write this plan, the README and the design/branding note.
- Add current defects to `PROBLEMS.md`; no runtime claim.

**Exit:** documentation validates; no route or behavior claim changes; the feature ledger is complete.

### Phase 1 — Shell and Home · **done, capture pending**

- Implement the Nocturne shell, rail, top bar, command/search surface and Home launchpad.
- Wire only real endpoints; unavailable services render explicitly.
- Add the mobile shell composition.

**Exit:** served Home at 1440/390/320; route contract; experience story; no horizontal overflow; keyboard navigation.

### Phase 2 — Components and Component workspace · **done, capture pending**

- Components kind tabs, search/filter, list/gallery/tree.
- Component Fixtures and In context modes.
- Asset data drawer with deep links.
- Preserve current evidence and version data.

**Exit:** every current asset tab is reachable; fixture drag/select/props; in-context device switch; evidence links; no lost feature in the ledger.

### Phase 3 — Design studio · **done, capture pending**

- Scenario/page/layer tree, artboard, inspector, contract, compare, present.
- Migrate the three legacy design routes.
- Preserve brief, composition, findings, verify and adoption handoff.

**Exit:** a fresh agent can open a page, compose, verify and present; the old routes redirect.

### Phase 4 — Review, Coverage, Capabilities, Activity, Settings, Create, Preview · **done, capture pending**

- Build the remaining pages against real projections.
- Add the findings queue and activity projection.
- Make unavailable/stale/empty states honest.

**Exit:** all 11 routes served; no page uses sample data; every write action has authority.

### Phase 5 — Responsive, accessibility and state coverage · **partial: overflow/nav verified, journeys not captured**

- 390 and 320 phone compositions; bottom navigation; sheets for inspector/drawer.
- Focus, keyboard drag alternative, reduced motion, forced colors, long labels.
- Loading, empty, partial, stale, error, permission-denied.
- Contrast verification for Nocturne.

**Exit:** accessibility gates and phone journeys pass; no hover-only action.

### Phase 6 — Evidence and experience contracts · **partial: contracts and stories done, capture evidence blocked**

- Add page stories for every new route.
- Author experience pages for Review, Activity and Create.
- Capture the journeys; add claims and bindings.
- Update the feature ledger with proof.

**Exit:** route guard, page stories and experience reconciliation pass; evidence is fresh for the shipped build.

### Phase 7 — Adoption and rollout · **not started**

- Dogfood the workbench on its own UI.
- Verify the shared RCL components used by the shell are current.
- Roll out to no consumer until the shell and data contracts are stable.

**Exit:** the workbench is self-hosted on current RCL assets; no synthetic chrome; operator review of the served build.

## Acceptance

A page is ready only when:

- it is served at its target route and the old route redirects;
- its experience contract and page story exist;
- its loading, empty, partial, stale, error and permission states are designed and captured;
- phone and desktop captures pass;
- every current feature in its ledger row is reachable;
- the evidence it shows is real, fresh and inspectable;
- the page uses current RCL assets for shared controls and surfaces;
- no sample data is presented as measured data.

“Beautiful” is necessary but not sufficient. The workbench is production-ready only when the data contracts, phone composition and evidence are proven.

## Non-goals

- No new runtime semantics for component publication or adoption.
- No change to immutable releases or release hashes.
- No AI editing, capture kernel or evaluation-engine work in this redesign; those remain separate workstreams.
- No replacement of BaseStyles as the shared component token authority.
- No synthetic component markup in the implementation.
- No mobile-as-shrunk-desktop.

## Open questions

1. **Workbench chrome token ownership** — scoped `ui/src/workbench-tokens.css` vs a BaseStyles release. Default recommendation: scoped, collision-checked, never imported by published components.
2. **Review queue ownership** — new projection in RCL vs composition through existing catalog/coverage/component-test owners.
3. **Create route name** — `/new` is the current proposal; confirm before implementation.
4. **Drawer deep-link vocabulary** — preserve the existing `?tab=` or introduce a new `?panel=`.
5. **Mobile navigation** — bottom tabs vs sheet; must be designed before Phase 5 exits.
6. **Activity retention** — how far back the trail reads, and whether it is a projection or a durable event store.

## References

- Operator experiment and artifacts: `~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/README.md`
- Feature ledger: `~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/COMPONENT-LEDGER.md`
- Selected D prototype: `~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/data/option-d-nocturne.html`
- Page-by-page review: `~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-workbench-redesign-20261118/data/rcl-workbench-redesign.html`
- [Architecture](ARCHITECTURE.md)
- [Flows](FLOWS.md)
- [Gestures](GESTURES.md)
- [Story contract](STORY-CONTRACT.md)
- [Style ownership](../reference/style-ownership.md)
- [Token contract](../reference/token-contract.md)
- [Checked behavior register](../internal/TESTING.md#behavior-claim-register)
- [Known problems](../internal/PROBLEMS.md)
