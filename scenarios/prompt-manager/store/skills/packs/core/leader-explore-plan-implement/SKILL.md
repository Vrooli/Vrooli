## Practice focus: Leader Explore-Plan-Implement Pipeline

Orchestrate the **full lifecycle of delegated technical work**: explore the problem space, author an implementation plan, then coordinate implementation across team members. This pipeline composes leaf skills (`systematic-exploration`, `implementation-plan-authoring`) into a leader workflow with explicit phase gates, coordination protocols, and convergence patterns.

Required reading:
- `prompt-manager skill read systematic-exploration`
- `prompt-manager skill read implementation-plan-authoring`

Optional reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. When to Use This Pipeline**

#### **Pipeline Entry Decision Table**

| You have... | Area understood? | Plan exists? | Entry point |
|---|---|---|---|
| New work request, unfamiliar area | No | No | **Phase 1: Explore** |
| New work request, familiar area | Yes | No | **Phase 2: Plan** |
| Existing plan from prior session | Yes | Yes | **Phase 3: Implement** |
| Bug with known cause | Yes | N/A | Do not use this pipeline — use `scientific-debugging` |
| Creative exploration request | N/A | N/A | Do not use this pipeline — use `explore` steer skill |
| Single-agent work, no delegation needed | N/A | N/A | Do not use this pipeline — use leaf skills directly |

#### **When NOT to use this pipeline**

- **Single-agent work** — If you are the only worker, use `systematic-exploration` or `implementation-plan-authoring` directly without the pipeline coordination overhead.
- **Debugging** — Use `scientific-debugging` (different methodology, different artifacts).
- **Creative exploration** — Use the `explore` steer skill (different intent: divergent thinking vs. convergent investigation).
- **Operations or deployment** — Different coordination patterns apply.
- **Strategic prioritization** — This pipeline is about *how to execute*; strategic decisions about *what to execute* happen before the pipeline is invoked.

---

### **2. The Process**

```
┌─────────────────────────────────────────────────────────────────────────┐
│             LEADER EXPLORE-PLAN-IMPLEMENT PIPELINE                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌──────────────┐  GATE 1  ┌──────────┐  GATE 2  ┌───────────────┐   │
│   │   EXPLORE    │ ───────▶ │   PLAN   │ ───────▶ │  IMPLEMENT    │   │
│   │              │          │          │          │               │   │
│   │ (systematic- │          │(impl-plan│          │ (coordinate   │   │
│   │  exploration)│          │-authoring│          │  & delegate)  │   │
│   └──────────────┘          └──────────┘          └───────┬───────┘   │
│         │                        │                        │           │
│         │  REWORK ◀──────────────┘                        │           │
│         │  (plan reveals exploration gaps)                 │           │
│         │                                                 │           │
│         │  REWORK ◀───────────────────────────────────────┘           │
│         │  (implementation reveals wrong assumptions)                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Explore**

Invoke the `systematic-exploration` methodology to build understanding of the problem space.

**Entry criteria:** A work request or priority exists that requires understanding before planning.

**Leader actions:**
1. Read the exploration skill: `prompt-manager skill read systematic-exploration`
2. Define the investigation question based on the work request
3. Decide: perform exploration personally or delegate to a team member
   - Delegate when the area is large or the leader's time is better spent on other coordination
   - Perform personally when the area is small or the leader needs firsthand understanding for planning
4. If delegating: send assignment using the delegation template below
5. Review the findings report when complete

**Delegation message template:**
```
I need you to explore [area] using the systematic-exploration methodology.

Investigation question: [question]
Entry points: [files/endpoints/commands]
Boundaries: [in-scope / out-of-scope]
Depth budget: [how deep to go]

Deliver: Findings report answering the investigation question.
Read the skill first: prompt-manager skill read systematic-exploration
```

**Artifacts:** Exploration findings report.

---

### **Gate 1: Exploration → Planning**

Before proceeding to Plan, verify:
- [ ] Findings report exists and answers the investigation question
- [ ] Key components, patterns, and constraints are documented
- [ ] Another agent could plan implementation from the findings alone
- [ ] No critical unknowns remain (or unknowns are explicitly labeled with risk)

**If gate fails:** Return to Explore with a refined scope targeting the specific gap.

---

### **Phase 2: Plan**

Invoke the `implementation-plan-authoring` methodology to design the approach.

**Entry criteria:** Gate 1 is satisfied (exploration findings exist and are sufficient).

**Leader actions:**
1. Read the planning skill: `prompt-manager skill read implementation-plan-authoring`
2. Decide: author the plan personally or delegate to a team member
   - Delegate when the leader has strong findings and clear requirements to hand off
   - Author personally when the leader needs to make architectural decisions during planning
3. If delegating: send assignment using the delegation template below
4. Review the plan against the 13 mandatory sections from `implementation-plan-authoring`
5. Verify the plan respects all constraints discovered during exploration

**Delegation message template:**
```
I need you to author an implementation plan using the implementation-plan-authoring methodology.

Context: [findings report location or summary]
Requirements: [work request details]
Constraints discovered: [from exploration findings]
Hard rules: [if any, e.g., greenfield only]

