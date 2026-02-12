## Practice focus: Leader Research-Analyze-Plan Pipeline

Orchestrate the **full lifecycle of ecosystem capability improvement**: research the current state of skills and scenarios in a target area, analyze capability gaps with structured decision trees, and plan cross-cutting implementation that spans skills, scenarios, CLIs, and ecosystem registries. This pipeline composes leaf skills (`skill-improvement-suggestions`, `systematic-exploration`) into a leader workflow with explicit phase gates, decision frameworks, and convergence patterns for the unique challenge of improving the Vrooli skill/scenario ecosystem itself.

Required reading:
- `prompt-manager skill read skill-principles`
- `prompt-manager skill read skill-improvement-suggestions`

Optional reading:
- `prompt-manager skill read conversation-friction-analysis`
- `prompt-manager skill read capability-extraction`
- `prompt-manager skill read cli-steer`
- `prompt-manager skill read interoperability-steer`
- `prompt-manager skill read skill-authoring-practice`
- `prompt-manager skill read systematic-exploration`
- `prompt-manager skill read visited-tracker-tools`

---

### **1. When to Use This Pipeline**

#### **Pipeline Entry Decision Table**

| You have... | Area understood? | Gaps identified? | Entry point |
|---|---|---|---|
| New research direction (e.g., "UI reliability", "CLI structure") | No | No | **Phase 1: Research** |
| Friction analysis report from a prior conversation | Partially | Partially | **Phase 1: Research** (use report as input) |
| Known gaps, decisions needed on how to address | Yes | Yes | **Phase 2: Analyze** |
| Gap register exists, implementation not yet planned | Yes | Yes | **Phase 3: Decide** |
| Decision records exist, need implementation roadmap | Yes | Yes | **Phase 4: Plan** |
| Single skill needs improvement suggestions | N/A | N/A | Do not use this pipeline — use `skill-improvement-suggestions` directly |
| Single conversation needs friction analysis | N/A | N/A | Do not use this pipeline — use `conversation-friction-analysis` directly |
| Known bug with known fix | N/A | N/A | Do not use this pipeline — use `leader-triage-investigate-resolve` |
| Graph data shows structural issues (orphans, cycles, skillless agents) | Partially | Partially | **Phase 1: Research** (use graph queries as input) |
| Specific feature to build | N/A | N/A | Do not use this pipeline — use `leader-explore-plan-implement` |

#### **When NOT to use this pipeline**

- **Single-skill improvement** — Use `skill-improvement-suggestions` directly without pipeline overhead.
- **Single-conversation friction analysis** — Use `conversation-friction-analysis` directly.
- **Known implementation work** — Use `leader-explore-plan-implement` (different intent: executing known work vs. discovering what work to do).
- **Bug fixing** — Use `leader-triage-investigate-resolve` (different methodology).
- **Creative brainstorming** — Use the `explore` steer skill (different intent: divergent thinking).
- **Skill authoring** — Use `skill-authoring-practice` (this pipeline decides what to create; authoring guides decide how to create it).

---

### **2. The Process**

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│              LEADER RESEARCH-ANALYZE-PLAN PIPELINE                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌──────────┐ GATE 1 ┌──────────┐ GATE 2 ┌──────────┐ GATE 3 ┌──────────┐    │
│  │ RESEARCH │ ─────▶ │ ANALYZE  │ ─────▶ │ DECIDE   │ ─────▶ │  PLAN    │    │
│  │          │        │          │        │          │        │          │    │
│  │(skill-   │        │(gap      │        │(decision │        │(impl     │    │
│  │ improve  │        │ identif- │        │ trees)   │        │ roadmap) │    │
│  │ + explor-│        │ ication) │        │          │        │          │    │
│  │ ation)   │        │          │        │          │        │          │    │
│  └──────────┘        └──────────┘        └──────────┘        └────┬─────┘    │
│       │                    │                    │                  │          │
│       │  REWORK ◀──────────┘                    │                  │          │
│       │  (analysis reveals research gaps)        │                  │          │
│       │                                         │                  │          │
│       │  REWORK ◀───────────────────────────────┘                  │          │
│       │  (decisions reveal missing context)                         │          │
│       │                                                            │          │
│       │  REWORK ◀──────────────────────────────────────────────────┘          │
│       │  (planning reveals wrong assumptions about ecosystem)                  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Research**

