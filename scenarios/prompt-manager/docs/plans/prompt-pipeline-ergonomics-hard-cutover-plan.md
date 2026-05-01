# Prompt Pipeline Ergonomics Hard Cutover Plan

> Supersession note: this is a historical implementation plan. The live heartbeat prompt now uses `team-operating-policy` / `Operating Policy`; `team-operating-contract` and the standalone `team-coordination` prompt section are retired.

## Purpose

Implement a greenfield prompt-pipeline ergonomics cutover for prompt-manager team heartbeats without doing the XML migration yet.

The current team storage ontology cutover succeeded: agents now receive the `Continue / Observe / Propose / Operate` storage model, contract-derived team storage, typed team working state, and friction routing. The next bottleneck is prompt usability. The assembled heartbeat prompt still feels like a stack of source documents instead of a runtime job packet. This plan makes each member heartbeat clearer, more member-aware, and easier to maintain while keeping all current capability.

Target mental model:

```text
Active Task Brief
  -> who am I, what is this run for, what may I write, what must I output?

Context Pack
  -> agent files, team charter, contract, responsibilities, coordination, storage, inbox, prior handoff

Task Source
  -> HEARTBEAT.md as the complete member-authored task spec

Final Task Reminder
  -> concise end-anchor for what to do now
```

## Required Reading

Future implementers must run:

```bash
prompt-manager skill read documentation-health implementation-plan-authoring team-shared-docs-design
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Recommended context:

```bash
prompt-manager skill read team-coordination-independent team-coordination-leader-led team-coordination-peer
prompt-manager skill read conversation-friction-analysis capability-extraction documentation-health
prompt-manager skill read morning-vision-walk
```

Primary files:

- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder.go] - assembled heartbeat prompt sections, storage command rendering, execution brief.
- [CODE: scenarios/prompt-manager/api/heartbeat/prompt_builder_test.go] - section ordering, storage prose, prompt builder tests.
- [CODE: scenarios/prompt-manager/api/teamcontract/contract.go] - operating contract rendering, team storage rendering, plan-of-record grouping.
- [CODE: scenarios/prompt-manager/api/teamcontract/contract_test.go] - contract rendering tests and bundled contract validation.
- [CODE: scenarios/prompt-manager/store/schemas/team.schema.json] - team operating contract schema.
- [CODE: scenarios/prompt-manager/store/teams/*/team.json] - bundled team operating contracts and plan-of-record declarations.
- [CODE: scenarios/prompt-manager/store/teams/*/members/*/HEARTBEAT.md] - member task sources.
- [CODE: scenarios/prompt-manager/store/teams/*/members/*/RESPONSIBILITIES.md] - member role and judgment sources.
- [CODE: docs/director-swarm/] - director-swarm plan-of-record docs; currently lacks a folder hub.
- [CODE: docs/infra-health/README.md] - infra-health plan-of-record hub.
- [CODE: docs/marketing/README.md] - marketing plan-of-record hub.
- [CODE: docs/monetization/README.md] - monetization plan-of-record hub.
- [CODE: docs/narrative/README.md] - narrative plan-of-record hub.
- [CODE: docs/concepts/ARCHITECTURE.md] - root technical plan-of-record reference, currently sketch-level.
- [CODE: VISION.md] - operator-authored project vision.

## Hard Greenfield Rule

This is a hard cutover. Do not add compatibility shims, old/new dual renderers, legacy section aliases, or dead transitional code.

Required behavior:

- Replace `Execution Brief` with `Active Task Brief`.
- Add a final generated `Task Reminder` section after `HEARTBEAT.md`.
- Make storage primitive availability and storage commands member-aware.
- Replace plan-of-record grouped path-count rendering with hub-first navigation.
- Keep current markdown prompt format for this cutover.
- Do not implement XML tags in this cutover.
- Do not cap or truncate previous handoff injection in this cutover.
- Update production code, tests, docs, bundled teams, and prompt previews together.

Prohibited patterns:

- No `buildExecutionBriefSection` compatibility wrapper left behind after the cutover.
- No section kind named `execution-brief` in structured previews.
- No prompts that show write commands a member is forbidden to use.
- No "Always available: decisions" wording that implies every member can write decisions.
- No plan-of-record runtime guidance that lists vague grouped counts such as `7 docs under docs/monetization/` as the primary navigation surface.
- No XML migration, XML tag wrappers, or XML-specific tests in this phase.
- No handoff truncation policy, token cap, summarization, or lossy previous-handoff rewrite in this phase.

## Problem Statement

The storage ontology now gives agents the right primitives, but the full heartbeat prompt still has four practical problems:

1. **The task is not forcefully anchored at the start.** The current `Execution Brief` says the concrete task is defined at the end of the prompt, which is true but weak. Agents have to read a lot of context before the operational frame is fully clear.
2. **Generated storage commands are team-aware but not member-aware.** For example, `vision-walk-prep` forbids `decision` and `task` writes, but its generated storage command section still includes `prompt-manager team decision-add director-swarm ...`.
3. **Primitive availability wording is too broad.** "Always available: decisions" describes the conceptual primitive, but not the active member's allowed write surface.
4. **Plan-of-record navigation is too file-list-shaped.** Large plan-of-record sets are rendered as grouped counts with "Exact paths: see Document Authority above." That is token-efficient but not intuitive. Agents should start from the plan-of-record hub and follow its index/spokes.

The `vision-walk-prep` member makes the ergonomics problem visible. A structured preview showed:

```text
director-swarm/vision-walk-prep sections=11
agent-file SOUL.md                        310 chars
agent-file AGENTS.md                      151 chars
agent-file TOOLS.md                       369 chars
team-shared-charter shared/TEAM.md       1472 chars
execution-brief Execution Brief           510 chars
team-operating-contract                  1781 chars
team-responsibilities                     392 chars
team-coordination                         894 chars
team-storage-map                         4342 chars
last-handoff                            21319 chars
heartbeat-task                            728 chars
```

The long previous handoff is not itself a problem for this cutover. The problem is that the runtime prompt does not clearly frame the task before and after that large context block.

## Scope

In scope:

- Prompt-builder section redesign from `Execution Brief` to `Active Task Brief`.
- Final generated `Task Reminder`.
- Member-aware storage command rendering.
- Member-aware storage primitive wording.
- Prompt constants/template extraction to reduce test drift and string duplication.
- Plan-of-record hub/spoke audit and cleanup for all plan-of-record locations declared by bundled teams.
- Team contract updates to point plan-of-record guidance at hubs where the docs are self-navigating.
- Vision Walk Prep heartbeat purpose strengthening.
- Tests and validation commands covering prompt previews, bundled teams, and plan-of-record hub health.

Out of scope:

- XML prompt migration.
- Handoff truncation, summarization, or cap policy.
- API/CLI permission enforcement.
- Persistence backend changes.
- Decision approval semantics.
- New teams or changed team missions.
- New plan-of-record domains unrelated to currently declared team contract documents.

## Current Technical Context

### Prompt Assembly

`PromptBuilder.buildSectionList` currently assembles full team heartbeat prompts in this order:

```text
agent markdown files
team shared/TEAM.md
Execution Brief
Resolved Operating Contract
RESPONSIBILITIES.md
Team Org Context
Team Coordination
Storage Map
Team Inbox
Previous Handoff
HEARTBEAT.md
```

The flat prompt is still joined with markdown separators:

```text
---
```

Structured previews expose `PromptSection.Kind`, `Label`, `SourcePath`, and `Content`.

### Current Execution Brief

`buildExecutionBriefSection` renders:

```markdown
# Execution Brief

