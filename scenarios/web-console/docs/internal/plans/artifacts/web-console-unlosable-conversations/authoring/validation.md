Validation is evidence-driven and layered. Tests MUST assert the desired lifecycle contract, including post-finalization state, rather than preserving today’s accidental behavior.

## Contract and traceability

1. Repair PRD and requirement wording before implementation. Every requirement uses measurable EARS-style acceptance language and maps to tests through `[REQ:*]` tags.
2. Validate the requirement registry and generate a requirements report. The continuity capability must reach the repository’s required traceability level, including unit, integration, BAS, and operational evidence where applicable.
3. Validate protocol descriptors and generated clients after every contract change. Reject hand-maintained duplicate DTOs.

## Static and structural gates

- Compile and lint every affected Go, TypeScript, shell, and protocol surface using repository-native commands.
- Add ratchet tests that enumerate destructive persistence entry points. Only the canonical lifecycle owner and explicitly named reconciliation/retention components may mutate lifecycle records.
- Search for and eliminate ignored errors, delay-based cleanup, direct `Store.Delete`, legacy archive queries that inner-join through `sessions`, and duplicate provider-specific lifecycle implementations.
- Validate storage ownership and schema registration with `storage-manager validate scenario web-console`.

## Automated test pyramid

| Layer | Required evidence |
|---|---|
| Pure unit/property | State-machine transition matrix; illegal transitions; canonical identity normalization; cursor stability; confidence classification; topic derivation; receipt serialization |
| Repository integration | Transaction rollback, uniqueness, idempotency, monotonic revisions, soft delete/tombstone behavior, FTS/index semantics, mixed legacy/canonical rows |
| Service integration | All close/archive/undo/reopen/recover/delete callers route through the owner; provider artifact preservation; retry and restart behavior |
| Fault injection | Crash before/after each durable step, database failure, provider outage, Search Hub unavailable, Agent Manager unavailable, duplicate and reordered requests |
| Concurrency | Close vs archive, close vs reconnect, expiry vs reconnect, archive vs undo, reconciliation vs live write, delete vs search publication |
| API/CLI contract | Stable JSON, exit codes, pagination, filters, receipts, dry-run/apply distinction, actionable typed failures |
| UI component | All-state archive visibility, exact search, integrity warning, recovery flow, mobile layout, keyboard/accessibility, visible failure and retry |
| BAS end-to-end | Real process lifecycle through supported scenario commands, backend restarts, browser reload, archive/undo/reopen, exact recovered-thread lookup |
| Migration/reconciliation | Production-shaped fixture and an immutable copy of the live corpus; conservation manifest; ambiguous quarantine; second run performs zero mutations |
| Federation | Web Console source records appear in Agent Manager and Search Hub; updates/tombstones converge; local search remains independent |

## Mandatory incident regression

The original incident must become a named deterministic fixture and an optional operator-only live verification:

- pane ID `a7e71c3c-e422-4c89-916a-03f92906fb89`;
- thread ID `01a06a6b-88da-7422-b391-bb59c5f5e5e0`;
- transcript with 88 conversation events, a checkpoint, and a workspace pane but no session row;
- exact and partial queries from the recovered discussion must locate the conversation;
- inspection must explain its lineage and repair action;
- applying reconciliation twice must preserve counts and hashes on the second run.

Live-state evidence MUST be sanitized: store counts, hashes, classifications, commands, timestamps, and bounded redacted snippets—not raw prompt bodies or secrets.

## Operational proof

1. Capture a pre-change baseline collection using the Test Genie protocol.
2. Run narrow suites after each phase, then run Web Console comprehensive validation.
3. Run affected Agent Manager and Search Hub integration suites after the federation phase.
4. Start scenarios only through `make start` or `vrooli scenario start`; never execute binaries directly.
5. For every server-owned test run, invoke `test-genie runs wait --json <scenario> <run-id>` once; do not poll, and do not confuse client cancellation with abort.
6. Compare the final collection against the baseline and classify every delta. No unexplained regression is acceptable.
7. Exercise backup, apply, interruption, resume, and rollback on a disposable corpus before touching live state.
8. Produce a final conservation report: sessions, sources, events, artifacts, lifecycle receipts, recoverable conversations, quarantined cases, tombstones, and content hashes before/after.

## Evidence package

The executor must attach or link from Plan Manager:

- validated requirement report and traceability map;
- architecture decision record, lifecycle matrix, invariants, and temporal flows;
- generated protocol descriptor diff;
- baseline and final Test Genie collection IDs;
- unit/integration/BAS/fault/concurrency results;
- dry-run, apply, second-run, and rollback reconciliation reports;
- sanitized before/after integrity and conservation reports;
- CLI JSON captures and desktop/mobile UI screenshots;
- Agent Manager/Search Hub federation and tombstone evidence;
- skill validation and governed-program transcript;
- deletion inventory proving obsolete paths were removed;
- work record with trigger, approach, evidence, and outcome.

