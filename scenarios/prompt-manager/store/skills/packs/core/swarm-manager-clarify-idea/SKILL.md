# Clarify: Generate Questions

## Purpose

Generate targeted clarifying questions to reduce ambiguity, uncover hidden requirements, and ensure the backlog item is fully understood before implementation.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

## Scope

**In scope:**
- Reading existing item context (spec, archive, prior questions, user files)
- Identifying knowledge gaps that could affect implementation
- Generating prioritized, categorized questions
- Preserving existing Q&A from prior runs

**Out of scope:**
- Answering questions (that's the user's role)
- Making suggestions or proposing improvements (see `swarm-manager-suggest-idea`)
- Synthesizing a plan (see `swarm-manager-enhance-idea`)

## Output Requirements

**Primary output**: `clarify/questions.json` (see `swarm-manager-backlog-tools` for full schema)

Write output via CLI:
```bash
swarm-manager backlog file-upload --kind <kind> --name <name> --path clarify/questions.json --content '<content>'
```

### Categories

- **users**: Who uses this? What's their workflow? What do they expect?
- **technical**: Architecture, technology choices, implementation details
- **scope**: What's included/excluded? MVP vs full feature?
- **constraints**: Budget, timeline, compatibility requirements
- **integration**: How does this connect with existing systems?

## Success Criteria

- [ ] All available context read before generating questions (spec, archive, user files)
- [ ] Questions are specific, not vague
- [ ] Each question has clear business value
- [ ] Critical questions identified and prioritized
- [ ] No more than 10 questions (focus on most important)
- [ ] Existing questions and answers preserved on re-runs
- [ ] Options provided where applicable
- [ ] Output uploaded via CLI and verified

## Instructions

You are generating clarifying questions for a Swarm Manager backlog item. Your goal is to surface the most important unknowns that could affect implementation success.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Then read each available artifact, starting with the most refined:
   - `enhance/` — if a prior enhance run exists, read `enhance/summary.md` and staging artifacts first. These represent the most refined understanding of the idea and may preemptively answer many questions. Generate questions only for gaps that remain after reading enhance/.
   - `spec.json` — the item description and metadata (superseded by enhance/ if it exists)
   - `archive/` — user-provided materials (requirements docs, prior scenario artifacts, designs). These often contain answers to questions you might otherwise ask. Only use for content not already captured in enhance/.
   - `clarify/questions.json` — existing questions from a prior run (preserve these)
   - Any user-added files in the item root

   > **Why reading order matters:** The backlog folder represents a refinement pipeline (see `swarm-manager-backlog-tools` for the full source authority hierarchy). `enhance/` is the most refined source — if it exists, most questions about scope, architecture, and integration may already be answered there. `archive/` is raw source material — useful when enhance/ doesn't exist or doesn't cover a topic.

2. **Identify knowledge gaps**

   For each category, assess what's unknown:

   ```
   Is the gap answered by archive materials or spec.json?
     → Yes → Skip (don't ask what's already known)
     → No  → Could guessing wrong cause rework or failure?
              → Yes → Critical question
              → No  → Could it significantly affect quality or scope?
                       → Yes → Important question
                       → No  → Nice-to-have (only include if under budget)
   ```

3. **Prioritize ruthlessly**

   | Budget | Guideline |
   |--------|-----------|
   | Target | 5–7 questions |
   | Maximum | 10 questions |
   | Minimum | 3 questions (if the idea is already well-defined) |

   Better to have 5 excellent questions than 10 mediocre ones. Each question should unlock meaningful progress when answered.

4. **Handle re-runs**

   ```
   Does clarify/questions.json already exist?
     → No  → Generate fresh questions
     → Yes → Read existing questions
             Are all questions still relevant given current context?
               → Keep relevant questions unchanged (never modify existing answers)
               → Remove questions that archive/context now answers (move to archive)
             Is the total under budget after removals?
               → Yes → Add new questions only if genuine gaps remain
               → No  → Do not add more questions
   ```

5. **Write output**

   Write the questions file via CLI (see `swarm-manager-backlog-tools` for the full schema):
   ```bash
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path clarify/questions.json --content '<json content>'
   ```

6. **Verify**

   ```bash
   swarm-manager backlog file-get --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path clarify/questions.json
   ```

   Confirm the file was written correctly and the question count is within budget.

### Importance Classification (Convergence Table)

| Signal | Importance | Rationale |
|--------|-----------|-----------|
| Answer determines architecture or tech stack | `critical` | Wrong guess = major rework |
| Answer affects data model or API contract | `critical` | Downstream dependents break |
| Answer determines user-facing behavior | `important` | Affects UX but not architecture |
| Answer affects scope boundaries | `important` | Determines what's built vs deferred |
| Answer affects non-functional requirements (perf, scale) | `important` | Affects design but recoverable |
| Answer improves polish or edge case handling | `nice-to-have` | Low rework cost if guessed wrong |
| Answer is about timeline or process preferences | `nice-to-have` | Informational, not structural |

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

- **Don't** ask questions already answered in the description or archive materials
- **Don't** ask hypothetical questions unlikely to affect implementation
- **Don't** include more than 10 questions — quality over quantity
- **Don't** modify existing answers — they're user input
- **Don't** ask compound questions — split into separate questions
- **Don't** write JSON output directly to disk — use the backlog CLI

## Troubleshooting & Edge Cases

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 for `clarify/questions.json` | Normal on first run — generate fresh questions |
| `file-upload` fails | Check that kind and name match an existing backlog item: `swarm-manager backlog get --kind <kind> --name <name>` |
| Archive contains a full PRD with detailed requirements | Many questions may already be answered — focus only on genuine gaps |
| All critical questions are already answered by context | Generate fewer questions (minimum 3) focused on nice-to-have clarity |
| Prior run has 10 questions, all still relevant | Do not add more — respect the budget ceiling |
