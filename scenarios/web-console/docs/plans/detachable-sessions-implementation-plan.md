# Detachable Sessions — Implementation Plan

## 1. Purpose

Web console terminal sessions are currently in-memory and die when the server restarts. This makes it impossible to use web console to develop web console itself — any restart kills the agent's session mid-work. This plan introduces **detachable sessions** backed by tmux, a configurable backend registry, and automatic reconnection on restart, so sessions can survive server restarts.

## 2. Required Reading

```bash
# Skills for the implementing agent
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement test

# Existing codebase context
cat scenarios/web-console/docs/internal/SEAMS.md
cat scenarios/web-console/docs/internal/PROGRESS.md
cat scenarios/web-console/api/pty.go
cat scenarios/web-console/api/session.go
cat scenarios/web-console/api/session_policy.go
cat scenarios/web-console/api/session_handlers.go
cat scenarios/web-console/api/config.go
cat scenarios/web-console/api/main.go
cat scenarios/web-console/ui/src/components/TerminalLauncher.tsx
cat scenarios/web-console/ui/src/components/settings/SessionManagementSection.tsx
cat scenarios/web-console/ui/src/consts/policy-options.ts
cat scenarios/web-console/ui/src/hooks/useSessionManager.ts
cat scenarios/web-console/ui/src/lib/api.ts
```

## 3. Problem Statement

When web console's API process terminates (restart, crash, upgrade), every terminal session is destroyed because:

1. **PTY processes are direct children** of the API process — they receive SIGHUP and die.
2. **Session state is purely in-memory** — the `SessionManager.sessions` map is lost.
3. **No discovery mechanism exists** to find sessions that might have survived independently.

This forces users who want to modify web console to switch to a completely separate app (Agent Manager), which is friction that defeats the purpose of having a capable terminal in the browser.

### Root Causes

| Cause | Layer | Impact |
|-------|-------|--------|
| PTY is a direct child process | `pty.go` / OS | Shell dies with server |
| Session map is in-memory only | `session.go` | State lost on restart |
| No backend abstraction | `session.go` | Can't swap PTY strategies |
| No session metadata persistence | `main.go` / DB | Can't reconstruct sessions |
| Launch UI has no backend selector | `TerminalLauncher.tsx` | Users can't choose persistence |
| Settings have no default backend config | Settings UI | No way to set preferences |

## 4. Scope

### In Scope

- Backend registry system defining available session backends (raw PTY, tmux)
- tmux-backed PTY implementation behind the existing `PTY` interface
- Session metadata persistence in SQLite for restart recovery
- Server-startup discovery and re-registration of surviving tmux sessions
- Launch dialog updates: backend selector + timeout selector (pre-populated from defaults)
- Settings page updates: default backend + default timeout policy
- Frontend reconnection to recovered sessions (leveraging existing hydration)
- Comprehensive test coverage for all new code
- Documentation updates (SEAMS.md, PROGRESS.md, api-endpoints.md, configuration.md)

### Out of Scope

- Live migration of running raw PTY sessions to tmux (not possible — see section 12)
- Additional backends beyond raw PTY and tmux (future work; registry supports it)
- tmux installation automation (documented as a prerequisite)
- Changes to the WebSocket transport layer (it already supports reconnection)
- Changes to the output history/replay mechanism (works as-is with tmux)
- Desktop/mobile tier concerns for this feature (tmux is server-only by nature)

## 5. Current Technical Context

### Key Files and Their Roles

| File | Role | Relevance |
|------|------|-----------|
| `api/pty.go` | PTY interface + raw factory | **Primary extension point** — new tmux factory here |
| `api/session.go` | Session struct + SessionManager | Needs backend field, metadata persistence hooks |
| `api/session_policy.go` | Expiration policy + sweeper | Mostly unchanged; sweeper must skip tmux cleanup edge cases |
| `api/session_handlers.go` | HTTP CRUD for sessions | Accepts `backend` field in create, returns it in responses |
| `api/config.go` | Config struct + env loading | New `WC_DEFAULT_BACKEND` env var |
| `api/main.go` | Server init + schema | Discovery on startup, schema migration for session metadata |
| `ui/src/components/TerminalLauncher.tsx` | Launch popup | Add backend + timeout selectors |
| `ui/src/components/settings/SessionManagementSection.tsx` | Session settings | Show backend per session |
| `ui/src/consts/policy-options.ts` | Policy constants | Add backend option constants |
| `ui/src/hooks/useSessionManager.ts` | Session lifecycle | Pass backend to create, handle recovered sessions |
| `ui/src/lib/api.ts` | API client | Add backend fields to types and calls |