Member: `<agent-id>`
Team: `<team-id>` (`<team-name>`)
Lane: `<member-lane>`
Task: `<first heading from HEARTBEAT.md>`

This heartbeat's concrete task is defined at the end of this prompt in `# Heartbeat Task (HEARTBEAT.md)`.

Use the sections below as operating context. If a section conflicts with the heartbeat task, follow the authority order in `# Storage Map`.
```

This is useful but not sufficient as an immediate operating frame.

### Current Storage Command Problem

`buildAvailableStorageCommandsSection(team)` only sees the team. It does not see the active member contract, so it renders team-level capabilities even when the current member cannot use them.

Example from `vision-walk-prep`:

- Contract forbids `decision` and `task`.
- Active storage commands still include `prompt-manager team decision-add director-swarm ...`.

### Current Team Storage Problem

`RenderTeamStorage` ends with:

```markdown
Always available:
- decisions: propose reviewable changes
- knowledge: record structured observations and friction signals
- handoff: preserve next-run continuity
```

This is conceptually true at the team-system level, but it is misleading for members that are read-only or forbidden from decision writes.

For large plan-of-record lists, `RenderTeamStorage` groups by prefix/policy/consumer:

```markdown
- 7 docs under `docs/monetization/`
  Policy: `operator-curated-via-decisions`
  Consumers: `monetization, director-swarm, marketing-crew, landing-page-business-suite`
  Exact paths: see `## Document Authority` above.
