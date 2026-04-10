## Practice focus: Leader Triage-Investigate-Resolve Pipeline

Orchestrate the **full lifecycle of delegated problem resolution**: triage severity and blast radius, drive hypothesis-based investigation through team members, then coordinate the fix with regression testing and verification. This pipeline composes leaf skills (`triage-methodology`, `scientific-debugging`) into a leader workflow with explicit phase gates, coordination protocols, and investigation cycles.

Required reading:
- `prompt-manager skill read triage-methodology`
- `prompt-manager skill read scientific-debugging`

Optional reading:
- `prompt-manager skill read skill-principles`

---

### **1. When to Use This Pipeline**

#### **Pipeline Entry Decision Table**

| You have... | Severity known? | Root cause known? | Entry point |
|---|---|---|---|
| Bug report, severity unknown | No | No | **Phase 1: Triage** |
| Bug report, severity already assessed | Yes | No | **Phase 2: Investigate** |
| Root cause confirmed, fix needed | Yes | Yes | **Phase 3: Resolve** |
| Not a bug / already has known fix | N/A | N/A | Do not use this pipeline — apply known fix directly |
| Performance issue | N/A | N/A | Do not use this pipeline — requires profiling methodology |
| Security incident | N/A | N/A | Do not use this pipeline — requires containment-first response |
| Single-agent debugging, no delegation | N/A | N/A | Do not use this pipeline — use `scientific-debugging` directly |

#### **When NOT to use this pipeline**

- **Single-agent work** — If you are the only debugger, use `scientific-debugging` directly without pipeline coordination overhead.
- **Known fixes** — If the root cause and fix are already known, apply the fix directly.
- **Performance debugging** — Requires profiling tools and measurement methodology, not hypothesis testing.
- **Security incidents** — Require containment before investigation; different response protocol.
- **Feature work** — Use `leader-explore-plan-implement` instead.

---

### **2. The Process**

```
┌─────────────────────────────────────────────────────────────────────────┐
│           LEADER TRIAGE-INVESTIGATE-RESOLVE PIPELINE                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌──────────┐  GATE 1  ┌─────────────┐  GATE 2  ┌───────────┐        │
│   │  TRIAGE  │ ───────▶ │ INVESTIGATE │ ───────▶ │  RESOLVE  │        │
│   │          │          │             │          │           │        │
│   │(triage-  │          │(scientific- │          │(fix +     │        │
│   │methodolo-│          │ debugging)  │          │ verify)   │        │
│   │gy)       │          │             │          │           │        │
│   └──────────┘          └──────┬──────┘          └─────┬─────┘        │
│                                │                       │              │
│                         ┌──────┴──────┐                │              │
│                         ▼             │                │              │
│                   Hypothesis    REWORK ◀───────────────┘              │
│                   Confirmed?    (fix didn't resolve bug)              │
│                    │      │                                            │
│               YES  │      │  NO                                        │
│                    ▼      ▼                                            │
│              (proceed)   Generate new hypothesis                       │
│                          (cycle within Investigate)                    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Triage**

Invoke the `triage-methodology` to assess severity and determine response approach.

**Entry criteria:** A bug report, incident, or problem escalation has arrived.

**Leader actions:**
1. Read the triage skill: `prompt-manager skill read triage-methodology`
2. Gather symptoms, reproduction steps, affected areas, and timeline
3. Assess severity (P0-P3) using the severity matrix and blast radius checklist
4. Decide response urgency and approach
5. If P0/P1: proceed immediately to Investigate. If P2/P3: schedule or backlog per urgency table.

**The leader typically performs triage personally** — it requires judgment about severity and blast radius that shouldn't be delegated. If the leader needs more evidence before assessing, they can delegate symptom gathering:

**Delegation message template (symptom gathering only):**
```
I need you to gather evidence for a bug triage.

Problem: [initial report]
What I need:
1. Can you reproduce the bug? Document exact steps.
2. What error messages or logs do you see?
3. What scenarios or components are affected?
4. When did this start? What changed recently?

