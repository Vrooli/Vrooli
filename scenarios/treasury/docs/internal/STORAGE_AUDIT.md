# Treasury Storage Audit

Date: 2026-08-19

## Verdict

Treasury's durable state belongs in one scenario-owned SQLite database at
`${SCENARIO_DATA_DIR}/treasury.db`. The storage declaration in
`.vrooli/service.json` marks that directory as non-regenerable owned data with
a 100 GiB review ceiling. Money Ledger journals, credential-manager secrets,
notification delivery state, and x402 facilitator state are external and are
not claimed by Treasury.

`storage-manager validate scenario treasury` passes. The audit corrected the
book and budget repositories so production modules retain the routed database
handle rather than capturing its primary `*sql.DB` connection. Domain schemas
remain colocated with their owning repositories.

## Integrity and isolation

Financial ownership is enforced at the database boundary. Budgets, mandates,
instruments, authorizations, and approval chains cannot be reassigned across
books. Authorization and approval ownership use domain-owned binding tables so
an existing SQLite file upgrades without an unsafe `ALTER TABLE`; guarded
transactions create the record and binding atomically.

Storage Manager reports the deliberate cross-domain foreign keys and guard
triggers. They are retained because Phase 14 explicitly requires schema-level
book isolation and because removing them would turn a custody invariant into a
best-effort service check. The corresponding raw-SQL rejection tests are tagged
`[REQ:TRS-P1-004]`. No new shared database or private host remediation path is
introduced.

## Retention, backup, and growth

Terminal settlement evidence and idempotency records use the documented
180-day minimum retention window. A database backup must be atomic and include
the idempotency ledger; restoring a partial database could permit duplicate
rail execution. Automated rails remain operationally blocked until the owned
database is registered with Data Backup Manager, as recorded in the runbook.

Measured inbound x402 load (five runs of 32 concurrent payers) completed in
532.8–560.4 ms with no lock failures, so SQLite remains below the declared
migration trigger. The trigger is sustained inbound concurrency that makes the
single-writer lock bind, not speculative scale.

## Evidence

- `storage-manager validate scenario treasury`
- `api/internal/book/sqlite_test.go`
- `api/internal/authorization/sqlite_test.go`
- `api/internal/approval/sqlite_test.go`
- `api/internal/settlement/sqlite_test.go`
- `docs/concepts/DATA.md`
- `docs/operations/RUNBOOK.md`

