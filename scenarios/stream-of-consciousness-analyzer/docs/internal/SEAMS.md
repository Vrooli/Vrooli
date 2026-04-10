# Integration Seams

Testability boundaries and responsibility zones for the Stream of Consciousness Analyzer.

## Service Interfaces (Primary Seam)

Handlers accept **interfaces**, not concrete service types. This is the key seam that enables unit testing without a database.

| Interface | Implemented By | Test Mock | Purpose |
|-----------|---------------|-----------|---------|
| `SchemeStore` | `SchemeService` | `mockSchemes` | Scheme CRUD operations |
| `InformationStore` | `InformationService` | `mockInfo` | Canvas item CRUD |
| `ThoughtStore` | `ThoughtService` | `mockThoughts` | Thought + edge CRUD |
| `ExportStore` | `ExportService` | `mockExport` | Graph export |
| `SuggestionProvider` | `SuggestionService` | `mockSuggestions` | LLM provider + suggestions |

**File**: `api/interfaces.go` defines all interfaces.
**Pattern**: Handler factories like `handleListSchemes(svc SchemeStore)` accept the interface. Production code passes `*SchemeService` (which satisfies `SchemeStore`); tests pass `*mockSchemes`.

### Compile-Time Interface Checks

`mocks_test.go` includes `var _ SchemeStore = (*SchemeService)(nil)` assertions to catch interface drift at compile time.

## Service Boundaries

### API ↔ Database

| Seam | Location | Test Strategy |
|------|----------|---------------|
| SchemeService | `api/scheme_service.go` | Satisfies `SchemeStore`; testable via test DB or mock |
| InformationService | `api/information_service.go` | Satisfies `InformationStore` |
| ThoughtService | `api/thought_service.go` | Satisfies `ThoughtStore` |
| ExportService | `api/export_service.go` | Satisfies `ExportStore` |
| SuggestionService | `api/suggestion_service.go` | Satisfies `SuggestionProvider`; LLM calls are HTTP-based (mockable) |
| Schema migration | `api/schema.go` | Tested implicitly by service tests that create tables |

**Pattern**: Each service takes `*sql.DB` directly. The interface boundary sits between handlers and services, allowing handler tests to bypass the database entirely.

### Environment Reader Seam

| Seam | Location | Test Strategy |
|------|----------|---------------|
| `EnvReader` function type | `api/suggestion_service.go` | Injectable env reader for controlling provider configuration in tests |

**Pattern**: `NewSuggestionService(db)` uses `os.Getenv` (production). `NewSuggestionServiceWithEnv(db, env)` accepts a custom `EnvReader` function, allowing tests to control `OLLAMA_URL` and `OPENROUTER_API_KEY` without modifying process-level env vars. Tests in `suggestion_service_test.go` exercise all provider fallback paths through this seam.

### API ↔ LLM Providers

| Seam | Location | Test Strategy |
|------|----------|---------------|
| Ollama client | `api/suggestion_service.go` | HTTP calls to `OLLAMA_URL`; testable via httptest server |
| OpenRouter fallback | `api/suggestion_service.go` | HTTP calls to OpenRouter API; testable via httptest server |

**Key design**: Provider fallback is handled inside `SuggestionService.GetActiveProvider()`. The provider list is checked via `/api/v1/providers` endpoint.

### UI ↔ API

| Seam | Location | Test Strategy |
|------|----------|---------------|
| API client | `ui/src/lib/api.ts` | `apiFetch` helper wraps `fetch()`; mockable in tests |
| Component data loading | Each component | React components call `apiFetch` in `useEffect`; testable via MSW or fetch mocks |

**Pattern**: UI components use a centralized `apiFetch()` function that prepends the API base URL. This is the single point to mock for UI testing.

### CLI ↔ API

| Seam | Location | Test Strategy |
|------|----------|---------------|
| CLI HTTP client | `cli/` | CLI calls API endpoints; testable via httptest or running API |

## Health Check Boundaries

- `/health` — Used by lifecycle system for process health monitoring. Includes database ping as critical check.
- `/api/v1/health` — Used by UI iframe bridge for API connectivity status.

Both share the same handler via `api-core/health` package.

## Cross-Scenario Integration Points

| Integration | Direction | Endpoint |
|-------------|-----------|----------|
| Export to other scenarios | Outbound | `GET /api/v1/schemes/{id}/export` |
| PostgreSQL resource | Inbound | Connection via `POSTGRES_*` env vars |
| Ollama resource | Inbound | HTTP calls to `OLLAMA_URL` |