Report back with findings. Do not investigate root cause yet.
```

**Artifacts:** Triage report (severity, blast radius, response decision).

---

### **Gate 1: Triage → Investigation**

Before proceeding to Investigate, verify:
- [ ] Severity assigned (P0/P1/P2/P3) with reasoning
- [ ] Blast radius documented (isolated / cross-scenario / expanding)
- [ ] Response urgency determined (immediate / same-day / next-sprint / backlog)
- [ ] Problem is reproducible (or intermittent conditions are documented)

**If gate fails:**
- Cannot reproduce → Assign monitoring; re-triage when new evidence appears
- Severity unclear → Escalate for second opinion before committing investigation resources

---

### **Phase 2: Investigate**

Invoke `scientific-debugging` methodology through team delegation. This phase may cycle multiple times as hypotheses are tested and rejected.

**Entry criteria:** Gate 1 is satisfied (triage report exists, problem is actionable).

**Leader actions:**
1. Read the debugging skill: `prompt-manager skill read scientific-debugging`
2. **Generate hypotheses** — Based on the triage report, produce 2-5 falsifiable hypotheses
3. **Prioritize hypotheses** — Review returned hypotheses; rank by likelihood using evidence and recency
4. **Design and run experiments** — Test the top hypothesis with a specific, minimal experiment
5. **Evaluate results:**
   - If confirmed → proceed to Gate 2
   - If rejected → cycle back to step 2 with new information
   - If inconclusive → refine the test and re-delegate
6. **Track the investigation log** — Record each hypothesis, test, and result

**Delegation message template (hypothesis generation):**
```
I need you to generate hypotheses for a bug we're investigating.

Triage report:
- Severity: [P-level]
- Symptoms: [from triage]
- Reproduction: [steps]
- Affected areas: [components]
- Timeline: [when started, what changed]

Generate 2-3 falsifiable hypotheses. For each:
1. State the claim (what you think is causing the bug)
2. What we'd expect to see if it's true
3. What we'd see if it's false
4. A specific test to validate it
5. Likelihood (High/Medium/Low) with reasoning

Read the skill first: prompt-manager skill read scientific-debugging
```

**Delegation message template (experimentation):**
```
I need you to test a hypothesis about [bug description].

Hypothesis: [specific claim]
Test plan: [what to do]
Expected if true: [observable result]
Expected if false: [observable result]

Execute the test and report:
1. What you did (exact steps)
2. What you observed
3. Whether the hypothesis is confirmed, rejected, or inconclusive
4. Any new information discovered

Read the skill first: prompt-manager skill read scientific-debugging
```

**Investigation Cycle Decision:**

| Test Result | Hypothesis Status | Next Action |
|-------------|-------------------|-------------|
| Evidence supports hypothesis | Confirmed | Proceed to Gate 2 |
| Evidence contradicts hypothesis | Rejected | Delegate new hypothesis generation with updated evidence |
| Evidence is inconclusive | Needs refinement | Design a more targeted test; re-run the experiment |
| Unexpected evidence found | New information | Incorporate into next hypothesis cycle |
| 3+ hypotheses rejected | Investigation stalled | Re-examine symptoms; consider re-triaging (different problem than assumed) |

**Artifacts:** Hypothesis log (each hypothesis + test + result), confirmed root cause.

---

### **Gate 2: Investigation → Resolution**

Before proceeding to Resolve, verify:
- [ ] Root cause identified and confirmed by experiment
- [ ] Evidence supports the root cause (not just correlation)
- [ ] The root cause explains all observed symptoms
- [ ] Hypothesis log is documented

**If gate fails:**
- Root cause only explains some symptoms → there may be multiple bugs; re-triage the unexplained symptoms separately
- Evidence is circumstantial → design a more definitive test before proceeding

---

### **Phase 3: Resolve**

Coordinate fix implementation, verification, and documentation.

**Entry criteria:** Gate 2 is satisfied (root cause confirmed with evidence).

**Leader actions:**
1. **Plan fix implementation** — Document root cause and requirements for the fix
2. **Review the fix** — Verify it addresses root cause, not just symptoms
3. **Verify regression test** — Confirm a failing test was written before the fix, and it passes after
4. **Verify full test suite** — Confirm no regressions introduced
5. **Verify original reproduction** — Confirm the original bug no longer occurs
6. **Delegate documentation** — Ensure root cause analysis is recorded

**Delegation message template (fix implementation):**
```
I need you to implement a fix for [bug description].

Root cause: [confirmed root cause with evidence]
Affected files: [from investigation]
Constraints: [from triage — severity level, blast radius considerations]

Requirements:
1. Write a failing test FIRST that reproduces the bug
2. Implement the minimal fix that addresses the root cause
3. Confirm the failing test now passes
4. Run the full test suite — no regressions
5. Document the root cause in the commit/PR description

Read the skill first: prompt-manager skill read scientific-debugging (Phase 5: Fix)
```

**Completion criteria:**
- [ ] Regression test written and passing
- [ ] Fix addresses root cause, not symptoms
- [ ] Full test suite passes
- [ ] Original bug no longer reproducible
- [ ] Root cause documented

**Root Cause Documentation Template:**
```markdown
# Root Cause Analysis: [Bug Title]

## Bug
[Brief description of symptoms]

## Root Cause
[Technical explanation of WHY the bug occurred]

