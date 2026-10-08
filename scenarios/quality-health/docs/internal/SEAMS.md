# Seams — Quality Health

## Purpose

This file names the seams Phase 2 should implement so Quality Health can be tested with fixtures before Test Genie cutover removes old producers.

## Required Seams

| Seam | Interface Responsibility | Fake/Test Use |
|---|---|---|
| Code Facts client | Fetch scenario surfaces, parse units, languages, frameworks, package managers, roots, and source facts. | Return compliant, non-compliant, unavailable, and partial discovery fixtures. |
| Scenario locator | Resolve scenario slug to repository path and metadata. | Audit temp fixtures without touching real scenarios. |
| Filesystem reader/writer | Read config/source files and apply explicit autofix edits. | Use temp directories for dry-run/apply tests. |
| Command executor | Run lint/type commands with timeout and capture structured output. | Assert timeout, nonzero exit, missing command, and success behavior. |
| Clock / ID generator | Stamp run metadata and stable test IDs. | Deterministic run IDs and timestamps in golden tests. |
| Contract registry | Provide rule definitions and evaluators. | Unit test applicability and severity without command execution. |
| Finding store | Optional v1 lookup for `explain`. | Can be in-memory until run history exists. |

## Fixture Contract

Phase 2 must import or recreate fixtures from the user plan store directory `quality-health-phase0/fixtures/`.

Required fixture families:

- compliant React/Vite TypeScript config with protective comments,
- strict TypeScript config missing protective comments,
- compliant ESLint safety config with header and per-rule comments,
- ESLint safety rules missing required comments,
- dangerous TypeScript source patterns,
- compliant Go module and golangci config,
- missing Go module/config variants,
- strict `.vrooli/testing.json`,
- Makefile quality gate variants.

## Anti-Seams

- Do not parse Code Facts by shelling out to `rg` as the main discovery path.
- Do not make CLI commands call evaluators directly.
- Do not make UI-specific finding models.
- Do not hide command execution behind global process state; pass an executor seam.

## Cross-References

- [Architecture](../concepts/ARCHITECTURE.md)
- [Quality Contracts](../reference/quality-contracts.md)
- [Autofix](../reference/autofix.md)
