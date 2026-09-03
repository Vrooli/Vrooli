# Testing — Quality Health

## Strategy

Quality Health testing must prove rule parity before old producers are removed.

## Required Test Layers

| Layer | Purpose |
|---|---|
| Contract unit tests | Applicability, severity, expected evidence, protective comments. |
| Fixture audits | Positive and negative fixtures for every migrated rule ID. |
| Code Facts seam tests | Fake discovered, partial, and unavailable surface inventories. |
| Command executor tests | Success, nonzero exit, missing tool, timeout, bounded output. |
| Autofix tests | Dry-run no mutation, apply explicit mutation, comments inserted/preserved. |
| CLI tests | Human and JSON output, thin API invocation. |
| UI tests | Loading, empty, error, degraded, populated findings, autofix preview. |
| Scenario tests | `vrooli scenario test quality-health`. |

## Rule-Parity Fixture Requirements

Use the user plan store directory `quality-health-phase0/fixtures/` as the starting fixture set.

Every migrated rule ID needs at least one failing fixture and one passing or not-applicable fixture:

- `TS_CONFIG_STRICT`
- `ESLINT_SAFETY_RULES`
- `TS_DANGEROUS_PATTERNS`
- `ESLINT_TYPED_CONFIG`
- `TYPECHECK_PLANNER_COVERAGE`
- `TESTING_CONFIG_LINT_STRICT`
- `GO_MOD_PRESENT_FOR_API_OR_CLI`
- `GO_LINT_CONFIG_PRESENT`
- `GO_LINT_REQUIRED_LINTERS`
- `MAKEFILE_QUALITY_GATES`

## Protective Comment Tests

Add explicit negative tests where config values are strict but required comments are missing. These are not optional docs checks; they are quality-contract checks.

## Validation Commands

```bash
vrooli scenario requirements validate quality-health
vrooli scenario test quality-health
quality-health audit quality-health --json
```

Use Test Genie durable-run reattach commands if a scenario test backgrounds or times out.

## Cross-References

- [Quality Contracts](../reference/quality-contracts.md)
- [Finding Schema](../reference/finding-schema.md)
- [Autofix](../reference/autofix.md)
- [SEAMS](SEAMS.md)
