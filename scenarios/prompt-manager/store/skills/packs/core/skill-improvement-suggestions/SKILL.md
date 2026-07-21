## Meta focus: Skill Improvement Suggestions

Analyze **{{SKILL}}** and propose meaningful improvements to its tools, wording, or structure. Improvement asks one question: **is the contract cheap and well-conditioned?** The sibling skill `skill-validation` asks the complementary question — is the contract true and executable — so route correctness findings there. Audit on two axes that compose: **efficiency** (token cost, manual loops, promotion candidates — §3) and **conditioning quality** (does the text converge behavior — §4). An inefficient skill and a poorly-conditioning skill are different defects with different fixes.

Required reading:
- `prompt-manager skill read {{SKILL}}`
- `docs/agent-system/SKILL_AUTHORING.md`
- `docs/agent-system/PROMOTION_LADDER.md`

Conditional required reading (when {{SKILL}} includes CLI guidance or CLI workflows):
- `prompt-manager skill read cli-steer`
  - `cli-steer` is the canonical guidance for setting up CLIs properly (cli-core usage, API parity, output contracts, and professional CLI UX patterns).

---

### **1. Why This Matters**

Use this skill to reduce execution friction without losing correctness.

- Improve tooling first where manual loops repeat.
- Keep skill guidance concise and decision-oriented.
- Apply the promotion ladder (`docs/agent-system/PROMOTION_LADDER.md`): interim prose -> promote -> retire.
- Prioritize changes that reduce drift, retries, and token cost across repeated use.

The economics: a 5,000-token skill read 100 times costs 500,000 input tokens; the same job done in 2,000 tokens costs 200,000. Confusion compounds the cost — unclear guidance produces wrong assumptions, wasted runs, and human intervention. The goal is maximum clarity in minimum words; every sentence earns its place.

---

### **2. Category Scope**

**In scope:**
- Suggesting new CLI tools that automate manual patterns
- Proposing improvements to existing tools (general-purpose only)
- Identifying Action candidates when one Vrooli-controlled CLI command can own deterministic execution
- Improving skill wording, structure, clarity, and conditioning quality
- Identifying where research could be automated

**Out of scope:**
- Implementing the improvements (focus is on suggestions)
- Scenario-specific feature requests (belongs in PRD or issues)
- Creating new skills (see skill-authoring-* guides)
- Tool usage instructions (belongs in Tools skills)

Lifecycle rule: `docs/agent-system/PROMOTION_LADDER.md` owns the lifecycle, the retirement and retention criteria, and — in §"Output requirement for meta analyses" — the mandate that this skill classify each major workflow instruction as `Keep` / `Collapse to Action/CLI contract` / `Delete`. Do not stop at promotion suggestions; explicitly identify what skill prose collapses or deletes after promotion.

Handoff note: if a prior `skill-validation` report exists, read its findings first and reuse that signal before adding new suggestions.

---

### **3. Defect Patterns — Efficiency Axis**

Walk {{SKILL}} against this table row by row. Each row names a defect, its observable signal, and the improvement direction. Two worked examples follow the table; they set the bar for what a real improvement looks like.

| # | Defect pattern | Signal in {{SKILL}} | Improvement direction |
|---|---|---|---|
| E1 | Complex multi-step process | Temp files, tracked IDs, 5+ sequential steps for one outcome | General-purpose tool capability that collapses the steps (worked example A) |
| E2 | Unintuitive steps | "Now do X" with no why; a step needs context the skill never set up | Add the missing why + verification inline, or promote it into the tool's output |
| E3 | Unexplained concepts | Fails the **fresh agent test** — could an agent that has never seen this skill, scenario, or codebase follow it correctly on the first try? References like "the registry" or "the pattern" with no path or example | Add path + inline example; delete internal implementation details (agents need WHAT and HOW-TO-USE, not HOW-IT-WORKS) |
| E4 | Manual research a tool could do | "Read the whole file to check…", "search for whether…" repeated across workflows | CLI search/read command with filters — structured read/write that narrows the search space and enforces schema |
| E5 | Length without decisions | Long prose, few decisions; padding, restated context, throat-clearing | Cut to decision-bearing text |
| E6 | Duplicating tool output | Skill restates startup, error-recovery, or usage text the CLI already prints on failure | Trust the tool's messages; delete the duplicate prose (it drifts, the tool doesn't) |
| E7 | Verbose command reference | >5 commands documented with flags — `--help` duplication | Add `<group> help` subcommands to the CLI; skill keeps a pointer. Docs live in the tool, always accurate |
| E8 | Async pattern where sync works | Polling loops, sleep/status-check prose | Tool `--wait` mode; the skill becomes a reference card, not a tutorial (worked example B) |
| E9 | Long-tail accretion in troubleshooting | Repeated similar entries; "run X then Y then inspect Z" chains; manual interpretation that should be CLI next-actions | Promote to CLI output contract or tool capability per `docs/agent-system/PROMOTION_LADDER.md`; delete superseded prose |

