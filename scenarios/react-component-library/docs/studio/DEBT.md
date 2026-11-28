# React Component Library debt

Re-baselined 2026-11-22 against the active rehabilitation contract in
[goal/TARGETS.md](../internal/goal/TARGETS.md), the implementation status in
[WORKBENCH-PROGRAM.md](../internal/WORKBENCH-PROGRAM.md), and current gaps in
[PROBLEMS.md](../internal/PROBLEMS.md). This file tracks verified open debt;
older findings are preserved in [DEBT-HISTORY-2026-11-19.md](DEBT-HISTORY-2026-11-19.md)
and are not current claims.

## Open delivery debt

| Priority | Debt | Current evidence | Closure evidence |
|---|---|---|---|
| P0 | Rehabilitation is not fully certified. | Latest retained 33-phase Test Genie run `20261122-124253-457fa6e9` completed **FAIL**: 23 passed, six failed, one provider-unavailable, two not evaluated, one skipped. Failures include Home Lighthouse 0.61 vs 0.75, tidiness, workflow, experience-contract, Stripe outcomes, and signature-blind companion findings. Experience required authenticated BAS; component tests timed out; Agent Manager conformance was unavailable. The performance artifact has no report or source-owner breakdown. Afterward, focused API, CLI, renderer/importer, adoption/storage, and component/search suites passed; UI coverage remains 178 files / 1,175 tests at 91.88% statements, 85.01% branches, 85.32% functions from the latest full run. No current full-source certification receipt exists. | Current-source validation receipt with applicable findings resolved or owner-routed, and BAS evidence attached where the contract requires it. |
| P0 | Source-anchored drift still lacks authenticated BAS capture evidence. | Per-state source image anchors now persist their digest and provenance; candidates remain visibly separate until a reviewer explicitly accepts one. Source-to-capture comparisons and heatmaps persist against immutable source-baseline and capture-receipt identities. The fresh v46 browser journey returned HTTP 504 with BAS message `workflow execution requires an authenticated account`; no capture ID or evidence was returned. The full import/edit/RCL replacement/reorder/reparent/phone/health journey otherwise passed. No authenticated workspace-state capture receipt is attached. | Run an authenticated BAS capture for each selected screen and compare it to its explicitly accepted source anchor; retain visible history tied to the workspace state and source revision. |
| P1 | The strict source-line reduction gate is open. | The latest 2026-11-22 production-only inventory measures +12,134 lines across the scenario and catalog package in the shared worktree. Shared concurrent edits prevent attributing that total to RCL; no reviewed owner-by-owner replacement ledger proves a net decrease. This continuation removed 55 unused UI lines, but that local deletion does not close the aggregate gate. | Reviewed owner-scoped replacement ledger demonstrates a net reduction without removing required behavior, tests, or recovery paths. |
| P1 | Ask answer quality still lacks independent review. | The repeated citation that previously caused `music player` to return HTTP 503 is deduplicated by the answer decoder, while unknown and missing citations remain invalid. Two sequential post-fix 20-query gates passed 20/20 with unique citations resolving to supplied result IDs; p95 was 2.88 s and 2.34 s against the 8 s target. Metrics are in `artifacts/ask-latency/2026-11-22-postfix-summary.md`. Human-reviewed quality denominator remains open. | Freeze the answer set and obtain independent human review; retain current latency and citation receipts. |
| P1 | Legacy Experience page composition is not fully cut over to workspace ownership. | The audit found three composition-bearing pages among 320 pages and 10 nonempty sketches. Dashboard and conversations have explicit adoption with atomic workspace-plus-immutable-receipt writes, unchanged-retry reuse, source preservation, and retained obligation evidence. Their native output parity remains partial, so legacy pages/renderers stay in place. `agent-detail` stays rejected; `InspectPageContract` inventories its 10 state descriptions, 10 elements, 8 claims, two async region lifecycles, and test bindings without claiming execution. | Preserve every source page until native rendering, behavior, accessibility, and persistence parity is proven; then retire legacy composition owners through a reviewed deletion ledger. |
| P1 | Eight-screen visual acceptance is not operator-reviewed. | Seed v46 has fresh served 390×844 Present captures for all eight screens on build `sha256:d661c054856d97bcf0798b2ee53a74f2679248ea977cd8ffeeaef62b4f92a049`; all returned HTTP 200 with no page errors or horizontal overflow. Run-before pace/navigation align with the supplied JPEG. Plant-day’s editable Monstera leaf is now lobed; the same fixed comparison reports descriptive MAE 84.211 on v45 and 83.312 on v46. Captures and comparison are in the experience-engineering experiment under `data/served-current-v46/` and `captures/eight-screens-reference-vs-local-v46.*`. Visible differences remain, especially Wind-down, Plant-day, and Phrase-practice; MAE is not an acceptance score. No operator acceptance or authenticated BAS evidence exists. | Continue source-backed fidelity work, then obtain authenticated capture/drift evidence tied to immutable source and state identities and operator review. |
| P2 | Old browser template drafts still need the bounded bookmark reader. | `readLegacyTemplateForMigration` reads unsynced `rcl.workbench.templates.v1` drafts. Legacy component-context URLs now show an explicit import choice; the server copy is created only after the operator confirms, and the local source remains intact. No privacy-safe usage signal or closed published compatibility window exists. | Close the documented compatibility window, obtain privacy-safe counts, then remove the bridge and migration tests. |
| P2 | The production build retains a large lazy CodeEditor chunk. | UI lint and typecheck pass. Production build emits CodeEditor at 2.57 MB (667 KB gzip) and the TypeScript worker at 6.04 MB. The editor is already lazy loaded at the explicit edit boundary. | Keep the existing lazy boundary and document an owner-approved budget or reduce emitted editor payload without making draft work eager. |