### Existing Seams That Enable This Work

1. **PTY Factory Seam** (`PTYFactory func(spec SessionLaunchSpec) (PTY, error)`) — The `SessionManager` already accepts an injected factory. The tmux backend is a new factory implementation, not a rewrite.

2. **Session Create Flow** — `SessionManager.Create()` calls the factory, then starts a read loop. The read loop reads from the `PTY` interface, which tmux will implement identically.

3. **WebSocket Reconnection** — `useTerminalSocket.ts` already handles reconnection with exponential backoff and history offset resumption. Surviving sessions just need to exist on the server side.

4. **Workspace Persistence** — Pane metadata (name, color, theme, sort order) is already stored in SQLite. Session metadata persistence follows the same pattern.

5. **Hydration Flow** — `useSessionManager` already calls `listSessions()` on mount and rebuilds panes from backend state. If sessions survive restart, hydration works without changes.

## 6. Target End State

After implementation:

1. Users can choose "Standard" (raw PTY) or "Persistent" (tmux) when creating a session, with the default configurable in settings.
2. Persistent sessions survive web console restarts — on startup, the server discovers running tmux sessions, re-registers them, and the frontend reconnects seamlessly.
3. The backend registry reports which backends are available (tmux might not be installed), and the UI only shows available options.
4. Session metadata (ID, backend type, policy, created_at) is persisted in SQLite so it can be restored on startup.
5. All new code is covered by tests. The tmux backend is tested via the same `PTY` interface contract tests that cover the raw backend.

## 7. Implementation Strategy

### Phase 1: Backend Registry (API)

**Goal:** Establish the registry pattern and availability detection without changing any existing behavior.

**Files to create/modify:**
- **Create** `api/backend_registry.go` — Registry type, backend descriptors, availability checks
- **Modify** `api/config.go` — Add `DefaultBackend string` field (env: `WC_DEFAULT_BACKEND`, default: `"standard"`)
- **Modify** `api/main.go` — Initialize registry, expose via capabilities endpoint

**Backend Registry Design:**

```go
// BackendID is a typed string for backend identifiers.
type BackendID string

const (
    BackendStandard  BackendID = "standard"
    BackendPersistent BackendID = "persistent"
)

// BackendDescriptor describes a session backend's capabilities and availability.
type BackendDescriptor struct {
    ID            BackendID
    DisplayName   string
    Description   string
    SurvivesRestart bool
    Available     bool
    Reason        string // Why unavailable, if applicable
}

// BackendRegistry tracks available session backends.
type BackendRegistry struct {
    backends map[BackendID]BackendDescriptor
    factories map[BackendID]PTYFactory
}

func NewBackendRegistry() *BackendRegistry
func (r *BackendRegistry) Register(desc BackendDescriptor, factory PTYFactory)
func (r *BackendRegistry) Get(id BackendID) (BackendDescriptor, bool)
func (r *BackendRegistry) Factory(id BackendID) (PTYFactory, bool)
func (r *BackendRegistry) Available() []BackendDescriptor
func (r *BackendRegistry) IsAvailable(id BackendID) bool
```

**Availability Detection:**

```go
// checkTmuxAvailable probes for tmux and returns availability + reason.
func checkTmuxAvailable() (bool, string) {
    path, err := exec.LookPath("tmux")
    if err != nil {
        return false, "tmux is not installed"
    }
    // Verify minimum version (tmux 2.6+ for -f flag support)
    out, err := exec.Command(path, "-V").Output()
    // Parse version, check >= 2.6
    ...
    return true, ""
}
```

**API Endpoint:**

Extend the existing capabilities endpoint (`GET /api/v1/capabilities`) to include:

```json
{
  "session_backends": [
    {
      "id": "standard",
      "display_name": "Standard",
      "description": "In-memory session. Fast and lightweight, but lost on restart.",
      "survives_restart": false,
      "available": true
    },
    {
      "id": "persistent",
      "display_name": "Persistent",
      "description": "Backed by tmux. Survives web console restarts.",
      "survives_restart": true,
      "available": true
    }
  ],
  "default_backend": "standard"
}
```

**Tests:**
- `api/backend_registry_test.go` — Registration, lookup, availability filtering, factory dispatch, unknown backend handling

---

