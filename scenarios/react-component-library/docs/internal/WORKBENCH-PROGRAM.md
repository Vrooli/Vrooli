# Workbench implementation program

Status (2026-11-22): **phases 0–5 are substantially implemented; workspace
migration for the legacy studio stores, unified hybrid search, design/template
legacy-owner retirement, and Workspace JSON transport migration are complete.
Ask is integrated. After the answer decoder began removing redundant citation
IDs in first-seen order, two sequential live 20-query gates passed 20/20 with
unique citations resolving to supplied result IDs; p95 was 2.88 s and 2.34 s
against the 8 s target. Unknown IDs and missing/invalid citation fields remain
strictly rejected. Independent answer-quality review remains open; no
rights-receipt bypass is acceptable. See
`artifacts/ask-latency/2026-11-22-postfix-summary.md`.
Two more live 20-query gates on 2026-11-22 passed 20/20 with unique citations
resolving to supplied results (p95 7.91 s and 4.51 s; 8 s target); retained
at `artifacts/ask-latency/2026-11-22-semantic-slice-live.json` and
`artifacts/ask-latency/2026-11-22-semantic-slice-repeat.json`. One sample had
only 92 ms of p95 headroom, so these samples do not establish stable latency
over time. The managed API currently reports 821 indexed documents and a READY
semantic leg. The focused Go integration test now exercises RCL search through
the shared hybrid engine and verifies vector payloads are projected only via
SQLite. It uses shared `ai-go/search/searchtest` companions; focused and race
tests pass. Unit Health plan-only run
`uh-20261122-140345-afdb776c87489f9c8f439a4a6416bf7b` reports zero findings
for the API workspace and did not execute tests. Remaining Ask work is
independent answer-quality review and longer-run
latency certification; the live citation contract passed all four post-fix
20-query gates.
The existing live hybrid owner gate also completed 100/100 ordinary Search
queries on the managed service: p50 404.3 ms, p95 522.2 ms, max 850.4 ms
against the unchanged 1,500 ms hybrid target, with the curated workspace first
for `landing page`. The JSON is retained at
`artifacts/ask-latency/2026-11-22-hybrid-100-query-semantic-slice.json`; this
gate does not ask the model or establish answer quality. Independent human
review of Ask answer quality remains open.
**Tidiness continuation (2026-11-22):** the 1,375-line
`api/internal/components/indexer.go` is split along indexing orchestration,
manifest projection, version/story reading, source-header validation, and
catalog metadata. The core is 384 lines and the largest extracted file is 464;
within these changed production files, the >500-line count fell by one. The
named Connect error maps now use ordered mapping tables; `gocyclo` reports
`searchConnectError` at 5 and `ToConnectError` at 4. Focused component and search
handler tests pass. A fresh Test Genie run (`20261122-151900-186bdf42`) skipped
tidiness because tidiness-manager's graph-admission wait timed out; its retained
receipt is `artifacts/tidiness/2026-11-22-indexer-cohesion-provider-skipped.json`.
The 1,454-line components SQLite repository was also split by ingestion, reads,
projections, maintenance, and versions; the largest resulting file is 482
lines, and `go test ./internal/components` passes. Do not treat the prior
63-file observation as a fresh whole-scenario count; the new manager scan did
not complete.
The composition still embedded in scenario experience pages remains an open
ownership migration, even though the old SketchService RPC surface is retired.
Experience pages with compositions now offer an explicit atomic Import into
workspace action: it saves the workspace and immutable source-recovery receipt
together, reuses unchanged retries, rejects changed source, and leaves the
source page intact. The receipt reports unsupported render bindings; it does
not establish native output parity. `agent-detail` remains excluded, with its
state, element, claim, async-lifecycle, and test-binding obligations captured by
`experienceimport.InspectPageContract`.
The one-release bookmark bridge remains for unsynced old component-context
links. The 100-query hybrid search gate passes. The served browser journey
exercises the original eight-screen HTML/CSS import, HTML edit, RCL replacement,
layer-tree reorder, canvas reparent with reload persistence, 320/390 px overflow,
and a 320 px workspace text-edit pass; it reports partial until a capture
response is observed. Authenticated capture/drift evidence,
human-reviewed Ask quality, longer-run Ask latency certification, and the strict
net-negative line gate remain open. The curated visual study is now seedVersion
44; selected source-native RCL Button replacements raised Money-night RCL share
from 6% to 9.3%, and Plant-day/Plant-night backgrounds now match their SVG
cutout colors. Seed v45 aligns Run-before navigation with the reference. After supported restart
`startop-3a1adf53a11bfbd8e75607d1d6e800f7`, all eight 390×844 Present-mode
views were freshly captured on build `sha256:beb9c477009a892679e6794cb517a9795694d51b6a026421eb429b3efc5ddaad`; each returned HTTP 200, with no browser errors or horizontal overflow. The captures and paired descriptive comparison are retained under `data/served-current-v45/` and `captures/eight-screens-reference-vs-local-v45.*` in the operator experiment. These local captures do not constitute operator acceptance or authenticated BAS evidence.

This program records the move
from a split workbench to `five` destinations, one workspace aggregate, one
search, one health/stats owner, bounded registry/authoring contracts, and an
explicit deletion path.

**Latest focused continuation (2026-11-22):** UI `pnpm test:coverage` passes 178
files / 1,175 tests at 91.88% statements, 85.01% branches, and 85.32%
functions; the committed 85% floors remain unchanged. Full UI lint, type-check,
and production build pass. The full Go API suite, full CLI suite at
75.0% aggregate coverage, and `make lint-go` pass. Review detail now uses
published Button, Input, Select, and
Textarea assets instead of native controls; the decision test selects through
the accessible RCL Select. The capture-outline import now uses the FileReader
contract supported in browser/test runtimes, registers its user-facing copy in
the string catalog, and has focused coverage. Its first lint run exposed the
issues; corrected focused tests and the full UI suite pass. Full UI lint passes
when run separately after one concurrent ESLint/Vitest Vite-temp-config race.
Experience-import receipts now verify all deterministic receipt fields, reload
and tamper safely, and persist atomically with their workspace. Dashboard and
conversations remain partially represented by native workspace layers; their
source pages stay intact and render parity is not claimed. Agent-detail stays
outside conversion with its source obligations machine-inventoried. Ask gate
artifacts are in `artifacts/ask-latency/`.

**Served v43 continuation (2026-11-22):** the supported lifecycle restart
completed healthy at operation `startop-7fde04d618804ad2381df6947a5e18b0`; API
build identity is `sha256:9a537b4cb5d9dd45893c633c81d7acb352d4a4b51c4846aa9fd0595e2ec50717`.
Eight curated phone workspaces were captured at 390×844 with no browser errors.
The paired JPEG comparison and per-screen MAE values are in the operator
experiment; they describe alignment only. The current moved-source browser
journey also imports the filtered money-dark region with all three CSS files,
edits it, applies a Button replacement, reorders and reparents layers with
persistence, and passes the five destinations at 320/390 px without page
errors. Its BAS capture step was intentionally skipped because this environment
has no authenticated account, so there is still no capture receipt, durable
drift history, or operator visual acceptance.

**Fresh served build (2026-11-22):** UI lint and typecheck pass; production
build passes after regenerating the owning 175-contract type map. The build
retains the known 2.57 MB CodeEditor chunk and 6.04 MB TypeScript worker.
Supported `make restart` completed at
`startop-3309b05605bbb86e02aa1e99217326fa`; API and UI report matching identity
`sha256:371ed65e976d2feb9690b4b0747d3f82bc53277ff5a9181ecc971abb213b9d5b`.
All five destinations return HTTP 200. Current 1440×1000, 390×844, and 320×720
Start captures are archived in the operator experiment; Start now exposes the
RCL Tabs result-kind filters from the reference. The Money-night editor capture
shows the selected seed workspace, two phone frames, layer tree, rounded 9%
RCL share, and Visual drift panel. Build identity is
`sha256:aad9eda63bf907d8c8aea815aea55dcd8040a855862f076ecf0420d0b560e693`.
The full 390×844 v41 sheet and fixed-crop comparison against the supplied JPEG
are recorded in the experience-engineering operator experiment. **Correction
(2026-11-22):** inspection found that the file labeled as the v41 comparison
uses normalized captures named `*-v28.png`; its MAE values do not describe the
v41 sheet and are withdrawn as current parity evidence. The comparison utility
now accepts separate capture suffix/version labels and normalizes high-DPI
screenshots; a corrected v41 comparison uses the actual `eight-screens-live-v41`
captures and is retained in the operator experiment. Its MAE remains descriptive,
not a visual-quality score. Neither image is an authenticated BAS receipt or
operator acceptance.

**Eight-screen import and served v42 continuation (2026-11-22):** the importer
is split into a stable facade, region/tree mapper with inherited-style
preservation, sanitized preview renderer, and shared types. Focused parser/SVG
tests pass 83/83 and the import-dialog behavior/a11y suite passes 9/9. The
journey locates the canonical moved HTML bundle and exercised selected-region
import, edit/save, RCL replacement, tree reorder, canvas reparent persistence,
health indicators, and five routes at 320/390 px on the fresh healthy build
`sha256:a022f543243bed5d06faafdbb5ace6908438c5cd0e1092cb91fe5bc2d722ef7e`.
Eight local 390×844 captures from seed v42 and the reference comparison live in
the operator experiment. RGB MAE ranges from 24.875 to 55.407; the metric is
descriptive only. Journey status remains partial because BAS capture returned
HTTP 504 without an authenticated account. This does not create an admitted
capture receipt, durable drift history, or operator acceptance. After the
catalog and generated strings were reconciled through their owning commands,
the full UI coverage run passes 1,164 tests across 178 files at 91.82%
statements, 85.02% branches, and 85.17% functions. Full UI lint and type-check
also pass with five existing published-catalog warnings.
Source-baseline persistence and source-to-capture
comparison are implemented with candidate/accepted status and immutable history;
focused Go and UI checks pass. An authenticated BAS capture and operator review
are still absent. Test Genie comprehensive run `20261122-112705-bd693ca1`
completed **FAIL** at 11:58 UTC. It started before the source-baseline changes,
so it cannot certify them; its failed phases include docs, performance, unit,
workflow, experience-contract, tidiness and Stripe outcomes, with experience
not evaluated and component tests not evaluated. Agent conformance was
unavailable. See the durable run record before attributing individual findings.

