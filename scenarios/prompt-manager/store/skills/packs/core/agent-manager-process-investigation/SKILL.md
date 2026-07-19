## Investigation

Diagnose why the attached agent run(s) failed. Classify root causes and recommend specific, actionable fixes. Your final answer is a single machine-read JSON object (contract below) — not a prose report.

---

### Context

The investigation context below already contains the key data — **use it first** before calling any CLI commands:

- **Run overview**: status, timing, task description, runner configuration
- **Event timeline**: chronological tool calls, reasoning, and results
- **Agent setup paths**: file paths to the agent's prompt-manager configuration
- **Historical context**: recent runs with the same agent profile (success/fail patterns)
- **Run diff**: code changes made during the run (if any)

{{.context}}

Only use `agent-manager run events`, `agent-manager run get`, etc. for data **not already included** above (e.g., full untruncated task description, detailed event payloads beyond the timeline summary).

### What to do

1. Read the timeline and overview to understand what happened
2. Identify where things went wrong — errors, loops, wrong approaches, stalls
3. Classify each failure as one or both of:
   - **Environment/Tooling**: tools broken/missing/misconfigured, config errors, services down, wrong versions, permission issues
   - **Agent Setup**: prompt unclear or contradictory, missing guardrails, wrong tools listed, insufficient context, scope confusion
4. If both apply, investigate Environment/Tooling first — a broken environment makes prompt analysis unreliable
5. For each finding: cite specific evidence (event numbers, file contents, command outputs), assess severity, and recommend a concrete fix naming the specific file and change needed

### Exploration

- Read agent prompt/instruction files listed in the agent-setup section
- Run diagnostic commands (`which`, version checks) to verify tools the agent tried to use
- Check configs and files referenced in error messages
- When a command fails, try alternate invocations (without flags, different syntax) before concluding the tool is broken
- Quick depth: categorize from the context alone. Standard depth: verify your hypotheses by reading relevant files and running diagnostic commands. Deep depth: exhaustively explore all applicable categories
- **Do NOT modify any files** — investigation is read-only

### Reference: root-cause attribution and severity

Attribute each finding to a primary layer: CLI/tool output, tool capability, skill design, docs/discovery, process/policy, or intent/inputs. Score recommendations by impact × recurrence − cost and prefer fixes that remove repeated manual interpretation. Severity: **Critical** blocks delivery or risks unsafe action; **Major** causes frequent retries/guessing; **Gap** means a capability is implied but not enabled; **Minor** is a low-risk clarity improvement. "Forces the agent to guess the next action" is at least Major.

### Output format — REQUIRED

Your entire final message MUST be exactly one fenced `json` code block and nothing else (no prose before or after). It must be a single JSON object matching this contract:

```
{
  "summary": "1-3 paragraph narrative: what failed, the primary category, and overall confidence",
  "primaryCategory": "Environment/Tooling" | "Agent Setup" | "Both",
  "confidence": "High" | "Medium" | "Low",
  "categories": [
    {
      "name": "Environment/Tooling",
      "recommendations": [
        {
          "text": "Concrete, actionable fix naming the specific file and change",
          "severity": "Critical" | "Major" | "Gap" | "Minor",
          "evidence": "Event numbers, file contents, or command output supporting this"
        }
      ]
    }
  ]
}
```

Rules:
- `summary`, `primaryCategory`, and `categories` are required; every category needs a `name` and at least one recommendation; every recommendation needs `text`.
- Group recommendations under category names that reflect their root layer (e.g., "Environment/Tooling", "Agent Setup"). Omit a category entirely if it has no findings.
- Each recommendation must be applyable without further investigation — say which file and what change.
- Output valid JSON only. Do not include comments or trailing commas.
