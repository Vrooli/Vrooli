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

Any new testing idea should be evaluated against these existing capabilities to avoid duplication and maximize composability with the existing ecosystem.

## Target End State
<!-- TBD — What does the system look like when this idea is implemented? -->

## Implementation Strategy
<!-- TBD — Blocked on fundamental scope and approach decisions. 11 decisions across rounds 1-3 remain unresolved. At minimum, round 1 decisions d1 (core idea), d3 (problem), and round 2 decision d1 (scope) must be answered before this section can be populated. -->

## Contract Decisions
<!-- TBD — API/CLI/data model contracts cannot be defined until the idea is scoped. -->

## Testing Plan
<!-- TBD — We need to know what we're building before we can define a test plan. -->

## Rollout / Validation Checklist
<!-- TBD -->

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Idea too vague to act on | High | High | Workshop rounds 1-3 present 11 targeted decisions to force specificity |
| Overlap with existing test tooling | Medium | Medium | Survey existing test infrastructure before building (see Current Technical Context) |
| Decision paralysis from too many open questions | Medium | Medium | Prioritize round 1 decisions first — they gate all downstream planning |

## Non-goals / Prohibited Patterns
<!-- TBD — Will be defined after scope is established. -->

## Definition of Done
<!-- TBD — Cannot define completion criteria without knowing what we're building. -->
