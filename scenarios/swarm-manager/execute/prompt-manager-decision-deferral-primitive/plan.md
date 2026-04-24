# Implementation Plan: Decision Deferral Primitive

## Purpose

Add a first-class `deferred` lifecycle state to the prompt-manager decision workflow, alongside a `revisit_after` ISO-date field. Operators can defer a pending decision for a time window (e.g., "revisit in 7 heartbeats after detectRateLimit fix soaks") without semantically-wrong "reject" workarounds. Deferred decisions auto-re-surface in the pending queue once `revisit_after` passes. The feature ships across API + CLI + UI per the three-surface parity rule.

## Required Reading

```bash
prompt-manager skill read seam-discovery-and-enforcement api-steer cli-steer interoperability-steer test
```

Rationale:
- `seam-discovery-and-enforcement` — keep the status enum + transition validation at a single authoritative seam
- `api-steer` — PATCH semantics, validation, error shapes for the deferral transition
- `cli-steer` — flag naming, help text, human-output conventions for `decision-defer`
- `interoperability-steer` — three-surface parity (API/CLI/UI ship together); this is the initiative's load-bearing rule
- `test` — test patterns for lifecycle transitions and queue filtering

## Greenfield Constraint

This is greenfield work. No backwards-compat shims for old decision JSONL rows: the `revisit_after` field is optional (zero value = not deferred), so existing rows remain valid without migration. Do not add "legacy mode" flags, `// removed` comments, or deprecated wrappers.

## Problem Statement

The prompt-manager decision status enum (`pending | accepted | rejected | running | completed`) has no primitive for "I want to come back to this in N days." Today operators work around this by `decision-reject --notes="deferring until X"`, which:

1. **Corrupts the decision-history signal.** Rejected-on-merit and rejected-to-defer look identical in the log. Analytics and "what did the team actually disagree with?" queries are unreliable.
2. **Forces re-creation.** When the operator wants to revisit, they must manually re-file the proposal — losing the original option set, rationale, and context.
3. **Has no auto-resurface.** The operator must remember to check back. In a vision-walk context, "just re-surface this in 7 heartbeats" is the natural ergonomic ask.

**Concrete trigger:** 2026-04-24 vision walk, operator wanted to defer `dec-1776984436121140045` (run-introspector tier-1 verification gate) for ~7 heartbeats to let a recent detectRateLimit fix soak.

## Scope

**In scope:**
- Add `deferred` to `DecisionStatus*` constants in `api/store/models.go`
- Add optional `RevisitAfter *string` (ISO-8601 date) to `DecisionEntry`
- New API transition: `pending → deferred` (requires `revisit_after`); `deferred → pending` (auto, when date passes)
- Filter deferred decisions from pending-queue endpoints until `revisit_after <= now`
- New CLI command: `prompt-manager team decision-defer <team-id> <id> --revisit-after=YYYY-MM-DD [--notes="..."]`
- UI: deferred status badge + separate "Deferred" tab in DecisionLogView; PendingDecisionsPopover excludes deferred
- Re-surfacing: when a deferred decision's date passes, it rejoins the pending queue with a visible note that it was previously deferred (so operator knows the history)
- Tests: CLI command, API transition, pending-queue filter, re-surface behavior

**Out of scope:**
- Partial-accept with modifications (sibling: `execute/prompt-manager-decision-partial-accept-with-modifications`)
- `decision-show` options-in-default-output (sibling: `fix/prompt-manager-decision-show-options-default-output`) — but the `decision-defer` CLI output should include `revisit_after` when set, since that is a new field introduced by this item
- Heartbeat-list lifecycle distinctions (sibling: `fix/prompt-manager-heartbeat-list-lifecycle-states`)
- Notifications/reminders when deferred decisions re-surface (possible follow-up; not required for MVP)
- Repeat-defer limits or max-defer-window policy

## Cross-Initiative Implications

This item is a member of `prompt-manager-decision-workflow-polish`. Siblings cover adjacent but non-overlapping gaps:

