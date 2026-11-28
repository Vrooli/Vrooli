# Architecture — {{SCENARIO_DISPLAY_NAME}}

## Scenario Purpose

Record this scenario's actual system shape and durable architecture decisions. Describe the shared React-Vite pattern only where it affects this scenario. Read the scenario's [service declaration](../../.vrooli/service.json) before describing runtime behavior.

## Scenario Shape

Describe this scenario's actual API, UI, CLI, and contract surfaces. Link responsibilities to their code and domain owners.

## System Boundaries

Record product-owned code, shared packages, and external dependency boundaries for this scenario.

## Contracts And Data Flow

Describe scenario-specific data and request paths. Link exceptions or alternate contracts to their implementation.

## Shared Infrastructure

List shared infrastructure this scenario uses and its boundary. Keep product behavior with its domain owner.

## Extension Rules

Describe where future changes should land. Keep capability, workflow, and persistence ownership in `DOMAINS.md`, `FLOWS.md`, and `DATA.md`.

## Architecture Maturity

State maturity only with current local code, tests, or operating evidence; replace starter claims as the implementation changes.

## Intentional Deviations

Record durable deviations from the shared guide, with a reason and revisit trigger.

## Documentation Architecture

Keep one local source for each concern and link details rather than copying inventories.

## Cross-References

- [Scenario integrations](INTEGRATIONS.md)
- [Scenario data](DATA.md)