### Phase 2: Session Metadata Persistence (API)

**Goal:** Persist session metadata in SQLite so sessions can be reconstructed after restart.

**Files to modify:**
- **Modify** `initialization/sqlite/schema.sql` — Add `sessions` table
- **Modify** `api/session.go` — Add `Backend BackendID` field to `Session`, add metadata store interface
- **Create** `api/session_store.go` — SQLite-backed session metadata repository
- **Modify** `api/main.go` — Initialize store, inject into SessionManager

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    backend     TEXT NOT NULL DEFAULT 'standard',
    shell       TEXT NOT NULL,
    cols        INTEGER NOT NULL,
    rows        INTEGER NOT NULL,
    policy_mode TEXT NOT NULL DEFAULT 'never',
    policy_duration TEXT,
    created_at  TEXT NOT NULL,  -- RFC 3339
    detached    INTEGER NOT NULL DEFAULT 0  -- 1 if session is expected to survive restart
);
```

**Session Metadata Store Interface:**

```go
// SessionMetadataStore persists session metadata for restart recovery.
type SessionMetadataStore interface {
    Save(meta SessionMetadata) error
    Get(id string) (SessionMetadata, error)
    List() ([]SessionMetadata, error)
    Delete(id string) error
    UpdatePolicy(id string, policy ExpirationPolicy) error
}

type SessionMetadata struct {
    ID              string
    Backend         BackendID
    Shell           string
    Cols            uint16
    Rows            uint16
    Policy          ExpirationPolicy
    CreatedAt       time.Time
    Detached        bool
}
```

**Integration Points:**
- `SessionManager.Create()` — After successful PTY spawn, persist metadata
- `SessionManager.Delete()` — Remove metadata
- `Session.SetPolicy()` — Update persisted policy
- Auto-cleanup goroutine — Remove metadata when PTY exits

**Tests:**
- `api/session_store_test.go` — CRUD operations, concurrent access, policy updates, cleanup on delete

---

### Phase 3: tmux PTY Backend (API)

**Goal:** Implement the tmux-backed `PTY` interface.

**Files to create:**
- **Create** `api/pty_tmux.go` — tmux PTY implementation
- **Create** `api/pty_tmux_test.go` — Unit + integration tests

**System Dependency:**

tmux is installed as a common system dependency via `scripts/lib/system/common_deps.sh` (line: `system::check_and_install "tmux"`). Running `vrooli setup` installs it automatically. The backend registry's availability check (`checkTmuxAvailable()`) confirms it's present at runtime.

**tmux Session Naming Convention:**

```
wc-{sessionID}
```

Prefix `wc-` ensures web console sessions are distinguishable from user tmux sessions.

**tmux PTY Implementation:**

```go
// tmuxPTY implements the PTY interface using a tmux session as the backing process.
type tmuxPTY struct {
    sessionName string   // tmux session name: "wc-{id}"
    ptmx        *os.File // PTY master connected to tmux client
    cmd         *exec.Cmd // tmux attach-session process (for I/O)
    shellPID    int      // PID of the shell inside tmux (for HasChildProcess)
    mu          sync.Mutex
    closed      bool
}
```

**Key Operations:**

| Operation | tmux Command | Notes |
|-----------|-------------|-------|
| Create | `tmux new-session -d -s wc-{id} -x {cols} -y {rows} {shell}` | Detached creation |
| Attach (for I/O) | `tmux attach-session -t wc-{id}` via PTY | Read/Write go through this |
| Resize | `tmux resize-window -t wc-{id} -x {cols} -y {rows}` | Direct tmux command |
| Kill | `tmux kill-session -t wc-{id}` | Clean termination |
| HasChildProcess | Check tmux pane PID's children | Via `tmux display-message -p '#{pane_pid}'` then `/proc/{pid}/task/{pid}/children` |
| ExitCode | Track via tmux `remain-on-exit` + `pane_dead_status` | Set `remain-on-exit on` so we can read exit status |

**Factory Function:**

```go
func tmuxPTYFactory(spec SessionLaunchSpec) (PTY, error) {
    sessionName := "wc-" + spec.SessionID

    // 1. Create detached tmux session with the target shell
    createCmd := exec.Command("tmux", "new-session", "-d",
        "-s", sessionName,
        "-x", strconv.Itoa(int(spec.Cols)),
        "-y", strconv.Itoa(int(spec.Rows)),
        spec.Shell,
    )
    createCmd.Env = buildTmuxEnv(spec) // Same filtering as defaultPTYFactory
    if err := createCmd.Run(); err != nil {
        return nil, fmt.Errorf("tmux new-session: %w", err)
    }

    // 2. Configure remain-on-exit for exit code detection
    exec.Command("tmux", "set-option", "-t", sessionName, "remain-on-exit", "on").Run()

    // 3. Attach to tmux session via a PTY for I/O streaming
    attachCmd := exec.Command("tmux", "attach-session", "-t", sessionName)
    ptmx, err := pty.Start(attachCmd)
    if err != nil {
        exec.Command("tmux", "kill-session", "-t", sessionName).Run()
        return nil, fmt.Errorf("tmux attach: %w", err)
    }

    return &tmuxPTY{
        sessionName: sessionName,
        ptmx:        ptmx,
        cmd:         attachCmd,
    }, nil
}
```

**Discovery Function (for restart recovery):**

```go
// DiscoverTmuxSessions finds surviving web console tmux sessions.
func DiscoverTmuxSessions() ([]string, error) {
    out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
    if err != nil {
        // tmux not running or no sessions — not an error
        return nil, nil
    }
    var sessions []string
    for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
        if strings.HasPrefix(line, "wc-") {
            sessions = append(sessions, strings.TrimPrefix(line, "wc-"))
        }
    }
    return sessions, nil
}
```

**Environment Handling:**

The same `filterClaudeEnv` → `filterServiceEnv` → `ensureTermEnv` → `applySessionEnv` chain from `pty.go` is applied to the tmux session's environment. Factor this into a shared helper:

```go
// buildSessionEnv constructs the filtered environment for a new session.
// Used by both defaultPTYFactory and tmuxPTYFactory.
func buildSessionEnv(spec SessionLaunchSpec) []string {
    return applySessionEnv(
        ensureTermEnv(
            filterServiceEnv(
                filterClaudeEnv(os.Environ()),
            ),
        ),
        spec.Env,
    )
}
```

**Tests:**
- Unit tests with mock tmux commands (exec seam)
- Integration tests that require tmux (build-tagged `//go:build tmux`)
- PTY interface contract tests shared between raw and tmux backends

