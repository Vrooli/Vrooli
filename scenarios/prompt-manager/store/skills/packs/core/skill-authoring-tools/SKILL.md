## Meta focus: Tools Skill Authoring

Guide for creating **Tools** skills (where `modes[0] = "Tools"`). Tools skills teach how to use a resource, scenario, or tool safely and effectively.

Required reading:
- `prompt-manager skill read skill-principles`

---

### **1. Category Scope**

**In scope:**
- How to use a tool or resource correctly
- Prerequisites, setup, and safe defaults
- Common failure modes and recovery steps
- Validation and verification steps

**Out of scope:**
- Scenario feature design or refactors
- Details of tool implementation (focus should be on *using* the tool)

---

### **2. Recommended Structure**

Tools skills should be pragmatic and action-oriented, but not over-templated.

Recommended shape:
1. **Core flow content** - concise usage path, decision guidance, prerequisites, key commands, verification, guardrails
2. **`Troubleshooting & Edge Cases`** - all long-tail operational content in one place

Important:
- Do not force rigid section naming for core flow content.
- Do require the exact section name `Troubleshooting & Edge Cases` when operational complexity exists.

Operational complexity rule:
- If the tool has multi-step workflows, mutable state, external dependencies, or recurring failure patterns, include `Troubleshooting & Edge Cases`.
- If the tool is genuinely simple/stable, you may omit it and add one explicit line such as: `No known operational edge cases for standard usage.`

---

### **3. Convergence Patterns**

Use concise decision tables to avoid inconsistent tool usage.

Example:

| Situation | Preferred tool action | Avoid |
|---|---|---|
| Need UI regression validation | Run BAS workflow | Manual clicking |
| Need API contract confirmation | Use reference docs | Guessing from UI |
| Multiple scenarios running | Use lifecycle commands | Direct process execution |

---

### **4. CLI Output Contract (Human-First)**

For agent workflows, default CLI output is the canonical path.

- Prefer direct commands without `--json` / `--raw`
- Do not add `jq` pipelines in normal workflows
- Prefer selector-based commands over ID extraction when available
- Use machine-readable output only when default output is too long or ambiguous for reliable execution, and state that exception explicitly

---

### **5. Guardrails and Safety**

Tools skills should call out prohibited usage explicitly:
- Do not bypass scenario lifecycle tooling
- Do not run direct binaries when a lifecycle command exists
- Do not mutate shared resources without verification

If a tool has strict do-not-do rules, include them near the top.

---

### **6. Troubleshooting & Edge Cases (Standardized Section)**

When present, this section is the single place for:
- Failure matrices (`symptom -> likely cause -> first check -> fix`)
- Rare edge cases and stop conditions
- Diagnostic command order
- Manual recovery and handoff guidance

Keep this section separate from the primary workflow so the core path stays concise.

Promotion rule:
- If an item in this section is frequent/repetitive, prefer improving CLI output contracts or adding tool capabilities rather than expanding prose further.
- If multiple entries share the same root cause, add a short promotion note (what should move to CLI/tooling, or why it should remain manual).
- Keep this as guidance, not a rigid section template.

---

### **7. Registration Notes**

Follow **Skill Principles** and ensure:
- `modes[0]` is **Tools**
- The description names the tool and the primary value it provides

---

### **8. Output Expectations**

You may update:
- Tools skills to improve clarity, safety, or troubleshooting coverage
- `metadata.json` entries for Tools skills

You must:
- Provide safe defaults and guardrails
- Include verification steps
- Keep guidance tool-focused, not feature-focused
- Prefer human-first CLI output patterns and avoid parser-dependent workflows by default
- Keep long-tail failures centralized under `Troubleshooting & Edge Cases` when applicable
- For recurring troubleshooting patterns, include at least one CLI/tool promotion consideration or a brief rationale for not promoting yet
