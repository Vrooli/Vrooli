# Security — Quality Health

## Scope

Quality Health reads local scenario source and config files and may execute configured lint/type commands. It should never expose secrets or apply broad source rewrites.

## Data Sensitivity

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| File paths | low to medium | audit | Paths may reveal project structure. |
| Config excerpts | medium | contracts | Redact secrets if configs include tokens. |
| Command output | medium | commands | Bound output length and redact obvious secret patterns. |
| Autofix preview | medium | autofix | Contains proposed config edits. |

## Command Execution Rules

- Use explicit command resolution, not arbitrary user shell strings.
- Enforce timeouts.
- Capture bounded stdout/stderr.
- Return structured failure reasons.
- Do not run commands during config-only audits unless requested.

## Mutation Rules

- Dry-run is default.
- `--apply` is explicit.
- Only deterministic config files are mutable in v1.
- Never auto-edit source suppressions in v1.

## Cross-References

- [Autofix](../reference/autofix.md)
- [Quality Contracts](../reference/quality-contracts.md)
- [SEAMS](SEAMS.md)
