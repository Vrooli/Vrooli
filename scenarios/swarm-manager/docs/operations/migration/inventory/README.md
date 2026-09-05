# Migration inventory (generated)

Deterministic, byte-stable snapshots produced by the **temporary** tool
`scenarios/swarm-manager/tools/statemigrate-inventory/` (deleted in Phase 9).

- `inventory-phase1.json` — machine-readable inventory (baseline).
- `inventory-phase1-summary.md` — human-readable rollup.

Regenerate (read-only; never mutates runtime state):

```sh
cd ../../../../tools/statemigrate-inventory
SCENARIO_ROOT=$(cd ../.. && pwd) go run . --out-dir ../../docs/operations/migration/inventory
```

Two runs over unchanged live state produce byte-identical files. The
`totals.content_hash` is the pre/post-migration reconciliation anchor — see
`../RUNBOOK.md`. During Phase 8, write `pre-migration.json` / `post-migration.json`
here alongside the baseline.
