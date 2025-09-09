# Scaffolding Phase

## Purpose
Scaffolding creates the minimal viable structure that future improvers can build upon. The goal is NOT to implement everything, but to create a solid foundation.

## Scaffolding Allocation: 20% of Total Effort

### Scaffolding Philosophy
Think of scaffolding as **planting a seed**:
- Create the right structure
- Implement core patterns
- Provide clear extension points
- Leave room for growth

**Quality over Completeness** - Better to have 20% perfectly structured than 80% poorly organized.

## Scaffolding Process

### Step 1: Template Selection
Based on research, choose approach:

```bash
# Option A: Copy from template
cp -r scripts/scenarios/templates/[template-type]/* scenarios/[new-name]/
# OR
cp -r scripts/resources/templates/[template-type]/* resources/[new-name]/

# Option B: Copy from similar existing
cp -r scenarios/[similar-scenario]/* scenarios/[new-name]/
# OR  
cp -r resources/[similar-resource]/* resources/[new-name]/

# Option C: Hybrid approach
# Take structure from template, patterns from existing
```

### Step 2: Reference Existing Implementations
Study similar resources and scenarios to understand:
- Directory structure patterns
- Configuration approaches
- Integration patterns
- Documentation style

Use `vrooli resource [name] content` and `vrooli scenario [name] content` to explore existing implementations.

### Step 3: Core Implementation
Implement ONLY the essentials:
1. **Health check endpoint** - Must respond to health checks
2. **Basic lifecycle** - Must start/stop cleanly  
3. **One P0 requirement** - Prove the concept works
4. **Basic CLI command** - Minimum interaction

Security requirements are handled by other prompt sections.

### Step 4: Configuration
Create basic configuration files following the patterns from your template or reference implementations. Check existing similar projects for configuration examples.

### Step 5: Documentation
Create minimal but clear documentation:
- README.md with purpose and basic usage
- PRD.md as the primary deliverable
- Basic API/CLI documentation if applicable

Reference existing documentation patterns from similar resources/scenarios.

## Scaffolding Success Criteria
- Structure matches established patterns
- Health checks work
- One P0 requirement is demonstrably functional
- Clear documentation for improvers
- Follows security and lifecycle patterns

## Common Scaffolding Mistakes

**Over-Engineering**
❌ Bad: Implementing all P0 requirements
✅ Good: One P0 + solid structure

**Template Deviation**
❌ Bad: Creating unique structure
✅ Good: Following established patterns

**No Reference Study**
❌ Bad: Creating from scratch
✅ Good: Learning from existing implementations

## Remember
- Templates are in `scripts/[type]/templates/`
- Study existing implementations with `vrooli [type] [name] content`
- Focus on structure, not completeness
- Leave clear extension points for improvers

## When Scaffolding Is Complete
After completing all scaffolding work (PRD + structure + basic implementation), follow the comprehensive completion protocol:

<!-- task-completion-protocol.md already included in base sections -->

Proper task completion transforms your scaffolding work into permanent ecosystem knowledge and ensures smooth handoff to improvers.