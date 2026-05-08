# Architecture

This document is the durable mental model for the scenario. Keep it
focused on product capabilities, surfaces, domain ownership, shared
infrastructure, and intentional deviations from the standard shape.

## Last Updated

[Date]

## Surfaces

| Surface | Role | Owns | Notes |
|---|---|---|---|
| API | [core behavior] | [domain logic, persistence, integrations] | [notes] |
| UI | [presentation] | [feature rendering, browser interactions] | [notes] |
| CLI | [operator/agent wrapper] | [arguments, output formatting] | [notes] |
| Contracts | [wire shape] | [proto/schema/manifest source of truth] | [notes] |

## Domain Map

| Domain | Surface(s) | Primary Archetype | Secondary Traits | Source Paths | Notes |
|---|---|---|---|---|---|
| [domain] | [api/ui/cli] | [CRUD/workflow/integration/etc.] | [traits] | [path:...] | [notes] |

## Shared Infrastructure

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| [path] | [generic mechanics] | [why shared] | [domains/surfaces] |

## Main Flows

| Flow | Entry Point | Domain Owner | Steps | Output |
|---|---|---|---|---|
| [flow] | [path/command/route] | [domain] | [summary] | [result] |

## Architecture Maturity

| Surface | Level | Evidence | Remaining Drift |
|---|---|---|---|
| API | [0-5] | [paths/tests/docs] | [gap or none] |
| UI | [0-5] | [paths/tests/docs] | [gap or none] |
| CLI | [0-5] | [paths/tests/docs] | [gap or none] |
| Docs | [0-5] | [paths/tests/docs] | [gap or none] |

## Intentional Deviations

- [Date]: [Deviation from template or standard pattern, why it exists, when to revisit.]

## Cross-References

- `path:docs/internal/SEAMS.md` — boundary registry and test substitution points.
- `path:docs/internal/PROBLEMS.md` — unresolved architecture drift and deferred refactors.
