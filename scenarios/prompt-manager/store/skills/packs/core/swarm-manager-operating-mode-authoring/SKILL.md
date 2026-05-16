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

## Startup Routine

1. First inspect attached `startup_brief` context. For Operating Mode Authoring it summarizes registered modes, decision metadata, scope/run strategies, phase counts, and authoring drill-down commands.
2. If no startup brief is attached or the operator asks about the latest catalog, refresh once with:

   ```bash
   swarm-manager sessions startup-brief --id "$VROOLI_SWARM_MANAGER_SESSION_ID" --refresh --json
   ```

   If no session ID is available, use `swarm-manager operating-mode brief --json`.
3. For broad classification prompts, compare against the startup brief before reading mode files or docs. Recommend an existing mode unless the workflow needs a distinct phase graph, artifact contract, audit policy, metrics semantics, or backlog reconciliation behavior.
4. First-response budget: before the first useful answer, use the attached brief and at most one targeted catalog/detail command:

   ```bash
   swarm-manager operating-mode list --json
   swarm-manager operating-mode get --mode <mode> --json
   ```

5. Treat the required reading list below as conditional after classification. Read it before producing an implementation plan, code changes, registry edits, prompt skill changes, or detailed authoring rationale; do not front-load all of it for a simple "does this need a mode?" question.

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
   - required progress/verdict/handoff/criteria/backlog-sync
   - transition rules after completion
   - **reconcile phase**: every initiative-scoped mode MUST declare exactly one phase of `Kind: PhaseKindReconcile`. Its `AutoStartAfter` lists the predecessor that closes the iteration (typically `review`). It sets `RequiresBacklogSync: true`. It is the mode's only terminal phase. Its purpose is to read prior round artifacts and emit a `BacklogSyncPlan` proposal aligning the backlog with the work just completed. Modes without a reconcile phase silently let the backlog drift.
3. Draft decision metadata that the picker, details page, and how-to-choose dialog will surface to operators. Each list must have at least one entry; entries must be plain prose an operator can scan, not engineer-shaped capability deltas:
   - `bestFor` — the work shapes this mode is the right pick for
   - `notFor` — the work shapes this mode handles poorly
   - `tradeoffs` — the structural tradeoffs an operator accepts when choosing this mode
   - `whenInDoubtPickInstead` — the registered mode an operator should pick if they're unsure between this and another (leave empty only if this mode is itself a safe default; never reference self)
4. Define the structured `operating_mode_result` contract expected from each phase.
5. Define backlog reconciliation behavior:
   - whether completed member items may be marked complete
   - whether proposal-backed create/update/follow-up reconciliation is allowed
   - when operators should apply reconciliation
   - the mode's `BacklogSyncPolicy.ApplyMode` value. v1 only implements `BacklogSyncApplyOperatorGated`; the auto-apply variants land as `ErrApplyModeNotImplemented` (HTTP 501) until the apply path is wired. Default to operator-gated unless a separate plan has explicitly added an auto-apply implementation.
6. Define metrics semantics:
   - phases that count toward replan sample size
   - phases that count toward acceptance sample size
   - success/acceptance verdict values
   - any mode-specific metrics that should become future work
7. List prompt-manager skills to create, one per phase. The reconcile phase's SKILL.md MUST substitute the shared `{{BACKLOG_SYNC_PROPOSAL_SNIPPET}}` template variable so the proposal envelope contract stays single-sourced (see `api/internal/operatingmode/promptcatalog/snippets.go`).
8. List tests and docs required to make the mode safe.

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
2. Populate decision metadata in the mode definition: `BestFor` (≥1 entry), `NotFor` (≥1 entry), `Tradeoffs` (≥1 entry), and `WhenInDoubtPickInstead` (a registered `Mode` constant or empty). The registry validator rejects empty `BestFor`/`NotFor`/`Tradeoffs` lists at startup, so the API will fail to boot if any of them are missing.
3. Add the mode constant and registry map entry.
4. Add one prompt-manager skill per phase. Skill IDs must match the registry-derived `SkillID`.
5. Update `scenarios/swarm-manager/ui/src/components/initiative/operating-mode/decision-flow.config.ts` to include at least one terminal-question path that selects the new mode. The how-to-choose decision flow validates referenced modes against the catalog at render time; an unreferenced mode will not be reachable through the guided flow, and a reference to a renamed/removed mode renders a visible error chip.
6. Add or update tests for:
   - registry validation
   - prompt catalog generation and validation
   - phase action transitions
   - output contract enforcement
   - result bindings
   - metrics policy
   - backend capability projection
   - lifecycle behavior unique to the mode
7. Add operator docs for when to use the mode and how to run it.
8. Update `docs/manifest.json` if the docs should appear in the Swarm Manager docs UI.

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

Validate that decision metadata reached the wire and the how-to-choose flow points at the new mode:

```bash
# Decision metadata appears on the catalog response. Replace <new> with the mode token.
curl -s http://localhost:9999/api/v1/operating-modes \
  | jq '.modes[] | select(.mode=="<new>") | {best_for, not_for, tradeoffs, when_in_doubt_pick_instead}'

# decision-flow.config.ts contains at least one terminal node referencing the new mode.
rg -n '"<new>"' scenarios/swarm-manager/ui/src/components/initiative/operating-mode/decision-flow.config.ts
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
- [ ] Decision metadata is populated: `BestFor`, `NotFor`, `Tradeoffs` each have ≥1 plain-prose entry, and `WhenInDoubtPickInstead` is either empty or a registered mode (never self).
- [ ] `decision-flow.config.ts` includes at least one terminal node referencing the new mode.
- [ ] Every phase has a stable token and lowercase snake-case purpose.
- [ ] Every phase declares `Kind` (one of investigate/execute/review/reconcile).
- [ ] **Mode declares exactly one `Kind: PhaseKindReconcile` phase with `AutoStartAfter: []Phase{<predecessor>}` and `RequiresBacklogSync: true`. Reconcile is the only terminal phase.**
- [ ] **`BacklogSync.ApplyMode` is set to `BacklogSyncApplyOperatorGated`. (Auto-apply values are not implemented in v1 and surface as HTTP 501 at runtime.)**
- [ ] **The reconcile prompt skill substitutes `{{BACKLOG_SYNC_PROPOSAL_SNIPPET}}` so the proposal envelope contract is single-sourced.**
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
