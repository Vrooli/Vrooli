# Heartbeat Middle Context Hard-Cutover Implementation Plan

> Supersession note: this is a historical implementation plan. The live heartbeat prompt no longer renders separate `Team Coordination` or `Resolved Operating Contract` sections; those concerns are consolidated into the generated `Operating Policy` section.

## Purpose

Hard-cut the prompt-manager heartbeat prompt shape and stored team/agent prompt files to reduce duplicated middle-context doctrine while preserving the intentional task sandwich:

- task/job orientation early, so the member can interpret all following context through the active job;
- task reminder late, so the final instruction remains fresh when execution starts.

This plan is intentionally scoped to the prompt **middle payload** and stored prompt-file cleanup. It does not redesign persistent storage ontology, notebook semantics, plan-of-record semantics, or the Storage Map.

## Required Reading

Future implementing agents must run:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health seam-discovery-and-enforcement utils-unification test
```

Useful source files:

- `scenarios/prompt-manager/api/heartbeat/prompt_builder.go`
- `scenarios/prompt-manager/api/heartbeat/prompt_templates.go`
- `scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go`
- `scenarios/prompt-manager/api/teamcontract/contract.go`
- `scenarios/prompt-manager/api/teamcontract/contract_test.go`
- `scenarios/prompt-manager/store/agents/*/{AGENTS.md,TOOLS.md,SOUL.md}`
- `scenarios/prompt-manager/store/teams/*/shared/TEAM.md`
- `scenarios/prompt-manager/store/teams/*/members/*/{RESPONSIBILITIES.md,HEARTBEAT.md}`

Evidence commands used while authoring this plan:

```bash
rg -n "Read SOUL|SOUL.md|Primary Skills|prompt-manager skill read|Members|Coordination Pattern|Living Docs|Notebook|plan-of-record|HEARTBEAT.md|TOOLS.md" scenarios/prompt-manager/store/agents scenarios/prompt-manager/store/teams -g '*.md'
rg -n "buildSectionList|buildActiveTaskBriefSection|buildTaskReminderSection|buildCoordinationSkillSection|buildOrgContextSection|buildStorageMapSection|RenderTeamStorage|RenderMember" scenarios/prompt-manager/api/heartbeat scenarios/prompt-manager/api/teamcontract
find scenarios/prompt-manager/store/teams -path '*/shared/TEAM.md' -print -exec wc -l {} \;
```

## Greenfield Hard Rule

This is a greenfield cutover. Do not add legacy compatibility, old/new prompt modes, feature flags, fallback renderers, duplicate templates, or dead compatibility tests.

When the implementation lands:

- the revised section order is the only heartbeat prompt order;
- the cleaned team/agent file conventions are the only stored prompt-file conventions;
- tests assert the new behavior directly;
- obsolete assertions for the old order or old duplicate text are deleted or rewritten, not preserved behind compatibility branches.

## Problem Statement

The heartbeat prompt currently uses a valid sandwich idea, but the middle contents are noisy. The current full build order in `buildSectionList` is:

1. Agent markdown files
2. Team charter
3. Active Task Brief
4. Resolved Operating Contract
5. Team Responsibilities
6. Team Org Context
7. Team Coordination
8. Storage Map
9. Team Inbox
10. Previous Heartbeat Handoff
11. Heartbeat Task
12. Task Reminder

The deliberate top/bottom task duplication is useful and should remain. The problematic duplication is in the middle:

- `AGENTS.md` often says `Read SOUL.md for identity alignment`, even though `SOUL.md` is already injected in the same prompt.
- `AGENTS.md`, `TOOLS.md`, and `RESPONSIBILITIES.md` often duplicate skill lists.
- `TEAM.md` files often contain member rosters, member file-structure explanations, leaderless coordination doctrine, generic notebook/plan-of-record explanations, and key-skill lists.
- Generated sections already know enough about team mode, org context, storage, allowed writes, forbidden writes, and primitive availability to carry much of that runtime context.
- `meta-optimization/shared/TEAM.md` is the clearest overload: it mixes mission, roster, coordination doctrine, member file layout, operating rules, contrarian framework, notebook doctrine, and cross-team coordination.

The result is a long middle context pack where agents read supporting doctrine before the active job and encounter repeated or irrelevant guidance.

## Scope

In scope:

- Reorder generated heartbeat sections so the top Active Task Brief comes before supporting agent/team files.
- Preserve the final Task Reminder.
- Document the task sandwich rationale in code comments and tests.
- Keep Storage Map content and semantics unchanged.
- Clean stored `AGENTS.md` files so they refer to already-included `SOUL.md` as context, not as a file to read.
- Normalize skill/tool ownership so skills are not listed redundantly across `AGENTS.md`, `TOOLS.md`, and `RESPONSIBILITIES.md`.
- Shrink shared `TEAM.md` files to mission, scope/boundaries, and team-specific principles.
- Remove leaderless roster/coordination/file-structure boilerplate from shared `TEAM.md`.
- Ensure generated org/coordination context is only shown when behaviorally relevant by team mode/capabilities.
- Update tests to assert the new greenfield shape and absence of obsolete prompt text.

Out of scope:

- Any redesign of Storage Map wording, authority order, storage ontology, friction-capture rules, plan-of-record behavior, notebook behavior, or generated storage command content.
- Creating role skills or moving heartbeat loops into skills. That is a follow-up architecture step.
- Changing team operating-contract data schemas unless a narrow field is needed to remove duplication.
- Changing team decision modes, caps, allowed writes, forbidden writes, notebook paths, shared-state paths, or persisted team data semantics.
- Changing agent-manager spawning behavior outside the prompt text it receives from prompt-manager.
- Changing swarm-manager backlog/task semantics.

## Current Technical Context

### Prompt Builder

`buildSectionList` in `scenarios/prompt-manager/api/heartbeat/prompt_builder.go` owns the runtime section sequence. It currently appends agent files and `shared/TEAM.md` before `Active Task Brief`.

`buildActiveTaskBriefSection` already renders:

- active member/team identity;
- lane;
- primary task heading;
- write surface;
- required memory;
- handoff rule;
- decision cap;
- conflict rule pointing at Storage Map.

`buildTaskReminderSection` already provides the final reminder. This is intentionally duplicated with the top brief and should remain.

`buildStorageMapSection` owns the storage ontology, friction capture, authority order, team storage, and available commands. This plan must not alter it.

### Contract Renderer

`teamcontract.RenderMember` in `api/teamcontract/contract.go` emits the full resolved operating contract. It is useful context but overlaps with `Active Task Brief` and `Storage Map`. This plan may reposition it, but should not change its content unless tests require minor heading/order consistency.

`teamcontract.RenderTeamStorage` emits the contract-derived storage details inside Storage Map. Do not redesign it in this plan.

### Stored Prompt Files

Observed examples:

- Most agent `AGENTS.md` files contain `Read SOUL.md for identity alignment.`
- Several agent `AGENTS.md` files and `TOOLS.md` files both list skills.
- `TOOLS.md` files use a `Primary Skills` section for skills and a `Tool Access` section for actual commands.
- Several member `RESPONSIBILITIES.md` files also list skills.
- Shared team charters range from 18 to 71 lines, with `meta-optimization/shared/TEAM.md` at 71 lines and containing multiple layers of doctrine.

## Target End State

### Runtime Section Order

The generated full heartbeat prompt should use this greenfield order:

1. **Active Task Brief**
   Generated. First. Gives the active job, lane, write rules, required memory, and explains that the complete heartbeat source appears later.

2. **Team Inbox**
   Generated, conditional. If messages exist, show them early because they may affect the current job.

3. **Previous Heartbeat Handoff**
   Generated from `last-handoff.md`, conditional. Early enough to inform context reading.

4. **Storage Map**
   Generated. Unchanged content. Kept before writable/readable supporting docs because it defines authority and persistence rules.

5. **Team Org Context**
   Generated, conditional. Present only for leader-led/org-chart/reporting configurations where it changes behavior. Omitted for leaderless/independent teams unless a capability explicitly requires it.

6. **Team Coordination**
   Generated, conditional. Present only when coordination mode changes behavior beyond staying in lane. For leaderless teams, either omit entirely or render only a short machine-derived rule if tests establish it is required. Do not reintroduce hand-written leaderless doctrine.

7. **Resolved Operating Contract**
   Generated. Supporting contract detail for auditability.

8. **Team Responsibilities**
   Human-written member support file. Supporting context, not primary steering.

9. **Team Charter**
   Human-written shared support file. Lean mission/scope/principles only.

10. **Agent Files (Markdown)**
   Human-written agent support files. `SOUL.md`, `AGENTS.md`, `TOOLS.md`, and any notes remain included, but no longer precede the active job.

11. **Heartbeat Task**
   Human-written full task source from `HEARTBEAT.md`.

12. **Task Reminder**
   Generated final instruction. Kept intentionally for the sandwich.

If implementation shows a simpler order is clearly better, the invariant is: `Active Task Brief` first, `Heartbeat Task` near the end, `Task Reminder` last, and Storage Map unchanged.

### Stored File Conventions

`AGENTS.md` target:

```md
# AGENTS

