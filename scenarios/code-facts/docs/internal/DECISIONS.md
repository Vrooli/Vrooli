# Decisions

## D-001 — Broker, Do Not Parse

Code Facts will broker `go-code-graph` and `typescript-code-graph` instead of parsing supported source languages. This preserves clear ownership and prevents every validator from reinventing parser logic.

## D-002 — Explicit Evidence Statuses

Proof outputs use `proven`, `missing`, `contradicted`, `unsupported`, and `unknown`. This prevents provider failures or unsupported targets from becoming false passes.

## D-003 — Keep Cache Diagnostics First-Class

Cache metadata is part of the report contract, not an implementation detail, because downstream validators need to understand reuse and staleness.
