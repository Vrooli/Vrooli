---
name: "leader-research-analyze-plan"
description: "Leader-focused pipeline that orchestrates ecosystem capability research, gap analysis with structured decision trees, and cross-cutting implementation planning across skills, scenarios, and CLIs. Composes skill-improvement-suggestions and systematic-exploration into a four-phase workflow for identifying and addressing capability gaps in the Vrooli ecosystem."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","pipeline","leadership","delegation","coordination","methodology","ecosystem","meta-optimization"]
  icon: "search"
  status: "active"
  revision: 1
  createdAt: "2026-02-11T00:00:00Z"
  updatedAt: "2026-02-11T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager graph", "prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Leader Research-Analyze-Plan Pipeline

Orchestrate the full lifecycle of ecosystem capability improvement: research the current state of skills and scenarios in a target area, analyze capability gaps with structured decision trees, decide how to close each gap, and plan cross-cutting implementation across skills, scenarios, CLIs, and registries. This is a four-phase gated pipeline whose distinct value is the gap-analysis decision trees and the ecosystem-integration loop.

Required reading:
- `prompt-manager skill read team-coordination-leader-led` — the shared gated-pipeline contract (phase shape, gate/rework semantics, delegation template, convergence checklists, shared anti-patterns). This skill adds only what is specific to research-analyze-plan.
- `docs/agent-system/SKILL_AUTHORING.md`, `docs/agent-system/PROMOTION_LADDER.md`
- `prompt-manager skill read skill-improvement-suggestions` — the Research-phase leaf.

Optional: `prompt-manager skill read conversation-friction-analysis capability-extraction cli-steer interoperability-steer skill-authoring-practice systematic-exploration visited-tracker-tools`

---

### 1. Phase-to-leaf mapping

| Phase | Leaf skill / activity | Artifact |
|---|---|---|
| Research | `skill-improvement-suggestions` + `systematic-exploration` (delegated per skill/scenario) | Synthesized research summary + per-target reports |
| Analyze | Gap extraction + classification | Capability Gap Register |
| Decide | The four decision trees below | Decision Records |
| Plan | Sequencing + ecosystem-integration loop | Implementation Roadmap + Ecosystem Impact Assessment |

Run the phases per the shared contract's phase shape (Gate 1 after Research, Gate 2 after Analyze, Gate 3 after Decide).

### 2. Pipeline entry work table

| You have... | Area understood? | Gaps identified? | Entry point |
|---|---|---|---|
| New research direction (e.g., "UI reliability") | No | No | Phase 1: Research |
| Friction / graph-health report from a prior run | Partially | Partially | Phase 1: Research (use it as input) |
| Known gaps, decisions needed | Yes | Yes | Phase 2: Analyze |
| Gap register exists, not yet decided | Yes | Yes | Phase 3: Decide |
| Decision records exist, need a roadmap | Yes | Yes | Phase 4: Plan |
| Single skill needs improvement | N/A | N/A | Not this pipeline — `skill-improvement-suggestions` directly |
| Single conversation needs friction analysis | N/A | N/A | Not this pipeline — `conversation-friction-analysis` directly |
| Known bug with known fix | N/A | N/A | Not this pipeline — `leader-triage-investigate-resolve` |
| Specific feature to build | N/A | N/A | Not this pipeline — `leader-explore-plan-implement` |

### 3. Research phase — input classification

Classify the input, load the conditional reading, then scope the research area with graph queries (`prompt-manager graph health --type skill|agent|team`, `orphaned-skills`, `skillless-agents`, `cliless-skills`, `circular-refs`, `popular --type skill`) and delegate `skill-improvement-suggestions` per skill and `systematic-exploration` per scenario.

| Input type | Conditional reading | Research focus |
|---|---|---|
| Friction analysis report | `conversation-friction-analysis` | Extract friction events; research each root-cause layer |
| Performance observation | None | Performance-related skills and scenario monitoring |
| CLI structure concern | `cli-steer` | Affected scenario CLIs and cli-core patterns |
| UI reliability concern | `ui-health` | UI-facing skills and deployment-context coverage |
| Skill coverage question | `skill-authoring-practice` | Existing skills in the target domain |
| Cross-scenario integration issue | `interoperability-steer` | Scenario boundaries and integration points |
| Agent capability audit | `capability-extraction` | Target agents' AGENTS.md for extractable methodologies |
| Graph health / structural data | None | Graph queries to find low-health and structurally deficient entities |
| Generic research direction | None | Broad survey of skills and scenarios in the area |

### 4. Gate criteria

- **Gate 1 (Research → Analyze):** every relevant skill surveyed (improvement report or explicit skip); every relevant scenario CLI explored (findings or explicit skip); input artifacts processed; a research summary synthesizes across delegates; another agent could identify gaps from it alone.
- **Gate 2 (Analyze → Decide):** the Gap Register exists with every gap classified and prioritized; each gap has evidence traceable to research; ecosystem breadth is assessed per gap.
- **Gate 3 (Decide → Plan):** every prioritized gap has a Decision Record naming its tree branch and rationale; cross-cutting decisions are grouped; ecosystem integration needs and prose-retirement candidates are identified (per `PROMOTION_LADDER.md`).

### 5. Analyze phase — gap classification

Extract gap signals, classify each, assess ecosystem impact with `prompt-manager graph popular` / `graph node <id>` (incoming edges = breadth), and prioritize with `priority = (impact × breadth) − implementation_cost`.

