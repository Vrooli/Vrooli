# Finalize Post-Approval Refresh Reconciliation Plan

## 1. Purpose

Make Swarm Manager reliably reflect a backlog item's **post-approval finalized state** after a sandboxed finalize run leaves `needs_review`, is manually approved in Agent Manager, and completes applying to the canonical repo.

The bug is not that Agent Manager failed to finalize. The bug is that Swarm Manager can stop polling at the exact moment the run transitions from `needs_review` to `complete`, leaving the detail page stuck on stale pre-approval workshop/readiness data. The result is a false `Finalize` CTA even though the canonical backlog files already contain a finalized round and updated `plan.md`.

This is **not a greenfield rewrite**. The implementation should preserve the current activity/readiness architecture and fix the reconciliation seam with the smallest coherent change set.

## 2. Required Reading

Run before implementation:

```bash
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read scientific-debugging
```

Primary files to read first:

- `scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx`
- `scenarios/swarm-manager/ui/src/hooks/useBacklogQueries.ts`
- `scenarios/swarm-manager/ui/src/hooks/useBacklogDetailData.ts`
- `scenarios/swarm-manager/ui/src/stores/agent-activities-store.ts`
- `scenarios/swarm-manager/ui/src/hooks/useActivityTimeline.ts`
- `scenarios/swarm-manager/ui/src/lib/agent-activity-utils.ts`
- `scenarios/swarm-manager/api/internal/backlog/maturity_summary.go`
- `scenarios/swarm-manager/api/internal/workshop/workshop.go`
- `scenarios/swarm-manager/api/internal/agentactivity/polling.go`

Command evidence gathered on April 27, 2026:

```bash
vrooli scenario status swarm-manager
curl -s http://127.0.0.1:16421/api/v1/backlog/maturity-summary | jq '.items[] | select(.kind=="execute" and .name=="agent-manager-sandbox-auto-apply-defaults")'
curl -s http://127.0.0.1:16421/api/v1/backlog/execute/agent-manager-sandbox-auto-apply-defaults/files | jq '.'
curl -s 'http://127.0.0.1:16421/api/v1/agent-activities?owner_type=backlog&owner_kind=execute&owner_name=agent-manager-sandbox-auto-apply-defaults' | jq '.'
```

## 3. Problem Statement

Observed user-facing behavior:

1. A finalize run reaches Agent Manager `needs_review`.
2. The user manually approves the run.
3. The finalize changes are applied to the canonical repo.
4. Swarm Manager's Activity tab shows the finalize attempt and later shows it as successful.
5. The backlog detail header still offers `Finalize`, as if the finalize round never landed.

Concrete evidence for the reproduced item `execute/agent-manager-sandbox-auto-apply-defaults`:

- Canonical repo contains `workshop/round-003.json` with `mode: "finalize"` and `pending_synthesis: false`.
- Canonical repo contains updated `plan.md`.
- Live Swarm Manager API reports:
  - `rounds_completed: 3`
  - `ready: true`
  - `pending_synthesis: false`
- Live agent-activity API shows finalize runs `475b2704-74e3-429f-9e1c-1fb3252c30ca` and `862593ae-4520-4e47-8c21-fcf4e5fa4ff5` as `status: "complete"`.

Root cause:

- The detail page uses the active-only agent-activity store to decide whether the item is still "blocking".
- Files, workshop rounds, and maturity summary poll only while `agentRunIsBlocking` is true.
- Once approval transitions the activity from `needs_review` to `complete`, the active-only store drops the record.
- `agentRunIsBlocking` flips to false immediately.
- Polling stops before the page necessarily fetches the new canonical repo state.
- The header then computes item actions from stale `workshopRounds` / `readinessData`, so `pendingSynthesis` may remain true locally and `Finalize` stays visible.

This is a **state reconciliation bug in Swarm Manager**, not an Agent Manager apply failure.

## 4. Scope

### In scope

- Fix the backlog detail page so it performs a deterministic refresh when a tracked backlog activity transitions from active/blocking to terminal.
- Make the page recompute `workshopRounds`, `readinessData`, and CTA state from freshly fetched canonical data after approval.
- Preserve the existing distinction between "currently executing" and "awaiting manual review", while ensuring either path still leads to a final refresh after terminal completion.
- Add regression tests for the exact post-approval transition sequence.
- Validate the fix against the real backlog item shape that reproduced the bug.

### Out of scope

- Changing Agent Manager's approval/apply semantics.
- Changing workspace-sandbox apply behavior.
- Reworking the entire activity architecture or replacing polling with webhooks.
- Broad UI redesign of activity badges or button wording beyond what is necessary for correctness.
- Changing how finalize rounds are authored or how `pending_synthesis` is computed on the backend.

## 5. Current Technical Context

### Client-side state composition

