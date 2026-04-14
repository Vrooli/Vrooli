# Architectural Decisions Record

This document records durable project-level decisions that still shape how Vrooli should be built and documented.

## How To Read This File

Each entry captures:

- context
- decision
- consequences
- current status

This file is for enduring decisions, not short-lived implementation details.

## ADR-001: Vrooli Is A Platform, Not A Single App

- Status: accepted

### Context

Vrooli outgrew the mental model of a single product with a fixed feature surface. The project increasingly revolves around software that creates, governs, and improves other software.

### Decision

Treat Vrooli as a platform for orchestrating resources, scenarios, packages, and operator workflows rather than as a monolithic application.

### Consequences

- project docs must describe the system as a platform
- subsystem boundaries matter more than one global app narrative
- scenario and resource ecosystems become first-class, not peripheral

## ADR-002: Resources And Scenarios Are First-Class Primitives

- Status: accepted

### Context

The project needs a stable way to distinguish raw capability from composed application behavior.

### Decision

Use **resources** as the capability layer and **scenarios** as the composition layer.

### Consequences

- project docs should explain the relationship clearly
- resource-specific implementation detail belongs in resource docs
- scenario-specific design belongs in scenario docs
- project-level docs should avoid duplicating either layer

## ADR-003: The Root Control Plane Is Go-Native

- Status: accepted

### Context

Shell-heavy and transitional control paths made the platform harder to reason about, harder to validate, and less credible as a cross-platform system.

### Decision

Document and evolve the root control plane around the Go-native `vrooli` CLI and its supporting packages under `cmd/`, `internal/`, and governed packages.

### Consequences

- root documentation must treat `vrooli` as the canonical control surface
- shell-era project-level workflows should not be documented as authoritative
- project-level architecture should be described as cross-platform and Go-native

## ADR-004: Cross-Platform Is The Project-Level Contract

- Status: accepted

### Context

The repository contains migration history and mixed implementation maturity, but the project needs a stable future-state contract that repo-aware tooling can rely on.

### Decision

Project-level documentation and repo-aware tooling should align with the future-state cross-platform contract, not normalize legacy layout or shell-era assumptions as permanent architecture.

### Consequences

- canonical path and layout rules must come from the repo contract
- ad hoc repo-root logic should be treated as debt, not precedent
- docs should distinguish migration artifacts from current contract

## ADR-005: Tier 1 Is The Mature Deployment Story Today

- Status: accepted

### Context

The project has multiple deployment aspirations, but not all targets are equally mature.

### Decision

Document Tier 1 local/dev-stack deployment as the primary supported current path, while treating desktop, mobile, SaaS, and appliance-style deployment as tiered work with explicit gaps and roadmaps.

### Consequences

- root and project docs must avoid implying equal maturity across tiers
- deployment docs should stay explicit about current viability versus roadmap
- packaging and portability docs should be framed as tier-aware work

## ADR-006: Documentation Must Separate Current Truth From Vision

- Status: accepted

### Context

Vrooli's ambition is a strength, but docs become misleading when roadmap, experiments, and future-state goals are described as present-day supported behavior.

### Decision

Keep the high-ambition tone, but explicitly separate:

- current truth
- active transformation work
- long-range vision

### Consequences

- canonical docs should prefer precise maturity framing
- plans remain plans unless elevated into canonical docs
- vision docs should inspire without being treated as operational truth

## ADR-007: Scenario-Local Operations Trump Ad Hoc Project Recipes

- Status: accepted

### Context

Scenario lifecycles need consistent, composable operational flows that work with the lifecycle system rather than bypassing it.

### Decision

For individual scenarios, prefer scenario-local Makefile operations such as:

- `make start`
- `make test`
- `make logs`
- `make stop`

Use root CLI flows when operating across scenarios or at the project-control-plane layer.

### Consequences

- docs should point developers to scenario-local lifecycle commands for scenario work
- direct execution shortcuts that bypass lifecycle coordination should not be recommended

## ADR-008: Shared Packages Need Governance

- Status: accepted

### Context

Without explicit governance, shared packages become a source of hidden coupling, drift, and unbounded maintenance cost.

### Decision

Treat shared packages under `packages/` as governed assets with explicit adoption, validation, and refresh rules.

### Consequences

- package usage should follow documented governance
- package docs belong in canonical project reference material
- scenarios should remain intentionally independent where possible

## Notes

- Use [../repo-contract.md](../repo-contract.md) for structural rules.
- Use [../package-governance.md](../package-governance.md) for shared-package policy.
- Use [roadmap.md](roadmap.md) for current strategic direction.
