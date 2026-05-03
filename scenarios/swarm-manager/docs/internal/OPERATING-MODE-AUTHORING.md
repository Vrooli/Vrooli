# Operating Mode Authoring Guide

> Current State (2026-05-01): Swarm Manager operating modes are static Go definitions. A new mode should be authorable through one focused mode definition file, prompt-manager skills, targeted tests, and docs. Shared framework control flow should not need mode-specific branches.

This guide is the source material for future agents and for a future `mode-authoring` skill. It explains where operating-mode behavior belongs, what files should usually change, and which framework files should stay untouched when adding a new methodology.

## Agent-Assisted Authoring

Prompt Manager provides the `swarm-manager-operating-mode-authoring` skill for turning an operator-described methodology into a static Swarm Manager operating mode proposal or implementation. Use that skill when the operator describes a repeated agent workflow such as "generate a plan, let successive agents drain phases, classify progress, then review" and wants to decide whether it should become a first-class Swarm Manager mode.

The skill is intentionally an authoring workflow, not a runtime plugin mechanism. It should first produce a mode proposal that identifies scope, phases, transitions, artifacts, output contracts, backlog reconciliation policy, metrics semantics, prompt skills, tests, and docs. It should implement code only when explicitly asked to proceed after the proposal is accepted.

## Authoring Rule

Adding a new static operating mode should mostly require:

1. One new `api/internal/operatingmode/mode_<name>.go` definition file.
2. A new registry entry in `api/internal/operatingmode/registry.go`.
3. Prompt-manager skills matching the mode phases.
4. Focused tests proving registry, prompt catalog, capabilities, transitions, artifacts, metrics, and lifecycle behavior.
5. User/internal docs for the new operator workflow.

Do not add mode-specific branches to handlers, stats aggregation, UI components, CLI commands, prompt catalog lookup, artifact appliers, phase runners, or lock/activity shared packages. If a new mode appears to require that, first ask whether the missing behavior is a registry policy that should be generalized.

## Files To Touch

Typical files for a new initiative-scoped mode:

| File | Expected change |
|---|---|
| `api/internal/operatingmode/mode_<name>.go` | New mode definition built with `buildInitiativeMode`, including decision metadata (`BestFor`, `NotFor`, `Tradeoffs`, `WhenInDoubtPickInstead`) |
| `api/internal/operatingmode/registry.go` | Add the mode constant and one registry map entry |
| `api/internal/operatingmode/registry_test.go` | Validate the new mode definition and capability contract |
| `api/internal/operatingmode/service_test.go` or a focused test file | Exercise lifecycle paths that are unique to the mode |
| Prompt-manager skill source | Add one skill per phase, matching `SkillID`/`CatalogID` |
| `ui/src/components/initiative/operating-mode/decision-flow.config.ts` | Add at least one terminal-question path that selects the new mode in the how-to-choose flow |
| `docs/concepts/EXECUTION-MODES.md` or `docs/guides/<mode>.md` | Explain when operators should use the mode |
| `docs/manifest.json` | Register new operator docs when they should appear in the docs UI |

Files that should not need mode-specific edits:

| File | Why it should stay generic |
|---|---|
| `api/internal/operatingmode/state.go` | Phase availability is derived from `PhaseGraph.Transitions` and `TransitionRules` |
| `api/internal/operatingmode/artifact_applier.go` | Derived artifacts are declared through `ResultBindings` |
| `api/internal/promptcatalog/catalog.go` | Operating-mode prompt entries are generated from the registry |
| `api/internal/stats/engine.go` | Replan and acceptance semantics come from `MetricsPolicy` |
| `api/internal/agentactivity/types.go` | Initiative-owned activity purposes accept registry-authored purpose tokens |
| `api/internal/initiativelock/lock.go` | Lock purposes are registry-authored strings |
| `ui/src/components/initiative/operating-mode/` | Controls should render backend-declared capabilities and phase actions |
| `cli/cmd_initiatives_operating_mode.go` | CLI remains a thin API wrapper over catalog/workspace responses |

## Definition Shape

Initiative-scoped modes should use `buildInitiativeMode` from `api/internal/operatingmode/definition_builder.go`. The builder keeps repeated policy in one place:

- initiative scope
- mode artifact root and round root
- prompt catalog ID and skill ID derivation
- phase profile map generation
- default backlog reconciliation capabilities
- metrics event source
- initiative-exclusive locking
- operating-mode UI tab
- structured output contract defaults
- required artifact extraction
- result-binding artifact merging
- default lock purpose from activity purpose