Survey the current state of skills, scenarios, and CLIs in the target area.

**Entry criteria:** A research direction or input artifact exists (research question, friction report, performance observation, CLI concern, etc.).

**Leader actions:**
1. **Classify the input type** using the Input Classification Table (below)
2. **Load conditional reading** based on input type
3. **Scope the research area** using the relationship graph to identify concrete targets:
   - Run `prompt-manager graph health --type skill` / `--type agent` / `--type team` to find lowest-health entities in the target area
   - Run targeted structural queries based on input type: `prompt-manager graph orphaned-skills`, `skillless-agents`, `empty-teams`, `cliless-skills`, `circular-refs`
   - Run `prompt-manager graph popular --type skill --limit 10` to identify high-leverage targets (widely-used but low-health)
   - Use `prompt-manager graph node <id>` to inspect specific entities and their connections
   - Cross-reference graph results with the research direction to select the most relevant entities for investigation
4. **Delegate skill-level analysis** — for each relevant skill, delegate `skill-improvement-suggestions` analysis
5. **Delegate codebase exploration** — for scenario CLIs and capabilities in scope, delegate `systematic-exploration`
6. **If input is a friction report** — extract friction events and their root-cause attributions as research inputs
7. **Collect and synthesize** research findings from all delegates

**Input Classification Table:**

| Input Type | Conditional Reading | Research Focus |
|---|---|---|
| Friction analysis report | `conversation-friction-analysis` | Extract friction events; research each root-cause layer |
| Performance observation | None | Research performance-related skills and scenario monitoring capabilities |
| CLI structure concern | `cli-steer` | Research affected scenario CLIs and cli-core patterns |
| UI reliability concern | `vrooli-ui-interop`, `react-stability` | Research UI-facing skills and deployment-context coverage |
| Skill coverage question | `skill-authoring-practice` | Research existing skills in the target domain |
| Cross-scenario integration issue | `interoperability-steer` | Research scenario boundaries and integration points |
| Agent capability audit | `capability-extraction` | Audit target agents' AGENTS.md for extractable methodologies; use extraction specs as research input |
| Graph health/structural data | None | Use graph queries (`health`, `orphaned-skills`, `skillless-agents`, `cliless-skills`, `circular-refs`) to identify low-health and structurally deficient entities; investigate root causes |
| Generic research direction | None | Broad survey of skills and scenarios in the area |

**Delegation message template (skill analysis):**
```
I need you to analyze [skill-id] for improvement opportunities using the
skill-improvement-suggestions methodology.

Focus area: [research direction]
Context: [why this skill is relevant to the research]

Deliver: Structured improvement report per the skill-improvement-suggestions
output format.

Read the skill first: prompt-manager skill read skill-improvement-suggestions
Then read the target: prompt-manager skill read [skill-id]
```

**Delegation message template (codebase exploration):**
```
I need you to explore [scenario/area] using the systematic-exploration methodology.

Investigation question: What capabilities exist in [area], what patterns are used,
and where are the gaps relative to [research direction]?
Entry points: [files/CLIs/endpoints]
Boundaries: [in-scope / out-of-scope]

Deliver: Findings report answering the investigation question.
Read the skill first: prompt-manager skill read systematic-exploration
```

**Artifacts:**
- Skill improvement reports (per skill analyzed)
- Exploration findings (per scenario/area explored)
- Synthesized research summary

---

### **Gate 1: Research → Analysis**

Before proceeding to Analyze, verify:
- [ ] All relevant skills in the target area have been surveyed (improvement reports exist or explicit skip rationale)
- [ ] All relevant scenario CLIs have been explored (findings reports exist or explicit skip rationale)
- [ ] Input artifacts (friction reports, etc.) have been processed and their signals incorporated
- [ ] Research summary synthesizes findings across all delegates
- [ ] Another agent could identify gaps from the research summary alone

**If gate fails:** Return to Research with a refined scope targeting the specific gap.

---

### **Phase 2: Analyze**

Identify and catalog capability gaps from the research findings.

**Entry criteria:** Gate 1 is satisfied (research summary and individual reports exist).

**Leader actions:**
1. **Extract gap signals** from each research input:
   - From skill-improvement-suggestions reports: tool suggestions, tool improvements, wording issues
   - From exploration findings: missing CLI commands, capability holes, integration gaps
   - From friction reports: repeated manual patterns, missing automation, process friction
