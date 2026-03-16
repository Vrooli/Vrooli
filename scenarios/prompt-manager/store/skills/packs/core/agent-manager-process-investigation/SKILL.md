## Investigation

Diagnose why the attached agent run(s) failed. Produce a structured report classifying root causes and recommending specific, actionable fixes.

---

### What you have

Context attachments below already contain the key data — **use them first** before calling any CLI commands:
- **Run overview**: status, timing, task description, runner configuration
- **Event timeline**: chronological tool calls, reasoning, and results
- **Agent setup paths**: file paths to the agent's prompt-manager configuration
- **Historical context**: recent runs with the same agent profile (success/fail patterns)
- **Run diff**: code changes made during the run (if any)

Only use `agent-manager run events`, `agent-manager run get`, etc. for data **not already included** in the attachments (e.g., full untruncated task description, detailed event payloads beyond the timeline summary).

The investigation methodology and any reference skills are already included in your system prompt — do not try to fetch skill files from disk.

### What to do

1. Read the attached timeline and overview to understand what happened
2. Identify where things went wrong — errors, loops, wrong approaches, stalls
3. Classify each failure as one or both of:
   - **Environment/Tooling**: tools broken/missing/misconfigured, config errors, services down, wrong versions, permission issues
   - **Agent Setup**: prompt unclear or contradictory, missing guardrails, wrong tools listed, insufficient context, scope confusion
4. If both apply, investigate Environment/Tooling first — a broken environment makes prompt analysis unreliable
5. For each finding: cite specific evidence (event numbers, file contents, command outputs), assess severity, and recommend a concrete fix naming the specific file and change needed

### Exploration

- Read agent prompt/instruction files listed in the agent-setup attachment
- Run diagnostic commands (`which`, version checks) to verify tools the agent tried to use
- Check configs and files referenced in error messages
- When a command fails, try alternate invocations (without flags, different syntax) before concluding the tool is broken
- Quick depth: categorize from the attachments alone. Standard depth: verify your hypotheses by reading relevant files and running diagnostic commands. Deep depth: exhaustively explore all applicable categories
- **Do NOT modify any files** — investigation is read-only

### Output format

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
| ID | Finding | Evidence | Severity | Recommendation |
|---|---|---|---|---|
| E1 | ... | ... | ... | ... |

## Agent Setup Findings
| ID | Finding | Evidence | Severity | Recommendation |
|---|---|---|---|---|
| A1 | ... | ... | ... | ... |

## Recommendations Summary
| Priority | ID | Category | Recommendation | Expected Impact |
|---|---|---|---|---|
| 1 | ... | ... | ... | ... |

## Risks and Caveats
- ...
```

If a category has no findings, include the header with "No findings in this category."

Each recommendation should be actionable without further investigation — if a file needs to change, say which file and what the change is.
