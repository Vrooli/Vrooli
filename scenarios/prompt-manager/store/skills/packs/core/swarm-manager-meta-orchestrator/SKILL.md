# Meta-Orchestrator: Vision to Backlog Pipeline

## Purpose

Translate a high-level vision (whiteboard photo, stream-of-consciousness text, meeting notes, strategic brief) into a complete set of backlog items with classifications, priorities, dependencies, and initiative groupings. This skill is an **import and planning tool** — it takes messy ideas and structures them into Swarm Manager's backlog. It does NOT manage ongoing workshopping or execution; those happen through the Swarm Manager UI and other skills after items are created.

**Required reading:**
- `prompt-manager skill read swarm-manager-backlog-tools` — data model, folder structure, CLI commands

## Input Context

You are given a high-level vision from a user (human or director-swarm agent). It may arrive as:

- **Raw text** — stream-of-consciousness, meeting notes, bullet lists
- **Image** — whiteboard photo, architecture diagram, screenshot
- **Mixed** — text with embedded references to images or files
- **Strategic brief** — structured document with goals, constraints, timeline

**Source context (when driven by director-swarm):**
- `{{DECISIONS_JSONL}}` — strategic decisions log (Now/Near/Far priorities)
- `{{TASKS_JSON}}` — current task assignments
- `{{LAST_HANDOFF}}` — most recent director handoff notes

## Scope

**In scope:**
- Parsing vision input into discrete candidate work items
- Conversational clarification of priorities, dependencies, and sequencing
- Grouping items into initiatives (logical bundles of related work)
- Batch-creating backlog items via CLI (creation auto-triggers workshop round 1)
- Setting cross-item dependencies
- Updating existing backlog items when new context emerges (not duplicating them)
- Re-prioritizing items as new information emerges during the session

**Out of scope:**
- Managing or monitoring workshop rounds (handled via Swarm Manager UI)
- Executing plans (delegates to `swarm-manager-process-*`)
- Deep research (delegates to `swarm-manager-research-*`)
- Modifying `archive/` folders on existing items
- Direct code changes

---

## Phase 1: INTAKE — Extract Work Items

**Goal:** Parse the vision input into a structured list of candidate work items.

### Agent Behavior

1. Read the entire vision input (text, image description, or both)
2. Check the existing backlog for potential duplicates: `swarm-manager backlog list`
3. Extract discrete work items — each representing a distinct unit of value or work
4. For each candidate item, determine:
   - **Kind** (idea / research / fix / execute / chore)
   - **Title** (imperative form, concise)
   - **Description** (one sentence)
   - **Priority** (1-10, where 1 = highest)
   - **Effort** (XS / S / M / L / XL — see effort estimation guidelines below)
   - **Scope** — the target scenario or directory (e.g., `scenarios/web-console`). Identify which part of the codebase each item targets. If an item spans multiple scenarios, note it for splitting in Phase 2.
   - **Tags** (relevant categorization)
   - **Confidence** (0.0-1.0) in the classification
5. Present a structured summary table to the user

### Handling Different Input Types

| Input Type | Strategy |
|------------|----------|
| **Text only** | Direct extraction. Split on semantic boundaries (topic shifts, "also", "and another thing", numbered lists). |
| **Image only** | Describe what you observe. Extract text/structure, identify groupings (boxes, arrows, columns). Flag low-confidence extractions. |
| **Mixed** | Process text first, then use the image to supplement/confirm. Cross-reference items found in both. |
| **Short input** (< 3 items likely) | Still go through intake. Offer: "This seems focused on [topic]. Should I also consider [related area]?" |
| **Large input** (> 15 items likely) | Group into themes first, present themes for confirmation, then expand into individual items. |

### Mapping Messy Input to Swarm Manager's Model

Real-world input (whiteboards, brain dumps) often has structure that doesn't map cleanly to Swarm Manager's flat initiative → items model. Use your best judgment to bridge the gap:

- **Nested sub-groups** (e.g., "Monetization" containing "bundle-manager" and "brand-manager" sub-lists): Flatten into individual items with descriptive titles and use tags or name prefixes to preserve the grouping (e.g., `monetization-bundle-manager-cleanup`). Use `depends_on` to preserve any implied ordering.
- **Numbered lists**: Don't assume numbers mean priority. They might indicate phases, logical grouping, or just the order someone thought of things. Ask once: "Your items are numbered 1-3 — does this represent execution order, priority, or just grouping?"
- **Color-coding / spatial clusters**: Treat visual grouping boundaries (colors, boxes, whiteboard regions) as initiative boundaries unless the user indicates otherwise.
- **Items that don't fit a single kind**: Pick the closest kind and note the ambiguity. Don't force-fit — the workshopping phase will refine it later.