2. **Classify each gap** using the Gap Classification Table (below)
3. **Assess ecosystem impact** for each gap — use `prompt-manager graph popular` and `prompt-manager graph node <id>` to quantify how many entities reference the affected node (incoming edge count = breadth score)
4. **Prioritize gaps** using the priority formula: `priority = (impact × breadth) - implementation_cost`
5. **Produce the Capability Gap Register** (artifact format below)

**Gap Classification Table:**

| Signal | Gap Type | Example |
|---|---|---|
| Skill says "manually check X" repeatedly | Missing automation | No CLI command to check interop compliance |
| Skill covers area but guidance is wrong/stale | Skill quality deficit | Outdated CLI command references |
| No skill covers this methodology | Missing skill | No skill for performance investigation |
| Scenario CLI lacks a command for a common operation | Missing CLI capability | No `interop` subcommand in app-monitor |
| Scenario exists but doesn't expose capability to ecosystem | Missing integration | app-monitor has checks but no scenario-auditor external rules |
| Multiple scenarios duplicate the same logic | Missing shared capability | Each scenario implements its own health check pattern |
| Skill references tool that doesn't exist | Skill-tooling gap | Skill assumes CLI command that was never built |
| Agent AGENTS.md contains inline methodology used by multiple agents | Embedded agent capability | Operations-chief's work routing table inlined instead of being a reusable skill |

**Capability Gap Register format:**
```markdown
# Capability Gap Register: [Research Direction]

## Summary
[2-3 sentences: total gaps found, dominant gap types, highest-impact findings]

## Gaps

### GAP-001: [Title]
- Type: [from Gap Classification Table]
- Affected skills: [list]
- Affected scenarios: [list]
- Evidence: [quote from research]
- Impact: [1-5]
- Breadth: [1-5, how many skills/scenarios affected]
- Estimated cost: [1-5]
- Priority score: [impact × breadth - cost]

### GAP-002: [Title]
...

## Priority Ranking
1. GAP-XXX — [reason]
2. GAP-YYY — [reason]
3. ...
```

**Artifacts:** Capability Gap Register

---

### **Gate 2: Analysis → Decision**

Before proceeding to Decide, verify:
- [ ] Capability Gap Register exists with all gaps classified and prioritized
- [ ] Each gap has evidence traceable to research findings
- [ ] Ecosystem impact (breadth) is assessed for each gap
- [ ] No critical research gaps remain (or are explicitly flagged for rework)

**If gate fails:** Return to Research for targeted investigation of the under-researched area.

---

### **Phase 3: Decide**

Apply structured decision trees to determine HOW to address each gap.

**Entry criteria:** Gate 2 is satisfied (Capability Gap Register exists and is prioritized).

**Leader actions:**
1. **For each gap in priority order**, apply the relevant decision tree(s) from below
2. **Record the decision** with rationale for each gap
3. **Identify cross-cutting decisions** — gaps that should be addressed together
4. **Assess ecosystem registration needs** — which new capabilities need to be discoverable by other scenarios/skills?
5. **Produce Decision Records** (artifact format below)

#### **Decision Tree 1: Create New Skill vs. Improve Existing Skill**

```
Gap involves methodology or guidance?
│
├── Does an existing skill already cover this area?
│   │
│   ├── YES → Is the gap a missing methodology (how-to)?
│   │   │
│   │   ├── YES → Does adding it exceed the existing skill's scope?
│   │   │   │
│   │   │   ├── YES → CREATE NEW SKILL (distinct mental model)
│   │   │   └── NO  → IMPROVE EXISTING SKILL (extend coverage)
│   │   │
│   │   └── NO (gap is quality/wording) → IMPROVE EXISTING SKILL
│   │
│   └── NO → Is this a recurring, scenario-agnostic methodology?
│       │
│       ├── YES → CREATE NEW SKILL
│       └── NO  → Consider: Is it scenario-specific guidance?
│           │
│           ├── YES → Add to scenario docs, not a skill
│           └── NO  → CREATE NEW SKILL (fills ecosystem gap)
```

#### **Decision Tree 2: Adopt/Extend Existing Scenario vs. Create New Scenario**

