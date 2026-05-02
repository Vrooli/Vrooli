# Swarm Manager Operating Mode Authoring

## Purpose

Convert an operator-described agent workflow into a reviewed static Swarm Manager operating mode proposal, then into implementation only after the operator asks to proceed.

Use this skill when a repeated agent methodology should become a first-class Swarm Manager mode with phase orchestration, prompt skills, artifacts, audit-safe backlog reconciliation, metrics, tests, and docs.

Required reading:

- `docs/agent-system/SKILL_AUTHORING.md`

```bash
prompt-manager skill read skill-authoring-tools implementation-plan-authoring documentation-health
```

Also read:

- `scenarios/swarm-manager/docs/internal/OPERATING-MODE-AUTHORING.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md#operating-mode-boundary`
- `scenarios/swarm-manager/api/internal/operatingmode/registry.go`
- `scenarios/swarm-manager/api/internal/operatingmode/definition_builder.go`
- `scenarios/swarm-manager/api/internal/operatingmode/synthetic_mode_test.go`

## Scope

In scope:

- classify whether a described methodology deserves a new Swarm Manager operating mode
- produce a mode proposal before code changes
- design static mode definitions, phase graphs, transition rules, artifacts, output contracts, prompt skills, profile policy, metrics semantics, and backlog reconciliation policy
- implement a static mode when the operator explicitly asks to proceed
- validate registry, prompt catalog, lifecycle behavior, stats, UI/CLI capabilities, skill sync, and docs

Out of scope:

- runtime plugin loading
- runtime mode creation through UI data files
- automatic mode discovery from prior sessions
- bypassing Swarm Manager's backlog reconciliation APIs
- adding mode-specific behavior branches to shared framework code

## Workflow

### Phase 1: Intake And Classification

Entry criteria: the operator describes a recurring workflow or asks to add a new execution/operating mode.

Actions:

1. Restate the methodology as a short workflow, naming the unit of work and what an agent does in each step.
2. Identify the natural scope:
   - backlog item
   - initiative
   - scenario
   - repo/workspace
   - composite scope
3. Decide whether this belongs in a new operating mode, an existing mode, a prompt-manager skill, a CLI command, or an Action.
4. Ask only the clarifying questions needed to remove ambiguity about scope, phase boundaries, audit requirements, and validation.

Exit criteria:

- the methodology has a candidate scope and a recommended implementation path
- any "do not build a new mode" conclusion is explained concretely

Artifact: methodology classification summary.

### Phase 2: Mode Proposal

Entry criteria: the workflow appears to need a new static operating mode.

Actions:

1. Draft the mode ID, label, scope, run strategy, artifact root, and prompt catalog prefix.
2. Draft the phase graph:
   - phase token
   - purpose token
   - profile key
   - whether the phase writes repo files
   - required artifacts
   - required progress/verdict/handoff/criteria
   - transition rules after completion
3. Define the structured `operating_mode_result` contract expected from each phase.
4. Define backlog reconciliation behavior:
   - whether completed member items may be marked complete
   - whether proposal-backed create/update/follow-up reconciliation is allowed
   - when operators should apply reconciliation
5. Define metrics semantics:
   - phases that count toward replan sample size
   - phases that count toward acceptance sample size
   - success/acceptance verdict values
   - any mode-specific metrics that should become future work
6. List prompt-manager skills to create, one per phase.
7. List tests and docs required to make the mode safe.

Exit criteria:

- the operator can review a complete mode proposal before code changes

Artifact: mode proposal with phases, contracts, files, and validation commands.

### Phase 3: Implementation Plan

Entry criteria: the operator approves the mode proposal but has not explicitly asked for immediate implementation.

Actions:

1. Create or update a durable implementation plan using `implementation-plan-authoring`.
2. Include the required reading command from this skill.
3. Include the exact files expected to change:
   - `scenarios/swarm-manager/api/internal/operatingmode/mode_<name>.go`
   - `scenarios/swarm-manager/api/internal/operatingmode/registry.go`
   - operating-mode tests
   - prompt-manager skill directories
   - Swarm Manager docs and manifest entries when operator-facing docs are added
4. Include prohibited files/patterns:
   - no mode-specific shared framework branches
   - no direct backlog `spec.json` mutation for audit changes
   - no fallback prompts for repo-writing phases
5. Include validation commands.

Exit criteria:

- a future agent can implement the mode without chat history

Artifact: implementation plan file.

### Phase 4: Static Mode Implementation

Entry criteria: the operator explicitly asks to implement the approved proposal.

Actions:

1. Add one focused `mode_<name>.go` definition using `buildInitiativeMode`.
2. Add the mode constant and registry map entry.
3. Add one prompt-manager skill per phase. Skill IDs must match the registry-derived `SkillID`.
4. Add or update tests for:
   - registry validation
   - prompt catalog generation and validation
   - phase action transitions
   - output contract enforcement
   - result bindings
   - metrics policy
   - backend capability projection
   - lifecycle behavior unique to the mode
5. Add operator docs for when to use the mode and how to run it.
6. Update `docs/manifest.json` if the docs should appear in the Swarm Manager docs UI.

