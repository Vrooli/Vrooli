# Workshop Decision Triage Sync Implementation Plan

## Purpose

Implement the full conversational workshop-decision flow as a professional, long-term solution without routing execution through additional Swarm Manager workshop rounds. This plan consolidates the existing initiative intent into one direct implementation artifact covering `swarm-manager` API/CLI work, the `workshop-decision-prep` director-swarm member, the `workshop-decision-sync` prompt-manager skill, and the required automated coverage.

## Hard Rules

### Greenfield Constraint

This implementation must be treated as greenfield product work, not a stopgap:

- No hacky shortcuts, temporary scripts, or “just make it work” glue.
- No compatibility shims or shadow flows unless a real existing contract already requires them.
- No new UI-only logic for ranking or decision orchestration when the contract belongs in API/CLI.
- No new question-level answer endpoint in this slice.
- Keep screaming architecture and clear seams: ranking, prep synthesis, and conversational skill behavior must each have an obvious home.

These rules are part of the Definition of Done, not just implementation guidance.

## Required Reading

Run this before implementation:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Recommended additional reading:

```bash
prompt-manager skill read skill-principles skill-validation
```

Primary repo context:

- `scenarios/swarm-manager/initiatives/workshop-decision-triage/orchestration-summary.md`
- `scenarios/swarm-manager/execute/swarm-manager-pending-questions-priority-sort/plan.md`
- `scenarios/swarm-manager/execute/workshop-decision-prep-agent/spec.json`
- `scenarios/swarm-manager/execute/workshop-decision-sync-skill/spec.json`
- `scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/SKILL.md`
- `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md`

## Problem Statement

Swarm Manager already stores per-backlog-item workshop decisions, but answering them currently depends on the Swarm Manager UI. That UI forces repeated initiative/item/context reconstruction and makes short operator decision sessions inefficient. The existing initiative `workshop-decision-triage` correctly identifies the solution: pre-stage the context off the critical path, then expose the decision loop as a conversational skill.

The current implementation gap is broad but well-bounded:

- `GET /api/v1/backlog/pending-questions` exists, but is currently raw and unordered, so it cannot reliably drive “top K most important decisions first”.
- There is no `swarm-manager backlog pending-questions` CLI wrapper for skills or heartbeat agents.
- The `workshop-decision-prep` member does not exist.
- The `workshop-decision-sync` skill does not exist.
- The original backlog specs contain at least one stale repo path: the prep-agent spec says to update `scenarios/prompt-manager/store/teams/director-swarm/TEAM.md`, but the actual team doc path is `scenarios/prompt-manager/store/teams/director-swarm/shared/TEAM.md`.

This plan replaces the slow backlog-roundtrip path with a direct implementation path while preserving the same long-term architecture the initiative was already converging toward.

## Scope

### In Scope

- Server-side ranking/filtering for pending workshop questions in `swarm-manager`
- Thin CLI wrapper for pending questions in `swarm-manager`
- New director-swarm member `workshop-decision-prep`
- New prompt-manager skill `workshop-decision-sync`
- Automated tests across API, CLI, prompt-manager member outputs, and skill-supporting seams
- Minimal documentation updates required to preserve discoverability and future handoff quality

### Out of Scope

- New question-level answer API
- Review-source conversational flow (`source=review`)
- General Swarm Manager UI redesign
- New initiative metadata mutation flows
- Reworking morning-vision-walk itself beyond borrowing its structure as a pattern reference

## Current Technical Context

### Existing Contracts and Primitives

- Pending questions endpoint exists at `scenarios/swarm-manager/api/internal/backlog/pending_questions.go`.
- Workshop round save exists at `scenarios/swarm-manager/api/internal/backlog/workshop_save.go`.
- Routes already exist for:
  - `GET /api/v1/backlog/pending-questions`
  - `POST /api/v1/backlog/{kind}/{name}/workshop/save`
  - clarification endpoints under `/workshop/clarification/...`
- Clarification CLI already exists at `scenarios/swarm-manager/cli/cmd_clarification.go`.
- UI-side ranking logic already exists in:
  - `scenarios/swarm-manager/ui/src/lib/dependency-sort.ts`
  - `scenarios/swarm-manager/ui/src/lib/backlog-sort.ts`
- UI workshop-save client already exists in:
  - `scenarios/swarm-manager/ui/src/services/backlog/workshop-service.ts`
  - `scenarios/swarm-manager/ui/src/hooks/useDecisionStreamLogic.ts`

### Existing Patterns to Reuse

- Skill metadata/content shape: `scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/`
- Prep-member file layout and handoff discipline: `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/`
- Early-return queue discipline for members: `scenarios/prompt-manager/store/teams/director-swarm/members/portfolio-manager/HEARTBEAT.md`

### Current Evidence

