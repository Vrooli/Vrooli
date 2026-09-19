---
name: "skill-authoring-tools"
description: "Authoring guide for Tools skills covering safe usage, guardrails, and verification."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta","tools"]
  tags: ["skill","authoring"]
  icon: "wrench"
  status: "active"
  revision: 45
  createdAt: "2026-01-28T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
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

Use concise work tables to avoid inconsistent tool usage.

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
- Before concluding no CLI exists for an operation, apply `docs/agent-system/PROMOTION_LADDER.md` §"Conversion procedure" step 2 (`cli-health search query "<operation>"` finds an existing owner). Whether a no-hit operation stays in prose or becomes a promotion candidate is decided by the three-part test in `docs/agent-system/PROMOTION_LADDER.md` §"When to attempt CLI/Action conversion", not by a default assumption.
- If multiple entries share the same root cause, add a short promotion note (what should move to CLI/tooling, or why it should remain manual).
- Keep this as guidance, not a rigid section template.
- Apply the canonical lifecycle from `docs/agent-system/PROMOTION_LADDER.md`.

---

### **7. Usage Skills for Scenarios: Decision Trees, Rungs, Learning Spine, Program Steps**

A scenario's **usage** role (canon: `docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets") is a tools skill with four additions. Apply them when the skill is declared as a scenario's usage role; plain tools skills may borrow them.

**7.1 The body is a decision tree.** Open with the task the agent has, branch on observable conditions, and end every branch in a leaf that is one step. Prefer the tree form from canon §"Convergence patterns"; use a work table when the branches are flat.

**7.2 Every leaf carries a rung label.** Write `[S0]` to `[S4]` at the end of the leaf. What each rung means, and what a leaf at that rung may name, is the table in `docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" › "Step rungs"; do not restate it. Do not label a leaf for a program or Action that does not exist.

**7.3 The learning spine, when a scope is declared.** If `metadata.learning.scope` is set, the skill has a "Before acting" step (recall) and an "After acting, always" step (capture) around the tree, with the entry kinds named. Memory mechanics are `vrooli-memory`'s; cite `prompt-manager skill read vrooli-memory`, do not restate scopes, pins, or rules. Curation leaves (pin after the third confirmation, supersede when advice fails, propose a rule after a repeated pattern) sit inside the tree at the branch where the evidence appears.

**7.4 In-use settings table.** One table of `symptom → setting move` for changes the agent may make without a diff (a session profile refresh, a wait strategy, a retention flag, a role or profile choice). Each row names the command and says what to journal.

**7.5 Promotion pass before publishing.** For each `[S1]` leaf whose command returns a deterministic pass/fail with next steps, decide: keep as a leaf, or cite the Action that wraps it. Reference content already carried by `--help` or a CLI contract is deleted, not summarized (`PROMOTION_LADDER.md`).

Frontmatter for a scenario usage skill carries the full Vrooli block and, when learning applies:

```yaml
metadata:
  learning:
    scope: "<scenario>-usage"
    capture: "every attempt"
```

---

### **8. Output Expectations**

You may update:
- Tools skills to improve clarity, safety, or troubleshooting coverage
- `skill.json` entries for Tools skills

You must:
- For a scenario usage role: a decision tree with rung-labeled leaves, the learning spine when a scope is declared, and an in-use settings table (§7)
- Provide safe defaults and guardrails
- Include verification steps
- Keep guidance tool-focused, not feature-focused
- Prefer human-first CLI output patterns and avoid parser-dependent workflows by default
- Keep long-tail failures centralized under `Troubleshooting & Edge Cases` when applicable
- For recurring troubleshooting patterns, include at least one CLI/tool promotion consideration or a brief rationale for not promoting yet
- For deterministic one-command operations, include an Action promotion consideration or a brief rationale for keeping the operation in prose

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "tools"` and a description that names the tool and the primary value it provides.
