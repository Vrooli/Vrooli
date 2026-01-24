## Steer focus: Skill Authoring

Guide for creating and updating skills that steer AI agents effectively. Skills are principle-based guidance documents that help agents make good decisions without dictating exact steps.

---

### **1. Principles Over Prescriptions**

Skills guide *thinking*, not dictate *steps*. Good skills create a mental model that helps agents reason about problems.

**Good guidance:**
- "Prioritize stability-critical code paths"
- "Validate data at system boundaries"
- "Handle all states explicitly: loading, error, empty, success"

**Bad guidance:**
- "Edit src/components/Button.tsx on line 42"
- "Add exactly 5 tests per file"
- "Complete this in 30 minutes"

The goal is to teach *what to care about*, not *what to do*. Agents should be able to apply the principles to novel situations.

---

### **2. Clear Intent Statement**

Every skill opens with a clear statement of purpose:

```markdown
## Steer focus: [Skill Name]

[1-2 sentence summary of what this skill steers toward]
```

The summary should answer: "If I only read this sentence, what would I prioritize?"

**Examples:**
- "Prioritize **hardening React UI components against runtime crashes** across this scenario."
- "Focus on **test coverage for critical user journeys** rather than superficial coverage metrics."

---

### **3. Boundary Definition**

Every skill must explicitly define what's IN scope and OUT of scope. This prevents conflicts between skills and keeps agents focused.

**What's IN scope:**
- List specific areas the skill addresses
- Name the types of changes that are appropriate

**What's OUT of scope:**
- List what should NOT be changed
- Name adjacent concerns that belong to other skills

**Example:**
```markdown
### **Maintain Scenario Constraints**

* Do **not** change the scenario's core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to [skill focus]
* Prefer **incremental, localized improvements** over ambitious rewrites
```

---

### **4. The 8-Section Structure**

Skills follow a consistent structure that makes them scannable and predictable:

1. **Focus statement** - What this skill steers toward
2. **Tooling prerequisites** - Required setup (optional)
3-7. **Core principles** - Numbered sections covering main guidance areas
8. **Memory management** - Integration with visited-tracker (optional)
9. **Scenario constraints** - What's out of scope
10. **Output expectations** - What can/must be changed

The exact number of principle sections varies, but the structure should feel consistent.

---

### **5. Anti-Gaming Measures**

Skills should distinguish between "real improvements" and "superficial changes" to prevent metric-gaming:

**Examples of anti-gaming guidance:**
- "Avoid superficial changes that rename variables or restructure code without materially improving crash resistance."
- "Do not pad test counts with trivial tests that don't validate real behavior."
- "Focus on tests that would catch actual bugs, not tests that inflate coverage numbers."

If a skill can be gamed by making shallow changes, add explicit guidance about what "real" progress looks like.

---

### **6. Memory Integration**

For skills that involve systematic work across many files, integrate with `visited-tracker`:

```markdown
### **Memory Management with Visited Tracker**

**At the start of each iteration:**
visited-tracker least-visited --location scenarios/{{TARGET}}/ui --tag [skill-tag] --limit 5

**After analyzing each file:**
visited-tracker visit <file-path> --location scenarios/{{TARGET}}/ui --tag [skill-tag] --note "<summary>"
```

This prevents repeated work across conversation loops and ensures systematic coverage.

---

### **7. Protective Comments**

When skills include configuration blocks (tsconfig, eslint, etc.), wrap them in protective comments that explain:
- Why the configuration exists
- What problems it prevents
- What NOT to do

**Example:**
```jsonc
{
  "compilerOptions": {
    // ╔════════════════════════════════════════════════════════════════╗
    // ║  SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN              ║
    // ║  These rules prevent runtime crashes. If you encounter errors:║
    // ║  ✅ DO: Fix with optional chaining (?.) or null checks        ║
    // ║  ❌ DON'T: Remove the rules or use @ts-ignore                 ║
    // ╚════════════════════════════════════════════════════════════════╝
    "strict": true
  }
}
```

Future agents need to understand *why* rules exist before they can safely modify them.

---

### **8. Registration**

To publish a skill:

1. **Create the markdown file** in `scenarios/prompt-manager/skills/core/` (or `drafts/` for WIP)
2. **Add metadata entry** in `metadata.json`:

```json
{
  "id": "my-skill",
  "file": "my-skill.md",
  "name": "My Skill",
  "description": "Brief description of what this skill steers toward",
  "modes": ["Steer", "CategoryMode"],
  "tags": ["skill"],
  "icon": "LucideIconName",
  "targetToolId": null,
  "draft": false,
  "createdAt": "YYYY-MM-DDTHH:MM:SSZ",
  "updatedAt": "YYYY-MM-DDTHH:MM:SSZ"
}
```

3. **Test loading** - Verify the skill appears correctly in prompt-manager UI

---

### **9. Maintain Skill System Constraints**

* Do **not** create skills for one-off tasks (use direct instructions instead)
* Do **not** duplicate guidance that belongs in CLAUDE.md or scenario-specific docs
* Do **not** make skills overly prescriptive about file locations or exact implementations
* Prefer **updating existing skills** when guidance can be naturally extended
* Skills should be **transferable** across scenarios, not tied to specific codebases

---

### **10. Output Expectations**

You may update:
* Existing skill files for clarity, completeness, or correcting outdated guidance
* metadata.json to register new skills or update existing entries

You **must**:
* Preserve principle-based guidance style
* Include boundary definitions (what's in/out of scope)
* Include output expectations section
* Test that skills load correctly in prompt-manager
* Follow the consistent structure pattern

**Avoid:**
* Time-based constraints ("do X in 30 minutes")
* File-level directives ("edit src/foo.ts")
* Output quotas ("add 5 tests per file")
* Tool-specific instructions that may not apply to all scenarios
