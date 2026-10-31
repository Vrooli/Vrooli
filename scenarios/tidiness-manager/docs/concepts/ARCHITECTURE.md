# Architecture

## Purpose Of This Document

This document maps Tidiness Manager's current surfaces and ownership boundaries so future agents can extend it without drifting back into static-quality policy.

## Scenario Shape

Tidiness Manager has three active product surfaces:

- Go API under `api/` for scan orchestration, persistence, campaign state, issue queues, and UI data.
- Go CLI under `cli/` for agent workflows.
- React/Vite UI under `ui/` for human inspection and campaign management.

## System Boundaries

Tidiness Manager owns maintainability findings: file length, technical-debt markers, complexity, duplication, coupling, AI refactor suggestions, issue status, and cleanup campaigns.

Quality Health owns static-quality contracts: TypeScript strictness, ESLint safety rules, Go lint configuration, Makefile quality gates, and `.vrooli/testing.json` lint handler policy.

Test Genie owns orchestration. Its `tidiness` phase delegates to Tidiness Manager, while its `quality` phase delegates to Quality Health.

## Contracts And Data Flow

Scans enter through CLI commands or HTTP endpoints. The API normalizes scenario identity, locates files, computes metrics, stores results, and returns findings or dashboard data.

Campaign flows use persisted campaign state plus optional visited-tracker integration. Smart scans add AI-sourced issue candidates when resources are available.

## Maintainability Source Inventory

Maintainability scans include source in custom components and supporting code,
including Playwright sidecars. Component names are not runtime-data categories.
The shared `api-core/pathfilter` policy excludes dependencies, build output and
runtime data; target exclusions remain explicit. Tidiness Manager collects one
file inventory through its existing scanner and derives language groups from that
same inventory for length, complexity, duplication, coupling and detail views.
It must not maintain a second list of conventional source directories.

CodeFacts owns declared component surfaces and parse-unit evidence. That inventory
confirms a sidecar's product ownership, while maintainability scanning also covers
supporting source outside declared components. Reusing the existing local file
inventory avoids adding a remote dependency to a filesystem metric computation.
Missing or unsupported analyzer output remains distinct from a clean finding set.

## Shared Infrastructure

- PostgreSQL stores issues, metrics, scan history, and campaign state.
- Lifecycle assigns API and UI ports.
- Optional analyzer tools enrich complexity and duplication results.
- Optional resource-claude-code/resource-codes support smart scans.
- Optional visited-tracker improves prioritization and campaign handoff.

## Extension Rules

- Add new maintainability detectors as scan findings, not as lint/type policy.
- Keep path validation centralized through scanner/coordinator seams.
- Prefer CLI and API contracts over ad hoc scripts.
- Update `docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`, and requirements when behavior changes.

## Architecture Maturity

The scenario has a clear post-cutover ownership boundary and active test coverage for API, CLI, UI, quality, and tidiness phases. Remaining hardening work is tracked in `docs/internal/PROBLEMS.md`.

## Intentional Deviations

The legacy light scan still exposes lint/type parser support for historical stored workflows. It is compatibility surface, not ownership of static-quality policy.

## Documentation Architecture

Concept docs explain mental models, reference docs list stable interfaces, operations docs cover runtime work, and internal docs preserve agent memory.

## Cross-References

- `DOMAINS.md`
- `FLOWS.md`
- `DATA.md`
- `INTEGRATIONS.md`
- `../internal/SEAMS.md`
- `../internal/TESTING.md`

Current qualification limit (BAS-RF-064): the JavaScript duplication adapter
still ignores its supplied inventory and malformed output can become an empty
result. TypeScript complexity is explicitly unsupported. Neither absence
establishes clean source; retain analyzer coverage limits beside native totals.
