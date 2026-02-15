# Suggest: Propose Improvements

## Purpose

Propose thoughtful improvements, alternative approaches, and enhancements that could make the backlog item more valuable, feasible, or aligned with Vrooli's vision.

## Input Context

See `swarm-manager-backlog-tools` for folder structure and artifact schemas.

- Existing `suggest/suggestions.json` if present (preserve existing)

## Output Requirements

**Primary output**: `suggest/suggestions.json` (see `swarm-manager-backlog-tools` for full schema)

### Categories

- **architecture**: Technical design improvements, better patterns
- **ux**: User experience enhancements, usability improvements
- **scope**: Adjust scope (add valuable features, remove bloat)
- **risk**: Risk mitigation, safety improvements
- **opportunity**: Monetization, integration, strategic alignment

## Success Criteria

- [ ] Each suggestion adds clear value
- [ ] Details explain the "why" not just the "what"
- [ ] High-impact suggestions prioritized
- [ ] No more than 7 suggestions (focused, not overwhelming)
- [ ] Existing suggestions and decisions preserved
- [ ] Suggestions consider Vrooli ecosystem fit

## Instructions

You are suggesting improvements for a Swarm Manager backlog item. Your goal is to add value through thoughtful suggestions that make the item better, not to overwhelm with options.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Suggestion Generation Steps

1. **Understand current state**
   - Review `spec.json` thoroughly
   - Check `clarify/questions.json` for answered questions
   - Review any research or context files

2. **Consider improvement dimensions**
   - **Architecture**: Is there a better technical approach?
   - **UX**: Could the user experience be improved?
   - **Scope**: Should anything be added or removed?
   - **Risk**: Are there risks that should be mitigated?
   - **Opportunity**: Are there strategic angles being missed?

3. **Evaluate Vrooli ecosystem fit**
   - Does this integrate well with existing scenarios?
   - Could this become a reusable capability?
   - Are there compound intelligence opportunities?

4. **Prioritize by impact**
   - **High**: Would significantly improve outcome
   - **Medium**: Would noticeably improve outcome
   - **Low**: Nice improvement but not essential

5. **Craft clear suggestions**
   - One suggestion per improvement
   - Clear one-line summary
   - Detailed rationale explaining why

6. **Preserve existing work**
   - Keep existing suggestions
   - Respect accepted/rejected status
   - Only add truly new suggestions

### Suggestion Templates by Category

**Architecture:**
- "Consider [pattern] for [reason]"
- "Replace [approach] with [better approach] because [benefit]"
- "Add [component] to enable [capability]"

**UX:**
- "Add [feature] to improve [user workflow]"
- "Simplify [interaction] by [method]"
- "Provide [feedback] when [event] occurs"

**Scope:**
- "Add [feature] to complete the user journey"
- "Remove [feature] to reduce complexity and ship faster"
- "Defer [feature] to v2 to focus on core value"

**Risk:**
- "Add [safeguard] to prevent [problem]"
- "Include [fallback] in case [failure] occurs"
- "Add [validation] to catch [error] early"

**Opportunity:**
- "This could be monetized as [product]"
- "This enables future [capability]"
- "Integrate with [scenario] to [benefit]"

### Output Format

Write `suggest/suggestions.json`:

```json
{
  "suggestions": [
    {
      "id": "s1",
      "suggestion": "[One-line improvement summary]",
      "details": "[2-4 sentences explaining rationale and benefits]",
      "category": "[category]",
      "impact": "[impact]",
      "status": "pending",
      "rejection_reason": ""
    }
  ],
  "generated_at": "[ISO-8601 timestamp]",
  "max_suggestions": 7
}
```

## Quality Guidelines

**Good suggestions:**
- "Add rate limiting to prevent API abuse" (risk mitigation, clear reason)
- "Use Qdrant for semantic search instead of keyword matching" (architecture, specific improvement)
- "Expose this as a standalone API for third-party integrations" (opportunity, strategic value)

**Poor suggestions:**
- "Make it faster" (vague, no actionable detail)
- "Add more features" (not specific)
- "Consider best practices" (not actionable)

## Anti-Patterns

- **Don't** suggest things already in the description
- **Don't** suggest contradictory improvements
- **Don't** include more than 7 suggestions - be selective
- **Don't** modify existing suggestion statuses
- **Don't** suggest for the sake of suggesting - each must add value
