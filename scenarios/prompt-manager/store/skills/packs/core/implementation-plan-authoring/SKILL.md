## Practice focus: Implementation Plan Authoring

Create a durable implementation plan through `vrooli plans add --stdin` by default, preserving full execution context when conversation history is about to compact. This skill standardizes how to capture the problem, constraints, execution approach, and acceptance criteria so any future agent can continue without prior chat context.

Required reading:
- `prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement`

Optional reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read skill-validation`

---

### 1. When to Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| Context window is getting tight and work must continue later | Yes | Prevents loss of critical implementation reasoning |
| User asked for a "plan file" or "implementation plan document" | Yes | Produces a reusable execution artifact |
| Quick one-step fix with no follow-up risk | No | Plan overhead is unnecessary |
| Pure brainstorming with no execution intent | No | Use normal discussion flow |

---

### 2. Scope Boundaries

**In scope:**
- Create/update a single implementation plan through `vrooli plans add --stdin` unless an in-repo durable artifact is explicitly requested
- Preserve problem statement, root causes, constraints, and action plan
- Include explicit required reading commands for future agents
- Include Action commands as evidence or validation steps when relevant, while keeping required-reading discovery focused on skills and methodologies
- Define objective acceptance criteria and test/validation gates
- Document strict constraints such as greenfield/no-compatibility when requested

**Out of scope:**
- Implementing the planned code changes
- Replacing design docs, PRDs, or user-facing product docs
- Writing migration guides unless explicitly requested

---

### 3. Inputs

Required:
- The concrete problem being solved
- The target code areas/systems involved
- Any hard constraints (for example: greenfield only, no compatibility layers)

Optional but recommended:
- Live command evidence showing failures or friction
- Adjacent skills/tools that should be treated as source-of-truth

---

### 4. Convergence Pattern

Use this decision flow to keep outputs consistent:

1. **Was implementation already started?**
   - Yes: document current state, known blockers, and exact resume point.
   - No: document baseline, risks, and first executable phase.

2. **Did the user request strict architecture constraints?**
   - Yes: add a hard-rule section (for example: "Greenfield Constraint") and repeat it in Definition of Done.
   - No: include standard guardrails only.

3. **Is there command-level evidence?**
   - Yes: include exact command snippets and observed outputs/errors.
   - No: label assumptions clearly and provide verification commands to replace assumptions.

4. **Is work multi-component?**
   - Yes: phase by component with explicit dependencies and ordering.
   - No: keep phases minimal and linear.

---

### 5. Plan Authoring Workflow

#### Step A: Establish canonical context

Run required reading first:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Then gather implementation evidence (commands, files, observed failures) before writing.

#### Step B: Create the scratch plan through the CLI

Default behavior:
- Use `vrooli plans add --title "<topic>" --stdin` and write the plan content to stdin.
- Report the saved path and plan id printed by the command.
- Do not hard-code the plan storage directory; the CLI owns the location.

In-repo exceptions:
- Use an in-repo plan only when the user explicitly asks for it, the plan is being promoted to durable documentation, or a scenario-specific workflow requires it.
- Swarm Manager backlog items keep using their item-local `plan.md` / `conclusion.md` artifacts.

#### Step C: Write the plan with mandatory sections

Every plan must include:
1. Purpose
2. Required Reading (explicit commands)
3. Problem Statement
4. Scope (in/out)
5. Current Technical Context (key files/components)
6. Target End State
7. Implementation Strategy (phased)
8. Contract Decisions (API/CLI/data model behavior)
9. Testing Plan
10. Rollout/Validation Checklist
11. Risks + Mitigations
12. Non-goals / Prohibited Patterns
13. Definition of Done

If user requests strict constraints (for example greenfield):
- Add a dedicated hard-rule section near the top.

#### Step D: Validate quality before finishing

Quality checks:
- Another agent could execute without this chat history
- All major claims are tied to files/commands or marked assumptions
- Constraints are explicit, repeated where needed, and testable
- Acceptance criteria are objective (pass/fail), not narrative

---

### 6. Guardrails

- Do not write vague plans ("improve X", "refactor Y") without concrete deliverables.
- Do not hide assumptions; mark unknowns and how to resolve them.
- Do not mix implementation and plan-authoring in the same step unless explicitly asked.
- Do not include legacy/migration/compatibility guidance when user requested greenfield-only.

---

### 7. Output Expectations

**Must produce:**
- A saved implementation plan file path
- A plan detailed enough for handoff to a future agent with no chat context
- Required-reading command block, including:
  - `prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement`
- Action validation or discovery commands only when they are directly relevant to the plan's execution path

**May include:**
- Copy-pastable command snippets
- File-level references for targeted implementation
- Prioritized phase ordering with dependency notes

**Must not include:**
- Placeholder-only sections with no actionable content
- Contradictory constraints (for example greenfield + migration bridge)
- Implicit assumptions presented as facts
