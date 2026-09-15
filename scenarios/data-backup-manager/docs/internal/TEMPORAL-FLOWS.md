# Temporal flows

## Recovery drill

`PreviewDrill` is read-only. It resolves plan membership and finds the latest
successful snapshot. `RunDrill` persists a requested record before scheduling
the existing asynchronous verified-restore operation. The drill then records
the linked restore id, waits for the restore's terminal state, and persists
`verified` only when the restore is verified. Any restore failure, timeout,
evidence-persistence failure, or restart reconciliation persists `failed` with
a next action. Scratch cleanup is completed before restore/verify/audit success
is published; cleanup failure is terminal evidence and remains visible.

The scheduler uses the persisted latest drill per plan × target × destination
to decide whether the interval has elapsed. It does not depend on in-memory
last-fire state, and its deterministic scheduled idempotency key makes a
replayed tick safe.

## Critical topology

Plan creation/update resolves target classification and destination topology
before persistence. Full-primary is unrestricted by critical policy;
critical-primary and critical-secondary require classified targets. Secondary
requires two non-overlapping destination roots. Source/destination overlap is
rejected so a local failure cannot destroy both the source and its recovery
copy.