---

### Phase 4: Session Recovery on Startup (API)

**Goal:** On server start, discover surviving tmux sessions, match against persisted metadata, and re-register them.

**Files to modify:**
- **Modify** `api/session.go` — Add `Recover()` method to SessionManager
- **Modify** `api/main.go` — Call recovery before serving requests

**Recovery Flow:**

```
Server starts
  → Load session metadata from SQLite (all rows where detached=1)
  → Discover live tmux sessions (DiscoverTmuxSessions)
  → For each metadata row:
      → If tmux session exists:
          → Re-attach PTY (tmuxPTYFactory equivalent, but attach-only)
          → Register in SessionManager with original ID, policy, timestamps
          → Start readLoop goroutine
          → Log recovery event
      → If tmux session is gone:
          → Delete stale metadata row
          → Log orphan cleanup event
  → For tmux sessions with no metadata:
      → Kill orphaned tmux session (safety cleanup)
      → Log orphan cleanup event
```

**Re-attach vs Create:**

A recovered session uses a variant of the tmux factory that attaches to an existing session rather than creating one:

```go
func tmuxReattachFactory(sessionName string) PTYFactory {
    return func(spec SessionLaunchSpec) (PTY, error) {
        // Skip tmux new-session — session already exists
        // Just attach for I/O
        attachCmd := exec.Command("tmux", "attach-session", "-t", sessionName)
        ptmx, err := pty.Start(attachCmd)
        if err != nil {
            return nil, fmt.Errorf("tmux reattach %s: %w", sessionName, err)
        }
        return &tmuxPTY{
            sessionName: sessionName,
            ptmx:        ptmx,
            cmd:         attachCmd,
        }, nil
    }
}
```

**SessionManager.Recover():**

```go
func (sm *SessionManager) Recover(store SessionMetadataStore, registry *BackendRegistry) RecoveryReport {
    // 1. Load persisted metadata for detached sessions
    // 2. Discover live tmux sessions
    // 3. Match, re-register, or clean up
    // Returns report: {Recovered: N, OrphanedMetadata: N, OrphanedTmux: N}
}
```

