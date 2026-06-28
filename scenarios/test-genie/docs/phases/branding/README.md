# `branding` phase

The `branding` phase is a **thin delegating phase**: it calls the
`brand-manager` scenario's shared `ScenarioValidationService` and maps
scenario-scoped brand-identity findings into the shared
`FINDING_SOURCE_BRANDING` channel. test-genie does not inspect a scenario's
branding itself; those checks live in `brand-manager` — the single scenario that
both authors and validates branding — alongside its brand-authoring API/CLI/UI.

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Test Genie reads the shared `status`, `assessment.local`, and
`assessment.findings` fields. Each assessment finding is mapped to an
`ArchitectureFinding{Source: FINDING_SOURCE_BRANDING}`, so it carries a
deterministic stable ID, normalized severity, and the per-source effort default.
The phase summary carries brand-manager's `current_level`, `next_level`,
`clean`, and `unknown_count` convergence signals; phase pass/fail still comes
only from `status`.

The phase is **optional**: when `brand-manager` is not running or has no
branding to assess for the target, the phase skips rather than fails.

## Rules

brand-manager owns the rule semantics and severity tier. The branding maturity
ladder evaluates, per scenario:

| Rule | What it checks | Default severity | Auto-fix |
|---|---|---|---|
| `has-display-name` | `service.json` declares a non-placeholder display name | error | no |
| `has-color-system` | a design-token / theme source defines the core color tokens | warning | partial |
| `has-typography` | heading + body font tokens are defined | info | no |
| `has-logo` | a logo asset is present | warning | no (image generation is non-deterministic) |
| `has-favicon` | a favicon is present / referenced | warning | yes (inject default path) |
| `wcag-aa-contrast` | core color pairings meet WCAG 2.1 AA | warning | no |
| `brand-markers-applied` | applied CSS `/* brand-manager:* */` markers + manifest `_brand` keys are present where a brand is assigned | info | yes (inject missing markers) |

## Severity contract

This phase only normalizes the emitted severity string:

| brand-manager severity | normalized | feeds the `branding` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` | ERROR | **yes** |
| `SEVERITY_WARNING` | WARNING | advisory |
| `SEVERITY_INFO` | INFO | advisory |

## Auto-fix

Deterministic rules expose `PreviewFix` (dry-run) and `ApplyFix` (writes) over
the same `ScenarioValidationService` contract. Non-deterministic gaps (e.g. logo
image generation) report a finding with guidance and no fix candidate.
