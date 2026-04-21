# Workshop: Iterative Research Exploration

## Purpose

Run one workshop round for a **research** backlog item. Analyze the research question, explore investigation directions, generate targeted decisions and informational findings, self-assess readiness across 5 research-oriented dimensions, and update the draft conclusion based on accumulated user responses and findings.

Research items are fundamentally different from other kinds (idea, fix, execute, chore). The workshop rounds ARE the research work itself — each round investigates, discovers, and refines understanding. The goal is not an implementation plan but a research conclusion with actionable findings.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read research-conclusion-authoring` — canonical conclusion structure, mandatory sections, quality gates, and guardrails for `conclusion.md`.

**Required reading:** `prompt-manager skill read swarm-manager-initiative-context` — how to load the initiative's members, upstream, and downstream in one call, and the reuse-before-create heuristic that your conclusion's Actions section must honor.

## Scope

**In scope:**
- Reading all existing item context (spec, conclusion, prior workshop rounds, archive, user files)
- Investigating the research question through codebase exploration, analysis, and synthesis
- Generating a mix of decisions and informational items tailored to the research question
- Self-assessing readiness across 5 research-oriented dimensions
- Updating `conclusion.md` with refined findings and actions based on accumulated answers
- Writing the round file (`workshop/round-NNN.json`)

**Out of scope:**
- Implementing changes to the codebase (research produces conclusions, not code changes)
- Modifying `archive/` — it contains user-provided materials and must not be altered
- Queueing the item for execution
- Creating implementation plans (research items produce `conclusion.md`, not `plan.md`)

**Important:** This skill runs within Swarm Manager. Any actions that result from research findings should create work through the swarm-manager CLI (e.g., creating new backlog items), not attempt direct implementation.

## Output Requirements

All writes via CLI using `--stdin` with a heredoc:
```bash
swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path <path> --stdin <<'EOF'
<content>
EOF
```

### Every Round Produces

1. **`workshop/round-{{ROUND_NUMBER}}.json`** — the round file (see schema below)
2. **`conclusion.md`** — updated research conclusion (create if first round, update if subsequent)

## Workshop Round Schema

```json
{
  "round": {{ROUND_NUMBER}},
  "generated_at": "<ISO-8601 timestamp>",
  "mode": "workshop",
  "pending_synthesis": false,
  "readiness": {
    "problem_clarity": 0,
    "scope_defined": 0,
    "approach_solid": 0,
    "testable": 0,
    "risk_awareness": 0
  },
  "items": [
    {
      "id": "d1",
      "type": "decision",
      "topic": "Investigation direction",
      "context": "Why this matters and what was found",
      "options": [
        {"key": "A", "label": "Explore approach X", "rationale": "Covers the most common case", "recommended": true},
        {"key": "B", "label": "Investigate approach Y", "rationale": "Addresses edge cases"},
        {"key": "C", "label": "Other", "rationale": "Provide your own direction"}
      ],
      "selected": null,
      "freeform": null,
      "notes": null
    },
    {
      "id": "i1",
      "type": "info",
      "text": "Found that the existing implementation uses pattern Z, which contradicts the initial assumption"
    }
  ],
  "plan_updates": "Brief description of what conclusion sections were created or updated this round"
}
```

Every workshop-generated round must set `"mode": "workshop"` and `"pending_synthesis": false`. Swarm Manager flips `pending_synthesis` to `true` only after the user saves fully-answered decisions.

### Item Types

| Type | Purpose | User Action |
|------|---------|-------------|
| `decision` | A clarification or direction choice for the user to guide the investigation. The agent presents researched alternatives with rationale for each. | Select an option (A, B, C...) or choose "Other" and provide freeform input. Optional notes. |
| `info` | Share a noteworthy finding, observation, or important discovery from the investigation | Read-only — no action needed |

### Decision Item Guidelines

- Every decision MUST have at least 2 options (A, B) and should usually include an "Other" option as the last choice
- Options are lettered A, B, C, D... (not numbered)
- Each option needs a clear `label` AND `rationale` explaining tradeoffs
- Set `"recommended": true` on exactly one option per decision to indicate the agent's pick
- Research decisions typically ask: "Which direction should I investigate?", "Is this what you meant?", "Should I go deeper on X or move to Y?"
- For scoping questions (like "how broad should the investigation be?"), present concrete boundaries as options
- Use `id` prefixes: `d1`, `d2`... for decisions; `i1`, `i2`... for info items

### Readiness Dimensions (Research-Oriented)

Score each dimension honestly from 0-3 based on the CURRENT state of the conclusion:

| Dimension | What It Measures | 0 | 1 | 2 | 3 |
|-----------|-----------------|---|---|---|---|
| `problem_clarity` | Is the research question well-defined? | No research question articulated | Vague topic area | Clear question, some ambiguity in scope | Precise, well-bounded research question |
| `scope_defined` | Are investigation boundaries clear? | No boundaries set | General area identified | Clear in-scope/out-of-scope boundaries | Crisp boundaries with explicit exclusions |
| `approach_solid` | Is the methodology sound and findings substantive? | No methodology or findings | General direction, preliminary observations | Sound methodology, substantive findings emerging | Rigorous methodology, comprehensive findings with evidence |
| `testable` | Can we verify the findings are accurate? | No verification possible | Some claims could be checked | Key findings have verification paths | All findings verifiable with documented evidence |
| `risk_awareness` | Are limitations and unknowns identified? | Not considered | Some gaps noted | Key limitations documented with impact | Comprehensive limitations, confidence levels, and knowledge gaps |

**Scoring rules:**
- Be honest — do not inflate scores to appear further along than reality
- Score based on conclusion content, not assumptions
- A dimension at 0 means no relevant content exists in the conclusion for that area
- A dimension at 3 means the research on that aspect is thorough and well-supported

## Research Conclusion Format

The `conclusion.md` file structure, mandatory sections, quality gates, and guardrails are defined by the `research-conclusion-authoring` skill (loaded via required reading above). Follow that skill exactly when creating or updating `conclusion.md`.

**Workshop-specific notes:**
- Not every section needs content from round 1. Fill what you can and leave sections as `<!-- TBD -->` when information is insufficient. Each subsequent round should fill more sections.
- On the first round, create a scaffold with the research question and initial findings from exploration.
- On subsequent rounds, refine findings and fill gaps based on accumulated investigation and user responses.

## Instructions

You are running workshop round {{ROUND_NUMBER}} for a swarm-manager **research** backlog item.

**Item context:**
- Kind: research
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get --kind research --name {{ITEM_NAME}}
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Then read each available artifact:
   - `conclusion.md` — current research conclusion draft (if exists)
   - `workshop/` — all prior round files (to understand what's been investigated, asked, answered)
   - `spec.json` — original item description and metadata
   - `archive/` — user-provided materials
   - Any user-uploaded files
   - **Initiative context** — If this item belongs to initiative `{{ITEM_INITIATIVE}}`, load the full neighborhood in one call:
     ```bash
     swarm-manager initiatives context --name {{ITEM_INITIATIVE}}
     swarm-manager initiatives files --name {{ITEM_INITIATIVE}}
     ```
     The `context` command returns the initiative + its member items (with current status and depends_on) + upstream initiatives (what this blocks on) + downstream initiatives (what this unblocks). Use this to:
     - understand which sibling items already cover parts of the research question
     - identify sibling items that may be invalidated, reprioritized, or moved by findings
     - detect cross-initiative sequencing implications (e.g., your findings reveal this initiative no longer depends on an upstream)
     
     Read initiative files (orchestration summaries, decision logs, strategy docs) for additional strategic context.

2. **Analyze prior rounds** (if ROUND_NUMBER > 1)

   Review all prior workshop rounds. For each:
   - Note decisions with a `selected` value — these are settled, incorporate into the conclusion
   - Note decisions with `selected: null` — still pending, do not re-ask unless context has materially changed
   - Note any freeform responses on "Other" selections — incorporate these as user direction
   - Note info items from prior rounds — these are accumulated findings, build on them

3. **Define the research question** (round 1) or **refine it** (subsequent rounds)

   On round 1:
   - Extract the core research question from the item title, description, and any archive materials
   - Identify what a successful answer would look like
   - Determine what constraints or scope limits apply

   On subsequent rounds:
   - Refine the question based on user decisions and accumulated findings
   - Narrow or broaden scope as directed by user responses

4. **Investigate**

   This is the core research work. Based on the current question and direction:
   - Search relevant codebase areas
   - Review documentation and existing implementations
   - Analyze patterns and connections
   - Identify evidence for or against hypotheses
   - Document all sources and reasoning

   **Methodological rigor** (adapted from research best practices):
   - Define what you are looking for before searching
   - Plan which sources and methods you will use
   - Gather information systematically, not haphazardly
   - Analyze findings by synthesizing across sources
   - Note contradictions, gaps, and confidence levels

5. **Generate workshop items**

   Based on the investigation, produce a focused set of items:

   **Target counts:**

   | Decisions | Info | Focus |
   |-----------|------|-------|
   | 2-4 | 2-5 | Investigation direction, scope clarification, methodology, key findings |

   Research rounds typically have MORE info items than other kinds because the findings themselves are valuable outputs. Decisions guide the direction of further investigation.

   **Initiative-impact decisions to consider** (any round, when findings surface implications):
   - "Does any finding supersede a sibling item in this initiative? If so, should it be deleted, retitled, or reprioritized?"
   - "Do findings change this initiative's `depends_on`? (e.g., an upstream is no longer a blocker; a new dependency surfaces.)"
   - "Does any finding affect an item in an upstream or downstream initiative that the orchestrator should know about?"

   Answered decisions of this form become the driver of `Update backlog item`, `Delete backlog item`, or `Update initiative` actions in the final `conclusion.md`.

   **Decision examples for research:**
   - "Which subsystem should I investigate next?"
   - "Is this the aspect of X you wanted explored?"
   - "Should I go deeper on finding Y or move to a new area?"
   - "The evidence suggests Z — should I verify this further or accept it?"

   **Quality rules:**
   - Decisions should clarify direction or scope, not general curiosity
   - Info items should share genuinely useful findings with evidence
   - Do not repeat decisions from prior rounds unless context has materially changed
   - Do not re-present decisions that the user has already resolved
   - Pre-select options when inferable from existing context (set `selected` to the key)
   - Use IDs like `d1`, `d2`... for decisions, `i1`, `i2`... for info items (unique within the round)

6. **Score readiness**

   Evaluate each dimension honestly based on the current state of the conclusion AFTER incorporating findings from this round. Use the research-oriented scoring rubric above.

7. **Update conclusion.md**

   Incorporate all findings and settled decisions into the conclusion:
   - Resolved decisions become established directions and facts
   - Freeform responses on "Other" selections become user-specified investigation paths
   - New findings expand the Findings section
   - If this is round 1, create the scaffold with research question and initial findings
   - If subsequent round, refine findings and fill gaps
   - Update the Actions section as the research reveals what should happen next

   ```bash
   swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path conclusion.md --stdin <<'EOF'
   <updated conclusion content>
   EOF
   ```

8. **Write the round file**

   ```bash
   swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path workshop/round-{{ROUND_NUMBER}}.json --stdin <<'EOF'
   <round JSON>
   EOF
   ```

   Use zero-padded 3-digit round numbers: `round-001.json`, `round-002.json`, etc.

9. **Verify outputs**

   ```bash
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Confirm both `conclusion.md` and `workshop/round-{{ROUND_NUMBER}}.json` were created.