Minimal structure:

```go
package operatingmode

const ModeExample Mode = "example-mode"

func exampleModeDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                ModeExample,
		Label:               "Example Mode",
		RunStrategy:         RunStrategyOperatorGatedLoop,
		ArtifactRoot:        "modes/example-mode",
		PromptCatalogPrefix: "swarm-manager-example-mode",
		DefaultProfileKey:   ProfileDeepWork,
		StartPhase:          "assess",
		Terminal:            []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"assess": {"execute"},
			"execute": {"review", "assess"},
			"review": {"assess"},
		},
		TransitionRules: map[Phase][]TransitionRule{
			"execute": {
				{
					When: TransitionCondition{
						Kind:       TransitionConditionPayloadBool,
						PayloadKey: payloadReplanNeeded,
						BoolValue:  true,
					},
					Next: []Phase{"assess"},
				},
				{
					When: TransitionCondition{Kind: TransitionConditionAlways},
					Next: []Phase{"review"},
				},
			},
		},
		Phases: []initiativePhaseSpec{
			{
				Phase:           "assess",
				Purpose:         "example_mode_assess",
				PromptPurpose:   "Assess the initiative state for example-mode execution.",
				ProfileKey:      ProfileDeepWork,
				OutputArtifacts: []ArtifactDefinition{requiredOutputArtifact("modes/example-mode/assessment.md", "text/markdown")},
			},
			{
				Phase:      "execute",
				Purpose:    "example_mode_execute",
				ProfileKey: ProfileDeepWork,
				WritesRepo: true,
				Metrics:    PhaseMetricsSpec{CountsReplanSample: true},
			},
			{
				Phase:            "review",
				Purpose:          "example_mode_review",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
		},
	})
}
```

After adding the definition, register it in `registry.go`:

```go
const (
	ModeExample Mode = "example-mode"
)

var registry = map[Mode]Definition{
	ModeItemLevel:       itemLevelDefinition(),
	ModeHolisticLoop:    holisticLoopDefinition(),
	ModePhasedPlanDrain: phasedPlanDrainDefinition(),
	ModeExample:         exampleModeDefinition(),
}
```

## Decision Metadata

Every `Definition` carries four fields that drive operator-facing decision support in the picker, the details page, and the how-to-choose dialog. The registry validator rejects empty `BestFor`, `NotFor`, or `Tradeoffs` lists at startup, and rejects a `WhenInDoubtPickInstead` that references an unregistered mode or itself, so the API will fail to boot if any of these constraints are violated.

| Field | Use | Constraint |
|---|---|---|
| `BestFor []string` | Plain-prose work shapes this mode is the right pick for | At least one entry; entries cannot be blank |
| `NotFor []string` | Plain-prose work shapes this mode handles poorly | At least one entry; entries cannot be blank |
| `Tradeoffs []string` | Structural tradeoffs an operator accepts when choosing this mode | At least one entry; entries cannot be blank |
| `WhenInDoubtPickInstead Mode` | Registered mode an operator should pick if unsure between this one and another | Empty (mode is itself a safe default), or a registered mode that is not self |

For modes built with `buildInitiativeMode`, the four fields live on `initiativeModeSpec` and are propagated into the resulting `Definition`. For modes constructed via direct `Definition{}` literals (such as `mode_item_level.go`), populate the fields directly on the literal.

The catalog wire response uses snake_case JSON keys (`best_for`, `not_for`, `tradeoffs`, `when_in_doubt_pick_instead`). The three lists do not use `omitempty` because the validator already guarantees they are non-empty; an empty list arriving on the wire is itself a contract violation worth surfacing. Operator-editable overlay (`OverlayStore`) is intentionally not extended to these fields — they are semantic strategy claims authored alongside the mode definition and change via redeploy.

## Phase Fields

Use `initiativePhaseSpec` for each phase:

