# Workbench legacy-owner retirement audit

Audit date: 2026-11-21

The component-context and unbound experience adoption routes are implemented;
they are not outstanding retirement blockers. `App.tsx` routes component
bookmarks through `ComponentContextRoute`, sends bound experience pages to the
workspace studio, and presents unbound pages with `AdoptExperiencePage`.
Focused UI tests cover component-version preservation, stale-route suppression,
unbound page adoption, and concurrent adoption recovery. The only intentional
legacy composition compatibility code left is the read-only browser-template
bookmark bridge listed below.

This records the consumers and compatibility evidence checked before removing
legacy design, template, sketch, or browser-local composition owners. The
workbench target and ledger remain in [WORKBENCH-PROGRAM.md](WORKBENCH-PROGRAM.md).

## Retirement evidence

| Owner | Current consumers and evidence | Safe disposition now |
|---|---|---|
| Browser design cache (`rcl.designs.v1`) | `/design` now mounts the workspace index/studio; component-context and experience routes perform explicit workspace adoption. Workspace startup imports `studio_designs` rows and their `studio_design_assets` uploads through `internal/workspaces.Migrate`. | Removed the browser-local fallback; server-owned workspace persistence is the only mounted composition path. `DesignStudioPage`, `useStudio`, and the orphaned legacy page story are deleted. |
| `internal/designs`, `/api/designs`, `studio_designs` | `internal/workspaces.Migrate` imports source rows and binary uploads, records receipts, validates unique experience bindings, then drops all three legacy source tables in the same transaction. The workspace tests prove preservation on import failure and retirement after success. | The CRUD module, repository, schema owner, and API routes are deleted. New databases do not create legacy tables; old databases retain source data unless the full import transaction succeeds. Workspace type projections remain in `ui/src/workbench/design-types.ts`, without transport code. |
| `internal/templates`, `/api/templates`, `studio_templates` | The workspace migration imports custom template rows at stable bookmark-compatible IDs and drops the source table in the same transaction as the other legacy stores. Failed imports roll back both workspace writes and table retirement. | The API module, schema, repository, and all routes are deleted. Old server-side rows become curated workspace aggregates; no ongoing template store or HTTP adapter remains. |
| `internal/adoptions` and AdoptionsService | The picker, recommendations, registry reconciliation, provenance discovery, and explicit discovery confirmation share scenario identity and were the five RPCs above the service ceiling. UI calls continue through the generated contract; the CLI manifest keeps them intentionally omitted as internal/UI workflows. | Those five methods now belong to `AdoptionIntakeService`; `AdoptionsService` retains the 15 registry, compatibility, and lifecycle methods. Both services mount under the same scenario module and global auth/router treatment. The transfer preserves dependency, component, library, content, file, and repository ownership. `ConfirmDiscovery` preserves the existing write/error order and makes no retry-idempotency claim. |
| `internal/sketch` composition and SketchService | The CLI retains its command names and routes. Thirteen page-sketch operations and the mounted `ListDesignPages` UI inventory moved to `ExperienceService`; six capture lifecycle operations moved to `WorkspaceCaptureService`; seven acceptance and critique operations moved to `ReviewService`; the remaining 11 assist and candidate RPCs moved to `WorkspaceAssistService` and `WorkspaceCandidateService`. | The empty `SketchService` contract is retired. The sketch package remains as a shared wire-message and domain package. Each bounded service has a generated contract and endpoint parity registration; capture producer and candidate semantics remain unchanged. Review keeps narrow ports into Sketch acceptance evaluation/storage and critique/capture evidence owners. |
| Composition still embedded in scenario experience pages | `api/internal/sketch.Document` includes template, render ports/bindings/fixtures, placements, unplaced records, notes, and regions. `Store.Read/Save` decodes and writes `sketch` inside the authored `experience/pages/<page>.json`, with revision history under scenario data. `ExperienceService` exposes `GetSketch`, `PutSketch`, `ImportPage`, `Place`, `Placeholder`, `AddNote`, `Unplace`, `SetTemplate`, `BuildBrief`, `GetHistory`, and `RecoverSketch`; candidate render/placement handlers also read this Store. CLI `sketch` commands call those ExperienceService procedures. | **Not safe to delete or bulk-strip yet.** A repository inventory finds 320 experience page files and 10 nonempty sketch objects; three are composition-bearing: `switchboard/dashboard` and `switchboard/conversations` have template/render contracts (10 placements on conversations), while `switchboard/agent-detail` has one placement but no template/render contract. `api/internal/experienceimport.InventoryCompositions` reports stable IDs, hashes, region/placement counts, and converter support. Dashboard and conversations now have deterministic converters and explicit adoption that atomically persists a workspace with an immutable recovery receipt while preserving the source page. `VerifyResult` checks source hash/revision, target hash/binding, state adapters, and retained contract evidence. Agent-detail remains unsupported. Receipt-backed adoption proves recovery and obligation retention, not native rendering or execution of those obligations. The native TemplateNode canvas still lacks render parity for legacy `CollectionPage` exports, bindings/fixtures, authored slot paths, and interactive actions. No rendered-output comparison or source retirement exists. |
| Browser template cache (`rcl.workbench.templates.v1`) | Old custom drafts that never reached the server cannot be imported by the server migration. The component-context bookmark route has a small read-only parser for this cache; errors become an explicit “could not migrate” route message. | All shelf CRUD, server sync, cross-tab queues, pending-write state, and silent local fallback code are deleted. Keep the read-only bookmark bridge until the published URL compatibility window has elapsed and privacy-safe usage evidence has been collected; no telemetry exists for the browser-local cache today. |
| Other browser state | `workbench/preferences.ts` stores device preferences. `ComponentPage.tsx` stores in-context slot positions under `rcl.workbench.fixtures.*`. The selected-design key and `useStudio` hook were deleted with the legacy editor. | Preferences and slot positions are device/UI state, not durable composition. The obsolete selected-design state has been removed. |