### Handling Duplicates

Duplicate detection happens at two levels:

1. **Within the current session**: If the same concept appears in multiple groups (e.g., "approval gating" under both Deployment and Swarms), treat it as **one item with multiple dependents**, not two items. Ask: "I see [item] mentioned in [group A] and [group B]. I'll create it once and mark both groups as depending on it — sound right?"
2. **Against existing backlog**: Check `swarm-manager backlog list` before creating. If a match exists, **update the existing item** with any new context from this session rather than creating a duplicate. Use `backlog update` to add new information, tags, or dependencies.

### Effort Estimation

Assign an effort estimate (XS/S/M/L/XL) to each item for consistent sizing across sessions:

| Size | Criteria | Examples |
|------|----------|----------|
| **XL** | New application/scenario from scratch; major architectural overhaul | "Build dedicated emulator app", "Create new SaaS product" |
| **L** | Major feature for existing app; significant multi-component refactor | "Add live desktop subscription", "Interop with other clients/apps" |
| **M** | Moderate feature with clear scope; multi-file changes | "Add NFC option to web-console", "Expand swarm-manager UX" |
| **S** | Small feature, bug fix, or targeted cleanup | "Fix sandbox edge case", "Clean up brand-manager" |
| **XS** | Trivial change, config tweak, documentation | "Update marketplace listing", "Fix typo in prompt" |

**Bump up one level** if the item: involves unfamiliar technology, crosses multiple scenarios, requires new infrastructure, or has unclear requirements even after clarification.

### Handling Vague Items

When items are too vague to classify confidently (e.g., "clean up", "various fixes"):

1. Present them grouped as "needs expansion" in the intake output
2. Offer your best-guess classification with low confidence
3. Let the user choose: **clarify now** (produces a better item) or **defer** (keep it vague; the auto-triggered workshop round will ask for clarification later)

Don't guess silently — flag the ambiguity and let the user decide when to resolve it.

### Intake Output Template

```
I've parsed your vision into [N] candidate work items:

| # | Kind    | Title                        | Scope                    | Priority | Effort | Confidence |
|---|---------|------------------------------|--------------------------|----------|--------|------------|
| 1 | idea    | Build real-time dashboard     | scenarios/web-console    | 3        | L      | 0.9        |
| 2 | fix     | Fix auth timeout on refresh   | scenarios/swarm-manager  | 1        | S      | 0.95       |
| 3 | research| Evaluate vector DB options    | scenarios/agent-manager  | 5        | M      | 0.7        |
| ...                                                                                                   |

Items I'm less sure about:
- "[ambiguous text]" — Could be [kind A] or [kind B]. Which fits better?
- "[vague reference]" — Couldn't extract a clear action. Clarify now, or keep vague and let workshopping sort it out?

Potential duplicates with existing backlog:
- "[item title]" looks similar to existing [kind]/[name]. Skip or create separate?

Before I proceed:
1. Are any items missing from what you described?
2. Should any items be merged or split?
3. Any kind classifications feel wrong?
```

---

## Phase 2: CLARIFY & STRUCTURE — Priorities, Dependencies, Initiatives

**Goal:** Refine the item list through targeted questions about relationships, priorities, and grouping.

### Agent Behavior

1. Analyze inter-item dependencies (does item B require item A to be done first?)
2. Identify natural groupings (items that share a theme, target the same scenario, or form a feature set)
3. Suggest initiative groupings with names
4. Ask clarifying questions in **batched form** (not one-at-a-time)

### Clarifying Question Rounds

**Round 1 — Structure (always ask):**
- Dependencies: "I see [item A] likely needs to be done before [item B, C]. Is that right? Any other ordering constraints?"
- Grouping: "These items seem to form [N] natural groups: [group names]. Does this grouping make sense?"
- Missing items: "To complete [initiative X], you might also need [suggested item]. Should I add it?"
- Cross-scenario items: If any item spans multiple scenarios, suggest splitting: "Item [X] touches both [scenario A] and [scenario B]. I'd recommend splitting it into two items with `depends_on` links so each has a clear `scope`. Should I split it?"
- Ambiguous scope: If a candidate item's target scenario is unclear, ask: "Item [X] — which scenario does this target? (e.g., `scenarios/web-console`, `scenarios/swarm-manager`, or something else?)"

