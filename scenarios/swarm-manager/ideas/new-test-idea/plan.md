# Implementation Plan: New Test Idea

## 1. Purpose

Bootstrap a new test idea into a concrete, implementable scenario within the Vrooli ecosystem. The exact nature and scope of this idea need to be defined through workshop refinement.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 3. Problem Statement

A new test idea has been proposed with minimal initial context. The description ("A new test idea") does not yet specify what problem this solves, who it serves, or what value it delivers. This plan serves as a scaffold to be filled in as workshop rounds clarify the vision.

## 4. Scope

### In Scope
<!-- TBD — depends on workshop decisions about what this idea actually is -->

### Out of Scope
<!-- TBD -->

## 5. Current Technical Context

No existing implementation. This is a greenfield idea with no prior code, dependencies, or infrastructure in place.

### Key Files/Components
<!-- TBD — will be populated once the idea's domain and target scenario are defined -->

## 6. Target End State

<!-- TBD — depends on the nature of the idea, its target users, and its integration with Vrooli -->

## 7. Implementation Strategy

### Phase 1: Define and Scope
- Clarify what the idea is, who it serves, and what value it provides
- Determine whether this is a standalone scenario or an enhancement to an existing one
- Identify required Vrooli resources (postgres, redis, ollama, etc.)
- Define acceptance criteria

### Phase 2: Design
<!-- TBD -->

### Phase 3: Implement
<!-- TBD -->

### Phase 4: Test and Validate
<!-- TBD -->

## 8. Contract Decisions

<!-- TBD — API endpoints, data models, CLI commands to be defined after scoping -->

## 9. Testing Plan

<!-- TBD — will be defined once scope and implementation approach are clear -->

## 10. Rollout / Validation Checklist

- [ ] Scenario builds successfully
- [ ] All tests pass
- [ ] service.json is valid
- [ ] No hardcoded secrets
- [ ] Integration with Vrooli resources verified

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Idea is too vague to implement | High | Workshop rounds to refine scope and requirements |
| Overlaps with existing scenarios | Medium | Review existing scenario catalog before proceeding |
| Insufficient resource dependencies defined | Medium | Identify required resources during scoping |

## 12. Non-goals / Prohibited Patterns

- Do not implement before scope is clearly defined
- Do not duplicate functionality already available in existing scenarios
- No hardcoded secrets or credentials

## 13. Definition of Done

- [ ] Idea is clearly defined with specific problem statement and target users
- [ ] Acceptance criteria are documented and testable
- [ ] Implementation is complete per the scoped phases
- [ ] All tests pass
- [ ] Scenario integrates with Vrooli ecosystem
- [ ] Completion summary written
