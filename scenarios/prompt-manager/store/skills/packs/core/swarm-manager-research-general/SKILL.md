# Deep Research: General

## Purpose

Conduct thorough research for execute tasks, research-kind items, or other general investigations. Gather information, analyze options, and provide actionable findings.

## Input Context

- Item folder at `{{ITEM_FOLDER}}`
- `spec.json` containing item metadata
- Research target: {{RESEARCH_TARGET}} (if specified)
- Any user-added context files in the folder

## Output Requirements

**Primary output**: `research/summary.md`
**Supporting files**: Add to `research/` as needed (data, analysis, references)

The summary must include:
1. Executive summary
2. Research scope and methodology
3. Key findings
4. Analysis and implications
5. Recommendations
6. References and sources

## Success Criteria

- [ ] Research question clearly understood and addressed
- [ ] Methodology appropriate for the question
- [ ] Findings supported by evidence
- [ ] Analysis goes beyond surface-level
- [ ] Recommendations are actionable
- [ ] Sources documented for verification

## Instructions

You are conducting research for a Swarm Manager backlog item. Your goal is to provide thorough, accurate findings that enable informed decisions.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}
- Research Target: {{RESEARCH_TARGET}}

### Research Steps

1. **Define the research question**
   - What specifically needs to be answered?
   - What would a successful answer look like?
   - What constraints or scope limits apply?

2. **Plan the methodology**
   - What sources will you consult?
   - What analysis methods are appropriate?
   - How will you validate findings?

3. **Gather information**
   - Search relevant codebase areas
   - Review documentation
   - Consult external sources if needed
   - Document all sources

4. **Analyze findings**
   - Synthesize information from multiple sources
   - Identify patterns and insights
   - Note contradictions or gaps
   - Draw evidence-based conclusions

5. **Develop recommendations**
   - What actions do findings suggest?
   - What are the trade-offs?
   - What's the confidence level?

6. **Document thoroughly**
   - Structure findings clearly
   - Include supporting evidence
   - Make it useful for future reference

### Output Format

Write `research/summary.md` with this structure:

```markdown
# Research Summary: {{ITEM_TITLE}}

## Executive Summary
[2-3 sentences: research question, key finding, recommended action]

## Research Scope

### Question
[The specific question(s) being researched]

### Scope
- In scope: [What's covered]
- Out of scope: [What's not covered]

### Methodology
[How the research was conducted]

## Key Findings

### Finding 1: [Title]
**Summary**: [One sentence]
**Evidence**: [Supporting data/references]
**Implications**: [What this means]

### Finding 2: [Title]
**Summary**: [One sentence]
**Evidence**: [Supporting data/references]
**Implications**: [What this means]

### Finding 3: [Title]
**Summary**: [One sentence]
**Evidence**: [Supporting data/references]
**Implications**: [What this means]

## Analysis

### Synthesis
[How findings connect, patterns observed]

### Gaps and Uncertainties
[What couldn't be determined, confidence levels]

### Comparison (if applicable)
| Option | Pros | Cons | Fit |
|--------|------|------|-----|
| [A] | [List] | [List] | [Rating] |
| [B] | [List] | [List] | [Rating] |

## Recommendations

### Primary Recommendation
[What to do, with reasoning]

### Alternative Approaches
[Other valid options if primary doesn't fit]

### Next Steps
1. [Action 1]
2. [Action 2]
3. [Action 3]

## References

### Internal Sources
- `path/to/file.go` - [What it provided]
- [Scenario name] - [Relevance]

### External Sources
- [URL/Document] - [What it provided]
```

## Quality Guidelines

**Good research:**
- Question clearly framed
- Methodology transparent
- Findings evidence-based
- Analysis insightful
- Recommendations actionable

**Poor research:**
- Vague scope
- Sources not documented
- Opinions presented as facts
- Surface-level analysis
- Generic recommendations

## Anti-Patterns

- **Don't** start researching without clarifying the question
- **Don't** present opinions without evidence
- **Don't** ignore contradictory information
- **Don't** overcomplicate the output - match depth to need
- **Don't** forget to document sources - reproducibility matters

## Research Type Variations

### For Execute Tasks
Focus on:
- Prerequisites and dependencies
- Step-by-step execution plan
- Verification criteria
- Rollback procedures if applicable

### For Technology Research
Focus on:
- Comparison criteria
- Real-world usage examples
- Integration with Vrooli stack
- Community and support health

### For Process Research
Focus on:
- Current state analysis
- Pain points and inefficiencies
- Best practices from similar systems
- Implementation roadmap