```
Gap involves automation or CLI capability?
│
├── Does a scenario already OWN this domain?
│   │
│   ├── YES → Does the scenario's CLI already have related commands?
│   │   │
│   │   ├── YES → EXTEND EXISTING CLI (add commands/flags)
│   │   └── NO  → ADD NEW COMMAND GROUP to existing scenario CLI
│   │
│   └── NO → Would the capability benefit from an existing scenario's
│       infrastructure (database, API framework, existing integrations)?
│       │
│       ├── YES → ADD CAPABILITY to that scenario (natural fit)
│       └── NO  → Is the capability cross-cutting (touches 3+ scenarios)?
│           │
│           ├── YES → CREATE NEW SCENARIO (new bounded context)
│           └── NO  → Add to the most-related existing scenario
```

#### **Decision Tree 3: Fix Existing CLI Commands vs. Create New Commands**

```
Gap involves CLI behavior?
│
├── Is the gap a limitation of an existing command?
│   │
│   ├── YES → Would adding flags/options suffice?
│   │   │
│   │   ├── YES (and passes generality test) → ADD FLAGS
│   │   └── NO  → Does the command name still express the intent?
│   │       │
│   │       ├── YES → REFACTOR COMMAND internals
│   │       └── NO  → CREATE NEW COMMAND (better expresses intent)
│   │
│   └── NO (command doesn't exist) → Is the operation a natural
│       subcommand of an existing command group?
│       │
│       ├── YES → ADD SUBCOMMAND to existing group
│       └── NO  → CREATE NEW COMMAND GROUP
```

#### **Decision Tree 4: Ecosystem Integration Strategy**

```
New capability created?
│
├── Should other scenarios be able to discover/use this?
│   │
│   ├── YES → Is it a quality/compliance check?
│   │   │
│   │   ├── YES → Register as SCENARIO-AUDITOR EXTERNAL RULES
│   │   └── NO  → Is it a shared service other scenarios call?
│   │       │
│   │       ├── YES → Register with DISCOVERY/RESOLUTION
│   │       │          (api-core discovery pattern)
│   │       └── NO  → Is it a CLI tool other agents invoke?
│   │           │
│   │           ├── YES → Ensure CLI is in PATH via install.sh
│   │           └── NO  → No ecosystem registration needed
│   │
│   └── NO → No ecosystem registration needed
│
├── Should skills reference this new capability?
│   │
│   ├── YES → Plan SKILL UPDATES (add CLI commands, retire prose)
│   └── NO  → No skill updates needed
```

**Decision Record format:**
```markdown
### DR-001: [Decision Title]
- Gap: GAP-XXX
- Decision: [Create new skill / Improve existing / Extend scenario CLI / ...]
- Rationale: [Which decision tree, which branch, why]
- Affected artifacts:
  - Skills: [list of skills to create/update]
  - Scenarios: [list of scenarios to modify]
  - CLIs: [list of CLI changes]
  - Ecosystem: [registration actions]
- Dependencies: [DR-YYY must complete first because...]
- Prose retirement: [which skill sections become superseded]
```

**Artifacts:** Decision Records (one per gap or group of related gaps)

---

### **Gate 3: Decision → Planning**

Before proceeding to Plan, verify:
- [ ] Every prioritized gap has a Decision Record
- [ ] Decision tree branch and rationale are documented for each decision
- [ ] Cross-cutting decisions are identified and grouped
- [ ] Ecosystem integration needs are documented
- [ ] Prose retirement candidates are identified (per skill-principles lifecycle)
- [ ] No decisions require additional research (or rework is flagged)

**If gate fails:** Return to Research (missing context) or Analyze (gap classification unclear).

---

### **Phase 4: Plan**

Produce an implementation roadmap that sequences the work and accounts for ecosystem integration.

**Entry criteria:** Gate 3 is satisfied (Decision Records exist for all prioritized gaps).

**Leader actions:**
1. **Group decisions into implementation waves** based on dependencies
2. **Sequence within each wave** using the Implementation Sequencing Rules (below)
3. **For each work item, identify the appropriate downstream pipeline:**
   - New skill creation → `skill-authoring-practice` + `leader-explore-plan-implement`
   - Scenario CLI changes → `cli-steer` + `leader-explore-plan-implement`
   - Skill improvement → `skill-improvement-suggestions` (apply directly)
   - Ecosystem registration → Part of the scenario implementation
