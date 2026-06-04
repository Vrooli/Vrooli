## Meta focus: Skill Improvement Suggestions

Analyze **{{SKILL}}** and propose meaningful improvements to its tools, wording, or structure. This skill teaches the reasoning behind good suggestions—not just what to look for, but how to think about skill quality and why it matters.

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

---

### **2. Category Scope**

**In scope:**
- Suggesting new CLI tools that automate manual patterns
- Proposing improvements to existing tools (general-purpose only)
- Identifying Action candidates when one Vrooli-controlled CLI command can own deterministic execution
- Improving skill wording, structure, and clarity
- Identifying where research could be automated

**Out of scope:**
- Implementing the improvements (focus is on suggestions)
- Scenario-specific feature requests (belongs in PRD or issues)
- Creating new skills (see skill-authoring-* guides)
- Tool usage instructions (belongs in Tools skills)

Lifecycle rule:
- Follow the canonical lifecycle in `docs/agent-system/PROMOTION_LADDER.md`.
- Do not stop at promotion suggestions; explicitly identify what skill prose can be collapsed or deleted after promotion.

---

### **3. What Makes a Skill Inefficient**

#### **3.1 Complex/Multi-Step Processes**

When a skill requires multiple sequential steps, each step is a potential failure point. When those steps involve creating temporary artifacts, the agent must track state across operations.

**Real example: browser-automation-studio (before improvement)**

```
Old process for a simple smoke test:
1. Create a temporary workflow JSON file at /tmp/smoke-test.json
2. Write the proper workflow structure with nodes, edges, metadata
3. Execute: browser-automation-studio workflow execute --from-file /tmp/smoke-test.json --wait
4. Export results: browser-automation-studio workflow export <execution-id> --output /tmp/results
5. Navigate to results folder and find the relevant files
6. Consult the skill's "Understanding Output" section to interpret the tree diagram
7. Clean up temporary files
```

**What went wrong:** 7 steps for a smoke test. The agent must create files, remember execution IDs, navigate directories, and cross-reference documentation. High cognitive load, high failure rate.

**The improvement (what we actually did):**

```
New process:
1. Execute with inline steps:
   browser-automation-studio workflow execute \
     --step navigate "http://localhost:3000" waitUntil=networkidle \
     --step screenshot fullPage=true \
     --output /tmp/results \
     --wait

2. Read the auto-generated README.md at /tmp/results/README.md
```

**Why this is better:**
- `--step` flag: Eliminates temporary file creation—define workflows inline
- `--output` flag: Eliminates separate export command—direct path specification
- Auto-generated README.md: Eliminates "consult the skill" step—self-documenting output

**Key insight:** We didn't add a `--smoke-test` command. That would be too situational (only useful for smoke tests). Instead, we added **general-purpose capabilities** that happen to make smoke tests trivial, but also benefit debugging, CI, demos, and countless other use cases.

#### **3.2 Unintuitive Steps**

When steps don't follow naturally from the previous step, agents hesitate, re-read, and sometimes do the wrong thing.

**Signs of unintuitive steps:**
- The skill says "now do X" without explaining why X is necessary
- The next action requires context that wasn't set up
- The skill assumes knowledge that wasn't established

**Example pattern to fix:**
```
❌ "After creating the workflow, check the registry for conflicts."
   What registry? What conflicts? Why would there be conflicts?

✅ "After creating the workflow, verify no duplicate workflow IDs exist:
    browser-automation-studio workflow list | grep your-workflow-name
   Duplicates cause silent overwriting—the newer workflow replaces the older one."
```

#### **3.3 Unclear/Unexplained Concepts**

Skills often reference concepts without explaining what they are or why they matter. The agent is expected to "just know"—but agents don't have persistent memory of past sessions.

**The fresh agent test:** Could an agent that has never seen this skill, this scenario, or this codebase follow it correctly on the first try?

**Example from browser-automation-studio (before improvement):**

The original skill included sections like:
- "The workflow engine uses a DAG-based execution model with node dependencies..."
- "Internally, the timeline is stored as a protobuf message with..."
- "The selector registry implements a namespace-based lookup system..."

**Why this was wrong:** None of this matters for *using* browser-automation-studio. The agent needs to know: What can I do? How do I do it? What should I check?

**The fix:** Remove implementation details. Focus on capabilities and usage.

#### **3.4 Research That Could Be Automated**

Skills often instruct agents to "check" or "find" things that could be provided directly by a CLI tool.

**Real example: PROBLEMS.md management**

Current state (inefficient):
```
Many skills say:
- "Read the scenario's PROBLEMS.md to understand existing issues"
- "If you encounter a new problem, add it to PROBLEMS.md"
- "Check if this problem is already documented"
```

