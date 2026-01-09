# No-Lock Sandboxes (Workspace-Sandbox) - Implementation Plan

## Context
- Reservations are for semantic coordination (avoid concurrent agents working on overlapping code paths), not filesystem conflicts.
- Investigations/read-only workflows should be able to run in a sandbox without reserving paths, while still using a real project root and mount scope.
- Current create flow always derives reserved paths (defaults to scope) and enforces overlap checks.
- UI currently requires reserved paths and assumes at least one reserved path exists for display.

## Goals
- Support explicit "no-lock" sandbox creation while preserving existing behavior by default.
- Keep projectRoot and scopePath requirements intact.
- Skip overlap checks for lockless sandboxes.
- Surface a clear UX warning for no-lock runs.
- Preserve diff and approval behavior (changes outside reserved paths are already handled as "outside scope" in UI, which is acceptable for this use-case).

## Current Behavior References
- Reservation derivation and overlap checks: `scenarios/workspace-sandbox/api/internal/sandbox/service.go`.
- DB function `check_scope_overlap`: `scenarios/workspace-sandbox/initialization/postgres/schema.sql` and migration in `scenarios/workspace-sandbox/api/main.go`.
- UI Create Sandbox modal requires reserved paths: `scenarios/workspace-sandbox/ui/src/components/CreateSandboxDialog.tsx`.
- API/Types: `scenarios/workspace-sandbox/api/internal/types/types.go`.
- Repository hydration for reserved paths: `scenarios/workspace-sandbox/api/internal/repository/sandbox_repo.go`.

## Plan of Action

### 1) API & Domain: Add explicit no-lock flag
- Add `noLock` (or `lockless`) boolean to:
  - `CreateRequest` (API input)
  - `Sandbox` (stored state)
- Default: `false` to preserve current behavior.

Files:
- `scenarios/workspace-sandbox/api/internal/types/types.go`

### 2) Validation & Reservation logic
- Update `validateCreateRequest()` to:
  - Always require `projectRoot` and valid `scopePath` (unchanged).
  - If `noLock == true`:
    - Allow empty reserved paths.
    - Skip overlap checks.
  - If `noLock == false`:
    - Keep current defaulting behavior (reserved paths fall back to scope).
    - Keep overlap checks.
- Ensure `createAndMountSandbox()` handles empty reserved paths for no-lock sandboxes:
  - Store `ReservedPaths` as empty slice when noLock.
  - `ReservedPath` can be empty or a placeholder, but should not be auto-filled when noLock.

Files:
- `scenarios/workspace-sandbox/api/internal/sandbox/service.go`

### 3) DB Schema + Overlap Function
- Add `no_lock BOOLEAN NOT NULL DEFAULT false` to sandboxes.
- Ensure backfills do not force reserved_path(s) for lockless sandboxes.
- Update `check_scope_overlap` to ignore `no_lock = true` sandboxes.

Files:
- `scenarios/workspace-sandbox/initialization/postgres/schema.sql`
- `scenarios/workspace-sandbox/initialization/postgres/seed.sql`
- `scenarios/workspace-sandbox/api/main.go` (migration + function update)

### 4) Repository hydration behavior
- Avoid re-deriving reserved paths when `no_lock` is true.
- For lockless sandboxes, reserved paths should remain empty so UI can render the correct "No lock" state.

Files:
- `scenarios/workspace-sandbox/api/internal/repository/sandbox_repo.go`

### 5) UI: Create Sandbox modal UX
- Keep reserved paths required by default.
- Add checkbox: "No-lock run (read-only)".
  - Warning copy: "Use only when you plan to discard all changes. Concurrency protection is disabled."
- When checked:
  - Allow submit with zero reserved paths.
  - Disable reserved path inputs or mark as optional.
  - Send `noLock: true` in create request.

Files:
- `scenarios/workspace-sandbox/ui/src/components/CreateSandboxDialog.tsx`
- `scenarios/workspace-sandbox/ui/src/lib/api.ts`

### 6) UI: Display reserved path state
- If sandbox has `noLock == true` and no reserved paths, display "No lock" instead of "Not specified" or "/".

Files:
- `scenarios/workspace-sandbox/ui/src/components/SandboxList.tsx`
- `scenarios/workspace-sandbox/ui/src/components/SandboxDetail.tsx`

### 7) Docs & Tests
- Document the no-lock mode and intended usage.
- Update UI tests if they assume reserved paths are required.

Files:
- `scenarios/workspace-sandbox/README.md`
- `scenarios/workspace-sandbox/docs/ARCHITECTURE.md`
- `scenarios/workspace-sandbox/bas/cases/sandbox-lifecycle/ui/create_sandbox_dialog.json` (if needed)

## UX Notes
- Default behavior remains unchanged: reserved paths required and enforced.
- No-lock is an explicit opt-in with warning text.
- The change is safe because it only removes semantic coordination when explicitly requested.

## Open Questions
- Exact field name (`noLock` vs `lockless`) — recommend `noLock` for clarity.
- Whether to set `ReservedPath` to an empty string or omit it for lockless sandboxes; prefer empty and handle it in UI display.