4. **Plan the ecosystem integration loop** for each new capability (see Section 6)
5. **Produce the Implementation Roadmap** (artifact format below)
6. **Produce the Ecosystem Impact Assessment** (artifact format below)

**Implementation Sequencing Rules:**

| Dependency Pattern | Sequence |
|---|---|
| New scenario needed before CLI commands | Create scenario first |
| CLI commands needed before skill can reference them | Implement CLI first |
| Scenario-auditor rules needed before skill references automated checks | Implement rules first |
| Skill prose retirement depends on tool availability | Implement tool, then retire prose |
| Multiple skills reference the same new capability | Implement capability once, update all skills |

**Implementation Roadmap format:**
```markdown
# Implementation Roadmap: [Research Direction]

## Overview
- Research direction: [original input]
- Total gaps: [N]
- Gaps addressed: [M] (remaining deferred with rationale)
- Implementation waves: [W]

## Wave 1: [Theme — e.g., "Foundation capabilities"]
### Work Item 1.1: [Title]
- Addresses: GAP-XXX (DR-YYY)
- Type: [New scenario / CLI extension / Skill creation / Skill update / ...]
- Downstream pipeline: [leader-explore-plan-implement / skill-authoring-practice / ...]
- Estimated scope: [small / medium / large]
- Acceptance criteria: [objective, pass/fail]

### Work Item 1.2: [Title]
...

## Wave 2: [Theme — e.g., "Ecosystem integration"]
...

## Deferred Items
| Gap | Reason for Deferral | Revisit Trigger |
|---|---|---|
| GAP-ZZZ | Low priority / blocked by external dependency | [condition] |

## Skill Prose Retirement Schedule
| Skill | Section | Retirement Action | Trigger |
|---|---|---|---|
| [skill-id] | [section name] | Collapse / Delete | [work item that enables retirement] |
```

**Ecosystem Impact Assessment format:**
```markdown
# Ecosystem Impact Assessment: [Research Direction]

## New Capabilities Added
| Capability | Home Scenario | CLI Commands | Skills Affected |
|---|---|---|---|
| [name] | [scenario] | [commands] | [skills that will reference it] |

## Ecosystem Registrations
| Capability | Registration Type | Target |
|---|---|---|
| [name] | scenario-auditor external rules | [rule IDs] |
| [name] | discovery/resolution | [service name] |

## Skill Ecosystem Changes
| Skill | Change Type | Rationale |
|---|---|---|
| [skill-id] | New / Updated / Prose retired | [why] |

## Cross-Scenario Dependencies
| Source | Depends On | Nature |
|---|---|---|
| [scenario A] | [scenario B] | [what it needs] |

## Risk Assessment
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| [risk] | [H/M/L] | [H/M/L] | [how to handle] |
```

**Artifacts:** Implementation Roadmap, Ecosystem Impact Assessment

---

### **3. Rework Triggers**

Rework is cheaper than planning against wrong assumptions. When signals appear, return to the appropriate phase.

| Signal | During Phase | Action |
|---|---|---|
| "I don't know what skills exist in this area" | Analyze | Return to **Research**, scope the specific skill domain |
| "I'm unsure which scenario owns this domain" | Decide | Return to **Research**, explore scenario boundaries |
| "This gap is actually two separate issues" | Analyze | Split the gap, re-classify each half |
| "The decision tree doesn't cover this case" | Decide | Return to **Analyze** to reclassify the gap type |
| "Implementation would break an existing integration" | Plan | Return to **Research** to understand the integration |
| "Planned work duplicates what another scenario already does" | Plan | Return to **Decide** with updated context |
| "Priority ordering changes based on new information" | Any | Re-prioritize in Analyze, cascade to Decide and Plan |

---

### **4. Convergence Patterns**

#### **Delegation Sufficiency Checklist**

Before delegating any research task to a team member, verify:
- [ ] Team member has the research direction and context
- [ ] Specific deliverable is defined (improvement report / exploration findings)
- [ ] Required reading commands are included
- [ ] Scope boundaries are explicit
- [ ] Expected output format is specified

#### **Pipeline Completion Checklist**

