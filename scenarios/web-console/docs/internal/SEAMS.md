# Web Console — Seams & Responsibility Boundaries

Last updated: 2026-02-20

## Responsibility Zones

### 1. Entry / Presentation
**Owner**: `ui/src/components/`
- [CODE: ui/src/components/Workspace.tsx] — **Stable core**: pane grid layout, header, empty-state UI. Delegates all session logic to `useSessionManager` hook.
- [CODE: ui/src/components/ErrorBanner.tsx] — **Volatile edge**: reusable error display with category/recovery/retry. Single place to change error UX.
- [CODE: ui/src/components/TerminalPane.tsx] — xterm.js rendering only (no protocol logic)
- [CODE: ui/src/components/TerminalLauncher.tsx] — Modal UI for session creation and shortcut selection (reads shortcuts from [CODE: ui/src/consts/shortcuts.ts])
- [CODE: ui/src/components/SessionDrawer.tsx] — Sidebar with session list and delete controls
- [CODE: ui/src/components/MobileToolbar.tsx] — Floating key toolbar for mobile input injection

### 1b. Session Orchestration
**Owner**: [CODE: ui/src/hooks/useSessionManager.ts]
- `useSessionManager` — **Stable core**: pane state, session CRUD callbacks, terminal ref management, pending command bookkeeping. Separates lifecycle logic from layout.

### 2. Transport / Protocol
**Owner**: [CODE: ui/src/hooks/useTerminalSocket.ts] (client), [CODE: api/terminal_ws.go] (server)
- `useTerminalSocket` — Manages WebSocket connection, bidirectional I/O (stdin/stdout), resize messages, keepalive, and lifecycle events (exit, error, disconnect). Signals readiness via `onReady` callback. Accepts optional `createSocket` factory for test injection.
- `terminal_ws.go` — Server-side WebSocket upgrade, message framing, PTY I/O bridging, ping/pong

