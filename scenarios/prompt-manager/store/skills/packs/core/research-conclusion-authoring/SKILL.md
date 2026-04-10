## Practice focus: Research Conclusion Authoring

Create a durable research conclusion file that captures the research question, methodology, findings, and actionable next steps. This skill standardizes how research results are documented so any future agent can understand what was investigated, what was found, and what to do next.

---

### 1. When to Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| Research workshop rounds are converging on a conclusion | Yes | Produces a reusable research artifact |
| User asked for a "research conclusion" or "research summary" | Yes | Standardized output format |
| Research item is being processed after workshop | Yes | Execution agent needs this to act on findings |
| Implementation planning (non-research) | No | Use `implementation-plan-authoring` instead |

---

### 2. Scope Boundaries

**In scope:**
- Create/update a single `conclusion.md` file in the backlog item folder
- Preserve research question, methodology, findings, and actions
- Define concrete actions for what happens after the research
- Ensure the conclusion stands alone without needing workshop round context

**Out of scope:**
- Implementing the actions described in the conclusion
- Creating implementation plans (that is a different skill)
- Replacing design docs, PRDs, or user-facing product docs

---

### 3. Inputs

Required:
- The research question being investigated
- Findings from workshop rounds or direct investigation

Optional but recommended:
- Workshop round history showing the investigation path
- Evidence references (file paths, code snippets, external sources)

---

### 4. Conclusion Template

Every `conclusion.md` must follow this structure:

```markdown
# Research Conclusion: {title}

## Research Question
What we set out to answer. State the question clearly and precisely.

## Summary
2-3 sentence answer to the research question. This is the TL;DR — a reader should understand the core finding from this section alone.

## Methodology
How the research was conducted: what was examined, what tools/approaches were used, what sources were consulted.

## Findings
Detailed results organized by theme or sub-question.
Can reference supporting files in the backlog item folder.

### Finding 1: {title}
{Description with evidence}

### Finding 2: {title}
{Description with evidence}

## Limitations
What could not be determined, confidence levels for key findings, areas that would benefit from further investigation.

## Actions
Concrete instructions for what happens next. Each action is an explicit imperative.

### Action 1: {action type} — {description}
{Details}

### Action 2: {action type} — {description}
{Details}
```

---

### 5. Action Types

Actions tell the execution agent exactly what to do. Each action MUST specify its type.

#### `Create backlog item`
Create a new backlog item via the swarm-manager CLI. Specify all required fields:
- **kind**: idea, fix, execute, research, or chore
- **title**: Clear, descriptive title
- **description**: What needs to be done and why
- **initiative**: Preserve from the research item if applicable
- **priority**: Suggested priority level
- **effort**: Estimated effort
- **depends_on**: Any dependencies

Example:
```markdown
### Action 1: Create backlog item — Add caching layer to API responses
- **Kind**: execute
- **Title**: Add Redis caching to /api/v1/reports endpoint
- **Description**: Research found that report queries average 2.3s. Adding a Redis cache with 5-minute TTL would reduce p95 latency to under 200ms. See Finding 2 for benchmark data.
- **Initiative**: performance-improvements
- **Priority**: high
- **Effort**: medium
```

#### `Update document`
Make specific changes to an existing file. Specify:
- **File path**: Exact path to the file
- **What to change**: Precise description of the modification

Example:
```markdown
### Action 2: Update document — Add research findings to architecture docs
- **File**: scenarios/my-scenario/docs/ARCHITECTURE.md
- **Change**: Add a "Caching Strategy" section under "Performance" documenting the Redis caching pattern identified in Finding 2.
```

#### `No further action required`
When the research itself is the deliverable and no follow-up work is needed:

```markdown
### No further action required
This research produced a report. The findings above are the deliverable.
```

---

### 6. Quality Gates

Before finalizing a conclusion, verify:

- **Standalone readability**: Could an agent reading ONLY this file understand the research question, what was found, and what to do next? If not, add missing context.
- **Actionable actions**: Could an execution agent read the Actions section and know exactly what to do without ambiguity? Actions must be explicit imperatives, not suggestions or recommendations.
- **Evidence-backed findings**: Are findings supported by specific evidence (file paths, measurements, code references)? Unsupported claims must be marked as assumptions.
- **No duplication**: Findings should summarize and point to workshop rounds for detail, not duplicate everything from rounds.
- **Completeness**: The Limitations section must honestly acknowledge gaps. Research that claims no limitations is suspect.

---

### 7. Guardrails

- Do not write vague actions ("consider improving X", "look into Y") — every action must be a concrete imperative with enough detail to execute.
- Do not hide assumptions; mark unknowns explicitly and note how they could be resolved.
- Do not duplicate entire workshop round contents into findings — summarize and reference.
- Do not omit the Limitations section — every investigation has boundaries.
- Do not mix action types — each action gets its own section with a clear type label.
- Do not include implementation details in findings — findings describe what IS, actions describe what to DO.

---

### 8. Output Expectations

**Must produce:**
- A `conclusion.md` file following the template above
- A conclusion detailed enough that an execution agent can act on the Actions without additional context

**May include:**
- References to specific workshop rounds for deeper detail
- File paths and code references as evidence
- Comparison tables for evaluated options
- Confidence levels on findings

**Must not include:**
- Placeholder-only sections with no actionable content (use `<!-- TBD -->` during workshop, but final conclusions must not have placeholders)
- Implementation code or patches (those belong in execute items)
- Contradictory findings presented without resolution or acknowledgment
