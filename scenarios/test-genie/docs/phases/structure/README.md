# Structure Phase

**ID**: `structure`
**Timeout**: 60 seconds
**Optional**: No
**Source**: validation-provider (`structure-health`)

The structure phase validates that a scenario has a well-formed skeleton and
correctly wired lifecycle before any tests run. It runs first and fails fast if
basic requirements aren't met.

Test Genie no longer runs a native structure checker. The phase is **delegated
to the [`structure-health`](../../../../structure-health/) scenario** through the
shared `ScenarioValidationService` contract (the same delegation model used by
unit-health, measures-health, scenario-dependency-analyzer, and the other
provider-backed phases). Test Genie calls
`structure-health validate scenario <scenario>`, maps the returned
`MaturityAssessment` findings into the `FINDING_SOURCE_STRUCTURE` channel, and
gates the phase on finding severity.

## What Gets Validated

structure-health reconciles **code-facts ground truth** (the surfaces,
languages, and frameworks actually present, from the `code-facts` scenario)
against the scenario's **declared `service.json` intent**, profile-aware:

- **Skeleton**: `service.json` present/valid, `service.name == directory`,
  required top-level files, a surface directory per declared surface.
- **Lifecycle wiring**: each declared surface has a build → start →
  port + env_var → health-check chain; freshness checks are present per
  buildable surface (a missing `binaries`/`ui-bundle` check causes silent
  rebuilds and is flagged); the UI develop step serves the built production
  bundle rather than a dev server; dependency declarations are well-formed.
- **Profile-keyed conformance packs** (migrated from scenario-auditor's
  `structure`/`config`/`ui` rule packs): for the recognized default
  `react-vite-go` profile these reproduce the previous verdicts; an
  unrecognized profile downgrades them to advisory so non-Go / non-react-vite
  scenarios are not falsely failed.

Many of these findings are **auto-fixable** —
`structure-health fix-config run|apply <scenario>` previews/applies
format-preserving `service.json` and skeleton repairs (dry-run by default).

See the structure-health scenario's own documentation for the full rule
catalog, profile model, and auto-fix behavior.

## Severity & Gating

Findings carry a severity; `ERROR`/`BLOCKER` findings fail the phase. Because
the default profile reproduces the previous Structure + scenario-auditor
verdicts, already-conformant react-vite/Go scenarios see no new failures.

## Resilience

Structure is the first/fast-fail phase, so it now depends on the
`structure-health` provider being reachable. The provider-contract scan tracks
provider conformance, and Test Genie's delegated-phase path degrades gracefully
(treating an unreachable provider as an environmental skip rather than a hard
failure) — the same contract every other delegated phase relies on.

## Related Documentation

- [structure-health scenario](../../../../structure-health/) - the provider that
  backs this phase (rules, profiles, auto-fix, fleet intelligence)
- [CLI Manifest Contract](cli-approaches.md) - manifest-driven scenario CLI
  adapters (the CLI surface itself is validated by the `contracts` phase via
  cli-health)
- [UI Smoke Tests](ui-smoke.md) - BAS-based UI validation (the `smoke` phase)
- [Playbooks Directory Structure](../playbooks/directory-structure.md)

## See Also

- [Phases Overview](../README.md) - All phases
- [Dependencies Phase](../dependencies/README.md) - Next phase
