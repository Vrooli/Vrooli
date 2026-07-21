## Meta focus: Meta Skill Authoring

Guide for creating **Meta** skills (where `modes[0] = "Meta"`). Meta skills govern how the skill system itself evolves, stays coherent, and avoids drift.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md` — apply §"Skills are conditioning signals" (focality, interpretive entropy, verifiability, attention economy) when reviewing or governing any skill change

---

### **1. Category Scope**

**In scope:**
- Skill system governance and policy
- Authoring standards and shared vocabulary
- Lifecycle, registration, and maintenance rules
- Cross-skill conflict resolution

**Out of scope:**
- Scenario-specific implementation guidance
- Tool operation instructions (belongs in Tools)
- Discovery workflows (belongs in Search)

---

### **2. Recommended Structure**

Meta skills should be concise, clear, and durable:

1. **Focus statement** - what part of the skill system is being governed
2. **Definitions** - shared terminology or taxonomy
3. **Decision rules** - how to resolve ambiguous cases
4. **Boundaries** - what this skill does not control
5. **Evolution rules** - how to update or extend without sprawl
6. **Output expectations** - allowed changes and constraints

---

### **3. Convergence Patterns**

Use decision trees to prevent inconsistent governance.

Example:
```
Is this guidance already covered by an existing skill?
  -> Update the existing skill
Is it a new, reusable mental model?
  -> Create a new skill
Is it a one-off task or project detail?
  -> Do not create a skill
```

---

### **4. Anti-Drift Rules**

- Do not duplicate rules across multiple Meta skills
- Prefer a single source of truth with clear references
- Keep Meta skills short; they should be easy to audit and update

---

### **5. Registration Notes**

Follow **Skill Principles** and ensure:
- `modes[0]` is **Meta**
- The description states the governance surface clearly

---

### **6. Output Expectations**

You may update:
- Meta skills for clarity, conflict resolution, or lifecycle changes
- `metadata.json` entries for Meta skills

You must:
- Avoid conflicting governance rules
- Keep Meta skills transferable across scenarios
- Preserve explicit boundaries and update rules