**Worked example A — collapse a multi-step process into general-purpose tool capability** (`browser-automation-studio`, real improvement):

The old smoke-test process took 7 steps: write a temporary workflow JSON with nodes/edges/metadata, execute `workflow execute --from-file`, export results with the execution ID, navigate the results folder, consult the skill's "Understanding Output" section, clean up temp files. High cognitive load, high failure rate. The fix:

```
browser-automation-studio workflow execute \
  --step navigate "http://localhost:3000" waitUntil=networkidle \
  --step screenshot fullPage=true \
  --output /tmp/results \
  --wait
# then read the auto-generated /tmp/results/README.md
```

`--step` eliminates temp files, `--output` eliminates the export command, the auto-generated README eliminates the "consult the skill" step. **Key insight:** we did not add a `--smoke-test` flag — that would be situational. We added general-purpose capabilities that happen to make smoke tests trivial but also serve debugging, CI, and demos.

**Worked example B — sync over async** (`scenario-to-desktop`, real improvement):

The skill explained an async pattern: start `pipeline-run`, then poll `pipeline-status` in a `while true; do … sleep 10; done` loop. The fix: the tool gained `--wait`, and the skill's polling tutorial collapsed to one line — `scenario-to-desktop pipeline run my-scenario --platforms linux --wait`. **Key insight:** when a skill explains polling, ask whether the tool should support `--wait` instead. That is a tool improvement suggestion, not a wording edit.

---

### **4. Defect Patterns — Conditioning Axis**

A skill is a conditioning signal, not documentation. Walk {{SKILL}} row by row against the **conditioning defect table (C1–C5)** in `docs/agent-system/SKILL_AUTHORING.md` §"Conditioning defect patterns" — that table is the single source of truth for the pattern definitions, signals, and fixes; do not restate it. The routing rule lives there too: a *confirmed* C4 (divergence probe) belongs to `skill-validation`; C1, C2, C3, and C5 belong here.