Deliver: Implementation plan file at docs/plans/[topic]-implementation-plan.md
Read the skill first: prompt-manager skill read implementation-plan-authoring
```

**Mandatory plan sections** (from `implementation-plan-authoring`):
1. Purpose
2. Required Reading
3. Problem Statement
4. Scope (in/out)
5. Current Technical Context
6. Target End State
7. Implementation Strategy (phased)
8. Contract Decisions
9. Testing Plan
10. Rollout/Validation Checklist
11. Risks + Mitigations
12. Non-goals / Prohibited Patterns
13. Definition of Done

**Artifacts:** Implementation plan file.

---

### **Gate 2: Planning → Implementation**

Before proceeding to Implement, verify:
- [ ] Plan has all 13 mandatory sections
- [ ] Plan respects constraints from exploration findings
- [ ] Acceptance criteria are objective (pass/fail, not narrative)
- [ ] Phases are ordered with explicit dependencies
- [ ] Another agent could implement from the plan alone

**If gate fails:** Update the plan. If the plan reveals exploration gaps, return to Explore.

---

### **Phase 3: Implement**

Coordinate implementation across team members using the validated plan.

**Entry criteria:** Gate 2 is satisfied (validated implementation plan exists).

**Leader actions:**
1. **Break the plan into assignable work items** — One per team member or one per plan phase if sequential
2. **Assign work** using delegation messages with full context
3. **Track progress** — Establish check-in cadence; monitor via team messaging
4. **Review completed work** against plan acceptance criteria
5. **Integrate results** — When all phases complete, verify end-to-end Definition of Done

**Delegation message template:**
```
I need you to implement [phase N] from the plan at [plan-file-path].

Your scope: [specific deliverables from this phase]
Acceptance criteria: [from plan]
Constraints: [from plan]
Dependencies: [what must be done before/after your work]
Required reading: prompt-manager skill read [relevant-skills]

Report back when complete or if you are blocked.
```

**Coordination protocol** (current capabilities):
- Use `team message-send` for assignments and status checks (point-to-point)
- For multi-member assignments, send individual messages to each assignee
- Use `team org-list` to verify team structure before assigning
- Establish check-in cadence in the initial delegation message (e.g., "message me after each file is complete")
- Track assignments in the team shared doc or as plan file annotations

**Completion criteria:**
- [ ] All plan phases are implemented
- [ ] All acceptance criteria from the plan pass
- [ ] Tests pass (as specified in the plan's Testing Plan section)
- [ ] Documentation is updated (as specified in the plan)
- [ ] Definition of Done is satisfied

**Artifacts:** Implemented code, tests, documentation per plan.

---

### **3. Rework Triggers**

Rework is cheaper than implementing a wrong plan. When signals appear, return to the appropriate phase.

| Signal | During Phase | Action |
|---|---|---|
| "I don't understand how X works" | Plan | Return to **Explore**, scope X specifically |
| "The plan assumes X but code does Y" | Implement | Return to **Plan**, update assumptions |
| "I found a constraint not in the plan" | Implement | Return to **Explore + Plan** if systemic; update plan only if localized |
| "Requirements conflict with code reality" | Plan or Implement | **Escalate** to work requester |
| "Tests reveal unexpected behavior" | Implement | Return to **Explore** if behavior is not understood; fix plan if understood |

---

### **4. Convergence Patterns**

#### **Delegation Sufficiency Checklist**

Before delegating any phase to a team member, verify:
- [ ] Team member has access to all required context (findings, plan, skills)
- [ ] Acceptance criteria are explicit and objective
- [ ] Required reading commands are included
- [ ] Expected deliverable format is specified
- [ ] Check-in cadence is established
- [ ] Blocking dependencies are identified

#### **Pipeline Completion Checklist**

Before declaring the pipeline complete:
- [ ] Findings report exists (from Explore or pre-existing)
- [ ] Implementation plan exists with all 13 sections (from Plan or pre-existing)
- [ ] All plan phases are implemented
- [ ] Definition of Done criteria pass
- [ ] Rework loops (if any) are documented for future reference

---

### **5. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| **Skipping exploration** | Plan is built on wrong assumptions; rework during Implement | Invest in Explore phase — it saves rework later |
| **Exploring without a question** | Exploration wanders, never converges | State the investigation question upfront in the exploration brief |
| **Planning without exploration findings** | Plan is disconnected from code reality | Feed findings into planning; verify constraints match |
| **Delegating without context** | Agent wastes time re-discovering what the leader already knows | Include findings, plan reference, and skill reading in every delegation |
| **No phase gates** | Bad assumptions propagate unchecked through the pipeline | Use Gate 1 and Gate 2 checklists before proceeding |
| **Rework avoidance** | Sunk-cost thinking keeps a bad plan alive | Rework is cheaper than implementing a wrong plan |
| **Leader implements instead of delegates** | Defeats the pipeline purpose; leader becomes bottleneck | Leaders coordinate; team members implement |
| **Skipping check-ins during Implement** | Blocked agents sit idle; work diverges from plan | Establish cadence upfront; check in at each phase boundary |

---

### **6. Boundaries**

This pipeline covers **leader-coordinated technical work** that flows through exploration, planning, and implementation.

**Does NOT cover:**
- **Single-agent work** — Use leaf skills directly without pipeline overhead
- **Debugging workflows** — Use `scientific-debugging` (different methodology)
- **Creative exploration** — Use the `explore` steer skill (different intent)
- **Operations and deployment** — Different coordination patterns
- **Strategic planning** — Director-level strategy decides *what* to do; this pipeline decides *how* to execute

---

### **7. Output Expectations**

When applying this pipeline, you **must** produce:
1. **From Explore phase:** Findings report (or verify an existing one is sufficient)
2. **From Plan phase:** Implementation plan file with all 13 mandatory sections
3. **From Implement phase:** Completed deliverables matching plan acceptance criteria

You **should** also:
- Document which phases were used (full pipeline or partial entry)
- Record any rework loops and what triggered them (this improves future exploration quality)
- Update the plan file with implementation notes for future reference

**Quality bar:** The pipeline should reduce total rework by front-loading understanding. If you find yourself in multiple Implement-to-Explore rework loops, the exploration phase was insufficient.
