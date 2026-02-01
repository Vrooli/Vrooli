## Meta focus: Tools Skill Authoring

Guide for creating **Tools** skills (where `modes[0] = "Tools"`). Tools skills teach how to use a resource, scenario, or tool safely and effectively.

Required reading:
- `prompt-manager skills read skill-principles`

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

Tools skills should be pragmatic and action-oriented:

1. **Focus statement** - what the tool is for and when to use it
2. **When to use / when not to use** - decision guidance
3. **Prerequisites** - required setup or dependencies
4. **Core commands** - minimal safe set with examples
5. **Guardrails** - what not to do and why
6. **Troubleshooting** - common failure modes and fixes
7. **Verification** - how to confirm success

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

### **4. Guardrails and Safety**

Tools skills should call out prohibited usage explicitly:
- Do not bypass scenario lifecycle tooling
- Do not run direct binaries when a lifecycle command exists
- Do not mutate shared resources without verification

If a tool has strict do-not-do rules, include them near the top.

---

### **5. Registration Notes**

Follow **Skill Principles** and ensure:
- `modes[0]` is **Tools**
- The description names the tool and the primary value it provides

---

### **6. Output Expectations**

You may update:
- Tools skills to improve clarity, safety, or troubleshooting coverage
- `metadata.json` entries for Tools skills

You must:
- Provide safe defaults and guardrails
- Include verification steps
- Keep guidance tool-focused, not feature-focused
