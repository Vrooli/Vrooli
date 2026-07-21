## Meta focus: Tools Skill Authoring

Guide for creating **tools** skills (the authored skill declares `modes[0] = "tools"`). Tools skills teach how to use a resource, scenario, or tool safely and effectively.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `docs/agent-system/PROMOTION_LADDER.md`

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
- Do require the exact section name `Troubleshooting & Edge Cases` when operational complexity exists — the trigger rule and the simple-skill exemption live in `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars".

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

Human-first CLI consumption and selector-first workflows are universal quality bars — `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" owns them. When a tools skill needs machine-readable output, state the exception and its justification explicitly in the skill.

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
- If one Vrooli-controlled CLI command owns a deterministic operation, prefer exposing it as an Action and referencing that Action instead of documenting command prose inline.
- Before concluding no CLI exists for an operation, apply `docs/agent-system/PROMOTION_LADDER.md` §"Conversion procedure" step 2 (`cli-health search "<operation>"` finds an existing owner). Whether a no-hit operation stays in prose or becomes a promotion candidate is decided by the three-part test in `docs/agent-system/PROMOTION_LADDER.md` §"When to attempt CLI/Action conversion", not by a default assumption.
- If multiple entries share the same root cause, add a short promotion note (what should move to CLI/tooling, or why it should remain manual).
- Keep this as guidance, not a rigid section template.
- Apply the canonical lifecycle from `docs/agent-system/PROMOTION_LADDER.md`.

---

### **7. Output Expectations**

You may update:
- Tools skills to improve clarity, safety, or troubleshooting coverage
- `skill.json` entries for Tools skills

You must:
- Provide safe defaults and guardrails
- Include verification steps
- Keep guidance tool-focused, not feature-focused
- Prefer human-first CLI output patterns and avoid parser-dependent workflows by default
- Keep long-tail failures centralized under `Troubleshooting & Edge Cases` when applicable
- For recurring troubleshooting patterns, include at least one CLI/tool promotion consideration or a brief rationale for not promoting yet
- For deterministic one-command operations, include an Action promotion consideration or a brief rationale for keeping the operation in prose

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "tools"` and a description that names the tool and the primary value it provides.
