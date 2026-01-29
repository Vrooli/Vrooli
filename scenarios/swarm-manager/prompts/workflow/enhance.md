# Enhance: Synthesize Refined Plan

## Purpose

Synthesize clarifications and accepted suggestions into a refined, actionable plan. This is the final preparation step before processing/implementation.

## Input Context

- Item folder at `{{ITEM_FOLDER}}`
- `spec.json` containing item metadata
- `clarify/questions.json` with answers (required)
- `suggest/suggestions.json` with decisions (required)
- Any user-added context files

## Output Requirements

**Primary outputs**:
1. `enhance/summary.md` - Refined plan document
2. `spec.json` - Updated with enhanced description (if significant changes)

The summary must include:
1. Enhanced description incorporating all clarifications
2. Accepted suggestions integrated
3. Implementation notes
4. Scope boundaries
5. Success criteria
6. Ready-for-processing checklist

## Success Criteria

- [ ] All answered clarifying questions incorporated
- [ ] All accepted suggestions integrated
- [ ] Rejected suggestions acknowledged (not included)
- [ ] Scope clearly bounded
- [ ] Implementation-ready plan
- [ ] No ambiguities remaining

## Instructions

You are creating the enhanced specification for a Swarm Manager backlog item. Your goal is to synthesize all gathered information into a clear, implementation-ready plan.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Enhancement Steps

1. **Gather all inputs**
   - Read `clarify/questions.json` - note all answered questions
   - Read `suggest/suggestions.json` - identify accepted vs rejected
   - Review any additional context files

2. **Incorporate clarifications**
   - Transform Q&A into definitive statements
   - "Q: What auth method? A: OAuth" → "Uses OAuth 2.0 for authentication"
   - Address all answered questions

3. **Integrate accepted suggestions**
   - For each accepted suggestion, incorporate into the plan
   - Explain how the suggestion changes the approach
   - Link suggestions to specific plan sections

4. **Acknowledge rejected suggestions**
   - Note what was rejected and why (if reason provided)
   - Ensures transparency about decisions

5. **Define scope boundaries**
   - Explicitly state what's included
   - Explicitly state what's excluded
   - Note any deferred items (v2, future)

6. **Create implementation notes**
   - Technical considerations
   - Integration points
   - Dependencies

7. **Update spec.json if needed**
   - If the enhanced description is significantly different
   - Keep original description, add enhanced version

### Output Format

Write `enhance/summary.md`:

```markdown
# Enhanced Plan: {{ITEM_TITLE}}

## Overview
[2-3 sentence enhanced description that incorporates clarifications]

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| [Question] | [Answer] | [How it affects the plan] |

## Suggestions Integrated

### Accepted
| Suggestion | Integration |
|------------|-------------|
| [Suggestion] | [How it's incorporated] |

### Not Accepted
| Suggestion | Reason |
|------------|--------|
| [Suggestion] | [Why not included] |

## Refined Scope

### Included (Must Have)
- [Feature/capability 1]
- [Feature/capability 2]

### Included (Should Have)
- [Feature/capability 3]

### Excluded (Out of Scope)
- [Not included 1] - [Reason]
- [Not included 2] - [Reason]

### Deferred (Future)
- [Future feature 1] - Target: v2
- [Future feature 2] - Target: when needed

## Implementation Notes

### Technical Approach
[Key technical decisions and patterns]

### Integration Points
- [Scenario/Resource]: [How it integrates]

### Dependencies
- [Dependency]: [Why needed]

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| [Risk] | [How to handle] |

## Success Criteria
- [ ] [Measurable criterion 1]
- [ ] [Measurable criterion 2]
- [ ] [Measurable criterion 3]

## Ready for Processing Checklist
- [ ] All critical questions answered
- [ ] Scope clearly defined
- [ ] Technical approach validated
- [ ] Dependencies available
- [ ] Success criteria measurable

## Next Step
Ready for processing: [Yes/No]
If no: [What's blocking]
```

### Updating spec.json

If the enhanced description differs significantly from the original, update `spec.json`:

```json
{
  "description": "[original description]",
  "enhanced_description": "[new comprehensive description]",
  "enhanced_at": "[ISO-8601 timestamp]"
}
```

## Quality Guidelines

**Good enhancement:**
- Transforms Q&A into definitive statements
- Shows clear decision trail
- Removes ambiguity
- Ready for immediate implementation
- Scope is clear and bounded

**Poor enhancement:**
- Leaves questions unanswered
- Ignores suggestions without explanation
- Vague scope boundaries
- Still has ambiguities
- Not actionable

## Anti-Patterns

- **Don't** ignore unanswered questions - flag them as blockers
- **Don't** silently drop rejected suggestions - acknowledge them
- **Don't** add new scope not from clarifications/suggestions
- **Don't** create implementation details that contradict answers
- **Don't** leave the plan in an ambiguous state

## Template Variables

- `{{ITEM_NAME}}` - Sanitized folder name
- `{{ITEM_TITLE}}` - Human-readable title
- `{{ITEM_DESCRIPTION}}` - Full description
- `{{ITEM_KIND}}` - idea, fix, execute, research
- `{{ITEM_STATUS}}` - Current status
- `{{ITEM_PRIORITY}}` - Priority (0-100)
- `{{ITEM_TAGS}}` - Comma-separated tags
- `{{ITEM_FOLDER}}` - Absolute path to item folder