### Readiness Progression Guidance

As rounds progress, your focus should shift:

| Rounds Completed | Primary Focus |
|-----------------|---------------|
| 0 (first round) | Define research question, initial exploration, identify key areas |
| 1-2 | Deep investigation of primary areas, methodology refinement |
| 3-4 | Synthesis, cross-referencing, verification of key findings |
| 5+ | Polish conclusions, identify remaining gaps, finalize actions |

**When to suggest readiness:** If all dimensions are at 2+ and you believe the conclusion is solid enough to act on, include an info item noting: "This research appears ready for conclusion. Consider reviewing and processing." Do not inflate scores to reach this point — the system applies a boost formula that accounts for thoroughness over multiple rounds.

## Anti-Patterns

- **Don't** inflate readiness scores — be honest about what's missing
- **Don't** repeat decisions from prior rounds that were already resolved
- **Don't** present decisions with fewer than 2 options
- **Don't** generate more items than the target counts — focus on highest-impact gaps
- **Don't** modify files in `archive/` — these are user-provided
- **Don't** write files directly to disk — always use the backlog CLI
- **Don't** skip reading prior rounds — context accumulates across rounds
- **Don't** leave conclusion.md unchanged — every round should advance the conclusion
- **Don't** present decisions that could be resolved by reading existing context
- **Don't** try to produce an implementation plan — research produces conclusions, not plans
- **Don't** implement changes directly — create backlog items for follow-up work
- **Don't** present findings without evidence or source references
- **Don't** omit the "Other" option unless the choices are truly exhaustive (e.g., yes/no)

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `file-get` returns 404 for conclusion.md | Normal on first round — create the scaffold |
| `file-get` returns 404 for workshop/ | Normal on first round — create the directory with round-001.json |
| Prior round has unresolved decisions | Still pending — don't re-present, they're waiting for user input |
| All readiness dimensions already at 3 | Unusual but possible — generate minimal items focused on verification, or note readiness in an info item |
| Conflicting information between sources | Apply source authority: user answers > accepted decisions > conclusion.md > spec.json > archive |
| Very large workshop history | Focus on the latest 2-3 rounds and the settled decisions from earlier rounds |
| Research question is too broad | Generate a decision asking user to narrow scope with concrete options |
| Research question is too narrow | Note this as an info item and suggest broadening with specific directions |