```

This is compact but weak as navigation.

### Plan-of-Record Hub Audit

Observed state:

- `docs/monetization/README.md` is a good folder hub. It names the five axes, key files, ownership, consumers, and honesty conventions.
- `docs/marketing/README.md` is a good folder hub. It distinguishes plan-of-record from notebook and indexes key files/subfolders.
- `docs/marketing/post-types/README.md` is a strong per-entity hub with lifecycle, doc+skill discipline, decision tree, files, and cross-references.
- `docs/marketing/strategies/README.md` is a good strategy hub.
- `docs/narrative/README.md` is a good narrative canon hub.
- `docs/infra-health/README.md` is acceptable and compact, but can be strengthened with a clearer "start here" and consumer/use-case table.
- `docs/director-swarm/` has `PORTFOLIO_PHILOSOPHY.md`, `ROADMAP.md`, and `OUTCOMES_CHARTER.md`, but no `README.md` hub. This is the clearest hub-and-spokes gap.
- `docs/concepts/ARCHITECTURE.md` explicitly says it is sketch-level. That is acceptable as a known limitation, but plan-of-record prompts should not overstate it as exhaustive.
- `VISION.md` is a root singleton rather than folder-backed plan-of-record. It already includes ownership and narrative cross-reference. It does not need to be forced into a folder for this cutover.

## Target End State

Every full heartbeat prompt should:

1. Start with `# Active Task Brief`, not `# Execution Brief`.
2. State the member, team, mission for this run, primary task, allowed writes, forbidden writes, required observations, and required output at the top.
3. Keep the current context pack in the middle, with cleaner generated storage guidance.
4. Render storage primitives based on the active member's write rules.
5. Render storage commands based on the active member's allowed writes and team capabilities.
6. Render plan-of-record guidance as hub-first navigation.
7. Keep `Previous Handoff` untruncated.
8. End with a generated `# Task Reminder` after `HEARTBEAT.md`.
9. Give `vision-walk-prep` explicit downstream-consumer context for the `morning-vision-walk` skill.
10. Use shared prompt constants/templates so tests do not duplicate long production strings.

## Exact Proposed Section: Active Task Brief

Replace `# Execution Brief` with this generated section.

Template:

```markdown
# Active Task Brief

You are running one prompt-manager heartbeat as `<agent-id>` on `<team-id>` (`<team-name>`).

## Mission This Run

`<member-lane>`

## Primary Task

`<first-heading-from-HEARTBEAT.md-or-default>`

The complete task source is included later in `# Heartbeat Task (HEARTBEAT.md)`. Use this brief to stay oriented while reading the context pack.

## Write Surface

Allowed:
- `<allowed-write-1>`
- `<allowed-write-2>`

Forbidden:
- `<forbidden-write-1>`
- `<forbidden-write-2>`

## Required Memory

Knowledge topics:
- `<required-topic>`

Handoff:
- End with `## HANDOFF` as the final response section when the team requires handoff persistence.

## Operating Rule

If context sections disagree, follow the authority order in `# Storage Map`. If the heartbeat task conflicts with lower-priority context, follow the heartbeat task unless it violates the operator instruction or your write rules.
```

Rendering rules:

- If allowed writes are empty, render `Allowed: none declared`.
- If forbidden writes are empty, render `Forbidden: none declared`.
- Render paths in write rules through the same normalized path logic used by `RenderMember`.
- Render built-in write kinds (`knowledge`, `decision`, `handoff`, `task`) in human-readable form.
- Include required knowledge topics from the active member contract.
- Include decision cap summary when the member can write decisions:
  - `Decision cap: <n> new decisions this heartbeat; skip new decisions when <m> owned-context decisions are already pending.`
- If the member cannot write decisions, say:
  - `Decision writes: not allowed for this member. Review decisions when useful; do not create them.`
- When `includeHeartbeat=false`, render the same section but replace task detail with:
  - `The active heartbeat task is intentionally omitted from member context.`

## Exact Proposed Section: Task Reminder

Append this generated section after `# Heartbeat Task (HEARTBEAT.md)` in full heartbeat prompts only.

Template:

```markdown
# Task Reminder

Run this heartbeat as `<agent-id>` on `<team-id>`.

Focus on: `<first-heading-or-member-lane>`.

Do now:
1. Follow the task loop in `HEARTBEAT.md`.
2. Use only the write surfaces allowed in `# Active Task Brief`.
3. Record observations, friction, decisions, working-state updates, and handoff according to `# Storage Map`.
4. End with `## HANDOFF` when required, then stop.
```

Rendering rules:

- Omit decision language when decisions are not allowed:
  - Replace item 3 with `Record observations, friction, working-state updates, and handoff according to # Storage Map; do not create decisions.`
- Omit working-state update language if the member has no custom working-state write surfaces.
- Omit handoff item if team config does not require handoff.

## Storage Wording Cutover

Replace "Always available" with member-aware wording.

Template:

```markdown
Primitive availability for this member:
- decisions: `<review-only | write-allowed | unavailable>` — `<explanation>`
- knowledge: `<write-allowed | unavailable>` — `<explanation>`
- handoff: `<required | allowed | unavailable>` — `<explanation>`
- task board: `<write-allowed | review-only | unavailable>` — `<explanation>`
```

Recommended explanations:

- decisions review-only: `review pending decisions when useful; do not create decisions from this heartbeat`
- decisions write-allowed: `propose reviewable changes within your owned contexts and caps`
- knowledge write-allowed: `record structured observations and friction signals using required topics`
- handoff required: `preserve next-run continuity with final ## HANDOFF`
- task board write-allowed: `maintain live team work only when your task asks for it`

Do not render a primitive as write-allowed unless the active member contract permits that write kind.

## Storage Command Cutover

Make `buildAvailableStorageCommandsSection` member-aware.

Suggested signature:

```go
func (b *PromptBuilder) buildAvailableStorageCommandsSection(team *store.Team, agentID string) string
```

Or centralize the member policy in a helper first:

```go
type MemberStoragePolicy struct {
    CanWriteDecision bool
    CanWriteKnowledge bool
    RequiresHandoff bool
    CanWriteTask bool
    CanWriteWorkingStatePaths []string
    ForbiddenWriteLabels []string
}
```

Command rendering rules:

- Always allow read/list commands for enabled surfaces unless team capabilities hide that surface.
- Render mutation commands only when the member's allowed writes include the relevant kind/path.
- For read-only decision members, show only:
  - `Review decisions: prompt-manager team decision-list <team-id>`
  - `Decision writes are not allowed for this member.`
- For members with `decision` forbidden or `newDecisionCapPerHeartbeat=0`, do not show `decision-add`.
- For members with `task` forbidden, do not show `task-add` or `task-update`.
- For members without `knowledge`, do not show `knowledge-add`.
- If handoff is required, keep the final-response reminder.

## Plan-of-Record Hub-and-Spokes Cutover

### Principle

Runtime prompts should point agents to plan-of-record hubs, not dump every spoke file. A plan-of-record folder should be self-navigating:

```text
README.md / index doc
  -> what this plan covers
  -> ownership/write rules
  -> decision contexts for changes
  -> consumers
  -> file map / spokes
  -> how agents choose the next file
```

### Contract Rendering Target

Replace grouped path-count rendering with hub-first rows.

Suggested output:

```markdown
Plan of record, read/propose only:
- `docs/monetization/README.md`
  Policy: `operator-curated-via-decisions`
  Consumers: `monetization, director-swarm, marketing-crew`
  Use for: monetization strategy, catalog, tiers, pricing, channels, funnel, revenue lines, telemetry, benchmarks, operator-input guidance
  Navigation: start at the hub and follow its file map to the relevant spoke
- `VISION.md`
  Policy: `read-only`
  Consumers: `director-swarm, marketing-crew, monetization`
  Use for: operator-authored north-star vision; agents may flag drift but do not author it
```

### Data Model Options

Preferred greenfield shape:

```go
type PlanOfRecordDocument struct {
    ID          string    `json:"id"`
    Hub         *PathRef  `json:"hub,omitempty"`
    Paths       []PathRef `json:"paths"`
    WritePolicy string    `json:"writePolicy"`
    Consumers   []string  `json:"consumers,omitempty"`
    Rationale   string    `json:"rationale,omitempty"`
    UseFor      string    `json:"useFor,omitempty"`
}
```

Rules:

- `hub` is required when `paths` has more than one entry.
- If `paths` has exactly one entry, that path may serve as the hub.
- The hub path must exist unless explicitly optional with a reason.
- Every non-hub path must be reachable from the hub by a markdown link or table entry, except machine-readable files such as JSON that are explicitly listed in the hub file map.
- `useFor` is rendered in `Your Team Storage`; if absent, fall back to `rationale` or consumers.

Because this is a hard greenfield cutover, do not preserve the old grouped-count rendering as an alternate path.

### Plan-of-Record Docs to Audit and Update

Required updates:

1. `docs/director-swarm/README.md`
   - Create this hub.
   - Index `PORTFOLIO_PHILOSOPHY.md`, `ROADMAP.md`, `OUTCOMES_CHARTER.md`.
   - Explain relationship to `VISION.md`, `docs/concepts/ARCHITECTURE.md`, and Swarm Manager.
   - State owner/write rules and decision contexts (`initiative-portfolio`, `initiative-proposal`, `vision-update`, outcome contexts as applicable).
   - Explain which member uses which spoke:
     - `portfolio-manager`: philosophy + roadmap.
     - `vision-walk-prep`: all three director docs plus VISION/ARCHITECTURE drift.
     - `outcome-strategist`: outcomes charter once Command Center is real.

2. `docs/infra-health/README.md`
   - Strengthen as a navigational hub.
   - Add a consumer/use-case table:
     - runtime-health-scanner -> reliability targets / instrumentation roadmap
     - platform-code-auditor -> cross-platform ledger / instrumentation roadmap
     - infra-contrarian -> all three, challenge lens
     - director-swarm -> morning walk decisions
   - Keep compact; do not duplicate spokes.

