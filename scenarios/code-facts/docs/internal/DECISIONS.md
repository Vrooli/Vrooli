# Decisions

## D-001 — Reuse Parser Owners, Do Not Duplicate Them

Code Facts does not implement language parsers. Its Go provider calls the shared API Core `gograph` owner in-process; its TypeScript and JavaScript providers call `typescript-code-graph`. This keeps each parser implementation singular while avoiding an unnecessary Go service hop.

## D-002 — Explicit Evidence Statuses

Proof outputs use `proven`, `missing`, `contradicted`, `unsupported`, and `unknown`. This prevents provider failures or unsupported targets from becoming false passes.

## D-003 — Keep Cache Diagnostics First-Class

Cache metadata is part of the report contract, not an implementation detail, because downstream validators need to understand reuse and staleness.
