# Workshop Decision Triage Sync — Initiative Context

## Strategic Rationale
The swarm-manager workshop flow stages per-item decisions for the user to answer. Answering via the UI requires heavy per-question context reconstruction (initiative → item → topic → options → recommendation), so it goes underused even when it would unblock downstream agent work. An agent-led, eagerly-prepped conversational flow flips the cost: the agent does the context-loading off the critical path (every 3h heartbeat) and the user mostly reacts during a 5-minute session.

This complements morning-vision-walk without coupling. Vision walk covers prompt-manager team decisions (portfolio, marketing-crew, meta-optimization) once a day with broad strategic framing. This initiative covers swarm-manager workshop decisions (per-backlog-item multiple-choice questions) on demand, any time.

## Cross-Item Decisions

- **Source scope**: MVP covers `source=workshop` pending questions only. `source=review` (unreviewed targets/requirements) is a separate UX and deferred.
- **Priority ranking**: extend `pending-questions` endpoint server-side rather than client-side — any other consumer (UI, future specialists) benefits automatically. Reuse the existing ranker in swarm-manager; do NOT duplicate.
- **Freshness model**: SHA-256 content hash over canonical-form topic+context+options, not `updated_at`. No schema change. `Selected != nil` is the UI-answered-elsewhere signal.
- **Clarifications are async**: the sync skill spawns `backlog clarify` and immediately skips. Next prep run reads the resolved thread's `LatestImpact.ContextNote` and feeds it forward as pre-answer material.
- **Answer path**: fetch-patch-save the full round JSON via `/workshop/save`. A question-level endpoint is tracked as an idea and only lands if the fetch-patch-save pattern proves clunky.
- **Prep specialist discipline**: read-only. No side effects on backlog items, initiatives, or workshop rounds. Pure enrichment into `last-handoff.md`.

## Sequencing Notes

Within this initiative:
1. `execute/swarm-manager-pending-questions-priority-sort` MUST land before the prep agent — the prep agent's top-K selection assumes server-side priority ordering.
2. `execute/workshop-decision-prep-agent` MUST land before the skill — the skill's default path reads the prep handoff; lazy fallback is slower and shouldn't be the norm.
3. `execute/workshop-decision-sync-skill` can land before `chore/workshop-decision-sync-tests`; tests are the last closer.
4. `idea/workshop-question-level-answer-endpoint` stays unscheduled until skill implementation surfaces concrete friction.

## Deferred / Out of Scope

- Review-source questions (targets/requirements).
- Question-level answer endpoint (tracked by the idea item).
- Multi-user/multi-session concurrent answering on the same round (not a current problem; would be addressed by the idea item if it arises).
- Integrating with morning-vision-walk (intentionally decoupled; different data sources, different cadence).

## Implementation Reference Patterns

- Early-return-at-queue-full: `scenarios/prompt-manager/store/teams/director-swarm/members/portfolio-manager/HEARTBEAT.md`
- Member file layout: `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/`
- Skill-consumes-prep-handoff: `scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/SKILL.md`
- Existing endpoint to extend: `scenarios/swarm-manager/api/internal/backlog/pending_questions.go`
- Async clarification CLI used by the skill's escape hatch: `scenarios/swarm-manager/cli/cmd_clarification.go`

## Verification (end-to-end after all items land)

1. `swarm-manager backlog pending-questions --source workshop --json --limit 5` returns priority-ordered output matching the UI's order.
2. Trigger a manual heartbeat of `workshop-decision-prep`; inspect `last-handoff.md` — confirm initiative-grouped briefs, options with recommendation flags, content hashes, anticipated Q&A blocks.
3. Answer one workshop decision via the UI; re-invoke the skill; confirm that decision no longer appears.
4. Mutate a decision's options; re-run the skill; confirm that brief is dropped or refreshed.
5. Invoke the skill and walk through 3 decisions across 2 initiatives; confirm initiative/item intros printed once per transition, focus stack prevents clarifying-question topic pivot, skip-item and skip-initiative work, answer persists to the UI via `workshop/save`.
6. During a session, ask an unanticipated question; accept clarification spawn; confirm skill moves on without waiting; 10+ minutes later run the prep heartbeat; confirm the resolved `ContextNote` appears in the next brief.
7. Delete `last-handoff.md`; invoke the skill; confirm inline lazy prep runs with a smaller cap and session proceeds.
8. Restart swarm-manager (`vrooli scenario restart swarm-manager`) after the API change — but never restart any scenario the active Claude Code session is running inside.