3. `docs/monetization/README.md`
   - Already strong. Add explicit "start here for agents" guidance and a "which file for which question" decision table.
   - Ensure `REVENUE_LINES.md` mentions `revenue-lines/` spokes clearly enough that prompts can point to the hub/index rather than every revenue-line file.
   - Ensure `CHANNELS.md` remains the channel index for `channels/` spokes.

4. `docs/marketing/README.md`
   - Already strong. Add "start here for agents" guidance and clarify that `post-types/README.md`, `post-techniques/README.md`, `strategies/README.md`, and `rich-media/README.md` are sub-hubs.
   - The current team contract includes `docs/marketing/strategies/*` but not `docs/marketing/post-techniques/*` or `rich-media/*`; decide whether the contract should declare sub-hubs only rather than every spoke.

5. `docs/narrative/README.md`
   - Already good. Add a compact "which file answers which question" table if missing after review.

6. `docs/concepts/ARCHITECTURE.md`
   - Keep sketch-level status honest.
   - Ensure prompt rendering says "canonical technical reference, currently sketch-level" rather than implying it is exhaustive.

7. `VISION.md`
   - Leave as root singleton. Do not force a folder migration.
   - Ensure contract rendering describes it as operator-authored, read-only, north-star.

## Vision Walk Prep Strengthening

`vision-walk-prep` should explicitly optimize for the downstream `morning-vision-walk` skill.

Update `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md` to include this purpose near the top:

```markdown
## Downstream Consumer

Your `## HANDOFF` becomes the prep deliverable for the `morning-vision-walk` skill. The later walk agent reads `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md` and uses it to guide a conversational decision and ideation session with the operator.

Optimize for phase-aligned briefing notes: concise enough to skim, specific enough to support decisions, and clearly separated by the handoff headings below. Do not answer the operator's decisions. Prepare the context so the walk agent can ask clearly.
```

Keep the existing handoff shape unless the operator requests a separate redesign.

## Prompt Constants and Test Refactor

The current tests hard-code many production prompt strings. This makes every prose improvement expensive and brittle.

Implement a prompt text seam before or alongside the section changes.

Suggested files:

- `scenarios/prompt-manager/api/heartbeat/prompt_templates.go`
- `scenarios/prompt-manager/api/heartbeat/prompt_templates_test.go`
- Optional shared helper in `teamcontract` for storage primitive labels.

Suggested exported or package-private constants:

```go
const (
    SectionKindActiveTaskBrief = "active-task-brief"
    SectionKindTaskReminder = "task-reminder"
    ActiveTaskBriefHeading = "# Active Task Brief"
    TaskReminderHeading = "# Task Reminder"
    StorageMapHeading = "# Storage Map"
)
```

Guidelines:

- Keep production templates in production code, not test files.
- Tests may assert constants, section kinds, and short invariant phrases.
- Tests should not duplicate full multi-paragraph templates unless intentionally snapshotting a tiny, stable fixture.
- Prefer helper assertions:
  - `assertSectionOrder`
  - `assertSectionKindAbsent`
  - `assertMemberCommandVisible`
  - `assertMemberCommandHidden`
  - `assertPromptDoesNotContainLegacyPhrases`

## Implementation Strategy

### Phase 1 - Prompt Template Seam and Test Harness Cleanup

Owner profile: backend/testability worker.

Files likely touched:

- `api/heartbeat/prompt_builder.go`
- `api/heartbeat/prompt_builder_test.go`
- new `api/heartbeat/prompt_templates.go`
- possibly `api/heartbeat/prompt_test_helpers_test.go`

Tasks:

1. Extract prompt section headings, section kind strings, and repeated invariant phrases into constants/helpers.
2. Add test helper functions for structured section assertions.
3. Update current tests to use constants/helpers without changing behavior yet.
4. Ensure no behavior changes in prompt previews.

Acceptance:

- `go test ./heartbeat` passes.
- Current structured previews still use `execution-brief`.
- Test hard-coded prose duplication is reduced enough that future phases do not require retyping long prompt paragraphs.

### Phase 2 - Active Task Brief Cutover

Owner profile: heartbeat/backend worker.

Files likely touched:

- `api/heartbeat/prompt_builder.go`
- `api/heartbeat/prompt_builder_test.go`
- possibly `api/teamcontract/contract.go` for write-rule formatting helpers.

Tasks:

1. Delete `buildExecutionBriefSection`.
2. Add `buildActiveTaskBriefSection`.
3. Change structured section kind from `execution-brief` to `active-task-brief`.
4. Render member lane, primary task, allowed writes, forbidden writes, required knowledge topics, decision cap summary, and handoff requirement.
5. Update section ordering tests.
6. Add negative test that `# Execution Brief` never appears in full heartbeat prompts.

Acceptance:

- `prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json` shows `active-task-brief`, not `execution-brief`.
- `vision-walk-prep` active brief explicitly says decision writes are not allowed.
- `go test ./heartbeat ./teamcontract` passes.

