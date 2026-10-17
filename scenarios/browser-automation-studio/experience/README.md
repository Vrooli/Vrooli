# Browser Automation Studio — experience contract

BAS automates its own UI. Every region declared here is one its `bas/` cases can wait on by
terminal lifecycle state instead of by selector-presence timeout, so a broken step is detected
in milliseconds rather than after a 15-second `waitForSelector`.

## Declared today

| Page | Route | Region | Lifecycle | Backed by | Proven |
|---|---|---|---|---|---|
| `dashboard` | `/?tab=projects` | `projects-grid` | loading, ready, empty, error | `domains/projects/ProjectsTab.tsx` | **yes — exercised in a run** |
| `project-detail` | `/projects/:projectId` | `project-workflows` | loading, ready, empty, error | `domains/projects/ProjectDetail.tsx` | not yet — see below |

A run now reports `readiness_strategy=declared-surface` with
`route=/?tab=projects` and `required_surfaces=[projects-grid]`, so the contract is
consumed by the suite and not only by capture. Passing foundation cases land at
2.9–8.1s. The long failures that remain are all downstream of `project-detail`,
which is still unproven for the reason below.

`experience-manager spec validate browser-automation-studio` **passes**: no errors, no warnings.
The six remaining findings are `capture_unavailable` INFOs for `project-detail`, because
`/projects/:projectId` has no stable URL — see "Proving project-detail" below.

Fifteen `bas/` cases carry `metadata.labels.spec_entry_id` (7 `dashboard`, 8 `project-detail`),
assigned by each case's stated subject rather than by which pages it happens to traverse.

## Proving project-detail

`StateSetup` is deliberately URL-shaped only, so a page reachable solely by an entity id cannot
be captured without a stable URL. The demo project's id is server-generated and not
deterministic (`bas/seeds/` records whatever it got, and that record has already drifted from
the live row), so there is nothing stable to declare. Closing this needs one of:

1. a caller-specified id on `CreateProjectRequest` so the seed can pin a known project id;
2. a resolvable alias route such as `/projects/demo`, which changes product URL semantics; or
3. param resolution in `setup.route`, which is a new experience-manager capability.

All three cross a boundary this scenario does not own, so it is a decision rather than an edit.

Both render `ExperienceSurface` (adopted from React Component Library at
`ui/src/components/ExperienceSurface.tsx`), which emits `data-experience-surface` and
`data-experience-state`.

**Nothing is declared here that is not rendered.** A declared region the UI never renders is
worse than no declaration at all: a consumer builds a wait against a surface that never appears,
turning a fast failure into a guaranteed timeout. Regions are added to this spec in the same
change that wires them.

## Deliberately not declared yet

`workflow-editor` (builder canvas, node palette), `all-executions`, `all-workflows`,
`record-mode`, and `settings`. The dashboard and project-detail pair was chosen first because it
is the exact chain the foundation suite dies on: `navigate-dashboard → dismiss-tutorial →
click-projects-tab → wait-for-dashboard → click-project → open-workflows-tab`.

## Notes for the next scenario that adopts this

Things that cost time here, in the order they were hit:

1. **The catalog id and the spec id differ.** The spec's `component.library.component` is
   constrained to kebab-case (`experience-surface`), while the RCL catalog and its CLI want
   `react-component-library:ExperienceSurface`. Preflight fails with "component not found" until
   you use the fully-qualified PascalCase id.
2. **Adoption is blocked by a maturity floor nothing meets.** `adoptions preflight` returns
   `blocking=true` with `maturity_rung=missing` against `maturity_floor=verified`, and no asset
   in the 428-item catalog is at `verified`. `dependency_verdict` was `ok` and all 74 required
   tokens were already defined, so the block was purely the rung. Adopted with
   `--override-validation true`.
3. **Adoption writes an index-invalid document.** `adoptions apply` drops
   `experience/components/<component>.json` into the target scenario but does not add it to
   `experience/index.json`, so validation immediately fails `experience.index_parity`.
4. **That component document cannot pass in a consuming scenario.** It carries a relative
   `storyRef` whose specimen is not copied (`experience.ref_unresolved`), and its per-state
   bindings need a component harness the consumer does not have
   (`experience.capture_bindings_unjoined`). Copying the specimen in does not help — every file
   under `experience/components/` is itself treated as a document requiring an index entry. The
   component contract belongs in the library that owns and harnesses it; it was removed here.
