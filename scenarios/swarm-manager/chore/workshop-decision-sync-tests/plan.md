# End-to-end coverage for workshop-decision-prep heartbeat and workshop-decision-sync skill

## Required Reading

- `prompt-manager skill read swarm-manager-backlog-tools` — backlog item folder layout, CLI surface, mutation rules
- `prompt-manager skill read documentation-health` — keeping docs/test-plans accurate as the underlying behaviour evolves
- `prompt-manager skill read seam-discovery-and-enforcement` — choosing where to test (data seam vs UX seam) for high-leverage coverage
- `prompt-manager skill read swarm-manager-process-chore` — execution-time guidance for chore items

## Greenfield Constraint

**This is greenfield work.** No compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. New tests and new docs only — do not preserve hypothetical "old" testing surfaces.

## Purpose

Close the only remaining gap in the `workshop-decision-triage` initiative: produce automated and manual proofs that the **workshop-decision-prep heartbeat** (data path + freshness rules) and the **workshop-decision-sync skill** (conversational UX) behave correctly end-to-end. Once this chore lands the initiative is shippable.

## Problem Statement

Three execute items in `workshop-decision-triage` already shipped — priority-sort + CLI wrapper, the `workshop-decision-prep` director-swarm member, and the `workshop-decision-sync` skill. Their unit-level coverage exists (`pending_questions_test.go`, `app_test.go`), but the integrated story has no proof:

> *prep heartbeat reads pending decisions → writes a fresh `last-handoff.md` → sync skill answers one decision through `workshop/save` → next prep run drops the answered decision*

Plus the conversational behaviour (initiative grouping, focus stack, async clarify spawn) is documented prose with no validation harness.

If we ship without this chore: regressions in the rank/save/clarify HTTP path land silently, and operators discover skill-side UX bugs only when they hit them in real 5-minute sessions.

## Current Technical Context

Key files / components and what they own:

- `scenarios/swarm-manager/api/internal/backlog/pending_questions_test.go` — existing Go integration tests for the priority-sort and filter behaviour of `GET /api/v1/backlog/pending-questions`. Sibling helpers (`setupTestHandler`, `setupTestHandlerWithAgent`, `createTestItem`, `enableAutoAdvanceSettings`) are package-private and reused here.
- `scenarios/swarm-manager/api/internal/backlog/workshop_save_test.go` — existing Go integration tests for `POST /workshop/save`; defines `mockAgentService{result: agentmanager.RunResult{...}}`, `makeWorkshopSaveBody`, `workshopSaveRequest`. The same mock is reusable for the clarification endpoint (which also spawns an agent).
- `scenarios/swarm-manager/api/internal/backlog/handlers.go` (and friends) — HTTP handlers for pending-questions, workshop/save, workshop/clarification. The contract surface this chore validates.
- `scenarios/swarm-manager/cli/app_test.go` — existing CLI flag-forwarding tests (`TestCmdBacklogPendingQuestionsValidation`, `TestCmdBacklogPendingQuestionsRequestsExpectedEndpoint`). Done; not touched here.
- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/HEARTBEAT.md` — defines the canonical-hash recipe (step 4) the freshness check uses. The Go test mirrors that recipe explicitly so a recipe change fails CI.
- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/last-handoff.md` — currently a placeholder ("No staged briefs yet."); populated by the real heartbeat. The Go test does **not** depend on this file; the manual smoke plan does.
- `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/SKILL.md` — the conversational skill's primary instructions; loaded into agent context on every skill invocation, so it must stay lean.
- `scenarios/swarm-manager/api/internal/workshop/` — canonical `workshop.Round` and `workshop.Item` types; tests must seed using these types (not local copies) so the test data matches production wire shape.

