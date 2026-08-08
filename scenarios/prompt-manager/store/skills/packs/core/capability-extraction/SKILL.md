## Practice focus: Capability Extraction

Audit agent files (SOUL.md, AGENTS.md, TOOLS.md) to **identify embedded methodologies that should be extracted into reusable skills**. This methodology covers the first half of the promotion pipeline: `Agent instructions -> Skills`. Deterministic operations should continue toward Vrooli-controlled CLI implementation and Action exposure, while judgment stays in skills.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `docs/agent-system/PROMOTION_LADDER.md`
- `prompt-manager skill read skill-authoring-practice`

Optional reading:
- `prompt-manager skill read skill-validation`

---

### **1. When to Use This Methodology**

#### **Entry Decision Table**

| You have... | Use this? | Entry point |
|---|---|---|
| Agent AGENTS.md files that are long or contain inline workflows | Yes | **Phase 1: Audit** |
| Known duplication across multiple agents | Yes | **Phase 2: Classify** (skip broad audit) |
| Friction report showing agents re-inventing methodology | Yes | Use as input to **Phase 1: Audit** |
| Single skill needs improvement | No | Use `skill-improvement-suggestions` directly |
| Need to find CLI/Action promotion candidates from existing skills | No | Use meta-optimization's Action conversion workflow |
| Agent identity/personality tuning | No | That's SOUL.md content, not extractable methodology |
| Agent TOOLS.md needs skill references added | No | That's a consequence of extraction, not a trigger for it |

#### **When NOT to use this methodology**

- **Single-skill improvement** — Use `skill-improvement-suggestions` directly.
- **CLI/Action promotion of existing skills** — Use meta-optimization's Action conversion workflow (different scope: skills -> CLI -> Action, not agents -> skills).
- **Agent personality/identity design** — SOUL.md defines *who* the agent is; that's not extractable methodology.
- **Skill authoring** — This methodology identifies *what* to extract; use `skill-authoring-practice` for *how* to write the skill.
- **Skill validation** — Use `skill-validation` after the extracted skill is created.

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                    CAPABILITY EXTRACTION PROCESS                              │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────┐     ┌──────────┐     ┌────────────┐     ┌──────────┐         │
│   │  AUDIT  │ ──▶ │ CLASSIFY │ ──▶ │ PRIORITIZE │ ──▶ │   PLAN   │         │
│   │         │     │          │     │            │     │EXTRACTION│         │
│   │(read    │     │(test +   │     │(compound   │     │          │         │
│   │ agent   │     │ type)    │     │ impact)    │     │(specs)   │         │
│   │ files)  │     │          │     │            │     │          │         │
│   └─────────┘     └──────────┘     └────────────┘     └──────────┘         │
│                                                                              │
│   Downstream:                                                                │
│   ┌──────────────────────┐     ┌───────────────────────────────────┐        │
│   │ skill-authoring-     │     │ leader-research-analyze-plan      │        │
│   │ practice             │     │ (for CLI promotion of new skills) │        │
│   │ (to create the skill)│     │                                   │        │
│   └──────────────────────┘     └───────────────────────────────────┘        │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Audit**

**Entry criteria:** Agent file(s) need analysis for extractable patterns. Trigger may be: agent AGENTS.md is long, friction report identifies re-invented methodology, or periodic meta-optimization sweep.

**Actions:**
1. **Select audit scope** using the Scope Sizing Table (Section 4)
2. **Read AGENTS.md** for each target agent — focus on the Workflow section
3. **Read TOOLS.md** for each target agent — note which skills are already referenced
4. **Scan SOUL.md** briefly to understand the agent's identity (this is context, not extraction material)
5. **Identify embedded patterns** — look for:
   - Numbered workflow steps (especially with delegation points)
   - Decision frameworks with structured criteria
   - Inline work tables mapping inputs to actions
   - Communication formats or status protocols
   - Delegation protocols or assignment templates
   - Verification checklists embedded in workflow prose
6. **Tag each candidate** with: source agent, pattern type, approximate line count, descriptive label

**Exit criteria:**
- [ ] All target agent files have been read
- [ ] Each embedded pattern is tagged and listed
- [ ] Existing skill references in TOOLS.md are noted (to avoid re-extracting)

**Artifacts:**
- Candidate inventory (raw list of embedded patterns with source references)

---

### **Phase 2: Classify**

**Entry criteria:** Candidate inventory exists from Phase 1.

**Actions:**
1. **Apply the Extraction Test** (Section 4) to each candidate — score each question yes/no
2. **Discard candidates scoring <3** — mark as "leave in agent file" with brief rationale
3. **For candidates scoring ≥3**, classify using the Classification Table:

| Signal in AGENTS.md | Extraction Type | Skill Category |
|---|---|---|
| Multi-phase workflow with delegation points | Pipeline skill | practice (pipeline) |
| Decision framework with structured criteria | Practice skill | practice |
| Decision table mapping inputs to actions | Practice skill or CLI candidate | practice |
| Communication format used by multiple agents | Practice skill | practice |
| Single-agent methodology (no delegation) | Leaf practice skill | practice |
| Deterministic routing logic (no judgment needed) | CLI promotion candidate | n/a — route to `leader-research-analyze-plan` |

4. **Run the "Already Covered" Check** (Section 4) for each extractable candidate
5. **Identify cross-agent patterns** — same methodology embedded in 2+ agents means higher extraction value
6. **Group related candidates** — patterns that would naturally combine into a single skill

**Exit criteria:**
- [ ] Each candidate is classified as: extractable (with type) or "leave in agent"
- [ ] Overlap with existing skills is documented
- [ ] Cross-agent patterns are identified and grouped

**Artifacts:**
- Classified candidate register (each candidate: classification, extraction test score, existing skill overlap, cross-agent occurrences)

---

### **Phase 3: Prioritize**

**Entry criteria:** Classified candidate register exists.

**Actions:**
1. **Score each extractable candidate** using the compound impact formula:

   `compound_impact = (agents_who_benefit × promotion_potential) - extraction_cost`

   | Factor | Scale | Guidance |
   |---|---|---|
   | `agents_who_benefit` | 1-8 | Count of agents with the same or similar embedded pattern |
   | `promotion_potential` | 1 (Low), 2 (Medium), 3 (High) | How likely is this to eventually become a CLI capability? Deterministic patterns score High. |
   | `extraction_cost` | 1 (Low), 2 (Medium), 3 (High) | How complex is the extraction? Single work table = Low. Multi-phase pipeline = High. |

2. **Rank candidates** by compound impact score (descending)
3. **Draw the extraction threshold** — candidates above the line are worth extracting now; below are deferred
   - Recommended threshold: `compound_impact ≥ 3` (ensures net-positive effort)
   - Adjust threshold based on available capacity
4. **For deferred candidates**, record a revisit trigger (e.g., "re-evaluate when a third agent needs this pattern")

**Exit criteria:**
- [ ] All candidates scored and ranked
- [ ] Threshold is set with rationale
- [ ] Deferred candidates have revisit triggers

**Artifacts:**
- Prioritized extraction plan (ranked list with scores, threshold line, deferred items with triggers)

---

### **Phase 4: Plan Extraction**

**Entry criteria:** Prioritized extraction plan exists with candidates above threshold.

