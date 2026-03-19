## Meta focus: Team Tool Mapping

Govern how teams are equipped with scenario-based tool skills, using a lazy evaluation pattern that generates honest demand signals for development prioritization.

Required reading:
- `prompt-manager skill read skill-principles`
- `prompt-manager skill read skill-authoring-tools`

---

### **1. Definitions**

- **Tool skill**: A skill (with `modes[0] = "tools"`) that teaches an agent how to use a specific scenario's capabilities.
- **Lazy evaluation**: Presenting a tool skill without revealing the scenario's implementation status until the agent reads the skill to use it. This prevents availability bias from distorting the agent's task planning.
- **Demand signal**: A structured log entry created by an agent when it discovers a needed tool is not yet operational. Captures what was needed, why, and how important it was.
- **Tool mapping**: The process of identifying which scenarios should serve as tools for a team, writing tool skills for them, and assigning those skills to the appropriate team members.

---

### **2. Why Lazy Evaluation Matters**

```
WITHOUT lazy evaluation:              WITH lazy evaluation:

Agent reads prompt:                   Agent reads prompt:
"Tool X is not available"             "Skill X available"
         │                                     │
         ▼                                     ▼
Agent avoids planning                 Agent decides Task T
around Tool X entirely                needs Tool X
         │                                     │
         ▼                                     ▼
No demand signal generated            Agent reads skill,
(silent loss of information)          discovers status
                                               │
                                               ▼
                                      Agent logs structured
                                      demand signal for Tool X
                                               │
                                               ▼
                                      Signal informs build
                                      priority decisions
```

When agents know a tool is unavailable upfront, they route around it — and you never learn what they would have used it for. Lazy evaluation forces agents to commit to needing a tool based on the task, then discover its status. This produces honest demand signals that reflect actual need, not filtered-by-availability need.

---

### **3. When to Apply This Process**

| Trigger | Entry Point |
|---------|-------------|
| New team created | Full mapping (Phase 1-4) |
| New scenario completed that could serve as a tool | Phase 2 — check which teams would benefit |
| Periodic meta-optimization sweep (P4/P5) | Phase 1 — audit all teams for missing tools |
| Team reports low effectiveness or blocked work | Phase 1 — targeted audit of that team |
| Backlog idea created for a new scenario | Phase 3 — write anticipatory tool skill |

```
Is there a new team that needs tools?
  → Full mapping (all phases)
Is there a new scenario that teams could use?
  → Check existing teams, write skill, assign (Phase 2-4)
Is a team underperforming or blocked?
  → Audit that team's tool coverage (Phase 1, then fill gaps)
Is there a planned scenario in the backlog?
  → Write anticipatory skill with lazy evaluation (Phase 3)
None of the above?
  → Periodic sweep: check all teams for tool coverage gaps
```

---

### **4. The Process**

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   PHASE 1    │    │   PHASE 2    │    │   PHASE 3    │    │   PHASE 4    │
│  Audit Team  │ ─▶ │  Map         │ ─▶ │  Write Tool  │ ─▶ │  Assign to   │
│  Needs       │    │  Scenarios   │    │  Skills      │    │  Members     │
│              │    │              │    │              │    │              │
│ (mission,    │    │ (match       │    │ (follow      │    │ (role-based  │
│  workflows,  │    │  existing +  │    │  authoring   │    │  assignment, │
│  gaps)       │    │  planned     │    │  guide +     │    │  no status   │
│              │    │  scenarios)  │    │  lazy eval)  │    │  leakage)    │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
```

---

#### **Phase 1: Audit Team Needs**

**Goal:** Understand what a team needs to accomplish and what tools would make them effective.

**Actions:**
1. Read the team's `TEAM.md` — understand mission, content types, workflows, data sources
2. Read each member's `RESPONSIBILITIES.md` — understand individual duties and deliverables
3. List the team's current skill references — identify what tools they already have
4. Identify workflow steps where a scenario-based tool would help — look for:
   - Manual processes that a scenario could automate
   - Data needs that a scenario's API could serve
   - Content creation steps that a generation tool could support
   - Analysis or research steps that a specialized tool could accelerate

**Exit criteria:**
- [ ] Team mission and workflows understood
- [ ] Current tool coverage documented
- [ ] Gap list: workflow steps that lack tool support

---

#### **Phase 2: Map Scenarios to Gaps**

**Goal:** Match existing and planned scenarios to the team's identified needs.

**Actions:**
1. List scenarios that could fill each identified gap:
   - Check `scenarios/` directory for existing scenarios
   - Check `swarm-manager backlog list --kind idea` for planned scenarios
   - Check archived ideas for shelved concepts that match
2. For each candidate scenario, classify its readiness:

| Readiness | Definition | Skill Approach |
|-----------|-----------|----------------|
| **Operational** | Scenario is running, API/CLI functional | Write full tool skill with real endpoints |
| **Partial** | Scenario exists but features are incomplete | Write tool skill for working features, note planned features |
| **Planned** | Backlog idea exists, not yet built | Write anticipatory skill with lazy evaluation |
| **Unplanned** | No scenario exists, gap identified | Create backlog idea first, then write anticipatory skill |

3. Prioritize by compound impact — scenarios used by multiple teams are higher priority

**Exit criteria:**
- [ ] Each gap has zero or more candidate scenarios mapped
- [ ] Each candidate is classified by readiness
- [ ] Unmapped gaps are flagged for backlog idea creation

---

#### **Phase 3: Write Tool Skills**

**Goal:** Create a tool skill for each mapped scenario, following the lazy evaluation pattern.

**Write skills using `skill-authoring-tools`** with these additional rules:

**For operational/partial scenarios:**
- Write standard tool skills: prerequisites, core workflow, API reference, verification
- For features that are planned but not yet available, add a "Planned Capabilities" section at the end
- Include the standard instruction: "If you need any of these capabilities, log a decision or TODO noting what you needed, which feature would have helped, and why it matters for your current task."

**For planned/unplanned scenarios (the lazy evaluation pattern):**
- Lead with what the tool does — capabilities, use cases, decision tree (sections 1-2)
- Present it as a real tool that exists
- Place the "Operational Status" section after capabilities (section 3), revealing:
  - The scenario is in planning / not yet available
  - Where to find the backlog item (if applicable)
- Follow with "What to Do" instructions (section 4), requiring the agent to log:
  1. What the tool would have been used for
  2. Which capability was needed
  3. Target context (platform, audience, etc.)
  4. Why this tool was the right choice for the task
  5. Priority assessment relative to what the agent could do with available tools

**Critical rule — no status leakage:**
- Tool skills must NOT reveal their operational status in the skill name, description, or `skill.json` metadata
- The `skill.json` description should describe what the tool does, not whether it's available
- Status is only visible inside the SKILL.md content, after the agent has committed to reading it

**Registration:**
- Follow `skill-principles` section 7 for directory structure and metadata
- Use `modes[0] = "tools"`, tags should include the relevant domain (marketing, analytics, etc.)
- Run `prompt-manager skill sync` after creation

---

#### **Phase 4: Assign Skills to Team Members**

**Goal:** Add tool skill references to team member prompts, matched to each member's role.

**Assignment rules:**

| Member's role | Gets skills for |
|---------------|----------------|
| Leads / managers | All tools the team uses (strategic oversight) |
| Creators / implementers | Creation and production tools |
| Researchers / analysts | Analysis, research, and data tools |
| Reviewers / editors | Quality, optimization, and formatting tools |

**How to add skill references:**

Add an "Available Skills" section to each member's `RESPONSIBILITIES.md`:

```markdown
## Available Skills
Read the relevant skill before starting a task. Each skill contains
usage instructions, prerequisites, and current capabilities.

