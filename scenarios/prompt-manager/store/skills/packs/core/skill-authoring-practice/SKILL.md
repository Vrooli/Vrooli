## Meta focus: Practice Skill Authoring

Guide for creating **Practice** skills (where `modes[0] = "Practice"`). Practice skills define systematic engineering methodologies—repeatable approaches to recurring challenges that apply across scenarios, tools, and tech stacks.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. What Practice Skills Are**

Practice skills encode **how to approach a class of problems systematically**. They differ from other categories:

| Category | Focus | Scope | Example |
|----------|-------|-------|---------|
| Steer | What to build | Single scenario (`{{TARGET}}`) | "React components should handle all states" |
| Search | Where to find info | Single scenario (`{{TARGET}}`) | "Trace auth flow through codebase" |
| Tools | How to use X | Specific tool | "Use scenario-to-desktop CLI correctly" |
| **Practice** | How to think | **Any codebase** | "Debug using hypothesis-driven analysis" |

**Key characteristics:**
- **Methodology-focused** — Defines a repeatable process, not a destination
- **Scenario-agnostic** — No `{{TARGET}}` placeholder; applies anywhere
- **Knowledge-producing** — Generates artifacts (tests, docs, findings) that outlive the session
- **Falsifiable** — Includes decision points where hypotheses can be validated or rejected

**Examples of Practice skills:**
- Scientific Debugging — hypothesis-driven root cause analysis
- Code Review — systematic evaluation of changes for quality and correctness
- Incident Response — structured approach to production issues
- Performance Investigation — profiling and optimization methodology
- Security Audit — systematic vulnerability discovery

---

### **2. Category Scope**

**In scope:**
- Systematic approaches to recurring engineering challenges
- Multi-step workflows with explicit decision points
- Hypothesis generation and validation patterns
- Knowledge capture and transfer across sessions

**Out of scope:**
- Scenario-specific implementation guidance (belongs in Steer)
- Tool-specific operation instructions (belongs in Tools)
- Information discovery workflows (belongs in Search)
- Skill system governance (belongs in Meta)

---

### **3. Recommended Structure**

Practice skills follow a consistent structure:

1. **Focus statement** — What methodology this skill teaches
2. **When to use** — Problem patterns this methodology addresses
3. **The process** — Step-by-step workflow with decision points
4. **Convergence patterns** — Decision trees, tables, or diagrams for consistent application
5. **Artifacts** — What the process produces (tests, docs, findings)
6. **Anti-patterns** — Common mistakes and how to avoid them
7. **Boundaries** — What this methodology does NOT cover
8. **Output expectations** — What must be produced when applying this skill

---

### **4. The Process Section**

The core of every Practice skill is a **repeatable process**. Structure it as numbered phases with clear entry/exit criteria.

**Phase template:**
```markdown
### **Phase N: [Name]**

**Entry criteria:** [What must be true to start this phase]

**Actions:**
1. [Concrete step]
2. [Concrete step with decision point]
   - If X: [Branch A]
   - If Y: [Branch B]

**Exit criteria:** [What must be true to proceed]

**Artifacts:** [What this phase produces]
```

Each phase should be:
- **Independently valuable** — Produces something useful even if stopped early
- **Observable** — Has clear completion criteria
- **Documented** — Produces artifacts that persist beyond the session

---

### **5. Convergence Patterns**

Practice skills must include convergence patterns so agents apply the methodology consistently. Choose visual decision aids appropriate to your methodology:

#### **Process Flow Diagrams**

Show the overall workflow and decision branches. The structure depends on your methodology's nature.

**Example A — Linear with feedback loops (debugging):**
```
Observe ──▶ Hypothesize ──▶ Test ──▶ Analyze ──┬──▶ Fix ──▶ Verify
                 ▲                              │
                 └──── (rejected) ◀─────────────┘
```

**Example B — Parallel tracks (code review):**
```
┌─────────────────────────────────────────────────┐
│                  CODE REVIEW                    │
├─────────────────────────────────────────────────┤
│  Correctness    Security    Performance    Style│
│      │              │            │           │  │
│      ▼              ▼            ▼           ▼  │
│   [checks]      [checks]     [checks]    [checks]│
│      │              │            │           │  │
│      └──────────────┴────────────┴───────────┘  │
│                      ▼                          │
│               Synthesize Findings               │
└─────────────────────────────────────────────────┘
```

