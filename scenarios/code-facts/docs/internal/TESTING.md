# Testing — Code Facts

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
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## API testing

### CRUD reference — `notes` end-to-end

The `notes` domain is the canonical CRUD reference. New scenarios add
their first non-trivial mutation by copying its layering one file at a
time. The pattern from wire to render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/code-facts/v1/notes/notes.proto` | `Note`, `service NotesService`, `ListNotesResponse`, `CreateNoteRequest`, `CreateNoteResponse`, `GetNoteRequest`, `GetNoteResponse` |
| REST metadata contract | `packages/proto/schemas/code-facts/v1/notes/attachments.proto` | `Attachment` and `UploadAttachmentResponse` for the multipart upload exception |
| Connect error mapping | `internal/notes/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| REST error envelope | `packages/proto/schemas/code-facts/v1/shared/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
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
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
  prompt-manager skill read seam-discovery-and-enforcement
  prompt-manager skill read test
  prompt-manager skill read unit-testing-architecture-steer
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