| Field | Use |
|---|---|
| `Phase` | Stable phase token used in routes, rounds, stats, and UI |
| `Kind` | **Required.** One of `PhaseKindInvestigate \| Execute \| Review \| Reconcile`. Drives Operations Center column placement, lane utilization, and per-lane metrics. The validator rejects empty values. |
| `AutoStartAfter` | Optional. Length ≤ 1. When set, names the predecessor phase whose successful completion auto-starts this phase via the round refresher. Multi-predecessor races are out of scope in v1. |
| `Purpose` | Lowercase snake-case audit token for agent activity |
| `LockPurpose` | Optional lowercase snake-case token; defaults to `Purpose` |
| `PromptSuffix` | Optional suffix when prompt IDs should not use the raw phase token |
| `PromptTitle` | Prompt Center display title; defaults to a humanized phase |
| `PromptPurpose` | Prompt Center purpose text |
| `PromptTrigger` | Prompt Center trigger text |
| `ProfileKey` | Scenario-owned AgentManager profile key such as `swarm-manager/deep-work` |
| `WritesRepo` | True when the phase is expected to modify code/files |
| `OutputArtifacts` | Durable mode artifacts the phase may write |
| `ResultBindings` | Derived writes from structured result fields |
| `Metrics` | Per-phase opt-in for replan and acceptance metric samples |
| `RequiresProgress` | Completed result must include a valid progress decision |
| `RequiresVerdict` | Completed result must include a verdict |
| `RequiresHandoff` | Completed result must include a handoff |
| `RequiresCriteria` | Phase start requires initiative acceptance criteria |

Purpose tokens must be lowercase snake-case. Do not add a shared constant for each new phase in `agentactivity` or `initiativelock`; the registry owns those purpose strings.

### Choosing `Kind`

Phase kinds describe the **shape of work**, not the phase name:

- `PhaseKindInvestigate` — research, planning, scoping. Reads code/state and produces findings or plans. Examples: `holistic-loop` `investigate`, `holistic-loop` `plan`, `phased-plan-drain` `prepare_plan`.
- `PhaseKindExecute` — repository-modifying work. Writes code, runs tests, ships diffs. Examples: `holistic-loop` `execute`, `phased-plan-drain` `execute_next`.
- `PhaseKindReview` — judgment phases. Classify progress, render verdicts against acceptance criteria. Examples: `phased-plan-drain` `classify_progress`, `holistic-loop` `review`, `phased-plan-drain` `review`.
- `PhaseKindReconcile` — backlog-reality alignment. Reads prior round artifacts and emits a `BacklogSyncPlan` proposal aligning the backlog with what was actually done. (P4 introduces this kind on the two existing initiative-scoped modes.)

A phase belongs to exactly one kind. If multiple kinds seem to fit, the phase is probably doing too much — split it.

## Transitions

Every phase-to-phase edge must be declared in `Transitions`. Conditional routing belongs in `TransitionRules`.

Supported conditions:

| Condition | Use |
|---|---|
| `TransitionConditionAlways` | Default route after more specific rules |
| `TransitionConditionPayloadBool` | Route based on a boolean in the round payload, such as `replan_needed` |
| `TransitionConditionProgressDecision` | Route based on `operating_mode_result.progress.decision` |

Rules may return no next phase. This is how a classifier can intentionally block progression, such as a `blocked` progress decision.

Do not encode transition behavior in `state.go`, UI components, or CLI commands.

## Output Contracts

All initiative phase completions require a structured `operating_mode_result` envelope. The builder derives `OutputContract.RequiredArtifacts` from required `OutputArtifacts`, and it merges `ResultBindings` into the declared output list.

Use these flags to make phase outputs explicit:

- `OutputArtifacts`: agent-provided or derived files that belong under the mode artifact root.
- `RequiresProgress`: requires a valid `ProgressState`.
- `RequiresVerdict`: requires a review verdict.
- `RequiresHandoff`: requires a handoff summary for sequential continuation.
- `RequiresCriteria`: requires acceptance criteria before the phase can start.

The refresher must reject incomplete successful runs instead of advancing the graph with malformed output.

## Result Bindings

Result bindings convert structured result fields into durable artifacts without adding mode-specific code to the artifact applier.

Current binding:

```go
ResultBindings: []ResultBinding{
	progressResultArtifact("modes/example-mode/progress.json"),
},
RequiresProgress: true,
```

The bound artifact path must also be valid under the mode artifact root. The builder adds the bound artifact to `OutputArtifacts`, and validation fails if a binding writes an undeclared artifact.

## Prompt Catalog

Operating-mode prompt catalog entries are generated by `api/internal/operatingmode/prompt_catalog_entries.go` from phase definitions. `api/internal/promptcatalog/catalog.go` consumes those entries.

For each phase, create a prompt-manager skill whose ID matches the derived `SkillID`:

```text
<PromptCatalogPrefix>-<PromptSuffix or Phase>
```