### 3. Domain / Session Lifecycle
**Owner**: [CODE: api/session.go], [CODE: api/pty.go]
- `PTY` interface ([CODE: api/pty.go#PTY]) — Abstracts PTY process behind `Read`/`Write`/`SetSize`/`Close`/`Kill`. Default `realPTY` wraps creack/pty; tests substitute `fakePTY` (pipe-based).
- `PTYFactory` ([CODE: api/pty.go#PTYFactory]) — Function type `func(shell, cols, rows) (PTY, error)`. Injected into SessionManager via `NewSessionManagerWithFactory()`.
- `Session` — PTY process wrapper: delegates I/O to `PTY` interface, manages subscribe/unsubscribe/broadcast, offline buffer, exit signaling via `exitCh` channel
- `SessionManager` — Session CRUD, resize (delegates to `PTY.SetSize()`), auto-cleanup on exit (listens on `Session.Done()`)
- Key invariant: Session signals its own exit; SessionManager owns the cleanup decision

### 4. HTTP Transport (REST)
**Owner**: [CODE: api/session_handlers.go]
- Request parsing, response formatting, HTTP status codes
- [CODE: api/session_handlers.go#sessionToResponse] — domain-to-transport conversion
- Policy sub-resource handlers (`handleGetPolicy`, `handleUpdatePolicy`) — operate on `/sessions/{id}/policy`, co-located with other session endpoints
- Delegates all business logic to `SessionManager` and domain modules

### 5. Integration / Infrastructure
**Owner**: [CODE: api/main.go], [CODE: ui/src/lib/api.ts]
- `main.go` — Database connection, router setup, health checks, server lifecycle
- `api.ts` — HTTP/WS client functions, URL construction via `@vrooli/api-base`

### 6. Cross-Cutting
- **Logging**: `log.Printf()` in Go API (simple, adequate for single-user)
- **Error handling**: Structured JSON errors via [CODE: api/session_handlers.go#errorCatalog], TanStack Query in UI
- **Formatting**: [CODE: ui/src/lib/format.ts] — reusable shell name, time, ID truncation utilities
- **Selectors**: [CODE: ui/src/consts/selectors.ts] — centralized data-testid registry for automation
- **Shortcuts**: [CODE: ui/src/consts/shortcuts.ts] — **Volatile edge**: shortcut definitions, decoupled from launcher component

## Testability Seams

### PTY Factory Seam (API)
**File**: `api/pty.go`
**Purpose**: Decouple session management from real PTY/process spawning for fast, deterministic tests.

| Component | Production | Test |
|-----------|-----------|------|
| `PTYFactory` | `defaultPTYFactory` → `realPTY` (creack/pty + exec.Command) | `fakePTYFactory` → `fakePTYWithOutput` (io.Pipe-based) |
| `SessionManager` | `NewSessionManager()` uses `defaultPTYFactory` | `NewSessionManagerWithFactory(factory)` accepts any `PTYFactory` |
| `Session.pty` | `realPTY` (real shell process) | `fakePTYWithOutput` (pipe-based, instant I/O) |

**Benefits**: Tests run without spawning shell processes (faster, no OS dependencies for core logic), resize delegates to the `PTY` interface (testable without ioctl), kill/close behavior is verifiable via the fake's state.

### WebSocket Factory Seam (UI)
**File**: `ui/src/hooks/useTerminalSocket.ts`
**Purpose**: Decouple WebSocket transport from terminal protocol handling for testable hook behavior.

| Component | Production | Test |
|-----------|-----------|------|
| `createSocket` param | `defaultSocketFactory` → `new WebSocket(url)` | Custom factory returning mock/fake WebSocket |
| `ANSI` constants | Used internally for terminal messages | Exported for test assertions |
| `SocketFactory` type | `(url: string) => WebSocket` | Same signature, mock implementation |

**Benefits**: Hook can be tested with a mock WebSocket (no real connections needed), message handling logic (stdout/exit/error/ping) can be exercised in isolation.

### API-Base Mock Seam (UI)
**File**: `ui/src/test-utils/mocks.ts`
**Purpose**: Centralize `@vrooli/api-base` mock so all test files that depend on API URL resolution use a single, consistent factory.

| Component | Production | Test |
|-----------|-----------|------|
| `@vrooli/api-base` module | `resolveApiBase()` reads env/window config | `apiBaseMock()` returns deterministic localhost URLs |
| `buildApiUrl` / `buildWsUrl` | Constructs URLs from runtime base | Pass-through concatenation for predictable assertions |

**Benefits**: Eliminates 5-file mock duplication (previously each test file copied 7 lines of mock config with inconsistent port numbers). Single change point when the api-base interface evolves.

### Shared Test Doubles Seam (UI)
**File**: `ui/src/test-utils/mocks.ts`
**Purpose**: Provide reusable test doubles for WebSocket, terminal, and session data so tests focus on behavior, not boilerplate setup.

| Double | What It Replaces | Used By |
|--------|-----------------|---------|
| `FakeWebSocket` | Real `WebSocket` via `SocketFactory` seam | `useTerminalSocket.hook.test.ts` |
| `createMockTerminal()` | xterm.js `Terminal` instance | WebSocket hook tests |
| `findWriteCall()` | Inline assertion search across terminal writes | WebSocket hook tests |
| `makeSessions()` | Inline session data construction | Component tests (SessionDrawer, etc.) |
| `createMockSession()` | Inline `SessionInfo` object literals | Any test needing session data |
| `mockFetchSuccess()` / `mockFetchError()` | Repeated `globalThis.fetch = vi.fn(...)` pattern | API client tests |

**Benefits**: New tests can set up realistic test data in one line. Mock behavior is consistent across test files. Changes to data shapes (e.g., adding a field to `SessionInfo`) require updating one factory, not many test files.

### Policy Selection Parse Seam (UI)
**File**: `ui/src/consts/policy-options.ts`
**Purpose**: Centralize policy select value parsing to avoid duplicated string-splitting logic and undefined behavior across session UIs.

| Component | Before | After |
|-----------|--------|-------|
| `SessionDrawer` | Inline parse (`if val === "never" else split(":")`) | Uses `parsePolicySelection(value)` helper |
| `SessionsPage` | Inline parse with separate branch/split logic | Uses `parsePolicySelection(value)` helper |
| Invalid values | Implicitly assumed valid | Explicit `null` return; caller no-ops safely |

**Benefits**: Single source of truth for UI policy parsing decisions, tighter edge-case tests at seam boundaries, and reduced drift risk between pages.

### Provider Health API Injection Seam (UI)
**File**: `ui/src/components/ProviderHealthPanel.tsx`
**Purpose**: Decouple ProviderHealthPanel UI behavior from hard-coded API module imports by allowing seam-level dependency injection.

| Component | Production | Test |
|-----------|------------|------|
| `ProviderHealthPanel` API dependency | Default `getAIConfig`/`updateAIConfig` adapter | Injected `ProviderHealthPanelApi` fake |
| Refresh/toggle behavior tests | Required global/module mocking | Uses direct fake API object with deterministic calls |

**Benefits**: Reduced global mocking, lower test setup coupling, and easier behavior-focused tests for loading, refresh, toggle, and error states.

## Boundary Violations Fixed

### Phase 2 (2026-02-19) — Responsibility Boundaries
| Violation | Before | After |
|-----------|--------|-------|
| WebSocket protocol in TerminalPane | TerminalPane mixed xterm.js rendering with WS protocol | Extracted to `useTerminalSocket` hook |
| Data formatting in SessionDrawer JSX | Inline `split("/").pop()`, `toLocaleTimeString()` | Extracted to `lib/format.ts` utilities |
| setTimeout shortcut injection | `setTimeout(500)` timing assumption in Workspace | Event-driven `onReady` callback from TerminalPane |
| ANSI escape codes scattered | Hardcoded `\x1b[90m` in TerminalPane | Centralized `ANSI` constants in useTerminalSocket |
| Implicit onExit callback | `readLoop(onExit func(string))` mutated SessionManager | `exitCh` channel; SessionManager listens on `Done()` |
| Silent JSON decode errors | `_ = json.Decode()` in handler | Logged with `log.Printf` for debugging |

### Phase 3 (2026-02-19) — Seam Discovery & Enforcement
| Violation | Before | After |
|-----------|--------|-------|
| PTY creation hardcoded in SessionManager | `exec.Command` + `pty.StartWithSize` inline in `Create()` | `PTYFactory` function type; `defaultPTYFactory` in production |
| Session held raw `*os.File` + `*exec.Cmd` | `s.ptmx.Write()`, `s.cmd.Process.Kill()` | `s.pty.Write()`, `s.pty.Kill()` via PTY interface |
| Resize bypassed Session encapsulation | `pty.Setsize(sess.ptmx, ...)` in SessionManager | `sess.pty.SetSize(cols, rows)` via PTY interface |
| WebSocket hardcoded in hook | `new WebSocket(url)` in useEffect | `createSocket(url)` with injectable factory |
| ANSI constants private | `const ANSI` not exported | `export const ANSI` for test verification |

### Phase 8 (2026-02-19) — Change Axis & Evolution Resilience
| Change | Before | After |
|--------|--------|-------|
| Shortcut data in component | Hardcoded `DEFAULT_SHORTCUTS` array in `TerminalLauncher.tsx` | Extracted to `consts/shortcuts.ts`; component imports from data module |
| Error banner duplicated | Two inline error banner implementations in Workspace.tsx (empty + active state) | Single `ErrorBanner.tsx` component used in both states |
| Session logic in layout | Workspace.tsx mixed pane layout with session CRUD, ref management, error state | Extracted `useSessionManager` hook; Workspace is pure layout |
| No variation tests | Tests validated specific values only | Added structural invariant tests (`TestErrorCatalog_StructuralInvariants`, `TestSessionLimit_VariousLimits`, `shortcuts.test.ts`) |

## Remaining Ownership Issues

1. ~~**Shortcut defaults hardcoded** in `TerminalLauncher.tsx`~~ — **Resolved Phase 8**: Extracted to `consts/shortcuts.ts`
2. **No reconnect logic** — If WebSocket disconnects, no auto-reconnect. Should be a transport-layer concern in `useTerminalSocket`
3. **No session persistence** — In-memory only; SQLite backend is a P1 domain concern
4. **No structured logging** — Simple `log.Printf` across API; should use structured logger at integration boundaries
5. ~~**API client hardcoded in Workspace**~~ — **Resolved Phase 8**: Session lifecycle extracted to `useSessionManager` hook

## Change Axes

Primary axes of change identified in Phase 8, with current cost assessment and structural adjustments.

### Axis 1: Shortcut Profiles (P0-006, P1-010)
**What changes**: Adding/removing/modifying launch shortcuts, config-driven profiles
**Cost before Phase 8**: Medium — shortcuts hardcoded in `TerminalLauncher.tsx` component mixed with UI rendering
**Cost after Phase 8**: Low — all shortcut data lives in `consts/shortcuts.ts`, component only renders from props
**Files to touch**: `consts/shortcuts.ts` (data), optionally `TerminalLauncher.tsx` (if UI changes needed)
**Test coverage**: `shortcuts.test.ts` validates structural invariants (uniqueness, non-empty, PRD compliance)

### Axis 2: Toolbar Keys (P0-007)
**What changes**: Adding/removing mobile toolbar keys and escape sequences
**Cost**: Already low — `TOOLBAR_KEYS` array in `MobileToolbar.tsx` is declarative and self-contained
**Files to touch**: `MobileToolbar.tsx` (add entry to array)
**Test coverage**: `toolbar-keys.test.ts` validates escape sequences per key

### Axis 3: Error Codes & Recovery (API + UI)
**What changes**: Adding new error types, adjusting recovery hints, new categories
**Cost**: Low — API: add entry to `errorCatalog` map in `session_handlers.go`; UI: `ErrorBanner.tsx` renders any `ErrorInfo` shape
**Files to touch**: `session_handlers.go` (catalog entry), optionally `useTerminalSocket.ts` (WS recovery)
**Test coverage**: `TestErrorCatalog_StructuralInvariants` validates all entries have valid category, message, recovery, status. `TestWriteJSONError_UnknownCode_Fallback` verifies graceful degradation for new codes.
**Invariant**: Unknown codes fall back to `internal` category with generic recovery hint.

### Axis 4: Session Policies (P1-001)
**What changes**: Adding config knobs (env vars), new policy limits, expiration behavior
**Cost**: Low — `config.go` centralizes all tunables with env var mapping and validation/clamping
**Files to touch**: `config.go` (add field + env var), `session.go` (apply policy)
**Test coverage**: `config_test.go` covers defaults, env overrides, clamping, invalid fallback. `TestSessionLimit_VariousLimits` validates limit behavior across multiple values.

### Axis 5: WebSocket Protocol (P0-002b)
**What changes**: Adding message types, changing framing, adjusting handshake
**Cost**: High (inherently coupled) — requires coordinated changes in `terminal_ws.go` and `useTerminalSocket.ts`
**Files to touch**: `terminal_ws.go` (server), `useTerminalSocket.ts` (client), both message type definitions
**Mitigation**: Message types are string constants on both sides; `TerminalMessage` interface/struct serves as the protocol contract. Adding new types is additive and backward-compatible.

### Axis 6: Terminal Appearance
**What changes**: Theme colors, font stack, font size
**Cost**: Low — all values centralized in `consts/config.ts`, consumed by `TerminalPane.tsx` only
**Files to touch**: `consts/config.ts`
**Test coverage**: `config.test.ts` validates exports

### Axis 7: Pane Layout
**What changes**: Grid behavior, min dimensions, column logic
**Cost**: Low — grid constants in `consts/config.ts`, grid CSS logic isolated in `Workspace.tsx`
**Files to touch**: `consts/config.ts` (constants), `Workspace.tsx` (grid style block)

### Stable Core vs Volatile Edges Summary

| Module | Stability | Notes |
|--------|-----------|-------|
| `session.go` / `SessionManager` | **Stable core** | PTY lifecycle, subscribe/broadcast — unlikely to change shape |
| `pty.go` / PTY interface | **Stable core** | Abstraction boundary — changes only if PTY API changes |
| `terminal_ws.go` / WS protocol | **Stable core** | Message framing — additive changes only |
| `main.go` / server wiring | **Stable core** | Router + middleware — rarely touched |
| `useTerminalSocket.ts` | **Stable core** | WS lifecycle hook — additive message types only |
| `useSessionManager.ts` | **Stable core** | Session orchestration — change when API contract changes |
| `TerminalPane.tsx` | **Stable core** | xterm.js rendering — change only for terminal feature additions |
| `consts/shortcuts.ts` | **Volatile edge** | Shortcut definitions — expected to change with profiles (P1-010) |
| `consts/config.ts` | **Volatile edge** | UI tunables — expected to grow with new features |
| `config.go` | **Volatile edge** | API tunables — expected to grow with new policies (P1-001) |
| `errorCatalog` (session_handlers.go) | **Volatile edge** | Error definitions — grows with new error paths |
| `TOOLBAR_KEYS` (MobileToolbar.tsx) | **Volatile edge** | Key definitions — grows with new mobile keys |
| `ErrorBanner.tsx` | **Volatile edge** | Error display — changes with new recovery UX |
| `Workspace.tsx` | **Mixed** | Layout is stable; wiring to hooks is stable; grid behavior may evolve |

## Decision Points

Phase 14 extracted the following decision points into named, testable helpers. Each entry identifies **where the decision is made** and **what it decides**.

### Extracted Decision Helpers (API)

| Helper | File | Decision | Inputs |
|--------|------|----------|--------|
| `classifyCreateError(err)` | `session_handlers.go` | Maps session creation errors (limit, PTY, unknown) to HTTP error codes and categories | `error` sentinel type |
| `applySessionDefaults(shell, cols, rows)` | `session.go` | Substitutes configured defaults for zero/empty caller values | Zero-value convention |
| `isSessionLimitReached()` | `session.go` | Whether a new session should be rejected (MaxSessions cap; 0 = unlimited) | Session count, config |
| `buildPolicyResponse(sess, policy)` | `session_policy.go` | Constructs policy response with TTL/expiry fields (0 = never-expire) | Session creation time, policy mode |
| `resolveShell()` | `config.go` | Full shell fallback chain: `WC_DEFAULT_SHELL` → `$SHELL` → `/bin/sh` | Environment variables |
| `checkProviderResponse(resp, name)` | `ai_generate.go` | Whether an AI provider HTTP response indicates success (200) or failure | HTTP status code |
| `extractCommand(raw)` via `knownCodeFences` | `ai_generate.go` | Which markdown fences to strip from AI output (bash, sh, generic only) | Raw AI text |
| `customDurationMin` / `customDurationMax` | `session_policy.go` | Allowed range for custom policy durations (1m–7d) | Named constants |

### Extracted Decision Helpers (UI)

| Helper | File | Decision | Inputs |
|--------|------|----------|--------|
| `isCleanWsClose(code)` | `useTerminalSocket.ts` | Whether a WebSocket close is intentional (1000/1001) vs. unexpected | Close code |

### Decision Groupings by Domain

| Domain | Where Decisions Live | Key Functions |
|--------|---------------------|---------------|
| **Session lifecycle** | `session.go` | `applySessionDefaults`, `isSessionLimitReached`, `Create`, `Delete` |
| **Session policy** | `session_policy.go` | `ValidatePolicy`, `ResolveTTL`, `IsExpired`, `buildPolicyResponse` |
| **Error classification** | `session_handlers.go` | `classifyCreateError`, `writeJSONError`, `errorCatalog` |
| **AI command extraction** | `ai_generate.go` | `extractCommand`, `knownCodeFences`, `checkProviderResponse` |
| **Configuration** | `config.go` | `resolveShell`, `envInt`, `LoadConfig` |
| **WebSocket transport** | `terminal_ws.go` (server), `useTerminalSocket.ts` (client) | `isCleanWsClose`, WS message dispatch |

### Well-Extracted vs. Still-Scattered

**Well-extracted** (each has a single named "home"):
- Error classification → `classifyCreateError` + `errorCatalog`
- Policy response construction → `buildPolicyResponse`
- Shell resolution → `resolveShell`
- AI output cleaning → `extractCommand` with explicit `knownCodeFences`
- WS close classification → `isCleanWsClose`
- Session defaults → `applySessionDefaults`
- Session limit check → `isSessionLimitReached`

**Still scattered** (documented, not yet refactored):
- `parseDurationMs` (UI) vs `time.ParseDuration` (Go): two separate duration parsers with different capabilities; Go accepts `1h30m`, UI only handles single-unit `1h`/`30m`/`45s`
- WebSocket origin check: `CheckOrigin: func(r *http.Request) bool { return true }` — security decision expressed as always-true with comment justification only

## Architecture Clarity Notes

### Phase 15 — Cognitive Load Reduction (2026-02-19)

**Major simplifications:**
- **Error writing consolidation**: Renamed `writeJSONError(w, status, code, message)` → `writeCatalogError(w, code, message)`. The old function accepted a `status` parameter that always matched the catalog entry, creating a confusing dual-source pattern. The new function derives status entirely from the catalog, reducing the reader's mental burden from "which source wins?" to "catalog owns the contract".
- **Duration/countdown co-location**: Moved `parseDurationMs()` and `formatCountdown()` from `SessionDrawer.tsx` (where they were scattered among UI components) into `lib/format.ts` (where all formatting utilities live). Reading SessionDrawer no longer requires mentally separating formatting math from layout logic.
- **AiInput Enter key clarity**: Renamed `handleKeyDown` → `handleEnterKey` with explicit comment documenting the two-phase behavior (generate vs execute). Added a named `hasGeneratedCommand` variable to make the state-machine decision self-documenting.
- **WS teardown comment**: Added inline explanation of the `select/default` pattern in the output forwarder's `writerDone` close, making the "close-only-once" invariant immediately visible.

**Complexity hotspots remaining:**
- `SessionDrawer.tsx` still mixes the `useCountdown` hook (timing logic) with session list rendering — could be extracted to a standalone hook but the coupling is currently acceptable.
- AI input state machine has a 3-handler mutation surface (`onChange`, `handleEnterKey`, `handleGenerate`), but the states are now explicitly named.

## Architecture Alignment

### Phase 17 — Screaming Architecture Audit (2026-02-19)

**Mental Model**: Web Console is a browser terminal with 6 domain concepts:
1. Session Management (PTY-backed terminal sessions)
2. Terminal I/O (WebSocket bridge)
3. AI Command Generation (NL → shell command)
4. Shortcut Profiles (configurable launch shortcuts)
5. Observability (events, metrics)
6. Configuration (tunables)

**Logical → Physical Alignment**:

The API uses a hybrid organization:
- **Cross-cutting infrastructure** (`errors.go`) owns the error catalog, types, and HTTP error helpers — all handler files depend on this single source.
- **Core domain (sessions)** is layer-split: `session.go` (domain) + `session_handlers.go` (HTTP). Justified by being the largest feature.
- **Feature modules** (AI, shortcuts, metrics) are feature-sliced: each file owns domain + HTTP together.
- **AI generation** (`ai_generate.go`) owns the full pipeline including config-aware orchestration (`generateWithConfig`). Config storage/health lives in `ai_provider_config.go`.
- **Policy** is a special case: domain logic in `session_policy.go`, but HTTP handlers in `session_handlers.go` (because they're session sub-resource endpoints on `/sessions/{id}/policy`).

**Gaps fixed in iteration 1:**
| Gap | Before | After |
|-----|--------|-------|
| Policy handlers in wrong file | `session_policy.go` mixed domain logic with HTTP handlers for `/sessions/{id}/policy` | Handlers moved to `session_handlers.go`; `session_policy.go` is pure domain |
| Wrong error code for shortcut deletion | `handleDeleteShortcutProfile` returned `session_not_found` error code | New `profile_not_found` error code in catalog with correct recovery hint |
| Misplaced docs | `docs/PROBLEMS.md` and `docs/PROGRESS.md` duplicated at root and `docs/internal/` | Removed root copies; `docs/internal/` is canonical |
| RESEARCH.md not in standard location | `docs/RESEARCH.md` outside `internal/` | Moved to `docs/internal/RESEARCH.md`; manifest updated |
| File Map incomplete | Architecture doc listed only 19 files | Expanded to 35 files covering all API modules, UI pages, hooks, and utilities |
| No organizational pattern documented | Readers couldn't tell if feature-slicing was intentional | Added "Code Organization Pattern" section to ARCHITECTURE.md |

**Gaps fixed in iteration 2:**
| Gap | Before | After |
|-----|--------|-------|
| Error infrastructure in session_handlers.go | `session_handlers.go` mixed error catalog + types + helpers with session HTTP handlers (god file, 405 lines, 5+ concerns) | Extracted to `errors.go`; session_handlers.go is now pure session HTTP handlers |
| `generateWithConfig` misplaced in config file | `ai_provider_config.go` owned generation orchestration logic that only `ai_generate.go` called | Moved to `ai_generate.go`; config file owns only storage + health tracking |
| Duplicate POLICY_OPTIONS (UI) | Identical `POLICY_OPTIONS` array defined in both `SessionDrawer.tsx` and `SessionsPage.tsx` | Extracted to `consts/policy-options.ts`; both components import from single source |
| Duplicate countdown logic (UI) | `useCountdown` hook in `SessionDrawer.tsx` + `PolicyCountdown` component in `SessionsPage.tsx` duplicated same timer logic | Extracted shared `hooks/useCountdown.ts`; both consumers use the single hook |
| Inline ID truncation (UI) | `SessionsPage.tsx` used `session.id.slice(0, 8)` instead of existing `truncateId()` utility | Now uses `truncateId()` from `lib/format.ts` consistently |
| `policyKey` helper duplicated | Inline in `SessionDrawer.tsx` + repeated pattern in `SessionsPage.tsx` | Exported from `consts/policy-options.ts`; both components share it |

**Remaining drift** (documented, not addressed):
- Feature-sliced files (`shortcut_profiles.go`, `ai_provider_config.go`) could benefit from handler extraction if they grow larger, but current size doesn't warrant the split
- `ai_generate.go` contains both provider implementations (Ollama, OpenRouter) and the HTTP handler; if more providers are added, extracting providers into separate files would improve clarity

## Observability Surface

### Phase 20 (2026-02-19) — Signal & Feedback Surface Design

#### Key Observable States

| State | Where Surfaced | Signal |
|-------|---------------|--------|
| Server healthy / degraded | `GET /health`, `GET /api/v1/metrics` | Health endpoint returns DB status; metrics show uptime and counters |
| Session created / active | `GET /api/v1/sessions`, `GET /api/v1/events`, UI session list | Session list, `session.created` event, `ActiveSessions` metric |
| Session connected (WS) | `GET /api/v1/events`, `GET /api/v1/metrics` | `session.connected` event, `ActiveConnections` metric |
| Session exited | WS `exit` message (with real exit code), `[EVENT]` log, UI terminal text | Exit code forwarded to client; red text for non-zero, gray for clean exit |
| Session expired (policy) | `[EVENT]` log, `GET /api/v1/events` | `session.terminated` event with reason/policy/duration details |
| Policy updated | `[EVENT]` log, `GET /api/v1/events` | `session.policy_updated` event (named constant, not inline string) |
| AI generation attempted | `[EVENT]` log, `GET /api/v1/events`, `GET /api/v1/ai/health` | `ai.generate` event with provider name; per-provider health tracking |
| AI provider failure | API log, `GET /api/v1/ai/health`, UI error in AiInput | Per-provider error count/rate; structured error with recovery hint |
| Offline buffer overflow | API log (once per session) | `session %s: offline buffer full` — de-duplicated, one-shot |
| Policy update failure | UI inline error banner in SessionDrawer | Red banner with recovery hint, auto-dismiss 5s |
| Provider panel failure | UI inline error in ProviderHealthPanel | Amber warning with error message |

#### Signal Inventory

**HTTP Endpoints (observability-focused):**
| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Service + DB readiness |
| `GET /api/v1/metrics` | Atomic counters: sessions, connections, messages, AI, uptime |
| `GET /api/v1/events?limit=N` | Recent structured events from in-memory ring buffer (default 50, max 1000) |
| `GET /api/v1/ai/health` | Per-provider availability, latency, error rate |

**Structured Events** (all emitted via `EventLogger`, logged as `[EVENT] {json}`):
| Constant | Value | Details |
|----------|-------|---------|
| `EventSessionCreated` | `session.created` | shell, cols, rows |
| `EventSessionConnected` | `session.connected` | — |
| `EventSessionDisconnected` | `session.disconnected` | — |
| `EventSessionTerminated` | `session.terminated` | reason, policy, duration |
| `EventSessionDeleted` | `session.deleted` | — |
| `EventPaneResized` | `pane.resized` | cols, rows |
| `EventAIGenerate` | `ai.generate` | provider, prompt |
| `EventSessionPolicyUpdate` | `session.policy_updated` | mode, duration |

**WebSocket Protocol Signals:**
| Message | Direction | Signal |
|---------|-----------|--------|
| `exit` | Server→Client | Process exited; `code` field carries real exit code (0=clean, non-zero=failure) |
| `error` | Server→Client | Runtime error with known recovery hints for common cases |
| `pong` | Server→Client | Keepalive response confirming connection liveness |

**UI Feedback Surfaces:**
| Component | Signal Type | Behavior |
|-----------|------------|----------|
| App startup | Loading spinner | "Connecting to API..." with 3 retries, then error page with retry button |
| Session creation | ErrorBanner | Structured error with category, recovery hint, retry button; auto-dismiss 8s |
| Terminal exit | ANSI text | Gray "[Session ended]" for code 0; red "[Session ended with exit code N]" for non-zero |
| WS disconnect | ANSI text | Gray "[Disconnected]" for clean close; red "[Connection lost]" with recovery guidance |
| Policy update failure | Inline banner in drawer | Red alert with recovery hint, auto-dismiss 5s, dismissible |
| Provider panel failure | Inline warning | Amber warning showing error message |
| AI generation | Loading spinner + error | Spinning icon during generation; inline error with message on failure |

#### Remaining Signal Debt

1. **No event stream endpoint** — Events are polled via `GET /api/v1/events`. An SSE or WebSocket-based real-time event stream would enable live dashboards without polling. Low priority for single-user.
2. **No structured logging** — API uses `log.Printf` (text). A structured logger (slog) would enable machine-parseable log aggregation. Documented in PROBLEMS.md, deferred.
3. **No Prometheus/OpenTelemetry** — Metrics are JSON-only poll. External observability integration is a future concern.
4. **WebSocket reconnect** — No auto-reconnect on disconnect. Documented as remaining ownership issue.
5. **Session delete from UI** — No confirmation feedback beyond the session disappearing from the list. Low priority.