5. **States must be reachable by URL.** The first draft declared dashboard states at bare `/`,
   which renders the `home` tab, so the projects grid was not in the document and every claim
   failed. `setup.query` pinning `?tab=projects` fixed it. A scenario whose tabs are not
   URL-addressable cannot express its states in this contract at all.
6. **A claim with no `elements` fails with a generic message.** `element-present` resolves its
   subject through `claim.elements` → `bindings.elements` → a declared `elements[]` role. Omit
   `elements` and the claim reports "measured subjects did not satisfy the declared requirement"
   rather than saying it declared no subject.
7. **A surface needs an accessible name to be measurable.** Claims match against the
   accessibility tree, so a bare `<section>` does not resolve. Both surfaces pass `aria-label`,
   which is better accessibility and the thing that makes the claim provable.
8. **Geometry floors were measured on an animating page.** Reconciliation captures did not
   request reduced motion, so an element mid-transition could report an off-viewport position
   and fail a page that is fine once settled. Reconciliation now defaults to
   `motionPreference: reduce`; a target that declares its own preference still wins.
9. **A declared `loading` page-state cannot be captured once readiness works.** The capture
   waits for the region to reach a terminal state, so it will never snapshot the page mid-load
   and the state fails with `capture_bindings_unjoined`. The region's *lifecycle* still declares
   `loading` — that is its state machine — but the page's capture matrix must not ask for it.
   Both pages dropped their `loading` state.
10. **Restarting React Component Library empties its component index.** Every adoption command
   then fails with `component "X" not found`, which reads as "this component does not exist"
   rather than "the catalog is empty". Run `react-component-library components index` after a
   restart. This cost real debugging time here: it made a fixed id-resolution bug look unfixed.

11. **The suite cannot run at all until routed test isolation passes, and the message does not
   say so.** Every case reports `refused: mutating workflow execution requires proven routed
   test isolation`, which reads as a workflow-authoring problem. The actual gate is
   `storage-manager validate scenario <name>` returning PASSED. BAS was failing it on
   `DATABASE_PATH_FROM_ENVIRONMENT` because `api/database/connection.go` accepted a generic
   `DATABASE_URL` as a `file:` DSN. Fixed by resolving through `storage.SQLiteDSN` on the
   scenario identity; `BAS_SQLITE_PATH` stays because it names this scenario and the desktop
   bundle sets it. Check the gate FIRST when a whole suite refuses.

12. **That validation outlives its own client.** `storage-manager validate scenario` took 138s
   and then 360s on this host, and the CLI's HTTP client gives up before the server answers.
   The verdict is only visible by calling
   `http://localhost:<storage-manager>/api/v1/validation/validate/scenario/<name>` directly with
   a longer timeout.

13. **A registry `@selector/` reference can never satisfy region coverage.** Workflow Health's
   `assetCoversRegion` matches the RAW selector text, so `@selector/projects.gridSurface` does
   not contain the binding testid and does not count. Raw selectors normally earn a
   `selector_unregistered` warning, but `isAuthoredExperienceBinding` exempts exactly two forms
   for a testid binding: the bare testid, or `[data-testid="<id>"]`. The compound state
   selectors the lifecycle check needs are NOT exempt, so each one costs a warning. Warnings do
   not block, but the two checks do pull against each other.

14. **Lifecycle coverage requires a `loading` reference, which is racy to assert directly.**
   `assetsCoverLifecycle` needs both a `loading` and a terminal state mentioned in some
   selector on the route. Waiting for `loading` is a race — it may already be past. The honest
   way to satisfy it is an `ASSERTION_MODE_NOT_EXISTS` on the loading state, which doubles as a
   real check that the surface settled.

15. **A failed run leaks its test-pool lease.** The next run then fails with
   `install target test pool: already_exists: test pool already installed under lease "<old-run-id>"`.
   `workflow-health validate abort <old-run-id>` clears it.

17. **A tab-scoped region must not be declared required at the bare path.** The dashboard's
   route is `/`, but `projects-grid` only mounts under `?tab=projects`. Route matching
   originally discarded the query, so navigating to `/` resolved the region and injected a wait
   for a surface that could not be in the document yet — turning three PASSING cases into
   30-second failures. Two changes fixed it: `PageMatchesRoute` now treats a page whose runtime
   routes all carry a query as scoped to those queries, and the 14 cases that go on to use the
   Projects tab navigate straight to `/?tab=projects` instead of navigating to `/` and clicking.

