# Web Console — Unit Testing Architecture

Last updated: 2026-02-20

## Test Organization Status

### Go API (`api/`)
- [x] Tests co-located with source files (`*_test.go` alongside `*.go`)
- [x] Consistent naming (`TestFunctionName`, `TestType_Method` subtests)
- [x] Table-driven tests for validation logic (`config_test.go`, `decision_boundary_test.go`)
- [x] `t.Helper()` used in test helpers (implicit via Go idiom)
- [x] `t.Cleanup()` for resource cleanup
- [x] Subtests with `t.Run()` for organized test cases

### TypeScript UI (`ui/src/`)
- [x] Tests in `__tests__/` directory (consistent location)
- [x] Consistent naming (`*.test.ts`, `*.test.tsx`)
- [x] Vitest setup file (`test-utils/setup.ts`) with automatic cleanup
- [x] Shared test utilities (`test-utils/`) with mock factories
- [x] Shared render helper (`test-utils/render.tsx`) with QueryClient provider defaults
- [x] `globals: true` in vitest config (no boilerplate imports needed)

## Mock Organization Status

### Go API
- [x] Test doubles co-located in `*_test.go` files (Go idiom — correct)
- [x] `fakePTY` / `fakePTYWithOutput` in `pty_test.go`
- [x] `fakeAIProvider` in `ai_generate_test.go`
- [x] `newTestServer()` / `newFakeTestServer()` in `session_handlers_test.go`
- [x] No inline mock definitions scattered across unrelated test files

### TypeScript UI
- [x] Centralized mock module: `test-utils/mocks.ts`
  - `apiBaseMock()` — `@vrooli/api-base` mock factory (eliminates 5-file duplication)
  - `FakeWebSocket` — WebSocket test double with controllable lifecycle
  - `createFakeSocketPair()` — paired FakeWebSocket + SocketFactory
  - `createMockTerminal()` — xterm Terminal mock with I/O capture
  - `findWriteCall()` — assertion helper for terminal write calls
  - `makeSessions()` — session data factory for component tests
  - `createMockSession()` — single session factory with overrides
  - `mockFetchSuccess()` / `mockFetchError()` — fetch mock installers
- [x] Per-test module mocking via `vi.mock()` where component-specific
- [x] Component testability via `renderWithProviders` and react-query QueryClient injection

## Testability Status

### Go API
- [x] **PTY interface** — abstracts process I/O; tests inject `fakePTY`
- [x] **PTYFactory** function type — injected into `SessionManager` via `NewSessionManagerWithFactory()`
- [x] **ShortcutStore** interface — in-memory impl for tests, PG for production
- [x] **AIConfigStore** interface — in-memory impl for tests, PG for production
- [x] **AIProvider** interface — `fakeAIProvider` for tests
- [x] Compile-time interface compliance checks in `repository_test.go`

### TypeScript UI
- [x] **SocketFactory** type — injectable WebSocket creation seam
- [x] **`@vrooli/api-base`** — mockable via `apiBaseMock()` factory
- [x] **`lib/api.ts`** — all external HTTP calls centralized behind one module
- [x] **`hooks/useTerminalSocket`** — accepts `createSocket` param for test injection

## Infrastructure Status

- [x] Vitest configured in `vite.config.ts` with jsdom environment
- [x] Setup file auto-runs `@testing-library/jest-dom` and cleanup
- [x] Coverage configured (v8 provider, json-summary reporter)
- [ ] Testcontainers for Go database tests (deferred — in-memory stores sufficient for now)
- [x] CI runs tests successfully

## Test Count Summary

| Layer | Test Files | Tests | Pattern |
|-------|-----------|-------|---------|
| Go API | 12 | 101 | Co-located `*_test.go` |
| UI (utilities) | 12 | ~80 | `__tests__/*.test.ts` |
| UI (hooks) | 4 | ~50 | `__tests__/*.test.ts` |
| UI (components) | 8 | ~60 | `__tests__/*.test.tsx` |
| UI (API/features) | 6 | ~54 | `__tests__/*.test.ts` |
| **Total** | **42** | **345** | |

## Key Patterns

### Go: Factory Injection for Side Effects
```go
// Production: real PTY
sm := NewSessionManager()

// Test: fake PTY (no OS process)
sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
```

### Go: Decision Boundary Testing
Extracted decision helpers (`classifyCreateError`, `applySessionDefaults`, `isSessionLimitReached`, etc.) are tested in isolation via `decision_boundary_test.go`, separate from HTTP handler tests.

### TypeScript: Centralized Mock Factory
```typescript
import { apiBaseMock } from "../test-utils";
vi.mock("@vrooli/api-base", () => apiBaseMock());
```

### TypeScript: FakeWebSocket Pattern
```typescript
import { createFakeSocketPair, createMockTerminal } from "../test-utils";
const { fakeWs, createSocket } = createFakeSocketPair();
const terminal = createMockTerminal();
```

### TypeScript: Render Infrastructure Pattern
```typescript
import { renderWithProviders } from "../test-utils";
renderWithProviders(<IntegrationsPanel open={true} />);
```
Provides a consistent QueryClient-backed render path with retries disabled.

## Coverage Snapshot (2026-02-20)

- UI overall coverage (`pnpm vitest run --coverage`):
  - Statements: `77.22%`
  - Branches: `85.47%`
  - Functions: `76.15%`
  - Lines: `77.22%`
- New seam-focused coverage:
  - `consts/policy-options.ts`: `100%` statements/lines/functions
  - `components/IntegrationsPanel.tsx`: unified dependency health display with 9 test cases

## Remaining Improvements

1. **Testcontainers for PG stores** — `PGShortcutStore` and `PGAIConfigStore` still rely on interface-level coverage rather than DB-backed seam tests
2. **UI test placement** — tests are centralized in `__tests__/`; future refactors can incrementally co-locate tests with source modules
3. **Low-coverage entry files** — `App.tsx` and `main.tsx` remain unexercised by unit tests
