# PRD Progression for Improvers

## Purpose
PRD progression tracks and advances the implementation of requirements. The PRD is the living contract that guides all work - keeping it accurate is essential.

## Core Philosophy

**The PRD is Sacred**
- It defines the permanent capability
- It guides all future work  
- It must reflect reality
- It tracks our progress

Never fake progress. Never check boxes without verification.

## Checkbox Standards

### What Justifies a Checkmark ✅
A requirement is ONLY complete when:
1. Fully implemented as specified
2. Tests pass consistently  
3. Documentation exists
4. Can be demoed successfully
5. No known issues

### Partial Completion
Use descriptive notes for partial work:
```markdown
- [ ] Search with filters
  - Basic search works
  - Text search implemented  
  - Date filter not implemented
  - Category filter not implemented
```

### False Completion Correction
When finding false completions, uncheck and document:
```markdown
- [ ] Email notifications working
  - Note: Previously checked in error
  - SMTP not configured
  - Templates missing
  - No error handling
```

## Requirement States

- **Planned** [ ] - In PRD but not started
- **In Progress** [~] - Partially implemented, list what's done/remaining
- **Complete** [x] - Fully implemented, tested, documented
- **Blocked** [B] - Cannot proceed, document blocker
- **Deferred** [D] - Decided to postpone, document reason

## Priority Progression

### P0 Requirements (Must Have)
- Complete before moving to P1
- Exception: If blocked by external dependency
- Document any deviations

### P1 Requirements (Should Have)  
- Complete after all P0s
- Prioritize by value/effort ratio
- OK to defer some to next iteration

### P2 Requirements (Nice to Have)
- Only after P0 and critical P1s
- Often become P1s in next version
- Consider user feedback

## PRD Update Process

After each iteration:
1. Review what was implemented
2. Test each implementation
3. Update checkboxes accurately
4. Add implementation notes
5. Update overall progress

## Documentation Requirements

For each completed requirement, add to PRD:
```markdown
- [x] Requirement name
  - Implemented: [Date]
  - Test: `[Test command]`
  - Location: [Where implemented]
```

## Common Mistakes

**Premature Completion**
❌ Bad: Checking off when code is written
✅ Good: Checking off when tested and documented

**Vague Progress**  
❌ Bad: "Search is mostly done"
✅ Good: "Search: Basic + text done, filters remaining"

**Ignoring Blockers**
❌ Bad: Leaving blocked items unchecked
✅ Good: Marking [B] with explanation

## PRD Integrity

Before marking complete:
1. Write test that verifies requirement
2. Document verification command
3. Ensure feature is actually usable
4. Add to health checks if applicable

## Key Principles

**Honesty is essential** - False progress hurts everyone

**Partial is OK** - Better to show real state than fake completion

**Tests prove progress** - No test = not complete

**Documentation seals it** - Undocumented features don't exist

The PRD is the source of truth. Keep it accurate, keep it honest, keep it useful.