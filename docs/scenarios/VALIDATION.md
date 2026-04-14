# Scenario Validation

This page describes the current validation expectations for scenarios.

## Primary Principle

Validation should measure desired behavior and deployment readiness signals without depending on stale assumptions about old scenario layouts or old shell-first workflows.

## Canonical Commands

```bash
vrooli scenario test <name>
vrooli scenario requirements validate
vrooli scenario requirements report
vrooli scenario requirements sync
```

Preferred scenario-local flow:

```bash
cd scenarios/<scenario-name>
make test
```

## What Validation Should Cover

Depending on the scenario, validation may include:

- lifecycle correctness
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
vrooli scenario requirements snapshot
```

For the deeper testing stack, use the Test Genie docs via [../TESTING.md](../TESTING.md).

## Guidance

- Prefer validating one named scenario intentionally.
- Avoid claiming that one passing command automatically guarantees universal deployment readiness across every target tier.
- Keep validation expectations aligned with the scenario's actual manifests, lifecycle, and requirement files.

## Related

- [getting-started.md](getting-started.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [../TESTING.md](../TESTING.md)
