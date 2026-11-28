# Architecture — Hello Mobile

## Scenario Purpose

Record this scenario's actual system shape and durable architecture decisions. The [React-Vite architecture guide](../../../../docs/architecture/react-vite-scenario-architecture.md) owns the shared template pattern; this page owns only scenario-specific architecture.

## Scenario Shape

Describe this scenario's actual API, UI, CLI, and contract surfaces. Link each responsibility to its implementation and its [domain owner map](DOMAINS.md).

## System Boundaries

Read the scenario's [service declaration](../../.vrooli/service.json) for its declared runtime shape. Record product-owned code, shared packages, and external dependency boundaries that matter to this scenario.

## Contracts And Data Flow

Describe the scenario's data and request paths. The shared guide documents the React-Vite proto-backed default; link any local REST exceptions or alternate contracts to their implementation.

## Shared Infrastructure

List shared infrastructure this scenario actually uses and the boundary each use crosses. Keep product vocabulary and domain behavior with their owning domain.

## Extension Rules

Describe where future changes to this scenario should land. Keep capability ownership in [`DOMAINS.md`](DOMAINS.md), workflow ownership in [`FLOWS.md`](FLOWS.md), and persistence ownership in [`DATA.md`](DATA.md).

## Architecture Maturity

State maturity only with current local code, tests, or operating evidence. Replace starter claims when this scenario's implementation differs from the template.

## Intentional Deviations

Record durable deviations from the shared guide with their reason and revisit trigger. If none are known, confirm that against this scenario's code before writing that conclusion.

## Documentation Architecture

Keep one local source for each scenario concern: [`DOMAINS.md`](DOMAINS.md), [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md), and [`../internal/TESTING.md`](../internal/TESTING.md). Link details instead of copying their inventories.

## Cross-References

- [Shared React-Vite scenario architecture](../../../../docs/architecture/react-vite-scenario-architecture.md)
- [Declared scenario runtime contract](../../../../docs/architecture/the-declared-scenario.md)
- [Durable decisions](../internal/DECISIONS.md)
