# Vrooli Architecture

This document describes the current platform architecture at a high level.

## Core Model

Vrooli is built around a compounding loop:

1. agents use resources to solve problems
2. solutions become scenarios, workflows, packages, or patterns
3. those artifacts become reusable capabilities
4. future work starts from a stronger base

The platform is therefore best understood as a **local software foundry** rather than a single application.

## Primary Layers

### Control Plane

The root control plane is the Go-native `vrooli` CLI and its supporting project internals.

Its responsibilities include:

- setup and development lifecycle
- scenario lifecycle management
- resource lifecycle management
- package governance
- diagnostics and maintenance
- repo-contract validation and path resolution

The most important project entrypoints live under:

- `cmd/`
- `internal/`
- `packages/`

## Resources

Resources provide raw capability.

Typical categories include:

- AI and inference
- relational, cache, vector, and object storage
- browser and workflow automation
- secrets and infrastructure helpers
- supporting execution environments

Resources are not the end product. They are the capability layer that scenarios compose.

## Scenarios

Scenarios are the application and orchestration layer.

A scenario may include:

- UI
- API
- CLI
- tests
- manifests
- deployment metadata
- initialization or runtime assets

Some scenarios are user-facing products. Others are meta-scenarios that improve the platform itself.

## Meta-Systems

Vrooli increasingly relies on scenarios that improve other scenarios, operator workflows, and governance loops.

This includes areas such as:

- testing and requirement validation
- deployment planning
- issue tracking and backlog control
- swarm or team coordination
- browser automation and operator assist surfaces

## Deployment Reality

Project-level docs should distinguish between current and future deployment maturity.

- Tier 1 local/dev stack is the primary mature path today.
- Desktop, mobile, SaaS, and appliance targets are important, but their maturity depends on packaging, dependency fitness, and tier-specific constraints.

Use [../deployment/README.md](../deployment/README.md) as the canonical deployment truth.

## Cross-Platform Direction

The platform should now be documented as a cross-platform Go-native control plane.

That does not mean every scenario or every resource is equally portable yet. It means:

- the project-level contract is cross-platform
- the root CLI is Go-native
- repo-aware behaviors should be described in contract-backed terms
- shell-era assumptions should not be presented as the authoritative model

## Documentation Boundaries

Project-level architecture docs should explain:

- how the platform is organized
- how resources and scenarios relate
- what the root control plane is responsible for
- where to look next for deeper system-specific detail

They should not duplicate:

- scenario-specific design docs
- resource-specific implementation detail
- active implementation plans, unless clearly marked as plans
