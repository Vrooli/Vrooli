## Practice focus: Systematic Exploration

Apply a **structured investigation methodology** to any codebase or system: define what you need to learn, map the landscape, form understanding hypotheses, validate through targeted reading, and produce findings that inform downstream decisions. This methodology produces exploration reports that accelerate planning and implementation.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

Optional reading:
- `prompt-manager skill read skill-authoring-practice`

---

### **1. When to Use This Methodology**

Use Systematic Exploration when:
- Starting work on an unfamiliar codebase or subsystem
- A leader needs to understand the technical landscape before delegating design or implementation
- Planning requires understanding existing patterns, conventions, and constraints
- Multiple agents will work in the same area and need a shared map
- A prior exploration is stale and the codebase has changed significantly

**Do NOT use** for:
- Codebases you already understand well (skip to planning)
- Single-file changes with obvious scope
- Creative or blue-sky exploration (use the `explore` steer skill instead)
- Debugging root cause analysis (use `scientific-debugging` instead)

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                     SYSTEMATIC EXPLORATION PROCESS                           │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────┐     ┌─────────┐     ┌────────────┐     ┌──────────┐          │
│   │  SCOPE  │ ──▶ │   MAP   │ ──▶ │ UNDERSTAND │ ──▶ │ VALIDATE │          │
│   └─────────┘     └─────────┘     └─────┬──────┘     └─────┬────┘          │
│                                          │                   │              │
│                           ┌──────────────┴───────────────────┘              │
│                           ▼                                                  │
│                     Understanding                                            │
│                     Sufficient?                                              │
│                      │      │                                                │
│                 YES  │      │  NO                                            │
│                      ▼      ▼                                                │
│               ┌────────────┐  │                                              │
│               │ SYNTHESIZE │  └──▶ Refine scope or map                       │
│               └────────────┘       (return to MAP or UNDERSTAND)             │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Scope**

**Entry criteria:** A codebase or subsystem needs to be understood for a downstream task.

**Actions:**
1. **State the investigation question explicitly** — What do I need to learn? Why?
2. **Identify entry points** — Known files, endpoints, commands, error messages, or configuration
3. **Define exploration boundaries** — Which directories or packages are in scope; what is out of scope
4. **Set a depth budget** — How deep to go before synthesizing (e.g., "map the top-level structure" vs. "trace the full request lifecycle")

**Exit criteria:**
- [ ] Investigation question is written down
- [ ] Entry points are identified
- [ ] Boundaries are set (in-scope / out-of-scope)

**Artifacts:**
- Exploration brief (question + entry points + boundaries + depth budget)

---

### **Phase 2: Map**

**Entry criteria:** Exploration brief exists.

**Actions:**
1. **Traverse directory structure** from entry points outward
2. **Identify key files and modules** and their apparent roles
3. **Note naming conventions** and organizational patterns
4. **Identify external dependencies** and integration points
5. **Record the call graph or data flow** for the area of interest
6. **Note test coverage** — which areas have tests, which do not

**Map Depth Decision Table:**

| Signal | Action |
|--------|--------|
| Entry points span 2+ packages | Map broadly first, then deep-dive per package |
| Single package, complex logic | Go deep immediately; map internal call graph |
| Many files, shallow logic | Breadth-first scan; focus on conventions |
| Unknown technology or framework | Start with dependency and config files before code |
| Existing tests available | Read tests before implementation code — they reveal intent |

**Exit criteria:**
- [ ] Structural map exists showing key components and their relationships
- [ ] Naming conventions and organizational patterns are documented

**Artifacts:**
- Structural map (file list with roles, dependency diagram, data flow notes)

---

### **Phase 3: Understand**

**Entry criteria:** Structural map exists.

**Actions:**
1. **For each key component, hypothesize its responsibility and contract** — What does it own? What does it expose?
2. **Identify the "happy path"** through the system for the area of interest
3. **Note surprising patterns, inconsistencies, or gaps** — Things that don't match expectations
4. **Identify what test coverage reveals** about intended behavior and edge cases
5. **Record assumptions** — Mark anything you're inferring rather than confirming

**Exit criteria:**
- [ ] Understanding hypotheses are documented for all key components
- [ ] Happy path is traced
- [ ] Assumptions are explicitly labeled

**Artifacts:**
- Component hypotheses (what each piece does, how they interact, what surprised you)

---

### **Phase 4: Validate**

**Entry criteria:** Understanding hypotheses exist.