Exit criteria:

- the mode is production-registered, tested, documented, and visible through `/api/v1/operating-modes`

Artifact: code, skills, tests, and docs.

### Phase 5: Validation

Entry criteria: implementation changes are complete.

Actions:

Run targeted validation:

```bash
cd scenarios/swarm-manager/api
GOWORK=off go test ./internal/operatingmode ./internal/promptcatalog ./internal/stats ./internal/agentactivity ./internal/initiativelock
```

```bash
cd scenarios/swarm-manager/cli
GOWORK=off go test ./...
```

```bash
cd scenarios/swarm-manager/ui
pnpm test -- --run
```

Validate prompt-manager skill registration:

```bash
prompt-manager skill sync
prompt-manager skill show <new-phase-skill-id>
prompt-manager skill read <new-phase-skill-id>
```

When available, run scenario validation:

```bash
vrooli scenario start swarm-manager
test-genie execute swarm-manager --preset comprehensive
```

Exit criteria:

- validation passes or every unavailable command is documented with the exact blocker and the targeted checks that did pass

Artifact: validation summary.

## Convergence Patterns

### Should This Become A New Operating Mode?

| Workflow shape | Preferred path | Reason |
|---|---|---|
| One right-sized item can be workshopped, executed, and reviewed independently | Use `item-level` | Existing backlog execution has maximum control and auditability |
| Initiative-wide ground truth changes during execution and the plan must loop through investigation/replanning | New mode may fit, or use `holistic-loop` | The natural unit of validation is the initiative/system |
| A stable multi-phase plan should be drained by successive agents with durable handoffs | New mode may fit, or use `phased-plan-drain` | The main methodology is sequential continuation |
| Work is deterministic and could be one CLI command | Prefer CLI/Action | Deterministic execution belongs in tooling, not mode prose |
| Workflow is only prompt guidance with no lifecycle, metrics, or audit policy | Prefer a prompt-manager skill | A full mode is unnecessary overhead |
| Workflow needs runtime-editable phases or plugins | Do not implement as part of current Swarm Manager modes | Current architecture supports static Go definitions only |

### Static Mode Design Checklist

- [ ] Mode scope is explicit.
- [ ] Every phase has a stable token and lowercase snake-case purpose.
- [ ] Transitions are represented in `Transitions` and `TransitionRules`.
- [ ] Agent output contract is explicit for each phase.
- [ ] Artifacts live under the mode artifact root.
- [ ] Derived artifacts use `ResultBindings`.
- [ ] Backlog mutations flow through operating-mode reconciliation APIs.
- [ ] Metrics semantics are registry-owned.
- [ ] UI/CLI behavior can render from backend capabilities and phase actions.
- [ ] Tests prove no shared framework branch is needed.

## Anti-Patterns

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| Starting with code before a mode proposal | Hides methodology decisions in implementation | Produce a proposal first |
| Adding `if mode == ...` to shared runners, stats, UI, or CLI | Recreates technical debt and blocks future modes | Generalize missing behavior as registry policy |
| Letting agents directly edit backlog `spec.json` for mode audit | Breaks Swarm Manager's project-management paper trail | Use `complete-items` or `apply-backlog-sync` |
| Creating fallback prompts for mode phases | Repo-writing agents may run with generic unsafe instructions | Fail closed through prompt catalog and prompt-manager validation |
| Creating a mode for simple item work | Adds ceremony and loses item-level control | Use `item-level` |
| Making mode definitions runtime-editable in the first pass | Skips validation, tests, and profile/prompt guarantees | Keep modes static Go definitions |

## Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Phase start fails with missing prompt catalog entry | Skill ID or phase metadata does not match registry | `go test ./internal/operatingmode ./internal/promptcatalog -run 'Prompt|Catalog'` | Align `PromptCatalogPrefix`, `PromptSuffix`, and skill directory IDs |
| Phase completes in AgentManager but Swarm Manager marks it failed | `operating_mode_result` is missing required artifacts/progress/verdict/handoff | Inspect the round error and phase `OutputContract` | Fix the phase skill output contract or registry requirements |
| UI does not show expected controls | Capability derivation does not expose the behavior | Inspect `/api/v1/operating-modes` and workspace response | Update registry policy or capability derivation, not component mode checks |
| Stats omit replan/acceptance samples | Phase metrics policy did not opt in | Inspect `PhaseMetricsSpec` | Set `CountsReplanSample` or `CountsAcceptanceSample` on the appropriate phase |
| A new mode requires a framework branch | The registry lacks a policy concept | Add the smallest general policy and prove it with synthetic-mode tests |

Promotion note: if adding a mode repeatedly requires the same manual file creation and validation, promote scaffolding/doctor checks into Swarm Manager CLI before expanding this skill with more procedural prose.

## Output Expectations

When applying this skill, produce one of:

- a "do not create a new mode" recommendation with rationale and an alternate path
- a reviewed operating-mode proposal
- a durable implementation plan file
- an implemented and validated static operating mode

Always include validation commands and explicitly identify any command that could not be run.
