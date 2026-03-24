# Meta-Orchestrator: Vision to Backlog Pipeline

## Purpose

Translate a high-level vision (whiteboard photo, stream-of-consciousness text, meeting notes, strategic brief) into a complete set of backlog items with classifications, priorities, dependencies, and initiative groupings. This skill operates ABOVE the individual-item pipeline (classify → initialize → workshop → plan → execute) and orchestrates many items at once through conversational back-and-forth.

**Required reading:**
- `prompt-manager skill read swarm-manager-backlog-tools` — data model, folder structure, CLI commands
- `prompt-manager skill read swarm-manager-workshop` — workshop round schema (this skill aggregates workshop decisions across items)

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
- Batch-creating backlog items via CLI
- Setting cross-item dependencies
- Triggering batch initialization (workshop round 1 for each item)
- Monitoring initiative progress and surfacing aggregated workshop decisions
- Re-prioritizing items as new information emerges during the session

**Out of scope:**
- Running individual workshop rounds (delegates to `swarm-manager-workshop`)
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

### Intake Output Template

```
I've parsed your vision into [N] candidate work items:

| # | Kind    | Title                        | Priority | Confidence |
|---|---------|------------------------------|----------|------------|
| 1 | idea    | Build real-time dashboard     | 3        | 0.9        |
| 2 | fix     | Fix auth timeout on refresh   | 1        | 0.95       |
| 3 | research| Evaluate vector DB options    | 5        | 0.7        |
| ...                                                                 |

Items I'm less sure about:
- "[ambiguous text]" — Could be [kind A] or [kind B]. Which fits better?
- "[vague reference]" — Couldn't extract a clear action. Drop or clarify?

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

**Goal:** Batch-create all backlog items, set up initiatives, and trigger initialization.

### Agent Behavior

1. Batch-create all backlog items with `--initiative` flag (auto-creates initiative if needed)
2. Optionally update initiative metadata (title, description) via `initiatives update`
3. Trigger initialization for each item (spawns workshop round 1)
4. Report creation summary to user

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

# Step 3: Trigger initialization for each item.
swarm-manager backlog research --kind <kind> --name <name> --data '{"mode":"initialize"}'
```

For items without batch-create available (fallback), create individually:
```bash
swarm-manager backlog create --data '{
  "name": "<name>",
  "title": "<title>",
  "description": "<description>",
  "kind": "<kind>",
  "priority": <N>,
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
- [x] created: [kind]/[name] — [title]
- [x] created: [kind]/[name] — [title]
- [x] initialized: workshop round 1 spawned for all items

## Standalone Items ([Y] items)
- [x] created: [kind]/[name] — [title]

All items have been queued for initialization (workshop round 1).
I'll monitor their progress and surface decisions when ready.
```

---

## Phase 4: ORCHESTRATE — Monitor and Aggregate Decisions

**Goal:** Monitor workshop progress across all items and present decisions in batches rather than per-item.

### Agent Behavior

1. Poll item status: `swarm-manager overview` (or list items by initiative tag)
2. Read workshop rounds from all items: `swarm-manager backlog file-get --kind <kind> --name <name> --path workshop/round-001.json`
3. Aggregate pending decisions across items
4. Group similar decisions (e.g., "auth approach" decisions across 3 related items)
5. Present consolidated decision batches to the user (max 10 per message)
6. Apply user answers back to the appropriate items via file-upload
7. Trigger next workshop round for items that received responses

### Aggregated Decision Template

```
## Workshop Decisions — Round 1 Aggregate

### Cross-Cutting Decision: Authentication Approach
Affects: real-time-dashboard, user-management, api-gateway
- A: OAuth with Google (recommended for 2 of 3 items)
- B: JWT with custom auth
- C: Other

### Item-Specific Decisions

**real-time-dashboard** (readiness: 1.4/3.0)
- D1: Data refresh strategy — A: WebSocket streaming | B: Polling | C: Other
- D2: Chart library — A: Recharts | B: D3 direct | C: Other

**fix-auth-timeout** (readiness: 2.1/3.0)
- D1: Root cause — A: Token expiry race | B: Network timeout | C: Other

### Informational Items
- [i1] Found existing WebSocket infrastructure in app-monitor that could be reused
- [i2] The auth timeout has been reported 3 times in the last week

Your answers (format: "D1:A, D2:B" or answer naturally):
```

### Answer Routing

When the user answers:
1. Parse responses and map to the correct item's workshop round
2. For cross-cutting decisions, apply the answer to all affected items
3. Update each item's workshop round file with `selected` values:
   ```bash
   swarm-manager backlog file-upload --kind <kind> --name <name> --path workshop/round-001.json --stdin <<'EOF'
   <updated round JSON with selected values filled in>
   EOF
   ```