- `BacklogDetailsPage.tsx`
  - derives `latestAgentActivity` from the active-only store
  - computes `agentRunIsBusy` and `agentRunIsBlocking`
  - passes `agentRunIsBlocking` into `useBacklogDetailData`
- `agent-activities-store.ts`
  - polls `agentActivityService.list({ active: true })`
  - tracks only `pending | starting | running | needs_review`
  - drops the record once it becomes `complete`
- `useBacklogQueries.ts`
  - polls backlog files only while `agentRunIsBlocking`
  - polls workshop rounds only while `agentRunIsBlocking`
  - polls maturity summary only while `agentRunIsBlocking`
  - polls review rounds while `agentRunIsBlocking || isValidating`
- `useBacklogDetailData.ts`
  - computes `isWorkshopFinalized` from:
    - any round with `mode === "finalize"`
    - and `readinessData.pendingSynthesis === false`
  - feeds `agentRunning: agentRunIsBlocking` into item-action resolution

### Backend truth sources

- `maturity_summary.go` computes `pending_synthesis` from `workshop.NeedsSynthesis(latestRound)`.
- `workshop.go` treats finalize rounds as `NeedsSynthesis == false`.
- `agentactivity/polling.go` correctly refreshes activity status to `complete`.

### Key implication

The backend already reaches the correct finalized state. The missing behavior is a **final refresh handshake** on the client once approval completes.

## 6. Target End State

After the fix:

1. When a backlog-owned activity transitions from active/blocking (`pending | starting | running | needs_review`) to terminal (`complete | failed | cancelled`), the backlog detail page performs a one-shot reconciliation refresh.
2. That reconciliation refresh fetches at least:
   - backlog files
   - workshop rounds
   - maturity summary
   - execution history
   - review rounds when relevant
3. The page recomputes `isWorkshopFinalized` and item actions from fresh canonical data.
4. If the finalize round landed successfully, the page stops showing `Finalize`.
5. Reloading the page is no longer required to clear stale pre-approval state.
6. Activity history remains correct and still shows terminal finalize attempts.

## 7. Implementation Strategy

### Phase 1 — Isolate the transition seam

1. Add a dedicated derived transition signal in the backlog detail page or a focused hook:
   - previous `latestAgentActivity` status / run id
   - current `latestAgentActivity` status / run id
   - detect when the page moves from a tracked blocking activity to no active activity, or from blocking to a non-blocking terminal state if terminal activities become available locally
2. Keep this logic localized. Do not spread transition detection across multiple components.
3. Prefer a small focused helper or hook such as `useBacklogPostActivityReconciliation(...)` rather than embedding ad hoc `useEffect` logic directly in JSX-heavy page code.

### Phase 2 — Add a one-shot canonical refetch

1. When the transition seam detects "blocking activity ended", trigger a single reconciliation refresh.
2. The refresh must invalidate or refetch the data sources that determine finalize state:
   - `["backlog", backlogKind, name, "files"]`
   - `["backlog", backlogKind, name, "workshop-rounds", ...]`
   - `["backlog-maturity-summary"]`
   - `["executions", backlogKind, name]`
   - `["review-rounds", backlogKind, name]` when the page uses them for manual-review state
3. Prefer React Query invalidation/refetch over introducing a second polling system.
4. The reconciliation must run even if normal polling just stopped because `agentRunIsBlocking` became false.

### Phase 3 — Separate "active blocking" from "refresh gating"

1. Stop using `agentRunIsBlocking` as the only driver for "should we still fetch canonical repo state right now?"
2. Introduce a narrower semantic:
   - `agentRunIsBlocking` still means "the item is not yet clear for user action"
   - post-approval reconciliation is a separate short-lived refresh state
3. The simplest acceptable shape is:
   - normal interval polling while blocking
   - one-shot refetch on transition out of blocking
4. Avoid a long-lived extra polling mode unless tests prove the one-shot refresh is insufficient.

### Phase 4 — Fix item-action inputs

1. Review the `getItemActions(...)` call site in `useBacklogDetailData.ts`.
2. Ensure "agent currently running" is not inferred from blocking-only state.
3. Preserve the existing `needs_review` coordination semantics, but do not let them mask stale finalized repo state after approval.
4. If a minimal refactor is needed, split:
   - `agentExecuting`
   - `agentBlocking`
   - `postApprovalReconciling`

### Phase 5 — Keep timeline/history behavior intact

1. Do not break `useActivityTimeline(...)`, which intentionally fetches full activity history without `active=true`.
2. Keep the Activity tab capable of showing terminal finalize attempts.
3. If a shared helper is introduced, avoid coupling timeline history to the active-only store.

### Phase 6 — Validation against the reproduced item