18. **A readiness wait must never fail a case.** The injected node now carries
   `continue_on_error`, waits for `ATTACHED` rather than `VISIBLE` (an `empty` or `error`
   surface may render nothing visible), and is bounded at 8s instead of the 30s driver default.
   This is the invariant that makes the contract safe to enable fleet-wide: it can make a run
   faster or more informative, and it must never make one fail that would otherwise pass.
   `TestInjectedWaitCannotFailACase` locks it.

16. **Settle only inspects the top-level flow, so most cases report a fallback they did not
   take.** 38 of 57 cases have no navigate node of their own; they delegate to
   `actions/open-demo-project.json` through a subflow. `readiness.Settle` runs on
   `req.FlowDefinition` before subflows are inlined, so `NavigateTarget` finds nothing and the
   run logs `generic-navigation`. The declared waits still execute — they are authored into the
   shared action — but the reported strategy understates it. Only the 19 cases that navigate
   directly report `declared-surface`. Fixing the report means resolving subflows before
   settling, or attributing the subflow's strategy to its caller.

## Floor claims report one offender at a time

The geometry floors stop at the first failing node, so fixing one reveals the next. Fixing the
Schedules tab surfaced an undersized Search button; fixing that surfaced two 42px buttons, then
the search input, then a 28px card menu. Expect a queue, and re-run validation after each fix.

**The overflow floor measures layout rects, not painted output.** It reads
`getBoundingClientRect()`, which reports an element's transformed box even when an ancestor
clips it. So neither `overflow-x-auto` (tab strip) nor `overflow: hidden` (button glow)
satisfies it — both were tried and both failed. The element must not be *laid out* outside the
viewport at all, which is why the glow now animates `background-position` instead of translating
itself.

## Two environment traps that cost time

- **The lifecycle builds Go with `GOWORK=off`**, so a plain `go build ./...` can pass while
  `make start` fails. Reproduce a lifecycle build failure with
  `GOWORK=off go build ./...` inside `api/`.
- **`make build` can emit stale CSS.** After editing `ui/src/index.css`, confirm the change
  reached `ui/dist/assets/*.css` before concluding the fix did not work; `npx vite build`
  directly in `ui/` is the reliable path.

## Floor claims: fixed and remaining

Every one of these is a pre-existing mobile defect the contract surfaced rather than caused.

Fixed:

- Dashboard tab strip overflowed a 390px viewport — five labelled tabs cannot fit one row, so
  labels collapse to icons below `sm` with `aria-label` carrying the name. Note that
  `overflow-x-auto` does *not* satisfy this floor: it measures node bounds, not document scroll.
- Eight undersized touch targets against the 44px floor: mobile search (34px), header settings
  and help (34px), hero New project and Import (42px), header Import and New Project (42px),
  the project search input (42px), row action menu items, card footer actions, and the card
  actions menu (28px, which also had no accessible name).
- BAS honoured no `prefers-reduced-motion` at all. `ui/src/index.css` now carries the standard
  guard, which is both the accessibility default and what makes geometry capture deterministic.

- `floor-no-document-horizontal-overflow` — attributed by capturing the page through BAS's own
  CaptureService and walking the returned accessibility tree for negative-x nodes. The offender
  was `.hero-button-glow`, a shine sweep that is `position:absolute; inset:0;
  transform:translateX(-100%)`, so at rest it sits a full button-width to the left of its
  button. `.hero-button-primary` carried no `overflow:hidden`, so it painted outside: button at
  x=33 w=316 put the glow at exactly x=-283. Clipping the button fixes it for all five call
  sites.

## How a run uses this

`services/readiness` resolves a route's declared required surfaces and rewires the flow so the
waits run immediately after its first navigate, before any authored step. The ladder is:

1. **explicit-wait** — the flow already waits after navigate; the author has said what readiness
   means here and that wins outright.
2. **declared-surface** — a known local scenario's required surfaces are waited on by *terminal*
   lifecycle state. Terminal deliberately includes `error`: a failed page has finished, and
   reporting that in milliseconds is the point.
3. **generic-navigation** — everything else, unchanged from before, with a stated reason.

Both execution entry points call it. That matters: Workflow Health runs every `bas/` case through
`ExecuteAdhocWorkflow`, not `ExecuteWorkflow`, so wiring only the latter would have resolved the
contract for captures and ignored it for the suite the contract exists to fix.

The selected rung is logged at Info either way — including the fallback. A silent fallback is
indistinguishable from a fast pass, which is the failure mode being removed.

## Validate

```
experience-manager --auto-start spec validate browser-automation-studio
```