| Skill | Purpose |
|-------|---------|
| `prompt-manager skill read <skill-id>` | Brief purpose description |
```

**Critical rule — no status hints:**
- The member's prompt must NOT indicate which tools are available vs. planned
- All skills are listed the same way — as tools the member can read and use
- The member discovers the tool's status only when they read the skill
- Do NOT add notes like "(planned)", "(not yet available)", or "(coming soon)"

**Also update the team's `TEAM.md`:**
- Add or update the "Available Skills" section listing all team tool skills
- Use the same format: skill read command + brief purpose
- No status indicators

---

### **5. Maintaining Tool Mappings**

Tool mappings are not static. Maintain them through these triggers:

| Event | Action |
|-------|--------|
| Scenario ships (becomes operational) | Update tool skill: replace lazy evaluation sections with real API/CLI docs |
| Scenario adds new features | Update tool skill: move features from "Planned" to core workflow |
| New team member role added | Assess which existing skills apply to the new role |
| Demand signals accumulate for a planned tool | Escalate to director-swarm for build prioritization |
| Scenario is deprecated or archived | Remove or archive the corresponding tool skill, update team refs |

**Demand signal review cadence:**
- During periodic meta-optimization sweeps, review accumulated demand signals across all teams
- Aggregate signals by scenario to identify build priorities
- Report high-demand unbuilt scenarios to director-swarm

---

### **6. Boundaries**

**In scope:**
- Deciding which scenarios should have tool skills
- Writing tool skills following lazy evaluation or standard patterns
- Assigning skills to team members by role
- Maintaining skill-to-team mappings over time
- Reviewing demand signals for prioritization input

**Out of scope:**
- **Building the scenarios themselves** — this process identifies what to build, not how
- **Extracting methodologies from agent files** — use `capability-extraction` for that
- **Improving existing tool skill quality** — use `skill-improvement-suggestions`
- **Deciding strategic build priority** — this process generates demand signals; director-swarm decides priority
- **Writing non-tool skills** (steer, practice, search, meta) — use the appropriate authoring guide

---

### **7. Evolution Rules**

- If the lazy evaluation pattern proves too disruptive (agents spend excessive time on unavailable tools), add a lightweight cost signal without revealing status — e.g., "This tool requires setup time; assess priority before proceeding."
- If demand signal quality is low (vague or missing fields), tighten the logging template in anticipatory skills and re-audit.
- If a team has >10 tool skills, consider grouping related skills or creating a team-specific skill index.
- As the prompt-manager graph matures, prefer graph queries (`prompt-manager graph skillless-agents`, `prompt-manager graph empty-teams`) over manual audits for Phase 1 gap detection.

---

### **8. Output Expectations**

When applying this process, you **must** produce:
1. **Gap analysis** — Team needs vs. current tool coverage
2. **Scenario mapping** — Candidate scenarios matched to gaps, classified by readiness
3. **Tool skills** — Created following `skill-authoring-tools` + lazy evaluation rules
4. **Assignment table** — Which members get which skills, with rationale

You **must NOT:**
- Leak operational status into skill metadata, member prompts, or team docs
- Create tool skills for scenarios that have no backlog idea and no implementation (create the backlog idea first)
- Assign all skills to all members indiscriminately — match skills to roles
- Skip the "Available Skills" table format in member prompts (consistency matters for agent parsing)