**Moved-source browser journey (2026-11-22):** the journey now locates the
canonical bundle at `~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-eight-screen-workspace-20261121/data/eight-screens-source/`
and accepts `--source-root`; it no longer depends on the retired absolute path.
Against the served build `sha256:aad9eda63bf907d8c8aea815aea55dcd8040a855862f076ecf0420d0b560e693`,
the run imported the filtered “money dark” region with its three CSS files,
edited and saved imported content, applied an RCL Button replacement, reordered
and reparented layers with persistence, rendered workspace health/coverage, and
passed all five primary destinations at 320/390 px with no page errors. The
capture request returned HTTP 504 because BAS requires an authenticated account;
there is no live capture receipt or visual-drift history from this run. The
machine-normalized report is retained in the operator experiment. Nested region
extraction also now retains safe inherited typography/color/custom properties,
with 59 focused HTML-import tests, so omitted ancestors no longer strip source
tokens from the editable selection.

**Eight-screen seed v43 and moved-source journey (2026-11-22):** curated
workspace data now updates Plant-day facts and thumbnails, Plant-night layout
and selection, and Phrase-practice navigation. Fresh served captures cover all
eight screens at 390×844 with no page errors; the paired reference comparison
reports MAE from 24.874 to 57.090 as descriptive data only. The browser journey
against the moved source bundle also verifies selected-region import with all
three CSS files, edit/save, RCL Button replacement, layer reorder/reparent
persistence, health rendering, and all five destinations at 320/390 px. It
does not produce an authenticated BAS capture receipt or operator acceptance.
Artifacts are under
`~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-eight-screen-workspace-20261121/data/`
and `captures/`.

**Current-source Test Genie run (2026-11-22):** `20261122-124253-457fa6e9`
completed **FAIL** after 32 minutes: 23 passed, six failed, one provider
unavailable, two not evaluated, and one skipped. Its Lighthouse run scored Home
at 0.61 against the 0.75 error threshold. The unit phase repeated three
signature-blind `COMPANION_REIMPLEMENTED` findings; workflow/BAS requires an
authenticated account; experience-contract failed; Stripe lacks required test
receipts; agent-conformance's provider was unavailable; and component-tests
timed out after 1,201 seconds without a verdict. Experience was not evaluated
without authenticated capture. The docs phase passed. After this receipt,
`SuggestAdoptions` was decomposed from a complexity-29 handler into cohesive
helpers; focused adoptions tests and lint pass. The receipt predates that
refactor, so it does not certify the new handler structure. The performance
health CLI/API do not currently return an audit breakdown for the Lighthouse
failure; the CLI compile mismatch is documented in `PROBLEMS.md`.

**BAS workspace-state capture recovery check (2026-11-22):** read-only
`vrooli scenario status` reports both react-component-library and
browser-automation-studio running and healthy. This confirms process health
only; it does not establish an authenticated browser grant or a capture
receipt. The latest recorded served journey still received no response to
`CaptureWorkspaceState` within 180 seconds, and the latest recorded BAS capture
failure is HTTP 504 with the runner reporting that ad hoc capture requires an
authenticated account. No retry was initiated because the missing grant is
external to RCL and no receipt can be claimed from these observations. The
workspace-state experience contract now includes its declared `stale` lifecycle
state on the selected-composition region, matching the route's stale-revision
rendering. Recovery acceptance remains a successful authenticated capture with
a persisted receipt bound to the exact source revision, page, display and state.

**Earlier validation continuation (2026-11-22, before the citation fix):** API and CLI coverage command failures
from the prior full run are repaired: API import tests now use local fixtures,
and committed CLI primitive evidence has been regenerated from the manifest.
Those two pre-fix Ask gates each returned 19/20 uniquely cited answers and
one HTTP 503 (`music player`). Their results were superseded after the decoder
fix by the two 20/20 post-fix gates documented at the top of this program.
Pre-fix artifacts remain `artifacts/ask-latency/2026-11-22-run-{1,2}.json`
and `run-manifest.json`. Full Test Genie run
`20261122-030126-14b2db44` completed **FAIL** after 2,579 seconds: 24 phases
passed, five failed, and two skipped. Failed evidence includes the unit test
architecture detector, authenticated BAS workflows, the Experience Manager
maturity-contract response, and Stripe outcomes; Test Genie also reports a
provider-contract skip and a tidiness skip. API and CLI coverage commands
passed in that run. A later local components package run passes, and focused
UI regression checks pass (104 tests). A real served Start check found that an
empty search had been calling SearchService and showing an error after its
request timed out. Start now skips that request while blank; semantic readiness
probes are bounded to 750 ms and cached for 30 s so a slow AI health check does
not block lexical responses. Two earlier lifecycle starts failed before API
readiness, but a later source lifecycle operation,
`startop-ff1bee8e25cefe568f41cb5e664f07d8`, succeeded with matching source and
served identity `sha256:6b1e05855d22eb288dd075d290489bf5b129864d127ab9cecbcdb644f09e78b8`.
That first API health phase took 55 seconds of its 60-second window. After
versionledger batching and the canvas extraction, supported restart
`startop-da328dfb712ff9b9d4a3d098a7121da5` served matching identity
`sha256:a3676840e032a742ce6ca909aeba9d98a3b9b0bccdd462dbfd937d6c6281e90d`; it
completed in 99 seconds with a 22-second health phase. A repeat restart,
`startop-936b9cd5216507847f693b66cefc4b96`, served the same identity and
completed in 24 seconds with an 18.6-second health phase. Deploy-report now says
`changed:false`; direct API/UI health and Start return HTTP 200. The database seed
receipt is version 39. The Start route has been captured settled and returns HTTP
200. All eight curated workspace routes returned HTTP 200 with zero browser
page errors; the v39 local contact sheet and reference comparison are recorded
in the experiment. They do not replace authenticated BAS capture or operator
acceptance. Scenario page-sketch migration still requires native render parity
and persisted receipts before source deletion.

Startup-cost fix (2026-11-22): `versionLedger.Rebuild` runs before
`apiserver.Run` and previously issued multiple queries per stored version. A
read-only local DB snapshot has 943 component versions and 102,979 catalog
gate-evidence rows. The rebuild now batches projections into seven reads,
returns contextual query/scan errors, and tests values, retirement preservation,
error paths, and a fixed query budget across 50 versions. Two supported restarts
serve the change with matching build identity; health phases measured 22s and
18.6s, down from 55s on the prior build. The exact initializer cost remains
unprofiled, and the sample is too small to certify startup reliability.

Comprehensive Test Genie run `20261122-042810-9662df33` completed FAIL (24
passed, four failed, two provider-unavailable, two skipped, one not evaluated).
It was admitted before the version-ledger rebuild and later canvas changes. The
failing unit phase also reported stale CLI evidence and UI type errors; the CLI
evidence has since been regenerated and its full Go suite passes, and the UI
type-check plus focused stage/seed tests now pass. Remaining failures include
unit-health detector false positives, authenticated BAS workflows, an
Experience Manager phase-contract response, and missing Stripe receipts. Run
current comprehensive validation after the supported rebuild/startup check; do
not count this earlier receipt as current-source proof.

**Continuation evidence (2026-11-22):** the last earlier comprehensive run,
`20261122-013158-e43249f7`, is terminal **FAIL** and predates the current
focused repairs. Consolidated focused run `20261122-022822-70e8452b` passes
dependencies, docs, measures, programs, security, and skill-set phases. The docs phase reported
`809` warnings and `13` info findings; its blocking reference
and manifest gates are clear. The dependency
phase includes the service-dependency declarations and CLI module reconciliation.
The measures phase now records explicit reasons for retained projection and
recovery tables that do not have an independent historical outcome measure.
`prompt-manager skill-set validate react-component-library` and the Search Hub
intent case `find a reusable react component` pass. A fresh browser journey
confirmed keyboard navigation across all `five` destinations and Workspace
Studio layer reordering; the wider journey remains partial because authenticated
BAS capture returned 504. The serving API reported `821` indexed search
documents and a READY semantic leg. `capture-story` now declares its optional
BAS binding and degraded behavior; its program phase passes with two fixture
warnings. The one Gitleaks error on an empty PasswordInput story is covered by a
path-and-line scoped allowlist with default rules preserved; the security phase
now passes, with advisory warnings still visible. A fresh comprehensive
33-phase run (`20261122-022858-49a45c8e`) completed **FAIL**. Unit failed on two
API test command errors and three companion-helper shape findings; workflow
reported 26 BAS cases blocked by the missing authenticated account.
Experience-contract and component-tests failed provider maturity-contract
checks (phase/count mismatches), while tidiness and Stripe-test-outcomes each
reported one error. Detailed artifacts are available through
`test-genie runs artifacts react-component-library
20261122-022858-49a45c8e`; the six focused phases remain green. The catalog `runGate` handler
has been split into ordered matrix, calibration, execution, finalization,
persistence, and response helpers; its focused tests and lint pass. Focused
search, preview-seam, and catalog handler tests are recorded with their owners.
Review the
retained unit detector findings, current docs warnings, authenticated capture,
independent Ask-quality evidence, and experience-contract results before any
readiness claim.

