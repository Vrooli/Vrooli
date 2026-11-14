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
fix_class
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
- `info`: observations, degraded-mode notices, and coverage gaps.

Current rule-parity mapping:

- current `high` severity becomes `error`,
- current `medium` severity becomes `warning` unless Phase 2 intentionally promotes it.

## Fix Class And Autofix Honesty

Every rule declares one `fix_class`:

- `autofix`: deterministic config repair may exist, but a finding reports `autofix_available = true` only when the registered fixer can preview a safe edit for that exact finding.
- `detection_only`: Quality Health can identify the issue, but the repair needs source-level judgment or manual policy intent.

Missing files, parse errors, and unsupported config shapes must not report an autofix command. The aggregate `counts.autofixable_count` is the number of findings whose individual `autofix_available` flag is true.

## Contract Evaluation Status

Each discovered surface produces one `ContractEvaluation` whose `status` is:

- `passed`: the surface was evaluated and produced no findings,
- `warning`: the surface produced only warning-severity findings,
- `failed`: the surface produced at least one error-severity finding,
- `uncovered`: the surface was discovered but **no contract pack applies** to its language. `contract_id` is empty and a `QUALITY_COVERAGE_GAP` finding is emitted. This is distinct from `passed` — an unevaluated surface must never report a clean pass — and from the run-level `degraded` status (which means Code Facts itself was unavailable).

### `QUALITY_COVERAGE_GAP`

- `rule_id = QUALITY_COVERAGE_GAP`, `severity = info`, `category = coverage`.
- Message names the surface id and detected language, e.g. `surface pysvc (language=python) discovered but no quality contract applies`.
- Info-only: it does not flip run status to `failed` and does not fail the Test Genie quality phase, but it is counted in the summary (`N surface(s) uncovered`) and caps maturity (below).
- Its stable ID derives from the same hash as every other finding (`scenario + surface_id + rule_id + ...`); no schema change.

## Maturity

Quality maturity is deterministic:

- L0: no reliable quality audit,
- L1: surfaces discovered,
- L2: lint/type commands configured and runnable,
- L3: strict contracts enforced with no missing config,
- L4: suppressions/weakening controlled and findings remediated,
- L5: autofix and drift gates integrated into Test Genie comprehensive.

**Coverage cap:** a run with any `uncovered` surface cannot be promoted past **L2**, regardless of how clean the covered surfaces are. Claiming "strict contracts satisfied" (L3+) for a surface that received zero evaluation would be dishonest. The run status itself stays `passed` (the gap is info-only) unless a real error exists elsewhere.

## Test Genie Mapping

Prefer `FINDING_SOURCE_STANDARDS` for v1 and map the Test Genie `quality` phase to the `standards` dimension. Preserve Quality Health finding IDs in evidence when adapting to Test Genie's finding shape.