## Runtime Posture
- Use the included `SOUL.md` section as your behavioral baseline.
- Follow the active heartbeat task and generated write contract.
- Treat generated storage, coordination, and operating-contract sections as authoritative over this file.
```

No `AGENTS.md` should say to read `SOUL.md`, `HEARTBEAT.md`, or other files that are already injected into the prompt.

`TOOLS.md` target:

- Keep actual callable commands and non-obvious tool usage.
- Keep a single generic skill command if useful: `prompt-manager skill read <skill-id>`.
- Do not list role-specific primary skills if they are already listed in `HEARTBEAT.md`, `RESPONSIBILITIES.md`, or generated contract context.

`RESPONSIBILITIES.md` target:

- Role-specific responsibilities and required skills may live here when they are truly member-specific.
- Do not duplicate generic agent startup behavior.
- Do not describe other team members unless this member's responsibility requires it.

`TEAM.md` target:

```md
# Meta Optimization Team

## Mission
Apply evolutionary pressure to Vrooli's meta-layer so skills, agents, teams, and tool contracts become cheaper, sharper, more programmatic, and easier to retire when stale.

## Scope
Owns meta-layer optimization: skills, prompt-manager agents, team contracts, prompt surfaces, and run-derived lessons.

Does not own scenario code quality, monetization strategy, or new scenario design.