**Component-context cutover audit (2026-11-21):** The durable authoring path
is workspace-scoped: `/design?workspace=…` opens the workspace studio, and the
component detail action forks a curated workspace with the selected component,
story, and version bound into its saved state. The old `/design?component=…`
route remains only as a URL adoption adapter; its one-release local-cache
reader is still required for old links to custom drafts that never reached the
server. The retirement evidence required by the store table below is absent:
the compatibility-window close is not recorded and no privacy-safe old-link
usage signal exists. Focused component detail, adapter, and route tests pass
(3 files, 21 tests), including stale-request suppression and persistence of
the selected story binding. Therefore this audit closes no legacy owner: keep
the bounded read-only reader until both retirement conditions have evidence.
Do not add telemetry that records component/story or draft identifiers; any
future usage signal must be privacy-reviewed and count only legacy-route use.

**Fresh local UX readout (2026-11-21):** the served Start page was captured at
1440×900 and 390×844. It retains the mockup's headline/search hierarchy,
intent browsing, recent workspaces, Continue rail, library pulse, curated
templates and shortcuts; the mobile page has no horizontal overflow and keeps
the five-destination navigation. No Start code change was justified by that
comparison. The current eight-screen contact sheet and operator reference are
linked from the experiment README. The curated seed contains sixteen workspace
snapshots: eight art-directed product templates and eight imported eight-screen
studies. The current full UI suite passes 163 files / 1,102 tests after the
style-import fix. The eight-screen adoption review found no safe substitutions
beyond Buttons; do not infer maximal asset adoption from the renderer test
alone.

**Continuation update (2026-11-21):** the workspace renderer now mounts every
bound catalog component at its authored slot, and BAS state readiness waits for
all bound components. Moving the primary live slot no longer hides secondary
slots. Replacing an imported node retains its source visual props. The current
evidence read now rejects stale capture markers while keeping immutable receipt
history. Ask latency parsing rejects malformed response shapes and invalid SLO
types. The eight-screen study still has no safe additional catalog mapping
beyond Button; six separate RCL assets were adopted by Activity and Capabilities
without removing the weekly actor/action summary. Self-hosting now measures
36/265 against the unchanged floor of 36. Current validation: UI 163 files / 1,102
tests, TypeScript, UI lint, API and CLI `go test ./...`, Go lint, and production
build pass. The build retains a large CodeEditor chunk warning. `make check`
started Test Genie run `20261122-013158-e43249f7`; its server-owned result is
still pending. Authenticated BAS capture, native experience-page render parity,
independent Ask answer-quality review, a broader repeated-run latency
certification, and the strict net-negative source ledger remain open. The two
latest post-fix 20-query latency gates pass 20/20 at p95 2.88 s and 2.34 s.

**Brand Manager bootstrap seam (2026-11-21):** the retained RCL unit/full-run
reports checked (`20261121-160317-741e6852` and
`20261121-180053-62173067`) do not contain a `MISSING_INJECTABLE_SEAM`
finding, so this change does not claim to retire a current Test Genie finding.
The brand client did have a real ambient boundary: `BRAND_MANAGER_URL` was
read directly from the process environment before service discovery. Its new
`ClientOptions`/`ResolveBaseURLWithOptions` path allows tests and alternate
bootstraps to inject both environment lookup and discovery. Production still
uses `os.Getenv` and the existing discovery resolver, with explicit URL
precedence unchanged. Focused tests prove override precedence, injected
discovery, and client wiring; `GOWORK=off go test ./internal/brand -count=1`
passes. The active comprehensive Test Genie run
`20261122-013158-e43249f7` had no terminal report when this evidence was
recorded, so it is not used as validation for this seam.

**Fresh search readout (2026-11-21):** the lexical index has 821 documents.
Current-build checks rank the Marlow workspace first for `landing page`, the
exact Button asset first for `Button`, and Harbor before the exact
OnboardingFlow pattern for `onboarding flow` when Harbor has the stronger fused
score. Single-token exact asset names receive explicit asset intent; a stronger
multiword workspace realization stays outcome-first, and an exactly named
workspace wins an asset-name collision. The latest clean 100-query hybrid run
passed with p50 239.7 ms, p95 640.9 ms and maximum 853.1 ms. The prior Ask
provider failure is reported as `knw-1795296775217310438`; the latest full gate
passes, but human quality review remains outstanding. Scenario-qa heartbeat
delivery is paused by the global operator hold.

**Open composition ownership gap (2026-11-21):** the design/template migration
does not complete the experience-page split. `api/internal/sketch.Document`
still models `Template`, `Render`, `Placements`, `Unplaced`, and layout-bearing
`Regions`, and `Store.Save` still writes those fields into
`scenarios/<scenario>/experience/pages/*.json`. These are active fields, not
dead compatibility data: the repository currently has 320 experience page
files; 10 contain nonempty `sketch` objects. A fresh read-only inventory finds
three composition-bearing pages: `switchboard/dashboard` and
`switchboard/conversations` have template/render contracts (10 placements on
`conversations`), while `switchboard/agent-detail` has one placement without a
template/render contract. Other sketch regions mix layout metadata
with page-intent structure, so they cannot be mass-deleted by field name.
`experience/pages` must remain the intent/claims owner while composition moves
to Workspace aggregates. A read-only inventory in
`api/internal/experienceimport` now reports stable identities, source hashes,
region and placement counts, and converter support across the scenario tree.
`ConvertPage` produces dry-run workspace aggregates for dashboard and
conversations, maps render regions to native slot nodes, and embeds the exact
source sketch in each state adapter with a whole-page rollback receipt. For the
conversation page, two static `controls.button` placements with `Button` exports
now produce native Button fallbacks that preserve authored labels, primary
variant, pill shape, and accessible names. Tests
prove deterministic output and exact JSON round-trip without writing source or
workspace data. `VerifyResult` now rechecks the source-page hash/revision, the
target aggregate hash and binding, and the sketch held by every state before a
caller persists the pair; tests reject tampered source, sketch, target,
binding, or adapter evidence. `ConvertPage` and `VerifyResult` remain
side-effect-free dry-run operations. The separate explicit adoption action now
atomically persists the workspace and immutable recovery receipt while leaving
the source page intact; this receipt does not prove native rendering. The
placement-only agent-detail page remains explicitly unsupported. Studio
presentation sends supported adapters through the existing
read-only PreviewService; a UI test proves the source sketch and workspace
adapter produce the same request. The native editable canvas still does not execute the legacy `CollectionPage`
layout, arbitrary template bindings/fixtures, authored slot paths, or button
interactions. Other placements also remain unmapped, so full-page
`RenderParity` remains false. This is inventory and conversion
evidence, not a completed composition cutover. No source page was deleted, and
rendered-output comparison remains partial. Do not claim the sketch composition
deletion ledger is closed.

A fresh dashboard render-parity audit confirms the gap is structural: its
`CollectionPage` contract declares five render ports but no placements, and its
template bindings are empty. The three experience regions have semantic and
lifecycle intent but no authored mapping to those ports. The lone unplaced
reference upload names a Switchboard renderer replacement owner; it is not a
catalog template placement. The native workspace canvas can display empty slot
nodes but cannot execute the catalog template shell or its slot paths. The
supported converter and legacy PreviewService request tests pass; those prove
deterministic conversion and adapter equivalence only, not native render parity.
Keep `RenderParity=false` until Workspace supports template/region/binding/
fixture execution or Switchboard authors a complete native `TemplateNode` tree
with explicit mappings.

The conversations page makes the behavior gap concrete: it declares 10 planned
placements over 10 render ports and 18 named render stories/state fixture maps.
Its `$preview.actions` contains state transitions and `$handler` tokens for
create, reply, approval, and thread-selection flows, with state-specific
component props. `ConvertPage` currently emits one static tree per display; its
Button fallback synthesizes visual props from a static label and drops those
handlers. `TemplateNodeRenderer` has no equivalent action dispatcher. This is
not loss that a source receipt can repair: Workspace first needs an executable
contract for component props, state/story selection, safe action dispatch, and
CollectionPage template/slot-path rendering. Until then keep the legacy preview
adapter and `RenderParity=false`; test a narrow Button mapping only after its
source behavior is natively representable.

The placement-only `switchboard/agent-detail` record is not a safe workspace
conversion source. Its single placeholder region has no `fills.asset`,
template, render export, region map, binding, fixture, or slot path. The outer
experience contract also carries ten lifecycle/permission/data states,
accessibility elements and roles, UX/security/a11y claims, async region
lifecycle, test bindings, grant ownership/permissions, reversible writes, and
stale/partial-data rules. The dry-run receipt preserves the full source page,
but Workspace State currently retains only the sketch adapter. Do not infer a
layout or migrate this source until Switchboard defines the composition and
state/action/binding model, or supplies a domain renderer, and the importer
maps and verifies the required contract fields.

The focused source record is
`api/internal/experienceimport/testdata/switchboard/experience/pages/agent-detail.json`.
Its `sketch` has one `grant-region` placement whose `fills` contains only
`intent` and `placeholder` (no asset identity); it has no `template` or
`render` member. The surrounding page declares two async regions
(`grant-region`: loading/ready/error; `activity-region`:
loading/ready/empty/error), ten page states (default, loading, saving, success,
validation-error, request-error, permission-denied, stale, empty, partial), ten
element test bindings, and two region test bindings. Eight claims specify
reading order, plain-language grants, owner-only absence, profile ownership,
reversible failed writes, no silent grant widening, busy-save behavior, and
dark-theme distinction. None of those obligations is expressible by the
placement-only sketch or by the Workspace state adapter, which retains the
sketch as recovery data but does not execute it. The importer consequently
rejects the page before producing a target; `TestConvertPageRejectsPlacementOnlyComposition`
and `TestInventoryCompositionsReportsConverterSupport` assert that explicit
boundary (the inventory test is `TestInventoryCompositionsMatchesCurrentRepository`). A persisted receipt would prove source recovery only, not fulfill
these obligations or render parity. No conversion, source/workspace write, or
receipt was made; keep this page and its legacy owner intact until the contract
and rendered/accessibility/behavioral parity evidence exist.

