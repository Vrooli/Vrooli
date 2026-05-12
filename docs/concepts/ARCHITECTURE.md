# Vrooli Architecture

> **Owner:** `director-swarm` (drift detection via `vision-walk-prep` + `vision-update` decision context). **Author:** operator-direct. **Status:** sketch-level today; canonical-technical-reference expansion is tracked as a swarm-manager backlog candidate (flagged at vision walk #4, 2026-04-27). When expanded, this becomes the canonical "how Vrooli actually works" reference for technical readers, agents, and architectural-question answers. Until then, treat this as a pointer / starting-point and supplement with `README.md` and `VISION.md` for the broader picture.

This document describes the current platform architecture at a high level.

## Core Model

Vrooli is built around a compounding loop:

1. agents use resources to solve problems
2. solutions become scenarios, workflows, packages, or patterns
3. those artifacts become reusable capabilities
4. future work starts from a stronger base

The platform is therefore best understood as a **local software foundry** rather than a single application.

## Core Architectural Principles

### Wrap-not-use

**Agents should use scenarios, not external tools directly.** External tools (git, browsers, web APIs, search engines, etc.) are systematically replaced by Vrooli scenarios that wrap them. The long-run direction is to forbid direct external-tool use entirely; agents go through the scenario or fail. (Capability and reliability are not yet sufficient to enforce that hard rule today — the trajectory is set, the enforcement is gradual.)

**The maturation pattern (proven on GCT, BAS, others):**

1. Start as a simple wrapper — minimal logic, cheap to build because scenario templates and generation tooling keep improving.
2. Add custom capabilities incrementally as needs arise: permissions, analytics, identity-aware policies, integration with other scenarios, custom protections.
3. Eventually wrap the underlying tool's CLI itself with a script that warns or blocks direct use.

**Canonical examples:**

- **Git → Git Control Tower (GCT).** Wraps `git`. Already blocks destructive ops by agents. Coming: per-commit run-attribution from agent-manager workspace sandbox, auto-generated commit messages, auto-PR generation, identity-gated permissions, usage analytics.
- **Browser / web → Browser Automation Studio (BAS).** Wraps browser-use. Adds end-to-end UI testing, screenshot + video capture, known-issue handling, integration with scenario UIs.
- **Sandboxing → agent-manager workspace-sandbox.** Per-run file-change attribution feeds GCT. Coding-agent processes themselves run inside the sandbox via the `runner.Launcher` seam in protected mode (default since 2026-04-28); see `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`.

**Why this isn't "extra work" in the long run:**

- Scenario templates + reliable generation make initial wrapping cheap.
- A wrapper starts as a simple passthrough and gains custom capability only as needed — never speculatively.
- Identity comes from agent-manager (agents are spawned through it), so permission/analytics layers fall out naturally.
- Sandboxed agents can perform approved privileged work through controlled scenario surfaces instead of direct host access. For example, `vrooli scenario start|restart|stop` from a protected workspace is proxied through workspace-sandbox to the host lifecycle command, while arbitrary Docker, systemd, or filesystem mutation remains unavailable unless a Vrooli scenario deliberately wraps it.
- Each wrapper becomes a control point for future capability layering. Strategic value compounds.

**Corollary — internal scope discipline.** The wrap-not-use principle also applies to internal domain boundaries. Each Vrooli scenario stays in its own lane: domain-specific CLIs (marketing-publisher commands, swarm-manager backlog commands, etc.) live in their own scenarios, not bolted onto generic platforms like prompt-manager. Generic team / coordination primitives belong in prompt-manager; everything else in its domain scenario.

### Scenarios as substrate

Scenarios are the unit of accumulation, not just the unit of execution. Every solved problem crystallizes into a scenario; every scenario becomes a permanent capability future scenarios can compose. The platform's intelligence is the tech tree of scenarios it has built, plus the agents that build new ones.

This is why scenarios are dual-purpose by design — each is simultaneously a product (revenue-generating), a capability (composable), and a test (validates underlying resources work together). Treating scenarios as ephemeral tasks would lose the compounding.

### Operator steers, agents execute

The morning vision walk is the steering interface; the rest of the system runs on agent loops with structured decision channels. Operator authority is asserted through accepted decisions, not direct execution. Agents respect approval boundaries even when the agent is technically capable of acting unilaterally — the boundary is the contract, not a capability gap.

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
