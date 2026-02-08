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

---

### **5. Referencing Other Skills**

When a skill depends on another, reference it explicitly using the prompt-manager CLI pattern.

Required reading:
- `prompt-manager skill read <skill-id>`

Optional reading:
- `prompt-manager skill read <skill-id> <skill-id>`

Only require what is essential; keep optional lists short and relevant.

---

### **6. Registration and Metadata**

1. Create the skill directory in `scenarios/prompt-manager/store/skills/packs/core/<skill-id>/`
2. Add the following files:
   - `SKILL.md` - skill content
   - `skill.json` - metadata with `id`, `name`, `description`, `modes`, `tags`
3. Run `prompt-manager skill sync` to pick up changes
4. Verify via `prompt-manager skill show <id>`

---

### **7. Avoid Skill Sprawl**

Before creating a new skill:
- Search for existing skills that already cover the concept
- Prefer extending an existing skill when the guidance naturally fits
- Only create a new skill when it introduces a distinct, reusable mental model

---

### **8. Output Expectations**

You may:
- Add or update skill files in the packs directory
- Create new skill directories for genuinely new mental models

You must:
- Preserve principle-based guidance style
- Keep skills transferable across scenarios
- Include scope boundaries and output expectations
- Use convergence patterns when decision consistency matters
