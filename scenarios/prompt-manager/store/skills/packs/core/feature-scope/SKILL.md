---
name: "feature-scope"
description: "Add functionality while maintaining stability"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["scope"]
  tags: ["scope","feature","constraints"]
  icon: "plus-circle"
  status: "active"
  revision: 30
  createdAt: "2026-02-03T00:00:00Z"
  updatedAt: "2026-02-04T13:13:54Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Feature Scope

Session constraints for feature development. These boundaries ensure new functionality is added responsibly while maintaining system stability.

## Session Boundaries

### ALLOWED
- Adding new features and functionality
- Creating new components, functions, or modules
- Extending existing APIs with new capabilities
- Adding new UI elements and interactions
- Introducing new dependencies (with justification)
- Modifying existing code to support new features
- Adding new tests for new functionality

### NOT ALLOWED
- Removing existing functionality (without explicit approval)
- Breaking changes to public APIs
- Reducing test coverage
- Introducing known security vulnerabilities
- Adding features outside the specified scope
- Major architectural changes (propose separately)

## Quality Requirements

1. **Stability**: All existing tests must continue to pass
2. **Test Coverage**: New code must have adequate test coverage
3. **Documentation**: New features should be self-documenting or have comments
4. **Incremental Delivery**: Large features should be broken into reviewable chunks
5. **Error Handling**: New code must handle errors gracefully

## Verification Checklist

Before completing any feature task:
- [ ] Feature meets the specified requirements
- [ ] All existing tests pass
- [ ] New functionality has test coverage
- [ ] Error cases are handled
- [ ] No new changes made outside the target scope (changes may exist in other parts of the project due to parallel agents - leave these be)