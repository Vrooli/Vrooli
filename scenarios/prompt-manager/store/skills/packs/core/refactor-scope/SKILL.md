# Refactor Scope

Session constraints for refactoring work. These boundaries ensure changes improve code quality without altering functionality.

## Session Boundaries

### ALLOWED
- Improve code readability and structure
- Reduce complexity and duplication
- Extract reusable functions/components
- Rename for clarity
- Reorganize file/folder structure
- Improve type safety
- Add/improve comments for complex logic
- Performance optimizations that preserve behavior

### NOT ALLOWED
- Adding new features or functionality
- Changing observable behavior (inputs/outputs)
- Modifying public API contracts
- Adding new dependencies without explicit approval
- Removing existing functionality
- Changing business logic
- Adding new UI elements or interactions

## Quality Requirements

1. **Behavior Preservation**: All existing tests must pass without modification (unless fixing test bugs)
2. **Test Coverage**: Maintain or improve existing test coverage
3. **Incremental Changes**: Prefer small, verifiable improvements over large rewrites
4. **Greenfield**: Unless specified otherwise, assume code is greenfield and that changes should be made with no backwards compatability/legacy/etc.

## Verification Checklist

Before completing any refactoring task:
- [ ] All existing tests pass
- [ ] No new features were added
- [ ] Observable behavior is unchanged
- [ ] Code is demonstrably cleaner/simpler
- [ ] No new dependencies introduced
- [ ] No new changes made outside the target scope (changes may exist in other parts of the project due to parallel agents - leave these be)
