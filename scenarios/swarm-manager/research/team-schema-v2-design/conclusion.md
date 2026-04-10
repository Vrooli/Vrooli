# Research Conclusion: Team Schema v2 With Organizational Properties

## Research Question
What structured data should move from freeform TEAM.md into team.json v2 fields (objectives, kpis, consumers, contributions), what should remain as narrative, how should dependencies be computed, and what does the migration path look like?

## Summary
All 6 existing teams (of 8 listed) have been audited. The v2 schema should add 4 new array fields: `objectives`, `kpis`, `consumers`, and `contributions`. Dependencies should be computed (not stored) via a new CLI/API command that derives them from HEARTBEAT.md skill references, CLI tool usage, and cross-team coordination sections. TEAM.md retains operating procedures, workflows, methodology, brand voice, quality rubrics, and skill references — all inherently freeform "how we work" content. Migration is straightforward: add new fields with empty-array defaults, bump `schemaVersion` to 2, and have the prompt builder render v2 fields as natural language in heartbeat prompts.

## Methodology
- Read all team.json, TEAM.md, HEARTBEAT.md, RESPONSIBILITIES.md, org.json, and roles.json for each of the 6 existing teams
- Reviewed the Go `Team` struct in `models.go` and `BaseEntity` in `common.go`
- Reviewed the prompt builder in `prompt_builder.go` to understand how team.json fields are consumed
- Cross-referenced the orchestration summary for prior decisions

## Findings

### Finding 1: Only 6 of 8 Teams Exist on Disk
The spec lists 8 teams but `scenario-debug` and `scenario-refactor` do not exist under `scenarios/prompt-manager/store/teams/`. Only these 6 exist:
- **director-swarm** (enabled, single-process, approval mode)
- **marketing-crew** (disabled, single-process)
- **revenue-research** (disabled, multi-process)
- **meta-optimization** (disabled, multi-process)
- **scenario-feature** (disabled, multi-process)
- **scenario-qa** (enabled, multi-process)

### Finding 2: Current Go Model and Schema Versioning
The `Team` struct (`store/models.go:85-97`) embeds `BaseEntity` which has `Kind` and `SchemaVersion` fields. A global constant `CurrentSchemaVersion = 1` exists in `common.go:64`. The v2 migration needs to:
1. Add new fields to the `Team` struct
2. Bump `CurrentSchemaVersion` to 2 (or handle per-entity versioning)
3. Handle reading v1 files gracefully (missing fields → empty arrays)

The `Team` struct currently has: `ID`, `DisplayName`, `Mission`, `Enabled`, `SpawnMode`, `DecisionMode`, `Shared`, `Retention`, plus `BaseEntity` and `Timestamps`.

### Finding 3: Prompt Builder Usage Is Minimal
`prompt_builder.go` only reads `SpawnMode` (line 288-289, to select coordination skill variant) and `DecisionMode` (line 302, to add approval mode constraints) from the Team struct. `DisplayName` is used for labeling (line 315). All rich context comes from TEAM.md as freeform markdown. The v2 fields should be rendered by the prompt builder as natural language sections in the heartbeat prompt, giving agents readable briefings rather than raw JSON.

### Finding 4: Draft v2 Field Values Per Team

#### director-swarm
- **objectives**: ["Maintain initiative portfolio health with >80% items having clear status", "Ensure all active initiatives have acceptance criteria before execution", "Achieve first initiative completion"]
- **kpis**: [{name: "initiative_completion_rate", target: ">0"}, {name: "decision_turnaround_days", target: "<3"}, {name: "portfolio_items_with_clear_status_pct", target: ">80"}]
- **consumers**: ["marketing-crew", "revenue-research", "meta-optimization", "scenario-feature", "scenario-qa"] (all other teams receive strategic direction)
- **contributions**: [{type: "portfolio-decision", section: "strategy/portfolio", description: "Strategic initiative prioritization and Now/Near/Far decisions"}, {type: "approval-gate", section: "governance/approvals", description: "Human-approved execution authorizations for team deployments and backlog creation"}]