**Round 2 — Priority (ask if priorities are ambiguous):**
- Timeline: "Which of these are NOW (this week), NEAR (this month), or FAR (this quarter)?"
- Blockers: "Are any of these blocked by external factors (people, resources, decisions)?"
- MVP cut: "If you could only do 3 of these [N] items, which 3?"

**Round 3 — Refinement (ask only if needed):**
- Scope: "For [item X], is this a quick fix or a larger effort?"
- Ownership: "Should any of these be assigned to specific teams?"
- Risk: "Any items here that feel particularly risky or uncertain?"

### Batching Rules

- Present all Round 1 questions together in one message
- Wait for user response before presenting Round 2
- Skip Round 2/3 if the user's vision was already clear and structured
- Never ask more than 5 questions in a single message
- Frame questions as confirmations where possible ("I think X — is that right?" vs "What is X?")

### Clarify Output Template

```
Great, here's the refined plan:

## Initiative: [Name]
[1-sentence description of the initiative's goal]

| Order | Kind | Title | Priority | Depends On |
|-------|------|-------|----------|------------|
| 1     | fix  | ...   | 1        | —          |
| 2     | idea | ...   | 3        | #1         |
| 3     | idea | ...   | 3        | #1         |

## Initiative: [Name]
...

## Standalone Items
| Kind | Title | Priority |
|------|-------|----------|
| ...  | ...   | ...      |

[Round 2 questions if needed, or:]
Ready to create these items? I'll batch-create [N] backlog items across [M] initiatives.
```

---

## Phase 3: GENERATE — Create Items and Initiatives

**Goal:** Batch-create all backlog items and set up initiatives. Workshop round 1 is auto-triggered on item creation — do NOT manually trigger it.

### Agent Behavior

1. Batch-create all backlog items with `--initiative` flag (auto-creates initiative if needed)
2. Optionally update initiative metadata (title, description) via `initiatives update`
3. Report creation summary to user

### CLI Sequence

```bash
# Step 1: Batch-create all items with initiative assignment.
# The --initiative flag auto-creates the initiative if it doesn't exist.
# Initiative is a request-level field, NOT per-item.
cat > /tmp/meta-orch-items.json <<'EOF'
{
  "items": [
    {
      "name": "<name>",
      "title": "<title>",
      "description": "<description>",
      "kind": "<kind>",
      "priority": <N>,
      "effort": "<XS|S|M|L|XL>",
      "scope": "<scenarios/target-scenario>",
      "tags": ["<tag1>", "<tag2>"],
      "depends_on": ["<kind>/<name>", ...]
    },
    ...
  ]
}
EOF
swarm-manager backlog batch-create --file /tmp/meta-orch-items.json --initiative <initiative-name>

# Step 2 (optional): Update initiative with descriptive title and description.
swarm-manager initiatives update --name <initiative-name> --data '{
  "title": "<Initiative Title>",
  "description": "<1-sentence description>"
}'
# NOTE: Workshop round 1 is auto-triggered on item creation. Do NOT manually trigger it.
```

For items without batch-create available (fallback), create individually:
```bash
swarm-manager backlog create --data '{
  "name": "<name>",
  "title": "<title>",
  "description": "<description>",
  "kind": "<kind>",
  "priority": <N>,
  "effort": "<XS|S|M|L|XL>",
  "scope": "<scenarios/target-scenario>",
  "tags": ["<tag1>", "<tag2>"],
  "depends_on": ["<kind>/<name>", ...]
}'
# Then assign to initiative separately:
swarm-manager initiatives add-items --name <initiative-name> --items <kind>/<name>
```

### Generate Output Template

```
Created [N] backlog items across [M] initiatives:

## Initiative: [Name] ([X] items)
- [x] created: [kind]/[name] — [title] (effort: [size])
- [x] created: [kind]/[name] — [title] (effort: [size])

## Standalone Items ([Y] items)
- [x] created: [kind]/[name] — [title] (effort: [size])

Workshop round 1 has been auto-triggered for all new items.
You can monitor progress and answer workshop decisions through the Swarm Manager UI.
```

---

## Director-Swarm Integration

When the meta-orchestrator is invoked by the director-swarm (agent-driven mode), the following adaptations apply.

### Reading Strategic Context

Before intake, read and incorporate:
```bash
# Read strategic decisions
swarm-manager backlog file-get --kind idea --name _director-context --path decisions.jsonl 2>/dev/null || true

# Or read directly if available:
cat scenarios/prompt-manager/store/teams/director-swarm/shared/decisions.jsonl
cat scenarios/prompt-manager/store/teams/director-swarm/shared/tasks.json
cat scenarios/prompt-manager/store/teams/director-swarm/members/director/last-handoff.md
```

