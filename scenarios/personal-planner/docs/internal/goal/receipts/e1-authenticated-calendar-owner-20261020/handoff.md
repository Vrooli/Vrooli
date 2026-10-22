# Authenticated calendar event ownership slice handoff

Captured: 2026-10-21T00:04:22Z
Directive: ORCH-E1-ACTOR-NEW-CALENDAR-EVENT-OWNERSHIP-20261020
Worker task/run: cbad6395-7d01-4c2f-8e2b-ad41de6a471f / a7df133f-03ca-4217-b451-aa3dac6c8445
Parent run: 5d07e651-eaa6-48ec-89b4-5dc359059193
Anchor: gcta-790809a33666; checkout HEAD: 52097258290d1bff0191319242d4948e214af9e3

## Implementation

Changed only the five admitted API source/test files. Added an additive `calendar_event_owners` relation; existing event records are not assigned owners. Verified owners come from `identity.PrincipalFromContext(ctx)` and `Principal.OwnerSubject()`. Event create, owner relation, and idempotency record commit in one transaction. Owner-filtered list/get/idempotency lookup/update conceal other owners and rows lacking a relation. Cross-owner key reuse conflicts without returning the event; provider identity collisions do not adopt an unowned or other-owner record. Revision checks happen only after owner authorization. Existing event date/time and revision behavior remains covered by the SQLite tests.

The retained synthetic handler test now creates through REST, reads the same ID/revision through Connect, and checks owner B cannot list/get/update it on REST or Connect. It also seeds a legacy row without an owner relation and checks owner A and B cannot read/update it.

## Validation

Command: `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`
Exit code: 1. `internal/calendar` passed. The handler test passed its owner enforcement assertions and then failed while its existing CLI build compiled pre-existing dirty `cli/domains/goals/handlers.go:180`: `m.Goal.ProgressBasisPoints / 100` has type `*int64`. That file is outside this directive's write set and was already dirty before this slice. Current generated `goals.Goal.ProgressBasisPoints` is `*int64`; the failing source uses direct division at line 180, while the generated type exposes `GetProgressBasisPoints()`. Smallest amendment to clear this unrelated gate failure: authorize a one-line fix in `scenarios/personal-planner/cli/domains/goals/handlers.go` to use the getter, then rerun the prescribed focused command. No CLI or Proto verifier/fix was run because the admitted brief explicitly limits validation to the focused command and protects CLI/Proto paths. Exact captured command output is `focused-gate.log` (SHA-256 8230703ff6b41e4a523709e10ba4cbe1cafe9b661c267324e9caa7eac2a80e5c). The focused gate therefore remains unmet; no CLI/Proto change or secondary validation was performed.

## Baseline and growth

Admitted baseline manifest SHA-256: `01f6871225b82ef6eb5f377dda323c52e98c5ada70aba689c75e4960331b6efa`; it matched all five baseline files at start. Current manifest SHA-256: `08b52d0247314345d319d065529b0b7e76f50f89d1b3c048cb75f8169b090127`. It records exact before/after hashes, bytes, and line counts for every allowed source path.

Runtime line delta: +53 (events.go +14, events_sqlite.go +32, schema.sql +7). Test line delta: +200 (events_test.go +98, connect_events_test.go +102), exactly the hard test-growth limit. Cumulative E1 runtime growth is +1,784, below the +2,100 ceiling. No files/symbols were deleted. Existing unrelated dirty files were left as found.

Epoch check: `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs a7df133f-03ca-4217-b451-aa3dac6c8445 --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193` exited 0. Full output is retained in `epoch-check.log` (SHA-256 `d41b1dc2675ace7f7316d2e0402a69f482c2b3086f41588252612092f3324d96`). It reported 42 work units, no open directives, no flatline/overrun/growth step-back, the legacy-gates warning, and its historical cumulative runtime figure +1,731; the added source delta measured from the admitted baseline is +53 runtime lines. Several earlier slice-log records were reported malformed; this new line was not reported malformed. This check does not accept the slice or E1.

## Remaining limits

The focused gate did not exit successfully because of the pre-existing protected CLI build error. This slice establishes backend REST/Connect owner enforcement only; live UI/CLI actor propagation and full cross-surface parity remain open E1 work. No existing event/provider record was mapped, backfilled, claimed, modified, or exposed. No acceptance was written; independent review, parent mechanical acceptance, and E1/Q1 gates remain separate and open.