Examples:

- `swarm-manager-holistic-loop-investigate`
- `swarm-manager-phased-plan-execute-next`

`ValidatePromptCatalog` checks catalog ID, skill ID, mode, phase, and output paths. Phase starts fail closed if the prompt catalog is missing or the rendered prompt is empty.

## Metrics

Metrics semantics are registry-driven through `MetricsPolicy`.

Use `PhaseMetricsSpec` in the phase definition:

```go
Metrics: PhaseMetricsSpec{CountsReplanSample: true}
Metrics: PhaseMetricsSpec{CountsAcceptanceSample: true}
```

The builder derives:

- `Metrics.EventSource` from the mode ID.
- `Metrics.ReplanSamplePhases` from `CountsReplanSample`.
- `Metrics.AcceptanceSamplePhases` from `CountsAcceptanceSample`.
- `Metrics.AcceptedVerdicts` when at least one acceptance phase exists.

Do not add phase-name checks to `api/internal/stats/engine.go`.

## Capabilities

The API declares mode capabilities in both catalog and workspace responses. UI and CLI code should consume these fields instead of inferring mode behavior locally:

- `supports_phases`
- `can_start_phases`
- `can_complete_items`
- `can_apply_backlog_sync_proposals`
- `requires_acceptance_criteria`
- `supports_artifacts`
- `supports_handoffs`
- `uses_item_execution_flow`

Capabilities are derived in `api/internal/operatingmode/workspace.go` from registry policies and phase contracts. New mode behavior that affects rendering should be exposed through capability derivation or phase actions, not through frontend mode-name checks.

## Backlog Audit

Initiative modes can execute initiative-wide work, but backlog items remain the audit and scope trail. Agents must not edit backlog item `spec.json` files directly to mark work complete or create follow-ups.

Use the operating-mode reconciliation endpoints:

- `complete-items` for run-id-validated item completion.
- `apply-backlog-sync` for proposal-backed item create/update/follow-up reconciliation.

The reconciler attaches source metadata: entrypoint, initiative, mode, phase, round, run ID, and requester.

## Reconcile Phase Contract

Every initiative-scoped operating mode MUST declare exactly one phase of `Kind: PhaseKindReconcile` whose `AutoStartAfter` lists the predecessor that closes the iteration (typically `review`). The reconcile phase is the contract by which a mode proves it stays consistent with the backlog over time. Modes without a reconcile phase silently let the backlog drift from what the code actually does — the operator notices when the next iteration starts from a stale plan.

Required shape:

```go
{
    Phase:               "reconcile",
    Kind:                PhaseKindReconcile,
    AutoStartAfter:      []Phase{"review"},
    Purpose:             "<mode>_reconcile",
    PromptSuffix:        "reconcile",
    PromptTitle:         "<Mode Label> Backlog Reconcile",
    PromptTrigger:       "Round refresher auto-starts <mode> reconcile after review completes",
    PromptPurpose:       "Read prior round artifacts and propose backlog mutations that align the initiative with the work just completed.",
    ProfileKey:          ProfileAnalysis,
    RequiresBacklogSync: true,
}
```

Phase graph topology:

- `Transitions[predecessor] = []Phase{"reconcile"}` (replace any prior route out of the predecessor; document operator-driven loops as separate `StartPhase` calls, not phase-internal edges).
- `Terminal = []Phase{"reconcile"}` — reconcile is the only terminal phase.
- The reconcile phase has no outgoing transitions: continuation across iterations is the operator's call.

Backlog sync policy:

- The mode's `BacklogSync.ApplyMode` MUST be set; for v1 the only implemented value is `BacklogSyncApplyOperatorGated`. Other values (`auto-apply-safe`, `auto-apply-all`) cause `apply-backlog-sync` to return HTTP 501 with `apply_mode_not_implemented`. Do not set those values until the corresponding apply path lands.
- `BacklogSync.Capabilities` should include `propose_mutations` (so the proposal can land in the round payload) and any per-mode capabilities the reconcile prompt is allowed to exercise.

Prompt skill:

- Create a skill at `prompt-manager/store/skills/packs/core/<catalog-id>/SKILL.md` whose ID matches `<PromptCatalogPrefix>-reconcile`.
- The SKILL.md MUST substitute the shared `{{BACKLOG_SYNC_PROPOSAL_SNIPPET}}` template variable. The snippet content lives in `api/internal/operatingmode/promptcatalog/snippets.go`; both reconcile prompts and the initiative-feedback prompt render it from a single source so the proposal envelope contract cannot drift across surfaces.
- Adding a new `proposals.Op` requires extending the snippet first; the `proposals_test.go` reverse-coupling test fails until the snippet documents the new op.

