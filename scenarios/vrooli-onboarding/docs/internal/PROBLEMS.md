# Known Issues & Follow-Up Tasks

> This file records what is not yet proven. It is intentionally narrower than
> the target contract: a passing contract check does not make an untested
> deployment claim true.

## Current evidence

- Final authoritative run `20260811-112723-3656bc67`: **21/21 phases passed**.
- All six declared journeys have connected selector-resolved adhoc executions completed on 2026-08-11. The source cases remain registry-based `@selector/...` workflows; the adhoc runner currently fails to resolve scenario-owned selector tokens and its documented `--wait`/`--requires-video` flags are broken, so execution evidence is screenshot-backed but has no requested video artifact.
- `experience-manager spec validate vrooli-onboarding`: **L3, zero findings** after the picker binding and journey updates.
- `vrooli scenario requirements validate vrooli-onboarding`: **zero findings**; 65/70 requirements are complete, with profile intake and other P2 work intentionally planned.

- `vrooli scenario test vrooli-onboarding` final run `20260811-105557-49d03aed`:
  **21/21 phases passed**.
- `test-genie runs findings 20260811-105557-49d03aed`: **PASS, 21 phases,
  zero failed phases**.
- `experience-manager spec validate vrooli-onboarding`: **L3, zero findings**;
  all 11 pages and all 6 journeys are active.
- `vrooli scenario requirements validate vrooli-onboarding`: **zero findings**;
  65/70 requirements are complete, with profile intake and other P2 work
  intentionally remaining planned.
- API coverage is 75.1%, CLI coverage is 76.5%, and UI coverage is 96.32%
  statements / 85.21% branches (217 tests).
- Six BAS journey cases plus seven generated experience observers are registered
  and executed by the green workflow phase. The latest run is the durable
  evidence handle above.

## Remaining blockers and limitations

### Remote bridge acceptance is green; baseline comparison remains unavailable

The onboarding boundary and bridge unit tests pass, including the transport-
neutral remote selection/apply/readiness client. The final full bridge run
`20260811-104540-c8815d7f` passed **20/20 phases**, including unit, security,
proto, and UI-health. `proto-health validate scenario vrooli-bridge` also
passes; the remaining findings are warnings for pre-existing domain/template
layout and unreachable legacy messages.

### Plan baseline comparison is unavailable

The captured Git Control Tower baseline exists and is synchronized, but the
fresh Plan Manager validation `b2abdc86-1500-419e-adc0-8d2c54bb2dcb` is
**UNKNOWN / not-comparable**: scenario-to-desktop is clean, bridge is
preexisting, and onboarding cannot compare because the pre-edit baseline
provider was unavailable during the original capture (a stale
`scenario-dependency-analyzer` lifecycle lock, pid 409152). The provider is
healthy now and all current suites pass, but the original baseline evidence
cannot be repaired in place. Recapturing after implementation changes would
invalidate the pre-change regression baseline.

### Alternate-state experience evidence remains fixture-governed

The experience contract reconciles cleanly at L3, all pages/journeys are active,
and the default-route journeys execute. Alternate/error-state assertions that
need deterministic backend fixtures or computed-style evidence remain
aspirational; they are not represented as machine-proven claims.

### Advisory UI debt remains

- The scenario has local Button/SearchInput/StatusBadge components but has not
  adopted the external `react-component-library` catalog.
- UI health still reports advisory focus-zoom and screen-reader clipping
  heuristics, plus a raw empty-state primitive in the glossary.
- Template provenance is not declared for this pre-template scenario.

### Explicitly deferred scope

Profile preselection, integrations, mobile tiers, and credential lifecycle
repair/recovery remain outside this plan's shipped scope. The profile
requirement is deliberately planned rather than presented as complete.

## Resolved in this implementation

- Operator-state writes now use one schema-validated, locked merge-patch
  authority that preserves unknown fields and records apply completion.
- Resource enablement, host requirements, closure/union projection, readiness,
  credentials, surface metadata, session state, and apply reporting use the
  new authority and typed degraded responses.
- Desktop catalog requirements and missing-catalog packaging/API tests cover
  bundle mode; the runtime package tests pass.
- The UI, interactive CLI, declarative wizard, and bridge selection boundary
  share the same selection-to-patch translation; endpoint and no-dead-command
  contract tests pass.
- BAS selector manifests are generated as part of UI build/test, all six
  onboarding journeys are executable, and the comprehensive onboarding suite
  is green.
