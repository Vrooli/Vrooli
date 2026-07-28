# Architecture

## Purpose Of This Document

This is the authoritative architecture map. Historical design analyses are context, not contracts.

## Scenario Shape

Browser Automation Studio is a local Vrooli scenario with Go API and CLI surfaces, a React UI, and a Node Playwright driver sidecar. It turns visual workflows into typed browser instructions and durable replay evidence.

## System Boundaries

| Surface | Responsibility | Must not own |
| --- | --- | --- |
| `ui/` | Authoring, visualization, operator feedback | Execution semantics or wire types |
| `api/` | Validation, workflow compilation, orchestration, storage | Browser-process control |
| `playwright-driver/` | Session lifecycle and typed browser actions | Workflow persistence/policy |
| `packages/proto/` | Cross-language action and execution contracts | Product business rules |

## Contracts And Data Flow

UI workflow intent is validated and compiled by the API into proto-backed instructions. The API sends those to the driver; the driver returns normalized outcomes and artifacts. The API persists evidence and exposes it to UI, CLI, and replay export consumers.

## Shared Infrastructure

SQLite is routed through the scenario database layer; artifact storage and process lifecycle are scenario-managed. Test Genie owns scenario-suite execution.

## Extension Rules

Add a capability end-to-end: typed proto action, compiler validation, driver handler, UI authoring, then behavior-focused tests. Never add a parallel JSON instruction dialect.

## Architecture Maturity

The V2 typed instruction path is active. Deployment/commercial hardening is tracked by the operational and business documents, not inferred from old plans.

## Intentional Deviations

The established `browser_automation_studio.v1` proto package prefix is retained because changing it would be a cross-language wire-contract migration, not a cosmetic validator cleanup.

## Documentation Architecture

This document and [Domains](DOMAINS.md) are canonical. Operational procedures live in `docs/operations`; stable facts live in `docs/reference`; old plans are historical.

## Cross-References

- [Domains](DOMAINS.md)
- [Flows](FLOWS.md)
- [Data](DATA.md)
- [Integrations](INTEGRATIONS.md)
- [Seams](../SEAMS.md)
- [Testing](../../../../docs/TESTING.md)
