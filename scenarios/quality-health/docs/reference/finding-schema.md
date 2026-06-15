# Finding Schema

Quality Health findings are stable, agent-readable, and suitable for Test Genie consumption.

## Fields

```text
id
scenario
target_kind
surface_id
surface_kind
language
framework
rule_id
category
severity
file_path
symbol
message
evidence
expected
observed
why_it_matters
remediation
autofix_available
autofix_command
source_command
created_at
```

## Stable ID

Derive IDs from normalized evidence:

```text
sha256(scenario + surface_id + rule_id + file_path + symbol + normalized_expected + normalized_observed)
```

The same violation should produce the same ID across runs unless its normalized evidence changes.

## Severity

- `error`: static-quality contract breach that should gate the Test Genie quality phase.
- `warning`: important drift that may not block in v1.
- `info`: observations and degraded-mode notices.

Current rule-parity mapping:

- current `high` severity becomes `error`,
- current `medium` severity becomes `warning` unless Phase 2 intentionally promotes it.

## Maturity

Quality maturity is deterministic:

- L0: no reliable quality audit,
- L1: surfaces discovered,
- L2: lint/type commands configured and runnable,
- L3: strict contracts enforced with no missing config,
- L4: suppressions/weakening controlled and findings remediated,
- L5: autofix and drift gates integrated into Test Genie comprehensive.

## Test Genie Mapping

Prefer `FINDING_SOURCE_STANDARDS` for v1 and map the Test Genie `quality` phase to the `standards` dimension. Preserve Quality Health finding IDs in evidence when adapting to Test Genie's finding shape.