| Sibling | Overlap with this item |
|---------|------------------------|
| partial-accept-with-modifications | None — different transition (accept variant, not defer) |
| decision-show-options-default-output | Low — this item adds a `revisit_after` field that `decision-show` should render; if that fix ships first, this item adopts its output format. If this ships first, extend decision-show's human output to include `revisit_after` alongside options |
| decision-accept-no-options-ergonomics | None |
| heartbeat-list-lifecycle-states | None |
| decision-accept-initiative-proposal-auto-create | None |

No upstream or downstream initiatives. Orchestrator: no cross-initiative action needed.

## Current Technical Context

Key files (from codebase survey):

| Concern | File | Notes |
|---------|------|-------|
| Status enum | `api/store/models.go:444-450` | `DecisionStatus*` constants block |
| Decision struct | `api/store/models.go:461-477` | `DecisionEntry`; add `RevisitAfter *string` |
| Decision persistence | `api/store/team_store.go` | JSONL at `{storeDir}/teams/{teamID}/shared/decisions.jsonl`; no schema migration needed |
| Pending-queue filter | `api/heartbeat/handlers_pending.go:11-49` | `GetAllPendingDecisions()`; currently filters by `DecisionStatusPending` |
| Decision transitions | `api/heartbeat/handlers.go` | `AddDecision`, `UpdateDecisionHandler` |
| CLI decision cmds | `cli/teams/teams.go` | existing `decision-accept` (~L1088), `decision-reject` (~L1152), `decision-list` (~L1028); add `decision-defer` + `cmdDecisionDefer()` |
| UI log tabs + badge | `ui/src/components/editor/teamTabs/DecisionLogView.tsx:31-70` | StatusBadge component — add deferred badge |
| UI pending popover | `ui/src/components/tree/PendingDecisionsPopover.tsx` | Filter deferred from pending badge count |
| CLI decision tests | `cli/teams/decisions_test.go` | Pattern: `TestCmdDecisionUpdateRequiresAtLeastOneField` |
| API decision tests | `api/heartbeat/handlers_decision_test.go` | Lines 55+, 88+ |
| API pending tests | `api/heartbeat/handlers_pending_test.go` | Verify deferred exclusion |

Storage is JSONL on disk — adding the optional field requires no migration. Old rows deserialize with `RevisitAfter == nil`; new deferred rows have it set.

## Target End State

Operator runs:
```
prompt-manager team decision-defer team-abc dec-1776984436121140045 \
  --revisit-after=2026-05-01 \
  --notes="waiting to see if recent detectRateLimit fix eliminates the pattern"
```
and the decision:
- Transitions from `pending` to `deferred` with `revisit_after=2026-05-01` and the note appended
- Disappears from `GetAllPendingDecisions()` response and from the UI pending badge
- Shows up in a "Deferred" tab/section in the decision log with its revisit date
- On or after 2026-05-01, automatically reappears in pending (status flips back to `pending`) with a note indicating it was previously deferred, so the operator knows the history
- Can still be manually accepted/rejected from the deferred tab without waiting

## Implementation Strategy

### Phase 1 — Data model & persistence

1. Add `DecisionStatusDeferred = "deferred"` to the const block in `api/store/models.go:444-450`.
2. Add `RevisitAfter *string \`json:"revisit_after,omitempty"\`` to `DecisionEntry` in `api/store/models.go:461-477`.
3. In `team_store.go`:
   - `GetDecisions()` — verify existing filter-by-status continues to work (it will; new status is just another string value).
   - `UpdateDecision()` — callback already supports arbitrary field updates; no API-shape change.
4. Verify JSONL round-trip: write a deferred row, read it back, confirm field preserved.

### Phase 2 — API handler + pending-queue filtering

1. In `api/heartbeat/handlers.go`:
   - Extend `UpdateDecisionHandler` to accept `status: "deferred"` with a required `revisit_after` (ISO-8601 date, YYYY-MM-DD). Reject if date missing or malformed or in the past.
   - Allowed transitions: `pending → deferred`, `deferred → pending | accepted | rejected`. Any other target from `deferred` is 400.
