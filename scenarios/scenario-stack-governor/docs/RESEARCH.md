# Research Packet

## Uniqueness check
- Repo search: `rg -l "scenario-stack-governor" scenarios/` (new scenario, no prior implementation).

## Related scenarios
- `scenarios/scenario-auditor`: Hub for running/aggregating audits; this scenario is intended to become an external rule pack provider.
- `scenarios/scenario-completeness-scoring`: Measures objective completeness; complementary to “pass/fail governance rules”.
- `scenarios/test-genie`: Test orchestration; useful for executing scenario test suites but not a governance rules source of truth.

## Primary problem this scenario addresses
- Prevent repo-wide breakage due to tech-stack drift (e.g., Go workspace coupling, non-transitive `replace` directives, broken harness wiring), while explaining “why” so fixes are repeatable.