#### marketing-crew
- **objectives**: ["Publish 2+ dev log threads per week", "Maintain consistent brand voice across all content"]
- **kpis**: [{name: "content_published_per_week", target: ">=2"}, {name: "brand_voice_consistency", target: "pass"}]
- **consumers**: ["director-swarm"] (director uses marketing output for external communication context)
- **contributions**: [{type: "dev-log-published", section: "marketing/dev-logs", description: "X/Twitter dev log threads about Vrooli progress"}, {type: "campaign-launched", section: "marketing/campaigns", description: "Marketing campaigns for Vrooli features or milestones"}]

#### revenue-research
- **objectives**: ["Produce 1+ decision-ready opportunity brief per quarter", "Evaluate all feature team outputs for revenue potential"]
- **kpis**: [{name: "briefs_per_quarter", target: ">=1"}, {name: "recommendation_acceptance_rate", target: ">50%"}]
- **consumers**: ["director-swarm"] (director consumes opportunity briefs for portfolio decisions)
- **contributions**: [{type: "opportunity-brief", section: "revenue/opportunities", description: "Decision-ready revenue opportunity assessments with ranked options and trade-offs"}, {type: "market-analysis", section: "revenue/market-intelligence", description: "Competitive landscape and market demand analysis"}]

#### meta-optimization
- **objectives**: ["Maintain zero P1/P2 issues in the capability chain", "Improve average skill health score by 10% per quarter"]
- **kpis**: [{name: "p1_p2_open_issues", target: "0"}, {name: "avg_skill_health", target: ">0.7"}, {name: "orphaned_skills_count", target: "0"}]
- **consumers**: ["director-swarm"] (receives health reports and escalations)
- **contributions**: [{type: "health-report", section: "quality/meta-health", description: "Capability chain health assessments across skills, agents, and teams"}, {type: "skill-improvement", section: "quality/skill-improvements", description: "Optimized skills with before/after impact analysis"}]

#### scenario-feature
- **objectives**: ["Deliver execution-ready backlog items for all approved feature requests", "Maintain 100% definition-of-done compliance"]
- **kpis**: [{name: "backlog_items_execution_ready_pct", target: "100%"}, {name: "definition_of_done_compliance_pct", target: "100%"}]
- **consumers**: ["director-swarm", "scenario-qa"] (QA validates feature output quality)
- **contributions**: [{type: "feature-designed", section: "engineering/features", description: "Execution-ready feature designs with architecture, API contracts, and acceptance criteria"}, {type: "backlog-item-authored", section: "engineering/backlog", description: "Swarm Manager idea/execute items authored from feature requirements"}]

#### scenario-qa
- **objectives**: ["Review all priority scenarios within 24h of queue appearance", "Maintain zero unresolved critical quality findings"]
- **kpis**: [{name: "review_turnaround_hours", target: "<24"}, {name: "critical_findings_unresolved", target: "0"}, {name: "scenarios_reviewed_per_week", target: ">=3"}]
- **consumers**: ["director-swarm", "scenario-feature"] (feature team receives quality gates)
- **contributions**: [{type: "quality-audit", section: "quality/scenario-audits", description: "Dimensional quality assessments (architecture, security, tests, docs) with A-F scoring"}, {type: "deep-audit", section: "quality/deep-audits", description: "Steer-skill-based structural analysis findings with draft execution plans"}]

### Finding 5: What Moves From TEAM.md to Structured Fields
**Moves to team.json v2:**
- Mission statement → already in `mission` field (no change needed)
- Team objectives/goals (currently scattered in TEAM.md as priorities or success criteria)
- KPIs (currently implicit in TEAM.md or not stated)
- Consumer relationships (currently described in "Cross-Team Coordination" sections)
- Contribution types (currently implied by "Deliverables" sections in RESPONSIBILITIES.md)

