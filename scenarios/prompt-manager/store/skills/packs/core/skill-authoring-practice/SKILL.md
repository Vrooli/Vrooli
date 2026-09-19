---
name: "skill-authoring-practice"
description: "Authoring guide for Practice skills that define systematic engineering methodologies. Covers process structure, convergence patterns, knowledge capture, and artifact requirements."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta","practice"]
  tags: ["skill","practice","authoring"]
  icon: "wrench"
  status: "active"
  revision: 28
  createdAt: "2026-02-03T02:40:00Z"
  updatedAt: "2026-02-04T13:13:54Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
## Meta focus: Practice Skill Authoring

Guide for creating **practice** skills (the authored skill declares `modes[0] = "practice"`). Practice skills define systematic engineering methodologies — repeatable approaches to recurring challenges that apply across scenarios, tools, and tech stacks.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. What Practice Skills Are**

Practice skills encode **how to approach a class of problems systematically**. The six skill categories and how they differ are the SSOT in `docs/agent-system/PRIMITIVES.md` §"Skill" (the category taxonomy table) — read it there, do not restate it. The practice-specific contrast: where Steer optimizes what to build in one scenario, Platform shared-package evolution, Search where to find information, Tools how to use one tool, and Meta skill-system governance, Practice optimizes **how to think** — a repeatable methodology that applies to any codebase with no `{{TARGET}}`.

**Key characteristics:**
- **Methodology-focused** — defines a repeatable process, not a destination
- **Scenario-agnostic** — no `{{TARGET}}` placeholder; applies anywhere
- **Knowledge-producing** — generates artifacts (tests, docs, findings) that outlive the session
- **Falsifiable** — includes decision points where hypotheses can be validated or rejected

Examples: Scientific Debugging, Code Review, Incident Response, Performance Investigation, Security Audit.

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

### **3. The Process Section**

The core of every Practice skill is a **repeatable process**, structured as numbered phases with clear entry/exit criteria.

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

Each phase must be **independently valuable** (produces something useful even if stopped early), **observable** (clear completion criteria), and **documented** (artifacts persist beyond the session).

---

### **4. Convergence Patterns for Methodologies**

Canon owns the pattern forms (`docs/agent-system/SKILL_AUTHORING.md` §"Convergence patterns"). The practice-specific rule: pick the form that matches the methodology's shape — linear flow with feedback loops (debugging), parallel tracks that synthesize (review), triage-first branches (incident response), condition tables (escalation decisions), checklists (phase exit gates). Example of a linear flow with a feedback loop:

```
Observe ──▶ Hypothesize ──▶ Test ──▶ Analyze ──┬──▶ Fix ──▶ Verify
                 ▲                              │
                 └──── (rejected) ◀─────────────┘
```

The falsifiability characteristic lives here: every methodology needs at least one point where a hypothesis can be *rejected* and the flow routes back, not just forward.

---

### **5. Knowledge Capture**

Every application of a Practice skill must produce artifacts that **persist** (outlive the session), **explain** (document the why), and **prevent** (help future agents avoid the same issues). When authoring, specify which artifact types the methodology produces (tests, findings docs, code comments, runbooks, benchmarks) and where they live.

---

### **6. Anti-Patterns Section**

Every Practice skill includes an anti-patterns table warning against the methodology's characteristic mistakes:

```markdown
| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| Shotgun debugging | Obscures the cause | Systematic hypothesis testing |
```

---

### **7. Output Expectations**

You may update Practice skills for clarity, additional patterns, or improved workflows.

You must:
- Include a clear, numbered process with phases and entry/exit criteria
- Include convergence patterns matched to the methodology's shape (§4)
- Ensure the process is falsifiable (decision points where hypotheses can fail)
- Specify the artifacts the methodology produces (§5)
- Include a methodology-specific anti-patterns table (§6)
- Keep the methodology scenario-agnostic (no `{{TARGET}}`)

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "practice"`.
