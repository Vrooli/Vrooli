# Workflow Health provider lifecycle fence

This bounded repair implements the existing `common.v1.LifecycleMaintenanceService` contract for Workflow Health's durable validation producer. It persists admission state in the existing validation-run database, closes admission before observing active durable work, and reopens only for the exact operation/revision/token presented by Resume after replacement. The inventory reads the primary ledger and, when configured, its test pool. Both durable start RPCs share the gate; idempotent reattachment and read/wait/abort paths remain available.

A failed Prepare rolls back a fence newly closed by that invocation if authoritative inventory cannot be read. Cleanup uses a fresh five-second-bounded context while retaining an exact operation/revision ownership check. It never rolls back an already-existing fence adopted under explicit emergency override. Emergency adoption remains closed and reports current inventory without claiming drain. A Drain timeout likewise leaves the fence closed.

## Validation

See `validation.txt` for the named cancellation/routed-pool regressions, affected package tests, endpoint generation, and scoped diff check. No scenario lifecycle action or runtime state change was performed.

## Frozen source

`workflow-health-provider-fence.patch` is the reconstructed unified patch against the captured immediate preimages in `preimage/`; the two new lifecycle files are represented as additions. `workflow-health-provider-fence-source-SHA256SUMS.txt` identifies the final nine source files. Existing unrelated workspace changes are excluded.