## Recently closed

- **Read-only Go lint fallback (2026-11-22):** The shared scenario `lint-go`
  fallback no longer runs write-path `go fmt`. It now runs `go vet` and checks
  files with `gofumpt -l` or `gofmt -l`, leaving rewrites to `make fmt`. Verified
  the fallback with `golangci-lint` absent: a formatted fixture passed, and an
  intentionally unformatted fixture failed while its SHA-256 stayed unchanged.
  `make -n check` shows no formatter write command.
- **CLI coverage floor recovery (2026-11-22):** Added focused command-path
  tests in catalog, asset, preview, versions, and search after the second CLI
  suite wave reached 72.2%. Fresh `go test ./... -coverprofile=...` passes all
  CLI packages at 75.0% aggregate, meeting the unchanged committed floor.
- **UI sanitizer edge coverage (2026-11-22):** Added direct SVG sanitizer cases
  for CSS filtering, selector recovery, custom-property resolution, invalid
  mask references, xlink symbol expansion, byte limits, and Unicode data URLs.
  Removed an unreachable at-rule check already guaranteed by `staticCssRules`.
  The latest full `pnpm test:coverage` passes 178 files / 1,175 tests at 91.88%
  statements, 85.01% branches, and 85.32% functions; the reviewed 85% floors
  remain intact.
- **Experience importer contract coverage (2026-11-22):** Added renderer-input
  parity tests for dashboard and conversations and an explicit unsupported
  agent-detail check. These prove identical `PreviewService` requests, not final
  HTML/render parity; legacy storage and rendering remain until true parity.
- **Cohesion debt reduction (2026-11-22):** Split the 1,375-line component
  indexer into orchestration and four cohesive readers/projections; the core is
  384 lines, with no extracted production file over 464. Simplified the two
  reported Connect error mappers to cyclomatic scores 5 and 4. Focused component
  and search tests pass. A fresh scenario tidiness scan skipped at graph
  admission, so the prior whole-scenario long-file count remains unverified.
- **Receipt-backed experience adoption (2026-11-22):** The explicit UI action now
  writes the workspace and immutable recovery receipt atomically, preserves the
  source page, reuses unchanged retries, and rejects changed source. The result
  exposes parity gaps and supports receipt download. This is not native render
  parity; dashboard/conversations legacy owners remain active, and
  `agent-detail` stays excluded behind a deterministic obligation inventory.
- **Served eight-screen v43 review (2026-11-22):** After a supported healthy
  restart, captured all eight editable workspaces at 390×844 and generated a
  paired descriptive comparison. Plant-day facts/thumbnails, Plant-night
  hierarchy/selection, and Phrase-practice navigation now use ordinary workspace
  nodes; Wind-down remains unchanged. All captured pages reported no browser
  errors. The operator has not approved the visuals and BAS capture remains
  unauthenticated.
- **Served v44 content correction (2026-11-22):** Corrected Run-before's pace
  from 5:32/km to 5:52/km because the screenshot clearly shows 5:52/km and its
  39:46 / 6.8 km values agree. The v44 seed is served and verified in Present
  mode at 390×844. Start was captured at 1440×1000, 390×844 and 320×720; all
  five search-kind filters appeared and document width matched viewport width.
  Evidence is in the experiment's `data/served-current-v44/`. This single-screen
  capture is not a fresh all-eight comparison or operator approval.
