# Error Handling — Quality Health

## Purpose

Quality Health errors should help agents distinguish validation failures from infrastructure failures.

## Error Classes

| Class | Meaning | API Status |
|---|---|---|
| contract finding | Target violates a quality contract. | Audit response `failed`; not a transport error. |
| degraded discovery | Code Facts or optional evidence is unavailable. | Audit response `degraded`. |
| command failure | Lint/type command exits nonzero, times out, or is missing. | Structured command result and finding evidence. |
| invalid request | Caller target, rule ID, or surface filter is invalid. | Connect invalid argument. |
| internal error | Unexpected evaluator or filesystem failure. | Audit response `error` or Connect internal error. |

## Principles

- Do not hide degraded discovery behind a passing audit.
- Do not return unbounded command output.
- Prefer structured findings over transport errors when the target scenario is unhealthy.
- Include remediation for user-fixable errors.
- Redact secrets from command output and file evidence.

## Cross-References

- [Finding Schema](../reference/finding-schema.md)
- [Autofix](../reference/autofix.md)
- [SEAMS](SEAMS.md)
