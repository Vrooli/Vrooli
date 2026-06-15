# Architecture — Quality Health

## Purpose Of This Document

This document defines Quality Health's system shape, ownership boundary, data flow, extension rules, and phase handoff responsibilities.

It does not duplicate the rule catalog in [quality-contracts.md](../reference/quality-contracts.md), the API surface in [api-endpoints.md](../reference/api-endpoints.md), the CLI surface in [cli-commands.md](../reference/cli-commands.md), or test seams in [SEAMS.md](../internal/SEAMS.md).

## Scenario Shape

Quality Health is a meta / interface-enabler scenario with three user-facing surfaces and one contract layer.

```text
Code Facts
  -> Quality Health API
      -> contract registry
      -> audit orchestrator
      -> command runner
      -> autofix planner
      -> normalized findings
  -> Quality Health CLI
  -> Quality Health UI
  -> Test Genie quality phase
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Static-quality engine | Code Facts adapter, contract matching, findings, maturity, command execution, autofix planning | CLI formatting, browser state |
| CLI (`cli/`) | Agent/operator wrapper | Argument parsing, JSON/human rendering, API invocation | Quality policy logic |
| UI (`ui/`) | Operator inspection console | Audit overview, surface breakdown, findings workbench, autofix preview | Separate audit model or business logic |
| Proto/contracts | Wire shape | Request/response messages and generated clients | Hand-written DTO mirrors |

## System Boundaries

Quality Health owns:

- static-quality contract registry,
- lint/type command configuration checks,
- TypeScript/ESLint/Go quality config strictness,
- required protective comments in config files,
- suppression and weakening detection,
- normalized quality findings and maturity summaries,
- safe config autofix preview/apply,
- Test Genie `quality` provider behavior once cut over.

Quality Health does not own:

- unit testing,
- coverage policy,
- generic maintainability/tidiness debt,
- Scenario Auditor standards unrelated to static quality,
- template-specific quality policy.

Tidiness Manager remains the owner for long files, large functions, complexity, duplication, TODO/FIXME/HACK debt, stale/unvisited files, issue queues, and cleanup campaigns.

## Contracts And Data Flow

The API should expose proto/Connect contracts unless a generated template route forces a documented REST exception.

```text
AuditQualityRequest
  target scenario/path/project
  optional surfaces/rule_ids
  include_command_execution
  include_autofix_preview

AuditQualityResponse
  run metadata
  discovered surfaces
  contract evaluations
  findings
  command results
  maturity
  summary/next steps/degraded reason
```

The load-bearing invariant is that Code Facts discovers surfaces before contracts are selected. Filesystem conventions such as `ui/`, `api/`, and `cli/` may appear as evidence from Code Facts or a degraded fallback, but they must not become the core policy model.

## Domain Inventory

The implementation should replace the generated example domain with these domains:

- `surfaces`: Code Facts ingestion and normalized inventory.
- `contracts`: language/framework/surface contract registry.
- `audit`: orchestration, finding generation, maturity summary.
- `commands`: bounded lint/type command resolution and execution results.
- `autofix`: deterministic config fix preview/apply.
- `explain`: finding explanation and remediation detail.

See [DOMAINS.md](DOMAINS.md) for details.

## Shared Infrastructure

Keep shared helpers business-vocabulary-free. API server composition, health handlers, clock seams, test utilities, UI render helpers, and design tokens can remain shared. Contract logic should live in domain packages, not generic buckets.

## Extension Rules

1. Add or update proto messages before adding new API/CLI/UI payloads.
2. Add contract rules in the contract registry, keyed by language/framework/surface/tooling.
3. Add fixture tests for every rule and every protective-comment requirement.
4. Keep CLI commands thin over API calls.
5. Render UI from the API audit model; do not invent frontend-only finding categories.
6. Record any intentional rule weakening in docs and tests before implementation. The default is no weakening.

## Architecture Maturity

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| Foundation docs | Active | PRD, requirements, architecture, seams, reference docs | Product implementation begins in Phase 2. |
| API | Planned | Domain map and endpoint contract documented | Phase 2 must implement the domains. |
| CLI | Planned | Command contract documented | Phase 2 must replace template CLI domains. |
| UI | Planned | UI obligations documented | Phase 3 must replace placeholder UI. |
| Test Genie integration | Planned | Phase and maturity decisions documented | Phase 4 performs hard cutover. |

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-15 | Generated template sample domain removed during foundation setup. | Keeping a sample CRUD domain would confuse Quality Health ownership and Test Genie phase boundaries. | Phase 2 adds real Quality Health domains. |

## Documentation Architecture

| Concern | Canonical Document |
|---|---|
| Product targets | `PRD.md` |
| Domain ownership | `docs/concepts/DOMAINS.md` |
| Seams and fakes | `docs/internal/SEAMS.md` |
| Known gaps | `docs/internal/PROBLEMS.md` |
| Implementation progress | `docs/internal/PROGRESS.md` |
| Quality rule catalog | `docs/reference/quality-contracts.md` |
| Finding output shape | `docs/reference/finding-schema.md` |
| Autofix safety | `docs/reference/autofix.md` |

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [INTEGRATIONS.md](INTEGRATIONS.md)
- [SEAMS.md](../internal/SEAMS.md)
- [TESTING.md](../internal/TESTING.md)
