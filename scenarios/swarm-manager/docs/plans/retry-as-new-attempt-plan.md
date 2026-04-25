# Retry-as-New-Attempt for Backlog & Initiative Items

## 1. Purpose

Add a first-class **retry** action that re-dispatches a terminal-state backlog or initiative item against the **same scope** as a **new execution attempt**, while preserving the prior failed/completed attempt's full record (logs, finalization, dispatch row, outcome). This closes the gap where a user can only "follow up" (which derives new scope) when what they actually want is "run that again, the world changed but the work didn't."

Triggering example: an item ran, post-run analysis correctly flagged failure, user identified an environmental bug (e.g., agent-manager hesitation) and fixed it. The work to be done has not changed; only the environment has. A clean retry is the right action.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer utils-unification seam-discovery-and-enforcement idempotency-replay-safety-hardening
```

## 3. Greenfield Constraint

**This is greenfield work.** The existing `execution.Retry()` at `api/internal/execution/service_control.go:260-272` performs an in-place mutation (StatusFailed → StatusPending on the same row), which is incompatible with the new-attempt model. We **replace** it; we do not add a parallel API. No compatibility shims, no `RetryV2`, no `// kept for backwards-compat`. The only existing callers are the HTTP handler and the CLI command (both in this repo and updated in lockstep). External consumers are not in scope.

## 4. Problem Statement

Today, after an item reaches a terminal state, the user has two options:

- **Follow-up** (`POST /api/v1/execution/{id}/follow-up`) — creates a derived run with new scope/feedback context. Wrong tool when no scope change is wanted.
- **Retry (in-place)** (`POST /api/v1/execution/{id}/retry`) — destructively mutates the failed row back to pending and re-dispatches. Destroys audit history, has no UI surface, and is gated only on `StatusFailed` at the execution level.

Neither addresses the canonical "re-run this item, same scope, the prior attempt was an environmental no-op" workflow. Additionally:

- **Item-level terminal states are unreachable for retry.** When `review-decide` flips the *item* to `StatusCompleted | StatusFailed | StatusNeedsFollowup`, the item is locked. Calling the existing Retry on the prior execution does not reopen the item.
- **No backlog-item-level retry route** exists. Clients must know the latest execution_id to retry, which is a leaky abstraction.
- **No UI button** exists. `canRetryExecution()` is defined in `ui/src/lib/execution-utils.ts:168` but never called.
- **Initiative parity is missing** despite the memory invariant that initiative files have full feature parity with backlog files.

## 5. Scope

**In scope:**

- Replacing the execution-level `Retry` semantics with new-attempt creation (parent linkage, fresh dispatch).
- New backlog-item-level retry route + service method.
- New initiative-item-level retry route + service method (parity).
- CLI commands: `backlog retry`, `initiatives retry`, and updated semantics of `execution retry`.
- UI: Retry button on backlog detail page and initiative detail page; client method.
- Idempotency: dedup retry calls within a short window so double-clicks don't spawn duplicate attempts.
- Tests at every layer (Go unit + handler tests, CLI integration, UI Vitest with React Testing Library).
- Stats verification: confirm `execOutcome` rollups count each attempt distinctly without double-counting at the item level.

**Out of scope:**

- Auto-retry on environmental failure (e.g., heuristics that detect "agent crashed, retry automatically"). Retry is **strictly user-initiated** in this plan, mirroring the FollowUp invariant ("no auto-FollowUp path").
- Retrying a *specific historical* attempt other than the latest. The execution-id direct route already covers this advanced case; the backlog/initiative routes always target the latest terminal execution.
- Replaying side effects (e.g., undoing partial deliverables). Retry treats the workspace as continuing from current state — the agent is responsible for handling already-applied work via existing idempotency in `acceptance` checks.
- Changing the proto schema beyond adding the new request/response messages.

## 6. Current Technical Context

### API