Acceptance globs already constrain edits to:
- `scenarios/swarm-manager/api/internal/backlog/**`
- `scenarios/swarm-manager/cli/**` (reserved; no edits expected)
- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/**` (reserved; only touched if a freshness-rule clarification is surfaced by the test)
- `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/**`

No `acceptance_deny` is needed.

## Target End State

When this chore is complete:

1. A new Go integration test file `scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go` exists and passes. It exercises five sub-scenarios end-to-end at the data seam: seed → cap-respecting freshness → mutate-and-refresh → answer-and-drop → async-clarify seam.
2. A new operator checklist `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/TEST-PLAN.md` exists, structured into Setup / Inspection / Conversational Walk-through / Round-trip Proof / Pass-fail Rubric.
3. A one-line pointer is appended to the bottom of `SKILL.md`: *"See `TEST-PLAN.md` before merging behaviour changes."*
4. Running `vrooli scenario test swarm-manager unit` (or the package-level `go test ./scenarios/swarm-manager/api/internal/backlog/...`) passes with zero new failures.
5. `vrooli scenario restart swarm-manager` completes cleanly and `curl -s http://localhost:<swarm-manager-api-port>/health` returns 200.
6. No edits land outside the `acceptance_allow` globs.

## Implementation Strategy

### Decision Summary (workshop round-001 outcomes, all answered)

| Decision | Selected | Implication |
|---|---|---|
| d1 — Heartbeat-test fidelity | **C** Hybrid | Go integration test covers the data seam (no Claude run); manual TEST-PLAN.md covers the LLM enrichment narrative |
| d2 — Go test file location | **A** Inside `internal/backlog/` package | Reuse private sibling helpers; no acceptance-glob widening |
| d3 — Skill smoke plan location | **A** Colocated with skill + pointer in SKILL.md | High discoverability without bloating SKILL.md's per-invocation context |
| d4 — Async-clarify coverage | **C** Both | Go test asserts HTTP seam; manual checklist asserts UX (skill offers it, accepts, session continues) |

### Phase 1 — Go integration test (data seam)

Add `scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go`. Pattern follows `pending_questions_test.go` / `workshop_save_test.go` (uses `setupTestHandler` / `setupTestHandlerWithAgent`, `createTestItem`, `testutil.WriteJSONFile`).

Sub-tests inside that file:

1. **Seed** — 2 initiatives, 3 backlog items spread across them, each item with a `workshop/round-001.json` containing 2 unresolved decisions (6 open decisions total). Use canonical `workshop.Round` / `workshop.Item` types. Mirror the shape of `TestPendingQuestions_WorkshopDecisionsOnly`.
2. **Cap-respecting freshness simulation** — drive the heartbeat's deterministic core directly:
   - `GET /api/v1/backlog/pending-questions?source=workshop&limit=20` and capture the response.
   - Compute SHA-256 over canonical `topic + text + context + options` for each returned decision (recipe matches `HEARTBEAT.md` step 4; encoded in a 10-LoC helper inside the test file with a comment block citing that source).
   - Build a fake `last-handoff.md` payload pretending the cap (15) was already met using the seeded decisions; assert: when the same decisions/hashes are present, the freshness check returns "all valid → no enrichment needed" (early-return shape).
3. **Mutate-and-refresh** — modify one decision's `options` array on disk, recompute hashes, re-run the freshness check; assert *only* the mutated decision's brief is dropped, others remain valid.
4. **Answer-and-drop** — `POST /api/v1/backlog/{kind}/{name}/workshop/save` patching `selected` onto one decision (mirroring `TestWorkshopSave_ValidRound_WritesFile`). Re-query pending-questions; assert that decision is excluded from the next response and its hash no longer matches the (now-stale) cached brief.
5. **Async-clarify seam** — `POST /api/v1/backlog/{kind}/{name}/workshop/clarification` with `round_number` / `item_id` / `message`. Assert:
   - response contains `thread.id` and `thread.run_id`,
   - the targeted decision's `selected` is unchanged (clarification spawn must not mutate the decision),
   - a subsequent pending-questions call still returns that decision,
   - the mocked agent service receives the spawn call.
   Stub the agent service with `mockAgentService{result: agentmanager.RunResult{...}}` per `workshop_save_test.go`; no real agent runs.

Helpers:
- Canonical-hash helper (~10 LoC) lives next to the test, package-private. If reuse demand emerges later, move into `internal/workshop/`.
- Reuse: `setupTestHandlerWithAgent`, `createTestItem`, `enableAutoAdvanceSettings`, `makeWorkshopSaveBody`, `workshopSaveRequest`.

### Phase 2 — Manual skill-smoke test plan (UX seam)

Add `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/TEST-PLAN.md`. Operator-driven checklist, run after material changes to either the skill or the prep heartbeat. Structure:

1. **Setup** — seed 3 items × 2 initiatives (cite the Go test as a structural reference, or list `swarm-manager backlog create` invocations). Trigger heartbeat: `prompt-manager team heartbeat-trigger director-swarm workshop-decision-prep`.
2. **Inspection of `last-handoff.md`** — checklist asserting: grouped by initiative → item → decision; each decision block carries `kind/name/round/item_id/content_hash`; recommendation flags preserved on options; anticipated Q&A present.
3. **Conversational walk-through** — invoke the skill (`prompt-manager skill read workshop-decision-sync`, then run as a focused session). Verify:
   - (a) initiative summary printed once per initiative transition; backlog summary once per item transition,
   - (b) focus-stack: clarifying question mid-decision answered from current decision context only; no pivot to a different decision/initiative,
   - (c) async clarify: unanticipated question triggers an offer of `swarm-manager backlog clarify`; accepting runs the CLI and the session continues to the next decision without waiting for resolution.
4. **Round-trip proof** — answer one decision conversationally; confirm via `swarm-manager backlog pending-questions --source workshop --json` that the answered decision drops from the queue.
5. **Pass/fail rubric** — explicit lines for what counts as a pass (e.g., "all four checklist sections complete with no observed pivots, no stale briefs after answer"). Each section also lists "what would have to fail to invalidate this section" so future readers can update it when behaviour shifts.

Append a one-line pointer at the bottom of `SKILL.md`:
> *See `TEST-PLAN.md` before merging behaviour changes.*

The TEST-PLAN.md notes that the Go integration test (Phase 1) is the daily-CI safety net; the manual plan is the periodic UX validation — neither substitutes for the other.

### Phase 3 — Final cleanup and verification

Run in order, fix all surfaced issues even if pre-existing:
1. `go build ./scenarios/swarm-manager/...` — fix all compile errors.
2. `go test ./scenarios/swarm-manager/api/internal/backlog/...` — fix all failing tests.
3. `golangci-lint run ./scenarios/swarm-manager/api/internal/backlog/...` — fix all warnings in modified files.
4. `gofumpt -w scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go`.
5. `vrooli scenario restart swarm-manager`.
6. Verify health: `curl -s http://localhost:<swarm-manager-api-port>/health` returns 200.

## Contract Decisions

These are the contract assertions the new Go test pins down (any deviation = test must fail):

| Surface | Contract |
|---|---|
| `GET /api/v1/backlog/pending-questions?source=workshop&limit=N` | Returns only decisions where `selected == nil`; orders per the established priority-sort; respects `limit` and the `--initiative` filter. |
| Canonical hash recipe | SHA-256 over `topic + text + context + options` exactly as defined in `HEARTBEAT.md` step 4. Field rename, reorder, or coverage change must fail the test loudly. |
| `POST /api/v1/backlog/{kind}/{name}/workshop/save` | Persists the `selected` field for the targeted decision. After save, the decision is excluded from the next pending-questions response. |
| `POST /api/v1/backlog/{kind}/{name}/workshop/clarification` | Returns `thread.id` and `thread.run_id`. **Does not mutate** the targeted decision (`selected` stays untouched). Spawns the agent through the standard agent service interface (asserted via mock). |
| Skill smoke plan | Lives at `packs/core/workshop-decision-sync/TEST-PLAN.md`. SKILL.md gains exactly one trailing pointer line; no other SKILL.md edits. |

## Testing Plan

How we know the deliverables are correct:

**Go test is meaningful** — it must FAIL when any of these regressions are introduced (verify by hand-mutation during dev):
- `pending-questions` returns answered decisions (regression in `Selected != nil` filter).
- `workshop/save` does not persist the `selected` field.
- `clarification` endpoint mutates the decision's `selected` field.
- canonical-hash recipe changes silently (rename a field, reorder, drop one).

**Manual smoke plan is meaningful** — each section ends with a "what would have to fail to invalidate this section" line, so future readers can update the plan when behaviour shifts rather than discarding it.

**Test commands**:
```bash
go test ./scenarios/swarm-manager/api/internal/backlog/...
vrooli scenario test swarm-manager unit
```

The smoke plan is run by the operator manually on changes to the prep agent or the sync skill — not on every PR.

## Rollout/Validation Checklist

Run through in order during execution; do not mark the chore done until every box passes.

- [ ] New file `scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go` exists and contains all five sub-tests above.
- [ ] `go test ./scenarios/swarm-manager/api/internal/backlog/...` passes with no new failures.
- [ ] Hand-mutation check: temporarily break the `Selected != nil` filter, confirm the new test fails; revert.
- [ ] Hand-mutation check: temporarily change the canonical-hash recipe (rename a field), confirm the new test fails; revert.
- [ ] New file `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/TEST-PLAN.md` exists with all five sections plus per-section "what would invalidate this" lines.
- [ ] `SKILL.md` has the trailing pointer line and no other edits.
- [ ] `golangci-lint run` clean for modified files.
- [ ] `gofumpt -d` clean for modified files.
- [ ] No edits outside `acceptance_allow` globs (`git diff --name-only` audit).
- [ ] `vrooli scenario restart swarm-manager` completes cleanly.
- [ ] `curl -s http://localhost:<swarm-manager-api-port>/health` returns 200.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Go test duplicates the heartbeat's hash recipe; if heartbeat changes recipe, test passes silently against stale logic | Test asserts the recipe in a comment block + cites `HEARTBEAT.md` step 4 by name. Hash mismatch surfaces in CI within one prep cycle. |
| Manual smoke plan rots quickly | Plan colocated with the skill (`packs/core/workshop-decision-sync/TEST-PLAN.md`); SKILL.md trailing pointer keeps it discoverable. PRs that touch the skill see the plan immediately. |
| Heartbeat invocation in the smoke plan requires real Claude credit | Documented as such in TEST-PLAN.md; smoke plan is not run on every PR — only on prep-agent or sync-skill behavioural changes. Cheap at that cadence. |
| Test seeds drift from real workshop-round shape | Reuse canonical `workshop.Round` / `workshop.Item` types from `internal/workshop/` — same code path the production handler uses. |
| Async-clarify endpoint test flakes on agent mock setup | Reuse the proven `mockAgentService` pattern from `workshop_save_test.go` exactly; do not invent a new mock. |

## Cross-Initiative Implications

This is the closing chore of `workshop-decision-triage`. The three execute items it depends on are archived (per the round-002 feedback decision). Initiative has zero upstream and zero downstream initiatives (verified via `swarm-manager initiatives context`). No cross-initiative ripple. `morning-vision-walk` is intentionally decoupled (different cadence, different data source).

Sibling-overlap audit: no other member of this initiative covers the data-seam integration story or the UX smoke checklist — both deliverables are net-new. No `Update backlog item` candidate exists; `Create` is the right action shape.

Once this lands, mark the initiative shippable.

## Non-Goals / Prohibited Patterns

- **No real Claude/agent runs in CI** — the heartbeat invokes Claude through agent-manager; covering that with real LLM calls would be slow, costly, and flaky. The data seam is covered deterministically; the LLM enrichment narrative belongs in the manual smoke plan.
- **No re-test of the priority-sort or CLI flag forwarding** — already covered by `pending_questions_test.go` and `app_test.go`.
- **No UI smoke layer** — the UI workshop flow is out of scope (decided: `source=workshop` only, conversational surface).
- **No multi-user or concurrent-answering scenarios** — deferred per initiative orchestration summary.
- **No widening of `acceptance_allow`** — Phase-1 location (option d2=A) was chosen specifically to avoid widening; the test goes inside the existing `internal/backlog/**` glob.
- **No re-export of currently-private test helpers** — Phase-1 location avoids this; do not introduce exported helpers as a side effect.
- **No edits to SKILL.md beyond the single trailing pointer line** — SKILL.md is loaded into agent context on every skill invocation; bloat costs ongoing tokens.

## Definition of Done

The chore is done when **all** of the following are true:

1. `scenarios/swarm-manager/api/internal/backlog/workshop_decision_prep_integration_test.go` exists, contains the five sub-tests described in the Implementation Strategy, and passes via `go test ./scenarios/swarm-manager/api/internal/backlog/...`.
2. `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/TEST-PLAN.md` exists with all five sections (Setup, Inspection, Conversational Walk-through, Round-trip Proof, Pass/fail Rubric) and per-section "what would invalidate this" lines.
3. `SKILL.md` ends with exactly one new line pointing to `TEST-PLAN.md`; no other SKILL.md edits.
4. Hand-mutation checks (filter regression, hash-recipe regression) confirmed to fail the new test, then reverted.
5. `golangci-lint run` and `gofumpt -d` clean for all modified files.
6. `git diff --name-only` shows no edits outside `acceptance_allow` globs.
7. `vrooli scenario restart swarm-manager` completes cleanly; health endpoint returns 200.
8. Greenfield invariant respected: no compatibility shims, no dead code, no unused re-exports, no `// removed` comments.
