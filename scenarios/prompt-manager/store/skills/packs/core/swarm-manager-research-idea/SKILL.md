# Deep Research: Idea

## Purpose

Thoroughly research an idea's feasibility, dependencies, implementation approaches, and integration opportunities within the Vrooli ecosystem before development begins.

## Input Context

See `swarm-manager-backlog-tools` for folder structure and artifact schemas.

## Output Requirements

**Primary output**: `research/summary.md`
**Supporting files**: Add to `research/` as needed (architecture diagrams, dependency maps, API specs)

The summary must include:
1. Executive summary (2-3 sentences)
2. Feasibility assessment (technical, resource, timeline)
3. Dependency analysis (existing scenarios, resources, external services)
4. Implementation approaches (minimum 2 options with trade-offs)
5. Vrooli integration opportunities
6. Risks and mitigations
7. Recommended next steps

## Success Criteria

- [ ] Feasibility clearly stated with supporting evidence
- [ ] All Vrooli scenario dependencies identified
- [ ] At least 2 implementation approaches compared
- [ ] Integration with existing resources (Ollama, PostgreSQL, etc.) evaluated
- [ ] Risks quantified where possible (likelihood, impact)
- [ ] Research is actionable - reader knows what to do next

## Instructions

You are researching a backlog idea for the Swarm Manager. Your goal is to produce research that enables confident decision-making.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Research Steps

1. **Understand the idea thoroughly**
   - Read all files in `{{ITEM_FOLDER}}`
   - Identify explicit and implicit requirements
   - Note any ambiguities that need clarification

2. **Analyze uniqueness**
   - Search existing scenarios: does similar functionality exist?
   - If overlap exists: document what's different and why this idea adds value
   - Consider if this should enhance an existing scenario vs. create new

3. **Map dependencies**
   - Which Vrooli resources would this use? (postgres, redis, ollama, qdrant, etc.)
   - Which existing scenarios could this build on or integrate with?
   - What external APIs or services are required?
   - Create dependency diagram if complex

4. **Evaluate implementation approaches**
   - Option A: The straightforward approach
   - Option B: An alternative with different trade-offs
   - For each: effort estimate (S/M/L), pros, cons, technical risks

5. **Identify Vrooli ecosystem opportunities**
   - How could this become a reusable capability?
   - What future scenarios might benefit from this?
   - Are there monetization angles?

6. **Assess risks**
   - Technical risks (complexity, unknowns)
   - Resource risks (dependencies on external services)
   - Scope risks (feature creep potential)

7. **Synthesize recommendations**
   - Clear go/no-go recommendation with reasoning
   - If go: recommended approach and immediate next steps
   - If no-go: what would change the assessment

### Output Format

Write `research/summary.md` with this structure:

```markdown
# Research Summary: {{ITEM_TITLE}}

## Executive Summary
[2-3 sentences: what this is, is it feasible, recommended action]

## Feasibility Assessment

### Technical Feasibility
[Can we build this? What technical challenges exist?]

### Resource Requirements
[What resources/scenarios does this need? Are they available?]

### Timeline Estimate
[Rough sizing: days/weeks/months with assumptions]

## Dependency Analysis

### Vrooli Resources
- [Resource]: [How it would be used]

### Existing Scenarios
- [Scenario]: [Integration points]

### External Dependencies
- [Service/API]: [Purpose, risk if unavailable]

## Implementation Approaches

### Option A: [Name]
**Approach**: [Description]
**Pros**: [List]
**Cons**: [List]
**Effort**: [S/M/L]

### Option B: [Name]
**Approach**: [Description]
**Pros**: [List]
**Cons**: [List]
**Effort**: [S/M/L]

**Recommended**: [Option] because [reasoning]

## Vrooli Integration Opportunities
[How this fits the compound intelligence vision]
[Reusability potential]
[Monetization angles if applicable]

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| [Risk] | High/Med/Low | High/Med/Low | [Mitigation] |

## Next Steps
1. [Immediate action]
2. [Follow-up action]
3. [Future consideration]
```

## Quality Guidelines

**Good research:**
- Specific and actionable, not vague
- Considers the Vrooli ecosystem holistically
- Acknowledges unknowns explicitly
- Provides clear recommendation with reasoning
- Balances depth with conciseness

**Poor research:**
- Generic analysis that could apply to any project
- Missing dependency analysis
- No consideration of existing scenarios
- Risks listed without mitigations
- No clear next steps

## Anti-Patterns

- **Don't** ignore existing scenarios - always check for overlap
- **Don't** recommend without supporting evidence
- **Don't** skip the Vrooli integration analysis - it's core to our vision
- **Don't** produce research longer than necessary - value density matters
- **Don't** leave ambiguities unaddressed - flag them explicitly