Before declaring the pipeline complete:
- [ ] Research summary exists synthesizing all findings
- [ ] Capability Gap Register exists with classified and prioritized gaps
- [ ] Decision Records exist for all addressed gaps
- [ ] Implementation Roadmap exists with sequenced work items
- [ ] Ecosystem Impact Assessment exists
- [ ] Prose retirement candidates are identified with triggers
- [ ] Deferred items have explicit revisit triggers

#### **Cross-Cutting Concern Detection**

When reviewing gaps and decisions, actively look for:
- Gaps that affect the same scenario (group into one implementation wave)
- Gaps that require the same new capability (implement once, reference many)
- Skill updates that depend on the same CLI change (sequence CLI first)
- Ecosystem registrations that enable multiple downstream improvements

---

### **5. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Jumping to implementation** | Builds the wrong thing; misses ecosystem implications | Complete all four phases before implementing |
| **Analyzing without research** | Gap register is based on assumptions, not evidence | Research first; every gap must have evidence |
| **One gap, one solution** | Misses that a single capability can close multiple gaps | Look for cross-cutting capabilities during Decide |
| **Ignoring ecosystem integration** | New capability exists but nothing references it | Plan the full loop: implement → register → update skills → retire prose |
| **Treating skill updates as the only fix** | Adds prose where automation would be better | Apply the skill-principles lifecycle: prefer tool/CLI promotion over prose |
| **Not retiring superseded prose** | Skills grow forever; token cost increases | Every new capability must include a prose retirement plan |
| **Research scope creep** | Investigating everything instead of the target area | Set boundaries in Phase 1; expand only through deliberate rework |
| **Deciding without decision trees** | Inconsistent decisions; no reviewable rationale | Use the provided decision trees; record the branch taken |
| **Planning without sequencing** | Parallel work that actually has dependencies | Use the Implementation Sequencing Rules |

---

### **6. The Ecosystem Integration Loop**

This is the key pattern this pipeline captures. When a capability gap is identified, the fix is rarely "just update a skill." The full loop is:

```
1. Identify that a manual process in a skill could be automated
              │
              ▼
2. Find the RIGHT scenario to house that automation
   (or create one) — Decision Tree 2
              │
              ▼
3. Implement the capability in the scenario
   (CLI commands, API endpoints) — cli-steer, api-steer
              │
              ▼
4. Register it with the ecosystem
   (scenario-auditor external rules, discovery) — Decision Tree 4
              │
              ▼
5. Update the skill to reference the new CLI capability
   (replace manual instructions with CLI commands)
              │
              ▼
6. Retire superseded prose from the skill
   (skill-principles lifecycle: Collapse or Delete)
```

Every work item in the Implementation Roadmap should trace through this loop. Items that skip steps (e.g., implementing a CLI command without updating skills to reference it) leave value on the table.

---

### **7. Boundaries**

This pipeline covers **leader-orchestrated research and planning for ecosystem capability improvements** that flow through research, analysis, decision-making, and implementation planning.

**Does NOT cover:**
- **Single-skill analysis** — Use `skill-improvement-suggestions` directly
- **Single-conversation friction analysis** — Use `conversation-friction-analysis` directly
- **Implementation of planned work** — This pipeline produces the roadmap; `leader-explore-plan-implement` executes it
- **Bug fixing or incident response** — Use `leader-triage-investigate-resolve`
- **Skill authoring mechanics** — Use `skill-authoring-practice` for how to write skills
- **Strategic prioritization** — This pipeline operates on a research direction given to it; what to research is decided upstream

---

### **8. Output Expectations**

When applying this pipeline, you **must** produce:
1. **From Research phase:** Synthesized research summary with individual skill improvement reports and exploration findings
2. **From Analyze phase:** Capability Gap Register with classified and prioritized gaps
3. **From Decide phase:** Decision Records with rationale traced to decision trees
4. **From Plan phase:** Implementation Roadmap with sequenced work items and Ecosystem Impact Assessment

You **should** also:
- Document which phases were used (full pipeline or partial entry)
- Record any rework loops and what triggered them
- Identify quick wins that can be executed immediately without the full implementation pipeline
- Flag gaps that are symptoms of deeper architectural issues

**Quality bar:** The Implementation Roadmap should enable a team to execute the work using `leader-explore-plan-implement` for each work item, without needing to re-research the area or re-make the architectural decisions. The Ecosystem Impact Assessment should make it clear how the proposed changes affect other skills and scenarios.
