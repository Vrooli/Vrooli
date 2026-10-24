# E1 next slice — API idempotent retry

Parent run: `16d59037-b9a9-4442-8ae7-0c28921bd54b`; parent task: `27847f20-3191-47d9-b74e-76df6ef918f8`; selected effort/revision: `nooch-selected-delivery-2026-10-03` / `nooch-selected@2026-10-03:b013666f214be361`.

Pre-dispatch proposal anchor: `gcta-d36c7299521d` (E1, scope `scenarios/nutrition-planner/**`, HEAD `52097258290d1bff0191319242d4948e214af9e3`).

## Observed gap

Current repository `api/internal/planning/sqlite.go` records an apply receipt keyed by workspace, expected revision, and canonical payload, and its tests prove identical and late retry behavior. The Connect handler `api/handlers/planning/connect_handler.go` currently reads the current revision and returns stale conflict before calling the repository. After a successful commit whose response was lost, the identical retried request can therefore be rejected before the receipt is consulted. Confirm this call order and any current API test coverage before changing code.

## Assignment

Implement a maintainable API path that returns the original applied revision for an exact already-committed retry while preserving validation for new writes. A changed draft at the old expected revision must still return the existing stale conflict and cause no mutation. Preserve atomic occurrence persistence, selected recipe revisions, date-range preservation, lock rules, review-input checks, ownerless public behavior, and ACT-036 foreign-workspace integrity. Do not add auth, a new request field, migration, dependency, or BAS workflow mutation unless the current API contract proves it necessary.

Deliver one coherent slice: source change, focused handler/repository regressions, focused API validation, and one isolated R25 connected receiving that explicitly applies then repeats the identical request payload/expected revision (as if the first response were lost) and reads back the same workspace, revision and occurrence identities. The receiving must retain the original selected revision and route, exact BAS/flow/run IDs, source/served-build evidence, fixture-only network/write audit, and cleanup receipts. Preserve the first-useful/new-week contracts and all original E1 gates; do not bundle visual comparison, lifecycle restart, or unrelated behavior.

## Gates and limits

- Use existing planning/API owners and the normal interactive E1 worker profile, medium Luna model, under this parent and existing E1 child task if native task state permits. Verify no live duplicate before admission. Do not create a leader, alter effort/revision, or treat task/run completion as acceptance.
- Focused tests must assert expected behavior, including identical retry after commit, changed stale retry, exactly one revision increment, stable persisted occurrence identities, and preserved unaffected dates.
- Use only a fresh isolated temporary routed fixture for connected writes. No primary-data or external-provider writes. Stop/clean only through fixture and BAS owners.
- If the exact retry is not supported by the existing API semantics, do not weaken concurrency checks; return the source-grounded blocker and options to the parent.
- Return changed/verified/unverified/remaining with exact receipts and measured immediate-preimage deltas where available. Independent review remains with the parent; E1 and all epochs remain unaccepted.

## Worker prompt

Read `scenarios/nutrition-planner/docs/internal/goal/epochs/E1.md` and this brief. Follow the large-effort worker card at `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`. Work in the existing shared checkout. Keep the assignment above as your complete scope, validate the behavior you change, and report exact evidence and unknowns. Do not read `FEEDBACK.md`, accept E1, change E30, or write primary/provider data.