### Respecting Now/Near/Far

- Map items to the Now/Near/Far framework from `decisions.jsonl`
- Items aligned with **NOW** priorities → priority 1-3
- Items aligned with **NEAR** priorities → priority 4-6
- Items aligned with **FAR** priorities → priority 7-10
- Items conflicting with strategic decisions get flagged with a warning but still created — the director makes the final call

### Agent Mode Differences

| Phase | Human Mode | Agent/Director Mode |
|-------|-----------|-------------------|
| Intake | Full structured table | Same, but summary presented to director agent |
| Clarify | 2-3 question rounds | Skip Round 2/3 if strategic context provides clear priorities |
| Generate | Same | Same |

### Reporting Back

After completing Phase 3, write a summary:
```bash
swarm-manager backlog file-upload --kind <first-item-kind> --name <first-item-name> --path orchestration-summary.md --stdin <<'EOF'
# Meta-Orchestrator Summary

## Session: [date-topic]
## Source: [vision description]
## Date: [ISO-8601]

## Items Created
[table of items with kind, name, title, priority, initiative]

## Initiative Structure
[initiative groupings with dependency arrows]

## Strategic Alignment
- NOW items: [list]
- NEAR items: [list]
- FAR items: [list]
- Conflicts with current strategy: [list or "none"]

## Next Steps
[what needs to happen next — workshop decisions will surface through Swarm Manager UI]
EOF
```

---

## Relationship to Existing Skills

| Skill | Relationship | Details |
|-------|-------------|---------|
| `swarm-manager-classify-capture` | **Subsumes for multi-item** | Meta-orchestrator performs classification inline during Phase 1. For single captures (one sentence, one action), the normal classify-capture pipeline remains better. |
| `swarm-manager-initialize-backlog` | **Auto-triggered** | Workshop round 1 is auto-triggered on item creation. This skill does NOT manually trigger initialization. |
| `swarm-manager-workshop` | **Downstream** | After items are created, workshop rounds are managed through the Swarm Manager UI, not this skill. |
| `swarm-manager-process-*` | **Downstream** | After workshopping completes, items flow into the normal process pipeline. Not managed by this skill. |
| `swarm-manager-backlog-tools` | **Reads from** | All CLI commands and data model references follow the backlog-tools canonical spec. |
| `swarm-manager-recommendations` | **Reads from** | When items originate from team recommendations, respects the team-to-backlog contract. |

---

## Anti-Patterns

- **Don't** create items without user confirmation — always present the plan first
- **Don't** manually trigger workshop round 1 — it's auto-triggered on item creation
- **Don't** try to manage ongoing workshop rounds — that's done through the Swarm Manager UI
- **Don't** ask one question at a time — batch related questions together
- **Don't** skip Phase 2 for large visions (>5 items) — dependencies matter
- **Don't** create duplicate items — check within the session AND against the existing backlog; update existing items instead
- **Don't** ignore strategic context when running in director-swarm mode
- **Don't** modify items created by other skills without user approval
- **Don't** assume the user wants all extracted items — some may be noise
- **Don't** inflate confidence scores on ambiguous extractions — flag them for clarification
- **Don't** silently guess on vague items — flag them and let the user choose to clarify now or defer to workshopping

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Vision input produces 0 items | Ask user to elaborate. If truly empty, explain that no actionable items were found. |
| Duplicate of existing backlog item | Check `swarm-manager backlog list` before creating. Present potential duplicates — update existing items rather than creating new ones. |
| Same concept in multiple groups | Treat as one item with multiple dependents, not duplicate items. Ask user to confirm. |
| Initiative has circular dependencies | Detect during Phase 2 and present to user. At least one dependency must be dropped. |
| User wants to change items after Phase 3 | Items can be updated via `backlog update`. Re-run clarify phase for the changed items. |
| Director provides conflicting strategic signals | Flag the conflict explicitly. Do not auto-resolve — escalate for clarification. |
| Too many items (>20) for a single session | Suggest splitting into 2-3 focused sessions by initiative. Process one initiative at a time. |
| Image with ambiguous text | Present your best reading and flag uncertain words. Ask user to confirm/correct before classifying. |
| Nested structure doesn't fit flat model | Flatten with descriptive titles, use tags/name-prefixes to preserve grouping, and `depends_on` for ordering. |
| Batch-create not available | Fall back to individual `backlog create` calls in a loop. Same result, just slower. |
| Vague item user won't clarify | Keep it vague — create with low confidence rating. Auto-triggered workshop round will ask for clarification. |