**Actions:**
1. **Read the code that most directly tests each hypothesis** — Confirm or refute
2. **Check edge cases and error paths** — These often reveal true contracts
3. **Validate against existing tests** — Tests encode intended behavior
4. **If understanding is wrong, update hypothesis and re-read**

**Decision Table:**

| Validation Result | Action |
|-------------------|--------|
| Hypothesis confirmed by code and tests | Mark as confirmed, move to next component |
| Hypothesis contradicted by code | Update hypothesis, re-read with corrected understanding |
| No direct evidence available | Mark as assumed with confidence level, note in findings |
| Hypothesis reveals a gap or inconsistency in code | Document as a risk for downstream work |

**Exit criteria:**
- [ ] Each hypothesis is confirmed, corrected, or marked as assumed with confidence level
- [ ] No untested high-risk assumptions remain

**Artifacts:**
- Validated understanding with confidence levels per component

---

### **Phase 5: Synthesize**

**Entry criteria:** Validated understanding exists.

**Actions:**
1. **Write a structured findings report** answering the original investigation question
2. **List patterns and conventions discovered** that downstream work must follow
3. **Identify constraints** that downstream planning or implementation must respect
4. **Note risks, technical debt, or surprises** discovered during exploration
5. **Recommend specific entry points** for implementation work

**Findings Report Template:**
```markdown
# Exploration Findings: [Investigation Question]

## Answer
[Direct answer to the investigation question]

## Key Components
| Component | Role | Key File(s) | Confidence |
|-----------|------|-------------|------------|
| ... | ... | ... | High/Medium/Low |

## Patterns and Conventions
- [Convention 1]
- [Convention 2]

## Constraints for Downstream Work
- [Must do X]
- [Must not do Y]

## Risks and Technical Debt
- [Risk 1]
- [Debt 1]

## Recommended Implementation Entry Points
- [Start with file X because...]
- [Modify Y before Z because...]
```

**Exit criteria:**
- [ ] Findings report exists and answers the investigation question
- [ ] Another agent could plan implementation from these findings alone

**Artifacts:**
- Exploration findings report

---

### **3. Convergence Patterns**

#### **Sufficiency Checklist**

Before exiting exploration, verify:
- [ ] Can I answer the original investigation question?
- [ ] Do I know the key components and their responsibilities?
- [ ] Do I know the conventions downstream work must follow?
- [ ] Do I know the constraints downstream work must respect?
- [ ] Could another agent plan implementation from my findings alone?

If any answer is "no," return to the relevant phase (Map, Understand, or Validate) with a refined focus.

#### **Investigation Scope Sizing**

| Task Downstream | Exploration Depth | Typical Phases |
|-----------------|-------------------|----------------|
| Small bug fix in known area | Minimal | Scope + Map only |
| Feature addition in familiar codebase | Moderate | Scope + Map + Synthesize |
| Cross-cutting change in unfamiliar area | Full | All 5 phases |
| Architecture decision with multiple options | Full + comparative | All 5 phases per option |

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| **Reading every file sequentially** | Wastes time, loses forest for trees | Map first, then targeted reads |
| **No investigation question** | Exploration wanders without purpose | Start with explicit question and boundaries |
| **Skipping validation** | Assumptions carry into planning unchecked | Validate understanding against code and tests |
| **Not documenting findings** | Knowledge dies with the session | Write findings report for downstream agents |
| **Going too deep too early** | Rabbit holes before landscape is understood | Breadth-first map, then depth on key areas |
| **Confusing exploration with implementation** | Starts changing code during investigation | Keep exploration read-only until synthesis is complete |
| **Exploring without boundaries** | Scope creep into unrelated areas | Define in-scope/out-of-scope before starting |

---

### **5. Boundaries**

This methodology covers **investigative reading and understanding** of existing code and systems.

**Does NOT cover:**
- **Creative exploration** of new approaches — Use the `explore` steer skill
- **Debugging** root cause analysis — Use `scientific-debugging`
- **Performance profiling** — Different tooling and methodology
- **Security auditing** — Requires threat modeling framework
- **Implementation** of changes — Exploration output feeds into planning

---

### **6. Output Expectations**

When applying Systematic Exploration, you **must** produce:

1. **Exploration brief** — The investigation question, entry points, and boundaries
2. **Structural map** — Key components and their relationships
3. **Findings report** — Answers to the investigation question, patterns, constraints, risks, recommended entry points

You **should** also:
- Note any technical debt or risks discovered
- Identify test coverage gaps relevant to downstream work
- Record conventions that future changes must follow

**Quality bar:** Another agent should be able to plan implementation work from the findings report alone, without re-reading the same code.