`experienceimport.InspectPageContract(source)` now provides a deterministic
inventory of source obligation IDs without translating them into a template
DSL or claiming execution. The focused test
`TestInspectAgentDetailPageContractReportsParityObligations` confirms 10 state
IDs and all 10 descriptions, 10 element IDs, 8 claim IDs, 2 region IDs, 10
element binding IDs, and 2 region binding IDs; binding IDs are sorted for
stable evidence. It also checks both async region lifecycle state sequences
exactly. Verified with `go test ./internal/experienceimport -count=1` from the
scenario `api` directory. This narrows the audit gap by making obligations
machine-checkable; native rendering, editable action/state semantics, async
lifecycle execution, and rendered/accessibility/behavioral parity remain open.

Next, make the native workspace-state renderer execute supported legacy
template exports/ports and placements, then compare rendered output, accessibility
evidence, and behavior before a reviewed persistence operation writes a
workspace and rollback receipt. Keep each source sketch until its binding,
parity check, and recovery evidence are verified. Intent-only pages remain on
the intent reader path. Only then narrow `ExperienceService` to intent/claims
operations and remove sketch composition mutation/render/candidate paths after
CLI and other consumers have moved.

The rebuilt phone canvas detects expanded HTML-import layers and omits the
duplicate DeviceFrame status bar and island. The recovered source bundle is
`~/design-studies/eight-screens/index.html` with adjacent `css/` and `js/`.
The complete copy used by the experience-engineering operator experiment is
`~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-eight-screen-workspace-20261121/data/eight-screens-source/index.html`
with matching `css/` and `js/`; the older
`experience/design-studies/eight-screens/index.html` path is gone. The
recovered and experiment HTML copies are byte-identical (SHA-256
`e6611a2cf436be949af726b3ee00d51d92f526c280155b6592097ae88f53d11d`). The
operator-provided reference image is `~/Downloads/HTuJxm6XoAEn9YS.jpeg`, with a
durable copy under `docs/mockups/assets/`. Seed version 30 removed explanatory captions and restored source leaf-vein/stem styling. The source tree and workspace seed receipt are at version 39. Two supported lifecycle restarts now serve matching source/served identity on the current API build. Version 33 scales the editable Run-after ruler cap to the supplied JPEG's measured ~6×109px while recording that the original CSS cap is 3×26px; v34–v36 refined hero, copy, chart and CTA alignment. Version 37 follows the reference crop's three-item bottom navigation and records the four-item source HTML as a deliberate deviation. Version 38 adds the source's pressed state to six existing RCL Button layers across the three Money-night snapshots without changing visuals or interaction. Version 39 increases the Run-before content's bottom padding to 88px after the v37 contact sheet showed its Start CTA overlapping the 58px bottom navigation; a renderer test requires 30px clearance. The research experiment now contains a local seed-v39 contact sheet and per-screen captures, manually compared with the supplied reference. This is local browser evidence, not an authenticated BAS receipt or operator acceptance. The latest Run-after preview comparison is `docs/mockups/assets/run-after-reference-vs-proposed-v37.png`.

The Run-after screen comparison found a source/reference conflict: HTML CSS defines a 3×26px ruler cap, while the operator JPEG shows about 6×109px after crop normalization. Seed version 33 follows the supplied JPEG and records that source-CSS deviation in the editable layer label; versions 34–36 refine hero size, copy line breaks, chart alignment and CTA fit. Version 37 follows the screenshot's three-item bottom navigation and records its deviation from the source HTML. These are targeted visual refinements, not operator acceptance. `docs/mockups/assets/eight-screens-reference.jpeg` is a
durable copy of the operator JPEG; `tools/compare_eight_screen_reference.py`
reproduces manually aligned crops and descriptive RGB differences. Those
metrics are not a quality score: compression, crop alignment, resampling and
device-frame differences contribute to the result. Live BAS capture and
workspace drift receipts remain unverified because the workflow requires an
authenticated account. Across eight curated workspaces there are 47 RCL Button
layers among 710 display-state layers (6.62%); other layers remain custom unless
a semantic RCL match preserves the reference and interaction contract. Live
RCL Card defaults no longer force imported card styling via `!important`, so
sanitized author styles can override padding, surface, radius and shadow while
responsive defaults remain available. The importer also accepts safe camelCase
style keys such as `borderRadius`, which it previously dropped. Focused renderer
tests cover these styling paths.

The earlier 100-query hybrid search gate passed 100/100 queries, with p50 183.6 ms, p95 370.4 ms and maximum 422.2 ms, but asserted the obsolete `LandingPage`-first ordering. That ranking assertion is now corrected to require the concrete matching workspace first; latency figures remain historical. Earlier Ask latency experiments had intermittent provider errors and a long-tail response. On 2026-11-22 the live `music player` 503 was traced to a repeated citation ID rejected by RCL; the decoder now removes redundant IDs in first-seen order while preserving strict checks for unknown IDs and malformed fields. Two post-fix 20-query gates pass 20/20, with p95 2.88 s and 2.34 s (8 s target); see the current artifact at the top of this program. Human-reviewed answer quality remains open; no provider policy or rights checks were bypassed.
The last terminal comprehensive Test Genie run, `20261121-132723-0599033c`,
completed **FAIL** (17 passed, 14 failed, 1 skipped of 33 phases). A fresh full
run, `20261121-180053-62173067`, completed **FAIL**. Its experience phase is blocked at L0→L1 because BAS capture requires an authenticated account. The latest full API aggregate profile is 75.0% (28,569/38,092), meeting its committed 75% floor; it includes behavioral tests for theme routes/errors, sketch candidate/reference-asset behavior, authored-quality gate evaluation, catalog checks/evidence capture, adoption Reapply/BatchApply, locally owned calibration fixture materialization, story-argument/version-shape gate validation, source-root resolution, stamp-report classification, screenshot-based visual-review evidence revalidation, component-manifest updates, and version-presence reconciliation. The only failing Go test is
`TestLiveSelfHostingDoesNotRegressBelowReviewedFloor`: 30 RCL assets consumed
against the unchanged floor of 36. The latest full UI coverage suite passes all
committed floors at 91.51% statements, 85.03% branches and 85.17% functions
after focused route and page-story tests. The API `GOWORK=off go mod tidy -diff`
check is clean.
Other failures include dependency, docs, storage,
workflow, experience-contract, tidiness, security, measures, programs,
skill-set, Stripe-test-outcomes, and component-tests; agent-conformance was
unavailable. Workflow execution also requires authenticated BAS access. The
previous Test Genie run `20261121-093530-44212a43` completed **FAIL** (15 passed, 15
failed, 2 skipped of 33 phases). In that run, the experience phase was not evaluated
because BAS was unavailable. The unit phase
reported an API Go test command failure, API statement coverage 65.2% against
75%, and UI branch/function coverage below the committed 85% floor (80.05% and
77.95%; lines were 85.90%). The component-tests provider rejected a maturity
assessment whose 160 findings did not match its reported error count. Other
failures include dependency, docs, performance, storage, workflow,
experience-contract, security, measures, programs, proto, skill-set, and
Stripe-test phases; agent-conformance was unavailable. A direct uncached API
suite reproduced the Go test failure at
`TestLiveSelfHostingDoesNotRegressBelowReviewedFloor`: the UI consumes 22 RCL
assets against the reviewed floor of 36. The floor remains unchanged. Focused
seed/Start/Search/import tests (88/88), targeted ESLint, and TypeScript passed
after seed 28. The run does not establish readiness. After that run, the full
UI coverage suite passed all 991 tests in 157 files at 90.6% statements,
82.57% branches, and 82.84% functions; the committed 85% branch/function floors
remain unmet. The additional
StudioLayers coverage suite passes (100% functions for that module), and the
WorkspaceStudioPageContent tests now cover save retry, presentation/guide
controls, mobile sheets and HTML import (97.46% statements, 85.96% branches,
93.75% functions for that component). The HTML import sanitizer also now keeps
safe local SVG paint references while rejecting external URLs; all 67 import
tests pass. The production build passed with the existing oversized CodeEditor
chunk warning.
The latest full UI coverage suite passes all committed floors at 91.51%
statements, 85.03% branches, and 85.17% functions after focused route and
page-story tests. The full API coverage suite meets its floor at 75.0%
(28,569/38,092); the self-hosting gate now meets its reviewed floor at 36 assets
used / 265 implemented after six real integrations into Activity and
Capabilities. Full UI lint, type-check, and production build pass; the build still
reports the large CodeEditor chunk warning; the redundant dynamic/static i18n import warning
was removed. Proto-health reports zero errors and 23 advisory warnings;
`make -C packages/proto verify-committed-gen`, targeted `buf lint`, and the CLI
suite pass. The 26 moved messages preserve field number/type/name/json_name;
all 160 Connect procedure paths are unchanged and breaking checks report zero
wire/JSON breaks or unreconciled consumers. External clients using the former
protobuf message FQNs must regenerate. Requirements PR-002 now has a disposable
50 KiB warm reload benchmark: p50 233.2 ms, p95 250.2 ms, max 283.0 ms against
the 1 s target; this does not replace authenticated BAS evidence.
Workspace JSON operations use typed
Connect clients and handlers; REST serves asset bytes, receipt screenshots, and
visual-comparison heatmaps. The shared StatusBadge, AsyncPanel, and ErrorBoundary
drafts are represented in the generated release-hash ledger; published hashes
remain untouched. The §7 cleanup audit removed the unreferenced
`ui/src/api/versionHistory.ts` join helper and its stale coverage exclusion;
other legacy-looking paths checked still have live consumers. The 2026-11-21 code-only inventory
across the scenario (`.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.css`,
`.scss`, `.sql`, `.proto`) is roughly 15,220 net lines versus HEAD: 12,489
additions and 25,080 deletions in tracked files, plus 27,811 lines across 199
untracked source files. This counts the current shared worktree, including
generated and test sources; it is an inventory, not an accepted deletion-ledger
report. The strict net-negative gate remains open and the current work still
needs a reviewed deletion ledger.