Auto-dispatch behavior:

- The round refresher fires `maybeAutoStartNext` *after* the predecessor lock is released and *only* when the predecessor transitions to `RoundStatusCompleted`. Failed and cancelled runs are skipped — there is nothing reliable to reconcile against.
- On `agentactivity.ErrLaneSaturated`, the predecessor round is marked with a `pending_auto_start` payload entry; periodic `RefreshRound` calls retry the dispatch until lane capacity recovers. Do not implement custom retry logic in mode-specific code — the refresher is the single seam.

## Tests

At minimum, add or update tests for:

- Registry validation for the new mode and malformed variants.
- Prompt catalog generation and `ValidatePromptCatalog`.
- Catalog/workspace capabilities.
- Phase action transitions, including conditional routes.
- Output contract enforcement for required artifacts/progress/verdict/handoff.
- Result binding writes, if used.
- Metrics policy semantics, if the mode opts into replan or acceptance samples.
- Lifecycle behavior that is unique to the new mode.

Use `api/internal/operatingmode/synthetic_mode_test.go` as the reference harness for proving that a non-production mode can exercise the framework without leaking into the production registry.

Targeted validation:

```bash
cd scenarios/swarm-manager/api
GOWORK=off go test ./internal/operatingmode ./internal/promptcatalog ./internal/stats ./internal/agentactivity ./internal/initiativelock
```

When UI or CLI behavior changes:

```bash
cd scenarios/swarm-manager/cli && GOWORK=off go test ./...
cd scenarios/swarm-manager/ui && pnpm test -- --run
```

Scenario validation when available:

```bash
vrooli scenario start swarm-manager
test-genie execute swarm-manager --preset comprehensive
```

## Authoring Checklist

- [ ] The mode has one focused definition file.
- [ ] The registry map contains the new mode and `ValidateRegistry` passes.
- [ ] Decision metadata is populated: `BestFor`, `NotFor`, `Tradeoffs` each have ≥1 plain-prose entry; `WhenInDoubtPickInstead` is empty or a registered mode (never self).
- [ ] `decision-flow.config.ts` has at least one terminal-question path selecting the new mode.
- [ ] **Every phase declares `Kind` as one of `PhaseKindInvestigate | Execute | Review | Reconcile`. The validator rejects empty values; pick the kind that describes the *shape of work*, not the name.**
- [ ] **Any `AutoStartAfter` declarations are length ≤ 1 and reference a registered phase (not self).**
- [ ] **The mode declares exactly one `Kind: PhaseKindReconcile` phase, `AutoStartAfter: []Phase{<predecessor>}`, `RequiresBacklogSync: true`, and reconcile is the only terminal phase.**
- [ ] **`BacklogSync.ApplyMode` is set to `BacklogSyncApplyOperatorGated` (the only v1-implemented value).**
- [ ] **The reconcile prompt skill substitutes `{{BACKLOG_SYNC_PROPOSAL_SNIPPET}}`; surface-specific guidance (what to read first, how to scope rationales for this mode) lives in the skill body around the snippet, not duplicated.**
- [ ] All phase purpose tokens are lowercase snake-case and owned by the mode definition.
- [ ] Every phase has a matching prompt-manager skill.
- [ ] Prompt catalog validation catches missing or mismatched phase metadata.
- [ ] Transition behavior is declared through `Transitions` and `TransitionRules`.
- [ ] Derived artifact writes use `ResultBindings`.
- [ ] Metrics semantics are expressed through phase metrics policy.
- [ ] UI/CLI behavior uses backend capabilities and phase actions.
- [ ] Backlog mutations flow through operating-mode reconciliation endpoints.
- [ ] Tests prove the new mode does not require shared framework branches.

## Prohibited Patterns

- Do not add `if mode == ...` behavior branches to shared framework code.
- Do not make UI components decide phase sequencing or backlog sync availability.
- Do not duplicate generated operating-mode prompt catalog facts by hand.
- Do not weaken output contracts to accept incomplete agent results.
- Do not use fallback prompts for repo-writing phases.
- Do not add per-phase constants to shared activity or lock packages.
- Do not create runtime-editable modes as part of static authoring work.
