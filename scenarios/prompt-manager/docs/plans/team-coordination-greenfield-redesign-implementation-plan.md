# Prompt Manager Team Coordination Greenfield Redesign Implementation Plan

> Supersession note: this is a historical implementation plan. Coordination skills still exist as supplemental guidance, but the live heartbeat prompt now references them from inside the generated `Operating Policy` section instead of rendering a standalone `Team Coordination` section.

> Created: 2026-04-09  
> Status: Draft (ready for implementation)  
> Scope: `prompt-manager` team model, execution model, prompt assembly, API, CLI, UI, docs, and validation

## Purpose

Redesign Prompt Manager's team coordination system so it cleanly supports:

1. leader-led teams
2. peer-coordinated teams
3. fully independent specialist teams

This redesign must be implemented as a **greenfield replacement**, not an accretion on top of the current `spawnMode`-centric model.

## Required Reading

Run this exact command before implementation:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health api-steer cli-steer interoperability-steer seam-discovery-and-enforcement test react-coherence
```

Then run:

```bash
prompt-manager skill read utils-unification
```

## Hard Constraints (Non-Negotiable)

1. This work is **greenfield**.
2. Do **not** preserve the current leader-coordinator assumptions through compatibility shims.
3. Do **not** implement dual-read or dual-write behavior for old and new team contracts.
4. Do **not** keep dead fields, fallback mappers, implicit lead inference, or transitional prompt assembly logic.
5. Replace the current coordination contract atomically across store schema, API, CLI, UI, docs, fixtures, and tests.
6. If a concept survives the redesign, it must survive because it still belongs in the new architecture, not because it existed before.

## Problem Statement

Prompt Manager currently conflates multiple distinct concerns under `spawnMode` and related heartbeat behavior:

- runtime topology
- coordination style
- lead ownership
- queueing/concurrency policy
- prompt coordination instructions
- runner/profile defaults
- UI explanation of how a team works

As a result:

1. `multi-process` teams are treated as independent in some places but still serialized through a team-wide FIFO queue.
2. member prompts always inject relationship and messaging commands even when a team does not want messaging-based coordination.
3. `single-process` assumes a leader-centered Claude Code orchestration model.
4. lead selection is inferred from the org chart or member order instead of being explicit.
5. the QA team is operationally flat but still encoded as a hierarchy in team data.

This makes the product less efficient, less intuitive, and less maintainable than it should be.

## Scope

### In Scope

- Team domain model redesign
- Store schema redesign
- Prompt assembly redesign
- Execution queue/concurrency redesign
- API contract redesign
- CLI contract redesign
- UI editor/dashboard/prompt-preview redesign
- Documentation and manifest updates
- Fixture/team data updates
- Unit, contract, integration, and UI test updates

### Out of Scope

- unrelated heartbeat feature work
- unrelated 3D world improvements
- non-Prompt-Manager scenario changes except where Prompt Manager contracts are referenced by other scenarios' tests/docs
- backward compatibility for old team contract files or API shapes

## Current Technical Context

The existing implementation is primarily spread across:

- `scenarios/prompt-manager/store/schemas/team.schema.json`
- `scenarios/prompt-manager/api/store/models.go`
- `scenarios/prompt-manager/api/teams/models.go`
- `scenarios/prompt-manager/api/teams/handlers.go`
- `scenarios/prompt-manager/api/heartbeat/handlers.go`
- `scenarios/prompt-manager/api/heartbeat/team_execution.go`
- `scenarios/prompt-manager/api/heartbeat/scheduler.go`
- `scenarios/prompt-manager/api/heartbeat/prompt_builder.go`
- `scenarios/prompt-manager/api/interop/claude_code.go`
- `scenarios/prompt-manager/store/skills/packs/core/team-coordination-multi-process/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/team-coordination-single-process/SKILL.md`
- `scenarios/prompt-manager/ui/src/lib/schemas/team.schema.ts`
- `scenarios/prompt-manager/ui/src/services/teamService.ts`
- `scenarios/prompt-manager/ui/src/services/heartbeatService.ts`
- `scenarios/prompt-manager/ui/src/components/editor/teamTabs/TeamDashboardTab.tsx`
- `scenarios/prompt-manager/ui/src/components/editor/MemberDetailPanel.tsx`
- `scenarios/prompt-manager/ui/src/components/editor/MemberPromptPreview.tsx`
- `scenarios/prompt-manager/ui/src/components/editor/teamTabs/TeamPromptMatrixTab.tsx`
- `scenarios/prompt-manager/ui/src/hooks/useTeamActivity.ts`
- `scenarios/prompt-manager/ui/src/stores/teamActivityStore.ts`

Bundled team data that must be rewritten to match the new model includes:

- `scenarios/prompt-manager/store/teams/director-swarm/*`
- `scenarios/prompt-manager/store/teams/scenario-qa/*`
- `scenarios/prompt-manager/store/relations/team-member/*`

## Target End State

After implementation, Prompt Manager should treat team behavior as the composition of **separate, explicit policies** rather than one overloaded mode flag.

Every team should have first-class configuration for:

1. runtime topology
2. coordination pattern
3. concurrency policy
4. coordination capabilities
5. prompt assembly behavior
6. explicit leader identity where required

The product must make it obvious to users why a team behaves the way it does, and the code must reflect those concepts directly.

## Canonical Contract Decisions

### 1. Split Runtime, Coordination, and Concurrency

Replace the current `spawnMode`-centric contract with explicit sections.

Recommended shape:

```json
{
  "runtime": {
    "mode": "multi-process" | "single-process"
  },
  "coordination": {
    "pattern": "independent" | "peer" | "leader-led",
    "leadAgentId": "optional-but-required-for-leader-led",
    "reportingMode": "none" | "org-chart" | "leader",
    "messagingMode": "disabled" | "async-inbox" | "in-session",
    "capabilities": {
      "showOrgContext": true,
      "injectInbox": false,
      "allowPeerTriggers": false,
      "showTaskBoardGuidance": false,
      "showDecisionLogGuidance": false,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": {
    "queuePolicy": "serialized" | "bounded-parallel",
    "maxConcurrentRuns": 1
  }
}
```

Notes:

- `decisionMode` may remain as a separate top-level governance control.
- `orgChart` remains a relationship model, but it becomes optional and non-authoritative unless the coordination pattern requires it.

### 2. Explicit Lead Identity

- `leader-led` requires `coordination.leadAgentId`.
- `independent` and `peer` forbid `leadAgentId`.
- remove runtime lead inference from org chart root detection and "first member wins" behavior.

### 3. Policy-Driven Prompt Assembly

Prompt sections must be composed by policy, not by `spawnMode`.

Separate the following concerns into individually selectable sections:

- team charter
- responsibilities
- org context
- coordination skill/guidance
- inbox
- last handoff
- heartbeat task
- durable-state guidance

### 4. Policy-Driven Coordination Guidance

Replace the current two-skill model with three explicit coordination skills:

- `team-coordination-independent`
- `team-coordination-peer`
- `team-coordination-leader-led`

The active skill is resolved from `coordination.pattern`, not runtime mode alone.

### 5. Policy-Driven Execution

- `serialized` preserves the current single-active-run behavior.
- `bounded-parallel` allows up to `maxConcurrentRuns` active members per team.
- dedupe remains per member, not per team.

### 6. No Hidden Defaults That Recreate Legacy Behavior

Defaults must be explicit and documented. Avoid defaults like:

- "if no lead is set, use first member"
- "if org chart exists, treat it as authority"
- "if runtime is single-process, assume leader-led"

If a team contract is incomplete, validation should fail.

## Proposed Package and Module Organization

Refactor toward explicit domain ownership:

- `api/teamconfig/`
  - canonical team contract
  - validation
  - policy resolution
- `api/coordination/`
  - coordination-pattern resolution
  - prompt section selection
  - coordination skill resolution
- `api/execution/`
  - queue policies
  - concurrency management
  - run scheduling/execution state
- `api/interop/`
  - single-process Claude Code interop for leader-led teams only
- `api/teams/`
  - HTTP handlers and DTOs only
- `api/store/`
  - persistence only, no business-policy decisions

The implementation may reuse existing packages where appropriate, but the responsibility boundaries above must be the target shape.

## Implementation Strategy

## Phase 1: Redesign the Team Contract

### Goal

Establish the canonical domain model and eliminate the overloaded `spawnMode` contract.

### Work

1. Replace `team.schema.json` with the new team contract.
2. Update Go store models in `api/store/models.go`.
3. Update API DTOs in `api/teams/models.go`.
4. Update UI Zod schemas in `ui/src/lib/schemas/team.schema.ts`.
5. Rewrite bundled `team.json` fixtures to the new shape.
6. Remove old contract fields instead of keeping deprecated aliases.

### Exit Criteria

- a team can be represented without ambiguity
- the contract can encode independent, peer, and leader-led teams
- invalid combinations fail validation

## Phase 2: Build Team Policy Resolution as a First-Class Domain

### Goal

Introduce a single policy-resolution layer so every downstream system reads one normalized source of truth.

### Work

1. Add a canonical `ResolvedTeamPolicy` domain type.
2. Implement validation rules for:
   - leader requirement
   - org chart permissibility
   - messaging mode compatibility
   - queue policy compatibility
3. Ensure all prompt/execution/UI logic consumes `ResolvedTeamPolicy`.
4. Remove ad hoc policy branching from handlers and prompt-builder code.

### Exit Criteria

- Prompt Manager has one authoritative policy-resolution path
- no downstream component interprets team semantics independently

## Phase 3: Redesign Execution and Concurrency

### Goal

Make execution behavior match declared team policy.

### Work

1. Replace the current `TeamExecutionContext` model with policy-aware execution management.
2. Support:
   - `serialized`
   - `bounded-parallel`
3. Redesign execution status to report:
   - multiple active agents
   - queued agents
   - concurrency budget
4. Remove the assumption that multi-process means serialized-per-team.
5. Update scheduler and manual-trigger paths to use the new execution policy.

### Files

- `api/heartbeat/team_execution.go`
- `api/heartbeat/team_execution_store.go`
- `api/heartbeat/handlers.go`
- `api/heartbeat/scheduler.go`
- docs referencing team execution

### Exit Criteria

- QA-style teams can run two members independently when configured
- leader-led teams can still serialize if desired
- execution status reflects real runtime state

## Phase 4: Redesign Prompt Assembly

### Goal

Make prompt composition explicit, minimal, and policy-driven.

### Work

1. Split prompt-building into section builders with policy gates.
2. Separate:
   - org context
   - messaging commands
   - inbox injection
   - durable-state guidance
   - task/decision/knowledge guidance
3. Only include sections that the resolved team policy enables.
4. Remove unconditional relationship/messaging instruction injection.
5. Preserve handoff continuity as an independent capability.

### Files

- `api/heartbeat/prompt_builder.go`
- `api/heartbeat/prompt_builder_test.go`

### Exit Criteria

- an independent QA member prompt does not include unused coordination noise
- a leader-led prompt includes only the coordination surfaces that pattern requires
- prompt preview matches runtime prompt behavior exactly

## Phase 5: Redesign Coordination Skills and Claude Code Interop

### Goal

Make coordination instructions match actual operating models.

### Work

1. Replace current coordination skills with:
   - independent
   - peer
   - leader-led
2. Restrict Claude Code team spawn interop to `leader-led` single-process teams.
3. Remove generic "team lead" assumptions from interop code paths where the policy does not require them.
4. Ensure `member-context` remains usable only where it adds value.

### Files

- `store/skills/packs/core/team-coordination-*`
- `api/interop/claude_code.go`
- any tests around `BuildTeamLeadPrompt` and interop conversion

### Exit Criteria

- single-process does not automatically imply leader-led unless configured
- coordination guidance is pattern-specific and concise
- no team gets instructions for tools or flows it does not use

## Phase 6: Redesign API and CLI Contracts

### Goal

Expose the new model professionally and consistently.

### Work

1. Update create/update/list/detail endpoints to expose the new team contract.
2. Redesign execution-status responses for multi-run visibility.
3. Update trigger endpoints and errors to reflect policy-aware behavior.
4. Update CLI flags and help text.
5. Ensure CLI remains a thin wrapper over the API contract.

### Files

- `api/teams/handlers.go`
- `api/heartbeat/handlers.go`
- `cli/teams/teams.go`
- reference docs

### Exit Criteria

- API and CLI speak in the new concepts directly
- no CLI flag or API field leaks the old mental model

## Phase 7: Redesign the UI

### Goal

Make the new model intuitive, inspectable, and easy to configure.

### Work

1. Replace the current "Execution Mode" section with separate controls for:
   - runtime mode
   - coordination pattern
   - concurrency policy
2. Add explicit controls for coordination capabilities:
   - messaging
   - inbox injection
   - peer triggers
   - task board guidance
   - decision log guidance
   - knowledge log guidance
   - handoff requirement
3. Update org chart UX so teams can be truly flat.
4. Update prompt preview to show exactly which policy-gated sections are included.
5. Update team activity/status UI to show multiple active/upcoming members where relevant.
6. Update explanatory text and copy so users understand the difference between runtime and coordination.

### Files

- `ui/src/lib/schemas/team.schema.ts`
- `ui/src/services/teamService.ts`
- `ui/src/services/heartbeatService.ts`
- `ui/src/components/editor/teamTabs/TeamDashboardTab.tsx`
- `ui/src/components/editor/MemberDetailPanel.tsx`
- `ui/src/components/editor/MemberPromptPreview.tsx`
- `ui/src/components/editor/teamTabs/TeamPromptMatrixTab.tsx`
- `ui/src/hooks/useTeamActivity.ts`
- `ui/src/stores/teamActivityStore.ts`

### Exit Criteria

- users can configure an independent QA team without fighting hierarchy-oriented UI
- prompt preview and dashboard text are faithful to runtime behavior
- no UI surface implies a lead where none exists

## Phase 8: Rewrite Bundled Teams and Fixtures

### Goal

Ensure the repository's canonical team data demonstrates the new architecture correctly.

### Work

1. Rewrite `director-swarm` as an explicit leader-led team.
2. Rewrite `scenario-qa` as an explicit independent team.
3. Remove the QA org-chart edge unless the team truly requires it.
4. Rename QA role labels if needed so they stop implying coordination authority.
5. Update any prompt-manager reference docs/examples that still use the old model.

### Exit Criteria

- bundled teams serve as correct living examples
- QA is modeled the way it actually operates

## Phase 9: Documentation and Reference Cleanup

### Goal

Make docs as professional and authoritative as the code.

### Work

1. Update:
   - `docs/concepts/SWARM-MODEL.md`
   - `docs/concepts/TEAM-EXECUTION.md`
   - `docs/reference/api-endpoints.md`
   - `docs/reference/heartbeat-api.md`
   - `docs/reference/cli-commands.md`
   - `docs/reference/heartbeat-cli.md`
   - `docs/internal/SEAMS.md`
   - `docs/internal/PROBLEMS.md`
   - `docs/internal/PROGRESS.md`
2. Add `// DOC:` references in new or significantly refactored code where appropriate.
3. Ensure `docs/manifest.json` registers all new or moved docs.

### Exit Criteria

- docs teach the new mental model cleanly
- no doc still explains Prompt Manager through the old `spawnMode` assumptions

## Testing Plan

## Unit Tests

Add/refresh unit tests for:

1. team contract validation
2. policy resolution
3. execution queue policy behavior
4. prompt section gating
5. coordination skill resolution
6. explicit lead validation
7. execution-status aggregation
8. interop gating for leader-led single-process teams only

## Contract Tests

Add contract tests for:

1. API request/response shapes
2. CLI flag-to-API mapping
3. prompt preview output sections under each coordination pattern
4. execution-status JSON under serialized and bounded-parallel modes

## Integration Tests

Add integration tests for:

1. independent multi-process team with bounded parallelism
2. peer-coordinated team with async inbox messaging
3. leader-led single-process team using Claude Code interop
4. invalid team contracts rejected at creation/update time

## UI Tests

Add or update tests for:

1. dashboard controls and explanatory copy
2. prompt preview section visibility
3. org chart behavior for flat teams
4. activity/status surfaces showing multiple active/upcoming members
5. form validation for invalid policy combinations

## Scenario Validation

Use the scenario's canonical test entrypoints:

```bash
cd /home/matthalloran8/Vrooli/scenarios/prompt-manager && make test
```

and/or:

```bash
vrooli scenario test prompt-manager
```

Validation should include:

1. API tests
2. CLI tests
3. UI tests
4. scenario-level regression coverage for prompt preview and team execution behavior

## Rollout and Validation Checklist

### Pre-Implementation

- [ ] Finalize canonical contract names
- [ ] Approve independent / peer / leader-led pattern definitions
- [ ] Approve queue policy names and semantics

### Implementation

- [ ] Replace the team contract
- [ ] Introduce policy resolution
- [ ] replace execution manager with policy-aware runtime
- [ ] replace prompt assembly with section gating
- [ ] replace coordination skills
- [ ] update API
- [ ] update CLI
- [ ] update UI
- [ ] rewrite bundled teams
- [ ] update docs

### Validation

- [ ] All unit tests pass
- [ ] All contract tests pass
- [ ] All integration tests pass
- [ ] UI tests pass
- [ ] `make test` passes for `prompt-manager`
- [ ] Prompt preview matches actual runtime prompt composition
- [ ] QA team runs as a truly independent bounded-parallel team
- [ ] Director team runs as an explicit leader-led team

## Risks and Mitigations

### Risk 1: Over-Designing the Contract

Mitigation:

- keep the first contract minimal but complete
- only include knobs with a real runtime or prompt effect
- require every new field to be consumed by at least one production path and one test

### Risk 2: Partial Refactor Leaves Old Semantics Alive

Mitigation:

- remove old fields and branches in the same change stream
- use search-driven cleanup before merge
- fail validation instead of silently mapping old behavior

### Risk 3: UI and Runtime Drift

Mitigation:

- make prompt preview consume the same structured section builder as runtime
- make dashboard copy derive from resolved policy, not hardcoded `spawnMode` text

### Risk 4: Concurrency Bugs

Mitigation:

- isolate execution policy in a dedicated module with strong unit coverage
- keep serialization and bounded-parallel policies separately testable

### Risk 5: Docs Drift

Mitigation:

- update docs in the same implementation stream
- register new docs in manifest immediately
- add `// DOC:` references where new policy modules are introduced

## Non-Goals

1. preserving existing team JSON compatibility
2. keeping old `spawnMode` mental models alive through aliases
3. retaining the QA hierarchy unless it is intentionally reintroduced as product behavior
4. introducing unrelated workflow automation while redesigning the core team contract

## Prohibited Patterns

Do not introduce:

- `if oldField != "" { ... }` compatibility branches
- fallback lead inference
- mixed old/new prompt builders
- hidden policy defaults that recreate leader-led behavior
- duplicate policy interpretation logic in API, UI, and prompt-builder layers
- dead fields kept "just in case"

## Definition of Done

This redesign is done only when all of the following are true:

1. Prompt Manager can model leader-led, peer, and independent teams explicitly.
2. Runtime execution policy is independent from coordination pattern.
3. Prompt assembly is policy-driven and section-gated.
4. A team that disables messaging receives no messaging instructions or inbox prompt sections.
5. The QA team is represented as an independent team in canonical repo data.
6. The Director team is represented as an explicit leader-led team in canonical repo data.
7. API, CLI, UI, prompt preview, docs, and tests all use the same new concepts.
8. No legacy compatibility layer, deadcode retention, or fallback semantics remain.