It treats the target as **greenfield design** and the delivery as a **strangler
migration**: define the ideal domain, contracts, schema and UI first; build new
vertical slices beside the old ones; migrate data once; switch routes; delete the
old owners. Published URL redirects and one-release read-only adapters are the only
compatibility we keep.

Related decision artifacts:

- [WORKBENCH-MATURATION.md](../concepts/WORKBENCH-MATURATION.md) — `five` destinations and workspace model
- [SEARCH-AND-COMPOSITION.md](../concepts/SEARCH-AND-COMPOSITION.md) — search findings and target
- [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md) — historical D · Nocturne record
- [DEBT.md](../studio/DEBT.md), [PROBLEMS.md](PROBLEMS.md) — dated debt and gaps
- [WORKBENCH-MATURATION.html](../mockups/workbench-maturation.html) and [search-options.html](../mockups/search-options.html) — rendered prototypes

## 0. Delivery status (2026-11-21)

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
- The component-context route now explicitly adopts old component context into a
  workspace; unbound experience pages offer workspace adoption, and bound pages
  open the workspace studio. The unmounted `DesignStudioPage` and `useStudio`
  path, browser design fallback, and legacy template write API have been removed.
- Legacy `studio_designs`, `studio_design_assets`, and `studio_templates` rows
  migrate into workspace aggregates at stable IDs, with provenance, collision
  rollback, idempotent receipts, and no resurrection after deletion. Source
  tables are dropped in the same transaction after rows/assets import and
  experience bindings validate. Design/template APIs and repositories are
  retired; only a read-only browser-cache bridge remains for unsynced old
  component-context bookmarks. See
  [WORKBENCH-LEGACY-OWNERS-AUDIT.md](WORKBENCH-LEGACY-OWNERS-AUDIT.md).
- Search now combines ranked FTS with the `packages/ai-go/search` semantic leg;
  Ask sends bounded indexed evidence through AI Gateway with local-only routing
  and validates citations against supplied result IDs. Reconcile triggers are
  coalesced so startup and concurrent reindexing do not block API readiness.
  Semantic query failures and projection-reconcile failures have separate
  recovery conditions: a successful query can clear a transient query failure,
  while a failed/partial projection remains DEGRADED until a complete reconcile
  succeeds. `go test ./internal/search ./handlers/search` covers this recovery
  rule and the explicit DISABLED/DEGRADED/READY status cases.
- Workspace search documents include workspace summaries and related page types.
  The Marlow curated seed now realizes the landing-page archetype. Workspace
  state data, status, and capture ID survive aggregate save/reload. BAS captures
  the selected stored state through a chrome-free route; RCL verifies the returned
  screenshot and accessibility evidence before persisting capture status and ID.
  Stale revisions and concurrent edits cannot attach stale evidence.
- The command palette opens with focus in its search field. Keyboard tests cover
  navigation across Start, Library, Design, Review, Stats, and selecting studio
  layers and frames. The legacy page story was removed and executable stories now
  cover workspace design and adoption routes.
- **Second slice (2026-11-19):** the workspace studio renders the mature studio
  surfaces (layers, system tokens, canvas with zoom/guides, inspector,
  status bar, present mode, undo/redo, autosave with conflict stop); Review is a
  server-filtered paginated queue; workspaces/search/review/activity/health are
  promoted to the CLI with no omissions; search links address the workspace
  studio and its projection version forces a rebuild on shape changes; the seven
  published-byte drifts are covered by a reviewed repair ledger and the release
  hash check now detects unexplained drift; all `five` destinations and the
  workspace studio pass structural axe.

**Evidence**

- UI baseline before the visual-source slice: 134 files / 799 tests, type-check,
  18 experience-route checks, and production build pass. The focused capture page
  tests cover success/reload, unresolved retry, and terminal failure retry.
- API: full `go test ./...` passes after capture integration; focused migration,
  template adapter, search race, capture persistence, evidence invalidation, and
  failure retry tests pass. The dedicated route rendered the selected Marlow
  state through BAS and returned screenshot plus accessibility artifacts.
- Served API baseline from 2026-11-19: legacy designs and uploads migrated; 703
  search documents indexed; eight curated templates; Review total 13,926 in ~14
  ms; all five routes 200; no horizontal overflow at 320; reduced-motion and
  forced-colors checked. Recheck live counts and routes after the final restart.

**Third slice (2026-11-20):** workspace reference assets (upload, byte route,
region crop, place as layer, legacy uploads migrated); `AssistWorkspace` + the
studio AI tab + `workspaces assist` in the CLI; HTML import as a new page; eight
reviewed curated templates seeded at boot with immutable provenance and a
read-only fork flow; the Design shelf renders real template compositions. The
first edit on a blank stored composition now writes what was on screen instead
of silently doing nothing. The Start destination was rebuilt to the maturation
mockup: one search, browse by intent, recently opened, and a compact pulse rail
instead of a second dashboard. Templates are workspace aggregates, not a DSL:
`workspaces export-seed` promotes a built workspace into the reviewed shelf, and
a test renders every curated display through the public renderer. `golangci-lint` was rebuilt with Go 1.26, 30 dead CLI
functions were deleted, and both Go lints are clean. 320 px overflow was removed
from Start and Design; reduced motion and forced colors were measured.

**Fourth slice (2026-11-20):** the eight-screen visual study is represented by
eight individually forkable curated phone workspaces alongside the prior eight
workspaces. The HTML-region picker ranks explicitly named source regions first;
the supplied HTML file parses all eight screen articles within the safe scan and
layer limits. The studio canvas has a toggleable RCL/custom layer map. Selected
layers have a searchable, semantics-ranked replacement list for native RCL
primitives, layout patterns, and published catalog components. Component matches
bind through the existing live-slot renderer; bound slots count as RCL coverage.
Replacement preserves identity and authored fallback content, protects the frame
root, and avoids leaf candidates for containers. Layer rows support pointer and
keyboard reordering/reparenting through workspace history.

The relocated source entry is
`~/design-studies/eight-screens/index.html` (with `css/*.css`
and `js/app.js` beside it); the contact-sheet reference is
`~/Downloads/HTuJxm6XoAEn9YS.jpeg`. HTML import accepts a local
HTML/CSS selection or folder, selects named regions, maps them into editable RCL
layers, and stores a sanitized, script-free source view alongside the layers.
The source view runs in an empty-sandbox iframe with external resources disabled;
safe inline SVG is also copied into bounded, sanitized editable image layers, and
same-document sprite symbols are expanded without external references. Source
interactions are intentionally inert and must be rebuilt in the workspace. The
read-only source view retains full local CSS fidelity. Editable layers also keep
bounded, allowlisted static CSS matched from local stylesheets and inline styles,
including root custom properties used as design tokens. ID/class/type selector
specificity ordering is retained; nested at-rules are excluded from the static snapshot. Basic static
`::before`/`::after` artwork and text (including local `attr()` content) become
selectable child layers. Unsafe values and external URLs are dropped. Empty
styled source elements become selectable artwork layers with their geometry and
appearance. Editable CSS payload is capped at 256 KiB and HTML plus supplied CSS
at 2 MiB. The real source smoke imports all eight named screens. Whole-document
import reaches the 1,000-layer cap in the study's extra story content, after the
screen rail. Arbitrary source-code editing and remote runtime behavior are not
supported. The “Money, night” curated option now uses the imported source article
as an editable workspace with a 390×844 phone display. Import mapping keeps
source flex rows, inline typography and rich button contents editable instead of
flattening them; static initial chart bars are present in the HTML so the screen
does not depend on skipped JavaScript just to show its first frame. The captured
screen preserves the balance, chart, category cards and fixed navigation closely
enough for the first live source comparison. Remaining spacing and icon details,
as well as the other seven screens, still needed measured comparison before any
parity claim.

**Visual parity follow-up (2026-11-20):** rendered all eight curated screen
workspaces beside their source article captures at 390×844 and inspected the
[durable parity contact sheet](../mockups/assets/eight-screens-parity.png). The wrapper
surface correction keeps transparent source SVG artwork transparent and
square; focused source/renderer/seed tests pass (70 tests). Geometry and
artwork sizing are close across the set. The reference artifacts disagree in
two places: the static Money-day HTML capture has no runtime-populated coins,
while the supplied JPEG does; Plant-night HTML shows a white leaf, while the
JPEG shows lime. The curated seeds follow the supplied JPEG in both cases.
Fine status glyphs, spacing, and native interactive states still need live
workspace review. Source interactions remain inert on import and must be
rebuilt using workspace behavior.

**Visual review slice (2026-11-20):** workspace state capture receipts are now
immutable history. The studio can select two captures for one state, request a
comparison, and inspect the returned screenshot difference, changed-pixel
percentage and heat map. Capture ownership is checked against the workspace and
state; comparison findings do not change RCL coverage. The screen already
reports RCL layer share, painted-area share when measurable, overflow, touch
targets and declared-state gaps. The Checks tab now ranks observable next steps,
links overflow to its frame, points at state gaps, and offers the existing
best-match replacement for a selected custom layer. This checklist does not
claim an art-quality score; unmeasured evidence remains unknown.

