## Meta focus: Skill Principles

Guide for creating and updating skills that keep agent decisions consistent without over-prescription. Applies to every category (Steer, Search, Tools, Practice, Meta).

---

### **1. What Skills Are (and Are Not)**

Skills are reusable, principle-based guidance that create shared mental models across sessions. They are **not** one-off task instructions or file-specific to-do lists.

**Good:**
- "Prefer boundary validation at system edges"
- "Handle all states explicitly: loading, error, empty, success"
- "Generate falsifiable hypotheses before debugging"

**Avoid:**
- "Edit src/components/Button.tsx line 42"
- "Add exactly 5 tests per file"
- "Finish in 30 minutes"

---

### **2. Scope Boundaries (For This Skill)**

**In scope:**
- Skill purpose, categories, and baseline quality standards
- How to reference, register, and evolve skills

**Out of scope:**
- Category-specific authoring rules (see the per-category authoring guides)
- Scenario-specific implementation guidance

---

### **3. Choose the Right Category**

Use the category that matches the skill's primary outcome (this is `modes[0]` in metadata):

| Primary intent | Category (modes[0]) | What it optimizes |
|---|---|---|
| Build or improve scenario behavior | Steer | Architecture, quality, reliability |
| Find or map information | Search | Discovery, coverage, evidence |
| Use a tool/resource/scenario | Tools | Correct operation, safety, efficiency |
| Apply a systematic engineering methodology | Practice | Process rigor, repeatability, knowledge capture |
| Define or govern the skill system | Meta | Skill coherence, policy, lifecycle |

Decision check:
```
Is the skill about how to change the scenario itself?
  -> Steer
Is the skill about finding information or tracing implementation?
  -> Search
Is the skill about using a tool or resource correctly?
  -> Tools
Is the skill about HOW to approach a class of problems systematically?
  -> Practice
Is the skill about how skills should be written or governed?
  -> Meta
```

---

### **4. Universal Quality Bars**

Every skill must include:
- **Clear intent statement** at the top (1-2 sentences)
- **Boundary definition** (what is in scope and out of scope)
- **Convergence patterns** (decision trees/tables/diagrams) when choices must be consistent
- **Output expectations** describing what can/can't/must change or how results must be formatted
- **Human-first CLI consumption** by default: prefer direct CLI output and avoid parser pipelines (`--json`, `--raw`, `jq`) unless default output is too long or ambiguous for reliable execution
- **Selector-first workflows** by default: prefer stable human-readable selectors over extracted opaque IDs when tools support both

For skills with **operational CLI complexity** (multi-step workflows, mutable state, external dependencies, or non-trivial failure modes):
- Include a dedicated section named **`Troubleshooting & Edge Cases`**
- Keep failure matrices, rare gotchas, diagnostics, and manual recovery guidance in that section instead of spreading them across core workflow text
- Keep the main workflow readable and focused on standard execution
- Treat repeated troubleshooting clarifications as a tooling signal: prefer promoting them to CLI output contracts or tool capabilities before adding more prose

For simple/stable skills with no meaningful long-tail behavior:
- The section may be omitted, but state this explicitly (for example: `No known operational edge cases for standard usage.`)

---

### **5. Referencing Other Skills**

When a skill depends on another, reference it explicitly using the prompt-manager CLI pattern.

Required reading:
- `prompt-manager skill read <skill-id>`

Optional reading:
- `prompt-manager skill read <skill-id> <skill-id>`

Only require what is essential; keep optional lists short and relevant.

---

### **6. Promotion-Retirement Lifecycle (Canonical)**

Use one lifecycle for all CLI-operational skills:

1. **Interim prose guardrail**: add minimal skill guidance when tools do not yet provide deterministic output contracts.
2. **Promote to CLI/tool contract**: implement pass/fail signals, next-step guidance, and structured failure hints in the tool.
3. **Retire superseded prose**: remove or collapse skill instructions now covered by tool output contracts.

Retirement criteria:
- The CLI/tool can return a deterministic status for the workflow decision (`pass/fail` or equivalent).
- The CLI/tool output contains actionable next steps for common failures.
- Keeping both tool contract and detailed skill prose would duplicate volatile operational logic.

Retention criteria (do not retire):
- Safety constraints (`must not`, irreversible operations, credential handling).
- Scope boundaries and ownership boundaries.
- Human handoff rules where automation is intentionally impossible.

Output requirement for meta analyses (`skill-validation`, `skill-improvement-suggestions`, `conversation-friction-analysis`):
- Explicitly classify major workflow instructions as `Keep`, `Collapse to CLI contract`, or `Delete`.

---

### **7. Registration and Metadata**

1. Create the skill directory in `scenarios/prompt-manager/store/skills/packs/core/<skill-id>/`
2. Add the following files:
   - `SKILL.md` - skill content
   - `skill.json` - metadata with `id`, `name`, `description`, `modes`, `tags`
3. Run `prompt-manager skill sync` to pick up changes
4. Verify via `prompt-manager skill show <id>`

---

### **8. Avoid Skill Sprawl**

Before creating a new skill:
- Search for existing skills that already cover the concept
- Prefer extending an existing skill when the guidance naturally fits
- Only create a new skill when it introduces a distinct, reusable mental model

---

### **9. Output Expectations**

You may:
- Add or update skill files in the packs directory
- Create new skill directories for genuinely new mental models

You must:
- Preserve principle-based guidance style
- Keep skills transferable across scenarios
- Include scope boundaries and output expectations
- Use convergence patterns when decision consistency matters

---

### **10. Skill Architecture Heuristics**

Use these heuristics when creating or evolving any skill:

- **Optimize for entropy control, not content volume**: most skill failures come from unmanaged clarification growth, not missing facts.
- **Standardize only high-leverage constraints**: avoid rigid global templates; enforce only structures that materially improve execution consistency.
- **Keep the primary path clean**: standard execution should be easy to scan and run without long-tail context switching.
- **Isolate long-tail operations**: for CLI-operational complexity, centralize rare failures and manual recovery in `Troubleshooting & Edge Cases`.
- **Treat repeated prose as a product signal**: if the same workaround appears repeatedly, promote it to CLI output contracts or tool capabilities.
- **Prefer promotion + retirement loops**: when tooling improves, remove superseded prose to prevent one-way growth.
- **Prefer layered fixes**: use skill text as interim guardrails when needed, but prioritize durable CLI/tool improvements for recurring friction.
- **Use trigger-based governance**: apply heavier structure when operational complexity is present, not based on category labels alone.
- **Preserve dual usability**: default human-readable flows should be directly actionable, while machine-readable paths remain deterministic when needed.
- **Track complexity budget drift**: if gates/steps/long-tail prose increase, require explicit rationale and a retirement plan.
