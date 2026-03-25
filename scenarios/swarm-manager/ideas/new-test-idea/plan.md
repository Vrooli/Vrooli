# Implementation Plan: New Test Idea

## Purpose
<!-- TBD — Pending clarification of what "new test idea" refers to. Could be a testing framework, a test automation scenario, a QA tool, or something else entirely. -->

## Required Reading
```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## Problem Statement
<!-- TBD — The description "A new test idea" does not yet specify what problem is being solved, who experiences it, or what the desired outcome is. -->

## Scope

### In Scope
<!-- TBD — Depends on workshop decisions about what this idea actually entails. -->

### Out of Scope
<!-- TBD -->

## Current Technical Context

Vrooli's existing test infrastructure includes several relevant components:

| Component | Location | Purpose |
|-----------|----------|---------|
| `vrooli scenario test <name>` | CLI | Run scenario-level test suites |
| test-genie | `scenarios/test-genie/` | AI-assisted test generation scenario |
| Go test suites | `scenarios/*/api/*_test.go` | Per-scenario API unit/integration tests using testcontainers |
| Bats tests | Various `*.bats` files | Shell/CLI integration testing |
| agent-manager sandbox | `scenarios/agent-manager/` | Sandboxed execution environment for agents |
| swarm-manager backlog | `scenarios/swarm-manager/` | Orchestrates backlog items through workshop, research, and execution pipelines |

Any new testing idea should be evaluated against these existing capabilities to avoid duplication and maximize composability with the existing ecosystem.

## Target End State
<!-- TBD — What does the system look like when this idea is implemented? -->

## Implementation Strategy
<!-- TBD — Blocked on foundational decisions. 17 decisions across rounds 1-5 remain unresolved. The critical path requires answering just 3 decisions: (1) what is the core idea, (2) what problem does it solve, (3) which scenario does it target. All other decisions can be inferred or deferred once these three are resolved. -->

## Contract Decisions
<!-- TBD — API/CLI/data model contracts cannot be defined until the idea is scoped. -->

## Testing Plan
<!-- TBD — We need to know what we are building before we can define a test plan. -->

## Rollout / Validation Checklist
<!-- TBD -->

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Idea too vague to act on | High | High | Workshop rounds 1-5 present 17 targeted decisions; round 6 consolidates the 3 critical-path decisions |
| Overlap with existing test tooling | Medium | Medium | Survey existing test infrastructure before building (see Current Technical Context) |
| Decision paralysis from too many open questions | Medium | Medium | Round 6 consolidation: only 3 decisions required to unblock all planning |
| Workshop stalls without user engagement | High | High | Round 6 clearly states minimum-viable input: answer 3 decisions to unblock everything |

## Non-goals / Prohibited Patterns
<!-- TBD — Will be defined after scope is established. -->

## Definition of Done
<!-- TBD — Cannot define completion criteria without knowing what we are building. -->