- `api/internal/execution/service_control.go:260-272` — existing `Retry()` (in-place mutation; to be replaced).
- `api/internal/execution/followup.go:199-348` — `FollowUp()` reference implementation; the new `Retry` mirrors its structure (new `Record`, `ParentExecutionID`, fresh `SpawnBacklog`, dispatchStatusUpdate).
- `api/internal/execution/handler_lifecycle.go:98-112` — `Retry` HTTP handler (route stays, body shape unchanged for now).
- `api/internal/execution/handler.go:39` — route registration.
- `api/internal/execution/model.go:32-50` — execution statuses (`StatusFailed`, `StatusCompleted`, `StatusCanceled`, `StatusNeedsFixup` are the eligible parent states).
- `api/internal/backlog/types.go:37-53` — backlog statuses; `StatusInProgress` is the reopen target.
- `api/internal/backlog/review_decide.go:78-170` — only legitimate writer of terminal item statuses today; we add a second narrow writer (`ReopenForRetry`) and document the invariant accordingly.
- `api/internal/backlog/service.go:183-264` — `Service.Create()` for derived items (not used here; kept as reference).
- `api/internal/initiatives/service.go` and `handler.go` — initiative service mirror; needs parity addition.

### CLI

- `cli/cmd_execution.go:217, 411, 424-425` — current `execution retry` command. Behavior changes; surface stays.
- `cli/cmd_backlog.go` (1054 LOC) — adds a new `cmdBacklogRetry` subcommand.
- `cli/cmd_initiatives.go` (assumed by parity convention) — adds matching subcommand.

### UI