## Verification

- Focused API migration/module/handler tests — passed after transactional source-table retirement; endpoint manifest regenerated with no design or template REST routes.
- Focused UI route and browser-cache migration tests — passed; `ComponentContextRoute.test.tsx` covers durable component-context adoption and stale lookup suppression, and `AdoptExperiencePage.test.tsx` covers bound workspace creation and simultaneous-adoption recovery. The old template shelf sync module and its dead API writes are deleted.
- Experience and WorkspaceCapture service migrations: owning proto generation/manifests and `verify-committed-gen`; endpoint generation; focused API handler and module parity tests passed. The Experience slice also passed CLI and Design inventory UI checks.
- WorkspaceAssist and WorkspaceCandidate extraction: generated protobuf and
  Connect outputs, regenerated owning manifests/endpoints, and passed the
  committed-generation verifier. Focused API route/module parity tests and the
  full CLI suite with primitive evidence regeneration passed. The old
  SketchService route returns 404; all 11 calls appear only under their new
  bounded services.
- AdoptionIntake extraction: generated protobuf and Connect outputs, regenerated
  manifests/endpoints, and passed the committed-generation verifier. API route
  tests confirm the new service owns its methods with no old AdoptionsService
  paths. CLI tests with primitive evidence regeneration and focused UI caller
  checks preserve existing command and picker behavior.
- Integrated 2026-11-21 verification: full uncached API tests, CLI tests with
  primitive evidence regeneration, proto committed-generation check, all 140 UI
  test files / 833 tests, UI lint, type-check, and production build pass. The
  served source-import journey and 100-query hybrid latency gate pass. BAS
  capture is still blocked by its authenticated-account requirement, so live
  visual-drift receipts remain unverified.

## Sketch wire-message consumer map and migration boundary

The Sketch proto retains the shared wire messages used by page intent,
candidate, capture, and review contracts; its `SketchService` has been removed.
The 13 Experience/page-sketch compatibility RPCs live in `ExperienceService`,
the six capture lifecycle RPCs live in `WorkspaceCaptureService`, seven
acceptance/review-fact RPCs live in `ReviewService`, four assist/inference RPCs
live in `WorkspaceAssistService`, and seven candidate/render RPCs live in
`WorkspaceCandidateService`. The mounted UI calls in
`ui/src/workbench/data.ts` use `ExperienceService.ListDesignPages` to populate the Design
destination's scenario/page inventory. Repository-wide UI search found no
imports of `ui/src/api/sketch.ts` exports (`sketchClient`, `importSketchPage`,
or the reference-asset upload/generation helpers); that internal API module has
been removed. The server reference-asset routes remain mounted because this
dead-client evidence does not establish endpoint retirement or remove the
published compatibility requirement. The `/sketch/:scenario/:page` route is a
compatibility redirect into Design/workspace adoption, not a direct
SketchService client.