4. Trigger next workshop round: `swarm-manager backlog research --kind <kind> --name <name> --data '{"mode":"workshop"}'`

---

## Phase 5: EXECUTE COORDINATION — Queue Respecting Dependencies

**Goal:** Queue items for execution in the correct order based on dependencies and readiness.

### Agent Behavior

1. Monitor readiness scores across all items (from latest workshop round)
2. An item is "ready" when average readiness >= 2.0 across all 5 dimensions
3. Before queuing, check dependency chain — only queue items whose dependencies are completed
4. Queue ready items: `swarm-manager backlog batch-queue --items <kind/name>,<kind/name> --execute`
5. Track initiative completion percentage
6. Surface blocking items that need attention

### Execute Coordination Template

```
## Initiative: [Name] — Progress Report

| Item | Readiness | Status | Blocked By |
|------|-----------|--------|------------|
| fix-auth-timeout | 2.6/3.0 | ready to queue | — |
| real-time-dashboard | 1.8/3.0 | needs workshop | — |
| user-management | 1.2/3.0 | needs workshop | fix-auth-timeout |

Actions available:
1. Queue "fix-auth-timeout" for execution (no blockers, readiness sufficient)
2. Run another workshop round for "real-time-dashboard" (2 open decisions)
3. Wait for "fix-auth-timeout" to complete before workshopping "user-management"

Which actions should I take?
```

For single-item queue (fallback if batch-queue unavailable):
```bash
swarm-manager backlog queue --kind <kind> --name <name> --execute
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
| Orchestrate | All decisions surfaced | Auto-resolve decisions matching strategic context; escalate judgment calls |
| Execute | User chooses actions | Queue NOW items first; flag items outside active scenario allowlist |

### Reporting Back

After completing phases 3-5, write a summary:
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

## Decisions Made
- Auto-resolved: [N] (based on strategic context)
- Pending review: [N]

## Next Steps
[what needs to happen next]
EOF
```

---

## Relationship to Existing Skills

| Skill | Relationship | Details |
|-------|-------------|---------|
| `swarm-manager-classify-capture` | **Subsumes for multi-item** | Meta-orchestrator performs classification inline during Phase 1. For single captures (one sentence, one action), the normal classify-capture pipeline remains better. |
| `swarm-manager-initialize-backlog` | **Delegates to** | After Phase 3, triggers initialization for each created item via `research --data '{"mode":"initialize"}'`. |
| `swarm-manager-workshop` | **Aggregates output of** | During Phase 4, reads workshop rounds from multiple items and presents decisions in consolidated view. Individual rounds still run via the workshop skill. |
| `swarm-manager-process-*` | **Feeds into** | After Phase 5, queued items flow into the normal process pipeline. |
| `swarm-manager-backlog-tools` | **Reads from** | All CLI commands and data model references follow the backlog-tools canonical spec. |
| `swarm-manager-recommendations` | **Reads from** | When items originate from team recommendations, respects the team-to-backlog contract. |

---

## Anti-Patterns

- **Don't** create items without user confirmation — always present the plan first
- **Don't** run workshop rounds directly — delegate to `swarm-manager-workshop` via research command
- **Don't** ask one question at a time — batch related questions together
- **Don't** skip Phase 2 for large visions (>5 items) — dependencies matter
- **Don't** auto-queue items without checking dependencies
- **Don't** ignore strategic context when running in director-swarm mode
- **Don't** present more than 10 decisions in a single aggregated view — paginate
- **Don't** modify items created by other skills without user approval
- **Don't** assume the user wants all extracted items — some may be noise
- **Don't** inflate confidence scores on ambiguous extractions — flag them for clarification

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Vision input produces 0 items | Ask user to elaborate. If truly empty, explain that no actionable items were found. |
| Duplicate of existing backlog item | Check `swarm-manager backlog list` before creating. Present potential duplicates for confirmation. |
| Initiative has circular dependencies | Detect during Phase 2 and present to user. At least one dependency must be dropped. |
| Workshop initialization fails for an item | Retry once. If still failing, create the item but skip initialization and note the failure. |
| User wants to change items after Phase 3 | Items can be updated via `backlog update`. Re-run clarify phase for the changed items. |
| Director provides conflicting strategic signals | Flag the conflict explicitly. Do not auto-resolve — escalate for clarification. |
| Too many items (>20) for a single session | Suggest splitting into 2-3 focused sessions by initiative. Process one initiative at a time. |
| Image with no text | Describe what you see, extract any text/structure, present interpretation for confirmation before proceeding. |
| Batch-create not available | Fall back to individual `backlog create` calls in a loop. Same result, just slower. |
| Overview endpoint not available | Fall back to `backlog list` + manual aggregation. |