**Why this is inefficient:**
- Reading a growing PROBLEMS.md file burns tokens proportional to file size
- Searching manually for "is this problem documented?" is unreliable
- Writing to PROBLEMS.md has no structure enforcement
- Old problems accumulate, making the file longer and longer

**Proposed tool solution (knowledge-observatory CLI):**
```bash
# Structured read—returns relevant problems only
knowledge-observatory problems search "database connection timeout" --scenario landing-page-business-suite

# Structured write—enforces schema, adds metadata
knowledge-observatory problems add --scenario landing-page-business-suite \
  --title "API rate limiting not enforced" \
  --category security \
  --severity medium \
  --description "..."
```

**Why this is better:**  
- **Reduces search space:** Agent asks for relevant problems, not the whole file  
- **Enforces standards:** Schema validation, required fields, categories  

**This opens the door for adding enhanced capabilities later, such as:**  
- **Cleanup:** Automatic archiving or manual resetting of old problems  
- **Monitoring:** Track problem frequency, resolution time, patterns
- Additional tool enhancements - all without needing to update skills
- Extending this pattern to other standardized internal docs

**Key insight:** Every time a skill says "search for X" or "check if Y", ask: Could a tool do this better? If yes, that's a tool suggestion.

#### **3.5 Anything Long or Confusing**

Length itself is a cost. Every token the agent reads is a token it pays for. Every paragraph of explanation is a place where the agent might get confused.

