# Clarify: Generate Questions

## Purpose

Generate targeted clarifying questions to reduce ambiguity, uncover hidden requirements, and ensure the backlog item is fully understood before implementation.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure and artifact schemas.

- Existing `clarify/questions.json` if present (preserve existing Q&A)

## Output Requirements

**Primary output**: `clarify/questions.json` (see `swarm-manager-backlog-tools` for full schema)

### Categories

- **users**: Who uses this? What's their workflow? What do they expect?
- **technical**: Architecture, technology choices, implementation details
- **scope**: What's included/excluded? MVP vs full feature?
- **constraints**: Budget, timeline, compatibility requirements
- **integration**: How does this connect with existing systems?

## Success Criteria

- [ ] Questions are specific, not vague
- [ ] Each question has clear business value
- [ ] Critical questions identified and prioritized
- [ ] No more than 10 questions (focus on most important)
- [ ] Existing questions and answers preserved
- [ ] Options provided where applicable

## Instructions

You are generating clarifying questions for a Swarm Manager backlog item. Your goal is to surface the most important unknowns that could affect implementation success.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Question Generation Steps

1. **Read existing context**
   - Review `spec.json` thoroughly
   - Check for existing `clarify/questions.json`
   - Review any user-added files for context

2. **Identify knowledge gaps**
   - What's ambiguous in the description?
   - What assumptions are being made?
   - What could go wrong if we guess wrong?

3. **Categorize by importance**
   - **Critical**: Blocks implementation or could cause major rework
   - **Important**: Affects quality or scope significantly
   - **Nice-to-have**: Would improve outcome but not essential

4. **Prioritize ruthlessly**
   - Aim for 5-7 questions, max 10
   - Better to have 5 excellent questions than 10 mediocre ones
   - Each question should unlock meaningful progress

5. **Craft clear questions**
   - Be specific, not vague
   - One question per topic
   - Provide options where answers are constrained
   - Make it easy to answer

6. **Preserve existing work**
   - If `clarify/questions.json` exists, keep existing questions
   - Only add new questions if needed
   - Never modify existing answers

### Question Templates by Category

**Users:**
- Who is the primary user of this feature?
- What problem does this solve for them?
- What's their current workflow without this?

**Technical:**
- Which existing {{ITEM_KIND}} should this integrate with?
- What's the expected data volume/scale?
- Are there performance requirements?

**Scope:**
- What's the minimum viable version?
- What features can be deferred to v2?
- Are there explicit exclusions?

**Constraints:**
- What's the timeline expectation?
- Are there compatibility requirements?
- Budget/resource constraints?

**Integration:**
- How does this interact with [existing scenario]?
- What APIs need to be exposed/consumed?
- What data needs to be shared?

### Output Format

Write `clarify/questions.json`:

```json
{
  "questions": [
    {
      "id": "q1",
      "question": "[Specific question]?",
      "category": "[category]",
      "importance": "[importance]",
      "options": ["Option A", "Option B", "Other"],
      "answer": ""
    }
  ],
  "generated_at": "[ISO-8601 timestamp]",
  "max_questions": 10
}
```

## Quality Guidelines

**Good questions:**
- "What authentication method should users use: OAuth, API keys, or both?"
- "Should this support real-time updates, or is polling acceptable?"
- "What's the expected number of concurrent users?"

**Poor questions:**
- "What do you want?" (too vague)
- "Have you thought about security?" (not actionable)
- "What about edge cases?" (not specific)

## Anti-Patterns

- **Don't** ask questions already answered in the description
- **Don't** ask hypothetical questions unlikely to affect implementation
- **Don't** include more than 10 questions - quality over quantity
- **Don't** modify existing answers - they're user input
- **Don't** ask compound questions - split into separate questions
