# Vrooli Risks

This document tracks the most important project-level risks that affect platform credibility, maintainability, and strategic execution.

## Severity Levels

| Severity | Meaning |
|----------|---------|
| Critical | Could seriously damage trust, safety, or platform viability |
| High | Likely to slow or distort the platform in meaningful ways |
| Medium | Important, but manageable with disciplined follow-through |
| Low | Worth tracking, but not urgent |

## R-001: Documentation Drift

- Severity: high

### Risk

The project evolves faster than its documentation. Old architecture stories, old command flows, and old deployment assumptions linger and mislead contributors.

### Why It Matters

- contributors make decisions from obsolete assumptions
- architecture appears less coherent than it actually is
- support and onboarding costs increase

### Mitigation

- keep a canonical docs layer with explicit entrypoints
- mark compatibility docs and historical docs clearly
- treat plans as plans, not as default operational truth

## R-002: Transitional Architecture Confusion

- Severity: high

### Risk

The codebase still contains mixed historical layers, including shell-era and transitional patterns. If those are documented as current architecture, contributors reinforce debt instead of reducing it.

### Why It Matters

- migration effort slows down
- project-level cross-platform story becomes less credible
- repo-aware changes accumulate local exceptions

### Mitigation

- document the Go-native control plane as canonical
- align repo-aware behavior with the repo contract
- keep legacy paths visible only as reference or migration debt

## R-003: Overclaiming Maturity

- Severity: high

### Risk

Vrooli has strong vision and ambitious scope. If documentation describes roadmap or experimental work as production-ready truth, trust erodes.

### Why It Matters

- users and contributors make bad operational decisions
- deployment expectations drift beyond what the platform currently supports
- the project looks inconsistent rather than ambitious

### Mitigation

- separate current state, active work, and long-range vision
- keep Tier 1 deployment as the primary current recommendation
- use tiered deployment framing consistently

## R-004: Scenario Quality Variance

- Severity: high

### Risk

The scenario ecosystem is one of Vrooli's strengths, but uneven quality across scenarios can weaken confidence in the whole platform.

### Why It Matters

- platform behavior becomes hard to predict
- testing, deployment, and onboarding become inconsistent
- weaker scenarios distort mental models for future scenario authors

### Mitigation

- continue strengthening scenario completeness and validation tooling
- use scenario-local docs and lifecycle conventions consistently
- keep canonical scenario-system docs aligned with current standards

## R-005: Deployment Portability Gaps

- Severity: medium

### Risk

Cross-platform and multi-tier deployment direction is strong, but portability beyond Tier 1 still depends on dependency fitness, packaging work, and deployment intelligence that is not fully realized yet.

### Why It Matters

- teams may commit too early to unsupported targets
- packaging efforts may fragment without a shared model
- scenario authors may bake in assumptions that hurt portability

### Mitigation

- keep the Deployment Hub as the canonical source of deployment maturity
- treat bundle generation and target-specific packaging as tier-aware work
- prefer dependency analysis and explicit fitness scoring over hand-wavy portability claims

## R-006: Governance Drift

- Severity: medium

### Risk

As the ecosystem grows, package governance, repo contract rules, testing expectations, and documentation norms may drift apart.

### Why It Matters

- contributors lose a shared model of what “correct” looks like
- cross-scenario coupling and path assumptions creep back in
- validation becomes less effective over time

### Mitigation

- keep `make validate-repo-contract` and package-governance validation visible
- keep canonical docs small, clear, and current
- update policy docs when operational reality changes

## R-007: Strategic Diffusion

- Severity: medium

### Risk

Vrooli can point in many exciting directions at once: deployment, agent orchestration, testing, business generation, desktop/mobile packaging, domain specialization, and more. Without a coherent near-term focus, effort diffuses.

### Why It Matters

- velocity drops despite high activity
- contributors optimize local areas without reinforcing the platform core
- roadmap language becomes inspirational but operationally weak

### Mitigation

- keep near-term priorities concrete
- favor changes that improve the shared control plane and ecosystem quality
- make roadmap phases legible and current

## R-008: Security And Privacy Credibility Gap

- Severity: critical

### Risk

The project is explicitly positioned around local sovereignty and operator control. If operational or documentation practices overstate security posture, the platform's core promise is undermined.

### Why It Matters

- trust damage is severe when privacy/security claims are inaccurate
- users may apply the platform to sensitive environments with the wrong assumptions

### Mitigation

- avoid blanket compliance or security guarantees in general docs
- document security posture and deployment maturity honestly
- keep sensitive-environment guidance tied to real supported controls

## Review Rule

This file should be updated when a major platform shift changes what the most important project-level risks are. It should not turn into a generic enterprise risk template.