2. In `api/heartbeat/handlers_pending.go:11-49`:
   - After fetching `DecisionStatusPending` rows, also fetch `DecisionStatusDeferred` rows whose `RevisitAfter <= today`.
   - For each of those, flip status back to `pending` via `UpdateDecision()` and append a note like `"Re-surfaced from deferral on YYYY-MM-DD (deferred YYYY-MM-DD → YYYY-MM-DD)"`. This makes re-surfacing stateful/durable — prep-agent sees them in pending from that point on, even across restarts.
   - Return the unified pending set (original + re-surfaced).
3. Decide whether to short-circuit when there are no deferred rows due (cheap check: min `revisit_after` over deferred rows).

### Phase 3 — CLI

1. In `cli/teams/teams.go`, add `decision-defer` to the command dispatch table and implement `cmdDecisionDefer()` following the `cmdDecisionReject` shape.
2. Flags: `--revisit-after=YYYY-MM-DD` (required), `--notes="..."` (optional).
3. Validation (client-side): parse the date, reject if malformed or in the past. Delegate deeper validation to the API.
4. Human output: confirmation line with team id, decision id, new status, revisit date, and the note.
5. Update help text for `team` subcommand listing.

### Phase 4 — UI

1. In `DecisionLogView.tsx:31-70`, add a `deferred` case to `StatusBadge` — choose a distinct color (e.g., amber/gold, distinct from pending-amber; or a neutral gray with a clock icon).
2. Add a "Deferred" tab/filter to the decision-log view; deferred rows render with their `revisit_after` date visible.
3. On a deferred row, show Accept / Reject / Un-defer (return to pending now) actions. Clicking the row opens the original decision detail unchanged.
4. In `PendingDecisionsPopover.tsx`, filter out deferred rows from the count and the popover list. (If the backend already re-surfaces them on date pass, the UI does not need its own date check — it just trusts the backend pending list.)

### Phase 5 — Tests

1. **CLI** (`cli/teams/decisions_test.go`):
   - `TestCmdDecisionDeferRequiresRevisitAfter`
   - `TestCmdDecisionDeferRejectsPastDate`
   - `TestCmdDecisionDeferRejectsMalformedDate`
   - `TestCmdDecisionDeferHappyPath` — asserts status + revisit_after in persisted row
2. **API transition** (`api/heartbeat/handlers_decision_test.go`):
   - `TestUpdateDecision_DeferFromPending_RequiresRevisitAfter`
   - `TestUpdateDecision_DeferRejectsIllegalTransitions` (accepted→deferred, completed→deferred)
   - `TestUpdateDecision_AcceptFromDeferred` (deferred → accepted allowed)
3. **Pending queue** (`api/heartbeat/handlers_pending_test.go`):
   - `TestPendingQueue_ExcludesDeferredWithFutureDate`
   - `TestPendingQueue_ResurfacesDeferredOnDueDate` — seed deferred row with `revisit_after <= today`, call endpoint, assert row now has status `pending` and a re-surface note
4. **UI**: component-level tests for `StatusBadge` deferred case + PendingDecisionsPopover excluding deferred.

### Phase 6 — Cleanup & health verification