| Signal | Gap type |
|---|---|
| Skill says "manually check X" repeatedly | Missing automation |
| Skill covers area but guidance is wrong/stale | Skill quality deficit |
| No skill covers this methodology | Missing skill |
| Scenario CLI lacks a command for a common operation | Missing CLI capability |
| Scenario has capability but does not expose it to the ecosystem | Missing integration |
| Multiple scenarios duplicate the same logic | Missing shared capability |
| Skill references a tool that does not exist | Skill-tooling gap |
| Agent AGENTS.md inlines methodology used by multiple agents | Embedded agent capability |

**Capability Gap Register** records per gap: type, affected skills/scenarios, evidence quote, impact (1-5), breadth (1-5), estimated cost (1-5), priority score, and an overall priority ranking.

### 6. Decide phase — decision trees

Walk the relevant tree per gap in priority order and record the branch taken in a Decision Record.

**Tree 1 — new skill vs. improve existing:**
```
Existing skill covers this area?
├── YES → gap is a missing methodology (how-to)?
│   ├── YES → adding it exceeds the skill's scope? ── YES → CREATE NEW SKILL / NO → IMPROVE EXISTING
│   └── NO (quality/wording) → IMPROVE EXISTING
└── NO → recurring, scenario-agnostic methodology?
    ├── YES → CREATE NEW SKILL
    └── NO → scenario-specific? ── YES → scenario docs, not a skill / NO → CREATE NEW SKILL
```

**Tree 2 — adopt/extend scenario vs. create new:**
```
A scenario already OWNS this domain?
├── YES → its CLI has related commands? ── YES → EXTEND CLI / NO → ADD NEW COMMAND GROUP
└── NO → would it benefit from an existing scenario's infrastructure?
    ├── YES → ADD CAPABILITY there
    └── NO → cross-cutting (3+ scenarios)? ── YES → CREATE NEW SCENARIO / NO → add to most-related scenario
```

**Tree 3 — fix vs. create CLI commands:**
```
Gap is a limitation of an existing command?
├── YES → flags/options suffice (and pass the generality test)? ── YES → ADD FLAGS
│         └── NO → name still expresses intent? ── YES → REFACTOR INTERNALS / NO → CREATE NEW COMMAND
└── NO → natural subcommand of an existing group? ── YES → ADD SUBCOMMAND / NO → CREATE NEW COMMAND GROUP
```

**Tree 4 — ecosystem integration strategy:**
```
Should other scenarios discover/use this?
├── quality/compliance check → register as SCENARIO-AUDITOR EXTERNAL RULES
├── shared service other scenarios call → register with DISCOVERY/RESOLUTION
├── CLI tool other agents invoke → ensure it is on PATH via install.sh
└── none → no ecosystem registration
Then: should skills reference it? ── YES → plan SKILL UPDATES + prose retirement / NO → none
```

**Decision Record** records per gap: the decision, which tree/branch and why, affected artifacts (skills/scenarios/CLIs/ecosystem), dependencies on other records, and which skill prose it supersedes.

### 7. Plan phase — sequencing and ecosystem loop

Group decisions into dependency-ordered waves and sequence within each:

| Dependency pattern | Sequence |
|---|---|
| New scenario needed before its CLI commands | Create the scenario first |
| CLI commands needed before a skill references them | Implement the CLI first |
| Auditor rules needed before a skill references automated checks | Implement the rules first |
| Prose retirement depends on tool availability | Implement the tool, then retire the prose |
| Many skills reference one new capability | Implement it once; update all skills |

**The ecosystem-integration loop** is this pipeline's key pattern — a gap is rarely fixed by "just update a skill":

```
1. Identify a manual skill process that could be automated
2. Find the RIGHT scenario to house it (or create one) — Tree 2
3. Implement the capability (CLI/API) — cli-steer, api-steer
4. Register it with the ecosystem — Tree 4
5. Update the skill to call the new CLI instead of manual prose
6. Retire the superseded prose — PROMOTION_LADDER.md (Collapse/Delete)
```

Every roadmap work item traces this loop; items that skip steps (a CLI with no skill update) leave value on the table.

**Artifacts:** the **Implementation Roadmap** (waves of work items, each naming the gap/record it addresses, its type, its downstream pipeline — `leader-explore-plan-implement` or `skill-authoring-practice` — scope, and objective acceptance criteria; plus deferred items with revisit triggers and a prose-retirement schedule) and the **Ecosystem Impact Assessment** (new capabilities and their home scenarios, ecosystem registrations, skill changes, cross-scenario dependencies, and risks).

### 8. Rework triggers

| Signal | During phase | Action |
|---|---|---|
| "I don't know what skills exist here" | Analyze | Return to Research, scope the skill domain |
| "Unsure which scenario owns this domain" | Decide | Return to Research, explore scenario boundaries |
| "This gap is actually two issues" | Analyze | Split the gap, reclassify each half |
| "The decision tree does not cover this case" | Decide | Return to Analyze, reclassify the gap type |
| "Implementation would break an existing integration" | Plan | Return to Research to understand the integration |
| "Planned work duplicates another scenario" | Plan | Return to Decide with updated context |
| "Priority order changed on new information" | Any | Re-prioritize in Analyze, cascade to Decide and Plan |

### 9. Boundaries and output

Covers leader-orchestrated research and planning for ecosystem capability improvements. Does not cover single-skill analysis (`skill-improvement-suggestions`), single-conversation friction (`conversation-friction-analysis`), implementation of the roadmap (`leader-explore-plan-implement` per work item), bug fixing (`leader-triage-investigate-resolve`), skill-authoring mechanics (`skill-authoring-practice`), or upstream prioritization. Produce the four phase artifacts (or verify a pre-existing one at partial entry). The Roadmap must let a team execute each work item without re-researching or re-deciding; flag any gap that is a symptom of a deeper architectural issue.
