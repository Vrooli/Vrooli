# Flows — API Health

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Validate target scenario | validation | CLI/API/Test Genie calls `ValidateScenario`. | Assessment, native detail, and optional fixability metadata. | Request-scoped; no durable state. | Unit and provider-contract tests planned. |
| Live health probe | probe | Validation request includes execution or CLI runs `probe health`. | Timestamped probe evidence and schema findings. | Request-scoped with timeout/cancel; no retry loop. | httptest and lifecycle-discovery tests planned. |
| Fix preview/apply | remediation | CLI/API calls PreviewFix or ApplyFix with rule IDs. | Dry-run candidates or explicit target-file edits. | Request-scoped mutation with idempotency requirement. | Autofix tests planned. |
| Scenario-auditor migration accounting | migration | Developer updates migration ledger or parity fixtures. | Every legacy API rule classified exactly once. | Version-controlled ledger; no runtime state. | Ledger completeness test planned. |

## Flow Details

### Validate target scenario

1. Resolve target scenario path and service metadata.
2. Classify API applicability.
3. Load API source surfaces and endpoint metadata where present.
4. Run static validators grouped by capability.
5. Optionally call the live health probe flow.
6. Map findings through `.vrooli/maturity.json`.
7. Return shared assessment, native detail, status, and metrics.

Failures are reported as findings when target evidence is readable but
non-compliant, and as provider `ERROR` only when API Health itself cannot
execute the validation contract.

### Live health probe

1. Resolve the target API health URL through lifecycle metadata.
2. Build an HTTP request with a strict timeout and request context.
3. Execute exactly one probe; no polling and no process start outside lifecycle.
4. Validate status code, JSON content type, and health response schema.
5. Attach bounded evidence to native detail.

The probe never becomes a performance benchmark and never exercises product
endpoints by default.

### Fix preview/apply

1. Select fixers by requested rule IDs.
2. Recompute current findings and candidate edits.
3. Preview returns diffs without writing.
4. Apply writes only explicit candidates and reports what changed.
5. Second apply should be a no-op for the same candidate.

### Scenario-auditor migration accounting

1. Inventory old API rule IDs from `scenarios/scenario-auditor/api/rules/api/`.
2. Classify each as kept, redesigned, delegated, deferred, or rejected.
3. Add fixture coverage for migrated expectations.
4. Cut old rules only after API Health coverage is documented and verified.

## State Machines

The current P0 flows are request-scoped and do not require a formal state
machine. Fix apply has an idempotency invariant but no durable lifecycle state.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| remediation/fix | previewed, applied, skipped | apply without explicit request; second apply changing files again | planned autofix tests |

## Maturity Ladder

Temporal workflows mature in layers. API Health starts with request-scoped
flows; add formal flow models only if durable state or retry/cancel complexity
appears.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

If a future domain introduces durable workflow state, place its `flow/`
subdirectory beside that domain or UI feature and follow the standard
flow-verifier layout. Until then, keep validation, probe, and fix flows as
plain service tests rather than adding unnecessary formal artifacts.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Fleet readiness sweep | Could need background scheduling or persisted history. | Model when OT-P2-001 starts. |
| Probe history retention | Could introduce stale evidence and deletion semantics. | Model before persisting probe results. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
