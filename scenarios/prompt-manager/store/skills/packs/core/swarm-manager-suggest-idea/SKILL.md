# Suggest: Propose Improvements

## Purpose

Propose thoughtful improvements, alternative approaches, and enhancements that could make the backlog item more valuable, feasible, or aligned with Vrooli's vision.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

## Scope

**In scope:**
- Reading all available item context (spec, archive, clarify answers, research, user files)
- Evaluating improvement opportunities across architecture, UX, scope, risk, and strategy
- Generating prioritized, categorized suggestions
- Preserving existing suggestions and user decisions from prior runs

**Out of scope:**
- Generating clarifying questions (see `swarm-manager-clarify-idea`)
- Synthesizing a refined plan (see `swarm-manager-enhance-idea`)
- Implementing any changes directly

## Output Requirements

**Primary output**: `suggest/suggestions.json` (see `swarm-manager-backlog-tools` for full schema)

Write output via CLI:
```bash
swarm-manager backlog file-upload <kind> <name> suggest/suggestions.json <content>
```

### Categories

- **architecture**: Technical design improvements, better patterns
- **ux**: User experience enhancements, usability improvements
- **scope**: Adjust scope (add valuable features, remove bloat)
- **risk**: Risk mitigation, safety improvements
- **opportunity**: Monetization, integration, strategic alignment

## Success Criteria

- [ ] All available context read before generating suggestions (spec, archive, clarify, research)
- [ ] Each suggestion adds clear, specific value
- [ ] Details explain the "why" not just the "what"
- [ ] High-impact suggestions prioritized
- [ ] No more than 7 suggestions (focused, not overwhelming)
- [ ] Existing suggestions and user decisions preserved on re-runs
- [ ] Suggestions consider Vrooli ecosystem fit
- [ ] Output uploaded via CLI and verified

## Instructions

You are suggesting improvements for a Swarm Manager backlog item. Your goal is to add value through thoughtful suggestions that make the item better, not to overwhelm with options.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get {{ITEM_KIND}} {{ITEM_NAME}}
   swarm-manager backlog files {{ITEM_KIND}} {{ITEM_NAME}}
   ```

   Then read each available artifact, starting with the most refined:
   - `enhance/` — if a prior enhance run exists, read `enhance/summary.md` and staging artifacts first. These represent the most refined understanding of the idea. Suggest improvements to gaps or opportunities not already addressed in the enhanced plan.
   - `spec.json` — the item description and metadata (superseded by enhance/ if it exists)
   - `clarify/questions.json` — answered questions narrow the solution space; unanswered questions may reveal areas ripe for suggestions
   - `research/summary.md` — feasibility findings and implementation options
   - `archive/` — user-provided materials (requirements docs, prior scenario artifacts, designs). These reveal what the user values and what's been tried before. Only use for content not already captured in enhance/.
   - `suggest/suggestions.json` — existing suggestions from a prior run (preserve these)
   - Any user-added files in the item root

   > **Why reading order matters:** The backlog folder represents a refinement pipeline (see `swarm-manager-backlog-tools` for the full source authority hierarchy). `enhance/` is the most refined source — if it exists, focus suggestions on improving or extending the enhanced plan rather than duplicating what it already covers. `archive/` is raw source material that reveals what was tried before.

2. **Evaluate improvement dimensions**

   For each category, assess whether there's a meaningful improvement to suggest:

   ```
   Is there a clear improvement opportunity in this category?
     → No  → Skip the category
     → Yes → Is the improvement already covered by the spec or answered questions?
              → Yes → Skip (don't suggest what's already decided)
              → No  → Would implementing this suggestion significantly change the outcome?
                       → Yes → High impact
                       → No  → Would it noticeably improve quality or experience?
                                → Yes → Medium impact
                                → No  → Low impact (only include if under budget)
   ```

3. **Evaluate Vrooli ecosystem fit**

   For each potential suggestion, consider:

   | Question | If Yes |
   |----------|--------|
   | Does this integrate with existing scenarios? | Mention which scenarios and how in the details |
   | Could this become a reusable capability? | Flag as `opportunity` category |
   | Does this leverage shared resources (Ollama, Qdrant, etc.)? | Mention in architecture suggestion |
   | Does this create compound intelligence opportunities? | Flag as high-impact `opportunity` |

4. **Prioritize by budget**

   | Budget | Guideline |
   |--------|-----------|
   | Target | 3–5 suggestions |
   | Maximum | 7 suggestions |
   | Minimum | 1 suggestion (if the idea is already very well-defined) |

   Prefer fewer high-impact suggestions over many low-impact ones.

5. **Handle re-runs**

   ```
   Does suggest/suggestions.json already exist?
     → No  → Generate fresh suggestions
     → Yes → Read existing suggestions
             Preserve all existing suggestions and their statuses unchanged
             (never modify accepted/rejected/pending status or rejection reasons)
             Are there new improvement opportunities not covered by existing suggestions?
               → Yes, and total would be within budget → Add new suggestions
               → Yes, but at budget ceiling → Only add if new suggestion is higher impact
                 than an existing pending one (replace the lower-impact pending suggestion)
               → No → Do not add more suggestions
   ```

6. **Write output**

   Write the suggestions file via CLI (see `swarm-manager-backlog-tools` for the full schema):
   ```bash
   swarm-manager backlog file-upload {{ITEM_KIND}} {{ITEM_NAME}} suggest/suggestions.json '<json content>'
   ```

7. **Verify**

   ```bash
   swarm-manager backlog file-get {{ITEM_KIND}} {{ITEM_NAME}} suggest/suggestions.json
   ```

   Confirm the file was written correctly and the suggestion count is within budget.

### Impact Classification (Convergence Table)

| Signal | Impact | Rationale |
|--------|--------|-----------|
| Changes architecture or eliminates a technical risk | `high` | Structural improvement, hard to retrofit |
| Enables monetization or new capability reuse | `high` | Strategic value multiplier |
| Improves UX for a core workflow | `medium` | Noticeable user benefit |
| Reduces scope to ship faster without losing core value | `medium` | Accelerates delivery |
| Adds a safeguard against a likely failure mode | `medium` | Prevents rework |
| Improves polish, error messages, or edge case handling | `low` | Nice but not essential |
| Adds optional configurability or extensibility | `low` | Future-proofing, not immediate value |

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

- **Don't** suggest things already in the description or decided by answered questions
- **Don't** suggest contradictory improvements
- **Don't** include more than 7 suggestions — be selective
- **Don't** modify existing suggestion statuses or rejection reasons
- **Don't** suggest for the sake of suggesting — each must add value
- **Don't** ignore archive materials — they reveal user intent and past decisions
- **Don't** write JSON output directly to disk — use the backlog CLI

## Troubleshooting & Edge Cases

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 for `suggest/suggestions.json` | Normal on first run — generate fresh suggestions |
| `file-upload` fails | Check that kind and name match an existing backlog item: `swarm-manager backlog get <kind> <name>` |
| Clarify questions are unanswered | Suggestions can still be generated — note that some may become irrelevant once questions are answered |
| Archive contains a detailed PRD | Focus suggestions on gaps or improvements beyond what the PRD covers |
| All existing suggestions are already accepted/rejected | Only add new suggestions if genuinely novel opportunities exist |
| Suggestion conflicts with an answered question | The answered question wins — do not suggest contradicting it |