**Historical verification checkpoint (2026-11-20; follow-up work continued 2026-11-21):** the full UI suite passed 142 files / 837
tests; TypeScript, experience-route checks,
catalog/string checks, and the production build pass. The eight-screen source
smoke check parsed all eight article regions from the relocated bundle, with
static CSS and pseudo layers preserved plus an isolated source preview. All
eight curated seeds use source-derived editable layers; the reviewed seed bundle
was version 26 at this checkpoint and is version 28 after later follow-up.
Money, night also has a native `Month`
state using the source app's monthly totals, five bars and labels, and peak
value. That state is editable and selectable as a workspace state; imported
source buttons still do not execute their original transitions. The Money
night `47.60` label was present in the imported tree but clipped by the
renderer’s default overflow. Imported surfaces with pseudo children now
retain browser-default visible overflow unless the source explicitly sets it.
The final 390×844 capture shows the label. The workspace
studio hook is now 347 lines; persistence, workspace history, assets, structure,
and proposal behavior have separate owners. Page CSS is split by destination,
and the studio stylesheet is split into four ordered modules. A UI source scan
finds no page over 400 lines, production hook over 350, or CSS file over 1,500.
The production build still reports the existing dynamic/static i18n import and
large CodeEditor chunk warnings. API `go test ./...` and CLI `go test ./...` pass
at this revision; the API suite also caught and repaired a stale reconciliation
fixture that referenced the pre-split `ComponentsPage`. Focused visual-source,
layer-movement, catalog-replacement, receipt-history and stale-comparison tests
pass. Test Genie run
`20261120-181634-a87b9446` ended before terminal evaluation because its
control-plane demand lease failed while waiting for native verification; it is
not a product verdict.

**Live search readout (2026-11-20):** the API is healthy, the lexical index
contains 801 documents after the eight-screen seed update, and
`SEMANTIC_LEG_STATE_READY` is restored. The live “landing page” query returns
the curated Marlow workspace, LandingPage type and related design workspaces.
A newer 51-query hybrid search sample had no request failures; p50 was 534.9 ms,
p95 971.8 ms, and maximum 2,438.1 ms, within the provider's configured 1,500 ms
p95 target. “Landing page” ranked the curated Marlow workspace first. The Ask
schema uses only AI Gateway's supported JSON Schema subset; the service enforces
at least one unique supplied citation after decoding. On the rebuilt API, eight
live Ask requests all succeeded with citations resolving to supplied result IDs;
p50 was 3,053 ms, p95 6,791 ms, maximum 6,791 ms. A focused answer audit found
that broad evidence can still confuse imported screen regions with device
displays, and the index lacks procedural evidence for custom-layer replacement.
The provider's gibberish eval case has a score ceiling and its suite is registered.
The suite has only one reviewed positive and its formal eval run is blocked by
the independent human-review denominator requirement. These live samples are
not the 100-query release p95 gate. Ask's separate latency target and the
20-query gate were added on 2026-11-21; current live measurements are recorded
below.

The Ask evidence gap was subsequently narrowed on 2026-11-21. Its prompt now
distinguishes imported source regions from workspace device displays and grounds
custom-layer replacement answers in the indexed catalog-match workflow,
including container fit and preservation of layer identity/fallback content.
Focused `internal/search` tests cover both claims. Three live Ask requests
checked the region/display distinction, general custom-layer replacement, and
container replacement. All returned HTTP 200 and citations resolving to supplied
result IDs. These are **unreviewed smoke checks**, not human-rated answers or
quality certification. The respective request latencies were 3,891.5 ms,
3,697.7 ms, and 3,419.4 ms (median 3,697.7 ms); these measurements are separate
from answer quality.

The registered suite `react-component-library.workbench.primary` has two cases
and one positive label. Validation found that label live at rank 2, but no
independently reviewed denominator is frozen and there are no stored runs.
`search-hub evals run react-component-library.workbench.primary --tag
ask-quality-gap-followup --json` is blocked with
`failed_precondition: fixed denominator independent review is required for
suite "react-component-library.workbench.primary"`. The freeze operation
requires an independent reviewer identity and durable review evidence URI via
`search-hub evals denominator-freeze react-component-library.workbench.primary
--reviewer <identity> --evidence_uri <durable-uri>`; neither is available in
this work session, so no denominator or reviewer approval is claimed. Ask now
has a separate 8,000 ms p95 target and a 20-sample minimum in
`.vrooli/search.json`; it does not reuse the 1,500 ms hybrid retrieval budget.
`make ask-latency` runs the live explicit-Ask gate, which checks answer text and
unique citations against supplied result IDs while measuring end-to-end request
latency. `make test-ask-latency` runs its mock-server and contract tests. The
8,000 ms threshold leaves room above the prior 6,791 ms p95 sample; it is a
scenario latency budget, not an answer-quality certification. One 20-query run
on build `sha256:8d6cf15b7808a2790549b32b17e02828dba800a34ff9d564faccf6abf7429a79`
passed at p95 5,746.4 ms. A later run on
`sha256:93e9ee99946e0e5b17bb59993d38b8b168abd0658f15a3d9a6c5ad116f5f3938`
had 19/20 valid answers; “travel planner” timed out after 25,025 ms. The 19
successful calls had p50 4,429.7 ms and p95 7,008.4 ms, but the strict gate
failed because it requires all 20 requests to complete with valid citations.

The release gate now lives at
`tools/hybrid_latency_gate.py` with deterministic query coverage and a five
case mock-server test suite. On the restarted build
`sha256:ac7f5f75b37b24cc56060a0c5be44703fc9a6c4ac579a1caf92988b7c3fbc368`,
100/100 live queries succeeded: p50 194.55 ms, p95 424.76 ms, maximum 536.38
ms, and the curated Marlow workspace ranked first for “landing page”. The
gate passed its 1,500 ms p95 target. After adding a one-second readiness
snapshot cache with coalesced probes and error invalidation, 20 status reads
measured p50 0.8 ms, p95 2.1 ms, maximum 194.6 ms; semantic state was READY.
The hybrid release p95 gate is therefore closed for this build. Ask's separate
latency target and answer-quality denominator remain open.

**Remaining, in program order**

1. Close current code-owned quality and performance findings with fresh
   evidence. Test Genie run `20261122-124253-457fa6e9` is terminal **FAIL**
   (23 passed, six failed, one provider-unavailable, two not evaluated, one
   skipped). Lighthouse scored Home 0.61 against 0.75, but the retained artifact
   has no report or owner breakdown; the existing bundle-size observation is
   not source/run-bound and cannot explain it. The three Unit Health companion
   findings are signature-blind; the current Unit Health plan-only reports zero
   findings. Do not remove valid shared companions to satisfy that old detector.
   The tidiness error names 63 long files against its recorded 41-file
   baseline. A cohesion-based extraction split the component indexer from
   1,375 lines into a 384-line orchestrator plus four focused files (largest
   464); the two named Connect error mappers are now at cyclomatic complexity
   5 and 4. Focused component/search tests pass. A fresh tidiness validation
   skipped after graph-admission timeout, so the whole-scenario count remains
   unverified. Stripe outcome failure still needs owner attribution and current
   evidence. Run applicable owner-level gates after current edits before
   another full suite.
2. Obtain authenticated BAS state-capture and workflow evidence. Served Chromium
   journeys cover the five destinations, 320/390 px, HTML import/edit, RCL
   replacement, layer reorder/reparent persistence, and health display. The
   eight-screen v43 captures are served local screenshots, not BAS receipts.
   Seed v44 adds a fresh 390×844 Run-before capture and 1440/390/320 Start
   captures under `data/served-current-v44/`; these are also local browser
   evidence.
   BAS capture and workflow phases remain unauthenticated; no admitted capture
   receipt or operator visual acceptance exists.
3. Complete native experience-page render and behavior parity before retiring
   source composition. Dashboard and conversations have atomic workspace
   adoption, immutable recovery receipts, and retained obligation evidence;
   agent-detail is unsupported. The workspace canvas still cannot execute the
   legacy CollectionPage shell, bindings/fixtures, slot paths, or conversation
   actions. First define a native mapping for CollectionPage ports and each
   region's props, actions, and state transitions; the catalog dependency also
   has unmet capabilities such as reduced-motion. Translate only verified
   semantics and keep RenderParity=false for remaining gaps. Focused renderer
   and importer tests pass, but no rendered-output/accessibility/behavior
   comparison or source retirement is proven.
4. Review the eight-screen comparison and close visual-drift acceptance. The complete eight-screen v46 comparison is captured; MAE is descriptive only
and visible differences remain, especially Wind-down, Plant-day, and
Phrase-practice. v45 corrected Run-before pace and navigation; v46 changed
Plant-day's main leaf to an editable lobed SVG and showed a small descriptive
MAE change under the same comparison method. Continue source-backed fidelity
work and preserve semantics when replacing custom nodes. Authenticated capture,
reviewable drift receipts, and operator acceptance remain open.
5. Recount the RCL source deletion ledger after implementation and close the
   strict net-negative source-line gate. The current shared repository contains
   unrelated concurrent changes, so repository-wide line totals cannot be
   attributed to RCL. An owner-by-owner replacement ledger and scoped source
   recount are still required.
6. Resolve or clearly assign provider-owned certification failures, including
   authenticated workflows, experience-contract, Stripe outcomes, unavailable
   Agent Manager conformance, and component-tests timeout. Preserve the
   terminal run as historical evidence; focused green checks do not substitute
   for a current full certification receipt.

**Closed since the second slice (2026-11-20):** workspace assets and assist;
curated template seeds; HTML import; component-context and experience adoption;
template migration and transactional retirement of legacy design/template
tables and APIs; semantic search and Ask; keyboard journey unit coverage;
removal of the old studio and browser design fallback.

**Visual-design work still open:** the v37 contact sheet exposed Run-before's
CTA/nav overlap. Source seedVersion 39 adds bottom clearance and its renderer
test asserts 30px. A local v39 capture and reference comparison now cover all
eight curated routes; they remain unreviewed by the operator and are not
authenticated BAS evidence. The running service is healthy on the current source build. Two supported
restarts measured 22s and 18.6s for API health after the ledger batching change;
the earlier build took 55s. Remaining work includes more startup observations,
operator visual acceptance, and authenticated BAS capture/drift receipts. The served browser journey
verifies local HTML import/edit, RCL replacement, tree reorder, canvas reparent
persisted across reload, and health checks.