1. Re-run the reproduced item flow using a sandboxed finalize run that enters `needs_review`.
2. Approve it in Agent Manager.
3. Confirm the Swarm Manager detail page transitions to the finalized state without a manual reload.
4. Confirm canonical repo artifacts remain the source of truth:
   - `workshop/round-NNN.json`
   - `plan.md`
   - maturity summary `pending_synthesis`

## 8. Contract Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Source of truth for finalized state | Canonical backlog files + maturity summary | Matches current backend contract and observed correct API state |
| Trigger for reconciliation refresh | Transition from blocking activity to terminal/no active activity | This is the exact seam where stale state currently persists |
| Refresh model | One-shot React Query refetch/invalidation | Smallest coherent fix; avoids introducing another background poller |
| Activity history model | Keep full-history fetch separate from active-only store | Activity tab is already correct and should stay decoupled |
| Backend changes | None by default | Live API already computes correct finalized state for the reproduced item |
| Manual review semantics | Preserve `needs_review` as blocking, not executing | Coordination semantics stay intact while correctness is restored |

## 9. Testing Plan

### Unit / hook tests (UI)

- `useBacklogQueries` or the new reconciliation hook:
  - when `agentRunIsBlocking` changes from true to false after an active finalize activity, the hook triggers a one-shot refetch
  - no refetch loop occurs on stable idle state
- `useBacklogDetailData`:
  - stale pre-approval readiness (`pendingSynthesis=true`) updates to finalized readiness (`pendingSynthesis=false`) after reconciliation
  - item actions stop surfacing `Finalize` once reconciled data includes a finalize round and `pendingSynthesis=false`

### Component / page tests (UI)

- `BacklogDetailsPage`:
  - seed initial state with:
    - active finalize activity in `needs_review`
    - workshop rounds only through `round-002`
    - readiness summary `pending_synthesis=true`
  - simulate approval completion by removing the active activity and returning refetched canonical data with:
    - `round-003`
    - `mode: finalize`
    - `pending_synthesis=false`
  - assert the header no longer shows `Finalize`
- Add a regression test proving that Activity tab history can still show the finalize attempt as `complete`.

### API-level verification

- Keep a lightweight verification test or command assertion that the backend still reports:
  - `pending_synthesis=false` for finalized items
  - terminal finalize activity records via `/api/v1/agent-activities`

### Scenario validation

- Run the relevant Swarm Manager UI tests.
- Restart the scenario and repeat the reproduced flow against a live local instance.

## 10. Rollout/Validation Checklist

1. Implement the transition-detection seam and one-shot reconciliation refresh.
2. Run Swarm Manager UI tests covering backlog detail page action state.
3. Restart Swarm Manager:
   ```bash
   vrooli scenario restart swarm-manager
   ```
4. Reproduce the sandboxed finalize approval flow locally.
5. Confirm:
   - Activity history shows the finalize run as complete
   - canonical files include the finalize round
   - backlog detail page stops showing `Finalize` without reload
6. Re-check the live maturity summary endpoint for the item to confirm `pending_synthesis=false`.

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| One-shot refetch fires before canonical apply finishes | Header could still remain stale intermittently | Trigger reconciliation on the activity transition after Agent Manager marks the run terminal, and allow targeted retry/refetch-once if the first reconciliation still sees the pre-terminal snapshot |
| Refetch logic loops on every render | UI churn, needless traffic | Gate the reconciliation on an explicit state transition, not current boolean state |
| Fix is over-coupled to one reproduced item | Narrow bug fix, future variants missed | Base the trigger on lifecycle semantics (`blocking -> terminal`) rather than item-specific assumptions |
| Refetch set is incomplete | CTA still stale due to one untouched cache | Invalidate all finalize-state inputs together: files, workshop rounds, maturity summary, execution history, review rounds |
| Mixing executing and blocking state further muddies CTA logic | Future regressions | Keep the semantics explicit and localized in one hook/helper |

## 12. Non-goals / Prohibited Patterns

- Do not change Agent Manager or workspace-sandbox because the reproduced evidence does not justify it.
- Do not add a second long-running poller just to paper over a missing transition refresh.
- Do not couple canonical finalize state to activity history alone.
- Do not rely on full page reloads or navigation churn as the fix.
- Do not introduce a broad refactor of every backlog-detail hook unless the minimal seam fix proves insufficient.

## 13. Definition of Done

- Swarm Manager no longer requires a manual page reload after approving a sandboxed finalize run.
- A backlog item whose canonical repo now contains a finalize round and `pending_synthesis=false` stops showing `Finalize`.
- The fix is implemented through an explicit reconciliation seam, not incidental polling behavior.
- Automated tests cover the reproduced transition from `needs_review` to approved terminal completion.
- Live validation against a local Swarm Manager instance passes.
- `vrooli scenario restart swarm-manager` succeeds cleanly as the final cleanup step.
