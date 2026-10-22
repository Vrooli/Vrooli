# Planner I18 E1 worker brief — authenticated ownership for newly created native calendar events

Worker card: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`. Full epoch file: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`. Parent/orchestrator exact native run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; task: `790cb264-1a63-47eb-89e6-3b9ee3e16c23`. Directive: `ORCH-E1-ACTOR-NEW-CALENDAR-EVENT-OWNERSHIP-20261020`.

## Approved outcome and evidence

This bounded slice advances the already approved E1 outcome and Planner workspace contract: authenticated requests create and access native calendar events within the verified owner's scope across the existing REST/API, Connect and supported CLI paths. The current qualification receipt `receipts/e1-actor-workspace-qualification-20261020.md` shows that the shared API context carries a verified principal but Planner's calendar event repository does not use it. The security contract places workspace selection and per-record authorization in Planner. `identity.Principal.OwnerSubject()` is the shared verified owner-resolution API.

This slice handles **newly created native calendar events only**. Existing calendar rows have no verified owner and must remain unowned, unreadable and unmodifiable through an authenticated owner's path. Do not backfill, claim, expose, update, delete, migrate or otherwise assign any existing event or provider record. Do not answer or sync the pending legacy-record mapping decision. The current operator instruction says it remains pending. The separate-until-mapped signal observation does not authorize any data mapping in this task.

## Measured scope and budget

Anchor: `gcta-790809a33666`, scope `scenarios/personal-planner/**`, checkout head `52097258290d1bff0191319242d4948e214af9e3`. This anchor fingerprints the current dirty scenario state; do not restore or rewrite any unrelated byte.

Exact allowed write set (all other product paths are protected):
- `scenarios/personal-planner/api/internal/calendar/events.go`
- `scenarios/personal-planner/api/internal/calendar/events_sqlite.go`
- `scenarios/personal-planner/api/internal/calendar/schema.sql`
- `scenarios/personal-planner/api/internal/calendar/events_test.go`
- `scenarios/personal-planner/api/handlers/calendar/connect_events_test.go`

Baseline manifest SHA-256: `01f6871225b82ef6eb5f377dda323c52e98c5ada70aba689c75e4960331b6efa`; exact bytes are in `baseline/bytes/`. Baseline lines in this slice: 162 + 254 + 146 + 283 + 404 = 1,249 runtime/test lines total (the SQL schema is counted with runtime for the estimate boundary). Current cumulative E1 growth is +1,731 runtime lines against the +2,100 ceiling, leaving 369 runtime lines. Estimate +60 runtime/+120 test; hard stop at +120 runtime or cumulative +2,100 runtime, and at +200 test. No deletions. If the safe path requires existing-table backfill/alteration, another domain, API-core/auth, UI changes, or exceeds the hard stop, make no such expansion; return the evidence and the smallest bounded amendment needed.

## Required behavior

- Resolve the owner only from a verified Principal in request context using the shared owner-subject contract. Never trust event payload `subject`, tenant IDs, headers or CLI flags as ownership.
- Persist ownership for a newly created event atomically with the event and idempotency record. The schema change must be additive for the existing SQLite database: use a new owner-relation table if needed; no backfill or changes to existing rows.
- List, get, get-by-idempotency-key, create/replay and update must enforce the same owner. A different actor and an event without an owner relation must receive empty/not-found behavior without learning event contents. A stale revision remains a conflict only after owner authorization.
- A retry by the same owner must return the same event. The same client idempotency key used by another owner must not return or mutate the first owner's event. Existing provider identity collisions must remain conflicts and must not adopt an unowned/other-owner row.
- Use the shared request context already passed by REST/Connect handlers. Extend the retained synthetic REST/Connect handler tests to prove the same new event ID/revision for one verified owner through those handlers, and prove another verified owner cannot list/get/update it. Include an unowned legacy fixture and prove the authenticated owner cannot see or change it. Preserve the existing event date/time/revision behavior. Do not modify CLI or Proto paths. State plainly that this slice establishes backend REST/Connect owner enforcement only; live UI/CLI actor propagation and full cross-surface parity remain open E1 work.

## Validation and safety

Run only `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`; retain exact output, before/after hashes and line counts in this receipt directory. Tests must use the existing in-memory or routed test storage and synthetic principals only. Do not start the application against ordinary user storage, run a migration on product data, write provider data, run a broad Test Genie suite, modify Proto, install dependencies, alter profiles or weaken auth. Append your slice-log line and run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Do not write acceptance. Independent review and parent mechanical acceptance are separate; only this bounded behavior could be accepted, and E1/Q1 remains NOT ACCEPTED.

## Handoff

Report changed files, implementation behavior, exact validation output/exit code, before/after hashes and line counts, actual growth, and all limits or blockers. Keep legacy records and unrelated work untouched.
