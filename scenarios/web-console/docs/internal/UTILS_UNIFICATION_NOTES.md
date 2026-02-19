# Utils Unification Notes

## Last Updated
2026-02-19

## Summary
Utilities are well-structured with clear separation: `lib/` for pure functions and API client, `consts/` for configuration, `hooks/` for React state logic. No `helpers.ts` or `misc.ts` files exist. Both TS and Go sides now follow shared-utility patterns with minimal duplication.

## Completed Consolidations

### 1. ShortcutEntry / ShortcutEntryAPI type unification
- **Before**: `ShortcutEntry` in `consts/shortcuts.ts`, `ShortcutEntryAPI` in `lib/api.ts` — structurally identical
- **After**: `consts/shortcuts.ts` is the single source; `lib/api.ts` re-exports it
- **Eliminated**: Unsafe `as ShortcutEntry[]` cast in TerminalLauncher.tsx

### 2. ErrorInfo centralization
- **Before**: `ErrorInfo` defined in `components/ErrorBanner.tsx`, used by `useSessionManager.ts` (wrong dependency direction: hook → component)
- **After**: `ErrorInfo` lives in `lib/api.ts` alongside its producer `toErrorInfo()`. ErrorBanner re-exports for backward compatibility.

### 3. Selector registry completion
- **Before**: 11 of 35 test-ids registered (31% coverage)
- **After**: All component test-ids registered as either literal or dynamic selectors

### 4. Cross-language coupling documentation
- Added `CROSS-LANGUAGE COUPLING` comments linking Go ↔ TypeScript shared constants:
  - `presetDurations` (Go) ↔ `POLICY_OPTIONS` (TS)
  - `DefaultCols/DefaultRows` (Go) ↔ `DEFAULT_COLS/DEFAULT_ROWS` (TS)

### 5. Go API: `decodeJSON` helper extraction
- **Before**: 4 handlers had identical 5-line JSON decode + error response blocks (session_handlers.go, ai_generate.go, shortcut_profiles.go, ai_provider_config.go)
- **After**: `decodeJSON(w, r, &dst) bool` in `errors.go` — single place for body decoding logic
- **Eliminated**: 4 duplicate decode blocks (20 lines)

### 6. Go API: `lookupSession` helper extraction
- **Before**: 4 identical session-lookup-or-404 blocks across session_handlers.go (×3) and terminal_ws.go
- **After**: `(s *Server).lookupSession(w, r) *Session` in `errors.go` — single place for session resolution
- **Eliminated**: 4 duplicate lookup blocks (20 lines), inconsistent variable naming (`id` vs `sessionID`)

### 7. Go API: Provider timeout + system prompt constants
- **Before**: `30 * time.Second` hardcoded in 3 places; system prompt text duplicated between `buildSystemPrompt()` and `OpenRouterProvider.Generate()`
- **After**: `defaultProviderTimeout` constant and `systemPromptPrefix` constant in `ai_generate.go`
- **Eliminated**: Magic numbers and divergent prompt text

### 8. SettingsPage: toErrorInfo adoption
- **Before**: 4 inline `err instanceof Error ? err.message : "..."` expressions that bypassed `toErrorInfo()`, losing structured APIError fields (recovery hints, retry flag)
- **After**: All 4 catch blocks use `toErrorInfo(err).message`
- **Fixed**: SettingsPage now correctly extracts structured error info from API calls

### 9. Consistent cn() usage across all components
- **Before**: `cn()` from `lib/classnames.ts` was only used in `button.tsx`; 8 other components used template literal class concatenation
- **After**: All dynamic className expressions use `cn()` — SessionsPage, SessionDrawer, ProviderHealthPanel, MobileToolbar, ErrorBanner, Workspace
- **Eliminated**: Template literal className pattern (`className={\`...\`}`) from entire codebase

## Current Utility Architecture

```
ui/src/
├── lib/           # Pure utilities + API client
│   ├── ansi.ts        # ANSI escape codes (3 constants)
│   ├── api.ts         # API client, types, ErrorInfo, toErrorInfo
│   ├── classnames.ts  # cn() — clsx + tailwind-merge (used by ALL components)
│   ├── format.ts      # getShellName, formatSessionTime, truncateId, parseDurationMs, formatCountdown
│   ├── pluralize.ts   # pluralize()
│   └── slugify.ts     # slugify()
├── consts/        # Configuration constants
│   ├── config.ts      # Terminal theme, dimensions, retry settings
│   ├── policy-options.ts  # POLICY_OPTIONS, policyKey()
│   ├── selectors.ts   # Test-id selector registry (literal + dynamic)
│   ├── shortcuts.ts   # ShortcutEntry interface, DEFAULT_SHORTCUTS
│   └── toolbar-keys.ts   # TOOLBAR_KEYS, ToolbarKey
└── hooks/         # React hooks
    ├── useCountdown.ts       # Countdown timer (delegates to lib/format)
    ├── useHashRoute.ts       # Hash-based SPA router
    ├── useSessionManager.ts  # Session lifecycle orchestration
    └── useTerminalSocket.ts  # WebSocket connection manager

api/
├── errors.go          # Error catalog, writeJSON, writeCatalogError, decodeJSON, lookupSession, sanitizeID
├── ai_generate.go     # defaultProviderTimeout, systemPromptPrefix, buildSystemPrompt, extractCommand
└── ...
```

## Remaining Observations

### Not actionable (cross-language mirrors)
- Terminal dimensions: `DEFAULT_COLS/ROWS` (TS) ↔ `DefaultCols/Rows` (Go) — intentional, documented
- Policy presets: `POLICY_OPTIONS` (TS) ↔ `presetDurations` (Go) — intentional, documented

### Low priority
- Policy select widget is structurally similar between SessionDrawer and SessionsPage but differs enough in layout (compact drawer row vs full-page table cell) that extraction would require significant props without meaningful DRY benefit.
- Page layout shell (`flex h-screen flex-col bg-wc-surface-base`) repeats in Workspace, SessionsPage, SettingsPage — would need a `<PageLayout>` component, but the pattern is simple enough that extraction adds indirection without eliminating real drift risk.

## Notes
- No `shared/`, `helpers/`, or `common/` directories needed — the current `lib/consts/hooks` structure maps cleanly to the screaming architecture model.
- All utilities are testable with clear seams (clock injection in useCountdown, socket factory in useTerminalSocket, etc.)
- Go API now centralizes request decoding and session lookup in `errors.go`, reducing per-handler boilerplate significantly.