Conditioning suggestions use the SKILL WORDING IMPROVEMENT format (§5.4) and must name the defect row (C#) and the lens they improve.

---

### **5. Suggestion Formats**

#### **5.1 Tool Suggestions (New CLI capabilities)**

**When to suggest:** you observe repeated manual patterns (E1, E4) that could be automated.

**The transformation test:**

| Before (Manual) | After (Tool) |
|-----------------|--------------|
| Edit file directly | CLI command with schema validation |
| Search by reading entire file | CLI search with filters returning only relevant results |
| "Check if it worked" | Automated verification with clear pass/fail |
| Scattered notes across files | Centralized registry with query capability |

**Suggestion format:**
```
TOOL SUGGESTION: [tool-name]

Problem observed:
- [Describe the manual pattern you saw in {{SKILL}}]
- [Quantify if possible: "This pattern appears N times in the skill"]

Existing capabilities:
- [Any existing scenarios we could use directly or enhance. If none, say so and suggest a new scenario]

Proposed capabilities:
- [Command 1]: [What it does. What scenario it's for (existing or new)]
- [Command 2]: [What it does. What scenario it's for (existing or new)]

Value delivered:
- Reduces search space: [How it narrows what agent must consider]
- Enforces standards: [What schema/rules it enforces]
- Adds monitoring: [What becomes trackable/measurable]

Why this isn't too situational:
- [Explain how this benefits multiple skills/scenarios/use cases]
```

#### **5.2 Action Candidates (Expose Existing CLI Commands)**

**When to suggest:** a Vrooli-controlled CLI command already performs one deterministic operation, but agents must learn it through prose.

Check first:
- Run `prompt-manager discover "<operation>" --type all` to find existing Actions. `--type all` is *best-match relevance*, so phrase the query as the operation — not as a planning topic.
- If an exact Action exists, suggest improving or referencing it instead of creating a new one.
- If no CLI command exists, this is a CLI/tool suggestion or backlog item, not an Action candidate.

Suggestion format:
```
ACTION CANDIDATE: [action-id]

Existing CLI:
- [Exact Vrooli-controlled command]

Current prose cost:
- [Where the skill explains this manually and approximate size/usage]

Action contract:
- Inputs:
- Outputs:
- Permissions:
- runEligible recommendation:

Validation:
- [prompt-manager action validate / dry-run evidence, or reason blocked]

Prose retirement:
- Keep: [judgment/safety text]
- Collapse: [command prose replaced by Action reference]
- Delete: [fully superseded operational detail]
```

#### **5.3 Tool Improvements (Enhance existing CLI tools)**

**When to suggest:** an existing tool works but is awkward for common cases (E2, E6–E8).

**The generality test:** would this improvement help in multiple scenarios? Multiple skill types? Multiple use cases beyond the immediate one? If YES to 2+, it's a good suggestion. If only 1, it's too situational.

| Tool | ❌ Bad | Why Bad | ✅ Good | Why Good |
|------|--------|---------|---------|----------|
| browser-automation-studio | `--smoke-test` flag | Only helps smoke tests | `--step` for inline definition | Helps any quick workflow |
| browser-automation-studio | `--login-first` | Assumes one auth pattern | `--output` for direct export | Works for any workflow |

**Suggestion format:**
```
TOOL IMPROVEMENT: [tool-name]

Current limitation:
- [What users must do manually or awkwardly today]
- [Quote from {{SKILL}} showing the friction]

Proposed change:
- [Flag/feature name]: [Exact behavior]

Generality validation:
- Scenarios benefiting: [List 2-3]
- Skill types benefiting: [Steer, Search, Tools, Meta]
- Use cases: [List 3+]

Why NOT too situational:
- [Explain why this isn't a one-off convenience]
```

#### **5.4 Skill Wording/Structure Improvements**

**When to suggest:** the skill text is confusing, incomplete, poorly conditioning (any §4 row), or contains information that doesn't help the agent (E3, E5).

Check structure against the universal quality bars in `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" — do not restate them; anchor each finding to the bar or the §3/§4 defect row it violates.

**Suggestion format:**
```
SKILL WORDING IMPROVEMENT: {{SKILL}}

Current issue:
- [Quote the problematic text]
- [Defect row (E1–E9 / C1–C5) or canon bar violated]

Proposed revision:
[New text—complete enough to copy-paste]

Why this helps:
- Fresh agent understanding: [How it improves first-time comprehension]
- Token efficiency: [If applicable—how it reduces length]
- Conditioning: [If applicable—which lens improves and what gets deleted]
- Reliability: [How it reduces chance of misinterpretation]
```

---

### **6. Convergence Patterns**

**Which suggestion type applies?**

```
                    What did you observe in {{SKILL}}?
                                    │
        ┌───────────────────┬───────┴────────────┬─────────────────────┐
        ▼                   ▼                    ▼                     ▼
 Repeated manual      Tool works but       Text is confusing,   Hand-rolled rules
 pattern that could   awkward for          incomplete, or       a named standard
 be a CLI command?    common cases?        padded?              already encodes?
        │                   │                    │                     │
        ▼                   ▼                    ▼                     ▼
 → Tool Suggestion   → Tool Improvement   → Skill Wording      → Skill Wording
   (§5.1) or Action    (§5.3)               Improvement (§5.4,   Improvement (§5.4,
   Candidate (§5.2)                          efficiency axis)     conditioning axis)
```

**Priority ordering:**
1. **Action candidates** — highest impact when a stable one-command operation already exists
2. **Tool suggestions** — highest impact when the CLI/tool capability does not exist yet
3. **Tool improvements** — medium impact (reduces friction in existing tools)
4. **Retirement opportunities** — remove/collapse superseded prose after Actions or tool improvements
5. **Skill wording (both axes)** — essential but lower impact (prevents misunderstanding and drift)

For CLI-operational skills, scan `Troubleshooting & Edge Cases` first and classify each entry (promote to CLI output contract / promote to tool capability / keep as rare manual playbook) before proposing wording-only edits.

---

### **7. Anti-Patterns (What NOT to Suggest)**

#### **The Situational Flag**
```
❌ "--check-login-flow" flag for browser-automation-studio
   Only useful for login testing—not generalizable

✅ "--assert-mode" flag with options [exists, visible, text_contains]
   Useful for ANY assertion in ANY workflow
```

#### **The Implementation Detail Addition**
```
❌ "Add a section explaining how the workflow engine parses nodes"
   Agent doesn't need to know internal parsing

✅ "Add a CLI command that returns the exact JSON structure required"
   Agent needs to know WHAT to write, not HOW it's processed—and a CLI
   command eliminates a source of drift
```

#### **The Kitchen Sink Tool**
```
❌ Adding 15 flags to one command
   Becomes harder to use than the manual process

✅ One tool, one job; compose multiple simple tools
   Each tool is learnable and predictable
```

#### **The Duplicate Suggestion**
```
❌ Suggesting a tool that already exists

✅ Check first: search for existing scenarios and their CLI commands
   Build on what exists rather than duplicating
```

#### **The Vague Improvement**
```
❌ "The skill should be clearer"
   Not actionable—clearer how?

✅ "Section 3 says 'use the pattern' but doesn't show what pattern.
    Add this example: [specific code/command]"
   Specific, implementable
```

#### **The Decorated Rule Pile**
```
❌ "Add a note that these rules follow the principle of least surprise"
   Name-and-keep: the concept name conditions nothing beside the rules

✅ "Replace the 8 phrasing rules in §2 with 'write requirements in EARS
    form' and delete the rules EARS already encodes"
   Name-and-delete: the name replaces the rules
```

---

### **8. Evaluation Criteria by Skill Category**

Category-specific evaluation focus is owned by the matching authoring guide — `skill-authoring` for Steer; `skill-authoring-tools`, `-search`, `-practice`, `-platform`, `-meta` otherwise. Read the guide for {{SKILL}}'s category and evaluate against its rules; do not maintain a separate criteria list here.

---

### **9. Output Expectations**

**You must:**
- Read {{SKILL}} completely before making suggestions
- If {{SKILL}} includes CLI guidance/workflows, read `cli-steer` before proposing CLI-related suggestions
- Audit both axes: walk the §3 (efficiency) and §4 (conditioning) tables row by row
- Apply the generality test to all tool improvement suggestions
- Use the suggestion formats provided
- Explain reasoning—not just what to change, but why it matters
- Check for existing tools/guidance before suggesting new ones
- For CLI-operational skills, start from troubleshooting promotion opportunities before wording-only edits
- When suggesting a promoted CLI/tool fix, include a brief adopt/defer decision and why
- Include explicit prose retirement recommendations (`Keep/Collapse/Delete`, per `docs/agent-system/PROMOTION_LADDER.md` §"Output requirement for meta analyses") tied to each adopted promotion

**You may:**
- Suggest new CLI tools with clear capability specifications
- Suggest Action candidates with exact CLI targets and validation evidence
- Suggest general-purpose improvements to existing tools
- Suggest specific wording/structure changes to skills on either axis
- Prioritize suggestions by expected impact

**You must NOT:**
- Suggest situational flags that only benefit one use case
- Include implementation details in skill improvement suggestions
- Make vague suggestions without specific examples
- Suggest creating new skills (use skill-authoring-* guides instead)
- Cite a named standard you have not verified is real and widely documented

---

### **10. Report Format**

When analyzing {{SKILL}}, produce this structured report:

```markdown
# Improvement Suggestions for {{SKILL}}

## Summary
[2-3 sentences: Overall quality assessment, primary improvement opportunity, expected impact]

## Tool Suggestions
[If none: "No new tool suggestions—existing tools cover the capabilities needed."]

### [Tool Name]
[Use TOOL SUGGESTION format from §5.1]

## Action Candidates
[If none: "No Action candidates—no stable one-command operations are learned through prose."]

### [Action ID]
[Use ACTION CANDIDATE format from §5.2]

## Tool Improvements
[If none: "Existing tools used by this skill work well for their purpose."]

### [Tool Name] - [Improvement Name]
[Use TOOL IMPROVEMENT format from §5.3]

## Skill Wording/Structure Improvements
[There's almost always something here; tag each with its axis and defect row]

### [efficiency|conditioning] [E#/C#]: [Issue Name]
[Use SKILL WORDING IMPROVEMENT format from §5.4]

## Priority Ranking
1. [Highest impact improvement] — Why: [explanation of expected value]
2. [Second highest] — Why: [explanation]
3. [Third] — Why: [explanation]

## Promotion Decisions
- Adopt now: [promotions that should be implemented immediately]
- Defer: [promotions worth tracking later, with rationale]
- Keep as manual playbook: [items that should remain in troubleshooting]

## Prose Retirement Map (CLI-Operational Skills)
[Canonical table shape: `docs/agent-system/PROMOTION_LADDER.md` §"Output requirement for meta analyses"]

## Implementation Notes
[Any dependencies, ordering considerations, or caveats]
```