## Testing Recommendations

1. **Handler unit tests**: Use mock implementations of service interfaces (current approach, 70+ tests). No database needed.
2. **Service integration tests**: Use testcontainers (postgres:15-alpine) for real database testing.
3. **UI tests**: Mock `apiFetch` or use MSW for API simulation.
4. **Integration tests**: Start API with test database, hit endpoints via HTTP client.

## Change Axes

Primary ways this scenario is likely to evolve, ordered by likelihood:

| Axis | Current Cost | Where Changes Land | Notes |
|------|-------------|-------------------|-------|
| **New content types** (voice, URL, image) | Low | `InformationService.Create` type default, UI `TextCapture` component, DB `information.type` column | Type is a plain string; no enum or registry. New types need only a new input component and optional rendering logic. |
| **LLM integration** (real suggestions) | Medium | `SuggestionService.GenerateSuggestions` (stub), new HTTP client code | Provider selection is already abstracted. Main work is prompt construction + response parsing, localized to suggestion_service.go. |
| **New LLM providers** | Low | `NewSuggestionServiceWithEnv` provider list | Add a new `LLMProvider` entry. `GetActiveProvider` handles any number of primary/fallback providers without modification. |
| **Export format evolution** | Low | `ExportData` struct, `ExportFormatVersion` constant | Consumers key on `export_format` string. Additive fields are safe; breaking changes need a version bump. |
| **UI view modes** (beyond canvas + graph) | Medium | `App.tsx` view switcher, new component file | View mode is a string state; adding a new mode requires a component and a switch case. |
| **Cross-scheme features** | Low | `ThoughtService.List` filter, `Thought.SchemeID` nullable | Already designed for cross-scheme thoughts. Edges work across schemes today. |

**Stable cores** (unlikely to change): error classification, handler factory pattern, service interfaces, DB schema shape.

**Volatile edges** (likely to change): LLM prompt/response handling, content type rendering, provider list, UI layout.

## Decision Points

Key places where the system chooses between alternatives:

| Decision | Location | Inputs | Outcomes | Tested? |
|----------|----------|--------|----------|---------|
| **Error classification** | `errors.go:classifyAndWriteError` | Go error value | 404 (ErrNoRows), 409 (unique violation), 400 (FK violation), 500 (default) | Yes — 5 tests in errors_test.go |
| **Provider selection** | `suggestion_service.go:GetActiveProvider` | Provider list with Active/Fallback flags | Primary preferred → fallback if no primary → error if none | Yes — 4 tests in suggestion_service_test.go |
| **Content type defaulting** | `information_service.go:Create` | `input.Type` string | Empty → "text"; non-empty → passthrough | Yes — implicit in handler tests |
| **Thought scope filter** | `thought_service.go:List` | `schemeID` query param | Empty → all thoughts; non-empty → filtered by scheme | Yes — 2 tests in thought_handlers_test.go |
| **Edge self-loop guard** | `handlers.go:handleCreateEdge` | `sourceID`, `input.TargetID` | Same → 400 validation error; different → proceed | Yes — TestHandleCreateEdge_SelfLoop |
| **Link-mode state machine** | `ui/src/components/GraphView.tsx` | `linkSource` state | null → normal mode; LINK_MODE_WAITING → select source; thought-id → next click creates edge | UI-only, no backend test |
| **Retryable flag** | `errors.go:writeAPIError` | `ErrorCategory` | dependency → retryable=true; all others → retryable=false | Yes — TestWriteAPIError_DependencyRetryable |
| **Zoom bounds** | `ui/src/components/CanvasView.tsx:handleWheel` | scroll delta, current zoom | Clamped to [CANVAS_ZOOM_MIN, CANVAS_ZOOM_MAX] | Config constants tested |

## Remaining Weak Points

1. **No service-level testcontainers**: Services still need real DB for integration tests.
2. **SuggestionService HTTP calls**: `GenerateSuggestions` is a placeholder; real LLM calls would need httptest mocks. The `EnvReader` seam covers configuration but not the HTTP call itself.
3. **ExportService edge collection**: `collectSchemeEdges` queries per-thought and deduplicates. Could be replaced with a single SQL query using `WHERE source_id = ANY($1) OR target_id = ANY($1)` for better performance at scale.
4. **UI `apiFetch` seam**: Currently functional but not exercised via MSW or fetch mocks in tests.