**Actions:**
1. **For each candidate above threshold:**
   a. **Determine: new skill or extend existing?** — Apply the sprawl check from `docs/agent-system/SKILL_AUTHORING.md`:
      - Does an existing skill already cover this area? → Extend
      - Would adding this exceed the existing skill's scope? → New skill
      - Is this a distinct, reusable mental model? → New skill
   b. **Draft skill scope:**
      - Focus statement (1-2 sentences)
      - Boundaries (what's in scope, what stays in agent file)
      - Which practice skill format to follow (leaf vs. pipeline)
   c. **Define agent file changes:**
      - What lines of AGENTS.md will be replaced with a skill reference
      - What lines of TOOLS.md need the new skill added
      - What must remain in the agent file (identity, coordination points, agent-specific judgment)
   d. **Define verification** using the Shrink Verification checklist (Section 4)

2. **Identify cross-cutting extractions** — candidates that should combine into one skill rather than separate skills
3. **Sequence the extractions** — extract leaf skills before pipeline skills that compose them

**Exit criteria:**
- [ ] Extraction spec exists for each candidate above threshold
- [ ] Cross-cutting extractions are identified
- [ ] Extraction sequence is defined

**Artifacts:**
- Extraction specs (per candidate: skill to create/extend, agent file diff summary, verification checklist)

**Extraction Spec Template:**
```markdown
### Extraction: [Candidate Label]

**Source agents:** [list of agents containing this pattern]
**Extraction type:** [from Classification Table]
**Compound impact score:** [N]

**Skill action:** Create new / Extend [existing-skill-id]
**Proposed skill ID:** [kebab-case-id]
**Focus statement:** [1-2 sentences]

**Agent file changes:**
- [agent-id] AGENTS.md: Replace lines [N-M] (inline [pattern]) with skill reference
- [agent-id] TOOLS.md: Add [skill-id] to Primary Skills

**What stays in agent file:**
- [specific judgment, coordination points, identity-related content]

**Downstream:**
- Author skill using: `skill-authoring-practice`
- Validate with: `skill-validation`
- CLI promotion via: `leader-research-analyze-plan` (when applicable)
```

---

### **3. Convergence Patterns**

#### **The Extraction Test**

Apply to each candidate identified during Audit. Score each question yes (1) or no (0).

- [ ] Is this a repeatable process (not a one-time instruction)?
- [ ] Would another agent benefit from this same methodology?
- [ ] Can it be stated without referencing this specific agent's identity?
- [ ] Does it have decision points (not just "do X")?
- [ ] Could parts of it eventually become CLI capabilities?

**Threshold:** Score ≥3 = extractable. Score <3 = leave in agent file.

#### **The Shrink Verification**

Apply after an extraction is implemented to confirm it worked correctly.

- [ ] Agent AGENTS.md references the skill instead of inlining the methodology
- [ ] Agent AGENTS.md is measurably shorter (record line count before/after)
- [ ] The extracted skill works standalone (another agent could use it without the source agent's context)
- [ ] No functionality was lost (agent's workflow still covers all the same cases)
- [ ] TOOLS.md lists the new skill reference
- [ ] The extracted skill passes `skill-validation`

#### **The "Already Covered" Check**

Apply before creating a new skill for a candidate.

- [ ] `prompt-manager skill list` searched for skills in the same domain
- [ ] Existing skill content reviewed for overlap
- [ ] If partial overlap found: plan to extend existing skill rather than creating new
- [ ] If no overlap: proceed with new skill using `skill-authoring-practice`

#### **Scope Sizing Table**

| Audit Target | Typical Effort | When to Use |
|---|---|---|
| Single agent | Quick — 1 pass through all 4 phases | Specific agent identified as bloated |
| Single team's agents | Moderate — parallel audit, shared classification | Team effectiveness review |
| All leader agents | Full — all 4 phases, expect cross-cutting patterns | Periodic meta-optimization sweep |
| All agents | Large — delegate sub-audits per team to team members | Major ecosystem optimization initiative |

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Extracting everything** | Agent files become empty shells with no identity or judgment | Leave SOUL.md content and agent-specific judgment in the agent file |
| **Extracting agent identity** | Skills are methodology, not personality | SOUL.md defines *who*; skills define *how*. Keep them separate. |
| **Creating one skill per agent** | Defeats the reuse purpose; skill sprawl | Group by methodology, not by source agent. Cross-agent patterns → one skill. |
| **Extracting without checking existing skills** | Creates duplicates that diverge over time | Always run the "Already Covered" check before creating |
| **Extracting deterministic logic into a skill** | Skills are for judgment; deterministic execution belongs in Action contracts over CLIs | Flag as CLI/Action candidate and route to meta-optimization's Action conversion workflow |
| **Not verifying after extraction** | Agent may lose capabilities silently | Always run the Shrink Verification after implementation |
| **Extracting too early** | Pattern appears once, may not generalize | Wait until a pattern appears in ≥2 agents before extracting (unless the methodology is clearly general) |
| **Skipping prioritization** | Low-impact extractions consume effort that could improve high-impact ones | Always score compound impact; respect the threshold |

---

### **5. Boundaries**

This methodology covers **identifying and planning the extraction of embedded agent capabilities into reusable skills**.

**Does NOT cover:**
- **Actually authoring the extracted skill** — Use `skill-authoring-practice` (and the appropriate category-specific guide: `skill-authoring-practice` for practice skills, `skill-authoring-tools` for tools skills, etc.)
- **CLI/Action promotion of skills** — Use meta-optimization's Action conversion workflow (that pipeline handles `Skills -> CLI -> Action`)
- **Agent personality/identity design** — SOUL.md is not extractable; it defines agent character
- **Skill validation after creation** — Use `skill-validation` to verify the extracted skill meets quality bars
- **Skill improvement** — Use `skill-improvement-suggestions` for existing skills that need optimization

---

### **6. Output Expectations**

When applying Capability Extraction, you **must** produce:

1. **Candidate inventory** — Raw list of embedded patterns with source agent, pattern type, and line count
2. **Classified candidate register** — Each candidate scored, classified, and checked against existing skills
3. **Prioritized extraction plan** — Ranked by compound impact with threshold and deferred items

You **should** also produce:

4. **Extraction specs** — For each candidate above threshold: skill to create/extend, agent file changes, verification plan

**Quality bar:** Another agent should be able to create the extracted skills from the extraction specs alone, using `skill-authoring-practice`, without re-reading the original agent files.

You **must NOT:**
- Extract SOUL.md content (identity is not methodology)
- Create skills that only one agent would ever use (unless the methodology is clearly generalizable)
- Skip the "Already Covered" check
- Implement the extraction without running the Shrink Verification afterward
