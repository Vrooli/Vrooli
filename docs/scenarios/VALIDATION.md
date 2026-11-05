# Scenario Validation

This page describes the current validation expectations for scenarios.

## Primary Principle

Validation should measure desired behavior and deployment readiness signals without depending on stale assumptions about old scenario layouts or old shell-first workflows.

## Canonical Commands

```bash
vrooli scenario test <name>
vrooli scenario validate-env <name>
vrooli scenario requirements validate
vrooli scenario requirements report
vrooli scenario requirements sync
vrooli scenario requirements snapshot
```

Preferred scenario-local flow:

```bash
cd scenarios/<scenario-name>
make test
```

## What Validation Should Cover

Depending on the scenario, validation may include:

- lifecycle correctness
- environment injection correctness
- resource integration
- requirement coverage
- UI/API/CLI behavior
- deployment readiness for the scenario's intended tier
- completeness and maintainability signals

## Requirement-Aware Validation

Scenario requirements are now first-class enough that validation should not be documented as just “run a test script and hope.”

Useful commands:

```bash
vrooli scenario requirements init
vrooli scenario requirements validate
vrooli scenario requirements report
vrooli scenario requirements manual-log
vrooli scenario requirements lint-prd
vrooli scenario requirements phase
vrooli scenario requirements snapshot
```

For the deeper testing stack, use the Test Genie docs via [../TESTING.md](../TESTING.md).

## Health Maturity Reports

Health scenarios that have adopted the shared maturity assessment contract own their local maturity ladder in `scenarios/<provider>/.vrooli/maturity.json`.

When investigating a single validation domain, run the provider CLI without `--json` and treat the human report as the source of truth for current local maturity, next level, blockers, global-impact groups, and recommended skills:

```bash
proto-health validate scenario <name>
measures-health validate scenario <name>
security-health validate scenario <name>
cli-health validate scenario <name>
ui-health validate scenario <name>
```

Use `--json` only for automation such as Test Genie phase ingestion. See [../reference/health-maturity-assessments.md](../reference/health-maturity-assessments.md).

## Guidance

- Prefer validating one named scenario intentionally.
- Avoid claiming that one passing command automatically guarantees universal deployment readiness across every target tier.
- Keep validation expectations aligned with the scenario's actual manifests, lifecycle, and requirement files.
- Keep scenario-system guidance here small; scenario-specific test detail belongs with the scenario.

## Related

- [getting-started.md](getting-started.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [PRODUCTION_BUNDLES.md](PRODUCTION_BUNDLES.md)
- [../TESTING.md](../TESTING.md)