**Tests:**
- Recovery with matching metadata + tmux session → session available
- Recovery with metadata but no tmux session → metadata cleaned up
- Recovery with tmux session but no metadata → tmux session killed
- Recovery with no sessions → no-op
- Recovery idempotency (calling twice doesn't duplicate)

---

### Phase 5: API Contract Updates (API)

**Goal:** Update HTTP handlers to accept and return backend information.

**Files to modify:**
- **Modify** `api/session_handlers.go` — Add `backend` to create request/response
- **Modify** `api/session_handlers_test.go` — Test new field

**Create Session Request:**

```json
POST /api/v1/sessions
{
  "shell": "/bin/bash",
  "cols": 120,
  "rows": 40,
  "backend": "persistent",
  "policy": {
    "mode": "preset",
    "duration": "8h"
  }
}
```

All fields optional. `backend` defaults to server config `WC_DEFAULT_BACKEND`. `policy` defaults to server config default policy (new: `WC_DEFAULT_POLICY_MODE`, `WC_DEFAULT_POLICY_DURATION`).

**Session Response (updated):**

```json
{
  "id": "abc-123",
  "shell": "/bin/bash",
  "created_at": "2026-03-28T10:00:00Z",
  "cols": 120,
  "rows": 40,
  "backend": "persistent",
  "survives_restart": true,
  "policy": {
    "mode": "preset",
    "duration": "8h"
  },
  "busy": false
}
```

**Validation:**
- If `backend` is specified but not available → 422 with recovery hint: "tmux is not installed. Use 'standard' backend or install tmux."
- If `backend` is unknown → 422 with available backends listed

**Policy at Creation:**

Currently policy is set post-creation via `PUT /sessions/{id}/policy`. Add optional `policy` to the create request so both backend and policy can be chosen at launch time. This avoids a race between session creation and policy assignment.

**Tests:**
- Create with backend field → session has correct backend
- Create with unavailable backend → 422 error
- Create with policy → policy applied immediately
- List/Get returns backend and survives_restart fields
- Backward compat: create without backend → uses default

---

### Phase 6: Frontend — API Types and Backend Options (UI)

**Goal:** Update TypeScript types and constants for backend support.

**Files to modify:**
- **Modify** `ui/src/lib/api.ts` — Add backend fields to types and `createSession()` signature
- **Create** `ui/src/consts/backend-options.ts` — Backend option constants (mirrors registry)
- **Modify** `ui/src/consts/policy-options.ts` — No changes needed, but verify cross-language coupling comment

**Updated TypeScript Types:**

```typescript
// api.ts additions

type BackendID = "standard" | "persistent";

interface BackendOption {
  id: BackendID;
  displayName: string;
  description: string;
  survivesRestart: boolean;
  available: boolean;
  reason?: string;
}

interface SessionInfo {
  id: string;
  shell: string;
  created_at: string;
  cols: number;
  rows: number;
  backend: BackendID;
  survives_restart: boolean;
  policy: ExpirationPolicy;
  busy: boolean;
}

interface CreateSessionOpts {
  shell?: string;
  cols?: number;
  rows?: number;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
}

// Updated function signature
function createSession(opts?: CreateSessionOpts): Promise<SessionInfo>;
```

**Backend Options Constants:**

```typescript
// backend-options.ts
// CROSS-LANGUAGE COUPLING: Backend IDs must match BackendID constants in api/backend_registry.go

interface BackendOption {
  id: BackendID;
  label: string;
  description: string;
  survivesRestart: boolean;
}

const BACKEND_OPTIONS: BackendOption[] = [
  {
    id: "standard",
    label: "Standard",
    description: "Lightweight session. Lost if web console restarts.",
    survivesRestart: false,
  },
  {
    id: "persistent",
    label: "Persistent",
    description: "Survives restarts. Ideal for long-running tasks.",
    survivesRestart: true,
  },
];
```

**Tests:**
- `ui/src/consts/backend-options.test.ts` — Option structure validation, label/description non-empty
- `ui/src/lib/api.test.ts` — Update existing createSession tests to cover backend field

---

### Phase 7: Frontend — Launch Dialog Updates (UI)

**Goal:** Add backend and timeout selectors to the terminal launch dialog.

**Files to modify:**
- **Modify** `ui/src/components/TerminalLauncher.tsx` — Add two new selectors

**Design:**

The launch dialog currently shows:
1. Empty terminal button
2. Shortcut buttons
3. Custom command input

Add a **collapsible "Session Options" section** below the launch choices:

```
┌─────────────────────────────────────────┐
│  New Terminal                        [X] │
│                                          │
│  [Empty Terminal]                        │
│  [Claude Code]  [System Info]  [...]    │
│  ┌──────────────────────────────────┐   │
│  │ Custom command...            [Go] │   │
│  └──────────────────────────────────┘   │
│                                          │
│  ▾ Session Options                       │
│  ┌──────────────────────────────────┐   │
│  │ Backend:  [Standard ▾]           │   │
│  │ Timeout:  [Never ▾]             │   │
│  └──────────────────────────────────┘   │
│                                          │
│  ℹ Persistent sessions survive web      │
│    console restarts using tmux.         │
│                                          │
└─────────────────────────────────────────┘
```

**Behavior:**
- Both selectors default to values from settings (fetched via capabilities endpoint)
- Backend selector only shows backends where `available: true`
- If only one backend is available, the selector is hidden (no choice to make)
- Info text updates based on selected backend
- Timeout selector reuses the existing `POLICY_OPTIONS` constant
- `onLaunch` callback signature changes to pass `{ command?, backend?, policy? }`

**Props Update:**

```typescript
interface LaunchOptions {
  command?: string;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
}

interface TerminalLauncherProps {
  open: boolean;
  onClose: () => void;
  onLaunch: (options: LaunchOptions) => void;
  shortcuts?: ShortcutEntry[];
  isCreating?: boolean;
  defaultBackend?: BackendID;
  defaultPolicy?: ExpirationPolicy;
  availableBackends?: BackendOption[];
}
```

**Tests:**
- Renders backend selector when multiple backends available
- Hides backend selector when only one backend available
- Pre-populates selectors from defaults
- Passes selected backend and policy to onLaunch
- Shows appropriate info text per backend
- Disables unavailable backends

---

### Phase 8: Frontend — Settings Page Updates (UI)

**Goal:** Add default backend and default timeout to settings.

**Files to modify:**
- **Modify** `ui/src/components/settings/SessionManagementSection.tsx` — Show backend per session, add defaults section

**Design:**

Add a "Session Defaults" subsection at the top of session management:

```
Session Defaults
┌─────────────────────────────────────────┐
│ Default backend:  [Standard ▾]          │
│ Default timeout:  [Never ▾]             │
│                                          │
│ These defaults pre-populate the launch   │
│ dialog. You can override per session.    │
└─────────────────────────────────────────┘
```

Per-session display updates to show the backend type:

```
Session: "my-terminal"
Backend: Persistent  |  Timeout: 8h (2h 15m remaining)
[Rename] [Focus] [Terminate]
```

**Persistence:** Defaults are stored in the workspace settings (existing SQLite workspace tables or a new `settings` key-value table).

**Settings Store API:**

```json
GET /api/v1/settings/session-defaults
{
  "default_backend": "standard",
  "default_policy": { "mode": "never" }
}

PUT /api/v1/settings/session-defaults
{
  "default_backend": "persistent",
  "default_policy": { "mode": "preset", "duration": "8h" }
}
```

**Tests:**
- Renders backend badge per session
- Renders defaults section
- Changing default backend persists and reflects in launch dialog
- Changing default policy persists and reflects in launch dialog

---

### Phase 9: Frontend — Reconnection to Recovered Sessions (UI)

**Goal:** Ensure recovered sessions reconnect seamlessly after restart.

**Files to modify:**
- **Modify** `ui/src/hooks/useSessionManager.ts` — Handle recovered sessions during hydration

**Current Hydration Flow:**
1. `listSessions()` → get all active sessions from server
2. `getWorkspaceLayout()` → get pane metadata from server
3. Match panes to sessions, filter out orphaned panes

**After This Change:**
The flow is identical. The key insight is that **no UI changes are needed for reconnection** — the existing hydration already handles this. When the server recovers tmux sessions at startup, they appear in `listSessions()` and the workspace layout is already in SQLite. The frontend reconnects via the normal WebSocket path.

The only addition:
- Show a brief toast/banner: "Reconnected to N persistent session(s)" when recovered sessions are detected during hydration
- A "recovered" badge on sessions that were re-attached (transient, clears after first interaction)

**How to detect recovered sessions:**
- Add `recovered: boolean` to the session response (true for sessions restored from tmux, false for newly created ones; resets to false after first WebSocket connection)
- Or: compare session `created_at` against server `started_at` — if session is older than server uptime, it was recovered

**Tests:**
- Hydration with recovered sessions shows toast
- Recovered badge appears and clears
- WebSocket connects normally to recovered sessions
- Pane metadata is preserved across restart

---

### Phase 10: Documentation and Cleanup

**Goal:** Update all documentation to reflect the new architecture.

**Files to modify:**
- **Modify** `docs/internal/SEAMS.md` — Document backend registry seam, tmux factory seam, session store seam, discovery seam
- **Modify** `docs/internal/PROGRESS.md` — Add phase entry
- **Modify** `docs/reference/api-endpoints.md` — Document updated create request, capabilities response, settings endpoints
- **Modify** `docs/reference/configuration.md` — Document `WC_DEFAULT_BACKEND`, `WC_DEFAULT_POLICY_MODE`, `WC_DEFAULT_POLICY_DURATION`
- **Modify** `docs/concepts/ARCHITECTURE.md` — Add backend registry to architecture diagram
- **Modify** `docs/internal/PROBLEMS.md` — Close "sessions lost on restart" if it exists, note tmux dependency
- **Verify** all `// DOC:` and `[CODE:]` bidirectional references

## 8. Contract Decisions

### API Contracts

| Endpoint | Change | Backward Compatible? |
|----------|--------|---------------------|
| `POST /api/v1/sessions` | Add optional `backend` and `policy` fields | Yes — omission uses defaults |
| `GET /api/v1/sessions` | Response includes `backend`, `survives_restart` | Yes — additive fields |
| `GET /api/v1/sessions/{id}` | Same as above | Yes |
| `GET /api/v1/capabilities` | Add `session_backends` and `default_backend` | Yes — additive |
| `GET /api/v1/settings/session-defaults` | New endpoint | N/A |
| `PUT /api/v1/settings/session-defaults` | New endpoint | N/A |

### Data Model Contracts

| Table | Change |
|-------|--------|
| `sessions` (new) | Metadata for restart recovery |
| `settings` (new or extend workspace) | Key-value for session defaults |

### CLI Contracts

No CLI changes required.

### Cross-Language Coupling Points

| Concept | Go Location | TS Location | Coupling |
|---------|-------------|-------------|----------|
| Backend IDs | `api/backend_registry.go` | `ui/src/consts/backend-options.ts` | Exact string match |
| Policy modes | `api/session_policy.go` | `ui/src/consts/policy-options.ts` | Existing (unchanged) |
| Preset durations | `api/session_policy.go` | `ui/src/consts/policy-options.ts` | Existing (unchanged) |

## 9. Testing Plan

### Unit Tests (per phase)

| Phase | Test File | Key Cases |
|-------|-----------|-----------|
| 1 | `api/backend_registry_test.go` | Registration, lookup, availability, factory dispatch |
| 2 | `api/session_store_test.go` | CRUD, concurrent access, policy updates |
| 3 | `api/pty_tmux_test.go` | Create, read, write, resize, kill, exit code, child detection |
| 3 | `api/pty_contract_test.go` | Shared interface contract tests run against both backends |
| 4 | `api/session_recovery_test.go` | Match, orphan cleanup, idempotency |
| 5 | `api/session_handlers_test.go` | Backend field in create/response, validation |
| 6 | `ui/src/consts/backend-options.test.ts` | Option structure |
| 6 | `ui/src/lib/api.test.ts` | createSession with backend |
| 7 | `ui/src/components/TerminalLauncher.test.tsx` | Selector rendering, defaults, callbacks |
| 8 | `ui/src/components/settings/SessionManagementSection.test.tsx` | Defaults section, backend badge |
| 9 | `ui/src/hooks/useSessionManager.test.ts` | Recovery hydration, toast |

### Integration Tests

- **tmux integration** (build-tagged `//go:build tmux`): Full lifecycle — create tmux session, read/write, kill server (simulated), recover, verify session alive
- **HTTP integration**: Create persistent session → list → verify backend field → delete → verify tmux session cleaned up

### Test Seams

| Seam | Purpose | Implementation |
|------|---------|----------------|
| `PTYFactory` injection | Test tmux factory in isolation | Existing seam, extended |
| `BackendRegistry` injection | Test with/without tmux available | Mock availability check |
| `SessionMetadataStore` interface | Test recovery without SQLite | In-memory implementation |
| `DiscoverTmuxSessions` function | Test recovery with controlled tmux state | Injection or exec mock |
| tmux exec commands | Test without real tmux | `execCommand` variable seam (like `pty.go` pattern) |

### What Not to Test

- tmux's own behavior (it's battle-tested software)
- Raw PTY behavior (already tested; this plan only adds, doesn't change it)
- WebSocket reconnection (already tested and unchanged)

## 10. Rollout / Validation Checklist

1. [ ] `go build ./...` succeeds in `scenarios/web-console/api/`
2. [ ] `go test ./... -timeout 300s` passes (all existing + new tests)
3. [ ] `CGO_ENABLED=0 go build ./...` still works (no CGO deps introduced)
4. [ ] `gofumpt -l .` reports no formatting issues
5. [ ] `golangci-lint run` passes
6. [ ] UI builds: `cd ui && npm run build` succeeds
7. [ ] UI tests: `cd ui && npm test` passes (all existing + new tests)
8. [ ] `cd scenarios/web-console && make test` passes end-to-end
9. [ ] Manual smoke test: create standard session → works as before
10. [ ] Manual smoke test: create persistent session (if tmux installed) → survives `make stop && make start`
11. [ ] Manual smoke test: launch dialog shows correct defaults from settings
12. [ ] Manual smoke test: settings page shows backend per session
13. [ ] Capabilities endpoint returns correct backend availability
14. [ ] Documentation updated and bidirectional references valid
15. [ ] No dead code, no compatibility shims, no legacy patterns

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| tmux not installed on target system | Medium | Low — standard backend still works | Registry reports availability; UI hides unavailable backends; clear error message |
| tmux version incompatibility | Low | Medium | Version check in availability probe; document minimum version (2.6+) |
| Orphaned tmux sessions accumulate | Low | Low — resource waste | Discovery cleanup on startup; expiration sweeper applies to tmux sessions too |
| Session metadata out of sync with tmux | Low | Medium — phantom sessions | Startup reconciliation resolves all discrepancies; metadata is secondary to tmux reality |
| tmux attach I/O performance differs from raw PTY | Low | Low | Same `creack/pty` wrapping; tmux adds minimal overhead |
| Race between server shutdown and tmux session creation | Very Low | Low | tmux sessions are created detached first, then attached — creation is atomic |
| Multiple web console instances fighting over same tmux sessions | Low | High | Session ID in tmux name prevents collision; startup recovery is idempotent |

## 12. Non-Goals / Prohibited Patterns

- **No live migration** — Converting a running raw PTY session to tmux is not possible at the OS level. Do not attempt fake migration (spawn new + copy scrollback) as it breaks running processes. This is a launch-time choice only.
- **No compatibility shims** — This is greenfield. Don't add code to handle "old sessions without backend field" or "sessions created before this feature." The schema has a default, and that's sufficient.
- **No ad-hoc tmux installation** — tmux is registered in `scripts/lib/system/common_deps.sh` and installed via `vrooli setup`. The implementing agent must not install tmux manually or add scenario-local installation logic.
- **No custom session multiplexer** — Don't reimplement tmux. Use tmux directly.
- **No screen support** — tmux only. Screen is legacy and lacks programmatic control. The registry supports adding backends later if needed.
- **No desktop tier concerns** — tmux is inherently a server-side tool. Desktop bundles would use standard sessions. Don't add complexity for hypothetical desktop persistent sessions.
- **No changes to WebSocket transport** — The existing WebSocket layer works as-is for both backends. Don't touch it.
- **No dead code** — Don't leave unused types, commented-out code, or "TODO: remove" markers.

## 13. Definition of Done

All of the following must be true:

1. **Backend registry** exists with standard and persistent (tmux) backends registered, with runtime availability detection.
2. **tmux PTY** implements the full `PTY` interface and passes shared contract tests.
3. **Session metadata** is persisted in SQLite for detached sessions.
4. **Startup recovery** discovers surviving tmux sessions, matches metadata, and re-registers them. Orphans are cleaned up.
5. **API contracts** accept `backend` and `policy` at creation, return them in all session responses, and expose backends via capabilities.
6. **Launch dialog** presents backend and timeout selectors, pre-populated from settings-configured defaults.
7. **Settings page** allows configuring default backend and default timeout, and displays backend per session.
8. **Frontend reconnection** works seamlessly — recovered sessions appear in the pane bar after restart.
9. **All tests pass** — existing tests unbroken, new tests cover all phases.
10. **Documentation is updated** — SEAMS.md, PROGRESS.md, api-endpoints.md, configuration.md, ARCHITECTURE.md.
11. **No dead code, no compatibility shims, no legacy patterns.**
12. **`make test` passes** for the full scenario.
