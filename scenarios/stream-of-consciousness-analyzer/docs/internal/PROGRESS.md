# Progress

## v1.0 - Initial Release

| Date       | Author            | Status Snapshot | Notes |
|------------|-------------------|-----------------|-------|
| 2026-03-20 | Generator Agent | Initialization complete | Scenario scaffold, PRD, and requirements seeded |
| 2026-03-20 | Claude Opus 4.6 | All 6 operational targets implemented | API: 8 Go files (schema, models, 3 services, handlers, export, suggestions), 14 REST endpoints. UI: 4 custom components (SchemeList, TextCapture, CanvasView, GraphView), replaced template. Tests: 20 Go unit tests + 4 UI tests, all passing. Requirements: 8/8 implemented with test links. Score: 2→30. |
| 2026-03-20 | Claude Opus 4.6 | Architecture & standards hardening | Decomposed 4 single-req modules into 2 reqs each (8→12 total). Added 12 new Go unit tests (33 total API + 2 CLI). Removed manual validations (-2pt penalty gone). Added api/internal/<domain>/schema.sql, ESLint config with safety rules, protective tsconfig comments, INTEROP-CRITICAL annotations. Fixed broken docs link, lighthouse.json schema. All 10 test-genie phases pass. Score: 30→39 (penalty-free). |
| 2026-03-20 | Claude Opus 4.6 | Documentation health pass | Restructured docs/ to standard layout (PROBLEMS/PROGRESS/RESEARCH → docs/internal/). Created: manifest.json, QUICKSTART.md, concepts/ARCHITECTURE.md, reference/api-endpoints.md, internal/SEAMS.md. Added DOC: comments to all 9 API files and App.tsx. Added [CODE:] references in ARCHITECTURE.md and SEAMS.md. Updated README.md with documentation links. Score: 54→54 (docs health: 0.9→1.0; completeness score unchanged as it measures tests/features not docs). |

| 2026-03-20 | Claude Opus 4.6 | Seam discovery & unit test architecture | Introduced service interfaces (`SchemeStore`, `InformationStore`, `ThoughtStore`, `ExportStore`, `SuggestionProvider`) in `api/interfaces.go`. Updated all 20 handler factories to accept interfaces instead of concrete types. Updated `Server` struct to hold interfaces. Created 5 mock implementations with builder pattern in `handlers_test.go`. Added 34 new behavioral handler tests (create/read/update/delete + error paths). Total: 67 test functions (70 with subtests). Added `decodeJSON[T]` and `assertStatus` test helpers. Added compile-time interface compliance checks. Updated SEAMS.md with new interface seam documentation. Score: 54→54 (numerical score unchanged; scoring tool counts test files not functions, and can't parse Go [REQ:ID] annotations — see P6 in PROBLEMS.md. Actual test functions: 33→67, all passing. Architecture significantly improved via interface seams). |
| 2026-03-20 | Claude Opus 4.6 | Phase 3: Seam enforcement + test architecture reorg | Split monolithic `handlers_test.go` (1585 lines) into 8 focused test files: `mocks_test.go`, `testhelpers_test.go`, `model_test.go`, `thought_handlers_test.go`, `export_handlers_test.go`, `suggestion_handlers_test.go`, `suggestion_service_test.go`, `handlers_test.go`. Added `EnvReader` seam to `SuggestionService` for injectable env var access (`NewSuggestionServiceWithEnv`). Added 7 new tests: 2 missing requirement refs (`TestThoughtNilSchemeForCrossScheme`, `TestHandleExportScheme_Init`) + 5 env seam tests. Total: 73 test functions (76 with subtests), all passing. Test files: 3→10. Created `UNIT_TEST_ARCHITECTURE.md`. Updated SEAMS.md with env reader documentation. |
| 2026-03-20 | Claude Opus 4.6 | Phase 5: Control Surface & Tunable Levers + Utils Unification | **API**: Created `config.go` with named constants (`AppVersion`, `ExportFormatVersion`, `DefaultOllamaURL`, `OpenRouterURL`); wired into `main.go`, `export_service.go`, `suggestion_service.go`. Added `config_test.go` (7 tests: semver format, export convention, URL validation, service wiring). **UI**: Created `lib/config.ts` with 13 tunable levers (zoom bounds/factors, placement dimensions, edge rendering, graph height, link sentinel, textarea rows). Created `randomCanvasPosition()` utility in `lib/utils.ts`, consolidating duplicate logic from `TextCapture.tsx` and `GraphView.tsx`. Replaced `"__waiting__"` magic string with `LINK_MODE_WAITING` constant. Added `config.test.ts` (11 tests) and `utils.test.ts` (4 tests). **Docs**: Created `docs/reference/configuration.md` documenting all levers, env vars, and rationale for what is NOT exposed. Updated `manifest.json`. Total: 80 Go tests, 19 UI tests, all passing. Score: 80→80 (test count up, coverage improved). |

### Checklist
- [x] Storage layer (PostgreSQL schema with 4 tables)
- [x] Scheme CRUD API + UI shell (SchemeList sidebar)
- [x] Canvas view with drag/pan/zoom
- [ ] Information item types (text implemented, 6 more pending)
- [ ] Voice capture + Whisper integration
- [x] Thought graph view + edge management
- [ ] Agent chat integration
- [x] Export API (GET /api/v1/schemes/{id}/export)
- [x] LLM suggestion service + provider fallback
- [ ] Offline/sync layer (IndexedDB)
- [x] Cross-scheme navigation (nullable scheme_id on thoughts)
