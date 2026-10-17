# Flows — Template Manager

Template Manager has several stateful workflows because generation, validation, drift, autofix, and scheduled deep validation all cross process, filesystem, and test-genie boundaries.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Validation run | validation | CLI/API request or scheduler. | Persisted run with parsed findings and debt updates. | queued, running, succeeded, failed, canceled, timed_out. | Unit fake-runner tests plus live shallow/deep smoke. |
| Drift snapshot | validation | CLI/API drift request or scheduled run. | Fleet/template drift records and version lag. | running, succeeded, failed. | Parser/persistence tests and live drift proof. |
| Provenance autofix | phase-provider | test-genie fix preview/apply. | Target scenario gains adopted generation provenance. | preview, applicable, applied, no-op, failed. | Provider-contract live probe and disposable target apply. |
| Orientation guidance | guidance | API/CLI request for a target scenario. | Next incomplete gate with check contract. | computed from current target files; no long-running state. | Fixture tests for pre-finalize and finalized scenarios. |
| Deep-validate monitor | monitor | Scheduler interval. | Serialized deep validation run per registered template. | idle, due, running, skipped_busy, succeeded, failed. | Short-interval scheduler integration test. |
| Lifecycle engine operation | engine | API/CLI generate/orient/detemplate/validate/drift/cleanup/design/resource-template command. | Filesystem changes or lifecycle report. | operation-specific start, progress, success/failure. | Carried engine suites plus throwaway e2e proof. |

## Validation Run Flow

1. Caller requests a shallow/deep validation or drift run.
2. Validation service creates a run row with mode, target, template id, source, and runner metadata.
3. Runner executes the current engine path: subprocess seam before cutover, in-process engine after cutover.
4. Parser converts JSON output into phase results and findings.
5. Repository stores immutable run evidence.
6. Debt mapper upserts stable debt entries without duplicating repeated findings. A failed deep run resolves only superseded Test Genie summary entries before upserting its current failure-class entry; source and fleet debt retain their independent lifecycle.
7. API/CLI/UI expose run status and details.

Failure behavior: runner timeout or parse failure marks the run failed and records diagnostic text under a stable Test Genie failure class; it must not corrupt unrelated debt state.

## Provenance Autofix Flow

1. ValidateScenario reads target `.vrooli/service.json`.
2. Missing generation provenance returns L0 `PROVENANCE_MISSING` with an auto fixer.
3. PreviewFix computes the adopted-provenance patch without writing.
4. ApplyFix writes the latest default template id/version and marks the provenance as adopted, not generated.
5. A second preview/apply returns no-op.

Invariant: the templates phase stays static and cheap; it never starts deep validation.

## Monitor Flow

1. Scheduler wakes on configured interval.
2. If a validation is already in flight, it records skipped_busy and schedules the next check.
3. Otherwise it starts one deep validation per registered scenario template, serialized by capacity policy.
4. Each result writes through the same validation/debt path as user-triggered runs.
5. Monitor status reports last run, next run, streak, and current in-flight template.

## Lifecycle Engine Operation Flow

The hard-cutover phases move existing engine behavior into Template Manager without shims. Engine operations resolve repo root, read template/design/resource manifests, perform the requested filesystem work, and return typed reports. The old vrooli CLI command owners are deleted once Template Manager's API/CLI proves parity.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| validation / run | queued, running, succeeded, failed, canceled, timed_out | terminal to running; succeeded without parsed result; failed without diagnostic | Repository transition tests |
| phase-provider / autofix | previewed, applied, no_op, failed | apply without applicable finding; generated provenance for adopted legacy target | Autofix registry tests |
| monitor / deep validate | idle, due, running, skipped_busy, succeeded, failed | concurrent running jobs; next run before current terminal state | Scheduler tests |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain for each flow
- [`DATA.md`](DATA.md) — persisted run, debt, and monitor state
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — test-genie and platform dependencies
