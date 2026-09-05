# Vrooli Roadmap

This roadmap describes the platform's direction from the current state outward. It is intentionally organized by maturity, not by speculative dates.

## Current State

Vrooli should currently be understood as:

- a Go-native project control plane built around `vrooli`
- a platform organized around resources and scenarios
- a local-sovereign development and orchestration environment
- a system increasingly focused on governance, testing, deployment intelligence, and compounding capability

The most mature deployment path today is the Tier 1 local/dev stack described in the Deployment Hub.

## Near-Term Priorities

### 1. Strengthen The Canonical Control Plane

Priorities:

- continue reducing project-level shell-era assumptions
- keep the root CLI and repo-aware tooling coherent
- improve diagnostics, lifecycle clarity, and operator trust

### 2. Raise Scenario Ecosystem Quality

Priorities:

- strengthen completeness, validation, and requirement-driven testing
- improve scenario consistency across lifecycle, docs, and deployment readiness
- make high-quality scenarios better templates for future work

### 3. Improve Deployment Intelligence

Priorities:

- keep Tier 1 workflows strong
- improve dependency analysis, fitness evaluation, and portability reasoning
- support bundle- and tier-aware packaging paths without pretending all targets are equally mature

### 4. Improve Operator Surfaces

Priorities:

- keep the Web Console, voice/mobile interaction, and orchestration surfaces aligned with the real platform
- make it easier to steer the system at the initiative, backlog, and workflow level

## Mid-Term Direction

### 1. Portable Scenario Delivery

The goal is not simply “ship everything everywhere.” The goal is to build a credible portability layer that understands:

- dependency graphs
- fitness for each target tier
- required swaps and packaging constraints
- secrets and runtime differences by deployment target

### 2. Stronger Recursive Governance

Vrooli becomes more valuable as more of its own improvement loops become explicit and reliable.

This includes:

- issue tracking
- review and validation
- test generation and requirement coverage
- deployment planning
- multi-agent coordination and initiative execution

### 3. Better Composition Across Scenarios

The platform should get better at treating scenarios as reusable, intelligible building blocks rather than as isolated artifacts.

## Long-Term Vision

The long-range direction remains ambitious:

- domain-specialized Vrooli stacks
- increasingly autonomous software creation and governance
- broader deployment tiers including desktop, mobile, SaaS, and appliance-style delivery
- a platform where each completed capability makes future capabilities easier to build

That vision remains important, but should not replace current-state precision.

## What The Roadmap Is Not

This roadmap is not:

- a promise that every tier is currently production-ready
- a generic feature wishlist
- a substitute for active plans in `path:docs/plans/`

Use it to understand direction. Use canonical docs and plan documents to understand current truth and active implementation work.