- `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/` does not exist.
- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/` does not exist.
- `pending_questions.go` currently emits items in `LoadAll(nil)` order and performs no priority sort or server-side `source/limit/initiative` filtering.
- The `director-swarm` team file layout uses:
  - `scenarios/prompt-manager/store/teams/director-swarm/team.json`
  - `scenarios/prompt-manager/store/teams/director-swarm/shared/TEAM.md`

## Target End State

After implementation:

- `swarm-manager backlog pending-questions --source workshop --limit K --json` returns deterministic, priority-ordered workshop decision groups suitable for both heartbeat prep and skill fallback.
- `workshop-decision-prep` runs every 3 hours, computes/stabilizes briefing entries, and writes a durable `last-handoff.md`.
- `workshop-decision-sync` gives the operator a one-decision-at-a-time conversational surface with explicit focus-stack behavior, skip controls, and async clarification spawn.
- Answered questions persist through the existing fetch-patch-save workshop-round flow and disappear from future prep/skill sessions once `Selected` is populated.
- The implementation is clean enough that future agents can extend it to `source=review` or a later question-level endpoint without rewriting the architecture.

## Implementation Strategy

### Phase 1: Canonical Pending-Question Ranking and CLI

Implement the `swarm-manager` contract first. The prep member and the skill both depend on it.

Deliverables:

- Add a dedicated shared ranking seam in `scenarios/swarm-manager/api/internal/backlogrank/`.
- Port the ranking semantics from the UI TypeScript into Go:
  - dependency depth first
  - effective priority using transitive incomplete dependents
  - recency descending
  - stable final tiebreak on `(kind, name)`
- Extend `pending_questions.go` to support:
  - `source=workshop|review|all` with default `workshop`
  - `limit=N`
  - `initiative=NAME`
- Add CLI wrapper:
  - `swarm-manager backlog pending-questions [--source ...] [--limit N] [--initiative NAME] [--json]`

Why this package placement:

- `api/internal/backlogrank` is a better long-term seam than embedding the ranker directly in `backlog/`.
- It keeps ranking reusable by `overview`, `stats`, or future scenario surfaces without forcing them to import the backlog handler package.
- It preserves screaming architecture better than a generic helper file and is more future-proof than burying ranking in one handler.

### Phase 2: Prep Specialist as Read-Only Synthesis Layer

Create the new prompt-manager member:

- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/heartbeat.json`
- `.../HEARTBEAT.md`
- `.../RESPONSIBILITIES.md`
- `.../last-handoff.md`

Also update:

- `scenarios/prompt-manager/store/teams/director-swarm/shared/TEAM.md`

Behavioral contract:

- Read `swarm-manager backlog pending-questions --source workshop --json`.
- Build canonical hashes from decision content, not timestamps.
- Reuse cached briefs when still valid.
- Early-return when the handoff already contains enough valid briefs.
- Enrich each brief with initiative summary, backlog summary, anticipated Q&A, and any resolved clarification `LatestImpact.ContextNote`.
- Produce narrative grouping by initiative -> backlog item -> decision.
- Remain strictly read-only with respect to backlog items and initiatives.

Implementation note:

- Do not mirror the more elaborate walk-checkpoint logic from `vision-walk-prep`; that logic is specific to resumable walk state, not to workshop decision prep.
- Reuse the member file layout and “do not perform side effects” discipline, not the full handoff schema.

### Phase 3: Conversational Skill

Create the new skill:

- `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/skill.json`
- `.../SKILL.md`

Behavioral contract:

- Read the prep handoff first.
- Revalidate each brief live against fresh pending-questions JSON before using it.
- Drop stale or already-answered decisions.
- If the handoff is missing or stale, run lazy inline prep for a smaller batch.
- Present one decision at a time, preserving current focus until the current decision is resolved or skipped.
- Support:
  - answer-and-continue
  - skip-item
  - skip-initiative
  - clarification spawn and continue
- Persist answers by fetching the round JSON, patching the target decision item, and saving through the existing `/workshop/save` contract.

Implementation posture:

- This skill should behave like a professional operator workflow, not a freeform brainstorming prompt.
- It must explicitly prohibit answering on the operator’s behalf or mutating adjacent backlog/initiative data.

### Phase 4: Cross-Layer Test and Validation Closure

Close with automated coverage, not manual confidence.

Deliverables:

- Go tests for `backlogrank`
- Extended `pending_questions_test.go`
- CLI tests for the new subcommand
- Tests for prep-member output generation seams where practical
- Skill-support tests and/or deterministic smoke harness for the freshness/drop behavior defined by the initiative

This phase should also reconcile any stale assumptions in the existing item-level specs so future work reads the direct implementation as the new source of truth.

## Contract Decisions

### 1. Source Scope

MVP conversational flow is `source=workshop` only.

Rationale:

- This is already the initiative’s intended scope.
- Review-source questions are a different operator workflow and would dilute the first release.

### 2. Answer Persistence Path

Do not introduce a new question-level answer endpoint.

Use:

1. fetch current round JSON
2. patch target decision item
3. save via `/workshop/save`

Rationale:

- This keeps one source of truth for round persistence and auto-advance behavior.
- It avoids inventing a second mutation contract prematurely.