### Phase 3 - Member-Aware Storage Primitives and Commands

Owner profile: heartbeat/teamcontract worker.

Files likely touched:

- `api/heartbeat/prompt_builder.go`
- `api/teamcontract/contract.go`
- `api/teamcontract/contract_test.go`
- `api/heartbeat/prompt_builder_test.go`

Tasks:

1. Add a member storage policy helper that derives allowed/forbidden built-in write kinds and working-state paths from the active member contract.
2. Replace "Always available" with "Primitive availability for this member".
3. Make available storage commands accept `agentID` or a derived policy.
4. Render mutation commands only when allowed.
5. Keep read/list commands where useful.
6. Add tests for:
   - read-only member sees `decision-list` but not `decision-add`.
   - read-only member sees no task mutation commands.
   - member without knowledge write does not see `knowledge-add`.
   - member with decision write sees `decision-add`.
   - bundled `vision-walk-prep` prompt contains no `decision-add`.

Acceptance:

- `prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json | rg "decision-add|task-add|task-update"` returns no matches.
- `prompt-manager team prompt-preview-structured monetization financial-tracker --json` still shows allowed decision/knowledge commands.
- `go test ./heartbeat ./teamcontract` passes.

### Phase 4 - Plan-of-Record Hub Schema and Rendering Cutover

Owner profile: contract/schema worker.

Files likely touched:

- `api/teamcontract/contract.go`
- `api/teamcontract/contract_test.go`
- `store/schemas/team.schema.json`
- `store/teams/*/team.json`

Tasks:

1. Add `hub` and `useFor` to plan-of-record document schema and Go type.
2. Require `hub` when a plan-of-record declaration has more than one path.
3. Delete grouped-count rendering for plan-of-record docs.
4. Render hub-first rows with `Use for` and `Navigation`.
5. Validate that hub files exist.
6. Validate that multi-path docs have hub reachability where practical:
   - Markdown links for markdown spokes.
   - Explicit table/file-map entries for JSON spokes.
7. Update bundled team contracts:
   - `monetization` plan-of-record entries should point to `docs/monetization/README.md`, `CATALOG.md`, `CHANNELS.md`, `REVENUE_LINES.md`, or other true sub-hubs as appropriate.
   - `marketing-crew` should point to `docs/marketing/README.md`, `docs/marketing/post-types/README.md`, `docs/marketing/strategies/README.md`, `docs/narrative/README.md`, and monetization hubs where read-only.
   - `director-swarm` should use new `docs/director-swarm/README.md` once created, with `VISION.md` and `docs/concepts/ARCHITECTURE.md` as singleton references.
   - `infra-health` should use `docs/infra-health/README.md`.
8. Update tests that currently expect grouped counts.

Acceptance:

- `RenderTeamStorage` never emits `Exact paths: see`.
- `RenderTeamStorage` never emits grouped count rows like `docs under`.
- Bundled contracts validate.
- `go test ./teamcontract` passes.

### Phase 5 - Plan-of-Record Hub Documentation Cleanup

Owner profile: documentation worker.

Files likely touched:

- `docs/director-swarm/README.md` (new)
- `docs/infra-health/README.md`
- `docs/monetization/README.md`
- `docs/marketing/README.md`
- `docs/narrative/README.md`
- `docs/concepts/ARCHITECTURE.md`
- possibly `docs/manifest.json`

Tasks:

1. Create `docs/director-swarm/README.md` as the missing hub.
2. Strengthen the existing hubs according to the Plan-of-Record Docs to Audit and Update section.
3. Keep docs concise; do not duplicate spoke content.
4. Register new docs in `docs/manifest.json` if project-level manifest governs these docs.
5. Ensure every plan-of-record folder has:
   - owner/write rules,
   - consumer list,
   - file map,
   - "which file answers which question" guidance,
   - update path / decision contexts.

Acceptance:

- Every multi-file plan-of-record declared in bundled team contracts has a clear hub.
- Hub links to spokes are valid.
- `rg "docs/director-swarm/README.md" docs/manifest.json scenarios/prompt-manager/store/teams/director-swarm/team.json` shows expected registration/declaration once implemented.

### Phase 6 - Vision Walk Prep Prompt Strengthening

Owner profile: prompt/docs worker.

Files likely touched:

- `store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md`
- possibly `store/teams/director-swarm/members/vision-walk-prep/RESPONSIBILITIES.md`
- `api/heartbeat/prompt_builder_test.go` if prompt preview assertions are added.

Tasks:

1. Add the exact Downstream Consumer section proposed above.
2. Keep the task loop and handoff shape.
3. Avoid adding generic storage prose already generated by the prompt builder.
4. Preview the full prompt and ensure the Active Task Brief plus heartbeat source make the purpose clear.

Acceptance:

- `prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json` includes the downstream consumer text in `heartbeat-task`.
- The prompt makes clear that `last-handoff.md` is the prep deliverable consumed by `morning-vision-walk`.

### Phase 7 - Final Task Reminder

Owner profile: heartbeat/backend worker.

Files likely touched:

- `api/heartbeat/prompt_builder.go`
- `api/heartbeat/prompt_builder_test.go`

Tasks:

1. Add generated `Task Reminder` after `HEARTBEAT.md` for full heartbeat prompts.
2. Do not include it in member-context prompts where `includeHeartbeat=false`.
3. Make reminder member-aware:
   - no decision language when decision writes forbidden,
   - no working-state update language when no applicable write surface,
   - no handoff language when handoff not required.
4. Add structured preview tests for final section ordering.

Acceptance:

- Full prompt section order ends with:
  ```text
  Previous Handoff
  Heartbeat Task
  Task Reminder
  ```
- Member-context endpoint still omits both `HEARTBEAT.md` and `Task Reminder`.
- `go test ./heartbeat` passes.

### Phase 8 - Prompt Matrix Validation and Cleanup

Owner profile: validation worker.

Tasks:

1. Run prompt matrices for all bundled teams:
   ```bash
   prompt-manager team prompt-matrix director-swarm --json
   prompt-manager team prompt-matrix infra-health --json
   prompt-manager team prompt-matrix marketing-crew --json
   prompt-manager team prompt-matrix meta-optimization --json
   prompt-manager team prompt-matrix monetization --json
   prompt-manager team prompt-matrix scenario-qa --json
   ```
2. Verify no prompt contains:
   - `# Execution Brief`
   - `execution-brief`
   - `Always available:`
   - `Exact paths: see`
   - `docs under`
   - `decision-add` for decision-forbidden members
   - `task-add` or `task-update` for task-forbidden members
3. Spot-check full previews:
   ```bash
   prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json
   prompt-manager team prompt-preview-structured meta-optimization team-agent-optimizer --json
   prompt-manager team prompt-preview-structured monetization financial-tracker --json
   prompt-manager team prompt-preview-structured marketing-crew publisher --json
   prompt-manager team prompt-preview-structured infra-health runtime-health-scanner --json
   ```
4. Read the resulting prompts as an agent:
   - Is the run task clear in the first 30 seconds?
   - Are allowed writes obvious?
   - Are forbidden writes obvious?
   - Does plan-of-record navigation point to a hub?
   - Is the final reminder consistent with the contract?

Acceptance:

- All checks pass.
- Any residual confusing prompt is fixed in this cutover rather than documented as follow-up, unless it belongs to XML migration or handoff policy.

## Contract Decisions

### No XML Yet

Do not implement XML wrappers in this plan. XML remains a likely future improvement, but this cutover should first prove the prompt pipeline can render a clean task packet in markdown with better sectioning, member-aware commands, and hub-first docs.

Future XML migration should be a separate plan after this cutover has stable structured sections and template constants.

### No Handoff Cap Yet

Do not truncate or summarize previous handoffs. Token reduction is not the current goal; clarity and correctness are. Long handoffs may improve performance until there is evidence to the contrary.

### Prompt Section Names

Final section kinds:

```text
agent-file
team-shared-charter
active-task-brief
team-operating-contract
team-responsibilities
team-org-context
team-coordination
team-storage-map
team-inbox
last-handoff
heartbeat-task
task-reminder
```

`execution-brief` is deleted.

### Plan-of-Record Contract Semantics

The operating contract should distinguish:

- `hub`: where agents start.
- `paths`: spokes covered by the plan-of-record declaration.
- `useFor`: one-line purpose rendered in prompts.
- `writePolicy`: operator-curated/read-only rules.
- `consumers`: readers.

For singletons such as `VISION.md`, `hub` may equal the singleton path or be omitted if `paths` length is 1.

## Testing Plan

### Unit Tests

Run from `scenarios/prompt-manager/api`:

```bash
go test ./teamcontract ./heartbeat
```

Required test coverage:

- `Active Task Brief` replaces `Execution Brief`.
- Section ordering includes `active-task-brief` near the top.
- `Task Reminder` appears only in full heartbeat prompts.
- Member-aware decision commands.
- Member-aware task commands.
- Member-aware knowledge commands.
- Member-aware primitive availability text.
- Plan-of-record hub rendering.
- Multi-path plan-of-record docs require hubs.
- Grouped-count plan-of-record rendering is absent.
- Bundled team contracts validate.

### Prompt Preview Validation

Run:

```bash
prompt-manager team prompt-preview-structured director-swarm vision-walk-prep --json
prompt-manager team prompt-preview-structured meta-optimization team-agent-optimizer --json
prompt-manager team prompt-preview-structured monetization financial-tracker --json
prompt-manager team prompt-preview-structured marketing-crew publisher --json
prompt-manager team prompt-preview-structured infra-health runtime-health-scanner --json
```

