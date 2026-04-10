# Architecture Scope

Session constraints for architectural improvements. These boundaries ensure structural changes enhance the system without destabilizing functionality.

## Session Boundaries

### ALLOWED
- Reorganizing module/package structure
- Introducing new architectural patterns
- Modifying dependency relationships
- Creating new abstraction layers
- Consolidating duplicate structures
- Improving separation of concerns
- Updating configuration and wiring
- Modifying type hierarchies
- If greenfield (assume yes unless stated otherwise):
  - Changing APIs, interfaces, and proto types

### NOT ALLOWED
- Adding new user-facing features
- Changing business logic behavior
- Removing existing functionality
- Reducing test coverage
- Introducing known technical debt
- If brownfield (assume no unless stated otherwise):
  - Changing APIs, interfaces, and proto types

## Quality Requirements

1. **Behavior Preservation**: External behavior must remain unchanged
2. **Test Migration**: Tests may need updates to reflect new structure, but coverage must be maintained
3. **Incremental Changes**: Large architectural changes should be broken into reviewable steps
4. **Documentation**: Significant structural changes should be documented
5. **Greenfield**: Unless specified otherwise, assume code is greenfield and that changes should be made with no backwards compatability/legacy/etc.

## Verification Checklist

Before completing any architecture task:
- [ ] External behavior is unchanged
- [ ] All tests pass (updated as needed for new structure)
- [ ] Test coverage is maintained or improved
- [ ] New structure is cleaner/more maintainable
- [ ] No new features were added
- [ ] No new changes made outside the target scope (changes may exist in other parts of the project due to parallel agents - leave these be)
