# Agent Session Delete Implementation Plan

## Purpose

Add a professional, validated delete feature for Swarm Manager agent sessions from the session details page and supporting API surfaces. The user must be able to delete the current session after an explicit confirmation. Deleting a session removes the session record, transcript, session-owned proposal drafts, and session-owned artifact link records. It must not delete separate backlog items, initiatives, captures, operating-mode definitions, or files that were created from the session.

## Required Reading

Run these before implementation:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement test react-coherence ux api-steer cli-steer interoperability-steer
```

Discovery performed for this plan:

```bash
prompt-manager discover "swarm manager session delete" "destructive action confirmation UI" "react session details page" "api store mutation testing" --complexity moderate
```

The discovery result initially recommended `documentation-health seam-discovery-and-enforcement test scientific-debugging react-coherence ux`; this plan adds `implementation-plan-authoring`, `api-steer`, `cli-steer`, and `interoperability-steer` because this feature introduces a destructive API capability and should not leave API/CLI/contract parity ambiguous.

## Greenfield Constraint

This is greenfield work. Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. Implement the current intended behavior cleanly across the canonical session seams.

## Problem Statement

The session details page currently supports refresh, cancel, continue, proposal apply, and artifact navigation, but it has no way to remove a session. Users can accumulate test or obsolete sessions and have no cleanup affordance from the page where they are reviewing the conversation.

The current action layout also matters:

- Desktop already has refresh/cancel actions in the header action row, with mobile actions collapsed behind a header ellipsis menu.
- The delete action is destructive and should be discoverable but not placed as a primary inline button next to routine actions.
- Mobile should keep all header actions in the ellipsis menu.
- Session-owned proposals and artifact links are stored inside the session folder, but created entities are independent domain records and should be preserved by default.

## Scope

In scope:

- Add a typed delete API contract for agent sessions.
- Add a `DELETE /api/v1/agent-sessions/{session_id}` handler using proto JSON response semantics consistent with existing agent-session endpoints.
- Add service and store delete operations with tests.
- Stop an active agent run before deleting its session; if stopping fails, do not delete the session.
- Add UI service/store support for session deletion.
- Add session details UX:
  - Desktop: add an ellipsis action menu beside refresh/cancel and put `Delete session` inside it.
  - Mobile: add `Delete session` to the existing header ellipsis bottom sheet.
  - Confirmation dialog must clearly explain what is removed and what remains.
  - On success, navigate away from the deleted session route and remove the session from in-memory/persisted session lists.
- Add high-signal unit tests across API/store/service/UI.
- Add CLI parity or explicitly document and implement a small CLI command if the CLI command surface can be kept narrow.
- Update internal seam/coherence/interop documentation for the new delete seam.

Out of scope:

- Cascading deletion of created backlog items, initiatives, captures, operating-mode definitions, files, or other independent artifacts.
- A checkbox to delete created outputs. This is intentionally deferred because it crosses multiple domain stores, would require per-artifact permission and cascade semantics, and could destroy useful work. The first implementation should preserve outputs and state that explicitly in the confirmation.
- Archive/restore for sessions.
- Bulk session deletion.
- Changing session list filtering, session status taxonomy, or stats semantics beyond removing deleted sessions from normal list results.
- Migrating the whole agent-session API to Connect-RPC in this feature. Existing agent-session endpoints are proto-owned REST handlers; keep that boundary coherent and document it.

## Current Technical Context

Key backend files:

- `packages/proto/schemas/swarm-manager/v1/api/agent_session.proto`
  - Defines proto messages for list/get/create/continue/refresh/cancel/apply/artifact lookup.
  - There is currently no delete request/response.
- `scenarios/swarm-manager/api/internal/agentsessions/handler.go`
  - Registers REST routes under `/api/v1/agent-sessions`.
  - Existing lifecycle commands are `POST /continue`, `POST /refresh`, and `POST /cancel`.
- `scenarios/swarm-manager/api/internal/agentsessions/service.go`
  - Owns create/list/get/continue/refresh/cancel/proposal/artifact operations.
  - Cancels active sessions by calling `SessionSpawner.StopRun`.
  - Emits lifecycle events for create/start/continue/fail/cancel/complete/proposal/artifact.
- `scenarios/swarm-manager/api/internal/agentsessions/store.go`
  - Stores sessions under `agent-sessions/<session_id>/`.
  - A session folder contains `session.json`, `messages.jsonl`, `artifacts.jsonl`, and `proposals/*.json`.
  - `LoadSession` hydrates the session-owned proposal and artifact link records from files.
- `scenarios/swarm-manager/api/internal/agentsessions/*_test.go`
  - Existing tests cover proto JSON handler contracts, persistence, lifecycle, and proposal apply behavior.

Key UI files:

- `scenarios/swarm-manager/ui/src/pages/SessionDetailsPage.tsx`
  - Owns route/store orchestration and assembles header actions, mobile action sheet, conversation, and inspector/tabs.
  - Currently exposes desktop refresh/cancel directly and mobile refresh/cancel in a `BottomSheet`.
- `scenarios/swarm-manager/ui/src/services/agent-session-service.ts`
  - Central UI API service for agent sessions.
  - Has no `delete` method.
- `scenarios/swarm-manager/ui/src/stores/agent-session-store.ts`
  - Zustand store for sessions and artifact-by-entity cache.
  - Has no delete mutation and does not remove persisted sessions after delete.
- `scenarios/swarm-manager/ui/src/components/ui/confirm-dialog.tsx`
  - Existing destructive confirmation primitive with optional typed confirmation and checkbox support.
- `scenarios/swarm-manager/ui/src/pages/SessionDetailsPage.test.tsx`
  - Existing focused page tests cover desktop/mobile action placement, composer behavior, refresh/cancel, apply, and artifact navigation.

CLI context:

- `scenarios/swarm-manager/cli/app.go`
  - Registers CLI dependency commands.
- `scenarios/swarm-manager/cli/internal/support/dependencies.go`
  - Defines command function slots.
- `scenarios/swarm-manager/cli/domains/domains.go`
  - Registers domain subcommand groups.
- There is currently no agent-session CLI domain. Existing CLI only exposes session stats, not session management.

## Target End State

1. API supports deleting exactly one session by ID.
2. Deleting an active session attempts to stop its running Agent Manager run before deleting storage.
3. If the session does not exist, API returns the standard mapped not-found response.
4. If `StopRun` fails for an active session, API returns an error and leaves session storage intact.
5. Store deletion removes only the session folder and cannot path-traverse outside the `agent-sessions` root.
6. UI delete removes the session from the local store and persisted cache, then navigates away from `/sessions/:sessionId`.
7. Desktop has an ellipsis menu beside refresh/cancel with `Delete session` as the destructive action.
8. Mobile keeps refresh/cancel/delete inside the existing header ellipsis sheet.
9. Confirmation copy is explicit:
   - removed: conversation, session metadata, proposal drafts, artifact links;
   - preserved: created backlog items, initiatives, captures, operating-mode definitions, files, and agent activity records.
10. The delete action is disabled while any session mutation is in progress.
11. Tests fail if the API, store, UI service, UI store, desktop menu, mobile menu, confirmation, success navigation, or failure behavior regresses.

## Implementation Strategy

### Phase 1: Contract

Update `packages/proto/schemas/swarm-manager/v1/api/agent_session.proto`:

- Add `DeleteAgentSessionRequest`:
  - `string session_id = 1 [(buf.validate.field).string.min_len = 1];`
- Add `DeleteAgentSessionResponse`:
  - `string session_id = 1;`

Do not add a `delete_outputs` or cascade field in this iteration.

Run:

```bash
cd packages/proto
make generate
make lint
make check
```

If `make breaking` is available against the configured baseline branch, run it too. If it cannot run because the baseline branch is unavailable, record that in validation notes.

### Phase 2: Backend Store Seam

Update `scenarios/swarm-manager/api/internal/agentsessions/store.go`:

- Extend `Store` with:
  - `DeleteSession(sessionID string) error`
- Add a focused helper for safe session IDs before deletion:
  - trim whitespace;
  - reject empty IDs;
  - reject path separators and `.`/`..`;
  - require the current session ID prefix shape (`sess_`) unless an existing shared ID validator is found.
- Implement `FileStore.DeleteSession`:
  - acquire the store mutex;
  - load the session first to enforce existence and validation;
  - remove the exact session directory with `os.RemoveAll`;
  - map missing session to `ErrNotFound`.

Tests in `store_test.go`:

- Deleting a session removes `session.json`, `messages.jsonl`, `artifacts.jsonl`, and proposal files.
- Loading/listing the deleted session returns not found / omits it.
- Deleting a missing session returns `ErrNotFound`.
- Unsafe IDs cannot remove files outside `agent-sessions`.

### Phase 3: Backend Service

Update `service.go`:

- Add `Delete(ctx context.Context, sessionID string) error`.
- Load the session before deletion.
- If the session is active and has a `RunID` and `spawner != nil`, call `StopRun`.
- If `StopRun` returns an error, return `mapSpawnError(err)` and do not delete.
- Call `store.DeleteSession`.
- Add a delete event only if eventlog/stats should count deletes. Recommended:
  - add `EventAgentSessionDeleted` and `EmitAgentSessionDeleted` only for auditability;
  - do not subtract historical stats counters, because stats are event history.

Tests in `service_test.go`:

- Deleting an active session calls `StopRun` before storage removal.
- Stop failure leaves the session loadable.
- Deleting a terminal session removes storage without calling `StopRun`.
- Missing session maps to not found.

### Phase 4: Backend Handler

Update `handler.go`:

- Register:
  - `DELETE /api/v1/agent-sessions/{session_id}`
- Implement `Delete`:
  - set `req.SessionId` from mux vars;
  - validate with `httputil.ValidateProtoRequest`;
  - call `service.Delete`;
  - return `DeleteAgentSessionResponse{SessionId: req.SessionId}` as proto JSON.

Tests in `handler_test.go`:

- Delete endpoint returns 200 and proto field name `session_id`.
- Subsequent get returns 404.
- Active session delete stops the run through the service fake/spawner path.
- Missing delete returns 404.
- Invalid blank/path traversal ID returns 400.

### Phase 5: UI Service and Store

Update `ui/src/lib/api-endpoints.ts`:

- Reuse `agentSessionById(sessionId)` for delete or add a clear alias if useful.

Update `ui/src/services/proto/agent-session-contracts.ts`:

- Import `DeleteAgentSessionResponseSchema`.
- Export `deleteAgentSessionResponseSchema`.

Update `ui/src/services/agent-session-service.ts`:

- Add `delete(sessionId: string): Promise<string>` to `IAgentSessionService`.
- Call `apiClient.delete(API_ENDPOINTS.agentSessionById(sessionId))`.
- Parse `DeleteAgentSessionResponse` and return the deleted `sessionId`.

Update `ui/src/stores/agent-session-store.ts`:

- Add `deleteSession(sessionId: string): Promise<void>`.
- Set `isMutating` and clear `error`.
- On success:
  - remove the session from `sessions`;
  - remove cached `artifactsByEntity` entries whose artifacts only reference the deleted session, or recompute affected entries by filtering artifacts with that `sessionId`;
  - update localStorage with the new session list and current timestamp.
- On failure:
  - set a clear delete-specific error;
  - preserve existing session data.

Tests:

- `agent-session-service.test.ts` verifies `DELETE /agent-sessions/:id` and response parsing.
- `agent-session-store.test.ts` verifies successful delete removes the session and persisted cache, while failure preserves data and clears `isMutating`.

### Phase 6: Session Details UX

Use existing primitives rather than building a new modal/menu framework.

Recommended component extraction:

- Add `ui/src/components/session/SessionActionsMenu.tsx` for the shared action list used by desktop popover/menu and mobile sheet content.
- Add `ui/src/components/session/SessionDeleteDialog.tsx` as a thin session-specific adapter around `ConfirmDialog`.

Desktop:

- Keep refresh/cancel inline exactly where they are.
- Add a ghost icon button with `MoreVertical` beside refresh/cancel.
- Menu contents:
  - `Delete session` with `Trash2`, destructive styling.
- Opening delete closes the menu and opens the confirmation dialog.

Mobile:

- Keep the existing header ellipsis.
- Add `Delete session` below refresh/cancel in the bottom sheet.
- Destructive item should use `Trash2` and red/destructive text styling.

Confirmation dialog:

- Title: `Delete Session`
- Confirm label: `Delete Session`
- Strong confirmation text: use the session title when non-empty, otherwise the session ID.
- Description must be concise but explicit:
  - "This removes the conversation, session details, proposal drafts, and session artifact links. Created backlog items, initiatives, captures, files, and agent activity records stay in Swarm Manager."
- No checkbox for deleting created outputs in this implementation.

Success behavior:

- Call `deleteSession(session.id)`.
- Close dialog.
- Navigate away via `closeDetail()` or to the sessions list route. Prefer the same route/back behavior used for not-found fallback unless a canonical sessions-list path exists.

Failure behavior:

- Keep the dialog open or close it only if the existing pattern expects that; either way, show a visible `role="alert"` error and preserve the session page.
- Do not clear draft/composer state on delete failure.

Tests in `SessionDetailsPage.test.tsx`:

- Desktop refresh/cancel remain inline.
- Desktop has a header ellipsis action button.
- Desktop delete action is only inside the ellipsis menu.
- Mobile delete action appears inside the header ellipsis sheet.
- Confirmation dialog shows the preservation warning.
- Confirm button is disabled until strong confirmation text is typed.
- Successful delete calls store delete and navigates away.
- Delete failure shows an alert and does not navigate.
- Delete action is disabled while `isMutating` is true.

### Phase 7: CLI Parity

Because `cli-steer` expects API capabilities to be available through the CLI, implement the minimal session command group unless the team explicitly decides session management is UI-only.

Recommended small CLI scope:

- Add `sessions` subcommand group:
  - `sessions list`
  - `sessions get <session_id>`
  - `sessions delete <session_id> [--yes]`
- Use the same REST/proto JSON API style as the existing CLI commands in this scenario.
- Keep business logic in the API; the CLI only parses args, prompts/guards destructive delete, calls API, and formats output.
- If implementing only delete feels too narrow, still create the `sessions` group with list/get/delete so the command is discoverable and useful.

Tests:

- CLI route registration includes `sessions`.
- `sessions delete sess_1 --yes` calls `DELETE /agent-sessions/sess_1`.
- Without `--yes`, the command requires confirmation or returns an actionable message in non-interactive mode, matching existing CLI destructive-command patterns.

If CLI work is intentionally deferred, add an explicit `docs/internal/API_NOTES.md` or plan addendum entry explaining why API/UI parity is acceptable without CLI parity for this session-only UX feature. The preferred implementation is to include the CLI.

### Phase 8: Documentation

Update:

- `scenarios/swarm-manager/docs/internal/SEAMS.md`
  - Add the session delete seam: UI action -> store mutation -> UI service -> API handler -> service -> file store.
  - Document that session deletion does not cascade to created entities.
- `scenarios/swarm-manager/docs/internal/COHERENCE-NOTES.md`
  - Record the shared session actions/delete dialog components and the desktop/mobile action placement.
- `scenarios/swarm-manager/docs/internal/INTEROP_AUDIT.md`
  - Create or update with the proto-owned REST exception status for agent sessions and the new delete contract.

## Contract Decisions

- Endpoint: `DELETE /api/v1/agent-sessions/{session_id}`
- Request contract: `DeleteAgentSessionRequest`
- Response contract: `DeleteAgentSessionResponse { session_id }`
- Destructive semantics:
  - delete session storage;
  - delete transcript/proposal/artifact-link files as part of that storage;
  - preserve created external entities.
- Active-run semantics:
  - if active with a run ID, stop the run first;
  - if stop fails, deletion fails and storage remains.
- Idempotency:
  - first delete succeeds;
  - repeated delete returns not found.
- Validation:
  - reject blank and path-unsafe IDs.

## Testing Plan

Focused backend:

```bash
cd scenarios/swarm-manager
go test ./api/internal/agentsessions ./api/internal/eventlog ./api/internal/stats
```

Focused UI:

```bash
cd scenarios/swarm-manager/ui
pnpm run type-check
pnpm exec vitest run \
  src/pages/SessionDetailsPage.test.tsx \
  src/services/agent-session-service.test.ts \
  src/stores/agent-session-store.test.ts \
  --minWorkers=1 --maxWorkers=1
pnpm run lint
```

Proto:

```bash
cd packages/proto
make generate
make lint
make check
```

Scenario:

```bash
cd scenarios/swarm-manager
make test
make start
make logs
make stop
```

Known caveat from recent validation: full `make test` previously failed on existing scenario-wide CLI/proto/go.sum and standards blockers unrelated to session UI work. Re-run it anyway after implementation and record exact failures. Fix all issues in modified files.

## Rollout and Validation Checklist

- [ ] Proto schema includes delete request/response.
- [ ] Generated Go and TypeScript proto artifacts are updated.
- [ ] API route is registered and handler uses proto JSON response.
- [ ] Service stops active run before delete.
- [ ] Store delete is path-safe and removes only the session directory.
- [ ] UI service calls the delete endpoint and parses the response.
- [ ] UI store removes deleted sessions from memory and persisted cache.
- [ ] Desktop action layout has inline refresh/cancel plus header ellipsis delete.
- [ ] Mobile action sheet contains refresh/cancel/delete.
- [ ] Confirmation dialog explains exactly what is removed and preserved.
- [ ] Successful delete navigates away from the deleted route.
- [ ] Failed delete keeps the page usable and shows an alert.
- [ ] CLI parity is implemented or explicitly documented as deferred.
- [ ] Internal seam/coherence/interop docs are updated.
- [ ] Focused tests pass.
- [ ] Scenario lifecycle has been run and health checked.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---:|---|
| Deleting session storage while agent run continues | High | Service must stop active run before `DeleteSession`; abort delete if stop fails. |
| Path traversal in `RemoveAll` | High | Validate session IDs before building deletion path; load session first; reject separators and dot segments. |
| User expects created outputs to be deleted | Medium | Confirmation copy states created outputs remain; no cascade checkbox in first implementation. |
| CLI/API parity drifts | Medium | Add `sessions` CLI group with delete or explicitly document deferral. |
| UI route keeps showing stale deleted session from localStorage | Medium | Store delete must update persisted cache after removing the session. |
| Proto generated artifacts get out of sync | Medium | Run `packages/proto make generate/lint/check`. |
| Delete action appears too primary | Low | Put delete in ellipsis menu on desktop and mobile, with destructive styling. |

## Non-Goals and Prohibited Patterns

- Do not use `window.confirm`.
- Do not add a bottom floating action button on the session details page.
- Do not cascade-delete created backlog items, initiatives, captures, files, or agent activity records.
- Do not implement delete by only hiding sessions in the UI.
- Do not bypass the agent-session store from `SessionDetailsPage`.
- Do not call `os.RemoveAll` on a path derived from unvalidated user input.
- Do not edit generated proto files manually.
- Do not leave a dead or duplicate session action component after extraction.

## Cleanup and Health Verification

Final implementation must:

1. Fix all lint, type, and unit test issues in modified files, including pre-existing issues in those modified files.
2. Run focused backend, UI, and proto validation.
3. Run `cd scenarios/swarm-manager && make test`.
4. Restart and verify the scenario:

```bash
cd scenarios/swarm-manager
make stop
make start
make logs
make stop
```

If a health endpoint is available on the allocated scenario API port, verify it with `curl -s <base-url>/api/v1/health` or the scenario's documented health route. Record any scenario-wide failures that are unrelated to modified files, but do not ignore failures in touched files.

## Definition of Done

- A user can delete a session from the session details page on desktop and mobile.
- The user must explicitly confirm the destructive action.
- The UI clearly communicates that created outputs remain.
- The backend removes the session folder and no other domain data.
- Active agent runs are not orphaned silently.
- API, store, service, UI service, UI store, page UX, and CLI parity are covered by tests.
- Generated proto artifacts are current.
- Internal docs describe the new delete seam and interop posture.
- Focused validation passes, and scenario-level validation has been run with results recorded.
