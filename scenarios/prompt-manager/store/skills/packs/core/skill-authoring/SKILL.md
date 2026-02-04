## Steer focus: Skill Authoring

Guide for creating and updating skills that steer AI agents effectively. Skills are principle-based guidance documents that help agents make good decisions without dictating exact steps.

Required reading:
- `prompt-manager skills read skill-principles`

---

### **1. The Shared Mental Model Problem**

Skills exist to solve a fundamental problem: **multiple agents across different sessions need to conceptualize problems the same way.**

Without shared mental models, each agent (or even the same agent across sessions) makes different architectural decisions, leading to divergent implementations, duplication, and inconsistency. Generic advice like "keep code DRY" doesn't prevent this - agents need concrete patterns they can consistently apply.

**The goal of every skill:** Create a mental model so clear that any agent, in any session, would make the same structural decisions when facing similar problems.

---

### **2. Convergence Patterns**

The most effective skills provide **visual patterns** that agents can reference consistently. When something can be described as a decision tree, diagram, or table, agents gravitate toward it across sessions.

#### **Decision Trees**

For multi-step decisions, provide a visual flow:

```
                    Is this used in 2+ places?
                              │
              ┌───────────────┴───────────────┐
            YES                               NO
              │                                │
              ▼                                ▼
     Extract to shared/              Is this conceptually
     immediately                     reusable?
                                           │
                               ┌───────────┴───────────┐
                             YES                       NO
                               │                        │
                               ▼                        ▼
                       Design for reuse          Keep local
```

#### **Decision Tables**

For classification decisions, use YES/NO tables:

| Question | If YES | If NO |
|----------|--------|-------|
| Is it used by 2+ components? | Zustand store | Continue... |
| Will it persist across navigation? | Zustand store | Continue... |
| Is it purely ephemeral UI? | Local useState | Zustand store |

#### **Architecture Diagrams**

For layered systems, show the flow:

```
Components (consume state, never define shared state)
    ↓
Hooks (React state management, UI logic)
    ↓
Controllers (orchestration, business logic)
    ↓
Services (API calls, validation)
```

**Why these work:** Visual patterns are unambiguous. Prose like "consider whether the state is shared" leaves room for interpretation. A decision table with explicit criteria does not.

---

### **3. Principles Over Prescriptions**

While convergence patterns provide structure, skills should still guide *thinking*, not dictate *steps*.

**Good guidance:**
- "Prioritize stability-critical code paths"
- "Validate data at system boundaries"
- "Handle all states explicitly: loading, error, empty, success"

**Bad guidance:**
- "Edit src/components/Button.tsx on line 42"
- "Add exactly 5 tests per file"
- "Complete this in 30 minutes"

The goal is to teach *what to care about*, not *what to do*. Agents should be able to apply the principles to novel situations - but with enough concrete patterns that they arrive at consistent solutions.

---

### **4. Clear Intent Statement**

Every skill opens with a clear statement of purpose:

```markdown
## Steer focus: [Skill Name]

[1-2 sentence summary of what this skill steers toward]
```

The summary should answer: "If I only read this sentence, what would I prioritize?"

**Examples:**
- "Prioritize **hardening React UI components against runtime crashes** in `scenarios/{{TARGET}}/`."
- "Focus on **test coverage for critical user journeys** in `scenarios/{{TARGET}}/` rather than superficial coverage metrics."

#### **4.1 Steer Skills Must Target a Specific Scenario**

**Steer skills are always focused on a single scenario via the `{{TARGET}}` placeholder.** This is a hard requirement for all steer-mode skills.

The `{{TARGET}}` placeholder gets substituted with the actual scenario name when the skill is applied. This ensures:
- File paths reference the correct scenario (e.g., `scenarios/{{TARGET}}/ui/`, `scenarios/{{TARGET}}/api/`)
- Audit commands search the right directories
- Documentation is written to the scenario's docs folder
- The agent stays focused on one scenario rather than making sweeping cross-repo changes

**Where to use `{{TARGET}}`:**
- File paths: `scenarios/{{TARGET}}/ui/src/`
- Audit commands: `rg "pattern" scenarios/{{TARGET}}/`
- Documentation paths: `scenarios/{{TARGET}}/docs/internal/`
- visited-tracker commands: `--location scenarios/{{TARGET}}`

**Example opening for a steer skill:**
```markdown
## Steer focus: API Hardening

Prioritize **hardening API boundaries** in `scenarios/{{TARGET}}/api/` with proper validation, error handling, and type-safe contracts.

Your goal is to ensure the target scenario's API **fails gracefully** and provides clear error messages, rather than crashing or leaking internal details.
```

---

### **5. Boundary Definition**

Every skill must explicitly define what's IN scope and OUT of scope. This prevents conflicts between skills and keeps agents focused.

**What's IN scope:**
- List specific areas the skill addresses
- Name the types of changes that are appropriate

**What's OUT of scope:**
- List what should NOT be changed
- Name adjacent concerns that belong to other skills

**Note:** Session-level constraints (do not add features, preserve behavior, etc.) are now handled by **scope skills** rather than inline constraint sections. Skills with `defaultScope` in their metadata will automatically include the appropriate scope when loaded with `--with-scope`.

---

### **6. Skill Structure**

Skills follow a consistent structure that makes them scannable and predictable:

1. **Focus statement** - What this skill steers toward
2. **Tooling prerequisites** - Required setup (optional)
3. **Core principles** - Numbered sections with convergence patterns where applicable
4. **Audit section** - Assessment checklist for existing codebases (optional, see below)
5. **Memory management** - Comment/documentation guidelines and integration with memory-related tools (e.g. visited-tracker, knowledge-observatory)
6. **Output expectations** - What can/must be changed