**The economics:**
- A 5000-token skill used 100 times = 500,000 input tokens
- A 2000-token skill doing the same job = 200,000 input tokens
- Savings: 300,000 tokens = ~$3-9 depending on model (and that's just one skill)

**The reliability angle:** Confusion breeds errors. When a skill is unclear:
- Agents make assumptions
- Assumptions are sometimes wrong
- Wrong assumptions lead to wasted runs
- Wasted runs cost tokens + time + sometimes require human intervention

**The goal:** Maximum clarity in minimum words. Every sentence should earn its place.

#### **3.6 Duplicating Tool Output**

Skills often explain things the tool already tells users—startup instructions, error recovery, or usage hints that appear in CLI output.

**Real example: scenario-to-desktop (before improvement)**

The skill included a section on how to start the service:
```
3. **Start scenario-to-desktop**:
   cd scenarios/scenario-to-desktop && make start
   scenario-to-desktop status  # Verify running
```

But when the CLI is run while stopped, it already shows:
```
Error: scenario-to-desktop API is not reachable.

To auto-start the scenario:
  scenario-to-desktop --auto-start status

Or start manually:
  vrooli scenario start scenario-to-desktop
```

**The fix:** Trust tool error messages. Don't duplicate them in skills.

**Key insight:** If a tool provides actionable guidance on error/startup, the skill doesn't need to repeat it. This reduces skill size and eliminates a source of drift.

#### **3.7 Verbose Command References**

Skills that list every command and flag create two problems: (1) high token cost, and (2) drift when the CLI changes but the skill doesn't.

**Real example: scenario-to-desktop (before improvement)**

The skill had ~380 lines documenting 40+ commands with all their flags—essentially duplicating `--help` output.

**The fix:** Restructure the CLI with subcommand groups that have their own help:
```bash
# Before: skill documents every command
scenario-to-desktop pipeline-run <scenario> [--stages ...] [--platforms ...] [--wait]
scenario-to-desktop pipeline-status <id> [--verbose] [--json]
scenario-to-desktop telemetry-ingest <scenario> --file <path>
# ... 37 more commands with flags

# After: skill says "run help for details"
scenario-to-desktop pipeline help
scenario-to-desktop telemetry help
```

**Result:** Skill shrinks from ~540 to ~160 lines. Command documentation lives in the tool (always accurate, no drift).

**Key insight:** If a skill lists >5 commands with flags, consider whether the CLI should support `<group> help` subcommands. This is a tool improvement that enables a simpler skill.

#### **3.8 Async Patterns When Sync Works**

Async workflows require explaining polling loops, status checking, and sleep patterns. If the tool supports blocking mode, the skill can be much simpler.

**Real example: scenario-to-desktop (before improvement)**

The skill explained async pipeline execution:
```bash
# Start async (returns immediately)
scenario-to-desktop pipeline-run my-scenario --platforms linux

# Poll for completion
while true; do
  scenario-to-desktop pipeline-status $ID
  sleep 10
done
```

**The fix:** If the tool supports `--wait`, make that the default recommendation:
```bash
scenario-to-desktop pipeline run my-scenario --platforms linux --wait
```

**Result:** One line instead of a polling loop. The skill becomes a reference card, not a tutorial.

**Key insight:** When you see a skill explaining polling/status-checking patterns, ask: Could the tool support `--wait` instead? That's a tool improvement suggestion.

#### **3.9 Long-Tail Accretion in Troubleshooting**

For skills with operational CLI workflows, treat `Troubleshooting & Edge Cases` as the deliberate long-tail zone.

What to look for:
- Repeated entries with similar causes/fixes
- “Run X then Y then inspect Z” patterns that could be one command
- Manual interpretation steps that should be emitted as CLI next actions

Improvement direction:
- Promote repeated troubleshooting prose into CLI output contracts or tool capabilities
- Keep core workflow concise; keep only genuinely rare/manual cases in troubleshooting
- When tooling improves, recommend deleting superseded troubleshooting prose

---

### **4. Suggestion Categories + Retirement Pass**

Pre-pass for CLI-operational skills:
1. Scan `Troubleshooting & Edge Cases` first.
2. Mark each item as one of:
   - `Promote to CLI output contract`
   - `Promote to tool capability`
   - `Keep as rare/manual playbook item`
3. Use these marks to prioritize tool/tooling suggestions before wording edits.

Handoff note:
- If a prior `skill-validation` report exists, read its troubleshooting-related findings first and reuse that signal before adding new prose suggestions.

#### **4.1 Tool Suggestions (New CLI capabilities)**

**When to suggest:** You observe repeated manual patterns that could be automated.

**Identification signals:**
- Skill tells agent to read/write the same file types repeatedly
- Skill requires multi-step process that could be one command
- Information is scattered and must be manually gathered
- Verification requires manual inspection instead of pass/fail
- Skill documents one stable command that agents should discover and validate directly

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
- [Any existing scenarios we could potentially use directly or enhance. If none, explaining so and suggesting a new scenario]

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

#### **4.1a Action Candidates (Expose Existing CLI Commands)**

**When to suggest:** A Vrooli-controlled CLI command already performs one deterministic operation, but agents must learn it through prose.

Check first:
- Run `prompt-manager discover "<operation>" --type all` to find existing Actions. `--type all` is *best-match relevance* (skills and actions ranked purely by score, no curated topic packs), so phrase the query as the operation / what you need — not as a planning topic.
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

#### **4.2 Tool Improvements (Enhance existing CLI tools)**

**When to suggest:** An existing tool works but is awkward for common cases.

**The generality test:** Would this improvement help in:
- Multiple scenarios?
- Multiple skill types (Steer, Search, Tools)?
- Multiple use cases beyond the immediate one?

If YES to 2+, it's a good suggestion. If only 1, it's too situational.

**Good vs. Bad examples:**

| Tool | ❌ Bad | Why Bad | ✅ Good | Why Good |
|------|--------|---------|---------|----------|
| browser-automation-studio | `--smoke-test` flag | Only helps smoke tests | `--step` for inline definition | Helps any quick workflow |
| browser-automation-studio | `--login-first` | Assumes one auth pattern | `--output` for direct export | Works for any workflow |
| prompt-manager | `--validate-progress-skill` | One skill only | `--validate` for any skill | Works for all skills |

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

#### **4.3 Skill Wording/Structure Improvements**

**When to suggest:** The skill text is confusing, incomplete, or contains information that doesn't help the agent.

**The fresh agent test:** Could someone who has never seen this skill, scenario, or codebase follow it correctly?

**Common issues to look for:**

| Issue | Symptom | Fix |
|-------|---------|-----|
| Missing context | "Use the registry" (what registry?) | Add path, example, explanation |
| Assumed knowledge | "Follow the pattern" (what pattern?) | Include inline example |
| Implementation details | "Uses a DAG internally..." | Delete—focus on WHAT, not HOW |
| Vague guidance | "Consider the trade-offs" | Provide decision table |
| Missing scope | No boundaries defined | Add In scope / Out of scope |
| Multi-step without flow | Steps that don't connect | Add transitions, explain why each step |

**Structure checklist (every skill should have):**
- [ ] Intent statement (1-2 sentences)
- [ ] Scope boundaries (In scope / Out of scope)
- [ ] Convergence patterns (decision trees/tables where consistency matters)
- [ ] Output expectations (What you may/must/must not change)
- [ ] Self-contained examples (not just "see X" references)

**Suggestion format:**
```
SKILL WORDING IMPROVEMENT: {{SKILL}}

Current issue:
- [Quote the problematic text]
- [Explain why it's confusing or unhelpful]

Proposed revision:
[New text—complete enough to copy-paste]

Why this helps:
- Fresh agent understanding: [How it improves first-time comprehension]
- Token efficiency: [If applicable—how it reduces length]
- Reliability: [How it reduces chance of misinterpretation]
```

---

### **5. Convergence Patterns**

**Which suggestion type applies?**

```
                    What did you observe in {{SKILL}}?
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            ▼                       ▼                       ▼
     Repeated manual          Tool works but          Skill text is
     pattern that could       awkward for             confusing or
     be a CLI command?        common cases?           incomplete?
            │                       │                       │
            ▼                       ▼                       ▼
     → Tool Suggestion       → Tool Improvement      → Skill Wording
                                                       Improvement
```

**Priority ordering:**
1. **Action candidates** — Highest impact when a stable one-command operation already exists
2. **Tool suggestions** — Highest impact when the CLI/tool capability does not exist yet
3. **Tool improvements** — Medium impact (reduces friction in existing tools)
4. **Retirement opportunities** — Remove/collapse superseded prose after Actions or tool improvements
5. **Skill wording** — Essential but lower impact (prevents misunderstanding)

---

### **6. Anti-Patterns (What NOT to Suggest)**

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

✅ "Add an CLI command that returns the exact JSON structure required"
   Agent needs to know WHAT to write, not HOW it's processed. And using a CLI command for this eliminates a source of potential drift
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

✅ Check first: search for existing scenarios that might be relevant, and seeing what CLI commands they have available
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

---

### **7. Evaluation Criteria by Skill Category**

Different skill categories need different evaluation focus:

| Category | Primary Focus | Key Questions |
|----------|---------------|---------------|
| **Steer** | Convergence patterns | Are decision trees present? Can agents reach consistent decisions? |
| **Tools** | Safe usage paths | Are guardrails explicit? Is verification included? |
| **Search** | Output contracts | Is output format defined? Are stop conditions clear? |
| **Meta** | Governance clarity | Are boundaries explicit? Does this duplicate existing guidance? |

---

### **8. Output Expectations**

**You must:**
- Read {{SKILL}} completely before making suggestions
- If {{SKILL}} includes CLI guidance/workflows, read `cli-steer` before proposing CLI-related suggestions
- Apply the generality test to all tool improvement suggestions
- Use the suggestion formats provided
- Explain reasoning—not just what to change, but why it matters
- Check for existing tools/guidance before suggesting new ones
- For CLI-operational skills, start from troubleshooting promotion opportunities before wording-only edits
- When suggesting a promoted CLI/tool fix, include a brief adopt/defer decision and why
- Include explicit prose retirement recommendations (`Keep/Collapse/Delete`) tied to each adopted promotion

**You may:**
- Suggest new CLI tools with clear capability specifications
- Suggest Action candidates with exact CLI targets and validation evidence
- Suggest general-purpose improvements to existing tools
- Suggest specific wording/structure changes to skills
- Prioritize suggestions by expected impact

**You must NOT:**
- Suggest situational flags that only benefit one use case
- Include implementation details in skill improvement suggestions
- Make vague suggestions without specific examples
- Suggest creating new skills (use skill-authoring-* guides instead)

---

### **9. Report Format**

When analyzing {{SKILL}}, produce this structured report:

```markdown
# Improvement Suggestions for {{SKILL}}

## Summary
[2-3 sentences: Overall quality assessment, primary improvement opportunity, expected impact]

## Tool Suggestions
[If none: "No new tool suggestions—existing tools cover the capabilities needed."]

### [Tool Name]
[Use TOOL SUGGESTION format from Section 4.1]

## Tool Improvements
[If none: "Existing tools used by this skill work well for their purpose."]

### [Tool Name] - [Improvement Name]
[Use TOOL IMPROVEMENT format from Section 4.2]

## Skill Wording/Structure Improvements
[There's almost always something here]

### [Issue Name]
[Use SKILL WORDING IMPROVEMENT format from Section 4.3]

## Priority Ranking
1. [Highest impact improvement] — Why: [explanation of expected value]
2. [Second highest] — Why: [explanation]
3. [Third] — Why: [explanation]

## Implementation Notes
[Any dependencies, ordering considerations, or caveats]

## Promotion Decisions
- Adopt now: [promotions that should be implemented immediately]
- Defer: [promotions worth tracking later, with rationale]
- Keep as manual playbook: [items that should remain in troubleshooting]

## Troubleshooting Promotion Map (CLI-Operational Skills)
| Troubleshooting Item | Action | Target |
|---|---|---|
| ... | Promote to CLI output contract / Promote to tool capability / Keep in playbook | ... |

## Prose Retirement Map (CLI-Operational Skills)
| Skill Instruction / Gate | Decision (Keep/Collapse/Delete) | Trigger | Notes |
|---|---|---|---|
| ... | ... | CLI output contract available / not available | ... |

## Complexity Budget Impact
- Estimated gate/step count change: [before -> after]
- Estimated troubleshooting prose change: [before -> after]
- Net outcome: [reduced / neutral / increased] with rationale
```

---

### **10. Remember**

Prefer durable leverage over wording churn: improve CLI/tool contracts, then retire superseded prose.