- **Ask citation reliability (2026-11-22):** The answer decoder now removes
  repeated citation IDs in first-seen order before strict result validation.
  Unknown IDs and malformed/missing citation fields remain rejected. Focused
  decoder tests and the full API Go suite pass; two sequential live 20-query
  gates pass 20/20 with valid, unique citations, p95 2.88 s and 2.34 s against
  the 8 s target. See `artifacts/ask-latency/2026-11-22-postfix-summary.md`.
  Independent human answer-quality review remains open above.
- **BAS route contract fixtures and Go lint (2026-11-22):** Updated the
  component-test handler fixtures and inspection assertion to use the canonical
  `/api/v1/ux-capture/scenario` endpoint. The complete API Go suite and CLI Go
  suite pass, and `make lint-go` passes after gofumpt formatting in the
  experience-import and version-ledger sources. This closes the Go route-fixture
  mismatch only. When shared scenario discovery has no URL, the BAS executor
  now falls back to the validated, lifecycle-allocated `UI_PORT` for this local
  scenario; an explicit `RCL_UI_BASE_URL` or discovery URL still takes
  precedence. The focused executor/handler tests and full Go suites pass.
  Authenticated browser capture and full Test Genie acceptance remain open
  above.
- **Workbench component review budgets (2026-11-22):** Extracted cohesive
  responsibilities and brought the reviewed workbench pages/dialogs under
  the 400-line limit: `RegionCapture.tsx` 397, `HtmlImport.tsx` 381,
  `WorkspaceStudioPageContent.tsx` 389, `StudioAssist.tsx` 335,
  `TemplateRenderer.tsx` 334, and `StudioInspector.tsx` 312. The SVG sanitizer
  is now its own module; `htmlImport.ts` remains the parser/composition owner.
  Focused suites pass independently (RegionCapture 14, HTML import 9,
  StudioAssist 18, StudioInspector 5, TemplateRenderer 26, HTML sanitizer and
  import 74); UI TypeScript, targeted lint, and formatting checks pass. This
  closes the size-budget item: the UI source audit found no page component over
  400 lines, hook over 350, or CSS file over 1,500. The 464-line
  `primitives.tsx` is a multi-export primitive module, not a page component.
  This does not close the separate shared-worktree source-line gate below.
- **API coverage and self-hosting floors (2026-11-21):** API profile is 75.0%
  (28,569/38,092), meeting its 75% floor; self-hosting is 36/265 against the
  unchanged floor of 36. Rerun after later source changes before treating this
  as current acceptance evidence.
- **Storage validation (2026-11-21):** storage-manager reported zero findings
  after rooted filesystem writes and request-routed DB access. This does not
  certify other quality gates.
- **Unused UI clients and workbench component cleanup (2026-11-22):** removed
  the unconsumed dependency/version clients, their stale coverage exclusions
  and one obsolete version-service mock; extracted the placement panel and
  measurement/contrast probes. The latest stage tests pass (28 tests), as do
  lint and whitespace checks. See current-source records in
  [WORKBENCH-PROGRAM.md](../internal/WORKBENCH-PROGRAM.md).

- **UI branch and function coverage floors (2026-11-21):** recovered above the
  unchanged 85% thresholds. The latest full run reports 91.51% statements,
  85.03% branches and 85.17% functions after focused route and page-story
  coverage tests. Rerun after later source changes before treating this as
  current acceptance evidence.

## Verified removed owners

The design/template server stores and CRUD, browser template shelf writes, browser
findings/activity reconstruction, duplicate Home route, legacy design studio,
private catalog-search adapter, and SketchService RPC surface have been removed
or migrated as recorded in
[WORKBENCH-LEGACY-OWNERS-AUDIT.md](../internal/WORKBENCH-LEGACY-OWNERS-AUDIT.md).
The workspace aggregate owns new durable composition; experience pages own
intent. However, three existing scenario pages still store sketch composition
in experience JSON. Dashboard and conversations can be explicitly adopted into
workspaces with immutable recovery receipts, but native rendering parity is
incomplete; agent-detail is rejected with its obligations inventoried. Search,
Review, Activity, and Health have bounded server owners.

## Guardrails

- Do not lower reviewed coverage or adoption floors to make a gate pass.
- Keep curated workspaces immutable and fork-on-use.
- Do not add another durable composition store or a silent browser fallback.
- Treat a dated measurement as historical once code or evidence changes; rerun the owning gate before closure.
- For each deletion, preserve public behavior, migrations, recovery and evidence paths.
