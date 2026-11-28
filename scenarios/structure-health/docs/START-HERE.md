# Start Here — Structure Health

Structure Health is Vrooli's structural authority. It validates the repository
contract across scenarios, resources, tools, safeguards, teams, packages,
control-plane roots, documentation, and the project itself.

The executable rule catalog and generated coverage matrix are the source for
what is checked:

- [`reference/structure-rules.md`](reference/structure-rules.md) — generated
  rule catalog and claim identifiers.
- [`reference/structure-rule-coverage.md`](reference/structure-rule-coverage.md)
  — generated target-kind reachability matrix.
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) — this scenario's
  implementation shape and API/CLI boundaries.

Useful checks:

```bash
structure-health rules list --json
structure-health rules coverage
structure-health rules docs
structure-health validate scenario <name>
structure-health validate package --path packages/<name>
vrooli hygiene --json
```

The CLI and hygiene provider both call the same Structure Health validation
engine. A structural finding always carries a rule code, target, location, and
remediation. If Structure Health is unavailable, callers surface that
unavailability instead of manufacturing a local verdict.

Generated documentation is intentionally not edited by hand. Change the
catalog beside the rule engine, regenerate the pages, and run the Structure
Health API and scenario suites through Test Genie.