The CLI retains prior command names, routes, and flags. Page-sketch compatibility
commands use `ExperienceService`; acceptance commands live in the `review`
domain and use `ReviewService`; proposal/inference commands use
`WorkspaceAssistService`; candidate rendering uses `WorkspaceCandidateService`.

For the next extraction, classify the server RPCs by their actual current owner:

| Current owner boundary | RPCs | Remaining consumer / disposition |
|---|---|---|
| Experience intent and page-sketch compatibility | `ListDesignPages`, `GetSketch`, `PutSketch`, `VerifySketch`, `ImportPage`, `Place`, `Placeholder`, `AddNote`, `Unplace`, `SetTemplate`, `BuildBrief`, `GetHistory`, `RecoverSketch` | Migrated to `ExperienceService`; mounted UI inventory and all existing CLI command routes now call the new owner. Shared sketch wire messages remain in the sketch schema for compatibility. |
| Workspace assist and inference | `ProposeSketch`, `InferSketch`, `GetSketchInference`, `RefineCandidate` | Migrated to `WorkspaceAssistService`; CLI proposal/inference commands retain their exact command behavior and use the new client. The handler surface is restricted to these four RPCs. |
| Workspace candidate authoring/rendering | `ListCandidates`, `PlaceCandidateAsset`, `MapCandidateRegions`, `SaveCandidate`, `GetCandidate`, `RenderCandidate`, `RenderSketch` | Migrated to `WorkspaceCandidateService`; immutable candidate storage, exact asset placement, revision checks, and rendering semantics remain intact. `RenderCandidate` keeps its CLI command and flags. |
| Workspace capture lifecycle | `GetCaptureScreenshot`, `RetryCapture`, `CancelCapture`, `AttachCapture`, `CaptureCandidate`, `GetCapture` | Migrated to the bounded `WorkspaceCaptureService`; producer identity, evidence verification, screenshot resolution, and retry semantics remain with `designcapture`. |
| Review facts / acceptance | `AcceptCandidate`, `GetAcceptance`, `CheckCandidateAcceptance`, `ListCritiques`, `GetCritiqueRubric`, `RecordCritique`, `GetCritique` | Migrated to `ReviewService`. Acceptance keeps the sketch store's prepared-intent idempotency protocol and evaluates current candidate/render evidence through a narrow port. Critiques remain owned by `designcritique`, with capture evidence from `designcapture`; explicit unavailable evidence remains visible. |

The old SketchService's 11 RPCs are now split across `WorkspaceAssistService`
(four) and `WorkspaceCandidateService` (seven). Both are within the 16-RPC
target and have separate endpoint parity entries. `ComponentsService` has 12
RPCs; `ComponentAuthoringService` owns its 10-RPC write/edit surface.
`ExperienceService` (13), `WorkspaceCaptureService` (six), and `ReviewService`
(12 including the existing findings/disposition surface) are also within target.
The catalog-only search procedures have been retired; `CatalogService` now has
13 RPCs.

## Components authoring boundary migration

The original 22-method ComponentsService surface was checked against its
handler dependencies, UI transport, CLI client, endpoint descriptors, and the
existing `VersionsService` contract. No existing service had a compatible
contract for the authoring group, so it now has its own generated service:

| Responsibility | RPCs | Evidence and boundary constraint |
|---|---|---|
| Registry projection and indexing | `ListComponents`, `GetComponent`, `GetComponentByLibraryId`, `IndexComponents` | Registry projections and the indexer share the component repository and catalog projection. UI and CLI use the generated Components client. |
| Component authoring service | `InitializeComponent`, `IngestComponent`, `BeginComponentVersion`, `CheckComponentVersion`, `PublishComponentVersion`, `CreateComponentVersion`, `UpdateComponentManifest`, `RetireComponent`, `GetComponentContent`, `UpdateComponentContent` | Migrated together under a separate handler with narrow `AuthoringDeps`; it receives the same component service, repository, source root, and version ledger as API startup. CLI names and UI behavior remain stable. |
| Source/content read projection | `GetComponentVersionContent` | Reads a released manifest version artifact; distinct from active draft editing and its save listener. |
| Component-local history and import resolution | `ListComponentVersions`, `ResolveLibraryImport` | `VersionsService` owns recorded edit history and diffs; these methods instead expose manifest release folders and dependency resolution for imports. Similar names do not mean interchangeable data or semantics. |
| Stories and preview frame persistence | `ListComponentStories`, `ListPreviewFrames`, `PersistPreviewFrame` | Story discovery and frame metadata are tied to component release folders; frame persistence uses injected Preview composition functionality and creates authoring state. The existing PreviewService bundles/renders compositions and has no compatible messages for these operations. |
| Design-style catalog and fit validation | `ListDesignStyles`, `ValidateStyleFit` | Both are implemented by the components domain; style-fit also validates against scenario JSON and its component version. `ThemesService` exposes built-in runtime themes and is not this catalog or validation owner. |

The creation/release and content-authoring group now lives in the generated
`ComponentAuthoringService`: `InitializeComponent`,
`IngestComponent`, `BeginComponentVersion`, `CheckComponentVersion`,
`PublishComponentVersion`, `CreateComponentVersion`,
`UpdateComponentManifest`, `RetireComponent`, `GetComponentContent`, and
`UpdateComponentContent`. The authoring handler has a narrow `AuthoringDeps`
port set populated from the same service, repository, source root, and version
ledger used by API startup. The canonical mutation lock, ingest
observer/provenance, content store, and content-save listener remain at their
existing owners. ComponentsService is now 12 RPCs; ComponentAuthoringService
owns 10. The existing Components request/response message package remains
unchanged, while the procedures and CLI service bindings identify the new
owner. CLI command names and UI behaviors are unchanged.

Handler registration is in `api/handlers/components/module.go`; the registry
and authoring services have distinct generated handlers and endpoint
descriptors use each service's procedure constants. The generated CLI bindings move without changing
commands, and UI authoring helpers use the generated authoring client with the
existing longer authoring deadline. The endpoint manifest lists every method
once under its generated procedure. Focused API/CLI tests and UI type checks
passed after regeneration.

## Remaining cleanup debt

- `AdoptionsService` is at 15 RPCs and `AdoptionIntakeService` at five; both
  satisfy the 16-RPC boundary target. `CatalogService` is now at 13 after its
  duplicate catalog-only search procedures were retired in favor of the unified
  SearchService.
- Coverage, gate, health, readiness and capture remain live CatalogService
  responsibilities. Any future split must move a cohesive owner with its
  callers rather than delete a method based on its size.
- The private `internal/catalogsearch` adapter has been deleted. Experience-page
  import and proposal candidate search now use the unified
  `internal/search.Service` projection through its `CatalogSearch` port. Exact
  declared-ID lookup remains identity-only; normal candidate search preserves
  availability, page-template exclusion, and proposal metadata. The focused
  `TestSearchCatalogReturnsPageTemplateMetadataFromUnifiedProjection` test
  verifies page-template metadata and honest `declared` availability when live
  availability is not configured.
- `readLegacyTemplate` is the bounded one-release bookmark bridge. No usage
  telemetry exists for this browser-local path; the saved drafts never reached
  the server, so removal now would strand them. Remove it only after the published
  compatibility window has elapsed and a privacy-safe old-bookmark usage signal
  has been collected. Until then, keep the read-only bridge and its explicit
  migration failure message.
- Calibration fixtures now create `reconcile_evidence` and its viewport table
  through the single test-fixture schema helper at
  `api/internal/gates/calibration_schema.go`; the fixture constructors only
  seed rows. This does not change production schema ownership.
- The prior tracked source diff added 7,346 and removed 20,310 lines; new source
  files added 17,439 lines, for a net increase of 4,475 source lines versus
  `HEAD`. That measurement is historical. The worktree must be recounted after
  the active visual refinement before making a net-negative claim. Counts
  include tests and generated route artifacts and do not justify deleting active
  code to force a lower total.