### 3. Ranking Ownership

Ranking becomes server-owned for pending-question grouping.

The UI may continue to sort defensively, but after this work it should be effectively applying a no-op for already-canonical server output.

### 4. Prep Handoff Shape

The prep handoff is the canonical conversational staging artifact for this workflow.

It should be:

- durable
- human-readable
- machine-checkable enough to support hash validation and selective refresh

### 5. Clarification Model

Clarifications remain asynchronous.

The skill may spawn them and continue, but it must not block the conversation waiting for them to resolve.

## Testing Plan

### API

- Add unit tests for ranking primitives:
  - depth resolution
  - unblocking calculation
  - effective priority
  - cycle handling
  - final stable ordering
- Extend pending-questions tests for:
  - workshop-only filtering
  - `source=all`
  - `initiative` filtering
  - `limit`
  - stable sorted output
  - invalid query params

### CLI

- Add tests for:
  - argument parsing
  - `--json` passthrough
  - summary rendering
  - invalid `--source`

### Prompt-Manager Member

- Add deterministic tests or fixture-driven coverage for:
  - early-return when enough valid briefs exist
  - selective refresh when a single decision hash changes
  - dropping answered decisions
  - inclusion of clarification context notes

### Skill

- Add coverage for:
  - stale-brief drop behavior
  - lazy-prep fallback
  - focus-stack behavior
  - skip-item / skip-initiative transitions
  - clarify-spawn then continue

### Scenario-Level Validation

Use scenario-aware validation commands where appropriate:

- `vrooli scenario test swarm-manager`
- focused `go test` targets under `scenarios/swarm-manager/api/...` and `scenarios/swarm-manager/cli/...`
- focused prompt-manager tests if there is an existing harness for member/skill behavior

If no robust prompt-manager harness exists for the new member/skill surfaces, create targeted deterministic tests rather than relying on manual operator walkthroughs.

## Rollout / Validation Checklist

1. `pending-questions` returns deterministic priority order for workshop items.
2. CLI wrapper exists and mirrors the API without hidden business logic.
3. `workshop-decision-prep` is registered in `director-swarm` and writes a valid `last-handoff.md`.
4. `workshop-decision-sync` can consume the handoff and also survive missing/stale handoff state.
5. An answer saved through the skill path disappears from future prep/skill sessions.
6. Clarification spawn works without blocking the session.
7. Tests pass for API, CLI, and prompt-manager artifacts introduced by this work.
8. The final architecture still matches the initiative’s long-term intent and does not require a cleanup follow-up just to become “proper”.

## Risks and Mitigations

### Risk: Go and TypeScript ranking semantics drift

Mitigation:

- Port the TS logic directly.
- Lock behavior with ordering tests and UI-regression verification.

### Risk: Prep handoff becomes too prose-heavy or unstable

Mitigation:

- Keep a predictable section structure for each brief.
- Keep the hash input based only on canonical decision content, not on enriched prose.

### Risk: Skill instructions become too fuzzy to execute consistently

Mitigation:

- Write the skill like an operator protocol, not inspirational prose.
- Explicitly enumerate allowed actions and forbidden actions.

### Risk: Existing execute specs contain stale repo facts

Mitigation:

- Treat current specs as intent documents.
- Prefer live repo structure when conflicts arise.
- Record those corrections in implementation comments or supporting docs where needed.

## Non-Goals / Prohibited Patterns

- No direct backlog-item or initiative mutation from the prep member.
- No auto-answering for the operator.
- No UI-only implementation of priority ordering.
- No ad-hoc shell scripts replacing proper CLI surface.
- No second persistence path for workshop answers.
- No compatibility layer for an imagined older prep/skill format that does not actually exist.

## Definition of Done

This work is done only when all of the following are true:

- The full initiative slice is implemented, not just the first API item.
- The implementation is greenfield-clean and professionally organized.
- The ranking seam is explicit and reusable.
- The CLI is a thin, well-tested wrapper over the API.
- The prep member is read-only, deterministic, and handoff-driven.
- The skill is conversational but operationally strict.
- Automated tests cover the ranking, filtering, prep freshness, and save/clarify paths.
- A future agent can continue or extend this work from the repo and this plan file alone, without needing the original chat.

## Suggested File Targets

Primary new/modified files expected in implementation:

- `scenarios/swarm-manager/api/internal/backlogrank/`
- `scenarios/swarm-manager/api/internal/backlog/pending_questions.go`
- `scenarios/swarm-manager/api/internal/backlog/pending_questions_test.go`
- `scenarios/swarm-manager/cli/cmd_backlog.go`
- `scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/`
- `scenarios/prompt-manager/store/teams/director-swarm/shared/TEAM.md`
- `scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/`

## Resume Point

If implementation begins later, start with Phase 1 and do not skip ahead:

1. create `backlogrank`
2. land server-side pending-question sort/filtering
3. land CLI wrapper
4. then create prep member
5. then create skill
6. then close with tests
