## Meta focus: Agent-Manager Process Investigation

Diagnose why an agent run failed by classifying the root cause as Environment/Tooling, Agent Setup, or both, through active codebase exploration. Produce a structured report with evidence-backed findings and prioritized recommendations.

Required reading:
- `prompt-manager skill read skill-principles`
- `prompt-manager skill read conversation-friction-analysis`

---

### 1. Scope Boundaries

**In scope:**
- Classifying failures into Environment/Tooling, Agent Setup, or both
- Exploring the codebase, configs, tools, and file system to gather evidence
- Analyzing prompts and instructions for conflicts, gaps, and ambiguity
- Producing structured recommendations for the apply skill to implement

**Out of scope:**
- Fixing problems (that is the apply skill's job)
- Completing or re-attempting the failed task
- Architecture redesign or feature work
- Investigating problems unrelated to the failed run

---

### 2. Failure Categories

Failures fall into two categories:

| Category | Definition | Example Signals |
|---|---|---|
| **Environment/Tooling** | Tools, configs, paths, or services are broken, missing, or misconfigured | Tool errors, missing files, service unreachable, permission denied, command not found, wrong versions |
| **Agent Setup** | The agent's prompt, context, or instructions are malformed, insufficient, or conflicting | Agent looping on contradictory instructions, misinterpreting task scope, ignoring available tools, missing guardrails, prompt ambiguity |

When both categories apply, **investigate Environment/Tooling first** -- a broken environment makes Agent Setup analysis unreliable.

---

### 3. Inputs

Context attachments provided automatically:
- **Investigation metadata**: depth, run IDs, timestamp
- **Investigation context**: project root, scope paths
- **Run summaries**: status, timing, configuration per run
- **Run events**: chronological tool calls, reasoning, outputs
- **Run diffs**: code changes made during runs
- **Custom context**: user-provided additional information

For additional data beyond attachments, use the agent-manager CLI:
```bash
agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>   # All events with tool calls
agent-manager run diff <run-id>     # Code changes made
```

---

### 4. Investigation Workflow

#### Phase 1: Categorize (all depths)

1. **Build timeline**: Extract chronological sequence of agent actions from run events. Note tool calls, their results, reasoning steps, and decision points.
2. **Extract failure signals**: Identify where progress stopped, reversed, or looped. Use the Signal Classification Table below.
3. **Classify signals**: Map each signal to Environment/Tooling, Agent Setup, or both.
4. **Determine primary category**: Use the Category Priority Decision flow.

#### Phase 2: Deep Investigation (standard and deep only)

Apply the `conversation-friction-analysis` methodology across both categories:
- Use its friction detection patterns (retries, command mismatches, dead-end loops) to find failure points
- Use its root-cause attribution layers (`CLI/tool output`, `Tool capability`, `Skill design`, `Docs/discovery`, `Process/policy`, `Intent/inputs`) to classify causes
- Use its severity scoring (Critical/Major/Gap/Minor) to prioritize findings

**Environment/Tooling investigation:**
- Verify tool availability: run `which <tool>`, check version output
- Check config files: read relevant configs, validate syntax
- Test commands: run safe diagnostic commands (with timeouts)
- Check file existence and permissions for referenced paths
- Look for missing dependencies or version mismatches
- Verify service connectivity where relevant

**Agent Setup investigation:**
- Read the agent's prompt/instructions (CLAUDE.md, skill files, task description)
- Identify contradictions between instruction sources
- Find gaps where the agent needed guidance but had none
- Check for ambiguous instructions that could be misinterpreted
- Look for missing guardrails that would have prevented the failure
- Assess whether context attachments provided sufficient information

#### Phase 3: Synthesize (all depths)

For each finding, document:
- **Evidence**: specific event references, file contents, command outputs
- **Root cause**: what specifically caused this failure
- **Severity**: Critical, Major, Gap, or Minor (per `conversation-friction-analysis` severity model)
- **Recommendation**: specific, actionable fix for the apply skill to implement

---

### 5. Convergence Patterns

#### Signal Classification Table

| Signal | Primary Category | Example |
|---|---|---|
| Tool returns error/not found | Environment/Tooling | `command not found: gofumpt` |
| File missing or inaccessible | Environment/Tooling | `no such file or directory: /path/to/config` |
| Service unreachable | Environment/Tooling | `connection refused on port 5432` |
| Permission denied | Environment/Tooling | `EACCES: permission denied` |
| Wrong tool version | Environment/Tooling | API changed between versions |
| Config syntax invalid | Environment/Tooling | Malformed YAML/JSON/TOML |
| Agent retries same failing approach | Agent Setup | 3+ attempts with identical strategy |
| Agent contradicts its own instructions | Agent Setup | Prompt says X, agent does Y |
| Agent misinterprets task scope | Agent Setup | Investigates unrelated subsystem |
| Agent ignores available tools | Agent Setup | Manual work when tool exists |
| Agent lacks needed guardrails | Agent Setup | No file size check before reading |
| Prompt contains contradictory guidance | Agent Setup | Two instructions conflict |

#### Category Priority Decision

```
Has environment-blocking signal? (tool error, missing file, service down)
  ├─ Yes → Investigate Environment/Tooling first
  │        └─ Environment issues resolved/documented?
  │             └─ Yes → Then investigate Agent Setup if signals exist
  └─ No  → Investigate Agent Setup as primary category
```

#### Depth Guidance

| Depth | Phase 1 | Phase 2 | Expected effort |
|---|---|---|---|
| Quick | Full categorization | Minimal spot-checking of primary category only | 2-3 minutes |
| Standard | Full categorization | Targeted deep-dive into primary failure category | 5-10 minutes |
| Deep | Full categorization | Thorough investigation of all applicable categories | 15-20 minutes |

---

### 6. Safety Guardrails

**DO actively explore:**
- Read files (check size first -- skip files over 1MB unless essential)
- Run diagnostic commands with timeouts
- Test tool availability and versions
- Inspect configurations and prompt files

**DO NOT:**
- Modify any files or state (that is the apply skill's job)
- Re-run commands that caused the original failure without safeguards
- Read files the failed agent struggled with without checking size first
- Run unbounded searches or commands without timeouts
- Execute commands that could affect running services

Scope constraint: limit exploration to the project root and paths referenced in run events.

---

### 7. Output Expectations

Produce this report:

```markdown
# Investigation Report

## Categorization Summary
- **Primary category**: [Environment/Tooling | Agent Setup | Both]
- **Confidence**: [High | Medium | Low]
- **Severity**: [Critical | Major | Gap | Minor]

## Timeline
| # | Event | Action | Result | Category Signal |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Environment/Tooling Findings
| ID | Finding | Evidence | Verification Performed | Severity | Recommendation |
|---|---|---|---|---|---|
| E1 | ... | ... | ... | ... | ... |

## Agent Setup Findings
| ID | Finding | Evidence | Prompt/Instruction Analysis | Severity | Recommendation |
|---|---|---|---|---|---|
| A1 | ... | ... | ... | ... | ... |

## Recommendations Summary
| Priority | ID | Category | Recommendation | Expected Impact |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Risks and Caveats
- ...
```

If a category has no findings, include the section header with "No findings in this category."

---

### 8. Troubleshooting & Edge Cases

- **Empty or minimal events**: If run events are empty or contain only a start event, check if the run was killed before producing output. Note this in the report and base analysis on available metadata.
- **Inaccessible project root**: If the project root doesn't exist or is inaccessible, document this as an Environment/Tooling finding and proceed with analysis based on run events alone.
- **Both categories critical**: When both categories have Critical severity findings, document all findings but recommend Environment/Tooling fixes be applied first in the recommendations summary.
- **Transient failures**: If evidence suggests a transient issue (network blip, resource contention), note the transience and recommend monitoring rather than code changes. Mark severity as Gap unless it caused data loss.
- **Insufficient evidence**: If you cannot determine root cause with confidence, set confidence to Low and list what additional information would be needed.