**Example C — Triage-first (incident response):**
```
Alert ──▶ Assess Severity ──┬──▶ Critical ──▶ Mitigate ──▶ Investigate
                            ├──▶ High ──▶ Investigate ──▶ Fix
                            └──▶ Low ──▶ Queue for later
```

#### **Decision Tables**

Use tables when outcomes depend on multiple conditions. Structure depends on what decisions your methodology requires.

**Example — When to escalate (incident response):**

| User Impact | Data at Risk | Action |
|-------------|--------------|--------|
| Widespread | Yes | Immediate escalation, all hands |
| Widespread | No | Page on-call, begin mitigation |
| Limited | Yes | Security team + on-call |
| Limited | No | Standard investigation |

**Example — Review depth (code review):**

| Change Type | Test Coverage | Review Depth |
|-------------|---------------|--------------|
| Security-critical | Any | Deep review + security team |
| Core logic | < 80% | Request more tests first |
| Core logic | >= 80% | Standard review |
| Config/docs | Any | Quick review |

#### **Checklists**

Use checklists for phases requiring thoroughness. Tailor to what your methodology needs to verify.

**Example — Before approving (code review):**
```markdown
- [ ] Does the change do what the PR description claims?
- [ ] Are edge cases handled?
- [ ] Are there adequate tests?
- [ ] Any security implications?
```

**Example — Before closing incident:**
```markdown
- [ ] Root cause identified and documented?
- [ ] Fix deployed and verified?
- [ ] Monitoring added to catch recurrence?
- [ ] Post-mortem scheduled if warranted?
```

---

### **6. Knowledge Capture**

Practice skills must specify how knowledge is captured and transferred. Every application of the skill should produce artifacts that:

1. **Persist** — Outlive the session (tests, docs, code comments)
2. **Explain** — Document the "why," not just the "what"
3. **Prevent** — Help future agents avoid the same issues

**Common artifact types** (choose what fits your methodology):

| Artifact Type | Example Use Cases |
|---------------|-------------------|
| Tests | Debugging (regression tests), Code Review (suggested tests) |
| Documentation | Incident Response (post-mortems), Security Audit (findings reports) |
| Code comments | Any methodology that reveals non-obvious behavior |
| Runbooks | Incident Response (how to handle specific failures) |
| Metrics/dashboards | Performance Investigation (before/after benchmarks) |

When authoring, specify which artifacts your methodology produces and where they should live.

---

### **7. Anti-Pattern Section**

Every Practice skill should include an anti-patterns section warning against common mistakes. The specific anti-patterns depend on your methodology.

**Template:**
```markdown
### **Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| [Common mistake] | [Consequence] | [Correct approach] |
```

**Example anti-patterns by methodology:**

*Debugging:*
- Shotgun debugging → obscures cause → systematic hypothesis testing
- Fixing symptoms → bug returns → find root cause first

*Code Review:*
- Rubber-stamping → bugs slip through → use checklists
- Style nitpicking → misses real issues → prioritize correctness

*Incident Response:*
- Premature root cause → wrong fix → stabilize first, investigate second
- Hero culture → burnout, knowledge silos → documented runbooks

---

### **8. Boundaries**

Practice skills should explicitly state what they do NOT cover, pointing to other resources where appropriate.

**Template:**
```markdown
### **Boundaries**

This methodology covers [specific scope].

Does NOT cover:
- **[Related but different thing]** — [Where to look instead]
- **[Adjacent concern]** — [Where to look instead]
```

Be specific about scope to prevent misapplication.

---

### **9. Registration**

To publish a Practice skill:

1. **Create the directory** in `scenarios/prompt-manager/store/skills/packs/core/<skill-id>/`
2. **Add SKILL.md** with the skill content
3. **Add skill.json** with metadata including `modes: ["practice"]`
4. **Run sync** via `prompt-manager skill sync`
5. **Verify** via `prompt-manager skill show <id>`

---

### **10. Output Expectations**

You may update:
- Practice skills for clarity, additional patterns, or improved workflows
- Related skills that reference Practice methodologies

You must:
- Include a clear, numbered process with phases
- Include convergence patterns (diagrams, tables, checklists) appropriate to the methodology
- Specify artifacts produced by the methodology
- Include anti-patterns section with methodology-specific pitfalls
- Keep the methodology scenario-agnostic (no `{{TARGET}}`)
- Ensure the process is falsifiable (has decision points where hypotheses can fail)

**Avoid:**
- Scenario-specific guidance (use Steer instead)
- Tool-specific instructions (use Tools instead)
- Vague advice without concrete steps
- Processes without observable completion criteria
- Examples that only apply to one type of methodology
