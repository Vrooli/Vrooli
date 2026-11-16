# Problems and Solutions - Scenario Dependency Analyzer

## 2026-11-13 — Install gateway reliability pass

Agents were falling back to raw package managers because the governed gateway
refused approved installs, said "approved" in a dry run and refused the apply,
or damaged state. Evidence: QA reports `knw-1794618858815944773` and
`knw-1794619329121707178`, thirty days of `friction.error_signatures` for this
owner, and a refusal census over every pnpm, npm and Go surface in the fleet
(a request to add a small registry package, or to re-require an existing Go
module, planned through `PlanDiskAdmission`).

Ranked friction and what changed:

1. **Every external Go install was refused at apply** ("not backed by an exact
   repository-local module"): 216 of 216 Go surfaces in the census. Fixed:
   a published module is sized from the requirement closure it would add; a
   module the `go.mod` already requires and the cache already holds costs
   nothing. Census after: 488 of 488 admitted. `go get` that fails restores
   `go.mod`/`go.sum`.
2. **A dry run approved what apply refused.** Disk admission only ran on apply.
   Fixed: governance, planning and the capacity-broker check are one routine for
   both, and a preview reports its planned peak.
3. **pnpm apply wiped `node_modules`.** The gateway pinned `store/v10` while the
   lifecycle uses `store/v3/v10`, pnpm recreated `node_modules`, and an
   integrity-less lock entry then made it unrebuildable. Fixed: the store is
   whatever `pnpm store path` reports in the surface; a store mismatch never
   reaches pnpm (manifest and lock are updated, `node_modules` is left for
   `vrooli scenario setup`); the destructive "repair" retry is deleted; a lock
   entry pnpm refuses for missing tarball integrity is re-resolved in the staging
   copy. In the fleet census 61 surfaces are still linked by pnpm 9 (`store/v3`).
4. **Big shared packages could not be adopted** ("registry metadata request or
   byte budget exceeded"). The gateway re-simulated pnpm's resolver
   (ranges, overrides, packuments) to size a whole closure. Fixed: pnpm and npm
   resolve in a staging copy and only the locked packages `node_modules` lacks
   are sized. Adding `@vrooli/file-viewer` to `web-console` adds one lock entry.
   The 1,670-line resolver simulation (range parser, packument walk, release-age model) was replaced by a staged resolution that pnpm and npm share.
5. **A tarball URL could not be exact**, so it was refused. Fixed: a plain https
   `.tgz` URL is exact by construction; its lock integrity is verified against
   the downloaded bytes and printed in the preview.
6. **npm surfaces with dependencies were always refused** (41 `package-lock.json`
   surfaces). Fixed by the same staging. Two latent bugs surfaced: npm rejects
   `/dev/null` as both user and global config, and template lock generation
   passed pnpm's `--lockfile-only` to npm.
7. **`template/<id>` installs were always refused** (no size probe). Fixed with a
   fixed lock-only allowance.
8. **`unrecorded` refusals** (12 calls) were correct but their next step,
   `approve-observed`, fails for a package no scenario uses yet. The refusal now
   names `deps approved approve …` as well.
9. **`--exact` with a range or tag**, **`javascript`/`js` ecosystem**, **shared
   package named as a scenario**, **global `--dry-run` on a command that already
   previews**, and a **45 s client timeout** on a slow resolve: each is now a
   message that says what to do, or accepted.
10. **`deps approved search` stalled or returned EOF** when semantic ranking hung
    on a cold embedding model. Ranking now has a four-second budget and falls
    back to keywords. An EOF during an API restart is expected.
11. **No governed way to remove a JavaScript dependency.** `deps remove` was the
    Go root-module verb only, so pruning a surface meant a raw `pnpm remove`.
    Fixed: `deps remove npm/<pkg>... --scenario --surface` stages the removal like
    an install, previews the lockfile packages that leave, and applies with the
    same store, rollback and `source_updated` outcome.
12. **A Go install of a repository package was refused for four unrelated
    reasons, one behind another.** (a) The probe walked and sized the whole
    module tree, and the root module is the whole checkout: it failed on the first
    symlink or socket and then on the 2 GiB bound. (b) It walked the module's full
    requirement graph, 300+ modules the build prunes away, and asked the proxy
    about first-party modules that are never published (404). (c) Module zips were
    matched against the case-escaped path, so every module with an upper-case
    letter in its path (`BurntSushi`, `GoogleCloudPlatform`) failed. (d) A
    dependency whose go.mod carries `replace`/`exclude` was refused, though the
    go command ignores both outside the main module. Fixed by replacing the
    closure walker with the question the go command answers: `go list -deps`, run
    offline and read-only in the owning module, names the external modules the
    package needs; the module cache makes cached ones free and the rest are sized
    from the proxy like a published module. A repository module is used in place
    and adds nothing.
13. **A lockfile an older or raw pnpm wrote could not be changed through the
    gateway.** Two shapes, both seen on the first real prune and bump. A tarball
    URL entry with no integrity (pnpm itself refuses it, and `pnpm remove` re-writes
    it the same way, so the old "repair" claimed an integrity it never recorded):
    the URL is now downloaded once, hashed, and the lock gets a one-line
    `integrity:` edit that a reviewer sees in the preview (the digest matches what
    a fresh `pnpm add` records). A package locked at a version younger than the
    seven-day release-age policy (notification-hub/ui held `@napi-rs/canvas`
    1.1.1, 47 hours old): the policy is not relaxed; pnpm re-resolves that package
    to a mature version in the staging copy and the preview lists the lockfile
    change. A requested package that is itself too new is still refused.
14. **The gateway and the control plane disagreed about the pnpm store.** A package
    refresh (`vrooli package refresh file-viewer`) aborted with
    `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`: `packages/file-viewer/node_modules`
    was linked against `store/v10` while the user's pnpm config selects
    `store/v3/v10`. The control plane's package and setup installs now follow the
    recorded store (`internal/packagegov/pnpm_store.go`), and the gateway does the
    same when pnpm can select it, instead of skipping the relink. Three related
    gateway fixes: a flag after the package argument (`-D`, `--save-exact`)
    no longer hides the requested specifier from the tarball check; a `file:`
    dependency is compared with its governance record by the package it names,
    so `file:../react-component-library` (declared from `packages/`) matches the
    recorded `file:../../../packages/react-component-library`.

15. **`deps approved search` failed with "unavailable: unexpected EOF" or
    `deadline_exceeded` (13 calls in 30 days).** The deadline was the cold
    ranking stall fixed in item 10 (search now answers in 1-2 s). The EOF is an API
    that restarts mid-request. The CLI's governance client now repeats a
    read-only call (`List*`, `Search*`, `Explain*`, `Validate*`, `Get*`,
    `Preview*`) up to three times, with a note on stderr, when the API reports
    itself unavailable. Mutating calls (install, remove, upsert, approve) are never
    repeated, and a timeout is not retried.
16. **Two refusals named no next step.** `surface directory not found` now lists the
    surfaces the scenario has (and says when the scenario does not exist), where
    agents had guessed `tools/yt-dlp` or an `api` that is a package.
    `approve-observed` and `widen-range` for a package no scenario declares now
    say to use `deps approved approve ... --rationale` instead.
17. **A shared-package build refused with "source changed while the build was
    running" could not say what changed**, so the last three instances were
    diagnosed from file mtimes. The refusal and its lifecycle log line now name the
    changed files (first five, with a count, marked added, modified or removed, or
    "artifact selection"). The digests are unchanged, so existing freshness stamps
    stay valid. The build that rewrites its own `package.json` (RCL's materialize
    step) now declares it as an output so it is not read as an edit.

18. **A legacy-layout consumer could not be started or restarted.** The repair of
    item 14's mismatch ran only when no registry instance existed, but a start
    registers its own `starting` instance before it builds and a restart builds
    while the old instance runs, so both were refused and only a manual `setup`
    (about a minute of downtime) repaired it. Instances the operation owns are no
    longer other occupants; the old `node_modules` is renamed aside and restored if
    the install fails instead of being deleted first.
19. **Every first build after a proto change failed once** with `(1 changed:
    artifact selection)`. `protogen generate` publishes and selects the artifact
    it builds, and that selection move was read as a concurrent edit. A step that
    declares an artifact selection now has its own selection move accepted
    (freshness is recorded against it); a source-file change in the window is still
    refused.

20. **A security fix an existing range already admits could only be taken through
    an override that needed a per-scenario approval.** notification-hub/ui held
    `body-parser` 1.20.5 and `proxy-addr` 2.0.7 under two advisories; express 4.22.1
    asks for `~1.20.3` and `~2.0.7`, so 1.20.9 and 2.0.8 are already in range (both
    older than the seven-day release-age policy) and about ninety UI lockfiles are
    simply stale. `deps install npm/body-parser --version override:1.20.9` was refused
    with `scenario_not_allowed`, and widening that approval would have pinned every
    manifest to exact versions for good. New verb `deps update` (RPC
    `UpdateLockedDependency`) refreshes named locked packages inside their ranges in
    the staging copy, previews the closure change, and applies it with the install's
    store, admission and rollback. No manifest or approval is involved.
21. **`deps approved approve --help`, `remediate --help`, `explain --help` and the
    rest of the legacy commands printed only `Usage: <command>`.** The commands parse
    their own flag sets, so help had nothing to render. Help now reads each command's
    flags from its own flag set (`support.DescribeFlags`, run only when help for that
    command is requested) and adds the usage line and an example from one catalog
    (`commandDocs`) that the usage errors read too. A test fails for any subcommand of
    `deps` or `deps-approved` whose help has no flags or no example.

22. **Every first-party package needed its own governance record.** An in-repo
    package is reviewed by repository review, so the registry added a record per
    package and no extra safety; a new package could not be installed until
    someone wrote one. `governInstall` now admits a request the gateway can
    prove first-party from the repository layout (verdict `first_party`; npm
    `file:` link to a named package beneath `packages/`, Go package inside a
    module under `packages/`), writes one audit line per applied install to
    `.vrooli/dependencies/first-party-installs.jsonl`, and lets a recorded
    denial or deprecation still win. Validation stops asking for a record of
    the same proven `file:` links. Scenario modules and anything the layout
    cannot prove keep ordinary governance.

23. **A fleet security refresh needed package names per scenario.** The fix for an
    advisory is the same everywhere, but `deps update` only took package names, so a
    caller had to look up which locked packages each scenario's advisory names.
    `deps update --advisory <GHSA|CVE>` resolves the names from Security Health's
    exposure evidence for the scenario and surface, and the command is now a
    Connect-bound manifest command (`run_eligible`) so a governed program can run it.

24. **A lockfile-only update was held to the install-sized disk floor.** With the host
    inside its 120 GiB floor (a large unreclaimable build cache), every `--apply` was
    refused, including the ones that write kilobytes: a surface whose link is skipped
    publishes only `package.json` and the lockfile (receipts show 4 MiB planned, 0 to
    77 KB actual). `deps update --lock-only` publishes the inputs without linking, and
    any apply that will not link is now reserved as a bounded `metadata` disk job in
    the capacity owner (64 MiB total, 8 GiB floor, never lowering another job's floor).
    An apply that links node_modules still downloads into the package store and is
    held to the full floor. The same class admits a Go install that fetches nothing (a
    first-party `packages/` module or package through a local replace, or modules the
    cache already holds): it rewrites go.mod and go.sum, so it is reserved at its true
    size (about 25 MiB, with a 16 MiB build-cache allowance instead of 256 MiB) as
    `DiskAdmissionPlan.WritesInputsOnly`. A Go install with anything to download keeps
    the full floor. Note that a package of the root module has the root module's
    import path (`github.com/vrooli/vrooli/packages/<name>`), not a short one.

25. **`deps update` skipped transitive names named beside a direct one.** pnpm limits
    one `update` to the direct dependencies as soon as any name it is given is direct,
    so `deps update npm/postcss npm/qs` moved postcss and reported qs "unchanged" with
    a newer in-range qs available. The update now splits the names by the manifest and
    runs the transitive ones as a second command of the same staged resolution
    (`Resolution.ThenArgv`; the preview shows both commands).
26. **A fleet refresh needed advisory ids and a table of fixed versions.** The program
    asked its caller for up to eight ids and a `floors` string. `--advisory all`
    selects every exposure Security Health reports on the surface, and every update
    response now names the locked versions still inside an advisory
    (`exposed: <name>@<version> to <ids> (fix on this line: X | no fix on this line;
    fixed in ...)`), judged from the same evidence by release line (major, and minor
    for 0.x). The program reads those notes and the `floors` input is gone. An
    advisory whose evidence carries no range is never judged exposed. The evidence is
    one Security Health query per call, which returns at most 200 groups; the busiest
    scenario has 180, so a scenario near that bound would be judged on a prefix.
27. **A stale exact pin or override silently blocks a refresh.** An exact manifest
    pin or `pnpm.overrides` entry set to an earlier fix (brace-expansion 5.0.9,
    js-yaml 4.3.1, express>body-parser 1.20.6) keeps the lock below the next fix, and
    `deps update` correctly never changes a manifest. `deps install --version
    override:` refuses the move with `scenario_not_allowed` when the package's
    approved record lists scenario exceptions that do not include the scenario, even
    though an override of an already-locked transitive package adds no dependency.
    Left as a governance question: whether an override that only raises the pin of a
    package already in the lock should skip the scenario-exception check (it would
    still be held to the record's version range and any denial).

28. **`deps install --version override:` could not publish its own lock, and had no
    lock-only mode.** The override was written to the real `package.json` after
    admission, but the staged resolution had been made from the manifest without it, so
    the publish either found nothing to change or refused ("changed after the install
    was admitted"). The override is now applied to the staged manifest, the preview
    shows the lockfile effect, `--lock-only` is available (metadata admission), and
    `--override-selector` raises one exact selector (`brace-expansion@1`,
    `minimatch@3>brace-expansion`) instead of forcing a package-wide pin that no
    multi-line package can take.

Left alone, deliberately:

- pip, poetry, uv and cargo installs still have no bounded size probe; the
  Python path needs a hashed-lock design of its own (see the 2026-10-22 entry).
- yarn and bun surfaces: none exist in the fleet.
- Scenario-to-scenario Go module edges still follow the existing local-replace
  rules; only `packages/*` is first-party for admission.
- The "CLI rebuilt from current sources; restarting command..." line that
  friction counts as a failure (13 calls) is cli-core's stale-binary rebuild.
- `freshness` ambiguity for modules declared from several directories already
  has an actionable message in the current source.

## 2026-10-22 — Embedded Python lock forwarding

Image Tools' `api/requirements.txt` forwards to the embedded
`internal/pydeps/requirements.lock` and is not an installable host environment.
The gateway currently previews `pip install` against that wrapper, bypassing
the declared ranged-input/hashed-lock owner. Reproduced by the approved hybrid
scene delivery's `deps install pip/omegaconf@2.3.0 --scenario image-tools
--surface api --json` preview; no install was executed.

Required repair: recognize a single local lock forwarder, resolve it within the
declared surface including symlinks, update its canonical input and hashed lock
through the existing wheel-only compiler, and leave the forwarding file intact.
Refuse missing/escaping targets; preserve ordinary requirements manifests.
This is a bounded dependency-owner repair needed by the approved image-to-object
runtime, not a scenario-wide maturity claim. Keep source and lock publication
coherent and retain focused regression/native-consumer evidence.

Implementation checkpoint: forwarding now resolves to the contained canonical
owner, rejects symlinked/nonregular/oversized input and missing or escaping
closures, and updates the requested package without collapsing distinct PEP 503
identities. All source and lock files are staged before publication, original
modes are preserved, and publication errors restore earlier files. This does
not claim crash-atomic multi-file publication; conformance verification remains
required after interruption. Focused installgateway and dependencygovernance
packages pass. The native SDA instance adopted the repair on 2026-10-22 at
10:24:22 America/New_York; evidence is retained under the hybrid-device-scenes
full-delivery evidence root. TripoSR dependency resolution/qualification remains
separate work; a repaired route alone does not prove that runtime installed.

## Problems Discovered (2025-09-28)

### 1. CLI-API Graph Type Mismatch
**Problem**: The CLI expected graph types like "hierarchical", "network", "circular" but the API expected "resource", "scenario", "combined".
**Solution**: Fixed the CLI to use the correct API types and added mapping for user-friendly aliases.
**Status**: ✅ Fixed

### 2. Missing Optimize Command
**Problem**: The optimize command was specified in the PRD as P0 but was not implemented in the CLI.
**Solution**: Added the optimize command with basic functionality and a roadmap for full implementation.
**Status**: ✅ Implemented (basic version)

### 3. Database Initialization
**Problem**: The database schema exists but may not be properly initialized when the scenario starts.
**Solution**: The schema.sql is present but requires proper lifecycle integration for auto-initialization.
**Status**: ⚠️ Needs verification

### 4. AI Resource Integration
**Problem**: Qdrant-based semantic matching may fail silently if resources aren't running.
**Solution**: Added fallback heuristics when semantic resources are unavailable.
**Status**: ✅ Fallbacks implemented

### 5. Test Failures
**Problem**: The `make test` command fails at the API health check step.
**Solution**: The health endpoint works when tested directly, suggesting a timing or port issue in the test script.
**Status**: 🔍 Needs investigation

## Technical Debt

1. **Qdrant Integration**: Currently uses exec commands to call resource-qdrant, should use proper API client
2. **Optimization Engine**: Currently returns placeholder data, needs full implementation
3. **Circular Dependency Detection**: Graph algorithms exist in code but not fully integrated
4. **Historical Tracking**: Database tables exist but no automatic tracking implemented

## React-Vite 1.1 Migration Baseline (2026-06-14)

### Template Provenance Missing
**Problem**: `.vrooli/service.json` did not declare `generation.template.id`, so `ui-health validate scenario scenario-dependency-analyzer` skipped template-aware validation.
**Status**: Fixed. Provenance now declares React-Vite template version `1.1.0` with the current template manifest/content hashes. `vrooli scenario template drift scenario-dependency-analyzer --json` reports an ok status. Residual SDA-specific debts below remain tracked separately and do not represent template hash drift.

### CLI Manifest Missing
**Problem**: `cli-health validate scenario scenario-dependency-analyzer` reports `manifest.missing` for `cli/manifest.json`.
**Status**: Fixed for governance coverage. `cli/manifest.json` now declares the Connect-backed `graph actual` surface and `cli-health validate scenario scenario-dependency-analyzer` passes. Runtime command registration remains code-backed until a later manifest-loaded CLI migration.

### UI Architecture Drift
**Problem**: The UI still uses a monolithic `App.tsx` tab shell and custom server-state hooks rather than the template's providers, router, layout, React Query, selectors, i18n, theme, and test harness architecture.
**Status**: Partially fixed. The UI now has template slot layout scaffolding, React Router path navigation, selectors, generated strings scaffolding, theme/test/api/i18n slots, an ErrorBoundary, concrete feature folders for graph/deployment/catalog/governance, React Query-backed graph/catalog/governance server state, a generated proto/Connect Governance client, `react-i18next` provider usage, and a Vitest/Testing Library/a11y harness. Remaining UI work is broader test coverage and Phase 6 design-system/token cleanup.

### UI Test Coverage Floor
**Problem**: `pnpm test:coverage` now runs, but coverage is low because only the first routing/API/ErrorBoundary/a11y harness slice exists.
**Status**: Partially improved. Route tests now cover graph API failure rendering, graph table node selection and empty-filter states, catalog Scan & Apply mutation behavior, deployment degraded states from metadata gaps/blockers, and deployment scan failure messaging. Pure deployment status-helper tests cover critical/blocked/gap aggregation semantics. DAG export fallback/help behavior and deeper component coverage remain open before enforcing coverage thresholds.

### UI Bundle Size Warning
**Problem**: `pnpm run build` passes, but Vite reports the main minified JS chunk is above its default 500 kB warning threshold after adding router/query/i18n/test-adjacent runtime dependencies.
**Status**: Fixed. Orientation, graph, deployment, and catalog route surfaces are now lazy-loaded from `ui/src/app/routes.tsx`, so the production build emits route chunks and the main JS chunk stays below Vite's default warning threshold without raising the limit.

### UI Server CORS Blocked Browser Smoke Assets
**Problem**: Browser-driven smoke and Lighthouse phases saw `403 Origin not allowed` for built JS/CSS assets because SDA passed an empty default CORS allow-list to `createScenarioServer`. Direct curl checks without an `Origin` header returned `200 OK`, which masked the issue outside browser execution.
**Status**: Superseded (2026-11-14). The wildcard workaround was removed fleet-wide. API Base now recognizes same-origin requests from Fetch Metadata, or from `Origin` matching `Host`, so origin-bearing asset requests pass without any `corsOrigins` setting, and cross-origin pages are refused. See `packages/api-base/SECURITY.md`.

### Documentation Manifest Drift
**Problem**: `knowledge-observatory docs audit scenario-dependency-analyzer --json` reported top-level `docs/api.md`, `docs/cli.md`, and `docs/integration.md` as orphaned docs, and localhost examples in prose were parsed as unknown machine-readable references.
**Status**: Fixed. The legacy detailed API/CLI references, integration guide, and manifest are now registered in `docs/manifest.json`; URL prose was reworded to lifecycle-assigned port guidance; `knowledge-observatory docs health scenario-dependency-analyzer --json` now reports `health_score: 1`.

### UI Design-System Debt
**Problem**: Feature components still contain raw status and palette utility classes, especially in deployment readiness and graph panels.
**Status**: Improved. Semantic status tones now live in `ui/src/theme/status.ts` and cover the repeated readiness/drift/error states. Deployment status derivation has been extracted into pure helpers, and deployment readiness rendering is now split across intro, summary, status-list, status-badge, and details components. A raw semantic palette inventory over `ui/src/**/*.ts{x}` no longer finds the tracked red/yellow/green/blue/slate/gray/amber/emerald/rose/orange status class families. Remaining design-system work is mostly targeted coverage before coverage thresholds are enforced.

### Measures Coverage
**Problem**: `vrooli scenario test scenario-dependency-analyzer` failed in `phase-measures` because `scenario_dependencies` was a stateful domain with no declared measure or waiver.
**Status**: Fixed for this migration phase. `measures-health validate scenario scenario-dependency-analyzer` now passes with explicit waivers for `scenario_dependencies`, `scenario_metadata`, and `optimization_recommendations`. The `scenario_dependencies` waiver is intentional because the current graph RPC returns a nested interface graph, not a measure-shaped scalar/table; remove the waiver when a dedicated Count/ListScenarioDependencies Connect RPC exists.

### Runtime Status Drift
**Problem**: `vrooli scenario status scenario-dependency-analyzer` reports unhealthy even though live `/health`, `/api/v1/health/analysis`, and UI `/health` endpoints return `200 OK`.
**Status**: Fixed. The runtime health probe was rejecting `/api/v1/health/analysis` because it returned analysis diagnostics without the standard health envelope fields. The endpoint now preserves its diagnostic fields while also returning `service`, `timestamp`, `readiness`, and `version`; after `make restart`, `vrooli scenario status scenario-dependency-analyzer --json` reports `health_status: healthy`.

### API Architecture Convergence
**Problem**: `api/internal/app` still owns the central Gin router, service registry, and multiple REST adapters, so some domains remain less discoverable than the React-Vite template's screaming-architecture target.
**Status**: Partially improved. Scenario catalog list/detail REST presentation now lives in `api/internal/catalog` with its own adapter tests and a tiny local service contract. Deployment readiness, DAG export, and bundle manifest REST presentation now live in `api/internal/deployment` with adapter coverage. Graph REST presentation, centrality analytics, cycle detection, generated Connect graph presentation, and dependency graph construction now live in `api/internal/graph` with adapter/domain tests. Analysis/scan REST presentation lives in `api/internal/analysis`; stored dependencies and impact routes live in `api/internal/dependencies`; proposal, optimization, and core-set presentation live in their matching domain packages. `api/internal/app/service_registry.go` is now only service composition; focused app service implementations live in `service_*.go` files until their workspace/store/runtime dependencies can move cleanly into domain packages.

### Test Genie Execute Parser Regression
**Problem**: On 2026-06-15, `make test` reached SDA's test phase but failed before executing phases because `test-genie execute scenario-dependency-analyzer --preset comprehensive` treated `scenario-dependency-analyzer` as a phase name. Alternate flag ordering treated `quick` as a phase, and `execute --help` advertised `--dry-run` while the parser rejected it.
**Status**: No longer blocking SDA validation. The external issue remains filed as `knw-1781498226583665032` for Test Genie ownership, but `make test` completed successfully on 2026-06-15 after the UI route-chunking slice.

### Proto Reachability Notes
**Problem**: `proto-health validate scenario scenario-dependency-analyzer` emits informational `proto.possibly_unused` findings for graph and health messages because the validator does not yet see all served/consumed reachability.
**Status**: Non-blocking. Keep graph contracts because SDA is the fleet interface graph authority; close reachability evidence during Phase 2.

### Dependency Governance Self-Review
**Problem**: Approved dependency governance originally had no review memory for many direct SDA dependencies, which made SDA fail its own dependency-health target with advisory warnings.
**Status**: Improved on 2026-06-17. `scenario-dependency-analyzer deps approved validate scenario-dependency-analyzer --json` now passes with no findings after package-by-package registry review for SDA's direct Go, UI runtime, generated local, graph/i18n, Radix, accessibility-test, lint, and build-tool dependencies. The remaining governance debt is model-level: the registry currently keys one record per ecosystem/package, so packages with multiple valid major lines across the fleet need broad `minimum` policies or a future multi-record/scope-aware model to express parallel approved ranges without weakening review intent.

## Recommendations for Next Iteration

1. **Priority**: Implement proper resource client libraries instead of exec calls
2. **Database**: Add automatic migration runner on startup
3. **Testing**: Fix the test suite to properly wait for API readiness
4. **UI**: Add WebSocket support for real-time updates during analysis
5. **Performance**: Add Redis caching layer for frequently accessed dependency data

## Actual Interface Graph Migration (2026-06-13)

### Declared Dependencies Are Not Actual Interface Usage
**Problem**: SDA historically reported declared dependencies from `service.json` plus local scanner heuristics. That cannot answer whether a scenario actually imports another scenario's protobuf or generated Go client, and it cannot reliably flag `service.json` drift.

**Planned solution**: Consume batch facts from `proto-health` and `code-facts`, attribute import paths to scenario slugs, and expose an on-demand evidence-tagged interface graph.

**Status**: Planned by `scenario-dependency-analyzer-actual-interface-graph-and-import-drift`.

### Scanner Ownership Boundary
**Problem**: SDA's scanner owns too much language-specific evidence logic. `detectPortCalls` follows obsolete `resolveScenarioPortViaCLI` usage, and `detectCLIReferences` is superseded by Connect/proto imports. Retaining this style would duplicate code-facts and keep dependency evidence brittle.

**Planned solution**: Delete the obsolete/superseded detectors during the graph migration. Keep only interim non-import signals with precise comments until upstream AST facts exist.

**Status**: Planned cleanup.

### Follow-Up: AST Facts for Runtime and CLI Usage
**Problem**: Runtime `ResolveScenarioURL*` calls and `vrooli scenario run` shell-outs are real cross-scenario usage signals, but they are not import-level evidence and should not be rebuilt as regex scanners inside SDA.

**Follow-up plan**: `scenario-dependency-analyzer-code-evidence-via-ast-facts`

**Scope**: Add AST analyzers in `go-code-graph` and `typescript-code-graph` for modern discovery calls and scenario shell-outs, surface those through `code-facts`, then delete SDA's retained regex detectors and delegate resource-usage detection to fact providers.

## Unified Graph Re-Sourcing + Ingest Sweeper (2026-06-22)

### Centrality is now sourced from a unified, evidence-tagged store
**Resolved**: `graph_edges` (SQLite) merges `proto_import ∪ go_import ∪ declared ∪ vrooli_cli ∪ resource`, deduped by `(from,to)` keeping the highest-confidence source and the union of evidence. `graph/builder.go` + centrality read this store; the legacy "analyze regex detector is the only writer" gap is closed. A default-ON, freshness-gated ingest sweeper (`internal/graphsweeper`) keeps it current with concurrency cap, wall-clock budget, single-flight, circuit breaker, and last-good degradation. `graph rebuild` / `graph sweeper status` CLI + REST verbs expose it. See `docs/perf/2026-06-22-sda-graph-ingest.md`.

### Proto-package-name vs scenario-name attribution (known limitation)
**Problem**: Both `proto-health` (`ProtoImport.ToScenario`) and `code-facts` (import paths under `packages/proto/gen/go/<pkg>`) attribute cross-package usage by **proto-package name**, which is often NOT the owning scenario's slug (e.g. `architecture` is `architecture-cartographer`'s package; `measures` is `measures-health`'s). The fleet drift therefore lists targets like `architecture`, `cli`, `measures`, `scenario-validation` that are not scenarios.

**Containment**: Graph generation and centrality filter scenario→scenario edges to **known scenarios** (the detector catalog), so these package-name targets never pollute centrality/importance (verified: 105 real scenario nodes, zero package-name nodes). The interface-graph-driven `scan --apply` proposal path (`applyInterfaceGraphScenarios`) applies the same known-scenario filter, so it never proposes a package name as a declared scenario dependency. The live `/drift` display still shows the raw findings (it reports what proto-health/code-facts attribute).

**Follow-up**: A package→scenario ownership map (owned by `proto-health` / the AST-facts follow-up `scenario-dependency-analyzer-code-evidence-via-ast-facts`) is required to attribute these edges to the real owning scenario. Until then, `declared` edges in the union carry the true scenario→scenario dependency, and the honest INFO-severity `declared-without-import-evidence` finding remains.

### Declaration-drift remediation status
**Delivered**: The remediation engine now drives proposals from the unified interface-graph evidence (not the weak `vrooli_cli` regex), filtered to real scenarios. Running `scenario-dependency-analyzer scan <name> --apply-scenarios` proposes/declares a scenario's import-level dependencies correctly.

**Deferred (operator sweep)**: Applying these declarations across the fleet's under-declared scenarios is a per-scenario operator action — each `service.json` edit is reviewed individually (no mass script, per repo policy) and validated against the owning scenario's own suite. The baselined scenarios in this change (scenario-dependency-analyzer, scenario-completeness-scoring, test-genie) carry **no** real-scenario under-declarations (their drift findings are all package-name attribution noise), so no in-scope `service.json` edits were required.

## Work ladder

- Scoped repair, 2026-11-05: W3 deployment-report path expansion. The live API reached
  24 GiB RSS within eight minutes; its stored reports occupied 134,315,007,268 bytes
  across 180 runtime directories. A 16-owner fan-in regression produced 510 nodes.
  Reports and target DAG exports now reuse bounded owner expansion; active shared
  dependencies count once in tier totals, and oversized legacy report reads require
  recomputation. The regression now retains 30 edges and about 12 KiB. Focused
  deployment/app/graph-ingest/sweeper tests pass. Live Android report regeneration
  reduced 8,568,754,340 bytes to about 848 KiB with 75 unique dependencies, and the
  heap returned to about 11 MiB. The fleet refresh completed at 14:30:43 EST,
  reducing the 127 live reports to 46,773,805 bytes; the client attachment timed
  out at 300 seconds while the original server request continued to completion
  (10m3.927s, HTTP 200). Existing scanner exclusions now run before file reads
  and prune excluded directories; a FIFO regression verifies that excluded inputs
  are never opened. Focused scanner/app/deployment tests pass.
  Test Genie unit run `20261105-191856-6c42fdd2` failed on the trusted-base
  closure, two CLI freshness tests, and UI coverage thresholds. The edited
  implementation's focused checks passed; the broader scenario gate remains failed.
  This scoped implementation repair makes no W0–W2 claim;
  the broader contract finding below remains unchanged.

- Rung: W0 (goal/problem contract comparison)
- Evidence: Indexed goals `search-hub-corpus-buildout` and `search-hub-federation-adoption` direct search capabilities for this scenario, while the P0 PRD targets do not name those search capabilities. The W0 contract therefore remains false for the scenario's broader goal set; this portability plan only consumes SDA's platform-verdict implementation and does not rewrite its product contract.
- Constraint: Do not substitute the portability plan or dependency-governance evidence for the unresolved search-goal contract mismatch. No lower scenario gate was run under W0.
- Measured: 2026-08-26.

- Scoped repair, 2026-11-12: W3 synthetic report persistence. Storage census found
  eight dot-prefixed test subjects with persisted dependency reports above 1 GiB,
  including three Android subjects at about 8.57 GB each. Their reports predate
  the bounded owner expansion fix and exceed the 32 MiB read limit. `PersistReport`
  now keeps dot-prefixed synthetic subjects request-scoped and removes an
  oversized legacy report only when that exact subject is recomputed. Canonical
  scenario reports retain their existing persistence path. No tests were run in
  this pass; no W0–W2 claim is made.

## 2026-09-07 — Shared selector consolidation

W3 implementation review under the existing selector contract; no W0–W2 maturity promotion is claimed. The install gateway now supports template/<id> UI surfaces. It regenerates the template lock in a temporary scenario-shaped directory and preserves file dependencies relative to generated scenarios. Both template installs succeeded; focused gateway/governance tests pass. Unit run 20260907-213240-942054bc returned PASS with advisory findings. An existing missing protovalidate CLI dependency was repaired through the gateway so lifecycle rebuilding could proceed. Shared evidence and limitations: `packages/ui-selectors/README.md`.