## Fix
[What was changed and why this addresses the root cause]

## Prevention
[How to prevent similar bugs — patterns to watch for]

## Related Areas
[Other code that might have the same issue]
```

**Rework triggers:**
- Fix doesn't resolve the bug → Return to **Investigate** (root cause was wrong or incomplete)
- Fix introduces new bugs → Return to **Triage** for the new issue
- Fix is too invasive for current severity → Escalate for scope discussion

**Artifacts:** Regression test, fix implementation, root cause documentation.

---

### **3. Rework Triggers**

| Signal | During Phase | Action |
|---|---|---|
| Cannot reproduce the problem | Triage | Assign monitoring; re-triage when new evidence appears |
| 3+ hypotheses rejected | Investigate | Re-examine symptoms; consider re-triaging as a different problem |
| Root cause only explains some symptoms | Investigate | Likely multiple bugs; triage unexplained symptoms separately |
| Fix doesn't resolve the bug | Resolve | Return to **Investigate** — root cause was wrong or incomplete |
| Fix introduces new regressions | Resolve | Return to **Triage** for the new issue |
| Investigation reveals architectural problem | Investigate | Escalate to director-swarm; may need `leader-explore-plan-implement` instead |

---

### **4. Convergence Patterns**

#### **Delegation Sufficiency Checklist**

Before delegating any phase to a team member, verify:
- [ ] Team member has the triage report (context for all delegation)
- [ ] Specific deliverable is defined (hypotheses / test results / fix + regression test)
- [ ] Required reading commands are included
- [ ] Acceptance criteria are explicit
- [ ] Expected format is specified (hypothesis template / test report / root cause doc)

#### **Pipeline Completion Checklist**

Before declaring the pipeline complete:
- [ ] Triage report exists with severity, blast radius, and response decision
- [ ] Hypothesis log exists documenting the investigation path
- [ ] Root cause is confirmed with evidence
- [ ] Regression test exists and passes
- [ ] Full test suite passes
- [ ] Root cause documentation is recorded
- [ ] Original bug is no longer reproducible

#### **Severity-Based Time Expectations**

| Severity | Triage Phase | Investigation Phase | Resolution Phase |
|----------|-------------|--------------------|-----------------|
| P0 | Minutes | Hours | Hours |
| P1 | Hours | Same day | Same day |
| P2 | Same day | Next sprint | Next sprint |
| P3 | Whenever | Backlog | Backlog |

---

### **5. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| **Skipping triage** | Investigation without severity context wastes resources on low-priority bugs | Always triage first — even a quick assessment sets the right urgency |
| **Single-hypothesis fixation** | Confirmation bias leads to wrong root cause | Require 2-3 hypotheses; test the most likely but keep alternatives ready |
| **Fixing without understanding** | Symptom returns or moves to a different area | Confirm root cause with evidence before fixing |
| **Fixing without regression test** | Bug recurs later with no safety net | Always write failing test before implementing fix |
| **Skipping verification** | Fix may not actually resolve the original problem | Always reproduce original steps after fix to confirm resolution |
| **Fixing without a plan** | Rushed fixes miss root cause | Document root cause and fix approach before implementing |
| **Triaging by gut feeling** | Inconsistent severity assessment across bugs | Use the severity matrix from `triage-methodology` |
| **Not logging hypotheses** | Knowledge from rejected hypotheses is lost | Record every hypothesis, test, and result in the hypothesis log |

---

### **6. Boundaries**

This pipeline covers **leader-coordinated problem resolution** that flows through triage, investigation, and fix.

**Does NOT cover:**
- **Single-agent debugging** — Use `scientific-debugging` directly without coordination overhead
- **Performance debugging** — Requires profiling methodology, not hypothesis testing
- **Security incidents** — Require containment-first response before investigation
- **Feature development** — Use `leader-explore-plan-implement` pipeline
- **Strategic decisions** — This pipeline decides *how to fix*; what to prioritize is decided upstream

---

### **7. Output Expectations**

When applying this pipeline, you **must** produce:
1. **From Triage phase:** Triage report (severity, blast radius, response decision)
2. **From Investigate phase:** Hypothesis log and confirmed root cause with evidence
3. **From Resolve phase:** Regression test, fix implementation, root cause documentation

You **should** also:
- Document which phases were used (full pipeline or partial entry)
- Record investigation cycles (how many hypotheses were tested before confirmation)
- Check for similar patterns elsewhere in the codebase (noted in root cause doc)
- Report resolved bugs to QA team for inclusion in audit criteria

**Quality bar:** The root cause documentation should enable another engineer to understand why the bug happened, why the fix is correct, and how to prevent similar bugs — without re-investigating.