**Storage revalidation (2026-11-21):** storage-manager validation now passes with zero findings after rooted writes and cleanup for component source storage, adoption migration/rollback, calibration overlay/surface work, preview benchmark temp-root writes, versionledger archive publication, stamp-report promotion, graph reconciliation, retirement, generator output, component source-root creation and temporary report cleanup. All previously reported SQL handle captures were cleared through RoutedDB-backed query seams or narrow request-aware SQL interfaces. Isolation proof passes. Inventory
marks pages with active legacy template/render/placement content as
`composition-present`; adoption refuses to create a blank workspace over those
pages until rendering parity and a recoverable migration exist.
The prose false positive in `handlers/sketch/connect_handler.go` was reported
as `knw-1795280701582202396` and is now excluded by the refined shared SQL
classifier; focused analyzer tests pass. The scenario-qa heartbeat is globally
paused by an operator hold. Library Health query providers now use a narrow
RoutedDB-backed query port, and the isolation test proves request-context pool
selection. Forked workspaces clear source-scoped capture markers while
preserving authored state variants. The zero-finding storage result does not certify the remaining API coverage, self-hosting, BAS, or visual acceptance gates.
Reusable asset import, bounded non-SVG static CSS mapping, catalog binding,
allowlisted inline SVG styling, ranked health steps, general layer movement,
and visual comparison UI are implemented.

**Service-boundary refactor completed (2026-11-21):** `ExperienceService` owns the 13-RPC
scenario intent and page-sketch compatibility slice. Its 12 CLI operations keep
their existing command names and routes, and the mounted Design inventory uses
the new service. The six producer-backed capture operations live in
`WorkspaceCaptureService`. Seven acceptance/review-fact operations now live in
`ReviewService`, which now has 12 RPCs and uses narrow ports for current Sketch
evidence and review storage. The remaining 11 Sketch RPCs now belong to
`WorkspaceAssistService` (four) and `WorkspaceCandidateService` (seven); the
empty `SketchService` contract is retired. Experience, capture, assist, and
candidate services have separate generated contracts. Their runtime adapters
remain co-located with the shared sketch domain implementation.
`ComponentsService` now has 12 RPCs and `ComponentAuthoringService` owns the
10 creation, release, and source-editing RPCs. Both contracts have distinct
generated procedure sets and endpoint parity entries; the UI and CLI callers
use the authoring contract while preserving their existing workflows.
`AdoptionIntakeService` now owns the five scenario, recommendation,
reconciliation, discovery, and confirmation RPCs, leaving 15 on
`AdoptionsService`. All current services are within the 16-RPC ceiling. The
obsolete catalog-only SearchAssets/Status/Reindex RPCs were removed in the
2026-11-21 cleanup, leaving `CatalogService` at 13. The remaining map is in
[WORKBENCH-LEGACY-OWNERS-AUDIT.md](WORKBENCH-LEGACY-OWNERS-AUDIT.md).

**Historical integrated verification (2026-11-21; prior build):** `make -C packages/proto
verify-committed-gen`, uncached full API `go test -count=1 ./...`, CLI
`UPDATE_CLI_EVIDENCE=1 go test -count=1 ./...`, and UI `pnpm vitest run`
(140 files / 833 tests), lint, type-check, and production build pass. Lint and
type-check retain five pre-existing catalog warnings; the build retains the
i18n mixed-import and CodeEditor chunk-size warnings. On final healthy build
`sha256:0d1dbff4372ddf2a0af0d7851a5c8c6f20b064aa1f4dd8ba92f86041982f6289`, the
served 100-query hybrid gate passed (100/100; p50 177.0 ms, p95 357.0 ms, max
466.6 ms against the 1,500 ms p95 target). The served browser journey imported
the source at
`~/design-studies/eight-screens/index.html`, selected “Money,
night,” and verified the import/edit/replace/reparent flow with no page errors.
Its BAS capture step returned HTTP 504 because ad hoc capture requires an
authenticated account, so it did not produce live drift evidence. The current
code-only inventory is +5,106 lines versus HEAD: tracked files add 8,472 and
delete 21,910 lines, while 131 untracked source files contribute 18,544 lines.
The counting extensions and breakdown are recorded in §0. This does not meet
the net-negative deletion-ledger gate and remains reported debt.

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
- `go test ./...` passes; any failed gate is resolved or explicitly retired with
  current evidence.
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

## 2. Baseline snapshot (measured 2026-11-19; historical)

This section records the discovery baseline that motivated the program. It is
not a description of the current implementation. The implementation status and
remaining work are maintained in §0 above; use that section together with
`docs/internal/PROBLEMS.md` and `docs/internal/PROGRESS.md` for current truth.
In particular, the five-destination workbench, workspace aggregate, unified
search, Review/Stats services, and transactional legacy-store retirement listed
below as gaps have since been implemented.

### 2.1 Surfaces and scale at discovery

| Area | 2026-11-19 snapshot |
|---|---|
| Served UI routes | Home `/`, Components `/components`, Component `/assets/:id`, Preview, Design `/design` (+ scenario/page/sketch redirects), Review, Coverage, Capabilities, Activity, Settings, Create `/new` |
| Primary rail | 7 destinations + Settings + New |
| Workbench UI | ~22,300 lines of TS/TSX/CSS under `ui/src/workbench` |
| Largest files | `DesignStudioPage.tsx` 1,246; `useStudio.ts` 930; `templates/library.ts` 816; `ComponentPage.tsx` 790; `CoveragePage.tsx` 443; `HomePage.tsx` 569; `WorkbenchShell.tsx` 453 |
| API modules | 16 domain modules; 21 proto-backed Connect services; no legacy design/template REST modules |
| Largest API surfaces | `CatalogService` 16 RPCs; `AdoptionsService` 15 RPCs; `ExperienceService` 13 RPCs; `ReviewService` 12 RPCs; `ComponentsService` 12 RPCs |
| Catalog | 478 desired assets, 275 published manifests, 19 page types |
| Experience | 14 RCL experience pages, 54 experience components, 3 journeys |
| Studio data | 2 designs, 3 frames each; 0 template rows; 8 code-defined templates |

### 2.2 The central problem at discovery: one concept, five owners

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

### 2.3 Search at discovery

- Four UI surfaces: Home form (navigates with `?q=`), Components client filter,
  ⌘K client filter, studio template `<select>`.
- One server index: `internal/catalogsearch` over 478 catalog assets, token
  substring, no IDF/phrase/AND/type prior; not called by the UI.
- `searchDesignAssets()` is defined in the UI and unused.
- No workspace/page/state/command/evidence result kinds.
- `CatalogService.SearchAssets/Status/Reindex` exist but the UI ignores them.
- “landing page” on Components returns zero; ⌘K returns one scenario; the server
  index returns `LandingPage` plus every page type at 0.5.

### 2.4 API and contracts at discovery

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
- Calibration fixture DDL for `reconcile_evidence` is centralized in
  `internal/gates/calibration_schema.go`; production schema ownership is
  unchanged.
- The active selected-core contract at `docs/internal/goal/TARGETS.md` exists;
  its 12 RCL-REHAB obligations remain `not_implemented` until current evidence
  is attached. Workbench evidence does not certify the selected core.

### 2.5 Storage and migration at discovery

- SQLite stores components, versions, adoptions, evidence, version ledger,
  designs, templates, assets, workflows.
- Authored experience pages live as files inside each scenario.
- Migrations are a mix of `CREATE TABLE IF NOT EXISTS`, `internal/designs.Migrate`
  with `ensureColumn`, and domain-specific migration helpers. There is no single
  ordered migration ledger.
- Templates have their own table and API; designs have another.
- Local fallback stores are not a cache: they can become the only copy and cannot
  be promoted when an offline-created id reaches the server.

### 2.6 UI complexity and duplication at discovery

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

### 2.7 Measured quality baseline at discovery

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

### 2.8 Documentation and requirements contradictions at discovery

- `PRD.md` OT-P0-014 retains the original **eleven-page** acceptance as
  historical scope and now points to the current five-destination model.
- `WORKBENCH-REDESIGN.md` records the historical 11-page implementation;
  `WORKBENCH-MATURATION.md` proposes the five-page target. Both are useful only if
  one is clearly current and the other historical.
- `DESIGN.md`, the PRD and `requirements/25-rehabilitation` now resolve to
  `docs/internal/goal/TARGETS.md`; the selected-core obligations remain open.
- `docs/studio/DEBT.md` has been re-baselined on 2026-11-21 against this
  program, the rehabilitation contract and current problems. Its prior Nov 19
  audit is preserved as `DEBT-HISTORY-2026-11-19.md`; historical line numbers
  and findings are not current claims.
- The Design experience contract was reconciled to the workspace index/studio
  and the current executable route inventory check passes; recheck contracts
  when routes change.