- `ui/src/pages/BacklogDetailsPage.tsx:1-114` — `itemActions` object; extended with `canRetry`.
- `ui/src/components/backlog/backlog-action-buttons.tsx:124-128` — Follow-Up button location; Retry sits next to it.
- `ui/src/lib/execution-utils.ts:165-168` — `canFollowUpExecution` and `canRetryExecution` already defined; widen `canRetryExecution` to include `completed | canceled | needs_fixup` (parity with follow-up's gate) and update the call site.
- `ui/src/services/backlog/` and `ui/src/services/execution/` — add `retryBacklogItem(kind, name)`, `retryInitiative(id)`, and the execution-level mirror.
- Initiative detail page (find via `rg "InitiativeDetailsPage" ui/src`) — same Retry button.

### Proto

- `packages/proto/schemas/swarm-manager/v1/api/` — add `RetryExecutionRequest` (carries optional `note`), `RetryBacklogRequest`, `RetryInitiativeRequest`, and matching response messages. Regenerate Go/TS/Python via the standard proto build target.

### Stats

- `execOutcome` map keyed on `execution_id` (per 2026-04-22 stats repair); new attempts insert new rows naturally.
- `dispatchStatusAndLog` already handles new dispatches.
- **Open question to verify during phase 1**: does the backlog-item rollup view de-dupe by item, latest-attempt, or count-all-attempts? This determines whether retry attempts skew acceptance metrics. See Risks §11.

## 7. Target End State

A user clicking "Retry" on a terminal-state backlog item:

1. Sees an immediate UI confirmation that a new attempt is dispatched.
2. The item transitions back to `StatusInProgress` with a fresh `latest_execution_id`.
3. The prior execution row is **untouched** (status, logs, finalization preserved).
4. The new execution row has `parent_execution_id` pointing to the prior one, status `pending → starting → running`.
5. `review/decisions/{ts}-fail.json` (or whichever decision was prior) remains in the audit folder.
6. Stats show two distinct attempts; the item's "current state" reflects only the latest.
7. Same flow works identically for initiatives.

Calling `execution retry <id>` directly on a non-latest execution is allowed and creates a new attempt parented to that specific historical execution (advanced/forensic use case).

## 8. Implementation Strategy (Phased)

### Phase 1 — Foundation: stats + state-machine audit (no behavior change)

1. **Trace the item-level stats rollup.** Run `rg -n "execOutcome|latest_execution|attempts" api/internal/stats api/internal/backlog`. Document in this plan (in Risks §11) which rollups need adjusting. If item-level rollup uses `latest_execution_id` only, no change. If it sums over all executions for an item, decide: count retries or filter them. **Default decision: count all attempts** — environmental failures are real signal and a retry is its own data point.
2. **Document the `ReopenForRetry` invariant.** Update `docs/internal/INVARIANTS.md` (Replay/Idempotency section) with: "Terminal item statuses (`completed | failed | needs_followup`) may be transitioned back to `in_progress` *only* by `backlog.Service.ReopenForRetry`, called *only* from `execution.Service.Retry`." This is the second writer of terminal-bound transitions; the first is `review-decide`. Add the reverse: review-decide forward, retry backward.
3. **Validate transition allowance.** Read `packages/backlogstatus.IsValidTransition` (in `packages/proto` or wherever it lives — discover with `rg`) and confirm `completed → in_progress`, `failed → in_progress`, `needs_followup → in_progress` are or can be made permitted. If not, add them. This is the only state-machine widening.

### Phase 2 — Execution-layer retry rewrite

1. **Delete** `Retry` from `service_control.go:260-272`.
2. **Create** `api/internal/execution/retry.go` modeled on `followup.go`:
   - Function signature: `func (s *Service) Retry(ctx context.Context, req RetryRequest) (Record, error)`.
   - `RetryRequest`: `{ ExecutionID string; Note string }` (Note flows into the new run's StartedBy/Operation metadata for audit).
   - Eligible parent statuses: `StatusCompleted | StatusFailed | StatusCanceled | StatusNeedsFixup`. Reject otherwise with `apierr.BadRequest`.
   - Build a new `Record`:
     - Fresh `ExecutionID` via `idgen.Generate()`.
     - `BacklogKind`, `BacklogName` copied from parent.
     - `ParentExecutionID = parent.ExecutionID`.
     - `Status = StatusPending`.
     - `Mode = parent.Mode` (carry the original mode).
     - `StartedBy = "swarm-manager:retry"`.
     - `Operation = "retry"`.
     - `FixupAttempt = 0` (a retry is **not** a fixup; fixups have their own counter).
     - `PreviousStatus = string(parent.Status)`.
   - Build the prompt via `buildExecutionPrompt` with `RunType = "retry"` and **no** `ReviewFeedback` and **no** `FollowUpNote` — that's the whole point: same scope, no derived context.
   - **Always** spawn a fresh agent run (no `RunMode = "continue"` path; retry semantics are clean re-dispatch). This is a deliberate simplification vs. FollowUp.
   - Append the new record, save, `dispatchStatusUpdate(newRecord)`.
   - **Do not** mutate the parent record. (FollowUp also does not mutate the parent for non-fixup paths.)
3. **Idempotency guard.** Add a per-parent in-flight check inside the locked critical section: if any record exists with `ParentExecutionID == parent.ExecutionID && Status in {Pending, Starting, Running, Validating}`, return that record instead of creating a new one. This dedups double-click retries without persistent idempotency keys. (Discussed under api-steer §6.3.)
4. **Update HTTP handler** at `handler_lifecycle.go:98-112` to decode `RetryExecutionRequest` proto (new), call the new service method, and return the new execution record.
5. **Tests:** unit tests for `retry.go` covering: each eligible parent state; rejection of non-terminal parent; idempotency dedup; parent record untouched after retry.

### Phase 3 — Backlog-level retry route

1. **New service method** in `api/internal/backlog/service.go`: `func (s *Service) Retry(ctx context.Context, kind BacklogKind, name string, note string) (RetryResult, error)`.
   - Loads item; resolves the most recent execution row (assumes existing helper, find via `rg "latest.*execution|LatestExecution"`).
   - Calls `execution.Service.Retry(ctx, RetryRequest{ ExecutionID: latest.ExecutionID, Note: note })`.
   - Calls a new narrow `Service.ReopenForRetry(kind, name, newExecutionID)` (see step 2) **iff** the item is currently in a terminal status. If the item is in `in_progress | in_review | review_pending`, no item-level transition is needed (the new execution will drive it).
   - Returns `{ Item, NewExecutionID, ParentExecutionID }`.
2. **New `Service.ReopenForRetry`**:
   - Asserts `IsTerminalStatus(item.Status) == true`.
   - Sets `item.Status = StatusInProgress`, `item.Updated = now`.
   - Persists.
   - Writes a `review/decisions/{ts}-reopen.json` audit record (new decision type) noting `prior_status`, `new_execution_id`, `decided_by = "user:retry"` so the audit log shows the retry as a deliberate user action, not an unexplained terminal flip.
   - Emits event via `eventLogger.EmitBacklogStatusChanged` for downstream consumers (initiative review, stats).
3. **HTTP route**: `POST /api/v1/backlog/{kind}/{name}/retry`. Body: `RetryBacklogRequest { note: string }`. Response: `RetryBacklogResponse { item, new_execution_id, parent_execution_id }`.
4. **Reject non-existent latest execution.** If the item has no executions yet, return `apierr.BadRequest("item has no prior execution to retry")`. Retry only works on items that have actually been run.
5. **Tests:** handler test for happy path; handler test for terminal-item reopen; handler test for never-executed item; handler test for double-click idempotency.

### Phase 4 — Initiative parity (NOT NEEDED — finding)

**Discovered during implementation:** initiatives do not dispatch their own executions. They are bundles of backlog items; their status is a rollup of member item statuses. There is no `BacklogKind="initiative"` execution row, no `QueueInitiative` API, and no "an initiative ran and failed" concept. The earlier investigation that suggested initiative-level execution parity was incorrect.

The "retry" concept naturally maps onto backlog items only. An initiative whose member items failed gets re-driven by retrying those items individually — which the Phase 3 surface already provides. No initiative-level retry route, service method, or UI button is needed.

The memory invariant *"initiative files have full parity with backlog files"* refers to structural parity (workshop rounds, review rounds, conclusion docs) — not execution parity. This phase is closed without code changes.

### Phase 5 — CLI

1. **Update `cli/cmd_execution.go:424-425` (`cmdExecutionRetry`)**: behavior change is transparent at the CLI surface — same command, same flags. Add an optional `--note` flag forwarded as the request's Note field. Verify the printed output mentions both the new and parent execution IDs in human-readable mode (per `feedback_cli_default_human_output`).
2. **Add `cmdBacklogRetry`** in `cli/cmd_backlog.go`: `swarm-manager backlog retry <kind> <name> [--note "..."]`. Resolves to `POST /api/v1/backlog/{kind}/{name}/retry`. Default human output: `Retried {kind}/{name}: parent={parent_id} new={new_id} status={status}`.
3. **Add `cmdInitiativesRetry`** in the initiatives CLI file: `swarm-manager initiatives retry <id> [--note "..."]`.
4. Wire all three into the command dispatcher and `--help` output.
5. **Tests:** CLI integration tests using the existing test harness.

### Phase 6 — UI

1. **Widen `canRetryExecution`** in `ui/src/lib/execution-utils.ts:168` from `status === "failed"` to match `canFollowUpExecution` (`completed | failed | needs_fixup | canceled`).
2. **API client methods** in `ui/src/services/`:
   - `retryExecution(executionId, note?)` → `POST /api/v1/execution/{id}/retry`.
   - `retryBacklogItem(kind, name, note?)` → `POST /api/v1/backlog/{kind}/{name}/retry`.
   - `retryInitiative(id, note?)` → `POST /api/v1/initiatives/{id}/retry`.
3. **`BacklogDetailsPage.tsx`**: extend `itemActions` with `canRetry` (computed from latest execution's status via `canRetryExecution`). Wire `onRetry` handler that calls `retryBacklogItem` and on success refetches the item + navigates focus to the new execution view.
4. **`backlog-action-buttons.tsx`**: add a `<Button variant="secondary">Retry</Button>` next to Follow-Up, gated on `itemActions.canRetry`. Tooltip: "Re-run with the same scope. Use Follow-Up if the work needs to change." Disabled state when the latest execution is non-terminal.
5. **Initiative detail page**: mirror.
6. **Tests:** Vitest + RTL for the action button visibility matrix and click handler; mirror existing follow-up tests as reference.

### Phase 7 — Proto regeneration (NOT NEEDED — consistency with review-decide)

**Discovered during implementation:** the retry endpoints follow the same precedent as `review-decide` — small request/response shapes with one or two fields (`note`, `new_execution_id`, `parent_execution_id`, `status`). Adding proto messages just for these adds ceremony without value, and would diverge from the existing `review-decide` pattern. Plain JSON structs in Go (`RetryRequest`, `RetryResponse`) and TypeScript (the `RetryBacklogResponse` interface in `queue-service.ts`) are the canonical shape.

If a future endpoint adds richer fields (e.g., a structured retry context object), proto-ifying becomes attractive at that point. Greenfield principle: don't pre-build infrastructure for hypothetical needs. This phase is closed without code changes.

### Phase 8 — Documentation

1. Update `docs/internal/INVARIANTS.md` Replay/Idempotency section per Phase 1 step 2.
2. Update `docs/internal/SEAMS.md` if a new testable seam was introduced (`Service.Retry`, `Service.ReopenForRetry`).
3. Update `docs/manifest.json` if any user-facing docs are added.
4. Update the user-facing docs section that explains backlog actions to distinguish Retry vs Follow-Up clearly.

## 9. Contract Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Retry mutates parent? | No | Audit trail preservation; user's stated motivation. |
| Eligible parent execution states | `Completed`, `Failed`, `Canceled`, `NeedsFixup` | Mirrors `FollowUp`; covers all terminal/effectively-terminal states. Pending/Running/Validating reject. |
| Retry from `Completed`? | Yes | "Same scope, world changed" use case (e.g., re-run after dependency upgrade). |
| Item-level reopen path | New `ReopenForRetry` writer; only callable from `execution.Retry` | Keeps `review-decide` as the only forward terminal writer; retry is the only backward terminal writer. Symmetry, single-purpose functions. |
| Backlog/initiative routes target | Latest terminal execution only | Simple, predictable. Advanced cases use the execution-id route. |
| Idempotency window | In-flight dedup via state check (no persistent idempotency keys) | Sufficient for the double-click problem; avoids new storage. |
| FixupAttempt counter on retry | Reset to 0 | Retry is a user action, not an auto-fixup; conflating breaks fixup-attempt limits. |
| RunMode "continue" supported? | No | Retry semantics demand a clean run; "continue" is for follow-ups that build on agent context. |
| Operation field value | `"retry"` | Distinct from `"followup"` and `"fixup"` for stats slicing. |
| StartedBy | `"swarm-manager:retry"` | Matches FollowUp's `"swarm-manager:follow-up"` convention. |
| HTTP status on success | 202 Accepted | Matches FollowUp/Create; the new run is queued, not yet complete. |
| Error envelope | Existing `apierr` shape | API-steer consistency rule. |

## 10. Testing Plan

**Per `feedback_testing_over_manual.md`, all verification is via automated tests. No manual checklists.**

### Go (api)

- `api/internal/execution/retry_test.go`:
  - Retry from each eligible parent status creates new record, parent untouched.
  - Retry from each ineligible parent status returns BadRequest.
  - Idempotency: two concurrent Retry calls produce one new record.
  - `ParentExecutionID`, `Operation`, `StartedBy`, `Mode`, `FixupAttempt=0` on new record.
  - Prompt does not include review feedback or follow-up note.
- `api/internal/backlog/retry_test.go`:
  - Backlog-level retry on terminal item reopens to `in_progress` and creates new execution.
  - Backlog-level retry on non-terminal item creates new execution without item transition.
  - Backlog-level retry on never-executed item returns BadRequest.
  - Reopen audit record written.
- `api/internal/initiatives/retry_test.go`: parity matrix.
- Handler tests for all three new HTTP routes.

### CLI

- Integration tests for `execution retry`, `backlog retry`, `initiatives retry` covering happy path, error paths, and `--note` propagation.

### UI

- Vitest for `BacklogDetailsPage` retry button visibility (gated on `canRetry`).
- Vitest for action button matrix in `backlog-action-buttons.tsx`.
- Vitest for service client methods using the existing mock fetch harness.
- Mirror tests for the initiative detail page.

### Stats

- A scenario test that runs an item, fails it (review-decide → fail), retries it, completes it, and asserts the stats rollup shows two distinct attempts with the correct outcomes.

## 11. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Stats rollup at item level double-counts retries and skews acceptance rate | Phase 1 step 1 audits this before any code change. Documented decision: count all attempts; if rollup needs adjustment, do it in Phase 1. |
| `IsValidTransition` rejects `terminal → in_progress` transitions | Phase 1 step 3 verifies and widens if needed. The state machine is the source of truth. |
| Concurrent retries from different clients spawn duplicate runs | In-flight idempotency check in `Retry` (Phase 2 step 3) inside the locked section. Fully sufficient for double-clicks; not designed for distributed retry storms (out of scope). |
| Initiative retry diverges from backlog retry over time | Tests enforce parity matrix. Per `feedback_duplicate_before_extract.md`, accept the duplication for now. |
| Replacing in-place Retry breaks the existing CLI command's user expectations | The CLI surface stays identical; only the side effect changes. Users gain history preservation — a strict improvement. Document in CLI command help text. |
| `ReopenForRetry` becomes a backdoor for callers other than `Retry` | Make `ReopenForRetry` a method on the backlog `Service` whose only call site is the execution `Retry` flow. Add a code comment + the INVARIANTS.md entry. Future callers must justify themselves in review. |
| Retry creates new agent-manager session even when continuing would be cheaper | Acceptable: clean session is the contract. Users wanting continuation use Follow-Up. |
| User confuses Retry vs Follow-Up | Distinct UI tooltips, distinct CLI command names, distinct CLI output prefixes, distinct `Operation` values for log slicing. |

## 12. Non-goals / Prohibited Patterns

- **No auto-retry.** Retry is user-initiated only. Do not add a watchdog that retries failed items based on heuristics. (FollowUp has the same invariant for the same reasons; preserve the symmetry.)
- **No `RetryV2` / `RetryNew` / parallel APIs.** Replace, don't add.
- **No mutation of the parent execution record.** Ever. Audit preservation is the entire point.
- **No persistent idempotency-key table.** In-flight state check is sufficient and simpler.
- **No CLI flag like `--in-place` to opt back into the old behavior.** Not a transitional period; greenfield.
- **No removing review-decide as the forward terminal writer.** It and `ReopenForRetry` are siblings (forward and backward), not replacements.
- **No moving business logic into HTTP handlers** (api-steer §7). Handlers stay thin.

## 13. Definition of Done

- [ ] All tests in §10 pass; coverage of new code ≥ existing module average.
- [ ] `vrooli scenario test swarm-manager` is green.
- [ ] `go build ./...` and `golangci-lint run` clean across `scenarios/swarm-manager/api`, including any pre-existing issues in modified files (per planning guidelines).
- [ ] `cd scenarios/swarm-manager/cli && go build ./...` clean.
- [ ] `cd scenarios/swarm-manager/ui && npx tsc --noEmit && npx vitest run` clean.
- [ ] Proto regenerated; generated files committed.
- [ ] `INVARIANTS.md` updated with the `ReopenForRetry` invariant.
- [ ] `SEAMS.md` updated if new seams introduced.
- [ ] CLI `--help` shows new commands.
- [ ] UI Retry button renders on backlog detail page and initiative detail page; click triggers correct API call; success refetches state.
- [ ] `vrooli scenario restart swarm-manager` succeeds; API health check passes; UI loads.
- [ ] An end-to-end scenario test exercises: dispatch → fail (review-decide) → retry → succeed → stats show 2 attempts.
- [ ] No remaining references to the old in-place Retry semantics in code or docs.

## Final: Cleanup & Verification

1. `cd scenarios/swarm-manager/api && go build ./... && golangci-lint run && go test ./... -timeout 600s`
2. `cd scenarios/swarm-manager/cli && go build ./... && go test ./... -timeout 300s`
3. `cd scenarios/swarm-manager/ui && npx tsc --noEmit && npx vitest run`
4. Fix **all** lint, type, and test failures in modified files — including pre-existing ones.
5. `vrooli scenario restart swarm-manager`
6. Verify: `curl -s http://localhost:$(vrooli scenario port swarm-manager | grep API_PORT | awk '{print $2}')/health`
7. Open the UI, navigate to a terminal-state backlog item, click Retry, confirm a new execution appears with the prior one preserved in history.