- Run `go build ./...` (or the scenario's equivalent) and fix **all** type/lint errors in touched files, even pre-existing ones.
- Run `golangci-lint run` on touched packages and fix all findings in modified files.
- Run UI lint + type-check (`npm run lint`, `tsc --noEmit`) in `ui/`.
- Run the full decision-related test suite and make sure it is green.
- `vrooli scenario restart prompt-manager`.
- Health-check: create a decision, defer it with a future date, confirm it disappears from the popover; backdate `revisit_after` manually in the JSONL to today, hit pending, confirm it re-surfaces with the note.

## Contract Decisions

| Concern | Decision |
|---------|----------|
| `revisit_after` format | YYYY-MM-DD (date only, no time/tz). Day-granularity is enough for this use case ("7 heartbeats" rounds to days). |
| Past-date `revisit_after` on defer | Reject at API with 400 — deferring to yesterday is nonsensical |
| `revisit_after == today` | Allowed — becomes pending on next pending-queue read |
| Re-surface mechanism | Lazy / read-triggered: `GetAllPendingDecisions` promotes due deferred rows. No background cron. Reason: prompt-manager has no long-running scheduler loop for decisions today; the pending-queue read is frequent enough (every heartbeat) to act as the trigger. |
| Re-surface persistence | Durable: when promoted, the row's status is flipped back to `pending` on disk with an appended note, so subsequent reads see it as a normal pending. |
| Transitions from `deferred` | `deferred → pending` (auto on date or manual un-defer), `deferred → accepted`, `deferred → rejected`. Not allowed: `deferred → running`, `deferred → completed`. |
| Re-defer | Allowed: `pending → deferred` with a new `revisit_after` is legal even if the decision was previously deferred. |
| Notes concatenation | Append-only; each defer/re-surface adds a line, nothing is overwritten. Preserves audit trail. |

## Testing Plan

See Phase 5 above. Coverage targets:

- Every allowed transition has a positive test
- Every disallowed transition has a negative test (400 expected)
- Pending-queue filter covers: all pending kept, future-deferred excluded, due-deferred re-surfaced with note
- CLI covers: required flag, bad date, past date, happy path
- UI covers: badge render, popover exclusion, tab filter (snapshot or behavioral)

Manual QA: reproduce the triggering scenario — defer `dec-like` test decision for 7 days, confirm popover count drops, reset system clock forward (or manually edit JSONL), confirm re-surface.

## Rollout / Validation Checklist

- [ ] API: `deferred` status and `revisit_after` field present in responses
- [ ] API: `UpdateDecisionHandler` accepts/rejects transitions per the matrix
- [ ] API: pending-queue excludes future-deferred, re-surfaces due-deferred
- [ ] CLI: `prompt-manager team decision-defer --help` documents flags
- [ ] CLI: happy path + all negative tests green
- [ ] UI: Deferred tab renders; badge colored distinctly; popover count excludes deferred
- [ ] Scenario restart healthy; end-to-end defer → re-surface demonstrated manually
- [ ] No new lint/type warnings introduced; pre-existing ones in touched files fixed
- [ ] No writes to `scenarios/prompt-manager/secrets/**` or out-of-scope paths

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Lazy re-surface is missed if pending queue is never read | Low | Low | Heartbeat reads pending frequently; if a deployment drifts, re-surfacing is simply delayed, not lost |
| Clock skew between operator machine and server flips `revisit_after <= now` early | Low | Low | All comparison uses server clock; client-side past-date validation is a UX hint |
| Re-surface on every pending read could thrash if clock goes backwards | Very Low | Low | Once promoted, row is status=`pending`; no further comparison runs |
| UI shows stale pending count because popover caches | Medium | Low | Invalidate popover cache on decision mutation, as the codebase already does for accept/reject |
| Existing decision rows without `revisit_after` break JSON decoding | Very Low | Medium | Field is `*string` with `omitempty` — absent value deserializes to nil |
| Sibling item `decision-show-options-default-output` ships concurrently and conflicts in CLI output | Low | Low | Both touch `decision-show`; coordinate: if sibling merges first, this item just adds `revisit_after` to the already-refactored output block |

## Non-goals / Prohibited Patterns

- No background cron/scheduler to promote deferred rows (lazy promotion only)
- No "defer until condition X" primitive — dates only (complexity not justified for the use case)
- No repeat-defer rate limit
- No user notifications on re-surface
- No migration script — JSONL field additions are forward-compatible
- Do not introduce compat shims or leave `_unused` vars / `// removed` comments

## Definition of Done

- [ ] `DecisionStatusDeferred` const + `RevisitAfter` field landed in `api/store/models.go`
- [ ] `UpdateDecisionHandler` transitions implemented and tested
- [ ] `GetAllPendingDecisions` filters future-deferred and promotes due-deferred with audit note
- [ ] `prompt-manager team decision-defer` command available with help text, flags, validation
- [ ] UI: deferred badge + Deferred tab + popover exclusion
- [ ] All new tests green; no regressions in existing decision tests
- [ ] Scenario restarts healthy; manual end-to-end verified
- [ ] Lint/type-check clean in all touched files
- [ ] No changes outside `scenarios/prompt-manager/**`
- [ ] Greenfield: no legacy shims introduced