### 2.9 Known gaps and bugs at discovery

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
- At this 2026-11-19 snapshot, no `TARGETS.md` or single goal ledger existed;
  `TARGETS.md` was later restored as the active selected-core contract.
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
| `WorkspaceAssistService` | 4 now | propose/refine and inference status |
| `WorkspaceCandidateService` | 7 now | candidate listing, placement, region mapping, save and render |
| `WorkspaceCaptureService` | 6 current (target ≤16) | capture exact rendered candidates, status, screenshot, attach, cancel, retry |
| `ReviewService` | ~5 | findings list/get, dispositions, assign, acceptance/critique |
| `CatalogService` | ~8 | assets, graph, relationships, ports, structure |
| `PageTypeService` | ~3 | page types, regions, required states |
| `CoverageService` | ~4 | coverage, next work, run/record gate |
| `SearchService` | ~4 | query, suggest, status, reindex |
| `ActivityService` | ~2 | list/stream events |
| `LibraryHealthService` | ~2 | typed health snapshot by section |
| `ExperienceService` | 13 current (target ≤16) | scenario page inventory and file-backed intent read/write, verification, import, placement, history, recovery, template and brief workflows |
| `AdoptionsService` | 15 now | adoption registry, compatibility and lifecycle |
| `AdoptionIntakeService` | 5 now | scenario selection, recommendations, reconciliation and provenance discovery |
| Existing | unchanged | versions, deps, themes, preview, workflows, health and inventory |
| `ComponentsService` | 12 now | registry projection/index, manifest-version reads, stories, preview frames, styles |
| `ComponentAuthoringService` | 10 now | canonical component create/release lifecycle and guarded source editing |

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
- Resolve all references to `docs/internal/goal/TARGETS.md` (now present).
- Keep the PRD's original 11-page OT-P0-014 acceptance text as historical
  scope, with a clear pointer to the five-destination current model.
- Keep experience contracts and route inventory aligned when routes change; the
  Design contract and executable route check have been reconciled.
- ADRs: composition ownership, template-as-provenance, page type vs page,
  activity durability, findings disposition authority, search engine, no silent
  local fallback.

## 4. Original migration strategy (historical plan)

The steps below describe the intended sequence when the program began. Most
have since been executed; remaining certification is listed in §0. Do not use
this list as a current backlog.

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

## 5. Original program phases (historical plan; current status is in §0)

| Phase | Scope | Exit evidence | Deletions |
|---|---|---|---|
| **0. Truth and guardrails** | Re-baseline tests/gates; inventory; fix missing TARGETS; approve ADRs and target contracts; freeze old features | One status page; three Go failures assigned; docs links resolve; target schema/ADRs approved | Stale duplicate plan docs; broken references |
| **1. Contracts and schema** | workspace/search/findings/activity/health proto; migrations; generated clients; adapters; no UI | Proto parity; migration dry-run; repository tests; schema lint | REST design/template exceptions; inline DDL copies |
| **2. Workspace core + Design** | Workspace aggregate; pages/displays/states; revisions; assets; fork; migrate designs/templates/experience bindings; Design index + studio + flow | Harbor 4-page/12-display/12-state flow; template fork; legacy design opens; state evidence; 390/320 | `studio_templates`, `internal/templates`, code templates, local template store |
| **3. Search** | Unified index; FTS5 + `ai-go/search`; commands; Start + Spotlight + Ask; facets/status | Golden queries, p95, degradation, command shortcuts, no client cap | `catalogsearch` as the only index; 4 UI search paths; `searchDesignAssets` |
| **4. Start + Library** | Start Atlas/categories/continue/templates/pulse; Library recent/featured/collections; re-home Home modules | Relocation ledger complete; 5 routes; phone tabs | Home duplication, Coverage/Capabilities/Activity routes, Home-only widgets |
| **5. Review + Stats** | Findings projection/dispositions; activity events; health sections; Review queue; Stats 4 views | All four Stats views; durable attribution; disposition write; honest unavailable | `findingsFromCoverage`, UI activity derivation, 3 separate pages |
| **6. Cleanup and certification** | Delete legacy; fix 3 Go failures; one device/verdict/timestamp/error owner; a11y/performance; docs/experience; captures; net-negative report | All gates green; axe/phone/reduced/forced; no dead code; one canonical docs set | Legacy pages/CSS/exports, duplicate metrics/verdicts |

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
- `go test ./...` passes; the self-hosting floor is met at 36 consumed assets
  against 265 implemented, without lowering the reviewed minimum of 36.
- `make check` is read-only; formatters are separate.
- Deletion gates: no references to retired modules; no dead exports/CSS; no file
  over budget.
- Net-negative line report after cleanup.

## 7. Deletion and simplification ledger

| Remove / replace | Replaced by | Why |
|---|---|---|
| `internal/templates` + `/api/templates` + `studio_templates` | Workspace provenance `curated` + fork | One composition store |
| `ui/src/workbench/templates/local.ts` browser shelf and fallback | WorkspaceService | No browser-only template ownership |
| `ui/src/workbench/templates/library.ts` curated template shelf | Reviewed workspace seed + DB | Templates are workspace aggregates; the file now has only a neutral blank-page starter |
| `internal/designs` (after migration) | `internal/workspaces` | One aggregate |
| `sketch` composition (`document`, `store`, candidates, placements, revisions, template render) | `workspaces` + `experience` reader | **Open:** `Document`/`Store.Save` still author composition in experience JSON; inventory, bind/adopt, migrate with parity/recovery evidence, then remove legacy fields and operations (see open composition ownership gap above) |
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
| `ui/src/api/versionHistory.ts` | deleted; Vite coverage exclusion removed | Repository-wide search found no imports or consumers of `listVersionHistory` / `VersionHistoryRow`; the helper only joined APIs at a deleted call site |
| localStorage composition/design/template stores | Workspace API; explicit offline queue or read-only | No silent split brain |
| Calibration fixture DDL duplication | addressed in `api/internal/gates/calibration_schema.go`; production schema ownership remains open for its owning schema provider | Fixture DDL has one helper owner |
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

## 9. Original first-ten-days checklist (historical)

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
10. Continue the deletion ledger: remove dead CSS/exports. The `make check`
    Go-lint fallback was corrected to report formatting errors without rewriting
    files; `make fmt` remains the explicit formatting path. Do not delete the
    old design stores until migration parity.

## Appendix A — current inventory

### Routes and owners (current, 2026-11-21)

| Route | Current owner | Target |
|---|---|---|
| `/` | StartPage | Start |
| `/library` | LibraryPage | Library |
| `/components`, `/catalog` | redirects | `/library`, query preserved |
| `/assets/:id` | ComponentPage | Component (kept) |
| `/assets/:id/preview` | PreviewPage | Preview (kept) |
| `/design` | DesignIndexPage, WorkspaceStudioPage, ComponentContextRoute, AdoptExperiencePage | Design index/studio; component-context links fork curated workspaces; unbound experience pages offer adoption |
| `/design/:scenario/:page` | binding lookup, then AdoptExperiencePage or workspace redirect | Open bound workspace or explicitly adopt the unbound page |
| `/sketch/:scenario/:page` | redirect | `/design/:scenario/:page`, preserving query context |
| `/review` | ReviewPage | Review (kept, server findings) |
| `/coverage` | redirect | `/stats?view=quality` |
| `/activity` | redirect | `/stats?view=activity` |
| `/capabilities` | redirect | `/stats?view=services` |
| `/stats` | StatsPage | Overview, Quality, Activity, Services |
| `/settings` | SettingsPage | Settings (kept) |
| `/new` | CreatePage | Global New action/route |
| `/workspace-capture/:id` | WorkspaceStateCapturePage | Read-only capture renderer |

### Stores

| Store | Current | Target |
|---|---|---|
| SQLite `studio_designs`, `studio_design_assets`, `studio_templates` | Legacy sources imported transactionally by `internal/workspaces.Migrate`; dropped only after successful import and binding validation | Retired; workspace aggregates/assets own migrated data |
| SQLite `workspaces`, `workspace_pages`, `workspace_displays`, `workspace_states`, `workspace_assets`, `workspace_revisions` | Current durable composition aggregate | Kept as the single composition owner |
| Scenario `experience/pages/*.json` | Experience intent and page contracts | Intent stays owned by Experience; composition is linked through workspace bindings. Existing embedded sketch composition still needs the explicit migration slice documented above. |
| Browser `rcl.workbench.templates.v1` | Unsynced legacy drafts may be referenced by old component-context bookmarks; no usage telemetry exists | Keep the read-only bridge until the published compatibility window closes and privacy-safe usage evidence is collected |
| `workbench/preferences.ts`, `rcl.workbench.fixtures.*` | Device preferences and component-preview slot positions | Kept as local UI state; not composition persistence |
| `component_*`, `catalog_gate_evidence`, `version_ledger`, `assisted_workflows` | Existing domain owners | Kept |

### Historical evidence baseline (2026-11-19)

- UI: 123 files / 750 tests pass.
- Lint: 0 errors; 5 catalog warnings.
- Type-check/catalog: pass.
- Go: 3 failures in `internal/gates` and `internal/reconcile`.
- Captures: 11 routes have phone/desktop captures and axe at 390/320 per PROGRESS;
  full interaction-state phone journeys remain incomplete for 10.
- Requirements: 54 complete, 36 in progress, 16 planned, 16 not implemented,
  3 pending.

### Served keyboard/journey evidence (2026-11-21)

Against the local served workbench at `http://127.0.0.1:23906`, a fresh
headless Chromium keyboard run started at Start, moved focus through the primary
navigation, and used Enter to visit Library, Design, Review, and Stats. Each
destination rendered its page marker and announced its nav link with
`aria-current="page"`. From Design, Enter on **New workspace** opened a new
Workspace Studio; Space, Arrow Down, Enter reordered a layer-tree item. The
temporary workspace was deleted through `WorkspaceService.DeleteWorkspace`
afterward. No keyboard navigation defect reproduced.

The existing `tools/workbench_browser_journey.py` served run also passed
command-palette focus, `g d` to Design, all five destination routes without
horizontal overflow at 390px and 320px, Workspace Studio checks/replacement,
layer-tree pointer reorder, canvas reparent with reload persistence, and 320px
text editing. It returned
**partial**: `CaptureWorkspaceState` returned HTTP 504 because the capture
workflow requires an authenticated account; no capture evidence or visual-drift
history was produced. The public `react-component-library.itsagitime.com`
endpoint returned HTTP 403 to this environment, so the served evidence is for
the local scenario runtime only. This closes only the primary-route keyboard
journey sub-slice; it does not certify every route, phone interaction state,
external deployment, authenticated capture, axe/forced-colors/reduced-motion
journeys, or the whole workbench.