**Stays in TEAM.md (freeform narrative):**
- Operating procedures and loops (e.g., director-swarm's "Operating Loop" section)
- Decision processes and workflows
- Brand voice guidelines (marketing-crew)
- Quality dimensions and scoring rubrics (scenario-qa)
- Research methodology (revenue-research)
- Priority framework (meta-optimization's P1-P5 waterfall)
- Available skills references
- Charter boundaries and approval constraints
- Team deployment models and org descriptions

### Finding 6: Dependencies Should Be Computed
Per the orchestration summary decision, `dependencies` should NOT be a stored field. Instead, compute them from:
1. **HEARTBEAT.md skill references** — Parse `prompt-manager skill read <id>` references to find shared skills
2. **CLI tool usage** — Parse references to `swarm-manager`, `vrooli-autoheal`, `prompt-manager` commands
3. **Cross-team coordination sections** — Parse "Coordination Points" in RESPONSIBILITIES.md
4. **acceptance_allow paths** — When teams create backlog items targeting other teams' scenarios

A `prompt-manager graph team-dependencies` command (or similar) could derive this on demand.

### Finding 7: Migration Path v1 → v2
1. Add `Objectives`, `KPIs`, `Consumers`, `Contributions` fields to the `Team` struct with `omitempty` JSON tags
2. Update `CurrentSchemaVersion` to 2 (or introduce per-entity versioning if needed)
3. When reading a v1 team.json, treat missing v2 fields as empty arrays (Go zero-value behavior handles this naturally)
4. Write a one-time migration script or chore backlog item that populates the new fields for all 6 teams using the drafted values from Finding 4
5. Update `prompt_builder.go` to render v2 fields as natural language sections
6. Trim TEAM.md content that's now redundant (objectives, cross-team consumer descriptions, deliverable types) — keep freeform narrative

Backward compatibility is straightforward because Go's JSON unmarshaling ignores missing fields (they get zero values). No breaking changes for v1 readers.

## Limitations
- **2 missing teams**: scenario-debug and scenario-refactor don't exist on disk. Their v2 values can't be audited — they'll need to be drafted when created.
- **Draft values need validation**: The objectives, KPIs, consumers, and contributions drafted in Finding 4 are based on TEAM.md analysis. The user should review and adjust them.
- **Dependencies computation not fully specified**: The exact parsing logic for deriving dependencies from HEARTBEAT.md and RESPONSIBILITIES.md needs detailed design (regex patterns, graph traversal).
- **CurrentSchemaVersion is global**: The constant in `common.go` applies to ALL entities, not just teams. Bumping it to 2 affects skills, agents, etc. Per-entity versioning may be needed.

## Actions

### Action 1: Create backlog item — Add v2 fields to Team Go struct and team.json files
- **Kind**: execute
- **Title**: Implement team.json schema v2 with objectives, KPIs, consumers, and contributions
- **Description**: Add 4 new fields to the Team struct in models.go (Objectives []string, KPIs []KPI, Consumers []string, Contributions []Contribution). Define KPI and Contribution sub-structs. Handle v1 backward compatibility (missing fields = empty arrays). Populate all 6 existing team.json files with the drafted values from this research. Bump schema version. Update prompt_builder.go to render v2 fields as natural language in heartbeat prompts.
- **Initiative**: governance-and-tooling (or as appropriate)
- **Priority**: 2
- **Effort**: M

### Action 2: Create backlog item — Trim TEAM.md files after v2 migration
- **Kind**: chore
- **Title**: Remove structured content from TEAM.md files migrated to team.json v2
- **Description**: After v2 fields are populated, remove redundant content from TEAM.md: objectives/goals sections that are now in team.json, cross-team consumer descriptions that are now in `consumers`, deliverable type descriptions that are now in `contributions`. Keep all freeform narrative: operating procedures, workflows, methodologies, brand voice, quality rubrics, charter boundaries, skill references.
- **Priority**: 4
- **Effort**: S
- **depends_on**: [execute/team-schema-v2-implementation]

### Action 3: Create backlog item — Implement computed dependencies command
- **Kind**: execute
- **Title**: Add prompt-manager command to compute team dependencies from file references
- **Description**: Create a `prompt-manager graph team-dependencies` command (or similar) that derives inter-team dependencies by parsing: (1) HEARTBEAT.md skill references, (2) CLI tool usage patterns, (3) RESPONSIBILITIES.md coordination points, (4) acceptance_allow paths from team-created backlog items. Output a dependency graph showing which teams depend on which.
- **Priority**: 4
- **Effort**: M
