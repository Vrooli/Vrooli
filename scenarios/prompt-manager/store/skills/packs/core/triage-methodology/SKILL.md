## Practice focus: Triage Methodology

Apply a **structured severity assessment** to any incoming problem, finding, or escalation: gather evidence, assess severity and blast radius, then decide on response urgency and approach. This methodology produces triage reports that drive downstream investigation, resolution, or prioritization.

Required reading:
- `prompt-manager skill read skill-principles`

Optional reading:
- `prompt-manager skill read skill-authoring-practice`

---

### **1. When to Use This Methodology**

Use Triage Methodology when:
- A bug report or incident comes in and severity is unknown
- Audit findings need prioritization before action
- An escalation arrives and you need to decide response urgency
- Cross-team conflicts need severity-based prioritization
- Multiple problems compete for attention and you need a consistent ranking

**Do NOT use** for:
- Problems with already-assessed severity (skip to investigation or resolution)
- Known issues with established fixes (apply the fix directly)
- Strategic prioritization of work requests (that's a different decision framework)
- Creative or exploratory tasks (no severity to assess)

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        TRIAGE METHODOLOGY                                    │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐     ┌──────────┐     ┌──────────┐                            │
│   │  GATHER  │ ──▶ │  ASSESS  │ ──▶ │  DECIDE  │                            │
│   │          │     │          │     │          │                            │
│   │(symptoms,│     │(severity,│     │(urgency, │                            │
│   │ evidence)│     │ blast    │     │ approach)│                            │
│   └──────────┘     │ radius)  │     └──────────┘                            │
│                    └──────────┘                                              │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Gather**

**Entry criteria:** A problem report, finding, or escalation has arrived.

**Actions:**
1. **Collect symptoms** — What is the observable problem? Error messages, unexpected behavior, data anomalies
2. **Collect reproduction steps** — Can the problem be triggered consistently? Under what conditions?
3. **Identify affected areas** — Which scenarios, components, users, or data are impacted?
4. **Establish timeline** — When did it start? What changed around that time? Is it getting worse?
5. **Gather prior context** — Has this happened before? Are there related reports or known issues?

**Exit criteria:**
- [ ] Symptoms are documented
- [ ] Reproduction status is known (reproducible / intermittent / unknown)
- [ ] Affected areas are identified
- [ ] Timeline is established

**Artifacts:**
- Problem summary (symptoms + reproduction + affected areas + timeline)

---

### **Phase 2: Assess**

**Entry criteria:** Problem summary exists with symptoms and affected areas.

**Actions:**
1. **Determine user impact** — How many users are affected? How severely?
2. **Determine data risk** — Is data being corrupted, lost, or exposed?
3. **Assess blast radius** — Is the problem isolated or spreading? Could it affect other scenarios?
4. **Assign severity** using the severity matrix below
5. **Note uncertainty** — If assessment is based on incomplete evidence, label confidence level

**Severity Matrix:**

| User Impact | Data at Risk | Severity | Description |
|-------------|-------------|----------|-------------|
| Widespread (many users, core flow broken) | Yes | **P0** | Critical — system unusable or data integrity compromised |
| Widespread | No | **P1** | High — major functionality degraded for many users |
| Limited (few users, workaround exists) | Yes | **P1** | High — data risk elevates even limited-scope problems |
| Limited | No | **P2** | Medium — noticeable issue with manageable impact |
| Cosmetic or edge case | No | **P3** | Low — minor issue, no functional impact |

**Blast Radius Checklist:**
- [ ] Is the problem isolated to one scenario or does it cross scenario boundaries?
- [ ] Could the problem affect shared resources (database, queue, cache)?
- [ ] Is the problem getting worse over time (expanding blast radius)?
- [ ] Could the problem block other teams' work?

**Exit criteria:**
- [ ] Severity assigned (P0/P1/P2/P3) with reasoning
- [ ] Blast radius documented
- [ ] Confidence level noted (High / Medium / Low)

**Artifacts:**
- Severity assessment (severity level + blast radius + confidence + reasoning)

---

### **Phase 3: Decide**

**Entry criteria:** Severity assessment exists.

**Actions:**
1. **Determine response urgency** using the urgency table below
2. **Choose approach** — Who should handle this? What methodology applies?
3. **Assign ownership** — Who is responsible for driving resolution?
4. **Communicate the triage decision** — Inform relevant stakeholders

**Response Urgency Table:**

| Severity | Response Urgency | Approach |
|----------|-----------------|----------|
| **P0** | Immediate — drop everything | Assign dedicated team; begin investigation now; escalate to leadership |
| **P1** | Same day — prioritize over current work | Assign to appropriate team lead; begin investigation within hours |
| **P2** | Next sprint — schedule for upcoming work | Add to backlog with priority; assign when capacity allows |
| **P3** | Backlog — fix when convenient | Log for future cleanup; no dedicated assignment needed |

**Approach Decision Table:**

| Problem Type | Approach | Pipeline |
|-------------|----------|----------|
| Bug with unknown root cause | Hypothesis-driven debugging | `leader-triage-investigate-resolve` |
| Bug with known root cause | Direct fix | Create swarm-manager `fix` backlog item |
| Performance degradation | Profiling and optimization | Specialized performance methodology |
| Security concern | Containment then investigation | Escalate to security review |
| Quality finding from audit | Planned remediation | Add to improvement backlog |
| Cross-team dependency conflict | Coordination and negotiation | Operations-chief escalation |

**Exit criteria:**
- [ ] Response urgency determined
- [ ] Approach chosen
- [ ] Ownership assigned
- [ ] Stakeholders informed

**Artifacts:**
- Triage report (complete document combining all phases)

---

### **3. Convergence Patterns**

#### **Triage Report Template**

```markdown
# Triage Report: [Problem Title]

## Problem Summary
- **Symptoms:** [What was observed]
- **Reproduction:** [Reproducible / Intermittent / Unknown] — [Steps if known]
- **Affected Areas:** [Scenarios, components, users]
- **Timeline:** [When started, what changed]

## Assessment
- **Severity:** [P0 / P1 / P2 / P3]
- **Reasoning:** [Why this severity level]
- **Blast Radius:** [Isolated / Cross-scenario / Expanding]
- **Data Risk:** [Yes / No] — [Details if yes]
- **Confidence:** [High / Medium / Low]

## Decision
- **Response Urgency:** [Immediate / Same day / Next sprint / Backlog]
- **Approach:** [Which methodology or pipeline]
- **Owner:** [Who is responsible]
- **Next Action:** [Specific first step]
```

#### **Incomplete Evidence Decision**

| Evidence Available | Confidence | Action |
|-------------------|------------|--------|
| Full reproduction + clear symptoms | High | Assess and decide immediately |
| Partial symptoms, no reproduction | Medium | Assess with caveats; schedule investigation to fill gaps |
| Single report, cannot reproduce | Low | Log with low severity; monitor for additional reports |
| Conflicting evidence | Low | Escalate for second opinion before deciding |

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| **Triaging without reproduction** | Severity assessment is guesswork without knowing the trigger | Attempt reproduction before assessing; mark confidence as Low if unable |
| **Severity inflation** | Everything is P0 means nothing is prioritized | Use the severity matrix honestly; P2 and P3 are valid levels |
| **Severity deflation** | Downplaying issues to avoid work | Assess data risk separately; even limited-scope data issues are P1+ |
| **Skipping blast radius** | Isolated problem turns into cascade | Always check cross-scenario impact and shared resource effects |
| **Triaging by gut feeling** | Inconsistent prioritization across different leads | Use the severity matrix and urgency table for consistent decisions |
| **Holding a triage decision** | Problem worsens while waiting for perfect information | Decide with current evidence; re-triage if new information emerges |

---

### **5. Boundaries**

This methodology covers **severity assessment and response prioritization** for incoming problems and findings.

**Does NOT cover:**
- **Investigation** of root causes — Use `scientific-debugging`
- **Resolution** of problems — Use the appropriate fix methodology
- **Strategic prioritization** of work requests — Different decision framework (opportunity cost, business value)
- **Incident response** procedures — Triage is the first step, but containment and communication follow separate processes

---

### **6. Output Expectations**

When applying Triage Methodology, you **must** produce:

1. **Problem summary** — Symptoms, reproduction status, affected areas, timeline
2. **Severity assessment** — P0/P1/P2/P3 with reasoning and confidence level
3. **Triage report** — Complete document with problem summary, assessment, and response decision

You **should** also:
- Note whether this is a new problem or recurrence of a known issue
- Check if the problem relates to active work in other teams
- Flag expanding blast radius for immediate escalation

**Quality bar:** Another lead should be able to understand the severity, agree with the assessment reasoning, and act on the response decision without re-investigating the symptoms.