## Team-Specific Principles
- Prefer usage-grounded changes over aesthetic cleanup.
- Prefer programmatic conversion when repeated prose can become deterministic tooling.
- Proposals need a measurable baseline.
- Pruning is a first-class improvement path.
```

Apply the same pattern to all shared team charters:

- mission;
- scope/boundaries;
- team-specific principles.

Remove:

- member roster;
- member file-structure explanations;
- generic leaderless coordination doctrine;
- key-skill inventory;
- generic notebook/plan-of-record theory;
- cross-team relationship lists unless they are mission-critical boundaries not represented elsewhere.

## Implementation Strategy

### Phase 1 - Prompt-Order Hard Cutover

Edit `api/heartbeat/prompt_builder.go`:

- Move `Active Task Brief` construction to the beginning of team prompt assembly.
- Keep `Task Reminder` last.
- Move supporting files and doctrine after generated contract/storage/memory context.
- Keep `BuildContext` behavior consistent with the same middle order, except it still omits `HEARTBEAT.md`.
- Add a concise code comment near section assembly documenting the sandwich rationale:
  - top brief orients context reading;
  - full task/reminder at the end preserves recency;
  - middle sections must avoid duplicating doctrine that generated sections own.

Do not modify `buildStorageMapSection` or `teamcontract.RenderTeamStorage`.

### Phase 2 - Conditional Coordination Cleanup

Review `teamconfig.ShouldShowOrgContext`, `teamconfig.ShouldInjectInbox`, and `buildCoordinationSkillSection`.

Implement only the minimum needed so generated org/coordination context appears when it changes behavior:

- leader-led/org-chart/reporting members receive generated reporting context;
- leaderless independent members do not receive hand-written roster/leaderless doctrine;
- async inbox or peer trigger guidance remains generated only when those capabilities are enabled.

If the current teamconfig helpers already satisfy this after TEAM.md cleanup, do not add code.

Greenfield constraint: do not add a `legacyCoordinationPrompt` path or compatibility toggle.

### Phase 3 - Agent File Cleanup

Update every `scenarios/prompt-manager/store/agents/*/AGENTS.md`:

- replace `Read SOUL.md for identity alignment.` with wording that acknowledges the included section;
- remove instructions to read included files;
- remove duplicated skill inventories from `AGENTS.md` when skills are listed in `TOOLS.md` or `RESPONSIBILITIES.md`;
- keep only runtime posture and truly agent-specific constraints.

Update every `scenarios/prompt-manager/store/agents/*/TOOLS.md`:

- keep actual tool/CLI commands;
- remove role-specific skill lists that duplicate `RESPONSIBILITIES.md`;
- keep `prompt-manager skill read <skill-id>` as the generic capability if useful;
- preserve domain-specific command examples.

Do not delete `SOUL.md`; it remains injected and remains the behavioral baseline.

### Phase 4 - Team Charter Cleanup

Update all `scenarios/prompt-manager/store/teams/*/shared/TEAM.md` files to the lean pattern.

Priority order:

1. `meta-optimization/shared/TEAM.md`
2. `monetization/shared/TEAM.md`
3. `marketing-crew/shared/TEAM.md`
4. `director-swarm/shared/TEAM.md`
5. `infra-health/shared/TEAM.md`
6. `scenario-qa/shared/TEAM.md`

For each team:

- preserve mission and real ownership boundaries;
- preserve team-specific principles that change behavior;
- remove member rosters unless the team mode makes roster awareness behaviorally required;
- remove leaderless/no-lead boilerplate;
- remove member file-structure explanations;
- remove key-skill lists;
- remove generic storage doctrine.

For `meta-optimization`, ensure `debt-curator` keeps any required deeper notebook/promotion method in its own `RESPONSIBILITIES.md` or `HEARTBEAT.md`; do not force all members to read notebook doctrine through shared `TEAM.md`.

### Phase 5 - Responsibility/Skill Single-Source Cleanup

Audit `RESPONSIBILITIES.md` files and choose one member-specific skill listing surface:

- Prefer `RESPONSIBILITIES.md` for member-specific skill recommendations that explain role method.
- Use `TOOLS.md` for callable commands and generic skill access.
- Use `HEARTBEAT.md` when a skill is mandatory for that exact heartbeat loop.

Remove duplicate skill lists across these files. Do not create new role skills in this plan.

### Phase 6 - Tests And Validation

Update `api/heartbeat/prompt_builder_test.go`:

- assert `# Active Task Brief` is first for team heartbeat prompts;
- assert `# Task Reminder` remains last;
- assert `# Heartbeat Task (HEARTBEAT.md)` appears near the end before Task Reminder;
- assert Storage Map content is still present and unchanged at key anchors:
  - `## Continue`
  - `## Observe`
  - friction capture sentence
  - `## Propose`
  - `## Operate`
  - `## Authority Order`
  - `## Your Team Storage`
  - `Primitive availability for this member:`
  - `## Available Storage Commands`
- assert prompt no longer contains stale instructions such as `Read SOUL.md for identity alignment.` in fixture-created stored agent files where relevant.

Update or add store-content validation tests if an existing test harness covers packaged prompt files. If no such harness exists, add a focused Go test in the prompt-manager scenario that scans store prompt markdown for prohibited phrases:

- `Read SOUL.md`
- `Use the member HEARTBEAT.md` when used as a read instruction rather than a reference to the included task section
- `Each member has a`
- shared `TEAM.md` headings `## Members` for leaderless teams
- shared `TEAM.md` heading `## Key Skills`

Keep this as a content contract test, not a compatibility test.

## Contract Decisions

- The task sandwich is intentional and must be documented. Do not consolidate away `Task Reminder`.
- `SOUL.md` remains injected. Stored instructions may reference the included section but must not tell the agent to read it as an external file.
- `TEAM.md` is not the source of truth for roster, generated coordination behavior, storage ontology, member file layout, or skill inventory.
- Storage Map remains the source of truth for storage ontology and friction routing during this plan.
- Team operating contract remains the source of truth for write permissions, caps, read-only behavior, document declarations, and task parameters.
- This plan does not add schema compatibility or old prompt modes.

## Testing Plan

Run focused tests first:

```bash
cd scenarios/prompt-manager && go test ./api/heartbeat ./api/teamcontract
```

Run broader prompt-manager tests:

```bash
vrooli scenario test prompt-manager
```

If the full scenario test is too slow or environment-dependent, run the targeted Go packages plus any prompt-manager store/content tests added in Phase 6, and document the skipped gate with the exact blocker.

Manual inspection commands:

```bash
rg -n "Read SOUL.md|Each member has a|## Members|## Key Skills|Living Docs Under|Append observations that do not yet warrant decisions" scenarios/prompt-manager/store/agents scenarios/prompt-manager/store/teams -g '*.md'
rg -n "legacy|compat|old prompt|fallback prompt" scenarios/prompt-manager/api/heartbeat scenarios/prompt-manager/api/teamcontract
```

The first command should either return no matches or only explicitly justified matches outside leaderless shared team charters. The second should return no new compatibility code introduced by this work.

## Rollout / Validation Checklist

- [ ] `Active Task Brief` is first in full team heartbeat prompts.
- [ ] `Task Reminder` is last in full team heartbeat prompts.
- [ ] The prompt builder contains a concise sandwich-rationale comment.
- [ ] Storage Map text and semantics are not changed.
- [ ] Agent `AGENTS.md` files no longer instruct agents to read already-included `SOUL.md`.
- [ ] Skill inventories are not duplicated across `AGENTS.md`, `TOOLS.md`, and `RESPONSIBILITIES.md`.
- [ ] Shared `TEAM.md` files are lean mission/scope/principles documents.
- [ ] Leaderless shared `TEAM.md` files do not include rosters or no-lead boilerplate.
- [ ] Tests assert the new section order and prohibited stale text.
- [ ] No compatibility modes, fallback renderers, or dead legacy helpers are introduced.

## Risks And Mitigations

Risk: Removing too much team charter context weakens role performance.

Mitigation: Preserve true mission/scope/principle content and keep member-specific method in `RESPONSIBILITIES.md` / `HEARTBEAT.md`. Do not delete unique behavioral constraints; move them to the correct owner when needed.

Risk: Reordering prompt sections breaks tests or hidden assumptions.

Mitigation: Hard-cut tests to the new order and inspect `Build`, `BuildContext`, and `BuildStructured` together. Do not support old order.

Risk: Skill cleanup removes discoverability.

Mitigation: Keep one source of truth per skill reference: mandatory heartbeat skills in `HEARTBEAT.md`, role-method skills in `RESPONSIBILITIES.md`, generic command access in `TOOLS.md`.

Risk: Agents later try to alter Storage Map as part of this cleanup.

Mitigation: This plan repeatedly marks Storage Map redesign as out of scope. Tests should assert existing Storage Map anchors remain.

## Non-Goals / Prohibited Patterns

Do not:

- edit Storage Map wording, storage ontology, friction routing, authority order, notebook semantics, or plan-of-record semantics;
- remove the task sandwich or merge away the final Task Reminder;
- create role skills or migrate heartbeat loops into skills;
- add feature flags, legacy prompt modes, old/new compatibility switches, fallback renderers, or dead compatibility helpers;
- add migration guides for old prompt shapes;
- create parallel utility functions for prompt section assembly if existing builder functions can be updated directly;
- remove meaningful team-specific principles just because they are prose;
- tell agents to read files already injected into the prompt;
- keep duplicated skill lists in multiple stored prompt files;
- change operating-contract data semantics, decision caps, write permissions, or team storage paths.

## Definition Of Done

- The heartbeat prompt uses the new greenfield section order with `Active Task Brief` first and `Task Reminder` last.
- The task sandwich rationale is documented in code/tests.
- All shared `TEAM.md` files are reduced to lean mission/scope/principles charters.
- Agent `AGENTS.md` and `TOOLS.md` files follow the new single-source conventions.
- Required tests pass:

```bash
cd scenarios/prompt-manager && go test ./api/heartbeat ./api/teamcontract
```

- Scenario validation is attempted:

```bash
vrooli scenario test prompt-manager
```

- Searches confirm no stale prompt instructions or compatibility code were introduced.
- A future agent can execute this plan without needing the original conversation.