Validate:

- First generated team runtime section is `active-task-brief`.
- Final generated section is `task-reminder`.
- `vision-walk-prep` has no mutation commands it is forbidden to use.
- `vision-walk-prep` heartbeat task mentions the downstream `morning-vision-walk` consumer.
- Plan-of-record rows point to hubs.

### Prompt Matrix Validation

Run all bundled matrices:

```bash
for team in director-swarm infra-health marketing-crew meta-optimization monetization scenario-qa; do
  prompt-manager team prompt-matrix "$team" --json > "/tmp/${team}-prompt-matrix.json"
done
```

Then inspect/grep for prohibited strings.

### Documentation Validation

Suggested commands:

```bash
rg "docs under|Exact paths: see|Execution Brief|Always available:" scenarios/prompt-manager/api scenarios/prompt-manager/store/teams
rg "docs/director-swarm/README.md" docs/manifest.json scenarios/prompt-manager/store/teams/director-swarm/team.json
```

If available:

```bash
knowledge-observatory docs audit docs --json
```

Do not block the cutover on unrelated pre-existing docs audit findings unless they affect the plan-of-record hubs touched by this work.

## Rollout / Validation Checklist

- [ ] Prompt template constants/helpers extracted.
- [ ] `Execution Brief` deleted from production rendering.
- [ ] `Active Task Brief` rendered and tested.
- [ ] Storage primitive availability is member-aware.
- [ ] Storage mutation commands are member-aware.
- [ ] Plan-of-record data model supports hub-first rendering.
- [ ] Grouped count plan-of-record rendering deleted.
- [ ] `docs/director-swarm/README.md` created.
- [ ] Existing plan-of-record hubs strengthened where needed.
- [ ] Bundled team contracts updated to declare hubs/useFor.
- [ ] `vision-walk-prep` downstream consumer purpose added.
- [ ] `Task Reminder` rendered and tested.
- [ ] Full prompt matrices checked for prohibited strings.
- [ ] `go test ./teamcontract ./heartbeat` passes.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Active Task Brief duplicates too much contract text | Keep it summary-oriented: allowed/forbidden writes, required memory, primary task. Full contract remains below. |
| Member-aware command rendering hides useful read commands | Keep read/list commands available unless team capabilities hide the surface; only suppress mutation commands. |
| Hub-first plan-of-record rendering hides specific spokes agents need | Strengthen hubs with decision tables and file maps; member heartbeat prose can name concepts rather than paths. |
| Plan-of-record hub validation becomes too brittle | Validate existence and obvious markdown links first; avoid complex markdown AST requirements in first cutover. |
| Test cleanup becomes a refactor rabbit hole | Phase 1 only extracts prompt constants and assertion helpers needed for this cutover. |
| XML migration pressure re-enters scope | Keep XML explicitly out of scope and add it as a future plan after this cutover validates. |
| Handoff length remains visually dominant | Use stronger first and final task anchors; do not truncate until there is evidence and a separate handoff-policy plan. |

## Non-Goals / Prohibited Patterns

- Do not add XML tags or XML validation.
- Do not cap, truncate, summarize, or rewrite previous handoffs.
- Do not preserve `Execution Brief` under a different label.
- Do not keep grouped plan-of-record path counts.
- Do not render mutation commands forbidden by the active member contract.
- Do not add API/CLI enforcement as part of this cutover.
- Do not create broad docs refactors outside the plan-of-record hubs used by prompt-manager teams.
- Do not change team missions or decision semantics.

## Definition of Done

This plan is complete when:

1. Full heartbeat prompts start with a clear `Active Task Brief`.
2. Full heartbeat prompts end with a concise `Task Reminder`.
3. No generated prompt contains `# Execution Brief`, `execution-brief`, `Always available:`, `Exact paths: see`, or grouped plan-of-record count rows.
4. Read-only/forbidden members do not receive mutation commands for forbidden surfaces.
5. Plan-of-record runtime guidance points to hubs and explains how to navigate to spokes.
6. All declared multi-file plan-of-record surfaces have usable hubs.
7. `vision-walk-prep` explicitly knows its handoff is consumed by the `morning-vision-walk` skill.
8. Prompt text constants/templates reduce duplicated test prose.
9. `go test ./teamcontract ./heartbeat` passes.
10. Prompt matrices for all bundled teams pass the prohibited-string and member-write checks.

## Future Work: XML Prompt Packet Migration

After this cutover proves the markdown prompt packet is coherent, consider a separate XML migration plan.

That future plan should evaluate:

- XML section tags for generated context.
- XML tag naming and escaping.
- How markdown source files live inside XML wrappers.
- Structured preview changes.
- Snapshot/golden tests for tag balance and prompt hierarchy.
- Whether XML improves actual heartbeat outcomes before adopting it permanently.

Do not start XML work until this plan is complete and validated.
