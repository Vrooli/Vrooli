## Meta focus: Meta Skill Authoring

Guide for creating **meta** skills (the authored skill declares `modes[0] = "meta"`). Meta skills govern how the skill system itself evolves, stays coherent, and avoids drift.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md` — universal quality bars, structure, registration, and §"Skills are conditioning signals" (apply the four lenses when reviewing or governing any skill change)

---

### 1. Category Scope

**In scope:**
- Skill system governance and policy
- Authoring standards and shared vocabulary
- Lifecycle, registration, and maintenance rules
- Cross-skill conflict resolution

**Out of scope:**
- Scenario-specific implementation guidance (Steer)
- Tool operation instructions (Tools)
- Discovery workflows (Search)

---

### 2. The meta-specific rule

A meta skill is one governance surface with explicit ownership: name the surface it governs, the decision rules for ambiguous cases, and the boundary where its authority ends. Governance rules that already live in canon (`path:docs/agent-system/`) are cited, never restated — a meta skill that duplicates canon is itself the drift it exists to prevent.

---

### **3. Output Expectations**

You may update meta skills and their `skill.json` entries. You must keep governance rules conflict-free across meta skills, cite canon instead of restating it, and preserve explicit boundaries and ownership.
