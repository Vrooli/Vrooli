# Testing — React Component Library

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Catalog gate calibration and rendered evidence

### Preview evidence runner boundary

Use `pnpm run test:preview-evidence` for catalog component-story evidence. The
command delegates story execution and structured capture to the BAS-backed
component-test runner, then persists per-story artifact references and visual
gate evidence through `CatalogService.CaptureEvidence`. The isolated story
route remains the authoritative capture boundary; the RCL UI does not launch a
second browser runner.

Browser Automation Studio remains the preferred producer for ordinary
scenario-page workflows and lifecycle evidence. It is not a replacement for
the RCL story runner until it can select exact story IDs inside the preview
iframe, preserve frame/harness metadata, and return one authoritative artifact
per story. Do not add one-off browser scripts for individual components; extend
the shared runner and its machine-readable manifest instead.

The run manifest is the authoritative capture summary. Each failure includes
`stage`, `category`, `retryable`, and `message`; categories distinguish
environment, resolver/contract, product rendering, expectation, and capture
infrastructure failures. A story timeout is bounded independently from the
asset timeout with `RCL_PREVIEW_STORY_TIMEOUT_MS`. Earlier successful artifacts
remain in the output directory and manifest when a later story fails.

The default review request is the `core` set. Stories without evidence metadata
belong to that set automatically, and the runner infers their state from the
story ID. Add per-story evidence metadata only when the story needs an explicit
review-set or state override.

The runner supports controlled diagnostics for its own validation with
`RCL_PREVIEW_FORCE_FAILURE=blank-root|404|timeout|expectation`; these hooks are
test-only and must never be used for acceptance captures. Use a small
`RCL_PREVIEW_STORY_TIMEOUT_MS` for the timeout case. A forced failure must
produce a non-zero run and a manifest failure row—it must not become a blank
successful screenshot.

Catalog verdicts are valid only when their runner is backed by the oracle it
claims to measure. `unmeasured` is a first-class result: it is displayed beside
pass and fail, remains in every score denominator, and never becomes pass by
default. Blocking gates must own a planted-error fixture under
`catalog/calibration/<gate>/fixture.json`; run a gate's calibration with:

```bash
react-component-library catalog gates composition --calibration-only --json
```

The composition gate reads the build-stamped rendered tree, not imports or
hand-authored markers. Its production-ready floor is currently `0.8`, derived
from the measured cockpit corpus median of `1.0`; data-specific raw nodes are
allowed only as counted `data-bespoke` escapes with a non-empty reason. The
coverage report publishes `composition_blocked_asset_count` so a failed
composition gate is reflected in maturity rather than hidden in row details.

The component Tests surface keeps producer states separate: Integrity,
Behavior, Experience, and Cost are distinct views; clean Integrity collapses;
blocked and unmeasured are not failures; and failed experience claims use
`ClaimMeasurement` subjects for the overlay and the six evidence kinds
(screenshot, accessibility tree, computed style, layout box, console, and
performance). A claim without capture must render an explicit unmeasured state.

## API testing

### CRUD reference — `notes` end-to-end

The `notes` domain is the canonical CRUD reference. New scenarios add
their first non-trivial mutation by copying its layering one file at a
time. The pattern from wire to render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/react-component-library/v1/notes/notes.proto` | `Note`, `service NotesService`, `ListNotesResponse`, `CreateNoteRequest`, `CreateNoteResponse`, `GetNoteRequest`, `GetNoteResponse` |
| REST metadata contract | `packages/proto/schemas/react-component-library/v1/notes/attachments.proto` | `Attachment` and `UploadAttachmentResponse` for the multipart upload exception |
| Connect error mapping | `internal/notes/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| Domain types | `internal/notes/types.go::{Note, Attachment, CreateInput, ErrInvalidNote, ErrNoteNotFound}` | Domain-pure (no proto imports); typed sentinels translate into Connect errors at the handler edge |
| Repository interface | `internal/notes/repository.go::Repository` | Persistence seam — `Create` / `Get` / `List` |
| Repository impl | `internal/notes/sqlite.go::NewSQLiteRepository` | sqlite-backed `Repository`; production wires it once in `main.go` |
| Schema | `internal/notes/schema.{sql,go}::Schema()` | Domain-owned table DDL embedded via `go:embed`; collected by `internal/modules/registry.go::AllSchemas()` and applied at boot via `apidb.EnsureSchemas` |
| Repository test | `internal/notes/sqlite_test.go` | Real handle via `db.NewSQLite(t)` + `apidb.EnsureSchemas(ctx, d, ...providers...)` over system + notes (the canonical compose pattern) |
| Service | `internal/notes/service.go::Service` (+ `NewService`) | Application layer: validation (`title` required after whitespace trim), default substitution (`defaultListLimit = 100` when caller passes 0). Handler depends on this, not the repository. |
| Service test | `internal/notes/service_test.go` | Substitutes `mocks.FakeRepository` (from co-located `internal/notes/mocks/`); pins the validation, default-substitution, and error-propagation contracts |
| Connect handler test | `handlers/notes/connect_handler_test.go` | Substitutes `mocks.FakeService` and exercises the generated Connect client/handler path |
| Multipart handler test | `handlers/notes/attachments_handler_test.go` | Uses `blobstore.MemoryBlobStore` plus test metadata repositories to exercise file-upload success and error paths |
| Mocks | `internal/notes/mocks/{repository,service}.go::{FakeRepository,FakeService}` | Co-located with the domain (Pass-3 pattern) — `FakeRepository` carries state for service tests; `FakeService` records inputs for handler tests. Both use atomic call counters + per-method error knobs. Deleting `internal/notes/` takes them along. |
| UI client | `ui/src/api/notes.ts` | `notesClient = createClient(NotesService, transport)` plus `uploadAttachment` for multipart metadata |
| UI tests | `ui/src/api/notes.test.ts` + component tests | Mock generated client methods and `uploadAttachment`; REST helper tests stub `global.fetch` |
| CLI client | `cli/domains/notes/{register,handlers,attach_handler}.go` | `Register(core)` returns a `cliapp.SubcommandGroup`; handlers use generated Connect clients or `cliapp.UploadFile` and render via cli-core reports |
| CLI test | `cli/domains/notes/handlers_test.go` | Spins a real `httptest.Server` via `testutil.NewAPIServer`, captures stdout via `testutil.CaptureStdout` |

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work**: load the `seam-discovery-and-enforcement` and
  `unit-testing-architecture-steer` skills through prompt-manager before substantial test changes.
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