**Note:** Session-level constraints are handled by scope skills (e.g., `refactor-scope`, `feature-scope`) rather than inline sections. Set `defaultScope` in skill metadata when appropriate.  

The exact number of principle sections varies, but the structure should feel consistent.

#### **When to Include an Audit Section**

Some skills need to assess existing codebases at any maturity level before prescribing changes. For these, include an audit section with:

- **Concrete discovery commands** (grep, find, etc.) to identify current patterns
- **Red flag checklists** that indicate problems to address
- **A documentation template** for recording findings

**Example audit pattern:**
````markdown
### **Coherence Audit**

#### Step 1: State Inventory
```bash
rg "useState\(" --type tsx -c | sort -t: -k2 -nr | head -20
```

**Red flags:**
- [ ] Files with 10+ useState calls → candidate for consolidation
- [ ] Similar state shapes in multiple components → missing shared abstraction
````

Audit sections make skills applicable to brownfield projects, not just greenfield development.

---

### **7. Anti-Gaming Measures**

Skills should distinguish between "real improvements" and "superficial changes" to prevent metric-gaming:

**Examples of anti-gaming guidance:**
- "Avoid superficial changes that rename variables or restructure code without materially improving crash resistance."
- "Do not pad test counts with trivial tests that don't validate real behavior."
- "Focus on tests that would catch actual bugs, not tests that inflate coverage numbers."

If a skill can be gamed by making shallow changes, add explicit guidance about what "real" progress looks like.

---

### **8. Agent Memory Loop**

Agent sessions are stateless - when a session ends, all context is lost. For skills that involve investigation, audit, or systematic work, agents must explicitly read and write documentation to maintain continuity across sessions.

#### The Memory Loop Pattern

```
┌─────────────────────────────────────────────┐
│           AGENT MEMORY LOOP                 │
├─────────────────────────────────────────────┤
│                                             │
│  START SESSION                              │
│       │                                     │
│       ▼                                     │
│  READ existing findings docs                │
│  (understand what prior agents discovered)  │
│       │                                     │
│       ▼                                     │
│  DO the skill's work                        │
│  (informed by prior findings)               │
│       │                                     │
│       ▼                                     │
│  WRITE updated findings                     │
│  (so next agent can continue)               │
│       │                                     │
│       ▼                                     │
│  END SESSION                                │
│                                             │
└─────────────────────────────────────────────┘
```

Without READ: Agent duplicates or overwrites prior work  
Without WRITE: Discoveries are lost when session ends  

#### Specifying Documentation Patterns

Skills that involve investigation should specify:
1. **What to read** at session start
2. **What to write** at session end
3. **Document template** for consistent structure

The specific documents depend on what makes sense for the skill's purpose. A skill might:
- Use an existing doc pattern (e.g., SEAMS.md for boundary discovery)
- Define a new doc type specific to its needs
- Use multiple documents for different types of findings

**Conventional location:** `docs/internal/` for agent-produced findings (not user-facing docs)

#### Example: Skill with Documentation Pattern

````markdown
### **Documentation**

**At session start**, read existing findings:
- `docs/internal/MY_FINDINGS.md` - prior audit results

**At session end**, update findings:

Update `docs/internal/MY_FINDINGS.md` to reflect your findings:
* The code is the source of truth. Verify existing claims before extending.
* If the file exists, correct inaccuracies and add new findings.
* Create the `docs/internal/` directory if needed.

**Template:**
```markdown
# [Skill Name] Findings

## Last Updated
[Date]

## Summary
[Current state overview]

## Findings
- [Finding with file references]

## Priority Actions
1. [Most important next step]
```
````

#### Relationship to visited-tracker

The memory loop has two complementary parts:

| Tool | Tracks | Purpose |
|------|--------|---------|
| `visited-tracker` | Which files have been analyzed | Prevents re-investigating same files |
| Findings docs | What was discovered | Preserves knowledge for future sessions |

Use both together for skills involving systematic codebase work.

**visited-tracker commands:**
```bash
# At session start: find files not yet analyzed
visited-tracker least-visited --location scenarios/{{TARGET}} --tag [skill-tag] --limit 5

# After analyzing a file: mark as visited
visited-tracker visit <file-path> --location scenarios/{{TARGET}} --tag [skill-tag] --note "<summary>"
```

---

### **9. Protective Comments**

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

### **10. Registration**

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

### **11. Maintain Skill System Constraints**

* Do **not** create skills for one-off tasks (use direct instructions instead)
* Do **not** duplicate guidance that belongs in CLAUDE.md or scenario-specific docs
* Do **not** hardcode scenario names in steer skills—use `{{TARGET}}` placeholder instead
* Prefer **updating existing skills** when guidance can be naturally extended
* Skills should be **transferable** across scenarios via the `{{TARGET}}` substitution pattern

---

### **12. Output Expectations**

You may update:
* Existing skill files for clarity, completeness, or correcting outdated guidance
* metadata.json to register new skills or update existing entries

You **must**:
* Preserve principle-based guidance style
* Include convergence patterns (decision trees, tables, diagrams) where decisions need consistency
* Include boundary definitions (what's in/out of scope)
* Include output expectations section
* Test that skills load correctly in prompt-manager
* Follow the consistent structure pattern
* **For steer skills:** Include `{{TARGET}}` placeholder in file paths, audit commands, and documentation paths

**Avoid:**
* Time-based constraints ("do X in 30 minutes")
* Hardcoded file paths without `{{TARGET}}` in steer skills
* Output quotas ("add 5 tests per file")
* Tool-specific instructions that may not apply to all scenarios
* Prose-only guidance where a decision tree or table would be clearer
